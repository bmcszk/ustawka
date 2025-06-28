package service_test

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
	"ustawka/sejm"
	"ustawka/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewExportService(t *testing.T) {
	db := &MockDB{}
	exportService := service.NewExportService(db)
	
	assert.NotNil(t, exportService)
}

func TestExportActs_JSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping export test in short mode")
	}
	
	db := &MockDB{}
	exportService := service.NewExportService(db)
	
	// Mock data
	mockActs := []sejm.EnhancedAct{
		{
			ID:             "DU/2024/1",
			Title:          "Test Act 1",
			Year:           2024,
			Position:       1,
			Status:         "obowiązujący",
			DetailedStatus: "in_force",
			CurrentStage:   "Opublikowano",
			StageDate:      time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			DaysInStage:    30,
			InitiatorType:  "Government",
		},
		{
			ID:             "DU/2024/2",
			Title:          "Test Act 2",
			Year:           2024,
			Position:       2,
			Status:         "pending",
			DetailedStatus: "committee_work",
			CurrentStage:   "Komisja",
			StageDate:      time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			DaysInStage:    15,
			InitiatorType:  "Parliament",
		},
	}
	
	year := 2024
	db.On("GetEnhancedActs", mock.Anything, year).Return(mockActs, nil)
	
	req := &service.ExportRequest{
		Format: service.ExportFormatJSON,
		Year:   &year,
	}
	
	result, err := exportService.ExportActs(context.Background(), req)
	
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "application/json", result.ContentType)
	assert.Equal(t, 2, result.RecordCount)
	assert.True(t, len(result.Data) > 0)
	assert.Contains(t, result.Filename, "acts_2024")
	assert.Contains(t, result.Filename, ".json")
	
	// Verify JSON structure
	var exported map[string]any
	err = json.Unmarshal(result.Data, &exported)
	assert.NoError(t, err)
	assert.Contains(t, exported, "metadata")
	assert.Contains(t, exported, "acts")
}

func TestExportActs_CSV(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping export test in short mode")
	}
	
	db := &MockDB{}
	exportService := service.NewExportService(db)
	
	mockActs := []sejm.EnhancedAct{
		{
			ID:             "DU/2024/1",
			Title:          "Test Act 1",
			Year:           2024,
			Position:       1,
			Status:         "obowiązujący",
			DetailedStatus: "in_force",
			CurrentStage:   "Opublikowano",
			StageDate:      time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			DaysInStage:    30,
			InitiatorType:  "Government",
			SejmVotes:      []sejm.VotingRecord{{YesVotes: 300, NoVotes: 100}},
		},
	}
	
	year := 2024
	db.On("GetEnhancedActs", mock.Anything, year).Return(mockActs, nil)
	
	req := &service.ExportRequest{
		Format:        service.ExportFormatCSV,
		Year:          &year,
		IncludeVoting: true,
	}
	
	result, err := exportService.ExportActs(context.Background(), req)
	
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "text/csv", result.ContentType)
	assert.Equal(t, 1, result.RecordCount)
	assert.Contains(t, result.Filename, ".csv")
	
	// Verify CSV structure
	reader := csv.NewReader(strings.NewReader(string(result.Data)))
	records, err := reader.ReadAll()
	assert.NoError(t, err)
	assert.True(t, len(records) >= 2) // Header + at least one data row
	
	// Check header contains expected columns
	header := records[0]
	assert.Contains(t, header, "ID")
	assert.Contains(t, header, "Tytuł")
	assert.Contains(t, header, "Status")
	assert.Contains(t, header, "Głosowania Sejm") // Because IncludeVoting is true
}

