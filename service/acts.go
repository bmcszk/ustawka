package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
	"ustawka/metrics"
	"ustawka/sejm"
)

// SejmClient defines the interface for Sejm API operations
type SejmClient interface {
	GetActs(ctx context.Context, year int) ([]sejm.Act, error)
	GetActDetails(ctx context.Context, actID string) (*sejm.ActDetails, error)
}

// Database defines the interface for database operations
type Database interface {
	GetActs(ctx context.Context, year int) ([]sejm.Act, error)
	StoreActs(ctx context.Context, year int, acts []sejm.Act) error
	GetActDetails(ctx context.Context, actID string) (*sejm.ActDetails, error)
	StoreActDetails(ctx context.Context, details *sejm.ActDetails) error
	GetCacheAge(ctx context.Context, year int) (time.Duration, error)
	
	// Enhanced Act operations
	GetEnhancedActs(ctx context.Context, year int) ([]sejm.EnhancedAct, error)
	StoreEnhancedAct(ctx context.Context, act *sejm.EnhancedAct) error
}

// ActService provides business logic for legislative acts
type ActService struct {
	sejmClient SejmClient
	db         Database
	timeout    time.Duration
	cacheTTL   time.Duration
}

// BoardData organizes acts by status for the enhanced Kanban board view
type BoardData struct {
	// Legacy fields for backward compatibility
	Obowiazujace []sejm.Act
	Pending      []sejm.Act
	Uchylone     []sejm.Act
	
	// Enhanced lifecycle status columns
	Submitted          []sejm.EnhancedAct
	CommitteeWork      []sejm.EnhancedAct
	SejmReadings       []sejm.EnhancedAct
	SenateReview       []sejm.EnhancedAct
	PresidentialReview []sejm.EnhancedAct
	Published          []sejm.EnhancedAct
	InForce            []sejm.EnhancedAct
}

// Default values
const (
	defaultTimeout  = 5 * time.Second
	defaultCacheTTL = 24 * time.Hour
)

// NewActService creates a new ActService with configured dependencies
func NewActService(client SejmClient, database Database) *ActService {
	// Configure timeout
	timeout := defaultTimeout
	if timeoutStr := os.Getenv("SEJM_API_TIMEOUT"); timeoutStr != "" {
		if duration, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = duration
			slog.Info("Using custom API timeout", "timeout", timeout)
		} else {
			slog.Warn("Invalid SEJM_API_TIMEOUT value, using default", "value", timeoutStr, "default", defaultTimeout)
		}
	}

	// Configure cache TTL
	cacheTTL := defaultCacheTTL
	if ttlStr := os.Getenv("SEJM_CACHE_TTL"); ttlStr != "" {
		if duration, err := time.ParseDuration(ttlStr); err == nil {
			cacheTTL = duration
			slog.Info("Using custom cache TTL", "ttl", cacheTTL)
		} else {
			slog.Warn("Invalid SEJM_CACHE_TTL value, using default", "value", ttlStr, "default", defaultCacheTTL)
		}
	}

	return &ActService{
		sejmClient: client,
		db:         database,
		timeout:    timeout,
		cacheTTL:   cacheTTL,
	}
}

// NewActServiceWithConfig creates a new ActService with explicit configuration (primarily for testing)
func NewActServiceWithConfig(client SejmClient, database Database, timeout, cacheTTL time.Duration) *ActService {
	return &ActService{
		sejmClient: client,
		db:         database,
		timeout:    timeout,
		cacheTTL:   cacheTTL,
	}
}

// GetAvailableYears returns a list of years that have acts available
func (s *ActService) GetAvailableYears(ctx context.Context) ([]int, error) {
	metrics.IncrementAPI()
	currentYear := time.Now().Year()
	years := make([]int, 0)
	var lastErr error

	// Check each year from 2021 to current year
	for year := 2021; year <= currentYear; year++ {
		acts, err := s.getActsForYear(ctx, year)
		if err != nil {
			lastErr = err
			continue
		}

		if len(acts) > 0 {
			years = append(years, year)
		}
	}

	return validateYearResults(years, lastErr)
}

