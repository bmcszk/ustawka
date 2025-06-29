package service_test

import (
	"context"
	"testing"
	"time"
	"ustawka/sejm"
	"ustawka/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewComparisonService(t *testing.T) {
	db := &MockDB{}
	comparisonService := service.NewComparisonService(db)
	
	assert.NotNil(t, comparisonService)
}

func TestCompareActs_BasicDifferences(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping comparison test in short mode")
	}
	
	db := &MockDB{}
	comparisonService := service.NewComparisonService(db)
	
	// Mock acts with differences
	leftAct := sejm.EnhancedAct{
		ID:            "DU/2024/1",
		Title:         "Healthcare Reform Act",
		Year:          2024,
		Position:      1,
		Status:        "obowiązujący",
		DetailedStatus: "in_force",
		CurrentStage:  "Opublikowano",
		InitiatorType: "Government",
		DaysInStage:   30,
		StageDate:     time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
	}
	
	rightAct := sejm.EnhancedAct{
		ID:            "DU/2024/2",
		Title:         "Education Reform Act",
		Year:          2024,
		Position:      2,
		Status:        "pending",
		DetailedStatus: "committee_work",
		CurrentStage:  "Komisja Edukacji",
		InitiatorType: "Parliament",
		DaysInStage:   15,
		StageDate:     time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	
	// Mock database calls
	db.On("GetEnhancedActs", mock.Anything, 2024).Return([]sejm.EnhancedAct{leftAct, rightAct}, nil)
	db.On("GetEnhancedActs", mock.Anything, mock.AnythingOfType("int")).Return([]sejm.EnhancedAct{}, nil).Maybe()
	
	comparison, err := comparisonService.CompareActs(context.Background(), "DU/2024/1", "DU/2024/2")
	
	assert.NoError(t, err)
	assert.NotNil(t, comparison)
	assert.Equal(t, "DU/2024/1", comparison.LeftAct.ID)
	assert.Equal(t, "DU/2024/2", comparison.RightAct.ID)
	assert.True(t, len(comparison.Differences) > 0)
	
	// Check for expected differences
	foundTitleDiff := false
	foundStatusDiff := false
	
	for _, diff := range comparison.Differences {
		switch diff.Field {
		case "title":
			foundTitleDiff = true
			assert.Equal(t, "Healthcare Reform Act", diff.LeftValue)
			assert.Equal(t, "Education Reform Act", diff.RightValue)
		case "status":
			foundStatusDiff = true
			assert.Equal(t, "obowiązujący", diff.LeftValue)
			assert.Equal(t, "pending", diff.RightValue)
		}
	}
	
	assert.True(t, foundTitleDiff, "Should find title difference")
	assert.True(t, foundStatusDiff, "Should find status difference")
	
	// Check summary
	assert.True(t, comparison.Summary.DifferentFields > 0)
	assert.True(t, comparison.Summary.TotalFields > 0)
}

