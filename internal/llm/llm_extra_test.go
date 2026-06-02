package llm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/config"
)

func TestDisabledProvider_GenerateDecision_ReturnsError_Extra(t *testing.T) {
	p := NewDisabledProvider()
	result, err := p.GenerateDecision(context.Background(), LLMSafeContext{})
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM is disabled")
	assert.Contains(t, err.Error(), "LLM_ENABLED=false")
}

func TestNewDisabledProvider_Extra(t *testing.T) {
	p := NewDisabledProvider()
	assert.NotNil(t, p)
}

func TestDisabledProvider_ImplementsDecisionProvider_Extra(t *testing.T) {
	var _ DecisionProvider = NewDisabledProvider()
}

func TestLLMSafeContextEnvelope_Fields_Extra(t *testing.T) {
	env := &LLMSafeContextEnvelope{
		SchemaVersion: "v1",
		CaseID:        "case-001",
		AlertID:       "alert-001",
		ContextHash:   "abc123",
		PromptVersion: "v1.0",
		Evidence: []EvidenceItem{
			{Type: "metric", Key: "gmv", Value: 1000000.0},
			{Type: "alert", Key: "rule", Value: "gmv_drop"},
		},
		AllowedActions:   []string{ActionTypeNotifyOwner},
		ForbiddenActions: []string{ActionTypeCreateOutboxMessage},
		Governance:       GovernanceInfo{Classification: "L2", Role: "analyst"},
		RedactionSummary: RedactionSummary{
			TotalFields:   10,
			RedactedCount: 2,
			RedactedList:  []string{"email", "phone"},
			AppliedRole:   "analyst",
		},
		ConfigVersions: map[string]string{
			"alert_rules": "v1.0",
			"metrics":     "v2.0",
		},
	}

	assert.Equal(t, "v1", env.SchemaVersion)
	assert.Equal(t, "case-001", env.CaseID)
	assert.Equal(t, "alert-001", env.AlertID)
	assert.Equal(t, "abc123", env.ContextHash)
	assert.Len(t, env.Evidence, 2)
	assert.Equal(t, "metric", env.Evidence[0].Type)
	assert.Equal(t, 1000000.0, env.Evidence[0].Value)
	assert.Len(t, env.AllowedActions, 1)
	assert.Len(t, env.ForbiddenActions, 1)
	assert.Equal(t, "L2", env.Governance.Classification)
	assert.Equal(t, 10, env.RedactionSummary.TotalFields)
	assert.Equal(t, 2, env.RedactionSummary.RedactedCount)
	assert.Len(t, env.RedactionSummary.RedactedList, 2)
	assert.Len(t, env.ConfigVersions, 2)
}

func TestEvidenceItem_Fields_Extra(t *testing.T) {
	item := EvidenceItem{
		Type:  "metric",
		Key:   "gmv",
		Value: 1000000.0,
	}
	assert.Equal(t, "metric", item.Type)
	assert.Equal(t, "gmv", item.Key)
	assert.Equal(t, 1000000.0, item.Value)
}

func TestRedactionSummary_Fields_Extra(t *testing.T) {
	summary := RedactionSummary{
		TotalFields:   20,
		RedactedCount: 5,
		RedactedList:  []string{"email", "phone", "address", "ssn", "name"},
		AppliedRole:   "viewer",
	}
	assert.Equal(t, 20, summary.TotalFields)
	assert.Equal(t, 5, summary.RedactedCount)
	assert.Len(t, summary.RedactedList, 5)
	assert.Equal(t, "viewer", summary.AppliedRole)
}

func TestRuleBasedProvider_AllSeverities_Extra(t *testing.T) {
	p := NewRuleBasedProvider()
	severities := []string{SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow, "unknown"}
	for _, sev := range severities {
		t.Run(sev, func(t *testing.T) {
			input := LLMSafeContext{
				Trigger: TriggerInfo{
					Severity:      sev,
					MetricName:    "test_metric",
					CurrentValue:  100.0,
					BaselineValue: 200.0,
					DeltaPct:      -50.0,
				},
			}
			output, err := p.GenerateDecision(context.Background(), input)
			require.NoError(t, err)
			require.NotNil(t, output)
			assert.Equal(t, sev, output.Severity)
			assert.True(t, output.RequiresHumanReview)
			assert.NotEmpty(t, output.DecisionType)
			assert.NotEmpty(t, output.Summary)
			assert.NotEmpty(t, output.Rationale)
			assert.NotEmpty(t, output.RecommendedActions)
			assert.GreaterOrEqual(t, output.Confidence, 0.0)
			assert.LessOrEqual(t, output.Confidence, 1.0)
		})
	}
}

