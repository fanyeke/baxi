package adapter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"baxi/internal/action"
)

func TestFeishuAdapter_DryRunTrue(t *testing.T) {
	ctx := context.Background()
	adapter := NewFeishuAdapter(FeishuConfig{
		WebhookURL: "https://hooks.feishu.cn/webhook/test",
		Enabled:    true,
	})

	proposal := action.ActionProposal{
		ProposalID: "prop-001",
		CaseID:     "case-001",
		ActionType: "notify_owner",
	}

	result, err := adapter.Execute(ctx, proposal, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
	if !result.DryRun {
		t.Error("expected DryRun=true")
	}
	if result.DispatchPayload == nil {
		t.Fatal("expected DispatchPayload to be non-nil")
	}
	if ch, ok := result.DispatchPayload["channel"]; !ok || ch != "feishu" {
		t.Errorf("expected channel=feishu, got %v", ch)
	}
}

func TestFeishuAdapter_EmptyWebhook(t *testing.T) {
	ctx := context.Background()
	adapter := NewFeishuAdapter(FeishuConfig{
		WebhookURL: "",
	})

	proposal := action.ActionProposal{
		ProposalID: "prop-002",
		CaseID:     "case-002",
		ActionType: "export_report",
	}

	result, err := adapter.Execute(ctx, proposal, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Fatal("expected Success=false for unconfigured adapter")
	}
	if result.Error != "no chat_id configured (set FEISHU_CHAT_ID in env or feishu_app.yml)" {
		t.Errorf("unexpected error message: %q", result.Error)
	}
}

func TestFeishuAdapter_ExecuteSuccess(t *testing.T) {
	ctx := context.Background()
	adapter := NewFeishuAdapterWithClient(FeishuConfig{
		WebhookURL: "https://hooks.feishu.cn/webhook/valid",
		Enabled:    true,
		ChatID:     "oc_test_chat_id",
	}, &mockFeishuClient{})

	proposal := action.ActionProposal{
		ProposalID: "prop-003",
		CaseID:     "case-003",
		ActionType: "create_outbox_message",
		Title:      "Test outbox message",
	}

	result, err := adapter.Execute(ctx, proposal, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
	if result.DryRun {
		t.Error("expected DryRun=false")
	}
	if result.DispatchPayload == nil {
		t.Fatal("expected DispatchPayload to be non-nil")
	}
	if ch, ok := result.DispatchPayload["channel"]; !ok || ch != "feishu" {
		t.Errorf("expected channel=feishu, got %v", ch)
	}
}

func TestGitHubAdapter_DryRunTrue(t *testing.T) {
	ctx := context.Background()
	adapter := NewGitHubAdapter(GitHubConfig{
		Token: "ghp_test_token",
		Repo:  "owner/repo",
	})

	proposal := action.ActionProposal{
		ProposalID: "prop-004",
		CaseID:     "case-004",
		ActionType: "create_followup_task",
	}

	result, err := adapter.Execute(ctx, proposal, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
	if !result.DryRun {
		t.Error("expected DryRun=true")
	}
	if result.DispatchPayload == nil {
		t.Fatal("expected DispatchPayload to be non-nil")
	}
	if ch, ok := result.DispatchPayload["channel"]; !ok || ch != "github" {
		t.Errorf("expected channel=github, got %v", ch)
	}
}

func TestGitHubAdapter_EmptyToken(t *testing.T) {
	ctx := context.Background()
	adapter := NewGitHubAdapter(GitHubConfig{
		Token: "",
	})

	proposal := action.ActionProposal{
		ProposalID: "prop-005",
		CaseID:     "case-005",
		ActionType: "create_followup_task",
	}

	result, err := adapter.Execute(ctx, proposal, false)
	if err == nil {
		t.Fatal("expected error for empty token")
	}
	if err.Error() != "github token not configured" {
		t.Errorf("unexpected error message: %q", err.Error())
	}
	if result.Success {
		t.Error("expected Success=false on error")
	}
}

func TestGitHubAdapter_ExecuteSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"html_url": "https://github.com/my-org/my-repo/issues/1",
			"number":   1,
		})
	}))
	defer server.Close()

	ctx := context.Background()
	adapter := NewGitHubAdapter(GitHubConfig{
		Token: "ghp_valid_token",
		Repo:  "my-org/my-repo",
	})
	adapter.baseURL = server.URL

	proposal := action.ActionProposal{
		ProposalID: "prop-006",
		CaseID:     "case-006",
		ActionType: "create_followup_task",
		Title:      "Create follow-up GitHub issue",
	}

	result, err := adapter.Execute(ctx, proposal, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
	if result.DryRun {
		t.Error("expected DryRun=false")
	}
	if result.DispatchPayload == nil {
		t.Fatal("expected DispatchPayload to be non-nil")
	}
	if ch, ok := result.DispatchPayload["channel"]; !ok || ch != "github" {
		t.Errorf("expected channel=github, got %v", ch)
	}
}

func TestActionChannel_Mapping(t *testing.T) {
	tests := []struct {
		actionType string
		want       string
	}{
		{"export_report", "feishu"},
		{"notify_owner", "feishu"},
		{"create_followup_task", "github"},
		{"create_outbox_message", "feishu"},
		{"unknown_action", "unknown"},
		{"", "unknown"},
	}

	for _, tt := range tests {
		got := ActionChannel(tt.actionType)
		if got != tt.want {
			t.Errorf("ActionChannel(%q) = %q, want %q", tt.actionType, got, tt.want)
		}
	}
}
