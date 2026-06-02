package repository

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/testutil"
)

const decisionTableDDL = `
CREATE SCHEMA IF NOT EXISTS ai;

CREATE TABLE IF NOT EXISTS ai.decision_case (
    case_id TEXT PRIMARY KEY,
    alert_id TEXT,
    case_type TEXT,
    status TEXT DEFAULT 'open',
    context_json JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    source_type TEXT NOT NULL DEFAULT '',
    source_id TEXT NOT NULL DEFAULT '',
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
    case_id TEXT,
    model_version TEXT,
    prompt_hash TEXT,
    output_json JSONB,
    confidence NUMERIC(4,2),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    status TEXT,
    fallback_reason TEXT,
    validation_errors JSONB,
    recipe_id TEXT,
    context_hash TEXT,
    severity TEXT
);

CREATE TABLE IF NOT EXISTS ai.action_proposal (
    proposal_id TEXT PRIMARY KEY,
    case_id TEXT,
    decision_id TEXT,
    action_type TEXT NOT NULL,
    payload JSONB,
    apply_status TEXT DEFAULT 'pending',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    applied_at TIMESTAMPTZ,
    applied_by TEXT,
    title TEXT NOT NULL DEFAULT '',
    description TEXT,
    risk_level TEXT,
    requires_human_review BOOLEAN DEFAULT TRUE,
    context_hash TEXT,
    action_schema_version TEXT,
    evidence_refs TEXT,
    recipe_id TEXT
);
`

func setupDecisionTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	pool, err := pgxpool.New(ctx, pg.ConnectionString())
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	_, err = pool.Exec(ctx, decisionTableDDL)
	require.NoError(t, err)

	return pool
}

func insertTestDecisionCase(t *testing.T, pool *pgxpool.Pool, row *DecisionCaseRow) {
	t.Helper()
	repo := NewDecisionRepository()
	err := repo.CreateCase(context.Background(), pool, row)
	require.NoError(t, err)
}

func insertTestLLMDecision(t *testing.T, pool *pgxpool.Pool, row *LLMDecisionRow) {
	t.Helper()
	repo := NewDecisionRepository()
	err := repo.CreateDecision(context.Background(), pool, row)
	require.NoError(t, err)
}

func insertTestActionProposal(t *testing.T, pool *pgxpool.Pool, row *ActionProposalRow) {
	t.Helper()
	repo := NewDecisionRepository()
	err := repo.CreateProposal(context.Background(), pool, row)
	require.NoError(t, err)
}

func TestDecisionRepository_CreateAndGet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupDecisionTestDB(t)
	ctx := context.Background()
	repo := NewDecisionRepository()

	now := time.Now().UTC()
	ctxJSON := json.RawMessage(`{"key":"value"}`)
	govJSON := json.RawMessage(`{"policy":"p1"}`)

	row := &DecisionCaseRow{
		CaseID:                 "case-1",
		AlertID:                strPtr("alert-1"),
		CaseType:               strPtr("anomaly"),
		Status:                 "created",
		ContextJSON:            &ctxJSON,
		CreatedAt:              now,
		ResolvedAt:             nil,
		SourceType:             strPtr("rule_engine"),
		SourceID:               strPtr("rule-1"),
		ObjectType:             strPtr("seller"),
		ObjectID:               strPtr("seller-42"),
		Severity:               strPtr("high"),
		ContextHash:            strPtr("abc123"),
		GovernanceSnapshotJSON: &govJSON,
		CreatedBy:              strPtr("system"),
		ErrorMessage:           nil,
		UpdatedAt:              nil,
	}

	err := repo.CreateCase(ctx, pool, row)
	require.NoError(t, err)

	// Retrieve by ID
	fetched, err := repo.GetCaseByID(ctx, pool, "case-1")
	require.NoError(t, err)
	require.NotNil(t, fetched)

	assert.Equal(t, row.CaseID, fetched.CaseID)
	assert.Equal(t, *row.AlertID, *fetched.AlertID)
	assert.Equal(t, *row.CaseType, *fetched.CaseType)
	assert.Equal(t, row.Status, fetched.Status)
	assert.Equal(t, row.SourceType, fetched.SourceType)
	assert.Equal(t, row.SourceID, fetched.SourceID)
	assert.Equal(t, *row.ObjectType, *fetched.ObjectType)
	assert.Equal(t, *row.ObjectID, *fetched.ObjectID)
	assert.Equal(t, *row.Severity, *fetched.Severity)
	assert.Equal(t, *row.ContextHash, *fetched.ContextHash)
	assert.Equal(t, *row.CreatedBy, *fetched.CreatedBy)
	assert.Nil(t, fetched.ResolvedAt)
	assert.Nil(t, fetched.ErrorMessage)
	assert.Nil(t, fetched.UpdatedAt)
}