func TestCompareActs_SimilarActs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping comparison test in short mode")
	}
	
	db := &MockDB{}
	comparisonService := service.NewComparisonService(db)
	
	// Mock similar acts
	leftAct := sejm.EnhancedAct{
		ID:            "DU/2024/1",
		Title:         "Healthcare Reform Act",
		Year:          2024,
		Position:      1,
		Status:        "obowiązujący",
		DetailedStatus: "in_force",
		CurrentStage:  "Opublikowano",
		InitiatorType: "Government",
		DaysInStage:   30,
	}
	
	rightAct := sejm.EnhancedAct{
		ID:            "DU/2024/2",
		Title:         "Healthcare Amendment Act",
		Year:          2024,
		Position:      2,
		Status:        "obowiązujący",
		DetailedStatus: "in_force",
		CurrentStage:  "Opublikowano",
		InitiatorType: "Government",
		DaysInStage:   32, // Similar days in stage
	}
	
	db.On("GetEnhancedActs", mock.Anything, 2024).Return([]sejm.EnhancedAct{leftAct, rightAct}, nil)
	db.On("GetEnhancedActs", mock.Anything, mock.AnythingOfType("int")).Return([]sejm.EnhancedAct{}, nil).Maybe()
	
	comparison, err := comparisonService.CompareActs(context.Background(), "DU/2024/1", "DU/2024/2")
	
	assert.NoError(t, err)
	assert.NotNil(t, comparison)
	assert.True(t, len(comparison.Similarities) > 0)
	
	// Check for expected similarities
	foundYearSimilarity := false
	foundStatusSimilarity := false
	foundInitiatorSimilarity := false
	
	for _, sim := range comparison.Similarities {
		switch sim.Field {
		case "year":
			foundYearSimilarity = true
			assert.Equal(t, 2024, sim.Value)
		case "status":
			foundStatusSimilarity = true
			assert.Equal(t, "obowiązujący", sim.Value)
		case "initiator_type":
			foundInitiatorSimilarity = true
			assert.Equal(t, "Government", sim.Value)
		}
	}
	
	assert.True(t, foundYearSimilarity, "Should find year similarity")
	assert.True(t, foundStatusSimilarity, "Should find status similarity")
	assert.True(t, foundInitiatorSimilarity, "Should find initiator similarity")
}

func TestCompareActs_VotingDifferences(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping comparison test in short mode")
	}
	
	db := &MockDB{}
	comparisonService := service.NewComparisonService(db)
	
	leftAct := sejm.EnhancedAct{
		ID:       "DU/2024/1",
		Title:    "Act With Votes",
		Year:     2024,
		Position: 1,
		SejmVotes: []sejm.VotingRecord{
			{YesVotes: 300, NoVotes: 100, AbstainVotes: 50},
		},
	}
	
	rightAct := sejm.EnhancedAct{
		ID:       "DU/2024/2",
		Title:    "Act Without Votes",
		Year:     2024,
		Position: 2,
		SejmVotes: []sejm.VotingRecord{},
	}
	
	db.On("GetEnhancedActs", mock.Anything, 2024).Return([]sejm.EnhancedAct{leftAct, rightAct}, nil)
	db.On("GetEnhancedActs", mock.Anything, mock.AnythingOfType("int")).Return([]sejm.EnhancedAct{}, nil).Maybe()
	
	comparison, err := comparisonService.CompareActs(context.Background(), "DU/2024/1", "DU/2024/2")
	
	assert.NoError(t, err)
	assert.NotNil(t, comparison)
	
	// Check for voting differences
	foundVotingDiff := false
	for _, diff := range comparison.Differences {
		if diff.Field == "sejm_votes_count" {
			foundVotingDiff = true
			assert.Equal(t, 1, diff.LeftValue)
			assert.Equal(t, 0, diff.RightValue)
			break
		}
	}
	
	assert.True(t, foundVotingDiff, "Should find voting count difference")
}

func TestCompareActs_DiffSeverity(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping comparison test in short mode")
	}
	
	db := &MockDB{}
	comparisonService := service.NewComparisonService(db)
	
	leftAct := sejm.EnhancedAct{
		ID:       "DU/2024/1",
		Title:    "Act One",
		Year:     2024,
		Position: 1,
		Status:   "obowiązujący",
	}
	
	rightAct := sejm.EnhancedAct{
		ID:       "DU/2024/2",
		Title:    "Act Two",
		Year:     2024,
		Position: 2,
		Status:   "pending",
	}
	
	db.On("GetEnhancedActs", mock.Anything, 2024).Return([]sejm.EnhancedAct{leftAct, rightAct}, nil)
	db.On("GetEnhancedActs", mock.Anything, mock.AnythingOfType("int")).Return([]sejm.EnhancedAct{}, nil).Maybe()
	
	comparison, err := comparisonService.CompareActs(context.Background(), "DU/2024/1", "DU/2024/2")
	
	assert.NoError(t, err)
	assert.NotNil(t, comparison)
	
	// Check that status difference is marked as critical
	foundCriticalStatusDiff := false
	for _, diff := range comparison.Differences {
		if diff.Field == "status" {
			foundCriticalStatusDiff = true
			assert.Equal(t, "critical", diff.Severity)
			break
		}
	}
	
	assert.True(t, foundCriticalStatusDiff, "Status difference should be marked as critical")
	
	// Check summary counts
	assert.True(t, comparison.Summary.CriticalDiffs > 0, "Should have critical differences")
}

