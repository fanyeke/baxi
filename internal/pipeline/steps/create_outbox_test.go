package steps

import (
	"context"
	"os"
	"testing"

	"baxi/internal/pipeline"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// createOutboxOpsDDL creates the ops schema needed for create_outbox testing.
const createOutboxOpsDDL = `
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

CREATE TABLE IF NOT EXISTS ops.outbox_event (
    event_id            TEXT PRIMARY KEY,
    event_type          TEXT NOT NULL,
    source_type         TEXT NOT NULL,
    source_id           TEXT NOT NULL,
    payload_json        JSONB NOT NULL,
    target_channel      TEXT NOT NULL,
    status              TEXT DEFAULT 'pending',
    dispatch_attempts   BIGINT DEFAULT 0,
    last_dispatch_at    TIMESTAMPTZ,
    external_ref        TEXT,
    adapter_name        TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at        TIMESTAMPTZ,
    error_message       TEXT
);
`

// setupCreateOutboxTestDB creates the ops tables needed for create_outbox testing.
func setupCreateOutboxTestDB(t *testing.T) *pgxpool.Pool {
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
	if _, err := pool.Exec(ctx, createOutboxOpsDDL); err != nil {
		t.Fatalf("create ops tables: %v", err)
	}

	// Clean any leftover data
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE ops.recommendation CASCADE")
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE ops.task CASCADE")
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE ops.outbox_event CASCADE")

	return pool
}

// insertTestTasks inserts sample recommendations and tasks for outbox testing.
func insertTestTasks(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	ctx := context.Background()

	// Insert a recommendation first (FK constraint, though SET NULL on delete)
	_, err := pool.Exec(ctx, `
		INSERT INTO ops.recommendation (
			recommendation_id, alert_id, strategy_title, strategy_detail,
			target_object_type, target_object_id, risk_level, owner_role
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (recommendation_id) DO NOTHING
	`, "rec-gmv_drop_2018-10-17", "gmv_drop_2018-10-17",
		"Review gmv anomaly from rule gmv_drop",
		"GMV 7日均值较前14天均值下降超过15%",
		"global", "global", "high", "business_ops")
	if err != nil {
		t.Fatalf("insert recommendation: %v", err)
	}

	// Insert a global task (heuristic_strategy)
	_, err = pool.Exec(ctx, `
		INSERT INTO ops.task (
			task_id, recommendation_id, alert_id,
			task_title, task_description,
			target_object_type, target_object_id,
			task_source, owner_role, priority, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'pending')
		ON CONFLICT (task_id) DO NOTHING
	`, "task-gmv_drop_2018-10-17", "rec-gmv_drop_2018-10-17", "gmv_drop_2018-10-17",
		"Review gmv anomaly from rule gmv_drop",
		"GMV 7日均值较前14天均值下降超过15%",
		"global", "global",
		"heuristic_strategy", "business_ops", "high")
	if err != nil {
		t.Fatalf("insert task: %v", err)
	}

	// Insert a dimensional recommendation
	_, err = pool.Exec(ctx, `
		INSERT INTO ops.recommendation (
			recommendation_id, alert_id, strategy_title, strategy_detail,
			target_object_type, target_object_id, risk_level, owner_role
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (recommendation_id) DO NOTHING
	`, "dimrec-dim-76085bfcd31d", "dim-76085bfcd31d",
		"排查区域 SP 延迟配送",
		"区域延迟配送率超过20%且样本>=30单",
		"region", "SP", "high", "logistics_ops")
	if err != nil {
		t.Fatalf("insert dimensional recommendation: %v", err)
	}

	// Insert a dimensional task (dimensional_rule)
	_, err = pool.Exec(ctx, `
		INSERT INTO ops.task (
			task_id, recommendation_id, alert_id,
			task_title, task_description,
			target_object_type, target_object_id,
			task_source, owner_role, priority, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'pending')
		ON CONFLICT (task_id) DO NOTHING
	`, "dimtask-dim-76085bfcd31d", "dimrec-dim-76085bfcd31d", "dim-76085bfcd31d",
		"排查区域 SP 延迟配送",
		"区域延迟配送率超过20%且样本>=30单",
		"region", "SP",
		"dimensional_rule", "logistics_ops", "high")
	if err != nil {
		t.Fatalf("insert dimensional task: %v", err)
	}
}

func TestCreateOutboxStep_Name(t *testing.T) {
	step := NewCreateOutboxStep()
	if got := step.Name(); got != "create_outbox_events" {
		t.Errorf("expected name 'create_outbox_events', got %q", got)
	}
}

func TestCreateOutboxStep_HappyPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupCreateOutboxTestDB(t)
	defer pool.Close()

	insertTestTasks(t, pool)

	ctx := context.Background()
	step := NewCreateOutboxStep()

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
		t.Fatalf("CreateOutboxStep.Run failed: %v", err)
	}

	// Verify input count matches tasks
	if output.InputCount != 2 {
		t.Errorf("expected input_count 2, got %d", output.InputCount)
	}

	// Verify output count
	if output.OutputCount != 2 {
		t.Errorf("expected output_count 2, got %d", output.OutputCount)
	}

	// Verify all outbox events are in ops.outbox_event with status='pending'
	var eventCount int64
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM ops.outbox_event`).Scan(&eventCount); err != nil {
		t.Fatalf("count ops.outbox_event: %v", err)
	}
	if eventCount != 2 {
		t.Errorf("expected 2 rows in ops.outbox_event, got %d", eventCount)
	}

	// Verify all have status='pending'
	var pendingCount int64
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM ops.outbox_event WHERE status = 'pending'`).Scan(&pendingCount); err != nil {
		t.Fatalf("count pending events: %v", err)
	}
	if pendingCount != 2 {
		t.Errorf("expected 2 pending events, got %d", pendingCount)
	}

	// Verify specific events
	type eventRow struct {
		eventID, eventType, sourceType, sourceID, targetChannel, status string
	}

	rows, err := tx.Query(ctx, `
		SELECT event_id, event_type, source_type, source_id, target_channel, status
		FROM ops.outbox_event
		ORDER BY event_id
	`)
	if err != nil {
		t.Fatalf("query outbox events: %v", err)
	}
	defer rows.Close()

	var events []eventRow
	for rows.Next() {
		var e eventRow
		if err := rows.Scan(&e.eventID, &e.eventType, &e.sourceType, &e.sourceID, &e.targetChannel, &e.status); err != nil {
			t.Fatalf("scan event: %v", err)
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows iteration: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	// First event: global task → local_cli
	e0 := events[0]
	expectedID0 := "outbox-task-gmv_drop_2018-10-17"
	if e0.eventID != expectedID0 {
		t.Errorf("event[0].event_id: expected %q, got %q", expectedID0, e0.eventID)
	}
	if e0.eventType != "task_assigned" {
		t.Errorf("event[0].event_type: expected 'task_assigned', got %q", e0.eventType)
	}
	if e0.sourceType != "task" {
		t.Errorf("event[0].source_type: expected 'task', got %q", e0.sourceType)
	}
	if e0.sourceID != "task-gmv_drop_2018-10-17" {
		t.Errorf("event[0].source_id: expected 'task-gmv_drop_2018-10-17', got %q", e0.sourceID)
	}
	if e0.targetChannel != "local_cli" {
		t.Errorf("event[0].target_channel: expected 'local_cli' for heuristic_strategy, got %q", e0.targetChannel)
	}
	if e0.status != "pending" {
		t.Errorf("event[0].status: expected 'pending', got %q", e0.status)
	}

	// Second event: dimensional task → feishu_cli
	e1 := events[1]
	expectedID1 := "outbox-dimtask-dim-76085bfcd31d"
	if e1.eventID != expectedID1 {
		t.Errorf("event[1].event_id: expected %q, got %q", expectedID1, e1.eventID)
	}
	if e1.eventType != "task_assigned" {
		t.Errorf("event[1].event_type: expected 'task_assigned', got %q", e1.eventType)
	}
	if e1.sourceType != "task" {
		t.Errorf("event[1].source_type: expected 'task', got %q", e1.sourceType)
	}
	if e1.sourceID != "dimtask-dim-76085bfcd31d" {
		t.Errorf("event[1].source_id: expected 'dimtask-dim-76085bfcd31d', got %q", e1.sourceID)
	}
	if e1.targetChannel != "feishu_cli" {
		t.Errorf("event[1].target_channel: expected 'feishu_cli' for dimensional_rule, got %q", e1.targetChannel)
	}
	if e1.status != "pending" {
		t.Errorf("event[1].status: expected 'pending', got %q", e1.status)
	}
}

func TestCreateOutboxStep_IdempotentReRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupCreateOutboxTestDB(t)
	defer pool.Close()

	insertTestTasks(t, pool)

	ctx := context.Background()
	step := NewCreateOutboxStep()

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
	if output1.OutputCount != 2 {
		t.Errorf("first run: expected 2 events, got %d", output1.OutputCount)
	}
	if err := tx1.Commit(ctx); err != nil {
		t.Fatalf("commit tx1: %v", err)
	}

	// Second run — ON CONFLICT DO NOTHING should return 0 new rows
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

	// Total should still be 2
	var total int64
	if err := tx2.QueryRow(ctx, `SELECT COUNT(*) FROM ops.outbox_event`).Scan(&total); err != nil {
		t.Fatalf("count ops.outbox_event: %v", err)
	}
	if total != 2 {
		t.Errorf("idempotent re-run: expected 2 total rows, got %d", total)
	}

	// All should still have status='pending'
	var pendingCount int64
	if err := tx2.QueryRow(ctx, `SELECT COUNT(*) FROM ops.outbox_event WHERE status = 'pending'`).Scan(&pendingCount); err != nil {
		t.Fatalf("count pending: %v", err)
	}
	if pendingCount != 2 {
		t.Errorf("idempotent re-run: expected 2 pending events, got %d", pendingCount)
	}
}