// getActsForYear retrieves acts for a specific year from cache or API
func (s *ActService) getActsForYear(ctx context.Context, year int) ([]sejm.Act, error) {
	// Check cache first
	cacheAge, err := s.db.GetCacheAge(ctx, year)
	if err != nil {
		slog.Error("Error checking cache age", "year", year, "error", err)
		// Continue to fetch from API if cache check fails
	}

	var acts []sejm.Act
	if err == nil && cacheAge < s.cacheTTL {
		// Use cached data
		acts, err = s.db.GetActs(ctx, year)
		if err != nil {
			slog.Error("Error reading from cache", "year", year, "error", err)
			// Continue to fetch from API if cache read fails
		} else {
			metrics.IncrementCacheHit()
		}
	}

	if len(acts) == 0 {
		return s.fetchAndCacheActs(ctx, year)
	}

	return acts, nil
}

// validateYearResults validates and returns the final year results
func validateYearResults(years []int, lastErr error) ([]int, error) {
	// If we have no years and there was an error, return the error
	if len(years) == 0 && lastErr != nil {
		return nil, fmt.Errorf("failed to fetch any years: %w", lastErr)
	}

	// If we have no years but no error, return a specific error
	if len(years) == 0 {
		return nil, errors.New("no data available for any year")
	}

	return years, nil
}

// fetchAndCacheActs fetches acts from API and stores them in cache
func (s *ActService) fetchAndCacheActs(ctx context.Context, year int) ([]sejm.Act, error) {
	metrics.IncrementCacheMiss()
	// Create a new context with timeout only for the API call
	apiCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	
	// Fetch from API and update cache
	acts, err := s.sejmClient.GetActs(apiCtx, year)
	if err != nil {
		if err == context.DeadlineExceeded {
			slog.Warn("Timeout checking year", "year", year, "timeout", s.timeout)
		} else {
			slog.Error("Error checking year", "year", year, "error", err)
		}
		return nil, err
	}

	metrics.IncrementSejmAPI()

	// Store in cache using the original context
	if err := s.db.StoreActs(ctx, year, acts); err != nil {
		slog.Error("Error storing in cache", "year", year, "error", err)
		// Continue even if cache store fails
	}

	return acts, nil
}

// GetActsByYear retrieves acts for a specific year and organizes them for the board
func (s *ActService) GetActsByYear(ctx context.Context, year int) (*BoardData, error) {
	metrics.IncrementAPI()
	
	// Try to get enhanced acts first
	enhancedActs, err := s.db.GetEnhancedActs(ctx, year)
	if err != nil {
		slog.Debug("Enhanced acts not available, falling back to basic acts", "year", year, "error", err)
	}
	
	// If no enhanced acts available, fall back to basic acts
	if len(enhancedActs) == 0 {
		acts, err := s.getActsForYear(ctx, year)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch acts: %w", err)
		}

		if len(acts) == 0 {
			return nil, fmt.Errorf("no data available for year %d", year)
		}

		return organizeActsByStatus(acts), nil
	}
	
	return organizeEnhancedActsByStatus(enhancedActs), nil
}

