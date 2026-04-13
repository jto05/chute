// Package rodeoapp orchestrates the scrape-and-store cycle.
// It delegates HTTP fetching to ferdinand and persistence to the store layer.
package rodeoapp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/am29/ferdinand/app/domain/prorodeoapp"
	"github.com/jto05/chute/business/data/store/rodeodb"
	"github.com/jto05/chute/foundation/logger"
)

// App holds the dependencies for rodeo sync operations.
type App struct {
	log   *logger.Logger
	store *rodeodb.Store
}

// New constructs an App.
func New(log *logger.Logger, store *rodeodb.Store) *App {
	return &App{log: log, store: store}
}

// SyncAthletes fetches all contestants and stores result into storage.
func (a *App) SyncAthletes(ctx context.Context) error {
	a.log.Info("sync athletes started")

	// shared pool of ids
	ids := make(chan int, 5000) // at size 750 because workers consume as soon as fetched ids collected

	// Part 1: collect all contestantIDs
	var collectWg sync.WaitGroup // semaphore
	for _, letter := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
		collectWg.Add(1) // post semapohre

		go func(letter string) {
			defer collectWg.Done() // wait semaphore at end of func
			for idx := 1; ; idx++ {

				// fetch athletes by letter
				raw, err := prorodeoapp.FetchAthletesByLetter(ctx, letter, 100, idx)
				if err != nil {
					a.log.Error("fetch athlete ids", "letter", letter, "error", err)
					return
				}

				pageIDs, err := prorodeoapp.ParseAthleteIDs(raw)
				// NOTE: do i need to handle error in this case?
				if err != nil || len(pageIDs) == 0 { // return if no more IDs pulled
					return
				}

				// add ids to shared pool
				for _, id := range pageIDs {
					ids <- id
				}
			}
		}(string(letter))
	}

	// close ids channel when all letters are done fetching
	go func() {
		collectWg.Wait()
		close(ids)
	}()

	/*

		schema:

		-
		- blob biograph text (characters)


	*/

	// part 2: fetch athlete based on id from shared channel and store
	var fetchWg sync.WaitGroup
	numOfConsumers := 30

	for range numOfConsumers {
		fetchWg.Add(1) // post

		go func() {
			defer fetchWg.Done() // wait
			for id := range ids {
				// NOTE: should already stored athletes be skipped? should implement way to update exisitng athletes?
				if a.store.AthleteExists(id) {
					continue // for now skip existing athletes
				}

				raw, err := prorodeoapp.FetchAthlete(ctx, id)
				if err != nil {
					a.log.Error("fetch athlete", "id", id, "error", err)
					continue
				}

				athlete, err := prorodeoapp.ParseAthlete(raw)
				if err != nil {
					a.log.Error("parse athlete", "id", id, "error", err)
					continue
				}

				if err := a.store.SaveAthlete(athlete); err != nil {
					a.log.Error("save athlete", "id", id, "error", err)
				}

			}

			time.Sleep(50 * time.Millisecond)
		}()
	}

	return nil
}

// Sync fetches all completed rodeos in the given date range and persists their
// results to the store. Already-stored rodeos are skipped.
func (a *App) Sync(ctx context.Context, startDate, endDate string) error {
	a.log.Info("sync started", "start", startDate, "end", endDate)

	raw, err := prorodeoapp.FetchRodeosInDateRange(ctx, startDate, endDate)
	if err != nil {
		return fmt.Errorf("fetch schedule: %w", err)
	}

	rodeos, err := prorodeoapp.ParseCompletedRodeos(raw)
	if err != nil {
		return fmt.Errorf("parse schedule: %w", err)
	}

	var stored, skipped int
	for _, rodeo := range rodeos {
		if a.store.Exists(rodeo.RodeoId) {
			skipped++
			continue
		}

		results, err := prorodeoapp.FetchResults(ctx, rodeo.RodeoId)
		if err != nil {
			a.log.Error("fetch results", "rodeoID", rodeo.RodeoId, "error", err)
			continue
		}

		if err := a.store.Save(rodeo.RodeoId, results); err != nil {
			a.log.Error("save results", "rodeoID", rodeo.RodeoId, "error", err)
			continue
		}

		a.log.Info("stored", "rodeoID", rodeo.RodeoId, "name", rodeo.Name)
		stored++
	}

	a.log.Info("sync complete", "stored", stored, "skipped", skipped)
	return nil
}
