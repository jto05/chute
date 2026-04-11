// Package pdf provides primitives for rendering sheet sheets.
// Swap or extend the underlying PDF library here.
package pdf

import (
	"bytes"
	"fmt"

	"github.com/am29/ferdinand/app/domain/prorodeoapp"
)

// RenderSheetSheet produces a PDF byte slice from parsed rodeo results.
// TODO: implement layout with a real PDF library (e.g. github.com/signintech/gopdf).
func RenderSheetSheet(results []prorodeoapp.ScrapeResult) ([]byte, error) {
	if len(results) == 0 {
		return nil, fmt.Errorf("no results to render")
	}

	// Placeholder: emit a plain-text representation until a PDF library is chosen.
	var buf bytes.Buffer
	for _, r := range results {
		fmt.Fprintf(&buf, "=== %s (ID: %d) ===\n", r.RodeoName, r.RodeoID)
		fmt.Fprintf(&buf, "Location: %s, %s | Venue: %s\n\n", r.City, r.State, r.VenueName)

		for _, fp := range r.FirstPlaces {
			fmt.Fprintf(&buf, "Event: %s  Round: %s\n", fp.EventType, fp.Round)
			for _, c := range fp.Contestants {
				fmt.Fprintf(&buf, "  %s %s  (%s)\n", c.FirstName, c.LastName, c.Hometown)
			}
			for _, s := range fp.Stocks {
				fmt.Fprintf(&buf, "  Stock: %s [%s]\n", s.Name, s.Brand)
			}
			fmt.Fprintln(&buf)
		}
	}

	return buf.Bytes(), nil
}
