package adapter

import (
	"context"
	"testing"

	"baxi/internal/action"
)

// ──── getString ────────────────────────────────────────────────────────────

func TestGetString_NilMap(t *testing.T) {
	got := getString(nil, "key", "default")
	if got != "default" {
		t.Errorf("getString(nil, ...) = %q, want %q", got, "default")
	}
}

func TestGetString_EmptyMap(t *testing.T) {
	got := getString(map[string]interface{}{}, "key", "default")
	if got != "default" {
		t.Errorf("getString(empty, ...) = %q, want %q", got, "default")
	}
}

func TestGetString_NonStringValue(t *testing.T) {
	m := map[string]interface{}{"key": 42}
	got := getString(m, "key", "default")
	if got != "default" {
		t.Errorf("getString(int value) = %q, want %q", got, "default")
	}
}

func TestGetString_EmptyStringValue(t *testing.T) {
	m := map[string]interface{}{"key": ""}
	got := getString(m, "key", "default")
	if got != "" {
		t.Errorf("getString(empty string) = %q, want %q", got, "")
	}
}

func TestGetString_ValidString(t *testing.T) {
	m := map[string]interface{}{"key": "value"}
	got := getString(m, "key", "default")
	if got != "value" {
		t.Errorf("getString(valid) = %q, want %q", got, "value")
	}
}

// ──── getFloat ─────────────────────────────────────────────────────────────

func TestGetFloat_NilMap(t *testing.T) {
	got := getFloat(nil, "key", 3.14)
	if got != 3.14 {
		t.Errorf("getFloat(nil, ...) = %f, want %f", got, 3.14)
	}
}

func TestGetFloat_EmptyMap(t *testing.T) {
	got := getFloat(map[string]interface{}{}, "key", 3.14)
	if got != 3.14 {
		t.Errorf("getFloat(empty, ...) = %f, want %f", got, 3.14)
	}
}

func TestGetFloat_Float64Value(t *testing.T) {
	m := map[string]interface{}{"key": 2.718}
	got := getFloat(m, "key", 0)
	if got != 2.718 {
		t.Errorf("getFloat(float64) = %f, want %f", got, 2.718)
	}
}

func TestGetFloat_IntValue(t *testing.T) {
	m := map[string]interface{}{"key": 42}
	got := getFloat(m, "key", 0)
	if got != 42.0 {
		t.Errorf("getFloat(int) = %f, want %f", got, 42.0)
	}
}

func TestGetFloat_Int64Value(t *testing.T) {
	m := map[string]interface{}{"key": int64(100)}
	got := getFloat(m, "key", 0)
	if got != 100.0 {
		t.Errorf("getFloat(int64) = %f, want %f", got, 100.0)
	}
}

func TestGetFloat_StringValue(t *testing.T) {
	m := map[string]interface{}{"key": "not_a_number"}
	got := getFloat(m, "key", 5.0)
	if got != 5.0 {
		t.Errorf("getFloat(string) = %f, want %f", got, 5.0)
	}
}

// ──── CLIAdapter ───────────────────────────────────────────────────────────