func TestExportActs_PDF(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping export test in short mode")
	}
	
	db := &MockDB{}
	exportService := service.NewExportService(db)
	
	mockActs := []sejm.EnhancedAct{
		{
			ID:             "DU/2024/1",
			Title:          "Test Act 1",
			Year:           2024,
			Position:       1,
			Status:         "obowiązujący",
			DetailedStatus: "in_force",
			CurrentStage:   "Opublikowano",
			InitiatorType:  "Government",
		},
	}
	
	year := 2024
	db.On("GetEnhancedActs", mock.Anything, year).Return(mockActs, nil)
	
	req := &service.ExportRequest{
		Format: service.ExportFormatPDF,
		Year:   &year,
	}
	
	result, err := exportService.ExportActs(context.Background(), req)
	
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "application/pdf", result.ContentType)
	assert.Equal(t, 1, result.RecordCount)
	assert.Contains(t, result.Filename, ".pdf")
	assert.True(t, len(result.Data) > 0)
	
	// Basic check that PDF content contains expected text
	content := string(result.Data)
	assert.Contains(t, content, "RAPORT AKTÓW PRAWNYCH")
	assert.Contains(t, content, "Test Act 1")
}

func TestExportActs_WithFilters(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping export test in short mode")
	}
	
	db := &MockDB{}
	exportService := service.NewExportService(db)
	
	mockActs := []sejm.EnhancedAct{
		{
			ID:     "DU/2024/1",
			Title:  "Healthcare Act",
			Year:   2024,
			Status: "obowiązujący",
		},
		{
			ID:     "DU/2024/2",
			Title:  "Education Act",
			Year:   2024,
			Status: "pending",
		},
		{
			ID:     "DU/2024/3",
			Title:  "Healthcare Amendment",
			Year:   2024,
			Status: "obowiązujący",
		},
	}
	
	year := 2024
	db.On("GetEnhancedActs", mock.Anything, year).Return(mockActs, nil)
	
	// Test status filter
	req := &service.ExportRequest{
		Format: service.ExportFormatJSON,
		Year:   &year,
		Status: []string{"obowiązujący"},
	}
	
	result, err := exportService.ExportActs(context.Background(), req)
	
	assert.NoError(t, err)
	assert.Equal(t, 2, result.RecordCount) // Only acts with "obowiązujący" status
	
	// Test title filter
	req.Status = nil
	req.Title = "healthcare"
	
	result, err = exportService.ExportActs(context.Background(), req)
	
	assert.NoError(t, err)
	assert.Equal(t, 2, result.RecordCount) // Acts containing "healthcare" in title
}

func TestExportComparison_JSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping export test in short mode")
	}
	
	db := &MockDB{}
	exportService := service.NewExportService(db)
	
	comparison := &service.ActComparison{
		LeftAct: &sejm.EnhancedAct{
			ID:     "DU/2024/1",
			Title:  "Act A",
			Status: "obowiązujący",
		},
		RightAct: &sejm.EnhancedAct{
			ID:     "DU/2024/2",
			Title:  "Act B",
			Status: "pending",
		},
		Differences: []service.FieldDifference{
			{
				Field:         "status",
				FieldLabel:    "Status",
				LeftValue:     "obowiązujący",
				RightValue:    "pending",
				DifferenceType: "value_changed",
				Severity:      "critical",
				Description:   "Different status",
			},
		},
		Similarities: []service.FieldSimilarity{
			{
				Field:       "year",
				FieldLabel:  "Year",
				Value:       2024,
				Description: "Same year",
			},
		},
		ComparisonID: "test_comparison",
		CreatedAt:    time.Now(),
		Summary: service.ComparisonSummary{
			TotalFields:     10,
			DifferentFields: 1,
			SimilarFields:   9,
			CriticalDiffs:   1,
		},
	}
	
	result, err := exportService.ExportComparison(context.Background(), comparison, service.ExportFormatJSON)
	
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "application/json", result.ContentType)
	assert.Equal(t, 1, result.RecordCount)
	assert.Contains(t, result.Filename, "comparison_")
	assert.Contains(t, result.Filename, ".json")
	
	// Verify JSON structure
	var exported service.ActComparison
	err = json.Unmarshal(result.Data, &exported)
	assert.NoError(t, err)
	assert.Equal(t, comparison.ComparisonID, exported.ComparisonID)
	assert.Equal(t, len(comparison.Differences), len(exported.Differences))
}

