package review

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const auditTableDDL = `
CREATE SCHEMA IF NOT EXISTS audit;

CREATE TABLE IF NOT EXISTS audit.audit_log (
	id BIGSERIAL PRIMARY KEY,
	category TEXT,
	action TEXT,
	actor TEXT,
	resource_type TEXT,
	resource_id TEXT,
	metadata JSONB,
	created_at TIMESTAMPTZ DEFAULT NOW()
);
`

func setupServiceTestDB(t *testing.T) *pgxpool.Pool {
	pool := setupReviewTestDB(t)
	ctx := context.Background()
	_, err := pool.Exec(ctx, auditTableDDL)
	require.NoError(t, err)
	return pool
}

func TestReviewService_ApproveProposal_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupServiceTestDB(t)
	ctx := context.Background()
	repo := NewReviewRepository()
	svc := NewReviewService(repo, pool)

	insertTestDecisionCase(t, pool, "case-approve-1", "proposal_generated")
	insertTestActionProposal(t, pool, "prop-approve-1", "case-approve-1", "notify_owner", "proposed", "Test Proposal")

	record, err := svc.ApproveProposal(ctx, "prop-approve-1", "reviewer-1", "Looks good")
	require.NoError(t, err)
	require.NotNil(t, record)
	assert.Equal(t, "prop-approve-1", record.ProposalID)
	assert.Equal(t, "reviewer-1", record.ReviewerID)
	assert.Equal(t, VerdictApprove, record.Verdict)
	assert.Equal(t, "Looks good", record.Feedback)
	assert.NotEmpty(t, record.RecordID)

	var status string
	err = pool.QueryRow(ctx, `SELECT apply_status FROM ai.action_proposal WHERE proposal_id = $1`, "prop-approve-1").Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, "approved", status)

	fetched, err := repo.GetReviewByID(ctx, pool, record.RecordID)
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, VerdictApprove, fetched.Verdict)

	var caseStatus string
	err = pool.QueryRow(ctx, `SELECT status FROM ai.decision_case WHERE case_id = $1`, "case-approve-1").Scan(&caseStatus)
	require.NoError(t, err)
	assert.Equal(t, "review_required", caseStatus)

	var auditCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM audit.audit_log
		WHERE category = 'review' AND action = 'proposal_approved' AND resource_id = $1
	`, "prop-approve-1").Scan(&auditCount)
	require.NoError(t, err)
	assert.Equal(t, 1, auditCount)
}

func TestReviewService_ApproveProposal_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupServiceTestDB(t)
	ctx := context.Background()
	svc := NewReviewService(NewReviewRepository(), pool)

	record, err := svc.ApproveProposal(ctx, "nonexistent", "reviewer-1", "feedback")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrProposalNotFound), "expected ErrProposalNotFound, got: %v", err)
	assert.Nil(t, record)
}

func TestReviewService_ApproveProposal_InvalidState(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupServiceTestDB(t)
	ctx := context.Background()
	svc := NewReviewService(NewReviewRepository(), pool)

	insertTestDecisionCase(t, pool, "case-approve-inv-1", "open")
	insertTestActionProposal(t, pool, "prop-approve-inv-1", "case-approve-inv-1", "notify_owner", "approved", "Already Approved")

	record, err := svc.ApproveProposal(ctx, "prop-approve-inv-1", "reviewer-1", "feedback")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidState), "expected ErrInvalidState, got: %v", err)
	assert.Nil(t, record)
}

func TestReviewService_ApproveProposal_RollbackOnFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupReviewTestDB(t)
	ctx := context.Background()
	svc := NewReviewService(NewReviewRepository(), pool)

	insertTestDecisionCase(t, pool, "case-rollback-1", "proposal_generated")
	insertTestActionProposal(t, pool, "prop-rollback-1", "case-rollback-1", "notify_owner", "proposed", "Test")

	_, err := svc.ApproveProposal(ctx, "prop-rollback-1", "reviewer-1", "OK")
	require.Error(t, err)

	var status string
	err = pool.QueryRow(ctx, `SELECT apply_status FROM ai.action_proposal WHERE proposal_id = $1`, "prop-rollback-1").Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, "proposed", status, "proposal status should remain unchanged after rollback")

	var count int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM ai.review_record WHERE proposal_id = $1`, "prop-rollback-1").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "no review record should exist after rollback")
}

