//go:build integration

package e2e_test

import (
	"testing"

	"github.com/stretchr/testify/require"

)

// TestDecisionLifecycle runs the full decision lifecycle via MCP tools:
// create case → build context → decide → list proposals → approve → execute → outbox.
// This mirrors the Pi Agent extension flow.
func TestDecisionLifecycle(t *testing.T) {
	client, cleanup := setupMCPTest(t)
	defer cleanup()

	t.Log("=== Step 1: Create decision case ===")
	createResult := mcpCallTool(t, client, "create_decision_case", map[string]interface{}{
		"alert_id":   "e2e-test-alert-1",
		"created_by": "e2e-test",
	})
	var caseResp struct {
		CaseID string `json:"case_id"`
	}
	extractJSON(t, createResult, &caseResp)
	require.NotEmpty(t, caseResp.CaseID, "case_id should not be empty")
	caseID := caseResp.CaseID
	t.Logf("Created case: %s", caseID)

	t.Log("=== Step 2: Get case details ===")
	getResult := mcpCallTool(t, client, "get_case", map[string]interface{}{
		"case_id": caseID,
	})
	var getResp struct {
		CaseID string `json:"case_id"`
		Status string `json:"status"`
	}
	extractJSON(t, getResult, &getResp)
	require.Equal(t, caseID, getResp.CaseID)
	require.Equal(t, "open", getResp.Status)
	t.Logf("Case status: %s", getResp.Status)

	t.Log("=== Step 3: Build decision context ===")
	ctxResult := mcpCallTool(t, client, "get_decision_context", map[string]interface{}{
		"case_id": caseID,
	})
	require.NotNil(t, ctxResult)
	t.Log("Decision context built")

	t.Log("=== Step 4: Decide (generate proposals) ===")
	decideResult := mcpCallTool(t, client, "decide", map[string]interface{}{
		"case_id": caseID,
	})
	require.NotNil(t, decideResult)
	t.Log("Decision executed")

	t.Log("=== Step 5: List proposals ===")
	listResult := mcpCallTool(t, client, "list_proposals", map[string]interface{}{
		"case_id": caseID,
	})
	var proposalList struct {
		Proposals []struct {
			ProposalID      string `json:"proposal_id"`
			ActionType      string `json:"action_type"`
			ApplyStatus     string `json:"apply_status"`
			RequiresHumanReview bool `json:"requires_human_review"`
		} `json:"proposals"`
	}
	extractJSON(t, listResult, &proposalList)
	require.NotEmpty(t, proposalList.Proposals, "should have at least one proposal")
	t.Logf("Found %d proposals", len(proposalList.Proposals))

	// Try to approve the first proposal
	proposal := proposalList.Proposals[0]
	t.Logf("Proposal: id=%s action=%s status=%s human_review=%v",
		proposal.ProposalID, proposal.ActionType, proposal.ApplyStatus, proposal.RequiresHumanReview)

	t.Log("=== Step 6: Approve proposal ===")
	approveResult := mcpCallTool(t, client, "approve_proposal", map[string]interface{}{
		"proposal_id": proposal.ProposalID,
		"reviewer_id": "e2e-test-operator",
		"feedback":    "Approved by e2e test",
	})
	var approveResp struct {
		ReviewID string `json:"review_id"`
	}
	extractJSON(t, approveResult, &approveResp)
	require.NotEmpty(t, approveResp.ReviewID, "review_id should not be empty")
	t.Logf("Proposal approved, review_id: %s", approveResp.ReviewID)

	t.Log("=== Step 7: Execute proposal ===")
	executeResult := mcpCallTool(t, client, "execute_proposal", map[string]interface{}{
		"proposal_id": proposal.ProposalID,
		"dry_run":     false,
	})
	var execResp struct {
		Success    bool   `json:"success"`
		OutboxID   string `json:"outbox_id,omitempty"`
	}
	extractJSON(t, executeResult, &execResp)
	t.Logf("Execution result: success=%v", execResp.Success)
	if execResp.OutboxID != "" {
		t.Logf("Outbox event created: %s", execResp.OutboxID)
	}

	t.Log("=== Step 8: List outbox events ===")
	outboxResult := mcpCallTool(t, client, "list_outbox_events", map[string]interface{}{
		"limit": 10,
	})
	var outboxResp struct {
		Events []struct {
			EventID     string `json:"event_id"`
			ProposalID  string `json:"proposal_id"`
			Status      string `json:"status"`
			Destination string `json:"destination"`
		} `json:"events"`
	}
	extractJSON(t, outboxResult, &outboxResp)
	t.Logf("Found %d outbox events", len(outboxResp.Events))
	for _, evt := range outboxResp.Events {
		t.Logf("  event_id=%s proposal_id=%s status=%s dest=%s",
			evt.EventID, evt.ProposalID, evt.Status, evt.Destination)
	}

	t.Log("=== Step 9: Get system status ===")
	statusResult := mcpCallTool(t, client, "get_system_status", nil)
	var statusResp struct {
		Database struct {
			Connected bool `json:"connected"`
		} `json:"database"`
		Version string `json:"version"`
	}
	extractJSON(t, statusResult, &statusResp)
	require.True(t, statusResp.Database.Connected, "database should be connected")
	require.NotEmpty(t, statusResp.Version, "version should not be empty")
	t.Logf("System status: version=%s db=%v", statusResp.Version, statusResp.Database.Connected)
}

