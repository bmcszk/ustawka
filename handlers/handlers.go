package handlers

import (
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"ustawka/service"

	"github.com/go-chi/chi/v5"
)

// Handler handles HTTP requests for the application
type Handler struct {
	templates     *template.Template
	actService    *service.ActService
	searchService *service.SearchService
}

// NewHandler creates a new Handler instance with dependencies
func NewHandler(templates *template.Template, actService *service.ActService, searchService *service.SearchService) *Handler {
	return &Handler{
		templates:     templates,
		actService:    actService,
		searchService: searchService,
	}
}

// Home serves the main application page
func (h *Handler) Home(w http.ResponseWriter, _ *http.Request) {
	err := h.templates.ExecuteTemplate(w, "base.html", nil)
	if err != nil {
		slog.Error("Error executing template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// HandleYears returns available years with legislative acts
func (h *Handler) HandleYears(w http.ResponseWriter, r *http.Request) {
	years, err := h.actService.GetAvailableYears(r.Context())
	if err != nil {
		slog.Error("Error getting available years", "error", err)
		http.Error(w, "Failed to get available years", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(years); err != nil {
		slog.Error("Error encoding JSON", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// HandleActs returns acts for a specific year, organized by status
func (h *Handler) HandleActs(w http.ResponseWriter, r *http.Request) {
	yearStr := chi.URLParam(r, "year")
	if yearStr == "" {
		http.Error(w, "Year parameter is required", http.StatusBadRequest)
		return
	}

	yearInt, err := strconv.Atoi(yearStr)
	if err != nil {
		http.Error(w, "Invalid year parameter", http.StatusBadRequest)
		return
	}

	data, err := h.actService.GetActsByYear(r.Context(), yearInt)
	if err != nil {
		slog.Error("Error fetching acts", "error", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// If the request is from HTMX, render the board template
	if r.Header.Get("HX-Request") == "true" {
		err := h.templates.ExecuteTemplate(w, "board", data)
		if err != nil {
			slog.Error("Error executing template", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		return
	}

	// Otherwise return JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("Error encoding response", "error", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// HandleActDetails returns detailed information about a specific act
func (h *Handler) HandleActDetails(w http.ResponseWriter, r *http.Request) {
	year := chi.URLParam(r, "year")
	position := chi.URLParam(r, "position")
	if year == "" || position == "" {
		http.Error(w, "Year and position parameters are required", http.StatusBadRequest)
		return
	}

	details, err := h.actService.GetEnhancedActDetails(r.Context(), year, position)
	if err != nil {
		slog.Error("Error fetching act details", "error", err)
		http.Error(w, "Failed to fetch act details", http.StatusInternalServerError)
		return
	}

	// If the request is from HTMX, render the act details template
	if r.Header.Get("HX-Request") == "true" {
		err := h.templates.ExecuteTemplate(w, "act_details", details)
		if err != nil {
			slog.Error("Error executing template", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		return
	}

	// Otherwise return JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(details); err != nil {
		slog.Error("Error encoding response", "error", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// ViewActDetails serves the act details page
func (h *Handler) ViewActDetails(w http.ResponseWriter, r *http.Request) {
	year := chi.URLParam(r, "year")
	position := chi.URLParam(r, "position")
	if year == "" || position == "" {
		http.Error(w, "Year and position parameters are required", http.StatusBadRequest)
		return
	}

	details, err := h.actService.GetActDetails(r.Context(), year, position)
	if err != nil {
		slog.Error("Error fetching act details", "error", err)
		http.Error(w, "Failed to fetch act details", http.StatusInternalServerError)
		return
	}

	err = h.templates.ExecuteTemplate(w, "base.html", details)
	if err != nil {
		slog.Error("Error executing template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// HandleSearch performs advanced search with filtering
func (h *Handler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	// Parse search criteria from query parameters
	criteria := service.ParseSearchCriteria(r.URL.Query())
	
	// Perform search
	result, err := h.searchService.SearchActs(r.Context(), criteria)
	if err != nil {
		slog.Error("Error performing search", "error", err)
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}
	
	// If the request is from HTMX, render the search results template
	if r.Header.Get("HX-Request") == "true" {
		err := h.templates.ExecuteTemplate(w, "search_results", result)
		if err != nil {
			slog.Error("Error executing search results template", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		return
	}
	
	// Otherwise return JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		slog.Error("Error encoding search results", "error", err)
		http.Error(w, "Failed to encode search results", http.StatusInternalServerError)
		return
	}
}

// HandleSearchSuggestions provides auto-complete suggestions
func (h *Handler) HandleSearchSuggestions(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	field := r.URL.Query().Get("field")
	
	if query == "" || field == "" {
		http.Error(w, "Query and field parameters are required", http.StatusBadRequest)
		return
	}
	
	suggestions, err := h.searchService.GetSearchSuggestions(r.Context(), query, field)
	if err != nil {
		slog.Error("Error getting search suggestions", "error", err)
		http.Error(w, "Failed to get suggestions", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(suggestions); err != nil {
		slog.Error("Error encoding suggestions", "error", err)
		http.Error(w, "Failed to encode suggestions", http.StatusInternalServerError)
		return
	}
}

// HandleSearchFacets returns available filter options
func (h *Handler) HandleSearchFacets(w http.ResponseWriter, r *http.Request) {
	// Get a basic search with no filters to generate facets
	criteria := &service.SearchCriteria{
		Limit: 0, // Don't return actual results, just facets
	}
	
	result, err := h.searchService.SearchActs(r.Context(), criteria)
	if err != nil {
		slog.Error("Error getting search facets", "error", err)
		http.Error(w, "Failed to get facets", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result.Facets); err != nil {
		slog.Error("Error encoding facets", "error", err)
		http.Error(w, "Failed to encode facets", http.StatusInternalServerError)
		return
	}
}

// WriteJSON writes a JSON response
func WriteJSON(w http.ResponseWriter, data any) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(data)
}
