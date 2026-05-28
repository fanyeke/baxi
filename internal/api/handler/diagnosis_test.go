package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/api/dto"
	"baxi/internal/model"
)

type mockDiagnoser struct {
	result *model.DiagnosisResponse
	err    error
	called bool
}

func (m *mockDiagnoser) DiagnoseByRequestID(requestID string) (*model.DiagnosisResponse, error) {
	m.called = true
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func TestHandleDiagnosis_Success(t *testing.T) {
	result := &model.DiagnosisResponse{
		RequestID:       "req-123",
		Summary:         "connection refused",
		ErrorCode:       "E001",
		Diagnosis:       "db down",
		SuggestedAction: "restart db",
		RelatedLogs: []model.LogEntry{
			{Source: "error.log", Ts: "2024-01-01T00:00:00Z", Message: "connection refused"},
			{Source: "audit_dispatch.csv", Timestamp: "2024-01-01T00:00:00Z", OutboxID: "out-1", Status: "failed"},
		},
	}
	svc := &mockDiagnoser{result: result}
	h := NewDiagnosisHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/logs/diagnosis?request_id=req-123", nil)
	w := httptest.NewRecorder()
	h.HandleDiagnosis(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, svc.called)

	var resp dto.DiagnosisResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, "req-123", resp.RequestID)
	assert.Equal(t, "connection refused", resp.Summary)
	assert.Equal(t, "E001", resp.ErrorCode)
	assert.Equal(t, "db down", resp.Diagnosis)
	assert.Equal(t, "restart db", resp.SuggestedAction)
	require.Len(t, resp.RelatedLogs, 2)
	assert.Equal(t, "error.log", resp.RelatedLogs[0].Source)
	assert.Equal(t, "audit_dispatch.csv", resp.RelatedLogs[1].Source)
}

func TestHandleDiagnosis_NotFound(t *testing.T) {
	svc := &mockDiagnoser{result: nil}
	h := NewDiagnosisHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/logs/diagnosis?request_id=req-456", nil)
	w := httptest.NewRecorder()
	h.HandleDiagnosis(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.True(t, svc.called)

	apiErr := decodeAPIError(t, w)
	assert.Equal(t, "NOT_FOUND", apiErr.ErrorCode)
	assert.Contains(t, apiErr.Message, "req-456")
	assert.NotEmpty(t, apiErr.Diagnosis)
	assert.NotEmpty(t, apiErr.SuggestedAction)
}

func TestHandleDiagnosis_MissingRequestID(t *testing.T) {
	svc := &mockDiagnoser{}
	h := NewDiagnosisHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/logs/diagnosis", nil)
	w := httptest.NewRecorder()
	h.HandleDiagnosis(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, svc.called)

	apiErr := decodeAPIError(t, w)
	assert.Equal(t, "BAD_REQUEST", apiErr.ErrorCode)
}

func TestHandleDiagnosis_ServiceError(t *testing.T) {
	svc := &mockDiagnoser{err: errors.New("boom")}
	h := NewDiagnosisHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/logs/diagnosis?request_id=req-123", nil)
	w := httptest.NewRecorder()
	h.HandleDiagnosis(w, r)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	assert.True(t, svc.called)

	apiErr := decodeAPIError(t, w)
	assert.Equal(t, "INTERNAL_ERROR", apiErr.ErrorCode)
}

func TestHandleDiagnosis_ResponseFormat(t *testing.T) {
	result := &model.DiagnosisResponse{
		RequestID:       "req-123",
		Summary:         "test summary",
		ErrorCode:       "E001",
		Diagnosis:       "test diagnosis",
		SuggestedAction: "test action",
		RelatedLogs:     []model.LogEntry{},
	}
	svc := &mockDiagnoser{result: result}
	h := NewDiagnosisHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/logs/diagnosis?request_id=req-123", nil)
	w := httptest.NewRecorder()
	h.HandleDiagnosis(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)

	_, hasRequestID := body["request_id"]
	_, hasSummary := body["summary"]
	_, hasErrorCode := body["error_code"]
	_, hasDiagnosis := body["diagnosis"]
	_, hasSuggestedAction := body["suggested_action"]
	_, hasRelatedLogs := body["related_logs"]

	assert.True(t, hasRequestID, "response must have 'request_id' field")
	assert.True(t, hasSummary, "response must have 'summary' field")
	assert.True(t, hasErrorCode, "response must have 'error_code' field")
	assert.True(t, hasDiagnosis, "response must have 'diagnosis' field")
	assert.True(t, hasSuggestedAction, "response must have 'suggested_action' field")
	assert.True(t, hasRelatedLogs, "response must have 'related_logs' field")
}

func TestHandleDiagnosis_EmptyRelatedLogs(t *testing.T) {
	result := &model.DiagnosisResponse{
		RequestID:       "req-123",
		Summary:         "test",
		RelatedLogs:     []model.LogEntry{},
	}
	svc := &mockDiagnoser{result: result}
	h := NewDiagnosisHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/logs/diagnosis?request_id=req-123", nil)
	w := httptest.NewRecorder()
	h.HandleDiagnosis(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp dto.DiagnosisResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	assert.NotNil(t, resp.RelatedLogs)
	assert.Empty(t, resp.RelatedLogs)
}
