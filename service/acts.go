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
	GetParliamentaryProcesses(ctx context.Context, term int) ([]sejm.ParliamentaryProcess, error)
	GetParliamentaryProcess(ctx context.Context, term int, processNumber string) (*sejm.ParliamentaryProcess, error)
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
	GetEnhancedActByID(ctx context.Context, actID string) (*sejm.EnhancedAct, error)
	
	// Parliamentary Process operations
	GetParliamentaryProcesses(ctx context.Context, term int) ([]sejm.ParliamentaryProcess, error)
	StoreParliamentaryProcesses(ctx context.Context, term int, processes []sejm.ParliamentaryProcess) error
	GetParliamentaryProcessByNumber(ctx context.Context, term int, 
		processNumber string) (*sejm.ParliamentaryProcess, error)
	GetParliamentaryProcessCacheAge(ctx context.Context, term int) (time.Duration, error)
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
	
	// First try parliamentary processes
	if data := s.tryParliamentaryProcesses(ctx, year); data != nil {
		return data, nil
	}
	
	// Fall back to enhanced acts or basic acts
	return s.fallbackToLegacyData(ctx, year)
}

func (s *ActService) tryParliamentaryProcesses(ctx context.Context, year int) *BoardData {
	slog.Debug("Attempting to use parliamentary processes", "year", year)
	
	parliamentaryData, err := s.GetParliamentaryProcessesByYear(ctx, year)
	if err != nil {
		slog.Debug("Parliamentary processes failed", "year", year, "error", err)
		return nil
	}
	if parliamentaryData == nil {
		slog.Debug("Parliamentary processes returned nil", "year", year)
		return nil
	}
	
	if s.hasIntermediateStages(parliamentaryData) {
		s.logParliamentaryDataUsage(year, parliamentaryData)
		return parliamentaryData
	}
	
	slog.Debug("Parliamentary processes have no intermediate stages", "year", year)
	return nil
}

func (*ActService) hasIntermediateStages(data *BoardData) bool {
	return len(data.Submitted) > 0 ||
		len(data.CommitteeWork) > 0 ||
		len(data.SejmReadings) > 0 ||
		len(data.SenateReview) > 0 ||
		len(data.PresidentialReview) > 0
}

func (*ActService) logParliamentaryDataUsage(year int, data *BoardData) {
	slog.Info("Using parliamentary processes data", "year", year,
		"submitted", len(data.Submitted),
		"committee", len(data.CommitteeWork),
		"sejm", len(data.SejmReadings),
		"senate", len(data.SenateReview),
		"presidential", len(data.PresidentialReview))
}

func (s *ActService) fallbackToLegacyData(ctx context.Context, year int) (*BoardData, error) {
	enhancedActs, err := s.db.GetEnhancedActs(ctx, year)
	if err != nil {
		slog.Debug("Enhanced acts not available, falling back to basic acts", "year", year, "error", err)
	}
	
	if len(enhancedActs) == 0 {
		return s.getBasicActsData(ctx, year)
	}
	
	return organizeEnhancedActsByStatus(enhancedActs), nil
}

