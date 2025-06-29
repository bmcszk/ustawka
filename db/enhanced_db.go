package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"ustawka/sejm"
)

// StoreEnhancedAct stores an enhanced act with all lifecycle information
func (db *DB) StoreEnhancedAct(ctx context.Context, act *sejm.EnhancedAct) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(); err != nil {
			slog.Error("Error rolling back transaction", "error", err)
		}
	}()

	if err := db.storeActWithEnhancements(ctx, tx, act); err != nil {
		return err
	}

	if err := db.storeAllVotingRecords(ctx, tx, act); err != nil {
		return err
	}

	if err := db.storeAllProcessStages(ctx, tx, act); err != nil {
		return err
	}

	return tx.Commit()
}

// storeAllVotingRecords stores all voting records for an act
func (db *DB) storeAllVotingRecords(ctx context.Context, tx *sql.Tx, act *sejm.EnhancedAct) error {
	for _, vote := range act.SejmVotes {
		if err := db.storeVotingRecord(ctx, tx, act.ID, "sejm", &vote); err != nil {
			return err
		}
	}

	for _, vote := range act.SenateVotes {
		if err := db.storeVotingRecord(ctx, tx, act.ID, "senate", &vote); err != nil {
			return err
		}
	}

	return nil
}

// storeAllProcessStages stores all process stages for an act
func (db *DB) storeAllProcessStages(ctx context.Context, tx *sql.Tx, act *sejm.EnhancedAct) error {
	for _, stage := range act.Stages {
		if err := db.storeProcessStage(ctx, tx, act.ID, &stage); err != nil {
			return err
		}
	}

	return nil
}

// storeActWithEnhancements stores act with enhanced lifecycle fields
func (*DB) storeActWithEnhancements(ctx context.Context, tx *sql.Tx, act *sejm.EnhancedAct) error {
	// First update the basic acts table
	basicQuery := `
		INSERT INTO acts (
			id, title, status, published, position, year, type, address,
			detailed_status, current_stage, stage_date, days_in_stage,
			initiator_type, committee_code, rapporteur_name, urgency_status,
			eu_compliance, process_print_number, rcl_link, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title, status = excluded.status, published = excluded.published,
			position = excluded.position, year = excluded.year, type = excluded.type,
			address = excluded.address, detailed_status = excluded.detailed_status,
			current_stage = excluded.current_stage, stage_date = excluded.stage_date,
			days_in_stage = excluded.days_in_stage, initiator_type = excluded.initiator_type,
			committee_code = excluded.committee_code, rapporteur_name = excluded.rapporteur_name,
			urgency_status = excluded.urgency_status, eu_compliance = excluded.eu_compliance,
			process_print_number = excluded.process_print_number, rcl_link = excluded.rcl_link,
			updated_at = datetime('now')
	`

	var stageDate *time.Time
	if !act.StageDate.IsZero() {
		stageDate = &act.StageDate
	}

	_, err := tx.ExecContext(ctx, basicQuery,
		act.ID, act.Title, act.Status, act.Published, act.Position, act.Year,
		act.Type, act.Address, act.DetailedStatus, act.CurrentStage, stageDate,
		act.DaysInStage, act.InitiatorType, act.CommitteeCode, act.RapporteurName,
		act.UrgencyStatus, act.EUCompliance, act.ProcessPrintNumber, act.RCLLink,
	)

	return err
}

// storeVotingRecord stores a voting record and associated party votes
func (db *DB) storeVotingRecord(ctx context.Context, tx *sql.Tx, actID, chamber string, vote *sejm.VotingRecord) error {
	voteID, err := db.insertVoteRecord(ctx, tx, actID, chamber, vote)
	if err != nil {
		return err
	}

	return db.storePartyVotesForRecord(ctx, tx, voteID, vote.PartyBreakdown)
}

