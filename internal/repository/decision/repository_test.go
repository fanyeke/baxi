package decision

import (
	"context"
	"encoding/json"

	"baxi/internal/testutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/repository/common"
)

const decDDL = `
CREATE SCHEMA IF NOT EXISTS ai;
CREATE TABLE IF NOT EXISTS ai.decision_case (
    case_id TEXT PRIMARY KEY,
    alert_id TEXT,
    case_type TEXT,
    status TEXT NOT NULL DEFAULT 'open',
    context_json JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    source_type TEXT,
    source_id TEXT,
    object_type TEXT,
    object_id TEXT,
    severity TEXT,
    context_hash TEXT,
    governance_snapshot_json JSONB,
    created_by TEXT,
    error_message TEXT,
    updated_at TIMESTAMPTZ,
    alert_rules_version TEXT,
    alert_rules_hash TEXT,
    action_registry_version TEXT,
    action_registry_hash TEXT,
    context_snapshot_json JSONB,
    data_snapshot_json JSONB
);
CREATE TABLE IF NOT EXISTS ai.llm_decision (
    decision_id TEXT PRIMARY KEY,
    case_id TEXT REFERENCES ai.decision_case(case_id),
    model_version TEXT,
    prompt_hash TEXT,
    output_json JSONB,
    confidence DOUBLE PRECISION,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status TEXT,
    fallback_reason TEXT,
    validation_errors JSONB,
    recipe_id TEXT,
    context_hash TEXT,
    severity TEXT
);
CREATE TABLE IF NOT EXISTS ai.action_proposal (
    proposal_id TEXT PRIMARY KEY,
    case_id TEXT REFERENCES ai.decision_case(case_id),
    decision_id TEXT REFERENCES ai.llm_decision(decision_id),
    action_type TEXT NOT NULL,
    payload JSONB,
    apply_status TEXT DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    applied_at TIMESTAMPTZ,
    applied_by TEXT,
    title TEXT,
    description TEXT,
    risk_level TEXT,
    requires_human_review BOOLEAN DEFAULT FALSE,
    context_hash TEXT,
    action_schema_version TEXT,
    evidence_refs TEXT,
    recipe_id TEXT
);
`
func setupRepo(t *testing.T) (*Repository, *common.PoolProvider) {
	t.Helper()
	pool := testutil.SetupTestPool(t)
	ctx := context.Background()
	_, err := pool.Exec(ctx, decDDL)
	require.NoError(t, err)
	for _, tbl := range []string{"ai.action_proposal", "ai.llm_decision", "ai.decision_case"} {
		_, _ = pool.Exec(ctx, "TRUNCATE TABLE "+tbl+" CASCADE")
	}
	return NewRepository(common.NewPoolProvider(pool)), common.NewPoolProvider(pool)
}

func TestDecisionCreateAndGetCase(t *testing.T) {
	repo, _ := setupRepo(t)
	ctx := context.Background()
	aid := "alert-1"
	row := &DecisionCaseRow{
		CaseID:    "case-1",
		AlertID:   &aid,
		Status:    "open",
		CreatedBy: strPtr("tester"),
	}
	err := repo.CreateCase(ctx, row)
	require.NoError(t, err)

	got, err := repo.GetCaseByID(ctx, "case-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "case-1", got.CaseID)
	assert.Equal(t, "open", got.Status)
}

func TestDecisionGetCaseBySource(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	pool.Exec(ctx, `INSERT INTO ai.decision_case(case_id,alert_id,status,source_type,source_id) VALUES('cs1','alert-src','open','alert','alert-src')`)
	got, err := repo.GetCaseBySource(ctx, "alert", "alert-src")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "cs1", got.CaseID)
}

func TestDecisionUpdateCaseStatus(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	pool.Exec(ctx, `INSERT INTO ai.decision_case(case_id,status) VALUES('cu1','open')`)
	err := repo.UpdateCaseStatus(ctx, "cu1", "resolved", nil, nil, nil)
	require.NoError(t, err)
	got, err := repo.GetCaseByID(ctx, "cu1")
	require.NoError(t, err)
	assert.Equal(t, "resolved", got.Status)
}

func TestDecisionListCases(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	pool.Exec(ctx, `INSERT INTO ai.decision_case(case_id,status) VALUES('c1','open')`)
	pool.Exec(ctx, `INSERT INTO ai.decision_case(case_id,status) VALUES('c2','resolved')`)
	rows, total, err := repo.ListCases(ctx, CaseFilter{})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, rows, 2)
}

func TestDecisionListCasesFiltered(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	pool.Exec(ctx, `INSERT INTO ai.decision_case(case_id,status) VALUES('c1','open')`)
	pool.Exec(ctx, `INSERT INTO ai.decision_case(case_id,status) VALUES('c2','resolved')`)
	f := CaseFilter{Status: strPtr("open")}
	rows, total, err := repo.ListCases(ctx, f)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "c1", rows[0].CaseID)
}

func TestDecisionCreateDecision(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	pool.Exec(ctx, `INSERT INTO ai.decision_case(case_id,status) VALUES('cd1','open')`)
	row := &LLMDecisionRow{
		DecisionID: "dec-1",
		CaseID:     "cd1",
		ModelVersion: strPtr("gpt-4"),
		OutputJSON: ptrJSON(json.RawMessage(`{"action":"approve"}`)),
	}
	err := repo.CreateDecision(ctx, row)
	require.NoError(t, err)
	require.NotEmpty(t, row.DecisionID)
}

func strPtr(s string) *string { return &s }
func ptrJSON(j json.RawMessage) *json.RawMessage { return &j }
