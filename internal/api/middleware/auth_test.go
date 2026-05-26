package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testPassThroughHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestAuthMiddleware_MissingAuthorizationHeader(t *testing.T) {
	middleware := NewAuthMiddleware("this-is-a-valid-token-that-is-long-enough-32")
	handler := middleware(testPassThroughHandler())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	r = r.WithContext(context.WithValue(r.Context(), RequestIDKey, "test-rid"))

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}

	var apiErr APIError
	if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if apiErr.ErrorCode != UNAUTHORIZED {
		t.Errorf("expected error_code %q, got %q", UNAUTHORIZED, apiErr.ErrorCode)
	}
	if apiErr.RequestID != "test-rid" {
		t.Errorf("expected request_id 'test-rid', got %q", apiErr.RequestID)
	}
	if apiErr.Message != "Missing Authorization header" {
		t.Errorf("expected message 'Missing Authorization header', got %q", apiErr.Message)
	}
}

func TestAuthMiddleware_InvalidTokenFormat(t *testing.T) {
	middleware := NewAuthMiddleware("this-is-a-valid-token-that-is-long-enough-32")
	handler := middleware(testPassThroughHandler())

	tests := []struct {
		name    string
		header  string
		message string
	}{
		{"Basic auth", "Basic dXNlcjpwYXNz", "Invalid Authorization header format"},
		{"Digest auth", "Digest realm=test", "Invalid Authorization header format"},
		{"Bearer only", "Bearer", "Invalid Authorization header format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
			r.Header.Set("Authorization", tt.header)
			r = r.WithContext(context.WithValue(r.Context(), RequestIDKey, "test-rid"))

			handler.ServeHTTP(w, r)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusUnauthorized {
				t.Errorf("expected 401, got %d", resp.StatusCode)
			}

			var apiErr APIError
			if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
				t.Fatalf("failed to decode JSON: %v", err)
			}
			if apiErr.ErrorCode != UNAUTHORIZED {
				t.Errorf("expected error_code %q, got %q", UNAUTHORIZED, apiErr.ErrorCode)
			}
			if apiErr.Message != tt.message {
				t.Errorf("expected message %q, got %q", tt.message, apiErr.Message)
			}
		})
	}
}

func TestAuthMiddleware_EmptyBearerToken(t *testing.T) {
	middleware := NewAuthMiddleware("this-is-a-valid-token-that-is-long-enough-32")
	handler := middleware(testPassThroughHandler())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	r.Header.Set("Authorization", "Bearer ")
	r = r.WithContext(context.WithValue(r.Context(), RequestIDKey, "test-rid"))

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}

	var apiErr APIError
	if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if apiErr.ErrorCode != UNAUTHORIZED {
		t.Errorf("expected error_code %q, got %q", UNAUTHORIZED, apiErr.ErrorCode)
	}
	if apiErr.Message != "Missing token" {
		t.Errorf("expected message 'Missing token', got %q", apiErr.Message)
	}
}

func TestAuthMiddleware_ShortToken(t *testing.T) {
	middleware := NewAuthMiddleware("this-is-a-valid-token-that-is-long-enough-32")
	handler := middleware(testPassThroughHandler())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	r.Header.Set("Authorization", "Bearer short")
	r = r.WithContext(context.WithValue(r.Context(), RequestIDKey, "test-rid"))

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}

	var apiErr APIError
	if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if apiErr.ErrorCode != UNAUTHORIZED {
		t.Errorf("expected error_code %q, got %q", UNAUTHORIZED, apiErr.ErrorCode)
	}
	if apiErr.Message != "Invalid token" {
		t.Errorf("expected message 'Invalid token', got %q", apiErr.Message)
	}
}

