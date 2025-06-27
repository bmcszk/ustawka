package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"ustawka/sejm"
)

// EnrichmentService handles Act status enrichment and data enhancement
type EnrichmentService struct {
	sejmClient   *sejm.Client
	senateClient sejm.SenateClient
}

// EnrichmentResult contains the result of enrichment process
type EnrichmentResult struct {
	EnhancedAct     *sejm.EnhancedAct
	ConfidenceLevel string // "high", "medium", "low"
	DataSources     []string
	Warnings        []string
	ProcessedAt     time.Time
}

// NewEnrichmentService creates a new enrichment service
func NewEnrichmentService(sejmClient *sejm.Client, senateClient sejm.SenateClient) *EnrichmentService {
	return &EnrichmentService{
		sejmClient:   sejmClient,
		senateClient: senateClient,
	}
}

// EnrichAct performs comprehensive enrichment of an Act
func (es *EnrichmentService) EnrichAct(ctx context.Context, act *sejm.EnhancedAct) (*EnrichmentResult, error) {
	result := &EnrichmentResult{
		EnhancedAct:     act,
		ConfidenceLevel: "high",
		DataSources:     []string{},
		Warnings:        []string{},
		ProcessedAt:     time.Now(),
	}

	// 1. Enrich with process information
	if err := es.enrichWithProcessData(ctx, act, result); err != nil {
		slog.Warn("Failed to enrich with process data", "act_id", act.ID, "error", err)
		result.Warnings = append(result.Warnings, fmt.Sprintf("Process data enrichment failed: %v", err))
		result.ConfidenceLevel = "medium"
	}

	// 2. Determine enhanced status
	es.determineEnhancedStatus(act, result)

	// 3. Extract and enrich metadata
	es.enrichMetadata(act, result)

	// 4. Generate tags
	es.generateTags(act, result)

	// 5. Calculate stage metrics
	es.calculateStageMetrics(act, result)

	return result, nil
}

// enrichWithProcessData enriches Act with process information from Sejm API
func (es *EnrichmentService) enrichWithProcessData(ctx context.Context,
	act *sejm.EnhancedAct, result *EnrichmentResult) error {
	// Try to find process information using various strategies
	processInfo, err := es.findProcessInfo(ctx, act)
	if err != nil {
		return fmt.Errorf("failed to find process info: %w", err)
	}

	if processInfo == nil {
		result.Warnings = append(result.Warnings, "No process information found")
		return nil
	}

	result.DataSources = append(result.DataSources, "sejm_process_api")

	// Extract process information
	act.InitiatorType = sejm.DetermineInitiatorType(processInfo)
	act.UrgencyStatus = strings.ToLower(processInfo.UrgencyStatus)
	act.EUCompliance = processInfo.PrincipleOfSubsidiarity
	act.ProcessPrintNumber = processInfo.Number

	// Extract stages
	stages, currentStage := es.extractProcessStages(processInfo)
	act.Stages = stages
	
	if currentStage != nil {
		act.CurrentStage = currentStage.StageName
		act.StageDate = currentStage.StageDate
		act.DaysInStage = sejm.CalculateDaysInStage(currentStage.StageDate)
	}

	return nil
}

// findProcessInfo attempts to find process information for an Act
func (es *EnrichmentService) findProcessInfo(ctx context.Context, act *sejm.EnhancedAct) (*sejm.ProcessInfo, error) {
	// Strategy 1: Use existing process print number if available
	if act.ProcessPrintNumber != "" {
		return es.getProcessByPrintNumber(ctx, act.ProcessPrintNumber)
	}

	// Strategy 2: Search by title matching
	return es.searchProcessByTitle(ctx, act)
}

// getProcessByPrintNumber gets process info by print number
func (es *EnrichmentService) getProcessByPrintNumber(ctx context.Context,
	printNumber string) (*sejm.ProcessInfo, error) {
	// Extract term from current context (simplified - would be configurable)
	term := 10 // Current Sejm term
	
	processInfo, err := es.sejmClient.GetProcessInfo(ctx, term, printNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get process info for print %s: %w", printNumber, err)
	}

	return processInfo, nil
}

