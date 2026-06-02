package eval

import (
	"context"
	"encoding/json"
	"testing"

	"baxi/internal/llm"
	"github.com/stretchr/testify/assert"
)

// --- Tests: saveResult ---

func TestSaveResult_NilPool_NoOp(t *testing.T) {
	evaluator := NewDecisionEvaluator(nil)
	result := &EvalResult{
		EvalID:     "eval_001",
		EvalStatus: EvalStatusPass,
		Score:      0.95,
	}
	err := evaluator.saveResult(context.Background(), result)
	assert.NoError(t, err) // nil pool should return nil (no-op)
}

// --- Tests: generateEvalID ---

func TestGenerateEvalID_Format_Extra(t *testing.T) {
	id := generateEvalID()
	assert.Contains(t, id, "eval_")
	assert.GreaterOrEqual(t, len(id), 20)
}

func TestGenerateEvalID_Unique_Extra(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := generateEvalID()
		assert.False(t, ids[id], "duplicate eval ID: %s", id)
		ids[id] = true
	}
}

// --- Tests: randStr (additional) ---

func TestRandStr_Length_Zero(t *testing.T) {
	result := randStr(0)
	assert.Len(t, result, 0)
}

func TestRandStr_Length_One(t *testing.T) {
	result := randStr(1)
	assert.Len(t, result, 1)
}

func TestRandStr_Length_Large(t *testing.T) {
	result := randStr(1000)
	assert.Len(t, result, 1000)
}

func TestRandStr_ContainsOnlyValidChars_Extra(t *testing.T) {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := randStr(1000)
	for _, c := range result {
		found := false
		for _, l := range letters {
			if c == l {
				found = true
				break
			}
		}
		assert.True(t, found, "unexpected character: %c", c)
	}
}

func TestRandStr_Uniqueness_Extra(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		s := randStr(12)
		assert.False(t, seen[s], "duplicate random string: %s", s)
		seen[s] = true
	}
}

// --- Tests: evaluateDimensions ---

func TestEvaluateDimensions_NilOutput_Extra(t *testing.T) {
	evaluator := NewDecisionEvaluator(nil)
	dims := evaluator.evaluateDimensions(nil, nil)
	assert.Len(t, dims, 9)
	for _, d := range dims {
		assert.False(t, d.Passed)
		assert.Equal(t, 0.0, d.Score)
	}
}

func TestEvaluateDimensions_InvalidActions_Extra(t *testing.T) {
	evaluator := NewDecisionEvaluator(nil)
	output := &llm.DecisionOutput{
		DecisionType:        llm.DecisionTypeInvestigate,
		Severity:            llm.SeverityHigh,
		Summary:             "Valid summary text for context grounding",
		Rationale:           []string{"reason 1"},
		Confidence:          0.8,
		RequiresHumanReview: true,
		RecommendedActions: []llm.RecommendedAction{
			{ActionType: "invalid_action", OwnerRole: "ops"},
		},
	}

	dims := evaluator.evaluateDimensions(output, validAllowedActions())
	for _, d := range dims {
		if d.ID == "governance_compliance" {
			assert.False(t, d.Passed)
			assert.Equal(t, 0.0, d.Score)
		}
		if d.ID == "action_safety" {
			assert.False(t, d.Passed)
		}
	}
}

func TestEvaluateDimensions_AllPass_Extra(t *testing.T) {
	evaluator := NewDecisionEvaluator(nil)
	output := &llm.DecisionOutput{
		DecisionType:        llm.DecisionTypeInvestigate,
		Severity:            llm.SeverityMedium,
		Summary:             "A sufficiently long summary for context grounding check",
		Rationale:           []string{"reason 1", "reason 2"},
		Confidence:          0.8,
		RequiresHumanReview: true,
		RecommendedActions: []llm.RecommendedAction{
			{ActionType: llm.ActionTypeNotifyOwner, Priority: "high", OwnerRole: "ops"},
			{ActionType: llm.ActionTypeCreateFollowupTask, Priority: "medium", OwnerRole: "team"},
		},
	}

	dims := evaluator.evaluateDimensions(output, validAllowedActions())
	for _, d := range dims {
		assert.True(t, d.Passed, "expected dimension %s to pass", d.ID)
		assert.Equal(t, 1.0, d.Score, "expected score 1.0 for dimension %s", d.ID)
	}
}