// organizeActsByStatus organizes acts by their status for the board view
func organizeActsByStatus(acts []sejm.Act) *BoardData {
	data := &BoardData{
		Obowiazujace: make([]sejm.Act, 0),
		Pending:      make([]sejm.Act, 0),
		Uchylone:     make([]sejm.Act, 0),
		
		// Initialize enhanced status slices to empty, not nil
		Submitted:          make([]sejm.EnhancedAct, 0),
		CommitteeWork:      make([]sejm.EnhancedAct, 0),
		SejmReadings:       make([]sejm.EnhancedAct, 0),
		SenateReview:       make([]sejm.EnhancedAct, 0),
		PresidentialReview: make([]sejm.EnhancedAct, 0),
		Published:          make([]sejm.EnhancedAct, 0),
		InForce:            make([]sejm.EnhancedAct, 0),
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

	return data
}

// GetActDetails retrieves details for a specific act
func (s *ActService) GetActDetails(ctx context.Context, year, position string) (*sejm.ActDetails, error) {
	metrics.IncrementAPI()
	actID := fmt.Sprintf("DU/%s/%s", year, position)

	// Check cache first
	details, err := s.db.GetActDetails(ctx, actID)
	if err == nil && details != nil {
		metrics.IncrementCacheHit()
		return details, nil
	}

	metrics.IncrementCacheMiss()
	// Create a new context with timeout only for the API call
	apiCtx, cancel := context.WithTimeout(ctx, s.timeout)
	// Fetch from API
	details, err = s.sejmClient.GetActDetails(apiCtx, actID)
	cancel() // Cancel right after the API call

	if err != nil {
		return nil, fmt.Errorf("failed to fetch act details: %w", err)
	}

	metrics.IncrementSejmAPI()

	// Store in cache using the original context
	if err := s.db.StoreActDetails(ctx, details); err != nil {
		slog.Error("Error storing in cache", "act_id", actID, "error", err)
		// Continue even if cache store fails
	}

	return details, nil
}

// organizeEnhancedActsByStatus organizes enhanced acts by their detailed status for the enhanced board view
func organizeEnhancedActsByStatus(acts []sejm.EnhancedAct) *BoardData {
	data := &BoardData{
		// Initialize legacy slices for compatibility
		Obowiazujace: make([]sejm.Act, 0),
		Pending:      make([]sejm.Act, 0),
		Uchylone:     make([]sejm.Act, 0),
		
		// Initialize enhanced status slices
		Submitted:          make([]sejm.EnhancedAct, 0),
		CommitteeWork:      make([]sejm.EnhancedAct, 0),
		SejmReadings:       make([]sejm.EnhancedAct, 0),
		SenateReview:       make([]sejm.EnhancedAct, 0),
		PresidentialReview: make([]sejm.EnhancedAct, 0),
		Published:          make([]sejm.EnhancedAct, 0),
		InForce:            make([]sejm.EnhancedAct, 0),
	}

	for _, act := range acts {
		addToLegacyColumns(data, act)
		addToEnhancedColumns(data, act)
	}

	return data
}

// addToLegacyColumns adds acts to legacy columns for backward compatibility
func addToLegacyColumns(data *BoardData, act sejm.EnhancedAct) {
	basicAct := sejm.Act{
		ID:        act.ID,
		Title:     act.Title,
		Status:    act.Status,
		Published: act.Published,
		Position:  act.Position,
		Year:      act.Year,
		Type:      act.Type,
		Address:   act.Address,
	}
	
	status := strings.ToLower(strings.TrimSpace(act.Status))
	switch status {
	case "obowiązujący", "obowiazujacy":
		data.Obowiazujace = append(data.Obowiazujace, basicAct)
	case "uchylony":
		data.Uchylone = append(data.Uchylone, basicAct)
	default:
		data.Pending = append(data.Pending, basicAct)
	}
}

// addToEnhancedColumns adds acts to enhanced status columns
func addToEnhancedColumns(data *BoardData, act sejm.EnhancedAct) {
	detailedStatus := strings.ToLower(strings.TrimSpace(act.DetailedStatus))
	switch detailedStatus {
	case "submitted":
		data.Submitted = append(data.Submitted, act)
	case "committee_first_reading", "committee_work":
		data.CommitteeWork = append(data.CommitteeWork, act)
	case "second_reading", "third_reading":
		data.SejmReadings = append(data.SejmReadings, act)
	case "senate_review", "senate_accepted", "senate_amended", "senate_rejected":
		data.SenateReview = append(data.SenateReview, act)
	case "presidential_review", "presidential_signed", "presidential_veto":
		data.PresidentialReview = append(data.PresidentialReview, act)
	case "published":
		data.Published = append(data.Published, act)
	case "in_force":
		data.InForce = append(data.InForce, act)
	default:
		// Default to submitted if status is unknown
		data.Submitted = append(data.Submitted, act)
	}
}
