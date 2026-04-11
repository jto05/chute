// Package rodeoapp orchestrates the scrape-and-store cycle.
// It delegates HTTP fetching to ferdinand and persistence to the store layer.
package rodeoapp

import (
	"context"
	"fmt"

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
