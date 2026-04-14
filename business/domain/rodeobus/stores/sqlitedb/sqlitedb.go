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
      notes              TEXT     NOT NULL DEFAULT '',
      scraped_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
  );
  `
	_, err := db.Exec(schema)
	return err
}

// ==================================
// Athletes

// SaveAthlete inserts an athlete into existing Store.
func (s *Store) SaveAthlete(ctx context.Context, athlete prorodeoapp.Athlete) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO contestants
			(id, first_name, last_name, nick_name, hometown, photo_url,
			 birth_date, age, total_earnings, year_earnings, world_titles,
			 nfr_qualifications, date_joined, event_types, biography_text, is_active,
			 scraped_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
			first_name = excluded.first_name,
			last_name = excluded.last_name,
			nick_name = excluded.nick_name,
			hometown = excluded.hometown,
			photo_url = excluded.photo_url,
			birth_date = excluded.birth_date,
			age = excluded.age,
			total_earnings = excluded.total_earnings,
			year_earnings = excluded.year_earnings,
			world_titles = excluded.world_titles,
			nfr_qualifications = excluded.nfr_qualifications,
			date_joined = excluded.date_joined,
			event_types = excluded.event_types,
			biography_text = excluded.biography_text,
			is_active = excluded.is_active,
			scraped_at = excluded.scraped_at`,
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

// SaveAthleteBatch inserts a given slice of Athletes into existing Store.
func (s *Store) SaveAthleteBatch(ctx context.Context, batch []prorodeoapp.Athlete) error {
	// create transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// prepare upsert statement
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO contestants
			(id, first_name, last_name, nick_name, hometown, photo_url,
			 birth_date, age, total_earnings, year_earnings, world_titles,
			 nfr_qualifications, date_joined, event_types, biography_text, is_active,
			 scraped_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
			first_name = excluded.first_name,
			last_name = excluded.last_name,
			nick_name = excluded.nick_name,
			hometown = excluded.hometown,
			photo_url = excluded.photo_url,
			birth_date = excluded.birth_date,
			age = excluded.age,
			total_earnings = excluded.total_earnings,
			year_earnings = excluded.year_earnings,
			world_titles = excluded.world_titles,
			nfr_qualifications = excluded.nfr_qualifications,
			date_joined = excluded.date_joined,
			event_types = excluded.event_types,
			biography_text = excluded.biography_text,
			is_active = excluded.is_active,
			scraped_at = excluded.scraped_at`)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	// calculate time outside of loop so all times are synchronized in batch
	now := time.Now().UTC().Format(time.RFC3339)

	for _, athlete := range batch {
		_, err := stmt.ExecContext(ctx,
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
			now,
		)
		if err != nil {
			return fmt.Errorf("insert athlete %d: %w", athlete.ContestantID, err)
		}
	}

	// commit transaction to database
	return tx.Commit()
}
