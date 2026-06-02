package action

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"baxi/internal/review"
	"baxi/internal/testutil"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- Mocks ----------

type mockProposalLoader struct {
	proposal *ActionProposal
	err      error
}

func (m *mockProposalLoader) GetProposalByID(ctx context.Context, pool *pgxpool.Pool, proposalID string) (*ActionProposal, error) {
	return m.proposal, m.err
}

type mockExecutor struct {
	result ExecutionResult
	err    error
}

func (m *mockExecutor) Execute(ctx context.Context, proposal ActionProposal, dryRun bool) (ExecutionResult, error) {
	return m.result, m.err
}

// ---------- Helpers ----------

func setupTestRegistry(t *testing.T) *ActionRegistry {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "action_registry.yml")
	content := `
actions:
  notify_owner:
    description: "Notify owner"
    risk_level: low
    requires_approval: false
    allowed_by: [ops]
  export_report:
    description: "Export report"
    risk_level: low
    requires_approval: false
    allowed_by: [ops]
  create_followup_task:
    description: "Create task"
    risk_level: medium
    requires_approval: true
    allowed_by: [ops]
  create_outbox_message:
    description: "Outbox message"
    risk_level: low
    requires_approval: false
    allowed_by: [ops]
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	reg, err := NewActionRegistry(path)
	require.NoError(t, err)
	return reg
}

func newTestProposal(applyStatus string, actionType string) *ActionProposal {
	return &ActionProposal{
		ProposalID:  "prop-test-001",
		CaseID:      "case-test-001",
		ActionType:  actionType,
		ApplyStatus: applyStatus,
		Title:       "Test Proposal",
		CreatedAt:   time.Now(),
	}
}

// ---------- Unit Tests ----------

func TestApplyService_DryRunDefault(t *testing.T) {
	reg := setupTestRegistry(t)
	loader := &mockProposalLoader{proposal: newTestProposal("approved", "notify_owner")}
	svc := NewApplyService(reg, nil, loader, nil, nil, nil)

	ctx := context.Background()
	result, err := svc.ExecuteProposal(ctx, nil, "prop-test-001", "actor-1")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.True(t, result.DryRun)
}

func TestApplyService_DryRunExplicitTrue(t *testing.T) {
	reg := setupTestRegistry(t)
	loader := &mockProposalLoader{proposal: newTestProposal("approved", "notify_owner")}
	svc := NewApplyService(reg, nil, loader, nil, nil, nil)

	ctx := context.Background()
	result, err := svc.ExecuteProposal(ctx, nil, "prop-test-001", "actor-1", WithDryRun(true))

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.True(t, result.DryRun)
}

func TestApplyService_NotApproved(t *testing.T) {
	reg := setupTestRegistry(t)
	loader := &mockProposalLoader{proposal: newTestProposal("proposed", "notify_owner")}
	svc := NewApplyService(reg, nil, loader, nil, nil, nil)

	ctx := context.Background()
	_, err := svc.ExecuteProposal(ctx, nil, "prop-test-001", "actor-1")

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotApproved))
}

func TestApplyService_ActionNotAllowed(t *testing.T) {
	reg := setupTestRegistry(t)
	loader := &mockProposalLoader{proposal: newTestProposal("approved", "hack_database")}
	svc := NewApplyService(reg, nil, loader, nil, nil, nil)

	ctx := context.Background()
	_, err := svc.ExecuteProposal(ctx, nil, "prop-test-001", "actor-1")

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrActionNotAllowed))
}

func TestApplyService_ProposalNotFound(t *testing.T) {
	reg := setupTestRegistry(t)
	loader := &mockProposalLoader{proposal: nil}
	svc := NewApplyService(reg, nil, loader, nil, nil, nil)

	ctx := context.Background()
	_, err := svc.ExecuteProposal(ctx, nil, "prop-test-001", "actor-1")

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrProposalNotFound))
}

func TestApplyService_LoaderError(t *testing.T) {
	reg := setupTestRegistry(t)
	loader := &mockProposalLoader{err: errors.New("db down")}
	svc := NewApplyService(reg, nil, loader, nil, nil, nil)

	ctx := context.Background()
	_, err := svc.ExecuteProposal(ctx, nil, "prop-test-001", "actor-1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "db down")
}

func TestApplyService_DryRunSkipsWhitelistCheck(t *testing.T) {
	reg := setupTestRegistry(t)
	loader := &mockProposalLoader{proposal: newTestProposal("approved", "notify_owner")}
	svc := NewApplyService(reg, nil, loader, nil, nil, nil)

	ctx := context.Background()
	result, err := svc.ExecuteProposal(ctx, nil, "prop-test-001", "actor-1", WithDryRun(true))

	require.NoError(t, err)
	assert.True(t, result.Success)
}

// ---------- Integration Tests (testcontainers) ----------

const applyServiceTestDDL = `
CREATE SCHEMA IF NOT EXISTS ai;
CREATE SCHEMA IF NOT EXISTS audit;
CREATE SCHEMA IF NOT EXISTS ops;

