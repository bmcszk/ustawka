package db

import (
	"context"
	"log/slog"
)

// Migration represents a database schema migration
type Migration struct {
	ID          int
	Description string
	SQL         string
}

// GetMigrations returns all available database migrations
func GetMigrations() []Migration {
	return []Migration{
		{
			ID:          1,
			Description: "Add enhanced Act lifecycle tracking columns to acts table",
			SQL: `
				-- Add new columns to acts table for enhanced lifecycle tracking
				ALTER TABLE acts ADD COLUMN detailed_status VARCHAR(50);
				ALTER TABLE acts ADD COLUMN current_stage VARCHAR(100);
				ALTER TABLE acts ADD COLUMN stage_date TIMESTAMP;
				ALTER TABLE acts ADD COLUMN days_in_stage INTEGER;
				ALTER TABLE acts ADD COLUMN initiator_type VARCHAR(50);
				ALTER TABLE acts ADD COLUMN committee_code VARCHAR(10);
				ALTER TABLE acts ADD COLUMN rapporteur_name VARCHAR(100);
				ALTER TABLE acts ADD COLUMN urgency_status VARCHAR(20);
				ALTER TABLE acts ADD COLUMN eu_compliance BOOLEAN DEFAULT FALSE;
				ALTER TABLE acts ADD COLUMN process_print_number VARCHAR(20);
				ALTER TABLE acts ADD COLUMN rcl_link VARCHAR(255);
				
				-- Add indexes for enhanced queries
				CREATE INDEX IF NOT EXISTS idx_acts_detailed_status ON acts(detailed_status);
				CREATE INDEX IF NOT EXISTS idx_acts_current_stage ON acts(current_stage);
				CREATE INDEX IF NOT EXISTS idx_acts_initiator_type ON acts(initiator_type);
				CREATE INDEX IF NOT EXISTS idx_acts_committee_code ON acts(committee_code);
				CREATE INDEX IF NOT EXISTS idx_acts_urgency_status ON acts(urgency_status);
			`,
		},
		{
			ID:          2,
			Description: "Create act_votes table for Sejm and Senate voting records",
			SQL: `
				CREATE TABLE IF NOT EXISTS act_votes (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					act_id TEXT NOT NULL,
					chamber VARCHAR(10) NOT NULL CHECK (chamber IN ('sejm', 'senate')),
					vote_date TIMESTAMP NOT NULL,
					vote_type VARCHAR(50) NOT NULL, -- 'first_reading', 'amendment', 'final_passage', 'override'
					proceeding_number INTEGER,
					voting_number INTEGER,
					total_voted INTEGER NOT NULL,
					yes_votes INTEGER NOT NULL,
					no_votes INTEGER NOT NULL,
					abstain_votes INTEGER NOT NULL,
					absent_votes INTEGER NOT NULL,
					result VARCHAR(20) NOT NULL CHECK (result IN ('passed', 'failed')),
					voting_data TEXT, -- JSON field for full voting details
					created_at TEXT NOT NULL DEFAULT (datetime('now')),
					updated_at TEXT NOT NULL DEFAULT (datetime('now')),
					FOREIGN KEY (act_id) REFERENCES acts(id)
				);
				
				-- Indexes for voting queries
				CREATE INDEX IF NOT EXISTS idx_act_votes_act_id ON act_votes(act_id);
				CREATE INDEX IF NOT EXISTS idx_act_votes_chamber ON act_votes(chamber);
				CREATE INDEX IF NOT EXISTS idx_act_votes_vote_date ON act_votes(vote_date);
				CREATE INDEX IF NOT EXISTS idx_act_votes_vote_type ON act_votes(vote_type);
				CREATE INDEX IF NOT EXISTS idx_act_votes_result ON act_votes(result);
				
				-- Trigger for automatic updated_at timestamp
				CREATE TRIGGER IF NOT EXISTS update_act_votes_timestamp 
				AFTER UPDATE ON act_votes
				BEGIN
					UPDATE act_votes SET updated_at = datetime('now') WHERE id = NEW.id;
				END;
			`,
		},
		{
			ID:          3,
			Description: "Create party_votes table for party-level voting breakdowns",
			SQL: `
				CREATE TABLE IF NOT EXISTS party_votes (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					vote_id INTEGER NOT NULL,
					party_name VARCHAR(100) NOT NULL,
					party_code VARCHAR(20),
					total_members INTEGER NOT NULL,
					yes_votes INTEGER NOT NULL DEFAULT 0,
					no_votes INTEGER NOT NULL DEFAULT 0,
					abstain_votes INTEGER NOT NULL DEFAULT 0,
					absent_votes INTEGER NOT NULL DEFAULT 0,
					discipline_rate REAL, -- Percentage of party members voting with majority
					created_at TEXT NOT NULL DEFAULT (datetime('now')),
					updated_at TEXT NOT NULL DEFAULT (datetime('now')),
					FOREIGN KEY (vote_id) REFERENCES act_votes(id) ON DELETE CASCADE
				);
				
				-- Indexes for party voting analysis
				CREATE INDEX IF NOT EXISTS idx_party_votes_vote_id ON party_votes(vote_id);
				CREATE INDEX IF NOT EXISTS idx_party_votes_party_name ON party_votes(party_name);
				CREATE INDEX IF NOT EXISTS idx_party_votes_party_code ON party_votes(party_code);
				
				-- Trigger for automatic updated_at timestamp
				CREATE TRIGGER IF NOT EXISTS update_party_votes_timestamp 
				AFTER UPDATE ON party_votes
				BEGIN
					UPDATE party_votes SET updated_at = datetime('now') WHERE id = NEW.id;
				END;
			`,
		},
		{
			ID:          4,
			Description: "Create act_stages table for process stage tracking",
			SQL: `
				CREATE TABLE IF NOT EXISTS act_stages (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					act_id TEXT NOT NULL,
					stage_name VARCHAR(100) NOT NULL,
					stage_date TIMESTAMP NOT NULL,
					stage_order INTEGER NOT NULL, -- Order of stages for timeline
					committee_code VARCHAR(10),
					committee_name VARCHAR(200),
					rapporteur_name VARCHAR(100),
					notes TEXT,
					is_current BOOLEAN DEFAULT FALSE, -- Indicates current stage
					duration_days INTEGER, -- Days spent in this stage
					print_numbers TEXT, -- JSON array of associated print numbers
					created_at TEXT NOT NULL DEFAULT (datetime('now')),
					updated_at TEXT NOT NULL DEFAULT (datetime('now')),
					FOREIGN KEY (act_id) REFERENCES acts(id)
				);
				
				-- Indexes for stage tracking
				CREATE INDEX IF NOT EXISTS idx_act_stages_act_id ON act_stages(act_id);
				CREATE INDEX IF NOT EXISTS idx_act_stages_stage_date ON act_stages(stage_date);
				CREATE INDEX IF NOT EXISTS idx_act_stages_is_current ON act_stages(is_current);
				CREATE INDEX IF NOT EXISTS idx_act_stages_stage_order ON act_stages(stage_order);
				CREATE INDEX IF NOT EXISTS idx_act_stages_committee_code ON act_stages(committee_code);
				
				-- Trigger for automatic updated_at timestamp
				CREATE TRIGGER IF NOT EXISTS update_act_stages_timestamp 
				AFTER UPDATE ON act_stages
				BEGIN
					UPDATE act_stages SET updated_at = datetime('now') WHERE id = NEW.id;
				END;
			`,
		},
		{
			ID:          5,
			Description: "Create migration tracking table",
			SQL: `
				CREATE TABLE IF NOT EXISTS schema_migrations (
					id INTEGER PRIMARY KEY,
					applied_at TEXT NOT NULL DEFAULT (datetime('now'))
				);
			`,
		},
	}
}

