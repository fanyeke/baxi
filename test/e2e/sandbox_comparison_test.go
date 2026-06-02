//go:build integration

package e2e_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/review"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// insertDecisionCase inserts a minimal decision_case row so FK constraints on
// ai.proposal_sandbox are satisfied.  It does not go through the engine —
// just a bare row.
func insertDecisionCase(t *testing.T, ctx context.Context, pool *pgxpool.Pool, caseID, severity string) {
	t.Helper()
	_, err := pool.Exec(ctx, `
		INSERT INTO ai.decision_case (case_id, status, source_type, source_id,
		                              object_type, object_id, severity, created_at)
		VALUES ($1, 'created', 'e2e', $2, 'test', 'obj_1', $3, NOW())
	`, caseID, caseID, severity)
	require.NoError(t, err, "insert decision_case %s", caseID)
}

// insertProposal inserts a minimal action_proposal row so the FK on
// ai.proposal_sandbox.proposal_id is satisfied.
func insertProposal(t *testing.T, ctx context.Context, pool *pgxpool.Pool, proposalID, caseID string) {
	t.Helper()
	_, err := pool.Exec(ctx, `
		INSERT INTO ai.action_proposal (proposal_id, case_id, action_type, apply_status, title, created_at)
		VALUES ($1, $2, 'notify_owner', 'proposed', 'E2E sandbox test proposal', NOW())
	`, proposalID, caseID)
	require.NoError(t, err, "insert action_proposal %s", proposalID)
}

// sandboxData is a small helper that creates a map literal without having to
// type map[string]interface{} everywhere.
func sandboxData(kv ...interface{}) map[string]interface{} {
	m := make(map[string]interface{}, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i].(string)] = kv[i+1]
	}
	return m
}

// differenceMap converts a slice of differences to a map keyed by field name
// for easier assertion.
func differenceMap(diffs []review.Difference) map[string]review.Difference {
	m := make(map[string]review.Difference, len(diffs))
	for _, d := range diffs {
		m[d.Field] = d
	}
	return m
}

// ---------------------------------------------------------------------------
// Test 1: Full lifecycle: create → get → list → add proposal → compare
// ---------------------------------------------------------------------------

