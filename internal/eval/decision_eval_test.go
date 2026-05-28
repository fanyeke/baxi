package eval

import (
	"testing"

	"baxi/internal/llm"
)

// validAllowedActions returns the canonical set of allowed action types for tests.
func validAllowedActions() []string {
	return []string{
		llm.ActionTypeCreateFollowupTask,
		llm.ActionTypeNotifyOwner,
		llm.ActionTypeExportReport,
		llm.ActionTypeCreateOutboxMessage,
	}
}

// validDecisionOutput returns a valid DecisionOutput used as baseline for tests.
func validDecisionOutput() *llm.DecisionOutput {
	return &llm.DecisionOutput{
		DecisionType:       llm.DecisionTypeInvestigate,
		Severity:           llm.SeverityMedium,
		Summary:            "Test summary for evaluation",
		Rationale:          []string{"reason 1", "reason 2"},
		Confidence:         0.85,
		RequiresHumanReview: true,
		RecommendedActions: []llm.RecommendedAction{
			{ActionType: llm.ActionTypeNotifyOwner, Priority: "high", OwnerRole: "data_engineer"},
		},
	}
}

func TestEvaluate_ValidDecision_Passes(t *testing.T) {
	evaluator := NewDecisionEvaluator(nil)
	output := validDecisionOutput()

	result, err := evaluator.Evaluate(nil, "case-1", "decision-1", output, validAllowedActions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.EvalStatus != EvalStatusPass {
		t.Errorf("expected status=%q, got %q (score=%.2f)", EvalStatusPass, result.EvalStatus, result.Score)
	}
	if result.Score < 0.75 {
		t.Errorf("expected score >= 0.75, got %.2f", result.Score)
	}
	if result.EvalID == "" {
		t.Error("expected non-empty eval_id")
	}
	if result.DecisionCaseID != "case-1" {
		t.Errorf("expected decision_case_id=case-1, got %s", result.DecisionCaseID)
	}
	if result.LLMDecisionID != "decision-1" {
		t.Errorf("expected llm_decision_id=decision-1, got %s", result.LLMDecisionID)
	}
}

func TestEvaluate_InvalidDecision_ForbiddenAction_Fails(t *testing.T) {
	evaluator := NewDecisionEvaluator(nil)
	output := validDecisionOutput()
	output.RecommendedActions = []llm.RecommendedAction{
		{ActionType: "forbidden_action", Priority: "high", OwnerRole: "admin"},
	}

	result, err := evaluator.Evaluate(nil, "case-2", "decision-2", output, validAllowedActions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.EvalStatus != EvalStatusFail {
		t.Errorf("expected status=%q, got %q (score=%.2f)", EvalStatusFail, result.EvalStatus, result.Score)
	}
	if result.Score >= 0.75 {
		t.Errorf("expected score < 0.75 for invalid decision, got %.2f", result.Score)
	}
}

func TestEvaluate_EmptyRationale_ScoreReduced(t *testing.T) {
	evaluator := NewDecisionEvaluator(nil)
	output := validDecisionOutput()
	output.Rationale = []string{}

	result, err := evaluator.Evaluate(nil, "case-3", "decision-3", output, validAllowedActions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.EvalStatus != EvalStatusPass {
		t.Errorf("expected status=%q (8/9 dims), got %q (score=%.2f)", EvalStatusPass, result.EvalStatus, result.Score)
	}
	if result.Score < 0.88 || result.Score > 0.90 {
		t.Errorf("expected score ~0.89 (8/9), got %.2f", result.Score)
	}
}

func TestEvaluate_MissingHumanReview_DimensionFails(t *testing.T) {
	evaluator := NewDecisionEvaluator(nil)
	output := validDecisionOutput()
	output.RequiresHumanReview = false

	result, err := evaluator.Evaluate(nil, "case-4", "decision-4", output, validAllowedActions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Score < 0.77 || result.Score > 0.79 {
		t.Errorf("expected score ~0.78 (7/9), got %.2f", result.Score)
	}
}

func TestEvaluate_NilOutput_Fails(t *testing.T) {
	evaluator := NewDecisionEvaluator(nil)

	result, err := evaluator.Evaluate(nil, "case-5", "decision-5", nil, validAllowedActions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.EvalStatus != EvalStatusFail {
		t.Errorf("expected status=%q for nil output, got %q", EvalStatusFail, result.EvalStatus)
	}
	if result.Score != 0 {
		t.Errorf("expected score=0 for nil output, got %.2f", result.Score)
	}
}

func TestEvaluate_NoActions_ReducesScores(t *testing.T) {
	evaluator := NewDecisionEvaluator(nil)
	output := validDecisionOutput()
	output.RecommendedActions = []llm.RecommendedAction{}

	result, err := evaluator.Evaluate(nil, "case-6", "decision-6", output, validAllowedActions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Score < 0.55 || result.Score > 0.57 {
		t.Errorf("expected score ~0.56 (5/9), got %.2f", result.Score)
	}
	if result.EvalStatus != EvalStatusFail {
		t.Errorf("expected status=%q with no actions, got %q", EvalStatusFail, result.EvalStatus)
	}
}

func TestGenerateEvalID_UniqueAndFormatted(t *testing.T) {
	id1 := generateEvalID()
	id2 := generateEvalID()

	if id1 == id2 {
		t.Errorf("expected unique IDs, got same: %s", id1)
	}
	if len(id1) < 20 {
		t.Errorf("expected eval ID length >= 20, got %d: %s", len(id1), id1)
	}
	if id1[:5] != "eval_" {
		t.Errorf("expected eval ID to start with 'eval_', got %s", id1[:5])
	}
}

func TestDimensionResult_JSON(t *testing.T) {
	dr := DimensionResult{
		ID:      "test_dimension",
		Score:   1.0,
		Passed:  true,
		Details: map[string]string{"key": "value"},
	}

	if dr.ID != "test_dimension" {
		t.Errorf("expected ID=test_dimension, got %s", dr.ID)
	}
	if dr.Score != 1.0 {
		t.Errorf("expected Score=1.0, got %f", dr.Score)
	}
	if !dr.Passed {
		t.Error("expected Passed=true")
	}
}

func TestEvaluator_NilPool_DoesNotPanic(t *testing.T) {
	evaluator := NewDecisionEvaluator(nil)
	if evaluator.pool != nil {
		t.Error("expected pool to be nil")
	}

	output := validDecisionOutput()
	result, err := evaluator.Evaluate(nil, "case-nil-pool", "decision-nil", output, validAllowedActions())
	if err != nil {
		t.Fatalf("unexpected error with nil pool: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result with nil pool")
	}
}
