package llm

import (
	"context"
	"errors"
	"testing"
)

func TestNoOpAuditLogger_AllMethods(t *testing.T) {
	logger := &NoOpAuditLogger{}
	ctx := context.Background()

	logger.LogDecisionRequested(ctx, "case-1", "openai", "gpt-4o")
	logger.LogDecisionCompleted(ctx, "case-1", "openai", "gpt-4o", 150, &TokenUsage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150})
	logger.LogDecisionCompleted(ctx, "case-1", "openai", "gpt-4o", 150, nil)
	logger.LogDecisionFailed(ctx, "case-1", "openai", "gpt-4o", errors.New("rate limited"))
	logger.LogDecisionFailed(ctx, "case-1", "openai", "gpt-4o", nil)
	logger.LogDecisionValidationFailed(ctx, "case-1", []ValidationError{{Field: "confidence", Message: "out of range"}})
	logger.LogDecisionValidationFailed(ctx, "case-1", nil)
	logger.LogFallbackUsed(ctx, "case-1", "provider unavailable")
	logger.LogDecisionReplayed(ctx, "case-1", "orig-decision-1")
	logger.LogEvalCompleted(ctx, "case-1", "eval-1")

	// No assertions needed — verifying no panic on any method
}

func TestTokenUsage(t *testing.T) {
	usage := &TokenUsage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}
	if usage.PromptTokens != 100 {
		t.Errorf("expected PromptTokens=100, got %d", usage.PromptTokens)
	}
	if usage.CompletionTokens != 50 {
		t.Errorf("expected CompletionTokens=50, got %d", usage.CompletionTokens)
	}
	if usage.TotalTokens != 150 {
		t.Errorf("expected TotalTokens=150, got %d", usage.TotalTokens)
	}
}

func TestDBAuditLogger_NilPoolDoesNotPanic(t *testing.T) {
	logger := NewDBAuditLogger(nil)
	ctx := context.Background()

	logger.LogDecisionRequested(ctx, "case-1", "openai", "gpt-4o")
	logger.LogDecisionCompleted(ctx, "case-1", "openai", "gpt-4o", 100, nil)
	logger.LogDecisionFailed(ctx, "case-1", "openai", "gpt-4o", errors.New("fail"))
	logger.LogDecisionValidationFailed(ctx, "case-1", []ValidationError{{Field: "test", Message: "err"}})
	logger.LogFallbackUsed(ctx, "case-1", "reason")
	logger.LogDecisionReplayed(ctx, "case-1", "orig-1")
	logger.LogEvalCompleted(ctx, "case-1", "eval-1")
}


