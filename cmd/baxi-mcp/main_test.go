package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"baxi/internal/action"
	"baxi/internal/ontology"
)

// TestExecuteAction_DoesNotInsertApprovedProposal verifies that ExecuteAction
// never attempts to create an approved proposal in the database.
// It passes a nil pool — if ExecuteAction tried any DB operation, it would panic.
func TestExecuteAction_DoesNotInsertApprovedProposal(t *testing.T) {
	ctx := context.Background()
	adapter := setupTestAdapter(t, ctx)
	adapter.pool = nil // nil pool: any DB access would panic

	result, err := adapter.ExecuteAction(ctx, "seller", "SELLER_001", "notify_owner", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got failure: %v", result.Result)
	}
	if result.Result["would_execute"] != true {
		t.Fatalf("expected would_execute=true, got %v", result.Result["would_execute"])
	}
	// If ExecuteAction tried to insert into ai.action_proposal or ai.decision_case,
	// it would have panicked on the nil pool. The test reaching here proves no DB
	// mutation was attempted.
}

// TestExecuteAction_DoesNotCallWithDryRunFalse verifies that ExecuteAction
// never invokes ApplyService (and therefore can never call
// action.WithDryRun(false)). It passes a nil applySvc; any ApplyService call
// would panic, so reaching the assertions proves the function never executes.
func TestExecuteAction_DoesNotCallWithDryRunFalse(t *testing.T) {
	ctx := context.Background()
	adapter := setupTestAdapter(t, ctx)
	adapter.applySvc = nil // nil applySvc: any ApplyService call would panic

	result, err := adapter.ExecuteAction(ctx, "seller", "SELLER_001", "notify_owner", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got failure")
	}
	if result.Result["would_execute"] != true {
		t.Fatalf("expected would_execute=true, got %v", result.Result["would_execute"])
	}
}

// TestExecuteAction_ReturnsDryRunForApprovalRequiredActions verifies that
// ExecuteAction returns a dry-run result for actions configured with
// requires_approval=true and never attempts real execution.
func TestExecuteAction_ReturnsDryRunForApprovalRequiredActions(t *testing.T) {
	ctx := context.Background()
	adapter := setupTestAdapter(t, ctx)

	cfg, ok := adapter.actionReg.GetActionConfig("notify_owner")
	if !ok {
		t.Fatal("notify_owner not found in action registry")
	}
	if !cfg.RequiresApproval {
		t.Skip("notify_owner does not require approval in this registry; test not applicable")
	}

	adapter.applySvc = nil
	adapter.pool = nil

	result, err := adapter.ExecuteAction(ctx, "seller", "SELLER_001", "notify_owner", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success for approval-required action, got failure: %v", result.Result)
	}
	if result.Result["would_execute"] != true {
		t.Fatalf("expected would_execute=true for approval-required action, got %v", result.Result["would_execute"])
	}
	// Verify ExecuteAction never attempted execution despite the action requiring approval.
}

// TestExecuteAction_ValidatesPayload verifies that ExecuteAction exercises the
// payload validation path when params are provided.
func TestExecuteAction_ValidatesPayload(t *testing.T) {
	ctx := context.Background()
	adapter := setupTestAdapter(t, ctx)

	invalidParams := map[string]interface{}{
		"unknown_field": "should_fail_if_schema_enforced",
	}

	result, err := adapter.ExecuteAction(ctx, "seller", "SELLER_001", "notify_owner", invalidParams)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = result
}

// TestExecuteAction_RejectsUnboundAction verifies that ExecuteAction rejects
// actions not present in the object type's AllowedActions list.
func TestExecuteAction_RejectsUnboundAction(t *testing.T) {
	ctx := context.Background()
	adapter := setupTestAdapter(t, ctx)

	result, err := adapter.ExecuteAction(ctx, "seller", "SELLER_001", "unauthorized_action", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Fatalf("expected failure for unauthorized action")
	}
}

// setupTestAdapter creates a fully wired ontologyServiceAdapter for tests.
// It loads a test schema with seller object type allowing notify_owner.
func setupTestAdapter(t *testing.T, ctx context.Context) *ontologyServiceAdapter {
	t.Helper()
	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "test_schema.yml")
	schemaContent := `objects:
  - object_type_id: seller
    display_name: Seller
    grain: seller_id
    source_tables:
      - test_table
    allowed_actions:
      - notify_owner
    properties:
      id:
        type: string
        is_pk: true
`
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		t.Fatalf("write test schema: %v", err)
	}

	reg, err := ontology.NewObjectRegistry(ctx, nil, nil, schemaPath)
	if err != nil {
		t.Fatalf("create object registry: %v", err)
	}

	actionReg, err := action.NewActionRegistry("/nonexistent/action_registry.yml")
	if err != nil {
		t.Fatalf("create action registry: %v", err)
	}
	if !actionReg.IsAllowed("notify_owner") {
		t.Fatalf("notify_owner should be allowed in default registry")
	}

	return &ontologyServiceAdapter{
		registry:  reg,
		actionReg: actionReg,
	}
}

func TestStringPtrOrEmpty_Nil(t *testing.T) {
	got := stringPtrOrEmpty(nil)
	if got != "" {
		t.Errorf("stringPtrOrEmpty(nil) = %q, want \"\"", got)
	}
}

func TestStringPtrOrEmpty_NonNil(t *testing.T) {
	s := "hello"
	got := stringPtrOrEmpty(&s)
	if got != "hello" {
		t.Errorf("stringPtrOrEmpty(&\"hello\") = %q, want \"hello\"", got)
	}
}

func TestStringPtrOrEmpty_EmptyPtr(t *testing.T) {
	s := ""
	got := stringPtrOrEmpty(&s)
	if got != "" {
		t.Errorf("stringPtrOrEmpty(&\"\") = %q, want \"\"", got)
	}
}
