package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// taskTableDDL creates the ops.task table for testing.
// Mirrors migrations/005_ops_tables.sql.
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

// setupTaskTestDB creates the ops.task table and returns a pool.
func setupTaskTestDB(t *testing.T) *pgxpool.Pool {
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

	// Clean any leftover data
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE ops.task CASCADE")

	return pool
}

// insertTestTask inserts a single task row for testing.
func insertTestTask(t *testing.T, pool *pgxpool.Pool, taskID, status, priority, ownerRole string, createdAt time.Time) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		INSERT INTO ops.task (task_id, task_title, task_description, status, priority, owner_role, created_at, due_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`,
		taskID, "Test Task", "Test description",
		status, priority, ownerRole, createdAt, createdAt.Add(24*time.Hour),
	)
	if err != nil {
		t.Fatalf("insert task %s: %v", taskID, err)
	}
}

func TestTaskRepository_ListTasks_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupTaskTestDB(t)
	repo := NewTaskRepository()
	ctx := context.Background()

	rows, total, err := repo.ListTasks(ctx, pool, TaskFilters{}, 100, 0)
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("expected 0 rows, got %d", len(rows))
	}
	if total != 0 {
		t.Errorf("expected total=0, got %d", total)
	}
}

func TestTaskRepository_ListTasks_WithData(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupTaskTestDB(t)
	repo := NewTaskRepository()
	ctx := context.Background()

	now := time.Now().UTC()
	insertTestTask(t, pool, "task-1", "todo", "high", "logistics_ops", now)
	insertTestTask(t, pool, "task-2", "in_progress", "high", "logistics_ops", now.Add(-1*time.Hour))
	insertTestTask(t, pool, "task-3", "todo", "medium", "seller_ops", now.Add(-2*time.Hour))

	// Test 1: No filters — should return all 3 tasks
	rows, total, err := repo.ListTasks(ctx, pool, TaskFilters{}, 100, 0)
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}
	if total != 3 {
		t.Errorf("expected total=3, got %d", total)
	}
	if len(rows) != 3 {
		t.Errorf("expected 3 rows, got %d", len(rows))
	}

	// Verify order: created_at DESC (task-1 is newest)
	if rows[0].TaskID != "task-1" {
		t.Errorf("expected first row task-1, got %s", rows[0].TaskID)
	}

	// Test 2: Filter by status
	status := "todo"
	rows, total, err = repo.ListTasks(ctx, pool, TaskFilters{Status: &status}, 100, 0)
	if err != nil {
		t.Fatalf("ListTasks with status filter failed: %v", err)
	}
	if total != 2 {
		t.Errorf("expected total=2 for status=todo, got %d", total)
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(rows))
	}

	// Test 3: Filter by priority
	priority := "medium"
	rows, total, err = repo.ListTasks(ctx, pool, TaskFilters{Priority: &priority}, 100, 0)
	if err != nil {
		t.Fatalf("ListTasks with priority filter failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total=1 for priority=medium, got %d", total)
	}
	if len(rows) != 1 {
		t.Errorf("expected 1 row, got %d", len(rows))
	}
	if rows[0].TaskID != "task-3" {
		t.Errorf("expected task-3, got %s", rows[0].TaskID)
	}

	// Test 4: Filter by owner
	owner := "seller_ops"
	rows, total, err = repo.ListTasks(ctx, pool, TaskFilters{Owner: &owner}, 100, 0)
	if err != nil {
		t.Fatalf("ListTasks with owner filter failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total=1 for owner=seller_ops, got %d", total)
	}
	if rows[0].OwnerRole == nil || *rows[0].OwnerRole != "seller_ops" {
		t.Errorf("expected owner_role=seller_ops, got %v", rows[0].OwnerRole)
	}

	// Test 5: Combined filters
	status = "todo"
	owner = "logistics_ops"
	rows, total, err = repo.ListTasks(ctx, pool, TaskFilters{Status: &status, Owner: &owner}, 100, 0)
	if err != nil {
		t.Fatalf("ListTasks with combined filters failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total=1 for combined filters, got %d", total)
	}
	if rows[0].TaskID != "task-1" {
		t.Errorf("expected task-1, got %s", rows[0].TaskID)
	}
}

func TestTaskRepository_ListTasks_Pagination(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupTaskTestDB(t)
	repo := NewTaskRepository()
	ctx := context.Background()

	now := time.Now().UTC()
	taskIDs := []string{"task-a", "task-b", "task-c", "task-d", "task-e"}
	for i, id := range taskIDs {
		insertTestTask(t, pool, id, "todo", "high", "logistics_ops", now.Add(-time.Duration(i)*time.Hour))
	}

	// Test: limit=2, offset=0
	rows, total, err := repo.ListTasks(ctx, pool, TaskFilters{}, 2, 0)
	if err != nil {
		t.Fatalf("ListTasks with pagination failed: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total=5, got %d", total)
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(rows))
	}

	// Test: limit=2, offset=2 (should get next 2)
	rows, total, err = repo.ListTasks(ctx, pool, TaskFilters{}, 2, 2)
	if err != nil {
		t.Fatalf("ListTasks with offset pagination failed: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total=5, got %d", total)
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(rows))
	}
}

func TestTaskRepository_ListTasks_NullableFields(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupTaskTestDB(t)
	repo := NewTaskRepository()
	ctx := context.Background()

	// Insert a task with nullable fields set to NULL
	_, err := pool.Exec(ctx, `
		INSERT INTO ops.task (task_id, task_title, status, priority, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, "task-nullable", "Nullable Fields Task", "todo", "low", time.Now().UTC())
	if err != nil {
		t.Fatalf("insert nullable task: %v", err)
	}

	rows, total, err := repo.ListTasks(ctx, pool, TaskFilters{}, 100, 0)
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total=1, got %d", total)
	}

	// Verify nullable fields are nil
	row := rows[0]
	if row.RecommendationID != nil {
		t.Errorf("expected nil RecommendationID, got %v", *row.RecommendationID)
	}
	if row.AlertID != nil {
		t.Errorf("expected nil AlertID, got %v", *row.AlertID)
	}
	if row.TaskDescription != nil {
		t.Errorf("expected nil TaskDescription, got %v", *row.TaskDescription)
	}
	if row.OwnerRole != nil {
		t.Errorf("expected nil OwnerRole, got %v", *row.OwnerRole)
	}
	if row.OwnerUserID != nil {
		t.Errorf("expected nil OwnerUserID, got %v", *row.OwnerUserID)
	}
	if row.Feedback != nil {
		t.Errorf("expected nil Feedback, got %v", *row.Feedback)
	}
	if row.CompletedAt != nil {
		t.Errorf("expected nil CompletedAt, got %v", *row.CompletedAt)
	}
	if row.DueAt != nil {
		t.Errorf("expected nil DueAt, got %v", *row.DueAt)
	}
}