func TestEvaluateDimensions_EmptyActions_Extra(t *testing.T) {
	evaluator := NewDecisionEvaluator(nil)
	output := &llm.DecisionOutput{
		DecisionType:        llm.DecisionTypeInvestigate,
		Severity:            llm.SeverityMedium,
		Summary:             "A sufficiently long summary for context grounding check",
		Rationale:           []string{"reason"},
		Confidence:          0.8,
		RequiresHumanReview: true,
		RecommendedActions:  []llm.RecommendedAction{},
	}

	dims := evaluator.evaluateDimensions(output, validAllowedActions())
	for _, d := range dims {
		switch d.ID {
		case "not_overgeneralized", "has_owner_role", "action_relevance", "governance_compliance":
			// Empty actions: governance_compliance checks len>0, fails
			// not_overgeneralized checks len>0, fails
			// has_owner_role checks len>0, fails
			// action_relevance checks len>0, fails
			assert.False(t, d.Passed, "expected %s to fail with empty actions", d.ID)
		case "action_safety":
			// action_safety checks all actions are allowed; with no actions loop doesn't execute, passes
			assert.True(t, d.Passed, "expected action_safety to pass with empty actions")
		}
	}
}

// --- Tests: Evaluate ---

func TestEvaluate_WithValidOutput_Extra(t *testing.T) {
	evaluator := NewDecisionEvaluator(nil)
	output := &llm.DecisionOutput{
		DecisionType:        llm.DecisionTypeInvestigate,
		Severity:            llm.SeverityHigh,
		Summary:             "This is a sufficiently long summary for context grounding check",
		Rationale:           []string{"metric anomaly detected"},
		Confidence:          0.85,
		RequiresHumanReview: true,
		RecommendedActions: []llm.RecommendedAction{
			{ActionType: llm.ActionTypeNotifyOwner, Priority: "high", OwnerRole: "ops"},
		},
	}

	result, err := evaluator.Evaluate(context.Background(), "case-1", "decision-1", output, validAllowedActions())
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.EvalID)
	assert.Contains(t, result.EvalID, "eval_")
	assert.Equal(t, "case-1", result.DecisionCaseID)
	assert.Equal(t, "decision-1", result.LLMDecisionID)
	assert.Equal(t, "all_dimensions", result.EvalRuleID)
	assert.NotEmpty(t, result.DetailsJSON)
	assert.False(t, result.CreatedAt.IsZero())
}

func TestEvaluate_WithNilOutput_Extra(t *testing.T) {
	evaluator := NewDecisionEvaluator(nil)
	result, err := evaluator.Evaluate(context.Background(), "case-1", "decision-1", nil, validAllowedActions())
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, EvalStatusFail, result.EvalStatus)
	assert.Less(t, result.Score, 0.75)
}

func TestEvaluate_FailStatus_Extra(t *testing.T) {
	evaluator := NewDecisionEvaluator(nil)
	output := &llm.DecisionOutput{
		DecisionType:        "invalid",
		Severity:            "unknown",
		Summary:             "short",
		Rationale:           []string{},
		Confidence:          0.1,
		RequiresHumanReview: false,
		RecommendedActions: []llm.RecommendedAction{
			{ActionType: "bad_action", Priority: "unknown", OwnerRole: ""},
		},
	}

	result, err := evaluator.Evaluate(context.Background(), "case-1", "decision-1", output, validAllowedActions())
	assert.NoError(t, err)
	assert.Equal(t, EvalStatusFail, result.EvalStatus)
}

// --- Tests: EvalStatus constants ---

func TestEvalStatusConstants_Extra(t *testing.T) {
	assert.Equal(t, "pass", EvalStatusPass)
	assert.Equal(t, "fail", EvalStatusFail)
	assert.Equal(t, "partial", EvalStatusPartial)
}

// --- Tests: DimensionResult ---

func TestDimensionResult_Fields_Extra(t *testing.T) {
	dr := DimensionResult{
		ID:      "test_dim",
		Score:   0.75,
		Passed:  true,
		Details: map[string]string{"key": "val"},
	}
	data, err := json.Marshal(dr)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "test_dim")
}

// --- Tests: EvalResult with full fields ---

func TestEvalResult_JSON_Extra(t *testing.T) {
	result := &EvalResult{
		EvalID:         "eval_001",
		DecisionCaseID: "case-001",
		LLMDecisionID:  "decision-001",
		EvalRuleID:     "all_dimensions",
		EvalStatus:     EvalStatusPass,
		Score:          0.95,
		DetailsJSON:    json.RawMessage(`[]`),
	}
	data, err := json.Marshal(result)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "eval_001")
}
