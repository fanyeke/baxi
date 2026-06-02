package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/api/dto"
	"baxi/internal/model"
)

type mockLogService struct {
	recentItems  []model.LogItem
	recentTotal  int
	errorItems   []model.LogItem
	errorTotal   int
	auditItems   []model.LogItem
	auditTotal   int
	err          error
	recentCalled bool
	errorCalled  bool
	auditCalled  bool
}

func (m *mockLogService) ListRecent(_ context.Context, _, _ int) (*model.LogListResponse, error) {
	m.recentCalled = true
	if m.err != nil {
		return nil, m.err
	}
	return &model.LogListResponse{Items: m.recentItems, Total: m.recentTotal}, nil
}

func (m *mockLogService) ListErrors(_ context.Context, _, _ int) (*model.LogListResponse, error) {
	m.errorCalled = true
	if m.err != nil {
		return nil, m.err
	}
	return &model.LogListResponse{Items: m.errorItems, Total: m.errorTotal}, nil
}

func (m *mockLogService) ListAudit(_ context.Context, _, _ int) (*model.LogListResponse, error) {
	m.auditCalled = true
	if m.err != nil {
		return nil, m.err
	}
	return &model.LogListResponse{Items: m.auditItems, Total: m.auditTotal}, nil
}

func TestHandleListRecent_Success(t *testing.T) {
	now := time.Now().UTC()
	items := []model.LogItem{
		{
			LogType:   "pipeline_step",
			Level:     "info",
			Message:   "build_metric_daily completed",
			RequestID: nil,
			CreatedAt: now,
		},
	}
	svc := &mockLogService{recentItems: items, recentTotal: 1}
	h := NewLogHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/logs/recent", nil)
	w := httptest.NewRecorder()
	h.HandleListRecent(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp dto.LogListResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, 1, resp.Total)
	assert.Equal(t, "pipeline_step", resp.Items[0].LogType)
	assert.True(t, svc.recentCalled)
}

func TestHandleListRecent_Empty(t *testing.T) {
	svc := &mockLogService{recentItems: []model.LogItem{}, recentTotal: 0}
	h := NewLogHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/logs/recent", nil)
	w := httptest.NewRecorder()
	h.HandleListRecent(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp dto.LogListResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Empty(t, resp.Items)
	assert.Equal(t, 0, resp.Total)
}

func TestHandleListRecent_BadPagination(t *testing.T) {
	svc := &mockLogService{}
	h := NewLogHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/logs/recent?limit=abc", nil)
	w := httptest.NewRecorder()
	h.HandleListRecent(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var body map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)
	assert.Contains(t, body["message"].(string), "invalid limit")
}

func TestHandleListRecent_Error(t *testing.T) {
	svc := &mockLogService{err: assert.AnError}
	h := NewLogHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/logs/recent", nil)
	w := httptest.NewRecorder()
	h.HandleListRecent(w, r)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandleListErrors_Success(t *testing.T) {
	now := time.Now().UTC()
	items := []model.LogItem{
		{
			LogType:   "error_log",
			Level:     "error",
			Message:   "connection refused",
			RequestID: strPtr("req-123"),
			CreatedAt: now,
		},
	}
	svc := &mockLogService{errorItems: items, errorTotal: 1}
	h := NewLogHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/logs/errors", nil)
	w := httptest.NewRecorder()
	h.HandleListErrors(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp dto.LogListResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, "error_log", resp.Items[0].LogType)
	assert.Equal(t, "connection refused", resp.Items[0].Message)
	require.NotNil(t, resp.Items[0].RequestID)
	assert.Equal(t, "req-123", *resp.Items[0].RequestID)
	assert.True(t, svc.errorCalled)
}

func TestHandleListErrors_Empty(t *testing.T) {
	svc := &mockLogService{errorItems: []model.LogItem{}, errorTotal: 0}
	h := NewLogHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/logs/errors", nil)
	w := httptest.NewRecorder()
	h.HandleListErrors(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp dto.LogListResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Empty(t, resp.Items)
	assert.Equal(t, 0, resp.Total)
}

func TestHandleListErrors_Error(t *testing.T) {
	svc := &mockLogService{err: assert.AnError}
	h := NewLogHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/logs/errors", nil)
	w := httptest.NewRecorder()
	h.HandleListErrors(w, r)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandleListAudit_Success(t *testing.T) {
	now := time.Now().UTC()
	items := []model.LogItem{
		{
			LogType:   "audit_log",
			Level:     "info",
			Message:   "dispatch on outbox",
			RequestID: nil,
			CreatedAt: now,
		},
	}
	svc := &mockLogService{auditItems: items, auditTotal: 5}
	h := NewLogHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/logs/audit", nil)
	w := httptest.NewRecorder()
	h.HandleListAudit(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp dto.LogListResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, 5, resp.Total)
	assert.True(t, svc.auditCalled)
}

func TestHandleListAudit_Empty(t *testing.T) {
	svc := &mockLogService{auditItems: []model.LogItem{}, auditTotal: 0}
	h := NewLogHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/logs/audit", nil)
	w := httptest.NewRecorder()
	h.HandleListAudit(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp dto.LogListResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Empty(t, resp.Items)
	assert.Equal(t, 0, resp.Total)
}

func TestHandleListAudit_Error(t *testing.T) {
	svc := &mockLogService{err: assert.AnError}
	h := NewLogHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/logs/audit", nil)
	w := httptest.NewRecorder()
	h.HandleListAudit(w, r)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandleListRecent_ResponseFormat(t *testing.T) {
	svc := &mockLogService{recentItems: []model.LogItem{}, recentTotal: 0}
	h := NewLogHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/logs/recent", nil)
	w := httptest.NewRecorder()
	h.HandleListRecent(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)

	_, hasItems := body["items"]
	_, hasTotal := body["total"]
	assert.True(t, hasItems, "response must have 'items' field")
	assert.True(t, hasTotal, "response must have 'total' field")
	_, hasPagination := body["pagination"]
	assert.False(t, hasPagination, "response must NOT have 'pagination' field (use old format)")
}

func TestHandleListErrors_ResponseFormat(t *testing.T) {
	svc := &mockLogService{errorItems: []model.LogItem{}, errorTotal: 0}
	h := NewLogHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/logs/errors", nil)
	w := httptest.NewRecorder()
	h.HandleListErrors(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)

	_, hasItems := body["items"]
	_, hasTotal := body["total"]
	assert.True(t, hasItems, "response must have 'items' field")
	assert.True(t, hasTotal, "response must have 'total' field")
}

func TestHandleListAudit_ResponseFormat(t *testing.T) {
	svc := &mockLogService{auditItems: []model.LogItem{}, auditTotal: 0}
	h := NewLogHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/logs/audit", nil)
	w := httptest.NewRecorder()
	h.HandleListAudit(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)

	_, hasItems := body["items"]
	_, hasTotal := body["total"]
	assert.True(t, hasItems, "response must have 'items' field")
	assert.True(t, hasTotal, "response must have 'total' field")
}
