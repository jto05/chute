// Package web provides HTTP primitives shared across services.
package web

import (
	"encoding/json"
	"net/http"

	"github.com/jto05/chute/foundation/logger"
)

// Mux wraps http.ServeMux with common middleware applied at construction.
type Mux struct {
	*http.ServeMux
	log *logger.Logger
}

// NewMux constructs a Mux with logging middleware.
func NewMux(log *logger.Logger) *Mux {
	return &Mux{ServeMux: http.NewServeMux(), log: log}
}

// Respond encodes v as JSON and writes it with the given status code.
func Respond(w http.ResponseWriter, v any, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// nothing useful we can do at this point
		return
	}
}

// RespondError writes a JSON error body.
func RespondError(w http.ResponseWriter, err error, statusCode int) {
	Respond(w, map[string]string{"error": err.Error()}, statusCode)
}
