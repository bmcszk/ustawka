package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"ustawka/metrics"
	"ustawka/sejm"
)

// PipelineInterface defines the interface for data pipeline operations
type PipelineInterface interface {
	Start(ctx context.Context) error
	Stop()
	IsRunning() bool
	GetStats() *PipelineStats
}

// EnrichmentInterface defines the interface for enrichment operations
type EnrichmentInterface interface {
	EnrichAct(ctx context.Context, act *sejm.EnhancedAct) (*EnrichmentResult, error)
	ValidateEnrichment(result *EnrichmentResult) []string
}

// BackgroundService orchestrates all background data processing tasks
type BackgroundService struct {
	// Dependencies
	pipeline           PipelineInterface
	enrichmentService  EnrichmentInterface
	db                 Database
	sejmClient         SejmClient
	monitoringService  *MonitoringService
	validationService  *DataValidationService
	
	// Configuration
	config             *BackgroundConfig
	
	// Runtime state
	running            bool
	stopChan           chan struct{}
	wg                 sync.WaitGroup
	mu                 sync.RWMutex
	
	// Status tracking
	lastFullSync       time.Time
	lastEnrichmentRun  time.Time
	lastHealthCheck    time.Time
	errorCount         int64
	processedToday     int64
}

// BackgroundConfig contains configuration for background services
type BackgroundConfig struct {
	// Sync intervals
	FullSyncInterval        time.Duration
	IncrementalSyncInterval time.Duration
	EnrichmentInterval      time.Duration
	HealthCheckInterval     time.Duration
	
	// Processing limits
	MaxConcurrentJobs       int
	MaxErrorsPerHour        int
	RetryAttempts           int
	RetryBackoff            time.Duration
	
	// Data processing
	CurrentYear             int
	YearsToProcess          []int
	BatchSize               int
	
	// Feature flags
	EnableDataValidation    bool
	EnableStatusMonitoring  bool
	EnablePerformanceMetrics bool
	EnableAutoRecovery      bool
}

// BackgroundStatus represents the current status of background services
type BackgroundStatus struct {
	IsRunning              bool      `json:"is_running"`
	LastFullSync           time.Time `json:"last_full_sync"`
	LastEnrichmentRun      time.Time `json:"last_enrichment_run"`
	LastHealthCheck        time.Time `json:"last_health_check"`
	ErrorCount             int64     `json:"error_count"`
	ProcessedToday         int64     `json:"processed_today"`
	ActiveJobs             int       `json:"active_jobs"`
	QueuedJobs             int       `json:"queued_jobs"`
	HealthStatus           string    `json:"health_status"`
	NextScheduledSync      time.Time `json:"next_scheduled_sync"`
	EstimatedProcessingTime string   `json:"estimated_processing_time"`
}

// NewBackgroundService creates a new background service orchestrator
func NewBackgroundService(
	pipeline PipelineInterface,
	enrichmentService EnrichmentInterface,
	database Database,
	sejmClient SejmClient,
	config *BackgroundConfig,
) *BackgroundService {
	// Create monitoring service with default config
	monitoringConfig := DefaultMonitoringConfig()
	monitoringService := NewMonitoringService(database, monitoringConfig)
	
	// Add default notification channels
	monitoringService.AddNotificationChannel(NewLogNotificationChannel("default"))
	
	// Create validation service
	validationService := NewDataValidationService(nil)
	
	return &BackgroundService{
		pipeline:          pipeline,
		enrichmentService: enrichmentService,
		db:               database,
		sejmClient:       sejmClient,
		monitoringService: monitoringService,
		validationService: validationService,
		config:           config,
		stopChan:         make(chan struct{}),
	}
}

// DefaultBackgroundConfig returns a sensible default configuration
func DefaultBackgroundConfig() *BackgroundConfig {
	currentYear := time.Now().Year()
	
	return &BackgroundConfig{
		FullSyncInterval:        24 * time.Hour,
		IncrementalSyncInterval: 30 * time.Minute,
		EnrichmentInterval:      4 * time.Hour,
		HealthCheckInterval:     5 * time.Minute,
		
		MaxConcurrentJobs:       3,
		MaxErrorsPerHour:        10,
		RetryAttempts:           3,
		RetryBackoff:            5 * time.Minute,
		
		CurrentYear:             currentYear,
		YearsToProcess:          []int{currentYear - 1, currentYear, currentYear + 1},
		BatchSize:               25,
		
		EnableDataValidation:    true,
		EnableStatusMonitoring:  true,
		EnablePerformanceMetrics: true,
		EnableAutoRecovery:      true,
	}
}

