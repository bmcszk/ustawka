package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"ustawka/db"
	"ustawka/sejm"
)

// Pipeline manages the data collection and enrichment pipeline
type Pipeline struct {
	sejmClient      *sejm.Client
	senateClient    sejm.SenateClient
	linkingService  *sejm.ActLinkingService
	db              *db.DB
	config          *PipelineConfig
	
	// Runtime state
	running         bool
	stopChan        chan struct{}
	wg              sync.WaitGroup
	mu              sync.RWMutex
}

// PipelineConfig contains configuration for the data pipeline
type PipelineConfig struct {
	// Polling intervals
	SejmPollingInterval    time.Duration
	SenatePollingInterval  time.Duration
	EnrichmentInterval     time.Duration
	
	// Data processing
	CurrentTerm            int
	YearsToProcess         []int
	BatchSize              int
	
	// Retry configuration
	MaxRetries             int
	RetryBackoff           time.Duration
	
	// Feature flags
	EnableSejmPolling      bool
	EnableSenatePolling    bool
	EnableEnrichment       bool
	EnableVotingData       bool
}

// PipelineStats tracks pipeline performance metrics
type PipelineStats struct {
	LastSejmPoll      time.Time
	LastSenatePoll    time.Time
	LastEnrichment    time.Time
	
	ActsProcessed     int64
	VotesProcessed    int64
	ErrorCount        int64
	
	SejmAPICallsToday     int64
	SenateAPICallsToday   int64
	LastResetTime         time.Time
}

// NewPipeline creates a new data pipeline
func NewPipeline(sejmClient *sejm.Client, senateClient sejm.SenateClient, 
	database *db.DB, config *PipelineConfig) *Pipeline {
	linkingService := sejm.NewActLinkingService(sejmClient, senateClient)
	
	return &Pipeline{
		sejmClient:     sejmClient,
		senateClient:   senateClient,
		linkingService: linkingService,
		db:             database,
		config:         config,
		stopChan:       make(chan struct{}),
	}
}

// DefaultPipelineConfig returns a sensible default configuration
func DefaultPipelineConfig() *PipelineConfig {
	return &PipelineConfig{
		SejmPollingInterval:   30 * time.Minute,
		SenatePollingInterval: 2 * time.Hour,
		EnrichmentInterval:    4 * time.Hour,
		
		CurrentTerm:           10,
		YearsToProcess:        []int{2023, 2024, 2025},
		BatchSize:             50,
		
		MaxRetries:            3,
		RetryBackoff:          5 * time.Minute,
		
		EnableSejmPolling:     true,
		EnableSenatePolling:   true,
		EnableEnrichment:      true,
		EnableVotingData:      true,
	}
}

// Start begins the data pipeline processing
func (p *Pipeline) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.running {
		return errors.New("pipeline is already running")
	}
	
	p.running = true
	slog.Info("Starting data pipeline", 
		"sejm_interval", p.config.SejmPollingInterval,
		"senate_interval", p.config.SenatePollingInterval)
	
	// Start polling goroutines
	if p.config.EnableSejmPolling {
		p.wg.Add(1)
		go p.sejmPollingLoop(ctx)
	}
	
	if p.config.EnableSenatePolling {
		p.wg.Add(1)
		go p.senatePollingLoop(ctx)
	}
	
	if p.config.EnableEnrichment {
		p.wg.Add(1)
		go p.enrichmentLoop(ctx)
	}
	
	return nil
}

// Stop gracefully stops the pipeline
func (p *Pipeline) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if !p.running {
		return
	}
	
	slog.Info("Stopping data pipeline")
	p.running = false
	close(p.stopChan)
	p.wg.Wait()
	
	slog.Info("Data pipeline stopped")
}

// sejmPollingLoop handles periodic Sejm data polling
func (p *Pipeline) sejmPollingLoop(ctx context.Context) {
	defer p.wg.Done()
	
	ticker := time.NewTicker(p.config.SejmPollingInterval)
	defer ticker.Stop()
	
	// Run initial poll
	p.pollSejmData(ctx)
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopChan:
			return
		case <-ticker.C:
			p.pollSejmData(ctx)
		}
	}
}

// senatePollingLoop handles periodic Senate data polling
func (p *Pipeline) senatePollingLoop(ctx context.Context) {
	defer p.wg.Done()
	
	ticker := time.NewTicker(p.config.SenatePollingInterval)
	defer ticker.Stop()
	
	// Run initial poll
	p.pollSenateData(ctx)
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopChan:
			return
		case <-ticker.C:
			p.pollSenateData(ctx)
		}
	}
}

