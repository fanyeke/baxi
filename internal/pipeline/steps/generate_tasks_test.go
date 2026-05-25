package steps

import (
	"context"
	"os"
	"testing"

	"baxi/internal/pipeline"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// generateTasksOpsDDL creates the ops schema needed for generate_tasks testing.
const generateTasksOpsDDL = `
CREATE SCHEMA IF NOT EXISTS ops;

CREATE TABLE IF NOT EXISTS ops.recommendation (
    recommendation_id   TEXT PRIMARY KEY,
    alert_id            TEXT,
    decision_source     TEXT NOT NULL DEFAULT 'heuristic',
    rule_id             TEXT,
    strategy_title      TEXT NOT NULL,
    strategy_detail     TEXT,
    target_object_type  TEXT,
    target_object_id    TEXT,
    expected_impact     TEXT,
    risk_level          TEXT,
    confidence          TEXT,
    requires_approval   BOOLEAN DEFAULT FALSE,
    approval_status     TEXT DEFAULT 'draft',
    execution_status    TEXT DEFAULT 'draft',
    owner_role          TEXT,
    success_metric      TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

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

// setupGenerateTasksTestDB creates ops tables for generate_tasks testing.
func setupGenerateTasksTestDB(t *testing.T) *pgxpool.Pool {
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

	if _, err := pool.Exec(ctx, generateTasksOpsDDL); err != nil {
		t.Fatalf("create ops tables: %v", err)
	}

	// Clean any leftover data
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE ops.recommendation CASCADE")
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE ops.task CASCADE")

	return pool
}

// insertTestRecommendations inserts sample recommendations for testing.
func insertTestRecommendations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	ctx := context.Background()
	recs := []struct {
		id, alertID, title, detail, objType, objID, riskLevel, ownerRole string
	}{
		{
			id: "rec-gmv_drop_2018-10-17", alertID: "gmv_drop_2018-10-17",
			title: "Review gmv anomaly from rule gmv_drop",
			detail: "GMV 7日均值较前14天均值下降超过15%",
			objType: "global", objID: "global",
			riskLevel: "high", ownerRole: "business_ops",
		},
		{
			id: "dimrec-dim-76085bfcd31d", alertID: "dim-76085bfcd31d",
			title: "排查区域 SP 延迟配送",
			detail: "区域延迟配送率超过20%且样本>=30单",
			objType: "region", objID: "SP",
			riskLevel: "high", ownerRole: "logistics_ops",
		},
		{
			id: "dimrec-dim-8bbbe8e62d34", alertID: "dim-8bbbe8e62d34",
			title: "排查卖家 1f50f920176fa81dab994f9023523100 评分异常",
			detail: "卖家评分低于3.5且样本>=20单",
			objType: "seller", objID: "1f50f920176fa81dab994f9023523100",
			riskLevel: "medium", ownerRole: "seller_ops",
		},
		{
			id: "dimrec-dim-455b469ba24c", alertID: "dim-455b469ba24c",
			title: "排查品类 health_beauty GMV 下降",
			detail: "品类GMV环比下降超过20%且样本>=30单",
			objType: "category", objID: "health_beauty",
			riskLevel: "medium", ownerRole: "category_ops",
		},
		{
			id: "dimrec-dim-3bb9eaf850d5", alertID: "dim-3bb9eaf850d5",
			title: "排查区域 SP 取消率异常",
			detail: "区域取消率超过5%且样本>=30单",
			objType: "region", objID: "SP",
			riskLevel: "medium", ownerRole: "logistics_ops",
		},
	}

	for _, r := range recs {
		_, err := pool.Exec(ctx, `
			INSERT INTO ops.recommendation (
				recommendation_id, alert_id, strategy_title, strategy_detail,
				target_object_type, target_object_id, risk_level, owner_role
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (recommendation_id) DO NOTHING
		`, r.id, r.alertID, r.title, r.detail, r.objType, r.objID, r.riskLevel, r.ownerRole)
		if err != nil {
			t.Fatalf("insert recommendation %s: %v", r.id, err)
		}
	}
}

func TestGenerateTasksStep_Name(t *testing.T) {
	step := NewGenerateTasksStep()
	if got := step.Name(); got != "generate_tasks" {
		t.Errorf("expected name 'generate_tasks', got %q", got)
	}
}

func TestGenerateTasksStep_HappyPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGenerateTasksTestDB(t)
	defer pool.Close()

	insertTestRecommendations(t, pool)

	ctx := context.Background()
	step := NewGenerateTasksStep()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	output, err := step.Run(ctx, tx, pipeline.StepInput{
		DataDir: t.TempDir(),
		Logger:  zap.NewNop(),
	})
	if err != nil {
		t.Fatalf("GenerateTasksStep.Run failed: %v", err)
	}

	// Verify input count matches recommendations
	if output.InputCount != 5 {
		t.Errorf("expected input_count 5, got %d", output.InputCount)
	}

	// Verify output count matches expected tasks (1:1 with recommendations)
	if output.OutputCount != 5 {
		t.Errorf("expected output_count 5, got %d", output.OutputCount)
	}

	// Verify all tasks are in ops.task with status='pending'
	var taskCount int64
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM ops.task`).Scan(&taskCount); err != nil {
		t.Fatalf("count ops.task: %v", err)
	}
	if taskCount != 5 {
		t.Errorf("expected 5 rows in ops.task, got %d", taskCount)
	}

	// Verify all tasks have status='pending'
	var pendingCount int64
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM ops.task WHERE status = 'pending'`).Scan(&pendingCount); err != nil {
		t.Fatalf("count pending tasks: %v", err)
	}
	if pendingCount != 5 {
		t.Errorf("expected 5 pending tasks, got %d", pendingCount)
	}

	// Verify specific tasks exist with correct properties
	type taskRow struct {
		taskID, recID, alertID     string
		title, description         string
		objType, objID             string
		source, ownerRole, priority string
		status                     string
	}

	rows, err := tx.Query(ctx, `
		SELECT task_id, recommendation_id, COALESCE(alert_id, ''),
		       task_title, COALESCE(task_description, ''),
		       COALESCE(target_object_type, ''), COALESCE(target_object_id, ''),
		       task_source, COALESCE(owner_role, ''), priority, status
		FROM ops.task
		ORDER BY task_id
	`)
	if err != nil {
		t.Fatalf("query tasks: %v", err)
	}
	defer rows.Close()

	var tasks []taskRow
	for rows.Next() {
		var r taskRow
		if err := rows.Scan(&r.taskID, &r.recID, &r.alertID,
			&r.title, &r.description,
			&r.objType, &r.objID,
			&r.source, &r.ownerRole, &r.priority, &r.status); err != nil {
			t.Fatalf("scan task: %v", err)
		}
		tasks = append(tasks, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows iteration: %v", err)
	}

	if len(tasks) != 5 {
		t.Fatalf("expected 5 tasks, got %d", len(tasks))
	}

	// Verify first task (global recommendation)
	t0 := tasks[0]
	if t0.taskID != "task-gmv_drop_2018-10-17" {
		t.Errorf("task[0].task_id: expected 'task-gmv_drop_2018-10-17', got %q", t0.taskID)
	}
	if t0.recID != "rec-gmv_drop_2018-10-17" {
		t.Errorf("task[0].recommendation_id mismatch")
	}
	if t0.alertID != "gmv_drop_2018-10-17" {
		t.Errorf("task[0].alert_id mismatch")
	}
	if t0.source != "heuristic_strategy" {
		t.Errorf("task[0].task_source: expected 'heuristic_strategy', got %q", t0.source)
	}
	if t0.priority != "high" {
		t.Errorf("task[0].priority: expected 'high', got %q", t0.priority)
	}
	if t0.status != "pending" {
		t.Errorf("task[0].status: expected 'pending', got %q", t0.status)
	}
	if t0.ownerRole != "business_ops" {
		t.Errorf("task[0].owner_role: expected 'business_ops', got %q", t0.ownerRole)
	}

	// Verify second task (dimensional recommendation)
	t1 := tasks[1]
	if t1.taskID != "dimtask-dim-3bb9eaf850d5" {
		t.Errorf("task[1].task_id: expected 'dimtask-dim-3bb9eaf850d5', got %q", t1.taskID)
	}
	if t1.source != "dimensional_rule" {
		t.Errorf("task[1].task_source: expected 'dimensional_rule', got %q", t1.source)
	}
	if t1.priority != "medium" {
		t.Errorf("task[1].priority: expected 'medium', got %q", t1.priority)
	}
	if t1.objType != "region" {
		t.Errorf("task[1].target_object_type: expected 'region', got %q", t1.objType)
	}
	if t1.objID != "SP" {
		t.Errorf("task[1].target_object_id: expected 'SP', got %q", t1.objID)
	}
}

func TestGenerateTasksStep_IdempotentReRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGenerateTasksTestDB(t)
	defer pool.Close()

	insertTestRecommendations(t, pool)

	ctx := context.Background()
	step := NewGenerateTasksStep()

	// First run
	tx1, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx1: %v", err)
	}

	output1, err := step.Run(ctx, tx1, pipeline.StepInput{
		DataDir: t.TempDir(),
		Logger:  zap.NewNop(),
	})
	if err != nil {
		tx1.Rollback(ctx)
		t.Fatalf("first run failed: %v", err)
	}
	if output1.OutputCount != 5 {
		t.Errorf("first run: expected 5 tasks, got %d", output1.OutputCount)
	}
	if err := tx1.Commit(ctx); err != nil {
		t.Fatalf("commit tx1: %v", err)
	}

	// Second run — ON CONFLICT DO NOTHING should prevent duplicates
	tx2, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx2: %v", err)
	}
	defer tx2.Rollback(ctx)

	output2, err := step.Run(ctx, tx2, pipeline.StepInput{
		DataDir: t.TempDir(),
		Logger:  zap.NewNop(),
	})
	if err != nil {
		t.Fatalf("second run failed: %v", err)
	}
	if output2.OutputCount != 0 {
		t.Errorf("second run (idempotent): expected 0 new rows, got %d", output2.OutputCount)
	}

	// Total should still be 5
	var total int64
	if err := tx2.QueryRow(ctx, `SELECT COUNT(*) FROM ops.task`).Scan(&total); err != nil {
		t.Fatalf("count ops.task: %v", err)
	}
	if total != 5 {
		t.Errorf("idempotent re-run: expected 5 total rows, got %d", total)
	}

	// All should still have status='pending'
	var pendingCount int64
	if err := tx2.QueryRow(ctx, `SELECT COUNT(*) FROM ops.task WHERE status = 'pending'`).Scan(&pendingCount); err != nil {
		t.Fatalf("count pending: %v", err)
	}
	if pendingCount != 5 {
		t.Errorf("idempotent re-run: expected 5 pending tasks, got %d", pendingCount)
	}
}

func TestGenerateTasksStep_EmptyRecommendations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupGenerateTasksTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	step := NewGenerateTasksStep()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	output, err := step.Run(ctx, tx, pipeline.StepInput{
		DataDir: t.TempDir(),
		Logger:  zap.NewNop(),
	})
	if err != nil {
		t.Fatalf("Run with empty recommendations failed: %v", err)
	}

	if output.InputCount != 0 {
		t.Errorf("expected input_count 0, got %d", output.InputCount)
	}
	if output.OutputCount != 0 {
		t.Errorf("expected output_count 0, got %d", output.OutputCount)
	}
}