func TestGetComparisonSuggestions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping suggestions test in short mode")
	}
	
	db := &MockDB{}
	comparisonService := service.NewComparisonService(db)
	
	baseAct := sejm.EnhancedAct{
		ID:            "DU/2024/1",
		Title:         "Healthcare Reform Act",
		Year:          2024,
		Position:      1,
		Status:        "obowiązujący",
		InitiatorType: "Government",
	}
	
	similarAct := sejm.EnhancedAct{
		ID:            "DU/2024/2",
		Title:         "Healthcare Amendment Act",
		Year:          2024,
		Position:      2,
		Status:        "obowiązujący",
		InitiatorType: "Government",
	}
	
	differentAct := sejm.EnhancedAct{
		ID:            "DU/2024/3",
		Title:         "Transportation Act",
		Year:          2024,
		Position:      3,
		Status:        "pending",
		InitiatorType: "Parliament",
	}
	
	allActs := []sejm.EnhancedAct{baseAct, similarAct, differentAct}
	
	// Mock database calls for base act and surrounding years
	db.On("GetEnhancedActs", mock.Anything, 2024).Return(allActs, nil)
	db.On("GetEnhancedActs", mock.Anything, 2023).Return([]sejm.EnhancedAct{}, nil)
	db.On("GetEnhancedActs", mock.Anything, 2025).Return([]sejm.EnhancedAct{}, nil)
	
	suggestions, err := comparisonService.GetComparisonSuggestions(context.Background(), "DU/2024/1")
	
	assert.NoError(t, err)
	assert.NotNil(t, suggestions)
	assert.True(t, len(suggestions) > 0)
	
	// The similar act should be suggested
	foundSimilarAct := false
	for _, suggestion := range suggestions {
		if suggestion.ID == "DU/2024/2" {
			foundSimilarAct = true
			break
		}
	}
	
	assert.True(t, foundSimilarAct, "Should suggest the similar act")
}

func TestCalculateTextSimilarity(t *testing.T) {
	db := &MockDB{}
	comparisonService := service.NewComparisonService(db)
	
	// Test exact match
	similarity := comparisonService.CalculateTextSimilarity("Healthcare Reform Act", "Healthcare Reform Act")
	assert.True(t, similarity > 0.5, "Exact match should have high similarity")
	
	// Test partial match
	similarity = comparisonService.CalculateTextSimilarity("Healthcare Reform Act", "Healthcare Amendment Act")
	assert.True(t, similarity > 0.0, "Partial match should have some similarity")
	assert.True(t, similarity < 1.0, "Partial match should not be perfect")
	
	// Test no match
	similarity = comparisonService.CalculateTextSimilarity("Healthcare Act", "Transportation Bill")
	assert.True(t, similarity < 0.5)
	
	// Test empty strings
	similarity = comparisonService.CalculateTextSimilarity("", "Healthcare Act")
	assert.Equal(t, 0.0, similarity)
}