func TestDecisionRepository_GetBySource(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupDecisionTestDB(t)
	ctx := context.Background()
	repo := NewDecisionRepository()

	now := time.Now().UTC()

	row := &DecisionCaseRow{
		CaseID:     "case-src-1",
		Status:     "open",
		CreatedAt:  now,
		SourceType: strPtr("alert"),
		SourceID:   strPtr("src-42"),
	}
	insertTestDecisionCase(t, pool, row)

	// Retrieve by source
	fetched, err := repo.GetCaseBySource(ctx, pool, "alert", "src-42")
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, "case-src-1", fetched.CaseID)
	assert.Equal(t, "alert", *fetched.SourceType)
	assert.Equal(t, "src-42", *fetched.SourceID)

	// Non-existent source
	_, err = repo.GetCaseBySource(ctx, pool, "alert", "nonexistent")
	assert.Error(t, err)
}

func TestDecisionRepository_UpdateCaseStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupDecisionTestDB(t)
	ctx := context.Background()
	repo := NewDecisionRepository()

	now := time.Now().UTC()

	row := &DecisionCaseRow{
		CaseID:     "case-upd-1",
		Status:     "created",
		CreatedAt:  now,
		SourceType: strPtr("test"),
		SourceID:   strPtr("test-1"),
	}
	insertTestDecisionCase(t, pool, row)

	ctxJSON := json.RawMessage(`{"updated":true}`)
	ctxHash := "newhash"
	govJSON := json.RawMessage(`{"snapshot":"v2"}`)

	err := repo.UpdateCaseStatus(ctx, pool, "case-upd-1", "context_built", &ctxJSON, &ctxHash, &govJSON)
	require.NoError(t, err)

	fetched, err := repo.GetCaseByID(ctx, pool, "case-upd-1")
	require.NoError(t, err)
	assert.Equal(t, "context_built", fetched.Status)
	assert.Equal(t, "newhash", *fetched.ContextHash)
	assert.NotNil(t, fetched.UpdatedAt)
}

func TestDecisionRepository_ListCases(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupDecisionTestDB(t)
	ctx := context.Background()
	repo := NewDecisionRepository()

	now := time.Now().UTC()
	high := "high"
	created := "created"

	cases := []*DecisionCaseRow{
		{CaseID: "case-l-1", Status: "created", CreatedAt: now.Add(-3 * time.Hour), SourceType: strPtr("engine"), SourceID: strPtr("e1"), Severity: &high},
		{CaseID: "case-l-2", Status: "created", CreatedAt: now.Add(-2 * time.Hour), SourceType: strPtr("engine"), SourceID: strPtr("e2"), Severity: strPtr("medium")},
		{CaseID: "case-l-3", Status: "open", CreatedAt: now.Add(-1 * time.Hour), SourceType: strPtr("alert"), SourceID: strPtr("a1"), Severity: &high},
	}
	for _, c := range cases {
		insertTestDecisionCase(t, pool, c)
	}

	// Test: no filters
	results, total, err := repo.ListCases(ctx, pool, CaseFilter{Limit: 100, Offset: 0})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, results, 3)

	// Test: filter by status
	results, total, err = repo.ListCases(ctx, pool, CaseFilter{Status: &created, Limit: 100, Offset: 0})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, results, 2)
	for _, r := range results {
		assert.Equal(t, "created", r.Status)
	}

	// Test: filter by severity
	results, total, err = repo.ListCases(ctx, pool, CaseFilter{Severity: &high, Limit: 100, Offset: 0})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, results, 2)
	for _, r := range results {
		assert.Equal(t, "high", *r.Severity)
	}

	// Test: filter by source_type
	engine := "engine"
	results, total, err = repo.ListCases(ctx, pool, CaseFilter{SourceType: &engine, Limit: 100, Offset: 0})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, results, 2)

	// Test: combined filters
	results, total, err = repo.ListCases(ctx, pool, CaseFilter{Status: &created, Severity: &high, Limit: 100, Offset: 0})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, results, 1)
	assert.Equal(t, "case-l-1", results[0].CaseID)

	// Test: pagination
	results, total, err = repo.ListCases(ctx, pool, CaseFilter{Limit: 2, Offset: 0})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, results, 2)
	assert.Equal(t, "case-l-3", results[0].CaseID) // newest first

	results, total, err = repo.ListCases(ctx, pool, CaseFilter{Limit: 2, Offset: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, results, 1)
	assert.Equal(t, "case-l-1", results[0].CaseID)
}

