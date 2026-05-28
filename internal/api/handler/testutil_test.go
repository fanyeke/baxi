package handler

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"baxi/internal/api/middleware"
)

func decodeAPIError(t *testing.T, w *httptest.ResponseRecorder) middleware.APIError {
	t.Helper()
	var apiErr middleware.APIError
	if err := json.NewDecoder(w.Body).Decode(&apiErr); err != nil {
		t.Fatalf("failed to decode API error response: %v", err)
	}
	return apiErr
}

func assertAPIError(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedCode string) middleware.APIError {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("expected status %d, got %d", expectedStatus, w.Code)
	}

	apiErr := decodeAPIError(t, w)

	if apiErr.ErrorCode != expectedCode {
		t.Errorf("expected error_code %q, got %q", expectedCode, apiErr.ErrorCode)
	}

	if apiErr.RequestID == "" {
		t.Error("expected request_id to be non-empty")
	}

	if apiErr.Message == "" {
		t.Error("expected message to be non-empty")
	}

	if apiErr.Diagnosis == "" {
		t.Error("expected diagnosis to be non-empty")
	}

	if apiErr.SuggestedAction == "" {
		t.Error("expected suggested_action to be non-empty")
	}

	return apiErr
}

func assertSuccessResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("expected status %d, got %d", expectedStatus, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}
}
