package llm

import (
	"context"
	"testing"
)

func TestDisabledProvider_ReturnsError(t *testing.T) {
	p := NewDisabledProvider()
	ctx := context.Background()
	input := LLMSafeContext{CaseID: "test-123"}

	out, err := p.GenerateDecision(ctx, input)

	if err == nil {
		t.Fatal("expected error from DisabledProvider, got nil")
	}
	if out != nil {
		t.Fatal("expected nil DecisionOutput from DisabledProvider, got non-nil")
	}

	expectedMsg := "LLM is disabled: LLM_ENABLED=false"
	if err.Error() != expectedMsg {
		t.Errorf("expected error %q, got %q", expectedMsg, err.Error())
	}
}

func TestDisabledProvider_SatisfiesInterface(t *testing.T) {
	// Compile-time check: DisabledProvider implements DecisionProvider.
	var _ DecisionProvider = (*DisabledProvider)(nil)
}

func TestDecisionOutput_ZeroValues(t *testing.T) {
	var d DecisionOutput

	if d.DecisionType != "" {
		t.Errorf("expected empty DecisionType, got %q", d.DecisionType)
	}
	if d.Severity != "" {
		t.Errorf("expected empty Severity, got %q", d.Severity)
	}
	if d.Summary != "" {
		t.Errorf("expected empty Summary, got %q", d.Summary)
	}
	if d.Rationale != nil {
		t.Errorf("expected nil Rationale, got %v", d.Rationale)
	}
	if d.RecommendedActions != nil {
		t.Errorf("expected nil RecommendedActions, got %v", d.RecommendedActions)
	}
	if d.Confidence != 0 {
		t.Errorf("expected 0 Confidence, got %f", d.Confidence)
	}
	if d.RequiresHumanReview {
		t.Errorf("expected RequiresHumanReview=false, got true")
	}
}

func TestRecommendedAction_ZeroValues(t *testing.T) {
	var a RecommendedAction

	if a.ActionType != "" {
		t.Errorf("expected empty ActionType, got %q", a.ActionType)
	}
	if a.Priority != "" {
		t.Errorf("expected empty Priority, got %q", a.Priority)
	}
	if a.OwnerRole != "" {
		t.Errorf("expected empty OwnerRole, got %q", a.OwnerRole)
	}
	if a.Payload != nil {
		t.Errorf("expected nil Payload, got %v", a.Payload)
	}
}

func TestLLMSafeContext_ZeroValues(t *testing.T) {
	var c LLMSafeContext

	if c.CaseID != "" {
		t.Errorf("expected empty CaseID, got %q", c.CaseID)
	}
	if c.AllowedActions != nil {
		t.Errorf("expected nil AllowedActions, got %v", c.AllowedActions)
	}
	if c.ForbiddenActions != nil {
		t.Errorf("expected nil ForbiddenActions, got %v", c.ForbiddenActions)
	}
}

func TestDecisionTypeConstants(t *testing.T) {
	types := []struct {
		name  string
		value string
	}{
		{"DecisionTypeMonitor", DecisionTypeMonitor},
		{"DecisionTypeInvestigate", DecisionTypeInvestigate},
		{"DecisionTypeOptimize", DecisionTypeOptimize},
		{"DecisionTypeIntervention", DecisionTypeIntervention},
		{"DecisionTypeExperiment", DecisionTypeExperiment},
	}

	seen := make(map[string]string)
	for _, tc := range types {
		if tc.value == "" {
			t.Errorf("constant %s has empty value", tc.name)
		}
		if prev, ok := seen[tc.value]; ok {
			t.Errorf("duplicate value %q for constants %s and %s", tc.value, prev, tc.name)
		}
		seen[tc.value] = tc.name
	}

	if len(types) < 4 {
		t.Errorf("expected at least 4 decision type constants, got %d", len(types))
	}
}

func TestSeverityConstants(t *testing.T) {
	severities := []struct {
		name  string
		value string
	}{
		{"SeverityLow", SeverityLow},
		{"SeverityMedium", SeverityMedium},
		{"SeverityHigh", SeverityHigh},
		{"SeverityCritical", SeverityCritical},
	}

	seen := make(map[string]string)
	for _, tc := range severities {
		if tc.value == "" {
			t.Errorf("constant %s has empty value", tc.name)
		}
		if prev, ok := seen[tc.value]; ok {
			t.Errorf("duplicate value %q for constants %s and %s", tc.value, prev, tc.name)
		}
		seen[tc.value] = tc.name
	}

	if len(severities) < 3 {
		t.Errorf("expected at least 3 severity constants, got %d", len(severities))
	}
}

func TestActionTypeConstants(t *testing.T) {
	actions := []struct {
		name  string
		value string
	}{
		{"ActionTypeCreateFollowupTask", ActionTypeCreateFollowupTask},
		{"ActionTypeNotifyOwner", ActionTypeNotifyOwner},
		{"ActionTypeExportReport", ActionTypeExportReport},
		{"ActionTypeEscalateToHuman", ActionTypeEscalateToHuman},
	}

	seen := make(map[string]string)
	for _, tc := range actions {
		if tc.value == "" {
			t.Errorf("constant %s has empty value", tc.name)
		}
		if prev, ok := seen[tc.value]; ok {
			t.Errorf("duplicate value %q for constants %s and %s", tc.value, prev, tc.name)
		}
		seen[tc.value] = tc.name
	}

	if len(actions) < 3 {
		t.Errorf("expected at least 3 action type constants, got %d", len(actions))
	}
}

func TestTriggerInfo_ZeroValues(t *testing.T) {
	var ti TriggerInfo

	if ti.AlertID != "" {
		t.Errorf("expected empty AlertID, got %q", ti.AlertID)
	}
	if ti.CurrentValue != 0 {
		t.Errorf("expected 0 CurrentValue, got %f", ti.CurrentValue)
	}
	if ti.BaselineValue != 0 {
		t.Errorf("expected 0 BaselineValue, got %f", ti.BaselineValue)
	}
	if ti.DeltaPct != 0 {
		t.Errorf("expected 0 DeltaPct, got %f", ti.DeltaPct)
	}
}

func TestObjectContext_ZeroValues(t *testing.T) {
	var oc ObjectContext

	if oc.ObjectType != "" {
		t.Errorf("expected empty ObjectType, got %q", oc.ObjectType)
	}
	if oc.ObjectID != "" {
		t.Errorf("expected empty ObjectID, got %q", oc.ObjectID)
	}
	if oc.Properties != nil {
		t.Errorf("expected nil Properties, got %v", oc.Properties)
	}
}

func TestGovernanceInfo_ZeroValues(t *testing.T) {
	var gi GovernanceInfo

	if gi.Classification != "" {
		t.Errorf("expected empty Classification, got %q", gi.Classification)
	}
	if gi.RedactionApplied {
		t.Errorf("expected RedactionApplied=false, got true")
	}
	if gi.RedactedFields != nil {
		t.Errorf("expected nil RedactedFields, got %v", gi.RedactedFields)
	}
	if gi.Role != "" {
		t.Errorf("expected empty Role, got %q", gi.Role)
	}
}
