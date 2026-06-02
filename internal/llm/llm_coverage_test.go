package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"baxi/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──── GenerateDecisionRaw ──────────────────────────────────────────────────

func TestGenerateDecisionRaw_ValidResponse(t *testing.T) {
	validJSON := validDecisionJSON()
	mockServer := newMockOpenAIServer(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"created": 1234567890,
			"model":   "gpt-4o-mini",
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"message":       map[string]interface{}{"role": "assistant", "content": validJSON, "refusal": ""},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{"prompt_tokens": 50, "completion_tokens": 100, "total_tokens": 150},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	})
	defer mockServer.Close()

	provider := newTestOpenAIProvider(t, mockServer.URL)
	output, rawContent, err := provider.GenerateDecisionRaw(context.Background(), testInput())
	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.NotEmpty(t, rawContent)
	assert.Equal(t, "investigate", output.DecisionType)
	assert.True(t, output.RequiresHumanReview)
	assert.Equal(t, "decision_output.v1", output.SchemaVersion)
}

func TestGenerateDecisionRaw_EmptyChoices(t *testing.T) {
	mockServer := newMockOpenAIServer(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"id":      "chatcmpl-test",
			"choices": []map[string]interface{}{},
			"usage":   map[string]interface{}{"prompt_tokens": 10, "completion_tokens": 0, "total_tokens": 10},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	})
	defer mockServer.Close()

	provider := newTestOpenAIProvider(t, mockServer.URL)
	_, _, err := provider.GenerateDecisionRaw(context.Background(), testInput())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty choices")
}

func TestGenerateDecisionRaw_Refusal(t *testing.T) {
	mockServer := newMockOpenAIServer(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"id":      "chatcmpl-test",
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"message":       map[string]interface{}{"role": "assistant", "content": "", "refusal": "I refuse"},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	})
	defer mockServer.Close()

	provider := newTestOpenAIProvider(t, mockServer.URL)
	_, _, err := provider.GenerateDecisionRaw(context.Background(), testInput())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "refusal")
}

func TestGenerateDecisionRaw_InvalidJSON(t *testing.T) {
	mockServer := newMockOpenAIServer(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"id":      "chatcmpl-test",
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"message":       map[string]interface{}{"role": "assistant", "content": "not json at all", "refusal": ""},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	})
	defer mockServer.Close()

	provider := newTestOpenAIProvider(t, mockServer.URL)
	_, _, err := provider.GenerateDecisionRaw(context.Background(), testInput())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse LLM response")
}

func TestGenerateDecisionRaw_Timeout(t *testing.T) {
	mockServer := newMockOpenAIServer(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	})
	defer mockServer.Close()

	cfg := &config.Config{
		LLMAPIKey:         "sk-test",
		LLMModel:          "gpt-4o-mini",
		LLMAPIBase:        mockServer.URL,
		LLMTemperature:    0.2,
		LLMMaxTokens:      2048,
		LLMTimeoutSeconds: 1,
	}
	registry, _ := NewPromptRegistry()
	provider, _ := NewOpenAIProvider(cfg, registry)

	_, _, err := provider.GenerateDecisionRaw(context.Background(), testInput())
	require.Error(t, err)
	var perr *ProviderError
	assert.ErrorAs(t, err, &perr)
}

func TestGenerateDecisionRaw_500(t *testing.T) {
	mockServer := newMockOpenAIServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "server error"})
	})
	defer mockServer.Close()

	provider := newTestOpenAIProvider(t, mockServer.URL)
	_, _, err := provider.GenerateDecisionRaw(context.Background(), testInput())
	require.Error(t, err)
	var perr *ProviderError
	assert.ErrorAs(t, err, &perr)
}

