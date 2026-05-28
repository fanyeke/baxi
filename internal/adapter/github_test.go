package adapter

import (
	"context"
	"strings"
	"testing"

	"baxi/internal/action"
)

func TestGitHubAdapter_BuildLabels_FullPayload(t *testing.T) {
	a := NewGitHubAdapter(GitHubConfig{Token: "tkn", Repo: "o/r"})
	payload := map[string]interface{}{
		"severity":   "high",
		"owner_role": "platform",
	}
	labels := a.BuildLabels(payload)
	want := []string{"alert", "high", "platform"}
	if len(labels) != len(want) {
		t.Fatalf("expected %v, got %v", want, labels)
	}
	for i := range want {
		if labels[i] != want[i] {
			t.Errorf("labels[%d] = %q, want %q", i, labels[i], want[i])
		}
	}
}

func TestGitHubAdapter_BuildLabels_Defaults(t *testing.T) {
	a := NewGitHubAdapter(GitHubConfig{})
	labels := a.BuildLabels(nil)
	want := []string{"alert", "medium", "unassigned"}
	if len(labels) != len(want) {
		t.Fatalf("expected %v, got %v", want, labels)
	}
	for i := range want {
		if labels[i] != want[i] {
			t.Errorf("labels[%d] = %q, want %q", i, labels[i], want[i])
		}
	}
}

func TestGitHubAdapter_BuildLabels_EmptyPayload(t *testing.T) {
	a := NewGitHubAdapter(GitHubConfig{})
	labels := a.BuildLabels(map[string]interface{}{})
	want := []string{"alert", "medium", "unassigned"}
	if len(labels) != len(want) {
		t.Fatalf("expected %v, got %v", want, labels)
	}
}

func TestGitHubAdapter_BuildLabels_Deduplicates(t *testing.T) {
	a := NewGitHubAdapter(GitHubConfig{})
	payload := map[string]interface{}{
		"severity":   "alert",
		"owner_role": "alert",
	}
	labels := a.BuildLabels(payload)
	if len(labels) != 1 {
		t.Fatalf("expected 1 unique label, got %d: %v", len(labels), labels)
	}
	if labels[0] != "alert" {
		t.Errorf("unexpected labels: %v", labels)
	}
}

func TestGitHubAdapter_BuildLabels_TrimsWhitespace(t *testing.T) {
	a := NewGitHubAdapter(GitHubConfig{})
	payload := map[string]interface{}{
		"severity":   "  high  ",
		"owner_role": "",
	}
	labels := a.BuildLabels(payload)
	if len(labels) != 2 {
		t.Fatalf("expected 2 labels, got %d: %v", len(labels), labels)
	}
	if labels[0] != "alert" || labels[1] != "high" {
		t.Errorf("unexpected labels after trim: %v", labels)
	}
}

func TestGitHubAdapter_BuildIssue_FullPayload(t *testing.T) {
	a := NewGitHubAdapter(GitHubConfig{Token: "tkn", Repo: "o/r"})
	proposal := action.ActionProposal{
		ProposalID: "prop-001",
		CaseID:     "case-001",
		ActionType: "create_followup_task",
		Payload: map[string]interface{}{
			"rule_id":       "rule-42",
			"metric_name":   "gmv",
			"current_value": "1200",
			"baseline_value": "1000",
			"change_rate":   0.20,
			"severity":      "high",
			"owner_role":    "platform",
		},
	}
	issue := a.BuildIssue(proposal)
	wantTitle := "[Alert] rule-42: gmv anomaly detected"
	if issue.Title != wantTitle {
		t.Errorf("title = %q, want %q", issue.Title, wantTitle)
	}
	if !strings.Contains(issue.Body, "Rule**: rule-42") {
		t.Errorf("body missing rule_id: %q", issue.Body)
	}
	if !strings.Contains(issue.Body, "Metric**: gmv") {
		t.Errorf("body missing metric_name: %q", issue.Body)
	}
	if !strings.Contains(issue.Body, "Current Value**: 1200") {
		t.Errorf("body missing current_value: %q", issue.Body)
	}
	if !strings.Contains(issue.Body, "Baseline**: 1000") {
		t.Errorf("body missing baseline_value: %q", issue.Body)
	}
	if !strings.Contains(issue.Body, "Change**: 20.0%") {
		t.Errorf("body missing change_rate: %q", issue.Body)
	}
	if !strings.Contains(issue.Body, "Severity**: high") {
		t.Errorf("body missing severity: %q", issue.Body)
	}
	if !strings.Contains(issue.Body, "Owner**: platform") {
		t.Errorf("body missing owner_role: %q", issue.Body)
	}
	if len(issue.Labels) != 3 {
		t.Errorf("expected 3 labels, got %d: %v", len(issue.Labels), issue.Labels)
	}
}