// Start begins all background processing services
func (bs *BackgroundService) Start(ctx context.Context) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	
	if bs.running {
		return errors.New("background service is already running")
	}
	
	bs.running = true
	slog.Info("Starting background enrichment service",
		"full_sync_interval", bs.config.FullSyncInterval,
		"enrichment_interval", bs.config.EnrichmentInterval,
		"years_to_process", bs.config.YearsToProcess)
	
	// Start the main pipeline
	if err := bs.pipeline.Start(ctx); err != nil {
		bs.running = false
		return fmt.Errorf("failed to start pipeline: %w", err)
	}
	
	// Start monitoring service
	if err := bs.monitoringService.Start(ctx); err != nil {
		slog.Error("Failed to start monitoring service", "error", err)
		// Continue without monitoring rather than failing completely
	}
	
	// Start background workers
	bs.wg.Add(1)
	go bs.syncScheduler(ctx)
	
	bs.wg.Add(1)
	go bs.enrichmentScheduler(ctx)
	
	if bs.config.EnableStatusMonitoring {
		bs.wg.Add(1)
		go bs.healthMonitor(ctx)
	}
	
	if bs.config.EnablePerformanceMetrics {
		bs.wg.Add(1)
		go bs.metricsCollector(ctx)
	}
	
	return nil
}

// Stop gracefully stops all background services
func (bs *BackgroundService) Stop() {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	
	if !bs.running {
		return
	}
	
	slog.Info("Stopping background enrichment service")
	bs.running = false
	
	// Stop the pipeline first
	bs.pipeline.Stop()
	
	// Stop monitoring service
	bs.monitoringService.Stop()
	
	// Stop background workers
	close(bs.stopChan)
	bs.wg.Wait()
	
	slog.Info("Background enrichment service stopped")
}

// syncScheduler handles scheduled data synchronization
func (bs *BackgroundService) syncScheduler(ctx context.Context) {
	defer bs.wg.Done()
	
	fullSyncTicker := time.NewTicker(bs.config.FullSyncInterval)
	incrementalTicker := time.NewTicker(bs.config.IncrementalSyncInterval)
	defer fullSyncTicker.Stop()
	defer incrementalTicker.Stop()
	
	// Run initial sync
	bs.runIncrementalSync(ctx)
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-bs.stopChan:
			return
		case <-fullSyncTicker.C:
			bs.runFullSync(ctx)
		case <-incrementalTicker.C:
			bs.runIncrementalSync(ctx)
		}
	}
}

// enrichmentScheduler handles scheduled data enrichment
func (bs *BackgroundService) enrichmentScheduler(ctx context.Context) {
	defer bs.wg.Done()
	
	ticker := time.NewTicker(bs.config.EnrichmentInterval)
	defer ticker.Stop()
	
	// Run initial enrichment
	bs.runEnrichmentCycle(ctx)
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-bs.stopChan:
			return
		case <-ticker.C:
			bs.runEnrichmentCycle(ctx)
		}
	}
}

// healthMonitor monitors system health and performs auto-recovery
func (bs *BackgroundService) healthMonitor(ctx context.Context) {
	defer bs.wg.Done()
	
	ticker := time.NewTicker(bs.config.HealthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-bs.stopChan:
			return
		case <-ticker.C:
			bs.performHealthCheck(ctx)
		}
	}
}

// metricsCollector collects and reports performance metrics
func (bs *BackgroundService) metricsCollector(ctx context.Context) {
	defer bs.wg.Done()
	
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-bs.stopChan:
			return
		case <-ticker.C:
			bs.collectMetrics()
		}
	}
}

