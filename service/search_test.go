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

func TestNewSearchService(t *testing.T) {
	db := &MockDB{}
	searchService := service.NewSearchService(db)
	
	assert.NotNil(t, searchService)
}

func TestSearchActs_BasicQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping search test in short mode")
	}
	
	db := &MockDB{}
	searchService := service.NewSearchService(db)
	
	// Mock data
	mockActs := []sejm.EnhancedAct{
		{
			ID:             "DU/2024/1",
			Title:          "Test Act About Healthcare",
			Year:           2024,
			Position:       1,
			Status:         "obowiązujący",
			DetailedStatus: "in_force",
			CurrentStage:   "Opublikowano",
			InitiatorType:  "Government",
			DaysInStage:    30,
			StageDate:      time.Now().Add(-30 * 24 * time.Hour),
		},
		{
			ID:             "DU/2024/2",
			Title:          "Education Reform Act",
			Year:           2024,
			Position:       2,
			Status:         "pending",
			DetailedStatus: "committee_work",
			CurrentStage:   "Komisja Edukacji",
			InitiatorType:  "Parliament",
			DaysInStage:    15,
			StageDate:      time.Now().Add(-15 * 24 * time.Hour),
		},
	}
	
	db.On("GetEnhancedActs", mock.Anything, mock.AnythingOfType("int")).Return(mockActs, nil)
	
	criteria := &service.SearchCriteria{
		Query: "healthcare",
	}
	
	result, err := searchService.SearchActs(context.Background(), criteria)
	
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.FilteredCount)
	assert.Equal(t, 2, result.TotalCount)
	assert.Len(t, result.Acts, 1)
	assert.Equal(t, "Test Act About Healthcare", result.Acts[0].Title)
	assert.True(t, result.SearchTime > 0)
}

func TestSearchActs_StatusFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping search test in short mode")
	}
	
	db := &MockDB{}
	searchService := service.NewSearchService(db)
	
	mockActs := []sejm.EnhancedAct{
		{
			ID:             "DU/2024/1",
			Title:          "Act One",
			Year:           2024,
			Position:       1,
			Status:         "obowiązujący",
			DetailedStatus: "in_force",
		},
		{
			ID:             "DU/2024/2",
			Title:          "Act Two",
			Year:           2024,
			Position:       2,
			Status:         "pending",
			DetailedStatus: "committee_work",
		},
	}
	
	db.On("GetEnhancedActs", mock.Anything, mock.AnythingOfType("int")).Return(mockActs, nil)
	
	criteria := &service.SearchCriteria{
		Statuses: []string{"obowiązujący"},
	}
	
	result, err := searchService.SearchActs(context.Background(), criteria)
	
	assert.NoError(t, err)
	assert.Equal(t, 1, result.FilteredCount)
	assert.Len(t, result.Acts, 1)
	assert.Equal(t, "Act One", result.Acts[0].Title)
}

func TestSearchActs_DateRangeFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping search test in short mode")
	}
	
	db := &MockDB{}
	searchService := service.NewSearchService(db)
	
	yearFrom := 2023
	yearTo := 2024
	
	mockActs := []sejm.EnhancedAct{
		{
			ID:       "DU/2023/1",
			Title:    "Old Act",
			Year:     2023,
			Position: 1,
		},
		{
			ID:       "DU/2024/1",
			Title:    "New Act",
			Year:     2024,
			Position: 1,
		},
		{
			ID:       "DU/2025/1",
			Title:    "Future Act",
			Year:     2025,
			Position: 1,
		},
	}
	
	db.On("GetEnhancedActs", mock.Anything, 2023).Return([]sejm.EnhancedAct{mockActs[0]}, nil)
	db.On("GetEnhancedActs", mock.Anything, 2024).Return([]sejm.EnhancedAct{mockActs[1]}, nil)
	
	criteria := &service.SearchCriteria{
		YearFrom: &yearFrom,
		YearTo:   &yearTo,
	}
	
	result, err := searchService.SearchActs(context.Background(), criteria)
	
	assert.NoError(t, err)
	assert.Equal(t, 2, result.FilteredCount)
	assert.Len(t, result.Acts, 2)
}

func TestSearchActs_VotingFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping search test in short mode")
	}
	
	db := &MockDB{}
	searchService := service.NewSearchService(db)
	
	hasVotes := true
	
	mockActs := []sejm.EnhancedAct{
		{
			ID:        "DU/2024/1",
			Title:     "Act With Votes",
			Year:      2024,
			Position:  1,
			SejmVotes: []sejm.VotingRecord{{YesVotes: 300, NoVotes: 100}},
		},
		{
			ID:       "DU/2024/2",
			Title:    "Act Without Votes",
			Year:     2024,
			Position: 2,
		},
	}
	
	db.On("GetEnhancedActs", mock.Anything, mock.AnythingOfType("int")).Return(mockActs, nil)
	
	criteria := &service.SearchCriteria{
		HasSejmVotes: &hasVotes,
	}
	
	result, err := searchService.SearchActs(context.Background(), criteria)
	
	assert.NoError(t, err)
	assert.Equal(t, 1, result.FilteredCount)
	assert.Len(t, result.Acts, 1)
	assert.Equal(t, "Act With Votes", result.Acts[0].Title)
}