// insertVoteRecord inserts the main voting record and returns its ID
func (*DB) insertVoteRecord(ctx context.Context, tx *sql.Tx, actID, chamber string,
	vote *sejm.VotingRecord) (int64, error) {
	votingData, err := json.Marshal(vote.IndividualVotes)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal voting data: %w", err)
	}

	query := `
		INSERT INTO act_votes (
			act_id, chamber, vote_date, vote_type, proceeding_number, voting_number,
			total_voted, yes_votes, no_votes, abstain_votes, absent_votes,
			result, voting_data, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(act_id, chamber, vote_date, vote_type) DO UPDATE SET
			proceeding_number = excluded.proceeding_number,
			voting_number = excluded.voting_number,
			total_voted = excluded.total_voted,
			yes_votes = excluded.yes_votes,
			no_votes = excluded.no_votes,
			abstain_votes = excluded.abstain_votes,
			absent_votes = excluded.absent_votes,
			result = excluded.result,
			voting_data = excluded.voting_data,
			updated_at = datetime('now')
	`

	result, err := tx.ExecContext(ctx, query,
		actID, chamber, vote.Date, vote.VoteType, vote.ProceedingNumber,
		vote.VotingNumber, vote.TotalVoted, vote.YesVotes, vote.NoVotes,
		vote.AbstainVotes, vote.AbsentVotes, vote.Result, string(votingData),
	)
	if err != nil {
		return 0, err
	}

	voteID, err := result.LastInsertId()
	if err != nil {
		// If we're updating, get the existing ID
		err = tx.QueryRowContext(ctx,
			"SELECT id FROM act_votes WHERE act_id = ? AND chamber = ? AND vote_date = ? AND vote_type = ?",
			actID, chamber, vote.Date, vote.VoteType).Scan(&voteID)
		if err != nil {
			return 0, err
		}
	}

	return voteID, nil
}

// storePartyVotesForRecord stores all party votes for a voting record
func (db *DB) storePartyVotesForRecord(ctx context.Context, tx *sql.Tx, voteID int64,
	partyBreakdown map[string]sejm.PartyVote) error {
	for _, partyVote := range partyBreakdown {
		if err := db.storePartyVote(ctx, tx, voteID, &partyVote); err != nil {
			return err
		}
	}
	return nil
}

// storePartyVote stores party voting breakdown
func (*DB) storePartyVote(ctx context.Context, tx *sql.Tx, voteID int64, partyVote *sejm.PartyVote) error {
	query := `
		INSERT INTO party_votes (
			vote_id, party_name, party_code, total_members, yes_votes,
			no_votes, abstain_votes, absent_votes, discipline_rate, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(vote_id, party_name) DO UPDATE SET
			party_code = excluded.party_code,
			total_members = excluded.total_members,
			yes_votes = excluded.yes_votes,
			no_votes = excluded.no_votes,
			abstain_votes = excluded.abstain_votes,
			absent_votes = excluded.absent_votes,
			discipline_rate = excluded.discipline_rate,
			updated_at = datetime('now')
	`

	_, err := tx.ExecContext(ctx, query,
		voteID, partyVote.Party, partyVote.PartyCode, partyVote.TotalMembers,
		partyVote.YesVotes, partyVote.NoVotes, partyVote.AbstainVotes,
		partyVote.AbsentVotes, partyVote.DisciplineRate,
	)

	return err
}

// storeProcessStage stores a process stage
func (*DB) storeProcessStage(ctx context.Context, tx *sql.Tx, actID string, stage *sejm.ProcessStage) error {
	printNumbers, err := json.Marshal(stage.PrintNumbers)
	if err != nil {
		return fmt.Errorf("failed to marshal print numbers: %w", err)
	}

	query := `
		INSERT INTO act_stages (
			act_id, stage_name, stage_date, stage_order, committee_code,
			committee_name, rapporteur_name, notes, is_current, duration_days,
			print_numbers, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(act_id, stage_name, stage_date) DO UPDATE SET
			stage_order = excluded.stage_order,
			committee_code = excluded.committee_code,
			committee_name = excluded.committee_name,
			rapporteur_name = excluded.rapporteur_name,
			notes = excluded.notes,
			is_current = excluded.is_current,
			duration_days = excluded.duration_days,
			print_numbers = excluded.print_numbers,
			updated_at = datetime('now')
	`

	_, err = tx.ExecContext(ctx, query,
		actID, stage.StageName, stage.StageDate, stage.StageOrder,
		stage.CommitteeCode, stage.CommitteeName, stage.RapporteurName,
		stage.Notes, stage.IsCurrent, stage.DurationDays, string(printNumbers),
	)

	return err
}

