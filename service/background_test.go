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

// MockEnrichmentService is a mock for the enrichment service
type MockEnrichmentService struct {
	mock.Mock
}

// Ensure MockEnrichmentService implements EnrichmentInterface
var _ service.EnrichmentInterface = (*MockEnrichmentService)(nil)

func (m *MockEnrichmentService) EnrichAct(
	ctx context.Context, 
	act *sejm.EnhancedAct,
) (*service.EnrichmentResult, error) {
	args := m.Called(ctx, act)
	result, ok := args.Get(0).(*service.EnrichmentResult)
	if !ok {
		return nil, args.Error(1)
	}
	return result, args.Error(1)
}

func (m *MockEnrichmentService) ValidateEnrichment(result *service.EnrichmentResult) []string {
	args := m.Called(result)
	if result, ok := args.Get(0).([]string); ok {
		return result
	}
	return nil
}

// MockPipeline is a mock for the pipeline
type MockPipeline struct {
	mock.Mock
	running bool
}

// Ensure MockPipeline implements PipelineInterface
var _ service.PipelineInterface = (*MockPipeline)(nil)

func (m *MockPipeline) Start(ctx context.Context) error {
	args := m.Called(ctx)
	if args.Error(0) == nil {
		m.running = true
	}
	return args.Error(0)
}

func (m *MockPipeline) Stop() {
	m.Called()
	m.running = false
}

func (m *MockPipeline) IsRunning() bool {
	return m.running
}

func (m *MockPipeline) GetStats() *service.PipelineStats {
	args := m.Called()
	if stats, ok := args.Get(0).(*service.PipelineStats); ok {
		return stats
	}
	return nil
}

func TestNewBackgroundService(t *testing.T) {
	pipeline := &MockPipeline{}
	enrichment := &MockEnrichmentService{}
	db := &MockDB{}
	config := service.DefaultBackgroundConfig()

	mockSejmClient := &MockSejmClient{}
	bs := service.NewBackgroundService(pipeline, enrichment, db, mockSejmClient, config)

	assert.NotNil(t, bs)
	assert.False(t, bs.GetStatus().IsRunning)
}

func TestDefaultBackgroundConfig(t *testing.T) {
	config := service.DefaultBackgroundConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 24*time.Hour, config.FullSyncInterval)
	assert.Equal(t, 30*time.Minute, config.IncrementalSyncInterval)
	assert.Equal(t, 4*time.Hour, config.EnrichmentInterval)
	assert.Equal(t, 5*time.Minute, config.HealthCheckInterval)
	assert.True(t, config.EnableDataValidation)
	assert.True(t, config.EnableStatusMonitoring)
	assert.True(t, config.EnablePerformanceMetrics)
	assert.True(t, config.EnableAutoRecovery)
	assert.Equal(t, 3, config.MaxConcurrentJobs)
	assert.Equal(t, 25, config.BatchSize)
	assert.Contains(t, config.YearsToProcess, time.Now().Year())
}