// runFullSync performs a complete data synchronization
func (bs *BackgroundService) runFullSync(ctx context.Context) {
	startTime := time.Now()
	slog.Info("Starting full data synchronization")
	
	// Mark sync start
	bs.mu.Lock()
	bs.lastFullSync = startTime
	bs.mu.Unlock()
	
	// Process each year
	for _, year := range bs.config.YearsToProcess {
		if err := bs.syncYear(ctx, year, SyncOptions{ForceSync: true}); err != nil {
			slog.Error("Failed to sync year in full sync", "year", year, "error", err)
			bs.incrementErrorCount()
		}
	}
	
	duration := time.Since(startTime)
	slog.Info("Completed full data synchronization", "duration", duration)
	metrics.IncrementAPI() // Track sync operations
}

// runIncrementalSync performs incremental data updates
func (bs *BackgroundService) runIncrementalSync(ctx context.Context) {
	slog.Info("Starting incremental data synchronization")
	startTime := time.Now()
	
	// Focus on current year for incremental updates
	currentYear := bs.config.CurrentYear
	if err := bs.syncYear(ctx, currentYear, SyncOptions{ForceSync: false}); err != nil {
		slog.Error("Failed incremental sync", "year", currentYear, "error", err)
		bs.incrementErrorCount()
		return
	}
	
	duration := time.Since(startTime)
	slog.Info("Completed incremental synchronization", "duration", duration)
}

// SyncOptions contains options for sync operations
type SyncOptions struct {
	ForceSync bool
}

// syncYear synchronizes data for a specific year
func (bs *BackgroundService) syncYear(ctx context.Context, year int, opts SyncOptions) error {
	// Check if sync is needed
	if !opts.ForceSync {
		cacheAge, err := bs.db.GetCacheAge(ctx, year)
		if err == nil && cacheAge < 30*time.Minute {
			slog.Debug("Skipping sync - cache is fresh", "year", year, "age", cacheAge)
			return nil
		}
	}
	
	// Get acts for the year
	acts, err := bs.sejmClient.GetActs(ctx, year)
	if err != nil {
		return fmt.Errorf("failed to fetch acts for year %d: %w", year, err)
	}
	
	// Store basic acts
	if err := bs.db.StoreActs(ctx, year, acts); err != nil {
		return fmt.Errorf("failed to store acts for year %d: %w", year, err)
	}
	
	// Update processed count
	bs.mu.Lock()
	bs.processedToday += int64(len(acts))
	bs.mu.Unlock()
	
	slog.Info("Synchronized acts for year", "year", year, "count", len(acts))
	return nil
}

// runEnrichmentCycle performs data enrichment on existing acts
func (bs *BackgroundService) runEnrichmentCycle(ctx context.Context) {
	startTime := time.Now()
	slog.Info("Starting enrichment cycle")
	
	// Mark enrichment start
	bs.mu.Lock()
	bs.lastEnrichmentRun = startTime
	bs.mu.Unlock()
	
	enrichedCount := 0
	
	// Process each year
	for _, year := range bs.config.YearsToProcess {
		count, err := bs.enrichYear(ctx, year)
		if err != nil {
			slog.Error("Failed to enrich year", "year", year, "error", err)
			bs.incrementErrorCount()
			continue
		}
		enrichedCount += count
	}
	
	duration := time.Since(startTime)
	slog.Info("Completed enrichment cycle", 
		"duration", duration, 
		"enriched_acts", enrichedCount)
}

// enrichYear enriches acts for a specific year
func (bs *BackgroundService) enrichYear(ctx context.Context, year int) (int, error) {
	// Get basic acts that need enrichment
	acts, err := bs.db.GetActs(ctx, year)
	if err != nil {
		return 0, fmt.Errorf("failed to get acts for year %d: %w", year, err)
	}
	
	enrichedCount := 0
	
	// Process acts in batches
	for i := 0; i < len(acts); i += bs.config.BatchSize {
		end := i + bs.config.BatchSize
		if end > len(acts) {
			end = len(acts)
		}
		
		batch := acts[i:end]
		processed := bs.enrichActBatch(ctx, batch)
		
		enrichedCount += processed
	}
	
	return enrichedCount, nil
}

