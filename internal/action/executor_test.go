package action

import (
	"context"
	"testing"
)

func TestNoOpExecutor_DryRunTrue(t *testing.T) {
	ctx := context.Background()
	exec := NewNoOpExecutor()

	proposal := ActionProposal{
		ProposalID: "prop-001",
		CaseID:     "case-001",
		ActionType: "notify_owner",
	}

	result, err := exec.Execute(ctx, proposal, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
	if !result.DryRun {
		t.Error("expected DryRun=true when dryRun=true")
	}
	if result.Error != "" {
		t.Errorf("unexpected error string: %s", result.Error)
	}
}

func TestNoOpExecutor_DryRunFalse(t *testing.T) {
	ctx := context.Background()
	exec := NewNoOpExecutor()

	proposal := ActionProposal{
		ProposalID: "prop-002",
		CaseID:     "case-002",
		ActionType: "export_report",
	}

	result, err := exec.Execute(ctx, proposal, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
	if result.DryRun {
		t.Error("expected DryRun=false when dryRun=false")
	}
	if result.Error != "" {
		t.Errorf("unexpected error string: %s", result.Error)
	}
}

func TestNoOpExecutor_DispatchPayload(t *testing.T) {
	ctx := context.Background()
	exec := NewNoOpExecutor()

	proposal := ActionProposal{
		ProposalID: "prop-003",
		CaseID:     "case-003",
		ActionType: "create_outbox_message",
	}

	result, err := exec.Execute(ctx, proposal, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.DispatchPayload == nil {
		t.Fatal("expected DispatchPayload to be non-nil")
	}

	tests := []struct {
		key      string
		expected interface{}
	}{
		{"action_type", "create_outbox_message"},
		{"proposal_id", "prop-003"},
		{"case_id", "case-003"},
		{"dry_run", true},
	}

	for _, tt := range tests {
		got, ok := result.DispatchPayload[tt.key]
		if !ok {
			t.Errorf("expected DispatchPayload to contain key %q", tt.key)
			continue
		}
		if got != tt.expected {
			t.Errorf("DispatchPayload[%q] = %v, want %v", tt.key, got, tt.expected)
		}
	}
}

// Compile-time check: *NoOpExecutor must satisfy ActionExecutor.
var _ ActionExecutor = (*NoOpExecutor)(nil)
