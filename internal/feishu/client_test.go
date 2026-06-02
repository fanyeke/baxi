package feishu

import (
	"testing"
)

func TestNewClient_DryRun(t *testing.T) {
	c := NewClient("app-id", "app-secret", "app-token", true)

	if c.dryRun != true {
		t.Error("expected dryRun to be true")
	}
	if c.appID != "app-id" {
		t.Errorf("appID = %q, want %q", c.appID, "app-id")
	}
	if c.appSecret != "app-secret" {
		t.Errorf("appSecret = %q, want %q", c.appSecret, "app-secret")
	}
	if c.baseURL != feishuBaseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, feishuBaseURL)
	}
	if c.httpClient == nil {
		t.Error("expected httpClient to be non-nil")
	}
}

func TestNewClient_LiveMode(t *testing.T) {
	c := NewClient("live-id", "live-secret", "live-token", false)

	if c.dryRun != false {
		t.Error("expected dryRun to be false")
	}
}

func TestClient_GetTenantAccessToken_DryRun(t *testing.T) {
	c := NewClient("id", "secret", "token", true)

	token, err := c.getTenantAccessToken()
	if err != nil {
		t.Fatalf("getTenantAccessToken() error = %v", err)
	}
	if token != "dry_run_token" {
		t.Errorf("token = %q, want %q", token, "dry_run_token")
	}

	// Second call should return cached token
	token2, err := c.getTenantAccessToken()
	if err != nil {
		t.Fatalf("second getTenantAccessToken() error = %v", err)
	}
	if token2 != "dry_run_token" {
		t.Errorf("second token = %q, want %q", token2, "dry_run_token")
	}
}

func TestClient_ListRecords_DryRun(t *testing.T) {
	c := NewClient("id", "secret", "token", true)

	records, err := c.ListRecords("table-1", 100, nil)
	if err != nil {
		t.Fatalf("ListRecords() error = %v", err)
	}
	if len(records) != 0 {
		t.Errorf("records = %v, want empty", records)
	}
}

func TestClient_ListRecords_DryRun_WithFilter(t *testing.T) {
	c := NewClient("id", "secret", "token", true)

	records, err := c.ListRecords("table-1", 50, map[string]any{"field": "value"})
	if err != nil {
		t.Fatalf("ListRecords() error = %v", err)
	}
	if len(records) != 0 {
		t.Errorf("records = %v, want empty", records)
	}
}

func TestClient_UpsertByKey_DryRun(t *testing.T) {
	c := NewClient("id", "secret", "token", true)

	records := []map[string]any{
		{"name": "alice", "role": "admin"},
		{"name": "bob", "role": "user"},
	}

	created, updated, err := c.UpsertByKey("table-1", records, "name")
	if err != nil {
		t.Fatalf("UpsertByKey() error = %v", err)
	}
	if len(created) != 2 {
		t.Errorf("created = %d, want 2", len(created))
	}
	if len(updated) != 0 {
		t.Errorf("updated = %d, want 0", len(updated))
	}
}

func TestClient_UpsertByKey_DryRun_EmptyRecords(t *testing.T) {
	c := NewClient("id", "secret", "token", true)

	created, updated, err := c.UpsertByKey("table-1", []map[string]any{}, "name")
	if err != nil {
		t.Fatalf("UpsertByKey() error = %v", err)
	}
	if len(created) != 0 {
		t.Errorf("created = %d, want 0", len(created))
	}
	if len(updated) != 0 {
		t.Errorf("updated = %d, want 0", len(updated))
	}
}

func TestClient_SendMessage_DryRun(t *testing.T) {
	c := NewClient("id", "secret", "token", true)

	msgID, err := c.SendMessage("chat-1", "hello world", false)
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
	if msgID == "" {
		t.Fatal("expected non-empty message ID")
	}
	if len(msgID) <= len("dry_run_message_") {
		t.Errorf("message ID = %q, expected longer format", msgID)
	}
}

func TestClient_SendMessage_DryRun_WithDryRunParam(t *testing.T) {
	c := NewClient("id", "secret", "token", false) // dryRun = false at client level

	// But pass dryRun = true at call level
	msgID, err := c.SendMessage("chat-1", "hello", true)
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
	if msgID == "" {
		t.Fatal("expected non-empty message ID")
	}
}

func TestClient_SendMessage_DryRun_ClientDryRunDominates(t *testing.T) {
	c := NewClient("id", "secret", "token", true) // dryRun = true
	// Even passing dryRun=false at call level, client-level dryRun should dominate
	msgID, err := c.SendMessage("chat-1", "hello", false)
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
	if msgID == "" {
		t.Fatal("expected non-empty message ID")
	}
}

func TestBatchToFields(t *testing.T) {
	records := []map[string]any{
		{"name": "alice"},
		{"name": "bob"},
	}

	result := batchToFields(records)
	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}
	for i, r := range result {
		fields, ok := r["fields"].(map[string]any)
		if !ok {
			t.Errorf("result[%d][\"fields\"] is not a map", i)
			continue
		}
		if fields["name"] == nil {
			t.Errorf("result[%d][\"fields\"][\"name\"] is nil", i)
		}
	}
}

func TestBatchToFields_Empty(t *testing.T) {
	result := batchToFields([]map[string]any{})
	if len(result) != 0 {
		t.Errorf("len(result) = %d, want 0", len(result))
	}
}
