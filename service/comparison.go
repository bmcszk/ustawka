package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"ustawka/sejm"
)

// ActComparison represents a comparison between two acts
type ActComparison struct {
	LeftAct       *sejm.EnhancedAct    `json:"left_act"`
	RightAct      *sejm.EnhancedAct    `json:"right_act"`
	Differences   []FieldDifference    `json:"differences"`
	Similarities  []FieldSimilarity    `json:"similarities"`
	ComparisonID  string               `json:"comparison_id"`
	CreatedAt     time.Time            `json:"created_at"`
	Summary       ComparisonSummary    `json:"summary"`
}

// FieldDifference represents a difference between two act fields
type FieldDifference struct {
	Field         string `json:"field"`
	FieldLabel    string `json:"field_label"`
	LeftValue     any    `json:"left_value"`
	RightValue    any    `json:"right_value"`
	DifferenceType string `json:"difference_type"`
	Severity      string `json:"severity"`
	Description   string `json:"description"`
}

// FieldSimilarity represents a similarity between two act fields
type FieldSimilarity struct {
	Field       string `json:"field"`
	FieldLabel  string `json:"field_label"`
	Value       any    `json:"value"`
	Description string `json:"description"`
}

// ComparisonSummary provides overview statistics
type ComparisonSummary struct {
	TotalFields      int `json:"total_fields"`
	DifferentFields  int `json:"different_fields"`
	SimilarFields    int `json:"similar_fields"`
	CriticalDiffs    int `json:"critical_diffs"`
	MajorDiffs       int `json:"major_diffs"`
	MinorDiffs       int `json:"minor_diffs"`
}

// diffType represents the type of difference (internal)
type diffType string

const (
	diffTypeValueChanged diffType = "value_changed"
	diffTypeAdded        diffType = "added"
	diffTypeRemoved      diffType = "removed"
	diffTypeModified     diffType = "modified"
)

// diffSeverity represents the importance of a difference (internal)
type diffSeverity string

const (
	diffSeverityCritical diffSeverity = "critical"
	diffSeverityMajor    diffSeverity = "major"
	diffSeverityMinor    diffSeverity = "minor"
	diffSeverityInfo     diffSeverity = "info"
)

// ComparisonService provides act comparison functionality
type ComparisonService struct {
	db Database
}

// NewComparisonService creates a new comparison service
func NewComparisonService(db Database) *ComparisonService {
	return &ComparisonService{
		db: db,
	}
}

// CompareActs compares two acts and returns detailed differences
func (cs *ComparisonService) CompareActs(ctx context.Context, leftID, rightID string) (*ActComparison, error) {
	// Fetch both acts
	leftAct, err := cs.getActByID(ctx, leftID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch left act %s: %w", leftID, err)
	}
	
	rightAct, err := cs.getActByID(ctx, rightID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch right act %s: %w", rightID, err)
	}
	
	// Generate comparison
	comparison := &ActComparison{
		LeftAct:      leftAct,
		RightAct:     rightAct,
		ComparisonID: fmt.Sprintf("%s_vs_%s", leftID, rightID),
		CreatedAt:    time.Now(),
	}
	
	// Compare all relevant fields
	comparison.Differences = cs.findDifferences(leftAct, rightAct)
	comparison.Similarities = cs.findSimilarities(leftAct, rightAct)
	comparison.Summary = cs.generateSummary(comparison.Differences, comparison.Similarities)
	
	return comparison, nil
}

// getActByID retrieves an act by ID (handles both year/position and direct ID formats)
func (cs *ComparisonService) getActByID(ctx context.Context, id string) (*sejm.EnhancedAct, error) {
	// Parse ID format: either "DU/YEAR/POSITION" or direct lookup
	parts := strings.Split(id, "/")
	if len(parts) >= 3 && parts[0] == "DU" {
		// Format: DU/YEAR/POSITION
		year := parts[1]
		position := parts[2]
		
		// Try to get enhanced act details first
		details, err := cs.getEnhancedActDetails(ctx, year, position)
		if err == nil {
			return details, nil
		}
		
		// Fallback to basic act lookup
		return cs.getBasicActAsEnhanced(ctx, year, position)
	}
	
	// Direct ID search across all years
	return cs.searchActByDirectID(ctx, id)
}

