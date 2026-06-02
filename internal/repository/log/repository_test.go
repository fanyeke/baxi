package log

import (
	"context"

	"baxi/internal/testutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/repository/common"
)

const logDDL = `
CREATE SCHEMA IF NOT EXISTS audit;
CREATE TABLE IF NOT EXISTS audit.api_request_log (
    id          BIGSERIAL PRIMARY KEY,
    method      TEXT,
    path        TEXT,
    status_code INT,
    duration_ms BIGINT,
    request_id  TEXT,
    started_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS audit.pipeline_run (
    run_id       BIGSERIAL PRIMARY KEY,
    run_type     TEXT,
    mode         TEXT,
    status       TEXT,
    started_at   TIMESTAMPTZ,
    finished_at  TIMESTAMPTZ,
    input_count  BIGINT DEFAULT 0,
    output_count BIGINT DEFAULT 0,
    error_message TEXT,
    created_at   TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS audit.pipeline_step_run (
    id           BIGSERIAL PRIMARY KEY,
    run_id       BIGINT,
    step_name    TEXT,
    status       TEXT,
    started_at   TIMESTAMPTZ,
    finished_at  TIMESTAMPTZ,
    error_message TEXT,
    details_json JSONB,
    created_at   TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS audit.error_log (
    id            BIGSERIAL PRIMARY KEY,
    level         TEXT,
    message       TEXT,
    request_id    TEXT,
    stack_trace   TEXT,
    error_message TEXT,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS audit.audit_log (
    id            BIGSERIAL PRIMARY KEY,
    action        TEXT,
    resource_type TEXT,
    entity_type   TEXT,
    entity_id     TEXT,
    actor         TEXT,
    details_json  JSONB,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);
`
func setupRepo(t *testing.T) (*Repository, *common.PoolProvider) {
	t.Helper()
	pool := testutil.SetupTestPool(t)
	ctx := context.Background()
	_, err := pool.Exec(ctx, logDDL)
	require.NoError(t, err)
	for _, tbl := range []string{"audit.api_request_log", "audit.pipeline_run", "audit.pipeline_step_run", "audit.error_log", "audit.audit_log"} {
		_, _ = pool.Exec(ctx, "TRUNCATE TABLE "+tbl+" CASCADE")
	}
	return NewRepository(common.NewPoolProvider(pool)), common.NewPoolProvider(pool)
}

func TestLogListRecent_Empty(t *testing.T) {
	repo, _ := setupRepo(t)
	ctx := context.Background()
	logs, total, err := repo.ListRecentLogs(ctx, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, logs)
}

func TestLogListRecent(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	pool.Exec(ctx, `INSERT INTO audit.api_request_log(method,path,status_code,created_at) VALUES('GET','/health',200,NOW())`)
	logs, total, err := repo.ListRecentLogs(ctx, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, logs, 1)
	assert.Equal(t, "api_request", logs[0].LogType)
}

func TestLogListErrorLogs_Empty(t *testing.T) {
	repo, _ := setupRepo(t)
	ctx := context.Background()
	logs, total, err := repo.ListErrorLogs(ctx, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, logs)
}

func TestLogListErrorLogs(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	pool.Exec(ctx, `INSERT INTO audit.error_log(error_message,created_at) VALUES('something broke',NOW())`)
	logs, total, err := repo.ListErrorLogs(ctx, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, logs, 1)
}

func TestLogListAuditLogs_Empty(t *testing.T) {
	repo, _ := setupRepo(t)
	ctx := context.Background()
	logs, total, err := repo.ListAuditLogs(ctx, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, logs)
}

func TestLogListAuditLogs(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	pool.Exec(ctx, `INSERT INTO audit.audit_log(action,entity_type,entity_id,actor,created_at) VALUES('create','case','c1','admin',NOW())`)
	logs, total, err := repo.ListAuditLogs(ctx, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, logs, 1)
	assert.Equal(t, "audit_log", logs[0].LogType)
}
