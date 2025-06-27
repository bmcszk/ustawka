package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"ustawka/sejm"
)

// ValidationLevel represents the severity of validation issues
type ValidationLevel string

const (
	ValidationLevelError   ValidationLevel = "error"   // Critical issues that must be fixed
	ValidationLevelWarning ValidationLevel = "warning" // Issues that should be reviewed
	ValidationLevelInfo    ValidationLevel = "info"    // Informational notices
)

// ValidationIssue represents a validation problem
type ValidationIssue struct {
	Level       ValidationLevel `json:"level"`
	Field       string          `json:"field"`
	Message     string          `json:"message"`
	Value       interface{}     `json:"value,omitempty"`
	Suggestion  string          `json:"suggestion,omitempty"`
	Code        string          `json:"code"`
}

// ValidationResult contains the results of validation
type ValidationResult struct {
	IsValid    bool               `json:"is_valid"`
	Issues     []ValidationIssue  `json:"issues"`
	Summary    ValidationSummary  `json:"summary"`
	ValidatedAt time.Time         `json:"validated_at"`
}

// ValidationSummary provides a summary of validation results
type ValidationSummary struct {
	TotalIssues   int `json:"total_issues"`
	ErrorCount    int `json:"error_count"`
	WarningCount  int `json:"warning_count"`
	InfoCount     int `json:"info_count"`
	PassedChecks  int `json:"passed_checks"`
	TotalChecks   int `json:"total_checks"`
}

// ValidationConfig contains configuration for validation rules
type ValidationConfig struct {
	// Strictness levels
	EnableStrictValidation  bool
	RequireAllFields       bool
	ValidateReferences     bool
	CheckDataConsistency   bool
	
	// Performance settings
	MaxValidationTime      time.Duration
	EnableAsyncValidation  bool
	BatchValidationSize    int
	
	// Rule configuration
	EnabledRules          []string
	DisabledRules         []string
	CustomRules           map[string]ValidationRule
}

// ValidationRule defines a validation rule
type ValidationRule interface {
	GetName() string
	GetDescription() string
	Validate(ctx context.Context, act *sejm.EnhancedAct) []ValidationIssue
}

// DataValidationService provides comprehensive data validation
type DataValidationService struct {
	config *ValidationConfig
	rules  []ValidationRule
}

// NewDataValidationService creates a new validation service
func NewDataValidationService(config *ValidationConfig) *DataValidationService {
	if config == nil {
		config = DefaultValidationConfig()
	}
	
	service := &DataValidationService{
		config: config,
		rules:  make([]ValidationRule, 0),
	}
	
	// Register default validation rules
	service.registerDefaultRules()
	
	return service
}

// DefaultValidationConfig returns a sensible default configuration
func DefaultValidationConfig() *ValidationConfig {
	return &ValidationConfig{
		EnableStrictValidation: false,
		RequireAllFields:      false,
		ValidateReferences:    true,
		CheckDataConsistency:  true,
		
		MaxValidationTime:     30 * time.Second,
		EnableAsyncValidation: false,
		BatchValidationSize:   25,
		
		EnabledRules: []string{
			"basic_fields",
			"status_consistency",
			"date_validation",
			"numeric_ranges",
			"text_quality",
			"reference_integrity",
		},
		DisabledRules: []string{},
		CustomRules:   make(map[string]ValidationRule),
	}
}

// ValidateAct validates a single enhanced act
func (dvs *DataValidationService) ValidateAct(ctx context.Context, act *sejm.EnhancedAct) *ValidationResult {
	startTime := time.Now()
	
	var allIssues []ValidationIssue
	totalChecks := 0
	passedChecks := 0
	
	// Run all enabled validation rules
	for _, rule := range dvs.rules {
		if !dvs.isRuleEnabled(rule.GetName()) {
			continue
		}
		
		totalChecks++
		issues := rule.Validate(ctx, act)
		
		if len(issues) == 0 {
			passedChecks++
		} else {
			allIssues = append(allIssues, issues...)
		}
		
		// Check timeout
		if time.Since(startTime) > dvs.config.MaxValidationTime {
			allIssues = append(allIssues, ValidationIssue{
				Level:   ValidationLevelWarning,
				Field:   "validation",
				Message: "Validation timeout exceeded",
				Code:    "VALIDATION_TIMEOUT",
			})
			break
		}
	}
	
	// Calculate summary
	summary := dvs.calculateSummary(allIssues, totalChecks, passedChecks)
	
	return &ValidationResult{
		IsValid:     summary.ErrorCount == 0,
		Issues:      allIssues,
		Summary:     summary,
		ValidatedAt: time.Now(),
	}
}