func TestSearchActs_Sorting(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping search test in short mode")
	}
	
	db := &MockDB{}
	searchService := service.NewSearchService(db)
	
	mockActs := []sejm.EnhancedAct{
		{
			ID:       "DU/2024/3",
			Title:    "C Act",
			Year:     2024,
			Position: 3,
		},
		{
			ID:       "DU/2024/1",
			Title:    "A Act",
			Year:     2024,
			Position: 1,
		},
		{
			ID:       "DU/2024/2",
			Title:    "B Act",
			Year:     2024,
			Position: 2,
		},
	}
	
	db.On("GetEnhancedActs", mock.Anything, mock.AnythingOfType("int")).Return(mockActs, nil)
	
	// Test title sorting ascending
	criteria := &service.SearchCriteria{
		SortBy:    "title",
		SortOrder: "asc",
	}
	
	result, err := searchService.SearchActs(context.Background(), criteria)
	
	assert.NoError(t, err)
	assert.Len(t, result.Acts, 3)
	assert.Equal(t, "A Act", result.Acts[0].Title)
	assert.Equal(t, "B Act", result.Acts[1].Title)
	assert.Equal(t, "C Act", result.Acts[2].Title)
}

func TestSearchActs_Pagination(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping search test in short mode")
	}
	
	db := &MockDB{}
	searchService := service.NewSearchService(db)
	
	var mockActs []sejm.EnhancedAct
	for i := 1; i <= 5; i++ {
		mockActs = append(mockActs, sejm.EnhancedAct{
			ID:       "DU/2024/" + string(rune(i)),
			Title:    "Act " + string(rune(i)),
			Year:     2024,
			Position: i,
		})
	}
	
	db.On("GetEnhancedActs", mock.Anything, mock.AnythingOfType("int")).Return(mockActs, nil)
	
	criteria := &service.SearchCriteria{
		Limit:  2,
		Offset: 1,
	}
	
	result, err := searchService.SearchActs(context.Background(), criteria)
	
	assert.NoError(t, err)
	assert.Equal(t, 5, result.FilteredCount)
	assert.Len(t, result.Acts, 2)
	// Should get acts at positions 1 and 2 (after offset of 1)
	assert.Equal(t, 2, result.Acts[0].Position)
	assert.Equal(t, 3, result.Acts[1].Position)
}

func TestSearchActs_Facets(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping search test in short mode")
	}
	
	db := &MockDB{}
	searchService := service.NewSearchService(db)
	
	mockActs := []sejm.EnhancedAct{
		{
			ID:            "DU/2024/1",
			Title:         "Act One",
			Year:          2024,
			Position:      1,
			Status:        "obowiązujący",
			CurrentStage:  "Opublikowano",
			InitiatorType: "Government",
			DaysInStage:   30,
		},
		{
			ID:            "DU/2024/2",
			Title:         "Act Two",
			Year:          2024,
			Position:      2,
			Status:        "pending",
			CurrentStage:  "Komisja",
			InitiatorType: "Government",
			DaysInStage:   15,
		},
		{
			ID:            "DU/2024/3",
			Title:         "Act Three",
			Year:          2024,
			Position:      3,
			Status:        "pending",
			CurrentStage:  "Komisja",
			InitiatorType: "Parliament",
			DaysInStage:   45,
		},
	}
	
	db.On("GetEnhancedActs", mock.Anything, mock.AnythingOfType("int")).Return(mockActs, nil)
	
	criteria := &service.SearchCriteria{}
	
	result, err := searchService.SearchActs(context.Background(), criteria)
	
	assert.NoError(t, err)
	assert.NotNil(t, result.Facets)
	
	// Check status facets
	assert.True(t, len(result.Facets.AvailableStatuses) > 0)
	
	// Check stage facets
	assert.True(t, len(result.Facets.AvailableStages) > 0)
	
	// Check year range
	assert.Equal(t, 2024, result.Facets.YearRange.Min)
	assert.Equal(t, 2024, result.Facets.YearRange.Max)
	
	// Check days range
	assert.Equal(t, 15, result.Facets.DaysInStageRange.Min)
	assert.Equal(t, 45, result.Facets.DaysInStageRange.Max)
}

