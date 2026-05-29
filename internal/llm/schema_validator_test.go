package llm

import (
	"testing"
)

func validAllowedActions() []string {
	return []string{
		ActionTypeCreateFollowupTask,
		ActionTypeNotifyOwner,
		ActionTypeExportReport,
		ActionTypeEscalateToHuman,
	}
}

func validDecisionOutput() *DecisionOutput {
	return &DecisionOutput{
		DecisionType:        DecisionTypeInvestigate,
		Severity:            SeverityMedium,
		Summary:             "Test summary",
		Rationale:           []string{"reason 1", "reason 2"},
		Confidence:          0.85,
		RequiresHumanReview: true,
		RecommendedActions: []RecommendedAction{
			{ActionType: ActionTypeNotifyOwner, Priority: "high", OwnerRole: "data_engineer"},
		},
	}
}

func TestValidator_ValidDecision_Passes(t *testing.T) {
	output := validDecisionOutput()
	allowed := validAllowedActions()

	result := ValidateDecision(output, allowed)

	if !result.Valid {
		t.Errorf("expected Valid=true, got false with errors: %v", result.Errors)
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected 0 errors, got %d: %v", len(result.Errors), result.Errors)
	}
}

func TestValidator_InvalidDecisionType_ReturnsError(t *testing.T) {
	output := validDecisionOutput()
	output.DecisionType = "bogus_type"

	result := ValidateDecision(output, validAllowedActions())

	if result.Valid {
		t.Fatal("expected Valid=false for invalid decision_type, got true")
	}
	if !containsField(result.Errors, "decision_type") {
		t.Errorf("expected error on field 'decision_type', got errors: %v", result.Errors)
	}
}

func TestValidator_InvalidSeverity_ReturnsError(t *testing.T) {
	output := validDecisionOutput()
	output.Severity = "unknown_severity"

	result := ValidateDecision(output, validAllowedActions())

	if result.Valid {
		t.Fatal("expected Valid=false for invalid severity, got true")
	}
	if !containsField(result.Errors, "severity") {
		t.Errorf("expected error on field 'severity', got errors: %v", result.Errors)
	}
}

func TestValidator_ConfidenceAboveOne_ReturnsError(t *testing.T) {
	output := validDecisionOutput()
	output.Confidence = 1.5

	result := ValidateDecision(output, validAllowedActions())

	if result.Valid {
		t.Fatal("expected Valid=false for confidence > 1, got true")
	}
	if !containsField(result.Errors, "confidence") {
		t.Errorf("expected error on field 'confidence', got errors: %v", result.Errors)
	}
}

func TestValidator_ConfidenceBelowZero_ReturnsError(t *testing.T) {
	output := validDecisionOutput()
	output.Confidence = -0.1

	result := ValidateDecision(output, validAllowedActions())

	if result.Valid {
		t.Fatal("expected Valid=false for confidence < 0, got true")
	}
	if !containsField(result.Errors, "confidence") {
		t.Errorf("expected error on field 'confidence', got errors: %v", result.Errors)
	}
}

func TestValidator_RequiresHumanReviewFalse_ReturnsError(t *testing.T) {
	output := validDecisionOutput()
	output.RequiresHumanReview = false

	result := ValidateDecision(output, validAllowedActions())

	if result.Valid {
		t.Fatal("expected Valid=false for requires_human_review=false, got true")
	}
	if !containsField(result.Errors, "requires_human_review") {
		t.Errorf("expected error on field 'requires_human_review', got errors: %v", result.Errors)
	}
}

func TestValidator_ActionNotInAllowedActions_ReturnsError(t *testing.T) {
	output := validDecisionOutput()
	output.RecommendedActions = []RecommendedAction{
		{ActionType: "restart_server"},
	}
	allowed := []string{ActionTypeNotifyOwner, ActionTypeExportReport}

	result := ValidateDecision(output, allowed)

	if result.Valid {
		t.Fatal("expected Valid=false for action not in allowed_actions, got true")
	}
	if !containsField(result.Errors, "recommended_actions[0].action_type") {
		t.Errorf("expected error on 'recommended_actions[0].action_type', got errors: %v", result.Errors)
	}
}

func TestValidator_InvalidActionType_ReturnsError(t *testing.T) {
	output := validDecisionOutput()
	output.RecommendedActions = []RecommendedAction{
		{ActionType: "bogus_action"},
	}

	result := ValidateDecision(output, validAllowedActions())

	if result.Valid {
		t.Fatal("expected Valid=false for invalid action_type, got true")
	}
	if !containsField(result.Errors, "recommended_actions[0].action_type") {
		t.Errorf("expected error on 'recommended_actions[0].action_type', got errors: %v", result.Errors)
	}
}

func TestValidator_NilOutput_ReturnsError(t *testing.T) {
	result := ValidateDecision(nil, validAllowedActions())

	if result.Valid {
		t.Fatal("expected Valid=false for nil output, got true")
	}
	if !containsField(result.Errors, "output") {
		t.Errorf("expected error on field 'output', got errors: %v", result.Errors)
	}
}

func TestValidator_MultipleErrors_Accumulate(t *testing.T) {
	output := &DecisionOutput{
		DecisionType:        "",
		Severity:            "",
		Confidence:          -0.5,
		RequiresHumanReview: false,
		RecommendedActions: []RecommendedAction{
			{ActionType: "invalid_action"},
		},
	}

	result := ValidateDecision(output, []string{})

	if result.Valid {
		t.Fatal("expected Valid=false for multiple errors, got true")
	}

	// Must have at least 4 errors: decision_type, severity, confidence, requires_human_review
	// plus potentially recommended_actions errors
	expectedFields := []string{"decision_type", "severity", "confidence", "requires_human_review"}
	for _, field := range expectedFields {
		if !containsField(result.Errors, field) {
			t.Errorf("expected error on field %q in multi-error test, got errors: %v", field, result.Errors)
		}
	}

	if len(result.Errors) < 4 {
		t.Errorf("expected at least 4 errors accumulated, got %d: %v", len(result.Errors), result.Errors)
	}
}

func TestValidator_ValidateDecisionErrors_EmptyOnValid(t *testing.T) {
	output := validDecisionOutput()
	msg := ValidateDecisionErrors(output, validAllowedActions())
	if msg != "" {
		t.Errorf("expected empty string for valid output, got %q", msg)
	}
}

func TestValidator_ValidateDecisionErrors_NonEmptyOnInvalid(t *testing.T) {
	output := validDecisionOutput()
	output.DecisionType = "bad"
	msg := ValidateDecisionErrors(output, validAllowedActions())
	if msg == "" {
		t.Fatal("expected non-empty error summary, got empty string")
	}
}

func TestValidator_ConfidenceBoundaryZero_IsValid(t *testing.T) {
	output := validDecisionOutput()
	output.Confidence = 0

	result := ValidateDecision(output, validAllowedActions())

	if !result.Valid {
		t.Errorf("expected Valid=true for confidence=0, got false: %v", result.Errors)
	}
}

func TestValidator_ConfidenceBoundaryOne_IsValid(t *testing.T) {
	output := validDecisionOutput()
	output.Confidence = 1.0

	result := ValidateDecision(output, validAllowedActions())

	if !result.Valid {
		t.Errorf("expected Valid=true for confidence=1.0, got false: %v", result.Errors)
	}
}

// containsField checks if any ValidationError in the slice has the given field name.
func containsField(errors []ValidationError, field string) bool {
	for _, e := range errors {
		if e.Field == field {
			return true
		}
	}
	return false
}
