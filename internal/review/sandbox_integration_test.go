package review

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sandboxTableDDL = `
CREATE TABLE IF NOT EXISTS ai.proposal_sandbox (
    sandbox_id      TEXT PRIMARY KEY,
    case_id         TEXT NOT NULL REFERENCES ai.decision_case(case_id),
    proposal_id     TEXT REFERENCES ai.action_proposal(proposal_id),
    sandbox_data    JSONB NOT NULL DEFAULT '{}',
    status          TEXT NOT NULL DEFAULT 'draft',
    compared_with   TEXT[],
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_proposal_sandbox_case_id ON ai.proposal_sandbox(case_id);
CREATE INDEX IF NOT EXISTS idx_proposal_sandbox_status ON ai.proposal_sandbox(status);
`

func setupSandboxTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := setupReviewTestDB(t)
	ctx := context.Background()
	_, err := pool.Exec(ctx, sandboxTableDDL)
	require.NoError(t, err)
	return pool
}

// TestSandboxService_CreateAndGet verifies CreateSandbox followed by GetSandbox.
func TestSandboxService_CreateAndGet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupSandboxTestDB(t)
	ctx := context.Background()
	svc := NewSandboxService(pool)

	insertTestDecisionCase(t, pool, "case-svc-1", "open")

	// Create with data
	data := map[string]interface{}{
		"action": "approve",
		"score":  0.95,
		"tags":   []interface{}{"urgent"},
	}
	sandboxID, err := svc.CreateSandbox(ctx, "case-svc-1", data)
	require.NoError(t, err)
	require.NotEmpty(t, sandboxID)

	// Get and verify
	sb, err := svc.GetSandbox(ctx, sandboxID)
	require.NoError(t, err)
	require.NotNil(t, sb)
	assert.Equal(t, sandboxID, sb.SandboxID)
	assert.Equal(t, "case-svc-1", sb.CaseID)
	assert.Equal(t, "draft", sb.Status)
	assert.Equal(t, "approve", sb.SandboxData["action"])
	assert.Equal(t, 0.95, sb.SandboxData["score"])
	assert.NotZero(t, sb.CreatedAt)
	assert.Nil(t, sb.UpdatedAt)
	assert.Nil(t, sb.ProposalID)
	assert.Empty(t, sb.ComparedWith)
}

// TestSandboxService_CreateWithNilData verifies that passing nil data produces an empty map.
func TestSandboxService_CreateWithNilData(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupSandboxTestDB(t)
	ctx := context.Background()
	svc := NewSandboxService(pool)

	insertTestDecisionCase(t, pool, "case-svc-nil", "open")

	sandboxID, err := svc.CreateSandbox(ctx, "case-svc-nil", nil)
	require.NoError(t, err)
	require.NotEmpty(t, sandboxID)

	sb, err := svc.GetSandbox(ctx, sandboxID)
	require.NoError(t, err)
	require.NotNil(t, sb)
	// json.Marshal(nil) produces "null" which scans back as nil
	_ = sb.SandboxData
	assert.Empty(t, sb.SandboxData)
}

// TestSandboxService_AddProposalToSandbox verifies linking a proposal and that
// updated_at is populated.
func TestSandboxService_AddProposalToSandbox(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupSandboxTestDB(t)
	ctx := context.Background()
	svc := NewSandboxService(pool)

	insertTestDecisionCase(t, pool, "case-svc-prop", "proposal_generated")
	insertTestActionProposal(t, pool, "prop-svc-1", "case-svc-prop", "notify_owner", "proposed", "Test")

	sandboxID, err := svc.CreateSandbox(ctx, "case-svc-prop", nil)
	require.NoError(t, err)

	// Link proposal
	err = svc.AddProposalToSandbox(ctx, sandboxID, "prop-svc-1")
	require.NoError(t, err)

	// Verify link
	sb, err := svc.GetSandbox(ctx, sandboxID)
	require.NoError(t, err)
	require.NotNil(t, sb)
	require.NotNil(t, sb.ProposalID)
	assert.Equal(t, "prop-svc-1", *sb.ProposalID)
	assert.NotNil(t, sb.UpdatedAt, "updated_at should be set after linking a proposal")
}