func TestAuthMiddleware_KnownWeakToken(t *testing.T) {
	middleware := NewAuthMiddleware("this-is-a-valid-token-that-is-long-enough-32")
	handler := middleware(testPassThroughHandler())

	tests := []struct {
		name  string
		token string
	}{
		{"test-token", "test-token"},
		{"changeme", "changeme"},
		{"admin", "admin"},
		{"password", "password"},
		{"REPLACE_ME", "REPLACE_ME"},
		{"secret", "secret"},
		{"your-secret-token-here", "your-secret-token-here"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
			r.Header.Set("Authorization", "Bearer "+tt.token)
			r = r.WithContext(context.WithValue(r.Context(), RequestIDKey, "test-rid"))

			handler.ServeHTTP(w, r)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusUnauthorized {
				t.Errorf("expected 401, got %d", resp.StatusCode)
			}

			var apiErr APIError
			if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
				t.Fatalf("failed to decode JSON: %v", err)
			}
			if apiErr.ErrorCode != UNAUTHORIZED {
				t.Errorf("expected error_code %q, got %q", UNAUTHORIZED, apiErr.ErrorCode)
			}
			if apiErr.Message != "Invalid token" {
				t.Errorf("expected message 'Invalid token', got %q", apiErr.Message)
			}
		})
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	middleware := NewAuthMiddleware("this-is-a-valid-token-that-is-long-enough-32")
	handler := middleware(testPassThroughHandler())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	r.Header.Set("Authorization", "Bearer this-is-a-wrong-token-that-is-still-long-enough")
	r = r.WithContext(context.WithValue(r.Context(), RequestIDKey, "test-rid"))

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}

	var apiErr APIError
	if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if apiErr.ErrorCode != UNAUTHORIZED {
		t.Errorf("expected error_code %q, got %q", UNAUTHORIZED, apiErr.ErrorCode)
	}
	if apiErr.Message != "Invalid token" {
		t.Errorf("expected message 'Invalid token', got %q", apiErr.Message)
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	expectedToken := "this-is-a-valid-token-that-is-long-enough-32"
	middleware := NewAuthMiddleware(expectedToken)
	handler := middleware(testPassThroughHandler())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	r.Header.Set("Authorization", "Bearer "+expectedToken)
	r = r.WithContext(context.WithValue(r.Context(), RequestIDKey, "test-rid"))

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestAuthMiddleware_PublicPathSkipsAuth(t *testing.T) {
	middleware := NewAuthMiddleware(
		"this-is-a-valid-token-that-is-long-enough-32",
		"/api/v1/health",
		"/api/v1/public",
	)

	var called bool
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if !called {
		t.Error("expected handler to be called for public path")
	}
}

func TestAuthMiddleware_SecondPublicPathSkipsAuth(t *testing.T) {
	middleware := NewAuthMiddleware(
		"this-is-a-valid-token-that-is-long-enough-32",
		"/api/v1/health",
		"/api/v1/public",
	)

	var called bool
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/public", nil)

	handler.ServeHTTP(w, r)

	if !called {
		t.Error("expected handler to be called for public path")
	}
}

func TestAuthMiddleware_DefaultPublicPath(t *testing.T) {
	middleware := NewAuthMiddleware("this-is-a-valid-token-that-is-long-enough-32")

	var called bool
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if !called {
		t.Error("expected handler to be called for default public path")
	}
}

func TestAuthMiddleware_ActorInContext(t *testing.T) {
	expectedToken := "this-is-a-valid-token-that-is-long-enough-32"
	middleware := NewAuthMiddleware(expectedToken)

	var actor string
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor = GetActor(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	r.Header.Set("Authorization", "Bearer "+expectedToken)
	r = r.WithContext(context.WithValue(r.Context(), RequestIDKey, "test-rid"))

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if actor != "qoder" {
		t.Errorf("expected actor 'qoder', got %q", actor)
	}
}

func TestGetActor_Unknown(t *testing.T) {
	ctx := context.Background()
	if actor := GetActor(ctx); actor != "unknown" {
		t.Errorf("expected 'unknown', got %q", actor)
	}
}
