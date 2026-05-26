package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewCORSMiddleware_PreflightAllowedOrigin(t *testing.T) {
	handler := NewCORSMiddleware("http://localhost:5173,http://localhost:3000")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called for preflight requests")
	}))

	req := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, resp.StatusCode)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Errorf("expected Access-Control-Allow-Origin %q, got %q", "http://localhost:5173", got)
	}
	if got := resp.Header.Get("Access-Control-Allow-Methods"); got != "GET, POST, OPTIONS" {
		t.Errorf("expected Access-Control-Allow-Methods %q, got %q", "GET, POST, OPTIONS", got)
	}
	if got := resp.Header.Get("Access-Control-Allow-Headers"); got != "Authorization, Content-Type, X-Request-ID" {
		t.Errorf("expected Access-Control-Allow-Headers %q, got %q", "Authorization, Content-Type, X-Request-ID", got)
	}
	if got := resp.Header.Get("Access-Control-Max-Age"); got != "86400" {
		t.Errorf("expected Access-Control-Max-Age %q, got %q", "86400", got)
	}
}

func TestNewCORSMiddleware_PreflightDisallowedOrigin(t *testing.T) {
	var called bool
	handler := NewCORSMiddleware("http://localhost:5173")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
	req.Header.Set("Origin", "http://evil.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	resp.Body.Close()

	if !called {
		t.Error("expected next handler to be called for disallowed origin")
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Error("expected no CORS headers for disallowed origin")
	}
}

func TestNewCORSMiddleware_ActualRequestAllowedOrigin(t *testing.T) {
	var called bool
	handler := NewCORSMiddleware("http://localhost:5173")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	resp.Body.Close()

	if !called {
		t.Error("expected next handler to be called for actual request")
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Errorf("expected Access-Control-Allow-Origin %q, got %q", "http://localhost:5173", got)
	}
}

func TestNewCORSMiddleware_ActualRequestDisallowedOrigin(t *testing.T) {
	var called bool
	handler := NewCORSMiddleware("http://localhost:5173")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Origin", "http://evil.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	resp.Body.Close()

	if !called {
		t.Error("expected next handler to be called for actual request")
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Error("expected no CORS headers for disallowed origin")
	}
}

func TestNewCORSMiddleware_MultipleAllowedOrigins(t *testing.T) {
	handler := NewCORSMiddleware("http://a.example.com,http://b.example.com,http://c.example.com")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name    string
		origin  string
		allowed bool
	}{
		{"first allowed", "http://a.example.com", true},
		{"second allowed", "http://b.example.com", true},
		{"third allowed", "http://c.example.com", true},
		{"disallowed", "http://d.example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
			req.Header.Set("Origin", tt.origin)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if tt.allowed {
				if got := rec.Header().Get("Access-Control-Allow-Origin"); got != tt.origin {
					t.Errorf("expected Access-Control-Allow-Origin %q, got %q", tt.origin, got)
				}
			} else {
				if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
					t.Errorf("expected no CORS headers, got %q", got)
				}
			}
		})
	}
}

func TestNewCORSMiddleware_SingleAllowedOrigin(t *testing.T) {
	handler := NewCORSMiddleware("http://localhost:3000")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Errorf("expected Access-Control-Allow-Origin %q, got %q", "http://localhost:3000", got)
	}
}

func TestNewCORSMiddleware_EmptyOrigin(t *testing.T) {
	var called bool
	handler := NewCORSMiddleware("http://localhost:5173")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	// No Origin header set
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	resp.Body.Close()

	if !called {
		t.Error("expected next handler to be called when no origin header")
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Error("expected no CORS headers when no origin header")
	}
}

func TestNewCORSMiddleware_NoWildcard(t *testing.T) {
	handler := NewCORSMiddleware("http://localhost:5173")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Origin", "*")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no Access-Control-Allow-Origin for wildcard origin, got %q", got)
	}
}

func TestNewCORSMiddleware_WhiteSpaceInOrigins(t *testing.T) {
	var called bool
	handler := NewCORSMiddleware(" http://localhost:5173 , http://localhost:3000 ")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	// Should match after trimming
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Errorf("expected Access-Control-Allow-Origin %q, got %q", "http://localhost:5173", got)
	}
	if !called {
		t.Error("expected next handler to be called")
	}
}
