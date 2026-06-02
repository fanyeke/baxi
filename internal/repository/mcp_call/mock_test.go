package mcp_call

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/repository/common"
)

func TestMockCreate(t *testing.T) {
	var capturedArgs []interface{}
	mock := &common.MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			capturedArgs = args
			now := time.Now().UTC()
			return common.NewMockRow(int64(1), now)
		},
	}

	repo := NewRepository(mock)
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
	assert.Equal(t, int64(1), entry.CallID)
	require.Len(t, capturedArgs, 9)
	assert.Equal(t, "my-server", capturedArgs[1])
	assert.Equal(t, "get_weather", capturedArgs[2])
}

func TestMockCreate_Error(t *testing.T) {
	mock := &common.MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return common.NewMockRowError(fmt.Errorf("insert failed"))
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	entry := &MCPCall{
		ServerName: "my-server",
		ToolName:   "get_weather",
		Status:     "success",
	}

	err := repo.Create(ctx, entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insert failed")
}

func TestMockGetByID(t *testing.T) {
	now := time.Now().UTC()
	mock := &common.MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return common.NewMockRow(
				int64(1), strPtr("req-001"), "my-server", "get_weather",
				json.RawMessage(`{"city":"Beijing"}`), json.RawMessage(`{"temp":22}`),
				"success", nil, int64Ptr(1200), now,
			)
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	call, err := repo.GetByID(ctx, 1)

	require.NoError(t, err)
	require.NotNil(t, call)
	assert.Equal(t, int64(1), call.CallID)
	assert.Equal(t, "my-server", call.ServerName)
	assert.Equal(t, "get_weather", call.ToolName)
	assert.Equal(t, "success", call.Status)
}

func TestMockGetByID_NotFound(t *testing.T) {
	mock := &common.MockQuerier{
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return common.NewMockRowError(fmt.Errorf("no rows in result set"))
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	call, err := repo.GetByID(ctx, 999)

	assert.Error(t, err)
	assert.Nil(t, call)
}

func TestMockList(t *testing.T) {
	now := time.Now().UTC()
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			return common.NewMockRows([][]interface{}{
				{int64(1), strPtr("req-001"), "server-a", "tool-a",
					json.RawMessage(`{}`), json.RawMessage(`{}`),
					"success", nil, int64Ptr(100), now, 2},
				{int64(2), strPtr("req-002"), "server-b", "tool-b",
					json.RawMessage(`{}`), json.RawMessage(`{}`),
					"error", nil, int64Ptr(200), now, 2},
			}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	results, total, err := repo.List(ctx, 10, 0)

	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, results, 2)
	assert.Equal(t, int64(1), results[0].CallID)
	assert.Equal(t, "server-a", results[0].ServerName)
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
	now := time.Now().UTC()
	mock := &common.MockQuerier{
		QueryFunc: func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
			capturedArgs = args
			return common.NewMockRows([][]interface{}{
				{int64(1), nil, "server-a", "tool-a",
					json.RawMessage(`{}`), json.RawMessage(`{}`),
					"success", nil, nil, now, 5},
			}), nil
		},
	}

	repo := NewRepository(mock)
	ctx := context.Background()
	_, _, err := repo.List(ctx, 2, 3)

	require.NoError(t, err)
	require.Len(t, capturedArgs, 2)
	assert.Equal(t, 2, capturedArgs[0])
	assert.Equal(t, 3, capturedArgs[1])
}