// searchProcessByTitle searches for process by title matching
func (es *EnrichmentService) searchProcessByTitle(ctx context.Context, _ *sejm.EnhancedAct) (*sejm.ProcessInfo, error) {
	// This would require searching through prints and matching titles
	// For now, we'll implement a simplified version
	
	term := 10
	prints, err := es.sejmClient.GetPrints(ctx, term)
	if err != nil {
		return nil, fmt.Errorf("failed to get prints: %w", err)
	}

	// This would require parsing prints data and matching titles
	// For now, return nil to indicate no match found
	slog.Debug("Print search not yet implemented", "prints_count", len(prints))
	return nil, nil
}

// extractProcessStages extracts process stages from Sejm process info
func (*EnrichmentService) extractProcessStages(processInfo *sejm.ProcessInfo) (
	[]sejm.ProcessStage, *sejm.ProcessStage) {
	var stages []sejm.ProcessStage
	var currentStage *sejm.ProcessStage

	for i, apiStage := range processInfo.Stages {
		stageDate, _ := time.Parse("2006-01-02", apiStage.Date)
		
		stage := sejm.ProcessStage{
			ID:            i + 1,
			StageName:     apiStage.StageName,
			StageDate:     stageDate,
			StageOrder:    i + 1,
			PrintNumbers:  []string{apiStage.PrintNumber},
			IsCurrent:     i == len(processInfo.Stages)-1, // Last stage is current
		}

		// Calculate duration if not the first stage
		if i > 0 {
			prevStageDate, _ := time.Parse("2006-01-02", processInfo.Stages[i-1].Date)
			stage.DurationDays = int(stageDate.Sub(prevStageDate).Hours() / 24)
		}

		stages = append(stages, stage)

		// Update current stage
		if stage.IsCurrent {
			currentStage = &stage
		}
	}

	return stages, currentStage
}

// determineEnhancedStatus determines the enhanced status based on available information
func (es *EnrichmentService) determineEnhancedStatus(act *sejm.EnhancedAct, result *EnrichmentResult) {
	// If we already have a detailed status, validate it
	if act.DetailedStatus != "" && act.DetailedStatus != "unknown" {
		return
	}

	// Determine status based on available data
	status := es.inferStatusFromData(act)
	act.DetailedStatus = status

	// Add confidence information
	if status == "unknown" {
		result.ConfidenceLevel = "low"
		result.Warnings = append(result.Warnings, "Could not determine detailed status")
	}
}

// inferStatusFromData infers status from available Act data
func (es *EnrichmentService) inferStatusFromData(act *sejm.EnhancedAct) string {
	// Check if Act is published
	if act.Status == "obowiązujący" || act.Status == "in_force" {
		return "in_force"
	}
	
	if act.Published != "" && act.Published != "null" {
		return "published"
	}

	// Check process stages for status clues
	if len(act.Stages) > 0 {
		return es.inferFromStages(act.Stages)
	}

	// Fallback to basic mapping
	return sejm.GetEnhancedStatus(act.Status)
}

// inferFromStages infers status from process stages
func (*EnrichmentService) inferFromStages(stages []sejm.ProcessStage) string {
	if len(stages) == 0 {
		return "unknown"
	}

	// Get the latest stage
	latestStage := stages[len(stages)-1]
	stageName := strings.ToLower(latestStage.StageName)

	// Check different stage types
	if status := inferSenateStatus(stageName); status != "" {
		return status
	}
	
	if status := inferSejmStatus(stageName); status != "" {
		return status
	}
	
	return inferOtherStatus(stageName)
}

// inferSenateStatus checks for Senate-related statuses
func inferSenateStatus(stageName string) string {
	if !strings.Contains(stageName, "senat") {
		return ""
	}
	
	if strings.Contains(stageName, "przyjęty") || strings.Contains(stageName, "zaakceptowany") {
		return "senate_accepted"
	}
	if strings.Contains(stageName, "odrzucony") {
		return "senate_rejected"
	}
	return "senate_review"
}

// inferSejmStatus checks for Sejm-related statuses
func inferSejmStatus(stageName string) string {
	if !strings.Contains(stageName, "sejm") {
		return ""
	}
	
	if strings.Contains(stageName, "przyjęty") || strings.Contains(stageName, "uchwalony") {
		return "passed_sejm"
	}
	if strings.Contains(stageName, "trzecie czytanie") {
		return "third_reading"
	}
	if strings.Contains(stageName, "drugie czytanie") {
		return "second_reading"
	}
	return ""
}

