package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"baxi/internal/action"
)

// ──── newRealFeishuClient ──────────────────────────────────────────────────

func TestNewRealFeishuClient(t *testing.T) {
	c := newRealFeishuClient("app1", "secret1", false)
	if c == nil {
		t.Fatal("expected non-nil client")
	}
	if c.appID != "app1" {
		t.Errorf("expected appID=app1, got %s", c.appID)
	}
	if c.appSecret != "secret1" {
		t.Errorf("expected appSecret=secret1, got %s", c.appSecret)
	}
	if c.dryRun {
		t.Error("expected dryRun=false")
	}
}

func TestNewRealFeishuClient_DryRun(t *testing.T) {
	c := newRealFeishuClient("app1", "secret1", true)
	if !c.dryRun {
		t.Error("expected dryRun=true")
	}
}

// ──── getTenantAccessToken ─────────────────────────────────────────────────

func TestGetTenantAccessToken_DryRun(t *testing.T) {
	c := newRealFeishuClient("app1", "secret1", true)
	token, err := c.getTenantAccessToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "dry_run_token" {
		t.Errorf("expected dry_run_token, got %s", token)
	}
}

func TestGetTenantAccessToken_CachedToken(t *testing.T) {
	c := newRealFeishuClient("app1", "secret1", false)
	c.accessToken = "cached_token"
	c.tokenExpiry = time.Now().Add(1 * time.Hour)

	token, err := c.getTenantAccessToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "cached_token" {
		t.Errorf("expected cached_token, got %s", token)
	}
}

func TestGetTenantAccessToken_ExpiredToken(t *testing.T) {
	c := newRealFeishuClient("app1", "secret1", false)
	c.accessToken = "expired_token"
	c.tokenExpiry = time.Now().Add(-1 * time.Hour) // expired

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tenant_access_token": "new_token",
			"expire":              7200,
		})
	}))
	defer server.Close()

	c.baseURL = server.URL
	token, err := c.getTenantAccessToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "new_token" {
		t.Errorf("expected new_token, got %s", token)
	}
}

func TestGetTenantAccessToken_HTTPError(t *testing.T) {
	c := newRealFeishuClient("app1", "secret1", false)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	c.baseURL = server.URL
	// Client now correctly returns error when JSON parsing fails
	_, err := c.getTenantAccessToken()
	if err == nil {
		t.Fatal("expected error for HTTP 500 with invalid JSON")
	}
}

func TestGetTenantAccessToken_InvalidJSON(t *testing.T) {
	c := newRealFeishuClient("app1", "secret1", false)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	c.baseURL = server.URL
	// Client now correctly returns error when JSON is malformed
	_, err := c.getTenantAccessToken()
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}

func TestGetTenantAccessToken_MissingExpire(t *testing.T) {
	c := newRealFeishuClient("app1", "secret1", false)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		// Response without "expire" field
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tenant_access_token": "token_no_expire",
		})
	}))
	defer server.Close()

	c.baseURL = server.URL
	token, err := c.getTenantAccessToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "token_no_expire" {
		t.Errorf("expected token_no_expire, got %s", token)
	}
}

// ──── sendMessage ──────────────────────────────────────────────────────────

func TestSendMessage_DryRun(t *testing.T) {
	c := newRealFeishuClient("app1", "secret1", true)
	msgID, err := c.sendMessage("chat123", "hello", "text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msgID == "" {
		t.Error("expected non-empty message ID")
	}
}

func TestSendMessage_Success(t *testing.T) {
	c := newRealFeishuClient("app1", "secret1", false)
	c.accessToken = "valid_token"
	c.tokenExpiry = time.Now().Add(1 * time.Hour)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"message_id": "msg_12345",
			},
		})
	}))
	defer server.Close()

	c.baseURL = server.URL
	msgID, err := c.sendMessage("chat123", "hello", "text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msgID != "msg_12345" {
		t.Errorf("expected msg_12345, got %s", msgID)
	}
}

func TestSendMessage_APIError(t *testing.T) {
	c := newRealFeishuClient("app1", "secret1", false)
	c.accessToken = "valid_token"
	c.tokenExpiry = time.Now().Add(1 * time.Hour)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 170002,
			"msg":  "chat not found",
		})
	}))
	defer server.Close()

	c.baseURL = server.URL
	_, err := c.sendMessage("chat123", "hello", "text")
	if err == nil {
		t.Fatal("expected error for API error code")
	}
	if err != nil && !containsStr2(err.Error(), "170002") {
		t.Errorf("expected 170002 in error, got: %s", err.Error())
	}
}