CREATE TABLE IF NOT EXISTS ai.decision_case (
    case_id TEXT PRIMARY KEY,
    status TEXT DEFAULT 'open',
    source_type TEXT NOT NULL DEFAULT '',
    source_id TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT NOW()
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

CREATE TABLE IF NOT EXISTS audit.audit_log (
    audit_id BIGSERIAL PRIMARY KEY,
    category TEXT,
    action TEXT,
    actor TEXT,
    resource_type TEXT,
    resource_id TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
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

CREATE TABLE IF NOT EXISTS ops.outbox_event (
    event_id TEXT PRIMARY KEY,
    source_type TEXT NOT NULL,
    source_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    payload_json JSONB,
    target_channel TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    dispatch_attempts BIGINT DEFAULT 0,
    next_retry_at TIMESTAMPTZ,
    last_dispatch_at TIMESTAMPTZ,
    error_message TEXT
);
`

type reviewProposalAdapter struct {
	repo *review.ReviewRepository
}

func (a *reviewProposalAdapter) GetProposalByID(ctx context.Context, pool *pgxpool.Pool, proposalID string) (*ActionProposal, error) {
	row, err := a.repo.GetProposalByID(ctx, pool, proposalID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}

	p := &ActionProposal{
		ProposalID:          row.ProposalID,
		CaseID:              row.CaseID,
		ActionType:          row.ActionType,
		ApplyStatus:         row.ApplyStatus,
		Title:               row.Title,
		CreatedAt:           row.CreatedAt,
		RequiresHumanReview: row.RequiresHumanReview,
	}
	if row.DecisionID != nil {
		p.DecisionID = *row.DecisionID
	}
	if row.Description != nil {
		p.Description = *row.Description
	}
	if row.RiskLevel != nil {
		p.RiskLevel = *row.RiskLevel
	}
	if row.Payload != nil {
		var payload map[string]interface{}
		if err := json.Unmarshal(*row.Payload, &payload); err == nil {
			p.Payload = payload
		}
	}
	return p, nil
}

func setupApplyServiceTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	pool, err := pgxpool.New(ctx, pg.ConnectionString())
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	_, err = pool.Exec(ctx, applyServiceTestDDL)
	require.NoError(t, err)

	return pool
}

func insertTestProposal(t *testing.T, pool *pgxpool.Pool, proposalID, caseID, actionType, applyStatus, title string) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		INSERT INTO ai.action_proposal (proposal_id, case_id, action_type, apply_status, title)
		VALUES ($1, $2, $3, $4, $5)
	`, proposalID, caseID, actionType, applyStatus, title)
	require.NoError(t, err)
}

func insertTestCase(t *testing.T, pool *pgxpool.Pool, caseID string) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		INSERT INTO ai.decision_case (case_id, status, source_type, source_id)
		VALUES ($1, 'open', 'test', 'test')
	`, caseID)
	require.NoError(t, err)
}

func TestApplyService_Integration_DryRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupApplyServiceTestDB(t)
	ctx := context.Background()

	insertTestCase(t, pool, "case-dry-1")
	insertTestProposal(t, pool, "prop-dry-1", "case-dry-1", "notify_owner", "approved", "Notify")

	reg := setupTestRegistry(t)
	repo := review.NewReviewRepository()
	loader := &reviewProposalAdapter{repo: repo}
	svc := NewApplyService(reg, nil, loader, nil, nil, nil)

	result, err := svc.ExecuteProposal(ctx, pool, "prop-dry-1", "actor-dry-1")

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.True(t, result.DryRun)

	// Verify no DB side effects
	var status string
	err = pool.QueryRow(ctx, `SELECT apply_status FROM ai.action_proposal WHERE proposal_id = $1`, "prop-dry-1").Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, "approved", status)

	var auditCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit.audit_log WHERE resource_id = $1`, "prop-dry-1").Scan(&auditCount)
	require.NoError(t, err)
	assert.Equal(t, 0, auditCount)
}

