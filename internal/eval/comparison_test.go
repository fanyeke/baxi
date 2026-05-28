package eval

import (
	"testing"

	"baxi/internal/llm"
)

func TestCompareIdentical(t *testing.T) {
	actions := []llm.RecommendedAction{
		{ActionType: llm.ActionTypeNotifyOwner, Priority: "high", OwnerRole: "ops"},
		{ActionType: llm.ActionTypeCreateOutboxMessage, Priority: "high", OwnerRole: "ops"},
	}
	output := &llm.DecisionOutput{
		DecisionType:       llm.DecisionTypeInvestigate,
		Severity:           llm.SeverityMedium,
		Summary:            "identical decision",
		Rationale:          []string{"reason"},
		RecommendedActions: actions,
		Confidence:         0.85,
		RequiresHumanReview: true,
	}

	result := Compare("case-identical", output, output)

	if !result.DecisionTypeMatch {
		t.Error("expected DecisionTypeMatch=true for identical decisions")
	}
	if !result.SeverityMatch {
		t.Error("expected SeverityMatch=true for identical decisions")
	}
	if result.ActionOverlap != 1.0 {
		t.Errorf("expected ActionOverlap=1.0, got %.2f", result.ActionOverlap)
	}
	if result.ConfidenceDiff != 0 {
		t.Errorf("expected ConfidenceDiff=0, got %.2f", result.ConfidenceDiff)
	}
	if result.LLMDecisionType != llm.DecisionTypeInvestigate {
		t.Errorf("expected LLMDecisionType=%q, got %q", llm.DecisionTypeInvestigate, result.LLMDecisionType)
	}
	if result.RuleDecisionType != llm.DecisionTypeInvestigate {
		t.Errorf("expected RuleDecisionType=%q, got %q", llm.DecisionTypeInvestigate, result.RuleDecisionType)
	}
	if result.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
}

func TestCompareDifferentTypes(t *testing.T) {
	llmDecision := &llm.DecisionOutput{
		DecisionType: llm.DecisionTypeInvestigate,
		Severity:     llm.SeverityMedium,
		Confidence:   0.85,
	}
	ruleDecision := &llm.DecisionOutput{
		DecisionType: llm.DecisionTypeMonitor,
		Severity:     llm.SeverityLow,
		Confidence:   0.60,
	}

	result := Compare("case-diff-types", llmDecision, ruleDecision)

	if result.DecisionTypeMatch {
		t.Error("expected DecisionTypeMatch=false for different decision types")
	}
	if result.SeverityMatch {
		t.Error("expected SeverityMatch=false for different severities")
	}
	if result.LLMDecisionType != llm.DecisionTypeInvestigate {
		t.Errorf("expected LLMDecisionType=%q, got %q", llm.DecisionTypeInvestigate, result.LLMDecisionType)
	}
	if result.RuleDecisionType != llm.DecisionTypeMonitor {
		t.Errorf("expected RuleDecisionType=%q, got %q", llm.DecisionTypeMonitor, result.RuleDecisionType)
	}
	if result.ConfidenceDiff != 0.25 {
		t.Errorf("expected ConfidenceDiff=0.25, got %.2f", result.ConfidenceDiff)
	}
}

func TestCompareDifferentActions(t *testing.T) {
	llmDecision := &llm.DecisionOutput{
		DecisionType: llm.DecisionTypeInvestigate,
		Severity:     llm.SeverityMedium,
		RecommendedActions: []llm.RecommendedAction{
			{ActionType: llm.ActionTypeNotifyOwner, Priority: "high", OwnerRole: "ops"},
			{ActionType: llm.ActionTypeCreateOutboxMessage, Priority: "high", OwnerRole: "ops"},
		},
	}
	ruleDecision := &llm.DecisionOutput{
		DecisionType: llm.DecisionTypeInvestigate,
		Severity:     llm.SeverityMedium,
		RecommendedActions: []llm.RecommendedAction{
			{ActionType: llm.ActionTypeNotifyOwner, Priority: "low", OwnerRole: "analyst"},
			{ActionType: llm.ActionTypeCreateFollowupTask, Priority: "medium", OwnerRole: "analyst"},
		},
	}

	result := Compare("case-diff-actions", llmDecision, ruleDecision)

	if result.ActionOverlap >= 1.0 {
		t.Errorf("expected ActionOverlap < 1.0 for different action sets, got %.2f", result.ActionOverlap)
	}
	if result.ActionOverlap <= 0 {
		t.Errorf("expected ActionOverlap > 0 (one shared action), got %.2f", result.ActionOverlap)
	}
	// NotifyOwner is shared between both: intersection=1, union=3 → 1/3
	if result.ActionOverlap != 1.0/3.0 {
		t.Errorf("expected ActionOverlap=0.333..., got %.4f", result.ActionOverlap)
	}
}

