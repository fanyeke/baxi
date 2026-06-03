package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestWriteError_ProducesCorrectJSON verifies that WriteError returns
// a properly formatted JSON response with all 5 fields.
func TestWriteError_ProducesCorrectJSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	// Set request_id in context
	r = r.WithContext(context.WithValue(r.Context(), RequestIDKey, "test-rid-123"))

	WriteError(w, r, http.StatusNotFound, NOT_FOUND,
		"Resource not found",
		"Requested outbox event does not exist",
		"Check the event ID and retry",
	)

	resp := w.Result()
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", resp.StatusCode)
	}

	// Check Content-Type
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}

	// Decode and verify JSON body
	var apiErr APIError
	if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if apiErr.RequestID != "test-rid-123" {
		t.Errorf("expected request_id 'test-rid-123', got %q", apiErr.RequestID)
	}
	if apiErr.ErrorCode != NOT_FOUND {
		t.Errorf("expected error_code %q, got %q", NOT_FOUND, apiErr.ErrorCode)
	}
	if apiErr.Message != "Resource not found" {
		t.Errorf("expected message 'Resource not found', got %q", apiErr.Message)
	}
	if apiErr.Diagnosis != "Requested outbox event does not exist" {
		t.Errorf("expected diagnosis 'Requested outbox event does not exist', got %q", apiErr.Diagnosis)
	}
	if apiErr.SuggestedAction != "Check the event ID and retry" {
		t.Errorf("expected suggested_action 'Check the event ID and retry', got %q", apiErr.SuggestedAction)
	}
}

// TestWriteError_IncludesRequestID verifies that the request_id from
// context is included in the error response.
func TestWriteError_IncludesRequestID(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = r.WithContext(context.WithValue(r.Context(), RequestIDKey, "custom-request-id"))

	WriteError(w, r, http.StatusBadRequest, BAD_REQUEST, "bad request", "test", "test")

	resp := w.Result()
	defer resp.Body.Close()

	var apiErr APIError
	if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if apiErr.RequestID != "custom-request-id" {
		t.Errorf("expected request_id 'custom-request-id', got %q", apiErr.RequestID)
	}
}

// TestWriteError_FallbackUnknown verifies that WriteError uses "unknown"
// as the request_id when no request_id is set in context.
func TestWriteError_FallbackUnknown(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	// No request_id set in context

	WriteError(w, r, http.StatusInternalServerError, INTERNAL_ERROR, "error", "test", "test")

	resp := w.Result()
	defer resp.Body.Close()

	var apiErr APIError
	if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if apiErr.RequestID != "unknown" {
		t.Errorf("expected request_id 'unknown', got %q", apiErr.RequestID)
	}
}

// TestRecoveryMiddleware_CatchesPanic verifies that RecoveryMiddleware
// catches panics and returns a 500 INTERNAL_ERROR response.
func TestRecoveryMiddleware_CatchesPanic(t *testing.T) {
	handler := RecoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = r.WithContext(context.WithValue(r.Context(), RequestIDKey, "panic-request"))

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}

	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}

	var apiErr APIError
	if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if apiErr.ErrorCode != INTERNAL_ERROR {
		t.Errorf("expected error_code %q, got %q", INTERNAL_ERROR, apiErr.ErrorCode)
	}
	if apiErr.RequestID != "panic-request" {
		t.Errorf("expected request_id 'panic-request', got %q", apiErr.RequestID)
	}
}

// TestRecoveryMiddleware_NoPanic verifies that RecoveryMiddleware passes
// through requests that do not panic.
func TestRecoveryMiddleware_NoPanic(t *testing.T) {
	var called bool
	handler := RecoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	handler.ServeHTTP(w, r)

	if !called {
		t.Error("expected handler to be called")
	}
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Result().StatusCode)
	}
}

// TestWriteError_AllErrorCodes verifies that all defined error codes
// can be used with WriteError without issues.
func TestWriteError_AllErrorCodes(t *testing.T) {
	codes := []struct {
		constant string
		name     string
	}{
		{UNAUTHORIZED, "UNAUTHORIZED"},
		{FORBIDDEN, "FORBIDDEN"},
		{BAD_REQUEST, "BAD_REQUEST"},
		{NOT_FOUND, "NOT_FOUND"},
		{DB_QUERY_FAILED, "DB_QUERY_FAILED"},
		{INTERNAL_ERROR, "INTERNAL_ERROR"},
		{VALIDATION_FAILED, "VALIDATION_FAILED"},
	}

	for _, c := range codes {
		t.Run(c.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r = r.WithContext(context.WithValue(r.Context(), RequestIDKey, "test"))

			WriteError(w, r, http.StatusInternalServerError, c.constant, c.name, "diagnosis", "action")

			resp := w.Result()
			defer resp.Body.Close()

			var apiErr APIError
			if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
				t.Fatalf("failed to decode JSON for code %s: %v", c.name, err)
			}

			if apiErr.ErrorCode != c.constant {
				t.Errorf("expected error_code %q, got %q", c.constant, apiErr.ErrorCode)
			}
		})
	}
}

// ──── New error constants ──────────────────────────────────────────────────────

func TestErrorConstants_CONFLICT(t *testing.T) {
	if CONFLICT != "CONFLICT" {
		t.Errorf("expected CONFLICT to be \"CONFLICT\", got %q", CONFLICT)
	}
}

func TestErrorConstants_SERVICE_UNAVAILABLE(t *testing.T) {
	if SERVICE_UNAVAILABLE != "SERVICE_UNAVAILABLE" {
		t.Errorf("expected SERVICE_UNAVAILABLE to be \"SERVICE_UNAVAILABLE\", got %q", SERVICE_UNAVAILABLE)
	}
}

// ──── WriteErrorWithDetails ────────────────────────────────────────────────────

func TestWriteErrorWithDetails(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = r.WithContext(context.WithValue(r.Context(), RequestIDKey, "test-rid"))

	details := map[string]interface{}{
		"fields": []map[string]string{
			{"field": "name", "message": "name is required"},
		},
	}
	WriteErrorWithDetails(w, r, http.StatusBadRequest, VALIDATION_FAILED,
		"Validation failed", "Check input", "Fix errors", details)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if body["error_code"] != VALIDATION_FAILED {
		t.Errorf("expected error_code %q, got %q", VALIDATION_FAILED, body["error_code"])
	}

	detailsRaw, ok := body["details"]
	if !ok {
		t.Fatal("expected details field in response")
	}
	detailsMap, ok := detailsRaw.(map[string]interface{})
	if !ok {
		t.Fatalf("expected details to be an object, got %T", detailsRaw)
	}
	fields, ok := detailsMap["fields"]
	if !ok {
		t.Fatal("expected details.fields in response")
	}
	fieldsArr, ok := fields.([]interface{})
	if !ok {
		t.Fatalf("expected details.fields to be an array, got %T", fields)
	}
	if len(fieldsArr) != 1 {
		t.Errorf("expected 1 field error, got %d", len(fieldsArr))
	}
}

func TestAPIError_DetailsOmitted(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = r.WithContext(context.WithValue(r.Context(), RequestIDKey, "test-rid"))

	WriteErrorWithDetails(w, r, http.StatusNotFound, NOT_FOUND,
		"Not found", "Resource missing", "Check ID", nil)

	resp := w.Result()
	defer resp.Body.Close()

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if _, exists := body["details"]; exists {
		t.Error("expected details field to be omitted when nil")
	}
}
