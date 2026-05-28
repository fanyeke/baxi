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
	"baxi/internal/repository"
)

const svcOutboxTableDDL = `
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

func setupSvcOutboxTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	ctx := context.Background()
	_, err = pool.Exec(ctx, svcOutboxTableDDL)
	require.NoError(t, err)

	_, _ = pool.Exec(ctx, "TRUNCATE TABLE ops.outbox_event CASCADE")
	return pool
}

func insertSvcTestEvent(t *testing.T, pool *pgxpool.Pool, id, eventType, sourceType, sourceID, channel, status string, attempts int, lastDispatch *time.Time, createdAt time.Time) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO ops.outbox_event (event_id, event_type, source_type, source_id, payload_json, target_channel, status, dispatch_attempts, last_dispatch_at, created_at)
		VALUES ($1, $2, $3, $4, '{}', $5, $6, $7, $8, $9)
	`, id, eventType, sourceType, sourceID, channel, status, attempts, lastDispatch, createdAt)
	require.NoError(t, err)
}

func TestOutboxService_List(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupSvcOutboxTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC()
	insertSvcTestEvent(t, pool, "evt-1", "alert", "rule_engine", "rule-1", "feishu", "pending", 0, nil, now.Add(-2*time.Hour))
	insertSvcTestEvent(t, pool, "evt-2", "task", "scheduler", "task-1", "email", "dispatched", 1, &now, now.Add(-1*time.Hour))
	insertSvcTestEvent(t, pool, "evt-3", "alert", "rule_engine", "rule-2", "manual", "skipped", 2, &now, now)

	svc := NewOutboxService(repository.NewOutboxRepository(), pool)

	t.Run("no filters returns all items ordered by created_at DESC", func(t *testing.T) {
		resp, err := svc.List(ctx, model.OutboxFilters{}, 10, 0)
		require.NoError(t, err)
		assert.Equal(t, 3, resp.Total)
		assert.Len(t, resp.Items, 3)
		assert.Equal(t, "evt-3", resp.Items[0].OutboxID)
		assert.Equal(t, "evt-1", resp.Items[2].OutboxID)
	})

	t.Run("filter by status", func(t *testing.T) {
		resp, err := svc.List(ctx, model.OutboxFilters{Status: strPtr("pending")}, 10, 0)
		require.NoError(t, err)
		assert.Equal(t, 1, resp.Total)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "pending", resp.Items[0].Status)
	})

	t.Run("filter by channel", func(t *testing.T) {
		resp, err := svc.List(ctx, model.OutboxFilters{Channel: strPtr("feishu")}, 10, 0)
		require.NoError(t, err)
		assert.Equal(t, 1, resp.Total)
		assert.Equal(t, "feishu", resp.Items[0].TargetChannel)
	})

	t.Run("pagination limits results but preserves total", func(t *testing.T) {
		resp, err := svc.List(ctx, model.OutboxFilters{}, 2, 0)
		require.NoError(t, err)
		assert.Equal(t, 3, resp.Total)
		assert.Len(t, resp.Items, 2)
	})

	t.Run("DTO fields are populated correctly", func(t *testing.T) {
		resp, err := svc.List(ctx, model.OutboxFilters{Channel: strPtr("email")}, 10, 0)
		require.NoError(t, err)
		require.Len(t, resp.Items, 1)

		item := resp.Items[0]
		assert.Equal(t, "evt-2", item.OutboxID)
		assert.Equal(t, "task", item.EventType)
		assert.Equal(t, "scheduler", item.SourceType)
		assert.Equal(t, "task-1", item.SourceID)
		assert.Equal(t, "email", item.TargetChannel)
		assert.Equal(t, "dispatched", item.Status)
		assert.Equal(t, 1, item.DispatchAttempts)
		assert.NotNil(t, item.LastDispatchAt)
	})
}

func TestOutboxService_List_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	pool := setupSvcOutboxTestDB(t)
	ctx := context.Background()

	svc := NewOutboxService(repository.NewOutboxRepository(), pool)
	resp, err := svc.List(ctx, model.OutboxFilters{}, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Total)
	assert.Empty(t, resp.Items)
}

func strPtr(s string) *string {
	return &s
}