func TestBackgroundServiceStart(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*MockPipeline, *MockEnrichmentService, *MockDB)
		expectError    bool
		errorContains  string
	}{
		{
			name: "Successful start",
			setupMocks: func(mp *MockPipeline, me *MockEnrichmentService, md *MockDB) {
				mp.On("Start", mock.Anything).Return(nil).Once()
				mp.On("GetStats").Return(&service.PipelineStats{
					LastSejmPoll:    time.Now(),
					LastSenatePoll:  time.Now(),
					LastEnrichment:  time.Now(),
					ActsProcessed:   100,
					VotesProcessed:  50,
					ErrorCount:      0,
				}).Maybe()
				// Add mock for monitoring service initialization
				md.On("GetEnhancedActs", mock.Anything, mock.AnythingOfType("int")).
					Return([]sejm.EnhancedAct{}, nil).Maybe()
				// Add mock for background sync operations
				md.On("GetCacheAge", mock.Anything, mock.AnythingOfType("int")).
					Return(time.Hour, nil).Maybe()
				md.On("GetActs", mock.Anything, mock.AnythingOfType("int")).
					Return([]sejm.Act{}, nil).Maybe()
				md.On("StoreActs", mock.Anything, mock.AnythingOfType("int"), mock.Anything).
					Return(nil).Maybe()
				md.On("StoreEnhancedAct", mock.Anything, mock.Anything).
					Return(nil).Maybe()
				// Add mock for Stop method
				mp.On("Stop").Return().Maybe()
			},
			expectError: false,
		},
		{
			name: "Pipeline start failure",
			setupMocks: func(mp *MockPipeline, me *MockEnrichmentService, md *MockDB) {
				mp.On("Start", mock.Anything).Return(assert.AnError).Once()
			},
			expectError:   true,
			errorContains: "failed to start pipeline",
		},
		{
			name: "Already running",
			setupMocks: func(mp *MockPipeline, me *MockEnrichmentService, md *MockDB) {
				// No setup needed for this test
			},
			expectError:   true,
			errorContains: "already running",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline := &MockPipeline{}
			enrichment := &MockEnrichmentService{}
			db := &MockDB{}
			config := service.DefaultBackgroundConfig()
			
			// Disable time-based workers for faster tests
			config.FullSyncInterval = 24 * time.Hour
			config.EnrichmentInterval = 24 * time.Hour
			config.HealthCheckInterval = 24 * time.Hour
			// Disable features that require additional mocks
			config.EnableStatusMonitoring = false
			config.EnablePerformanceMetrics = false

			mockSejmClient := &MockSejmClient{}
			mockSejmClient.On("GetActs", mock.Anything, mock.AnythingOfType("int")).
				Return([]sejm.Act{}, nil).Maybe()
	bs := service.NewBackgroundService(pipeline, enrichment, db, mockSejmClient, config)

			// For "already running" test, start service first
			if tt.name == "Already running" {
				pipeline.On("Start", mock.Anything).Return(nil).Once()
				pipeline.On("GetStats").Return(&service.PipelineStats{}).Maybe()
				err := bs.Start(context.Background())
				assert.NoError(t, err)
				defer bs.Stop()
			}

			tt.setupMocks(pipeline, enrichment, db)

			err := bs.Start(context.Background())
			
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.True(t, bs.GetStatus().IsRunning)
				
				// Clean up
				bs.Stop()
			}

			pipeline.AssertExpectations(t)
			enrichment.AssertExpectations(t)
			db.AssertExpectations(t)
		})
	}
}

func TestBackgroundServiceStop(t *testing.T) {
	pipeline := &MockPipeline{}
	enrichment := &MockEnrichmentService{}
	db := &MockDB{}
	config := service.DefaultBackgroundConfig()
	
	// Disable time-based workers for faster tests
	config.FullSyncInterval = 24 * time.Hour
	config.EnrichmentInterval = 24 * time.Hour
	config.HealthCheckInterval = 24 * time.Hour

	mockSejmClient := &MockSejmClient{}
	bs := service.NewBackgroundService(pipeline, enrichment, db, mockSejmClient, config)

	// Setup mocks for start
	pipeline.On("Start", mock.Anything).Return(nil).Once()
	pipeline.On("Stop").Return().Once()
	pipeline.On("GetStats").Return(&service.PipelineStats{}).Maybe()

	// Start the service
	err := bs.Start(context.Background())
	assert.NoError(t, err)
	assert.True(t, bs.GetStatus().IsRunning)

	// Stop the service
	bs.Stop()
	assert.False(t, bs.GetStatus().IsRunning)

	// Stopping again should be safe
	bs.Stop()

	pipeline.AssertExpectations(t)
}

func TestBackgroundServiceGetStatus(t *testing.T) {
	pipeline := &MockPipeline{}
	enrichment := &MockEnrichmentService{}
	db := &MockDB{}
	config := service.DefaultBackgroundConfig()

	mockSejmClient := &MockSejmClient{}
	bs := service.NewBackgroundService(pipeline, enrichment, db, mockSejmClient, config)

	status := bs.GetStatus()
	assert.NotNil(t, status)
	assert.False(t, status.IsRunning)
	assert.Equal(t, "healthy", status.HealthStatus)
	assert.Equal(t, int64(0), status.ErrorCount)
	assert.Equal(t, int64(0), status.ProcessedToday)
}

