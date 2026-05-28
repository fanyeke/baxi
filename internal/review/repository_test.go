package review

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

const reviewTableDDL = `
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
    updated_at TIMESTAMPTZ
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
    CONSTRAINT chk_action_proposal_apply_status CHECK (apply_status IN ('proposed', 'approved', 'rejected', 'applying', 'applied', 'failed')),
    CONSTRAINT chk_action_proposal_action_type CHECK (action_type IN ('create_followup_task', 'notify_owner', 'export_report', 'create_outbox_message'))
);

CREATE TABLE IF NOT EXISTS ai.review_record (
    review_id TEXT PRIMARY KEY,
    proposal_id TEXT,
    reviewer_id TEXT,
    verdict TEXT,
    feedback TEXT,
    reviewed_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT chk_review_record_verdict CHECK (verdict IN ('approve', 'reject', 'cancel'))
);
`

func setupReviewTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	pool, err := pgxpool.New(ctx, pg.ConnectionString())
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	_, err = pool.Exec(ctx, reviewTableDDL)
	require.NoError(t, err)

	return pool
}

func insertTestDecisionCase(t *testing.T, pool *pgxpool.Pool, caseID, status string) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		INSERT INTO ai.decision_case (case_id, status, source_type, source_id)
		VALUES ($1, $2, 'test', 'test')
	`, caseID, status)
	require.NoError(t, err)
}

func insertTestActionProposal(t *testing.T, pool *pgxpool.Pool, proposalID, caseID, actionType, applyStatus, title string) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		INSERT INTO ai.action_proposal (proposal_id, case_id, action_type, apply_status, title)
		VALUES ($1, $2, $3, $4, $5)
	`, proposalID, caseID, actionType, applyStatus, title)
	require.NoError(t, err)
}

func TestReviewRepository_InsertReview(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupReviewTestDB(t)
	ctx := context.Background()
	repo := NewReviewRepository()

	insertTestDecisionCase(t, pool, "case-insert-1", "open")
	insertTestActionProposal(t, pool, "prop-insert-1", "case-insert-1", "notify_owner", "proposed", "Test Proposal")

	now := time.Now().UTC().Truncate(time.Microsecond)
	record := &ReviewRecord{
		RecordID:   "review-1",
		ProposalID: "prop-insert-1",
		ReviewerID: "user-1",
		Verdict:    VerdictApprove,
		Feedback:   "Looks good",
		CreatedAt:  now,
	}

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)

	err = repo.InsertReview(ctx, tx, record)
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	var fetchedReviewID string
	var fetchedReviewedAt time.Time
	err = pool.QueryRow(ctx, `SELECT review_id, reviewed_at FROM ai.review_record WHERE review_id = $1`, "review-1").
		Scan(&fetchedReviewID, &fetchedReviewedAt)
	require.NoError(t, err)
	assert.Equal(t, "review-1", fetchedReviewID)
	assert.WithinDuration(t, now, fetchedReviewedAt, time.Second)
}