// ValidateActBatch validates multiple acts in a batch
func (dvs *DataValidationService) ValidateActBatch(ctx context.Context, acts []sejm.EnhancedAct) map[string]*ValidationResult {
	results := make(map[string]*ValidationResult)
	
	// Process in configurable batch sizes
	batchSize := dvs.config.BatchValidationSize
	if batchSize <= 0 {
		batchSize = len(acts)
	}
	
	for i := 0; i < len(acts); i += batchSize {
		end := i + batchSize
		if end > len(acts) {
			end = len(acts)
		}
		
		batch := acts[i:end]
		for _, act := range batch {
			results[act.ID] = dvs.ValidateAct(ctx, &act)
		}
	}
	
	return results
}

// registerDefaultRules registers the built-in validation rules
func (dvs *DataValidationService) registerDefaultRules() {
	dvs.rules = append(dvs.rules,
		NewBasicFieldsRule(),
		NewStatusConsistencyRule(),
		NewDateValidationRule(),
		NewNumericRangesRule(),
		NewTextQualityRule(),
		NewReferenceIntegrityRule(),
	)
}

// isRuleEnabled checks if a validation rule is enabled
func (dvs *DataValidationService) isRuleEnabled(ruleName string) bool {
	// Check if explicitly disabled
	for _, disabled := range dvs.config.DisabledRules {
		if disabled == ruleName {
			return false
		}
	}
	
	// Check if explicitly enabled (if EnabledRules is specified)
	if len(dvs.config.EnabledRules) > 0 {
		for _, enabled := range dvs.config.EnabledRules {
			if enabled == ruleName {
				return true
			}
		}
		return false
	}
	
	return true
}

// calculateSummary calculates validation summary statistics
func (dvs *DataValidationService) calculateSummary(issues []ValidationIssue, totalChecks, passedChecks int) ValidationSummary {
	summary := ValidationSummary{
		TotalIssues:  len(issues),
		TotalChecks:  totalChecks,
		PassedChecks: passedChecks,
	}
	
	for _, issue := range issues {
		switch issue.Level {
		case ValidationLevelError:
			summary.ErrorCount++
		case ValidationLevelWarning:
			summary.WarningCount++
		case ValidationLevelInfo:
			summary.InfoCount++
		}
	}
	
	return summary
}

// Basic Fields Validation Rule
type BasicFieldsRule struct{}

func NewBasicFieldsRule() *BasicFieldsRule {
	return &BasicFieldsRule{}
}

func (r *BasicFieldsRule) GetName() string {
	return "basic_fields"
}

func (r *BasicFieldsRule) GetDescription() string {
	return "Validates that required basic fields are present and properly formatted"
}