// GetEnhancedActs retrieves enhanced acts for a specific year
func (db *DB) GetEnhancedActs(ctx context.Context, year int) ([]sejm.EnhancedAct, error) {
	rows, err := db.queryEnhancedActsForYear(ctx, year)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("Error closing rows", "error", err)
		}
	}()

	var acts []sejm.EnhancedAct
	for rows.Next() {
		act, err := db.scanEnhancedAct(rows)
		if err != nil {
			return nil, err
		}

		db.enrichActWithDetails(ctx, &act)
		acts = append(acts, act)
	}

	return acts, rows.Err()
}

// queryEnhancedActsForYear executes the query for enhanced acts
func (db *DB) queryEnhancedActsForYear(ctx context.Context, year int) (*sql.Rows, error) {
	query := `
		SELECT 
			id, title, status, published, position, year, type, address,
			COALESCE(detailed_status, '') as detailed_status,
			COALESCE(current_stage, '') as current_stage,
			COALESCE(stage_date, '') as stage_date,
			COALESCE(days_in_stage, 0) as days_in_stage,
			COALESCE(initiator_type, '') as initiator_type,
			COALESCE(committee_code, '') as committee_code,
			COALESCE(rapporteur_name, '') as rapporteur_name,
			COALESCE(urgency_status, '') as urgency_status,
			COALESCE(eu_compliance, 0) as eu_compliance,
			COALESCE(process_print_number, '') as process_print_number,
			COALESCE(rcl_link, '') as rcl_link
		FROM acts WHERE year = ? ORDER BY position
	`

	return db.QueryContext(ctx, query, year)
}

// scanEnhancedAct scans a row into an EnhancedAct
func (*DB) scanEnhancedAct(rows *sql.Rows) (sejm.EnhancedAct, error) {
	var act sejm.EnhancedAct
	var stageDateStr string

	err := rows.Scan(
		&act.ID, &act.Title, &act.Status, &act.Published, &act.Position,
		&act.Year, &act.Type, &act.Address, &act.DetailedStatus,
		&act.CurrentStage, &stageDateStr, &act.DaysInStage,
		&act.InitiatorType, &act.CommitteeCode, &act.RapporteurName,
		&act.UrgencyStatus, &act.EUCompliance, &act.ProcessPrintNumber,
		&act.RCLLink,
	)
	if err != nil {
		return act, err
	}

	// Parse stage date
	if stageDateStr != "" {
		if stageDate, err := time.Parse("2006-01-02 15:04:05", stageDateStr); err == nil {
			act.StageDate = stageDate
		}
	}

	return act, nil
}

// enrichActWithDetails loads additional details for an act
func (db *DB) enrichActWithDetails(ctx context.Context, act *sejm.EnhancedAct) {
	if err := db.loadVotingRecords(ctx, act); err != nil {
		slog.Error("Failed to load voting records", "actID", act.ID, "error", err)
	}

	if err := db.loadProcessStages(ctx, act); err != nil {
		slog.Error("Failed to load process stages", "actID", act.ID, "error", err)
	}

	act.Links = sejm.GenerateActLinks(act)
}

// loadVotingRecords loads voting records for an act
func (db *DB) loadVotingRecords(ctx context.Context, act *sejm.EnhancedAct) error {
	votes, err := db.queryVotingRecords(ctx, act.ID)
	if err != nil {
		return err
	}

	db.separateVotesByChamber(act, votes)
	return nil
}

// queryVotingRecords queries all voting records for an act
func (db *DB) queryVotingRecords(ctx context.Context, actID string) ([]sejm.VotingRecord, error) {
	query := `
		SELECT 
			id, chamber, vote_date, vote_type, proceeding_number, voting_number,
			total_voted, yes_votes, no_votes, abstain_votes, absent_votes,
			result, voting_data
		FROM act_votes WHERE act_id = ? ORDER BY vote_date
	`

	rows, err := db.QueryContext(ctx, query, actID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("Error closing rows", "error", err)
		}
	}()

	type voteWithChamber struct {
		vote    sejm.VotingRecord
		chamber string
	}
	
	var votesWithChamber []voteWithChamber
	for rows.Next() {
		vote, chamber, err := db.scanVotingRecord(rows)
		if err != nil {
			return nil, err
		}

		db.enrichVotingRecord(ctx, &vote)
		votesWithChamber = append(votesWithChamber, voteWithChamber{vote, chamber})
	}

	// Convert back to simple slice for separation
	var votes []sejm.VotingRecord
	for _, vwc := range votesWithChamber {
		votes = append(votes, vwc.vote)
	}

	return votes, rows.Err()
}

