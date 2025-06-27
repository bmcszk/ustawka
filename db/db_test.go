package db_test

import (
	"context"
	"os"
	"testing"
	"time"
	"ustawka/db"
	"ustawka/sejm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*db.DB, func()) {
	t.Helper()
	// Create a temporary database file
	tmpfile, err := os.CreateTemp("", "testdb-*.db")
	require.NoError(t, err)

	database, err := db.New(tmpfile.Name())
	require.NoError(t, err)

	// Return cleanup function
	cleanup := func() {
		database.Close()
		os.Remove(tmpfile.Name())
	}

	return database, cleanup
}

func TestStoreAndGetActs(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	year := 2024

	// Test data
	acts := []sejm.Act{
		{
			ID:        "DU/2024/1",
			Title:     "Test Act 1",
			Status:    "obowiązujący",
			Published: "2024-01-01",
			Position:  1,
			Year:      year,
			Type:      "Ustawa",
			Address:   "WDU20240000001",
		},
		{
			ID:        "DU/2024/2",
			Title:     "Test Act 2",
			Status:    "uchylony",
			Published: "2024-01-02",
			Position:  2,
			Year:      year,
			Type:      "Rozporządzenie",
			Address:   "WDU20240000002",
		},
	}

	// Store acts
	err := database.StoreActs(ctx, year, acts)
	require.NoError(t, err)

	// Retrieve acts
	retrieved, err := database.GetActs(ctx, year)
	require.NoError(t, err)
	assert.Len(t, retrieved, 2)
	assert.Equal(t, acts, retrieved)

	// Test cache age
	time.Sleep(time.Millisecond) // Ensure some time has passed
	age, err := database.GetCacheAge(ctx, year)
	require.NoError(t, err)
	assert.True(t, age > 0)
	assert.True(t, age < time.Second)
}

func TestStoreAndGetActDetails(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Test data
	details := &sejm.ActDetails{
		ID:        "DU/2024/1",
		Title:     "Test Act Details",
		Status:    "obowiązujący",
		Published: "2024-01-01",
	}

	// Store details
	err := database.StoreActDetails(ctx, details)
	require.NoError(t, err)

	// Retrieve details
	retrieved, err := database.GetActDetails(ctx, details.ID)
	require.NoError(t, err)
	assert.Equal(t, details, retrieved)

	// Test non-existent details
	nonExistent, err := database.GetActDetails(ctx, "DU/2024/999")
	require.NoError(t, err)
	assert.Nil(t, nonExistent)
}

func TestUpdateActDetails(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Initial details
	details := &sejm.ActDetails{
		ID:        "DU/2024/1",
		Title:     "Initial Title",
		Status:    "obowiązujący",
		Published: "2024-01-01",
	}

	// Store initial details
	err := database.StoreActDetails(ctx, details)
	require.NoError(t, err)

	// Update details
	updated := &sejm.ActDetails{
		ID:        "DU/2024/1",
		Title:     "Updated Title",
		Status:    "uchylony",
		Published: "2024-01-02",
	}

	err = database.StoreActDetails(ctx, updated)
	require.NoError(t, err)

	// Retrieve updated details
	retrieved, err := database.GetActDetails(ctx, details.ID)
	require.NoError(t, err)
	assert.Equal(t, updated, retrieved)
}
