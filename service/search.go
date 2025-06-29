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
	AvailableStatuses      []facetItem `json:"available_statuses"`
	AvailableStages        []facetItem `json:"available_stages"`
	AvailableInitiators    []facetItem `json:"available_initiators"`
	AvailableCommittees    []facetItem `json:"available_committees"`
	AvailableTags          []facetItem `json:"available_tags"`
	YearRange              YearRange   `json:"year_range"`
	DaysInStageRange       searchRange `json:"days_in_stage_range"`
}

// facetItem represents a filter option with count
type facetItem struct {
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
type searchRange struct {
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
	return s.matchesTextFilters(act, criteria) &&
		s.matchesStatusFilters(act, criteria) &&
		s.matchesRangeFilters(act, criteria) &&
		s.matchesVotingFilters(act, criteria) &&
		s.matchesMetadataFilters(act, criteria)
}

// matchesTextFilters checks text-based search criteria
func (*SearchService) matchesTextFilters(act *sejm.EnhancedAct, criteria *SearchCriteria) bool {
	return matchesGeneralQuery(act, criteria.Query) &&
		matchesTitleSearch(act, criteria.TitleSearch) &&
		matchesInitiatorSearch(act, criteria.InitiatorSearch)
}

// matchesGeneralQuery checks if act matches general text query
func matchesGeneralQuery(act *sejm.EnhancedAct, query string) bool {
	if query == "" {
		return true
	}
	
	queryLower := strings.ToLower(query)
	searchText := strings.ToLower(act.Title + " " + act.ID + " " + act.CurrentStage + " " + act.InitiatorType)
	return strings.Contains(searchText, queryLower)
}

// matchesTitleSearch checks if act title matches search criteria
func matchesTitleSearch(act *sejm.EnhancedAct, titleSearch string) bool {
	if titleSearch == "" {
		return true
	}
	
	return strings.Contains(strings.ToLower(act.Title), strings.ToLower(titleSearch))
}

// matchesInitiatorSearch checks if act initiator matches search criteria
func matchesInitiatorSearch(act *sejm.EnhancedAct, initiatorSearch string) bool {
	if initiatorSearch == "" {
		return true
	}
	
	return strings.Contains(strings.ToLower(act.InitiatorType), strings.ToLower(initiatorSearch))
}

// matchesStatusFilters checks status-related criteria
func (*SearchService) matchesStatusFilters(act *sejm.EnhancedAct, criteria *SearchCriteria) bool {
	return matchesBasicStatusFilter(act, criteria.Statuses) &&
		matchesDetailedStatusFilter(act, criteria.DetailedStatuses) &&
		matchesCurrentStageFilter(act, criteria.CurrentStages)
}

// matchesBasicStatusFilter checks if act matches basic status criteria
func matchesBasicStatusFilter(act *sejm.EnhancedAct, statuses []string) bool {
	if len(statuses) == 0 {
		return true
	}
	
	for _, status := range statuses {
		if act.Status == status {
			return true
		}
	}
	return false
}

// matchesDetailedStatusFilter checks if act matches detailed status criteria
func matchesDetailedStatusFilter(act *sejm.EnhancedAct, detailedStatuses []string) bool {
	if len(detailedStatuses) == 0 {
		return true
	}
	
	for _, status := range detailedStatuses {
		if act.DetailedStatus == status {
			return true
		}
	}
	return false
}

// matchesCurrentStageFilter checks if act matches current stage criteria
func matchesCurrentStageFilter(act *sejm.EnhancedAct, currentStages []string) bool {
	if len(currentStages) == 0 {
		return true
	}
	
	for _, stage := range currentStages {
		if strings.Contains(act.CurrentStage, stage) {
			return true
		}
	}
	return false
}

// matchesRangeFilters checks numeric and date range criteria
func (*SearchService) matchesRangeFilters(act *sejm.EnhancedAct, criteria *SearchCriteria) bool {
	return matchesYearRange(act, criteria.YearFrom, criteria.YearTo) &&
		matchesPositionRange(act, criteria.PositionFrom, criteria.PositionTo) &&
		matchesDaysInStageRange(act, criteria.DaysInStageMin, criteria.DaysInStageMax) &&
		matchesStageDateRange(act, criteria.StageFrom, criteria.StageTo)
}

// matchesYearRange checks if act year is within specified range
func matchesYearRange(act *sejm.EnhancedAct, yearFrom, yearTo *int) bool {
	if yearFrom != nil && act.Year < *yearFrom {
		return false
	}
	if yearTo != nil && act.Year > *yearTo {
		return false
	}
	return true
}

// matchesPositionRange checks if act position is within specified range
func matchesPositionRange(act *sejm.EnhancedAct, positionFrom, positionTo *int) bool {
	if positionFrom != nil && act.Position < *positionFrom {
		return false
	}
	if positionTo != nil && act.Position > *positionTo {
		return false
	}
	return true
}

// matchesDaysInStageRange checks if days in stage is within specified range
func matchesDaysInStageRange(act *sejm.EnhancedAct, daysMin, daysMax *int) bool {
	if daysMin != nil && act.DaysInStage < *daysMin {
		return false
	}
	if daysMax != nil && act.DaysInStage > *daysMax {
		return false
	}
	return true
}

// matchesStageDateRange checks if stage date is within specified range
func matchesStageDateRange(act *sejm.EnhancedAct, stageFrom, stageTo *time.Time) bool {
	if act.StageDate.IsZero() {
		return true
	}
	if stageFrom != nil && act.StageDate.Before(*stageFrom) {
		return false
	}
	if stageTo != nil && act.StageDate.After(*stageTo) {
		return false
	}
	return true
}

// matchesVotingFilters checks voting-related criteria
func (s *SearchService) matchesVotingFilters(act *sejm.EnhancedAct, criteria *SearchCriteria) bool {
	return s.matchesVotingPresence(act, criteria) && s.matchesVotingResult(act, criteria.VotingResult)
}

// matchesVotingPresence checks if act matches voting presence criteria
func (*SearchService) matchesVotingPresence(act *sejm.EnhancedAct, criteria *SearchCriteria) bool {
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
	
	return true
}

// matchesVotingResult checks if act matches voting result criteria
func (s *SearchService) matchesVotingResult(act *sejm.EnhancedAct, votingResult string) bool {
	if votingResult == "" {
		return true
	}
	
	switch votingResult {
	case "passed":
		return s.hasPassedVotes(act)
	case "failed":
		return s.hasFailedVotes(act)
	case "pending":
		return len(act.SejmVotes) == 0 && len(act.SenateVotes) == 0
	default:
		return true
	}
}

// matchesMetadataFilters checks metadata-related criteria
func (*SearchService) matchesMetadataFilters(act *sejm.EnhancedAct, criteria *SearchCriteria) bool {
	return matchesCommitteeFilter(act, criteria.CommitteeCodes) &&
		matchesInitiatorTypeFilter(act, criteria.InitiatorTypes) &&
		matchesTagsFilter(act, criteria.Tags)
}

// matchesCommitteeFilter checks if act matches committee code criteria
func matchesCommitteeFilter(act *sejm.EnhancedAct, committeeCodes []string) bool {
	if len(committeeCodes) == 0 {
		return true
	}
	
	for _, code := range committeeCodes {
		if strings.Contains(act.CommitteeCode, code) {
			return true
		}
	}
	return false
}

// matchesInitiatorTypeFilter checks if act matches initiator type criteria
func matchesInitiatorTypeFilter(act *sejm.EnhancedAct, initiatorTypes []string) bool {
	if len(initiatorTypes) == 0 {
		return true
	}
	
	for _, initiator := range initiatorTypes {
		if strings.Contains(act.InitiatorType, initiator) {
			return true
		}
	}
	return false
}

// matchesTagsFilter checks if act matches tag criteria
func matchesTagsFilter(act *sejm.EnhancedAct, tags []string) bool {
	if len(tags) == 0 {
		return true
	}
	
	for _, tag := range tags {
		for _, actTag := range act.Tags {
			if strings.EqualFold(actTag, tag) {
				return true
			}
		}
	}
	return false
}

// hasPassedVotes checks if act has passed votes
func (*SearchService) hasPassedVotes(act *sejm.EnhancedAct) bool {
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
func (*SearchService) hasFailedVotes(act *sejm.EnhancedAct) bool {
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
	s.setSortDefaults(criteria)
	sort.Slice(acts, func(i, j int) bool {
		return s.compareActs(&acts[i], &acts[j], criteria)
	})
}

// setSortDefaults sets default sort criteria if not specified
func (*SearchService) setSortDefaults(criteria *SearchCriteria) {
	if criteria.SortBy == "" {
		criteria.SortBy = "date"
	}
	if criteria.SortOrder == "" {
		criteria.SortOrder = "desc"
	}
}

// compareActs compares two acts based on sort criteria
func (*SearchService) compareActs(left, right *sejm.EnhancedAct, criteria *SearchCriteria) bool {
	less := compareActsByField(left, right, criteria.SortBy)
	if criteria.SortOrder == "desc" {
		return !less
	}
	return less
}

// compareActsByField compares acts by specific field
func compareActsByField(left, right *sejm.EnhancedAct, sortBy string) bool {
	switch sortBy {
	case "title":
		return left.Title < right.Title
	case "position":
		return left.Position < right.Position
	case "year":
		return left.Year < right.Year
	case "stage_date":
		return left.StageDate.Before(right.StageDate)
	case "days_in_stage":
		return left.DaysInStage < right.DaysInStage
	default: // "date" - sort by year and position
		if left.Year != right.Year {
			return left.Year < right.Year
		}
		return left.Position < right.Position
	}
}

// applyPagination applies limit and offset to results
func (*SearchService) applyPagination(acts []sejm.EnhancedAct, criteria *SearchCriteria) []sejm.EnhancedAct {
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
func (s *SearchService) generateFacets(_, filteredActs []sejm.EnhancedAct) SearchFacets {
	counts := s.initializeFacetCounts()
	ranges := s.initializeRanges()
	
	s.processFacetData(filteredActs, counts, ranges)
	
	return s.buildFacets(counts, ranges)
}

// facetCounts holds all counting maps
type facetCounts struct {
	status    map[string]int
	stage     map[string]int
	initiator map[string]int
	committee map[string]int
	tag       map[string]int
}

// facetRanges holds min/max ranges
type facetRanges struct {
	minYear, maxYear int
	minDays, maxDays int
}

// initializeFacetCounts creates empty counting maps
func (*SearchService) initializeFacetCounts() *facetCounts {
	return &facetCounts{
		status:    make(map[string]int),
		stage:     make(map[string]int),
		initiator: make(map[string]int),
		committee: make(map[string]int),
		tag:       make(map[string]int),
	}
}

// initializeRanges creates initial range values
func (*SearchService) initializeRanges() *facetRanges {
	return &facetRanges{
		minYear: 9999,
		maxYear: 0,
		minDays: 999999,
		maxDays: 0,
	}
}

// processFacetData processes acts to populate counts and ranges
func (*SearchService) processFacetData(acts []sejm.EnhancedAct, counts *facetCounts, ranges *facetRanges) {
	for _, act := range acts {
		updateStatusCounts(counts, &act)
		updateMetadataCounts(counts, &act)
		updateRanges(ranges, &act)
	}
}

// updateStatusCounts updates status and stage counts
func updateStatusCounts(counts *facetCounts, act *sejm.EnhancedAct) {
	if act.Status != "" {
		counts.status[act.Status]++
	}
	if act.DetailedStatus != "" {
		counts.status[act.DetailedStatus]++
	}
	if act.CurrentStage != "" {
		counts.stage[act.CurrentStage]++
	}
}

// updateMetadataCounts updates initiator, committee, and tag counts
func updateMetadataCounts(counts *facetCounts, act *sejm.EnhancedAct) {
	if act.InitiatorType != "" {
		counts.initiator[act.InitiatorType]++
	}
	if act.CommitteeCode != "" {
		counts.committee[act.CommitteeCode]++
	}
	for _, tag := range act.Tags {
		if tag != "" {
			counts.tag[tag]++
		}
	}
}

// updateRanges updates min/max ranges
func updateRanges(ranges *facetRanges, act *sejm.EnhancedAct) {
	if act.Year < ranges.minYear {
		ranges.minYear = act.Year
	}
	if act.Year > ranges.maxYear {
		ranges.maxYear = act.Year
	}
	if act.DaysInStage < ranges.minDays {
		ranges.minDays = act.DaysInStage
	}
	if act.DaysInStage > ranges.maxDays {
		ranges.maxDays = act.DaysInStage
	}
}

// buildFacets constructs final SearchFacets from counts and ranges
func (s *SearchService) buildFacets(counts *facetCounts, ranges *facetRanges) SearchFacets {
	return SearchFacets{
		AvailableStatuses:   s.countsToFacets(counts.status),
		AvailableStages:     s.countsToFacets(counts.stage),
		AvailableInitiators: s.countsToFacets(counts.initiator),
		AvailableCommittees: s.countsToFacets(counts.committee),
		AvailableTags:       s.countsToFacets(counts.tag),
		YearRange:           YearRange{Min: ranges.minYear, Max: ranges.maxYear},
		DaysInStageRange:    searchRange{Min: ranges.minDays, Max: ranges.maxDays},
	}
}

// countsToFacets converts count map to sorted facet items
func (*SearchService) countsToFacets(counts map[string]int) []facetItem {
	var items []facetItem
	
	for value, count := range counts {
		items = append(items, facetItem{
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
	
	parseTextFilters(params, criteria)
	parseStatusFilters(params, criteria)
	parseDateFilters(params, criteria)
	parseNumericFilters(params, criteria)
	parseVotingFilters(params, criteria)
	parseMetadataFilters(params, criteria)
	parseSortingAndPagination(params, criteria)
	
	return criteria
}

// parseTextFilters parses text-based search parameters
func parseTextFilters(params map[string][]string, criteria *SearchCriteria) {
	if query := getParam(params, "q"); query != "" {
		criteria.Query = query
	}
	if title := getParam(params, "title"); title != "" {
		criteria.TitleSearch = title
	}
	if initiator := getParam(params, "initiator"); initiator != "" {
		criteria.InitiatorSearch = initiator
	}
}

// parseStatusFilters parses status-related parameters
func parseStatusFilters(params map[string][]string, criteria *SearchCriteria) {
	criteria.Statuses = getParams(params, "status")
	criteria.DetailedStatuses = getParams(params, "detailed_status")
	criteria.CurrentStages = getParams(params, "stage")
}

// parseDateFilters parses date-related parameters
func parseDateFilters(params map[string][]string, criteria *SearchCriteria) {
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
}

// parseNumericFilters parses numeric range parameters
func parseNumericFilters(params map[string][]string, criteria *SearchCriteria) {
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
}

// parseVotingFilters parses voting-related parameters
func parseVotingFilters(params map[string][]string, criteria *SearchCriteria) {
	if hasSeimVotes := getBoolParam(params, "has_sejm_votes"); hasSeimVotes != nil {
		criteria.HasSejmVotes = hasSeimVotes
	}
	if hasSenateVotes := getBoolParam(params, "has_senate_votes"); hasSenateVotes != nil {
		criteria.HasSenateVotes = hasSenateVotes
	}
	if votingResult := getParam(params, "voting_result"); votingResult != "" {
		criteria.VotingResult = votingResult
	}
}

// parseMetadataFilters parses metadata-related parameters
func parseMetadataFilters(params map[string][]string, criteria *SearchCriteria) {
	criteria.CommitteeCodes = getParams(params, "committee")
	criteria.Tags = getParams(params, "tag")
	criteria.InitiatorTypes = getParams(params, "initiator_type")
}

// parseSortingAndPagination parses sorting and pagination parameters
func parseSortingAndPagination(params map[string][]string, criteria *SearchCriteria) {
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
	allActs := s.getRecentActs(ctx)
	suggestions := s.extractSuggestions(allActs, query, field)
	return s.formatSuggestions(suggestions), nil
}

// getRecentActs retrieves acts from recent years
func (s *SearchService) getRecentActs(ctx context.Context) []sejm.EnhancedAct {
	currentYear := time.Now().Year()
	var allActs []sejm.EnhancedAct
	
	for year := currentYear - 1; year <= currentYear; year++ {
		acts, err := s.db.GetEnhancedActs(ctx, year)
		if err != nil {
			continue
		}
		allActs = append(allActs, acts...)
	}
	
	return allActs
}

// extractSuggestions extracts matching field values from acts
func (*SearchService) extractSuggestions(acts []sejm.EnhancedAct, query, field string) map[string]bool {
	suggestions := make(map[string]bool)
	queryLower := strings.ToLower(query)
	
	for _, act := range acts {
		fieldValue := getFieldValue(&act, field)
		if shouldIncludeSuggestion(fieldValue, queryLower) {
			suggestions[fieldValue] = true
		}
	}
	
	return suggestions
}

// getFieldValue extracts the specified field value from an act
func getFieldValue(act *sejm.EnhancedAct, field string) string {
	switch field {
	case "title":
		return act.Title
	case "initiator":
		return act.InitiatorType
	case "stage":
		return act.CurrentStage
	case "committee":
		return act.CommitteeCode
	default:
		return ""
	}
}

// shouldIncludeSuggestion determines if a field value should be included as suggestion
func shouldIncludeSuggestion(fieldValue, queryLower string) bool {
	return fieldValue != "" && strings.Contains(strings.ToLower(fieldValue), queryLower)
}

// formatSuggestions converts suggestion map to sorted, limited slice
func (*SearchService) formatSuggestions(suggestions map[string]bool) []string {
	var result []string
	for suggestion := range suggestions {
		result = append(result, suggestion)
	}
	
	sort.Strings(result)
	
	if len(result) > 10 {
		result = result[:10]
	}
	
	return result
}