func TestDecisionRepository_CreateDecision(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupDecisionTestDB(t)
	ctx := context.Background()
	repo := NewDecisionRepository()

	now := time.Now().UTC()

	// Create a case first (required by FK if defined)
	caseRow := &DecisionCaseRow{
		CaseID:     "case-dec-1",
		Status:     "open",
		CreatedAt:  now,
		SourceType: strPtr("test"),
		SourceID:   strPtr("test-dec-1"),
	}
	insertTestDecisionCase(t, pool, caseRow)

	confidence := 0.95
	outputJSON := json.RawMessage(`{"action":"notify"}`)
	validationJSON := json.RawMessage(`[]`)

	decision := &LLMDecisionRow{
		DecisionID:       "dec-1",
		CaseID:           "case-dec-1",
		ModelVersion:     strPtr("gpt-4"),
		PromptHash:       strPtr("hash123"),
		OutputJSON:       &outputJSON,
		Confidence:       &confidence,
		CreatedAt:        now,
		Status:           strPtr("completed"),
		FallbackReason:   nil,
		ValidationErrors: &validationJSON,
	}

	err := repo.CreateDecision(ctx, pool, decision)
	require.NoError(t, err)
}

func TestDecisionRepository_CreateProposal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupDecisionTestDB(t)
	ctx := context.Background()
	repo := NewDecisionRepository()

	now := time.Now().UTC()

	// Create a case first
	caseRow := &DecisionCaseRow{
		CaseID:     "case-prop-1",
		Status:     "proposal_generated",
		CreatedAt:  now,
		SourceType: strPtr("test"),
		SourceID:   strPtr("test-prop-1"),
	}
	insertTestDecisionCase(t, pool, caseRow)

	payload := json.RawMessage(`{"task":"review"}`)

	proposal := &ActionProposalRow{
		ProposalID:          "prop-1",
		CaseID:              "case-prop-1",
		DecisionID:          nil,
		ActionType:          "notify_owner",
		Payload:             &payload,
		ApplyStatus:         "proposed",
		CreatedAt:           now,
		AppliedAt:           nil,
		AppliedBy:           nil,
		Title:               "Notify seller about anomaly",
		Description:         strPtr("Send notification to seller about recent anomaly"),
		RiskLevel:           strPtr("medium"),
		RequiresHumanReview: true,
	}

	err := repo.CreateProposal(ctx, pool, proposal)
	require.NoError(t, err)
}

func TestDecisionRepository_ListProposalsByCase(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupDecisionTestDB(t)
	ctx := context.Background()
	repo := NewDecisionRepository()

	now := time.Now().UTC()

	// Create a case
	caseRow := &DecisionCaseRow{
		CaseID:     "case-prop-list-1",
		Status:     "proposal_generated",
		CreatedAt:  now,
		SourceType: strPtr("test"),
		SourceID:   strPtr("test-prop-list"),
	}
	insertTestDecisionCase(t, pool, caseRow)

	// Create proposals
	payload1 := json.RawMessage(`{"order":1}`)
	payload2 := json.RawMessage(`{"order":2}`)

	prop1 := &ActionProposalRow{
		ProposalID: "prop-l-1", CaseID: "case-prop-list-1",
		ActionType: "notify_owner", ApplyStatus: "proposed",
		CreatedAt: now.Add(-2 * time.Hour), Title: "First proposal",
		RequiresHumanReview: true,
		Payload:             &payload1,
	}
	prop2 := &ActionProposalRow{
		ProposalID: "prop-l-2", CaseID: "case-prop-list-1",
		ActionType: "create_followup_task", ApplyStatus: "approved",
		CreatedAt: now.Add(-1 * time.Hour), Title: "Second proposal",
		RequiresHumanReview: true,
		Payload:             &payload2,
	}

	insertTestActionProposal(t, pool, prop1)
	insertTestActionProposal(t, pool, prop2)

	// List by case
	results, err := repo.ListProposalsByCase(ctx, pool, "case-prop-list-1")
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "prop-l-1", results[0].ProposalID) // ordered by created_at ASC
	assert.Equal(t, "prop-l-2", results[1].ProposalID)

	// Verify fields
	assert.Equal(t, "First proposal", results[0].Title)
	assert.Equal(t, "notify_owner", results[0].ActionType)
	assert.Equal(t, "proposed", results[0].ApplyStatus)
	assert.True(t, results[0].RequiresHumanReview)

	// Empty case
	empty, err := repo.ListProposalsByCase(ctx, pool, "nonexistent")
	require.NoError(t, err)
	assert.Empty(t, empty)
}

// strPtr is a helper to create *string literals.
func strPtr(s string) *string {
	return &s
}
