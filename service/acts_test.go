package service

import (
	"context"
	"errors"
	"testing"
	"time"
	"ustawka/sejm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSejmClient is a mock implementation of the Sejm client
type MockSejmClient struct {
	mock.Mock
}

func (m *MockSejmClient) GetActs(ctx context.Context, year int) ([]sejm.Act, error) {
	args := m.Called(ctx, year)
	return args.Get(0).([]sejm.Act), args.Error(1)
}

func (m *MockSejmClient) GetActDetails(ctx context.Context, actID string) (*sejm.ActDetails, error) {
	args := m.Called(ctx, actID)
	return args.Get(0).(*sejm.ActDetails), args.Error(1)
}

func TestGetAvailableYears(t *testing.T) {
	mockClient := new(MockSejmClient)
	service := NewActService(mockClient)
	ctx := context.Background()
	currentYear := time.Now().Year()

	// Test case 1: All years have data
	for year := 2021; year <= currentYear; year++ {
		mockClient.On("GetActs", ctx, year).Return([]sejm.Act{{Title: "Test Act"}}, nil)
	}

	years, err := service.GetAvailableYears(ctx)
	assert.NoError(t, err)
	expectedYears := make([]int, 0)
	for year := 2021; year <= currentYear; year++ {
		expectedYears = append(expectedYears, year)
	}
	assert.Equal(t, expectedYears, years)

	// Test case 2: Some years have no data
	mockClient = new(MockSejmClient)
	service = NewActService(mockClient)

	// Mock even years to have data, odd years to have no data
	for year := 2021; year <= currentYear; year++ {
		var acts []sejm.Act
		if year%2 == 0 {
			acts = []sejm.Act{{Title: "Test Act"}}
		} else {
			acts = []sejm.Act{}
		}
		mockClient.On("GetActs", ctx, year).Return(acts, nil)
	}

	years, err = service.GetAvailableYears(ctx)
	assert.NoError(t, err)

	expectedYears = make([]int, 0)
	for year := 2021; year <= currentYear; year++ {
		if year%2 == 0 {
			expectedYears = append(expectedYears, year)
		}
	}
	assert.Equal(t, expectedYears, years)
}

func TestGetActsByYear(t *testing.T) {
	mockClient := new(MockSejmClient)
	service := NewActService(mockClient)
	ctx := context.Background()

	// Test case 1: Acts with different statuses
	acts := []sejm.Act{
		{Title: "Act 1", Status: "obowiązujący"},
		{Title: "Act 2", Status: "uchylony"},
		{Title: "Act 3", Status: "W przygotowaniu"},
		{Title: "Act 4", Status: ""},
	}

	mockClient.On("GetActs", ctx, 2024).Return(acts, nil)

	data, err := service.GetActsByYear(ctx, 2024)
	assert.NoError(t, err)
	assert.Len(t, data.Obowiazujace, 1)
	assert.Len(t, data.Uchylone, 1)
	assert.Len(t, data.Pending, 2)

	// Test case 2: No acts available
	mockClient = new(MockSejmClient)
	service = NewActService(mockClient)
	mockClient.On("GetActs", ctx, 2024).Return([]sejm.Act{}, nil)

	data, err = service.GetActsByYear(ctx, 2024)
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "no data available")
}

func TestGetActDetails(t *testing.T) {
	mockClient := new(MockSejmClient)
	service := NewActService(mockClient)
	ctx := context.Background()

	expectedDetails := &sejm.ActDetails{
		ID:        "DU/2024/123",
		Title:     "Test Act",
		Status:    "obowiązujący",
		Published: "2024-01-01",
	}

	mockClient.On("GetActDetails", ctx, "DU/2024/123").Return(expectedDetails, nil)

	details, err := service.GetActDetails(ctx, "2024", "123")
	assert.NoError(t, err)
	assert.Equal(t, expectedDetails, details)

	// Test case 2: Error from client
	mockClient = new(MockSejmClient)
	service = NewActService(mockClient)
	mockErr := errors.New("API error")
	mockClient.On("GetActDetails", ctx, "DU/2024/123").Return((*sejm.ActDetails)(nil), mockErr)

	details, err = service.GetActDetails(ctx, "2024", "123")
	assert.Error(t, err)
	assert.Nil(t, details)
	assert.Contains(t, err.Error(), "failed to fetch act details")
}
