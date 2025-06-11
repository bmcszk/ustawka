package sejm_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"ustawka/sejm"
)

// TestRealAPI tests against the actual Sejm API
func TestRealAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real API test in short mode")
	}

	client := sejm.NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test GetActs for 2024
	t.Run("2024", func(t *testing.T) {
		testRealAPIForYearWithDetails(ctx, t, client, 2024)
	})

	// Test GetActs for 2021
	t.Run("2021", func(t *testing.T) {
		testRealAPIForYearBasic(ctx, t, client, 2021)
	})
}

// Unit tests below this line
func TestGetActs(t *testing.T) {
	client := sejm.NewClient()
	ctx := context.Background()

	acts, err := client.GetActs(ctx, 2021)
	if err != nil {
		t.Fatalf("Failed to get acts: %v", err)
	}

	if len(acts) == 0 {
		t.Fatal("Expected non-empty acts list")
	}

	// Verify first act structure
	validateActStructure(t, acts[0], 2021)
}

// validateActStructure validates the structure of an Act
func validateActStructure(t *testing.T, act sejm.Act, expectedYear int) {
	t.Helper()
	
	validateActStringFields(t, act)
	validateActNumericFields(t, act, expectedYear)
}

// validateActStringFields validates string fields of an Act
func validateActStringFields(t *testing.T, act sejm.Act) {
	t.Helper()
	
	if act.ID == "" {
		t.Error("Expected non-empty ID")
	}
	if act.Title == "" {
		t.Error("Expected non-empty Title")
	}
	if act.Status == "" {
		t.Error("Expected non-empty Status")
	}
	if act.Published == "" {
		t.Error("Expected non-empty Published date")
	}
	if act.Type == "" {
		t.Error("Expected non-empty Type")
	}
	if act.Address == "" {
		t.Error("Expected non-empty Address")
	}
}

// validateActNumericFields validates numeric fields of an Act
func validateActNumericFields(t *testing.T, act sejm.Act, expectedYear int) {
	t.Helper()
	
	if act.Position == 0 {
		t.Error("Expected non-zero Position")
	}
	if act.Year != expectedYear {
		t.Errorf("Expected Year to be %d, got %d", expectedYear, act.Year)
	}
}

func TestGetActDetails(t *testing.T) {
	client := sejm.NewClient()
	ctx := context.Background()

	// First get a list of acts to have a valid ID
	acts, err := client.GetActs(ctx, 2021)
	if err != nil {
		t.Fatalf("Failed to get acts: %v", err)
	}
	if len(acts) == 0 {
		t.Fatal("Expected non-empty acts list")
	}

	// Get details for the first act
	details, err := client.GetActDetails(ctx, acts[0].ID)
	if err != nil {
		t.Fatalf("Failed to get act details: %v", err)
	}

	if details.ID == "" {
		t.Error("Expected non-empty ID")
	}
	if details.Title == "" {
		t.Error("Expected non-empty Title")
	}
	if details.Status == "" {
		t.Error("Expected non-empty Status")
	}
	if details.Published == "" {
		t.Error("Expected non-empty Published date")
	}
}

func TestGetActsError(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create a client with the test server URL
	client := sejm.NewClientWithURL(server.URL)

	// Test GetActs with error
	_, err := client.GetActs(context.Background(), 2024)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestGetActDetailsError(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create a client with the test server URL
	client := sejm.NewClientWithURL(server.URL)

	// Test GetActDetails with error
	_, err := client.GetActDetails(context.Background(), "DU/2024/1")
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestGetYearString(t *testing.T) {
	act := sejm.Act{Year: 2024}
	if act.GetYearString() != "2024" {
		t.Errorf("Expected '2024', got '%s'", act.GetYearString())
	}
}

// testRealAPIForYearWithDetails tests API functionality for a specific year including details
func testRealAPIForYearWithDetails(ctx context.Context, t *testing.T, client *sejm.Client, year int) {
	t.Helper()
	
	acts := fetchAndValidateActs(ctx, t, client, year)
	testActDetailsAndLogging(ctx, t, client, acts)
	t.Logf("Successfully fetched %d acts from real API for %d", len(acts), year)
}

// testRealAPIForYearBasic tests basic API functionality for a specific year
func testRealAPIForYearBasic(ctx context.Context, t *testing.T, client *sejm.Client, year int) {
	t.Helper()
	
	acts := fetchAndValidateActs(ctx, t, client, year)
	t.Logf("Successfully fetched %d acts from real API for %d", len(acts), year)
}

// fetchAndValidateActs fetches acts and validates them for a year
func fetchAndValidateActs(ctx context.Context, t *testing.T, client *sejm.Client, year int) []sejm.Act {
	t.Helper()
	
	acts, err := client.GetActs(ctx, year)
	if err != nil {
		t.Fatalf("Failed to fetch acts from real API for %d: %v", year, err)
	}

	if len(acts) == 0 {
		t.Fatalf("Expected non-empty acts list from real API for %d", year)
	}

	validateActsForYear(t, acts, year)
	return acts
}

// validateActsForYear ensures all acts belong to the expected year
func validateActsForYear(t *testing.T, acts []sejm.Act, expectedYear int) {
	t.Helper()
	
	for i, act := range acts {
		if act.Year != expectedYear {
			t.Errorf("Act at index %d has wrong year: got %d, want %d", i, act.Year, expectedYear)
		}
	}
}

// testActDetailsAndLogging tests act details fetching and logs sample data
func testActDetailsAndLogging(ctx context.Context, t *testing.T, client *sejm.Client, acts []sejm.Act) {
	t.Helper()
	
	t.Logf("Sample act from %d: %+v", acts[0].Year, acts[0])

	// Test GetActDetails with the first act
	details, err := client.GetActDetails(ctx, acts[0].ID)
	if err != nil {
		t.Fatalf("Failed to fetch act details from real API: %v", err)
	}

	if details == nil {
		t.Fatal("Expected non-nil act details from real API")
	}

	t.Logf("Sample act details from real API: %+v", details)

	// Log the structure of the first few acts for debugging
	t.Log("Real API response structure:")
	for i := 0; i < 3 && i < len(acts); i++ {
		actJSON, _ := json.MarshalIndent(acts[i], "", "  ")
		t.Logf("Act %d:\n%s", i+1, string(actJSON))
	}
}
