package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/model"
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
		fn   func(ctx context.Context, limit, offset int) (*model.LogListResponse, error)
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

// === File-based log reading tests (migrated from Python log_reader.py) ===

func TestTailJSONL_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	lines := []string{
		`{"ts":"2024-01-01T00:00:00Z","msg":"first"}`,
		`{"ts":"2024-01-01T00:01:00Z","msg":"second"}`,
		`{"ts":"2024-01-01T00:02:00Z","msg":"third"}`,
	}
	content := strings.Join(lines, "\n") + "\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	entries, err := tailJSONL(path, 2)
	require.NoError(t, err)
	assert.Len(t, entries, 2)
	assert.Equal(t, "third", entries[0]["msg"])
	assert.Equal(t, "second", entries[1]["msg"])
}

func TestTailJSONL_MissingFile(t *testing.T) {
	entries, err := tailJSONL("/nonexistent/path/file.jsonl", 10)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestTailJSONL_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.jsonl")
	require.NoError(t, os.WriteFile(path, []byte{}, 0644))

	entries, err := tailJSONL(path, 10)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestTailJSONL_MalformedLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "malformed.jsonl")

	lines := []string{
		`{"ts":"2024-01-01T00:00:00Z","msg":"valid"}`,
		`this is not json`,
		`{"ts":"2024-01-01T00:02:00Z","msg":"also valid"}`,
	}
	content := strings.Join(lines, "\n") + "\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	entries, err := tailJSONL(path, 10)
	require.NoError(t, err)
	assert.Len(t, entries, 2)
	assert.Equal(t, "also valid", entries[0]["msg"])
	assert.Equal(t, "valid", entries[1]["msg"])
}

func TestTailJSONL_ExactLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	lines := []string{
		`{"msg":"1"}`,
		`{"msg":"2"}`,
		`{"msg":"3"}`,
		`{"msg":"4"}`,
		`{"msg":"5"}`,
	}
	content := strings.Join(lines, "\n") + "\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	entries, err := tailJSONL(path, 3)
	require.NoError(t, err)
	assert.Len(t, entries, 3)
	assert.Equal(t, "5", entries[0]["msg"])
	assert.Equal(t, "4", entries[1]["msg"])
	assert.Equal(t, "3", entries[2]["msg"])
}

func TestTailJSONL_ZeroLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(`{"msg":"x"}`+"\n"), 0644))

	entries, err := tailJSONL(path, 0)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestTailJSONL_NoTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	lines := []string{
		`{"msg":"first"}`,
		`{"msg":"second"}`,
	}
	content := strings.Join(lines, "\n") // no trailing newline
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	entries, err := tailJSONL(path, 10)
	require.NoError(t, err)
	assert.Len(t, entries, 2)
	assert.Equal(t, "second", entries[0]["msg"])
	assert.Equal(t, "first", entries[1]["msg"])
}

func TestLogService_ReadLogErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "error.log")

	lines := []string{
		`{"ts":"2024-01-01T00:00:00Z","request_id":"req-1","msg":"error1"}`,
		`{"ts":"2024-01-01T00:01:00Z","request_id":"req-2","msg":"error2"}`,
		`{"ts":"2024-01-01T00:02:00Z","request_id":"req-1","msg":"error3"}`,
	}
	content := strings.Join(lines, "\n") + "\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	svc := NewLogService(nil, nil)

	// No filter
	entries, err := svc.ReadLogErrors(path, nil, 10)
	require.NoError(t, err)
	assert.Len(t, entries, 3)

	// Filter by request_id
	reqID := "req-1"
	entries, err = svc.ReadLogErrors(path, &reqID, 10)
	require.NoError(t, err)
	assert.Len(t, entries, 2)
	assert.Equal(t, "error3", entries[0]["msg"])
	assert.Equal(t, "error1", entries[1]["msg"])
}

func TestLogService_ReadLogErrors_MissingFile(t *testing.T) {
	svc := NewLogService(nil, nil)
	entries, err := svc.ReadLogErrors("/nonexistent/error.log", nil, 10)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestLogService_ReadLogErrors_LimitCap(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "error.log")

	var lines []string
	for i := 0; i < 10; i++ {
		lines = append(lines, `{"msg":"error"}`)
	}
	content := strings.Join(lines, "\n") + "\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	svc := NewLogService(nil, nil)
	entries, err := svc.ReadLogErrors(path, nil, 1000)
	require.NoError(t, err)
	assert.Len(t, entries, 10) // capped at 500 but we only have 10
}

