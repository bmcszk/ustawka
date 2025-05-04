package sejm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestRealAPI tests against the actual Sejm API
func TestRealAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real API test in short mode")
	}

	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test GetActs for 2024
	t.Run("2024", func(t *testing.T) {
		acts, err := client.GetActs(ctx, 2024)
		if err != nil {
			t.Fatalf("Failed to fetch acts from real API for 2024: %v", err)
		}

		if len(acts) == 0 {
			t.Fatal("Expected non-empty acts list from real API for 2024")
		}

		t.Logf("Sample act from 2024: %+v", acts[0])

		// Test GetActDetails with the first act
		details, err := client.GetActDetails(ctx, acts[0].ID)
		if err != nil {
			t.Fatalf("Failed to fetch act details from real API: %v", err)
		}

		if details == nil {
			t.Fatal("Expected non-nil act details from real API")
		}

		// Log sample act details for debugging
		t.Logf("Sample act details from real API: %+v", details)

		// Log the structure of the first few acts for debugging
		t.Log("Real API response structure:")
		for i := 0; i < 3 && i < len(acts); i++ {
			actJSON, _ := json.MarshalIndent(acts[i], "", "  ")
			t.Logf("Act %d:\n%s", i+1, string(actJSON))
		}
	})

	// Test GetActs for 2021
	t.Run("2021", func(t *testing.T) {
		acts, err := client.GetActs(ctx, 2021)
		if err != nil {
			t.Fatalf("Failed to fetch acts from real API for 2021: %v", err)
		}

		if len(acts) == 0 {
			t.Fatal("Expected non-empty acts list from real API for 2021")
		}

		// Verify that all acts are from 2021
		for i, act := range acts {
			if act.Year != 2021 {
				t.Errorf("Act at index %d has wrong year: got %d, want 2021", i, act.Year)
			}
		}

		t.Logf("Sample act from 2021: %+v", acts[0])
		t.Logf("Number of acts from 2021: %d", len(acts))
	})
}

// Unit tests below this line
func TestGetActs(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	acts, err := client.GetActs(ctx, 2021)
	if err != nil {
		t.Fatalf("Failed to get acts: %v", err)
	}

	if len(acts) == 0 {
		t.Fatal("Expected non-empty acts list")
	}

	// Verify first act structure
	firstAct := acts[0]
	if firstAct.ID == "" {
		t.Error("Expected non-empty ID")
	}
	if firstAct.Title == "" {
		t.Error("Expected non-empty Title")
	}
	if firstAct.Status == "" {
		t.Error("Expected non-empty Status")
	}
	if firstAct.Published == "" {
		t.Error("Expected non-empty Published date")
	}
	if firstAct.Position == 0 {
		t.Error("Expected non-zero Position")
	}
	if firstAct.Year != 2021 {
		t.Errorf("Expected Year to be 2021, got %d", firstAct.Year)
	}
	if firstAct.Type == "" {
		t.Error("Expected non-empty Type")
	}
	if firstAct.Address == "" {
		t.Error("Expected non-empty Address")
	}
}

func TestGetActDetails(t *testing.T) {
	client := NewClient()
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create a client with the test server URL
	client := NewClient()
	client.baseURL = server.URL

	// Test GetActs with error
	_, err := client.GetActs(context.Background(), 2024)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestGetActDetailsError(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create a client with the test server URL
	client := NewClient()
	client.baseURL = server.URL

	// Test GetActDetails with error
	_, err := client.GetActDetails(context.Background(), "DU/2024/1")
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestGetYearString(t *testing.T) {
	act := Act{Year: 2024}
	if act.GetYearString() != "2024" {
		t.Errorf("Expected '2024', got '%s'", act.GetYearString())
	}
}