func TestGitHubAdapter_BuildIssue_NilPayload(t *testing.T) {
	a := NewGitHubAdapter(GitHubConfig{})
	proposal := action.ActionProposal{
		ProposalID: "prop-002",
		CaseID:     "case-002",
		Payload:    nil,
	}
	issue := a.BuildIssue(proposal)
	wantTitle := "[Alert] unknown: metric anomaly detected"
	if issue.Title != wantTitle {
		t.Errorf("title = %q, want %q", issue.Title, wantTitle)
	}
	if !strings.Contains(issue.Body, "Current Value**: N/A") {
		t.Errorf("body missing default current: %q", issue.Body)
	}
	if !strings.Contains(issue.Body, "Change**: 0.0%") {
		t.Errorf("body missing default change: %q", issue.Body)
	}
	if len(issue.Labels) != 3 || issue.Labels[1] != "medium" {
		t.Errorf("expected default severity label, got %v", issue.Labels)
	}
}

func TestGitHubAdapter_BuildIssue_EmptyPayload(t *testing.T) {
	a := NewGitHubAdapter(GitHubConfig{})
	proposal := action.ActionProposal{
		Payload: map[string]interface{}{},
	}
	issue := a.BuildIssue(proposal)
	if issue.Title == "" {
		t.Error("expected non-empty title")
	}
	if issue.Body == "" {
		t.Error("expected non-empty body")
	}
}

func TestGitHubAdapter_BuildIssue_IntegerChangeRate(t *testing.T) {
	a := NewGitHubAdapter(GitHubConfig{})
	proposal := action.ActionProposal{
		Payload: map[string]interface{}{
			"change_rate": 15,
		},
	}
	issue := a.BuildIssue(proposal)
	if !strings.Contains(issue.Body, "Change**: 1500.0%") {
		t.Errorf("expected int change_rate to be treated as 1500%%: %q", issue.Body)
	}
}

func TestGitHubAdapter_BuildIssue_Int64ChangeRate(t *testing.T) {
	a := NewGitHubAdapter(GitHubConfig{})
	proposal := action.ActionProposal{
		Payload: map[string]interface{}{
			"change_rate": int64(5),
		},
	}
	issue := a.BuildIssue(proposal)
	if !strings.Contains(issue.Body, "Change**: 500.0%") {
		t.Errorf("expected int64 change_rate to be treated as 500%%: %q", issue.Body)
	}
}

func TestGitHubAdapter_BuildComment(t *testing.T) {
	a := NewGitHubAdapter(GitHubConfig{})
	proposal := action.ActionProposal{
		ProposalID: "prop-003",
		CaseID:     "case-003",
		ActionType: "create_followup_task",
		Payload: map[string]interface{}{
			"severity": "critical",
		},
	}
	comment := a.BuildComment(proposal)
	if !strings.Contains(comment.Body, "prop-003") {
		t.Errorf("comment missing proposal_id: %q", comment.Body)
	}
	if !strings.Contains(comment.Body, "case-003") {
		t.Errorf("comment missing case_id: %q", comment.Body)
	}
	if !strings.Contains(comment.Body, "create_followup_task") {
		t.Errorf("comment missing action_type: %q", comment.Body)
	}
	if !strings.Contains(comment.Body, "critical") {
		t.Errorf("comment missing severity: %q", comment.Body)
	}
}

func TestGitHubAdapter_BuildComment_NilPayload(t *testing.T) {
	a := NewGitHubAdapter(GitHubConfig{})
	proposal := action.ActionProposal{
		ProposalID: "prop-004",
		CaseID:     "case-004",
		ActionType: "create_followup_task",
		Payload:    nil,
	}
	comment := a.BuildComment(proposal)
	if comment.Body == "" {
		t.Error("expected non-empty comment body")
	}
	if !strings.Contains(comment.Body, "medium") {
		t.Errorf("expected default severity medium: %q", comment.Body)
	}
}

