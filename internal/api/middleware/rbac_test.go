package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRBACMiddleware_NoRequiredRoles(t *testing.T) {
	rbac := NewRBACMiddleware()

	var called bool
	handler := rbac(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/protected", nil)
	r = r.WithContext(context.WithValue(r.Context(), RequestIDKey, "test-rid"))
	r = r.WithContext(context.WithValue(r.Context(), IdentityKey, &UserIdentity{
		Username: "viewer-user",
		Roles:    []string{"viewer"},
	}))

	handler.ServeHTTP(w, r)

	if !called {
		t.Error("expected handler to be called when no required roles specified")
	}
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Result().StatusCode)
	}
}

func TestRBACMiddleware_UserHasRequiredRole(t *testing.T) {
	rbac := NewRBACMiddleware("admin", "analyst")

	var called bool
	handler := rbac(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/protected", nil)
	r = r.WithContext(context.WithValue(r.Context(), RequestIDKey, "test-rid"))
	r = r.WithContext(context.WithValue(r.Context(), IdentityKey, &UserIdentity{
		Username: "alice",
		Roles:    []string{"analyst"},
	}))

	handler.ServeHTTP(w, r)

	if !called {
		t.Error("expected handler to be called when user has required role")
	}
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Result().StatusCode)
	}
}

func TestRBACMiddleware_UserLacksRequiredRole(t *testing.T) {
	rbac := NewRBACMiddleware("admin")

	handler := rbac(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/protected", nil)
	r = r.WithContext(context.WithValue(r.Context(), RequestIDKey, "test-rid"))
	r = r.WithContext(context.WithValue(r.Context(), IdentityKey, &UserIdentity{
		Username: "viewer-user",
		Roles:    []string{"viewer"},
	}))

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}

	var apiErr APIError
	if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if apiErr.ErrorCode != FORBIDDEN {
		t.Errorf("expected error_code %q, got %q", FORBIDDEN, apiErr.ErrorCode)
	}
}

func TestRBACMiddleware_MultipleRolesUserHasOne(t *testing.T) {
	rbac := NewRBACMiddleware("admin", "marketing_ops")

	var called bool
	handler := rbac(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/protected", nil)
	r = r.WithContext(context.WithValue(r.Context(), RequestIDKey, "test-rid"))
	r = r.WithContext(context.WithValue(r.Context(), IdentityKey, &UserIdentity{
		Username: "ops-user",
		Roles:    []string{"marketing_ops", "viewer"},
	}))

	handler.ServeHTTP(w, r)

	if !called {
		t.Error("expected handler to be called when user has one of the required roles")
	}
}

func TestRBACMiddleware_UserWithNoRoles(t *testing.T) {
	rbac := NewRBACMiddleware("admin")

	handler := rbac(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/protected", nil)
	r = r.WithContext(context.WithValue(r.Context(), RequestIDKey, "test-rid"))
	r = r.WithContext(context.WithValue(r.Context(), IdentityKey, &UserIdentity{
		Username: "no-role-user",
		Roles:    []string{},
	}))

	handler.ServeHTTP(w, r)

	if w.Result().StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Result().StatusCode)
	}
}

func TestRBACMiddleware_NoIdentityInContext_UsesLegacy(t *testing.T) {
	rbac := NewRBACMiddleware("viewer")

	var called bool
	handler := rbac(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/protected", nil)
	r = r.WithContext(context.WithValue(r.Context(), RequestIDKey, "test-rid"))

	handler.ServeHTTP(w, r)

	if !called {
		t.Error("expected handler to be called — legacy identity has viewer role")
	}
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Result().StatusCode)
	}
}

func TestRBACMiddleware_LegacyIdentity_IsViewer(t *testing.T) {
	rbac := NewRBACMiddleware("admin")

	handler := rbac(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called — legacy viewer cannot access admin route")
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/protected", nil)
	r = r.WithContext(context.WithValue(r.Context(), RequestIDKey, "test-rid"))

	handler.ServeHTTP(w, r)

	if w.Result().StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Result().StatusCode)
	}
}

func TestRBACMiddleware_FullChain_AuthThenRBAC(t *testing.T) {
	token := "this-is-a-valid-token-that-is-long-enough-32"
	authMw := NewAuthMiddleware(token)
	rbacMw := NewRBACMiddleware("viewer")

	var identity *UserIdentity
	handler := authMw(rbacMw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity = GetIdentity(r.Context())
		w.WriteHeader(http.StatusOK)
	})))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/protected", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	r = r.WithContext(context.WithValue(r.Context(), RequestIDKey, "test-rid"))

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if identity.Username != "qoder" {
		t.Errorf("expected username 'qoder', got %q", identity.Username)
	}
	if len(identity.Roles) != 1 || identity.Roles[0] != "viewer" {
		t.Errorf("expected roles [viewer], got %v", identity.Roles)
	}
}

func TestRBACMiddleware_FullChain_AuthThenRBAC_Denied(t *testing.T) {
	token := "this-is-a-valid-token-that-is-long-enough-32"
	authMw := NewAuthMiddleware(token)
	rbacMw := NewRBACMiddleware("analyst")

	handler := authMw(rbacMw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	})))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/protected", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	r = r.WithContext(context.WithValue(r.Context(), RequestIDKey, "test-rid"))

	handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}
