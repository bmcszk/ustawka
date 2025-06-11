package service_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
	"ustawka/sejm"
	"ustawka/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSejmClient is a mock implementation of the Sejm client
type MockSejmClient struct {
	mock.Mock
}

// Ensure MockSejmClient implements service.SejmClient
var _ service.SejmClient = (*MockSejmClient)(nil)

func (m *MockSejmClient) GetActs(ctx context.Context, year int) ([]sejm.Act, error) {
	args := m.Called(ctx, year)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	acts, ok := args.Get(0).([]sejm.Act)
	if !ok {
		return nil, args.Error(1)
	}
	return acts, args.Error(1)
}

func (m *MockSejmClient) GetActDetails(ctx context.Context, actID string) (*sejm.ActDetails, error) {
	args := m.Called(ctx, actID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	details, ok := args.Get(0).(*sejm.ActDetails)
	if !ok {
		return nil, args.Error(1)
	}
	return details, args.Error(1)
}

// MockDB is a mock implementation of the database
type MockDB struct {
	mock.Mock
}

// Ensure MockDB implements service.Database
var _ service.Database = (*MockDB)(nil)

func (m *MockDB) GetActs(ctx context.Context, year int) ([]sejm.Act, error) {
	args := m.Called(ctx, year)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	acts, ok := args.Get(0).([]sejm.Act)
	if !ok {
		return nil, args.Error(1)
	}
	return acts, args.Error(1)
}

func (m *MockDB) StoreActs(ctx context.Context, year int, acts []sejm.Act) error {
	args := m.Called(ctx, year, acts)
	return args.Error(0)
}

func (m *MockDB) GetActDetails(ctx context.Context, actID string) (*sejm.ActDetails, error) {
	args := m.Called(ctx, actID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	details, ok := args.Get(0).(*sejm.ActDetails)
	if !ok {
		return nil, args.Error(1)
	}
	return details, args.Error(1)
}

func (m *MockDB) StoreActDetails(ctx context.Context, details *sejm.ActDetails) error {
	args := m.Called(ctx, details)
	return args.Error(0)
}

func (m *MockDB) GetCacheAge(ctx context.Context, year int) (time.Duration, error) {
	args := m.Called(ctx, year)
	duration, ok := args.Get(0).(time.Duration)
	if !ok {
		return 0, args.Error(1)
	}
	return duration, args.Error(1)
}

func TestGetAvailableYears(t *testing.T) {
	tests := getAvailableYearsTestCases()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runAvailableYearsTest(t, tt)
		})
	}
}

// getAvailableYearsTestCases returns test cases for GetAvailableYears
func getAvailableYearsTestCases() []struct {
	name          string
	setupMocks    func(*MockSejmClient, *MockDB)
	expectedYears []int
	expectedError bool
	errorContains string
} {
	return []struct {
		name          string
		setupMocks    func(*MockSejmClient, *MockDB)
		expectedYears []int
		expectedError bool
		errorContains string
	}{
		{
			name: "All years available from API",
			setupMocks: setupAllYearsAvailable,
			expectedYears: []int{2021, 2022, 2023, 2024, 2025},
			expectedError: false,
		},
		{
			name: "Mixed cache and API data",
			setupMocks: setupMixedCacheAndAPI,
			expectedYears: []int{2022, 2023, 2024},
			expectedError: false,
		},
		{
			name: "All cache errors",
			setupMocks: setupAllCacheErrors,
			expectedYears: nil,
			expectedError: true,
			errorContains: "failed to fetch any years",
		},
		{
			name: "Cache store errors",
			setupMocks: setupCacheStoreErrors,
			expectedYears: []int{2021, 2022, 2023, 2024, 2025},
			expectedError: false,
		},
	}
}

// setupAllYearsAvailable sets up mocks for all years available from API
func setupAllYearsAvailable(mc *MockSejmClient, md *MockDB) {
	for _, year := range []int{2021, 2022, 2023, 2024, 2025} {
		md.On("GetCacheAge", mock.Anything, year).Return(25*time.Hour, nil).Once()
		actID := fmt.Sprintf("DU/%d/1", year)
		mc.On("GetActs", mock.Anything, year).Return([]sejm.Act{{ID: actID}}, nil).Once()
		md.On("StoreActs", mock.Anything, year, mock.Anything).Return(nil).Once()
	}
}