func TestCLIAdapter_DryRun_Extra(t *testing.T) {
	adapter := NewCLIAdapter(CLIConfig{LogPath: "/tmp/test.log", Enabled: true})
	proposal := action.ActionProposal{
		ProposalID: "p1",
		CaseID:     "c1",
		ActionType: "export_report",
		Payload:    map[string]interface{}{"rule_id": "rule-123"},
	}

	result, err := adapter.Execute(context.Background(), proposal, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
	if !result.DryRun {
		t.Error("expected DryRun=true")
	}
}

func TestCLIAdapter_EmptyLogPath_Extra(t *testing.T) {
	adapter := NewCLIAdapter(CLIConfig{LogPath: "", Enabled: true})
	proposal := action.ActionProposal{
		ProposalID: "p2",
		CaseID:     "c2",
		ActionType: "export_report",
	}

	_, err := adapter.Execute(context.Background(), proposal, false)
	if err == nil {
		t.Fatal("expected error for empty log path")
	}
}

func TestCLIAdapter_NilPayload_Extra(t *testing.T) {
	adapter := NewCLIAdapter(CLIConfig{LogPath: "", Enabled: true})
	proposal := action.ActionProposal{
		ProposalID: "p3",
		CaseID:     "c3",
		ActionType: "export_report",
		Payload:    nil,
	}

	result, _ := adapter.Execute(context.Background(), proposal, true)
	if !result.Success {
		t.Error("expected success in dry run even with nil payload")
	}
	if ch, ok := result.DispatchPayload["channel"]; !ok || ch != "feishu" {
		t.Errorf("expected channel=feishu, got %v", ch)
	}
}

func TestCLIAdapter_InvalidRuleID_Extra(t *testing.T) {
	adapter := NewCLIAdapter(CLIConfig{LogPath: "", Enabled: true})
	proposal := action.ActionProposal{
		ProposalID: "p4",
		CaseID:     "c4",
		ActionType: "export_report",
		Payload:    map[string]interface{}{"rule_id": "invalid id with spaces!"},
	}

	result, _ := adapter.Execute(context.Background(), proposal, true)
	ruleID, ok := result.DispatchPayload["rule_id"]
	if !ok || ruleID != "unknown" {
		t.Errorf("expected rule_id=unknown for invalid rule_id, got %v", ruleID)
	}
}

func TestCLIAdapter_LongRuleID_Extra(t *testing.T) {
	adapter := NewCLIAdapter(CLIConfig{LogPath: "", Enabled: true})
	longID := "a"
	for len(longID) < 70 {
		longID += "b"
	}
	proposal := action.ActionProposal{
		ProposalID: "p5",
		CaseID:     "c5",
		ActionType: "export_report",
		Payload:    map[string]interface{}{"rule_id": longID},
	}

	result, _ := adapter.Execute(context.Background(), proposal, true)
	ruleID := result.DispatchPayload["rule_id"]
	if ruleID != "unknown" {
		t.Errorf("expected rule_id=unknown for long rule_id, got %v", ruleID)
	}
}

func TestCLIAdapter_NonStringPayload_Extra(t *testing.T) {
	adapter := NewCLIAdapter(CLIConfig{LogPath: "", Enabled: true})
	proposal := action.ActionProposal{
		ProposalID: "p6",
		CaseID:     "c6",
		ActionType: "export_report",
		Payload:    map[string]interface{}{"rule_id": 42},
	}

	result, _ := adapter.Execute(context.Background(), proposal, true)
	ruleID := result.DispatchPayload["rule_id"]
	if ruleID != "unknown" {
		t.Errorf("expected rule_id=unknown for non-string rule_id, got %v", ruleID)
	}
}

func TestCLIAdapter_ExecuteWithValidLog_Extra(t *testing.T) {
	dir := t.TempDir()
	logPath := dir + "/test_log.csv"
	adapter := NewCLIAdapter(CLIConfig{LogPath: logPath, Enabled: true})
	proposal := action.ActionProposal{
		ProposalID: "p7",
		CaseID:     "c7",
		ActionType: "export_report",
		Payload:    map[string]interface{}{"rule_id": "rule_abc"},
	}

	result, err := adapter.Execute(context.Background(), proposal, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if result.OutboxEventID != logPath {
		t.Errorf("expected OutboxEventID=%s, got %s", logPath, result.OutboxEventID)
	}
}

func TestCLIAdapter_SecondWriteSkipsHeader_Extra(t *testing.T) {
	dir := t.TempDir()
	logPath := dir + "/test_log2.csv"
	adapter := NewCLIAdapter(CLIConfig{LogPath: logPath, Enabled: true})
	proposal := action.ActionProposal{
		ProposalID: "p8",
		CaseID:     "c8",
		ActionType: "export_report",
	}

	_, err := adapter.Execute(context.Background(), proposal, false)
	if err != nil {
		t.Fatalf("first write: %v", err)
	}

	proposal.ProposalID = "p9"
	_, err = adapter.Execute(context.Background(), proposal, false)
	if err != nil {
		t.Fatalf("second write: %v", err)
	}
}

// ──── FeishuAdapter ────────────────────────────────────────────────────────

func TestFeishuAdapter_NilClient_NotDryRun_Extra(t *testing.T) {
	adapter := NewFeishuAdapter(FeishuConfig{
		ChatID: "oc_test",
	})
	proposal := action.ActionProposal{
		ProposalID: "p10",
		CaseID:     "c10",
		ActionType: "export_report",
	}

	result, err := adapter.Execute(context.Background(), proposal, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected Success=false with nil client")
	}
}

func TestFeishuAdapter_FormatMessage_NilPayload_Extra(t *testing.T) {
	adapter := NewFeishuAdapter(FeishuConfig{})
	proposal := action.ActionProposal{
		Payload: nil,
	}

	msg, err := adapter.formatMessage(proposal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == "" {
		t.Error("expected non-empty message")
	}
}

func TestFeishuAdapter_FormatMessage_AllFields_Extra(t *testing.T) {
	adapter := NewFeishuAdapter(FeishuConfig{})
	proposal := action.ActionProposal{
		Payload: map[string]interface{}{
			"rule_id":        "rule_123",
			"metric_name":    "gmv",
			"current_value":  "1000",
			"baseline_value": "1500",
			"severity":       "high",
			"owner_role":     "ops",
		},
	}

	msg, err := adapter.formatMessage(proposal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !containsStr(msg, "rule_123") {
		t.Errorf("expected message to contain rule_id, got: %s", msg)
	}
	if !containsStr(msg, "gmv") {
		t.Errorf("expected message to contain metric_name")
	}
	if !containsStr(msg, "1000") {
		t.Errorf("expected message to contain current_value")
	}
	if !containsStr(msg, "1500") {
		t.Errorf("expected message to contain baseline_value")
	}
	if !containsStr(msg, "high") {
		t.Errorf("expected message to contain severity")
	}
	if !containsStr(msg, "ops") {
		t.Errorf("expected message to contain owner_role")
	}
}

func TestFeishuAdapter_FormatMessage_PartialFields_Extra(t *testing.T) {
	adapter := NewFeishuAdapter(FeishuConfig{})
	proposal := action.ActionProposal{
		Payload: map[string]interface{}{
			"rule_id": "rule_abc",
		},
	}

	msg, err := adapter.formatMessage(proposal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !containsStr(msg, "rule_abc") {
		t.Error("expected message to contain rule_id")
	}
	if containsStr(msg, "Current:") {
		t.Error("message should not contain 'Current:' when no current_value")
	}
}

func TestFeishuAdapter_GetChatID_Empty_Extra(t *testing.T) {
	adapter := NewFeishuAdapter(FeishuConfig{ChatID: ""})
	if adapter.getChatID() != "" {
		t.Error("expected empty chat ID")
	}
}

func TestFeishuAdapter_GetChatID_Set_Extra(t *testing.T) {
	adapter := NewFeishuAdapter(FeishuConfig{ChatID: "oc_test"})
	if adapter.getChatID() != "oc_test" {
		t.Errorf("expected oc_test, got %s", adapter.getChatID())
	}
}

// ──── GitHubAdapter (unique tests only) ────────────────────────────────────

func TestGitHubAdapter_Execute_NoToken_NotDryRun_Extra(t *testing.T) {
	adapter := NewGitHubAdapter(GitHubConfig{Token: "", Repo: "owner/repo"})
	proposal := action.ActionProposal{
		ProposalID: "p13",
		CaseID:     "c13",
		ActionType: "create_followup_task",
		Payload:    map[string]interface{}{"rule_id": "r1"},
	}

	_, err := adapter.Execute(context.Background(), proposal, false)
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

// ──── ManualAdapter ────────────────────────────────────────────────────────

func TestManualAdapter_DryRun_Extra(t *testing.T) {
	adapter := NewManualAdapter(ManualConfig{Enabled: true})
	proposal := action.ActionProposal{
		ProposalID: "p14",
		CaseID:     "c14",
		ActionType: "notify_owner",
		Payload:    map[string]interface{}{"rule_id": "rule_xyz"},
	}

	result, err := adapter.Execute(context.Background(), proposal, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success in dry run")
	}
	if !result.DryRun {
		t.Error("expected DryRun=true")
	}
}

func TestManualAdapter_NormalMode_Extra(t *testing.T) {
	adapter := NewManualAdapter(ManualConfig{Enabled: true})
	proposal := action.ActionProposal{
		ProposalID: "p15",
		CaseID:     "c15",
		ActionType: "notify_owner",
		Payload:    map[string]interface{}{"rule_id": "rule_xyz"},
	}

	result, err := adapter.Execute(context.Background(), proposal, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
}

func TestManualAdapter_NilPayload_Extra(t *testing.T) {
	adapter := NewManualAdapter(ManualConfig{Enabled: true})
	proposal := action.ActionProposal{
		ProposalID: "p16",
		CaseID:     "c16",
		ActionType: "notify_owner",
		Payload:    nil,
	}

	result, err := adapter.Execute(context.Background(), proposal, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ruleID := result.DispatchPayload["rule_id"]
	if ruleID != "unknown" {
		t.Errorf("expected rule_id=unknown for nil payload, got %v", ruleID)
	}
}

// ──── NewFeishuAdapter with client creation ────────────────────────────────

func TestNewFeishuAdapter_WithCredentials(t *testing.T) {
	adapter := NewFeishuAdapter(FeishuConfig{
		AppID:     "app123",
		AppSecret: "secret123",
	})
	if adapter == nil {
		t.Fatal("expected non-nil adapter")
	}
}

func TestNewFeishuAdapter_WithoutCredentials(t *testing.T) {
	adapter := NewFeishuAdapter(FeishuConfig{})
	if adapter == nil {
		t.Fatal("expected non-nil adapter")
	}
}

// ──── NewMockFeishuClient ──────────────────────────────────────────────────

func TestNewMockFeishuClient_Extra(t *testing.T) {
	client := NewMockFeishuClient()
	if client == nil {
		t.Fatal("expected non-nil mock client")
	}
	token, err := client.getTenantAccessToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "mock-token" {
		t.Errorf("expected mock-token, got %s", token)
	}
	msgID, err := client.sendMessage("chat123", "hello", "text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msgID != "mock-msg-id" {
		t.Errorf("expected mock-msg-id, got %s", msgID)
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || findSubstr(s, substr))
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