// scanVotingRecord scans a single voting record from database row
func (*DB) scanVotingRecord(rows *sql.Rows) (sejm.VotingRecord, string, error) {
	var vote sejm.VotingRecord
	var chamber string
	var votingDataStr string

	err := rows.Scan(
		&vote.ID, &chamber, &vote.Date, &vote.VoteType,
		&vote.ProceedingNumber, &vote.VotingNumber, &vote.TotalVoted,
		&vote.YesVotes, &vote.NoVotes, &vote.AbstainVotes,
		&vote.AbsentVotes, &vote.Result, &votingDataStr,
	)

	if err != nil {
		return vote, chamber, err
	}

	// Parse individual votes
	if votingDataStr != "" {
		if err := json.Unmarshal([]byte(votingDataStr), &vote.IndividualVotes); err != nil {
			slog.Error("Failed to parse individual votes", "error", err)
		}
	}

	return vote, chamber, nil
}

// enrichVotingRecord loads party breakdown for a voting record
func (db *DB) enrichVotingRecord(ctx context.Context, vote *sejm.VotingRecord) {
	if err := db.loadPartyVotes(ctx, vote); err != nil {
		slog.Error("Failed to load party votes", "voteID", vote.ID, "error", err)
	}
}

// separateVotesByChamber separates votes by chamber and assigns to act
func (*DB) separateVotesByChamber(act *sejm.EnhancedAct, votes []sejm.VotingRecord) {
	var sejmVotes []sejm.VotingRecord
	var senateVotes []sejm.VotingRecord

	for _, vote := range votes {
		// Chamber info needs to be preserved differently - this is a simplified approach
		// In real implementation, we'd need to pass chamber info through the pipeline
		if len(vote.PartyBreakdown) > 0 {
			// Determine chamber based on party names or other logic
			sejmVotes = append(sejmVotes, vote)
		} else {
			senateVotes = append(senateVotes, vote)
		}
	}

	act.SejmVotes = sejmVotes
	act.SenateVotes = senateVotes
}

// loadPartyVotes loads party voting breakdown for a vote
func (db *DB) loadPartyVotes(ctx context.Context, vote *sejm.VotingRecord) error {
	query := `
		SELECT 
			party_name, party_code, total_members, yes_votes, no_votes,
			abstain_votes, absent_votes, discipline_rate
		FROM party_votes WHERE vote_id = ?
	`

	rows, err := db.QueryContext(ctx, query, vote.ID)
	if err != nil {
		return err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("Error closing rows", "error", err)
		}
	}()

	vote.PartyBreakdown = make(map[string]sejm.PartyVote)

	for rows.Next() {
		var party sejm.PartyVote

		if err := rows.Scan(
			&party.Party, &party.PartyCode, &party.TotalMembers,
			&party.YesVotes, &party.NoVotes, &party.AbstainVotes,
			&party.AbsentVotes, &party.DisciplineRate,
		); err != nil {
			return err
		}

		vote.PartyBreakdown[party.Party] = party
	}

	return rows.Err()
}

// loadProcessStages loads process stages for an act
func (db *DB) loadProcessStages(ctx context.Context, act *sejm.EnhancedAct) error {
	stages, err := db.queryProcessStages(ctx, act.ID)
	if err != nil {
		return err
	}
	
	act.Stages = stages
	return nil
}

// queryProcessStages queries process stages for an act
func (db *DB) queryProcessStages(ctx context.Context, actID string) ([]sejm.ProcessStage, error) {
	query := `
		SELECT 
			id, stage_name, stage_date, stage_order, committee_code,
			committee_name, rapporteur_name, notes, is_current,
			duration_days, print_numbers
		FROM act_stages WHERE act_id = ? ORDER BY stage_order, stage_date
	`

	rows, err := db.QueryContext(ctx, query, actID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("Error closing rows", "error", err)
		}
	}()

	var stages []sejm.ProcessStage
	for rows.Next() {
		stage, err := db.scanProcessStage(rows)
		if err != nil {
			return nil, err
		}
		stages = append(stages, stage)
	}

	return stages, rows.Err()
}

