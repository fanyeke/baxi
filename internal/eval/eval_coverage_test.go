package eval

import (
	"context"
	"encoding/json"
	"testing"

	"baxi/internal/llm"
)

// ──── computeDecisionDiff ──────────────────────────────────────────────────

func TestComputeDecisionDiff_BothNil(t *testing.T) {
	diff := computeDecisionDiff(nil, nil)
	if diff != nil {
		t.Errorf("expected nil diff for nil inputs, got %v", diff)
	}
}

func TestComputeDecisionDiff_OriginalNil(t *testing.T) {
	diff := computeDecisionDiff(nil, &llm.DecisionOutput{})
	if diff != nil {
		t.Errorf("expected nil diff when original is nil, got %v", diff)
	}
}

func TestComputeDecisionDiff_ReplayedNil(t *testing.T) {
	diff := computeDecisionDiff(&llm.DecisionOutput{}, nil)
	if diff != nil {
		t.Errorf("expected nil diff when replayed is nil, got %v", diff)
	}
}

func TestComputeDecisionDiff_Identical(t *testing.T) {
	original := &llm.DecisionOutput{
		DecisionType: llm.DecisionTypeInvestigate,
		Severity:     llm.SeverityMedium,
		Confidence:   0.8,
		Summary:      "same summary",
		Rationale:    []string{"reason1"},
		RecommendedActions: []llm.RecommendedAction{
			{ActionType: llm.ActionTypeNotifyOwner},
		},
	}
	replayed := &llm.DecisionOutput{
		DecisionType: llm.DecisionTypeInvestigate,
		Severity:     llm.SeverityMedium,
		Confidence:   0.8,
		Summary:      "same summary",
		Rationale:    []string{"reason1"},
		RecommendedActions: []llm.RecommendedAction{
			{ActionType: llm.ActionTypeNotifyOwner},
		},
	}

	diff := computeDecisionDiff(original, replayed)
	if diff == nil {
		t.Fatal("expected non-nil diff")
	}
	if !diff.DecisionTypeMatch {
		t.Error("expected DecisionTypeMatch=true")
	}
	if !diff.SeverityMatch {
		t.Error("expected SeverityMatch=true")
	}
	if diff.ConfidenceDiff != 0 {
		t.Errorf("expected ConfidenceDiff=0, got %f", diff.ConfidenceDiff)
	}
	if diff.SummaryChanged {
		t.Error("expected SummaryChanged=false")
	}
	if diff.RationaleChanged {
		t.Error("expected RationaleChanged=false")
	}
}

func TestComputeDecisionDiff_AllDifferent(t *testing.T) {
	original := &llm.DecisionOutput{
		DecisionType: llm.DecisionTypeMonitor,
		Severity:     llm.SeverityLow,
		Confidence:   0.3,
		Summary:      "original",
		Rationale:    []string{"r1"},
		RecommendedActions: []llm.RecommendedAction{
			{ActionType: llm.ActionTypeNotifyOwner},
		},
	}
	replayed := &llm.DecisionOutput{
		DecisionType: llm.DecisionTypeInvestigate,
		Severity:     llm.SeverityHigh,
		Confidence:   0.9,
		Summary:      "replayed",
		Rationale:    []string{"r2"},
		RecommendedActions: []llm.RecommendedAction{
			{ActionType: llm.ActionTypeCreateFollowupTask},
		},
	}

	diff := computeDecisionDiff(original, replayed)
	if diff == nil {
		t.Fatal("expected non-nil diff")
	}
	if diff.DecisionTypeMatch {
		t.Error("expected DecisionTypeMatch=false")
	}
	if diff.SeverityMatch {
		t.Error("expected SeverityMatch=false")
	}
	if diff.ConfidenceDiff < 0.599 || diff.ConfidenceDiff > 0.601 {
		t.Errorf("expected ConfidenceDiff ~0.6, got %f", diff.ConfidenceDiff)
	}
	if !diff.SummaryChanged {
		t.Error("expected SummaryChanged=true")
	}
	if !diff.RationaleChanged {
		t.Error("expected RationaleChanged=true")
	}
}

