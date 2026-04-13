// Package rodeobus defines the core domain interface for rodeo and athlete data.
package rodeobus

import (
	"context"

	"github.com/am29/ferdinand/app/domain/prorodeoapp"
)

// Storer defines the storage operations required by the application layer.
// The underlying implementation (SQLite, file tree, etc.) is hidden behind this interface.
type Storer interface {
	// Rodeos
	RodeoExists(ctx context.Context, rodeoID int) bool
	SaveRodeo(ctx context.Context, rodeoID int, data []byte) error
	LoadRodeo(ctx context.Context, rodeoID int) ([]byte, error)
	ListRodeos(ctx context.Context) ([]int, error)

	// Athletes
	AthleteExists(ctx context.Context, contestantID int) bool
	SaveAthlete(ctx context.Context, athlete prorodeoapp.Athlete) error
	LoadAthlete(ctx context.Context, contestantID int) (prorodeoapp.Athlete, error)
	ListAthletes(ctx context.Context) ([]int, error)
}
