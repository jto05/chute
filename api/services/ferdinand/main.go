// ferdinand is a long-running service that periodically fetches rodeo results
// from prorodeo.com and persists them to the data store.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jto05/chute/app/domain/rodeoapp"
	"github.com/jto05/chute/business/data/store/rodeodb"
	"github.com/jto05/chute/foundation/logger"
)

var build = "develop"

func main() {
	log := logger.New()

	if err := run(log); err != nil {
		log.Error("startup", "error", err)
		os.Exit(1)
	}
}

func run(log *logger.Logger) error {
	log.Info("startup", "version", build)

	// TODO: load from env / config file.
	cfg := struct {
		ScrapeInterval time.Duration
		DataDir        string
		StartDate      string
		EndDate        string
	}{
		ScrapeInterval: 6 * time.Hour,
		DataDir:        "data/results/rodeo",
		StartDate:      "1/1/2026",
		EndDate:        "12/31/2026",
	}

	store := rodeodb.New(cfg.DataDir)
	app := rodeoapp.New(log, store)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.Info("ferdinand running", "interval", cfg.ScrapeInterval)

	// // Run immediately on startup, then on the ticker.
	// if err := app.Sync(ctx, cfg.StartDate, cfg.EndDate); err != nil {
	// 	log.Error("sync", "error", err)
	// }

	if err := app.SyncAthletes(ctx); err != nil {
		log.Error("sync", "error", err)
	}

	ticker := time.NewTicker(cfg.ScrapeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// if err := app.Sync(ctx, cfg.StartDate, cfg.EndDate); err != nil {
			// 	log.Error("sync", "error", err)
			// }
			if err := app.SyncAthletes(ctx); err != nil {
				log.Error("sync", "error", err)
			}
		case <-ctx.Done():
			fmt.Println()
			log.Info("shutdown", "reason", ctx.Err())
			return nil
		}
	}
}