func TestGitHubAdapter_CreateIssue_Success(t *testing.T) {
	a := NewGitHubAdapter(GitHubConfig{Token: "ghp_token", Repo: "owner/repo"})
	issue := GitHubIssue{
		Title:  "Test Issue",
		Body:   "Test body",
		Labels: []string{"alert", "high"},
	}
	url, err := a.CreateIssue(context.Background(), issue)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "https://github.com/owner/repo/issues/preview"
	if url != want {
		t.Errorf("url = %q, want %q", url, want)
	}
}

func TestGitHubAdapter_CreateIssue_EmptyToken(t *testing.T) {
	a := NewGitHubAdapter(GitHubConfig{Repo: "owner/repo"})
	_, err := a.CreateIssue(context.Background(), GitHubIssue{Title: "t"})
	if err == nil {
		t.Fatal("expected error for empty token")
	}
	if err.Error() != "github token not configured" {
		t.Errorf("unexpected error: %q", err.Error())
	}
}

func TestGitHubAdapter_CreateIssue_EmptyRepo(t *testing.T) {
	a := NewGitHubAdapter(GitHubConfig{Token: "tkn"})
	_, err := a.CreateIssue(context.Background(), GitHubIssue{Title: "t"})
	if err == nil {
		t.Fatal("expected error for empty repo")
	}
	if err.Error() != "github repo not configured" {
		t.Errorf("unexpected error: %q", err.Error())
	}
}