func TestComputeDecisionDiff_EmptyActions_BothEmpty(t *testing.T) {
	original := &llm.DecisionOutput{
		RecommendedActions: []llm.RecommendedAction{},
	}
	replayed := &llm.DecisionOutput{
		RecommendedActions: []llm.RecommendedAction{},
	}

	diff := computeDecisionDiff(original, replayed)
	if diff == nil {
		t.Fatal("expected non-nil diff")
	}
	// Both empty = 1.0 overlap
	if diff.ActionOverlap != 1.0 {
		t.Errorf("expected ActionOverlap=1.0 for empty actions, got %f", diff.ActionOverlap)
	}
}

func TestComputeDecisionDiff_PartialActionOverlap(t *testing.T) {
	original := &llm.DecisionOutput{
		RecommendedActions: []llm.RecommendedAction{
			{ActionType: llm.ActionTypeNotifyOwner},
			{ActionType: llm.ActionTypeCreateFollowupTask},
		},
	}
	replayed := &llm.DecisionOutput{
		RecommendedActions: []llm.RecommendedAction{
			{ActionType: llm.ActionTypeNotifyOwner},
			{ActionType: llm.ActionTypeExportReport},
		},
	}

	diff := computeDecisionDiff(original, replayed)
	if diff == nil {
		t.Fatal("expected non-nil diff")
	}
	// Intersection: notify_owner (1), Union: notify_owner + followup + export (3)
	// Jaccard: 1/3 = 0.333...
	if diff.ActionOverlap < 0.33 || diff.ActionOverlap > 0.34 {
		t.Errorf("expected ActionOverlap ~0.333, got %f", diff.ActionOverlap)
	}
}

// ──── ReplayLegacy ─────────────────────────────────────────────────────────