func TestReviewRepository_GetReviewByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupReviewTestDB(t)
	ctx := context.Background()
	repo := NewReviewRepository()

	insertTestDecisionCase(t, pool, "case-get-1", "open")
	insertTestActionProposal(t, pool, "prop-get-1", "case-get-1", "notify_owner", "proposed", "Test Proposal")

	now := time.Now().UTC().Truncate(time.Microsecond)
	_, err := pool.Exec(ctx, `
		INSERT INTO ai.review_record (review_id, proposal_id, reviewer_id, verdict, feedback, reviewed_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, "review-get-1", "prop-get-1", "user-1", "approve", "Good to go", now)
	require.NoError(t, err)

	fetched, err := repo.GetReviewByID(ctx, pool, "review-get-1")
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, "review-get-1", fetched.RecordID)
	assert.Equal(t, "prop-get-1", fetched.ProposalID)
	assert.Equal(t, "user-1", fetched.ReviewerID)
	assert.Equal(t, VerdictApprove, fetched.Verdict)
	assert.Equal(t, "Good to go", fetched.Feedback)
	assert.WithinDuration(t, now, fetched.CreatedAt, time.Second)

	notFound, err := repo.GetReviewByID(ctx, pool, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestReviewRepository_GetReviewsByProposal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupReviewTestDB(t)
	ctx := context.Background()
	repo := NewReviewRepository()

	insertTestDecisionCase(t, pool, "case-list-1", "open")
	insertTestActionProposal(t, pool, "prop-list-1", "case-list-1", "notify_owner", "proposed", "Test Proposal")
	insertTestActionProposal(t, pool, "prop-list-2", "case-list-1", "create_followup_task", "proposed", "Another Proposal")

	now := time.Now().UTC()
	_, err := pool.Exec(ctx, `
		INSERT INTO ai.review_record (review_id, proposal_id, reviewer_id, verdict, feedback, reviewed_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, "review-list-1", "prop-list-1", "user-1", "approve", "First review", now.Add(-2*time.Hour))
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO ai.review_record (review_id, proposal_id, reviewer_id, verdict, feedback, reviewed_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, "review-list-2", "prop-list-1", "user-2", "reject", "Second review", now.Add(-1*time.Hour))
	require.NoError(t, err)

	results, err := repo.GetReviewsByProposal(ctx, pool, "prop-list-1")
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "review-list-2", results[0].RecordID)
	assert.Equal(t, "review-list-1", results[1].RecordID)
	assert.Equal(t, VerdictReject, results[0].Verdict)
	assert.Equal(t, VerdictApprove, results[1].Verdict)

	empty, err := repo.GetReviewsByProposal(ctx, pool, "prop-list-2")
	require.NoError(t, err)
	assert.Empty(t, empty)
	assert.NotNil(t, empty)

	none, err := repo.GetReviewsByProposal(ctx, pool, "nonexistent")
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestReviewRepository_UpdateProposalApplyStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupReviewTestDB(t)
	ctx := context.Background()
	repo := NewReviewRepository()

	insertTestDecisionCase(t, pool, "case-upd-prop-1", "open")
	insertTestActionProposal(t, pool, "prop-upd-1", "case-upd-prop-1", "notify_owner", "proposed", "Test Proposal")

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)

	err = repo.UpdateProposalApplyStatus(ctx, tx, "prop-upd-1", "approved")
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	var status string
	err = pool.QueryRow(ctx, `SELECT apply_status FROM ai.action_proposal WHERE proposal_id = $1`, "prop-upd-1").Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, "approved", status)

	tx2, err := pool.Begin(ctx)
	require.NoError(t, err)
	err = repo.UpdateProposalApplyStatus(ctx, tx2, "nonexistent", "applied")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
	_ = tx2.Rollback(ctx)
}

func TestReviewRepository_GetProposalByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupReviewTestDB(t)
	ctx := context.Background()
	repo := NewReviewRepository()

	insertTestDecisionCase(t, pool, "case-get-prop-1", "open")

	payload := json.RawMessage(`{"task":"review"}`)
	_, err := pool.Exec(ctx, `
		INSERT INTO ai.action_proposal (
			proposal_id, case_id, action_type, apply_status, title,
			description, risk_level, requires_human_review, payload
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, "prop-get-1", "case-get-prop-1", "notify_owner", "proposed", "Notify Owner",
		"Send notification", "medium", true, &payload)
	require.NoError(t, err)

	fetched, err := repo.GetProposalByID(ctx, pool, "prop-get-1")
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, "prop-get-1", fetched.ProposalID)
	assert.Equal(t, "case-get-prop-1", fetched.CaseID)
	assert.Equal(t, "notify_owner", fetched.ActionType)
	assert.Equal(t, "proposed", fetched.ApplyStatus)
	assert.Equal(t, "Notify Owner", fetched.Title)
	assert.Equal(t, "Send notification", *fetched.Description)
	assert.Equal(t, "medium", *fetched.RiskLevel)
	assert.True(t, fetched.RequiresHumanReview)
	require.NotNil(t, fetched.Payload)
	assert.JSONEq(t, `{"task":"review"}`, string(*fetched.Payload))

	notFound, err := repo.GetProposalByID(ctx, pool, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestReviewRepository_UpdateCaseStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupReviewTestDB(t)
	ctx := context.Background()
	repo := NewReviewRepository()

	insertTestDecisionCase(t, pool, "case-upd-status-1", "open")

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)

	err = repo.UpdateCaseStatus(ctx, tx, "case-upd-status-1", "review_required")
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	var status string
	var updatedAt *time.Time
	err = pool.QueryRow(ctx, `SELECT status, updated_at FROM ai.decision_case WHERE case_id = $1`, "case-upd-status-1").
		Scan(&status, &updatedAt)
	require.NoError(t, err)
	assert.Equal(t, "review_required", status)
	assert.NotNil(t, updatedAt)

	tx2, err := pool.Begin(ctx)
	require.NoError(t, err)
	err = repo.UpdateCaseStatus(ctx, tx2, "nonexistent", "closed")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
	_ = tx2.Rollback(ctx)
}
