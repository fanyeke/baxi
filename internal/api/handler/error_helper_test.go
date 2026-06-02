package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"

	"baxi/internal/api/middleware"
)

// ──── classifyError ────────────────────────────────────────────────────────

func TestClassifyError_ErrNoRows(t *testing.T) {
	status, code := classifyError(pgx.ErrNoRows)
	assert.Equal(t, http.StatusNotFound, status)
	assert.Equal(t, middleware.NOT_FOUND, code)
}

func TestClassifyError_GenericError(t *testing.T) {
	status, code := classifyError(errors.New("something bad"))
	assert.Equal(t, http.StatusInternalServerError, status)
	assert.Equal(t, middleware.INTERNAL_ERROR, code)
}

func TestClassifyError_WrappedNoRows(t *testing.T) {
	wrapped := errors.New("wrapped: " + pgx.ErrNoRows.Error())
	// pgx.ErrNoRows must be wrapped with errors.Is-compatible wrapping
	// Since we're using a plain errors.New, it won't match by Is. Let's wrap properly.
	wrapped = errors.Join(pgx.ErrNoRows, errors.New("query failed"))
	status, code := classifyError(wrapped)
	assert.Equal(t, http.StatusNotFound, status)
	assert.Equal(t, middleware.NOT_FOUND, code)
}

func TestClassifyError_NilError(t *testing.T) {
	// classifyError is defensive but nil shouldn't normally occur
	status, code := classifyError(nil)
	assert.Equal(t, http.StatusInternalServerError, status)
	assert.Equal(t, middleware.INTERNAL_ERROR, code)
}

// ──── writeServiceError ────────────────────────────────────────────────────

func TestWriteServiceError_NotFound(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	writeServiceError(w, r, pgx.ErrNoRows, "fallback message")

	resp := w.Result()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	var body map[string]interface{}
	decodeJSON(t, resp, &body)
	assert.Equal(t, middleware.NOT_FOUND, body["error_code"])
	assert.Equal(t, "The requested resource was not found", body["message"])
}

func TestWriteServiceError_InternalError(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	writeServiceError(w, r, errors.New("db timeout"), "custom fallback")

	resp := w.Result()
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var body map[string]interface{}
	decodeJSON(t, resp, &body)
	assert.Equal(t, middleware.INTERNAL_ERROR, body["error_code"])
	assert.Equal(t, "custom fallback", body["message"])
}

// ──── defaultDiagnosis ─────────────────────────────────────────────────────

func TestDefaultDiagnosis_AllCases(t *testing.T) {
	tests := []struct {
		code string
		want string
	}{
		{middleware.UNAUTHORIZED,      "The request lacks valid authentication credentials"},
		{middleware.FORBIDDEN,         "The server understood the request but refuses to authorize it"},
		{middleware.BAD_REQUEST,       "The request contains invalid parameters or malformed data"},
		{middleware.NOT_FOUND,         "The requested resource does not exist at the specified path"},
		{middleware.DB_QUERY_FAILED,   "A database query failed to execute successfully"},
		{middleware.VALIDATION_FAILED, "The request data failed validation checks"},
		{middleware.INTERNAL_ERROR,    "An unexpected error occurred while processing the request"},
		{"unknown_code",               "An error occurred while processing the request"},
		{"",                           "An error occurred while processing the request"},
	}
	for _, tt := range tests {
		got := defaultDiagnosis(tt.code)
		assert.Equal(t, tt.want, got, "defaultDiagnosis(%q)", tt.code)
	}
}

// ──── defaultAction ────────────────────────────────────────────────────────

func TestDefaultAction_AllCases(t *testing.T) {
	tests := []struct {
		code string
		want string
	}{
		{middleware.UNAUTHORIZED,      "Include a valid Authorization header with a Bearer token"},
		{middleware.FORBIDDEN,         "Check that you have permission to access this resource"},
		{middleware.BAD_REQUEST,       "Review the request parameters and try again"},
		{middleware.NOT_FOUND,         "Verify the resource ID and try again"},
		{middleware.DB_QUERY_FAILED,   "Retry the request; contact support if the issue persists"},
		{middleware.VALIDATION_FAILED, "Fix the validation errors and retry"},
		{middleware.INTERNAL_ERROR,    "Retry the request; contact support if the issue persists"},
		{"unknown_code",               "Retry the request or contact support"},
		{"",                           "Retry the request or contact support"},
	}
	for _, tt := range tests {
		got := defaultAction(tt.code)
		assert.Equal(t, tt.want, got, "defaultAction(%q)", tt.code)
	}
}

// ──── writeError ───────────────────────────────────────────────────────────

func TestWriteError_FormatsErrorResponse(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	writeError(w, r, http.StatusBadRequest, middleware.BAD_REQUEST, "invalid input")

	resp := w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var body map[string]interface{}
	decodeJSON(t, resp, &body)
	assert.Equal(t, middleware.BAD_REQUEST, body["error_code"])
	assert.Equal(t, "invalid input", body["message"])
	assert.Equal(t, "The request contains invalid parameters or malformed data", body["diagnosis"])
	assert.Equal(t, "Review the request parameters and try again", body["suggested_action"])
}

func TestWriteError_NotFound(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	writeError(w, r, http.StatusNotFound, middleware.NOT_FOUND, "resource missing")

	resp := w.Result()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	var body map[string]interface{}
	decodeJSON(t, resp, &body)
	assert.Equal(t, middleware.NOT_FOUND, body["error_code"])
	assert.Equal(t, "resource missing", body["message"])
	assert.Equal(t, "The requested resource does not exist at the specified path", body["diagnosis"])
	assert.Equal(t, "Verify the resource ID and try again", body["suggested_action"])
}

func TestWriteError_InternalError(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR, "unexpected")

	resp := w.Result()
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var body map[string]interface{}
	decodeJSON(t, resp, &body)
	assert.Equal(t, middleware.INTERNAL_ERROR, body["error_code"])
	assert.Equal(t, "unexpected", body["message"])
}

// ──── writeErrorWithDetails ────────────────────────────────────────────────

func TestWriteErrorWithDetails_OverridesDefaults(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	writeErrorWithDetails(w, r, http.StatusForbidden, middleware.FORBIDDEN,
		"access denied", "custom diagnosis", "custom action")

	resp := w.Result()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	var body map[string]interface{}
	decodeJSON(t, resp, &body)
	assert.Equal(t, middleware.FORBIDDEN, body["error_code"])
	assert.Equal(t, "access denied", body["message"])
	assert.Equal(t, "custom diagnosis", body["diagnosis"])
	assert.Equal(t, "custom action", body["suggested_action"])
}

// ──── helper ───────────────────────────────────────────────────────────────

func decodeJSON(t *testing.T, resp *http.Response, v interface{}) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
}
