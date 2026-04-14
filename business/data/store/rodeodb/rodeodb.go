// Package rodeodb implements file-tree storage for raw rodeo and athlete JSON.
// Layout on disk:
//
//	data/results/
//	  rodeo/
//	    <rodeoID>/
//	      results.json
//	  athletes/
//	    <contestantID>/
//	      athlete.json
//
// This can be swapped for a database-backed implementation later by changing
// what New() returns — the caller only sees *Store.
package rodeodb

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/am29/ferdinand/app/domain/prorodeoapp"
)

// Store manages rodeo and athlete files under a base directory.
type Store struct {
	baseDir    string
	athleteDir string
}

/*

schema:
- contestants one table
	- per contestant add timestamp
	- later on, sync via timestamp (recently changed)

*/

// New constructs a Store rooted at baseDir.
// The athlete directory is derived as a sibling of baseDir named "athletes".
func New(baseDir string) *Store {
	return &Store{
		baseDir:    baseDir,
		athleteDir: filepath.Join(filepath.Dir(baseDir), "athletes"),
	}
}

// Exists reports whether results for rodeoID have already been stored.
func (s *Store) Exists(rodeoID int) bool {
	_, err := os.Stat(s.filePath(rodeoID))
	return err == nil
}

// Save writes raw JSON bytes for rodeoID, creating the directory if needed.
func (s *Store) Save(rodeoID int, data []byte) error {
	dir := s.dirPath(rodeoID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dir %s: %w", dir, err)
	}
	if err := os.WriteFile(s.filePath(rodeoID), data, 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

// Load reads the raw JSON bytes for rodeoID.
func (s *Store) Load(rodeoID int) ([]byte, error) {
	data, err := os.ReadFile(s.filePath(rodeoID))
	if err != nil {
		return nil, fmt.Errorf("rodeo %d not found: %w", rodeoID, err)
	}
	return data, nil
}

// List returns the IDs of all stored rodeos.
func (s *Store) List() ([]int, error) {
	entries, err := os.ReadDir(s.baseDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read data dir: %w", err)
	}

	var ids []int
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (s *Store) dirPath(rodeoID int) string {
	return filepath.Join(s.baseDir, strconv.Itoa(rodeoID))
}

func (s *Store) filePath(rodeoID int) string {
	return filepath.Join(s.dirPath(rodeoID), "results.json")
}

// =============================================================================
// Athletes

// AthleteExists reports whether a profile for contestantID has already been stored.
func (s *Store) AthleteExists(contestantID int) bool {
	_, err := os.Stat(s.athleteFilePath(contestantID))
	return err == nil
}

// SaveAthlete marshals the Athlete to JSON and writes it to disk.
func (s *Store) SaveAthlete(athlete prorodeoapp.Athlete) error {
	data, err := json.MarshalIndent(athlete, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal athlete %d: %w", athlete.ContestantID, err)
	}

	dir := s.athleteDirPath(athlete.ContestantID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dir %s: %w", dir, err)
	}

	if err := os.WriteFile(s.athleteFilePath(athlete.ContestantID), data, 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

// LoadAthlete reads and unmarshals the stored profile for contestantID.
func (s *Store) LoadAthlete(contestantID int) (prorodeoapp.Athlete, error) {
	data, err := os.ReadFile(s.athleteFilePath(contestantID))
	if err != nil {
		return prorodeoapp.Athlete{}, fmt.Errorf("athlete %d not found: %w", contestantID, err)
	}

	var athlete prorodeoapp.Athlete
	if err := json.Unmarshal(data, &athlete); err != nil {
		return prorodeoapp.Athlete{}, fmt.Errorf("unmarshal athlete %d: %w", contestantID, err)
	}
	return athlete, nil
}

// ListAthletes returns the IDs of all stored athletes.
func (s *Store) ListAthletes() ([]int, error) {
	entries, err := os.ReadDir(s.athleteDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read athlete dir: %w", err)
	}

	var ids []int
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (s *Store) athleteDirPath(contestantID int) string {
	return filepath.Join(s.athleteDir, strconv.Itoa(contestantID))
}

func (s *Store) athleteFilePath(contestantID int) string {
	return filepath.Join(s.athleteDirPath(contestantID), "athlete.json")
}