func TestSendMessage_SuccessNoData(t *testing.T) {
	c := newRealFeishuClient("app1", "secret1", false)
	c.accessToken = "valid_token"
	c.tokenExpiry = time.Now().Add(1 * time.Hour)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		// Response with code=0 but no data field
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 0,
		})
	}))
	defer server.Close()

	c.baseURL = server.URL
	msgID, err := c.sendMessage("chat123", "hello", "text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No data means msgID is empty
	if msgID != "" {
		t.Errorf("expected empty msgID when no data, got %s", msgID)
	}
}

func TestSendMessage_TokenFetchError(t *testing.T) {
	c := newRealFeishuClient("app1", "secret1", false)
	// Use a non-existent server to cause connection refused
	c.baseURL = "http://127.0.0.1:1" // port 1 is not listening

	_, err := c.sendMessage("chat123", "hello", "text")
	if err == nil {
		t.Fatal("expected error when token fetch fails with connection refused")
	}
}

func TestSendMessage_RateLimit(t *testing.T) {
	c := newRealFeishuClient("app1", "secret1", false)
	c.accessToken = "valid_token"
	c.tokenExpiry = time.Now().Add(1 * time.Hour)

	attempt := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if attempt <= 2 {
			// Rate limit on first 2 attempts
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(429)
			json.NewEncoder(w).Encode(map[string]interface{}{"msg": "rate limit"})
		} else {
			// Success on 3rd attempt
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 0,
				"data": map[string]interface{}{"message_id": "msg_ok"},
			})
		}
	}))
	defer server.Close()

	c.baseURL = server.URL
	msgID, err := c.sendMessage("chat123", "hello", "text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msgID != "msg_ok" {
		t.Errorf("expected msg_ok, got %s", msgID)
	}
	if attempt < 3 {
		t.Errorf("expected at least 3 attempts, got %d", attempt)
	}
}

// ──── FeishuAdapter with mock error client ─────────────────────────────────

type errorFeishuClient struct{}

func (e *errorFeishuClient) getTenantAccessToken() (string, error) {
	return "", fmt.Errorf("token fetch failed")
}

func (e *errorFeishuClient) sendMessage(chatID, content, msgType string) (string, error) {
	return "", fmt.Errorf("send message failed")
}

func TestFeishuAdapter_MockErrorClient(t *testing.T) {
	adapter := NewFeishuAdapterWithClient(FeishuConfig{
		ChatID: "oc_test",
	}, &errorFeishuClient{})

	proposal := action.ActionProposal{
		ProposalID: "p-err",
		CaseID:     "c-err",
		ActionType: "export_report",
	}
	result, err := adapter.Execute(context.Background(), proposal, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected Success=false with error client")
	}
	if result.Error == "" {
		t.Error("expected non-empty error message")
	}
}

// ──── GitHub doRequest error paths ────────────────────────────────────────

func TestGitHubDoRequest_422(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(422)
		w.Write([]byte(`{"message":"Validation Failed"}`))
	}))
	defer server.Close()

	adapter := NewGitHubAdapter(GitHubConfig{Token: "ghp_test", Repo: "o/r"})
	adapter.baseURL = server.URL

	_, err := adapter.doRequest("POST", "/test", map[string]string{"key": "val"})
	if err == nil {
		t.Fatal("expected error for 422")
	}
	if !containsStr2(err.Error(), "422") {
		t.Errorf("expected 422 in error, got: %s", err.Error())
	}
}

func TestGitHubDoRequest_DefaultError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(502)
		w.Write([]byte("bad gateway"))
	}))
	defer server.Close()

	adapter := NewGitHubAdapter(GitHubConfig{Token: "ghp_test", Repo: "o/r"})
	adapter.baseURL = server.URL

	_, err := adapter.doRequest("POST", "/test", nil)
	if err == nil {
		t.Fatal("expected error for 502")
	}
	if !containsStr2(err.Error(), "502") {
		t.Errorf("expected 502 in error, got: %s", err.Error())
	}
}

func TestGitHubDoRequest_NilBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	adapter := NewGitHubAdapter(GitHubConfig{Token: "ghp_test", Repo: "o/r"})
	adapter.baseURL = server.URL

	body, err := adapter.doRequest("GET", "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body == nil {
		t.Error("expected non-nil body")
	}
}

func TestGitHubAdapter_AddLabels_Success_Extra(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	adapter := NewGitHubAdapter(GitHubConfig{Token: "ghp_token", Repo: "o/r"})
	adapter.baseURL = server.URL

	err := adapter.AddLabels(context.Background(), 1, []string{"bug", "urgent"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGitHubAdapter_AddComment_Success_Extra(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
		w.Write([]byte(`{"id": 1}`))
	}))
	defer server.Close()

	adapter := NewGitHubAdapter(GitHubConfig{Token: "ghp_token", Repo: "o/r"})
	adapter.baseURL = server.URL

	err := adapter.AddComment(context.Background(), 1, GitHubComment{Body: "test comment"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ──── helpers ──────────────────────────────────────────────────────────────

func containsStr2(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
