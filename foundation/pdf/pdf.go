// Package pdf renders contestant cheat-sheet PDFs.
package pdf

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/go-pdf/fpdf"
	"github.com/jto05/chute/business/store"
)

const (
	pageW  = 210.0 // A4 mm
	margin = 12.0
	col    = pageW - margin*2
)

// RenderRoster produces a PDF cheat sheet for the given roster.
// notes maps ContestantID → freeform text to print at the bottom of each card.
func RenderRoster(rodeoName, rodeoDate string, athletes []store.AthleteResult, notes map[int]string) ([]byte, error) {
	f := fpdf.New("P", "mm", "A4", "")
	f.SetMargins(margin, margin, margin)
	f.SetAutoPageBreak(true, margin)
	f.AddPage()

	// ── Header ──────────────────────────────────────────────────────────────
	f.SetFont("Helvetica", "B", 20)
	f.SetFillColor(30, 30, 30)
	f.SetTextColor(255, 255, 255)
	f.CellFormat(col, 12, "CHUTE — ANNOUNCER SHEET", "0", 1, "C", true, 0, "")

	f.SetFont("Helvetica", "", 11)
	f.SetFillColor(60, 60, 60)
	title := rodeoName
	if rodeoDate != "" {
		title = fmt.Sprintf("%s  |  %s", rodeoName, rodeoDate)
	}
	f.CellFormat(col, 8, title, "0", 1, "C", true, 0, "")
	f.Ln(4)

	// ── Group by event type ──────────────────────────────────────────────────
	grouped := groupByEvent(athletes)
	for _, event := range orderedEvents(grouped) {
		group := grouped[event]

		// Event band
		f.SetFont("Helvetica", "B", 11)
		f.SetFillColor(200, 200, 200)
		f.SetTextColor(30, 30, 30)
		f.CellFormat(col, 7, strings.ToUpper(event), "0", 1, "L", true, 0, "")
		f.Ln(1)

		for _, a := range group {
			renderContestantCard(f, a, notes[a.ContestantID])
		}
		f.Ln(3)
	}

	// Write to buffer
	var buf bytes.Buffer
	if err := f.Output(&buf); err != nil {
		return nil, fmt.Errorf("pdf output: %w", err)
	}
	return buf.Bytes(), nil
}

// renderContestantCard draws a single contestant block.
func renderContestantCard(f *fpdf.Fpdf, a store.AthleteResult, notes string) {
	f.SetFillColor(250, 250, 250)
	f.SetDrawColor(220, 220, 220)
	f.SetTextColor(20, 20, 20)

	// Name row
	f.SetFont("Helvetica", "B", 13)
	name := fmt.Sprintf("%s %s", a.FirstName, a.LastName)
	f.CellFormat(col, 7, name, "LRT", 1, "L", true, 0, "")

	// Hometown + earnings row
	f.SetFont("Helvetica", "", 9)
	hometown := a.Hometown
	if hometown == "" {
		hometown = "–"
	}
	earnings := fmt.Sprintf("Career: $%s  |  Year: $%s",
		formatMoney(a.TotalEarnings), formatMoney(a.YearEarnings))
	f.CellFormat(col/2, 6, hometown, "L", 0, "L", true, 0, "")
	f.CellFormat(col/2, 6, earnings, "R", 1, "R", true, 0, "")

	// Notes row (only if present)
	if notes != "" {
		f.SetFont("Helvetica", "I", 9)
		f.SetFillColor(245, 245, 220) // light yellow tint
		f.MultiCell(col, 5, notes, "LR", "L", true)
	}

	// Bottom border
	f.CellFormat(col, 1, "", "LRB", 1, "", true, 0, "")
	f.Ln(2)
}

// formatMoney formats a float64 as a comma-separated dollar string (no cents).
func formatMoney(v float64) string {
	s := fmt.Sprintf("%.0f", v)
	// insert commas
	out := make([]byte, 0, len(s)+4)
	for i, c := range s {
		pos := len(s) - i
		if i > 0 && pos%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(c))
	}
	return string(out)
}

// groupByEvent partitions athletes by their first listed event type.
func groupByEvent(athletes []store.AthleteResult) map[string][]store.AthleteResult {
	m := make(map[string][]store.AthleteResult)
	for _, a := range athletes {
		ev := firstEvent(a.EventTypes)
		m[ev] = append(m[ev], a)
	}
	return m
}

func firstEvent(eventTypes string) string {
	parts := strings.SplitN(eventTypes, ",", 2)
	ev := strings.TrimSpace(parts[0])
	if ev == "" {
		return "Other"
	}
	return ev
}

// orderedEvents returns event keys sorted by preferred rodeo order.
func orderedEvents(m map[string][]store.AthleteResult) []string {
	preferred := []string{
		"Bareback Riding", "Steer Wrestling", "Team Roping", "Saddle Bronc Riding",
		"Tie-Down Roping", "Barrel Racing", "Bull Riding", "Other",
	}
	var out []string
	seen := map[string]bool{}
	for _, k := range preferred {
		if _, ok := m[k]; ok {
			out = append(out, k)
			seen[k] = true
		}
	}
	// append any remaining events not in preferred list
	for k := range m {
		if !seen[k] {
			out = append(out, k)
		}
	}
	return out
}
