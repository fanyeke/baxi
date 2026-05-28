package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/api/dto"
	"baxi/internal/repository"
)

const svcLogTableDDL = `
CREATE SCHEMA IF NOT EXISTS audit;

CREATE TABLE IF NOT EXISTS audit.api_request_log (
    log_id            BIGSERIAL PRIMARY KEY,
    request_id        TEXT,
    method            TEXT,
    path              TEXT,
    status_code       BIGINT,
    user_agent        TEXT,
    client_ip         TEXT,
    request_body_json JSONB,
    response_body_json JSONB,
    duration_ms       BIGINT,
    created_at        TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS audit.pipeline_run (
    run_id        TEXT PRIMARY KEY,
    run_type      TEXT NOT NULL,
    mode          TEXT NOT NULL,
    status        TEXT NOT NULL,
    started_at    TIMESTAMPTZ NOT NULL,
    finished_at   TIMESTAMPTZ,
    input_count   BIGINT DEFAULT 0,
    output_count  BIGINT DEFAULT 0,
    error_message TEXT
);

CREATE TABLE IF NOT EXISTS audit.pipeline_step_run (
    step_run_id    TEXT PRIMARY KEY,
    pipeline_run_id TEXT,
    step_name      TEXT,
    step_order     BIGINT,
    status         TEXT,
    started_at     TIMESTAMPTZ,
    finished_at    TIMESTAMPTZ,
    input_count    BIGINT,
    output_count   BIGINT,
    error_message  TEXT
);

CREATE TABLE IF NOT EXISTS audit.error_log (
    error_id      BIGSERIAL PRIMARY KEY,
    request_id    TEXT,
    error_type    TEXT,
    error_message TEXT,
    stack_trace   TEXT,
    details       JSONB,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS audit.audit_log (
    audit_id      BIGSERIAL PRIMARY KEY,
    category      TEXT,
    action        TEXT,
    actor         TEXT,
    resource_type TEXT,
    resource_id   TEXT,
    metadata      JSONB,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);
`

func setupSvcLogTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	ctx := context.Background()
	_, err = pool.Exec(ctx, svcLogTableDDL)
	require.NoError(t, err)

	// Clean all log tables
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE audit.api_request_log, audit.pipeline_run, audit.pipeline_step_run, audit.error_log, audit.audit_log CASCADE")

	return pool
}