// getEnhancedActDetails retrieves enhanced act details by year and position
func (*ComparisonService) getEnhancedActDetails(_ context.Context, year, position string) (*sejm.EnhancedAct, error) {
	// This would use the same logic as the existing act service
	// For now, return a basic implementation
	return nil, fmt.Errorf("enhanced act details not found for %s/%s", year, position)
}

// getBasicActAsEnhanced converts basic act to enhanced act format
func (*ComparisonService) getBasicActAsEnhanced(_ context.Context, year, position string) (*sejm.EnhancedAct, error) {
	// This would convert basic act data to enhanced format
	return nil, fmt.Errorf("basic act not found for %s/%s", year, position)
}

// searchActByDirectID searches for an act by direct ID across all data
func (cs *ComparisonService) searchActByDirectID(ctx context.Context, id string) (*sejm.EnhancedAct, error) {
	currentYear := time.Now().Year()
	yearRange := cs.getSearchYearRange(currentYear)
	
	for year := yearRange.start; year <= yearRange.end; year++ {
		if act := cs.searchActInYear(ctx, id, year); act != nil {
			return act, nil
		}
	}
	
	return nil, fmt.Errorf("act not found with ID: %s", id)
}

// yearRange represents a range of years to search
type yearRange struct {
	start, end int
}

// getSearchYearRange returns the range of years to search for acts
func (*ComparisonService) getSearchYearRange(currentYear int) yearRange {
	return yearRange{
		start: currentYear - 5,
		end:   currentYear + 1,
	}
}

// searchActInYear searches for an act in a specific year
func (cs *ComparisonService) searchActInYear(ctx context.Context, id string, year int) *sejm.EnhancedAct {
	acts, err := cs.db.GetEnhancedActs(ctx, year)
	if err != nil {
		return nil
	}
	
	for _, act := range acts {
		if act.ID == id {
			return &act
		}
	}
	
	return nil
}

// findDifferences identifies all differences between two acts
func (cs *ComparisonService) findDifferences(left, right *sejm.EnhancedAct) []FieldDifference {
	var differences []FieldDifference
	
	// Basic fields comparison
	differences = append(differences, cs.compareBasicFields(left, right)...)
	
	// Status and stage comparison
	differences = append(differences, cs.compareStatusFields(left, right)...)
	
	// Voting comparison
	differences = append(differences, cs.compareVotingFields(left, right)...)
	
	// Timing comparison
	differences = append(differences, cs.compareTimingFields(left, right)...)
	
	// Metadata comparison
	differences = append(differences, cs.compareMetadataFields(left, right)...)
	
	return differences
}

// compareBasicFields compares basic act information
func (*ComparisonService) compareBasicFields(left, right *sejm.EnhancedAct) []FieldDifference {
	var diffs []FieldDifference
	
	// Title comparison
	if left.Title != right.Title {
		diffs = append(diffs, FieldDifference{
			Field:         "title",
			FieldLabel:    "Tytuł",
			LeftValue:     left.Title,
			RightValue:    right.Title,
			DifferenceType: string(diffTypeValueChanged),
			Severity:      string(diffSeverityMajor),
			Description:   "Tytuły aktów prawnych się różnią",
		})
	}
	
	// Year comparison
	if left.Year != right.Year {
		diffs = append(diffs, FieldDifference{
			Field:         "year",
			FieldLabel:    "Rok",
			LeftValue:     left.Year,
			RightValue:    right.Year,
			DifferenceType: string(diffTypeValueChanged),
			Severity:      string(diffSeverityMajor),
			Description:   "Akty pochodzą z różnych lat",
		})
	}
	
	// Position comparison
	if left.Position != right.Position {
		diffs = append(diffs, FieldDifference{
			Field:         "position",
			FieldLabel:    "Pozycja",
			LeftValue:     left.Position,
			RightValue:    right.Position,
			DifferenceType: string(diffTypeValueChanged),
			Severity:      string(diffSeverityMinor),
			Description:   "Różne pozycje w dzienniku ustaw",
		})
	}
	
	// Initiator comparison
	if left.InitiatorType != right.InitiatorType {
		diffs = append(diffs, FieldDifference{
			Field:         "initiator_type",
			FieldLabel:    "Inicjator",
			LeftValue:     left.InitiatorType,
			RightValue:    right.InitiatorType,
			DifferenceType: string(diffTypeValueChanged),
			Severity:      string(diffSeverityMajor),
			Description:   "Różni inicjatorzy aktów prawnych",
		})
	}
	
	return diffs
}

