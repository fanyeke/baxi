package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/config"
)

func TestParsePromptFilename_Valid(t *testing.T) {
	tests := []struct {
		name       string
		filename   string
		promptID   string
		promptType string
		version    string
	}{
		{"system", "decision_system_v1.md", "decision_support", "system", "v1"},
		{"user", "decision_user_v1.md", "decision_support", "user", "v1"},
		{"repair", "decision_repair_v1.md", "decision_support", "repair", "v1"},
		{"multi_word_domain", "my_custom_domain_system_v2.md", "my_custom_domain_support", "system", "v2"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			id, pType, ver, err := parsePromptFilename(tc.filename)
			require.NoError(t, err)
			assert.Equal(t, tc.promptID, id)
			assert.Equal(t, tc.promptType, pType)
			assert.Equal(t, tc.version, ver)
		})
	}
}

func TestParsePromptFilename_InvalidFormats(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{"too_few_parts", "decision.md"},
		{"no_version_prefix", "decision_system_1.md"},
		{"single_part", "decision.md"},
		{"empty", ".md"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, _, err := parsePromptFilename(tc.filename)
			assert.Error(t, err)
		})
	}
}

func TestNewPromptRegistry_Success(t *testing.T) {
	reg, err := NewPromptRegistry()
	require.NoError(t, err)
	require.NotNil(t, reg)
	// Should have at least one prompt loaded
	assert.NotEmpty(t, reg.List())
}

func TestNewPromptRegistry_LoadAndHash(t *testing.T) {
	reg, err := NewPromptRegistry()
	require.NoError(t, err)

	ids := reg.List()
	require.NotEmpty(t, ids)

	for _, id := range ids {
		tmpl, err := reg.Load(id)
		assert.NoError(t, err)
		assert.NotNil(t, tmpl)
		assert.NotEmpty(t, tmpl.SystemPrompt)
		assert.NotEmpty(t, tmpl.UserTemplate)
		assert.NotEmpty(t, tmpl.Hash)

		hash, err := reg.Hash(id)
		assert.NoError(t, err)
		assert.Equal(t, tmpl.Hash, hash)
	}
}

func TestPromptRegistry_RenderUserPrompt_Success(t *testing.T) {
	reg, err := NewPromptRegistry()
	require.NoError(t, err)

	ids := reg.List()
	require.NotEmpty(t, ids)

	// Find a prompt that supports rendering
	for _, id := range ids {
		tmpl, err := reg.Load(id)
		require.NoError(t, err)

		data := UserPromptData{
			ContextJSON:      `{"test": "value"}`,
			AllowedActions:   []string{ActionTypeNotifyOwner},
			ForbiddenActions: []string{},
		}

		rendered, err := reg.RenderUserPrompt(id, data)
		assert.NoError(t, err)
		assert.NotEmpty(t, rendered)
		_ = tmpl
		return // test one prompt is enough
	}
}

func TestLLMSafeContext_Fields(t *testing.T) {
	ctx := LLMSafeContext{
		CaseID: "case-001",
		Trigger: TriggerInfo{
			AlertID:       "alert-001",
			RuleID:        "gmv_drop",
			Severity:      "high",
			MetricName:    "gmv",
			CurrentValue:  1000.0,
			BaselineValue: 1500.0,
			DeltaPct:      -33.3,
		},
		ObjectContext: ObjectContext{
			ObjectType: "order",
			ObjectID:   "order-001",
			Properties: map[string]interface{}{"status": "delivered"},
		},
		GovernanceInfo: GovernanceInfo{
			Classification:   "L2",
			RedactionApplied: true,
			RedactedFields:   []string{"email"},
			Role:             "analyst",
		},
		AllowedActions:   []string{ActionTypeNotifyOwner},
		ForbiddenActions: []string{ActionTypeCreateOutboxMessage},
		EnrichedObjects: []EnrichedObjectData{
			{
				LinkName:   "customer",
				Depth:      1,
				ObjectType: "customer",
				ObjectID:   "cust-001",
				Properties: map[string]interface{}{"name": "Test"},
			},
		},
	}

	assert.Equal(t, "case-001", ctx.CaseID)
	assert.Equal(t, "alert-001", ctx.Trigger.AlertID)
	assert.Equal(t, "gmv_drop", ctx.Trigger.RuleID)
	assert.Equal(t, "high", ctx.Trigger.Severity)
	assert.Equal(t, "order", ctx.ObjectContext.ObjectType)
	assert.True(t, ctx.GovernanceInfo.RedactionApplied)
	assert.Len(t, ctx.EnrichedObjects, 1)
	assert.Equal(t, "customer", ctx.EnrichedObjects[0].ObjectType)
}

