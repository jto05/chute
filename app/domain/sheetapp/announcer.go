// Package sheetapp serves the contestant search UI and generates cheat-sheet PDFs.
package sheetapp

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/jto05/chute/business/domain/rodeobus/stores/sqlitedb"
	"github.com/jto05/chute/foundation/logger"
	"github.com/jto05/chute/foundation/pdf"
	"github.com/jto05/chute/foundation/web"
)

// App holds the dependencies for sheet PDF generation.
type App struct {
	log   *logger.Logger
	store *sqlitedb.Store
	tmpl  *template.Template
}

// New constructs an App. tmpl is the parsed HTML template set.
func New(log *logger.Logger, store *sqlitedb.Store, tmpl *template.Template) *App {
	return &App{log: log, store: store, tmpl: tmpl}
}

// Routes registers all sheet HTTP routes on the provided mux.
func (a *App) Routes(mux *web.Mux) {
	mux.HandleFunc("GET /", a.index)
	mux.HandleFunc("GET /api/contestants/search", a.searchContestants)
	mux.HandleFunc("GET /api/contestants/{id}", a.contestantDetail)
	mux.HandleFunc("POST /api/sheet/pdf", a.generatePDF)
}

// index serves the main HTML page.
func (a *App) index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.tmpl.ExecuteTemplate(w, "index.html", nil); err != nil {
		a.log.Error("render index", "error", err)
	}
}

// searchContestants returns an HTML partial of matching contestants (HTMX target).
func (a *App) searchContestants(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if len(q) < 2 {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<p class="hint">Type at least 2 characters to search.</p>`)
		return
	}

	results, err := a.store.SearchAthletes(r.Context(), q)
	if err != nil {
		a.log.Error("search athletes", "q", q, "error", err)
		http.Error(w, "search failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.tmpl.ExecuteTemplate(w, "search_results.html", results); err != nil {
		a.log.Error("render search results", "error", err)
	}
}

// contestantDetail returns the full-info HTML partial for one contestant.
func (a *App) contestantDetail(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	d, err := a.store.LoadAthleteDetail(r.Context(), id)
	if err != nil {
		a.log.Error("load athlete detail", "id", id, "error", err)
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	view := buildDetailView(d)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.tmpl.ExecuteTemplate(w, "contestant_detail.html", view); err != nil {
		a.log.Error("render contestant detail", "error", err)
	}
}

// generatePDF accepts a JSON roster and returns a PDF.
func (a *App) generatePDF(w http.ResponseWriter, r *http.Request) {
	var req PDFRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.RespondError(w, fmt.Errorf("bad request: %w", err), http.StatusBadRequest)
		return
	}

	if len(req.Contestants) == 0 {
		web.RespondError(w, fmt.Errorf("no contestants selected"), http.StatusBadRequest)
		return
	}

	athletes := make([]sqlitedb.AthleteResult, 0, len(req.Contestants))
	notes := make(map[int]string, len(req.Contestants))
	for _, entry := range req.Contestants {
		id, err := strconv.Atoi(entry.ID)
		if err != nil {
			continue
		}
		athlete, err := a.store.LoadAthlete(r.Context(), id)
		if err != nil {
			a.log.Error("load athlete", "id", id, "error", err)
			continue
		}
		athletes = append(athletes, athlete)
		if entry.Notes != "" {
			notes[id] = entry.Notes
		}
	}

	data, err := pdf.RenderRoster(req.RodeoName, req.RodeoDate, athletes, notes)
	if err != nil {
		web.RespondError(w, err, http.StatusInternalServerError)
		return
	}

	filename := "chute-roster.pdf"
	if req.RodeoName != "" {
		filename = req.RodeoName + ".pdf"
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, filename))
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// ── View model ────────────────────────────────────────────────────────────────

// ContestantDetailView is a template-ready view of AthleteDetail.
type ContestantDetailView struct {
	ID                int
	FullName          string
	NickName          string
	Hometown          string
	PhotoURL          string
	Age               string
	TotalEarnings     string
	YearEarnings      string
	WorldTitles       string
	NFRQualifications string
	EventTypes        []string
	EventTypesRaw     string // comma-joined, used in JS data attribute
	BiographyText     string
}

func buildDetailView(d sqlitedb.AthleteDetail) ContestantDetailView {
	v := ContestantDetailView{
		ID:                d.ContestantID,
		FullName:          d.FirstName + " " + d.LastName,
		TotalEarnings:     "$" + fmtMoney(d.TotalEarnings),
		YearEarnings:      "$" + fmtMoney(d.YearEarnings),
		Age:               derefInt(d.Age),
		WorldTitles:       derefInt(d.WorldTitles),
		NFRQualifications: derefInt(d.NFRQualifications),
		EventTypes:        strings.Split(d.EventTypes, ","),
		EventTypesRaw:     d.EventTypes,
	}
	if d.NickName != nil && *d.NickName != "" {
		v.NickName = *d.NickName
	}
	if d.Hometown != nil {
		v.Hometown = *d.Hometown
	}
	if d.PhotoURL != nil {
		v.PhotoURL = *d.PhotoURL
	}
	if d.BiographyText != nil {
		v.BiographyText = *d.BiographyText
	}
	return v
}

func derefInt(n *int64) string {
	if n != nil {
		return strconv.FormatInt(*n, 10)
	}
	return "–"
}

func fmtMoney(v float64) string {
	s := fmt.Sprintf("%.0f", v)
	out := make([]byte, 0, len(s)+4)
	for i := range s {
		pos := len(s) - i
		if i > 0 && pos%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, s[i])
	}
	return string(out)
}