func TestParseSearchCriteria(t *testing.T) {
	params := map[string][]string{
		"q":              {"healthcare"},
		"title":          {"education"},
		"status":         {"obowiązujący", "pending"},
		"year_from":      {"2023"},
		"year_to":        {"2024"},
		"has_sejm_votes": {"true"},
		"sort":           {"title"},
		"order":          {"asc"},
		"limit":          {"25"},
		"offset":         {"10"},
	}
	
	criteria := service.ParseSearchCriteria(params)
	
	assert.Equal(t, "healthcare", criteria.Query)
	assert.Equal(t, "education", criteria.TitleSearch)
	assert.Equal(t, []string{"obowiązujący", "pending"}, criteria.Statuses)
	assert.Equal(t, 2023, *criteria.YearFrom)
	assert.Equal(t, 2024, *criteria.YearTo)
	assert.True(t, *criteria.HasSejmVotes)
	assert.Equal(t, "title", criteria.SortBy)
	assert.Equal(t, "asc", criteria.SortOrder)
	assert.Equal(t, 25, criteria.Limit)
	assert.Equal(t, 10, criteria.Offset)
}

func TestGetSearchSuggestions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping search suggestions test in short mode")
	}
	
	db := &MockDB{}
	searchService := service.NewSearchService(db)
	
	mockActs := []sejm.EnhancedAct{
		{
			ID:            "DU/2024/1",
			Title:         "Healthcare Reform Act",
			InitiatorType: "Government Ministry",
			CurrentStage:  "Committee Review",
		},
		{
			ID:            "DU/2024/2",
			Title:         "Healthcare Improvement Bill",
			InitiatorType: "Government Agency",
			CurrentStage:  "Committee Discussion",
		},
		{
			ID:            "DU/2024/3",
			Title:         "Education Reform Act",
			InitiatorType: "Parliament Member",
			CurrentStage:  "Senate Review",
		},
	}
	
	db.On("GetEnhancedActs", mock.Anything, mock.AnythingOfType("int")).Return(mockActs, nil)
	
	// Test title suggestions
	suggestions, err := searchService.GetSearchSuggestions(context.Background(), "health", "title")
	
	assert.NoError(t, err)
	assert.True(t, len(suggestions) >= 2)
	assert.Contains(t, suggestions, "Healthcare Reform Act")
	assert.Contains(t, suggestions, "Healthcare Improvement Bill")
	
	// Test initiator suggestions
	suggestions, err = searchService.GetSearchSuggestions(context.Background(), "government", "initiator")
	
	assert.NoError(t, err)
	assert.True(t, len(suggestions) >= 2)
	assert.Contains(t, suggestions, "Government Ministry")
	assert.Contains(t, suggestions, "Government Agency")
}

func TestMatchesFilters_ComplexCriteria(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping complex filter test in short mode")
	}
	
	db := &MockDB{}
	searchService := service.NewSearchService(db)
	
	act := &sejm.EnhancedAct{
		ID:             "DU/2024/1",
		Title:          "Healthcare Reform Act",
		Year:           2024,
		Position:       1,
		Status:         "obowiązujący",
		DetailedStatus: "in_force",
		CurrentStage:   "Opublikowano",
		InitiatorType:  "Government",
		DaysInStage:    30,
		SejmVotes:      []sejm.VotingRecord{{YesVotes: 300, NoVotes: 100}},
		Tags:           []string{"healthcare", "reform"},
	}
	
	// Mock the database call
	db.On("GetEnhancedActs", mock.Anything, mock.AnythingOfType("int")).Return([]sejm.EnhancedAct{*act}, nil)
	
	// Should match: healthcare in title, status obowiązujący, has votes
	hasVotes := true
	daysMin := 20
	daysMax := 40
	
	criteria := &service.SearchCriteria{
		Query:          "healthcare",
		Statuses:       []string{"obowiązujący"},
		HasSejmVotes:   &hasVotes,
		DaysInStageMin: &daysMin,
		DaysInStageMax: &daysMax,
		Tags:           []string{"healthcare"},
	}
	
	result, err := searchService.SearchActs(context.Background(), criteria)
	
	assert.NoError(t, err)
	assert.Equal(t, 1, result.FilteredCount)
	assert.Len(t, result.Acts, 1)
	
	// Should not match: different status
	criteria.Statuses = []string{"pending"}
	result, err = searchService.SearchActs(context.Background(), criteria)
	
	assert.NoError(t, err)
	assert.Equal(t, 0, result.FilteredCount)
	assert.Len(t, result.Acts, 0)
}

func BenchmarkSearchActs(b *testing.B) {
	db := &MockDB{}
	searchService := service.NewSearchService(db)
	
	// Create a larger dataset for benchmarking
	var mockActs []sejm.EnhancedAct
	for i := 1; i <= 100; i++ {
		mockActs = append(mockActs, sejm.EnhancedAct{
			ID:       "DU/2024/" + string(rune(i)),
			Title:    "Test Act " + string(rune(i)),
			Year:     2024,
			Position: i,
			Status:   "obowiązujący",
		})
	}
	
	db.On("GetEnhancedActs", mock.Anything, mock.AnythingOfType("int")).Return(mockActs, nil)
	
	criteria := &service.SearchCriteria{
		Query: "test",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := searchService.SearchActs(context.Background(), criteria)
		if err != nil {
			b.Error(err)
		}
	}
}