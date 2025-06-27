package service_test

import (
	"context"
	"testing"
	"time"
	"ustawka/sejm"
	"ustawka/service"

	"github.com/stretchr/testify/assert"
)

func TestNewDataValidationService(t *testing.T) {
	validationService := service.NewDataValidationService(nil)
	assert.NotNil(t, validationService)
	
	config := &service.ValidationConfig{
		EnableStrictValidation: true,
		RequireAllFields:      true,
		MaxValidationTime:     10 * time.Second,
	}
	validationService2 := service.NewDataValidationService(config)
	assert.NotNil(t, validationService2)
}

func TestDefaultValidationConfig(t *testing.T) {
	config := service.DefaultValidationConfig()
	
	assert.NotNil(t, config)
	assert.False(t, config.EnableStrictValidation)
	assert.True(t, config.ValidateReferences)
	assert.Contains(t, config.EnabledRules, "basic_fields")
}

func TestValidateActValid(t *testing.T) {
	validationService := service.NewDataValidationService(nil)
	ctx := context.Background()
	
	act := &sejm.EnhancedAct{
		ID:       "DU/2024/1",
		Title:    "Test Act Title",
		Year:     2024,
		Position: 1,
		Status:   "obowiązujący",
	}
	
	result := validationService.ValidateAct(ctx, act)
	assert.True(t, result.IsValid)
	assert.Equal(t, 0, result.Summary.ErrorCount)
}

func TestValidateActInvalid(t *testing.T) {
	validationService := service.NewDataValidationService(nil)
	ctx := context.Background()
	
	act := &sejm.EnhancedAct{
		ID:       "", // Missing ID
		Title:    "",  // Missing title
		Year:     2024,
		Position: 0, // Invalid position
	}
	
	result := validationService.ValidateAct(ctx, act)
	assert.False(t, result.IsValid)
	assert.Greater(t, result.Summary.ErrorCount, 0)
}

func TestValidateActBatch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping batch validation test in short mode")
	}
	
	validationService := service.NewDataValidationService(nil)
	ctx := context.Background()
	
	acts := []sejm.EnhancedAct{
		{
			ID:       "DU/2024/1",
			Title:    "Valid Act 1",
			Year:     2024,
			Position: 1,
		},
		{
			ID:       "", // Invalid act
			Title:    "Invalid Act",
			Year:     2024,
			Position: 3,
		},
	}
	
	results := validationService.ValidateActBatch(ctx, acts)
	
	assert.Equal(t, 2, len(results))
	assert.True(t, results["DU/2024/1"].IsValid)
	assert.False(t, results[""].IsValid)
}

func TestValidationServiceGetStats(t *testing.T) {
	validationService := service.NewDataValidationService(nil)
	
	stats := validationService.GetValidationStats()
	
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "enabled_rules")
	assert.Contains(t, stats, "rules")
	
	rules, ok := stats["rules"].([]string)
	assert.True(t, ok)
	assert.Contains(t, rules, "basic_fields")
}

func BenchmarkValidateAct(b *testing.B) {
	validationService := service.NewDataValidationService(nil)
	ctx := context.Background()
	
	act := &sejm.EnhancedAct{
		ID:       "DU/2024/1",
		Title:    "Benchmark Test Act",
		Year:     2024,
		Position: 1,
		Status:   "obowiązujący",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := validationService.ValidateAct(ctx, act)
		_ = result
	}
}