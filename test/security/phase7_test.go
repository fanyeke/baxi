package security

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"baxi/internal/action"
	"baxi/internal/adapter"
	"baxi/internal/review"
	"baxi/internal/testutil"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func migrationsDir() string {
	dir, _ := os.Getwd()
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, "migrations")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		dir = filepath.Dir(dir)
	}
	return "migrations"
}

func actionRegistryPath() string {
	dir, _ := os.Getwd()
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, "config", "action_registry.yml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		dir = filepath.Dir(dir)
	}
	return "config/action_registry.yml"
}

type proposalLoaderAdapter struct {
	repo *review.ReviewRepository
}

func (p *proposalLoaderAdapter) GetProposalByID(ctx context.Context, pool *pgxpool.Pool, proposalID string) (*action.ActionProposal, error) {
	row, err := p.repo.GetProposalByID(ctx, pool, proposalID)
	if err != nil || row == nil {
		return nil, err
	}
	ap := &action.ActionProposal{
		ProposalID:          row.ProposalID,
		CaseID:              row.CaseID,
		ActionType:          row.ActionType,
		Title:               row.Title,
		ApplyStatus:         row.ApplyStatus,
		CreatedAt:           row.CreatedAt,
		RequiresHumanReview: row.RequiresHumanReview,
	}
	if row.DecisionID != nil {
		ap.DecisionID = *row.DecisionID
	}
	if row.Description != nil {
		ap.Description = *row.Description
	}
	if row.RiskLevel != nil {
		ap.RiskLevel = *row.RiskLevel
	}
	return ap, nil
}

// setupTestDB starts a test PostgreSQL container, runs migrations, and returns the pool.
func setupTestDB(t *testing.T) (*testutil.PostgresContainer, *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { pg.Terminate(ctx) })

	pool, err := pgxpool.New(ctx, pg.ConnectionString())
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close() })

	require.NoError(t, pg.RunMigrations(ctx, migrationsDir()))

	return pg, pool
}

// insertTestCase inserts a decision_case and action_proposal with the given parameters.
func insertTestCase(ctx context.Context, pool *pgxpool.Pool, caseID, proposalID, actionType, applyStatus string) error {
	if _, err := pool.Exec(ctx,
		`INSERT INTO ai.decision_case (case_id, status, created_at) VALUES ($1, 'proposal_generated', NOW())`,
		caseID); err != nil {
		return err
	}
	_, err := pool.Exec(ctx,
		`INSERT INTO ai.action_proposal (proposal_id, case_id, action_type, apply_status, title, created_at)
		 VALUES ($1, $2, $3, $4, 'Security test proposal', NOW())`,
		proposalID, caseID, actionType, applyStatus)
	return err
}

func TestPhase7Security_UnapprovedExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping security test")
	}
	ctx := context.Background()
	_, pool := setupTestDB(t)

	proposalID := "prop_sec_unapproved"
	require.NoError(t, insertTestCase(ctx, pool, "case_sec_unapproved", proposalID, "notify_owner", "proposed"))

	registry, err := action.NewActionRegistry(actionRegistryPath())
	require.NoError(t, err)

	loader := &proposalLoaderAdapter{repo: review.NewReviewRepository()}
	applySvc := action.NewApplyService(registry, nil, loader)

	_, err = applySvc.ExecuteProposal(ctx, pool, proposalID, "attacker", action.WithDryRun(false))
	require.Error(t, err)
	require.ErrorIs(t, err, action.ErrNotApproved)
}

func TestPhase7Security_RejectedExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping security test")
	}
	ctx := context.Background()
	_, pool := setupTestDB(t)

	proposalID := "prop_sec_rejected"
	require.NoError(t, insertTestCase(ctx, pool, "case_sec_rejected", proposalID, "notify_owner", "rejected"))

	registry, err := action.NewActionRegistry(actionRegistryPath())
	require.NoError(t, err)

	loader := &proposalLoaderAdapter{repo: review.NewReviewRepository()}
	applySvc := action.NewApplyService(registry, nil, loader)

	_, err = applySvc.ExecuteProposal(ctx, pool, proposalID, "attacker", action.WithDryRun(false))
	require.Error(t, err)
	require.ErrorIs(t, err, action.ErrNotApproved)
}

func TestPhase7Security_NonWhitelistExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping security test")
	}
	ctx := context.Background()
	_, pool := setupTestDB(t)

	proposalID := "prop_sec_nonwhitelist"
	require.NoError(t, insertTestCase(ctx, pool, "case_sec_nonwhitelist", proposalID, "delete_database", "approved"))

	registry, err := action.NewActionRegistry(actionRegistryPath())
	require.NoError(t, err)
	assert.False(t, registry.IsAllowed("delete_database"))

	loader := &proposalLoaderAdapter{repo: review.NewReviewRepository()}
	applySvc := action.NewApplyService(registry, nil, loader)

	_, err = applySvc.ExecuteProposal(ctx, pool, proposalID, "attacker", action.WithDryRun(false))
	require.Error(t, err)
	require.ErrorIs(t, err, action.ErrActionNotAllowed)
}

