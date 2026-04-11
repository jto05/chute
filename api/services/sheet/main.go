// sheet is an HTTP service that serves rodeo data and generates
// sheet PDFs on demand.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jto05/chute/app/domain/sheetapp"
	"github.com/jto05/chute/business/data/store/rodeodb"
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
		Host            string
		ReadTimeout     time.Duration
		WriteTimeout    time.Duration
		IdleTimeout     time.Duration
		ShutdownTimeout time.Duration
		DataDir         string
	}{
		Host:            "0.0.0.0:3000",
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    10 * time.Second,
		IdleTimeout:     120 * time.Second,
		ShutdownTimeout: 15 * time.Second,
		DataDir:         "data/results",
	}

	store := rodeodb.New(cfg.DataDir)
	app := sheetapp.New(log, store)

	mux := web.NewMux(log)
	app.Routes(mux)

	srv := http.Server{
		Addr:         cfg.Host,
		Handler:      mux,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	serverErrors := make(chan error, 1)
	go func() {
		log.Info("startup", "host", cfg.Host)
		serverErrors <- srv.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		log.Info("shutdown", "signal", sig)
		ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			srv.Close()
			return fmt.Errorf("graceful shutdown: %w", err)
		}
	}

	return nil
}