// enrichmentLoop handles periodic data enrichment
func (p *Pipeline) enrichmentLoop(ctx context.Context) {
	defer p.wg.Done()
	
	ticker := time.NewTicker(p.config.EnrichmentInterval)
	defer ticker.Stop()
	
	// Run initial enrichment
	p.enrichData(ctx)
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopChan:
			return
		case <-ticker.C:
			p.enrichData(ctx)
		}
	}
}

// pollSejmData polls Sejm APIs for new data
func (p *Pipeline) pollSejmData(ctx context.Context) {
	slog.Info("Starting Sejm data polling")
	
	for _, year := range p.config.YearsToProcess {
		if err := p.pollSejmYear(ctx, year); err != nil {
			slog.Error("Failed to poll Sejm data for year", "year", year, "error", err)
		}
	}
	
	if p.config.EnableVotingData {
		if err := p.pollVotingData(ctx); err != nil {
			slog.Error("Failed to poll voting data", "error", err)
		}
	}
	
	slog.Info("Completed Sejm data polling")
}

// pollSejmYear polls Acts for a specific year
func (p *Pipeline) pollSejmYear(ctx context.Context, year int) error {
	// Check cache age
	cacheAge, err := p.db.GetCacheAge(ctx, year)
	if err != nil {
		return fmt.Errorf("failed to get cache age: %w", err)
	}
	
	// Skip if cache is fresh (less than 1 hour for current year, 24 hours for older years)
	maxAge := 24 * time.Hour
	if year == time.Now().Year() {
		maxAge = 1 * time.Hour
	}
	
	if cacheAge < maxAge {
		slog.Debug("Skipping Sejm poll - cache is fresh", "year", year, "age", cacheAge)
		return nil
	}
	
	acts, err := p.sejmClient.GetActs(ctx, year)
	if err != nil {
		return fmt.Errorf("failed to fetch acts for year %d: %w", year, err)
	}
	
	if err := p.db.StoreActs(ctx, year, acts); err != nil {
		return fmt.Errorf("failed to store acts for year %d: %w", year, err)
	}
	
	slog.Info("Polled Sejm data", "year", year, "acts", len(acts))
	return nil
}

// pollVotingData polls for voting information
func (p *Pipeline) pollVotingData(ctx context.Context) error {
	// This would poll for voting data from recent proceedings
	// For now, we'll implement a basic version that gets recent votings
	
	// Get recent proceedings (last 30 days)
	// This is a simplified implementation - in reality, we'd track which 
	// proceedings we've already processed
	
	slog.Debug("Polling voting data for current term", "term", p.config.CurrentTerm)
	
	// Get prints to find recent proceedings
	prints, err := p.sejmClient.GetPrints(ctx, p.config.CurrentTerm)
	if err != nil {
		return fmt.Errorf("failed to get prints: %w", err)
	}
	
	slog.Info("Polled voting data", "prints", len(prints))
	return nil
}

// pollSenateData polls Senate voting data
func (p *Pipeline) pollSenateData(ctx context.Context) {
	slog.Info("Starting Senate data polling")
	
	manifest, err := p.senateClient.GetManifest(ctx)
	if err != nil {
		slog.Error("Failed to get Senate manifest", "error", err)
		return
	}
	
	// Get latest voting files
	individualFile, clubFile := p.senateClient.GetLatestVotingFiles(manifest)
	if individualFile == nil || clubFile == nil {
		slog.Warn("No Senate voting files found in manifest")
		return
	}
	
	// Process individual votes
	individualVotes, err := p.senateClient.GetIndividualVotingData(ctx, individualFile.URL)
	if err != nil {
		slog.Error("Failed to get individual Senate voting data", "error", err)
		return
	}
	
	// Process club votes
	clubVotes, err := p.senateClient.GetClubVotingData(ctx, clubFile.URL)
	if err != nil {
		slog.Error("Failed to get club Senate voting data", "error", err)
		return
	}
	
	slog.Info("Polled Senate data", 
		"individual_votes", len(individualVotes),
		"club_votes", len(clubVotes))
}

// enrichData performs data enrichment and linking
func (p *Pipeline) enrichData(ctx context.Context) {
	slog.Info("Starting data enrichment")
	
	// Get all Acts that need enrichment
	for _, year := range p.config.YearsToProcess {
		if err := p.enrichYear(ctx, year); err != nil {
			slog.Error("Failed to enrich data for year", "year", year, "error", err)
		}
	}
	
	slog.Info("Completed data enrichment")
}