func (r *BasicFieldsRule) Validate(_ context.Context, act *sejm.EnhancedAct) []ValidationIssue {
	var issues []ValidationIssue
	
	// Validate ID format
	if act.ID == "" {
		issues = append(issues, ValidationIssue{
			Level:   ValidationLevelError,
			Field:   "ID",
			Message: "Act ID is required",
			Code:    "MISSING_ID",
		})
	} else if !regexp.MustCompile(`^DU/\d{4}/\d+$`).MatchString(act.ID) {
		issues = append(issues, ValidationIssue{
			Level:      ValidationLevelWarning,
			Field:      "ID",
			Message:    "Act ID format may be invalid",
			Value:      act.ID,
			Suggestion: "Expected format: DU/YYYY/nnnn",
			Code:       "INVALID_ID_FORMAT",
		})
	}
	
	// Validate title
	if act.Title == "" {
		issues = append(issues, ValidationIssue{
			Level:   ValidationLevelError,
			Field:   "Title",
			Message: "Act title is required",
			Code:    "MISSING_TITLE",
		})
	} else if len(act.Title) < 10 {
		issues = append(issues, ValidationIssue{
			Level:      ValidationLevelWarning,
			Field:      "Title",
			Message:    "Act title seems too short",
			Value:      len(act.Title),
			Suggestion: "Consider checking if title is complete",
			Code:       "SHORT_TITLE",
		})
	}
	
	// Validate year
	currentYear := time.Now().Year()
	if act.Year < 1989 || act.Year > currentYear+1 {
		issues = append(issues, ValidationIssue{
			Level:      ValidationLevelWarning,
			Field:      "Year",
			Message:    "Act year seems outside reasonable range",
			Value:      act.Year,
			Suggestion: fmt.Sprintf("Expected year between 1989 and %d", currentYear+1),
			Code:       "INVALID_YEAR",
		})
	}
	
	// Validate position
	if act.Position <= 0 {
		issues = append(issues, ValidationIssue{
			Level:   ValidationLevelError,
			Field:   "Position",
			Message: "Act position must be a positive integer",
			Value:   act.Position,
			Code:    "INVALID_POSITION",
		})
	}
	
	return issues
}

// Status Consistency Validation Rule
type StatusConsistencyRule struct{}

func NewStatusConsistencyRule() *StatusConsistencyRule {
	return &StatusConsistencyRule{}
}

func (r *StatusConsistencyRule) GetName() string {
	return "status_consistency"
}

func (r *StatusConsistencyRule) GetDescription() string {
	return "Validates consistency between basic status and detailed status"
}

func (r *StatusConsistencyRule) Validate(_ context.Context, act *sejm.EnhancedAct) []ValidationIssue {
	var issues []ValidationIssue
	
	// Check status consistency
	if act.Status == "obowiązujący" && !strings.Contains(act.DetailedStatus, "force") {
		if act.DetailedStatus != "in_force" && act.DetailedStatus != "published" {
			issues = append(issues, ValidationIssue{
				Level:      ValidationLevelWarning,
				Field:      "DetailedStatus",
				Message:    "Detailed status inconsistent with basic status",
				Value:      fmt.Sprintf("Basic: %s, Detailed: %s", act.Status, act.DetailedStatus),
				Suggestion: "Check if detailed status should be 'in_force' or 'published'",
				Code:       "STATUS_INCONSISTENCY",
			})
		}
	}
	
	// Validate current stage matches detailed status
	if act.CurrentStage != "" && act.DetailedStatus != "" {
		if !r.validateStageStatusMatch(act, &issues) {
			// Stage mismatch handled by helper function
		}
	}
	
	return issues
}

// validateStageStatusMatch is a helper to validate stage-status consistency
func (r *StatusConsistencyRule) validateStageStatusMatch(act *sejm.EnhancedAct, issues *[]ValidationIssue) bool {
	expectedStages := map[string][]string{
		"submitted":      {"Wpłynął", "Submitted"},
		"committee_work": {"Komisja", "Committee"},
		"senate_review":  {"Senat", "Senate"},
		"in_force":       {"Weszła w życie", "In Force", "Opublikowano"},
	}
	
	stages, exists := expectedStages[act.DetailedStatus]
	if !exists {
		return true
	}
	
	for _, expectedStage := range stages {
		if strings.Contains(act.CurrentStage, expectedStage) {
			return true
		}
	}
	
	*issues = append(*issues, ValidationIssue{
		Level:   ValidationLevelInfo,
		Field:   "CurrentStage",
		Message: "Current stage may not match detailed status",
		Value:   fmt.Sprintf("Stage: %s, Status: %s", act.CurrentStage, act.DetailedStatus),
		Code:    "STAGE_STATUS_MISMATCH",
	})
	return false
}

// Date Validation Rule
type DateValidationRule struct{}

func NewDateValidationRule() *DateValidationRule {
	return &DateValidationRule{}
}

