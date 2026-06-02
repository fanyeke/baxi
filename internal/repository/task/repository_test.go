package task

import (
	"context"

	"baxi/internal/testutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/repository/common"
)

const taskDDL = `
CREATE SCHEMA IF NOT EXISTS ops;
CREATE TABLE IF NOT EXISTS ops.task (
    task_id             TEXT PRIMARY KEY,
    recommendation_id   TEXT,
    alert_id            TEXT,
    task_title          TEXT NOT NULL,
    task_description    TEXT,
    target_object_type  TEXT,
    target_object_id    TEXT,
    owner_role          TEXT,
    owner_user_id       TEXT,
    priority            TEXT DEFAULT 'medium',
    due_at              TIMESTAMPTZ,
    status              TEXT DEFAULT 'todo',
    feedback            TEXT,
    completed_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`
func setupRepo(t *testing.T) (*Repository, *common.PoolProvider) {
	t.Helper()
	pool := testutil.SetupTestPool(t)
	ctx := context.Background()
	_, err := pool.Exec(ctx, taskDDL)
	require.NoError(t, err)
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE ops.task CASCADE")
	return NewRepository(common.NewPoolProvider(pool)), common.NewPoolProvider(pool)
}

func TestTaskListTasks(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		pool.Exec(ctx, `INSERT INTO ops.task(task_id,task_title,status) VALUES($1,$2,'todo')`, "t"+itoa(i), "task "+itoa(i))
	}
	rows, total, err := repo.ListTasks(ctx, TaskFilters{}, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, rows, 3)
}

func TestTaskFilterByStatus(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	pool.Exec(ctx, `INSERT INTO ops.task(task_id,task_title,status) VALUES('t1','todo task','todo')`)
	pool.Exec(ctx, `INSERT INTO ops.task(task_id,task_title,status) VALUES('t2','done task','completed')`)
	f := TaskFilters{Status: strPtr("todo")}
	rows, total, err := repo.ListTasks(ctx, f, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "t1", rows[0].TaskID)
}

func TestTaskFilterByPriority(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	pool.Exec(ctx, `INSERT INTO ops.task(task_id,task_title,priority,status) VALUES('t1','urgent','high','todo')`)
	pool.Exec(ctx, `INSERT INTO ops.task(task_id,task_title,priority,status) VALUES('t2','normal','medium','todo')`)
	f := TaskFilters{Priority: strPtr("high")}
	rows, total, err := repo.ListTasks(ctx, f, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "t1", rows[0].TaskID)
}

func TestTaskFilterByOwner(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	pool.Exec(ctx, `INSERT INTO ops.task(task_id,task_title,owner_role,status) VALUES('t1','mine','admin','todo')`)
	pool.Exec(ctx, `INSERT INTO ops.task(task_id,task_title,owner_role,status) VALUES('t2','theirs','user','todo')`)
	f := TaskFilters{Owner: strPtr("admin")}
	rows, total, err := repo.ListTasks(ctx, f, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "t1", rows[0].TaskID)
}

func TestTaskPagination(t *testing.T) {
	repo, pool := setupRepo(t)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		pool.Exec(ctx, `INSERT INTO ops.task(task_id,task_title,status) VALUES($1,$2,'todo')`, "tp"+itoa(i), "t "+itoa(i))
	}
	rows, total, err := repo.ListTasks(ctx, TaskFilters{}, 2, 0)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, rows, 2)
}

func strPtr(s string) *string { return &s }
func itoa(n int) string {
	if n == 0 { return "0" }
	s := ""
	for n > 0 { s = string(rune('0'+n%10)) + s; n /= 10 }
	return s
}