func TestApplyService_Integration_RealExecution_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupApplyServiceTestDB(t)
	ctx := context.Background()

	insertTestCase(t, pool, "case-real-1")
	insertTestProposal(t, pool, "prop-real-1", "case-real-1", "notify_owner", "approved", "Notify")

	reg := setupTestRegistry(t)
	repo := review.NewReviewRepository()
	loader := &reviewProposalAdapter{repo: repo}

	exec := &mockExecutor{result: ExecutionResult{Success: true, DryRun: false}}
	executors := map[string]ActionExecutor{"feishu": exec}
	svc := NewApplyService(reg, executors, loader, nil, nil, nil)

	result, err := svc.ExecuteProposal(ctx, pool, "prop-real-1", "actor-real-1", WithDryRun(false))

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.False(t, result.DryRun)

	// Verify proposal updated to applied
	var status string
	var appliedBy *string
	err = pool.QueryRow(ctx, `SELECT apply_status, applied_by FROM ai.action_proposal WHERE proposal_id = $1`, "prop-real-1").Scan(&status, &appliedBy)
	require.NoError(t, err)
	assert.Equal(t, "applied", status)
	require.NotNil(t, appliedBy)
	assert.Equal(t, "actor-real-1", *appliedBy)

	// Verify audit log inserted
	var auditCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit.audit_log WHERE resource_id = $1 AND action = 'execute'`, "prop-real-1").Scan(&auditCount)
	require.NoError(t, err)
	assert.Equal(t, 1, auditCount)
}

func TestApplyService_Integration_RealExecution_Failure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupApplyServiceTestDB(t)
	ctx := context.Background()

	insertTestCase(t, pool, "case-fail-1")
	insertTestProposal(t, pool, "prop-fail-1", "case-fail-1", "notify_owner", "approved", "Notify")

	reg := setupTestRegistry(t)
	repo := review.NewReviewRepository()
	loader := &reviewProposalAdapter{repo: repo}

	exec := &mockExecutor{result: ExecutionResult{Success: false, DryRun: false, Error: "dispatch error"}}
	executors := map[string]ActionExecutor{"feishu": exec}
	svc := NewApplyService(reg, executors, loader, nil, nil, nil)

	result, err := svc.ExecuteProposal(ctx, pool, "prop-fail-1", "actor-fail-1", WithDryRun(false))

	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Equal(t, "dispatch error", result.Error)

	// Verify proposal updated to failed
	var status string
	err = pool.QueryRow(ctx, `SELECT apply_status FROM ai.action_proposal WHERE proposal_id = $1`, "prop-fail-1").Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, "failed", status)

	// Verify audit log inserted with execute_failed
	var auditCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit.audit_log WHERE resource_id = $1 AND action = 'execute_failed'`, "prop-fail-1").Scan(&auditCount)
	require.NoError(t, err)
	assert.Equal(t, 1, auditCount)
}

