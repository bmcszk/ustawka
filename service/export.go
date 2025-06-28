package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"ustawka/sejm"
)

// ExportService handles data export functionality
type ExportService struct {
	db Database
}

// NewExportService creates a new export service
func NewExportService(db Database) *ExportService {
	return &ExportService{
		db: db,
	}
}

// ExportFormat represents supported export formats
type ExportFormat string

const (
	ExportFormatJSON ExportFormat = "json"
	ExportFormatCSV  ExportFormat = "csv"
	ExportFormatPDF  ExportFormat = "pdf"
)

// ExportRequest contains parameters for export operation
type ExportRequest struct {
	Format      ExportFormat `json:"format"`
	Year        *int         `json:"year,omitempty"`
	Status      []string     `json:"status,omitempty"`
	Title       string       `json:"title,omitempty"`
	IncludeVoting bool       `json:"include_voting"`
	IncludeStages bool       `json:"include_stages"`
	DateFrom    *time.Time   `json:"date_from,omitempty"`
	DateTo      *time.Time   `json:"date_to,omitempty"`
}

// ExportResult contains the exported data and metadata
type ExportResult struct {
	Data        []byte    `json:"data"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	Size        int       `json:"size"`
	RecordCount int       `json:"record_count"`
	GeneratedAt time.Time `json:"generated_at"`
}

// ExportActs exports legislative acts based on the provided criteria
func (es *ExportService) ExportActs(ctx context.Context, req *ExportRequest) (*ExportResult, error) {
	// Get acts based on criteria
	acts, err := es.getFilteredActs(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get filtered acts: %w", err)
	}

	// Generate export data based on format
	var data []byte
	var contentType string
	var filename string

	switch req.Format {
	case ExportFormatJSON:
		data, err = es.exportActsJSON(acts, req)
		contentType = "application/json"
		filename = es.generateFilename("acts", "json", req.Year)
	case ExportFormatCSV:
		data, err = es.exportActsCSV(acts, req)
		contentType = "text/csv"
		filename = es.generateFilename("acts", "csv", req.Year)
	case ExportFormatPDF:
		data, err = es.exportActsPDF(acts, req)
		contentType = "application/pdf"
		filename = es.generateFilename("acts", "pdf", req.Year)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", req.Format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to export acts as %s: %w", req.Format, err)
	}

	return &ExportResult{
		Data:        data,
		Filename:    filename,
		ContentType: contentType,
		Size:        len(data),
		RecordCount: len(acts),
		GeneratedAt: time.Now(),
	}, nil
}

// ExportComparison exports act comparison results
func (es *ExportService) ExportComparison(ctx context.Context, comparison *ActComparison, format ExportFormat) (*ExportResult, error) {
	var data []byte
	var contentType string
	var filename string
	var err error

	switch format {
	case ExportFormatJSON:
		data, err = json.MarshalIndent(comparison, "", "  ")
		contentType = "application/json"
		filename = fmt.Sprintf("comparison_%s_vs_%s_%s.json", 
			sanitizeFilename(comparison.LeftAct.ID), 
			sanitizeFilename(comparison.RightAct.ID),
			time.Now().Format("20060102_150405"))
	case ExportFormatCSV:
		data, err = es.exportComparisonCSV(comparison)
		contentType = "text/csv"
		filename = fmt.Sprintf("comparison_%s_vs_%s_%s.csv", 
			sanitizeFilename(comparison.LeftAct.ID), 
			sanitizeFilename(comparison.RightAct.ID),
			time.Now().Format("20060102_150405"))
	case ExportFormatPDF:
		data, err = es.exportComparisonPDF(comparison)
		contentType = "application/pdf"
		filename = fmt.Sprintf("comparison_%s_vs_%s_%s.pdf", 
			sanitizeFilename(comparison.LeftAct.ID), 
			sanitizeFilename(comparison.RightAct.ID),
			time.Now().Format("20060102_150405"))
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to export comparison as %s: %w", format, err)
	}

	return &ExportResult{
		Data:        data,
		Filename:    filename,
		ContentType: contentType,
		Size:        len(data),
		RecordCount: 1, // One comparison
		GeneratedAt: time.Now(),
	}, nil
}

// getFilteredActs retrieves acts based on export criteria
func (es *ExportService) getFilteredActs(ctx context.Context, req *ExportRequest) ([]sejm.EnhancedAct, error) {
	var allActs []sejm.EnhancedAct

	if req.Year != nil {
		// Get acts for specific year
		acts, err := es.db.GetEnhancedActs(ctx, *req.Year)
		if err != nil {
			return nil, err
		}
		allActs = acts
	} else {
		// Get acts for current and recent years
		currentYear := time.Now().Year()
		for year := currentYear - 2; year <= currentYear; year++ {
			acts, err := es.db.GetEnhancedActs(ctx, year)
			if err != nil {
				continue // Skip years with errors
			}
			allActs = append(allActs, acts...)
		}
	}

	// Apply filters
	filtered := make([]sejm.EnhancedAct, 0, len(allActs))
	for _, act := range allActs {
		if es.matchesFilters(&act, req) {
			filtered = append(filtered, act)
		}
	}

	return filtered, nil
}

// matchesFilters checks if an act matches the export criteria
func (es *ExportService) matchesFilters(act *sejm.EnhancedAct, req *ExportRequest) bool {
	// Status filter
	if len(req.Status) > 0 {
		found := false
		for _, status := range req.Status {
			if act.Status == status {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Title filter
	if req.Title != "" {
		if !strings.Contains(strings.ToLower(act.Title), strings.ToLower(req.Title)) {
			return false
		}
	}

	// Date filters
	if req.DateFrom != nil && !act.StageDate.IsZero() {
		if act.StageDate.Before(*req.DateFrom) {
			return false
		}
	}

	if req.DateTo != nil && !act.StageDate.IsZero() {
		if act.StageDate.After(*req.DateTo) {
			return false
		}
	}

	return true
}

// exportActsJSON exports acts as JSON
func (es *ExportService) exportActsJSON(acts []sejm.EnhancedAct, req *ExportRequest) ([]byte, error) {
	export := map[string]interface{}{
		"metadata": map[string]interface{}{
			"exported_at":    time.Now(),
			"record_count":   len(acts),
			"export_format":  "json",
			"criteria":       req,
		},
		"acts": acts,
	}

	return json.MarshalIndent(export, "", "  ")
}

// exportActsCSV exports acts as CSV
func (es *ExportService) exportActsCSV(acts []sejm.EnhancedAct, req *ExportRequest) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header
	header := []string{
		"ID", "Tytuł", "Rok", "Pozycja", "Status", "Status szczegółowy",
		"Aktualny etap", "Data etapu", "Dni w etapie", "Inicjator",
	}

	if req.IncludeVoting {
		header = append(header, "Głosowania Sejm", "Głosowania Senat")
	}

	if req.IncludeStages {
		header = append(header, "Liczba etapów")
	}

	if err := writer.Write(header); err != nil {
		return nil, err
	}

	// Write data rows
	for _, act := range acts {
		row := []string{
			act.ID,
			act.Title,
			strconv.Itoa(act.Year),
			strconv.Itoa(act.Position),
			act.Status,
			act.DetailedStatus,
			act.CurrentStage,
			act.StageDate.Format("2006-01-02"),
			strconv.Itoa(act.DaysInStage),
			act.InitiatorType,
		}

		if req.IncludeVoting {
			row = append(row, 
				strconv.Itoa(len(act.SejmVotes)),
				strconv.Itoa(len(act.SenateVotes)))
		}

		if req.IncludeStages {
			row = append(row, strconv.Itoa(len(act.Stages)))
		}

		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// exportActsPDF exports acts as PDF (basic implementation)
func (es *ExportService) exportActsPDF(acts []sejm.EnhancedAct, req *ExportRequest) ([]byte, error) {
	// Basic PDF implementation - in a real implementation, you'd use a PDF library
	// For now, we'll create a text-based representation
	var buf bytes.Buffer
	
	buf.WriteString("RAPORT AKTÓW PRAWNYCH\n")
	buf.WriteString("======================\n\n")
	buf.WriteString(fmt.Sprintf("Wygenerowano: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	buf.WriteString(fmt.Sprintf("Liczba aktów: %d\n\n", len(acts)))

	for i, act := range acts {
		buf.WriteString(fmt.Sprintf("%d. %s\n", i+1, act.Title))
		buf.WriteString(fmt.Sprintf("   ID: %s\n", act.ID))
		buf.WriteString(fmt.Sprintf("   Status: %s\n", act.Status))
		buf.WriteString(fmt.Sprintf("   Rok: %d, Pozycja: %d\n", act.Year, act.Position))
		buf.WriteString(fmt.Sprintf("   Etap: %s\n", act.CurrentStage))
		if !act.StageDate.IsZero() {
			buf.WriteString(fmt.Sprintf("   Data etapu: %s (%d dni)\n", 
				act.StageDate.Format("2006-01-02"), act.DaysInStage))
		}
		buf.WriteString(fmt.Sprintf("   Inicjator: %s\n", act.InitiatorType))
		
		if req.IncludeVoting && (len(act.SejmVotes) > 0 || len(act.SenateVotes) > 0) {
			buf.WriteString(fmt.Sprintf("   Głosowania: Sejm (%d), Senat (%d)\n", 
				len(act.SejmVotes), len(act.SenateVotes)))
		}
		
		buf.WriteString("\n")
	}

	// Note: In a real implementation, you would use a proper PDF library like:
	// - github.com/jung-kurt/gofpdf
	// - github.com/johnfercher/maroto
	// - github.com/signintech/gopdf
	
	return buf.Bytes(), nil
}

// exportComparisonCSV exports comparison results as CSV
func (es *ExportService) exportComparisonCSV(comparison *ActComparison) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write metadata
	metadata := [][]string{
		{"Porównanie aktów prawnych"},
		{"Akt A", comparison.LeftAct.ID, comparison.LeftAct.Title},
		{"Akt B", comparison.RightAct.ID, comparison.RightAct.Title},
		{"Data porównania", comparison.CreatedAt.Format("2006-01-02 15:04:05")},
		{""},
		{"Podsumowanie"},
		{"Łączna liczba pól", strconv.Itoa(comparison.Summary.TotalFields)},
		{"Różnice", strconv.Itoa(comparison.Summary.DifferentFields)},
		{"Podobieństwa", strconv.Itoa(comparison.Summary.SimilarFields)},
		{"Krytyczne różnice", strconv.Itoa(comparison.Summary.CriticalDiffs)},
		{""},
		{"Różnice szczegółowe"},
		{"Pole", "Etykieta", "Wartość A", "Wartość B", "Typ różnicy", "Ważność", "Opis"},
	}

	for _, row := range metadata {
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	// Write differences
	for _, diff := range comparison.Differences {
		row := []string{
			diff.Field,
			diff.FieldLabel,
			fmt.Sprintf("%v", diff.LeftValue),
			fmt.Sprintf("%v", diff.RightValue),
			string(diff.DifferenceType),
			string(diff.Severity),
			diff.Description,
		}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	// Write similarities
	if len(comparison.Similarities) > 0 {
		if err := es.writeSimilaritiesToCSV(writer, comparison.Similarities); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	return buf.Bytes(), nil
}

func (*ExportService) writeSimilaritiesToCSV(writer *csv.Writer, similarities []FieldSimilarity) error {
	if err := writer.Write([]string{""}); err != nil {
		return fmt.Errorf("failed to write CSV separator: %w", err)
	}
	if err := writer.Write([]string{"Podobieństwa"}); err != nil {
		return fmt.Errorf("failed to write CSV similarities header: %w", err)
	}
	if err := writer.Write([]string{"Pole", "Etykieta", "Wartość", "Opis"}); err != nil {
		return fmt.Errorf("failed to write CSV similarities columns: %w", err)
	}

	for _, sim := range similarities {
		row := []string{
			sim.Field,
			sim.FieldLabel,
			fmt.Sprintf("%v", sim.Value),
			sim.Description,
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	return nil
}

// exportComparisonPDF exports comparison results as PDF (basic implementation)
func (es *ExportService) exportComparisonPDF(comparison *ActComparison) ([]byte, error) {
	var buf bytes.Buffer
	
	buf.WriteString("PORÓWNANIE AKTÓW PRAWNYCH\n")
	buf.WriteString("==========================\n\n")
	
	buf.WriteString(fmt.Sprintf("Data porównania: %s\n\n", comparison.CreatedAt.Format("2006-01-02 15:04:05")))
	
	buf.WriteString("AKT A:\n")
	buf.WriteString(fmt.Sprintf("  ID: %s\n", comparison.LeftAct.ID))
	buf.WriteString(fmt.Sprintf("  Tytuł: %s\n", comparison.LeftAct.Title))
	buf.WriteString(fmt.Sprintf("  Status: %s\n\n", comparison.LeftAct.Status))
	
	buf.WriteString("AKT B:\n")
	buf.WriteString(fmt.Sprintf("  ID: %s\n", comparison.RightAct.ID))
	buf.WriteString(fmt.Sprintf("  Tytuł: %s\n", comparison.RightAct.Title))
	buf.WriteString(fmt.Sprintf("  Status: %s\n\n", comparison.RightAct.Status))
	
	buf.WriteString("PODSUMOWANIE:\n")
	buf.WriteString(fmt.Sprintf("  Łączna liczba pól: %d\n", comparison.Summary.TotalFields))
	buf.WriteString(fmt.Sprintf("  Różnice: %d\n", comparison.Summary.DifferentFields))
	buf.WriteString(fmt.Sprintf("  Podobieństwa: %d\n", comparison.Summary.SimilarFields))
	buf.WriteString(fmt.Sprintf("  Krytyczne różnice: %d\n\n", comparison.Summary.CriticalDiffs))
	
	if len(comparison.Differences) > 0 {
		buf.WriteString("RÓŻNICE:\n")
		for i, diff := range comparison.Differences {
			buf.WriteString(fmt.Sprintf("%d. %s (%s)\n", i+1, diff.FieldLabel, diff.Severity))
			buf.WriteString(fmt.Sprintf("   Akt A: %v\n", diff.LeftValue))
			buf.WriteString(fmt.Sprintf("   Akt B: %v\n", diff.RightValue))
			buf.WriteString(fmt.Sprintf("   Opis: %s\n\n", diff.Description))
		}
	}
	
	if len(comparison.Similarities) > 0 {
		buf.WriteString("PODOBIEŃSTWA:\n")
		for i, sim := range comparison.Similarities {
			buf.WriteString(fmt.Sprintf("%d. %s\n", i+1, sim.FieldLabel))
			buf.WriteString(fmt.Sprintf("   Wartość: %v\n", sim.Value))
			buf.WriteString(fmt.Sprintf("   Opis: %s\n\n", sim.Description))
		}
	}

	return buf.Bytes(), nil
}

// generateFilename generates a filename for export
func (es *ExportService) generateFilename(prefix, extension string, year *int) string {
	timestamp := time.Now().Format("20060102_150405")
	
	if year != nil {
		return fmt.Sprintf("%s_%d_%s.%s", prefix, *year, timestamp, extension)
	}
	
	return fmt.Sprintf("%s_%s.%s", prefix, timestamp, extension)
}

// sanitizeFilename removes invalid characters from filename
func sanitizeFilename(input string) string {
	// Replace invalid filename characters
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)
	
	return replacer.Replace(input)
}