//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/testutil"
)

const contextTableDDL = `
CREATE SCHEMA IF NOT EXISTS audit;

CREATE TABLE IF NOT EXISTS audit.pipeline_run (
    run_id        TEXT PRIMARY KEY,
    run_type      TEXT NOT NULL,
    mode          TEXT NOT NULL,
    status        TEXT NOT NULL,
    started_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at   TIMESTAMPTZ,
    input_count   BIGINT DEFAULT 0,
    output_count  BIGINT DEFAULT 0,
    error_message TEXT
);

CREATE SCHEMA IF NOT EXISTS ops;

CREATE TABLE IF NOT EXISTS ops.metric_alert (
    alert_id       TEXT PRIMARY KEY,
    severity       TEXT NOT NULL,
    metric_name    TEXT NOT NULL,
    status         TEXT NOT NULL DEFAULT 'new',
    event_date     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    rule_id        TEXT,
    current_value  DOUBLE PRECISION DEFAULT 0,
    baseline_value DOUBLE PRECISION DEFAULT 0
);

CREATE TABLE IF NOT EXISTS ops.task (
    task_id             TEXT PRIMARY KEY,
    task_title          TEXT NOT NULL,
    status              TEXT DEFAULT 'todo',
    owner_role          TEXT,
    priority            TEXT DEFAULT 'medium',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    task_description    TEXT,
    due_at              TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS ops.outbox_event (
    event_id            TEXT PRIMARY KEY,
    event_type          TEXT NOT NULL,
    source_type         TEXT NOT NULL DEFAULT 'system',
    source_id           TEXT NOT NULL DEFAULT 'unknown',
    payload_json        JSONB NOT NULL DEFAULT '{}',
    target_channel      TEXT NOT NULL DEFAULT 'console',
    status              TEXT DEFAULT 'pending',
    dispatch_attempts   BIGINT DEFAULT 0,
    last_dispatch_at    TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`

func setupContextTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx := context.Background()
	pg, err := testutil.StartPostgres(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	pool, err := pgxpool.New(ctx, pg.ConnectionString())
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	_, err = pool.Exec(ctx, contextTableDDL)
	require.NoError(t, err)

	return pool
}

func insertPipelineRun(t *testing.T, pool *pgxpool.Pool, runID, runType, mode, status string, startedAt time.Time) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO audit.pipeline_run (run_id, run_type, mode, status, started_at, input_count, output_count)
		VALUES ($1, $2, $3, $4, $5, 100, 50)
	`, runID, runType, mode, status, startedAt)
	require.NoError(t, err)
}

func insertAlertForContext(t *testing.T, pool *pgxpool.Pool, alertID, severity, metric, status string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO ops.metric_alert (alert_id, severity, metric_name, status, event_date)
		VALUES ($1, $2, $3, $4, NOW())
	`, alertID, severity, metric, status)
	require.NoError(t, err)
}

func insertTaskForContext(t *testing.T, pool *pgxpool.Pool, taskID, status, owner string, createdAt time.Time) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO ops.task (task_id, task_title, status, owner_role, priority, created_at)
		VALUES ($1, $2, $3, $4, 'high', $5)
	`, taskID, "Test task", status, owner, createdAt)
	require.NoError(t, err)
}

func insertOutboxForContext(t *testing.T, pool *pgxpool.Pool, eventID, eventType, status string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO ops.outbox_event (event_id, event_type, source_type, source_id, payload_json, target_channel, status)
		VALUES ($1, $2, 'test', 'src-1', '{}', 'feishu', $3)
	`, eventID, eventType, status)
	require.NoError(t, err)
}

func TestContextRepository_GetLastPipelineRun_Found(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupContextTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC()
	insertPipelineRun(t, pool, "run-1", "ingest", "auto", "completed", now.Add(-1*time.Hour))

	repo := NewContextRepository()
	info, err := repo.GetLastPipelineRun(ctx, pool)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, int64(1), info.RunID)
	assert.Equal(t, "completed", info.Status)
	assert.NotEmpty(t, info.StartedAt)
}

func TestContextRepository_GetLastPipelineRun_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupContextTestDB(t)
	ctx := context.Background()

	repo := NewContextRepository()
	info, err := repo.GetLastPipelineRun(ctx, pool)
	require.NoError(t, err)
	assert.Nil(t, info)
}

func TestContextRepository_GetLastPipelineRun_ReturnsLatest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupContextTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()

	insertPipelineRun(t, pool, "run-old", "ingest", "auto", "completed", now.Add(-2*time.Hour))
	insertPipelineRun(t, pool, "run-new", "metrics", "auto", "running", now.Add(-30*time.Minute))

	repo := NewContextRepository()
	info, err := repo.GetLastPipelineRun(ctx, pool)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "running", info.Status)
}

