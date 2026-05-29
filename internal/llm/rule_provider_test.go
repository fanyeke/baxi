package llm

import (
	"context"
	"testing"
)

func TestRuleBasedProvider_SatisfiesInterface(t *testing.T) {
	var _ DecisionProvider = (*RuleBasedProvider)(nil)
}

func TestRuleBasedProvider_GenerateDecision(t *testing.T) {
	tests := []struct {
		name        string
		severity    string
		wantType    string
		wantConf    float64
		wantActions int
		wantSummary string
	}{
		{
			name:        "critical severity escalates to human",
			severity:    SeverityCritical,
			wantType:    ActionTypeEscalateToHuman,
			wantConf:    0.95,
			wantActions: 2,
			wantSummary: "Alert for test_metric triggered with critical severity. Current value 50 vs baseline 100 (delta -50%).",
		},
		{
			name:        "high severity escalates to human",
			severity:    SeverityHigh,
			wantType:    ActionTypeEscalateToHuman,
			wantConf:    0.85,
			wantActions: 2,
			wantSummary: "Alert for test_metric triggered with high severity. Current value 50 vs baseline 100 (delta -50%).",
		},
		{
			name:        "medium severity investigates",
			severity:    SeverityMedium,
			wantType:    DecisionTypeInvestigate,
			wantConf:    0.72,
			wantActions: 2,
			wantSummary: "Moderate deviation detected for test_metric.",
		},
		{
			name:        "low severity monitors only",
			severity:    SeverityLow,
			wantType:    DecisionTypeMonitor,
			wantConf:    0.60,
			wantActions: 1,
			wantSummary: "Slight deviation detected for test_metric. No action required.",
		},
		{
			name:        "unknown severity investigates",
			severity:    "unknown",
			wantType:    DecisionTypeInvestigate,
			wantConf:    0.50,
			wantActions: 1,
			wantSummary: "Anomaly detected for test_metric with unknown severity.",
		},
	}

	ctx := context.Background()
	p := NewRuleBasedProvider()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := LLMSafeContext{
				CaseID: "case-001",
				Trigger: TriggerInfo{
					AlertID:       "alert-001",
					RuleID:        "rule-001",
					Severity:      tt.severity,
					MetricName:    "test_metric",
					CurrentValue:  50,
					BaselineValue: 100,
					DeltaPct:      -50,
				},
			}

			out, err := p.GenerateDecision(ctx, input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out == nil {
				t.Fatal("expected non-nil DecisionOutput")
			}

			if out.DecisionType != tt.wantType {
				t.Errorf("DecisionType = %q, want %q", out.DecisionType, tt.wantType)
			}
			if out.Severity != tt.severity {
				t.Errorf("Severity = %q, want %q", out.Severity, tt.severity)
			}
			if out.Confidence != tt.wantConf {
				t.Errorf("Confidence = %v, want %v", out.Confidence, tt.wantConf)
			}
			if len(out.RecommendedActions) != tt.wantActions {
				t.Errorf("len(RecommendedActions) = %d, want %d", len(out.RecommendedActions), tt.wantActions)
			}
			if out.Summary != tt.wantSummary {
				t.Errorf("Summary = %q, want %q", out.Summary, tt.wantSummary)
			}
			if !out.RequiresHumanReview {
				t.Errorf("RequiresHumanReview = false, want true")
			}
			if len(out.Rationale) == 0 {
				t.Errorf("Rationale is empty, expected non-empty")
			}
		})
	}
}

func TestRuleBasedProvider_AllDecisionsRequireHumanReview(t *testing.T) {
	ctx := context.Background()
	p := NewRuleBasedProvider()
	severities := []string{SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow, "unknown"}

	for _, s := range severities {
		t.Run(s, func(t *testing.T) {
			input := LLMSafeContext{
				CaseID: "case-review",
				Trigger: TriggerInfo{
					Severity:   s,
					MetricName: "cpu_usage",
				},
			}
			out, err := p.GenerateDecision(ctx, input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !out.RequiresHumanReview {
				t.Errorf("RequiresHumanReview = false for severity %q", s)
			}
		})
	}
}

func TestRuleBasedProvider_AllConfidencesInRange(t *testing.T) {
	ctx := context.Background()
	p := NewRuleBasedProvider()
	severities := []string{SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow, "unknown"}

	for _, s := range severities {
		t.Run(s, func(t *testing.T) {
			input := LLMSafeContext{
				CaseID: "case-conf",
				Trigger: TriggerInfo{
					Severity:   s,
					MetricName: "memory_usage",
				},
			}
			out, err := p.GenerateDecision(ctx, input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out.Confidence < 0 || out.Confidence > 1 {
				t.Errorf("Confidence = %v is outside [0,1] for severity %q", out.Confidence, s)
			}
		})
	}
}

func TestRuleBasedProvider_AllActionsAreAllowed(t *testing.T) {
	allowed := map[string]bool{
		ActionTypeCreateFollowupTask: true,
		ActionTypeNotifyOwner:        true,
		ActionTypeExportReport:       true,
		ActionTypeEscalateToHuman:    true,
	}

	ctx := context.Background()
	p := NewRuleBasedProvider()
	severities := []string{SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow, "unknown"}

	for _, s := range severities {
		t.Run(s, func(t *testing.T) {
			input := LLMSafeContext{
				CaseID: "case-actions",
				Trigger: TriggerInfo{
					Severity:   s,
					MetricName: "disk_io",
				},
			}
			out, err := p.GenerateDecision(ctx, input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for i, a := range out.RecommendedActions {
				if !allowed[a.ActionType] {
					t.Errorf("RecommendedActions[%d].ActionType = %q is not in allowed set", i, a.ActionType)
				}
			}
		})
	}
}
