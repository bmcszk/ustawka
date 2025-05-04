package db

import (
	"context"
	"os"
	"testing"
	"time"
	"ustawka/sejm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*DB, func()) {
	// Create a temporary database file
	tmpfile, err := os.CreateTemp("", "testdb-*.db")
	require.NoError(t, err)

	db, err := New(tmpfile.Name())
	require.NoError(t, err)

	// Return cleanup function
	cleanup := func() {
		db.Close()
		os.Remove(tmpfile.Name())
	}

	return db, cleanup
}

func TestStoreAndGetActs(t *testing.T) {
	db, cleanup := setupTestDB(t)
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
	err := db.StoreActs(ctx, year, acts)
	require.NoError(t, err)

	// Retrieve acts
	retrieved, err := db.GetActs(ctx, year)
	require.NoError(t, err)
	assert.Len(t, retrieved, 2)
	assert.Equal(t, acts, retrieved)

	// Test cache age
	time.Sleep(time.Millisecond) // Ensure some time has passed
	age, err := db.GetCacheAge(ctx, year)
	require.NoError(t, err)
	assert.True(t, age > 0)
	assert.True(t, age < time.Second)
}

func TestStoreAndGetActDetails(t *testing.T) {
	db, cleanup := setupTestDB(t)
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
	err := db.StoreActDetails(ctx, details)
	require.NoError(t, err)

	// Retrieve details
	retrieved, err := db.GetActDetails(ctx, details.ID)
	require.NoError(t, err)
	assert.Equal(t, details, retrieved)

	// Test non-existent details
	nonExistent, err := db.GetActDetails(ctx, "DU/2024/999")
	require.NoError(t, err)
	assert.Nil(t, nonExistent)
}

func TestUpdateActDetails(t *testing.T) {
	db, cleanup := setupTestDB(t)
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
	err := db.StoreActDetails(ctx, details)
	require.NoError(t, err)

	// Update details
	updated := &sejm.ActDetails{
		ID:        "DU/2024/1",
		Title:     "Updated Title",
		Status:    "uchylony",
		Published: "2024-01-02",
	}

	err = db.StoreActDetails(ctx, updated)
	require.NoError(t, err)

	// Retrieve updated details
	retrieved, err := db.GetActDetails(ctx, details.ID)
	require.NoError(t, err)
	assert.Equal(t, updated, retrieved)
}