func TestRuleBasedProvider_CriticalSeverity_Extra(t *testing.T) {
	p := NewRuleBasedProvider()
	input := LLMSafeContext{
		Trigger: TriggerInfo{
			Severity:      SeverityCritical,
			MetricName:    "fraud_rate",
			CurrentValue:  0.15,
			BaselineValue: 0.01,
			DeltaPct:      1400.0,
		},
	}
	output, err := p.GenerateDecision(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, SeverityCritical, output.Severity)
	assert.Equal(t, 0.95, output.Confidence)
	assert.Len(t, output.RecommendedActions, 2)
}

func TestRuleBasedProvider_HighSeverity_Extra(t *testing.T) {
	p := NewRuleBasedProvider()
	input := LLMSafeContext{
		Trigger: TriggerInfo{
			Severity:      SeverityHigh,
			MetricName:    "cancel_rate",
			CurrentValue:  0.10,
			BaselineValue: 0.02,
			DeltaPct:      400.0,
		},
	}
	output, err := p.GenerateDecision(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, SeverityHigh, output.Severity)
	assert.Equal(t, 0.85, output.Confidence)
	assert.Len(t, output.RecommendedActions, 2)
}

func TestRuleBasedProvider_MediumSeverity_Extra(t *testing.T) {
	p := NewRuleBasedProvider()
	input := LLMSafeContext{
		Trigger: TriggerInfo{
			Severity:      SeverityMedium,
			MetricName:    "delivery_delay",
			CurrentValue:  5.0,
			BaselineValue: 3.0,
			DeltaPct:      66.0,
		},
	}
	output, err := p.GenerateDecision(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, SeverityMedium, output.Severity)
	assert.Equal(t, 0.72, output.Confidence)
	assert.Equal(t, DecisionTypeInvestigate, output.DecisionType)
	assert.Len(t, output.RecommendedActions, 2)
}

func TestRuleBasedProvider_LowSeverity_Extra(t *testing.T) {
	p := NewRuleBasedProvider()
	input := LLMSafeContext{
		Trigger: TriggerInfo{
			Severity:      SeverityLow,
			MetricName:    "minor_anomaly",
			CurrentValue:  1.0,
			BaselineValue: 1.1,
			DeltaPct:      -9.0,
		},
	}
	output, err := p.GenerateDecision(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, SeverityLow, output.Severity)
	assert.Equal(t, 0.60, output.Confidence)
	assert.Equal(t, DecisionTypeMonitor, output.DecisionType)
	assert.Len(t, output.RecommendedActions, 1)
}

func TestRuleBasedProvider_DefaultSeverity_Extra(t *testing.T) {
	p := NewRuleBasedProvider()
	input := LLMSafeContext{
		Trigger: TriggerInfo{
			Severity:   "bogus",
			MetricName: "test",
		},
	}
	output, err := p.GenerateDecision(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, 0.50, output.Confidence)
	assert.Equal(t, DecisionTypeInvestigate, output.DecisionType)
	assert.Contains(t, output.Summary, "unknown severity")
}

func TestNoOpAuditLogger_AllMethods_Extra(t *testing.T) {
	logger := &NoOpAuditLogger{}
	ctx := context.Background()

	logger.LogDecisionRequested(ctx, "case-1", "openai", "gpt-4o")
	logger.LogDecisionCompleted(ctx, "case-1", "openai", "gpt-4o", 100, &TokenUsage{TotalTokens: 500})
	logger.LogDecisionCompleted(ctx, "case-1", "openai", "gpt-4o", 100, nil)
	logger.LogDecisionFailed(ctx, "case-1", "openai", "gpt-4o", assert.AnError)
	logger.LogDecisionFailed(ctx, "case-1", "openai", "gpt-4o", nil)
	logger.LogDecisionValidationFailed(ctx, "case-1", []ValidationError{{Field: "test", Message: "error"}})
	logger.LogDecisionValidationFailed(ctx, "case-1", nil)
	logger.LogFallbackUsed(ctx, "case-1", "reason")
	logger.LogDecisionReplayed(ctx, "case-1", "orig-decision-1")
	logger.LogEvalCompleted(ctx, "case-1", "eval-1")
}