func TestSandboxComparison_FullLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, pool, cleanup := setupTestDB(t)
	defer cleanup()

	svc := review.NewSandboxService(pool)

	// ----------------------------------------------------------------
	// Setup: two decision cases
	// ----------------------------------------------------------------
	insertDecisionCase(t, ctx, pool, "sbx-lifecycle-case-1", "low")
	insertDecisionCase(t, ctx, pool, "sbx-lifecycle-case-2", "critical")

	// ----------------------------------------------------------------
	// 1. Create sandbox with structured data
	// ----------------------------------------------------------------
	data1 := sandboxData(
		"action_type", "notify_owner",
		"risk_score", 0.85,
		"active", true,
		"priority", "high",
		"tags", []interface{}{"urgent", "compliance"},
	)
	saID1, err := svc.CreateSandbox(ctx, "sbx-lifecycle-case-1", data1)
	require.NoError(t, err, "create sandbox 1")
	require.NotEmpty(t, saID1, "sandbox 1 id")

	// ----------------------------------------------------------------
	// 2. Create a second sandbox with different data
	// ----------------------------------------------------------------
	data2 := sandboxData(
		"action_type", "block",
		"risk_score", 0.95,
		"active", false,
		"priority", "critical",
		"tags", []interface{}{"critical"},
		"reason", "threshold_exceeded",
	)
	saID2, err := svc.CreateSandbox(ctx, "sbx-lifecycle-case-2", data2)
	require.NoError(t, err, "create sandbox 2")
	require.NotEmpty(t, saID2, "sandbox 2 id")
	assert.NotEqual(t, saID1, saID2, "sandbox IDs must be unique")

	// ----------------------------------------------------------------
	// 3. Get sandbox 1 — verify all stored fields
	// ----------------------------------------------------------------
	got1, err := svc.GetSandbox(ctx, saID1)
	require.NoError(t, err, "get sandbox 1")
	require.NotNil(t, got1)
	assert.Equal(t, saID1, got1.SandboxID)
	assert.Equal(t, "sbx-lifecycle-case-1", got1.CaseID)
	assert.Equal(t, "draft", got1.Status)
	assert.Nil(t, got1.ProposalID, "no proposal linked yet")
	assert.Empty(t, got1.ComparedWith, "no comparisons yet")
	assert.Equal(t, 0.85, got1.SandboxData["risk_score"])
	assert.Equal(t, "notify_owner", got1.SandboxData["action_type"])
	assert.Equal(t, true, got1.SandboxData["active"])
	assert.NotZero(t, got1.CreatedAt, "created_at must be set")
	assert.Nil(t, got1.UpdatedAt, "updated_at should be nil on fresh sandbox")

	// ----------------------------------------------------------------
	// 4. Get sandbox 2 — verify its data
	// ----------------------------------------------------------------
	got2, err := svc.GetSandbox(ctx, saID2)
	require.NoError(t, err, "get sandbox 2")
	require.NotNil(t, got2)
	assert.Equal(t, "block", got2.SandboxData["action_type"])
	assert.Equal(t, 0.95, got2.SandboxData["risk_score"])
	assert.Equal(t, false, got2.SandboxData["active"])
	reason, hasReason := got2.SandboxData["reason"]
	assert.True(t, hasReason, "sandbox 2 should have reason field")
	assert.Equal(t, "threshold_exceeded", reason)

	// ----------------------------------------------------------------
	// 5. List sandboxes — must contain both
	// ----------------------------------------------------------------
	all, err := svc.ListSandboxes(ctx)
	require.NoError(t, err, "list sandboxes")
	found := make(map[string]bool)
	for _, sb := range all {
		found[sb.SandboxID] = true
	}
	assert.True(t, found[saID1], "sandbox 1 should be in list")
	assert.True(t, found[saID2], "sandbox 2 should be in list")

	// ----------------------------------------------------------------
	// 6. Add a proposal to sandbox 1 and verify the link
	// ----------------------------------------------------------------
	insertProposal(t, ctx, pool, "prop-lifecycle-1", "sbx-lifecycle-case-1")
	err = svc.AddProposalToSandbox(ctx, saID1, "prop-lifecycle-1")
	require.NoError(t, err, "add proposal to sandbox 1")

	got1after, err := svc.GetSandbox(ctx, saID1)
	require.NoError(t, err)
	require.NotNil(t, got1after.ProposalID)
	assert.Equal(t, "prop-lifecycle-1", *got1after.ProposalID)
	assert.NotNil(t, got1after.UpdatedAt, "updated_at should be set after update")

	// ----------------------------------------------------------------
	// 7. Compare sandbox 1 vs sandbox 2 — should find differences
	// ----------------------------------------------------------------
	result, err := svc.CompareSandbox(ctx, saID1, saID2)
	require.NoError(t, err, "compare sandboxes")
	require.NotNil(t, result)
	assert.Equal(t, saID1, result.Sandbox1ID)
	assert.Equal(t, saID2, result.Sandbox2ID)

	dm := differenceMap(result.Differences)
	t.Logf("found %d differences between sandboxes", len(result.Differences))

	// action_type: notify_owner → block
	if d, ok := dm["action_type"]; ok {
		assert.Equal(t, "notify_owner", d.Value1)
		assert.Equal(t, "block", d.Value2)
	} else {
		t.Error("expected difference for action_type")
	}

	// risk_score: 0.85 → 0.95
	if d, ok := dm["risk_score"]; ok {
		assert.Equal(t, 0.85, d.Value1)
		assert.Equal(t, 0.95, d.Value2)
	} else {
		t.Error("expected difference for risk_score")
	}

	// active: true → false
	if d, ok := dm["active"]; ok {
		assert.Equal(t, true, d.Value1)
		assert.Equal(t, false, d.Value2)
	} else {
		t.Error("expected difference for active")
	}

	// priority: "high" → "critical"
	if d, ok := dm["priority"]; ok {
		assert.Equal(t, "high", d.Value1)
		assert.Equal(t, "critical", d.Value2)
	} else {
		t.Error("expected difference for priority")
	}

	// tags: ["urgent","compliance"] → ["critical"]
	if d, ok := dm["tags"]; ok {
		assert.NotEqual(t, d.Value1, d.Value2, "tags should differ")
	} else {
		t.Error("expected difference for tags")
	}

	// reason: missing in sandbox 1, present in sandbox 2
	if d, ok := dm["reason"]; ok {
		assert.Nil(t, d.Value1, "sandbox 1 has no reason field")
		assert.Equal(t, "threshold_exceeded", d.Value2)
	} else {
		t.Error("expected difference for reason (missing vs present)")
	}

	// ----------------------------------------------------------------
	// 8. Compare sandbox with itself — must have zero differences
	// ----------------------------------------------------------------
	same, err := svc.CompareSandbox(ctx, saID1, saID1)
	require.NoError(t, err)
	require.NotNil(t, same)
	assert.Empty(t, same.Differences, "comparing sandbox with itself should produce no diffs")
}

