package llm

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type LLMAuditLogger interface {
	LogDecisionRequested(ctx context.Context, caseID, provider, model string)
	LogDecisionCompleted(ctx context.Context, caseID, provider, model string, latencyMs int64, usage *TokenUsage)
	LogDecisionFailed(ctx context.Context, caseID, provider, model string, err error)
	LogDecisionValidationFailed(ctx context.Context, caseID string, errors []ValidationError)
	LogFallbackUsed(ctx context.Context, caseID string, reason string)
	LogDecisionReplayed(ctx context.Context, caseID, originalDecisionID string)
	LogEvalCompleted(ctx context.Context, caseID, evalID string)
}

type DBAuditLogger struct {
	pool *pgxpool.Pool
}

func NewDBAuditLogger(pool *pgxpool.Pool) *DBAuditLogger {
	return &DBAuditLogger{pool: pool}
}

func (l *DBAuditLogger) insertAudit(ctx context.Context, action string, caseID string, metadata map[string]interface{}) {
	if l.pool == nil {
		return
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return
	}

	query := `
		INSERT INTO audit.audit_log (category, action, actor, resource_type, resource_id, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err = l.pool.Exec(ctx, query, "llm_decision", action, "system", "decision_case", caseID, metadataJSON)
	if err != nil {
		return
	}
}

func (l *DBAuditLogger) LogDecisionRequested(ctx context.Context, caseID, provider, model string) {
	l.insertAudit(ctx, "requested", caseID, map[string]interface{}{
		"provider": provider,
		"model":    model,
	})
}

func (l *DBAuditLogger) LogDecisionCompleted(ctx context.Context, caseID, provider, model string, latencyMs int64, usage *TokenUsage) {
	metadata := map[string]interface{}{
		"provider":   provider,
		"model":      model,
		"latency_ms": latencyMs,
	}
	if usage != nil {
		metadata["token_usage"] = usage
	}
	l.insertAudit(ctx, "completed", caseID, metadata)
}

func (l *DBAuditLogger) LogDecisionFailed(ctx context.Context, caseID, provider, model string, err error) {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	l.insertAudit(ctx, "failed", caseID, map[string]interface{}{
		"provider": provider,
		"model":    model,
		"error":    errMsg,
	})
}

func (l *DBAuditLogger) LogDecisionValidationFailed(ctx context.Context, caseID string, errors []ValidationError) {
	errList := make([]map[string]string, len(errors))
	for i, e := range errors {
		errList[i] = map[string]string{
			"field":   e.Field,
			"message": e.Message,
		}
	}
	l.insertAudit(ctx, "validation_failed", caseID, map[string]interface{}{
		"validation_errors": errList,
	})
}

func (l *DBAuditLogger) LogFallbackUsed(ctx context.Context, caseID string, reason string) {
	l.insertAudit(ctx, "fallback", caseID, map[string]interface{}{
		"reason": reason,
	})
}

func (l *DBAuditLogger) LogDecisionReplayed(ctx context.Context, caseID, originalDecisionID string) {
	l.insertAudit(ctx, "replayed", caseID, map[string]interface{}{
		"original_decision_id": originalDecisionID,
	})
}

func (l *DBAuditLogger) LogEvalCompleted(ctx context.Context, caseID, evalID string) {
	l.insertAudit(ctx, "eval_completed", caseID, map[string]interface{}{
		"eval_id": evalID,
	})
}

var _ LLMAuditLogger = (*DBAuditLogger)(nil)

type NoOpAuditLogger struct{}

var _ LLMAuditLogger = (*NoOpAuditLogger)(nil)

func (l *NoOpAuditLogger) LogDecisionRequested(ctx context.Context, caseID, provider, model string) {
}
func (l *NoOpAuditLogger) LogDecisionCompleted(ctx context.Context, caseID, provider, model string, latencyMs int64, usage *TokenUsage) {
}
func (l *NoOpAuditLogger) LogDecisionFailed(ctx context.Context, caseID, provider, model string, err error) {
}
func (l *NoOpAuditLogger) LogDecisionValidationFailed(ctx context.Context, caseID string, errors []ValidationError) {
}
func (l *NoOpAuditLogger) LogFallbackUsed(ctx context.Context, caseID string, reason string) {
}
func (l *NoOpAuditLogger) LogDecisionReplayed(ctx context.Context, caseID, originalDecisionID string) {
}
func (l *NoOpAuditLogger) LogEvalCompleted(ctx context.Context, caseID, evalID string) {
}
