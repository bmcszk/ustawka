package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"ustawka/sejm"
)

// StatusChangeEvent represents a change in act status
type StatusChangeEvent struct {
	ActID         string                 `json:"act_id"`
	Title         string                 `json:"title"`
	PreviousStatus string                `json:"previous_status"`
	NewStatus     string                 `json:"new_status"`
	ChangeTime    time.Time              `json:"change_time"`
	ChangeType    StatusChangeType       `json:"change_type"`
	Metadata      map[string]any `json:"metadata"`
}

// StatusChangeType represents the type of status change
type StatusChangeType string

const (
	// StatusChangeTypeProgression represents moving forward in process
	StatusChangeTypeProgression StatusChangeType = "progression"
	// StatusChangeTypeRegression represents moving backward (rare)
	StatusChangeTypeRegression  StatusChangeType = "regression"
	// StatusChangeTypeVoting represents voting occurred
	StatusChangeTypeVoting      StatusChangeType = "voting"
	// StatusChangeTypePublication represents published or entered force
	StatusChangeTypePublication StatusChangeType = "publication"
	// StatusChangeTypeRejection represents rejected or vetoed
	StatusChangeTypeRejection   StatusChangeType = "rejection"
)

// NotificationChannel represents a notification delivery channel
type NotificationChannel interface {
	Send(ctx context.Context, event *StatusChangeEvent) error
	GetChannelType() string
}

// MonitoringService handles act status change monitoring and notifications
type MonitoringService struct {
	db                Database
	channels          []NotificationChannel
	config            *monitoringConfig
	
	// State tracking
	lastSnapshot      map[string]*sejm.EnhancedAct
	mu                sync.RWMutex
	running           bool
	stopChan          chan struct{}
	
	// Statistics
	eventsGenerated   int64
	notificationsSent int64
	lastCheckTime     time.Time
}

// monitoringConfig contains configuration for the monitoring service
type monitoringConfig struct {
	// Check intervals
	CheckInterval         time.Duration
	SnapshotRetention     time.Duration
	
	// Change detection
	EnableVotingDetection bool
	EnableStageTracking   bool
	EnableTimelineUpdates bool
	
	// Notification settings
	NotifyOnProgression   bool
	NotifyOnVoting        bool
	NotifyOnPublication   bool
	NotifyOnRejection     bool
	
	// Filtering
	MinimumChangeThreshold int           // Minimum significance level (1-10)
	MonitoredYears        []int         // Years to monitor
	ExcludedStatuses      []string      // Statuses to ignore
	
	// Performance
	BatchSize             int
	MaxConcurrentChecks   int
}

// NewMonitoringService creates a new monitoring service with default config
func NewMonitoringService(database Database) *MonitoringService {
	return NewMonitoringServiceWithConfig(database, nil)
}

// NewMonitoringServiceWithConfig creates a new monitoring service with custom config
func NewMonitoringServiceWithConfig(database Database, config *monitoringConfig) *MonitoringService {
	if config == nil {
		config = createDefaultMonitoringConfig()
	}
	
	return &MonitoringService{
		db:           database,
		channels:     make([]NotificationChannel, 0),
		config:       config,
		lastSnapshot: make(map[string]*sejm.EnhancedAct),
		stopChan:     make(chan struct{}),
	}
}

// DefaultMonitoringConfig returns a sensible default configuration
func createDefaultMonitoringConfig() *monitoringConfig {
	currentYear := time.Now().Year()
	
	return &monitoringConfig{
		CheckInterval:         15 * time.Minute,
		SnapshotRetention:     7 * 24 * time.Hour, // 7 days
		
		EnableVotingDetection: true,
		EnableStageTracking:   true,
		EnableTimelineUpdates: true,
		
		NotifyOnProgression:   true,
		NotifyOnVoting:        true,
		NotifyOnPublication:   true,
		NotifyOnRejection:     true,
		
		MinimumChangeThreshold: 3,
		MonitoredYears:        []int{currentYear - 1, currentYear, currentYear + 1},
		ExcludedStatuses:      []string{"unknown", ""},
		
		BatchSize:             50,
		MaxConcurrentChecks:   5,
	}
}

// DefaultMonitoringConfig returns a sensible default configuration (for external access)
func DefaultMonitoringConfig() map[string]any {
	config := createDefaultMonitoringConfig()
	return map[string]any{
		"check_interval":         config.CheckInterval,
		"snapshot_retention":     config.SnapshotRetention,
		"enable_voting_detection": config.EnableVotingDetection,
		"enable_stage_tracking":   config.EnableStageTracking,
		"enable_timeline_updates": config.EnableTimelineUpdates,
		"notify_on_progression":   config.NotifyOnProgression,
		"notify_on_voting":        config.NotifyOnVoting,
		"notify_on_publication":   config.NotifyOnPublication,
		"notify_on_rejection":     config.NotifyOnRejection,
		"minimum_change_threshold": config.MinimumChangeThreshold,
		"monitored_years":         config.MonitoredYears,
		"excluded_statuses":       config.ExcludedStatuses,
		"batch_size":              config.BatchSize,
		"max_concurrent_checks":   config.MaxConcurrentChecks,
	}
}