// setupMixedCacheAndAPI sets up mocks for mixed cache and API scenarios
func setupMixedCacheAndAPI(mc *MockSejmClient, md *MockDB) {
	// 2021: not in cache, no data
	md.On("GetCacheAge", mock.Anything, 2021).Return(25*time.Hour, nil).Once()
	mc.On("GetActs", mock.Anything, 2021).Return([]sejm.Act{}, nil).Once()
	md.On("StoreActs", mock.Anything, 2021, mock.Anything).Return(nil).Once()

	// 2022: in cache, has data
	md.On("GetCacheAge", mock.Anything, 2022).Return(1*time.Hour, nil).Once()
	md.On("GetActs", mock.Anything, 2022).Return([]sejm.Act{{ID: "DU/2022/1"}}, nil).Once()

	// 2023: cache error, API success
	md.On("GetCacheAge", mock.Anything, 2023).Return(0*time.Hour, errors.New("cache error")).Once()
	mc.On("GetActs", mock.Anything, 2023).Return([]sejm.Act{{ID: "DU/2023/1"}}, nil).Once()
	md.On("StoreActs", mock.Anything, 2023, mock.Anything).Return(nil).Once()

	// 2024: cache read error, API success
	md.On("GetCacheAge", mock.Anything, 2024).Return(1*time.Hour, nil).Once()
	md.On("GetActs", mock.Anything, 2024).Return([]sejm.Act{}, errors.New("cache read error")).Once()
	mc.On("GetActs", mock.Anything, 2024).Return([]sejm.Act{{ID: "DU/2024/1"}}, nil).Once()
	md.On("StoreActs", mock.Anything, 2024, mock.Anything).Return(nil).Once()

	// 2025: API error
	md.On("GetCacheAge", mock.Anything, 2025).Return(25*time.Hour, nil).Once()
	mc.On("GetActs", mock.Anything, 2025).Return([]sejm.Act{}, errors.New("API error")).Once()
}

// setupAllCacheErrors sets up mocks for all cache errors scenario
func setupAllCacheErrors(mc *MockSejmClient, md *MockDB) {
	for _, year := range []int{2021, 2022, 2023, 2024, 2025} {
		md.On("GetCacheAge", mock.Anything, year).Return(0*time.Hour, errors.New("cache error")).Once()
		mc.On("GetActs", mock.Anything, year).Return(nil, errors.New("API error")).Once()
	}
}

// setupCacheStoreErrors sets up mocks for cache store errors scenario
func setupCacheStoreErrors(mc *MockSejmClient, md *MockDB) {
	for _, year := range []int{2021, 2022, 2023, 2024, 2025} {
		md.On("GetCacheAge", mock.Anything, year).Return(25*time.Hour, nil).Once()
		actID := fmt.Sprintf("DU/%d/1", year)
		mc.On("GetActs", mock.Anything, year).Return([]sejm.Act{{ID: actID}}, nil).Once()
		md.On("StoreActs", mock.Anything, year, mock.Anything).Return(errors.New("store error")).Once()
	}
}

// runAvailableYearsTest executes a single test case for GetAvailableYears
func runAvailableYearsTest(t *testing.T, tt struct {
	name          string
	setupMocks    func(*MockSejmClient, *MockDB)
	expectedYears []int
	expectedError bool
	errorContains string
}) {
	t.Helper()
	mockClient := new(MockSejmClient)
	mockDB := new(MockDB)
	srv := service.NewActServiceWithConfig(mockClient, mockDB, 5*time.Second, 24*time.Hour)

	tt.setupMocks(mockClient, mockDB)

	years, err := srv.GetAvailableYears(context.Background())
	if tt.expectedError {
		assert.Error(t, err)
		if tt.errorContains != "" {
			assert.Contains(t, err.Error(), tt.errorContains)
		}
	} else {
		assert.NoError(t, err)
		assert.Equal(t, tt.expectedYears, years)
	}

	mockClient.AssertExpectations(t)
	mockDB.AssertExpectations(t)
}