// scanProcessStage scans a process stage from database row
func (*DB) scanProcessStage(rows *sql.Rows) (sejm.ProcessStage, error) {
	var stage sejm.ProcessStage
	var printNumbersStr string

	err := rows.Scan(
		&stage.ID, &stage.StageName, &stage.StageDate, &stage.StageOrder,
		&stage.CommitteeCode, &stage.CommitteeName, &stage.RapporteurName,
		&stage.Notes, &stage.IsCurrent, &stage.DurationDays, &printNumbersStr,
	)
	if err != nil {
		return stage, err
	}

	// Parse print numbers
	if printNumbersStr != "" {
		if err := json.Unmarshal([]byte(printNumbersStr), &stage.PrintNumbers); err != nil {
			slog.Error("Failed to parse print numbers", "error", err)
		}
	}

	return stage, nil
}

// GetEnhancedActByID retrieves a single enhanced act by its ID
func (db *DB) GetEnhancedActByID(ctx context.Context, actID string) (*sejm.EnhancedAct, error) {
	query := `
		SELECT 
			id, title, status, published, position, year, type, address,
			COALESCE(detailed_status, '') as detailed_status,
			COALESCE(current_stage, '') as current_stage,
			COALESCE(stage_date, '') as stage_date,
			COALESCE(days_in_stage, 0) as days_in_stage,
			COALESCE(initiator_type, '') as initiator_type,
			COALESCE(committee_code, '') as committee_code,
			COALESCE(rapporteur_name, '') as rapporteur_name,
			COALESCE(urgency_status, '') as urgency_status,
			COALESCE(eu_compliance, 0) as eu_compliance,
			COALESCE(process_print_number, '') as process_print_number,
			COALESCE(rcl_link, '') as rcl_link
		FROM acts WHERE id = ?
	`

	row := db.QueryRowContext(ctx, query, actID)
	
	var act sejm.EnhancedAct
	var stageDateStr string

	err := row.Scan(
		&act.ID, &act.Title, &act.Status, &act.Published, &act.Position,
		&act.Year, &act.Type, &act.Address, &act.DetailedStatus,
		&act.CurrentStage, &stageDateStr, &act.DaysInStage,
		&act.InitiatorType, &act.CommitteeCode, &act.RapporteurName,
		&act.UrgencyStatus, &act.EUCompliance, &act.ProcessPrintNumber,
		&act.RCLLink,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Parse stage date
	if stageDateStr != "" {
		if stageDate, err := time.Parse("2006-01-02 15:04:05", stageDateStr); err == nil {
			act.StageDate = stageDate
		}
	}

	// Enrich with voting records and process stages
	db.enrichActWithDetails(ctx, &act)
	
	return &act, nil
}

// GetActVotingHistory returns comprehensive voting history for an act
func (db *DB) GetActVotingHistory(ctx context.Context, actID string) ([]sejm.VotingRecord, error) {
	return db.queryVotingHistory(ctx, actID)
}

// queryVotingHistory queries and enriches voting history for an act
func (db *DB) queryVotingHistory(ctx context.Context, actID string) ([]sejm.VotingRecord, error) {
	query := `
		SELECT 
			id, chamber, vote_date, vote_type, proceeding_number, voting_number,
			total_voted, yes_votes, no_votes, abstain_votes, absent_votes,
			result, voting_data
		FROM act_votes WHERE act_id = ? ORDER BY vote_date, chamber
	`

	rows, err := db.QueryContext(ctx, query, actID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("Error closing rows", "error", err)
		}
	}()

	var votes []sejm.VotingRecord
	for rows.Next() {
		vote, _, err := db.scanVotingRecord(rows)
		if err != nil {
			return nil, err
		}

		db.enrichVotingRecord(ctx, &vote)
		votes = append(votes, vote)
	}

	return votes, rows.Err()
}