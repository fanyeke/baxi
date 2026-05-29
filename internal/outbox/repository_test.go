package outbox

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

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
    next_retry_at       TIMESTAMPTZ,
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
		"task_id":  "task-gmv_drop_2018-10-17",
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
			"task_id": "task-foo",
			"index":   i,
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

func TestOutboxRepository_GetPendingEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOutboxTestDB(t)
	defer pool.Close()

	repo := NewOutboxRepository()
	ctx := context.Background()

	payload, _ := json.Marshal(map[string]string{"key": "value"})

	event1 := &OutboxEvent{
		EventID: "pending-1", EventType: "task_assigned", SourceType: "task",
		SourceID: "task-1", Status: "pending", Payload: payload, TargetChannel: "local_cli",
	}
	event2 := &OutboxEvent{
		EventID: "pending-2", EventType: "notify_owner", SourceType: "alert",
		SourceID: "alert-1", Status: "pending", Payload: payload, TargetChannel: "feishu",
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	_, err = repo.CreateEvent(ctx, tx, event2)
	if err != nil {
		t.Fatalf("CreateEvent event2: %v", err)
	}
	_, err = repo.CreateEvent(ctx, tx, event1)
	if err != nil {
		t.Fatalf("CreateEvent event1: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit tx: %v", err)
	}

	events, err := repo.GetPendingEvents(ctx, pool, 10)
	if err != nil {
		t.Fatalf("GetPendingEvents failed: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].EventID != "pending-2" {
		t.Errorf("expected first event 'pending-2' (oldest), got %q", events[0].EventID)
	}
	if events[1].EventID != "pending-1" {
		t.Errorf("expected second event 'pending-1' (newest), got %q", events[1].EventID)
	}
}

func TestOutboxRepository_GetPendingEvents_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOutboxTestDB(t)
	defer pool.Close()

	repo := NewOutboxRepository()
	ctx := context.Background()

	events, err := repo.GetPendingEvents(ctx, pool, 10)
	if err != nil {
		t.Fatalf("GetPendingEvents failed: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestOutboxRepository_GetPendingEvents_FailedRetry(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOutboxTestDB(t)
	defer pool.Close()

	repo := NewOutboxRepository()
	ctx := context.Background()

	payload, _ := json.Marshal(map[string]string{"key": "value"})

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO ops.outbox_event (event_id, event_type, source_type, source_id, payload_json, target_channel, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, "event-dispatched", "task_assigned", "task", "task-d", payload, "local_cli", "dispatched")
	if err != nil {
		t.Fatalf("insert dispatched event: %v", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO ops.outbox_event (event_id, event_type, source_type, source_id, payload_json, target_channel, status, dispatch_attempts, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, "event-failed", "notify_owner", "alert", "alert-f", payload, "feishu", "failed", 1, "connection timeout")
	if err != nil {
		t.Fatalf("insert failed event: %v", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO ops.outbox_event (event_id, event_type, source_type, source_id, payload_json, target_channel, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, "event-pending", "export_report", "report", "report-p", payload, "email", "pending")
	if err != nil {
		t.Fatalf("insert pending event: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit tx: %v", err)
	}

	events, err := repo.GetPendingEvents(ctx, pool, 10)
	if err != nil {
		t.Fatalf("GetPendingEvents failed: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events (pending + failed), got %d", len(events))
	}
	ids := make(map[string]bool)
	for _, e := range events {
		ids[e.EventID] = true
	}
	if !ids["event-failed"] {
		t.Errorf("expected failed event to be returned for retry")
	}
	if !ids["event-pending"] {
		t.Errorf("expected pending event to be returned")
	}
	if ids["event-dispatched"] {
		t.Errorf("did not expect dispatched event to be returned")
	}
}

func TestOutboxRepository_GetPendingEvents_Limit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOutboxTestDB(t)
	defer pool.Close()

	repo := NewOutboxRepository()
	ctx := context.Background()

	payload, _ := json.Marshal(map[string]string{"key": "value"})

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	for i := 0; i < 5; i++ {
		_, err := tx.Exec(ctx, `
			INSERT INTO ops.outbox_event (event_id, event_type, source_type, source_id, payload_json, target_channel, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, "limit-event-"+itoa(i), "task_assigned", "task", "task-"+itoa(i), payload, "local_cli", "pending")
		if err != nil {
			t.Fatalf("insert event %d: %v", i, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit tx: %v", err)
	}

	events, err := repo.GetPendingEvents(ctx, pool, 3)
	if err != nil {
		t.Fatalf("GetPendingEvents failed: %v", err)
	}
	if len(events) != 3 {
		t.Errorf("expected 3 events (limited), got %d", len(events))
	}
}

func TestOutboxRepository_MarkDispatched(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOutboxTestDB(t)
	defer pool.Close()

	repo := NewOutboxRepository()
	ctx := context.Background()

	payload, _ := json.Marshal(map[string]string{"key": "value"})

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO ops.outbox_event (event_id, event_type, source_type, source_id, payload_json, target_channel, status, dispatch_attempts)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, "dispatch-me", "task_assigned", "task", "task-d", payload, "local_cli", "pending", 0)
	if err != nil {
		t.Fatalf("insert event: %v", err)
	}

	err = repo.MarkDispatched(ctx, tx, "dispatch-me")
	if err != nil {
		t.Fatalf("MarkDispatched failed: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit tx: %v", err)
	}

	var status string
	var attempts int64
	var dispatchedAt *time.Time
	err = pool.QueryRow(ctx, `
		SELECT status, dispatch_attempts, last_dispatch_at
		FROM ops.outbox_event WHERE event_id = $1
	`, "dispatch-me").Scan(&status, &attempts, &dispatchedAt)
	if err != nil {
		t.Fatalf("query event: %v", err)
	}
	if status != "dispatched" {
		t.Errorf("expected status 'dispatched', got %q", status)
	}
	if attempts != 1 {
		t.Errorf("expected dispatch_attempts=1, got %d", attempts)
	}
	if dispatchedAt == nil {
		t.Errorf("expected last_dispatch_at to be set, got nil")
	}
}

func TestOutboxRepository_MarkDispatched_NotFound(t *testing.T) {
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

	err = repo.MarkDispatched(ctx, tx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent event, got nil")
	}
	if err.Error() != "outbox event nonexistent not found for mark dispatched" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestOutboxRepository_MarkFailed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOutboxTestDB(t)
	defer pool.Close()

	repo := NewOutboxRepository()
	ctx := context.Background()

	payload, _ := json.Marshal(map[string]string{"key": "value"})

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO ops.outbox_event (event_id, event_type, source_type, source_id, payload_json, target_channel, status, dispatch_attempts)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, "fail-me", "task_assigned", "task", "task-f", payload, "local_cli", "pending", 2)
	if err != nil {
		t.Fatalf("insert event: %v", err)
	}

	err = repo.MarkFailed(ctx, tx, "fail-me", "connection refused")
	if err != nil {
		t.Fatalf("MarkFailed failed: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit tx: %v", err)
	}

	var status string
	var attempts int64
	var dispatchedAt *time.Time
	var errMsg *string
	err = pool.QueryRow(ctx, `
		SELECT status, dispatch_attempts, last_dispatch_at, error_message
		FROM ops.outbox_event WHERE event_id = $1
	`, "fail-me").Scan(&status, &attempts, &dispatchedAt, &errMsg)
	if err != nil {
		t.Fatalf("query event: %v", err)
	}
	if status != "failed" {
		t.Errorf("expected status 'failed', got %q", status)
	}
	if attempts != 3 {
		t.Errorf("expected dispatch_attempts=3 (was 2, +1), got %d", attempts)
	}
	if dispatchedAt == nil {
		t.Errorf("expected last_dispatch_at to be set, got nil")
	}
	if errMsg == nil || *errMsg != "connection refused" {
		t.Errorf("expected error_message 'connection refused', got %v", errMsg)
	}
}

func TestOutboxRepository_MarkFailed_NotFound(t *testing.T) {
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

	err = repo.MarkFailed(ctx, tx, "nonexistent", "error")
	if err == nil {
		t.Fatal("expected error for nonexistent event, got nil")
	}
	if err.Error() != "outbox event nonexistent not found for mark failed" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestOutboxRepository_UpdateMaxAttempts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOutboxTestDB(t)
	defer pool.Close()

	repo := NewOutboxRepository()
	ctx := context.Background()

	payload, _ := json.Marshal(map[string]string{"key": "value"})

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO ops.outbox_event (event_id, event_type, source_type, source_id, payload_json, target_channel, status, dispatch_attempts)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, "max-out", "task_assigned", "task", "task-m", payload, "local_cli", "failed", 3)
	if err != nil {
		t.Fatalf("insert event: %v", err)
	}

	err = repo.UpdateMaxAttempts(ctx, tx, "max-out")
	if err != nil {
		t.Fatalf("UpdateMaxAttempts failed: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit tx: %v", err)
	}

	var status string
	var attempts int64
	var errMsg *string
	err = pool.QueryRow(ctx, `
		SELECT status, dispatch_attempts, error_message
		FROM ops.outbox_event WHERE event_id = $1
	`, "max-out").Scan(&status, &attempts, &errMsg)
	if err != nil {
		t.Fatalf("query event: %v", err)
	}
	if status != "failed" {
		t.Errorf("expected status 'failed', got %q", status)
	}
	if attempts != 4 {
		t.Errorf("expected dispatch_attempts=4 (was 3, +1), got %d", attempts)
	}
	if errMsg == nil || *errMsg != "max retry attempts reached" {
		t.Errorf("expected error_message 'max retry attempts reached', got %v", errMsg)
	}
}

func TestOutboxRepository_UpdateMaxAttempts_NotFound(t *testing.T) {
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

	err = repo.UpdateMaxAttempts(ctx, tx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent event, got nil")
	}
	if err.Error() != "outbox event nonexistent not found for max attempts update" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestOutboxRepository_GetEventByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOutboxTestDB(t)
	defer pool.Close()

	repo := NewOutboxRepository()
	ctx := context.Background()

	payload, _ := json.Marshal(map[string]string{"task_id": "task-123"})

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	_, err = repo.CreateEvent(ctx, tx, &OutboxEvent{
		EventID: "get-by-id", EventType: "notify_owner", SourceType: "alert",
		SourceID: "alert-1", Status: "pending", Payload: payload, TargetChannel: "feishu",
	})
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit tx: %v", err)
	}

	event, err := repo.GetEventByID(ctx, pool, "get-by-id")
	if err != nil {
		t.Fatalf("GetEventByID failed: %v", err)
	}
	if event == nil {
		t.Fatal("expected event, got nil")
	}
	if event.EventID != "get-by-id" {
		t.Errorf("expected EventID 'get-by-id', got %q", event.EventID)
	}
	if event.EventType != "notify_owner" {
		t.Errorf("expected EventType 'notify_owner', got %q", event.EventType)
	}
	if event.SourceType != "alert" {
		t.Errorf("expected SourceType 'alert', got %q", event.SourceType)
	}
	if event.SourceID != "alert-1" {
		t.Errorf("expected SourceID 'alert-1', got %q", event.SourceID)
	}
	if event.Status != "pending" {
		t.Errorf("expected Status 'pending', got %q", event.Status)
	}
	if event.TargetChannel != "feishu" {
		t.Errorf("expected TargetChannel 'feishu', got %q", event.TargetChannel)
	}
}

func TestOutboxRepository_GetEventByID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOutboxTestDB(t)
	defer pool.Close()

	repo := NewOutboxRepository()
	ctx := context.Background()

	event, err := repo.GetEventByID(ctx, pool, "does-not-exist")
	if err != nil {
		t.Fatalf("GetEventByID failed: %v", err)
	}
	if event != nil {
		t.Fatal("expected nil for nonexistent event, got non-nil")
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