// compareStatusFields compares status and stage information
func (*ComparisonService) compareStatusFields(left, right *sejm.EnhancedAct) []FieldDifference {
	var diffs []FieldDifference
	
	// Status comparison
	if left.Status != right.Status {
		diffs = append(diffs, FieldDifference{
			Field:         "status",
			FieldLabel:    "Status",
			LeftValue:     left.Status,
			RightValue:    right.Status,
			DifferenceType: string(diffTypeValueChanged),
			Severity:      string(diffSeverityCritical),
			Description:   "Różne statusy aktów prawnych",
		})
	}
	
	// Detailed status comparison
	if left.DetailedStatus != right.DetailedStatus {
		diffs = append(diffs, FieldDifference{
			Field:         "detailed_status",
			FieldLabel:    "Status szczegółowy",
			LeftValue:     left.DetailedStatus,
			RightValue:    right.DetailedStatus,
			DifferenceType: string(diffTypeValueChanged),
			Severity:      string(diffSeverityMajor),
			Description:   "Różne szczegółowe statusy",
		})
	}
	
	// Current stage comparison
	if left.CurrentStage != right.CurrentStage {
		diffs = append(diffs, FieldDifference{
			Field:         "current_stage",
			FieldLabel:    "Aktualny etap",
			LeftValue:     left.CurrentStage,
			RightValue:    right.CurrentStage,
			DifferenceType: string(diffTypeValueChanged),
			Severity:      string(diffSeverityMajor),
			Description:   "Akty znajdują się w różnych etapach procedury",
		})
	}
	
	// Days in stage comparison
	if left.DaysInStage != right.DaysInStage {
		severity := string(diffSeverityMinor)
		if abs(left.DaysInStage-right.DaysInStage) > 30 {
			severity = string(diffSeverityMajor)
		}
		
		diffs = append(diffs, FieldDifference{
			Field:         "days_in_stage",
			FieldLabel:    "Dni w etapie",
			LeftValue:     left.DaysInStage,
			RightValue:    right.DaysInStage,
			DifferenceType: string(diffTypeValueChanged),
			Severity:      severity,
			Description:   fmt.Sprintf("Różnica w czasie trwania etapu: %d dni", 
				abs(left.DaysInStage-right.DaysInStage)),
		})
	}
	
	return diffs
}