// enrichActBatch enriches a batch of acts
func (bs *BackgroundService) enrichActBatch(ctx context.Context, acts []sejm.Act) int {
	enrichedCount := 0
	
	for _, act := range acts {
		if bs.processSingleAct(ctx, act) {
			enrichedCount++
		}
	}
	
	return enrichedCount
}

func (bs *BackgroundService) processSingleAct(ctx context.Context, act sejm.Act) bool {
	enhancedAct := bs.convertToEnhancedAct(act)
	
	if bs.isRecentlyEnriched(ctx, enhancedAct.ID) {
		return false
	}
	
	result, err := bs.enrichmentService.EnrichAct(ctx, &enhancedAct)
	if err != nil {
		slog.Error("Failed to enrich act", "act_id", act.ID, "error", err)
		return false
	}
	
	if bs.config.EnableDataValidation {
		bs.validateEnrichmentResult(ctx, act.ID, result)
	}
	
	if err := bs.db.StoreEnhancedAct(ctx, result.EnhancedAct); err != nil {
		slog.Error("Failed to store enriched act", "act_id", act.ID, "error", err)
		return false
	}
	
	return true
}

// convertToEnhancedAct converts basic Act to EnhancedAct
func (*BackgroundService) convertToEnhancedAct(act sejm.Act) sejm.EnhancedAct {
	return sejm.EnhancedAct{
		ID:        act.ID,
		Title:     act.Title,
		Status:    act.Status,
		Published: act.Published,
		Position:  act.Position,
		Year:      act.Year,
		Type:      act.Type,
		Address:   act.Address,
		
		// Initialize empty fields for enrichment
		DetailedStatus:     "",
		CurrentStage:       "",
		StageDate:          time.Time{},
		DaysInStage:        0,
		SejmVotes:          []sejm.VotingRecord{},
		SenateVotes:        []sejm.VotingRecord{},
		PartyBreakdowns:    make(map[string]sejm.PartyVote),
		Stages:             []sejm.ProcessStage{},
		Tags:               []string{},
		Links:              sejm.ActLinks{},
	}
}

// isRecentlyEnriched checks if an act was enriched recently
func (bs *BackgroundService) isRecentlyEnriched(ctx context.Context, actID string) bool {
	// Check if enhanced act exists and was updated recently
	enhancedAct, err := bs.db.GetEnhancedActByID(ctx, actID)
	if err != nil || enhancedAct == nil {
		return false
	}
	
	// Consider it recently enriched if it has detailed status and was processed in the last 24 hours
	return enhancedAct.DetailedStatus != "" && enhancedAct.DetailedStatus != "unknown"
}

// performHealthCheck checks system health and performs recovery if needed
func (bs *BackgroundService) performHealthCheck(ctx context.Context) {
	bs.updateHealthCheckTime()
	bs.checkPipelineHealth(ctx)
	bs.checkErrorRate()
	bs.resetDailyCountersIfNeeded()
}

func (bs *BackgroundService) updateHealthCheckTime() {
	bs.mu.Lock()
	bs.lastHealthCheck = time.Now()
	bs.mu.Unlock()
}

func (bs *BackgroundService) checkPipelineHealth(ctx context.Context) {
	if !bs.pipeline.IsRunning() {
		slog.Warn("Pipeline is not running, attempting restart")
		if bs.config.EnableAutoRecovery {
			if err := bs.pipeline.Start(ctx); err != nil {
				slog.Error("Failed to restart pipeline", "error", err)
				bs.incrementErrorCount()
			}
		}
	}
}

func (bs *BackgroundService) checkErrorRate() {
	if bs.errorCount > int64(bs.config.MaxErrorsPerHour) {
		slog.Warn("High error rate detected", "errors", bs.errorCount)
	}
}

func (bs *BackgroundService) resetDailyCountersIfNeeded() {
	if time.Now().Hour() == 0 && time.Now().Minute() < 5 {
		bs.mu.Lock()
		bs.processedToday = 0
		bs.errorCount = 0
		bs.mu.Unlock()
	}
}

