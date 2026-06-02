package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/api/dto"
)

func TestCapabilitiesEndpoint_ResponseFormat(t *testing.T) {
	t.Setenv("API_BEARER_TOKEN", testBearerToken)
	s := newTestServer(t, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/qoder/capabilities", nil)
	r.Header.Set("Authorization", "Bearer "+testBearerToken)
	s.router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp dto.CapabilitiesResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, "read_only", resp.Mode)
	assert.Equal(t, "0.6.0", resp.Version)
	assert.True(t, resp.CanReadStatus)
	assert.True(t, resp.CanReadAlerts)
	assert.True(t, resp.CanReadTasks)
	assert.True(t, resp.CanReadOutbox)
	assert.True(t, resp.CanReadGovernance)
	assert.True(t, resp.CanReadLogs)
	assert.False(t, resp.CanWriteReports)
	assert.False(t, resp.CanExecuteActions)
}

func TestCapabilitiesEndpoint_ObjectShape(t *testing.T) {
	t.Setenv("API_BEARER_TOKEN", testBearerToken)
	s := newTestServer(t, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/qoder/capabilities", nil)
	r.Header.Set("Authorization", "Bearer "+testBearerToken)
	s.router.ServeHTTP(w, r)

	var body map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)

	// Must use object-shaped format (NOT array)
	_, isArray := body["items"]
	assert.False(t, isArray, "response must NOT be an array-shaped format")

	// Must have all expected fields at top level
	assert.Equal(t, "read_only", body["mode"])
	assert.Equal(t, "0.6.0", body["version"])
	assert.Equal(t, true, body["can_read_status"])
	assert.Equal(t, true, body["can_read_alerts"])
	assert.Equal(t, true, body["can_read_tasks"])
	assert.Equal(t, true, body["can_read_outbox"])
	assert.Equal(t, true, body["can_read_governance"])
	assert.Equal(t, true, body["can_read_logs"])
	assert.Equal(t, false, body["can_write_reports"])
	assert.Equal(t, false, body["can_execute_actions"])
}

func TestCapabilitiesEndpoint_NoAuthRequired(t *testing.T) {
	t.Setenv("API_BEARER_TOKEN", testBearerToken)
	s := newTestServer(t, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/qoder/capabilities", nil)
	r.Header.Set("Authorization", "Bearer "+testBearerToken)
	s.router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code, "capabilities endpoint should be accessible with valid auth")
}

func TestCapabilitiesEndpoint_NoDBRequired(t *testing.T) {
	t.Setenv("API_BEARER_TOKEN", testBearerToken)
	s := newTestServer(t, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/qoder/capabilities", nil)
	r.Header.Set("Authorization", "Bearer "+testBearerToken)
	s.router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code, "capabilities endpoint must work without DB connection")
}

func TestCapabilitiesEndpoint_RejectsNonGET(t *testing.T) {
	t.Setenv("API_BEARER_TOKEN", testBearerToken)
	s := newTestServer(t, nil)
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, "/api/v1/qoder/capabilities", nil)
		r.Header.Set("Authorization", "Bearer "+testBearerToken)
		s.router.ServeHTTP(w, r)
		assert.Equal(t, http.StatusMethodNotAllowed, w.Code, "%s should be rejected", method)
	}
}

func TestContextEndpoint_Registered(t *testing.T) {
	t.Setenv("API_BEARER_TOKEN", testBearerToken)
	s := newTestServer(t, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/qoder/context", nil)
	r.Header.Set("Authorization", "Bearer "+testBearerToken)
	s.router.ServeHTTP(w, r)

	assert.NotEqual(t, http.StatusNotFound, w.Code, "context endpoint should be registered")
}

func TestContextEndpoint_ObjectShape(t *testing.T) {
	t.Setenv("API_BEARER_TOKEN", testBearerToken)
	s := newTestServer(t, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/qoder/context", nil)
	r.Header.Set("Authorization", "Bearer "+testBearerToken)
	s.router.ServeHTTP(w, r)

	var body map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)

	_, isArray := body["items"]
	assert.False(t, isArray, "response must NOT be array-shaped format")

	expectedFields := []string{"request_id", "system", "summary", "top_alerts", "open_tasks", "pending_outbox", "recent_diagnosis", "allowed_actions", "forbidden_actions"}
	for _, field := range expectedFields {
		_, exists := body[field]
		assert.True(t, exists, "response should have field: %s", field)
	}
}

func TestContextEndpoint_RejectsNonGET(t *testing.T) {
	t.Setenv("API_BEARER_TOKEN", testBearerToken)
	s := newTestServer(t, nil)
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, "/api/v1/qoder/context", nil)
		r.Header.Set("Authorization", "Bearer "+testBearerToken)
		s.router.ServeHTTP(w, r)
		assert.Equal(t, http.StatusMethodNotAllowed, w.Code, "%s should be rejected", method)
	}
}