// compareVotingFields compares voting information
func (cs *ComparisonService) compareVotingFields(left, right *sejm.EnhancedAct) []FieldDifference {
	var diffs []FieldDifference
	
	// Sejm votes comparison
	leftSejmVotes := len(left.SejmVotes)
	rightSejmVotes := len(right.SejmVotes)
	
	if leftSejmVotes != rightSejmVotes {
		diffs = append(diffs, FieldDifference{
			Field:         "sejm_votes_count",
			FieldLabel:    "Liczba głosowań w Sejmie",
			LeftValue:     leftSejmVotes,
			RightValue:    rightSejmVotes,
			DifferenceType: string(diffTypeValueChanged),
			Severity:      string(diffSeverityMajor),
			Description:   "Różna liczba głosowań w Sejmie",
		})
	}
	
	// Senate votes comparison
	leftSenateVotes := len(left.SenateVotes)
	rightSenateVotes := len(right.SenateVotes)
	
	if leftSenateVotes != rightSenateVotes {
		diffs = append(diffs, FieldDifference{
			Field:         "senate_votes_count",
			FieldLabel:    "Liczba głosowań w Senacie",
			LeftValue:     leftSenateVotes,
			RightValue:    rightSenateVotes,
			DifferenceType: string(diffTypeValueChanged),
			Severity:      string(diffSeverityMajor),
			Description:   "Różna liczba głosowań w Senacie",
		})
	}
	
	// Compare voting results if both have votes
	if leftSejmVotes > 0 && rightSejmVotes > 0 {
		diffs = append(diffs, cs.compareVotingResults(left.SejmVotes, right.SejmVotes, "Sejm")...)
	}
	
	if leftSenateVotes > 0 && rightSenateVotes > 0 {
		diffs = append(diffs, cs.compareVotingResults(left.SenateVotes, right.SenateVotes, "Senat")...)
	}
	
	return diffs
}

// compareVotingResults compares specific voting results
func (*ComparisonService) compareVotingResults(
	leftVotes, rightVotes []sejm.VotingRecord, chamber string,
) []FieldDifference {
	var diffs []FieldDifference
	
	// Compare most recent votes
	if len(leftVotes) > 0 && len(rightVotes) > 0 {
		leftVote := leftVotes[len(leftVotes)-1]
		rightVote := rightVotes[len(rightVotes)-1]
		
		// Compare yes votes
		if leftVote.YesVotes != rightVote.YesVotes {
			diffs = append(diffs, FieldDifference{
				Field:         fmt.Sprintf("%s_yes_votes", strings.ToLower(chamber)),
				FieldLabel:    fmt.Sprintf("Głosy za (%s)", chamber),
				LeftValue:     leftVote.YesVotes,
				RightValue:    rightVote.YesVotes,
				DifferenceType: string(diffTypeValueChanged),
				Severity:      string(diffSeverityMajor),
				Description:   fmt.Sprintf("Różna liczba głosów za w %s", chamber),
			})
		}
		
		// Compare no votes
		if leftVote.NoVotes != rightVote.NoVotes {
			diffs = append(diffs, FieldDifference{
				Field:         fmt.Sprintf("%s_no_votes", strings.ToLower(chamber)),
				FieldLabel:    fmt.Sprintf("Głosy przeciw (%s)", chamber),
				LeftValue:     leftVote.NoVotes,
				RightValue:    rightVote.NoVotes,
				DifferenceType: string(diffTypeValueChanged),
				Severity:      string(diffSeverityMajor),
				Description:   fmt.Sprintf("Różna liczba głosów przeciw w %s", chamber),
			})
		}
	}
	
	return diffs
}

// compareTimingFields compares timing-related information
func (*ComparisonService) compareTimingFields(left, right *sejm.EnhancedAct) []FieldDifference {
	var diffs []FieldDifference
	
	// Stage date comparison
	if !left.StageDate.IsZero() && !right.StageDate.IsZero() {
		if !left.StageDate.Equal(right.StageDate) {
			daysDiff := int(left.StageDate.Sub(right.StageDate).Hours() / 24)
			severity := string(diffSeverityMinor)
			if abs(daysDiff) > 30 {
				severity = string(diffSeverityMajor)
			}
			
			diffs = append(diffs, FieldDifference{
				Field:         "stage_date",
				FieldLabel:    "Data etapu",
				LeftValue:     left.StageDate.Format("2006-01-02"),
				RightValue:    right.StageDate.Format("2006-01-02"),
				DifferenceType: string(diffTypeValueChanged),
				Severity:      severity,
				Description:   fmt.Sprintf("Różnica w dacie etapu: %d dni", daysDiff),
			})
		}
	}
	
	return diffs
}