func TestExportComparison_CSV(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping export test in short mode")
	}
	
	db := &MockDB{}
	exportService := service.NewExportService(db)
	
	comparison := &service.ActComparison{
		LeftAct: &sejm.EnhancedAct{
			ID:     "DU/2024/1",
			Title:  "Act A",
		},
		RightAct: &sejm.EnhancedAct{
			ID:     "DU/2024/2",
			Title:  "Act B",
		},
		Differences: []service.FieldDifference{
			{
				Field:         "title",
				FieldLabel:    "Tytuł",
				LeftValue:     "Act A",
				RightValue:    "Act B",
				DifferenceType: "value_changed",
				Severity:      "major",
				Description:   "Different titles",
			},
		},
		CreatedAt: time.Now(),
		Summary: service.ComparisonSummary{
			TotalFields:     5,
			DifferentFields: 1,
			SimilarFields:   4,
		},
	}
	
	result, err := exportService.ExportComparison(context.Background(), comparison, service.ExportFormatCSV)
	
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "text/csv", result.ContentType)
	assert.Contains(t, result.Filename, ".csv")
	
	// Verify CSV contains comparison data
	content := string(result.Data)
	assert.Contains(t, content, "Porównanie aktów prawnych")
	assert.Contains(t, content, "DU/2024/1")
	assert.Contains(t, content, "DU/2024/2")
	assert.Contains(t, content, "Różnice szczegółowe")
}

func TestExportComparison_PDF(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping export test in short mode")
	}
	
	db := &MockDB{}
	exportService := service.NewExportService(db)
	
	comparison := &service.ActComparison{
		LeftAct: &sejm.EnhancedAct{
			ID:     "DU/2024/1",
			Title:  "Healthcare Act",
			Status: "obowiązujący",
		},
		RightAct: &sejm.EnhancedAct{
			ID:     "DU/2024/2",
			Title:  "Education Act",
			Status: "pending",
		},
		Differences: []service.FieldDifference{
			{
				Field:         "title",
				FieldLabel:    "Tytuł",
				LeftValue:     "Healthcare Act",
				RightValue:    "Education Act",
				DifferenceType: "value_changed",
				Severity:      "major",
				Description:   "Different act titles",
			},
		},
		CreatedAt: time.Now(),
		Summary: service.ComparisonSummary{
			TotalFields:     8,
			DifferentFields: 3,
			SimilarFields:   5,
			CriticalDiffs:   1,
		},
	}
	
	result, err := exportService.ExportComparison(context.Background(), comparison, service.ExportFormatPDF)
	
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "application/pdf", result.ContentType)
	assert.Contains(t, result.Filename, ".pdf")
	
	// Verify PDF content
	content := string(result.Data)
	assert.Contains(t, content, "PORÓWNANIE AKTÓW PRAWNYCH")
	assert.Contains(t, content, "Healthcare Act")
	assert.Contains(t, content, "Education Act")
	assert.Contains(t, content, "RÓŻNICE:")
	assert.Contains(t, content, "PODSUMOWANIE:")
}

func TestExportRequest_UnsupportedFormat(t *testing.T) {
	db := &MockDB{}
	exportService := service.NewExportService(db)
	
	// Mock the database calls that will be made before format validation
	currentYear := time.Now().Year()
	for i := 0; i < 3; i++ {
		year := currentYear - i
		db.On("GetEnhancedActs", mock.Anything, year).Return([]sejm.EnhancedAct{}, nil)
	}
	
	req := &service.ExportRequest{
		Format: "unsupported",
	}
	
	result, err := exportService.ExportActs(context.Background(), req)
	
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unsupported export format")
}