// ---------------------------------------------------------------------------
// Test 2: Identical sandboxes produce zero differences
// ---------------------------------------------------------------------------

func TestSandboxComparison_IdenticalSandboxes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, pool, cleanup := setupTestDB(t)
	defer cleanup()

	svc := review.NewSandboxService(pool)
	insertDecisionCase(t, ctx, pool, "sbx-ident-case-1", "low")
	insertDecisionCase(t, ctx, pool, "sbx-ident-case-2", "low")

	data := sandboxData(
		"alert", "high-cpu",
		"threshold", 90.0,
		"enabled", true,
	)
	saID1, err := svc.CreateSandbox(ctx, "sbx-ident-case-1", data)
	require.NoError(t, err)

	saID2, err := svc.CreateSandbox(ctx, "sbx-ident-case-2", data)
	require.NoError(t, err)

	// Both sandboxes have identical data → no differences
	result, err := svc.CompareSandbox(ctx, saID1, saID2)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Differences, "identical data must produce no differences")
}

// ---------------------------------------------------------------------------
// Test 3: One sandbox has a field the other doesn't (both directions)
// ---------------------------------------------------------------------------

func TestSandboxComparison_MissingField(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, pool, cleanup := setupTestDB(t)
	defer cleanup()

	svc := review.NewSandboxService(pool)
	insertDecisionCase(t, ctx, pool, "sbx-miss-case-1", "low")
	insertDecisionCase(t, ctx, pool, "sbx-miss-case-2", "low")

	// Sandbox 1 has all four fields
	saID1, err := svc.CreateSandbox(ctx, "sbx-miss-case-1", sandboxData(
		"a", "x",
		"b", "y",
		"c", "z",
		"d", "w",
	))
	require.NoError(t, err)

	// Sandbox 2 is missing field "c" and "d"
	saID2, err := svc.CreateSandbox(ctx, "sbx-miss-case-2", sandboxData(
		"a", "x",
		"b", "y",
	))
	require.NoError(t, err)

	result, err := svc.CompareSandbox(ctx, saID1, saID2)
	require.NoError(t, err)
	dm := differenceMap(result.Differences)

	// c: present in 1, missing in 2
	if d, ok := dm["c"]; ok {
		assert.Equal(t, "z", d.Value1)
		assert.Nil(t, d.Value2, "value2 should be nil for missing field")
	} else {
		t.Error("expected difference for field c (present vs missing)")
	}

	// d: present in 1, missing in 2
	if d, ok := dm["d"]; ok {
		assert.Equal(t, "w", d.Value1)
		assert.Nil(t, d.Value2, "value2 should be nil for missing field")
	} else {
		t.Error("expected difference for field d (present vs missing)")
	}

	// a and b are the same — should NOT appear as differences
	_, hasA := dm["a"]
	_, hasB := dm["b"]
	assert.False(t, hasA, "field 'a' is identical and must not appear in diffs")
	assert.False(t, hasB, "field 'b' is identical and must not appear in diffs")
}

