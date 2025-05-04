package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
	"ustawka/sejm"
)

// SejmClient defines the interface for Sejm API operations
type SejmClient interface {
	GetActs(ctx context.Context, year int) ([]sejm.Act, error)
	GetActDetails(ctx context.Context, actID string) (*sejm.ActDetails, error)
}

type ActService struct {
	sejmClient SejmClient
	timeout    time.Duration
}

type KanbanData struct {
	Obowiazujace []sejm.Act
	Pending      []sejm.Act
	Uchylone     []sejm.Act
}

// Default timeout for API requests
const defaultTimeout = 5 * time.Second

func NewActService(client SejmClient) *ActService {
	timeout := defaultTimeout
	if timeoutStr := os.Getenv("SEJM_API_TIMEOUT"); timeoutStr != "" {
		if duration, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = duration
			slog.Info("Using custom API timeout", "timeout", timeout)
		} else {
			slog.Warn("Invalid SEJM_API_TIMEOUT value, using default", "value", timeoutStr, "default", defaultTimeout)
		}
	}

	return &ActService{
		sejmClient: client,
		timeout:    timeout,
	}
}

// GetAvailableYears returns a list of years that have acts available
func (s *ActService) GetAvailableYears(ctx context.Context) ([]int, error) {
	currentYear := time.Now().Year()
	years := make([]int, 0)

	// Check each year from 2021 to current year
	for year := 2021; year <= currentYear; year++ {
		// Create a new context with timeout for each year check
		yearCtx, cancel := context.WithTimeout(ctx, s.timeout)
		defer cancel()

		acts, err := s.sejmClient.GetActs(yearCtx, year)
		if err != nil {
			if err == context.DeadlineExceeded {
				slog.Warn("Timeout checking year", "year", year, "timeout", s.timeout)
			} else {
				slog.Error("Error checking year", "year", year, "error", err)
			}
			continue
		}
		if len(acts) > 0 {
			years = append(years, year)
		}
	}

	return years, nil
}

// GetActsByYear retrieves acts for a specific year and organizes them for the Kanban board
func (s *ActService) GetActsByYear(ctx context.Context, year int) (*KanbanData, error) {
	acts, err := s.sejmClient.GetActs(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch acts: %w", err)
	}

	if len(acts) == 0 {
		return nil, fmt.Errorf("no data available for year %d", year)
	}

	// Organize acts by status for Kanban board
	data := &KanbanData{
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

	return data, nil
}

// GetActDetails retrieves detailed information about a specific act
func (s *ActService) GetActDetails(ctx context.Context, year, position string) (*sejm.ActDetails, error) {
	actID := fmt.Sprintf("DU/%s/%s", year, position)
	details, err := s.sejmClient.GetActDetails(ctx, actID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch act details: %w", err)
	}

	return details, nil
}