func TestCalculateSimilarityScore(t *testing.T) {
	db := &MockDB{}
	comparisonService := service.NewComparisonService(db)
	
	baseAct := &sejm.EnhancedAct{
		Title:         "Healthcare Reform Act",
		Year:          2024,
		Status:        "obowiązujący",
		InitiatorType: "Government",
	}
	
	// Very similar act
	similarAct := &sejm.EnhancedAct{
		Title:         "Healthcare Amendment Act",
		Year:          2024,
		Status:        "obowiązujący",
		InitiatorType: "Government",
	}
	
	// Different act
	differentAct := &sejm.EnhancedAct{
		Title:         "Transportation Infrastructure Bill",
		Year:          2023,
		Status:        "pending",
		InitiatorType: "Parliament",
	}
	
	similarScore := comparisonService.CalculateSimilarityScore(baseAct, similarAct)
	differentScore := comparisonService.CalculateSimilarityScore(baseAct, differentAct)
	
	assert.True(t, similarScore > differentScore, "Similar act should have higher score")
	assert.True(t, similarScore > 0.5, "Similar act should have high score")
	assert.True(t, differentScore < 0.5, "Different act should have low score")
}

func TestComparisonSummary(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping summary test in short mode")
	}
	
	db := &MockDB{}
	comparisonService := service.NewComparisonService(db)
	
	leftAct := sejm.EnhancedAct{
		ID:            "DU/2024/1",
		Title:         "Act One",
		Year:          2024,
		Position:      1,
		Status:        "obowiązujący",
		DetailedStatus: "in_force",
		InitiatorType: "Government",
	}
	
	rightAct := sejm.EnhancedAct{
		ID:            "DU/2024/2",
		Title:         "Act Two",
		Year:          2024,
		Position:      2,
		Status:        "pending",
		DetailedStatus: "committee_work",
		InitiatorType: "Parliament",
	}
	
	db.On("GetEnhancedActs", mock.Anything, 2024).Return([]sejm.EnhancedAct{leftAct, rightAct}, nil)
	db.On("GetEnhancedActs", mock.Anything, mock.AnythingOfType("int")).Return([]sejm.EnhancedAct{}, nil).Maybe()
	
	comparison, err := comparisonService.CompareActs(context.Background(), "DU/2024/1", "DU/2024/2")
	
	assert.NoError(t, err)
	assert.NotNil(t, comparison)
	
	summary := comparison.Summary
	assert.True(t, summary.TotalFields > 0)
	assert.True(t, summary.DifferentFields > 0)
	assert.True(t, summary.SimilarFields >= 0)
	assert.Equal(t, summary.TotalFields, summary.DifferentFields+summary.SimilarFields)
	
	// Should have at least one critical difference (status)
	assert.True(t, summary.CriticalDiffs > 0)
}

func TestCompareActs_ActNotFound(t *testing.T) {
	db := &MockDB{}
	comparisonService := service.NewComparisonService(db)
	
	// Mock empty database
	db.On("GetEnhancedActs", mock.Anything, mock.AnythingOfType("int")).Return([]sejm.EnhancedAct{}, nil)
	
	comparison, err := comparisonService.CompareActs(context.Background(), "DU/2024/999", "DU/2024/998")
	
	assert.Error(t, err)
	assert.Nil(t, comparison)
	assert.Contains(t, err.Error(), "failed to fetch")
}

func BenchmarkCompareActs(b *testing.B) {
	db := &MockDB{}
	comparisonService := service.NewComparisonService(db)
	
	leftAct := sejm.EnhancedAct{
		ID:            "DU/2024/1",
		Title:         "Benchmark Act One",
		Year:          2024,
		Position:      1,
		Status:        "obowiązujący",
		DetailedStatus: "in_force",
		SejmVotes:     []sejm.VotingRecord{{YesVotes: 300, NoVotes: 100}},
	}
	
	rightAct := sejm.EnhancedAct{
		ID:            "DU/2024/2",
		Title:         "Benchmark Act Two",
		Year:          2024,
		Position:      2,
		Status:        "pending",
		DetailedStatus: "committee_work",
		SejmVotes:     []sejm.VotingRecord{{YesVotes: 250, NoVotes: 150}},
	}
	
	db.On("GetEnhancedActs", mock.Anything, mock.AnythingOfType("int")).
		Return([]sejm.EnhancedAct{leftAct, rightAct}, nil)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := comparisonService.CompareActs(context.Background(), "DU/2024/1", "DU/2024/2")
		if err != nil {
			b.Error(err)
		}
	}
}