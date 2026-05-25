package outbox

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// outboxTableDDL creates the ops.outbox_event table for testing.
// Mirrors migrations/005_ops_tables.sql.
const outboxTableDDL = `
CREATE SCHEMA IF NOT EXISTS ops;

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

// setupOutboxTestDB creates the ops.outbox_event table and returns a pool.
func setupOutboxTestDB(t *testing.T) *pgxpool.Pool {
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
	if _, err := pool.Exec(ctx, outboxTableDDL); err != nil {
		t.Fatalf("create ops.outbox_event: %v", err)
	}

	// Clean any leftover data
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE ops.outbox_event CASCADE")

	return pool
}

func TestOutboxRepository_CreateEvent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOutboxTestDB(t)
	defer pool.Close()

	repo := NewOutboxRepository()
	ctx := context.Background()

	payload, _ := json.Marshal(map[string]interface{}{
		"task_id": "task-gmv_drop_2018-10-17",
		"priority": "high",
	})

	event := &OutboxEvent{
		EventID:       "outbox-task-gmv_drop_2018-10-17",
		EventType:     "task_assigned",
		SourceType:    "task",
		SourceID:      "task-gmv_drop_2018-10-17",
		Status:        "pending",
		Payload:       payload,
		TargetChannel: "local_cli",
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	inserted, err := repo.CreateEvent(ctx, tx, event)
	if err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}
	if !inserted {
		t.Errorf("expected inserted=true, got false")
	}

	// Verify the row exists
	var status string
	var eventType, sourceType, sourceID, targetChannel string
	err = tx.QueryRow(ctx, `
		SELECT status, event_type, source_type, source_id, target_channel
		FROM ops.outbox_event WHERE event_id = $1
	`, event.EventID).Scan(&status, &eventType, &sourceType, &sourceID, &targetChannel)
	if err != nil {
		t.Fatalf("query outbox event: %v", err)
	}
	if status != "pending" {
		t.Errorf("expected status 'pending', got %q", status)
	}
	if eventType != "task_assigned" {
		t.Errorf("expected event_type 'task_assigned', got %q", eventType)
	}
	if sourceType != "task" {
		t.Errorf("expected source_type 'task', got %q", sourceType)
	}
	if sourceID != "task-gmv_drop_2018-10-17" {
		t.Errorf("expected source_id 'task-gmv_drop_2018-10-17', got %q", sourceID)
	}
	if targetChannel != "local_cli" {
		t.Errorf("expected target_channel 'local_cli', got %q", targetChannel)
	}
}

func TestOutboxRepository_CreateEvent_Idempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOutboxTestDB(t)
	defer pool.Close()

	repo := NewOutboxRepository()
	ctx := context.Background()

	payload, _ := json.Marshal(map[string]string{"key": "value"})

	event := &OutboxEvent{
		EventID:       "test-idempotent-event",
		EventType:     "task_assigned",
		SourceType:    "task",
		SourceID:      "test-task",
		Status:        "pending",
		Payload:       payload,
		TargetChannel: "local_cli",
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	// First insert
	inserted, err := repo.CreateEvent(ctx, tx, event)
	if err != nil {
		t.Fatalf("first CreateEvent failed: %v", err)
	}
	if !inserted {
		t.Errorf("expected first insert to return inserted=true")
	}

	// Second insert with same event_id – ON CONFLICT DO NOTHING
	inserted, err = repo.CreateEvent(ctx, tx, event)
	if err != nil {
		t.Fatalf("second CreateEvent failed: %v", err)
	}
	if inserted {
		t.Errorf("expected second insert (idempotent) to return inserted=false")
	}

	// Verify only one row
	var count int64
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM ops.outbox_event WHERE event_id = $1`, event.EventID).Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 row, got %d", count)
	}
}

func TestOutboxRepository_CreateEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOutboxTestDB(t)
	defer pool.Close()

	repo := NewOutboxRepository()
	ctx := context.Background()

	events := make([]OutboxEvent, 3)
	for i := 0; i < 3; i++ {
		payload, _ := json.Marshal(map[string]interface{}{
			"task_id":   "task-foo",
			"index":     i,
		})
		events[i] = OutboxEvent{
			EventID:       "outbox-task-foo-" + itoa(i),
			EventType:     "task_assigned",
			SourceType:    "task",
			SourceID:      "task-foo",
			Status:        "pending",
			Payload:       payload,
			TargetChannel: "local_cli",
		}
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	inserted, err := repo.CreateEvents(ctx, tx, events)
	if err != nil {
		t.Fatalf("CreateEvents failed: %v", err)
	}
	if inserted != 3 {
		t.Errorf("expected 3 inserted, got %d", inserted)
	}

	// Verify row count
	var count int64
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM ops.outbox_event`).Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 rows, got %d", count)
	}
}

func TestOutboxRepository_CreateEvents_Idempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOutboxTestDB(t)
	defer pool.Close()

	repo := NewOutboxRepository()
	ctx := context.Background()

	payload, _ := json.Marshal(map[string]string{"key": "value"})

	event := OutboxEvent{
		EventID:       "dup-event",
		EventType:     "task_assigned",
		SourceType:    "task",
		SourceID:      "task-dup",
		Status:        "pending",
		Payload:       payload,
		TargetChannel: "local_cli",
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	// Insert 3 events where 2 have the same event_id
	inserted, err := repo.CreateEvents(ctx, tx, []OutboxEvent{event, event, {
		EventID:       "unique-event",
		EventType:     "task_assigned",
		SourceType:    "task",
		SourceID:      "task-unique",
		Status:        "pending",
		Payload:       payload,
		TargetChannel: "local_cli",
	}})
	if err != nil {
		t.Fatalf("CreateEvents failed: %v", err)
	}
	if inserted != 2 {
		t.Errorf("expected 2 newly inserted rows, got %d", inserted)
	}

	var count int64
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM ops.outbox_event`).Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 total rows, got %d", count)
	}
}

func TestOutboxRepository_CreateEvents_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOutboxTestDB(t)
	defer pool.Close()

	repo := NewOutboxRepository()
	ctx := context.Background()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	inserted, err := repo.CreateEvents(ctx, tx, nil)
	if err != nil {
		t.Fatalf("CreateEvents with nil slice failed: %v", err)
	}
	if inserted != 0 {
		t.Errorf("expected 0 for empty input, got %d", inserted)
	}

	inserted, err = repo.CreateEvents(ctx, tx, []OutboxEvent{})
	if err != nil {
		t.Fatalf("CreateEvents with empty slice failed: %v", err)
	}
	if inserted != 0 {
		t.Errorf("expected 0 for empty slice, got %d", inserted)
	}
}

// itoa converts an int to its ASCII decimal string representation.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