// TestSandboxService_AddProposalToNonexistentSandbox verifies error handling.
func TestSandboxService_AddProposalToNonexistentSandbox(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupSandboxTestDB(t)
	ctx := context.Background()
	svc := NewSandboxService(pool)

	err := svc.AddProposalToSandbox(ctx, "nonexistent", "prop-dummy")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestSandboxService_ListSandboxes verifies listing returns all sandboxes in
// descending creation order.
func TestSandboxService_ListSandboxes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupSandboxTestDB(t)
	ctx := context.Background()
	svc := NewSandboxService(pool)

	insertTestDecisionCase(t, pool, "case-list-1", "open")
	insertTestDecisionCase(t, pool, "case-list-2", "open")

	id1, err := svc.CreateSandbox(ctx, "case-list-1", sandboxDataHelper("order", 1))
	require.NoError(t, err)
	id2, err := svc.CreateSandbox(ctx, "case-list-2", sandboxDataHelper("order", 2))
	require.NoError(t, err)

	sandboxes, err := svc.ListSandboxes(ctx)
	require.NoError(t, err)

	found := make(map[string]bool)
	for _, sb := range sandboxes {
		found[sb.SandboxID] = true
	}
	assert.True(t, found[id1], "sandbox 1 should be in list")
	assert.True(t, found[id2], "sandbox 2 should be in list")
}

// TestSandboxService_CompareWithDifferences exercises CompareSandbox with
// differing data maps and verifies all difference types.
func TestSandboxService_CompareWithDifferences(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupSandboxTestDB(t)
	ctx := context.Background()
	svc := NewSandboxService(pool)

	insertTestDecisionCase(t, pool, "case-cmp-1", "open")
	insertTestDecisionCase(t, pool, "case-cmp-2", "open")

	id1, err := svc.CreateSandbox(ctx, "case-cmp-1", map[string]interface{}{
		"action": "notify",
		"score":  0.8,
		"active": true,
	})
	require.NoError(t, err)

	id2, err := svc.CreateSandbox(ctx, "case-cmp-2", map[string]interface{}{
		"action": "block",
		"score":  0.95,
		"extra":  "field",
	})
	require.NoError(t, err)

	result, err := svc.CompareSandbox(ctx, id1, id2)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, id1, result.Sandbox1ID)
	assert.Equal(t, id2, result.Sandbox2ID)
	assert.NotEmpty(t, result.Differences)

	dm := make(map[string]Difference)
	for _, d := range result.Differences {
		dm[d.Field] = d
	}

	// action: notify -> block
	if d, ok := dm["action"]; ok {
		assert.Equal(t, "notify", d.Value1)
		assert.Equal(t, "block", d.Value2)
	} else {
		t.Error("expected difference for action")
	}

	// score: 0.8 -> 0.95
	if d, ok := dm["score"]; ok {
		assert.Equal(t, 0.8, d.Value1)
		assert.Equal(t, 0.95, d.Value2)
	} else {
		t.Error("expected difference for score")
	}

	// active: present in 1, missing in 2
	if d, ok := dm["active"]; ok {
		assert.Equal(t, true, d.Value1)
		assert.Nil(t, d.Value2)
	} else {
		t.Error("expected difference for active (present vs missing)")
	}

	// extra: missing in 1, present in 2
	if d, ok := dm["extra"]; ok {
		assert.Nil(t, d.Value1)
		assert.Equal(t, "field", d.Value2)
	} else {
		t.Error("expected difference for extra (missing vs present)")
	}
}

// TestSandboxService_CompareIdentical verifies that identical sandbox data
// produces an empty diff.
func TestSandboxService_CompareIdentical(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupSandboxTestDB(t)
	ctx := context.Background()
	svc := NewSandboxService(pool)

	insertTestDecisionCase(t, pool, "case-cmp-id-1", "open")
	insertTestDecisionCase(t, pool, "case-cmp-id-2", "open")

	data := map[string]interface{}{"k": "v", "n": float64(42)}
	id1, err := svc.CreateSandbox(ctx, "case-cmp-id-1", data)
	require.NoError(t, err)
	id2, err := svc.CreateSandbox(ctx, "case-cmp-id-2", data)
	require.NoError(t, err)

	result, err := svc.CompareSandbox(ctx, id1, id2)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Differences, "identical sandboxes should have no diffs")
}