// inferOtherStatus checks for other status types
func inferOtherStatus(stageName string) string {
	if strings.Contains(stageName, "komisja") {
		return "committee_work"
	}
	if strings.Contains(stageName, "wpłynął") {
		return "submitted"
	}
	return "unknown"
}

// enrichMetadata extracts and enriches metadata from Act information
func (es *EnrichmentService) enrichMetadata(act *sejm.EnhancedAct, result *EnrichmentResult) {
	// Extract committee information from title or stages
	act.CommitteeCode = es.extractCommitteeCode(act)
	
	// Extract rapporteur information if available
	act.RapporteurName = es.extractRapporteur(act)
	
	// Generate RCL link if possible
	if act.RCLLink == "" {
		act.RCLLink = es.generateRCLLink(act)
	}

	result.DataSources = append(result.DataSources, "metadata_extraction")
}

// extractCommitteeCode extracts committee code from available information
func (es *EnrichmentService) extractCommitteeCode(act *sejm.EnhancedAct) string {
	// Check stages for committee information
	for _, stage := range act.Stages {
		if stage.CommitteeCode != "" {
			return stage.CommitteeCode
		}
	}

	// Extract from title using patterns
	return es.inferCommitteeFromTitle(act.Title)
}

// inferCommitteeFromTitle infers committee from Act title
func (*EnrichmentService) inferCommitteeFromTitle(title string) string {
	titleLower := strings.ToLower(title)
	
	// Define committee patterns
	committees := map[string][]string{
		"GOS": {"gospodarki", "economy", "business"},
		"FIN": {"finansów", "finance", "budget", "tax"},
		"EDU": {"edukacji", "education", "nauki", "science"},
		"SOC": {"polityki społecznej", "social", "pracy", "work"},
		"ENV": {"środowiska", "environment", "climat"},
		"TRA": {"transportu", "transport", "infrastruktury"},
		"HEA": {"zdrowia", "health", "medical"},
		"AGR": {"rolnictwa", "agriculture", "farming"},
		"DEF": {"obrony", "defense", "military"},
		"FOR": {"spraw zagranicznych", "foreign"},
	}

	for code, keywords := range committees {
		for _, keyword := range keywords {
			if strings.Contains(titleLower, keyword) {
				return code
			}
		}
	}

	return ""
}

// extractRapporteur extracts rapporteur information
func (*EnrichmentService) extractRapporteur(act *sejm.EnhancedAct) string {
	// Check stages for rapporteur information
	for _, stage := range act.Stages {
		if stage.RapporteurName != "" {
			return stage.RapporteurName
		}
	}

	return ""
}

// generateRCLLink generates RCL (Legislative Process Portal) link
func (*EnrichmentService) generateRCLLink(act *sejm.EnhancedAct) string {
	if act.ProcessPrintNumber == "" {
		return ""
	}

	// Generate standard RCL link format
	return fmt.Sprintf("https://orka.sejm.gov.pl/proc10.nsf/ustawy/%s.htm", act.ProcessPrintNumber)
}

// generateTags generates relevant tags for the Act
func (es *EnrichmentService) generateTags(act *sejm.EnhancedAct, _ *EnrichmentResult) {
	var tags []string

	// Add urgency tag
	if act.UrgencyStatus == "urgent" {
		tags = append(tags, "urgent")
	}

	// Add EU compliance tag
	if act.EUCompliance {
		tags = append(tags, "eu-law")
	}

	// Add type-based tags
	switch strings.ToLower(act.Type) {
	case "ustawa":
		tags = append(tags, "act")
	case "rozporządzenie":
		tags = append(tags, "regulation")
	case "uchwała":
		tags = append(tags, "resolution")
	}

	// Add subject-based tags from title analysis
	tags = append(tags, es.extractSubjectTags(act.Title)...)

	// Add process-based tags
	if len(act.SejmVotes) > 0 {
		tags = append(tags, "voted-sejm")
	}
	if len(act.SenateVotes) > 0 {
		tags = append(tags, "voted-senate")
	}

	act.Tags = tags
}

