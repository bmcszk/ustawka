package service_test

import (
	"context"
	"testing"
	"time"
	"ustawka/sejm"
	"ustawka/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockNotificationChannel is a mock notification channel for testing
type MockNotificationChannel struct {
	mock.Mock
	channelType string
	sentEvents  []*service.StatusChangeEvent
}

func NewMockNotificationChannel(channelType string) *MockNotificationChannel {
	return &MockNotificationChannel{
		channelType: channelType,
		sentEvents:  make([]*service.StatusChangeEvent, 0),
	}
}

func (m *MockNotificationChannel) Send(ctx context.Context, event *service.StatusChangeEvent) error {
	args := m.Called(ctx, event)
	if args.Error(0) == nil {
		m.sentEvents = append(m.sentEvents, event)
	}
	return args.Error(0)
}

func (m *MockNotificationChannel) GetChannelType() string {
	return m.channelType
}

func (m *MockNotificationChannel) GetSentEvents() []*service.StatusChangeEvent {
	return m.sentEvents
}

func TestNewMonitoringService(t *testing.T) {
	db := &MockDB{}
	
	ms := service.NewMonitoringService(db)
	
	assert.NotNil(t, ms)
	assert.NotNil(t, ms.GetStats())
	stats := ms.GetStats()
	assert.False(t, stats["running"].(bool))
	assert.Equal(t, int64(0), stats["events_generated"])
}

func TestDefaultMonitoringConfig(t *testing.T) {
	config := service.DefaultMonitoringConfig()
	
	assert.NotNil(t, config)
	assert.Equal(t, 15*time.Minute, config["check_interval"])
	assert.True(t, config["enable_voting_detection"].(bool))
	assert.True(t, config["enable_stage_tracking"].(bool))
	assert.True(t, config["notify_on_progression"].(bool))
	assert.True(t, config["notify_on_voting"].(bool))
	assert.Equal(t, 50, config["batch_size"])
	assert.Contains(t, config["monitored_years"], time.Now().Year())
}

func TestAddNotificationChannel(t *testing.T) {
	db := &MockDB{}
	ms := service.NewMonitoringService(db)
	
	channel1 := NewMockNotificationChannel("test1")
	channel2 := NewMockNotificationChannel("test2")
	
	ms.AddNotificationChannel(channel1)
	ms.AddNotificationChannel(channel2)
	
	stats := ms.GetStats()
	assert.Equal(t, 2, stats["notification_channels"])
}

func TestMonitoringServiceStart(t *testing.T) {
	tests := []struct {
		name        string
		setupMocks  func(*MockDB)
		expectError bool
	}{
		{
			name: "Successful start",
			setupMocks: func(md *MockDB) {
				// Mock the initialization call
				md.On("GetEnhancedActs", mock.Anything, mock.AnythingOfType("int")).
					Return([]sejm.EnhancedAct{
						{
							ID:             "DU/2024/1",
							Title:          "Test Act",
							DetailedStatus: "submitted",
							Year:           2024,
						},
					}, nil).Maybe()
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &MockDB{}
			
			ms := service.NewMonitoringService(db)
			
			tt.setupMocks(db)
			
			err := ms.Start(context.Background())
			
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				stats := ms.GetStats()
				assert.True(t, stats["running"].(bool))
				
				// Clean up
				ms.Stop()
			}
			
			db.AssertExpectations(t)
		})
	}
}

func TestMonitoringServiceStop(t *testing.T) {
	db := &MockDB{}
	
	ms := service.NewMonitoringService(db)
	
	// Mock initialization
	db.On("GetEnhancedActs", mock.Anything, mock.AnythingOfType("int")).
		Return([]sejm.EnhancedAct{}, nil).Maybe()
	
	err := ms.Start(context.Background())
	assert.NoError(t, err)
	
	stats := ms.GetStats()
	assert.True(t, stats["running"].(bool))
	
	ms.Stop()
	
	stats = ms.GetStats()
	assert.False(t, stats["running"].(bool))
	
	// Stopping again should be safe
	ms.Stop()
}

func TestStatusChangeDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping status change detection test in short mode")
	}
	
	db := &MockDB{}
	
	ms := service.NewMonitoringService(db)
	
	// Mock notification channel
	mockChannel := NewMockNotificationChannel("test")
	mockChannel.On("Send", mock.Anything, mock.Anything).Return(nil).Maybe()
	ms.AddNotificationChannel(mockChannel)
	
	// Initial state
	initialActs := []sejm.EnhancedAct{
		{
			ID:             "DU/2024/1",
			Title:          "Test Act",
			DetailedStatus: "submitted",
			Year:           2024,
			CurrentStage:   "Initial Stage",
		},
	}
	
	// Changed state
	changedActs := []sejm.EnhancedAct{
		{
			ID:             "DU/2024/1",
			Title:          "Test Act",
			DetailedStatus: "committee_work", // Status changed
			Year:           2024,
			CurrentStage:   "Committee Review", // Stage changed
		},
	}
	
	// Setup mocks for initialization
	db.On("GetEnhancedActs", mock.Anything, 2024).Return(initialActs, nil).Once()
	
	// Setup mocks for subsequent checks
	db.On("GetEnhancedActs", mock.Anything, 2024).Return(changedActs, nil).Maybe()
	
	// Start monitoring
	err := ms.Start(context.Background())
	assert.NoError(t, err)
	defer ms.Stop()
	
	// Wait for at least one check cycle
	time.Sleep(100 * time.Millisecond)
	
	// Verify that changes were detected and notifications sent
	stats := ms.GetStats()
	assert.True(t, stats["events_generated"].(int64) > 0)
	
	// Check that mock channel received notifications
	sentEvents := mockChannel.GetSentEvents()
	assert.True(t, len(sentEvents) > 0, "Expected at least one notification to be sent")
	
	db.AssertExpectations(t)
}

func TestLogNotificationChannel(t *testing.T) {
	channel := service.NewLogNotificationChannel("test-log")
	
	assert.Equal(t, "log", channel.GetChannelType())
	
	event := &service.StatusChangeEvent{
		ActID:          "DU/2024/1",
		Title:          "Test Act",
		PreviousStatus: "submitted",
		NewStatus:      "committee_work",
		ChangeTime:     time.Now(),
		ChangeType:     "progression",
	}
	
	err := channel.Send(context.Background(), event)
	assert.NoError(t, err)
}

func TestWebhookNotificationChannel(t *testing.T) {
	channel := service.NewWebhookNotificationChannel("test-webhook", "https://example.com/webhook")
	
	assert.Equal(t, "webhook", channel.GetChannelType())
	
	event := &service.StatusChangeEvent{
		ActID:          "DU/2024/1",
		Title:          "Test Act",
		PreviousStatus: "submitted",
		NewStatus:      "committee_work",
		ChangeTime:     time.Now(),
		ChangeType:     "progression",
	}
	
	// This will just log since we don't have real webhook implementation
	err := channel.Send(context.Background(), event)
	assert.NoError(t, err)
}

func TestMonitoringServiceStats(t *testing.T) {
	db := &MockDB{}
	ms := service.NewMonitoringService(db)
	
	// Add some channels
	ms.AddNotificationChannel(NewMockNotificationChannel("test1"))
	ms.AddNotificationChannel(NewMockNotificationChannel("test2"))
	
	stats := ms.GetStats()
	
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "running")
	assert.Contains(t, stats, "monitored_acts")
	assert.Contains(t, stats, "events_generated")
	assert.Contains(t, stats, "notifications_sent")
	assert.Contains(t, stats, "notification_channels")
	assert.Contains(t, stats, "monitored_years")
	
	assert.False(t, stats["running"].(bool))
	assert.Equal(t, 2, stats["notification_channels"])
	assert.Equal(t, int64(0), stats["events_generated"])
	assert.Equal(t, int64(0), stats["notifications_sent"])
}

func TestMonitoringConfigValidation(t *testing.T) {
	config := service.DefaultMonitoringConfig()
	
	// Test that all required fields are set
	assert.Greater(t, config["check_interval"].(time.Duration), time.Duration(0))
	assert.Greater(t, config["batch_size"].(int), 0)
	assert.Greater(t, len(config["monitored_years"].([]int)), 0)
	assert.True(t, config["notify_on_progression"].(bool) || config["notify_on_voting"].(bool) || 
		config["notify_on_publication"].(bool) || config["notify_on_rejection"].(bool))
}

// Benchmark tests for performance validation
func BenchmarkMonitoringServiceGetStats(b *testing.B) {
	db := &MockDB{}
	ms := service.NewMonitoringService(db)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stats := ms.GetStats()
		_ = stats
	}
}

func BenchmarkStatusChangeDetection(b *testing.B) {
	db := &MockDB{}
	ms := service.NewMonitoringService(db)
	
	// Setup test data
	testActs := make([]sejm.EnhancedAct, 100)
	for i := 0; i < 100; i++ {
		testActs[i] = sejm.EnhancedAct{
			ID:             string(rune('A' + i)),
			Title:          "Test Act",
			DetailedStatus: "submitted",
			Year:           2024,
		}
	}
	
	db.On("GetEnhancedActs", mock.Anything, mock.AnythingOfType("int")).
		Return(testActs, nil).Maybe()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// This would test the performance of change detection logic
		// In a real benchmark, we'd call the internal methods
		_ = ms.GetStats()
	}
}