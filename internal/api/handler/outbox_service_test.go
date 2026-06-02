package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"time"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"

	"baxi/internal/model"
)

// mockOutboxService implements OutboxService for testing.
type mockOutboxService struct {
	listFn           func(ctx context.Context, filters model.OutboxFilters, limit, offset int) (*model.OutboxListResponse, error)
	getEventFn       func(ctx context.Context, id string) (*OutboxDetailItem, error)
	dispatchFn       func(ctx context.Context, id string) error
	cancelFn         func(ctx context.Context, id string) error
	batchDispatchFn  func(ctx context.Context, dryRun bool, channel string, limit int) (*BatchDispatchResponse, error)
}

func (m *mockOutboxService) List(ctx context.Context, filters model.OutboxFilters, limit, offset int) (*model.OutboxListResponse, error) {
	return m.listFn(ctx, filters, limit, offset)
}
func (m *mockOutboxService) GetEvent(ctx context.Context, id string) (*OutboxDetailItem, error) {
	return m.getEventFn(ctx, id)
}
func (m *mockOutboxService) DispatchEvent(ctx context.Context, id string) error {
	return m.dispatchFn(ctx, id)
}
func (m *mockOutboxService) CancelEvent(ctx context.Context, id string) error {
	return m.cancelFn(ctx, id)
}
func (m *mockOutboxService) BatchDispatch(ctx context.Context, dryRun bool, channel string, limit int) (*BatchDispatchResponse, error) {
	return m.batchDispatchFn(ctx, dryRun, channel, limit)
}

func TestNewOutboxHandler_NonNil(t *testing.T) {
	h := NewOutboxHandler(&mockOutboxService{})
	assert.NotNil(t, h)
}

func TestHandleListOutbox_Success(t *testing.T) {
	h := NewOutboxHandler(&mockOutboxService{
		listFn: func(_ context.Context, _ model.OutboxFilters, limit, offset int) (*model.OutboxListResponse, error) {
			return &model.OutboxListResponse{
				Items: []model.OutboxEvent{{OutboxID: "evt-001", EventType: "test", Status: "pending"}},
				Total: 1,
			}, nil
		},
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/outbox?limit=10&offset=0", nil)

	h.HandleListOutbox(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	decodeJSON(t, resp, &body)
	assert.Equal(t, float64(1), body["total"])
	items := body["items"].([]interface{})
	assert.Len(t, items, 1)
}

func TestHandleListOutbox_InvalidPagination(t *testing.T) {
	h := NewOutboxHandler(&mockOutboxService{
		listFn: func(_ context.Context, _ model.OutboxFilters, limit, offset int) (*model.OutboxListResponse, error) {
			// ParsePagination clamps -5 to limit=1, offset=0, so this should succeed
			return &model.OutboxListResponse{
				Items: []model.OutboxEvent{},
				Total: 0,
			}, nil
		},
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/outbox?limit=-5", nil)
	h.HandleListOutbox(w, r)
	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestHandleListOutbox_ServiceError(t *testing.T) {
	h := NewOutboxHandler(&mockOutboxService{
		listFn: func(_ context.Context, _ model.OutboxFilters, limit, offset int) (*model.OutboxListResponse, error) {
			return nil, errors.New("db error")
		},
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/outbox?limit=10&offset=0", nil)

	h.HandleListOutbox(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestHandleDispatch_Success(t *testing.T) {
	h := NewOutboxHandler(&mockOutboxService{
		dispatchFn: func(_ context.Context, id string) error {
			return nil
		},
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/outbox/evt-001/dispatch", nil)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("id", "evt-001")

	h.HandleDispatch(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestHandleDispatch_NotFound(t *testing.T) {
	h := NewOutboxHandler(&mockOutboxService{
		dispatchFn: func(_ context.Context, id string) error {
			return ErrEventNotFound{}
		},
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/outbox/evt-999/dispatch", nil)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("id", "evt-999")

	h.HandleDispatch(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestHandleDispatch_InvalidState(t *testing.T) {
	h := NewOutboxHandler(&mockOutboxService{
		dispatchFn: func(_ context.Context, id string) error {
			return ErrInvalidState{Status: "dispatched"}
		},
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/outbox/evt-001/dispatch", nil)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("id", "evt-001")

	h.HandleDispatch(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestHandleDispatch_GenericError(t *testing.T) {
	h := NewOutboxHandler(&mockOutboxService{
		dispatchFn: func(_ context.Context, id string) error {
			return errors.New("unexpected error")
		},
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/outbox/evt-001/dispatch", nil)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("id", "evt-001")

	h.HandleDispatch(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestHandleCancel_Success(t *testing.T) {
	h := NewOutboxHandler(&mockOutboxService{
		getEventFn: func(_ context.Context, id string) (*OutboxDetailItem, error) {
			return &OutboxDetailItem{EventID: id, Status: "pending"}, nil
		},
		cancelFn: func(_ context.Context, id string) error {
			return nil
		},
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/outbox/evt-001/cancel", nil)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("id", "evt-001")

	h.HandleCancel(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestHandleCancel_NotFound(t *testing.T) {
	h := NewOutboxHandler(&mockOutboxService{
		getEventFn: func(_ context.Context, id string) (*OutboxDetailItem, error) {
			return nil, ErrEventNotFound{}
		},
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/outbox/evt-999/cancel", nil)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("id", "evt-999")

	h.HandleCancel(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestHandleGetDetail_Success(t *testing.T) {
	now := time.Now()
	h := NewOutboxHandler(&mockOutboxService{
		getEventFn: func(_ context.Context, id string) (*OutboxDetailItem, error) {
			return &OutboxDetailItem{
				EventID:   id,
				EventType: "pipeline.complete",
				Status:    "pending",
				CreatedAt: now,
			}, nil
		},
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/outbox/evt-001", nil)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("id", "evt-001")

	h.HandleGetDetail(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	decodeJSON(t, resp, &body)
	assert.Equal(t, "evt-001", body["event_id"])
}

func TestHandleGetDetail_NotFound(t *testing.T) {
	h := NewOutboxHandler(&mockOutboxService{
		getEventFn: func(_ context.Context, id string) (*OutboxDetailItem, error) {
			return nil, ErrEventNotFound{}
		},
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/outbox/evt-999", nil)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("id", "evt-999")

	h.HandleGetDetail(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