// RunMigrations executes pending database migrations
func (db *DB) RunMigrations(ctx context.Context) error {
	if err := db.createMigrationTable(ctx); err != nil {
		return err
	}

	return db.applyPendingMigrations(ctx)
}

// applyPendingMigrations applies all pending migrations
func (db *DB) applyPendingMigrations(ctx context.Context) error {
	migrations := GetMigrations()
	
	for _, migration := range migrations {
		if err := db.processMigration(ctx, migration); err != nil {
			return err
		}
	}
	
	return nil
}

// processMigration processes a single migration
func (db *DB) processMigration(ctx context.Context, migration Migration) error {
	applied, err := db.isMigrationApplied(ctx, migration.ID)
	if err != nil {
		return err
	}
	
	if applied {
		slog.Debug("Migration already applied", "id", migration.ID, "description", migration.Description)
		return nil
	}
	
	slog.Info("Applying migration", "id", migration.ID, "description", migration.Description)
	
	if err := db.applyMigration(ctx, migration); err != nil {
		return err
	}
	
	slog.Info("Migration applied successfully", "id", migration.ID)
	return nil
}

// createMigrationTable creates the schema_migrations table if it doesn't exist
func (db *DB) createMigrationTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id INTEGER PRIMARY KEY,
			applied_at TEXT NOT NULL DEFAULT (datetime('now'))
		)
	`
	_, err := db.ExecContext(ctx, query)
	return err
}

// isMigrationApplied checks if a migration has already been applied
func (db *DB) isMigrationApplied(ctx context.Context, migrationID int) (bool, error) {
	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE id = ?", migrationID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// applyMigration executes a migration within a transaction
func (db *DB) applyMigration(ctx context.Context, migration Migration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(); err != nil {
			slog.Error("Error rolling back migration transaction", "error", err)
		}
	}()

	// Execute the migration SQL
	if _, err := tx.ExecContext(ctx, migration.SQL); err != nil {
		return err
	}

	// Record that migration was applied
	if _, err := tx.ExecContext(ctx, 
		"INSERT INTO schema_migrations (id) VALUES (?)", 
		migration.ID); err != nil {
		return err
	}

	return tx.Commit()
}

// GetAppliedMigrations returns a list of applied migration IDs
func (db *DB) GetAppliedMigrations(ctx context.Context) ([]int, error) {
	exists, err := db.migrationTableExists(ctx)
	if err != nil {
		return nil, err
	}
	
	if !exists {
		return []int{}, nil
	}

	return db.queryAppliedMigrations(ctx)
}

// migrationTableExists checks if the migration table exists
func (db *DB) migrationTableExists(ctx context.Context) (bool, error) {
	var tableExists int
	err := db.QueryRowContext(ctx, 
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'").Scan(&tableExists)
	return tableExists > 0, err
}

// queryAppliedMigrations queries the list of applied migrations
func (db *DB) queryAppliedMigrations(ctx context.Context) ([]int, error) {
	rows, err := db.QueryContext(ctx, "SELECT id FROM schema_migrations ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("Error closing rows", "error", err)
		}
	}()

	var migrations []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		migrations = append(migrations, id)
	}

	return migrations, rows.Err()
}