func TestGitHubAdapter_AddLabels_Success(t *testing.T) {
	a := NewGitHubAdapter(GitHubConfig{Token: "tkn", Repo: "o/r"})
	err := a.AddLabels(context.Background(), 42, []string{"bug", "urgent"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGitHubAdapter_AddLabels_EmptyToken(t *testing.T) {
	a := NewGitHubAdapter(GitHubConfig{Repo: "o/r"})
	err := a.AddLabels(context.Background(), 1, []string{"bug"})
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestGitHubAdapter_AddLabels_EmptyRepo(t *testing.T) {
	a := NewGitHubAdapter(GitHubConfig{Token: "tkn"})
	err := a.AddLabels(context.Background(), 1, []string{"bug"})
	if err == nil {
		t.Fatal("expected error for empty repo")
	}
}

func TestGitHubAdapter_AddComment_Success(t *testing.T) {
	a := NewGitHubAdapter(GitHubConfig{Token: "tkn", Repo: "o/r"})
	comment := GitHubComment{Body: "Follow-up comment"}
	err := a.AddComment(context.Background(), 99, comment)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGitHubAdapter_AddComment_EmptyToken(t *testing.T) {
	a := NewGitHubAdapter(GitHubConfig{Repo: "o/r"})
	err := a.AddComment(context.Background(), 1, GitHubComment{Body: "x"})
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestGitHubAdapter_AddComment_EmptyRepo(t *testing.T) {
	a := NewGitHubAdapter(GitHubConfig{Token: "tkn"})
	err := a.AddComment(context.Background(), 1, GitHubComment{Body: "x"})
	if err == nil {
		t.Fatal("expected error for empty repo")
	}
}

func TestGitHubAdapter_Execute_DryRun(t *testing.T) {
	ctx := context.Background()
	a := NewGitHubAdapter(GitHubConfig{Token: "tkn", Repo: "o/r"})
	proposal := action.ActionProposal{
		ProposalID: "prop-005",
		CaseID:     "case-005",
		ActionType: "create_followup_task",
		Title:      "Follow-up task",
		Payload: map[string]interface{}{
			"rule_id":     "rule-99",
			"metric_name": "orders",
			"severity":    "low",
		},
	}
	result, err := a.Execute(ctx, proposal, true)
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
		t.Fatal("expected DispatchPayload")
	}
	if ch, ok := result.DispatchPayload["channel"]; !ok || ch != "github" {
		t.Errorf("expected channel=github, got %v", ch)
	}
	issueRaw, ok := result.DispatchPayload["issue"]
	if !ok {
		t.Fatal("expected issue in payload")
	}
	issue, ok := issueRaw.(map[string]interface{})
	if !ok {
		t.Fatalf("expected issue to be map, got %T", issueRaw)
	}
	if title, ok := issue["title"].(string); !ok || !strings.Contains(title, "rule-99") {
		t.Errorf("expected title to contain rule-99, got %q", title)
	}
	if labels, ok := issue["labels"].([]string); !ok || len(labels) == 0 {
		t.Errorf("expected non-empty labels, got %v", issue["labels"])
	}
}

func TestGitHubAdapter_Execute_EmptyToken(t *testing.T) {
	ctx := context.Background()
	a := NewGitHubAdapter(GitHubConfig{Repo: "o/r"})
	proposal := action.ActionProposal{
		ProposalID: "prop-006",
		CaseID:     "case-006",
		ActionType: "create_followup_task",
	}
	result, err := a.Execute(ctx, proposal, false)
	if err == nil {
		t.Fatal("expected error for empty token")
	}
	if err.Error() != "github token not configured" {
		t.Errorf("unexpected error: %q", err.Error())
	}
	if result.Success {
		t.Error("expected Success=false")
	}
}

func TestGitHubAdapter_Execute_Success(t *testing.T) {
	ctx := context.Background()
	a := NewGitHubAdapter(GitHubConfig{Token: "ghp_valid", Repo: "my-org/my-repo"})
	proposal := action.ActionProposal{
		ProposalID: "prop-007",
		CaseID:     "case-007",
		ActionType: "create_followup_task",
		Title:      "Create follow-up GitHub issue",
		Payload: map[string]interface{}{
			"rule_id":       "rule-1",
			"metric_name":   "revenue",
			"current_value": "5000",
			"baseline_value": "4500",
			"change_rate":   0.111,
			"severity":      "medium",
			"owner_role":    "revenue-team",
		},
	}
	result, err := a.Execute(ctx, proposal, false)
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
		t.Fatal("expected DispatchPayload")
	}
	if ch, ok := result.DispatchPayload["channel"]; !ok || ch != "github" {
		t.Errorf("expected channel=github, got %v", ch)
	}
	issueRaw, ok := result.DispatchPayload["issue"]
	if !ok {
		t.Fatal("expected issue in payload")
	}
	issue, ok := issueRaw.(map[string]interface{})
	if !ok {
		t.Fatalf("expected issue to be map, got %T", issueRaw)
	}
	if body, ok := issue["body"].(string); !ok || !strings.Contains(body, "revenue") {
		t.Errorf("expected body to contain revenue, got %q", body)
	}
	if labels, ok := issue["labels"].([]string); !ok || len(labels) != 3 {
		t.Errorf("expected 3 labels, got %v", labels)
	}
}

func TestGitHubAdapter_Execute_NilPayload(t *testing.T) {
	ctx := context.Background()
	a := NewGitHubAdapter(GitHubConfig{Token: "tkn", Repo: "o/r"})
	proposal := action.ActionProposal{
		ProposalID: "prop-008",
		CaseID:     "case-008",
		ActionType: "create_followup_task",
		Payload:    nil,
	}
	result, err := a.Execute(ctx, proposal, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	issueRaw, ok := result.DispatchPayload["issue"]
	if !ok {
		t.Fatal("expected issue in payload")
	}
	issue, ok := issueRaw.(map[string]interface{})
	if !ok {
		t.Fatalf("expected issue to be map, got %T", issueRaw)
	}
	if title, ok := issue["title"].(string); !ok || title == "" {
		t.Errorf("expected non-empty title, got %q", title)
	}
}

func TestGitHubAdapter_getString(t *testing.T) {
	m := map[string]interface{}{
		"key": "value",
	}
	if got := getString(m, "key", "def"); got != "value" {
		t.Errorf("getString(m, key) = %q, want value", got)
	}
	if got := getString(m, "missing", "def"); got != "def" {
		t.Errorf("getString(m, missing) = %q, want def", got)
	}
	if got := getString(nil, "key", "def"); got != "def" {
		t.Errorf("getString(nil, key) = %q, want def", got)
	}
	if got := getString(m, "wrong_type", "def"); got != "def" {
		t.Errorf("getString(m, wrong_type) = %q, want def", got)
	}
}

func TestGitHubAdapter_getFloat(t *testing.T) {
	m := map[string]interface{}{
		"f": 1.5,
		"i": 10,
		"i64": int64(20),
	}
	if got := getFloat(m, "f", 0); got != 1.5 {
		t.Errorf("getFloat(f) = %v, want 1.5", got)
	}
	if got := getFloat(m, "i", 0); got != 10.0 {
		t.Errorf("getFloat(i) = %v, want 10", got)
	}
	if got := getFloat(m, "i64", 0); got != 20.0 {
		t.Errorf("getFloat(i64) = %v, want 20", got)
	}
	if got := getFloat(m, "missing", 99.0); got != 99.0 {
		t.Errorf("getFloat(missing) = %v, want 99", got)
	}
	if got := getFloat(nil, "key", 42.0); got != 42.0 {
		t.Errorf("getFloat(nil) = %v, want 42", got)
	}
}
