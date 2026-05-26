package decision

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"baxi/internal/llm"
	"baxi/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DecisionEngineRepository defines the storage interface for the decision engine.
type DecisionEngineRepository interface {
	CreateDecision(ctx context.Context, pool *pgxpool.Pool, row *repository.LLMDecisionRow) error
	UpdateCaseStatus(ctx context.Context, pool *pgxpool.Pool, caseID string, status string, contextJSON *json.RawMessage, contextHash *string, governanceSnapshot *json.RawMessage) error
	GetCaseByID(ctx context.Context, pool *pgxpool.Pool, caseID string) (*repository.DecisionCaseRow, error)
}

// DecisionEngine orchestrates decision generation with validation and rule-based fallback.
type DecisionEngine struct {
	provider llm.DecisionProvider
	repo     DecisionEngineRepository
	pool     *pgxpool.Pool
	fallback llm.DecisionProvider
}

// NewDecisionEngine creates a new DecisionEngine with the given primary provider and repository.
// The fallback is automatically set to a RuleBasedProvider.
func NewDecisionEngine(provider llm.DecisionProvider, repo DecisionEngineRepository, pool *pgxpool.Pool) *DecisionEngine {
	return &DecisionEngine{
		provider: provider,
		repo:     repo,
		pool:     pool,
		fallback: llm.NewRuleBasedProvider(),
	}
}

// GenerateDecision generates a decision for the given case using the primary provider,
// validates the output, and falls back to a rule-based provider if validation fails.
func (e *DecisionEngine) GenerateDecision(ctx context.Context, caseID string, context *DecisionContext) (*llm.DecisionOutput, error) {
	llmSafeContext := buildLLMSafeContext(context)

	output, err := e.provider.GenerateDecision(ctx, llmSafeContext)
	if err != nil {
		return e.handleProviderError(ctx, caseID, llmSafeContext, err)
	}

	result := llm.ValidateDecision(output, context.AllowedActions)
	if result.Valid {
		if saveErr := e.saveDecision(ctx, caseID, output, "valid", nil, nil); saveErr != nil {
			return output, saveErr
		}
		if updateErr := e.updateCaseStatus(ctx, caseID, "decision_generated"); updateErr != nil {
			return output, updateErr
		}
		return output, nil
	}

	return e.handleInvalidOutput(ctx, caseID, llmSafeContext, output, result.Errors)
}

func (e *DecisionEngine) handleProviderError(ctx context.Context, caseID string, llmSafeContext llm.LLMSafeContext, providerErr error) (*llm.DecisionOutput, error) {
	fallbackOutput, fallbackErr := e.fallback.GenerateDecision(ctx, llmSafeContext)
	if fallbackErr != nil {
		reason := fmt.Sprintf("provider error: %v; fallback error: %v", providerErr, fallbackErr)
		_ = e.saveDecision(ctx, caseID, nil, "failed", nil, &reason)
		_ = e.updateCaseStatus(ctx, caseID, "failed")
		return nil, fmt.Errorf("provider error: %w; fallback error: %v", providerErr, fallbackErr)
	}

	reason := "provider error: " + providerErr.Error()
	if saveErr := e.saveDecision(ctx, caseID, fallbackOutput, "fallback", nil, &reason); saveErr != nil {
		return fallbackOutput, saveErr
	}
	if updateErr := e.updateCaseStatus(ctx, caseID, "decision_generated"); updateErr != nil {
		return fallbackOutput, updateErr
	}
	return fallbackOutput, nil
}

func (e *DecisionEngine) handleInvalidOutput(ctx context.Context, caseID string, llmSafeContext llm.LLMSafeContext, output *llm.DecisionOutput, validationErrors []llm.ValidationError) (*llm.DecisionOutput, error) {
	reason := llm.ValidateDecisionErrors(output, llmSafeContext.AllowedActions)

	fallbackOutput, fallbackErr := e.fallback.GenerateDecision(ctx, llmSafeContext)
	if fallbackErr != nil {
		failReason := fmt.Sprintf("validation failed: %s; fallback error: %v", reason, fallbackErr)
		_ = e.saveDecision(ctx, caseID, output, "failed", validationErrors, &failReason)
		_ = e.updateCaseStatus(ctx, caseID, "failed")
		return nil, fmt.Errorf("validation failed and fallback error: %v", fallbackErr)
	}

	if saveErr := e.saveDecision(ctx, caseID, fallbackOutput, "fallback", validationErrors, &reason); saveErr != nil {
		return fallbackOutput, saveErr
	}
	if updateErr := e.updateCaseStatus(ctx, caseID, "decision_generated"); updateErr != nil {
		return fallbackOutput, updateErr
	}
	return fallbackOutput, nil
}

func (e *DecisionEngine) saveDecision(ctx context.Context, caseID string, output *llm.DecisionOutput, status string, validationErrors []llm.ValidationError, fallbackReason *string) error {
	var outputJSON *json.RawMessage
	if output != nil {
		data, err := json.Marshal(output)
		if err != nil {
			return fmt.Errorf("marshal decision output: %w", err)
		}
		raw := json.RawMessage(data)
		outputJSON = &raw
	}

	var validationErrorsJSON *json.RawMessage
	if len(validationErrors) > 0 {
		data, err := json.Marshal(validationErrors)
		if err != nil {
			return fmt.Errorf("marshal validation errors: %w", err)
		}
		raw := json.RawMessage(data)
		validationErrorsJSON = &raw
	}

	var confidence *float64
	if output != nil {
		confidence = &output.Confidence
	}

	statusStr := status
	row := &repository.LLMDecisionRow{
		DecisionID:       GenerateDecisionID(),
		CaseID:           caseID,
		OutputJSON:       outputJSON,
		Confidence:       confidence,
		CreatedAt:        time.Now(),
		Status:           &statusStr,
		FallbackReason:   fallbackReason,
		ValidationErrors: validationErrorsJSON,
	}

	return e.repo.CreateDecision(ctx, e.pool, row)
}

func (e *DecisionEngine) updateCaseStatus(ctx context.Context, caseID string, status string) error {
	return e.repo.UpdateCaseStatus(ctx, e.pool, caseID, status, nil, nil, nil)
}

func buildLLMSafeContext(dc *DecisionContext) llm.LLMSafeContext {
	return llm.LLMSafeContext{
		CaseID: dc.DecisionCaseID,
		Trigger: llm.TriggerInfo{
			AlertID:       dc.Trigger.AlertID,
			RuleID:        dc.Trigger.RuleID,
			Severity:      dc.Trigger.Severity,
			MetricName:    dc.Trigger.MetricName,
			CurrentValue:  dc.Trigger.CurrentValue,
			BaselineValue: dc.Trigger.BaselineValue,
			DeltaPct:      dc.Trigger.DeltaPct,
		},
		ObjectContext: llm.ObjectContext{
			ObjectType: dc.ObjectContext.ObjectType,
			ObjectID:   dc.ObjectContext.ObjectID,
			Properties: dc.ObjectContext.Properties,
		},
		GovernanceInfo: llm.GovernanceInfo{
			Classification:   dc.Governance.Classification,
			RedactionApplied: dc.Governance.RedactionApplied,
			RedactedFields:   dc.Governance.RedactedFields,
			Role:             dc.Governance.Role,
		},
		AllowedActions:   dc.AllowedActions,
		ForbiddenActions: dc.ForbiddenActions,
	}
}