func (s *ActService) getBasicActsData(ctx context.Context, year int) (*BoardData, error) {
	acts, err := s.getActsForYear(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch acts: %w", err)
	}
	
	if len(acts) == 0 {
		return nil, fmt.Errorf("no data available for year %d", year)
	}
	
	return organizeActsByStatus(acts), nil
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

// GetEnhancedActDetails retrieves enhanced details for a specific act
func (s *ActService) GetEnhancedActDetails(ctx context.Context, year, position string) (any, error) {
	metrics.IncrementAPI()
	
	// Always return ActDetails since the template expects ActDetails fields
	// TODO: Create a unified template that works with both EnhancedAct and ActDetails
	return s.GetActDetails(ctx, year, position)
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
	detailedStatus := getEffectiveDetailedStatus(act)
	appendActToColumn(data, act, detailedStatus)
}

// getEffectiveDetailedStatus returns the detailed status, falling back to mapped basic status if needed
func getEffectiveDetailedStatus(act sejm.EnhancedAct) string {
	detailedStatus := strings.ToLower(strings.TrimSpace(act.DetailedStatus))
	
	if detailedStatus == "" || detailedStatus == "unknown" {
		return mapBasicStatusToDetailed(act.Status)
	}
	
	return detailedStatus
}

// appendActToColumn appends the act to the appropriate column based on detailed status
func appendActToColumn(data *BoardData, act sejm.EnhancedAct, detailedStatus string) {
	switch {
	case detailedStatus == "submitted":
		data.Submitted = append(data.Submitted, act)
	case isCommitteeStatus(detailedStatus):
		data.CommitteeWork = append(data.CommitteeWork, act)
	case isSejmReadingStatus(detailedStatus):
		data.SejmReadings = append(data.SejmReadings, act)
	case isSenateStatus(detailedStatus):
		data.SenateReview = append(data.SenateReview, act)
	case isPresidentialStatus(detailedStatus):
		data.PresidentialReview = append(data.PresidentialReview, act)
	case detailedStatus == "published":
		data.Published = append(data.Published, act)
	case detailedStatus == "in_force":
		data.InForce = append(data.InForce, act)
	default:
		data.Submitted = append(data.Submitted, act)
	}
}

// isCommitteeStatus checks if status indicates committee work
func isCommitteeStatus(status string) bool {
	return status == "committee_first_reading" || status == "committee_work"
}

// isSejmReadingStatus checks if status indicates Sejm readings
func isSejmReadingStatus(status string) bool {
	return status == "second_reading" || status == "third_reading"
}

// isSenateStatus checks if status indicates Senate review
func isSenateStatus(status string) bool {
	return status == "senate_review" || status == "senate_accepted" || 
		   status == "senate_amended" || status == "senate_rejected"
}

// isPresidentialStatus checks if status indicates Presidential review
func isPresidentialStatus(status string) bool {
	return status == "presidential_review" || status == "presidential_signed" || 
		   status == "presidential_veto"
}

// statusMappings defines the mapping from basic Polish status to detailed status
var statusMappings = map[string]string{
	"obowiązujący":              "in_force",
	"obowiazujacy":              "in_force",
	"akt posiada tekst jednolity": "in_force",
	"akt objęty tekstem jednolitym": "in_force", 
	"tekst jednolity":           "in_force",
	"uchylony":                  "repealed",
	"uznany za uchylony":        "repealed",
	"wygaśnięcie aktu":          "repealed",
	"wygasniecie aktu":          "repealed",
	"akt jednorazowy":           "in_force",
	"akt indywidualny":          "in_force",
	"bez statusu":               "published",
	"w przygotowaniu":           "submitted",
	"projekt":                   "submitted",
	"w komisji":                 "committee_work",
	"komisja":                   "committee_work",
	"ii czytanie":               "second_reading",
	"drugie czytanie":           "second_reading",
	"iii czytanie":              "third_reading",
	"trzecie czytanie":          "third_reading",
	"w senacie":                 "senate_review",
	"senat":                     "senate_review",
	"u prezydenta":              "presidential_review",
	"prezydent":                 "presidential_review",
	"opublikowany":              "published",
}

// mapBasicStatusToDetailed maps basic act status to detailed status for better categorization
func mapBasicStatusToDetailed(basicStatus string) string {
	status := strings.ToLower(strings.TrimSpace(basicStatus))
	
	if detailedStatus, exists := statusMappings[status]; exists {
		return detailedStatus
	}
	
	return "submitted"
}

// GetParliamentaryProcessesByYear retrieves parliamentary processes for a specific year 
// and organizes them for the board
func (s *ActService) GetParliamentaryProcessesByYear(ctx context.Context, year int) (*BoardData, error) {
	metrics.IncrementAPI()
	
	// Determine term based on year 
	// Note: Parliamentary process API appears to only contain procedural processes at term transitions
	// Term 10: November 2023 only (procedural), Term 11: currently empty
	term := 10 // Default to 10th term
	if year >= 2024 {
		term = 11
	}
	
	slog.Debug("Fetching parliamentary processes", "year", year, "term", term)
	
	// Try to get parliamentary processes from cache first
	processes, err := s.getParliamentaryProcessesForTerm(ctx, term)
	if err != nil {
		slog.Debug("Failed to fetch parliamentary processes", "year", year, "term", term, "error", err)
		return nil, fmt.Errorf("failed to fetch parliamentary processes: %w", err)
	}
	
	slog.Debug("Retrieved parliamentary processes", "year", year, "term", term, "total_count", len(processes))
	
	// Convert processes to enhanced acts and filter by year
	enhancedActs := make([]sejm.EnhancedAct, 0)
	for _, process := range processes {
		enhanced := sejm.ConvertParliamentaryProcessToEnhancedAct(&process)
		slog.Debug("Converted process", "process_number", process.Number, 
			"enhanced_year", enhanced.Year, "target_year", year)
		if enhanced.Year == year {
			enhancedActs = append(enhancedActs, *enhanced)
		}
	}
	
	slog.Debug("Filtered parliamentary processes by year", "year", year, "matched_count", len(enhancedActs))
	
	if len(enhancedActs) == 0 {
		slog.Debug("No parliamentary processes matched year filter", "year", year, "term", term)
		return nil, fmt.Errorf("no parliamentary processes available for year %d", year)
	}
	
	return organizeEnhancedActsByStatus(enhancedActs), nil
}

// getParliamentaryProcessesForTerm retrieves parliamentary processes for a specific term from cache or API
func (s *ActService) getParliamentaryProcessesForTerm(ctx context.Context, 
	term int) ([]sejm.ParliamentaryProcess, error) {
	// Check cache first
	cacheAge, err := s.db.GetParliamentaryProcessCacheAge(ctx, term)
	if err != nil {
		slog.Error("Error checking parliamentary process cache age", "term", term, "error", err)
		// Continue to fetch from API if cache check fails
	}
	
	var processes []sejm.ParliamentaryProcess
	if err == nil && cacheAge < s.cacheTTL {
		// Use cached data
		processes, err = s.db.GetParliamentaryProcesses(ctx, term)
		if err != nil {
			slog.Error("Error reading parliamentary processes from cache", "term", term, "error", err)
			// Continue to fetch from API if cache read fails
		} else {
			slog.Debug("Using cached parliamentary processes", "term", term, 
				"count", len(processes), "cache_age", cacheAge)
			metrics.IncrementCacheHit()
		}
	}
	
	if len(processes) == 0 {
		slog.Debug("Cache miss or empty, fetching from API", "term", term)
		return s.fetchAndCacheParliamentaryProcesses(ctx, term)
	}
	
	return processes, nil
}

// fetchAndCacheParliamentaryProcesses fetches parliamentary processes from API and stores them in cache
func (s *ActService) fetchAndCacheParliamentaryProcesses(ctx context.Context, 
	term int) ([]sejm.ParliamentaryProcess, error) {
	metrics.IncrementCacheMiss()
	// Create a new context with timeout only for the API call
	apiCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	
	// Fetch from API and update cache
	processes, err := s.sejmClient.GetParliamentaryProcesses(apiCtx, term)
	if err != nil {
		if err == context.DeadlineExceeded {
			slog.Warn("Timeout fetching parliamentary processes", "term", term, "timeout", s.timeout)
		} else {
			slog.Error("Error fetching parliamentary processes", "term", term, "error", err)
		}
		return nil, err
	}
	
	metrics.IncrementSejmAPI()
	
	// Store in cache using the original context
	if err := s.db.StoreParliamentaryProcesses(ctx, term, processes); err != nil {
		slog.Error("Error storing parliamentary processes in cache", "term", term, "error", err)
		// Continue even if cache store fails
	}
	
	return processes, nil
}
