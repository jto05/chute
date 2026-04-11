// Package rodeodb implements file-tree storage for raw rodeo result JSON.
// Layout on disk:
//
//	<dataDir>/
//	  <rodeoID>/
//	    results.json
//
// This can be swapped for a database-backed implementation later by changing
// what New() returns — the caller only sees *Store.
package rodeodb

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// Store manages rodeo result files under a base directory.
type Store struct {
	baseDir string
}

// New constructs a Store rooted at baseDir.
func New(baseDir string) *Store {
	return &Store{baseDir: baseDir}
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
