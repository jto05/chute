package sqlitedb

import (
	"context"
	"database/sql"
	// "encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/am29/ferdinand/app/domain/prorodeoapp"
	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("apply tables: %w", err)
	}

	return &Store{db: db}, nil
}

// applyTables
func createTables(db *sql.DB) error {
	/*
		NOTE:  make table for livestock, rodeos, and contestants (prorodeo)
			- rodeos empty for now :(
	*/
	const schema = `
  CREATE TABLE IF NOT EXISTS rodeos (
      id           INTEGER PRIMARY KEY,
      results_json TEXT     NOT NULL,
      scraped_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
  );

  CREATE TABLE IF NOT EXISTS contestants (
      id                 INTEGER PRIMARY KEY,
      first_name         TEXT     NOT NULL,
      last_name          TEXT     NOT NULL,
      nick_name          TEXT,
      hometown           TEXT,
      photo_url          TEXT,
      birth_date         DATETIME,
      age                INTEGER,
      total_earnings     REAL     NOT NULL DEFAULT 0,
      year_earnings      REAL     NOT NULL DEFAULT 0,
      world_titles       INTEGER,
      nfr_qualifications INTEGER,
      date_joined        DATETIME,
      event_types        TEXT,
      biography_text     TEXT,
      is_active          INTEGER  NOT NULL DEFAULT 1,
      scraped_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
  );
  `
	_, err := db.Exec(schema)
	return err
}

// ==================================
// Athletes

func (s *Store) SaveAthlete(ctx context.Context, athlete prorodeoapp.Athlete) error {
	_, err := s.db.ExecContext(ctx, `
						INSERT OR REPLACE INTO contestants
										(id, first_name, last_name, nick_name, hometown, photo_url,
										 birth_date, age, total_earnings, year_earnings, world_titles,
										 nfr_qualifications, date_joined, event_types, biography_text, is_active,
										 scraped_at)
						VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		athlete.ContestantID,
		athlete.FirstName,
		athlete.LastName,
		athlete.NickName,
		athlete.Hometown,
		athlete.PhotoURL,
		athlete.BirthDate.UTC().Format(time.RFC3339),
		athlete.Age,
		athlete.TotalEarnings,
		athlete.YearEarnings,
		athlete.WorldTitles,
		athlete.NFRQualifications,
		athlete.DateJoined.UTC().Format(time.RFC3339),
		strings.Join(athlete.EventTypes, ","),
		athlete.BiographyText,
		athlete.IsActive,
		time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("save athlete %d: %w", athlete.ContestantID, err)
	}
	return nil
}
