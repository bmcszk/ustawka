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
	templates  *template.Template
	actService *service.ActService
}

// NewHandler creates a new Handler instance with dependencies
func NewHandler(templates *template.Template, actService *service.ActService) *Handler {
	return &Handler{
		templates:  templates,
		actService: actService,
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

	details, err := h.actService.GetActDetails(r.Context(), year, position)
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