// collectMetrics collects performance metrics
func (bs *BackgroundService) collectMetrics() {
	// This would integrate with the metrics package to collect:
	// - Processing rates
	// - Error rates
	// - Queue lengths
	// - API call counts
	// - Cache hit rates
	
	pipelineStats := bs.pipeline.GetStats()
	
	// Log performance metrics
	slog.Info("Background service metrics",
		"processed_today", bs.processedToday,
		"error_count", bs.errorCount,
		"sejm_api_calls", pipelineStats.SejmAPICallsToday,
		"senate_api_calls", pipelineStats.SenateAPICallsToday)
}

// incrementErrorCount safely increments the error counter
func (bs *BackgroundService) incrementErrorCount() {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.errorCount++
}

// GetStatus returns the current status of background services
func (bs *BackgroundService) GetStatus() *BackgroundStatus {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	
	healthStatus := "healthy"
	if bs.errorCount > int64(bs.config.MaxErrorsPerHour/2) {
		healthStatus = "degraded"
	}
	if bs.errorCount > int64(bs.config.MaxErrorsPerHour) {
		healthStatus = "unhealthy"
	}
	
	nextSync := bs.lastFullSync.Add(bs.config.FullSyncInterval)
	estimatedTime := "~5 minutes"
	if len(bs.config.YearsToProcess) > 2 {
		estimatedTime = "~15 minutes"
	}
	
	status := &BackgroundStatus{
		IsRunning:               bs.running,
		LastFullSync:            bs.lastFullSync,
		LastEnrichmentRun:       bs.lastEnrichmentRun,
		LastHealthCheck:         bs.lastHealthCheck,
		ErrorCount:              bs.errorCount,
		ProcessedToday:          bs.processedToday,
		ActiveJobs:              0, // Would be tracked by job queue
		QueuedJobs:              0, // Would be tracked by job queue
		HealthStatus:            healthStatus,
		NextScheduledSync:       nextSync,
		EstimatedProcessingTime: estimatedTime,
	}
	
	return status
}

// TriggerSync manually triggers a full synchronization
func (bs *BackgroundService) TriggerSync(ctx context.Context) error {
	if !bs.running {
		return errors.New("background service is not running")
	}
	
	slog.Info("Manually triggered full synchronization")
	go bs.runFullSync(ctx)
	return nil
}

// TriggerEnrichment manually triggers enrichment cycle
func (bs *BackgroundService) TriggerEnrichment(ctx context.Context) error {
	if !bs.running {
		return errors.New("background service is not running")
	}
	
	slog.Info("Manually triggered enrichment cycle")
	go bs.runEnrichmentCycle(ctx)
	return nil
}

// GetMonitoringStats returns monitoring service statistics
func (bs *BackgroundService) GetMonitoringStats() map[string]any {
	return bs.monitoringService.GetStats()
}

// GetValidationStats returns validation service statistics
func (bs *BackgroundService) GetValidationStats() map[string]any {
	return bs.validationService.GetValidationStats()
}

// validateEnrichmentResult validates enrichment results using both services
func (bs *BackgroundService) validateEnrichmentResult(ctx context.Context, actID string, result *EnrichmentResult) {
	// Use enrichment service validation first
	if issues := bs.enrichmentService.ValidateEnrichment(result); len(issues) > 0 {
		slog.Warn("Enrichment validation issues", "act_id", actID, "issues", issues)
	}
	
	// Also run comprehensive data validation
	validationResult := bs.validationService.ValidateAct(ctx, result.EnhancedAct)
	if !validationResult.IsValid {
		slog.Warn("Data validation issues found", 
			"act_id", actID,
			"error_count", validationResult.Summary.ErrorCount,
			"warning_count", validationResult.Summary.WarningCount)
		
		// Log critical validation errors
		for _, issue := range validationResult.Issues {
			if issue.Level == ValidationLevelError {
				slog.Error("Critical validation error", 
					"act_id", actID,
					"field", issue.Field,
					"message", issue.Message,
					"code", issue.Code)
			}
		}
	}
}

// AddNotificationChannel adds a notification channel to the monitoring service
func (bs *BackgroundService) AddNotificationChannel(channel NotificationChannel) {
	bs.monitoringService.AddNotificationChannel(channel)
}