func TestGenerateDecisionRaw_EmptySchemaVersion(t *testing.T) {
	// Response without schema_version should default to "decision_output.v1"
	validJSON := `{"decision_type":"investigate","severity":"medium","summary":"test","rationale":["r1"],"recommended_actions":[],"confidence":0.7,"requires_human_review":true}`
	mockServer := newMockOpenAIServer(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"id":      "chatcmpl-test",
			"choices": []map[string]interface{}{
				{
					"index":   0,
					"message": map[string]interface{}{"role": "assistant", "content": validJSON, "refusal": ""},
				},
			},
			"usage": map[string]interface{}{"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	})
	defer mockServer.Close()

	provider := newTestOpenAIProvider(t, mockServer.URL)
	output, raw, err := provider.GenerateDecisionRaw(context.Background(), testInput())
	require.NoError(t, err)
	assert.Equal(t, "decision_output.v1", output.SchemaVersion)
	assert.NotEmpty(t, raw)
}

// ──── ProviderError ────────────────────────────────────────────────────────

func TestProviderError_Error(t *testing.T) {
	err := &ProviderError{
		Err:       fmt.Errorf("connection refused"),
		Provider:  "openai",
		Model:     "gpt-4o",
		LatencyMs: 100,
	}
	msg := err.Error()
	assert.Contains(t, msg, "openai")
	assert.Contains(t, msg, "gpt-4o")
	assert.Contains(t, msg, "connection refused")
}

// ──── parsePromptFilename ──────────────────────────────────────────────────

func TestParsePromptFilename_AllTypes(t *testing.T) {
	// parsePromptFilename only validates format, not type validity.
	// All valid {domain}_{type}_v{version}.md patterns parse successfully.
	tests := []struct {
		filename string
		wantType string
	}{
		{"decision_system_v1.md", "system"},
		{"decision_user_v1.md", "user"},
		{"decision_repair_v1.md", "repair"},
		{"decision_custom_v1.md", "custom"},
	}
	for _, tc := range tests {
		t.Run(tc.filename, func(t *testing.T) {
			id, pType, _, err := parsePromptFilename(tc.filename)
			assert.NoError(t, err)
			assert.Equal(t, tc.wantType, pType)
			assert.NotEmpty(t, id)
		})
	}
}

// ──── RepairPromptRenderer ────────────────────────────────────────────────

func TestRepairPromptRenderer_RenderWithErrors(t *testing.T) {
	renderer, err := NewRepairPromptRenderer()
	require.NoError(t, err)

	errors := []ValidationError{
		{Field: "confidence", Message: "out of range"},
		{Field: "severity", Message: "invalid"},
	}

	rendered, err := renderer.RenderRepairPrompt(errors)
	require.NoError(t, err)
	assert.NotEmpty(t, rendered)
}

func TestRepairPromptRenderer_RenderEmptyErrors(t *testing.T) {
	renderer, err := NewRepairPromptRenderer()
	require.NoError(t, err)

	rendered, err := renderer.RenderRepairPrompt([]ValidationError{})
	require.NoError(t, err)
	assert.NotEmpty(t, rendered)
}

// ──── PromptRegistry ──────────────────────────────────────────────────────

func TestPromptRegistry_RenderUserPrompt_WithData(t *testing.T) {
	reg, err := NewPromptRegistry()
	require.NoError(t, err)

	ids := reg.List()
	require.NotEmpty(t, ids)

	for _, id := range ids {
		data := UserPromptData{
			ContextJSON:      `{"case_id":"test"}`,
			AllowedActions:   []string{"notify_owner"},
			ForbiddenActions: []string{"export_report"},
			EnrichedObjects: []EnrichedObjectData{
				{LinkName: "customer", Depth: 1, ObjectType: "customer", ObjectID: "c1"},
			},
		}
		rendered, err := reg.RenderUserPrompt(id, data)
		if err != nil {
			continue // some prompts may not have templates
		}
		assert.NotEmpty(t, rendered)
		return
	}
}

func TestPromptRegistry_NewPromptRegistry_AllLoaded(t *testing.T) {
	reg, err := NewPromptRegistry()
	require.NoError(t, err)

	ids := reg.List()
	assert.GreaterOrEqual(t, len(ids), 1, "should load at least 1 prompt")

	for _, id := range ids {
		tmpl, err := reg.Load(id)
		assert.NoError(t, err)
		assert.NotEmpty(t, tmpl.SystemPrompt)
		assert.NotEmpty(t, tmpl.UserTemplate)
		assert.NotEmpty(t, tmpl.Hash)
		assert.NotEmpty(t, tmpl.Version)
		assert.NotEmpty(t, tmpl.ID)
	}
}

// ──── RuleBasedProvider ────────────────────────────────────────────────────

func TestRuleBasedProvider_AllSeverities(t *testing.T) {
	p := NewRuleBasedProvider()
	severities := []string{
		SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow, "unknown",
	}

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
			assert.Equal(t, sev, output.Severity)
			assert.True(t, output.RequiresHumanReview)
			assert.NotEmpty(t, output.DecisionType)
			assert.NotEmpty(t, output.Summary)
			assert.NotEmpty(t, output.Rationale)
			assert.NotEmpty(t, output.RecommendedActions)
		})
	}
}

