package llm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ──── DBAuditLogger with nil pool ───────────────────────────────────────

func TestDBAuditLogger_NilPool_NoPanics(t *testing.T) {
	logger := NewDBAuditLogger(nil)
	ctx := context.Background()

	// All methods should be no-ops with nil pool, not panic
	logger.LogDecisionRequested(ctx, "case-1", "openai", "gpt-4")
	logger.LogDecisionCompleted(ctx, "case-1", "openai", "gpt-4", 100, &TokenUsage{PromptTokens: 10, CompletionTokens: 20, TotalTokens: 30})
	logger.LogDecisionFailed(ctx, "case-1", "openai", "gpt-4", nil)
	logger.LogDecisionValidationFailed(ctx, "case-1", []ValidationError{{Field: "severity", Message: "invalid"}})
	logger.LogFallbackUsed(ctx, "case-1", "timeout")
	logger.LogDecisionReplayed(ctx, "case-1", "original-dec-1")
	logger.LogEvalCompleted(ctx, "case-1", "eval-1")
}

func TestDBAuditLogger_NilPool_NilUsage(t *testing.T) {
	logger := NewDBAuditLogger(nil)
	ctx := context.Background()

	// nil usage should not panic
	logger.LogDecisionCompleted(ctx, "case-1", "openai", "gpt-4", 100, nil)
}

func TestDBAuditLogger_NilPool_NilError(t *testing.T) {
	logger := NewDBAuditLogger(nil)
	ctx := context.Background()

	// nil error should not panic
	logger.LogDecisionFailed(ctx, "case-1", "openai", "gpt-4", nil)
}

func TestDBAuditLogger_NilPool_EmptyValidationErrors(t *testing.T) {
	logger := NewDBAuditLogger(nil)
	ctx := context.Background()

	logger.LogDecisionValidationFailed(ctx, "case-1", []ValidationError{})
}

// ──── NoOpAuditLogger ──────────────────────────────────────────────────

func TestNoOpAuditLogger_AllMethods_Deep(t *testing.T) {
	logger := &NoOpAuditLogger{}
	ctx := context.Background()

	// All methods should be no-ops, not panic
	logger.LogDecisionRequested(ctx, "case-1", "openai", "gpt-4")
	logger.LogDecisionCompleted(ctx, "case-1", "openai", "gpt-4", 100, &TokenUsage{})
	logger.LogDecisionFailed(ctx, "case-1", "openai", "gpt-4", assert.AnError)
	logger.LogDecisionValidationFailed(ctx, "case-1", []ValidationError{{Field: "f", Message: "m"}})
	logger.LogFallbackUsed(ctx, "case-1", "reason")
	logger.LogDecisionReplayed(ctx, "case-1", "orig-1")
	logger.LogEvalCompleted(ctx, "case-1", "eval-1")
}

// ──── NewPromptRegistry ─────────────────────────────────────────────────

func TestNewPromptRegistry(t *testing.T) {
	reg, err := NewPromptRegistry()
	assert.NoError(t, err)
	assert.NotNil(t, reg)

	// Should have at least one prompt loaded
	prompts := reg.List()
	assert.NotEmpty(t, prompts)
}

func TestPromptRegistry_Load(t *testing.T) {
	reg, err := NewPromptRegistry()
	assert.NoError(t, err)

	prompts := reg.List()
	require := assert.New(t)
	require.NotEmpty(prompts)

	tmpl, err := reg.Load(prompts[0])
	assert.NoError(t, err)
	assert.NotNil(t, tmpl)
	assert.NotEmpty(t, tmpl.SystemPrompt)
	assert.NotEmpty(t, tmpl.UserTemplate)
	assert.NotEmpty(t, tmpl.Hash)
}

func TestPromptRegistry_Load_NotFound(t *testing.T) {
	reg, err := NewPromptRegistry()
	assert.NoError(t, err)

	_, err = reg.Load("nonexistent_prompt_id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestPromptRegistry_Hash(t *testing.T) {
	reg, err := NewPromptRegistry()
	assert.NoError(t, err)

	prompts := reg.List()
	require := assert.New(t)
	require.NotEmpty(prompts)

	hash, err := reg.Hash(prompts[0])
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
}

func TestPromptRegistry_Hash_NotFound(t *testing.T) {
	reg, err := NewPromptRegistry()
	assert.NoError(t, err)

	_, err = reg.Hash("nonexistent")
	assert.Error(t, err)
}

func TestPromptRegistry_List(t *testing.T) {
	reg, err := NewPromptRegistry()
	assert.NoError(t, err)

	list := reg.List()
	assert.NotEmpty(t, list)

	// All items should be non-empty strings
	for _, id := range list {
		assert.NotEmpty(t, id)
	}
}

func TestPromptRegistry_RenderUserPrompt(t *testing.T) {
	reg, err := NewPromptRegistry()
	assert.NoError(t, err)

	prompts := reg.List()
	require := assert.New(t)
	require.NotEmpty(prompts)

	// Try rendering with empty data
	result, err := reg.RenderUserPrompt(prompts[0], UserPromptData{
		ContextJSON:      "{}",
		AllowedActions:   []string{"notify_owner"},
		ForbiddenActions: []string{"delete_all"},
	})
	// May or may not succeed depending on template, but should not panic
	if err != nil {
		// If template requires fields, that's ok
		assert.Contains(t, err.Error(), "execute user template")
	} else {
		assert.NotEmpty(t, result)
	}
}

func TestPromptRegistry_RenderUserPrompt_NotFound(t *testing.T) {
	reg, err := NewPromptRegistry()
	assert.NoError(t, err)

	_, err = reg.RenderUserPrompt("nonexistent", UserPromptData{})
	assert.Error(t, err)
}

// ──── RepairPromptRenderer ─────────────────────────────────────────────

func TestNewRepairPromptRenderer(t *testing.T) {
	renderer, err := NewRepairPromptRenderer()
	assert.NoError(t, err)
	assert.NotNil(t, renderer)
}

func TestRepairPromptRenderer_RenderRepairPrompt(t *testing.T) {
	renderer, err := NewRepairPromptRenderer()
	assert.NoError(t, err)

	result, err := renderer.RenderRepairPrompt([]ValidationError{
		{Field: "severity", Message: "must be critical/high/medium/low"},
		{Field: "decision_type", Message: "unknown type"},
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestRepairPromptRenderer_RenderRepairPrompt_EmptyErrors(t *testing.T) {
	renderer, err := NewRepairPromptRenderer()
	assert.NoError(t, err)

	result, err := renderer.RenderRepairPrompt([]ValidationError{})
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

// ──── ValidationError ──────────────────────────────────────────────────

func TestValidationError_Fields(t *testing.T) {
	e := ValidationError{Field: "test", Message: "test message"}
	assert.Equal(t, "test", e.Field)
	assert.Equal(t, "test message", e.Message)
}

// ──── TokenUsage ────────────────────────────────────────────────────────

func TestTokenUsage_Fields(t *testing.T) {
	u := TokenUsage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150}
	assert.Equal(t, 100, u.PromptTokens)
	assert.Equal(t, 50, u.CompletionTokens)
	assert.Equal(t, 150, u.TotalTokens)
}
