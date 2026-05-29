package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const repoLogTableDDL = `
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

func setupRepoLogTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	ctx := context.Background()
	_, err = pool.Exec(ctx, repoLogTableDDL)
	require.NoError(t, err)

	_, _ = pool.Exec(ctx, "TRUNCATE TABLE audit.api_request_log, audit.pipeline_run, audit.pipeline_step_run, audit.error_log, audit.audit_log CASCADE")

	return pool
}

func insertRepoLogTestData(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()

	_, _ = pool.Exec(ctx, `
		INSERT INTO audit.api_request_log (request_id, method, path, status_code, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, "req-1", "GET", "/api/v1/health", 200, now.Add(-1*time.Hour))

	_, _ = pool.Exec(ctx, `
		INSERT INTO audit.pipeline_run (run_id, run_type, mode, status, started_at)
		VALUES ($1, $2, $3, $4, $5)
	`, "run-1", "ingest", "auto", "completed", now.Add(-2*time.Hour))

	_, _ = pool.Exec(ctx, `
		INSERT INTO audit.pipeline_step_run (step_run_id, pipeline_run_id, step_name, step_order, status, started_at, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, "step-1", "run-1", "ingest_raw", 1, "completed", now.Add(-2*time.Hour), nil)

	_, _ = pool.Exec(ctx, `
		INSERT INTO audit.pipeline_step_run (step_run_id, pipeline_run_id, step_name, step_order, status, started_at, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, "step-2", "run-1", "build_dwd", 2, "failed", now.Add(-90*time.Minute), "null value in column")

	_, _ = pool.Exec(ctx, `
		INSERT INTO audit.error_log (request_id, error_type, error_message, created_at)
		VALUES ($1, $2, $3, $4)
	`, "req-err-1", "database_error", "connection refused", now.Add(-30*time.Minute))

	_, _ = pool.Exec(ctx, `
		INSERT INTO audit.audit_log (category, action, actor, resource_type, resource_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, "dispatch", "send", "system", "outbox", "evt-1", now.Add(-10*time.Minute))
}

func TestLogRepository_ListRecentLogs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupRepoLogTestDB(t)
	insertRepoLogTestData(t, pool)
	repo := &LogRepository{}

	rows, total, err := repo.ListRecentLogs(context.Background(), pool, 10, 0)
	require.NoError(t, err)
	assert.Greater(t, len(rows), 0)
	assert.Greater(t, total, 0)

	// Should contain rows from all 3 sources
	types := map[string]bool{}
	for _, r := range rows {
		types[r.LogType] = true
		// Verify non-empty message
		assert.NotEmpty(t, r.Message)
		// Verify createdAt is populated
		assert.False(t, r.CreatedAt.IsZero())
	}
	assert.True(t, types["api_request"])
	assert.True(t, types["pipeline_run"])
	assert.True(t, types["pipeline_step"])

	// Verify ordering
	for i := 1; i < len(rows); i++ {
		assert.False(t, rows[i].CreatedAt.After(rows[i-1].CreatedAt),
			"rows should be ordered by created_at DESC")
	}
}

func TestLogRepository_ListRecentLogs_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupRepoLogTestDB(t)
	repo := &LogRepository{}

	rows, total, err := repo.ListRecentLogs(context.Background(), pool, 10, 0)
	require.NoError(t, err)
	assert.Empty(t, rows)
	assert.Equal(t, 0, total)
}

func TestLogRepository_ListRecentLogs_Pagination(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupRepoLogTestDB(t)
	insertRepoLogTestData(t, pool)
	repo := &LogRepository{}

	// limit 1
	rows, total, err := repo.ListRecentLogs(context.Background(), pool, 1, 0)
	require.NoError(t, err)
	assert.Len(t, rows, 1)
	assert.Greater(t, total, 1)

	// offset 1
	rows2, _, err := repo.ListRecentLogs(context.Background(), pool, 10, 1)
	require.NoError(t, err)
	assert.Greater(t, len(rows2), 0)

	// Should be different first item
	if len(rows) > 0 && len(rows2) > 0 {
		assert.NotEqual(t, rows[0].CreatedAt, rows2[0].CreatedAt)
	}
}

func TestLogRepository_ListErrorLogs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupRepoLogTestDB(t)
	insertRepoLogTestData(t, pool)
	repo := &LogRepository{}

	rows, total, err := repo.ListErrorLogs(context.Background(), pool, 10, 0)
	require.NoError(t, err)
	assert.Greater(t, len(rows), 0)
	assert.Greater(t, total, 0)

	types := map[string]bool{}
	for _, r := range rows {
		assert.Equal(t, "error", r.Level)
		types[r.LogType] = true
	}
	assert.True(t, types["error_log"])
	assert.True(t, types["pipeline_step"])
}

func TestLogRepository_ListErrorLogs_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupRepoLogTestDB(t)
	repo := &LogRepository{}

	rows, total, err := repo.ListErrorLogs(context.Background(), pool, 10, 0)
	require.NoError(t, err)
	assert.Empty(t, rows)
	assert.Equal(t, 0, total)
}

func TestLogRepository_ListAuditLogs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupRepoLogTestDB(t)
	insertRepoLogTestData(t, pool)
	repo := &LogRepository{}

	rows, total, err := repo.ListAuditLogs(context.Background(), pool, 10, 0)
	require.NoError(t, err)
	assert.Len(t, rows, 1)
	assert.Equal(t, 1, total)
	assert.Equal(t, "audit_log", rows[0].LogType)
	assert.Equal(t, "info", rows[0].Level)
	assert.Equal(t, "send on outbox", rows[0].Message)
	assert.Nil(t, rows[0].RequestID)
}

func TestLogRepository_ListAuditLogs_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupRepoLogTestDB(t)
	repo := &LogRepository{}

	rows, total, err := repo.ListAuditLogs(context.Background(), pool, 10, 0)
	require.NoError(t, err)
	assert.Empty(t, rows)
	assert.Equal(t, 0, total)
}

func TestLogRepository_ListRecentLogs_LevelMapping(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupRepoLogTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	repo := &LogRepository{}

	// Insert entries with different status codes / statuses
	_, _ = pool.Exec(ctx, `INSERT INTO audit.api_request_log (request_id, method, path, status_code, created_at) VALUES ($1,$2,$3,$4,$5)`,
		"req-200", "GET", "/ok", 200, now)
	_, _ = pool.Exec(ctx, `INSERT INTO audit.api_request_log (request_id, method, path, status_code, created_at) VALUES ($1,$2,$3,$4,$5)`,
		"req-302", "GET", "/redirect", 302, now)
	_, _ = pool.Exec(ctx, `INSERT INTO audit.api_request_log (request_id, method, path, status_code, created_at) VALUES ($1,$2,$3,$4,$5)`,
		"req-500", "GET", "/error", 500, now)
	_, _ = pool.Exec(ctx, `INSERT INTO audit.pipeline_run (run_id, run_type, mode, status, started_at) VALUES ($1,$2,$3,$4,$5)`,
		"run-ok", "test", "auto", "completed", now)
	_, _ = pool.Exec(ctx, `INSERT INTO audit.pipeline_run (run_id, run_type, mode, status, started_at) VALUES ($1,$2,$3,$4,$5)`,
		"run-fail", "test", "auto", "failed", now)
	_, _ = pool.Exec(ctx, `INSERT INTO audit.pipeline_run (run_id, run_type, mode, status, started_at) VALUES ($1,$2,$3,$4,$5)`,
		"run-pending", "test", "auto", "running", now)

	rows, _, err := repo.ListRecentLogs(ctx, pool, 10, 0)
	require.NoError(t, err)

	levelCount := map[string]int{}
	for _, r := range rows {
		levelCount[r.Level]++
	}

	assert.GreaterOrEqual(t, levelCount["info"], 2)  // 200 + completed
	assert.GreaterOrEqual(t, levelCount["warn"], 1)  // 302
	assert.GreaterOrEqual(t, levelCount["error"], 1) // 500 + failed
}
