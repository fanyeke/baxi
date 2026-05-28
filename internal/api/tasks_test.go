package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"baxi/internal/api/dto"
)

// setupTasksTestDB creates the ops.task table and returns a pool.
func setupTasksTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	t.Cleanup(pool.Close)

	ctx := context.Background()
	if _, err := pool.Exec(ctx, taskTableDDL); err != nil {
		t.Fatalf("create ops.task: %v", err)
	}
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE ops.task CASCADE")

	return pool
}

// taskTableDDL creates the ops.task table for testing.
const taskTableDDL = `
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

func TestHandleListTasks_NoAuth_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupTasksTestDB(t)
	logger := zap.NewNop()
	srv := New(logger, pool)

	// Insert test data
	_, err := pool.Exec(context.Background(), `
		INSERT INTO ops.task (task_id, task_title, task_description, status, priority, owner_role, owner_user_id, created_at)
		VALUES
		('task-1', 'Task One', 'Description one', 'todo', 'high', 'logistics_ops', NULL, NOW()),
		('task-2', 'Task Two', 'Description two', 'in_progress', 'high', 'logistics_ops', NULL, NOW() - INTERVAL '1 hour'),
		('task-3', 'Task Three', 'Description three', 'todo', 'medium', 'seller_ops', NULL, NOW() - INTERVAL '2 hours')
	`)
	if err != nil {
		t.Fatalf("insert test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	rec := httptest.NewRecorder()
	srv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp dto.TaskListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.Total != 3 {
		t.Errorf("expected total=3, got %d", resp.Total)
	}
	if len(resp.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(resp.Items))
	}
}

func TestHandleListTasks_Filters(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupTasksTestDB(t)
	logger := zap.NewNop()
	srv := New(logger, pool)

	// Insert test data
	_, err := pool.Exec(context.Background(), `
		INSERT INTO ops.task (task_id, task_title, task_description, status, priority, owner_role, created_at)
		VALUES
		('task-1', 'Task One', 'Desc', 'todo', 'high', 'logistics_ops', NOW()),
		('task-2', 'Task Two', 'Desc', 'in_progress', 'high', 'logistics_ops', NOW() - INTERVAL '1 hour'),
		('task-3', 'Task Three', 'Desc', 'todo', 'medium', 'seller_ops', NOW() - INTERVAL '2 hours')
	`)
	if err != nil {
		t.Fatalf("insert test data: %v", err)
	}

	tests := []struct {
		name       string
		query      string
		wantTotal  int
		wantStatus int
	}{
		{"filter by status", "?status=todo", 2, http.StatusOK},
		{"filter by priority", "?priority=medium", 1, http.StatusOK},
		{"filter by owner", "?owner=seller_ops", 1, http.StatusOK},
		{"pagination", "?limit=1&offset=0", 3, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks"+tt.query, nil)
			rec := httptest.NewRecorder()
			srv.router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}

			var resp dto.TaskListResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			if resp.Total != tt.wantTotal {
				t.Errorf("expected total=%d, got %d", tt.wantTotal, resp.Total)
			}
		})
	}
}

func TestHandleListTasks_EmptyDB(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupTasksTestDB(t)
	logger := zap.NewNop()
	srv := New(logger, pool)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	rec := httptest.NewRecorder()
	srv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var resp dto.TaskListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Total != 0 {
		t.Errorf("expected total=0, got %d", resp.Total)
	}
	if len(resp.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(resp.Items))
	}
}

func TestHandleListTasks_InvalidPagination(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupTasksTestDB(t)
	logger := zap.NewNop()
	srv := New(logger, pool)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks?limit=abc", nil)
	rec := httptest.NewRecorder()
	srv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}