// ──── DisabledProvider ─────────────────────────────────────────────────────

func TestDisabledProvider(t *testing.T) {
	p := NewDisabledProvider()
	_, err := p.GenerateDecision(context.Background(), LLMSafeContext{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM is disabled")
}

// ──── NoOpAuditLogger ──────────────────────────────────────────────────────

func TestNoOpAuditLogger_NoPanic(t *testing.T) {
	logger := &NoOpAuditLogger{}
	ctx := context.Background()

	// Verify all methods can be called without panic
	logger.LogDecisionRequested(ctx, "case-1", "openai", "gpt-4o")
	logger.LogDecisionCompleted(ctx, "case-1", "openai", "gpt-4o", 100, &TokenUsage{TotalTokens: 500})
	logger.LogDecisionCompleted(ctx, "case-1", "openai", "gpt-4o", 100, nil)
	logger.LogDecisionFailed(ctx, "case-1", "openai", "gpt-4o", fmt.Errorf("error"))
	logger.LogDecisionFailed(ctx, "case-1", "openai", "gpt-4o", nil)
	logger.LogDecisionValidationFailed(ctx, "case-1", []ValidationError{{Field: "f", Message: "m"}})
	logger.LogDecisionValidationFailed(ctx, "case-1", nil)
	logger.LogFallbackUsed(ctx, "case-1", "reason")
	logger.LogDecisionReplayed(ctx, "case-1", "orig-1")
	logger.LogEvalCompleted(ctx, "case-1", "eval-1")
}

// ──── DBAuditLogger ────────────────────────────────────────────────────────

func TestDBAuditLogger_NilPool_NoPanic(t *testing.T) {
	logger := NewDBAuditLogger(nil)
	ctx := context.Background()

	logger.LogDecisionRequested(ctx, "case-1", "openai", "gpt-4o")
	logger.LogDecisionCompleted(ctx, "case-1", "openai", "gpt-4o", 100, &TokenUsage{TotalTokens: 500})
	logger.LogDecisionCompleted(ctx, "case-1", "openai", "gpt-4o", 100, nil)
	logger.LogDecisionFailed(ctx, "case-1", "openai", "gpt-4o", fmt.Errorf("error"))
	logger.LogDecisionFailed(ctx, "case-1", "openai", "gpt-4o", nil)
	logger.LogDecisionValidationFailed(ctx, "case-1", []ValidationError{{Field: "f", Message: "m"}})
	logger.LogDecisionValidationFailed(ctx, "case-1", nil)
	logger.LogFallbackUsed(ctx, "case-1", "reason")
	logger.LogDecisionReplayed(ctx, "case-1", "orig-1")
	logger.LogEvalCompleted(ctx, "case-1", "eval-1")
}

// ──── ContextEnvelope ──────────────────────────────────────────────────────

func TestLLMSafeContextEnvelope_AllFields(t *testing.T) {
	env := &LLMSafeContextEnvelope{
		SchemaVersion:    "v1",
		CaseID:           "case-001",
		AlertID:          "alert-001",
		ContextHash:      "abc123",
		PromptVersion:    "v1.0",
		Evidence:         []EvidenceItem{{Type: "metric", Key: "gmv", Value: 1000.0}},
		AllowedActions:   []string{"notify_owner"},
		ForbiddenActions: []string{"export_report"},
		Governance:       GovernanceInfo{Classification: "L2", Role: "analyst"},
		RedactionSummary: RedactionSummary{TotalFields: 10, RedactedCount: 2},
		ConfigVersions:   map[string]string{"rules": "v1"},
	}

	data, err := json.Marshal(env)
	require.NoError(t, err)
	assert.Contains(t, string(data), "case-001")
	assert.Contains(t, string(data), "abc123")
}

// ──── SchemaValidator ──────────────────────────────────────────────────────

func TestValidateDecision_SchemaVersionValid(t *testing.T) {
	output := validDecisionOutput()
	output.SchemaVersion = "decision_output.v1"
	result := ValidateDecision(output, validAllowedActions())
	assert.True(t, result.Valid)
}

func TestValidateDecision_SchemaVersionInvalid(t *testing.T) {
	output := validDecisionOutput()
	output.SchemaVersion = "decision_output.v99"
	result := ValidateDecision(output, validAllowedActions())
	assert.False(t, result.Valid)
	assert.True(t, containsField(result.Errors, "schema_version"))
}

func TestValidateDecision_EmptySchemaVersion(t *testing.T) {
	output := validDecisionOutput()
	output.SchemaVersion = ""
	result := ValidateDecision(output, validAllowedActions())
	// Empty schema version should not cause an error (only non-empty non-v1 causes error)
	assert.True(t, result.Valid)
}

// ──── EnrichedObjectData ──────────────────────────────────────────────────

func TestEnrichedObjectData_Fields(t *testing.T) {
	obj := EnrichedObjectData{
		LinkName:   "customer",
		Depth:      1,
		ObjectType: "customer",
		ObjectID:   "cust-001",
		Properties: map[string]interface{}{"name": "Test"},
	}

	data, err := json.Marshal(obj)
	require.NoError(t, err)
	assert.Contains(t, string(data), "cust-001")
}

// ──── TriggerInfo ──────────────────────────────────────────────────────────

func TestTriggerInfo_JSON(t *testing.T) {
	trigger := TriggerInfo{
		AlertID:       "alert-1",
		RuleID:        "rule-1",
		Severity:      "high",
		MetricName:    "gmv",
		CurrentValue:  1000.0,
		BaselineValue: 1500.0,
		DeltaPct:      -33.3,
	}

	data, err := json.Marshal(trigger)
	require.NoError(t, err)
	assert.Contains(t, string(data), "alert-1")
	assert.Contains(t, string(data), "rule-1")
}

// ──── GovernanceInfo ───────────────────────────────────────────────────────

func TestGovernanceInfo_JSON(t *testing.T) {
	gi := GovernanceInfo{
		Classification:   "L2",
		RedactionApplied: true,
		RedactedFields:   []string{"email"},
		Role:             "analyst",
		RepairErrors:     []string{"fix1"},
	}

	data, err := json.Marshal(gi)
	require.NoError(t, err)
	assert.Contains(t, string(data), "L2")
	assert.Contains(t, string(data), "email")
}

// ──── DecisionOutput ───────────────────────────────────────────────────────

func TestDecisionOutput_JSON(t *testing.T) {
	output := &DecisionOutput{
		DecisionType:        DecisionTypeInvestigate,
		Severity:            SeverityMedium,
		Summary:             "Test",
		Rationale:           []string{"r1"},
		RecommendedActions:  []RecommendedAction{{ActionType: ActionTypeNotifyOwner, Priority: "high"}},
		Confidence:          0.8,
		RequiresHumanReview: true,
		SchemaVersion:       "decision_output.v1",
	}

	data, err := json.Marshal(output)
	require.NoError(t, err)
	assert.Contains(t, string(data), "decision_output.v1")
}
