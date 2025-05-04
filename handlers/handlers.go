package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
	"ustawka/sejm"

	"github.com/go-chi/chi/v5"
)

type KanbanData struct {
	Obowiazujace []sejm.Act
	Pending      []sejm.Act
	Uchylone     []sejm.Act
}

type Handler struct {
	templates  *template.Template
	sejmClient *sejm.Client
}

func NewHandler(templates *template.Template, sejmClient *sejm.Client) *Handler {
	return &Handler{
		templates:  templates,
		sejmClient: sejmClient,
	}
}

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	err := h.templates.ExecuteTemplate(w, "base.html", nil)
	if err != nil {
		slog.Error("Error executing template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) GetYears(w http.ResponseWriter, r *http.Request) {
	currentYear := time.Now().Year()
	years := make([]int, 0)

	// Check each year from 2021 to current year
	for year := 2021; year <= currentYear; year++ {
		acts, err := h.sejmClient.GetActs(year)
		if err != nil {
			slog.Error("Error checking year", "year", year, "error", err)
			continue
		}
		if len(acts) > 0 {
			years = append(years, year)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(years); err != nil {
		slog.Error("Error encoding JSON", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) GetActs(w http.ResponseWriter, r *http.Request) {
	year := chi.URLParam(r, "year")

	yearInt, err := strconv.Atoi(year)
	if err != nil {
		slog.Error("Invalid year format", "year", year, "error", err)
		http.Error(w, "Invalid year format", http.StatusBadRequest)
		return
	}

	// Check if the year is available
	acts, err := h.sejmClient.GetActs(yearInt)
	if err != nil {
		slog.Error("Error fetching acts", "year", yearInt, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(acts) == 0 {
		http.Error(w, fmt.Sprintf("No data available for year %d", yearInt), http.StatusNotFound)
		return
	}

	// Organize acts by status
	data := KanbanData{
		Obowiazujace: make([]sejm.Act, 0),
		Pending:      make([]sejm.Act, 0),
		Uchylone:     make([]sejm.Act, 0),
	}

	for _, act := range acts {
		status := strings.ToLower(strings.TrimSpace(act.Status))

		switch status {
		case "obowiązujący", "obowiazujacy":
			data.Obowiazujace = append(data.Obowiazujace, act)
		case "uchylony":
			data.Uchylone = append(data.Uchylone, act)
		default:
			if status == "" {
				act.Status = "W przygotowaniu"
			}
			data.Pending = append(data.Pending, act)
		}
	}

	// If the request is from HTMX, render the Kanban template
	if r.Header.Get("HX-Request") == "true" {
		err := h.templates.ExecuteTemplate(w, "kanban", data)
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
		slog.Error("Error encoding JSON", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) GetActDetails(w http.ResponseWriter, r *http.Request) {
	year := chi.URLParam(r, "year")
	position := chi.URLParam(r, "position")

	actID := fmt.Sprintf("DU/%s/%s", year, position)
	details, err := h.sejmClient.GetActDetails(actID)
	if err != nil {
		slog.Error("Error fetching act details", "actID", actID, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(details); err != nil {
		slog.Error("Error encoding JSON", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
