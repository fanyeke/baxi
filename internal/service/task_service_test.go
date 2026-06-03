package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/model"
	"baxi/internal/repository/common"
	taskRepo "baxi/internal/repository/task"
)

const svcTaskTableDDL = `
CREATE SCHEMA IF NOT EXISTS ops;

CREATE TABLE IF NOT EXISTS ops.task (
    task_id             TEXT PRIMARY KEY,
    recommendation_id   TEXT,
    alert_id            TEXT,
    task_title          TEXT NOT NULL,
    task_description    TEXT,
    target_object_type  TEXT,
    target_object_id    TEXT,
    task_source         TEXT DEFAULT 'heuristic_strategy',
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

func setupSvcTaskTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	ctx := context.Background()
	_, err = pool.Exec(ctx, svcTaskTableDDL)
	require.NoError(t, err)
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE ops.task CASCADE")
	return pool
}

func insertSvcTestTask(t *testing.T, pool *pgxpool.Pool, id, title, priority, status, ownerRole string, createdAt time.Time) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO ops.task (task_id, task_title, priority, status, owner_role, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, id, title, priority, status, ownerRole, createdAt)
	require.NoError(t, err)
}

func TestTaskService_ListTasks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupSvcTaskTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC()
	insertSvcTestTask(t, pool, "task-1", "Review order anomaly", "high", "todo", "analyst", now.Add(-2*time.Hour))
	insertSvcTestTask(t, pool, "task-2", "Update seller info", "medium", "in_progress", "admin", now.Add(-1*time.Hour))
	insertSvcTestTask(t, pool, "task-3", "Approve refund", "low", "done", "manager", now)

	repo := taskRepo.NewRepository(common.NewPoolProvider(pool))
	svc := NewTaskService(repo)

	t.Run("list all tasks", func(t *testing.T) {
		resp, err := svc.ListTasks(ctx, model.TaskFilters{}, 10, 0)
		require.NoError(t, err)
		assert.Equal(t, 3, resp.Total)
		assert.Len(t, resp.Items, 3)
	})

	t.Run("filter by status", func(t *testing.T) {
		status := "todo"
		resp, err := svc.ListTasks(ctx, model.TaskFilters{Status: &status}, 10, 0)
		require.NoError(t, err)
		assert.Equal(t, 1, resp.Total)
		assert.Equal(t, "task-1", resp.Items[0].TaskID)
	})

	t.Run("filter by priority", func(t *testing.T) {
		priority := "medium"
		resp, err := svc.ListTasks(ctx, model.TaskFilters{Priority: &priority}, 10, 0)
		require.NoError(t, err)
		assert.Equal(t, 1, resp.Total)
		assert.Equal(t, "medium", resp.Items[0].Priority)
	})

	t.Run("filter by owner", func(t *testing.T) {
		owner := "admin"
		resp, err := svc.ListTasks(ctx, model.TaskFilters{Owner: &owner}, 10, 0)
		require.NoError(t, err)
		assert.Equal(t, 1, resp.Total)
		assert.Equal(t, "admin", resp.Items[0].OwnerRole)
	})

	t.Run("pagination offset", func(t *testing.T) {
		resp, err := svc.ListTasks(ctx, model.TaskFilters{}, 1, 1)
		require.NoError(t, err)
		assert.Equal(t, 3, resp.Total)
		assert.Len(t, resp.Items, 1)
	})

	t.Run("empty result", func(t *testing.T) {
		status := "nonexistent"
		resp, err := svc.ListTasks(ctx, model.TaskFilters{Status: &status}, 10, 0)
		require.NoError(t, err)
		assert.Equal(t, 0, resp.Total)
		assert.Empty(t, resp.Items)
	})
}

func TestTaskService_DefaultValues(t *testing.T) {
	pool := setupSvcTaskTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC()
	_, err := pool.Exec(ctx, `
		INSERT INTO ops.task (task_id, task_title, created_at)
		VALUES ($1, $2, $3)
	`, "task-default-1", "Default priority task", now)
	require.NoError(t, err)

	repo := taskRepo.NewRepository(common.NewPoolProvider(pool))
	svc := NewTaskService(repo)
	resp, err := svc.ListTasks(ctx, model.TaskFilters{}, 10, 0)
	require.NoError(t, err)
	require.Len(t, resp.Items, 1)

	assert.Equal(t, "medium", resp.Items[0].Priority)
	assert.Equal(t, "todo", resp.Items[0].Status)
}