// compareMetadataFields compares metadata and reference information
func (*ComparisonService) compareMetadataFields(left, right *sejm.EnhancedAct) []FieldDifference {
	var diffs []FieldDifference
	
	// Committee comparison
	if left.CommitteeCode != right.CommitteeCode {
		diffs = append(diffs, FieldDifference{
			Field:         "committee_code",
			FieldLabel:    "Kod komisji",
			LeftValue:     left.CommitteeCode,
			RightValue:    right.CommitteeCode,
			DifferenceType: string(diffTypeValueChanged),
			Severity:      string(diffSeverityMinor),
			Description:   "Różne komisje odpowiedzialne za akty",
		})
	}
	
	// Tags comparison
	leftTags := strings.Join(left.Tags, ", ")
	rightTags := strings.Join(right.Tags, ", ")
	
	if leftTags != rightTags {
		diffs = append(diffs, FieldDifference{
			Field:         "tags",
			FieldLabel:    "Tagi",
			LeftValue:     leftTags,
			RightValue:    rightTags,
			DifferenceType: string(diffTypeValueChanged),
			Severity:      string(diffSeverityInfo),
			Description:   "Różne tagi kategoryzacyjne",
		})
	}
	
	return diffs
}

// findSimilarities identifies similarities between two acts
func (*ComparisonService) findSimilarities(left, right *sejm.EnhancedAct) []FieldSimilarity {
	var similarities []FieldSimilarity
	
	// Same year
	if left.Year == right.Year {
		similarities = append(similarities, FieldSimilarity{
			Field:       "year",
			FieldLabel:  "Rok",
			Value:       left.Year,
			Description: "Oba akty pochodzą z tego samego roku",
		})
	}
	
	// Same status
	if left.Status == right.Status {
		similarities = append(similarities, FieldSimilarity{
			Field:       "status",
			FieldLabel:  "Status",
			Value:       left.Status,
			Description: "Oba akty mają ten sam status",
		})
	}
	
	// Same initiator
	if left.InitiatorType == right.InitiatorType {
		similarities = append(similarities, FieldSimilarity{
			Field:       "initiator_type",
			FieldLabel:  "Inicjator",
			Value:       left.InitiatorType,
			Description: "Oba akty mają tego samego inicjatora",
		})
	}
	
	// Same current stage
	if left.CurrentStage == right.CurrentStage && left.CurrentStage != "" {
		similarities = append(similarities, FieldSimilarity{
			Field:       "current_stage",
			FieldLabel:  "Aktualny etap",
			Value:       left.CurrentStage,
			Description: "Oba akty znajdują się w tym samym etapie",
		})
	}
	
	// Similar days in stage (within 7 days)
	if abs(left.DaysInStage-right.DaysInStage) <= 7 {
		similarities = append(similarities, FieldSimilarity{
			Field:       "days_in_stage",
			FieldLabel:  "Dni w etapie",
			Value:       fmt.Sprintf("~%d dni", (left.DaysInStage+right.DaysInStage)/2),
			Description: "Podobny czas trwania w aktualnym etapie",
		})
	}
	
	return similarities
}

// generateSummary creates a summary of the comparison
func (*ComparisonService) generateSummary(
	differences []FieldDifference, similarities []FieldSimilarity,
) ComparisonSummary {
	summary := ComparisonSummary{
		TotalFields:     len(differences) + len(similarities),
		DifferentFields: len(differences),
		SimilarFields:   len(similarities),
	}
	
	// Count differences by severity
	for _, diff := range differences {
		switch diff.Severity {
		case string(diffSeverityCritical):
			summary.CriticalDiffs++
		case string(diffSeverityMajor):
			summary.MajorDiffs++
		case string(diffSeverityMinor):
			summary.MinorDiffs++
		}
	}
	
	return summary
}

// GetComparisonSuggestions returns suggested acts for comparison
func (cs *ComparisonService) GetComparisonSuggestions(
	ctx context.Context, actID string,
) ([]sejm.EnhancedAct, error) {
	baseAct, err := cs.getActByID(ctx, actID)
	if err != nil {
		return nil, err
	}
	
	suggestions := cs.findSimilarActs(ctx, baseAct)
	cs.sortSuggestionsByScore(baseAct, suggestions)
	
	return cs.limitSuggestions(suggestions), nil
}