// ---------------------------------------------------------------------------
// Test 4: Nested and mixed-type data comparisons
// ---------------------------------------------------------------------------

func TestSandboxComparison_NestedData(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, pool, cleanup := setupTestDB(t)
	defer cleanup()

	svc := review.NewSandboxService(pool)
	insertDecisionCase(t, ctx, pool, "sbx-nest-case-1", "low")
	insertDecisionCase(t, ctx, pool, "sbx-nest-case-2", "low")

	// Sandbox 1: nested map and mixed types
	saID1, err := svc.CreateSandbox(ctx, "sbx-nest-case-1", sandboxData(
		"metadata", map[string]interface{}{
			"source":  "alert",
			"version": 2,
			"owner":   "security-team",
			"config": map[string]interface{}{
				"retry_count": 3,
				"timeout_ms":  5000,
			},
		},
		"scores", []interface{}{0.9, 0.8, 0.95},
		"enabled", true,
	))
	require.NoError(t, err)

	// Sandbox 2: same top-level keys, different nested values
	saID2, err := svc.CreateSandbox(ctx, "sbx-nest-case-2", sandboxData(
		"metadata", map[string]interface{}{
			"source":  "manual",
			"version": 3,
			"owner":   "security-team",
			"config": map[string]interface{}{
				"retry_count": 5,
				"timeout_ms":  10000,
			},
		},
		"scores", []interface{}{0.9, 0.8, 0.99},
		"enabled", false,
	))
	require.NoError(t, err)

	result, err := svc.CompareSandbox(ctx, saID1, saID2)
	require.NoError(t, err)
	dm := differenceMap(result.Differences)

	// All three top-level fields should differ
	assert.Contains(t, dm, "metadata", "metadata should differ")
	assert.Contains(t, dm, "scores", "scores should differ")
	assert.Contains(t, dm, "enabled", "enabled should differ")

	// enabled: true → false
	if d, ok := dm["enabled"]; ok {
		assert.Equal(t, true, d.Value1)
		assert.Equal(t, false, d.Value2)
	}

	// No extra fields
	assert.Len(t, result.Differences, 3, "only 3 fields should differ")
}

// ---------------------------------------------------------------------------
// Test 5: Sandbox with empty data vs sandbox with data
// ---------------------------------------------------------------------------

func TestSandboxComparison_EmptyVsData(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, pool, cleanup := setupTestDB(t)
	defer cleanup()

	svc := review.NewSandboxService(pool)
	insertDecisionCase(t, ctx, pool, "sbx-empty-case-1", "low")
	insertDecisionCase(t, ctx, pool, "sbx-empty-case-2", "low")

	// Sandbox 1: no data passed (defaults to empty map{})
	saID1, err := svc.CreateSandbox(ctx, "sbx-empty-case-1", nil)
	require.NoError(t, err)

	// Sandbox 2: has data
	saID2, err := svc.CreateSandbox(ctx, "sbx-empty-case-2", sandboxData("key", "value"))
	require.NoError(t, err)

	result, err := svc.CompareSandbox(ctx, saID1, saID2)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Differences, "empty vs data should produce diffs")

	dm := differenceMap(result.Differences)
	if d, ok := dm["key"]; ok {
		assert.Nil(t, d.Value1, "sandbox 1 (empty) has nil for key")
		assert.Equal(t, "value", d.Value2)
	} else {
		t.Error("expected difference for field 'key'")
	}
}

// ---------------------------------------------------------------------------
// Test 6: Error cases — non-existent sandboxes
// ---------------------------------------------------------------------------