func (r *DateValidationRule) GetName() string {
	return "date_validation"
}

func (r *DateValidationRule) GetDescription() string {
	return "Validates date fields for logical consistency and reasonable ranges"
}

func (r *DateValidationRule) Validate(_ context.Context, act *sejm.EnhancedAct) []ValidationIssue {
	var issues []ValidationIssue
	
	now := time.Now()
	
	// Validate stage date
	if !act.StageDate.IsZero() {
		if act.StageDate.After(now) {
			issues = append(issues, ValidationIssue{
				Level:      ValidationLevelWarning,
				Field:      "StageDate",
				Message:    "Stage date is in the future",
				Value:      act.StageDate.Format("2006-01-02"),
				Suggestion: "Check if stage date is correct",
				Code:       "FUTURE_STAGE_DATE",
			})
		}
		
		if act.StageDate.Year() < 1989 {
			issues = append(issues, ValidationIssue{
				Level:      ValidationLevelWarning,
				Field:      "StageDate",
				Message:    "Stage date seems too old",
				Value:      act.StageDate.Format("2006-01-02"),
				Code:       "OLD_STAGE_DATE",
			})
		}
	}
	
	// Validate days in stage
	if act.DaysInStage < 0 {
		issues = append(issues, ValidationIssue{
			Level:      ValidationLevelError,
			Field:      "DaysInStage",
			Message:    "Days in stage cannot be negative",
			Value:      act.DaysInStage,
			Code:       "NEGATIVE_DAYS",
		})
	} else if act.DaysInStage > 365*5 { // 5 years
		issues = append(issues, ValidationIssue{
			Level:      ValidationLevelWarning,
			Field:      "DaysInStage",
			Message:    "Days in stage seems unusually high",
			Value:      act.DaysInStage,
			Suggestion: "Consider if this act has been stalled",
			Code:       "EXCESSIVE_DAYS",
		})
	}
	
	return issues
}

// Numeric Ranges Validation Rule
type NumericRangesRule struct{}

func NewNumericRangesRule() *NumericRangesRule {
	return &NumericRangesRule{}
}

func (r *NumericRangesRule) GetName() string {
	return "numeric_ranges"
}

func (r *NumericRangesRule) GetDescription() string {
	return "Validates numeric fields are within reasonable ranges"
}

func (r *NumericRangesRule) Validate(_ context.Context, act *sejm.EnhancedAct) []ValidationIssue {
	var issues []ValidationIssue
	
	// Validate voting counts
	for i, vote := range act.SejmVotes {
		totalVotes := vote.YesVotes + vote.NoVotes + vote.AbstainVotes + vote.AbsentVotes
		if vote.TotalVoted > 0 && totalVotes != vote.TotalVoted {
			issues = append(issues, ValidationIssue{
				Level:      ValidationLevelWarning,
				Field:      fmt.Sprintf("SejmVotes[%d].TotalVoted", i),
				Message:    "Vote totals don't match individual counts",
				Value:      fmt.Sprintf("Reported: %d, Calculated: %d", vote.TotalVoted, totalVotes),
				Code:       "VOTE_COUNT_MISMATCH",
			})
		}
		
		// Check for reasonable vote counts (Sejm has 460 members)
		if totalVotes > 500 {
			issues = append(issues, ValidationIssue{
				Level:      ValidationLevelWarning,
				Field:      fmt.Sprintf("SejmVotes[%d]", i),
				Message:    "Vote count exceeds expected maximum",
				Value:      totalVotes,
				Suggestion: "Sejm has 460 members, vote count seems high",
				Code:       "EXCESSIVE_VOTE_COUNT",
			})
		}
	}
	
	return issues
}

// Text Quality Validation Rule
type TextQualityRule struct{}

func NewTextQualityRule() *TextQualityRule {
	return &TextQualityRule{}
}

func (r *TextQualityRule) GetName() string {
	return "text_quality"
}

func (r *TextQualityRule) GetDescription() string {
	return "Validates text fields for quality and completeness"
}

