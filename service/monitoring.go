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
	Metadata      map[string]interface{} `json:"metadata"`
}

// StatusChangeType represents the type of status change
type StatusChangeType string

const (
	StatusChangeTypeProgression StatusChangeType = "progression"  // Moving forward in process
	StatusChangeTypeRegression  StatusChangeType = "regression"   // Moving backward (rare)
	StatusChangeTypeVoting      StatusChangeType = "voting"       // Voting occurred
	StatusChangeTypePublication StatusChangeType = "publication"  // Published or entered force
	StatusChangeTypeRejection   StatusChangeType = "rejection"    // Rejected or vetoed
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
	config            *MonitoringConfig
	
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

// MonitoringConfig contains configuration for the monitoring service
type MonitoringConfig struct {
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

// NewMonitoringService creates a new monitoring service
func NewMonitoringService(database Database, config *MonitoringConfig) *MonitoringService {
	if config == nil {
		config = DefaultMonitoringConfig()
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
func DefaultMonitoringConfig() *MonitoringConfig {
	currentYear := time.Now().Year()
	
	return &MonitoringConfig{
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
	if err := ms.initializeSnapshot(ctx); err != nil {
		ms.running = false
		return err
	}
	
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
func (ms *MonitoringService) initializeSnapshot(ctx context.Context) error {
	slog.Info("Initializing monitoring snapshot")
	
	for _, year := range ms.config.MonitoredYears {
		acts, err := ms.db.GetEnhancedActs(ctx, year)
		if err != nil {
			slog.Warn("Failed to get enhanced acts for snapshot", "year", year, "error", err)
			continue
		}
		
		for _, act := range acts {
			if ms.shouldMonitorAct(&act) {
				// Create a copy for the snapshot
				actCopy := act
				ms.lastSnapshot[act.ID] = &actCopy
			}
		}
	}
	
	slog.Info("Monitoring snapshot initialized", "acts_count", len(ms.lastSnapshot))
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
		
		// Get previous state
		ms.mu.RLock()
		previousAct, exists := ms.lastSnapshot[currentAct.ID]
		ms.mu.RUnlock()
		
		if !exists {
			// New act - add to monitoring but don't generate event
			ms.mu.Lock()
			actCopy := currentAct
			ms.lastSnapshot[currentAct.ID] = &actCopy
			ms.mu.Unlock()
			continue
		}
		
		// Detect changes
		if changeEvents := ms.detectChanges(previousAct, &currentAct); len(changeEvents) > 0 {
			changes = append(changes, changeEvents...)
			
			// Update snapshot
			ms.mu.Lock()
			actCopy := currentAct
			ms.lastSnapshot[currentAct.ID] = &actCopy
			ms.mu.Unlock()
		}
	}
	
	return changes, nil
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
			Metadata: map[string]interface{}{
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
			Metadata: map[string]interface{}{
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
			Metadata: map[string]interface{}{
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
func (ms *MonitoringService) categorizeStatusChange(previous, current string) StatusChangeType {
	// Define status progression order
	statusOrder := map[string]int{
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
	
	prevOrder, prevExists := statusOrder[previous]
	currOrder, currExists := statusOrder[current]
	
	if !prevExists || !currExists {
		return StatusChangeTypeProgression // Default
	}
	
	// Determine change type based on progression
	if currOrder > prevOrder {
		switch current {
		case "published", "in_force":
			return StatusChangeTypePublication
		default:
			return StatusChangeTypeProgression
		}
	} else if currOrder < prevOrder {
		return StatusChangeTypeRegression
	}
	
	// Check for rejection statuses
	if current == "senate_rejected" || current == "presidential_veto" {
		return StatusChangeTypeRejection
	}
	
	return StatusChangeTypeProgression
}

// hasVotingChanges checks if there are new voting records
func (ms *MonitoringService) hasVotingChanges(previous, current *sejm.EnhancedAct) bool {
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
		if !ms.shouldNotifyForChange(change) {
			continue
		}
		
		// Send notifications to all channels
		for _, channel := range ms.channels {
			if err := channel.Send(ctx, change); err != nil {
				slog.Error("Failed to send notification", 
					"channel", channel.GetChannelType(),
					"act_id", change.ActID,
					"error", err)
			} else {
				ms.mu.Lock()
				ms.notificationsSent++
				ms.mu.Unlock()
			}
		}
		
		ms.mu.Lock()
		ms.eventsGenerated++
		ms.mu.Unlock()
	}
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
func (ms *MonitoringService) GetStats() map[string]interface{} {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	
	return map[string]interface{}{
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
type LogNotificationChannel struct {
	name string
}

// NewLogNotificationChannel creates a new log-based notification channel
func NewLogNotificationChannel(name string) *LogNotificationChannel {
	return &LogNotificationChannel{name: name}
}

// Send sends a notification by logging it
func (lnc *LogNotificationChannel) Send(_ context.Context, event *StatusChangeEvent) error {
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
func (lnc *LogNotificationChannel) GetChannelType() string {
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
func (wnc *WebhookNotificationChannel) Send(ctx context.Context, event *StatusChangeEvent) error {
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
func (wnc *WebhookNotificationChannel) GetChannelType() string {
	return "webhook"
}