func TestDBAuditLogger_NilPool_Extra(t *testing.T) {
	logger := NewDBAuditLogger(nil)
	ctx := context.Background()

	logger.LogDecisionRequested(ctx, "case-1", "openai", "gpt-4o")
	logger.LogDecisionCompleted(ctx, "case-1", "openai", "gpt-4o", 100, nil)
	logger.LogDecisionFailed(ctx, "case-1", "openai", "gpt-4o", assert.AnError)
	logger.LogDecisionValidationFailed(ctx, "case-1", nil)
	logger.LogFallbackUsed(ctx, "case-1", "test")
	logger.LogDecisionReplayed(ctx, "case-1", "orig-1")
	logger.LogEvalCompleted(ctx, "case-1", "eval-1")
}

func TestTokenUsage_Fields_Extra(t *testing.T) {
	usage := &TokenUsage{
		PromptTokens:     100,
		CompletionTokens: 200,
		TotalTokens:      300,
	}
	assert.Equal(t, 100, usage.PromptTokens)
	assert.Equal(t, 200, usage.CompletionTokens)
	assert.Equal(t, 300, usage.TotalTokens)
}

func TestValidateDecision_AllActionTypes_Extra(t *testing.T) {
	actionTypes := []string{
		ActionTypeCreateFollowupTask,
		ActionTypeNotifyOwner,
		ActionTypeExportReport,
		ActionTypeCreateOutboxMessage,
		ActionTypeEscalateToHuman,
	}
	for _, actionType := range actionTypes {
		t.Run(actionType, func(t *testing.T) {
			output := validDecisionOutput()
			output.RecommendedActions = []RecommendedAction{
				{ActionType: actionType},
			}
			result := ValidateDecision(output, []string{actionType})
			for _, e := range result.Errors {
				assert.NotContains(t, e.Field, "action_type")
			}
		})
	}
}

func TestValidateDecision_SchemaVersion_Extra(t *testing.T) {
	output := validDecisionOutput()
	output.SchemaVersion = "decision_output.v1"
	result := ValidateDecision(output, validAllowedActions())
	assert.True(t, result.Valid)
}

func TestValidateDecision_InvalidSchemaVersion_Extra(t *testing.T) {
	output := validDecisionOutput()
	output.SchemaVersion = "decision_output.v99"
	result := ValidateDecision(output, validAllowedActions())
	assert.False(t, result.Valid)
	assert.True(t, containsField(result.Errors, "schema_version"))
}

func TestValidateDecisionErrors_FormatsMultiple_Extra(t *testing.T) {
	output := &DecisionOutput{}
	msg := ValidateDecisionErrors(output, []string{})
	assert.NotEmpty(t, msg)
	assert.Contains(t, msg, ";")
}

func TestValidationError_Error_Extra(t *testing.T) {
	e := ValidationError{Field: "test_field", Message: "is invalid"}
	assert.Equal(t, "test_field: is invalid", e.Error())
}

func TestProviderFactory_OpenAI_NilRegistry_Extra(t *testing.T) {
	cfg := &config.Config{
		LLMEnabled:  true,
		LLMProvider: "openai",
		LLMAPIKey:   "sk-test",
	}
	factory := NewProviderFactory(cfg, nil)
	provider, err := factory.CreateProvider()
	assert.NoError(t, err)
	_, ok := provider.(*RuleBasedProvider)
	assert.True(t, ok, "expected RuleBasedProvider when registry is nil")
}

func TestPromptRegistry_NotLoaded_Extra(t *testing.T) {
	reg := &PromptRegistry{prompts: make(map[string]*PromptTemplate)}
	_, err := reg.Load("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestPromptRegistry_Hash_NotFound_Extra(t *testing.T) {
	reg := &PromptRegistry{prompts: make(map[string]*PromptTemplate)}
	_, err := reg.Hash("nonexistent")
	assert.Error(t, err)
}

func TestPromptRegistry_List_Empty_Extra(t *testing.T) {
	reg := &PromptRegistry{prompts: make(map[string]*PromptTemplate)}
	ids := reg.List()
	assert.Empty(t, ids)
}

func TestPromptRegistry_RenderUserPrompt_NotFound_Extra(t *testing.T) {
	reg := &PromptRegistry{prompts: make(map[string]*PromptTemplate)}
	_, err := reg.RenderUserPrompt("nonexistent", UserPromptData{})
	assert.Error(t, err)
}
