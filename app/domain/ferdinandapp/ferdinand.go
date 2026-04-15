// Package ferdinandapp exposes a read API so local sheet instances can sync
// contestant data from the central ferdinand server.
package ferdinandapp

import (
	"net/http"
	"strconv"

	"github.com/jto05/chute/business/store"
	"github.com/jto05/chute/foundation/logger"
	"github.com/jto05/chute/foundation/web"
)

// App holds the dependencies for the ferdinand HTTP API.
type App struct {
	log   *logger.Logger
	store *store.Store
}

// New constructs an App.
func New(log *logger.Logger, store *store.Store) *App {
	return &App{log: log, store: store}
}

// Routes registers all ferdinand HTTP routes on the provided mux.
func (a *App) Routes(mux *web.Mux) {
	mux.HandleFunc("GET /api/athletes", a.athletes)
}

// athletes returns all contestants scraped after the given Unix timestamp.
// Query param: since=<unix_timestamp> (defaults to 0, i.e. all athletes).
//
// GET /api/athletes?since=1713000000
func (a *App) athletes(w http.ResponseWriter, r *http.Request) {
	var since int64
	if s := r.URL.Query().Get("since"); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			web.RespondError(w, err, http.StatusBadRequest)
			return
		}
		since = v
	}

	results, err := a.store.ListAthletesSince(r.Context(), since)
	if err != nil {
		a.log.Error("list athletes since", "since", since, "error", err)
		web.RespondError(w, err, http.StatusInternalServerError)
		return
	}

	web.Respond(w, results, http.StatusOK)
}
