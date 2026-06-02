package status

import (
	"context"

	"baxi/internal/testutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/repository/common"
)

const statusDDL = `
CREATE SCHEMA IF NOT EXISTS audit;
CREATE SCHEMA IF NOT EXISTS ops;
CREATE TABLE IF NOT EXISTS ops.metric_alert (
    alert_id    TEXT PRIMARY KEY,
    rule_id     TEXT,
    metric_name TEXT,
    severity    TEXT,
    current_value DOUBLE PRECISION,
    baseline_value DOUBLE PRECISION,
    status      TEXT,
    owner_role  TEXT,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS ops.task (
    task_id   TEXT PRIMARY KEY,
    task_title TEXT,
    status    TEXT DEFAULT 'todo',
    priority  TEXT DEFAULT 'medium',
    owner_role TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS ops.outbox_event (
    event_id  TEXT PRIMARY KEY,
    event_type TEXT,
    status    TEXT DEFAULT 'pending',
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE SCHEMA IF NOT EXISTS dwd;
CREATE SCHEMA IF NOT EXISTS mart;
CREATE TABLE IF NOT EXISTS dwd.order_level (
    order_id TEXT
);
CREATE TABLE IF NOT EXISTS dwd.item_level (
    order_id TEXT
);
CREATE TABLE IF NOT EXISTS mart.metric_snapshot (
    metric_name TEXT
);
CREATE TABLE IF NOT EXISTS mart.metric_dimension_daily (
    metric_name TEXT
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
`
func setupRepo(t *testing.T) (*Repository, *common.PoolProvider) {
	t.Helper()
	pool := testutil.SetupTestPool(t)
	ctx := context.Background()
	_, err := pool.Exec(ctx, statusDDL)
	require.NoError(t, err)
	for _, tbl := range []string{"ops.metric_alert", "ops.task", "ops.outbox_event", "audit.pipeline_run"} {
		_, _ = pool.Exec(ctx, "TRUNCATE TABLE "+tbl+" CASCADE")
	}
	return NewRepository(common.NewPoolProvider(pool)), common.NewPoolProvider(pool)
}

func TestStatusGetTableCounts(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	pool.Exec(ctx, `INSERT INTO ops.metric_alert(alert_id,owner_role) VALUES('a1','owner')`)
	pool.Exec(ctx, `INSERT INTO ops.task(task_id) VALUES('t1')`)
	pool.Exec(ctx, `INSERT INTO ops.outbox_event(event_id) VALUES('o1')`)
	counts, err := repo.GetTableCounts(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(counts), 3)
	m := make(map[string]int)
	for _, c := range counts { m[c.TableName] = c.RowCount }
	assert.Equal(t, 1, m["alert_events"])
	assert.Equal(t, 1, m["action_tasks"])
	assert.Equal(t, 1, m["event_outbox"])
}

func TestStatusGetLastPipelineRun_Empty(t *testing.T) {
	repo, _ := setupRepo(t)
	ctx := context.Background()
	run, err := repo.GetLastPipelineRun(ctx)
	require.NoError(t, err)
	assert.Nil(t, run)
}

func TestStatusGetLastPipelineRun(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	pool.Exec(ctx, `INSERT INTO audit.pipeline_run(run_type,mode,status,started_at) VALUES('full','auto','completed',NOW())`)
	run, err := repo.GetLastPipelineRun(ctx)
	require.NoError(t, err)
	require.NotNil(t, run)
	assert.Equal(t, "completed", run.Status)
}
