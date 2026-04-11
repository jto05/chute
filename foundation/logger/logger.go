// Package logger provides a thin wrapper around a structured logger.
// Swap the underlying library here without touching the rest of the codebase.
package logger

import (
	"log/slog"
	"os"
)

// Logger is the application logger type.
type Logger struct {
	*slog.Logger
}

// New constructs a Logger writing JSON to stdout.
func New() *Logger {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	return &Logger{slog.New(h)}
}