func TestCompareEmptyActions(t *testing.T) {
	llmDecision := &llm.DecisionOutput{
		DecisionType:       llm.DecisionTypeMonitor,
		Severity:           llm.SeverityLow,
		RecommendedActions: []llm.RecommendedAction{},
	}
	ruleDecision := &llm.DecisionOutput{
		DecisionType:       llm.DecisionTypeMonitor,
		Severity:           llm.SeverityLow,
		RecommendedActions: []llm.RecommendedAction{},
	}

	result := Compare("case-empty-actions", llmDecision, ruleDecision)

	if result.ActionOverlap != 1.0 {
		t.Errorf("expected ActionOverlap=1.0 for both empty, got %.2f", result.ActionOverlap)
	}
}

func TestJaccardIndex(t *testing.T) {
	tests := []struct {
		name  string
		a     []llm.RecommendedAction
		b     []llm.RecommendedAction
		want  float64
		label string
	}{
		{
			name: "identical sets",
			a: []llm.RecommendedAction{
				{ActionType: "notify", Priority: "high"},
				{ActionType: "alert", Priority: "low"},
			},
			b: []llm.RecommendedAction{
				{ActionType: "notify", Priority: "low"}, // different priority, same ActionType
				{ActionType: "alert", Priority: "high"},
			},
			want:  1.0,
			label: "identical",
		},
		{
			name: "disjoint sets",
			a: []llm.RecommendedAction{
				{ActionType: "action_a"},
			},
			b: []llm.RecommendedAction{
				{ActionType: "action_b"},
			},
			want:  0.0,
			label: "disjoint",
		},
		{
			name: "partial overlap",
			a: []llm.RecommendedAction{
				{ActionType: "shared"},
				{ActionType: "a_only"},
			},
			b: []llm.RecommendedAction{
				{ActionType: "shared"},
				{ActionType: "b_only"},
				{ActionType: "extra"},
			},
			want:  1.0 / 4.0,
			label: "partial",
		},
		{
			name:  "both empty",
			a:     []llm.RecommendedAction{},
			b:     []llm.RecommendedAction{},
			want:  1.0,
			label: "both empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jaccardIndex(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("jaccardIndex(%s): expected %.4f, got %.4f", tt.label, tt.want, got)
			}
		})
	}
}

func TestCompare_RequiresReviewFields(t *testing.T) {
	llmDecision := &llm.DecisionOutput{
		DecisionType:       llm.DecisionTypeInvestigate,
		RequiresHumanReview: false,
	}
	ruleDecision := &llm.DecisionOutput{
		DecisionType:       llm.DecisionTypeInvestigate,
		RequiresHumanReview: true,
	}

	result := Compare("case-review", llmDecision, ruleDecision)

	if result.LLMRequiresReview {
		t.Error("expected LLMRequiresReview=false")
	}
	if !result.RuleRequiresReview {
		t.Error("expected RuleRequiresReview=true")
	}
}

func TestCompare_ComparisonJSON(t *testing.T) {
	llmDecision := &llm.DecisionOutput{
		DecisionType: llm.DecisionTypeExperiment,
		Severity:     llm.SeverityHigh,
	}
	ruleDecision := &llm.DecisionOutput{
		DecisionType: llm.DecisionTypeMonitor,
		Severity:     llm.SeverityLow,
	}

	result := Compare("case-json", llmDecision, ruleDecision)

	if len(result.ComparisonJSON) == 0 {
		t.Fatal("expected non-empty ComparisonJSON")
	}

	if result.ComparisonJSON[0] != '{' {
		t.Error("expected ComparisonJSON to start with '{'")
	}
}