// extractSubjectTags extracts subject-based tags from title
func (*EnrichmentService) extractSubjectTags(title string) []string {
	var tags []string
	titleLower := strings.ToLower(title)

	// Define subject patterns
	subjects := map[string][]string{
		"budget":      {"budżet", "budget", "financial"},
		"tax":         {"podatek", "tax", "vat", "pit"},
		"healthcare":  {"zdrowie", "health", "medical", "hospital"},
		"education":   {"edukacja", "education", "szkoła", "school"},
		"environment": {"środowisko", "environment", "climat", "energia"},
		"covid":       {"covid", "pandemic", "coronavirus"},
		"digital":     {"cyfrowy", "digital", "internet", "it"},
		"defense":     {"obrona", "defense", "wojsko", "military"},
		"economy":     {"gospodarka", "economy", "business", "przedsiębiorstwa"},
		"transport":   {"transport", "drogi", "roads", "koleje"},
	}

	for tag, keywords := range subjects {
		for _, keyword := range keywords {
			if strings.Contains(titleLower, keyword) {
				tags = append(tags, tag)
				break
			}
		}
	}

	// Extract year-specific tags
	if currentYear := time.Now().Year(); strings.Contains(title, fmt.Sprintf("%d", currentYear)) {
		tags = append(tags, fmt.Sprintf("year-%d", currentYear))
	}

	return tags
}

// calculateStageMetrics calculates various stage-related metrics
func (*EnrichmentService) calculateStageMetrics(act *sejm.EnhancedAct, _ *EnrichmentResult) {
	if len(act.Stages) == 0 {
		return
	}

	// Calculate total process duration
	firstStage := act.Stages[0]
	lastStage := act.Stages[len(act.Stages)-1]
	totalDays := int(lastStage.StageDate.Sub(firstStage.StageDate).Hours() / 24)

	// Update days in stage for current stage
	if !act.StageDate.IsZero() {
		act.DaysInStage = sejm.CalculateDaysInStage(act.StageDate)
	}

	// Add performance indicators as tags
	if totalDays > 365 {
		act.Tags = append(act.Tags, "long-process")
	} else if totalDays < 30 {
		act.Tags = append(act.Tags, "fast-track")
	}

	if act.DaysInStage > 90 {
		act.Tags = append(act.Tags, "stalled")
	}
}

// ValidateEnrichment validates the enrichment results
func (*EnrichmentService) ValidateEnrichment(result *EnrichmentResult) []string {
	var issues []string
	act := result.EnhancedAct

	// Check required fields
	issues = append(issues, validateRequiredFields(act)...)
	
	// Check data consistency
	issues = append(issues, validateDataConsistency(act)...)
	
	// Check for reasonable values
	issues = append(issues, validateReasonableValues(act)...)

	return issues
}

// validateRequiredFields checks that required fields are present
func validateRequiredFields(act *sejm.EnhancedAct) []string {
	var issues []string
	
	if act.DetailedStatus == "" || act.DetailedStatus == "unknown" {
		issues = append(issues, "Missing or unknown detailed status")
	}

	if act.CurrentStage == "" {
		issues = append(issues, "Missing current stage information")
	}
	
	return issues
}

// validateDataConsistency checks for data consistency issues
func validateDataConsistency(act *sejm.EnhancedAct) []string {
	var issues []string
	
	if act.Status == "obowiązujący" && act.DetailedStatus != "in_force" {
		issues = append(issues, "Status inconsistency: marked as in force but detailed status differs")
	}

	if len(act.Stages) > 0 && act.StageDate.IsZero() {
		issues = append(issues, "Stages available but no stage date set")
	}
	
	return issues
}

// validateReasonableValues checks that values are within reasonable ranges
func validateReasonableValues(act *sejm.EnhancedAct) []string {
	var issues []string
	
	if act.DaysInStage < 0 {
		issues = append(issues, "Negative days in stage")
	}

	if act.DaysInStage > 365*5 { // 5 years
		issues = append(issues, "Unreasonably long time in stage")
	}
	
	return issues
}

// GetEnrichmentCapabilities returns the capabilities of the enrichment service
func (*EnrichmentService) GetEnrichmentCapabilities() map[string]any {
	return map[string]any{
		"process_tracking":     true,
		"status_determination": true,
		"metadata_extraction":  true,
		"tag_generation":       true,
		"stage_metrics":        true,
		"voting_integration":   false, // Not implemented in this version
		"senate_linking":       false, // Not implemented in this version
		"supported_years":      []int{2023, 2024, 2025},
		"supported_terms":      []int{10},
	}
}