func TestExportActs_NoYear_MultipleYears(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping export test in short mode")
	}
	
	db := &MockDB{}
	exportService := service.NewExportService(db)
	
	currentYear := time.Now().Year()
	
	// Mock data for multiple years
	acts2023 := []sejm.EnhancedAct{{ID: "DU/2023/1", Year: 2023}}
	acts2024 := []sejm.EnhancedAct{{ID: "DU/2024/1", Year: 2024}}
	
	db.On("GetEnhancedActs", mock.Anything, currentYear-2).Return(acts2023, nil)
	db.On("GetEnhancedActs", mock.Anything, currentYear-1).Return(acts2024, nil)
	db.On("GetEnhancedActs", mock.Anything, currentYear).Return([]sejm.EnhancedAct{}, nil)
	
	req := &service.ExportRequest{
		Format: service.ExportFormatJSON,
		// No year specified - should get multiple years
	}
	
	result, err := exportService.ExportActs(context.Background(), req)
	
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, result.RecordCount) // Acts from both years
}

func TestSanitizeFilename(t *testing.T) {
	// This tests the internal sanitizeFilename function through export
	db := &MockDB{}
	exportService := service.NewExportService(db)
	
	comparison := &service.ActComparison{
		LeftAct: &sejm.EnhancedAct{
			ID: "DU/2024/1:test*file?name",
		},
		RightAct: &sejm.EnhancedAct{
			ID: "DU/2024/2<>|test",
		},
		CreatedAt: time.Now(),
		Summary:   service.ComparisonSummary{},
	}
	
	result, err := exportService.ExportComparison(context.Background(), comparison, service.ExportFormatJSON)
	
	assert.NoError(t, err)
	assert.NotNil(t, result)
	
	// Filename should not contain invalid characters
	assert.NotContains(t, result.Filename, "/")
	assert.NotContains(t, result.Filename, ":")
	assert.NotContains(t, result.Filename, "*")
	assert.NotContains(t, result.Filename, "?")
	assert.NotContains(t, result.Filename, "<")
	assert.NotContains(t, result.Filename, ">")
	assert.NotContains(t, result.Filename, "|")
}

func TestExportResult_Metadata(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping export test in short mode")
	}
	
	db := &MockDB{}
	exportService := service.NewExportService(db)
	
	mockActs := []sejm.EnhancedAct{{ID: "DU/2024/1", Year: 2024}}
	year := 2024
	db.On("GetEnhancedActs", mock.Anything, year).Return(mockActs, nil)
	
	req := &service.ExportRequest{
		Format: service.ExportFormatJSON,
		Year:   &year,
	}
	
	result, err := exportService.ExportActs(context.Background(), req)
	
	assert.NoError(t, err)
	assert.NotNil(t, result)
	
	// Check metadata fields
	assert.True(t, len(result.Data) > 0)
	assert.True(t, len(result.Filename) > 0)
	assert.Equal(t, "application/json", result.ContentType)
	assert.Equal(t, len(result.Data), result.Size)
	assert.Equal(t, 1, result.RecordCount)
	assert.False(t, result.GeneratedAt.IsZero())
}

func BenchmarkExportActsJSON(b *testing.B) {
	db := &MockDB{}
	exportService := service.NewExportService(db)
	
	// Create larger dataset for benchmarking
	var mockActs []sejm.EnhancedAct
	for i := 1; i <= 100; i++ {
		mockActs = append(mockActs, sejm.EnhancedAct{
			ID:       fmt.Sprintf("DU/2024/%d", i),
			Title:    fmt.Sprintf("Test Act %d", i),
			Year:     2024,
			Position: i,
			Status:   "obowiązujący",
		})
	}
	
	year := 2024
	db.On("GetEnhancedActs", mock.Anything, year).Return(mockActs, nil)
	
	req := &service.ExportRequest{
		Format: service.ExportFormatJSON,
		Year:   &year,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := exportService.ExportActs(context.Background(), req)
		if err != nil {
			b.Error(err)
		}
	}
}