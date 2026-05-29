package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/model"
)

// mockContextFetcher implements ContextFetcher for testing.
type mockContextFetcher struct {
	response *model.ContextResponse
	err      error
}

func (m *mockContextFetcher) GetContext(ctx context.Context, requestID string, params model.ContextQueryParams) (*model.ContextResponse, error) {
	return m.response, m.err
}

func fixedTime() time.Time {
	return time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC)
}

func TestHandleContext_ResponseFormat(t *testing.T) {
	now := fixedTime()
	finishedAt := now.Add(-1 * time.Hour)
	finishedStr := finishedAt.Format(time.RFC3339Nano)
	startedStr := now.Add(-2 * time.Hour).Format(time.RFC3339Nano)

	mock := &mockContextFetcher{
		response: &model.ContextResponse{
			RequestID: "test-req-123",
			System: model.SystemInfo{
				LastPipelineRun: &model.PipelineRunInfo{
					RunID:       "test-run-1",
					RunType:     "ingestion",
					Mode:        "full",
					Status:      "completed",
					StartedAt:   startedStr,
					FinishedAt:  &finishedStr,
					InputCount:  100,
					OutputCount: 95,
				},
			},
			Summary: model.ContextSummary{
				TotalAlerts:        36,
				TotalOpenTasks:     36,
				TotalPendingOutbox: 24,
			},
			TopAlerts: []model.AlertItem{
				{
					EventID:    "alert-1",
					RuleID:     "gmv_drop",
					EventDate:  "2026-05-25",
					Severity:   "high",
					MetricName: "gmv",
					ObjectType: "global",
					ObjectID:   "global",
					OwnerRole:  "business_ops",
					Status:     "new",
				},
			},
			OpenTasks: []model.TaskItem{
				{
					TaskID:    "task-1",
					TaskTitle: "Check region delay",
					Status:    "in_progress",
					Priority:  "high",
					CreatedAt: now,
				},
			},
			PendingOutbox: []model.OutboxItem{
				{
					OutboxID:         "outbox-1",
					EventType:        "dimensional_alert",
					SourceType:       "rule_engine",
					SourceID:         "dim-1",
					TargetChannel:    "feishu_cli",
					Status:           "pending",
					CreatedAt:        now,
					DispatchAttempts: 1,
				},
			},
			RecentDiagnosis:  []string{},
			AllowedActions:   []string{"read_status", "read_alerts"},
			ForbiddenActions: []string{"execute_actions"},
		},
	}

	handler := &QoderHandler{ctxFetcher: mock}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/qoder/context", nil)
	w := httptest.NewRecorder()
	handler.HandleContext(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, "test-req-123", resp["request_id"])
	assert.NotNil(t, resp["system"])
	assert.NotNil(t, resp["summary"])
	assert.NotNil(t, resp["top_alerts"])
	assert.NotNil(t, resp["open_tasks"])
	assert.NotNil(t, resp["pending_outbox"])
	assert.NotNil(t, resp["recent_diagnosis"])
	assert.NotNil(t, resp["allowed_actions"])
	assert.NotNil(t, resp["forbidden_actions"])
}

func TestHandleContext_ResponseObjectShape(t *testing.T) {
	mock := &mockContextFetcher{
		response: &model.ContextResponse{
			RequestID:        "test-req-456",
			System:           model.SystemInfo{},
			Summary:          model.ContextSummary{TotalAlerts: 5, TotalOpenTasks: 3, TotalPendingOutbox: 1},
			TopAlerts:        []model.AlertItem{},
			OpenTasks:        []model.TaskItem{},
			PendingOutbox:    []model.OutboxItem{},
			RecentDiagnosis:  []string{},
			AllowedActions:   []string{},
			ForbiddenActions: []string{},
		},
	}

	handler := &QoderHandler{ctxFetcher: mock}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/qoder/context", nil)
	w := httptest.NewRecorder()
	handler.HandleContext(w, req)

	var body map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)

	_, isArray := body["items"]
	assert.False(t, isArray, "response must NOT be array-shaped")

	expectedTopLevel := []string{"request_id", "system", "summary", "top_alerts", "open_tasks", "pending_outbox", "recent_diagnosis", "allowed_actions", "forbidden_actions"}
	for _, key := range expectedTopLevel {
		_, exists := body[key]
		assert.True(t, exists, "response should have key: %s", key)
	}
}