func TestReviewService_RejectProposal_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupServiceTestDB(t)
	ctx := context.Background()
	repo := NewReviewRepository()
	svc := NewReviewService(repo, pool)

	insertTestDecisionCase(t, pool, "case-reject-1", "proposal_generated")
	insertTestActionProposal(t, pool, "prop-reject-1", "case-reject-1", "notify_owner", "proposed", "Test Proposal")

	record, err := svc.RejectProposal(ctx, "prop-reject-1", "reviewer-2", "Not acceptable")
	require.NoError(t, err)
	require.NotNil(t, record)
	assert.Equal(t, VerdictReject, record.Verdict)
	assert.Equal(t, "Not acceptable", record.Feedback)

	var status string
	err = pool.QueryRow(ctx, `SELECT apply_status FROM ai.action_proposal WHERE proposal_id = $1`, "prop-reject-1").Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, "rejected", status)

	var auditCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM audit.audit_log
		WHERE category = 'review' AND action = 'proposal_rejected' AND resource_id = $1
	`, "prop-reject-1").Scan(&auditCount)
	require.NoError(t, err)
	assert.Equal(t, 1, auditCount)
}

func TestReviewService_CancelProposal_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupServiceTestDB(t)
	ctx := context.Background()
	repo := NewReviewRepository()
	svc := NewReviewService(repo, pool)

	insertTestDecisionCase(t, pool, "case-cancel-1", "proposal_generated")
	insertTestActionProposal(t, pool, "prop-cancel-1", "case-cancel-1", "notify_owner", "proposed", "Test Proposal")

	record, err := svc.CancelProposal(ctx, "prop-cancel-1", "reviewer-3", "Cancelled by user")
	require.NoError(t, err)
	require.NotNil(t, record)
	assert.Equal(t, VerdictCancel, record.Verdict)
	assert.Equal(t, "Cancelled by user", record.Feedback)

	var status string
	err = pool.QueryRow(ctx, `SELECT apply_status FROM ai.action_proposal WHERE proposal_id = $1`, "prop-cancel-1").Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, "rejected", status)

	var auditCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM audit.audit_log
		WHERE category = 'review' AND action = 'proposal_cancelled' AND resource_id = $1
	`, "prop-cancel-1").Scan(&auditCount)
	require.NoError(t, err)
	assert.Equal(t, 1, auditCount)
}

func TestReviewService_GetReviewRecord(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupServiceTestDB(t)
	ctx := context.Background()
	repo := NewReviewRepository()
	svc := NewReviewService(repo, pool)

	insertTestDecisionCase(t, pool, "case-get-review-1", "open")
	insertTestActionProposal(t, pool, "prop-get-review-1", "case-get-review-1", "notify_owner", "proposed", "Test")

	now := time.Now().UTC()
	_, err := pool.Exec(ctx, `
		INSERT INTO ai.review_record (review_id, proposal_id, reviewer_id, verdict, feedback, reviewed_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, "review-get-1", "prop-get-review-1", "user-1", "approve", "Good", now)
	require.NoError(t, err)

	record, err := svc.GetReviewRecord(ctx, "review-get-1")
	require.NoError(t, err)
	require.NotNil(t, record)
	assert.Equal(t, "review-get-1", record.RecordID)
	assert.Equal(t, VerdictApprove, record.Verdict)

	notFound, err := svc.GetReviewRecord(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestReviewService_GetProposalByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupServiceTestDB(t)
	ctx := context.Background()
	svc := NewReviewService(NewReviewRepository(), pool)

	insertTestDecisionCase(t, pool, "case-get-prop-svc-1", "open")
	insertTestActionProposal(t, pool, "prop-get-svc-1", "case-get-prop-svc-1", "notify_owner", "proposed", "Test Proposal")

	proposal, err := svc.GetProposalByID(ctx, "prop-get-svc-1")
	require.NoError(t, err)
	require.NotNil(t, proposal)
	assert.Equal(t, "prop-get-svc-1", proposal.ProposalID)
	assert.Equal(t, "proposed", proposal.ApplyStatus)

	notFound, err := svc.GetProposalByID(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestReviewService_ApproveProposal_CaseStatusNotUpdated(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupServiceTestDB(t)
	ctx := context.Background()
	svc := NewReviewService(NewReviewRepository(), pool)

	insertTestDecisionCase(t, pool, "case-no-update-1", "open")
	insertTestActionProposal(t, pool, "prop-no-update-1", "case-no-update-1", "notify_owner", "proposed", "Test")

	_, err := svc.ApproveProposal(ctx, "prop-no-update-1", "reviewer-1", "OK")
	require.NoError(t, err)

	var caseStatus string
	err = pool.QueryRow(ctx, `SELECT status FROM ai.decision_case WHERE case_id = $1`, "case-no-update-1").Scan(&caseStatus)
	require.NoError(t, err)
	assert.Equal(t, "open", caseStatus)
}
