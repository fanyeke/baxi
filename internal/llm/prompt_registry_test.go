package llm

import (
	"strings"
	"testing"
)

func TestPromptRegistryLoad(t *testing.T) {
	reg, err := NewPromptRegistry()
	if err != nil {
		t.Fatalf("NewPromptRegistry() failed: %v", err)
	}

	tmpl, err := reg.Load("decision_support")
	if err != nil {
		t.Fatalf("Load(decision_support) failed: %v", err)
	}

	if tmpl.ID != "decision_support" {
		t.Errorf("expected ID=decision_support, got %q", tmpl.ID)
	}
	if tmpl.Version != "v1" {
		t.Errorf("expected Version=v1, got %q", tmpl.Version)
	}
	if tmpl.SystemPrompt == "" {
		t.Errorf("expected non-empty SystemPrompt")
	}
	if tmpl.UserTemplate == "" {
		t.Errorf("expected non-empty UserTemplate")
	}
	if tmpl.Hash == "" {
		t.Errorf("expected non-empty Hash")
	}
}

func TestPromptRegistryLoad_NotFound(t *testing.T) {
	reg, err := NewPromptRegistry()
	if err != nil {
		t.Fatalf("NewPromptRegistry() failed: %v", err)
	}

	_, err = reg.Load("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent prompt, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected error containing 'not found', got %q", err.Error())
	}
}

func TestPromptHash(t *testing.T) {
	reg, err := NewPromptRegistry()
	if err != nil {
		t.Fatalf("NewPromptRegistry() failed: %v", err)
	}

	hash, err := reg.Hash("decision_support")
	if err != nil {
		t.Fatalf("Hash(decision_support) failed: %v", err)
	}

	if len(hash) != 64 {
		t.Errorf("expected 64-char SHA-256 hex, got %d chars: %s", len(hash), hash)
	}

	for _, c := range hash {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("invalid hex character %q in hash %s", c, hash)
		}
	}
}

func TestPromptHash_NotFound(t *testing.T) {
	reg, err := NewPromptRegistry()
	if err != nil {
		t.Fatalf("NewPromptRegistry() failed: %v", err)
	}

	_, err = reg.Hash("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent prompt, got nil")
	}
}

func TestPromptRender(t *testing.T) {
	reg, err := NewPromptRegistry()
	if err != nil {
		t.Fatalf("NewPromptRegistry() failed: %v", err)
	}

	data := UserPromptData{
		ContextJSON:      `{"case_id":"test-123","metric":"revenue"}`,
		AllowedActions:   []string{"notify_owner", "create_followup_task"},
		ForbiddenActions: []string{"delete_data", "modify_pricing"},
	}

	output, err := reg.RenderUserPrompt("decision_support", data)
	if err != nil {
		t.Fatalf("RenderUserPrompt() failed: %v", err)
	}

	if !strings.Contains(output, data.ContextJSON) {
		t.Errorf("rendered output should contain ContextJSON")
	}
	for _, action := range data.AllowedActions {
		if !strings.Contains(output, action) {
			t.Errorf("rendered output should contain allowed action %q", action)
		}
	}
	for _, action := range data.ForbiddenActions {
		if !strings.Contains(output, action) {
			t.Errorf("rendered output should contain forbidden action %q", action)
		}
	}
}

func TestPromptList(t *testing.T) {
	reg, err := NewPromptRegistry()
	if err != nil {
		t.Fatalf("NewPromptRegistry() failed: %v", err)
	}

	ids := reg.List()
	if len(ids) == 0 {
		t.Fatal("expected non-empty List()")
	}

	found := false
	for _, id := range ids {
		if id == "decision_support" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected List() to contain 'decision_support', got %v", ids)
	}
}

func TestSystemPromptContainsSchema(t *testing.T) {
	reg, err := NewPromptRegistry()
	if err != nil {
		t.Fatalf("NewPromptRegistry() failed: %v", err)
	}

	tmpl, _ := reg.Load("decision_support")
	if !strings.Contains(tmpl.SystemPrompt, "decision_type") {
		t.Errorf("system prompt should contain JSON schema with decision_type")
	}
	if !strings.Contains(tmpl.SystemPrompt, "allowed_actions") {
		t.Errorf("system prompt should reference allowed_actions")
	}
	if !strings.Contains(tmpl.SystemPrompt, "requires_human_review") {
		t.Errorf("system prompt should require human_review")
	}
}

func TestUserTemplateIsTemplate(t *testing.T) {
	reg, err := NewPromptRegistry()
	if err != nil {
		t.Fatalf("NewPromptRegistry() failed: %v", err)
	}

	tmpl, _ := reg.Load("decision_support")
	if !strings.Contains(tmpl.UserTemplate, "{{.") {
		t.Errorf("user template should contain Go template syntax {{.}}")
	}
	if !strings.Contains(tmpl.UserTemplate, "{{range") {
		t.Errorf("user template should contain {{range}} for iterating actions")
	}
}

func TestNewPromptRegistry_LoadsBothFiles(t *testing.T) {
	reg, err := NewPromptRegistry()
	if err != nil {
		t.Fatalf("NewPromptRegistry() failed: %v", err)
	}

	ids := reg.List()
	if len(ids) != 1 {
		t.Errorf("expected exactly 1 prompt ID, got %d: %v", len(ids), ids)
	}
}
