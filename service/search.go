package service

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	"ustawka/sejm"
)

// SearchCriteria represents search and filter parameters
type SearchCriteria struct {
	// Text search
	Query          string   `json:"query"`
	TitleSearch    string   `json:"title_search"`
	InitiatorSearch string  `json:"initiator_search"`
	
	// Status filters
	Statuses       []string `json:"statuses"`
	DetailedStatuses []string `json:"detailed_statuses"`
	CurrentStages  []string `json:"current_stages"`
	
	// Date filters
	DateFrom       *time.Time `json:"date_from"`
	DateTo         *time.Time `json:"date_to"`
	StageFrom      *time.Time `json:"stage_from"`
	StageTo        *time.Time `json:"stage_to"`
	
	// Numeric filters
	YearFrom       *int     `json:"year_from"`
	YearTo         *int     `json:"year_to"`
	PositionFrom   *int     `json:"position_from"`
	PositionTo     *int     `json:"position_to"`
	DaysInStageMin *int     `json:"days_in_stage_min"`
	DaysInStageMax *int     `json:"days_in_stage_max"`
	
	// Voting filters
	HasSejmVotes   *bool    `json:"has_sejm_votes"`
	HasSenateVotes *bool    `json:"has_senate_votes"`
	VotingResult   string   `json:"voting_result"` // "passed", "failed", "pending"
	
	// Committee filters
	CommitteeCodes []string `json:"committee_codes"`
	
	// Tags and categorization
	Tags           []string `json:"tags"`
	InitiatorTypes []string `json:"initiator_types"`
	
	// Sorting and pagination
	SortBy         string   `json:"sort_by"`     // "title", "date", "position", "stage_date", "days_in_stage"
	SortOrder      string   `json:"sort_order"`  // "asc", "desc"
	Limit          int      `json:"limit"`
	Offset         int      `json:"offset"`
}

// SearchResult contains search results and metadata
type SearchResult struct {
	Acts         []sejm.EnhancedAct `json:"acts"`
	TotalCount   int                `json:"total_count"`
	FilteredCount int               `json:"filtered_count"`
	SearchTime   time.Duration      `json:"search_time"`
	Facets       SearchFacets       `json:"facets"`
}

// SearchFacets provides filter options based on current data
type SearchFacets struct {
	AvailableStatuses      []FacetItem `json:"available_statuses"`
	AvailableStages        []FacetItem `json:"available_stages"`
	AvailableInitiators    []FacetItem `json:"available_initiators"`
	AvailableCommittees    []FacetItem `json:"available_committees"`
	AvailableTags          []FacetItem `json:"available_tags"`
	YearRange              YearRange   `json:"year_range"`
	DaysInStageRange       Range       `json:"days_in_stage_range"`
}

// FacetItem represents a filter option with count
type FacetItem struct {
	Value string `json:"value"`
	Count int    `json:"count"`
	Label string `json:"label"`
}

// YearRange represents min/max years available
type YearRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

// Range represents min/max numeric values
type Range struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

// SearchService provides advanced search and filtering capabilities
type SearchService struct {
	db Database
}

// NewSearchService creates a new search service
func NewSearchService(db Database) *SearchService {
	return &SearchService{
		db: db,
	}
}

// SearchActs performs advanced search with filtering and sorting
func (s *SearchService) SearchActs(ctx context.Context, criteria *SearchCriteria) (*SearchResult, error) {
	startTime := time.Now()
	
	// Get all enhanced acts for the specified years
	var allActs []sejm.EnhancedAct
	
	// Determine years to search
	currentYear := time.Now().Year()
	yearFrom := currentYear - 2 // Default to last 3 years
	yearTo := currentYear + 1
	
	if criteria.YearFrom != nil {
		yearFrom = *criteria.YearFrom
	}
	if criteria.YearTo != nil {
		yearTo = *criteria.YearTo
	}
	
	// Collect acts from all relevant years
	for year := yearFrom; year <= yearTo; year++ {
		acts, err := s.db.GetEnhancedActs(ctx, year)
		if err != nil {
			// Continue with other years if one fails
			continue
		}
		allActs = append(allActs, acts...)
	}
	
	// Apply filters
	filteredActs := s.applyFilters(allActs, criteria)
	
	// Sort results
	s.sortActs(filteredActs, criteria)
	
	// Apply pagination
	paginatedActs := s.applyPagination(filteredActs, criteria)
	
	// Generate facets
	facets := s.generateFacets(allActs, filteredActs)
	
	return &SearchResult{
		Acts:          paginatedActs,
		TotalCount:    len(allActs),
		FilteredCount: len(filteredActs),
		SearchTime:    time.Since(startTime),
		Facets:        facets,
	}, nil
}

