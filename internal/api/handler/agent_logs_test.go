package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/service"
)

type mockAgentLogService struct {
	listFn func(ctx context.Context, limit, offset int) (*service.AgentLogListResponse, error)
}

func (m *mockAgentLogService) ListAgentLogs(ctx context.Context, limit, offset int) (*service.AgentLogListResponse, error) {
	return m.listFn(ctx, limit, offset)
}

func TestAgentLogHandler_List(t *testing.T) {
	svc := &mockAgentLogService{
		listFn: func(ctx context.Context, limit, offset int) (*service.AgentLogListResponse, error) {
			return &service.AgentLogListResponse{
				Items: []service.AgentExecutionLog{
					{
						ExecutionID: "exec-1",
						ToolName:    "test-tool",
						Status:      "success",
					},
				},
				Total: 1,
			}, nil
		},
	}
	h := NewAgentLogHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/logs/agent", nil)
	w := httptest.NewRecorder()
	h.HandleListAgentLogs(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)
	assert.Contains(t, body, "items")
	assert.Contains(t, body, "total")
}

func TestAgentLogHandler_List_Error(t *testing.T) {
	svc := &mockAgentLogService{
		listFn: func(ctx context.Context, limit, offset int) (*service.AgentLogListResponse, error) {
			return nil, errors.New("service error")
		},
	}
	h := NewAgentLogHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/logs/agent", nil)
	w := httptest.NewRecorder()
	h.HandleListAgentLogs(w, r)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAgentLogHandler_List_Empty(t *testing.T) {
	svc := &mockAgentLogService{
		listFn: func(ctx context.Context, limit, offset int) (*service.AgentLogListResponse, error) {
			return &service.AgentLogListResponse{Items: []service.AgentExecutionLog{}, Total: 0}, nil
		},
	}
	h := NewAgentLogHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/logs/agent", nil)
	w := httptest.NewRecorder()
	h.HandleListAgentLogs(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)
	items := body["items"].([]interface{})
	assert.Empty(t, items)
	assert.Equal(t, float64(0), body["total"])
}

func TestAgentLogHandler_List_BadPagination(t *testing.T) {
	svc := &mockAgentLogService{}
	h := NewAgentLogHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/logs/agent?limit=abc", nil)
	w := httptest.NewRecorder()
	h.HandleListAgentLogs(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var body map[string]string
	err := json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)
	assert.Contains(t, body["error"], "invalid limit")
}
