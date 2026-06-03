package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/action"
	"baxi/internal/model"
	"baxi/internal/outbox"
	"outboxRepo outboxRepo "baxi/internal/repository/outbox""
	"baxi/internal/service"
)

const hlrOutboxTableDDL = `
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

func setupHlrOutboxTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	ctx := context.Background()
	_, err = pool.Exec(ctx, hlrOutboxTableDDL)
	require.NoError(t, err)
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE ops.outbox_event CASCADE")
	return pool
}

func insertHlrTestEvent(t *testing.T, pool *pgxpool.Pool, id, eventType, sourceType, sourceID, channel, status string, attempts int, lastDispatch *time.Time, createdAt time.Time) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO ops.outbox_event (event_id, event_type, source_type, source_id, payload_json, target_channel, status, dispatch_attempts, last_dispatch_at, created_at)
		VALUES ($1, $2, $3, $4, '{}', $5, $6, $7, $8, $9)
	`, id, eventType, sourceType, sourceID, channel, status, attempts, lastDispatch, createdAt)
	require.NoError(t, err)
}

func newReqWithContext(ctx context.Context, method, url string) *http.Request {
	r := httptest.NewRequest(method, url, nil).WithContext(ctx)
	rctx := chi.NewRouteContext()
	r = r.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))
	return r
}

func newReqWithParam(ctx context.Context, method, url, param, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(param, value)
	r := httptest.NewRequest(method, url, nil).WithContext(ctx)
	r = r.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))
	return r
}

func newOutboxHandlerForTest(pool *pgxpool.Pool) *OutboxHandler {
	readRepo := outboxRepo.NewRepository(nil)
	writeRepo := outbox.NewOutboxRepository()
	svc := service.NewOutboxService(readRepo, pool)
	executors := map[string]action.ActionExecutor{
		"noop": action.NewNoOpExecutor(),
	}
	adapter := &testOutboxAdapter{
		readSvc:   svc,
		readRepo:  readRepo,
		writeRepo: writeRepo,
		pool:      pool,
		executors: executors,
	}
	return NewOutboxHandler(adapter)
}

type testOutboxAdapter struct {
	readSvc   *service.OutboxService
	readRepo  *repository.OutboxRepository
	writeRepo *outbox.OutboxRepository
	pool      *pgxpool.Pool
	executors map[string]action.ActionExecutor
}

func (a *testOutboxAdapter) List(ctx context.Context, filters model.OutboxFilters, limit, offset int) (*model.OutboxListResponse, error) {
	return a.readSvc.List(ctx, filters, limit, offset)
}

func (a *testOutboxAdapter) GetEvent(ctx context.Context, id string) (*OutboxDetailItem, error) {
	detail, err := a.readRepo.GetDetail(ctx, a.pool, id)
	if err != nil || detail == nil {
		return nil, err
	}
	item := &OutboxDetailItem{
		EventID:          detail.EventID,
		EventType:        detail.EventType,
		SourceType:       detail.SourceType,
		SourceID:         detail.SourceID,
		TargetChannel:    detail.TargetChannel,
		Status:           detail.Status,
		CreatedAt:        detail.CreatedAt,
		DispatchAttempts: detail.DispatchAttempts,
		LastDispatchAt:   detail.LastDispatchAt,
		ErrorMessage:     detail.ErrorMessage,
	}
	if detail.Payload != nil {
		item.Payload = string(detail.Payload)
	}
	return item, nil
}