func TestRecommendedAction_Fields(t *testing.T) {
	action := RecommendedAction{
		ActionType: ActionTypeNotifyOwner,
		Priority:   "high",
		OwnerRole:  "ops",
		Payload:    map[string]interface{}{"message": "test"},
	}
	assert.Equal(t, ActionTypeNotifyOwner, action.ActionType)
	assert.Equal(t, "high", action.Priority)
	assert.Equal(t, "ops", action.OwnerRole)
	assert.Equal(t, "test", action.Payload["message"])
}

func TestDecisionOutput_AllDecisionTypes(t *testing.T) {
	types := []string{
		DecisionTypeMonitor,
		DecisionTypeInvestigate,
		DecisionTypeOptimize,
		DecisionTypeIntervention,
		DecisionTypeExperiment,
	}
	for _, dt := range types {
		t.Run(dt, func(t *testing.T) {
			output := validDecisionOutput()
			output.DecisionType = dt
			result := ValidateDecision(output, validAllowedActions())
			// Should not have decision_type error
			for _, e := range result.Errors {
				assert.NotEqual(t, "decision_type", e.Field)
			}
		})
	}
}

func TestDecisionOutput_AllSeverities(t *testing.T) {
	severities := []string{
		SeverityLow,
		SeverityMedium,
		SeverityHigh,
		SeverityCritical,
	}
	for _, sev := range severities {
		t.Run(sev, func(t *testing.T) {
			output := validDecisionOutput()
			output.Severity = sev
			result := ValidateDecision(output, validAllowedActions())
			for _, e := range result.Errors {
				assert.NotEqual(t, "severity", e.Field)
			}
		})
	}
}

func TestValidateDecision_EmptyActionsAllowed(t *testing.T) {
	output := validDecisionOutput()
	output.RecommendedActions = nil
	result := ValidateDecision(output, []string{})
	assert.True(t, result.Valid)
}

func TestValidateDecision_MultipleActionsMultipleErrors(t *testing.T) {
	output := validDecisionOutput()
	output.RecommendedActions = []RecommendedAction{
		{ActionType: "bad_action_1"},
		{ActionType: "bad_action_2"},
		{ActionType: "bad_action_3"},
	}
	result := ValidateDecision(output, []string{})
	assert.False(t, result.Valid)
	// Each action should produce errors
	actionErrors := 0
	for _, e := range result.Errors {
		if containsField([]ValidationError{e}, "recommended_actions") || e.Field != "" {
			actionErrors++
		}
	}
	assert.GreaterOrEqual(t, actionErrors, 3)
}

func TestProviderFactory_AllBranches(t *testing.T) {
	tests := []struct {
		name          string
		provider      string
		enabled       bool
		expectType    string
		expectError   bool
		nilRegistry   bool
	}{
		{"disabled_override", "openai", false, "*llm.RuleBasedProvider", false, false},
		{"disabled_string", "disabled", true, "*llm.RuleBasedProvider", false, false},
		{"empty_string", "", true, "*llm.RuleBasedProvider", false, false},
		{"rule_based", "rule_based", true, "*llm.RuleBasedProvider", false, false},
		{"openai_nil_registry", "openai", true, "*llm.RuleBasedProvider", false, true},
		{"openai_compatible", "openai_compatible", true, "*llm.OpenAICompatibleProvider", false, false},
		{"unknown", "anthropic", true, "", true, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{
				LLMEnabled:  tc.enabled,
				LLMProvider: tc.provider,
				LLMAPIKey:   "sk-test",
				LLMModel:    "gpt-4o",
			}

			var reg *PromptRegistry
			if !tc.nilRegistry {
				var err error
				reg, err = NewPromptRegistry()
				require.NoError(t, err)
			}

			factory := NewProviderFactory(cfg, reg)
			provider, err := factory.CreateProvider()

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, provider)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)
			}
		})
	}
}

func TestRuleBasedProvider_GenerateDecision_NilTrigger(t *testing.T) {
	p := NewRuleBasedProvider()
	input := LLMSafeContext{}
	output, err := p.GenerateDecision(t.Context(), input)
	assert.NoError(t, err)
	assert.NotNil(t, output)
	// Default severity is empty string, hits the default case
	assert.Equal(t, DecisionTypeInvestigate, output.DecisionType)
}