func insertLogTestData(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()

	now := time.Now().UTC()

	// api_request_log
	_, err := pool.Exec(ctx, `
		INSERT INTO audit.api_request_log (request_id, method, path, status_code, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, "req-1", "GET", "/api/v1/health", 200, now.Add(-1*time.Hour))
	require.NoError(t, err)

	// pipeline_run
	_, err = pool.Exec(ctx, `
		INSERT INTO audit.pipeline_run (run_id, run_type, mode, status, started_at)
		VALUES ($1, $2, $3, $4, $5)
	`, "run-1", "ingest", "auto", "completed", now.Add(-2*time.Hour))
	require.NoError(t, err)

	// pipeline_step_run (success and failed)
	_, err = pool.Exec(ctx, `
		INSERT INTO audit.pipeline_step_run (step_run_id, pipeline_run_id, step_name, step_order, status, started_at, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, "step-1", "run-1", "ingest_raw", 1, "completed", now.Add(-2*time.Hour), nil)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO audit.pipeline_step_run (step_run_id, pipeline_run_id, step_name, step_order, status, started_at, finished_at, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, "step-2", "run-1", "build_dwd", 2, "failed", now.Add(-2*time.Hour), now.Add(-1*time.Hour), "null value in column")
	require.NoError(t, err)

	// error_log
	_, err = pool.Exec(ctx, `
		INSERT INTO audit.error_log (request_id, error_type, error_message, created_at)
		VALUES ($1, $2, $3, $4)
	`, "req-err-1", "database_error", "connection refused", now.Add(-30*time.Minute))
	require.NoError(t, err)

	// audit_log
	_, err = pool.Exec(ctx, `
		INSERT INTO audit.audit_log (category, action, actor, resource_type, resource_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, "dispatch", "send", "system", "outbox", "evt-1", now.Add(-10*time.Minute))
	require.NoError(t, err)
}

func TestLogService_ListRecent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupSvcLogTestDB(t)
	insertLogTestData(t, pool)

	repo := repository.NewLogRepository()
	svc := NewLogService(repo, pool)

	resp, err := svc.ListRecent(context.Background(), 10, 0)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Greater(t, len(resp.Items), 0)
	assert.Greater(t, resp.Total, 0)

	// Verify log_type values
	hasAPIRequest := false
	hasPipelineRun := false
	hasPipelineStep := false
	for _, item := range resp.Items {
		switch item.LogType {
		case "api_request":
			hasAPIRequest = true
		case "pipeline_run":
			hasPipelineRun = true
		case "pipeline_step":
			hasPipelineStep = true
		}
	}
	assert.True(t, hasAPIRequest, "should contain api_request log type")
	assert.True(t, hasPipelineRun, "should contain pipeline_run log type")
	assert.True(t, hasPipelineStep, "should contain pipeline_step log type")

	// Verify order (DESC)
	for i := 1; i < len(resp.Items); i++ {
		assert.False(t, resp.Items[i].CreatedAt.After(resp.Items[i-1].CreatedAt),
			"items should be ordered by created_at DESC")
	}
}

func TestLogService_ListRecent_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupSvcLogTestDB(t)
	repo := repository.NewLogRepository()
	svc := NewLogService(repo, pool)

	resp, err := svc.ListRecent(context.Background(), 10, 0)
	require.NoError(t, err)
	assert.Empty(t, resp.Items)
	assert.Equal(t, 0, resp.Total)
}

func TestLogService_ListErrors(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupSvcLogTestDB(t)
	insertLogTestData(t, pool)

	repo := repository.NewLogRepository()
	svc := NewLogService(repo, pool)

	resp, err := svc.ListErrors(context.Background(), 10, 0)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Greater(t, len(resp.Items), 0)
	assert.Greater(t, resp.Total, 0)

	// Should have both error_log and pipeline_step error entries
	hasErrorLog := false
	hasPipelineStep := false
	for _, item := range resp.Items {
		assert.Equal(t, "error", item.Level)
		switch item.LogType {
		case "error_log":
			hasErrorLog = true
			assert.Equal(t, "connection refused", item.Message)
			require.NotNil(t, item.RequestID)
			assert.Equal(t, "req-err-1", *item.RequestID)
		case "pipeline_step":
			hasPipelineStep = true
		}
	}
	assert.True(t, hasErrorLog, "should contain error_log entries")
	assert.True(t, hasPipelineStep, "should contain failed pipeline step entries")
}

func TestLogService_ListErrors_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupSvcLogTestDB(t)
	repo := repository.NewLogRepository()
	svc := NewLogService(repo, pool)

	resp, err := svc.ListErrors(context.Background(), 10, 0)
	require.NoError(t, err)
	assert.Empty(t, resp.Items)
	assert.Equal(t, 0, resp.Total)
}

func TestLogService_ListAudit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupSvcLogTestDB(t)
	insertLogTestData(t, pool)

	repo := repository.NewLogRepository()
	svc := NewLogService(repo, pool)

	resp, err := svc.ListAudit(context.Background(), 10, 0)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, 1, resp.Total)
	assert.Equal(t, "audit_log", resp.Items[0].LogType)
	assert.Equal(t, "info", resp.Items[0].Level)
}

func TestLogService_ListAudit_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupSvcLogTestDB(t)
	repo := repository.NewLogRepository()
	svc := NewLogService(repo, pool)

	resp, err := svc.ListAudit(context.Background(), 10, 0)
	require.NoError(t, err)
	assert.Empty(t, resp.Items)
	assert.Equal(t, 0, resp.Total)
}

func TestLogService_ListRecent_Pagination(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupSvcLogTestDB(t)
	insertLogTestData(t, pool)

	repo := repository.NewLogRepository()
	svc := NewLogService(repo, pool)

	// Get with limit 1
	resp, err := svc.ListRecent(context.Background(), 1, 0)
	require.NoError(t, err)
	assert.Len(t, resp.Items, 1)
	assert.Greater(t, resp.Total, 1) // total should be > 1 even with limit 1

	// Get with offset 1
	resp2, err := svc.ListRecent(context.Background(), 10, 1)
	require.NoError(t, err)
	assert.Greater(t, len(resp2.Items), 0)

	// Items should be different (pagination works)
	if len(resp.Items) > 0 && len(resp2.Items) > 0 {
		assert.NotEqual(t, resp.Items[0].CreatedAt, resp2.Items[0].CreatedAt,
			"offset should return different items")
	}
}

func TestLogService_ListAll_EmptyResponseFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupSvcLogTestDB(t)
	repo := repository.NewLogRepository()
	svc := NewLogService(repo, pool)

	tests := []struct {
		name string
		fn   func(ctx context.Context, limit, offset int) (*dto.LogListResponse, error)
	}{
		{"ListRecent", svc.ListRecent},
		{"ListErrors", svc.ListErrors},
		{"ListAudit", svc.ListAudit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := tt.fn(context.Background(), 10, 0)
			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.NotNil(t, resp.Items, "items should not be nil")
			assert.Empty(t, resp.Items, "items should be empty")
			assert.Equal(t, 0, resp.Total)
		})
	}
}
