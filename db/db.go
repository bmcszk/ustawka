package db

import (
	"context"
	"database/sql"
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
	query := `SELECT id, title, status, published 
			  FROM act_details WHERE id = ?`

	var details sejm.ActDetails
	err := db.QueryRowContext(ctx, query, actID).Scan(
		&details.ID,
		&details.Title,
		&details.Status,
		&details.Published,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &details, nil
}

// StoreActDetails stores act details in the cache
func (db *DB) StoreActDetails(ctx context.Context, details *sejm.ActDetails) error {
	query := `
		INSERT INTO act_details (id, title, status, published, updated_at)
		VALUES (?, ?, ?, ?, datetime('now'))
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			status = excluded.status,
			published = excluded.published,
			updated_at = datetime('now')
	`

	_, err := db.ExecContext(ctx, query,
		details.ID,
		details.Title,
		details.Status,
		details.Published,
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