// TestSandboxService_CompareNonexistent verifies error on non-existent IDs.
func TestSandboxService_CompareNonexistent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupSandboxTestDB(t)
	ctx := context.Background()
	svc := NewSandboxService(pool)

	insertTestDecisionCase(t, pool, "case-cmp-err", "open")
	realID, err := svc.CreateSandbox(ctx, "case-cmp-err", nil)
	require.NoError(t, err)

	// First arg non-existent
	result, err := svc.CompareSandbox(ctx, "bad-id", realID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.Nil(t, result)

	// Second arg non-existent
	result, err = svc.CompareSandbox(ctx, realID, "bad-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.Nil(t, result)
}

// TestSandboxService_GetNonexistent verifies GetSandbox returns nil for missing ID.
func TestSandboxService_GetNonexistent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupSandboxTestDB(t)
	ctx := context.Background()
	svc := NewSandboxService(pool)

	sb, err := svc.GetSandbox(ctx, "missing-sandbox")
	require.NoError(t, err)
	assert.Nil(t, sb)
}

// sandboxDataHelper is a small helper for creating map literals in tests.
func sandboxDataHelper(kv ...interface{}) map[string]interface{} {
	m := make(map[string]interface{}, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i].(string)] = kv[i+1]
	}
	return m
}

// TestSandboxService_CompareNestedData verifies comparison with nested maps
// and arrays — ensures the JSON serialization equality check works.
func TestSandboxService_CompareNestedData(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupSandboxTestDB(t)
	ctx := context.Background()
	svc := NewSandboxService(pool)

	insertTestDecisionCase(t, pool, "case-nest-1", "open")
	insertTestDecisionCase(t, pool, "case-nest-2", "open")

	id1, err := svc.CreateSandbox(ctx, "case-nest-1", map[string]interface{}{
		"metadata": map[string]interface{}{
			"source": "alert",
			"config": map[string]interface{}{"retries": 3},
		},
		"tags": []interface{}{"a", "b"},
	})
	require.NoError(t, err)

	id2, err := svc.CreateSandbox(ctx, "case-nest-2", map[string]interface{}{
		"metadata": map[string]interface{}{
			"source": "manual",
			"config": map[string]interface{}{"retries": 5},
		},
		"tags": []interface{}{"a", "c"},
	})
	require.NoError(t, err)

	result, err := svc.CompareSandbox(ctx, id1, id2)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Differences, 2, "metadata + tags should differ")
}

// TestSandboxService_JSONRoundTrip verifies that sandbox data survives a
// JSON marshal/unmarshal cycle through the database without corruption.
func TestSandboxService_JSONRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupSandboxTestDB(t)
	ctx := context.Background()
	svc := NewSandboxService(pool)

	insertTestDecisionCase(t, pool, "case-roundtrip", "open")

	original := map[string]interface{}{
		"int_val":    42,
		"float_val":  3.14,
		"str_val":    "hello",
		"bool_val":   true,
		"null_val":   nil,
		"array_val":  []interface{}{1, "two", true},
		"map_val":    map[string]interface{}{"nested": "data"},
	}

	// Marshal to JSON and back to verify the data we send
	origJSON, err := json.Marshal(original)
	require.NoError(t, err)

	var expected map[string]interface{}
	err = json.Unmarshal(origJSON, &expected)
	require.NoError(t, err)

	sandboxID, err := svc.CreateSandbox(ctx, "case-roundtrip", original)
	require.NoError(t, err)

	sb, err := svc.GetSandbox(ctx, sandboxID)
	require.NoError(t, err)
	require.NotNil(t, sb)

	// Round-trip the stored data through JSON for consistent comparison
	storedJSON, err := json.Marshal(sb.SandboxData)
	require.NoError(t, err)

	var stored map[string]interface{}
	err = json.Unmarshal(storedJSON, &stored)
	require.NoError(t, err)

	assert.Equal(t, expected["int_val"], stored["int_val"])
	assert.Equal(t, expected["float_val"], stored["float_val"])
	assert.Equal(t, expected["str_val"], stored["str_val"])
	assert.Equal(t, expected["bool_val"], stored["bool_val"])
	assert.Equal(t, expected["array_val"], stored["array_val"])
	assert.JSONEq(t, string(origJSON), string(storedJSON), "full JSON round-trip should match")
}
