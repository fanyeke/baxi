package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/api/dto"
	"baxi/internal/model"
)

type mockAlertService struct {
	items  []model.Alert
	total  int
	err    error
	called bool
}

func (m *mockAlertService) ListAlerts(_ context.Context, _ model.AlertFilters, _ string, _, _ int) (*model.AlertListResponse, error) {
	m.called = true
	if m.err != nil {
		return nil, m.err
	}
	return &model.AlertListResponse{Items: m.items, Total: m.total}, nil
}

func TestHandleListAlerts_NoFilters(t *testing.T) {
	items := []model.Alert{
		{
			EventID:   "gmv_drop_2018-10-17",
			RuleID:    "gmv_drop",
			EventDate: "2018-10-17",
			Severity:  "high",
			Status:    "new",
		},
	}
	svc := &mockAlertService{items: items, total: 36}
	h := NewAlertHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/alerts", nil)
	w := httptest.NewRecorder()
	h.HandleListAlerts(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp dto.AlertListResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, 36, resp.Total)
	assert.Equal(t, "gmv_drop_2018-10-17", resp.Items[0].EventID)
	assert.True(t, svc.called)
}

func TestHandleListAlerts_WithFilters(t *testing.T) {
	svc := &mockAlertService{items: []model.Alert{}, total: 0}
	h := NewAlertHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/alerts?severity=high&status=new&object_type=global&rule_id=gmv_drop&sort=created_at_desc&limit=10&offset=0", nil)
	w := httptest.NewRecorder()
	h.HandleListAlerts(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, svc.called)
}

func TestHandleListAlerts_EmptyResponse(t *testing.T) {
	svc := &mockAlertService{items: []model.Alert{}, total: 0}
	h := NewAlertHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/alerts", nil)
	w := httptest.NewRecorder()
	h.HandleListAlerts(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp dto.AlertListResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Empty(t, resp.Items)
	assert.Equal(t, 0, resp.Total)
}

func TestHandleListAlerts_BadPagination(t *testing.T) {
	svc := &mockAlertService{}
	h := NewAlertHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/alerts?limit=abc", nil)
	w := httptest.NewRecorder()
	h.HandleListAlerts(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var body map[string]string
	err := json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)
	assert.Contains(t, body["error"], "invalid limit")
}

func TestHandleListAlerts_ResponseFormat(t *testing.T) {
	svc := &mockAlertService{items: []model.Alert{}, total: 0}
	h := NewAlertHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/alerts", nil)
	w := httptest.NewRecorder()
	h.HandleListAlerts(w, r)

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

func TestHandleListAlerts_NullableFields(t *testing.T) {
	items := []model.Alert{
		{
			EventID:       "dim-test",
			RuleID:        "category_gmv_drop",
			EventDate:     "2018-08-20",
			Severity:      "medium",
			MetricName:    "gmv",
			ObjectType:    "category",
			ObjectID:      "health_beauty",
			CurrentValue:  ptr(4035.84),
			BaselineValue: ptr(5166.6129),
			ChangeRate:    ptr(-0.2189),
			OwnerRole:     "category_ops",
			Status:        "new",
			ImpactScore:   ptr(62.0),
		},
		{
			EventID:      "dim-null-impact",
			RuleID:       "region_cancel_rate_spike",
			EventDate:    "2018-08-20",
			Severity:     "high",
			MetricName:   "cancel_rate",
			ObjectType:   "region",
			ObjectID:     "RJ",
			CurrentValue: ptr(0.0769),
			ChangeRate:   ptr(0.0),
			OwnerRole:    "logistics_ops",
			Status:       "new",
			ImpactScore:  nil,
		},
	}
	svc := &mockAlertService{items: items, total: 2}
	h := NewAlertHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/alerts", nil)
	w := httptest.NewRecorder()
	h.HandleListAlerts(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp dto.AlertListResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	require.Len(t, resp.Items, 2)

	assert.NotNil(t, resp.Items[0].ImpactScore)
	assert.Equal(t, 62.0, *resp.Items[0].ImpactScore)
	assert.NotNil(t, resp.Items[0].BaselineValue)
	assert.Equal(t, 5166.6129, *resp.Items[0].BaselineValue)

	assert.Nil(t, resp.Items[1].ImpactScore)
	assert.Nil(t, resp.Items[1].BaselineValue)
}

func ptr(f float64) *float64 {
	return &f
}

func TestHandleListAlerts_Error(t *testing.T) {
	svc := &mockAlertService{err: assert.AnError}
	h := NewAlertHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/alerts", nil)
	w := httptest.NewRecorder()
	h.HandleListAlerts(w, r)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}
