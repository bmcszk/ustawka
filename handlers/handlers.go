package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"time"
	"ustawka/service"

	"github.com/go-chi/chi/v5"
)

// Handler handles HTTP requests for the application
type Handler struct {
	templates         *template.Template
	actService        *service.ActService
	searchService     *service.SearchService
	comparisonService *service.ComparisonService
	exportService     *service.ExportService
}

// NewHandler creates a new Handler instance with dependencies
func NewHandler(
	templates *template.Template, 
	actService *service.ActService, 
	searchService *service.SearchService,
	comparisonService *service.ComparisonService,
	exportService *service.ExportService,
) *Handler {
	return &Handler{
		templates:         templates,
		actService:        actService,
		searchService:     searchService,
		comparisonService: comparisonService,
		exportService:     exportService,
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

	details, err := h.actService.GetEnhancedActDetails(r.Context(), year, position)
	if err != nil {
		slog.Error("Error fetching act details", "error", err)
		http.Error(w, "Failed to fetch act details", http.StatusInternalServerError)
		return
	}

	// Debug log to check the type
	slog.Info("ViewActDetails returning type", "type", fmt.Sprintf("%T", details))

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

// HandleCompareActs compares two acts and returns detailed differences
func (h *Handler) HandleCompareActs(w http.ResponseWriter, r *http.Request) {
	leftID := r.URL.Query().Get("left")
	rightID := r.URL.Query().Get("right")
	
	if leftID == "" || rightID == "" {
		http.Error(w, "Both 'left' and 'right' act IDs are required", http.StatusBadRequest)
		return
	}
	
	comparison, err := h.comparisonService.CompareActs(r.Context(), leftID, rightID)
	if err != nil {
		slog.Error("Error comparing acts", "error", err, "left", leftID, "right", rightID)
		http.Error(w, "Failed to compare acts", http.StatusInternalServerError)
		return
	}
	
	// If the request is from HTMX, render the comparison template
	if r.Header.Get("HX-Request") == "true" {
		err := h.templates.ExecuteTemplate(w, "comparison_results", comparison)
		if err != nil {
			slog.Error("Error executing comparison template", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		return
	}
	
	// Otherwise return JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(comparison); err != nil {
		slog.Error("Error encoding comparison results", "error", err)
		http.Error(w, "Failed to encode comparison results", http.StatusInternalServerError)
		return
	}
}

// HandleComparisonSuggestions returns suggested acts for comparison
func (h *Handler) HandleComparisonSuggestions(w http.ResponseWriter, r *http.Request) {
	actID := r.URL.Query().Get("act_id")
	
	if actID == "" {
		http.Error(w, "Act ID parameter is required", http.StatusBadRequest)
		return
	}
	
	suggestions, err := h.comparisonService.GetComparisonSuggestions(r.Context(), actID)
	if err != nil {
		slog.Error("Error getting comparison suggestions", "error", err, "act_id", actID)
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

// HandleExportActs exports acts data in various formats
func (h *Handler) HandleExportActs(w http.ResponseWriter, r *http.Request) {
	req := h.parseExportRequest(r)
	
	result, err := h.exportService.ExportActs(r.Context(), req)
	if err != nil {
		slog.Error("Error exporting acts", "error", err, "format", req.Format)
		http.Error(w, "Failed to export acts", http.StatusInternalServerError)
		return
	}
	
	h.writeExportResponse(w, result)
}

func (h *Handler) parseExportRequest(r *http.Request) *service.ExportRequest {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}
	
	req := &service.ExportRequest{
		Format:        service.ExportFormat(format),
		IncludeVoting: r.URL.Query().Get("include_voting") == "true",
		IncludeStages: r.URL.Query().Get("include_stages") == "true",
	}
	
	h.parseOptionalParameters(r, req)
	return req
}

func (h *Handler) parseOptionalParameters(r *http.Request, req *service.ExportRequest) {
	h.parseYearParameter(r, req)
	h.parseStatusParameter(r, req)
	h.parseTitleParameter(r, req)
	h.parseDateParameters(r, req)
}

func (*Handler) parseYearParameter(r *http.Request, req *service.ExportRequest) {
	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if year, err := strconv.Atoi(yearStr); err == nil {
			req.Year = &year
		}
	}
}

func (*Handler) parseStatusParameter(r *http.Request, req *service.ExportRequest) {
	if statuses := r.URL.Query()["status"]; len(statuses) > 0 {
		req.Status = statuses
	}
}

func (*Handler) parseTitleParameter(r *http.Request, req *service.ExportRequest) {
	if title := r.URL.Query().Get("title"); title != "" {
		req.Title = title
	}
}

func (*Handler) parseDateParameters(r *http.Request, req *service.ExportRequest) {
	if dateFromStr := r.URL.Query().Get("date_from"); dateFromStr != "" {
		if dateFrom, err := time.Parse("2006-01-02", dateFromStr); err == nil {
			req.DateFrom = &dateFrom
		}
	}
	
	if dateToStr := r.URL.Query().Get("date_to"); dateToStr != "" {
		if dateTo, err := time.Parse("2006-01-02", dateToStr); err == nil {
			req.DateTo = &dateTo
		}
	}
}

func (*Handler) writeExportResponse(w http.ResponseWriter, result *service.ExportResult) {
	w.Header().Set("Content-Type", result.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", result.Filename))
	w.Header().Set("Content-Length", strconv.Itoa(result.Size))
	
	if _, err := w.Write(result.Data); err != nil {
		slog.Error("Error writing export data", "error", err)
	}
}

// HandleExportComparison exports comparison results in various formats
func (h *Handler) HandleExportComparison(w http.ResponseWriter, r *http.Request) {
	// Get comparison parameters
	leftID := r.URL.Query().Get("left")
	rightID := r.URL.Query().Get("right")
	format := r.URL.Query().Get("format")
	
	if leftID == "" || rightID == "" {
		http.Error(w, "Both 'left' and 'right' act IDs are required", http.StatusBadRequest)
		return
	}
	
	if format == "" {
		format = "json" // Default format
	}
	
	// Get comparison data
	comparison, err := h.comparisonService.CompareActs(r.Context(), leftID, rightID)
	if err != nil {
		slog.Error("Error getting comparison for export", "error", err, "left", leftID, "right", rightID)
		http.Error(w, "Failed to get comparison data", http.StatusInternalServerError)
		return
	}
	
	// Export comparison
	result, err := h.exportService.ExportComparison(r.Context(), comparison, service.ExportFormat(format))
	if err != nil {
		slog.Error("Error exporting comparison", "error", err, "format", format)
		http.Error(w, "Failed to export comparison", http.StatusInternalServerError)
		return
	}
	
	// Set response headers
	w.Header().Set("Content-Type", result.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", result.Filename))
	w.Header().Set("Content-Length", strconv.Itoa(result.Size))
	
	// Write data
	if _, err := w.Write(result.Data); err != nil {
		slog.Error("Error writing export data", "error", err)
		return
	}
}

// WriteJSON writes a JSON response
func WriteJSON(w http.ResponseWriter, data any) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(data)
}