func (a *testOutboxAdapter) DispatchEvent(ctx context.Context, id string) error {
	detail, err := a.readRepo.GetDetail(ctx, a.pool, id)
	if err != nil {
		return err
	}
	if detail == nil {
		return ErrEventNotFound{}
	}
	if detail.Status != "pending" && detail.Status != "failed" {
		return ErrInvalidState{Status: detail.Status}
	}
	executor := a.executors["noop"]
	result, err := executor.Execute(ctx, action.ActionProposal{ProposalID: id, ActionType: detail.EventType, CaseID: detail.SourceID}, true)
	if err != nil || !result.Success {
		return fmt.Errorf("dispatch failed")
	}
	tx, txErr := a.pool.Begin(ctx)
	if txErr != nil {
		return txErr
	}
	defer tx.Rollback(ctx)
	if err := a.writeRepo.MarkDispatched(ctx, tx, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (a *testOutboxAdapter) CancelEvent(ctx context.Context, id string) error {
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := a.writeRepo.MarkCancelled(ctx, tx, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func TestHandler_ListOutbox_NoFilters(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	pool := setupHlrOutboxTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	insertHlrTestEvent(t, pool, "evt-1", "alert", "rule_engine", "rule-1", "feishu", "pending", 0, nil, now.Add(-2*time.Hour))
	insertHlrTestEvent(t, pool, "evt-2", "task", "scheduler", "task-1", "email", "dispatched", 1, &now, now.Add(-1*time.Hour))
	handler := newOutboxHandlerForTest(pool)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/outbox", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	handler.HandleListOutbox(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	items, ok := resp["items"].([]interface{})
	require.True(t, ok)
	assert.Len(t, items, 2)
	total, ok := resp["total"].(float64)
	require.True(t, ok)
	assert.Equal(t, float64(2), total)
}

func TestHandler_ListOutbox_WithFilters(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	pool := setupHlrOutboxTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	insertHlrTestEvent(t, pool, "evt-1", "alert", "rule_engine", "rule-1", "feishu", "pending", 0, nil, now)
	insertHlrTestEvent(t, pool, "evt-2", "alert", "rule_engine", "rule-2", "email", "dispatched", 1, &now, now)
	handler := newOutboxHandlerForTest(pool)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/outbox?status=pending", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	handler.HandleListOutbox(w, req)
	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	items, ok := resp["items"].([]interface{})
	require.True(t, ok)
	assert.Len(t, items, 1)
	total, ok := resp["total"].(float64)
	require.True(t, ok)
	assert.Equal(t, float64(1), total)
}

func TestHandler_ListOutbox_WithPagination(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	pool := setupHlrOutboxTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	insertHlrTestEvent(t, pool, "evt-1", "alert", "rule", "rule-1", "feishu", "pending", 0, nil, now.Add(-3*time.Hour))
	insertHlrTestEvent(t, pool, "evt-2", "alert", "rule", "rule-2", "feishu", "pending", 0, nil, now.Add(-2*time.Hour))
	insertHlrTestEvent(t, pool, "evt-3", "alert", "rule", "rule-3", "feishu", "pending", 0, nil, now.Add(-1*time.Hour))
	handler := newOutboxHandlerForTest(pool)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/outbox?limit=2", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	handler.HandleListOutbox(w, req)
	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	items, ok := resp["items"].([]interface{})
	require.True(t, ok)
	assert.Len(t, items, 2)
	total, ok := resp["total"].(float64)
	require.True(t, ok)
	assert.Equal(t, float64(3), total)
}

func TestHandler_ListOutbox_EmptyResponse(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	pool := setupHlrOutboxTestDB(t)
	ctx := context.Background()
	handler := newOutboxHandlerForTest(pool)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/outbox", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	handler.HandleListOutbox(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	items, ok := resp["items"].([]interface{})
	require.True(t, ok)
	assert.Empty(t, items)
	total, ok := resp["total"].(float64)
	require.True(t, ok)
	assert.Equal(t, float64(0), total)
}

func TestHandler_ListOutbox_MultipleFilters(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	pool := setupHlrOutboxTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	insertHlrTestEvent(t, pool, "evt-1", "alert", "rule_engine", "rule-1", "feishu", "pending", 0, nil, now)
	insertHlrTestEvent(t, pool, "evt-2", "alert", "rule_engine", "rule-2", "feishu", "dispatched", 1, &now, now)
	insertHlrTestEvent(t, pool, "evt-3", "task", "scheduler", "task-1", "email", "pending", 0, nil, now)
	handler := newOutboxHandlerForTest(pool)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/outbox?status=pending&channel=feishu", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	handler.HandleListOutbox(w, req)
	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	items, ok := resp["items"].([]interface{})
	require.True(t, ok)
	assert.Len(t, items, 1)
	firstItem, ok := items[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "evt-1", firstItem["outbox_id"])
	assert.Equal(t, "alert", firstItem["event_type"])
	assert.Equal(t, "rule_engine", firstItem["source_type"])
	assert.Equal(t, "rule-1", firstItem["source_id"])
	assert.Equal(t, "feishu", firstItem["target_channel"])
	assert.Equal(t, "pending", firstItem["status"])
	assert.Equal(t, float64(0), firstItem["dispatch_attempts"])
}

func TestHandler_ListOutbox_InvalidLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	pool := setupHlrOutboxTestDB(t)
	ctx := context.Background()
	handler := newOutboxHandlerForTest(pool)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/outbox?limit=abc", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	handler.HandleListOutbox(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ListOutbox_AllQueryParams(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	pool := setupHlrOutboxTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	insertHlrTestEvent(t, pool, "evt-1", "alert", "rule_engine", "rule-1", "feishu", "pending", 0, nil, now.Add(-2*time.Hour))
	insertHlrTestEvent(t, pool, "evt-2", "alert", "rule_engine", "rule-2", "feishu", "pending", 0, nil, now.Add(-1*time.Hour))
	handler := newOutboxHandlerForTest(pool)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/outbox?status=pending&channel=feishu&event_type=alert&limit=1&offset=1", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	handler.HandleListOutbox(w, req)
	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	items, ok := resp["items"].([]interface{})
	require.True(t, ok)
	assert.Len(t, items, 1)
	total, ok := resp["total"].(float64)
	require.True(t, ok)
	assert.Equal(t, float64(2), total)
	firstItem, ok := items[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "evt-1", firstItem["outbox_id"])
}

func TestHandler_ListOutbox_ItemJSONStructure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	pool := setupHlrOutboxTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	insertHlrTestEvent(t, pool, "evt-1", "dimensional_alert", "dimensional_rule_engine", "dim-76085bfcd31d", "feishu_cli", "pending", 2, &now, now)
	handler := newOutboxHandlerForTest(pool)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/outbox", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	handler.HandleListOutbox(w, req)
	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	items, ok := resp["items"].([]interface{})
	require.True(t, ok)
	require.Len(t, items, 1)
	item, ok := items[0].(map[string]interface{})
	require.True(t, ok)
	expectedKeys := []string{"outbox_id", "event_type", "source_type", "source_id", "target_channel", "status", "created_at", "dispatch_attempts", "last_dispatch_at"}
	for _, key := range expectedKeys {
		_, exists := item[key]
		assert.True(t, exists, "item should have key: %s", key)
	}
	assert.Equal(t, "evt-1", item["outbox_id"])
	assert.Equal(t, "dimensional_alert", item["event_type"])
	assert.Equal(t, "dimensional_rule_engine", item["source_type"])
	assert.Equal(t, "dim-76085bfcd31d", item["source_id"])
	assert.Equal(t, "feishu_cli", item["target_channel"])
	assert.Equal(t, "pending", item["status"])
	assert.Equal(t, float64(2), item["dispatch_attempts"])
	assert.NotNil(t, item["last_dispatch_at"])
	assert.NotNil(t, item["created_at"])
}

func TestParseOutboxFilters(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/outbox?status=pending&channel=feishu&event_type=alert", nil)
	filters := parseOutboxFilters(r)
	require.NotNil(t, filters.Status)
	assert.Equal(t, "pending", *filters.Status)
	require.NotNil(t, filters.Channel)
	assert.Equal(t, "feishu", *filters.Channel)
	require.NotNil(t, filters.EventType)
	assert.Equal(t, "alert", *filters.EventType)
}

func TestParseOutboxFilters_Empty(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/outbox", nil)
	filters := parseOutboxFilters(r)
	assert.Nil(t, filters.Status)
	assert.Nil(t, filters.Channel)
	assert.Nil(t, filters.EventType)
}

func TestParseOutboxFilters_Partial(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/outbox?channel=email", nil)
	filters := parseOutboxFilters(r)
	assert.Nil(t, filters.Status)
	require.NotNil(t, filters.Channel)
	assert.Equal(t, "email", *filters.Channel)
	assert.Nil(t, filters.EventType)
}

func TestHandler_ListOutbox_ResponseFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	pool := setupHlrOutboxTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	insertHlrTestEvent(t, pool, "test-id", "test_event", "test_source", "test-src", "test_channel", "pending", 3, &now, now)
	handler := newOutboxHandlerForTest(pool)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/outbox", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	handler.HandleListOutbox(w, req)
	body := w.Body.String()
	assert.True(t, strings.Contains(body, `"items"`), "response should contain items array")
	assert.True(t, strings.Contains(body, `"total"`), "response should contain total field")
	assert.False(t, strings.Contains(body, `"pagination"`), "response should NOT contain pagination object (backward compat)")
}

func TestHandler_GetDetail_Found(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	pool := setupHlrOutboxTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	insertHlrTestEvent(t, pool, "detail-1", "alert", "rule_engine", "rule-1", "feishu", "pending", 0, nil, now)
	h := newOutboxHandlerForTest(pool)
	req := newReqWithParam(ctx, http.MethodGet, "/api/v1/outbox/detail-1", "id", "detail-1")
	w := httptest.NewRecorder()
	h.HandleGetDetail(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "detail-1", resp["event_id"])
	assert.Equal(t, "alert", resp["event_type"])
	assert.Equal(t, "pending", resp["status"])
	assert.Equal(t, "feishu", resp["target_channel"])
}

func TestHandler_GetDetail_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	pool := setupHlrOutboxTestDB(t)
	ctx := context.Background()
	h := newOutboxHandlerForTest(pool)
	req := newReqWithParam(ctx, http.MethodGet, "/api/v1/outbox/nonexistent", "id", "nonexistent")
	w := httptest.NewRecorder()
	h.HandleGetDetail(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_Dispatch_Pending(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	pool := setupHlrOutboxTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	insertHlrTestEvent(t, pool, "disp-1", "alert", "rule_engine", "rule-1", "noop", "pending", 0, nil, now)
	h := newOutboxHandlerForTest(pool)
	req := newReqWithParam(ctx, http.MethodPost, "/api/v1/outbox/disp-1/dispatch", "id", "disp-1")
	w := httptest.NewRecorder()
	h.HandleDispatch(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "disp-1", resp["event_id"])
	assert.Equal(t, "dispatched", resp["status"])
	var row struct {
		Status string
	}
	err = pool.QueryRow(ctx, "SELECT status FROM ops.outbox_event WHERE event_id = $1", "disp-1").Scan(&row.Status)
	require.NoError(t, err)
	assert.Equal(t, "dispatched", row.Status)
}

func TestHandler_Dispatch_AlreadyDispatched(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	pool := setupHlrOutboxTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	insertHlrTestEvent(t, pool, "disp-2", "alert", "rule_engine", "rule-2", "noop", "dispatched", 1, &now, now)
	h := newOutboxHandlerForTest(pool)
	req := newReqWithParam(ctx, http.MethodPost, "/api/v1/outbox/disp-2/dispatch", "id", "disp-2")
	w := httptest.NewRecorder()
	h.HandleDispatch(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestHandler_Dispatch_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	pool := setupHlrOutboxTestDB(t)
	ctx := context.Background()
	h := newOutboxHandlerForTest(pool)
	req := newReqWithParam(ctx, http.MethodPost, "/api/v1/outbox/nonexistent/dispatch", "id", "nonexistent")
	w := httptest.NewRecorder()
	h.HandleDispatch(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_Cancel_Pending(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	pool := setupHlrOutboxTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	insertHlrTestEvent(t, pool, "cancel-1", "alert", "rule_engine", "rule-1", "noop", "pending", 0, nil, now)
	h := newOutboxHandlerForTest(pool)
	req := newReqWithParam(ctx, http.MethodPost, "/api/v1/outbox/cancel-1/cancel", "id", "cancel-1")
	w := httptest.NewRecorder()
	h.HandleCancel(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "cancel-1", resp["event_id"])
	assert.Equal(t, "cancelled", resp["status"])
	var status string
	err = pool.QueryRow(ctx, "SELECT status FROM ops.outbox_event WHERE event_id = $1", "cancel-1").Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, "cancelled", status)
}

func TestHandler_Cancel_Dispatched(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	pool := setupHlrOutboxTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	insertHlrTestEvent(t, pool, "cancel-2", "alert", "rule_engine", "rule-2", "noop", "dispatched", 1, &now, now)
	h := newOutboxHandlerForTest(pool)
	req := newReqWithParam(ctx, http.MethodPost, "/api/v1/outbox/cancel-2/cancel", "id", "cancel-2")
	w := httptest.NewRecorder()
	h.HandleCancel(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestHandler_Cancel_Failed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	pool := setupHlrOutboxTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	insertHlrTestEvent(t, pool, "cancel-3", "alert", "rule_engine", "rule-3", "noop", "failed", 2, &now, now)
	h := newOutboxHandlerForTest(pool)
	req := newReqWithParam(ctx, http.MethodPost, "/api/v1/outbox/cancel-3/cancel", "id", "cancel-3")
	w := httptest.NewRecorder()
	h.HandleCancel(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var status string
	err := pool.QueryRow(ctx, "SELECT status FROM ops.outbox_event WHERE event_id = $1", "cancel-3").Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, "cancelled", status)
}

func TestHandler_GetDetail_ReturnsAllFields(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	pool := setupHlrOutboxTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	lastDispatch := now.Add(-10 * time.Minute)
	payload := `{"key":"value"}`
	_, err := pool.Exec(ctx, `
		INSERT INTO ops.outbox_event (event_id, event_type, source_type, source_id, payload_json, target_channel, status, dispatch_attempts, last_dispatch_at, error_message, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, "detail-full", "task", "scheduler", "task-100", payload, "feishu", "failed", 3, lastDispatch, "timeout occurred", now)
	require.NoError(t, err)
	h := newOutboxHandlerForTest(pool)
	req := newReqWithParam(ctx, http.MethodGet, "/api/v1/outbox/detail-full", "id", "detail-full")
	w := httptest.NewRecorder()
	h.HandleGetDetail(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	err = json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	expectedKeys := []string{"event_id", "event_type", "source_type", "source_id", "target_channel", "status", "payload_json", "dispatch_attempts", "last_dispatch_at", "last_error"}
	for _, key := range expectedKeys {
		_, exists := resp[key]
		assert.True(t, exists, "response should have key: %s", key)
	}
	assert.Equal(t, "detail-full", resp["event_id"])
	assert.Equal(t, "failed", resp["status"])
	assert.Equal(t, float64(3), resp["dispatch_attempts"])
	assert.Equal(t, "timeout occurred", resp["last_error"])
}