// findSimilarActs finds acts similar to the base act across related years
func (cs *ComparisonService) findSimilarActs(ctx context.Context, baseAct *sejm.EnhancedAct) []sejm.EnhancedAct {
	var suggestions []sejm.EnhancedAct
	
	for year := baseAct.Year - 1; year <= baseAct.Year+1; year++ {
		yearSuggestions := cs.findSimilarActsInYear(ctx, baseAct, year)
		suggestions = append(suggestions, yearSuggestions...)
	}
	
	return suggestions
}

// findSimilarActsInYear finds similar acts in a specific year
func (cs *ComparisonService) findSimilarActsInYear(
	ctx context.Context, baseAct *sejm.EnhancedAct, year int,
) []sejm.EnhancedAct {
	acts, err := cs.db.GetEnhancedActs(ctx, year)
	if err != nil {
		return nil
	}
	
	var suggestions []sejm.EnhancedAct
	for _, act := range acts {
		if cs.shouldIncludeAsSuggestion(baseAct, &act) {
			suggestions = append(suggestions, act)
		}
	}
	
	return suggestions
}

// shouldIncludeAsSuggestion determines if an act should be included as a suggestion
func (cs *ComparisonService) shouldIncludeAsSuggestion(baseAct, candidate *sejm.EnhancedAct) bool {
	if candidate.ID == baseAct.ID {
		return false // Skip the same act
	}
	
	score := cs.CalculateSimilarityScore(baseAct, candidate)
	return score > 0.3 // Threshold for suggestions
}

// sortSuggestionsByScore sorts suggestions by similarity score
func (cs *ComparisonService) sortSuggestionsByScore(baseAct *sejm.EnhancedAct, suggestions []sejm.EnhancedAct) {
	sort.Slice(suggestions, func(i, j int) bool {
		scoreI := cs.CalculateSimilarityScore(baseAct, &suggestions[i])
		scoreJ := cs.CalculateSimilarityScore(baseAct, &suggestions[j])
		return scoreI > scoreJ
	})
}

// limitSuggestions limits suggestions to top 10
func (*ComparisonService) limitSuggestions(suggestions []sejm.EnhancedAct) []sejm.EnhancedAct {
	if len(suggestions) > 10 {
		return suggestions[:10]
	}
	return suggestions
}

// CalculateSimilarityScore calculates a similarity score between two acts
func (cs *ComparisonService) CalculateSimilarityScore(
	base, candidate *sejm.EnhancedAct,
) float64 {
	score := 0.0
	
	// Same initiator type
	if base.InitiatorType == candidate.InitiatorType {
		score += 0.3
	}
	
	// Same status
	if base.Status == candidate.Status {
		score += 0.2
	}
	
	// Same year
	if base.Year == candidate.Year {
		score += 0.2
	}
	
	// Similar title (basic keyword matching)
	titleSimilarity := cs.CalculateTextSimilarity(base.Title, candidate.Title)
	score += titleSimilarity * 0.3
	
	return score
}

// CalculateTextSimilarity calculates basic text similarity
func (*ComparisonService) CalculateTextSimilarity(text1, text2 string) float64 {
	words1 := extractSignificantWords(text1)
	words2 := extractSignificantWords(text2)
	
	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}
	
	matches := countWordMatches(words1, words2)
	return float64(matches) / float64(maxInt(len(words1), len(words2)))
}

// extractSignificantWords extracts words longer than 3 characters
func extractSignificantWords(text string) []string {
	allWords := strings.Fields(strings.ToLower(text))
	var significantWords []string
	
	for _, word := range allWords {
		if len(word) > 3 {
			significantWords = append(significantWords, word)
		}
	}
	
	return significantWords
}

// countWordMatches counts matching words between two slices
func countWordMatches(words1, words2 []string) int {
	matches := 0
	for _, word1 := range words1 {
		for _, word2 := range words2 {
			if word1 == word2 {
				matches++
				break
			}
		}
	}
	return matches
}

// Helper functions
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}