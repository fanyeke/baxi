package mcp_call

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/repository/common"
)

const tableDDL = `
CREATE SCHEMA IF NOT EXISTS audit;

CREATE TABLE IF NOT EXISTS audit.mcp_call (
    call_id         BIGSERIAL PRIMARY KEY,
    request_id      TEXT,
    server_name     TEXT NOT NULL,
    tool_name       TEXT NOT NULL,
    input_args      JSONB,
    output_result   JSONB,
    status          TEXT NOT NULL,
    error_message   TEXT,
    duration_ms     BIGINT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`

func setupTestDB(t *testing.T) *Repository {
	t.Helper()

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	_, err = pool.Exec(ctx, tableDDL)
	require.NoError(t, err)

	provider := common.NewPoolProvider(pool)
	return NewRepository(provider)
}

func TestCreate(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	entry := &MCPCall{
		RequestID:    strPtr("req-001"),
		ServerName:   "my-server",
		ToolName:     "get_weather",
		InputArgs:    json.RawMessage(`{"city":"Beijing"}`),
		OutputResult: json.RawMessage(`{"temp":22}`),
		Status:       "success",
		DurationMs:   int64Ptr(1200),
	}

	err := repo.Create(ctx, entry)
	require.NoError(t, err)
	require.NotZero(t, entry.CallID, "CallID should be set by RETURNING")
	require.False(t, entry.CreatedAt.IsZero(), "CreatedAt should be set by RETURNING")

	// Verify by fetching
	got, err := repo.GetByID(ctx, entry.CallID)
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, entry.CallID, got.CallID)
	assert.Equal(t, entry.RequestID, got.RequestID)
	assert.Equal(t, entry.ServerName, got.ServerName)
	assert.Equal(t, entry.ToolName, got.ToolName)
	assert.JSONEq(t, string(entry.InputArgs), string(got.InputArgs))
	assert.JSONEq(t, string(entry.OutputResult), string(got.OutputResult))
	assert.Equal(t, entry.Status, got.Status)
	assert.Equal(t, entry.DurationMs, got.DurationMs)
	assert.WithinDuration(t, time.Now(), got.CreatedAt, 5*time.Second)
}

func TestGetByID_NotFound(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	got, err := repo.GetByID(ctx, 99999)
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestList(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	// Insert two records
	entry1 := &MCPCall{
		ServerName:   "server-a",
		ToolName:     "tool-a",
		InputArgs:    json.RawMessage(`{}`),
		OutputResult: json.RawMessage(`{}`),
		Status:       "success",
	}
	entry2 := &MCPCall{
		ServerName:   "server-b",
		ToolName:     "tool-b",
		InputArgs:    json.RawMessage(`{}`),
		OutputResult: json.RawMessage(`{}`),
		Status:       "error",
	}

	err := repo.Create(ctx, entry1)
	require.NoError(t, err)
	err = repo.Create(ctx, entry2)
	require.NoError(t, err)

	results, total, err := repo.List(ctx, 10, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, 2)
	assert.GreaterOrEqual(t, len(results), 2)

	// Test pagination: limit=1 should return 1 result
	results, total, err = repo.List(ctx, 1, 0)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.GreaterOrEqual(t, total, 2)
}

func strPtr(s string) *string {
	return &s
}

func int64Ptr(n int64) *int64 {
	return &n
}