func TestPhase7Security_DirectTableWrite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping security test")
	}
	ctx := context.Background()
	_, pool := setupTestDB(t)

	proposalID := "prop_sec_tablewrite"
	caseID := "case_sec_tablewrite"
	require.NoError(t, insertTestCase(ctx, pool, caseID, proposalID, "notify_owner", "approved"))

	registry, err := action.NewActionRegistry(actionRegistryPath())
	require.NoError(t, err)

	loader := &proposalLoaderAdapter{repo: review.NewReviewRepository()}
	applySvc := action.NewApplyService(registry, nil, loader)

	result, err := applySvc.ExecuteProposal(ctx, pool, proposalID, "actor-1", action.WithDryRun(false))
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "no executor found")

	var finalStatus string
	err = pool.QueryRow(ctx, `SELECT apply_status FROM ai.action_proposal WHERE proposal_id = $1`, proposalID).Scan(&finalStatus)
	require.NoError(t, err)
	assert.Equal(t, "failed", finalStatus)
	assert.NotEqual(t, "applied", finalStatus, "proposal must not reach applied status without proper executor")
}

func TestPhase7Security_BypassReview(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping security test")
	}
	ctx := context.Background()
	_, pool := setupTestDB(t)

	proposalID := "prop_sec_bypass"
	require.NoError(t, insertTestCase(ctx, pool, "case_sec_bypass", proposalID, "notify_owner", "approved"))

	reviewRepo := review.NewReviewRepository()
	reviewSvc := review.NewReviewService(reviewRepo, pool)

	_, err := reviewSvc.ApproveProposal(ctx, proposalID, "reviewer-1", "Looks good")
	require.Error(t, err)
	require.ErrorIs(t, err, review.ErrInvalidState)

	_, err = reviewSvc.RejectProposal(ctx, proposalID, "reviewer-2", "Rejecting")
	require.Error(t, err)
	require.ErrorIs(t, err, review.ErrInvalidState)

	_, err = reviewSvc.CancelProposal(ctx, proposalID, "reviewer-3", "Cancelling")
	require.Error(t, err)
	require.ErrorIs(t, err, review.ErrInvalidState)
}

func TestPhase7Security_AuditTampering(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping security test")
	}
	ctx := context.Background()
	_, pool := setupTestDB(t)

	caseID := "case_sec_audit"
	proposalID := "prop_sec_audit"
	require.NoError(t, insertTestCase(ctx, pool, caseID, proposalID, "notify_owner", "proposed"))

	reviewRepo := review.NewReviewRepository()
	reviewSvc := review.NewReviewService(reviewRepo, pool)

	_, err := reviewSvc.ApproveProposal(ctx, proposalID, "reviewer-audit", "Approved for audit test")
	require.NoError(t, err)

	var initialAuditCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit.audit_log`).Scan(&initialAuditCount)
	require.NoError(t, err)
	assert.Greater(t, initialAuditCount, 0, "audit logs should exist after approval")

	registry, err := action.NewActionRegistry(actionRegistryPath())
	require.NoError(t, err)

	loader := &proposalLoaderAdapter{repo: reviewRepo}
	applySvc := action.NewApplyService(registry, nil, loader)

	_, err = applySvc.ExecuteProposal(ctx, pool, proposalID, "audit-actor", action.WithDryRun(true))
	require.NoError(t, err)

	var afterDryRunAuditCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit.audit_log`).Scan(&afterDryRunAuditCount)
	require.NoError(t, err)

	var deletedCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit.audit_log WHERE action LIKE '%delete%'`).Scan(&deletedCount)
	require.NoError(t, err)
	assert.Equal(t, 0, deletedCount, "no audit logs should have delete action")

	var modifiedAuditCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit.audit_log WHERE action LIKE '%modify%' OR action LIKE '%update%'`).Scan(&modifiedAuditCount)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `UPDATE audit.audit_log SET metadata = '{"tampered": true}' WHERE resource_id = $1`, proposalID)
	require.NoError(t, err)

	var tamperedCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit.audit_log WHERE metadata::text LIKE '%tampered%'`).Scan(&tamperedCount)
	require.NoError(t, err)

	assert.Equal(t, initialAuditCount, afterDryRunAuditCount, "dry-run should not add audit entries")
	assert.Equal(t, 1, tamperedCount, "direct SQL can modify audit logs (defense must be at application layer)")
}

func TestPhase7Security_DryRunSafety(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping security test")
	}
	ctx := context.Background()
	_, pool := setupTestDB(t)

	caseID := "case_sec_dryrun"
	proposalID := "prop_sec_dryrun"
	require.NoError(t, insertTestCase(ctx, pool, caseID, proposalID, "notify_owner", "approved"))

	registry, err := action.NewActionRegistry(actionRegistryPath())
	require.NoError(t, err)

	feishuAdapter := adapter.NewFeishuAdapter(adapter.FeishuConfig{WebhookURL: "http://localhost:9999/test"})
	executors := map[string]action.ActionExecutor{
		"feishu": feishuAdapter,
	}

	loader := &proposalLoaderAdapter{repo: review.NewReviewRepository()}
	applySvc := action.NewApplyService(registry, executors, loader)

	result, err := applySvc.ExecuteProposal(ctx, pool, proposalID, "dryrun-actor", action.WithDryRun(true))
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.True(t, result.DryRun)
	assert.Empty(t, result.OutboxEventID, "dry-run must not create outbox events")

	var outboxCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM ops.outbox_event WHERE source_id = $1`, proposalID).Scan(&outboxCount)
	require.NoError(t, err)
	assert.Equal(t, 0, outboxCount, "no outbox events should exist after dry-run")

	var execAuditCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit.audit_log WHERE resource_id = $1 AND category = 'action_apply'`, proposalID).Scan(&execAuditCount)
	require.NoError(t, err)
	assert.Equal(t, 0, execAuditCount, "dry-run should not create audit logs")
}
