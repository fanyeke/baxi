package recommendation

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// opsTableDDL creates the ops schema and tables needed for testing.
const opsTableDDL = `
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

// setupTestDB creates the ops tables for testing.
func setupTestDB(t *testing.T) *pgxpool.Pool {
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

	if _, err := pool.Exec(ctx, opsTableDDL); err != nil {
		t.Fatalf("create ops tables: %v", err)
	}

	// Clean any leftover data
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE ops.recommendation CASCADE")
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE ops.task CASCADE")

	return pool
}

func TestTaskGenerator_GenerateTasks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()

	// Insert sample recommendations: 1 global + 4 dimensional = 5 total
	recommendations := []struct {
		recID         string
		alertID       string
		title         string
		detail        string
		objType       string
		objID         string
		riskLevel     string
		ownerRole     string
	}{
		{
			recID:     "rec-gmv_drop_2018-10-17",
			alertID:   "gmv_drop_2018-10-17",
			title:     "Review gmv anomaly from rule gmv_drop",
			detail:    "GMV 7日均值较前14天均值下降超过15% | 7d_avg=148.98, 14d_avg=1991.98",
			objType:   "global",
			objID:     "global",
			riskLevel: "high",
			ownerRole: "business_ops",
		},
		{
			recID:     "dimrec-dim-76085bfcd31d",
			alertID:   "dim-76085bfcd31d",
			title:     "排查区域 SP 延迟配送",
			detail:    "区域延迟配送率超过20%且样本>=30单",
			objType:   "region",
			objID:     "SP",
			riskLevel: "high",
			ownerRole: "logistics_ops",
		},
		{
			recID:     "dimrec-dim-8bbbe8e62d34",
			alertID:   "dim-8bbbe8e62d34",
			title:     "排查卖家 1f50f920176fa81dab994f9023523100 评分异常",
			detail:    "卖家评分低于3.5且样本>=20单",
			objType:   "seller",
			objID:     "1f50f920176fa81dab994f9023523100",
			riskLevel: "medium",
			ownerRole: "seller_ops",
		},
		{
			recID:     "dimrec-dim-455b469ba24c",
			alertID:   "dim-455b469ba24c",
			title:     "排查品类 health_beauty GMV 下降",
			detail:    "品类GMV环比下降超过20%且样本>=30单",
			objType:   "category",
			objID:     "health_beauty",
			riskLevel: "medium",
			ownerRole: "category_ops",
		},
		{
			recID:     "dimrec-dim-3bb9eaf850d5",
			alertID:   "dim-3bb9eaf850d5",
			title:     "排查区域 SP 取消率异常",
			detail:    "区域取消率超过5%且样本>=30单",
			objType:   "region",
			objID:     "SP",
			riskLevel: "medium",
			ownerRole: "logistics_ops",
		},
	}

	for _, r := range recommendations {
		_, err := pool.Exec(ctx, `
			INSERT INTO ops.recommendation (
				recommendation_id, alert_id, strategy_title, strategy_detail,
				target_object_type, target_object_id, risk_level, owner_role
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (recommendation_id) DO NOTHING
		`, r.recID, r.alertID, r.title, r.detail, r.objType, r.objID, r.riskLevel, r.ownerRole)
		if err != nil {
			t.Fatalf("insert recommendation %s: %v", r.recID, err)
		}
	}

	// Run task generation
	gen := NewTaskGenerator()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	tasks, err := gen.GenerateTasks(ctx, tx)
	if err != nil {
		t.Fatalf("GenerateTasks failed: %v", err)
	}

	// Verify count
	if len(tasks) != 5 {
		t.Fatalf("expected 5 tasks, got %d", len(tasks))
	}

	// Verify task ID transformations
	expectedTasks := map[string]struct {
		taskID      string
		recID       string
		title       string
		source      string
		priority    string
		status      string
	}{
		"task-gmv_drop_2018-10-17": {
			taskID: "task-gmv_drop_2018-10-17",
			recID:  "rec-gmv_drop_2018-10-17",
			title:  "Review gmv anomaly from rule gmv_drop",
			source: "heuristic_strategy",
			priority: "high",
			status: "pending",
		},
		"dimtask-dim-76085bfcd31d": {
			taskID: "dimtask-dim-76085bfcd31d",
			recID:  "dimrec-dim-76085bfcd31d",
			title:  "排查区域 SP 延迟配送",
			source: "dimensional_rule",
			priority: "high",
			status: "pending",
		},
		"dimtask-dim-8bbbe8e62d34": {
			taskID: "dimtask-dim-8bbbe8e62d34",
			recID:  "dimrec-dim-8bbbe8e62d34",
			title:  "排查卖家 1f50f920176fa81dab994f9023523100 评分异常",
			source: "dimensional_rule",
			priority: "medium",
			status: "pending",
		},
		"dimtask-dim-455b469ba24c": {
			taskID: "dimtask-dim-455b469ba24c",
			recID:  "dimrec-dim-455b469ba24c",
			title:  "排查品类 health_beauty GMV 下降",
			source: "dimensional_rule",
			priority: "medium",
			status: "pending",
		},
		"dimtask-dim-3bb9eaf850d5": {
			taskID: "dimtask-dim-3bb9eaf850d5",
			recID:  "dimrec-dim-3bb9eaf850d5",
			title:  "排查区域 SP 取消率异常",
			source: "dimensional_rule",
			priority: "medium",
			status: "pending",
		},
	}

	for _, task := range tasks {
		expected, ok := expectedTasks[task.TaskID]
		if !ok {
			t.Errorf("unexpected task_id: %s", task.TaskID)
			continue
		}
		if task.RecommendationID != expected.recID {
			t.Errorf("task %s: recommendation_id: expected %q, got %q", task.TaskID, expected.recID, task.RecommendationID)
		}
		if task.TaskTitle != expected.title {
			t.Errorf("task %s: title: expected %q, got %q", task.TaskID, expected.title, task.TaskTitle)
		}
		if task.TaskSource != expected.source {
			t.Errorf("task %s: source: expected %q, got %q", task.TaskID, expected.source, task.TaskSource)
		}
		if task.Priority != expected.priority {
			t.Errorf("task %s: priority: expected %q, got %q", task.TaskID, expected.priority, task.Priority)
		}
		if task.Status != expected.status {
			t.Errorf("task %s: status: expected %q, got %q", task.TaskID, expected.status, task.Status)
		}
	}

	// Verify target object mapping
	for _, task := range tasks {
		if task.TaskID == "dimtask-dim-76085bfcd31d" {
			if task.TargetObjectType != "region" {
				t.Errorf("dim-76085: obj_type: expected 'region', got %q", task.TargetObjectType)
			}
			if task.TargetObjectID != "SP" {
				t.Errorf("dim-76085: obj_id: expected 'SP', got %q", task.TargetObjectID)
			}
		}
	}

	// Verify owner_role mapping
	for _, task := range tasks {
		switch task.TaskID {
		case "task-gmv_drop_2018-10-17":
			if task.OwnerRole != "business_ops" {
				t.Errorf("gmv_drop: owner_role: expected 'business_ops', got %q", task.OwnerRole)
			}
		case "dimtask-dim-76085bfcd31d":
			if task.OwnerRole != "logistics_ops" {
				t.Errorf("dim-76085: owner_role: expected 'logistics_ops', got %q", task.OwnerRole)
			}
		}
	}

	// Verify alert_id mapping
	for _, task := range tasks {
		if task.TaskID == "task-gmv_drop_2018-10-17" && task.AlertID != "gmv_drop_2018-10-17" {
			t.Errorf("gmv_drop: alert_id: expected 'gmv_drop_2018-10-17', got %q", task.AlertID)
		}
	}
}

func TestDeriveTaskID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"rec-gmv_drop_2018-10-17", "task-gmv_drop_2018-10-17"},
		{"dimrec-dim-76085bfcd31d", "dimtask-dim-76085bfcd31d"},
		{"custom-prefix", "task-custom-prefix"},
		{"rec", "task-rec"},
		{"dimrec", "task-dimrec"},
	}

	for _, tt := range tests {
		got := deriveTaskID(tt.input)
		if got != tt.expected {
			t.Errorf("deriveTaskID(%q): expected %q, got %q", tt.input, tt.expected, got)
		}
	}
}

func TestDeriveTaskSource(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"rec-gmv_drop", "heuristic_strategy"},
		{"dimrec-dim-76085", "dimensional_rule"},
		{"unknown", "heuristic_strategy"},
	}

	for _, tt := range tests {
		got := deriveTaskSource(tt.input)
		if got != tt.expected {
			t.Errorf("deriveTaskSource(%q): expected %q, got %q", tt.input, tt.expected, got)
		}
	}
}

func TestDerivePriority(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"high", "high"},
		{"medium", "medium"},
		{"low", "low"},
		{"unknown", "medium"},
		{"", "medium"},
	}

	for _, tt := range tests {
		got := derivePriority(tt.input)
		if got != tt.expected {
			t.Errorf("derivePriority(%q): expected %q, got %q", tt.input, tt.expected, got)
		}
	}
}