func TestContextRepository_GetAlerts_All(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupContextTestDB(t)
	ctx := context.Background()

	insertAlertForContext(t, pool, "alert-1", "critical", "gmv", "open")
	insertAlertForContext(t, pool, "alert-2", "warning", "review_score", "acknowledged")

	repo := NewContextRepository()
	alerts, err := repo.GetAlerts(ctx, pool, "", 10)
	require.NoError(t, err)
	assert.Len(t, alerts, 2)
}

func TestContextRepository_GetAlerts_BySeverity(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupContextTestDB(t)
	ctx := context.Background()

	insertAlertForContext(t, pool, "alert-1", "critical", "gmv", "open")
	insertAlertForContext(t, pool, "alert-2", "warning", "review_score", "acknowledged")

	repo := NewContextRepository()
	alerts, err := repo.GetAlerts(ctx, pool, "critical", 10)
	require.NoError(t, err)
	assert.Len(t, alerts, 1)
	assert.Equal(t, "critical", alerts[0].Severity)
}

func TestContextRepository_GetAlerts_RespectsLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupContextTestDB(t)
	ctx := context.Background()

	insertAlertForContext(t, pool, "alert-1", "critical", "gmv", "open")
	insertAlertForContext(t, pool, "alert-2", "critical", "order_count", "open")

	repo := NewContextRepository()
	alerts, err := repo.GetAlerts(ctx, pool, "", 1)
	require.NoError(t, err)
	assert.Len(t, alerts, 1)
}

func TestContextRepository_GetAlerts_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupContextTestDB(t)
	ctx := context.Background()

	repo := NewContextRepository()
	alerts, err := repo.GetAlerts(ctx, pool, "", 10)
	require.NoError(t, err)
	assert.Empty(t, alerts)
}

func TestContextRepository_GetAlerts_NilPool(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	repo := NewContextRepository()
	alerts, err := repo.GetAlerts(context.Background(), nil, "", 10)
	require.NoError(t, err)
	assert.Empty(t, alerts)
}

func TestContextRepository_GetOpenTasks_WithData(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupContextTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()

	insertTaskForContext(t, pool, "task-1", "todo", "logistics_ops", now.Add(-1*time.Hour))
	insertTaskForContext(t, pool, "task-2", "in_progress", "seller_ops", now)
	insertTaskForContext(t, pool, "task-3", "done", "logistics_ops", now.Add(-2*time.Hour))

	repo := NewContextRepository()
	tasks, err := repo.GetOpenTasks(ctx, pool, 10)
	require.NoError(t, err)
	assert.Len(t, tasks, 2)
}

func TestContextRepository_GetOpenTasks_RespectsLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupContextTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()

	insertTaskForContext(t, pool, "task-1", "todo", "ops", now.Add(-2*time.Hour))
	insertTaskForContext(t, pool, "task-2", "todo", "ops", now.Add(-1*time.Hour))

	repo := NewContextRepository()
	tasks, err := repo.GetOpenTasks(ctx, pool, 1)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
}

func TestContextRepository_GetOpenTasks_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupContextTestDB(t)
	ctx := context.Background()

	repo := NewContextRepository()
	tasks, err := repo.GetOpenTasks(ctx, pool, 10)
	require.NoError(t, err)
	assert.Empty(t, tasks)
}

func TestContextRepository_GetOpenTasks_NilPool(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	repo := NewContextRepository()
	tasks, err := repo.GetOpenTasks(context.Background(), nil, 10)
	require.NoError(t, err)
	assert.Empty(t, tasks)
}

func TestContextRepository_GetPendingOutbox_WithData(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupContextTestDB(t)
	ctx := context.Background()

	insertOutboxForContext(t, pool, "evt-1", "alert.triggered", "pending")
	insertOutboxForContext(t, pool, "evt-2", "alert.triggered", "dispatched")
	insertOutboxForContext(t, pool, "evt-3", "task.assigned", "pending")

	repo := NewContextRepository()
	events, err := repo.GetPendingOutbox(ctx, pool, 10)
	require.NoError(t, err)
	assert.Len(t, events, 2)
}

func TestContextRepository_GetPendingOutbox_RespectsLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupContextTestDB(t)
	ctx := context.Background()

	insertOutboxForContext(t, pool, "evt-1", "alert", "pending")
	insertOutboxForContext(t, pool, "evt-2", "task", "pending")

	repo := NewContextRepository()
	events, err := repo.GetPendingOutbox(ctx, pool, 1)
	require.NoError(t, err)
	assert.Len(t, events, 1)
}

func TestContextRepository_GetPendingOutbox_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupContextTestDB(t)
	ctx := context.Background()

	repo := NewContextRepository()
	events, err := repo.GetPendingOutbox(ctx, pool, 10)
	require.NoError(t, err)
	assert.Empty(t, events)
}

func TestContextRepository_GetPendingOutbox_NilPool(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	repo := NewContextRepository()
	events, err := repo.GetPendingOutbox(context.Background(), nil, 10)
	require.NoError(t, err)
	assert.Empty(t, events)
}