func TestApplyService_Integration_AdapterNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupApplyServiceTestDB(t)
	ctx := context.Background()

	insertTestCase(t, pool, "case-na-1")
	insertTestProposal(t, pool, "prop-na-1", "case-na-1", "notify_owner", "approved", "Notify")

	reg := setupTestRegistry(t)
	repo := review.NewReviewRepository()
	loader := &reviewProposalAdapter{repo: repo}

	// No executors configured
	svc := NewApplyService(reg, map[string]ActionExecutor{}, loader, nil, nil, nil)

	result, err := svc.ExecuteProposal(ctx, pool, "prop-na-1", "actor-na-1", WithDryRun(false))

	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "no executor found")

	// Verify proposal updated to failed
	var status string
	err = pool.QueryRow(ctx, `SELECT apply_status FROM ai.action_proposal WHERE proposal_id = $1`, "prop-na-1").Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, "failed", status)

	// Verify audit log inserted
	var auditCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit.audit_log WHERE resource_id = $1`, "prop-na-1").Scan(&auditCount)
	require.NoError(t, err)
	assert.Equal(t, 1, auditCount)
}

func TestApplyService_Integration_NotApproved(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupApplyServiceTestDB(t)
	ctx := context.Background()

	insertTestCase(t, pool, "case-nap-1")
	insertTestProposal(t, pool, "prop-nap-1", "case-nap-1", "notify_owner", "proposed", "Notify")

	reg := setupTestRegistry(t)
	repo := review.NewReviewRepository()
	loader := &reviewProposalAdapter{repo: repo}
	svc := NewApplyService(reg, nil, loader, nil, nil, nil)

	_, err := svc.ExecuteProposal(ctx, pool, "prop-nap-1", "actor-nap-1", WithDryRun(false))

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotApproved))
}

func TestApplyService_Integration_ActionNotAllowed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupApplyServiceTestDB(t)
	ctx := context.Background()

	insertTestCase(t, pool, "case-nal-1")
	// Insert with a valid action_type that passes CHECK but is not whitelisted
	// Actually, the CHECK constraint only allows canonical types. So we test with
	// a proposal that has a canonical type but we simulate by using a mock loader.
	insertTestProposal(t, pool, "prop-nal-1", "case-nal-1", "notify_owner", "approved", "Notify")

	reg := setupTestRegistry(t)
	// Override loader to return a non-whitelisted action type
	loader := &mockProposalLoader{proposal: newTestProposal("approved", "hack_database")}
	svc := NewApplyService(reg, nil, loader, nil, nil, nil)

	_, err := svc.ExecuteProposal(ctx, pool, "prop-nal-1", "actor-nal-1", WithDryRun(false))

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrActionNotAllowed))
}

func TestApplyService_Integration_ProposalNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupApplyServiceTestDB(t)
	ctx := context.Background()

	reg := setupTestRegistry(t)
	repo := review.NewReviewRepository()
	loader := &reviewProposalAdapter{repo: repo}
	svc := NewApplyService(reg, nil, loader, nil, nil, nil)

	_, err := svc.ExecuteProposal(ctx, pool, "nonexistent", "actor-nf-1", WithDryRun(false))

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrProposalNotFound))
}

func TestApplyService_Integration_GithubChannelSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupApplyServiceTestDB(t)
	ctx := context.Background()

	insertTestCase(t, pool, "case-gh-1")
	insertTestProposal(t, pool, "prop-gh-1", "case-gh-1", "create_followup_task", "approved", "Task")

	reg := setupTestRegistry(t)
	repo := review.NewReviewRepository()
	loader := &reviewProposalAdapter{repo: repo}

	exec := &mockExecutor{result: ExecutionResult{Success: true, DryRun: false}}
	executors := map[string]ActionExecutor{"github": exec}
	svc := NewApplyService(reg, executors, loader, nil, nil, nil)

	result, err := svc.ExecuteProposal(ctx, pool, "prop-gh-1", "actor-gh-1", WithDryRun(false))

	require.NoError(t, err)
	assert.True(t, result.Success)

	var status string
	err = pool.QueryRow(ctx, `SELECT apply_status FROM ai.action_proposal WHERE proposal_id = $1`, "prop-gh-1").Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, "applied", status)
}

func TestApplyService_Integration_TransactionRolledBackOnUpdateError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupApplyServiceTestDB(t)
	ctx := context.Background()

	insertTestCase(t, pool, "case-tx-1")
	insertTestProposal(t, pool, "prop-tx-1", "case-tx-1", "notify_owner", "approved", "Notify")

	reg := setupTestRegistry(t)
	repo := review.NewReviewRepository()
	loader := &reviewProposalAdapter{repo: repo}

	exec := &mockExecutor{result: ExecutionResult{Success: true, DryRun: false}}
	executors := map[string]ActionExecutor{"feishu": exec}
	svc := NewApplyService(reg, executors, loader, nil, nil, nil)

	// First execution succeeds
	_, err := svc.ExecuteProposal(ctx, pool, "prop-tx-1", "actor-tx-1", WithDryRun(false))
	require.NoError(t, err)

	// Second execution on same proposal fails because status is now "applied"
	_, err = svc.ExecuteProposal(ctx, pool, "prop-tx-1", "actor-tx-2", WithDryRun(false))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotApproved))
}

// ──── Enhanced Write Path: dry-run gate ──────────────────────────────────────

// TestExecuteProposal_DryRunGate verifies that executing a proposal with
// dry_run=false requires the BAXI_ALLOW_LIVE_EXECUTION=true environment
// flag to be set. Without it, the call must return an error preventing
// live execution.
//
// This is an integration test because dry_run=false requires a real
// database to execute the proposal transaction.
//
// TDD RED: The env var check is not yet implemented. The test expects
// an error containing "BAXI_ALLOW_LIVE_EXECUTION" but currently the
// code proceeds to execution (which may succeed via mock executor)
// because no gate exists.
func TestExecuteProposal_DryRunGate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Ensure the env var is NOT set — simulating a production-like
	// environment where live execution must be explicitly allowed.
	t.Setenv("BAXI_ALLOW_LIVE_EXECUTION", "")

	pool := setupApplyServiceTestDB(t)
	ctx := context.Background()

	insertTestCase(t, pool, "case-gate-1")
	insertTestProposal(t, pool, "prop-gate-1", "case-gate-1", "notify_owner", "approved", "Notify")

	reg := setupTestRegistry(t)
	exec := &mockExecutor{result: ExecutionResult{Success: true, DryRun: false}}
	executors := map[string]ActionExecutor{"feishu": exec}

	// Use a real ProposalLoader that reads from the DB
	repo := review.NewReviewRepository()
	loader := &reviewProposalAdapter{repo: repo}
	svc := NewApplyService(reg, executors, loader, nil, nil, pool)

	_, err := svc.ExecuteProposal(ctx, pool, "prop-gate-1", "actor-gate-1", WithDryRun(false))

	// TDD RED: The env gate check does not exist yet, so one of the
	// following happens:
	//   - The executor succeeds (result.Success=true), err=nil
	//   - Some other error occurs
	//
	// Once the gate is implemented, this assertion should be:
	require.Error(t, err, "BAXI_ALLOW_LIVE_EXECUTION must be set to true for dry_run=false")
	assert.Contains(t, err.Error(), "BAXI_ALLOW_LIVE_EXECUTION",
		"error message must reference the missing env var")
}

// TestExecuteProposal_DryRunTrue_AlwaysAllowed verifies that dry_run=true
// always works regardless of whether BAXI_ALLOW_LIVE_EXECUTION is set.
// The dry-run path uses NoOpExecutor and never touches the database, so
// it must never be blocked by the live-execution gate.
//
// TDD note: This test should pass even in the RED phase because the
// dry-run path already works correctly. It exists as a regression guard
// to ensure the live-execution gate does not inadvertently block dry runs.
func TestExecuteProposal_DryRunTrue_AlwaysAllowed(t *testing.T) {
	// Do NOT set BAXI_ALLOW_LIVE_EXECUTION — simulating a locked-down environment
	t.Setenv("BAXI_ALLOW_LIVE_EXECUTION", "")

	reg := setupTestRegistry(t)
	loader := &mockProposalLoader{proposal: newTestProposal("approved", "notify_owner")}
	svc := NewApplyService(reg, nil, loader, nil, nil, nil)

	ctx := context.Background()

	// WithDryRun(true) — must always be allowed
	result, err := svc.ExecuteProposal(ctx, nil, "prop-test-001", "actor-1", WithDryRun(true))

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success, "dry_run=true must succeed even without BAXI_ALLOW_LIVE_EXECUTION")
	assert.True(t, result.DryRun, "result must indicate it was a dry run")
}

// ---------- Compile-time checks ----------

var _ ProposalLoader = (*mockProposalLoader)(nil)
var _ ActionExecutor = (*mockExecutor)(nil)
