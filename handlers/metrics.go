package handlers

import (
	"encoding/json"
	"net/http"
	"ustawka/metrics"
)

// MetricsHandler handles metrics endpoint
func MetricsHandler(w http.ResponseWriter, r *http.Request) {
	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")

	// Get metrics
	m := metrics.GetMetrics()

	// Encode metrics as JSON
	if err := json.NewEncoder(w).Encode(m); err != nil {
		http.Error(w, "Failed to encode metrics", http.StatusInternalServerError)
		return
	}
}
