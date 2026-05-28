package action

import (
	"context"
	"encoding/json"
	"testing"

	"baxi/internal/outbox"
	"baxi/internal/review"
	"baxi/internal/testutil"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- Mock Tx for unit tests ----------

type mockTx struct {
	pgx.Tx
	commitErr   error
	rollbackErr error
	inserted    *outbox.OutboxEvent
}

func (m *mockTx) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func (m *mockTx) Commit(ctx context.Context) error {
	return m.commitErr
}

func (m *mockTx) Rollback(ctx context.Context) error {
	return m.rollbackErr
}

// ---------- Unit Tests ----------

func TestCreateOutboxEventFromProposal_FeishuChannel(t *testing.T) {
	proposal := &ActionProposal{
		ProposalID: "prop-unit-001",
		CaseID:     "case-unit-001",
		ActionType: "notify_owner",
		Payload:    map[string]interface{}{"message": "test notification"},
	}

	tx := &mockTx{}
	ctx := context.Background()

	event, err := CreateOutboxEventFromProposal(ctx, tx, proposal)

	require.NoError(t, err)
	require.NotNil(t, event)
	assert.Equal(t, "action_execution", event.SourceType)
	assert.Equal(t, "prop-unit-001", event.SourceID)
	assert.Equal(t, "notify_owner", event.EventType)
	assert.Equal(t, "pending", event.Status)
	assert.Equal(t, "feishu", event.TargetChannel)
	assert.Equal(t, int64(0), event.DispatchAttempts)

	var envelope map[string]interface{}
	err = json.Unmarshal(event.Payload, &envelope)
	require.NoError(t, err)
	assert.Equal(t, "prop-unit-001", envelope["proposal_id"])
	assert.Equal(t, "case-unit-001", envelope["case_id"])
	assert.Equal(t, "notify_owner", envelope["action_type"])
	assert.NotNil(t, envelope["created_at"])

	payload, ok := envelope["payload"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test notification", payload["message"])
}

func TestCreateOutboxEventFromProposal_GithubChannel(t *testing.T) {
	proposal := &ActionProposal{
		ProposalID: "prop-unit-002",
		CaseID:     "case-unit-002",
		ActionType: "create_followup_task",
		Payload:    map[string]interface{}{"title": "Follow up"},
	}

	tx := &mockTx{}
	ctx := context.Background()

	event, err := CreateOutboxEventFromProposal(ctx, tx, proposal)

	require.NoError(t, err)
	assert.Equal(t, "github", event.TargetChannel)
	assert.Equal(t, "create_followup_task", event.EventType)
}

func TestCreateOutboxEventFromProposal_ExportReport(t *testing.T) {
	proposal := &ActionProposal{
		ProposalID: "prop-unit-003",
		CaseID:     "case-unit-003",
		ActionType: "export_report",
	}

	tx := &mockTx{}
	ctx := context.Background()

	event, err := CreateOutboxEventFromProposal(ctx, tx, proposal)

	require.NoError(t, err)
	assert.Equal(t, "feishu", event.TargetChannel)
}

func TestCreateOutboxEventFromProposal_CreateOutboxMessage(t *testing.T) {
	proposal := &ActionProposal{
		ProposalID: "prop-unit-004",
		CaseID:     "case-unit-004",
		ActionType: "create_outbox_message",
	}

	tx := &mockTx{}
	ctx := context.Background()

	event, err := CreateOutboxEventFromProposal(ctx, tx, proposal)

	require.NoError(t, err)
	assert.Equal(t, "feishu", event.TargetChannel)
}

func TestCreateOutboxEventFromProposal_NilPayload(t *testing.T) {
	proposal := &ActionProposal{
		ProposalID: "prop-unit-005",
		CaseID:     "case-unit-005",
		ActionType: "notify_owner",
		Payload:    nil,
	}

	tx := &mockTx{}
	ctx := context.Background()

	event, err := CreateOutboxEventFromProposal(ctx, tx, proposal)

	require.NoError(t, err)
	var envelope map[string]interface{}
	err = json.Unmarshal(event.Payload, &envelope)
	require.NoError(t, err)
	_, hasPayload := envelope["payload"]
	assert.False(t, hasPayload, "envelope should not contain payload key when proposal payload is nil")
}

func TestCreateOutboxEventFromProposal_EventIDFormat(t *testing.T) {
	proposal := &ActionProposal{
		ProposalID: "prop-unit-006",
		CaseID:     "case-unit-006",
		ActionType: "notify_owner",
	}

	tx := &mockTx{}
	ctx := context.Background()

	event, err := CreateOutboxEventFromProposal(ctx, tx, proposal)

	require.NoError(t, err)
	assert.Contains(t, event.EventID, "evt_")
}

// ---------- Integration Tests (testcontainers) ----------

const outboxIntegrationDDL = `
CREATE SCHEMA IF NOT EXISTS ai;
CREATE SCHEMA IF NOT EXISTS ops;

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

func setupOutboxIntegrationTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	pool, err := pgxpool.New(ctx, pg.ConnectionString())
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	_, err = pool.Exec(ctx, outboxIntegrationDDL)
	require.NoError(t, err)

	return pool
}

func setupOutboxApplyServiceTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	pool, err := pgxpool.New(ctx, pg.ConnectionString())
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	ddl := applyServiceTestDDL + `
CREATE SCHEMA IF NOT EXISTS ops;
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
	_, err = pool.Exec(ctx, ddl)
	require.NoError(t, err)

	return pool
}

func TestOutboxIntegration_InsertAndQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupOutboxIntegrationTestDB(t)
	ctx := context.Background()

	proposal := &ActionProposal{
		ProposalID: "prop-int-001",
		CaseID:     "case-int-001",
		ActionType: "notify_owner",
		Payload:    map[string]interface{}{"recipient": "user@example.com"},
	}

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)

	event, err := CreateOutboxEventFromProposal(ctx, tx, proposal)
	require.NoError(t, err)
	require.NotNil(t, event)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	var count int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM ops.outbox_event WHERE event_id = $1`, event.EventID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	var eventType, sourceType, sourceID, status, targetChannel string
	var payloadJSON json.RawMessage
	err = pool.QueryRow(ctx, `
		SELECT event_type, source_type, source_id, status, target_channel, payload_json
		FROM ops.outbox_event WHERE event_id = $1
	`, event.EventID).Scan(&eventType, &sourceType, &sourceID, &status, &targetChannel, &payloadJSON)
	require.NoError(t, err)

	assert.Equal(t, "notify_owner", eventType)
	assert.Equal(t, "action_execution", sourceType)
	assert.Equal(t, "prop-int-001", sourceID)
	assert.Equal(t, "pending", status)
	assert.Equal(t, "feishu", targetChannel)

	var envelope map[string]interface{}
	err = json.Unmarshal(payloadJSON, &envelope)
	require.NoError(t, err)
	assert.Equal(t, "prop-int-001", envelope["proposal_id"])
	assert.Equal(t, "case-int-001", envelope["case_id"])
}

func TestOutboxIntegration_GithubChannel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupOutboxIntegrationTestDB(t)
	ctx := context.Background()

	proposal := &ActionProposal{
		ProposalID: "prop-int-002",
		CaseID:     "case-int-002",
		ActionType: "create_followup_task",
		Payload:    map[string]interface{}{"title": "Review needed"},
	}

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)

	event, err := CreateOutboxEventFromProposal(ctx, tx, proposal)
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	var targetChannel string
	err = pool.QueryRow(ctx, `SELECT target_channel FROM ops.outbox_event WHERE event_id = $1`, event.EventID).Scan(&targetChannel)
	require.NoError(t, err)
	assert.Equal(t, "github", targetChannel)
}

func TestOutboxIntegration_RollbackOnError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupOutboxIntegrationTestDB(t)
	ctx := context.Background()

	proposal := &ActionProposal{
		ProposalID: "prop-int-003",
		CaseID:     "case-int-003",
		ActionType: "notify_owner",
	}

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)

	event, err := CreateOutboxEventFromProposal(ctx, tx, proposal)
	require.NoError(t, err)

	err = tx.Rollback(ctx)
	require.NoError(t, err)

	var count int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM ops.outbox_event WHERE event_id = $1`, event.EventID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestOutboxIntegration_DryRunDoesNotCreateOutboxEvent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupOutboxApplyServiceTestDB(t)
	ctx := context.Background()

	insertTestCase(t, pool, "case-dryrun-1")
	insertTestProposal(t, pool, "prop-dryrun-1", "case-dryrun-1", "notify_owner", "approved", "Notify")

	reg := setupTestRegistry(t)
	repo := review.NewReviewRepository()
	loader := &reviewProposalAdapter{repo: repo}
	svc := NewApplyService(reg, nil, loader, nil, nil, nil)

	result, err := svc.ExecuteProposal(ctx, pool, "prop-dryrun-1", "actor-dryrun-1")

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.True(t, result.DryRun)
	assert.Empty(t, result.OutboxEventID)

	var count int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM ops.outbox_event WHERE source_id = $1`, "prop-dryrun-1").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestOutboxIntegration_DoesNotCreateOutboxOnFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupOutboxApplyServiceTestDB(t)
	ctx := context.Background()

	insertTestCase(t, pool, "case-fail-outbox-1")
	insertTestProposal(t, pool, "prop-fail-outbox-1", "case-fail-outbox-1", "notify_owner", "approved", "Notify")

	reg := setupTestRegistry(t)
	repo := review.NewReviewRepository()
	loader := &reviewProposalAdapter{repo: repo}

	exec := &mockExecutor{result: ExecutionResult{Success: false, DryRun: false, Error: "dispatch failed"}}
	executors := map[string]ActionExecutor{"feishu": exec}
	svc := NewApplyService(reg, executors, loader, nil, nil, nil)

	result, err := svc.ExecuteProposal(ctx, pool, "prop-fail-outbox-1", "actor-fail-outbox-1", WithDryRun(false))

	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Empty(t, result.OutboxEventID)
}

func TestOutboxIntegration_RealExecutionCreatesOutboxEvent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool := setupOutboxApplyServiceTestDB(t)
	ctx := context.Background()

	insertTestCase(t, pool, "case-outbox-real-1")
	insertTestProposal(t, pool, "prop-outbox-real-1", "case-outbox-real-1", "notify_owner", "approved", "Notify")

	reg := setupTestRegistry(t)
	repo := review.NewReviewRepository()
	loader := &reviewProposalAdapter{repo: repo}

	exec := &mockExecutor{result: ExecutionResult{Success: true, DryRun: false}}
	executors := map[string]ActionExecutor{"feishu": exec}
	svc := NewApplyService(reg, executors, loader, nil, nil, nil)

	result, err := svc.ExecuteProposal(ctx, pool, "prop-outbox-real-1", "actor-outbox-real-1", WithDryRun(false))

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.NotEmpty(t, result.OutboxEventID)

	var status string
	err = pool.QueryRow(ctx, `SELECT status FROM ops.outbox_event WHERE event_id = $1`, result.OutboxEventID).Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, "pending", status)

	var sourceType, sourceID, eventType, targetChannel string
	err = pool.QueryRow(ctx, `
		SELECT source_type, source_id, event_type, target_channel
		FROM ops.outbox_event WHERE event_id = $1
	`, result.OutboxEventID).Scan(&sourceType, &sourceID, &eventType, &targetChannel)
	require.NoError(t, err)
	assert.Equal(t, "action_execution", sourceType)
	assert.Equal(t, "prop-outbox-real-1", sourceID)
	assert.Equal(t, "notify_owner", eventType)
	assert.Equal(t, "feishu", targetChannel)
}

// ---------- Compile-time checks ----------

var _ ActionExecutor = (*mockExecutor)(nil)
