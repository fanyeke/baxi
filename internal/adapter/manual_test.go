package adapter

import (
	"context"
	"testing"

	"baxi/internal/action"
)

func TestManualAdapter_DryRunTrue(t *testing.T) {
	ctx := context.Background()
	adapter := NewManualAdapter(ManualConfig{Enabled: true})

	proposal := action.ActionProposal{
		ProposalID: "prop-001",
		CaseID:     "case-001",
		ActionType: "notify_owner",
		Payload: map[string]interface{}{
			"rule_id": "rule-42",
		},
	}

	result, err := adapter.Execute(ctx, proposal, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
	if !result.DryRun {
		t.Error("expected DryRun=true")
	}
	if result.DispatchPayload == nil {
		t.Fatal("expected DispatchPayload to be non-nil")
	}
	if ch, ok := result.DispatchPayload["channel"]; !ok || ch != "manual" {
		t.Errorf("expected channel=manual, got %v", ch)
	}
	if msg, ok := result.DispatchPayload["message"]; !ok || msg != "Event queued for manual review: rule=rule-42" {
		t.Errorf("unexpected message: %v", msg)
	}
}

func TestManualAdapter_DryRunTrue_NoRuleID(t *testing.T) {
	ctx := context.Background()
	adapter := NewManualAdapter(ManualConfig{Enabled: true})

	proposal := action.ActionProposal{
		ProposalID: "prop-002",
		CaseID:     "case-002",
		ActionType: "export_report",
	}

	result, err := adapter.Execute(ctx, proposal, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
	if !result.DryRun {
		t.Error("expected DryRun=true")
	}
	if result.DispatchPayload == nil {
		t.Fatal("expected DispatchPayload to be non-nil")
	}
	if rid, ok := result.DispatchPayload["rule_id"]; !ok || rid != "unknown" {
		t.Errorf("expected rule_id=unknown, got %v", rid)
	}
}

func TestManualAdapter_DryRunTrue_NilPayload(t *testing.T) {
	ctx := context.Background()
	adapter := NewManualAdapter(ManualConfig{Enabled: true})

	proposal := action.ActionProposal{
		ProposalID: "prop-003",
		CaseID:     "case-003",
		ActionType: "create_outbox_message",
	}

	result, err := adapter.Execute(ctx, proposal, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
	if !result.DryRun {
		t.Error("expected DryRun=true")
	}
	if rid, ok := result.DispatchPayload["rule_id"]; !ok || rid != "unknown" {
		t.Errorf("expected rule_id=unknown, got %v", rid)
	}
}

func TestManualAdapter_ExecuteSuccess(t *testing.T) {
	ctx := context.Background()
	adapter := NewManualAdapter(ManualConfig{Enabled: true})

	proposal := action.ActionProposal{
		ProposalID: "prop-004",
		CaseID:     "case-004",
		ActionType: "create_followup_task",
		Title:      "Review this task",
		Payload: map[string]interface{}{
			"rule_id": "rule-99",
		},
	}

	result, err := adapter.Execute(ctx, proposal, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
	if result.DryRun {
		t.Error("expected DryRun=false")
	}
	if result.DispatchPayload == nil {
		t.Fatal("expected DispatchPayload to be non-nil")
	}
	if ch, ok := result.DispatchPayload["channel"]; !ok || ch != "manual" {
		t.Errorf("expected channel=manual, got %v", ch)
	}
	if msg, ok := result.DispatchPayload["message"]; !ok || msg != "Event queued for manual review: rule=rule-99" {
		t.Errorf("unexpected message: %v", msg)
	}
}

func TestManualAdapter_ExecuteSuccess_NoRuleID(t *testing.T) {
	ctx := context.Background()
	adapter := NewManualAdapter(ManualConfig{Enabled: true})

	proposal := action.ActionProposal{
		ProposalID: "prop-005",
		CaseID:     "case-005",
		ActionType: "unknown_action",
		Payload: map[string]interface{}{
			"other_key": "other_value",
		},
	}

	result, err := adapter.Execute(ctx, proposal, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
	if result.DryRun {
		t.Error("expected DryRun=false")
	}
	if rid, ok := result.DispatchPayload["rule_id"]; !ok || rid != "unknown" {
		t.Errorf("expected rule_id=unknown, got %v", rid)
	}
}

func TestManualAdapter_ExecuteSuccess_NilPayload(t *testing.T) {
	ctx := context.Background()
	adapter := NewManualAdapter(ManualConfig{Enabled: true})

	proposal := action.ActionProposal{
		ProposalID: "prop-006",
		CaseID:     "case-006",
		ActionType: "notify_owner",
	}

	result, err := adapter.Execute(ctx, proposal, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
	if rid, ok := result.DispatchPayload["rule_id"]; !ok || rid != "unknown" {
		t.Errorf("expected rule_id=unknown, got %v", rid)
	}
}

func TestManualAdapter_ExecuteSuccess_EmptyRuleID(t *testing.T) {
	ctx := context.Background()
	adapter := NewManualAdapter(ManualConfig{Enabled: true})

	proposal := action.ActionProposal{
		ProposalID: "prop-007",
		CaseID:     "case-007",
		ActionType: "export_report",
		Payload: map[string]interface{}{
			"rule_id": "",
		},
	}

	result, err := adapter.Execute(ctx, proposal, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
	if rid, ok := result.DispatchPayload["rule_id"]; !ok || rid != "unknown" {
		t.Errorf("expected rule_id=unknown for empty string, got %v", rid)
	}
}

func TestManualAdapter_ExecuteSuccess_NonStringRuleID(t *testing.T) {
	ctx := context.Background()
	adapter := NewManualAdapter(ManualConfig{Enabled: true})

	proposal := action.ActionProposal{
		ProposalID: "prop-008",
		CaseID:     "case-008",
		ActionType: "create_outbox_message",
		Payload: map[string]interface{}{
			"rule_id": 12345,
		},
	}

	result, err := adapter.Execute(ctx, proposal, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
	if rid, ok := result.DispatchPayload["rule_id"]; !ok || rid != "unknown" {
		t.Errorf("expected rule_id=unknown for non-string value, got %v", rid)
	}
}

func TestExtractRuleID(t *testing.T) {
	tests := []struct {
		name    string
		payload map[string]interface{}
		want    string
	}{
		{"nil payload", nil, "unknown"},
		{"empty payload", map[string]interface{}{}, "unknown"},
		{"valid rule_id", map[string]interface{}{"rule_id": "rule-1"}, "rule-1"},
		{"empty string rule_id", map[string]interface{}{"rule_id": ""}, "unknown"},
		{"non-string rule_id", map[string]interface{}{"rule_id": 42}, "unknown"},
		{"bool rule_id", map[string]interface{}{"rule_id": true}, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRuleID(tt.payload)
			if got != tt.want {
				t.Errorf("extractRuleID() = %q, want %q", got, tt.want)
			}
		})
	}
}
