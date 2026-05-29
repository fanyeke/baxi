package agent_execution

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
CREATE SCHEMA IF NOT EXISTS ai;

CREATE TABLE IF NOT EXISTS ai.agent_execution (
    execution_id    TEXT PRIMARY KEY,
    session_id      TEXT,
    tool_name       TEXT NOT NULL,
    input_args      JSONB,
    output_result   JSONB,
    status          TEXT NOT NULL,
    error_message   TEXT,
    duration_ms     BIGINT,
    llm_model       TEXT,
    llm_tokens      BIGINT,
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

	exec := &AgentExecution{
		ExecutionID:  "exec-001",
		SessionID:    strPtr("sess-001"),
		ToolName:     "analyze",
		InputArgs:    json.RawMessage(`{"query":"test"}`),
		OutputResult: json.RawMessage(`{"result":"ok"}`),
		Status:       "completed",
		DurationMs:   int64Ptr(1500),
		LLMModel:     strPtr("gpt-4"),
		LLMTokens:    int64Ptr(500),
	}

	err := repo.Create(ctx, exec)
	require.NoError(t, err)

	// Verify by fetching
	got, err := repo.GetByID(ctx, "exec-001")
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, exec.ExecutionID, got.ExecutionID)
	assert.Equal(t, exec.SessionID, got.SessionID)
	assert.Equal(t, exec.ToolName, got.ToolName)
	assert.JSONEq(t, string(exec.InputArgs), string(got.InputArgs))
	assert.JSONEq(t, string(exec.OutputResult), string(got.OutputResult))
	assert.Equal(t, exec.Status, got.Status)
	assert.Equal(t, exec.DurationMs, got.DurationMs)
	assert.Equal(t, exec.LLMModel, got.LLMModel)
	assert.Equal(t, exec.LLMTokens, got.LLMTokens)
	assert.WithinDuration(t, time.Now(), got.CreatedAt, 5*time.Second)
}

func TestGetByID_NotFound(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	got, err := repo.GetByID(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestList(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	// Insert two records
	exec1 := &AgentExecution{
		ExecutionID:  "exec-list-001",
		SessionID:    strPtr("sess-list"),
		ToolName:     "tool-a",
		InputArgs:    json.RawMessage(`{}`),
		OutputResult: json.RawMessage(`{}`),
		Status:       "completed",
	}
	exec2 := &AgentExecution{
		ExecutionID:  "exec-list-002",
		SessionID:    strPtr("sess-list"),
		ToolName:     "tool-b",
		InputArgs:    json.RawMessage(`{}`),
		OutputResult: json.RawMessage(`{}`),
		Status:       "failed",
	}

	err := repo.Create(ctx, exec1)
	require.NoError(t, err)
	err = repo.Create(ctx, exec2)
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