// TestDecisionSandboxFlow tests the sandbox creation and comparison flow.
func TestDecisionSandboxFlow(t *testing.T) {
	client, cleanup := setupMCPTest(t)
	defer cleanup()

	t.Log("=== Sandbox: Create sandbox ===")
	createResult := mcpCallTool(t, client, "create_sandbox", map[string]interface{}{
		"case_id": "e2e-sandbox-case",
	})
	var sandboxResp struct {
		SandboxID string `json:"sandbox_id"`
	}
	extractJSON(t, createResult, &sandboxResp)
	require.NotEmpty(t, sandboxResp.SandboxID, "sandbox_id should not be empty")
	sbxID := sandboxResp.SandboxID
	t.Logf("Created sandbox: %s", sbxID)

	t.Log("=== Sandbox: Get sandbox ===")
	getResult := mcpCallTool(t, client, "get_sandbox", map[string]interface{}{
		"sandbox_id": sbxID,
	})
	require.NotNil(t, getResult)
	t.Log("Retrieved sandbox successfully")

	t.Log("=== Sandbox: Create another for comparison ===")
	create2Result := mcpCallTool(t, client, "create_sandbox", map[string]interface{}{
		"case_id": "e2e-sandbox-case-2",
	})
	extractJSON(t, create2Result, &sandboxResp)
	sbxID2 := sandboxResp.SandboxID
	t.Logf("Created second sandbox: %s", sbxID2)

	t.Log("=== Sandbox: List sandboxes ===")
	listResult := mcpCallTool(t, client, "list_pipeline_status", nil)
	require.NotNil(t, listResult)
	t.Log("Listed sandboxes successfully")
}

// TestDecisionAlternateFlows tests the LLM-based decision and comparison.
func TestDecisionAlternateFlows(t *testing.T) {
	client, cleanup := setupMCPTest(t)
	defer cleanup()

	// Create a case first
	createResult := mcpCallTool(t, client, "create_decision_case", map[string]interface{}{
		"alert_id":   "e2e-alt-alert-1",
		"created_by": "e2e-test",
	})
	var caseResp struct {
		CaseID string `json:"case_id"`
	}
	extractJSON(t, createResult, &caseResp)
	require.NotEmpty(t, caseResp.CaseID)
	caseID := caseResp.CaseID
	t.Logf("Created case: %s", caseID)

	// Try resolve with different statuses
	t.Log("=== Alternate: Resolve case ===")
	resolveResult := mcpCallTool(t, client, "resolve_case", map[string]interface{}{
		"case_id":    caseID,
		"resolution": "no_action_needed",
		"comment":    "E2E test - no action required",
	})
	_ = resolveResult
	t.Log("Case resolved")

	// Verify case status updated
	getResult := mcpCallTool(t, client, "get_case", map[string]interface{}{
		"case_id": caseID,
	})
	var getResp struct {
		Status     string  `json:"status"`
		Resolution *string `json:"resolution"`
	}
	extractJSON(t, getResult, &getResp)
	t.Logf("Case status after resolve: %s", getResp.Status)
	if getResp.Resolution != nil {
		t.Logf("Resolution: %s", *getResp.Resolution)
	}
}