// AddNotificationChannel adds a notification channel
func (ms *MonitoringService) AddNotificationChannel(channel NotificationChannel) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.channels = append(ms.channels, channel)
	slog.Info("Added notification channel", "type", channel.GetChannelType())
}

// Start begins the monitoring service
func (ms *MonitoringService) Start(ctx context.Context) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	if ms.running {
		return nil
	}
	
	ms.running = true
	slog.Info("Starting act status monitoring service", 
		"check_interval", ms.config.CheckInterval,
		"monitored_years", ms.config.MonitoredYears)
	
	// Initialize snapshot
	ms.initializeSnapshot(ctx)
	
	// Start monitoring loop
	go ms.monitoringLoop(ctx)
	
	return nil
}

// Stop gracefully stops the monitoring service
func (ms *MonitoringService) Stop() {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	if !ms.running {
		return
	}
	
	slog.Info("Stopping act status monitoring service")
	ms.running = false
	close(ms.stopChan)
}

// initializeSnapshot creates the initial snapshot of all monitored acts
func (ms *MonitoringService) initializeSnapshot(ctx context.Context) {
	slog.Info("Initializing monitoring snapshot")
	
	for _, year := range ms.config.MonitoredYears {
		if err := ms.processYearForSnapshot(ctx, year); err != nil {
			slog.Warn("Failed to process year for snapshot", "year", year, "error", err)
		}
	}
	
	slog.Info("Monitoring snapshot initialized", "acts_count", len(ms.lastSnapshot))
}

// processYearForSnapshot processes acts from a specific year for snapshot
func (ms *MonitoringService) processYearForSnapshot(ctx context.Context, year int) error {
	acts, err := ms.db.GetEnhancedActs(ctx, year)
	if err != nil {
		return err
	}
	
	for _, act := range acts {
		if ms.shouldMonitorAct(&act) {
			// Create a copy for the snapshot
			actCopy := act
			ms.lastSnapshot[act.ID] = &actCopy
		}
	}
	
	return nil
}

// monitoringLoop is the main monitoring loop
func (ms *MonitoringService) monitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(ms.config.CheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ms.stopChan:
			return
		case <-ticker.C:
			ms.performStatusCheck(ctx)
		}
	}
}

// performStatusCheck checks for status changes across all monitored acts
func (ms *MonitoringService) performStatusCheck(ctx context.Context) {
	startTime := time.Now()
	slog.Info("Performing status change check")
	
	ms.mu.Lock()
	ms.lastCheckTime = startTime
	ms.mu.Unlock()
	
	var allChanges []*StatusChangeEvent
	
	// Check each monitored year
	for _, year := range ms.config.MonitoredYears {
		changes, err := ms.checkYearForChanges(ctx, year)
		if err != nil {
			slog.Error("Failed to check year for changes", "year", year, "error", err)
			continue
		}
		allChanges = append(allChanges, changes...)
	}
	
	// Send notifications for detected changes
	if len(allChanges) > 0 {
		ms.processStatusChanges(ctx, allChanges)
	}
	
	duration := time.Since(startTime)
	slog.Info("Status check completed", 
		"duration", duration,
		"changes_detected", len(allChanges))
}

// checkYearForChanges checks a specific year for status changes
func (ms *MonitoringService) checkYearForChanges(ctx context.Context, year int) ([]*StatusChangeEvent, error) {
	acts, err := ms.db.GetEnhancedActs(ctx, year)
	if err != nil {
		return nil, err
	}
	
	var changes []*StatusChangeEvent
	
	for _, currentAct := range acts {
		if !ms.shouldMonitorAct(&currentAct) {
			continue
		}
		
		actChanges := ms.processActForChanges(&currentAct)
		changes = append(changes, actChanges...)
	}
	
	return changes, nil
}

// processActForChanges processes a single act for changes
func (ms *MonitoringService) processActForChanges(currentAct *sejm.EnhancedAct) []*StatusChangeEvent {
	// Get previous state
	ms.mu.RLock()
	previousAct, exists := ms.lastSnapshot[currentAct.ID]
	ms.mu.RUnlock()
	
	if !exists {
		// New act - add to monitoring but don't generate event
		ms.addActToSnapshot(currentAct)
		return nil
	}
	
	// Detect changes
	changeEvents := ms.detectChanges(previousAct, currentAct)
	if len(changeEvents) > 0 {
		// Update snapshot
		ms.addActToSnapshot(currentAct)
	}
	
	return changeEvents
}

