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
type ValidationLevel = string

// Validation severity levels
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
	Value       any             `json:"value,omitempty"`
	Suggestion  string          `json:"suggestion,omitempty"`
	Code        string          `json:"code"`
}

// ValidationResult contains the results of validation
type ValidationResult struct {
	IsValid    bool               `json:"is_valid"`
	Issues     []ValidationIssue  `json:"issues"`
	Summary    validationSummary  `json:"summary"`
	ValidatedAt time.Time         `json:"validated_at"`
}

// validationSummary provides a summary of validation results
type validationSummary struct {
	TotalIssues   int `json:"total_issues"`
	ErrorCount    int `json:"error_count"`
	WarningCount  int `json:"warning_count"`
	InfoCount     int `json:"info_count"`
	PassedChecks  int `json:"passed_checks"`
	TotalChecks   int `json:"total_checks"`
}

// validationConfig contains configuration for validation rules
type validationConfig struct {
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
	config *validationConfig
	rules  []ValidationRule
}

// NewDataValidationService creates a new validation service with default config
func NewDataValidationService() *DataValidationService {
	return NewDataValidationServiceWithConfig(nil)
}

// NewDataValidationServiceWithConfig creates a new validation service with custom config
func NewDataValidationServiceWithConfig(config *validationConfig) *DataValidationService {
	if config == nil {
		config = createDefaultValidationConfig()
	}
	
	service := &DataValidationService{
		config: config,
		rules:  make([]ValidationRule, 0),
	}
	
	// Register default validation rules
	service.registerDefaultRules()
	
	return service
}

// DefaultValidationConfig returns a sensible default configuration (for external access)
func DefaultValidationConfig() map[string]any {
	config := createDefaultValidationConfig()
	return map[string]any{
		"enable_strict_validation": config.EnableStrictValidation,
		"require_all_fields":      config.RequireAllFields,
		"validate_references":     config.ValidateReferences,
		"check_data_consistency":  config.CheckDataConsistency,
		"max_validation_time":     config.MaxValidationTime,
		"enable_async_validation": config.EnableAsyncValidation,
		"batch_validation_size":   config.BatchValidationSize,
		"enabled_rules":           config.EnabledRules,
		"disabled_rules":          config.DisabledRules,
	}
}

