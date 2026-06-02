package agent_execution

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/repository/common"
)

func TestMockCreate(t *testing.T) {
	var capturedArgs []interface{}
	mock := &common.MockQuerier{
		ExecFunc: func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
			capturedArgs = args
			return common.MockCommandTag(1), nil
		},
	}

	repo := NewRepository(mock)
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
	require.Len(t, capturedArgs, 11)
	assert.Equal(t, "exec-001", capturedArgs[0])
	assert.Equal(t, "analyze", capturedArgs[2])
	assert.False(t, exec.CreatedAt.IsZero(), "CreatedAt should be set")
}

func TestMockCreate_Error(t *testing.T) {
	mock := &common.MockQuerier{
		ExecFunc: func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, fmt.Errorf("insert failed")
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	exec := &AgentExecution{
		ExecutionID: "exec-001",
		ToolName:    "analyze",
		Status:      "completed",
	}

	err := repo.Create(ctx, exec)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insert failed")
}

func TestMockGetByID(t *testing.T) {
	now := time.Now().UTC()
	mock := &common.MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return common.NewMockRow(
				"exec-001", strPtr("sess-001"), "analyze",
				json.RawMessage(`{"query":"test"}`), json.RawMessage(`{"result":"ok"}`),
				"completed", nil, int64Ptr(1500), strPtr("gpt-4"), int64Ptr(500), now,
			)
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	exec, err := repo.GetByID(ctx, "exec-001")

	require.NoError(t, err)
	require.NotNil(t, exec)
	assert.Equal(t, "exec-001", exec.ExecutionID)
	assert.Equal(t, "analyze", exec.ToolName)
	assert.Equal(t, "completed", exec.Status)
}

func TestMockGetByID_NotFound(t *testing.T) {
	mock := &common.MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return common.NewMockRowError(fmt.Errorf("no rows in result set"))
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	exec, err := repo.GetByID(ctx, "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, exec)
}

func TestMockList(t *testing.T) {
	now := time.Now().UTC()
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{
				{"exec-001", strPtr("sess-001"), "tool-a",
					json.RawMessage(`{}`), json.RawMessage(`{}`),
					"completed", nil, int64Ptr(100), strPtr("gpt-4"), int64Ptr(500), now, 2},
				{"exec-002", strPtr("sess-001"), "tool-b",
					json.RawMessage(`{}`), json.RawMessage(`{}`),
					"failed", nil, int64Ptr(200), strPtr("gpt-4"), int64Ptr(300), now, 2},
			}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	results, total, err := repo.List(ctx, 10, 0)

	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, results, 2)
	assert.Equal(t, "exec-001", results[0].ExecutionID)
	assert.Equal(t, "completed", results[0].Status)
}

func TestMockList_Empty(t *testing.T) {
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	results, total, err := repo.List(ctx, 10, 0)

	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, results)
}

func TestMockList_Pagination(t *testing.T) {
	var capturedArgs []interface{}
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			capturedArgs = args
			return common.NewMockRows([][]interface{}{}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	_, _, err := repo.List(ctx, 5, 10)

	require.NoError(t, err)
	require.Len(t, capturedArgs, 2)
	assert.Equal(t, 5, capturedArgs[0])
	assert.Equal(t, 10, capturedArgs[1])
}

func TestMockList_QueryError(t *testing.T) {
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	results, total, err := repo.List(ctx, 10, 0)

	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Equal(t, 0, total)
}

func strPtr(s string) *string {
	return &s
}

func int64Ptr(n int64) *int64 {
	return &n
}