// addActToSnapshot safely adds an act to the snapshot
func (ms *MonitoringService) addActToSnapshot(act *sejm.EnhancedAct) {
	ms.mu.Lock()
	actCopy := *act
	ms.lastSnapshot[act.ID] = &actCopy
	ms.mu.Unlock()
}

// detectChanges detects specific types of changes between two act states
func (ms *MonitoringService) detectChanges(previous, current *sejm.EnhancedAct) []*StatusChangeEvent {
	var events []*StatusChangeEvent
	
	// Check for status changes
	if previous.DetailedStatus != current.DetailedStatus {
		event := &StatusChangeEvent{
			ActID:          current.ID,
			Title:          current.Title,
			PreviousStatus: previous.DetailedStatus,
			NewStatus:      current.DetailedStatus,
			ChangeTime:     time.Now(),
			ChangeType:     ms.categorizeStatusChange(previous.DetailedStatus, current.DetailedStatus),
			Metadata: map[string]any{
				"year":              current.Year,
				"position":          current.Position,
				"current_stage":     current.CurrentStage,
				"days_in_stage":     current.DaysInStage,
				"initiator_type":    current.InitiatorType,
			},
		}
		events = append(events, event)
	}
	
	// Check for voting changes
	if ms.config.EnableVotingDetection && ms.hasVotingChanges(previous, current) {
		event := &StatusChangeEvent{
			ActID:          current.ID,
			Title:          current.Title,
			PreviousStatus: previous.DetailedStatus,
			NewStatus:      current.DetailedStatus,
			ChangeTime:     time.Now(),
			ChangeType:     StatusChangeTypeVoting,
			Metadata: map[string]any{
				"sejm_votes_count":   len(current.SejmVotes),
				"senate_votes_count": len(current.SenateVotes),
				"previous_sejm_votes": len(previous.SejmVotes),
				"previous_senate_votes": len(previous.SenateVotes),
			},
		}
		events = append(events, event)
	}
	
	// Check for stage changes
	if ms.config.EnableStageTracking && previous.CurrentStage != current.CurrentStage {
		event := &StatusChangeEvent{
			ActID:          current.ID,
			Title:          current.Title,
			PreviousStatus: previous.CurrentStage,
			NewStatus:      current.CurrentStage,
			ChangeTime:     time.Now(),
			ChangeType:     StatusChangeTypeProgression,
			Metadata: map[string]any{
				"stage_change": true,
				"previous_stage": previous.CurrentStage,
				"new_stage": current.CurrentStage,
				"stage_duration": current.DaysInStage,
			},
		}
		events = append(events, event)
	}
	
	return events
}

// categorizeStatusChange determines the type of status change
func (*MonitoringService) categorizeStatusChange(previous, current string) StatusChangeType {
	statusOrder := getStatusOrder()
	
	prevOrder, prevExists := statusOrder[previous]
	currOrder, currExists := statusOrder[current]
	
	if !prevExists || !currExists {
		return StatusChangeTypeProgression // Default
	}
	
	return determineChangeType(previous, current, prevOrder, currOrder)
}

// getStatusOrder returns the status progression mapping
func getStatusOrder() map[string]int {
	return map[string]int{
		"submitted":           1,
		"committee_work":      2,
		"second_reading":      3,
		"third_reading":       4,
		"passed_sejm":         5,
		"senate_review":       6,
		"senate_accepted":     7,
		"presidential_review": 8,
		"presidential_signed": 9,
		"published":           10,
		"in_force":            11,
	}
}

// determineChangeType determines the specific type of status change
func determineChangeType(_, current string, prevOrder, currOrder int) StatusChangeType {
	// Check for rejection statuses first
	if current == "senate_rejected" || current == "presidential_veto" {
		return StatusChangeTypeRejection
	}
	
	// Determine based on progression direction
	if currOrder > prevOrder {
		if current == "published" || current == "in_force" {
			return StatusChangeTypePublication
		}
		return StatusChangeTypeProgression
	} else if currOrder < prevOrder {
		return StatusChangeTypeRegression
	}
	
	return StatusChangeTypeProgression
}

// hasVotingChanges checks if there are new voting records
func (*MonitoringService) hasVotingChanges(previous, current *sejm.EnhancedAct) bool {
	return len(current.SejmVotes) > len(previous.SejmVotes) ||
		   len(current.SenateVotes) > len(previous.SenateVotes)
}

