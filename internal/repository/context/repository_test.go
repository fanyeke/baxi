package context

import (
	"context"

	"baxi/internal/testutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/repository/common"
)

const ctxDDL = `
CREATE SCHEMA IF NOT EXISTS ai;
CREATE SCHEMA IF NOT EXISTS audit;
CREATE SCHEMA IF NOT EXISTS ops;
CREATE TABLE IF NOT EXISTS ai.context_build (
    context_id    TEXT PRIMARY KEY,
    case_id       TEXT,
    build_type    TEXT,
    status        TEXT,
    input_json    JSONB,
    output_json   JSONB,
    created_at    TIMESTAMPTZ DEFAULT NOW()
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
CREATE TABLE IF NOT EXISTS ops.metric_alert (
    alert_id     TEXT PRIMARY KEY,
    severity     TEXT NOT NULL,
    metric_name  TEXT NOT NULL,
    status       TEXT DEFAULT 'new',
    event_date   DATE,
    owner_role   TEXT,
    created_at   TIMESTAMPTZ DEFAULT NOW()
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
    event_id      TEXT PRIMARY KEY,
    event_type    TEXT NOT NULL,
    status        TEXT DEFAULT 'pending',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`
func setupRepo(t *testing.T) (*Repository, *common.PoolProvider) {
	t.Helper()
	pool := testutil.SetupTestPool(t)
	ctx := context.Background()
	_, err := pool.Exec(ctx, ctxDDL)
	require.NoError(t, err)
	for _, tbl := range []string{"audit.pipeline_run", "ops.metric_alert", "ops.task", "ops.outbox_event"} {
		_, _ = pool.Exec(ctx, "TRUNCATE TABLE "+tbl+" CASCADE")
	}
	return NewRepository(common.NewPoolProvider(pool)), common.NewPoolProvider(pool)
}

func TestContextGetLastPipelineRun_Empty(t *testing.T) {
	repo, _ := setupRepo(t)
	ctx := context.Background()
	r, err := repo.GetLastPipelineRun(ctx)
	require.NoError(t, err)
	assert.Nil(t, r)
}

func TestContextGetLastPipelineRun(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	pool.Exec(ctx, `INSERT INTO audit.pipeline_run(run_id,run_type,mode,status,started_at) VALUES('1','full','auto','completed',NOW())`)
	r, err := repo.GetLastPipelineRun(ctx)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "completed", r.Status)
}

func TestContextGetAlerts(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	pool.Exec(ctx, `INSERT INTO ops.metric_alert(alert_id,severity,metric_name,status) VALUES('a1','high','m1','new')`)
	pool.Exec(ctx, `INSERT INTO ops.metric_alert(alert_id,severity,metric_name,status) VALUES('a2','low','m2','resolved')`)
	alerts, err := repo.GetAlerts(ctx, "", 10)
	require.NoError(t, err)
	assert.Len(t, alerts, 2)
}

func TestContextGetAlertsFiltered(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	pool.Exec(ctx, `INSERT INTO ops.metric_alert(alert_id,severity,metric_name,status) VALUES('a1','high','m1','new')`)
	pool.Exec(ctx, `INSERT INTO ops.metric_alert(alert_id,severity,metric_name,status) VALUES('a2','low','m2','new')`)
	alerts, err := repo.GetAlerts(ctx, "high", 10)
	require.NoError(t, err)
	assert.Len(t, alerts, 1)
	assert.Equal(t, "a1", alerts[0].AlertID)
}

func TestContextGetOpenTasks(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	pool.Exec(ctx, `INSERT INTO ops.task(task_id,task_title,status,owner_role) VALUES('t1','Do X','todo','admin')`)
	pool.Exec(ctx, `INSERT INTO ops.task(task_id,task_title,status,owner_role) VALUES('t2','Do Y','completed','user')`)
	tasks, err := repo.GetOpenTasks(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "t1", tasks[0].TaskID)
}

func TestContextGetPendingOutbox(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	pool.Exec(ctx, `INSERT INTO ops.outbox_event(event_id,event_type,status) VALUES('o1','alert','pending')`)
	pool.Exec(ctx, `INSERT INTO ops.outbox_event(event_id,event_type,status) VALUES('o2','alert','dispatched')`)
	events, err := repo.GetPendingOutbox(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, "o1", events[0].EventID)
}
