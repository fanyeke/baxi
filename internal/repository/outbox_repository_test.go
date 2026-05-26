package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const outboxTableDDL = `
CREATE SCHEMA IF NOT EXISTS ops;

CREATE TABLE IF NOT EXISTS ops.outbox_event (
    event_id            TEXT PRIMARY KEY,
    event_type          TEXT NOT NULL,
    source_type         TEXT NOT NULL,
    source_id           TEXT NOT NULL,
    payload_json        JSONB NOT NULL DEFAULT '{}',
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

func setupOutboxTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	ctx := context.Background()
	_, err = pool.Exec(ctx, outboxTableDDL)
	require.NoError(t, err)

	_, _ = pool.Exec(ctx, "TRUNCATE TABLE ops.outbox_event CASCADE")
	return pool
}

func insertTestEvent(t *testing.T, pool *pgxpool.Pool, id, eventType, sourceType, sourceID, channel, status string, attempts int, lastDispatch *time.Time, createdAt time.Time) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO ops.outbox_event (event_id, event_type, source_type, source_id, payload_json, target_channel, status, dispatch_attempts, last_dispatch_at, created_at)
		VALUES ($1, $2, $3, $4, '{}', $5, $6, $7, $8, $9)
	`, id, eventType, sourceType, sourceID, channel, status, attempts, lastDispatch, createdAt)
	require.NoError(t, err)
}

func TestOutboxRepository_ListOutboxEvents_NoFilters(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOutboxTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC()
	insertTestEvent(t, pool, "evt-1", "alert", "rule_engine", "rule-1", "feishu", "pending", 0, nil, now.Add(-2*time.Hour))
	insertTestEvent(t, pool, "evt-2", "alert", "rule_engine", "rule-2", "email", "dispatched", 1, &now, now.Add(-1*time.Hour))
	insertTestEvent(t, pool, "evt-3", "task", "scheduler", "task-1", "manual", "pending", 2, nil, now)

	repo := NewOutboxRepository()
	items, total, err := repo.ListOutboxEvents(ctx, pool, OutboxFilters{}, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 3, total, "total should match all rows")
	assert.Len(t, items, 3, "should return all items")

	assert.Equal(t, "evt-3", items[0].OutboxID)
	assert.Equal(t, "evt-1", items[2].OutboxID)
}

func TestOutboxRepository_ListOutboxEvents_FilterByStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOutboxTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC()
	insertTestEvent(t, pool, "evt-1", "alert", "rule_engine", "rule-1", "feishu", "pending", 0, nil, now)
	insertTestEvent(t, pool, "evt-2", "alert", "rule_engine", "rule-2", "email", "dispatched", 1, &now, now)

	pending := "pending"
	filters := OutboxFilters{Status: &pending}

	repo := NewOutboxRepository()
	items, total, err := repo.ListOutboxEvents(ctx, pool, filters, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, items, 1)
	assert.Equal(t, "evt-1", items[0].OutboxID)
}

func TestOutboxRepository_ListOutboxEvents_FilterByChannel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOutboxTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC()
	insertTestEvent(t, pool, "evt-1", "alert", "rule_engine", "rule-1", "feishu", "pending", 0, nil, now)
	insertTestEvent(t, pool, "evt-2", "alert", "rule_engine", "rule-2", "email", "dispatched", 1, &now, now)

	channel := "feishu"
	filters := OutboxFilters{Channel: &channel}

	repo := NewOutboxRepository()
	items, total, err := repo.ListOutboxEvents(ctx, pool, filters, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, items, 1)
	assert.Equal(t, "feishu", items[0].TargetChannel)
}

func TestOutboxRepository_ListOutboxEvents_FilterByEventType(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOutboxTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC()
	insertTestEvent(t, pool, "evt-1", "alert", "rule_engine", "rule-1", "feishu", "pending", 0, nil, now)
	insertTestEvent(t, pool, "evt-2", "task", "scheduler", "task-1", "feishu", "pending", 0, nil, now)

	eventType := "alert"
	filters := OutboxFilters{EventType: &eventType}

	repo := NewOutboxRepository()
	items, total, err := repo.ListOutboxEvents(ctx, pool, filters, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, items, 1)
	assert.Equal(t, "alert", items[0].EventType)
}

func TestOutboxRepository_ListOutboxEvents_CombinedFilters(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOutboxTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC()
	insertTestEvent(t, pool, "evt-1", "alert", "rule_engine", "rule-1", "feishu", "pending", 0, nil, now)
	insertTestEvent(t, pool, "evt-2", "alert", "rule_engine", "rule-2", "feishu", "dispatched", 1, &now, now)
	insertTestEvent(t, pool, "evt-3", "task", "scheduler", "task-1", "email", "pending", 0, nil, now)

	pending := "pending"
	feishu := "feishu"
	filters := OutboxFilters{Status: &pending, Channel: &feishu}

	repo := NewOutboxRepository()
	items, total, err := repo.ListOutboxEvents(ctx, pool, filters, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, items, 1)
	assert.Equal(t, "evt-1", items[0].OutboxID)
}

func TestOutboxRepository_ListOutboxEvents_Pagination(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOutboxTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC()
	insertTestEvent(t, pool, "evt-1", "alert", "rule", "rule-1", "feishu", "pending", 0, nil, now.Add(-3*time.Hour))
	insertTestEvent(t, pool, "evt-2", "alert", "rule", "rule-2", "feishu", "pending", 0, nil, now.Add(-2*time.Hour))
	insertTestEvent(t, pool, "evt-3", "alert", "rule", "rule-3", "feishu", "pending", 0, nil, now.Add(-1*time.Hour))

	repo := NewOutboxRepository()

	items, total, err := repo.ListOutboxEvents(ctx, pool, OutboxFilters{}, 2, 0)
	require.NoError(t, err)
	assert.Equal(t, 3, total, "total should be all matching rows")
	assert.Len(t, items, 2, "should return only limit")
	assert.Equal(t, "evt-3", items[0].OutboxID)
	assert.Equal(t, "evt-2", items[1].OutboxID)
}

func TestOutboxRepository_ListOutboxEvents_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOutboxTestDB(t)
	ctx := context.Background()

	repo := NewOutboxRepository()
	items, total, err := repo.ListOutboxEvents(ctx, pool, OutboxFilters{}, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, items)
}

func TestOutboxRepository_ListOutboxEvents_NullLastDispatch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupOutboxTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC()
	insertTestEvent(t, pool, "evt-1", "alert", "rule", "rule-1", "feishu", "pending", 0, nil, now)

	repo := NewOutboxRepository()
	items, total, err := repo.ListOutboxEvents(ctx, pool, OutboxFilters{}, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, items, 1)
	assert.Nil(t, items[0].LastDispatchAt, "last_dispatch_at should be nil when NULL in DB")
}