// applyFilters applies all search criteria to filter acts
func (s *SearchService) applyFilters(acts []sejm.EnhancedAct, criteria *SearchCriteria) []sejm.EnhancedAct {
	var filtered []sejm.EnhancedAct
	
	for _, act := range acts {
		if s.matchesFilters(&act, criteria) {
			filtered = append(filtered, act)
		}
	}
	
	return filtered
}

// matchesFilters checks if an act matches all filter criteria
func (s *SearchService) matchesFilters(act *sejm.EnhancedAct, criteria *SearchCriteria) bool {
	// Text search - search in title, ID, and other text fields
	if criteria.Query != "" {
		query := strings.ToLower(criteria.Query)
		searchText := strings.ToLower(act.Title + " " + act.ID + " " + act.CurrentStage + " " + act.InitiatorType)
		if !strings.Contains(searchText, query) {
			return false
		}
	}
	
	// Title-specific search
	if criteria.TitleSearch != "" {
		if !strings.Contains(strings.ToLower(act.Title), strings.ToLower(criteria.TitleSearch)) {
			return false
		}
	}
	
	// Initiator search
	if criteria.InitiatorSearch != "" {
		if !strings.Contains(strings.ToLower(act.InitiatorType), strings.ToLower(criteria.InitiatorSearch)) {
			return false
		}
	}
	
	// Status filters
	if len(criteria.Statuses) > 0 {
		found := false
		for _, status := range criteria.Statuses {
			if act.Status == status {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Detailed status filters
	if len(criteria.DetailedStatuses) > 0 {
		found := false
		for _, status := range criteria.DetailedStatuses {
			if act.DetailedStatus == status {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Current stage filters
	if len(criteria.CurrentStages) > 0 {
		found := false
		for _, stage := range criteria.CurrentStages {
			if strings.Contains(act.CurrentStage, stage) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Year range filter
	if criteria.YearFrom != nil && act.Year < *criteria.YearFrom {
		return false
	}
	if criteria.YearTo != nil && act.Year > *criteria.YearTo {
		return false
	}
	
	// Position range filter
	if criteria.PositionFrom != nil && act.Position < *criteria.PositionFrom {
		return false
	}
	if criteria.PositionTo != nil && act.Position > *criteria.PositionTo {
		return false
	}
	
	// Days in stage filter
	if criteria.DaysInStageMin != nil && act.DaysInStage < *criteria.DaysInStageMin {
		return false
	}
	if criteria.DaysInStageMax != nil && act.DaysInStage > *criteria.DaysInStageMax {
		return false
	}
	
	// Stage date filters
	if !act.StageDate.IsZero() {
		if criteria.StageFrom != nil && act.StageDate.Before(*criteria.StageFrom) {
			return false
		}
		if criteria.StageTo != nil && act.StageDate.After(*criteria.StageTo) {
			return false
		}
	}
	
	// Voting filters
	if criteria.HasSejmVotes != nil {
		hasSejmVotes := len(act.SejmVotes) > 0
		if *criteria.HasSejmVotes != hasSejmVotes {
			return false
		}
	}
	
	if criteria.HasSenateVotes != nil {
		hasSenateVotes := len(act.SenateVotes) > 0
		if *criteria.HasSenateVotes != hasSenateVotes {
			return false
		}
	}
	
	// Voting result filter
	if criteria.VotingResult != "" {
		switch criteria.VotingResult {
		case "passed":
			if !s.hasPassedVotes(act) {
				return false
			}
		case "failed":
			if !s.hasFailedVotes(act) {
				return false
			}
		case "pending":
			if len(act.SejmVotes) > 0 || len(act.SenateVotes) > 0 {
				return false
			}
		}
	}
	
	// Committee filter
	if len(criteria.CommitteeCodes) > 0 {
		found := false
		for _, code := range criteria.CommitteeCodes {
			if strings.Contains(act.CommitteeCode, code) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Initiator type filter
	if len(criteria.InitiatorTypes) > 0 {
		found := false
		for _, initiator := range criteria.InitiatorTypes {
			if strings.Contains(act.InitiatorType, initiator) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Tags filter
	if len(criteria.Tags) > 0 {
		found := false
		for _, tag := range criteria.Tags {
			for _, actTag := range act.Tags {
				if strings.EqualFold(actTag, tag) {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}
	
	return true
}

// hasPassedVotes checks if act has passed votes
func (s *SearchService) hasPassedVotes(act *sejm.EnhancedAct) bool {
	for _, vote := range act.SejmVotes {
		if vote.YesVotes > vote.NoVotes {
			return true
		}
	}
	for _, vote := range act.SenateVotes {
		if vote.YesVotes > vote.NoVotes {
			return true
		}
	}
	return false
}

// hasFailedVotes checks if act has failed votes
func (s *SearchService) hasFailedVotes(act *sejm.EnhancedAct) bool {
	for _, vote := range act.SejmVotes {
		if vote.NoVotes > vote.YesVotes {
			return true
		}
	}
	for _, vote := range act.SenateVotes {
		if vote.NoVotes > vote.YesVotes {
			return true
		}
	}
	return false
}

// sortActs sorts the filtered acts based on criteria
func (s *SearchService) sortActs(acts []sejm.EnhancedAct, criteria *SearchCriteria) {
	if criteria.SortBy == "" {
		criteria.SortBy = "date" // Default sort
	}
	if criteria.SortOrder == "" {
		criteria.SortOrder = "desc" // Default order
	}
	
	sort.Slice(acts, func(i, j int) bool {
		var less bool
		
		switch criteria.SortBy {
		case "title":
			less = acts[i].Title < acts[j].Title
		case "position":
			less = acts[i].Position < acts[j].Position
		case "year":
			less = acts[i].Year < acts[j].Year
		case "stage_date":
			less = acts[i].StageDate.Before(acts[j].StageDate)
		case "days_in_stage":
			less = acts[i].DaysInStage < acts[j].DaysInStage
		default: // "date" - sort by year and position
			if acts[i].Year != acts[j].Year {
				less = acts[i].Year < acts[j].Year
			} else {
				less = acts[i].Position < acts[j].Position
			}
		}
		
		if criteria.SortOrder == "desc" {
			return !less
		}
		return less
	})
}

// applyPagination applies limit and offset to results
func (s *SearchService) applyPagination(acts []sejm.EnhancedAct, criteria *SearchCriteria) []sejm.EnhancedAct {
	if criteria.Limit <= 0 {
		criteria.Limit = 50 // Default limit
	}
	
	start := criteria.Offset
	if start < 0 {
		start = 0
	}
	if start >= len(acts) {
		return []sejm.EnhancedAct{}
	}
	
	end := start + criteria.Limit
	if end > len(acts) {
		end = len(acts)
	}
	
	return acts[start:end]
}

// generateFacets creates facet data for filtering UI
func (s *SearchService) generateFacets(allActs, filteredActs []sejm.EnhancedAct) SearchFacets {
	facets := SearchFacets{}
	
	// Count occurrences for facets
	statusCounts := make(map[string]int)
	stageCounts := make(map[string]int)
	initiatorCounts := make(map[string]int)
	committeeCounts := make(map[string]int)
	tagCounts := make(map[string]int)
	
	minYear, maxYear := 9999, 0
	minDays, maxDays := 999999, 0
	
	for _, act := range filteredActs {
		// Status counts
		if act.Status != "" {
			statusCounts[act.Status]++
		}
		if act.DetailedStatus != "" {
			statusCounts[act.DetailedStatus]++
		}
		
		// Stage counts
		if act.CurrentStage != "" {
			stageCounts[act.CurrentStage]++
		}
		
		// Initiator counts
		if act.InitiatorType != "" {
			initiatorCounts[act.InitiatorType]++
		}
		
		// Committee counts
		if act.CommitteeCode != "" {
			committeeCounts[act.CommitteeCode]++
		}
		
		// Tag counts
		for _, tag := range act.Tags {
			if tag != "" {
				tagCounts[tag]++
			}
		}
		
		// Ranges
		if act.Year < minYear {
			minYear = act.Year
		}
		if act.Year > maxYear {
			maxYear = act.Year
		}
		
		if act.DaysInStage < minDays {
			minDays = act.DaysInStage
		}
		if act.DaysInStage > maxDays {
			maxDays = act.DaysInStage
		}
	}
	
	// Convert to facet items
	facets.AvailableStatuses = s.countsToFacets(statusCounts)
	facets.AvailableStages = s.countsToFacets(stageCounts)
	facets.AvailableInitiators = s.countsToFacets(initiatorCounts)
	facets.AvailableCommittees = s.countsToFacets(committeeCounts)
	facets.AvailableTags = s.countsToFacets(tagCounts)
	
	facets.YearRange = YearRange{Min: minYear, Max: maxYear}
	facets.DaysInStageRange = Range{Min: minDays, Max: maxDays}
	
	return facets
}

// countsToFacets converts count map to sorted facet items
func (s *SearchService) countsToFacets(counts map[string]int) []FacetItem {
	var items []FacetItem
	
	for value, count := range counts {
		items = append(items, FacetItem{
			Value: value,
			Count: count,
			Label: value,
		})
	}
	
	// Sort by count descending, then by value
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count != items[j].Count {
			return items[i].Count > items[j].Count
		}
		return items[i].Value < items[j].Value
	})
	
	return items
}

// ParseSearchCriteria parses search criteria from query parameters
func ParseSearchCriteria(params map[string][]string) *SearchCriteria {
	criteria := &SearchCriteria{}
	
	// Text search
	if query := getParam(params, "q"); query != "" {
		criteria.Query = query
	}
	if title := getParam(params, "title"); title != "" {
		criteria.TitleSearch = title
	}
	if initiator := getParam(params, "initiator"); initiator != "" {
		criteria.InitiatorSearch = initiator
	}
	
	// Status filters
	criteria.Statuses = getParams(params, "status")
	criteria.DetailedStatuses = getParams(params, "detailed_status")
	criteria.CurrentStages = getParams(params, "stage")
	
	// Date filters
	if dateFrom := getParam(params, "date_from"); dateFrom != "" {
		if t, err := time.Parse("2006-01-02", dateFrom); err == nil {
			criteria.DateFrom = &t
		}
	}
	if dateTo := getParam(params, "date_to"); dateTo != "" {
		if t, err := time.Parse("2006-01-02", dateTo); err == nil {
			criteria.DateTo = &t
		}
	}
	
	// Numeric filters
	if yearFrom := getIntParam(params, "year_from"); yearFrom != nil {
		criteria.YearFrom = yearFrom
	}
	if yearTo := getIntParam(params, "year_to"); yearTo != nil {
		criteria.YearTo = yearTo
	}
	if posFrom := getIntParam(params, "pos_from"); posFrom != nil {
		criteria.PositionFrom = posFrom
	}
	if posTo := getIntParam(params, "pos_to"); posTo != nil {
		criteria.PositionTo = posTo
	}
	if daysMin := getIntParam(params, "days_min"); daysMin != nil {
		criteria.DaysInStageMin = daysMin
	}
	if daysMax := getIntParam(params, "days_max"); daysMax != nil {
		criteria.DaysInStageMax = daysMax
	}
	
	// Voting filters
	if hasSeimVotes := getBoolParam(params, "has_sejm_votes"); hasSeimVotes != nil {
		criteria.HasSejmVotes = hasSeimVotes
	}
	if hasSenateVotes := getBoolParam(params, "has_senate_votes"); hasSenateVotes != nil {
		criteria.HasSenateVotes = hasSenateVotes
	}
	if votingResult := getParam(params, "voting_result"); votingResult != "" {
		criteria.VotingResult = votingResult
	}
	
	// Other filters
	criteria.CommitteeCodes = getParams(params, "committee")
	criteria.Tags = getParams(params, "tag")
	criteria.InitiatorTypes = getParams(params, "initiator_type")
	
	// Sorting and pagination
	if sortBy := getParam(params, "sort"); sortBy != "" {
		criteria.SortBy = sortBy
	}
	if sortOrder := getParam(params, "order"); sortOrder != "" {
		criteria.SortOrder = sortOrder
	}
	if limit := getIntParam(params, "limit"); limit != nil {
		criteria.Limit = *limit
	}
	if offset := getIntParam(params, "offset"); offset != nil {
		criteria.Offset = *offset
	}
	
	return criteria
}

// Helper functions for parameter parsing
func getParam(params map[string][]string, key string) string {
	if values, exists := params[key]; exists && len(values) > 0 {
		return values[0]
	}
	return ""
}

func getParams(params map[string][]string, key string) []string {
	if values, exists := params[key]; exists {
		return values
	}
	return nil
}

func getIntParam(params map[string][]string, key string) *int {
	if value := getParam(params, key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return &i
		}
	}
	return nil
}

func getBoolParam(params map[string][]string, key string) *bool {
	if value := getParam(params, key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return &b
		}
	}
	return nil
}

// GetSearchSuggestions provides auto-complete suggestions for search
func (s *SearchService) GetSearchSuggestions(ctx context.Context, query string, field string) ([]string, error) {
	// Get recent acts to extract suggestions from
	currentYear := time.Now().Year()
	var allActs []sejm.EnhancedAct
	
	for year := currentYear - 1; year <= currentYear; year++ {
		acts, err := s.db.GetEnhancedActs(ctx, year)
		if err != nil {
			continue
		}
		allActs = append(allActs, acts...)
	}
	
	suggestions := make(map[string]bool)
	query = strings.ToLower(query)
	
	for _, act := range allActs {
		var fieldValue string
		switch field {
		case "title":
			fieldValue = act.Title
		case "initiator":
			fieldValue = act.InitiatorType
		case "stage":
			fieldValue = act.CurrentStage
		case "committee":
			fieldValue = act.CommitteeCode
		default:
			continue
		}
		
		if fieldValue != "" && strings.Contains(strings.ToLower(fieldValue), query) {
			suggestions[fieldValue] = true
		}
	}
	
	// Convert to sorted slice
	var result []string
	for suggestion := range suggestions {
		result = append(result, suggestion)
	}
	
	sort.Strings(result)
	
	// Limit to top 10 suggestions
	if len(result) > 10 {
		result = result[:10]
	}
	
	return result, nil
}