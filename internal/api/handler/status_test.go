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
)

type mockStatusService struct {
	resp  *dto.StatusResponse
	err   error
	called bool
}

func (m *mockStatusService) GetStatus(_ context.Context) (*dto.StatusResponse, error) {
	m.called = true
	if m.err != nil {
		return nil, m.err
	}
	return m.resp, nil
}

func TestHandleStatus_Success(t *testing.T) {
	tables := map[string]int{
		"alert_events":          36,
		"action_tasks":          36,
		"dwd_order_level":       99441,
		"dwd_item_level":        112650,
		"event_outbox":          36,
		"metric_daily":          634,
		"metric_dimension_daily": 690326,
	}
	finishedAt := "2026-05-24 03:48:59.449642"
	resp := &dto.StatusResponse{
		Database: dto.DatabaseInfo{
			Path:   "postgresql://localhost:5432/baxi",
			Exists: true,
			Tables: tables,
		},
		LastPipelineRun: &dto.PipelineRun{
			RunID:       "ingest-full-2026-05-24T03:48:59",
			RunType:     "ingestion",
			Mode:        "full",
			Status:      "completed",
			StartedAt:   "2026-05-24T03:48:59.449458",
			FinishedAt:  &finishedAt,
			InputCount:  0,
			OutputCount: 0,
			ErrorMessage: nil,
		},
		Version: "0.6.0",
	}
	svc := &mockStatusService{resp: resp}
	h := NewStatusHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	w := httptest.NewRecorder()
	h.HandleStatus(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, svc.called)

	var got dto.StatusResponse
	err := json.NewDecoder(w.Body).Decode(&got)
	require.NoError(t, err)

	assert.Equal(t, "postgresql://localhost:5432/baxi", got.Database.Path)
	assert.True(t, got.Database.Exists)
	assert.Equal(t, 36, got.Database.Tables["alert_events"])
	assert.Equal(t, 99441, got.Database.Tables["dwd_order_level"])
	assert.Equal(t, "ingest-full-2026-05-24T03:48:59", got.LastPipelineRun.RunID)
	assert.Equal(t, "completed", got.LastPipelineRun.Status)
	assert.Equal(t, "0.6.0", got.Version)
}

func TestHandleStatus_NoPipelineRun(t *testing.T) {
	resp := &dto.StatusResponse{
		Database: dto.DatabaseInfo{
			Path:   "postgresql://localhost:5432/baxi",
			Exists: true,
			Tables: map[string]int{"alert_events": 36},
		},
		LastPipelineRun: nil,
		Version:         "0.6.0",
	}
	svc := &mockStatusService{resp: resp}
	h := NewStatusHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	w := httptest.NewRecorder()
	h.HandleStatus(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)

	assert.Nil(t, body["last_pipeline_run"], "last_pipeline_run should be null when no runs exist")
}

func TestHandleStatus_ResponseFormat(t *testing.T) {
	resp := &dto.StatusResponse{
		Database: dto.DatabaseInfo{
			Path:   "postgresql://localhost:5432/baxi",
			Exists: true,
			Tables: map[string]int{},
		},
		LastPipelineRun: nil,
		Version:         "0.6.0",
	}
	svc := &mockStatusService{resp: resp}
	h := NewStatusHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	w := httptest.NewRecorder()
	h.HandleStatus(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)

	_, hasDatabase := body["database"]
	assert.True(t, hasDatabase, "response must have 'database' field")

	_, hasVersion := body["version"]
	assert.True(t, hasVersion, "response must have 'version' field")

	database, ok := body["database"].(map[string]interface{})
	require.True(t, ok, "database must be an object")
	_, hasPath := database["path"]
	assert.True(t, hasPath, "database must have 'path' field")
	_, hasExists := database["exists"]
	assert.True(t, hasExists, "database must have 'exists' field")
	_, hasTables := database["tables"]
	assert.True(t, hasTables, "database must have 'tables' field")
}

func TestHandleStatus_Error(t *testing.T) {
	svc := &mockStatusService{err: assert.AnError}
	h := NewStatusHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	w := httptest.NewRecorder()
	h.HandleStatus(w, r)

	require.Equal(t, http.StatusInternalServerError, w.Code)

	var body map[string]string
	err := json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)
	assert.Contains(t, body["error"], "internal server error")
}

func TestHandleStatus_TableCounts(t *testing.T) {
	tables := map[string]int{
		"alert_events":          36,
		"action_tasks":          36,
		"event_outbox":          36,
		"dwd_order_level":       99441,
		"dwd_item_level":        112650,
		"metric_daily":          634,
		"metric_dimension_daily": 690326,
	}
	resp := &dto.StatusResponse{
		Database: dto.DatabaseInfo{
			Path:   "postgresql://localhost:5432/baxi",
			Exists: true,
			Tables: tables,
		},
		Version: "0.6.0",
	}
	svc := &mockStatusService{resp: resp}
	h := NewStatusHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	w := httptest.NewRecorder()
	h.HandleStatus(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var got dto.StatusResponse
	err := json.NewDecoder(w.Body).Decode(&got)
	require.NoError(t, err)

	assert.Equal(t, 7, len(got.Database.Tables))
	for tableName, expectedCount := range tables {
		assert.Equal(t, expectedCount, got.Database.Tables[tableName], "table %q count mismatch", tableName)
	}
}
