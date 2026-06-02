package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testBearerToken = "test-integration-token-long-enough-32-chars"
)

// allProtectedRoutes lists all protected endpoints for parameterized testing.
var allProtectedRoutes = []struct {
	name string
	path string
}{
	{"/api/v1/status", "/api/v1/status"},
	{"/api/v1/alerts", "/api/v1/alerts"},
	{"/api/v1/tasks", "/api/v1/tasks"},
	{"/api/v1/outbox", "/api/v1/outbox"},
	{"/api/v1/governance/status", "/api/v1/governance/status"},
	{"/api/v1/logs/recent", "/api/v1/logs/recent"},
	{"/api/v1/logs/errors", "/api/v1/logs/errors"},
	{"/api/v1/logs/audit", "/api/v1/logs/audit"},
	{"/api/v1/qoder/capabilities", "/api/v1/qoder/capabilities"},
	{"/api/v1/qoder/context", "/api/v1/qoder/context"},
}

func TestAllEndpoints_Registered(t *testing.T) {
	s := newTestServer(t, nil)

	t.Run("health", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
		s.router.ServeHTTP(w, r)
		assert.Equal(t, http.StatusOK, w.Code, "health should return 200")

		var body map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, "ok", body["status"])
		assert.Equal(t, "0.6.0", body["version"])
	})

	for _, ep := range allProtectedRoutes {
		t.Run(ep.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, ep.path, nil)
			s.router.ServeHTTP(w, r)
			assert.NotEqual(t, http.StatusNotFound, w.Code,
				"%s should be registered (got 404)", ep.path)
		})
	}
}

func TestHealthEndpoint_Public(t *testing.T) {
	t.Setenv("API_BEARER_TOKEN", testBearerToken)
	s := newTestServer(t, nil)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	s.router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code, "health must be accessible without auth")
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

func TestProtectedEndpoints_RequireAuth(t *testing.T) {
	t.Setenv("API_BEARER_TOKEN", testBearerToken)
	s := newTestServer(t, nil)

	for _, ep := range allProtectedRoutes {
		t.Run(ep.name+" no auth", func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, ep.path, nil)
			s.router.ServeHTTP(w, r)
			assert.Equal(t, http.StatusUnauthorized, w.Code,
				"%s should return 401 without auth", ep.path)
		})
	}
}

func TestProtectedEndpoints_InvalidToken(t *testing.T) {
	t.Setenv("API_BEARER_TOKEN", testBearerToken)
	s := newTestServer(t, nil)

	for _, ep := range allProtectedRoutes {
		t.Run(ep.name+" invalid token", func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, ep.path, nil)
			r.Header.Set("Authorization", "Bearer wrong-token-that-is-long-enough-32-xxx")
			s.router.ServeHTTP(w, r)
			assert.Equal(t, http.StatusUnauthorized, w.Code,
				"%s should return 401 with invalid token", ep.path)
		})
	}
}

func TestProtectedEndpoints_ValidToken(t *testing.T) {
	t.Setenv("API_BEARER_TOKEN", testBearerToken)
	s := newTestServer(t, nil)

	for _, ep := range allProtectedRoutes {
		t.Run(ep.name+" valid token", func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, ep.path, nil)
			r.Header.Set("Authorization", "Bearer "+testBearerToken)
			s.router.ServeHTTP(w, r)

			// Auth passed: must not be 401 or 404
			assert.NotEqual(t, http.StatusUnauthorized, w.Code,
				"%s should not return 401 with valid token", ep.path)
			assert.NotEqual(t, http.StatusNotFound, w.Code,
				"%s should be registered (not 404)", ep.path)
			// Some endpoints may panic with nil pool (chi recoverer returns 500 without JSON Content-Type),
			// but the key check is that auth did not reject the request.
		})
	}
}

func TestProtectedEndpoints_MissingAuthHeader(t *testing.T) {
	t.Setenv("API_BEARER_TOKEN", testBearerToken)
	s := newTestServer(t, nil)

	for _, ep := range allProtectedRoutes {
		t.Run(ep.name+" missing header", func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, ep.path, nil)
			// No Authorization header
			s.router.ServeHTTP(w, r)
			assert.Equal(t, http.StatusUnauthorized, w.Code,
				"%s should return 401 without Authorization header", ep.path)
		})
	}
}

func TestCORSPreflight(t *testing.T) {
	s := newTestServer(t, nil)

	tests := []struct {
		name   string
		method string
		path   string
		origin string
	}{
		{"health OPTIONS no origin", http.MethodOptions, "/api/v1/health", ""},
		{"health OPTIONS with origin", http.MethodOptions, "/api/v1/health", "http://localhost:5173"},
		{"status OPTIONS", http.MethodOptions, "/api/v1/status", ""},
		{"tasks OPTIONS", http.MethodOptions, "/api/v1/tasks", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.origin != "" {
				r.Header.Set("Origin", tt.origin)
			}
			// Should not panic or crash
			s.router.ServeHTTP(w, r)
			// OPTIONS to GET-only routes returns 405 from chi
			assert.NotEqual(t, http.StatusInternalServerError, w.Code,
				"OPTIONS should not cause 500")
		})
	}
}

func TestAuthMiddleware_ErrorResponseShape(t *testing.T) {
	t.Setenv("API_BEARER_TOKEN", testBearerToken)
	s := newTestServer(t, nil)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	s.router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err, "response body must be valid JSON")

	assert.Contains(t, body, "error_code")
	assert.Contains(t, body, "message")
	assert.Contains(t, body, "diagnosis")
	assert.Contains(t, body, "suggested_action")
	assert.Contains(t, body, "request_id")
	assert.Equal(t, "UNAUTHORIZED", body["error_code"])
}
