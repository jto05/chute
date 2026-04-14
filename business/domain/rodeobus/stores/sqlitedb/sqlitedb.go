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

// ==================================
// Search

// AthleteResult is a lightweight view of a contestant used for search results.
type AthleteResult struct {
	ContestantID  int
	FirstName     string
	LastName      string
	Hometown      string
	EventTypes    string
	TotalEarnings float64
	YearEarnings  float64
}

// SearchAthletes returns contestants matching all words in the query against the
// full name (first + last). Searching "stetson wright" requires both tokens to
// appear somewhere in the full name, so multi-word searches work correctly.
func (s *Store) SearchAthletes(ctx context.Context, q string) ([]AthleteResult, error) {
	tokens := strings.Fields(q)
	if len(tokens) == 0 {
		return nil, nil
	}

	// Build: WHERE (first_name||' '||last_name) LIKE ? [AND ...] for each token.
	clauses := make([]string, len(tokens))
	args := make([]any, len(tokens))
	for i, t := range tokens {
		clauses[i] = "(first_name || ' ' || last_name) LIKE ?"
		args[i] = "%" + t + "%"
	}

	query := `
		SELECT id, first_name, last_name, hometown, event_types, total_earnings, year_earnings
		FROM contestants
		WHERE ` + strings.Join(clauses, " AND ") + `
		ORDER BY last_name, first_name
		LIMIT 50`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("search athletes: %w", err)
	}
	defer rows.Close()

	var results []AthleteResult
	for rows.Next() {
		var r AthleteResult
		if err := rows.Scan(&r.ContestantID, &r.FirstName, &r.LastName, &r.Hometown, &r.EventTypes, &r.TotalEarnings, &r.YearEarnings); err != nil {
			return nil, fmt.Errorf("scan athlete: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// LoadAthlete returns the lightweight view for a single contestant by ID.
func (s *Store) LoadAthlete(ctx context.Context, id int) (AthleteResult, error) {
	var r AthleteResult
	err := s.db.QueryRowContext(ctx, `
		SELECT id, first_name, last_name, hometown, event_types, total_earnings, year_earnings
		FROM contestants WHERE id = ?`, id,
	).Scan(&r.ContestantID, &r.FirstName, &r.LastName, &r.Hometown, &r.EventTypes, &r.TotalEarnings, &r.YearEarnings)
	if err != nil {
		return AthleteResult{}, fmt.Errorf("load athlete %d: %w", id, err)
	}
	return r, nil
}

// AthleteDetail is the full contestant row, used for the detail panel.
// Nullable columns are represented as pointers so callers need no sql import.
type AthleteDetail struct {
	ContestantID      int
	FirstName         string
	LastName          string
	NickName          *string
	Hometown          *string
	PhotoURL          *string
	Age               *int64
	TotalEarnings     float64
	YearEarnings      float64
	WorldTitles       *int64
	NFRQualifications *int64
	EventTypes        string
	BiographyText     *string
}

// LoadAthleteDetail returns all display fields for a single contestant.
func (s *Store) LoadAthleteDetail(ctx context.Context, id int) (AthleteDetail, error) {
	// Scan nullables into sql.Null* locally, then convert to pointers.
	var (
		nickName          sql.NullString
		hometown          sql.NullString
		photoURL          sql.NullString
		age               sql.NullInt64
		worldTitles       sql.NullInt64
		nfrQualifications sql.NullInt64
		biographyText     sql.NullString
		d                 AthleteDetail
	)
	err := s.db.QueryRowContext(ctx, `
		SELECT id, first_name, last_name, nick_name, hometown, photo_url,
		       age, total_earnings, year_earnings, world_titles, nfr_qualifications,
		       event_types, biography_text
		FROM contestants WHERE id = ?`, id,
	).Scan(
		&d.ContestantID, &d.FirstName, &d.LastName,
		&nickName, &hometown, &photoURL,
		&age, &d.TotalEarnings, &d.YearEarnings,
		&worldTitles, &nfrQualifications,
		&d.EventTypes, &biographyText,
	)
	if err != nil {
		return AthleteDetail{}, fmt.Errorf("load athlete detail %d: %w", id, err)
	}

	if nickName.Valid {
		d.NickName = &nickName.String
	}
	if hometown.Valid {
		d.Hometown = &hometown.String
	}
	if photoURL.Valid {
		d.PhotoURL = &photoURL.String
	}
	if age.Valid {
		d.Age = &age.Int64
	}
	if worldTitles.Valid {
		d.WorldTitles = &worldTitles.Int64
	}
	if nfrQualifications.Valid {
		d.NFRQualifications = &nfrQualifications.Int64
	}
	if biographyText.Valid {
		d.BiographyText = &biographyText.String
	}
	return d, nil
}