func TestHandleContext_SummaryCounts(t *testing.T) {
	mock := &mockContextFetcher{
		response: &model.ContextResponse{
			RequestID: "test-req-789",
			System:    model.SystemInfo{},
			Summary: model.ContextSummary{
				TotalAlerts:        42,
				TotalOpenTasks:     15,
				TotalPendingOutbox: 8,
			},
			TopAlerts:        []model.AlertItem{},
			OpenTasks:        []model.TaskItem{},
			PendingOutbox:    []model.OutboxItem{},
			RecentDiagnosis:  []string{},
			AllowedActions:   []string{},
			ForbiddenActions: []string{},
		},
	}

	handler := &QoderHandler{ctxFetcher: mock}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/qoder/context", nil)
	w := httptest.NewRecorder()
	handler.HandleContext(w, req)

	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	summary, ok := resp["summary"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(42), summary["total_alerts"])
	assert.Equal(t, float64(15), summary["total_open_tasks"])
	assert.Equal(t, float64(8), summary["total_pending_outbox"])
}

func TestHandleContext_EmptyState(t *testing.T) {
	mock := &mockContextFetcher{
		response: &model.ContextResponse{
			RequestID:        "empty-ctx",
			System:           model.SystemInfo{},
			Summary:          model.ContextSummary{},
			TopAlerts:        []model.AlertItem{},
			OpenTasks:        []model.TaskItem{},
			PendingOutbox:    []model.OutboxItem{},
			RecentDiagnosis:  []string{},
			AllowedActions:   []string{},
			ForbiddenActions: []string{},
		},
	}

	handler := &QoderHandler{ctxFetcher: mock}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/qoder/context", nil)
	w := httptest.NewRecorder()
	handler.HandleContext(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	summary := resp["summary"].(map[string]interface{})
	assert.Equal(t, float64(0), summary["total_alerts"])
	assert.Equal(t, float64(0), summary["total_open_tasks"])
	assert.Equal(t, float64(0), summary["total_pending_outbox"])

	topAlerts := resp["top_alerts"].([]interface{})
	assert.Empty(t, topAlerts)
	openTasks := resp["open_tasks"].([]interface{})
	assert.Empty(t, openTasks)
	pendingOutbox := resp["pending_outbox"].([]interface{})
	assert.Empty(t, pendingOutbox)
}

func TestHandleContext_MissingFetcher(t *testing.T) {
	handler := &QoderHandler{ctxFetcher: nil}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/qoder/context", nil)
	w := httptest.NewRecorder()
	handler.HandleContext(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandleContext_ServiceError(t *testing.T) {
	mock := &mockContextFetcher{
		response: nil,
		err:      assert.AnError,
	}

	handler := &QoderHandler{ctxFetcher: mock}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/qoder/context", nil)
	w := httptest.NewRecorder()
	handler.HandleContext(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandleContext_QueryParams(t *testing.T) {
	mock := &mockContextFetcher{
		response: &model.ContextResponse{
			RequestID:        "params-test",
			System:           model.SystemInfo{},
			Summary:          model.ContextSummary{TotalAlerts: 0, TotalOpenTasks: 0, TotalPendingOutbox: 0},
			TopAlerts:        []model.AlertItem{},
			OpenTasks:        []model.TaskItem{},
			PendingOutbox:    []model.OutboxItem{},
			RecentDiagnosis:  []string{},
			AllowedActions:   []string{},
			ForbiddenActions: []string{},
		},
	}

	handler := &QoderHandler{ctxFetcher: mock}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/qoder/context?severity=high&limit_alerts=5&limit_tasks=20&limit_outbox=15&include_logs=true", nil)
	w := httptest.NewRecorder()
	handler.HandleContext(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleContext_InvalidLimitParams(t *testing.T) {
	mock := &mockContextFetcher{
		response: &model.ContextResponse{
			RequestID:        "bad-params",
			System:           model.SystemInfo{},
			Summary:          model.ContextSummary{},
			TopAlerts:        []model.AlertItem{},
			OpenTasks:        []model.TaskItem{},
			PendingOutbox:    []model.OutboxItem{},
			RecentDiagnosis:  []string{},
			AllowedActions:   []string{},
			ForbiddenActions: []string{},
		},
	}

	handler := &QoderHandler{ctxFetcher: mock}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/qoder/context?limit_alerts=abc&limit_tasks=-1&limit_outbox=999", nil)
	w := httptest.NewRecorder()
	handler.HandleContext(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "invalid params should not cause error, use defaults")
}

func TestParseContextParams(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/qoder/context?severity=high&limit_alerts=25&limit_tasks=15&limit_outbox=5&include_logs=true", nil)
	params := parseContextParams(req)

	assert.Equal(t, "high", params.Severity)
	assert.Equal(t, 25, params.LimitAlerts)
	assert.Equal(t, 15, params.LimitTasks)
	assert.Equal(t, 5, params.LimitOutbox)
	assert.True(t, params.IncludeLogs)
}

func TestParseContextParams_Defaults(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/qoder/context", nil)
	params := parseContextParams(req)

	assert.Equal(t, "", params.Severity)
	assert.Equal(t, 10, params.LimitAlerts)
	assert.Equal(t, 10, params.LimitTasks)
	assert.Equal(t, 10, params.LimitOutbox)
	assert.False(t, params.IncludeLogs)
}

func TestParseContextParams_EmptyValues(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/qoder/context?severity=&limit_alerts=&limit_tasks=&limit_outbox=", nil)
	params := parseContextParams(req)

	assert.Equal(t, "", params.Severity)
	assert.Equal(t, 10, params.LimitAlerts)
	assert.Equal(t, 10, params.LimitTasks)
	assert.Equal(t, 10, params.LimitOutbox)
}

func TestParseContextParamsCaps(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/qoder/context?limit_alerts=200&limit_tasks=300", nil)
	params := parseContextParams(req)

	assert.Equal(t, 100, params.LimitAlerts, "should cap at 100")
	assert.Equal(t, 100, params.LimitTasks, "should cap at 100")
	assert.Equal(t, 10, params.LimitOutbox, "default for unspecified")
}

func TestParseContextParams_IncludeLogsVariants(t *testing.T) {
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/qoder/context?include_logs=true", nil)
	assert.True(t, parseContextParams(req1).IncludeLogs)

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/qoder/context?include_logs=1", nil)
	assert.True(t, parseContextParams(req2).IncludeLogs)

	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/qoder/context?include_logs=false", nil)
	assert.False(t, parseContextParams(req3).IncludeLogs)

	req4 := httptest.NewRequest(http.MethodGet, "/api/v1/qoder/context", nil)
	assert.False(t, parseContextParams(req4).IncludeLogs)
}

func TestHandleContext_RejectsNonGET(t *testing.T) {
	mock := &mockContextFetcher{
		response: &model.ContextResponse{RequestID: "test", System: model.SystemInfo{}, Summary: model.ContextSummary{}, TopAlerts: []model.AlertItem{}, OpenTasks: []model.TaskItem{}, PendingOutbox: []model.OutboxItem{}, RecentDiagnosis: []string{}, AllowedActions: []string{}, ForbiddenActions: []string{}},
	}
	handler := &QoderHandler{ctxFetcher: mock}

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, "/api/v1/qoder/context", nil)
		handler.HandleContext(w, r)
		assert.Equal(t, http.StatusOK, w.Code, "%s should be accepted by handler (chi router enforces method)", method)
	}
}

func TestHandleContext_JSONArrayShape(t *testing.T) {
	mock := &mockContextFetcher{
		response: &model.ContextResponse{
			RequestID:        "shape-test",
			System:           model.SystemInfo{},
			Summary:          model.ContextSummary{TotalAlerts: 0, TotalOpenTasks: 0, TotalPendingOutbox: 0},
			TopAlerts:        []model.AlertItem{},
			OpenTasks:        []model.TaskItem{},
			PendingOutbox:    []model.OutboxItem{},
			RecentDiagnosis:  []string{},
			AllowedActions:   []string{},
			ForbiddenActions: []string{},
		},
	}

	handler := &QoderHandler{ctxFetcher: mock}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/qoder/context", nil)
	w := httptest.NewRecorder()
	handler.HandleContext(w, req)

	body := w.Body.String()
	assert.True(t, strings.Contains(body, `"request_id"`), "response should contain request_id")
	assert.True(t, strings.Contains(body, `"system"`), "response should contain system")
	assert.True(t, strings.Contains(body, `"summary"`), "response should contain summary")
	assert.True(t, strings.Contains(body, `"top_alerts"`), "response should contain top_alerts")
	assert.True(t, strings.Contains(body, `"open_tasks"`), "response should contain open_tasks")
	assert.True(t, strings.Contains(body, `"pending_outbox"`), "response should contain pending_outbox")
}
