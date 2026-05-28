package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthEndpoint_ResponseFormat(t *testing.T) {
	s := New(nil, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	s.router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp HealthResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, "ok", resp.Status)
	assert.Equal(t, apiVersion, resp.Version)
}

func TestHealthResponse_NoServiceField(t *testing.T) {
	s := New(nil, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	s.router.ServeHTTP(w, r)

	var body map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)

	_, hasService := body["service"]
	assert.False(t, hasService, "response should not contain 'service' field")
	_, hasVersion := body["version"]
	assert.True(t, hasVersion, "response should contain 'version' field")
	_, hasDBConnected := body["db_connected"]
	assert.True(t, hasDBConnected, "response should contain 'db_connected' field")
}

func TestHealthEndpoint_DBDisconnected(t *testing.T) {
	s := New(nil, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	s.router.ServeHTTP(w, r)

	var resp HealthResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	// With nil pool, db_connected must be false
	assert.False(t, resp.DBConnected)
}

func TestHealthEndpoint_NoAuthRequired(t *testing.T) {
	s := New(nil, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	s.router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code, "health endpoint should be accessible without auth")
}