func TestGetActsByYear(t *testing.T) {
	tests := []struct {
		name          string
		year          int
		setupMocks    func(*MockSejmClient, *MockDB)
		expectedData  *service.BoardData
		expectedError bool
		errorContains string
	}{
		{
			name: "Data from cache",
			year: 2024,
			setupMocks: func(_ *MockSejmClient, md *MockDB) {
				md.On("GetCacheAge", mock.Anything, 2024).Return(1*time.Hour, nil).Once()
				md.On("GetActs", mock.Anything, 2024).Return([]sejm.Act{
					{ID: "DU/2024/1", Status: "obowiązujący"},
					{ID: "DU/2024/2", Status: "uchylony"},
					{ID: "DU/2024/3", Status: "W przygotowaniu"},
				}, nil).Once()
			},
			expectedData: &service.BoardData{
				Obowiazujace: []sejm.Act{{ID: "DU/2024/1", Status: "obowiązujący"}},
				Uchylone:     []sejm.Act{{ID: "DU/2024/2", Status: "uchylony"}},
				Pending:      []sejm.Act{{ID: "DU/2024/3", Status: "W przygotowaniu"}},
			},
			expectedError: false,
		},
		{
			name: "Cache expired, data from API",
			year: 2024,
			setupMocks: func(mc *MockSejmClient, md *MockDB) {
				md.On("GetCacheAge", mock.Anything, 2024).Return(25*time.Hour, nil).Once()
				mc.On("GetActs", mock.Anything, 2024).Return([]sejm.Act{
					{ID: "DU/2024/1", Status: "obowiązujący"},
					{ID: "DU/2024/2", Status: "uchylony"},
				}, nil).Once()
				md.On("StoreActs", mock.Anything, 2024, mock.Anything).Return(nil).Once()
			},
			expectedData: &service.BoardData{
				Obowiazujace: []sejm.Act{{ID: "DU/2024/1", Status: "obowiązujący"}},
				Uchylone:     []sejm.Act{{ID: "DU/2024/2", Status: "uchylony"}},
				Pending:      []sejm.Act{},
			},
			expectedError: false,
		},
		{
			name: "Cache error, data from API",
			year: 2024,
			setupMocks: func(mc *MockSejmClient, md *MockDB) {
				md.On("GetCacheAge", mock.Anything, 2024).Return(0*time.Hour, errors.New("cache error")).Once()
				mc.On("GetActs", mock.Anything, 2024).Return([]sejm.Act{
					{ID: "DU/2024/1", Status: "obowiązujący"},
				}, nil).Once()
				md.On("StoreActs", mock.Anything, 2024, mock.Anything).Return(nil).Once()
			},
			expectedData: &service.BoardData{
				Obowiazujace: []sejm.Act{{ID: "DU/2024/1", Status: "obowiązujący"}},
				Uchylone:     []sejm.Act{},
				Pending:      []sejm.Act{},
			},
			expectedError: false,
		},
		{
			name: "Cache read error, data from API",
			year: 2024,
			setupMocks: func(mc *MockSejmClient, md *MockDB) {
				md.On("GetCacheAge", mock.Anything, 2024).Return(1*time.Hour, nil).Once()
				md.On("GetActs", mock.Anything, 2024).Return(nil, errors.New("cache read error")).Once()
				mc.On("GetActs", mock.Anything, 2024).Return([]sejm.Act{
					{ID: "DU/2024/1", Status: "obowiązujący"},
				}, nil).Once()
				md.On("StoreActs", mock.Anything, 2024, mock.Anything).Return(nil).Once()
			},
			expectedData: &service.BoardData{
				Obowiazujace: []sejm.Act{{ID: "DU/2024/1", Status: "obowiązujący"}},
				Uchylone:     []sejm.Act{},
				Pending:      []sejm.Act{},
			},
			expectedError: false,
		},
		{
			name: "API error",
			year: 2024,
			setupMocks: func(mc *MockSejmClient, md *MockDB) {
				md.On("GetCacheAge", mock.Anything, 2024).Return(25*time.Hour, nil).Once()
				mc.On("GetActs", mock.Anything, 2024).Return(nil, errors.New("API error")).Once()
			},
			expectedData:  nil,
			expectedError: true,
			errorContains: "failed to fetch acts",
		},
		{
			name: "No data available",
			year: 2024,
			setupMocks: func(mc *MockSejmClient, md *MockDB) {
				md.On("GetCacheAge", mock.Anything, 2024).Return(25*time.Hour, nil).Once()
				mc.On("GetActs", mock.Anything, 2024).Return([]sejm.Act{}, nil).Once()
				md.On("StoreActs", mock.Anything, 2024, mock.Anything).Return(nil).Once()
			},
			expectedData:  nil,
			expectedError: true,
			errorContains: "no data available for year",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runActsByYearTest(t, tt)
		})
	}
}