func TestLogService_ReadLogRecent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "api.log")

	lines := []string{
		`{"ts":"2024-01-01T00:00:00Z","msg":"api1"}`,
		`{"ts":"2024-01-01T00:01:00Z","msg":"api2"}`,
		`{"ts":"2024-01-01T00:02:00Z","msg":"api3"}`,
	}
	content := strings.Join(lines, "\n") + "\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	svc := NewLogService(nil, nil)
	entries, err := svc.ReadLogRecent(path, 2)
	require.NoError(t, err)
	assert.Len(t, entries, 2)
	assert.Equal(t, "api3", entries[0]["msg"])
	assert.Equal(t, "api2", entries[1]["msg"])
}

func TestLogService_ReadLogRecent_ZeroLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "api.log")
	require.NoError(t, os.WriteFile(path, []byte(`{"msg":"x"}`+"\n"), 0644))

	svc := NewLogService(nil, nil)
	entries, err := svc.ReadLogRecent(path, 0)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestLogService_ReadAuditLogs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.csv")

	content := "timestamp,outbox_id,status,action\n" +
		"2024-01-01T00:02:00Z,ob-1,sent,send\n" +
		"2024-01-01T00:01:00Z,ob-2,failed,retry\n" +
		"2024-01-01T00:03:00Z,ob-1,sent,send\n" +
		"2024-01-01T00:00:00Z,ob-3,pending,queue\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	svc := NewLogService(nil, nil)

	// No filter
	entries, err := svc.ReadAuditLogs(path, nil, nil, 10)
	require.NoError(t, err)
	assert.Len(t, entries, 4)
	// Should be sorted by timestamp desc (newest first)
	assert.Equal(t, "2024-01-01T00:03:00Z", entries[0]["timestamp"])
	assert.Equal(t, "2024-01-01T00:02:00Z", entries[1]["timestamp"])
	assert.Equal(t, "2024-01-01T00:01:00Z", entries[2]["timestamp"])
	assert.Equal(t, "2024-01-01T00:00:00Z", entries[3]["timestamp"])

	// Filter by outbox_id
	obID := "ob-1"
	entries, err = svc.ReadAuditLogs(path, &obID, nil, 10)
	require.NoError(t, err)
	assert.Len(t, entries, 2)
	for _, e := range entries {
		assert.Equal(t, "ob-1", e["outbox_id"])
	}

	// Filter by status
	st := "sent"
	entries, err = svc.ReadAuditLogs(path, nil, &st, 10)
	require.NoError(t, err)
	assert.Len(t, entries, 2)
	for _, e := range entries {
		assert.Equal(t, "sent", e["status"])
	}

	// Filter by both
	entries, err = svc.ReadAuditLogs(path, &obID, &st, 10)
	require.NoError(t, err)
	assert.Len(t, entries, 2)
	for _, e := range entries {
		assert.Equal(t, "ob-1", e["outbox_id"])
		assert.Equal(t, "sent", e["status"])
	}
}

func TestLogService_ReadAuditLogs_MissingFile(t *testing.T) {
	svc := NewLogService(nil, nil)
	entries, err := svc.ReadAuditLogs("/nonexistent/audit.csv", nil, nil, 10)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestLogService_ReadAuditLogs_EmptyCSV(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.csv")
	require.NoError(t, os.WriteFile(path, []byte(""), 0644))

	svc := NewLogService(nil, nil)
	entries, err := svc.ReadAuditLogs(path, nil, nil, 10)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestLogService_ReadAuditLogs_Limit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.csv")

	content := "timestamp,outbox_id,status\n"
	for i := 0; i < 10; i++ {
		content += "2024-01-01T00:00:00Z,ob-x,sent\n"
	}
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	svc := NewLogService(nil, nil)
	entries, err := svc.ReadAuditLogs(path, nil, nil, 5)
	require.NoError(t, err)
	assert.Len(t, entries, 5)
}

func TestLogService_ReadAuditLogs_LimitCap(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.csv")

	content := "timestamp,outbox_id,status\n"
	for i := 0; i < 10; i++ {
		content += "2024-01-01T00:00:00Z,ob-x,sent\n"
	}
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	svc := NewLogService(nil, nil)
	entries, err := svc.ReadAuditLogs(path, nil, nil, 1000)
	require.NoError(t, err)
	assert.Len(t, entries, 10) // capped at 500 but we only have 10
}
