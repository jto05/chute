// Package sheetapp assembles rodeo data into sheet-ready PDFs.
package sheetapp

import (
	"fmt"
	"net/http"

	"github.com/jto05/chute/business/data/store/rodeodb"
	"github.com/jto05/chute/foundation/logger"
	"github.com/jto05/chute/foundation/pdf"
	"github.com/jto05/chute/foundation/web"
	"github.com/am29/ferdinand/app/domain/prorodeoapp"
)

// App holds the dependencies for sheet PDF generation.
type App struct {
	log   *logger.Logger
	store *rodeodb.Store
}

// New constructs an App.
func New(log *logger.Logger, store *rodeodb.Store) *App {
	return &App{log: log, store: store}
}

// Routes registers all sheet HTTP routes on the provided mux.
func (a *App) Routes(mux *web.Mux) {
	mux.HandleFunc("GET /rodeos", a.listRodeos)
	mux.HandleFunc("GET /rodeos/{id}/pdf", a.generatePDF)
}

// listRodeos returns the IDs of all stored rodeos.
func (a *App) listRodeos(w http.ResponseWriter, r *http.Request) {
	ids, err := a.store.List()
	if err != nil {
		web.RespondError(w, err, http.StatusInternalServerError)
		return
	}
	web.Respond(w, ids, http.StatusOK)
}

// generatePDF renders an sheet sheet for the requested rodeo.
func (a *App) generatePDF(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		web.RespondError(w, fmt.Errorf("invalid rodeo id %q", idStr), http.StatusBadRequest)
		return
	}

	raw, err := a.store.Load(id)
	if err != nil {
		web.RespondError(w, err, http.StatusNotFound)
		return
	}

	results, err := prorodeoapp.ParseResults(raw)
	if err != nil {
		web.RespondError(w, err, http.StatusInternalServerError)
		return
	}

	data, err := pdf.RenderSheetSheet(results)
	if err != nil {
		web.RespondError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="rodeo-%d.pdf"`, id))
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