// createDefaultValidationConfig returns a sensible default configuration
func createDefaultValidationConfig() *validationConfig {
	return &validationConfig{
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
func (dvs *DataValidationService) ValidateActBatch(
	ctx context.Context, 
	acts []sejm.EnhancedAct,
) map[string]*ValidationResult {
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
	if dvs.isRuleDisabled(ruleName) {
		return false
	}
	
	return dvs.isRuleInEnabledList(ruleName)
}

func (dvs *DataValidationService) isRuleDisabled(ruleName string) bool {
	for _, disabled := range dvs.config.DisabledRules {
		if disabled == ruleName {
			return true
		}
	}
	return false
}

func (dvs *DataValidationService) isRuleInEnabledList(ruleName string) bool {
	if len(dvs.config.EnabledRules) == 0 {
		return true
	}
	
	for _, enabled := range dvs.config.EnabledRules {
		if enabled == ruleName {
			return true
		}
	}
	return false
}

// calculateSummary calculates validation summary statistics
func (*DataValidationService) calculateSummary(
	issues []ValidationIssue, 
	totalChecks, passedChecks int,
) validationSummary {
	summary := validationSummary{
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

// basicFieldsRule validates that required basic fields are present and properly formatted.
type basicFieldsRule struct{}

// NewBasicFieldsRule creates a new BasicFieldsRule validator.
func NewBasicFieldsRule() ValidationRule {
	return &basicFieldsRule{}
}

// GetName returns the name of the basic fields validation rule
func (*basicFieldsRule) GetName() string {
	return "basic_fields"
}

// GetDescription returns the description of the basic fields validation rule
func (*basicFieldsRule) GetDescription() string {
	return "Validates that required basic fields are present and properly formatted"
}

// Validate performs basic fields validation on an enhanced act
func (*basicFieldsRule) Validate(_ context.Context, act *sejm.EnhancedAct) []ValidationIssue {
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

// statusConsistencyRule validates consistency between basic status and detailed status.
type statusConsistencyRule struct{}

// NewStatusConsistencyRule creates a new StatusConsistencyRule validator.
func NewStatusConsistencyRule() ValidationRule {
	return &statusConsistencyRule{}
}

// GetName returns the name of the status consistency validation rule
func (*statusConsistencyRule) GetName() string {
	return "status_consistency"
}

// GetDescription returns the description of the status consistency validation rule
func (*statusConsistencyRule) GetDescription() string {
	return "Validates consistency between basic status and detailed status"
}

// Validate performs status consistency validation on an enhanced act
func (r *statusConsistencyRule) Validate(_ context.Context, act *sejm.EnhancedAct) []ValidationIssue {
	var issues []ValidationIssue
	
	r.checkBasicStatusConsistency(act, &issues)
	r.checkStageStatusConsistency(act, &issues)
	
	return issues
}

func (*statusConsistencyRule) checkBasicStatusConsistency(act *sejm.EnhancedAct, issues *[]ValidationIssue) {
	if act.Status == "obowiązujący" && !strings.Contains(act.DetailedStatus, "force") {
		if act.DetailedStatus != "in_force" && act.DetailedStatus != "published" {
			*issues = append(*issues, ValidationIssue{
				Level:      ValidationLevelWarning,
				Field:      "DetailedStatus",
				Message:    "Detailed status inconsistent with basic status",
				Value:      fmt.Sprintf("Basic: %s, Detailed: %s", act.Status, act.DetailedStatus),
				Suggestion: "Check if detailed status should be 'in_force' or 'published'",
				Code:       "STATUS_INCONSISTENCY",
			})
		}
	}
}

func (r *statusConsistencyRule) checkStageStatusConsistency(act *sejm.EnhancedAct, issues *[]ValidationIssue) {
	if act.CurrentStage != "" && act.DetailedStatus != "" {
		r.validateStageStatusMatch(act, issues)
	}
}

// validateStageStatusMatch is a helper to validate stage-status consistency
func (*statusConsistencyRule) validateStageStatusMatch(act *sejm.EnhancedAct, issues *[]ValidationIssue) bool {
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

// dateValidationRule validates date fields for logical consistency and reasonable ranges
type dateValidationRule struct{}

// NewDateValidationRule creates a new DateValidationRule validator
func NewDateValidationRule() ValidationRule {
	return &dateValidationRule{}
}

// GetName returns the name of the date validation rule
func (*dateValidationRule) GetName() string {
	return "date_validation"
}

// GetDescription returns the description of the date validation rule
func (*dateValidationRule) GetDescription() string {
	return "Validates date fields for logical consistency and reasonable ranges"
}

// Validate performs date validation on an enhanced act
func (*dateValidationRule) Validate(_ context.Context, act *sejm.EnhancedAct) []ValidationIssue {
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

// numericRangesRule validates numeric fields are within reasonable ranges
type numericRangesRule struct{}

// NewNumericRangesRule creates a new NumericRangesRule validator
func NewNumericRangesRule() ValidationRule {
	return &numericRangesRule{}
}

// GetName returns the name of the numeric ranges validation rule
func (*numericRangesRule) GetName() string {
	return "numeric_ranges"
}

// GetDescription returns the description of the numeric ranges validation rule
func (*numericRangesRule) GetDescription() string {
	return "Validates numeric fields are within reasonable ranges"
}

// Validate performs numeric ranges validation on an enhanced act
func (*numericRangesRule) Validate(_ context.Context, act *sejm.EnhancedAct) []ValidationIssue {
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

// textQualityRule validates text fields for quality and completeness
type textQualityRule struct{}

// NewTextQualityRule creates a new TextQualityRule validator
func NewTextQualityRule() ValidationRule {
	return &textQualityRule{}
}

// GetName returns the name of the text quality validation rule
func (*textQualityRule) GetName() string {
	return "text_quality"
}

// GetDescription returns the description of the text quality validation rule
func (*textQualityRule) GetDescription() string {
	return "Validates text fields for quality and completeness"
}

// Validate performs text quality validation on an enhanced act
func (*textQualityRule) Validate(_ context.Context, act *sejm.EnhancedAct) []ValidationIssue {
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

// referenceIntegrityRule validates links and references for accessibility and format
type referenceIntegrityRule struct{}

// NewReferenceIntegrityRule creates a new ReferenceIntegrityRule validator
func NewReferenceIntegrityRule() ValidationRule {
	return &referenceIntegrityRule{}
}

// GetName returns the name of the reference integrity validation rule
func (*referenceIntegrityRule) GetName() string {
	return "reference_integrity"
}

// GetDescription returns the description of the reference integrity validation rule
func (*referenceIntegrityRule) GetDescription() string {
	return "Validates links and references for accessibility and format"
}

// Validate performs reference integrity validation on an enhanced act
func (r *referenceIntegrityRule) Validate(_ context.Context, act *sejm.EnhancedAct) []ValidationIssue {
	var issues []ValidationIssue
	
	r.validateRCLLink(act, &issues)
	r.validateProcessPrintNumber(act, &issues)
	r.validateAddress(act, &issues)
	
	return issues
}

func (*referenceIntegrityRule) validateRCLLink(act *sejm.EnhancedAct, issues *[]ValidationIssue) {
	if act.RCLLink != "" && !strings.HasPrefix(act.RCLLink, "http") {
		*issues = append(*issues, ValidationIssue{
			Level:      ValidationLevelWarning,
			Field:      "RCLLink",
			Message:    "RCL link should be a valid URL",
			Value:      act.RCLLink,
			Suggestion: "Ensure link starts with http:// or https://",
			Code:       "INVALID_URL_FORMAT",
		})
	}
}

func (*referenceIntegrityRule) validateProcessPrintNumber(act *sejm.EnhancedAct, issues *[]ValidationIssue) {
	if act.ProcessPrintNumber != "" && !regexp.MustCompile(`^\d+$`).MatchString(act.ProcessPrintNumber) {
		*issues = append(*issues, ValidationIssue{
			Level:      ValidationLevelWarning,
			Field:      "ProcessPrintNumber",
			Message:    "Process print number should be numeric",
			Value:      act.ProcessPrintNumber,
			Code:       "INVALID_PRINT_NUMBER",
		})
	}
}

func (*referenceIntegrityRule) validateAddress(act *sejm.EnhancedAct, issues *[]ValidationIssue) {
	if act.Address != "" && !strings.HasPrefix(act.Address, "http") && !strings.Contains(act.Address, ".pdf") {
		*issues = append(*issues, ValidationIssue{
			Level:      ValidationLevelInfo,
			Field:      "Address",
			Message:    "Address may not be a valid document link",
			Value:      act.Address,
			Code:       "QUESTIONABLE_ADDRESS",
		})
	}
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