func TestBackgroundServiceTriggerSync(t *testing.T) {
	pipeline := &MockPipeline{}
	enrichment := &MockEnrichmentService{}
	db := &MockDB{}
	config := service.DefaultBackgroundConfig()

	mockSejmClient := &MockSejmClient{}
	bs := service.NewBackgroundService(pipeline, enrichment, db, mockSejmClient, config)

	// Test triggering sync when not running
	err := bs.TriggerSync(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")

	// Start the service
	pipeline.On("Start", mock.Anything).Return(nil).Once()
	pipeline.On("Stop").Return().Once()
	pipeline.On("GetStats").Return(&service.PipelineStats{}).Maybe()
	
	// Mock database calls for the triggered sync
	db.On("GetCacheAge", mock.Anything, mock.AnythingOfType("int")).Return(25*time.Hour, nil).Maybe()
	
	err = bs.Start(context.Background())
	assert.NoError(t, err)
	defer bs.Stop()

	// Test triggering sync when running
	err = bs.TriggerSync(context.Background())
	assert.NoError(t, err)

	pipeline.AssertExpectations(t)
	db.AssertExpectations(t)
}

func TestBackgroundServiceTriggerEnrichment(t *testing.T) {
	pipeline := &MockPipeline{}
	enrichment := &MockEnrichmentService{}
	db := &MockDB{}
	config := service.DefaultBackgroundConfig()

	mockSejmClient := &MockSejmClient{}
	bs := service.NewBackgroundService(pipeline, enrichment, db, mockSejmClient, config)

	// Test triggering enrichment when not running
	err := bs.TriggerEnrichment(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")

	// Start the service
	pipeline.On("Start", mock.Anything).Return(nil).Once()
	pipeline.On("Stop").Return().Once()
	pipeline.On("GetStats").Return(&service.PipelineStats{}).Maybe()
	
	// Mock database calls for the triggered enrichment
	db.On("GetActs", mock.Anything, mock.AnythingOfType("int")).Return([]sejm.Act{}, nil).Maybe()
	
	err = bs.Start(context.Background())
	assert.NoError(t, err)
	defer bs.Stop()

	// Test triggering enrichment when running
	err = bs.TriggerEnrichment(context.Background())
	assert.NoError(t, err)

	pipeline.AssertExpectations(t)
	db.AssertExpectations(t)
}

func TestBackgroundServiceHealthStatusLevels(t *testing.T) {
	pipeline := &MockPipeline{}
	enrichment := &MockEnrichmentService{}
	db := &MockDB{}
	config := service.DefaultBackgroundConfig()

	mockSejmClient := &MockSejmClient{}
	bs := service.NewBackgroundService(pipeline, enrichment, db, mockSejmClient, config)

	// Test healthy status (no errors)
	status := bs.GetStatus()
	assert.Equal(t, "healthy", status.HealthStatus)

	// This would need access to internal error counting
	// In a real implementation, we'd provide methods to simulate error conditions
}

// Benchmark tests for performance validation
func BenchmarkBackgroundServiceGetStatus(b *testing.B) {
	pipeline := &MockPipeline{}
	enrichment := &MockEnrichmentService{}
	db := &MockDB{}
	config := service.DefaultBackgroundConfig()

	mockSejmClient := &MockSejmClient{}
	bs := service.NewBackgroundService(pipeline, enrichment, db, mockSejmClient, config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		status := bs.GetStatus()
		_ = status
	}
}

func TestBackgroundServiceConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrency test in short mode")
	}

	pipeline := &MockPipeline{}
	enrichment := &MockEnrichmentService{}
	db := &MockDB{}
	config := service.DefaultBackgroundConfig()
	
	// Reduce intervals for faster testing
	config.FullSyncInterval = 100 * time.Millisecond
	config.IncrementalSyncInterval = 50 * time.Millisecond
	config.EnrichmentInterval = 75 * time.Millisecond
	config.HealthCheckInterval = 25 * time.Millisecond

	mockSejmClient := &MockSejmClient{}
	bs := service.NewBackgroundService(pipeline, enrichment, db, mockSejmClient, config)

	// Setup mocks
	pipeline.On("Start", mock.Anything).Return(nil).Once()
	pipeline.On("Stop").Return().Once()
	pipeline.On("IsRunning").Return(true).Maybe()
	pipeline.On("GetStats").Return(&service.PipelineStats{
		LastSejmPoll:   time.Now(),
		LastSenatePoll: time.Now(),
		ActsProcessed:  100,
	}).Maybe()
	
	// Mock database operations that will be called during sync cycles
	db.On("GetCacheAge", mock.Anything, mock.AnythingOfType("int")).Return(2*time.Hour, nil).Maybe()
	db.On("GetActs", mock.Anything, mock.AnythingOfType("int")).Return([]sejm.Act{}, nil).Maybe()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// Start service
	err := bs.Start(ctx)
	assert.NoError(t, err)

	// Let it run for a short time to test concurrent operations
	time.Sleep(150 * time.Millisecond)

	// Test concurrent status checks
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			status := bs.GetStatus()
			assert.NotNil(t, status)
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Stop service
	bs.Stop()

	pipeline.AssertExpectations(t)
}