func TestSandboxComparison_ErrorCases(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, pool, cleanup := setupTestDB(t)
	defer cleanup()

	svc := review.NewSandboxService(pool)
	insertDecisionCase(t, ctx, pool, "sbx-err-case", "low")

	saID, err := svc.CreateSandbox(ctx, "sbx-err-case", sandboxData("k", "v"))
	require.NoError(t, err)
	require.NotEmpty(t, saID)

	// 1. Compare with non-existent first sandbox
	result, err := svc.CompareSandbox(ctx, "sbx-nonexistent-1", saID)
	require.Error(t, err, "comparing with non-existent sandbox 1 should error")
	assert.Contains(t, err.Error(), "not found")
	assert.Nil(t, result)

	// 2. Compare with non-existent second sandbox
	result, err = svc.CompareSandbox(ctx, saID, "sbx-nonexistent-2")
	require.Error(t, err, "comparing with non-existent sandbox 2 should error")
	assert.Contains(t, err.Error(), "not found")
	assert.Nil(t, result)

	// 3. Get non-existent sandbox
	got, err := svc.GetSandbox(ctx, "sbx-nonexistent-3")
	require.NoError(t, err, "get non-existent should not error")
	assert.Nil(t, got, "get non-existent should return nil")

	// 4. Add proposal to non-existent sandbox
	err = svc.AddProposalToSandbox(ctx, "sbx-nonexistent-4", "prop-err")
	require.Error(t, err, "add proposal to non-existent sandbox should error")
	assert.Contains(t, err.Error(), "not found")
}

// ---------------------------------------------------------------------------
// Test 7: Multiple sandboxes with varying data — bulk comparison
// ---------------------------------------------------------------------------

func TestSandboxComparison_MultipleComparisons(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, pool, cleanup := setupTestDB(t)
	defer cleanup()

	svc := review.NewSandboxService(pool)

	// Create three cases and three sandboxes with different severity data
	for i := 1; i <= 3; i++ {
		caseID := "sbx-multi-case-" + string(rune('0'+i))
		insertDecisionCase(t, ctx, pool, caseID, "low")
	}

	saID1, err := svc.CreateSandbox(ctx, "sbx-multi-case-1", sandboxData("severity", "low", "escalate", false))
	require.NoError(t, err)

	saID2, err := svc.CreateSandbox(ctx, "sbx-multi-case-2", sandboxData("severity", "high", "escalate", true))
	require.NoError(t, err)

	saID3, err := svc.CreateSandbox(ctx, "sbx-multi-case-3", sandboxData("severity", "critical", "escalate", true))
	require.NoError(t, err)

	// Compare 1 vs 2
	r12, err := svc.CompareSandbox(ctx, saID1, saID2)
	require.NoError(t, err)
	dm12 := differenceMap(r12.Differences)
	assert.Contains(t, dm12, "severity")
	assert.Contains(t, dm12, "escalate")

	// Compare 2 vs 3
	r23, err := svc.CompareSandbox(ctx, saID2, saID3)
	require.NoError(t, err)
	dm23 := differenceMap(r23.Differences)
	assert.Contains(t, dm23, "severity")
	_, escalateDiff := dm23["escalate"]
	assert.False(t, escalateDiff, "escalate is identical (both true) — should not appear")

	// Compare 1 vs 3
	r13, err := svc.CompareSandbox(ctx, saID1, saID3)
	require.NoError(t, err)
	assert.NotEmpty(t, r13.Differences)
	dm13 := differenceMap(r13.Differences)
	assert.Contains(t, dm13, "severity")
	assert.Contains(t, dm13, "escalate")

	// Verify list returns all three
	all, err := svc.ListSandboxes(ctx)
	require.NoError(t, err)
	idSet := make(map[string]bool)
	for _, sb := range all {
		idSet[sb.SandboxID] = true
	}
	assert.True(t, idSet[saID1])
	assert.True(t, idSet[saID2])
	assert.True(t, idSet[saID3])
}