func TestReplayLegacy_DryRun(t *testing.T) {
	repo := &mockDecisionRepository{
		data: &ReplayData{
			CaseID:   "case-legacy",
			InputContext: mustMarshal(t, llm.LLMSafeContext{
				CaseID: "case-legacy",
			}),
			OriginalOutput: &llm.DecisionOutput{
				DecisionType: llm.DecisionTypeInvestigate,
				Severity:     llm.SeverityMedium,
			},
			Provider:      "rule_based",
			Model:         "rule_based",
			PromptVersion: "v1",
			ContextHash:   "abc123",
		},
	}
	logger := &llm.NoOpAuditLogger{}
	svc := NewReplayService(repo, nil, logger)

	result, err := svc.ReplayLegacy(nil, "case-legacy", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.DryRun {
		t.Error("expected DryRun=true")
	}
	if result.OriginalDecision == nil {
		t.Error("expected non-nil OriginalDecision")
	}
}

func TestReplayLegacy_NotDryRun(t *testing.T) {
	repo := &mockDecisionRepository{
		data: &ReplayData{
			CaseID:   "case-legacy2",
			InputContext: mustMarshal(t, llm.LLMSafeContext{
				CaseID: "case-legacy2",
			}),
			OriginalOutput: &llm.DecisionOutput{
				DecisionType: llm.DecisionTypeInvestigate,
				Severity:     llm.SeverityMedium,
			},
			Provider:      "rule_based",
			Model:         "rule_based",
			PromptVersion: "v1",
			ContextHash:   "abc123",
		},
	}
	provider := llm.NewRuleBasedProvider()
	logger := &llm.NoOpAuditLogger{}
	svc := NewReplayService(repo, provider, logger)

	result, err := svc.ReplayLegacy(nil, "case-legacy2", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.DryRun {
		t.Error("expected DryRun=false")
	}
	if result.ReplayedDecision == nil {
		t.Error("expected non-nil ReplayedDecision")
	}
	if result.Diff == nil {
		t.Error("expected non-nil Diff")
	}
}

// ──── Replay with context hash mismatch ────────────────────────────────────

func TestReplay_ContextHashMismatch(t *testing.T) {
	repo := &mockDecisionRepository{
		data: &ReplayData{
			CaseID:       "case-hash",
			InputContext: mustMarshal(t, llm.LLMSafeContext{}),
			ContextHash:  "actual_hash",
		},
	}
	logger := &llm.NoOpAuditLogger{}
	svc := NewReplayService(repo, nil, logger)

	_, err := svc.Replay(nil, "case-hash", ReplayOptions{
		ContextHash: "expected_hash",
		DryRun:      true,
	})
	if err == nil {
		t.Fatal("expected error for context hash mismatch")
	}
}

// ──── NewPGReplayRepository ────────────────────────────────────────────────

func TestNewPGReplayRepository(t *testing.T) {
	repo := NewPGReplayRepository(nil)
	if repo == nil {
		t.Fatal("expected non-nil repo")
	}
}

// ──── Jaccard index edge cases ────────────────────────────────────────────

func TestJaccardIndex_BothEmpty(t *testing.T) {
	result := jaccardIndex([]llm.RecommendedAction{}, []llm.RecommendedAction{})
	if result != 1.0 {
		t.Errorf("expected 1.0 for both empty, got %f", result)
	}
}

func TestJaccardIndex_OneEmpty(t *testing.T) {
	a := []llm.RecommendedAction{{ActionType: "notify_owner"}}
	b := []llm.RecommendedAction{}
	result := jaccardIndex(a, b)
	if result != 0.0 {
		t.Errorf("expected 0.0 for one empty, got %f", result)
	}
}

func TestJaccardIndex_CompleteOverlap(t *testing.T) {
	a := []llm.RecommendedAction{{ActionType: "notify_owner"}, {ActionType: "export_report"}}
	b := []llm.RecommendedAction{{ActionType: "notify_owner"}, {ActionType: "export_report"}}
	result := jaccardIndex(a, b)
	if result != 1.0 {
		t.Errorf("expected 1.0 for complete overlap, got %f", result)
	}
}

func TestJaccardIndex_NoOverlap(t *testing.T) {
	a := []llm.RecommendedAction{{ActionType: "notify_owner"}}
	b := []llm.RecommendedAction{{ActionType: "export_report"}}
	result := jaccardIndex(a, b)
	if result != 0.0 {
		t.Errorf("expected 0.0 for no overlap, got %f", result)
	}
}

// ──── DecisionComparison edge cases ────────────────────────────────────────

func TestCompare_NilComparisonJSON(t *testing.T) {
	llmDecision := &llm.DecisionOutput{
		DecisionType: llm.DecisionTypeInvestigate,
		Severity:     llm.SeverityMedium,
		Confidence:   0.8,
	}
	ruleDecision := &llm.DecisionOutput{
		DecisionType: llm.DecisionTypeInvestigate,
		Severity:     llm.SeverityMedium,
		Confidence:   0.6,
	}

	c := Compare("case-comp", llmDecision, ruleDecision)
	if c == nil {
		t.Fatal("expected non-nil comparison")
	}
	if !c.DecisionTypeMatch {
		t.Error("expected DecisionTypeMatch=true")
	}
	if !c.SeverityMatch {
		t.Error("expected SeverityMatch=true")
	}
	if c.ConfidenceDiff < 0.199 || c.ConfidenceDiff > 0.201 {
		t.Errorf("expected ConfidenceDiff ~0.2, got %f", c.ConfidenceDiff)
	}
	if c.ComparisonJSON == nil {
		t.Error("expected non-nil ComparisonJSON")
	}
}

func TestCompare_DifferentTypes(t *testing.T) {
	llmDecision := &llm.DecisionOutput{
		DecisionType: llm.DecisionTypeMonitor,
		Severity:     llm.SeverityLow,
	}
	ruleDecision := &llm.DecisionOutput{
		DecisionType: llm.DecisionTypeInvestigate,
		Severity:     llm.SeverityHigh,
	}

	c := Compare("case-diff", llmDecision, ruleDecision)
	if c.DecisionTypeMatch {
		t.Error("expected DecisionTypeMatch=false")
	}
	if c.SeverityMatch {
		t.Error("expected SeverityMatch=false")
	}
}

func TestCompare_EmptyLLMDecisionType(t *testing.T) {
	llmDecision := &llm.DecisionOutput{}
	ruleDecision := &llm.DecisionOutput{DecisionType: llm.DecisionTypeMonitor}

	c := Compare("case-empty", llmDecision, ruleDecision)
	if c.LLMDecisionType != "" {
		t.Errorf("expected empty LLMDecisionType, got %s", c.LLMDecisionType)
	}
	if c.RuleDecisionType != llm.DecisionTypeMonitor {
		t.Errorf("expected RuleDecisionType=monitor_only, got %s", c.RuleDecisionType)
	}
}

// ──── EvalResult fields ────────────────────────────────────────────────────

func TestEvalResult_Fields(t *testing.T) {
	result := &EvalResult{
		EvalID:         "eval_001",
		DecisionCaseID: "case-001",
		LLMDecisionID:  "decision-001",
		EvalRuleID:     "all_dimensions",
		EvalStatus:     EvalStatusPass,
		Score:          0.95,
	}
	if result.EvalID != "eval_001" {
		t.Errorf("expected EvalID=eval_001, got %s", result.EvalID)
	}
	if result.EvalStatus != EvalStatusPass {
		t.Errorf("expected EvalStatus=pass, got %s", result.EvalStatus)
	}
}

func TestDimensionCategories(t *testing.T) {
	if len(SafetyDimensions) != 3 {
		t.Errorf("expected 3 safety dimensions, got %d", len(SafetyDimensions))
	}
	if len(GroundingDimensions) != 3 {
		t.Errorf("expected 3 grounding dimensions, got %d", len(GroundingDimensions))
	}
	if len(UsefulnessDimensions) != 2 {
		t.Errorf("expected 2 usefulness dimensions, got %d", len(UsefulnessDimensions))
	}
}

// ──── evaluateDimensions edge cases ────────────────────────────────────────

func TestEvaluateDimensions_ShortSummary(t *testing.T) {
	evaluator := NewDecisionEvaluator(nil)
	output := &llm.DecisionOutput{
		DecisionType:        llm.DecisionTypeInvestigate,
		Severity:            llm.SeverityMedium,
		Summary:             "short",
		Rationale:           []string{"reason"},
		Confidence:          0.8,
		RequiresHumanReview: true,
		RecommendedActions: []llm.RecommendedAction{
			{ActionType: llm.ActionTypeNotifyOwner, OwnerRole: "ops"},
		},
	}

	dims := evaluator.evaluateDimensions(output, validAllowedActions())
	for _, d := range dims {
		if d.ID == "context_grounding" && d.Passed {
			t.Error("expected context_grounding to fail for short summary")
		}
	}
}

func TestEvaluateDimensions_DefaultAction(t *testing.T) {
	evaluator := NewDecisionEvaluator(nil)
	output := &llm.DecisionOutput{
		DecisionType:        llm.DecisionTypeInvestigate,
		Severity:            llm.SeverityMedium,
		Summary:             "Valid summary text",
		Rationale:           []string{"reason"},
		Confidence:          0.8,
		RequiresHumanReview: true,
		RecommendedActions: []llm.RecommendedAction{
			{ActionType: "default", OwnerRole: "ops"},
		},
	}

	dims := evaluator.evaluateDimensions(output, validAllowedActions())
	for _, d := range dims {
		if d.ID == "not_overgeneralized" && d.Passed {
			t.Error("expected not_overgeneralized to fail for 'default' action")
		}
	}
}

func TestEvaluateDimensions_EmptyOwnerRole(t *testing.T) {
	evaluator := NewDecisionEvaluator(nil)
	output := &llm.DecisionOutput{
		DecisionType:        llm.DecisionTypeInvestigate,
		Severity:            llm.SeverityMedium,
		Summary:             "Valid summary text",
		Rationale:           []string{"reason"},
		Confidence:          0.8,
		RequiresHumanReview: true,
		RecommendedActions: []llm.RecommendedAction{
			{ActionType: llm.ActionTypeNotifyOwner, OwnerRole: ""},
		},
	}

	dims := evaluator.evaluateDimensions(output, validAllowedActions())
	for _, d := range dims {
		if d.ID == "has_owner_role" && d.Passed {
			t.Error("expected has_owner_role to fail for empty owner_role")
		}
	}
}

// ──── Mock repository ──────────────────────────────────────────────────────

type mockDecisionRepository struct {
	data *ReplayData
	err  error
}

func (m *mockDecisionRepository) GetLLMDecisionByCaseID(_ context.Context, caseID string) (*ReplayData, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.data, nil
}

func mustMarshal(t *testing.T, v interface{}) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("mustMarshal: %v", err)
	}
	return data
}