// shouldMonitorAct determines if an act should be monitored
func (ms *MonitoringService) shouldMonitorAct(act *sejm.EnhancedAct) bool {
	// Check if status is excluded
	for _, excluded := range ms.config.ExcludedStatuses {
		if act.DetailedStatus == excluded {
			return false
		}
	}
	
	// Check if act is already completed (no more changes expected)
	completedStatuses := []string{"in_force", "rejected", "withdrawn"}
	for _, completed := range completedStatuses {
		if act.DetailedStatus == completed {
			return false
		}
	}
	
	return true
}

// processStatusChanges sends notifications for detected changes
func (ms *MonitoringService) processStatusChanges(ctx context.Context, changes []*StatusChangeEvent) {
	slog.Info("Processing status changes", "count", len(changes))
	
	for _, change := range changes {
		ms.processStatusChange(ctx, change)
	}
}

// processStatusChange processes a single status change event
func (ms *MonitoringService) processStatusChange(ctx context.Context, change *StatusChangeEvent) {
	if !ms.shouldNotifyForChange(change) {
		return
	}
	
	ms.sendNotificationsForChange(ctx, change)
	ms.incrementEventCounter()
}

// sendNotificationsForChange sends notifications to all channels
func (ms *MonitoringService) sendNotificationsForChange(ctx context.Context, change *StatusChangeEvent) {
	for _, channel := range ms.channels {
		if err := channel.Send(ctx, change); err != nil {
			slog.Error("Failed to send notification", 
				"channel", channel.GetChannelType(),
				"act_id", change.ActID,
				"error", err)
		} else {
			ms.incrementNotificationCounter()
		}
	}
}

// incrementEventCounter safely increments the events generated counter
func (ms *MonitoringService) incrementEventCounter() {
	ms.mu.Lock()
	ms.eventsGenerated++
	ms.mu.Unlock()
}

// incrementNotificationCounter safely increments the notifications sent counter
func (ms *MonitoringService) incrementNotificationCounter() {
	ms.mu.Lock()
	ms.notificationsSent++
	ms.mu.Unlock()
}

// shouldNotifyForChange determines if a change should trigger notifications
func (ms *MonitoringService) shouldNotifyForChange(change *StatusChangeEvent) bool {
	switch change.ChangeType {
	case StatusChangeTypeProgression:
		return ms.config.NotifyOnProgression
	case StatusChangeTypeVoting:
		return ms.config.NotifyOnVoting
	case StatusChangeTypePublication:
		return ms.config.NotifyOnPublication
	case StatusChangeTypeRejection:
		return ms.config.NotifyOnRejection
	default:
		return true
	}
}

// GetStats returns monitoring statistics
func (ms *MonitoringService) GetStats() map[string]any {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	
	return map[string]any{
		"running":              ms.running,
		"monitored_acts":       len(ms.lastSnapshot),
		"events_generated":     ms.eventsGenerated,
		"notifications_sent":   ms.notificationsSent,
		"last_check_time":      ms.lastCheckTime,
		"notification_channels": len(ms.channels),
		"monitored_years":      ms.config.MonitoredYears,
	}
}

// LogNotificationChannel implements a simple logging notification channel
type logNotificationChannel struct {
	name string
}

// NewLogNotificationChannel creates a new log-based notification channel
func NewLogNotificationChannel(name string) NotificationChannel {
	return &logNotificationChannel{name: name}
}

// Send sends a notification by logging it
func (lnc *logNotificationChannel) Send(_ context.Context, event *StatusChangeEvent) error {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return err
	}
	
	slog.Info("Status change notification",
		"channel", lnc.name,
		"act_id", event.ActID,
		"change_type", event.ChangeType,
		"previous_status", event.PreviousStatus,
		"new_status", event.NewStatus,
		"event", string(eventJSON))
	
	return nil
}

// GetChannelType returns the channel type
func (*logNotificationChannel) GetChannelType() string {
	return "log"
}

// WebhookNotificationChannel implements a webhook-based notification channel
type WebhookNotificationChannel struct {
	name       string
	webhookURL string
	timeout    time.Duration
}

// NewWebhookNotificationChannel creates a new webhook notification channel
func NewWebhookNotificationChannel(name, webhookURL string) *WebhookNotificationChannel {
	return &WebhookNotificationChannel{
		name:       name,
		webhookURL: webhookURL,
		timeout:    30 * time.Second,
	}
}

// Send sends a notification via webhook
func (wnc *WebhookNotificationChannel) Send(_ context.Context, event *StatusChangeEvent) error {
	// Implementation would make HTTP POST to webhook URL
	// For now, just log that webhook would be called
	slog.Info("Webhook notification",
		"channel", wnc.name,
		"webhook_url", wnc.webhookURL,
		"act_id", event.ActID,
		"change_type", event.ChangeType)
	
	return nil
}

// GetChannelType returns the channel type
func (*WebhookNotificationChannel) GetChannelType() string {
	return "webhook"
}