func TestCreateOutboxStep_EmptyTasks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupCreateOutboxTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	step := NewCreateOutboxStep()

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
		t.Fatalf("Run with empty tasks failed: %v", err)
	}

	if output.InputCount != 0 {
		t.Errorf("expected input_count 0, got %d", output.InputCount)
	}
	if output.OutputCount != 0 {
		t.Errorf("expected output_count 0, got %d", output.OutputCount)
	}
}

func TestDeriveTargetChannel(t *testing.T) {
	tests := []struct {
		taskSource string
		expected   string
	}{
		{"heuristic_strategy", "local_cli"},
		{"dimensional_rule", "feishu_cli"},
		{"unknown", "local_cli"},
		{"", "local_cli"},
	}

	for _, tt := range tests {
		got := deriveTargetChannel(tt.taskSource)
		if got != tt.expected {
			t.Errorf("deriveTargetChannel(%q): expected %q, got %q", tt.taskSource, tt.expected, got)
		}
	}
}

func TestIsDimensionalTask(t *testing.T) {
	tests := []struct {
		taskID   string
		expected bool
	}{
		{"dimtask-dim-76085bfcd31d", true},
		{"dimtask-foo", true},
		{"task-gmv_drop_2018-10-17", false},
		{"task-foo", false},
		{"", false},
	}

	for _, tt := range tests {
		got := IsDimensionalTask(tt.taskID)
		if got != tt.expected {
			t.Errorf("IsDimensionalTask(%q): expected %v, got %v", tt.taskID, tt.expected, got)
		}
	}
}
