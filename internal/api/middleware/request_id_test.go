package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestIDMiddleware_PropagatesProvidedHeader(t *testing.T) {
	expectedID := "my-test-request-id"

	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := GetRequestID(r.Context())
		if got != expectedID {
			t.Errorf("expected request ID %q, got %q", expectedID, got)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", expectedID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Request-ID") != expectedID {
		t.Errorf("response X-Request-ID header = %q, want %q", rec.Header().Get("X-Request-ID"), expectedID)
	}
}

func TestRequestIDMiddleware_GeneratesNewID(t *testing.T) {
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := GetRequestID(r.Context())
		if !strings.HasPrefix(got, "req_") {
			t.Errorf("generated request ID should start with %q, got %q", "req_", got)
		}
		if len(got) < 10 {
			t.Errorf("generated request ID too short: %q", got)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	headerID := rec.Header().Get("X-Request-ID")
	if headerID == "" {
		t.Error("response X-Request-ID header should not be empty")
	}
	if !strings.HasPrefix(headerID, "req_") {
		t.Errorf("response X-Request-ID should start with %q, got %q", "req_", headerID)
	}
}

func TestRequestIDMiddleware_SetsResponseHeader(t *testing.T) {
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "custom-id")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Request-ID"); got != "custom-id" {
		t.Errorf("response X-Request-ID header = %q, want %q", got, "custom-id")
	}
}

func TestGetRequestID_FromContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), RequestIDKey, "ctx-request-789")
	if got := GetRequestID(ctx); got != "ctx-request-789" {
		t.Errorf("GetRequestID() = %q, want %q", got, "ctx-request-789")
	}
}

func TestGetRequestID_EmptyContext(t *testing.T) {
	if got := GetRequestID(context.Background()); got != "unknown" {
		t.Errorf("GetRequestID() = %q, want %q", got, "unknown")
	}
}

func TestGetRequestID_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), RequestIDKey, 42)
	if got := GetRequestID(ctx); got != "unknown" {
		t.Errorf("GetRequestID() = %q, want %q", got, "unknown")
	}
}

func TestRequestIDMiddleware_UniquePerRequest(t *testing.T) {
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ids := make(map[string]bool)
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		id := rec.Header().Get("X-Request-ID")
		if ids[id] {
			t.Errorf("duplicate request ID generated: %q", id)
		}
		ids[id] = true
	}
}

func TestRequestIDMiddleware_GeneratedFormat(t *testing.T) {
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	id := rec.Header().Get("X-Request-ID")
	parts := strings.Split(id, "_")
	if len(parts) != 3 {
		t.Errorf("expected format req_<timestamp>_<random>, got %q (parts: %v)", id, parts)
	}
	if parts[0] != "req" {
		t.Errorf("expected prefix 'req', got %q", parts[0])
	}
	if parts[1] == "" {
		t.Error("timestamp part should not be empty")
	}
	if parts[2] == "" {
		t.Error("random part should not be empty")
	}
}

func TestRequestIDMiddleware_ContextCarriesThrough(t *testing.T) {
	var capturedID string
	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := RequestIDMiddleware(innerHandler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "propagate-test-id")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if capturedID != "propagate-test-id" {
		t.Errorf("context should carry the request ID, got %q, want %q", capturedID, "propagate-test-id")
	}
}
