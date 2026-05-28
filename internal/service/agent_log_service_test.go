package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agent_execution "baxi/internal/repository/agent_execution"
)

// mockAgentExecutionRepo implements AgentExecutionRepository with function fields.
type mockAgentExecutionRepo struct {
	createFn func(ctx context.Context, execution *agent_execution.AgentExecution) error
	listFn   func(ctx context.Context, limit, offset int) ([]agent_execution.AgentExecution, int, error)
}

func (m *mockAgentExecutionRepo) Create(ctx context.Context, execution *agent_execution.AgentExecution) error {
	return m.createFn(ctx, execution)
}

func (m *mockAgentExecutionRepo) List(ctx context.Context, limit, offset int) ([]agent_execution.AgentExecution, int, error) {
	return m.listFn(ctx, limit, offset)
}

func TestAgentLogService_LogExecution(t *testing.T) {
	var captured *agent_execution.AgentExecution
	mock := &mockAgentExecutionRepo{
		createFn: func(ctx context.Context, execution *agent_execution.AgentExecution) error {
			captured = execution
			return nil
		},
	}

	svc := NewAgentLogService(mock, nil)
	err := svc.LogExecution(context.Background(), &AgentExecutionLog{
		ExecutionID: "exec-1",
		ToolName:    "test-tool",
		Status:      "success",
	})

	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.Equal(t, "exec-1", captured.ExecutionID)
	assert.Equal(t, "test-tool", captured.ToolName)
	assert.Equal(t, "success", captured.Status)
}

func TestAgentLogService_LogExecution_Error(t *testing.T) {
	expectedErr := errors.New("db error")
	mock := &mockAgentExecutionRepo{
		createFn: func(ctx context.Context, execution *agent_execution.AgentExecution) error {
			return expectedErr
		},
	}

	svc := NewAgentLogService(mock, nil)
	err := svc.LogExecution(context.Background(), &AgentExecutionLog{
		ExecutionID: "exec-2",
		ToolName:    "test-tool",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
}

func TestAgentLogService_ListAgentLogs(t *testing.T) {
	mock := &mockAgentExecutionRepo{
		listFn: func(ctx context.Context, limit, offset int) ([]agent_execution.AgentExecution, int, error) {
			assert.Equal(t, 10, limit)
			assert.Equal(t, 0, offset)
			return []agent_execution.AgentExecution{
				{ExecutionID: "exec-1", ToolName: "tool-1", Status: "success"},
				{ExecutionID: "exec-2", ToolName: "tool-2", Status: "failed"},
			}, 2, nil
		},
	}

	svc := NewAgentLogService(mock, nil)
	resp, err := svc.ListAgentLogs(context.Background(), 10, 0)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 2, resp.Total)
	assert.Len(t, resp.Items, 2)
	assert.Equal(t, "exec-1", resp.Items[0].ExecutionID)
	assert.Equal(t, "tool-1", resp.Items[0].ToolName)
	assert.Equal(t, "success", resp.Items[0].Status)
	assert.Equal(t, "exec-2", resp.Items[1].ExecutionID)
	assert.Equal(t, "tool-2", resp.Items[1].ToolName)
	assert.Equal(t, "failed", resp.Items[1].Status)
}

func TestAgentLogService_ListAgentLogs_Error(t *testing.T) {
	expectedErr := errors.New("list error")
	mock := &mockAgentExecutionRepo{
		listFn: func(ctx context.Context, limit, offset int) ([]agent_execution.AgentExecution, int, error) {
			return nil, 0, expectedErr
		},
	}

	svc := NewAgentLogService(mock, nil)
	resp, err := svc.ListAgentLogs(context.Background(), 10, 0)

	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, resp)
}
