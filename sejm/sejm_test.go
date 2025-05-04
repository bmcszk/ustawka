package sejm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestRealAPI tests against the actual Sejm API
func TestRealAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real API test in short mode")
	}

	client := NewClient()

	// Test GetActs for 2024
	t.Run("2024", func(t *testing.T) {
		acts, err := client.GetActs(2024)
		if err != nil {
			t.Fatalf("Failed to fetch acts from real API for 2024: %v", err)
		}

		if len(acts) == 0 {
			t.Fatal("Expected non-empty acts list from real API for 2024")
		}

		t.Logf("Sample act from 2024: %+v", acts[0])

		// Test GetActDetails with the first act
		details, err := client.GetActDetails(acts[0].ID)
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
		acts, err := client.GetActs(2021)
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
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request
		if r.URL.Path != "/acts/DU/2024" {
			t.Errorf("Expected to request '/acts/DU/2024', got: %s", r.URL.Path)
		}

		// Return test data matching real API structure
		response := APIResponse{
			Items: []Act{
				{
					ID:        "DU/2024/1",
					Title:     "Rozporządzenie Ministra Obrony Narodowej z dnia 8 grudnia 2023 r. w sprawie należności pieniężnych żołnierzy zawodowych pełniących służbę poza granicami państwa",
					Status:    "obowiązujący",
					Published: "2024-01-02",
					Position:  1,
					Year:      2024,
					Type:      "Rozporządzenie",
					Address:   "WDU20240000001",
				},
				{
					ID:        "DU/2024/2",
					Title:     "Obwieszczenie Ministra Finansów z dnia 15 listopada 2023 r. w sprawie ogłoszenia jednolitego tekstu rozporządzenia Ministra Finansów, Funduszy i Polityki Regionalnej w sprawie sposobu, trybu oraz warunków prowadzenia działalności przez towarzystwa funduszy inwestycyjnych",
					Status:    "obowiązujący",
					Published: "2024-01-02",
					Position:  2,
					Year:      2024,
					Type:      "Obwieszczenie",
					Address:   "WDU20240000002",
				},
				{
					ID:        "DU/2024/3",
					Title:     "Rozporządzenie Ministra Funduszy i Polityki Regionalnej z dnia 28 grudnia 2023 r. w sprawie udzielania pomocy regionalnej na rzecz rozwoju obszarów miejskich w ramach regionalnych programów na lata 2021-2027",
					Status:    "obowiązujący",
					Published: "2024-01-02",
					Position:  3,
					Year:      2024,
					Type:      "Rozporządzenie",
					Address:   "WDU20240000003",
				},
			},
			Offset:     0,
			TotalCount: 3,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create a client with the test server URL
	client := NewClient()
	client.baseURL = server.URL

	// Test GetActs
	acts, err := client.GetActs(2024)
	if err != nil {
		t.Fatalf("Failed to fetch acts: %v", err)
	}

	if len(acts) != 3 {
		t.Errorf("Expected 3 acts, got %d", len(acts))
	}

	// Verify the first act matches real API structure
	expectedAct := Act{
		ID:        "DU/2024/1",
		Title:     "Rozporządzenie Ministra Obrony Narodowej z dnia 8 grudnia 2023 r. w sprawie należności pieniężnych żołnierzy zawodowych pełniących służbę poza granicami państwa",
		Status:    "obowiązujący",
		Published: "2024-01-02",
		Position:  1,
		Year:      2024,
		Type:      "Rozporządzenie",
		Address:   "WDU20240000001",
	}

	if acts[0] != expectedAct {
		t.Errorf("Expected act %+v, got %+v", expectedAct, acts[0])
	}

	// Log sample act for debugging
	t.Logf("Sample act from mock API: %+v", acts[0])
}

func TestGetActDetails(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request
		if r.URL.Path != "/acts/DU/2024/1" {
			t.Errorf("Expected to request '/acts/DU/2024/1', got: %s", r.URL.Path)
		}

		// Return test data matching real API structure
		response := ActDetails{
			ID:        "DU/2024/1",
			Title:     "Rozporządzenie Ministra Obrony Narodowej z dnia 8 grudnia 2023 r. w sprawie należności pieniężnych żołnierzy zawodowych pełniących służbę poza granicami państwa",
			Status:    "obowiązujący",
			Published: "2024-01-02",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create a client with the test server URL
	client := NewClient()
	client.baseURL = server.URL

	// Test GetActDetails
	details, err := client.GetActDetails("DU/2024/1")
	if err != nil {
		t.Fatalf("Failed to fetch act details: %v", err)
	}

	// Verify the details match real API structure
	expectedDetails := ActDetails{
		ID:        "DU/2024/1",
		Title:     "Rozporządzenie Ministra Obrony Narodowej z dnia 8 grudnia 2023 r. w sprawie należności pieniężnych żołnierzy zawodowych pełniących służbę poza granicami państwa",
		Status:    "obowiązujący",
		Published: "2024-01-02",
	}

	if *details != expectedDetails {
		t.Errorf("Expected details %+v, got %+v", expectedDetails, details)
	}

	// Log sample details for debugging
	t.Logf("Sample act details from mock API: %+v", details)
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
	_, err := client.GetActs(2024)
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
	_, err := client.GetActDetails("DU/2024/1")
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