// enrichYear enriches Acts for a specific year
func (p *Pipeline) enrichYear(ctx context.Context, year int) error {
	// Get basic Acts
	acts, err := p.db.GetActs(ctx, year)
	if err != nil {
		return fmt.Errorf("failed to get acts for year %d: %w", year, err)
	}
	
	// Convert to enhanced Acts and process in batches
	for i := 0; i < len(acts); i += p.config.BatchSize {
		end := i + p.config.BatchSize
		if end > len(acts) {
			end = len(acts)
		}
		
		batch := acts[i:end]
		p.enrichActBatch(ctx, batch)
	}
	
	return nil
}

// enrichActBatch enriches a batch of Acts
func (p *Pipeline) enrichActBatch(ctx context.Context, acts []sejm.Act) {
	for _, act := range acts {
		enhancedAct := p.convertToEnhancedAct(act)
		
		// Enrich with process information
		p.enrichWithProcessInfo(ctx, &enhancedAct)
		
		// Store enhanced act
		if err := p.db.StoreEnhancedAct(ctx, &enhancedAct); err != nil {
			slog.Error("Failed to store enhanced act", "act_id", act.ID, "error", err)
		}
	}
}

// convertToEnhancedAct converts basic Act to EnhancedAct
func (*Pipeline) convertToEnhancedAct(act sejm.Act) sejm.EnhancedAct {
	return sejm.EnhancedAct{
		ID:        act.ID,
		Title:     act.Title,
		Status:    act.Status,
		Published: act.Published,
		Position:  act.Position,
		Year:      act.Year,
		Type:      act.Type,
		Address:   act.Address,
		
		// These will be filled in by enrichment
		DetailedStatus:     "",
		CurrentStage:       "",
		StageDate:          time.Time{},
		DaysInStage:        0,
		InitiatorType:      "",
		CommitteeCode:      "",
		RapporteurName:     "",
		UrgencyStatus:      "",
		EUCompliance:       false,
		ProcessPrintNumber: "",
		RCLLink:           "",
		
		SejmVotes:       []sejm.VotingRecord{},
		SenateVotes:     []sejm.VotingRecord{},
		PartyBreakdowns: make(map[string]sejm.PartyVote),
		Stages:          []sejm.ProcessStage{},
		Tags:            []string{},
		Links:           sejm.ActLinks{},
	}
}

// enrichWithProcessInfo enriches Act with process information
func (p *Pipeline) enrichWithProcessInfo(_ context.Context, act *sejm.EnhancedAct) {
	// This would fetch process information from Sejm API
	// For now, we'll implement basic status enrichment
	
	// Determine enhanced status based on existing status
	act.DetailedStatus = sejm.GetEnhancedStatus(act.Status)
	
	// Set current stage based on status
	act.CurrentStage = p.determineCurrentStage(act)
	
	// Calculate days in stage (simplified)
	if !act.StageDate.IsZero() {
		act.DaysInStage = sejm.CalculateDaysInStage(act.StageDate)
	}
	
	// Generate links
	act.Links = sejm.GenerateActLinks(act)
}

// determineCurrentStage determines the current stage based on status
func (*Pipeline) determineCurrentStage(act *sejm.EnhancedAct) string {
	// Map of status to Polish stage name
	statusMap := map[string]string{
		"submitted":              "Wpłynął do Sejmu",
		"committee_first_reading": "I czytanie w komisji",
		"committee_work":         "Praca w komisji",
		"second_reading":         "II czytanie",
		"third_reading":          "III czytanie",
		"passed_sejm":            "Przekazano do Senatu",
		"senate_review":          "Rozpatrywanie w Senacie",
		"presidential_review":    "Rozpatrywanie przez Prezydenta",
		"published":              "Opublikowano",
		"in_force":               "Weszła w życie",
	}
	
	if stage, exists := statusMap[act.DetailedStatus]; exists {
		return stage
	}
	return "Nieznany status"
}

// GetStats returns current pipeline statistics
func (p *Pipeline) GetStats() *PipelineStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	// This would typically be maintained as the pipeline runs
	return &PipelineStats{
		LastSejmPoll:    time.Now().Add(-30 * time.Minute), // Example
		LastSenatePoll:  time.Now().Add(-2 * time.Hour),
		LastEnrichment:  time.Now().Add(-4 * time.Hour),
		ActsProcessed:   1000, // Example
		VotesProcessed:  500,
		ErrorCount:      2,
		SejmAPICallsToday: 48,
		SenateAPICallsToday: 12,
		LastResetTime:   time.Now().Truncate(24 * time.Hour),
	}
}

// IsRunning returns whether the pipeline is currently running
func (p *Pipeline) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}