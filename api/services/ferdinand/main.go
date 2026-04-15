// ferdinand is a long-running service that periodically fetches rodeo results
// from prorodeo.com and persists them to the data store.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jto05/chute/app/domain/ferdinandapp"
	"github.com/jto05/chute/app/domain/rodeoapp"
	"github.com/jto05/chute/business/store"
	"github.com/jto05/chute/foundation/logger"
	"github.com/jto05/chute/foundation/web"
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
		ScrapeInterval  time.Duration
		DataDir         string
		StartDate       string
		EndDate         string
		Host            string
		ReadTimeout     time.Duration
		WriteTimeout    time.Duration
		IdleTimeout     time.Duration
		ShutdownTimeout time.Duration
	}{
		ScrapeInterval:  6 * time.Hour,
		DataDir:         "data/chute.db",
		StartDate:       "1/1/2026",
		EndDate:         "12/31/2026",
		Host:            "0.0.0.0:4000",
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    10 * time.Second,
		IdleTimeout:     120 * time.Second,
		ShutdownTimeout: 15 * time.Second,
	}

	db, err := store.New(cfg.DataDir)
	if err != nil {
		log.Error("store creation", "error", err)
	}

	scraper := rodeoapp.New(log, db)
	api := ferdinandapp.New(log, db)

	mux := web.NewMux(log)
	api.Routes(mux)

	srv := http.Server{
		Addr:         cfg.Host,
		Handler:      mux,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	serverErrors := make(chan error, 1)
	go func() {
		log.Info("ferdinand api listening", "host", cfg.Host)
		serverErrors <- srv.ListenAndServe()
	}()

	// // Run immediately on startup, then on the ticker.
	// if err := scraper.Sync(ctx, cfg.StartDate, cfg.EndDate); err != nil {
	// 	log.Error("sync", "error", err)
	// }

	if err := scraper.SyncAthletes(ctx); err != nil {
		log.Error("sync", "error", err)
	}

	ticker := time.NewTicker(cfg.ScrapeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// if err := scraper.Sync(ctx, cfg.StartDate, cfg.EndDate); err != nil {
			// 	log.Error("sync", "error", err)
			// }
			if err := scraper.SyncAthletes(ctx); err != nil {
				log.Error("sync", "error", err)
			}
		case err := <-serverErrors:
			return fmt.Errorf("server error: %w", err)
		case <-ctx.Done():
			fmt.Println()
			log.Info("shutdown", "reason", ctx.Err())
			shutCtx, shutCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
			defer shutCancel()
			if err := srv.Shutdown(shutCtx); err != nil {
				srv.Close()
				return fmt.Errorf("graceful shutdown: %w", err)
			}
			return nil
		}
	}
}
