package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const handlerTestToken = "test-integration-token-long-enough-32-chars"

func TestHandleListTasks_InvalidPaginationParam(t *testing.T) {
	t.Setenv("API_BEARER_TOKEN", handlerTestToken)
	s := newTestServer(t, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/tasks?limit=abc", nil)
	r.Header.Set("Authorization", "Bearer "+handlerTestToken)
	s.router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var body map[string]string
	err := json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)
	assert.Contains(t, body["error"], "invalid")
}

func TestHandleListTasks_InvalidOffset(t *testing.T) {
	t.Setenv("API_BEARER_TOKEN", handlerTestToken)
	s := newTestServer(t, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/tasks?offset=xyz", nil)
	r.Header.Set("Authorization", "Bearer "+handlerTestToken)
	s.router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleListTasks_Registered(t *testing.T) {
	t.Setenv("API_BEARER_TOKEN", handlerTestToken)
	s := newTestServer(t, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/tasks?limit=abc", nil)
	r.Header.Set("Authorization", "Bearer "+handlerTestToken)
	s.router.ServeHTTP(w, r)

	// Verify the route is registered (not 404)
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}