func TestGetActDetails(t *testing.T) {
	tests := []struct {
		name          string
		year          string
		position      string
		setupMocks    func(*MockSejmClient, *MockDB)
		expectedData  *sejm.ActDetails
		expectedError bool
		errorContains string
	}{
		{
			name:     "Data from cache",
			year:     "2024",
			position: "123",
			setupMocks: func(_ *MockSejmClient, md *MockDB) {
				md.On("GetActDetails", mock.Anything, "DU/2024/123").Return(&sejm.ActDetails{
					ID:        "DU/2024/123",
					Title:     "Test Act",
					Status:    "obowiązujący",
					Published: "2024-01-01",
				}, nil).Once()
			},
			expectedData: &sejm.ActDetails{
				ID:        "DU/2024/123",
				Title:     "Test Act",
				Status:    "obowiązujący",
				Published: "2024-01-01",
			},
			expectedError: false,
		},
		{
			name:     "Cache miss, data from API",
			year:     "2024",
			position: "123",
			setupMocks: func(mc *MockSejmClient, md *MockDB) {
				md.On("GetActDetails", mock.Anything, "DU/2024/123").Return(nil, nil).Once()
				mc.On("GetActDetails", mock.Anything, "DU/2024/123").Return(&sejm.ActDetails{
					ID:        "DU/2024/123",
					Title:     "Test Act",
					Status:    "obowiązujący",
					Published: "2024-01-01",
				}, nil).Once()
				md.On("StoreActDetails", mock.Anything, mock.Anything).Return(nil).Once()
			},
			expectedData: &sejm.ActDetails{
				ID:        "DU/2024/123",
				Title:     "Test Act",
				Status:    "obowiązujący",
				Published: "2024-01-01",
			},
			expectedError: false,
		},
		{
			name:     "Cache error, data from API",
			year:     "2024",
			position: "123",
			setupMocks: func(mc *MockSejmClient, md *MockDB) {
				md.On("GetActDetails", mock.Anything, "DU/2024/123").Return(nil, errors.New("cache error")).Once()
				mc.On("GetActDetails", mock.Anything, "DU/2024/123").Return(&sejm.ActDetails{
					ID:        "DU/2024/123",
					Title:     "Test Act",
					Status:    "obowiązujący",
					Published: "2024-01-01",
				}, nil).Once()
				md.On("StoreActDetails", mock.Anything, mock.Anything).Return(nil).Once()
			},
			expectedData: &sejm.ActDetails{
				ID:        "DU/2024/123",
				Title:     "Test Act",
				Status:    "obowiązujący",
				Published: "2024-01-01",
			},
			expectedError: false,
		},
		{
			name:     "API error",
			year:     "2024",
			position: "123",
			setupMocks: func(mc *MockSejmClient, md *MockDB) {
				md.On("GetActDetails", mock.Anything, "DU/2024/123").Return(nil, nil).Once()
				mc.On("GetActDetails", mock.Anything, "DU/2024/123").Return(nil, errors.New("API error")).Once()
			},
			expectedData:  nil,
			expectedError: true,
			errorContains: "failed to fetch act details",
		},
		{
			name:     "Cache store error",
			year:     "2024",
			position: "123",
			setupMocks: func(mc *MockSejmClient, md *MockDB) {
				md.On("GetActDetails", mock.Anything, "DU/2024/123").Return(nil, nil).Once()
				mc.On("GetActDetails", mock.Anything, "DU/2024/123").Return(&sejm.ActDetails{
					ID:        "DU/2024/123",
					Title:     "Test Act",
					Status:    "obowiązujący",
					Published: "2024-01-01",
				}, nil).Once()
				md.On("StoreActDetails", mock.Anything, mock.Anything).Return(errors.New("store error")).Once()
			},
			expectedData: &sejm.ActDetails{
				ID:        "DU/2024/123",
				Title:     "Test Act",
				Status:    "obowiązujący",
				Published: "2024-01-01",
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runActDetailsTest(t, tt)
		})
	}
}

// runActsByYearTest executes a single test case for GetActsByYear
func runActsByYearTest(t *testing.T, tt struct {
	name          string
	year          int
	setupMocks    func(*MockSejmClient, *MockDB)
	expectedData  *service.BoardData
	expectedError bool
	errorContains string
}) {
	t.Helper()
	mockClient := new(MockSejmClient)
	mockDB := new(MockDB)
	srv := service.NewActServiceWithConfig(mockClient, mockDB, 5*time.Second, 24*time.Hour)

	tt.setupMocks(mockClient, mockDB)

	data, err := srv.GetActsByYear(context.Background(), tt.year)
	if tt.expectedError {
		assert.Error(t, err)
		if tt.errorContains != "" {
			assert.Contains(t, err.Error(), tt.errorContains)
		}
	} else {
		assert.NoError(t, err)
		assert.Equal(t, tt.expectedData, data)
	}

	mockClient.AssertExpectations(t)
	mockDB.AssertExpectations(t)
}

// runActDetailsTest executes a single test case for GetActDetails
func runActDetailsTest(t *testing.T, tt struct {
	name          string
	year          string
	position      string
	setupMocks    func(*MockSejmClient, *MockDB)
	expectedData  *sejm.ActDetails
	expectedError bool
	errorContains string
}) {
	t.Helper()
	mockClient := new(MockSejmClient)
	mockDB := new(MockDB)
	srv := service.NewActServiceWithConfig(mockClient, mockDB, 5*time.Second, 24*time.Hour)

	tt.setupMocks(mockClient, mockDB)

	data, err := srv.GetActDetails(context.Background(), tt.year, tt.position)
	if tt.expectedError {
		assert.Error(t, err)
		if tt.errorContains != "" {
			assert.Contains(t, err.Error(), tt.errorContains)
		}
	} else {
		assert.NoError(t, err)
		assert.Equal(t, tt.expectedData, data)
	}

	mockClient.AssertExpectations(t)
	mockDB.AssertExpectations(t)
}
