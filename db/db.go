package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"ustawka/sejm"

	_ "github.com/mattn/go-sqlite3"
)

// DB represents the database connection
type DB struct {
	*sql.DB
}

// New creates a new database connection
func New(dbPath string) (*DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	if err := createTables(db); err != nil {
		return nil, err
	}

	return &DB{db}, nil
}

// createTables creates the necessary tables if they don't exist
func createTables(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS acts (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			status TEXT NOT NULL,
			published TEXT NOT NULL,
			position INTEGER NOT NULL,
			year INTEGER NOT NULL,
			type TEXT NOT NULL,
			address TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS act_details (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			status TEXT NOT NULL,
			published TEXT NOT NULL,
			type TEXT NOT NULL,
			address TEXT NOT NULL,
			display_address TEXT NOT NULL,
			position INTEGER NOT NULL,
			year INTEGER NOT NULL,
			announcement_date TEXT,
			change_date TEXT,
			publisher TEXT,
			text_html BOOLEAN,
			text_pdf BOOLEAN,
			volume INTEGER,
			entry_into_force TEXT,
			in_force TEXT,
			keywords TEXT,
			keywords_names TEXT,
			released_by TEXT,
			texts TEXT,
			act_references TEXT,
			authorized_body TEXT,
			directives TEXT,
			obligated TEXT,
			previous_title TEXT,
			prints TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_acts_year ON acts(year)`,
		`CREATE INDEX IF NOT EXISTS idx_acts_status ON acts(status)`,
		`CREATE TRIGGER IF NOT EXISTS update_acts_timestamp 
		AFTER UPDATE ON acts
		BEGIN
			UPDATE acts SET updated_at = datetime('now') WHERE id = NEW.id;
		END`,
		`CREATE TRIGGER IF NOT EXISTS update_act_details_timestamp 
		AFTER UPDATE ON act_details
		BEGIN
			UPDATE act_details SET updated_at = datetime('now') WHERE id = NEW.id;
		END`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}

	return nil
}

// GetActs retrieves acts for a specific year from the cache
func (db *DB) GetActs(ctx context.Context, year int) ([]sejm.Act, error) {
	query := `SELECT id, title, status, published, position, year, type, address 
			  FROM acts WHERE year = ? ORDER BY position`

	rows, err := db.QueryContext(ctx, query, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var acts []sejm.Act
	for rows.Next() {
		var act sejm.Act
		if err := rows.Scan(
			&act.ID,
			&act.Title,
			&act.Status,
			&act.Published,
			&act.Position,
			&act.Year,
			&act.Type,
			&act.Address,
		); err != nil {
			return nil, err
		}
		acts = append(acts, act)
	}

	return acts, rows.Err()
}

// StoreActs stores acts for a specific year in the cache
func (db *DB) StoreActs(ctx context.Context, year int, acts []sejm.Act) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete existing acts for the year
	if _, err := tx.ExecContext(ctx, "DELETE FROM acts WHERE year = ?", year); err != nil {
		return err
	}

	// Insert new acts
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO acts (id, title, status, published, position, year, type, address, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, act := range acts {
		if _, err := stmt.ExecContext(ctx,
			act.ID,
			act.Title,
			act.Status,
			act.Published,
			act.Position,
			act.Year,
			act.Type,
			act.Address,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetActDetails retrieves act details from the cache
func (db *DB) GetActDetails(ctx context.Context, actID string) (*sejm.ActDetails, error) {
	query := `SELECT id, title, status, published, type, address, display_address, position, year,
			  announcement_date, change_date, publisher, text_html, text_pdf, volume,
			  entry_into_force, in_force, keywords, keywords_names, released_by, texts,
			  act_references, authorized_body, directives, obligated, previous_title, prints
			  FROM act_details WHERE id = ?`

	var details sejm.ActDetails
	var keywords, keywordsNames, releasedBy, texts, actReferences, authorizedBody, directives, obligated, previousTitle, prints string

	err := db.QueryRowContext(ctx, query, actID).Scan(
		&details.ID,
		&details.Title,
		&details.Status,
		&details.Published,
		&details.Type,
		&details.Address,
		&details.DisplayAddress,
		&details.Position,
		&details.Year,
		&details.AnnouncementDate,
		&details.ChangeDate,
		&details.Publisher,
		&details.TextHTML,
		&details.TextPDF,
		&details.Volume,
		&details.EntryIntoForce,
		&details.InForce,
		&keywords,
		&keywordsNames,
		&releasedBy,
		&texts,
		&actReferences,
		&authorizedBody,
		&directives,
		&obligated,
		&previousTitle,
		&prints,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Parse JSON strings into slices
	if err := json.Unmarshal([]byte(keywords), &details.Keywords); err != nil {
		return nil, fmt.Errorf("failed to parse keywords: %w", err)
	}
	if err := json.Unmarshal([]byte(keywordsNames), &details.KeywordsNames); err != nil {
		return nil, fmt.Errorf("failed to parse keywords names: %w", err)
	}
	if err := json.Unmarshal([]byte(releasedBy), &details.ReleasedBy); err != nil {
		return nil, fmt.Errorf("failed to parse released by: %w", err)
	}
	if err := json.Unmarshal([]byte(texts), &details.Texts); err != nil {
		return nil, fmt.Errorf("failed to parse texts: %w", err)
	}
	if err := json.Unmarshal([]byte(actReferences), &details.References); err != nil {
		return nil, fmt.Errorf("failed to parse references: %w", err)
	}
	if err := json.Unmarshal([]byte(authorizedBody), &details.AuthorizedBody); err != nil {
		return nil, fmt.Errorf("failed to parse authorized body: %w", err)
	}
	if err := json.Unmarshal([]byte(directives), &details.Directives); err != nil {
		return nil, fmt.Errorf("failed to parse directives: %w", err)
	}
	if err := json.Unmarshal([]byte(obligated), &details.Obligated); err != nil {
		return nil, fmt.Errorf("failed to parse obligated: %w", err)
	}
	if err := json.Unmarshal([]byte(previousTitle), &details.PreviousTitle); err != nil {
		return nil, fmt.Errorf("failed to parse previous title: %w", err)
	}
	if err := json.Unmarshal([]byte(prints), &details.Prints); err != nil {
		return nil, fmt.Errorf("failed to parse prints: %w", err)
	}

	return &details, nil
}

// StoreActDetails stores act details in the cache
func (db *DB) StoreActDetails(ctx context.Context, details *sejm.ActDetails) error {
	// Convert slices to JSON strings
	keywords, err := json.Marshal(details.Keywords)
	if err != nil {
		return fmt.Errorf("failed to marshal keywords: %w", err)
	}
	keywordsNames, err := json.Marshal(details.KeywordsNames)
	if err != nil {
		return fmt.Errorf("failed to marshal keywords names: %w", err)
	}
	releasedBy, err := json.Marshal(details.ReleasedBy)
	if err != nil {
		return fmt.Errorf("failed to marshal released by: %w", err)
	}
	texts, err := json.Marshal(details.Texts)
	if err != nil {
		return fmt.Errorf("failed to marshal texts: %w", err)
	}
	actReferences, err := json.Marshal(details.References)
	if err != nil {
		return fmt.Errorf("failed to marshal references: %w", err)
	}
	authorizedBody, err := json.Marshal(details.AuthorizedBody)
	if err != nil {
		return fmt.Errorf("failed to marshal authorized body: %w", err)
	}
	directives, err := json.Marshal(details.Directives)
	if err != nil {
		return fmt.Errorf("failed to marshal directives: %w", err)
	}
	obligated, err := json.Marshal(details.Obligated)
	if err != nil {
		return fmt.Errorf("failed to marshal obligated: %w", err)
	}
	previousTitle, err := json.Marshal(details.PreviousTitle)
	if err != nil {
		return fmt.Errorf("failed to marshal previous title: %w", err)
	}
	prints, err := json.Marshal(details.Prints)
	if err != nil {
		return fmt.Errorf("failed to marshal prints: %w", err)
	}

	query := `
		INSERT INTO act_details (
			id, title, status, published, type, address, display_address, position, year,
			announcement_date, change_date, publisher, text_html, text_pdf, volume,
			entry_into_force, in_force, keywords, keywords_names, released_by, texts,
			act_references, authorized_body, directives, obligated, previous_title, prints,
			updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			status = excluded.status,
			published = excluded.published,
			type = excluded.type,
			address = excluded.address,
			display_address = excluded.display_address,
			position = excluded.position,
			year = excluded.year,
			announcement_date = excluded.announcement_date,
			change_date = excluded.change_date,
			publisher = excluded.publisher,
			text_html = excluded.text_html,
			text_pdf = excluded.text_pdf,
			volume = excluded.volume,
			entry_into_force = excluded.entry_into_force,
			in_force = excluded.in_force,
			keywords = excluded.keywords,
			keywords_names = excluded.keywords_names,
			released_by = excluded.released_by,
			texts = excluded.texts,
			act_references = excluded.act_references,
			authorized_body = excluded.authorized_body,
			directives = excluded.directives,
			obligated = excluded.obligated,
			previous_title = excluded.previous_title,
			prints = excluded.prints,
			updated_at = datetime('now')
	`

	_, err = db.ExecContext(ctx, query,
		details.ID,
		details.Title,
		details.Status,
		details.Published,
		details.Type,
		details.Address,
		details.DisplayAddress,
		details.Position,
		details.Year,
		details.AnnouncementDate,
		details.ChangeDate,
		details.Publisher,
		details.TextHTML,
		details.TextPDF,
		details.Volume,
		details.EntryIntoForce,
		details.InForce,
		string(keywords),
		string(keywordsNames),
		string(releasedBy),
		string(texts),
		string(actReferences),
		string(authorizedBody),
		string(directives),
		string(obligated),
		string(previousTitle),
		string(prints),
	)
	return err
}

// GetCacheAge returns the age of the cache for a specific year
func (db *DB) GetCacheAge(ctx context.Context, year int) (time.Duration, error) {
	var updatedAt sql.NullString
	err := db.QueryRowContext(ctx,
		"SELECT strftime('%Y-%m-%d %H:%M:%f', MAX(updated_at)) FROM acts WHERE year = ?",
		year,
	).Scan(&updatedAt)
	if err == sql.ErrNoRows || !updatedAt.Valid {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	t, err := time.Parse("2006-01-02 15:04:05.999999999", updatedAt.String)
	if err != nil {
		return 0, err
	}

	return time.Since(t), nil
}