func (r *TextQualityRule) Validate(_ context.Context, act *sejm.EnhancedAct) []ValidationIssue {
	var issues []ValidationIssue
	
	// Check for placeholder or incomplete text
	placeholderTexts := []string{"TODO", "TBD", "...", "???", "N/A", "null", "undefined"}
	
	checkTextQuality := func(fieldName, text string) {
		for _, placeholder := range placeholderTexts {
			if strings.Contains(strings.ToLower(text), strings.ToLower(placeholder)) {
				issues = append(issues, ValidationIssue{
					Level:      ValidationLevelWarning,
					Field:      fieldName,
					Message:    "Field contains placeholder text",
					Value:      text,
					Suggestion: "Replace placeholder with actual content",
					Code:       "PLACEHOLDER_TEXT",
				})
				break
			}
		}
		
		// Check for excessive whitespace
		if strings.TrimSpace(text) != text {
			issues = append(issues, ValidationIssue{
				Level:      ValidationLevelInfo,
				Field:      fieldName,
				Message:    "Field has leading/trailing whitespace",
				Suggestion: "Trim whitespace from field",
				Code:       "WHITESPACE_ISSUES",
			})
		}
	}
	
	checkTextQuality("Title", act.Title)
	checkTextQuality("CurrentStage", act.CurrentStage)
	checkTextQuality("InitiatorType", act.InitiatorType)
	checkTextQuality("CommitteeCode", act.CommitteeCode)
	
	return issues
}

// Reference Integrity Validation Rule
type ReferenceIntegrityRule struct{}

func NewReferenceIntegrityRule() *ReferenceIntegrityRule {
	return &ReferenceIntegrityRule{}
}

func (r *ReferenceIntegrityRule) GetName() string {
	return "reference_integrity"
}

func (r *ReferenceIntegrityRule) GetDescription() string {
	return "Validates links and references for accessibility and format"
}

func (r *ReferenceIntegrityRule) Validate(_ context.Context, act *sejm.EnhancedAct) []ValidationIssue {
	var issues []ValidationIssue
	
	// Validate RCL link format
	if act.RCLLink != "" {
		if !strings.HasPrefix(act.RCLLink, "http") {
			issues = append(issues, ValidationIssue{
				Level:      ValidationLevelWarning,
				Field:      "RCLLink",
				Message:    "RCL link should be a valid URL",
				Value:      act.RCLLink,
				Suggestion: "Ensure link starts with http:// or https://",
				Code:       "INVALID_URL_FORMAT",
			})
		}
	}
	
	// Validate process print number format
	if act.ProcessPrintNumber != "" {
		if !regexp.MustCompile(`^\d+$`).MatchString(act.ProcessPrintNumber) {
			issues = append(issues, ValidationIssue{
				Level:      ValidationLevelWarning,
				Field:      "ProcessPrintNumber",
				Message:    "Process print number should be numeric",
				Value:      act.ProcessPrintNumber,
				Code:       "INVALID_PRINT_NUMBER",
			})
		}
	}
	
	// Validate address format
	if act.Address != "" {
		if !strings.HasPrefix(act.Address, "http") && !strings.Contains(act.Address, ".pdf") {
			issues = append(issues, ValidationIssue{
				Level:      ValidationLevelInfo,
				Field:      "Address",
				Message:    "Address may not be a valid document link",
				Value:      act.Address,
				Code:       "QUESTIONABLE_ADDRESS",
			})
		}
	}
	
	return issues
}

// GetValidationStats returns validation statistics
func (dvs *DataValidationService) GetValidationStats() map[string]any {
	return map[string]any{
		"enabled_rules":        len(dvs.rules),
		"total_rules":          len(dvs.rules),
		"max_validation_time":  dvs.config.MaxValidationTime,
		"strict_validation":    dvs.config.EnableStrictValidation,
		"batch_size":          dvs.config.BatchValidationSize,
		"rules":               dvs.getRuleNames(),
	}
}

// getRuleNames returns the names of all registered rules
func (dvs *DataValidationService) getRuleNames() []string {
	names := make([]string, len(dvs.rules))
	for i, rule := range dvs.rules {
		names[i] = rule.GetName()
	}
	return names
}