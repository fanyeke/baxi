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
	provider    llm.DecisionProvider
	repo        DecisionEngineRepository
	pool        *pgxpool.Pool
	fallback    llm.DecisionProvider
	auditLogger llm.LLMAuditLogger
	recorder    SnapshotRecorder
	repair      *llm.RepairPromptRenderer
}

// NewDecisionEngine creates a new DecisionEngine with the given primary provider and repository.
// The fallback is automatically set to a RuleBasedProvider.
func NewDecisionEngine(provider llm.DecisionProvider, repo DecisionEngineRepository, pool *pgxpool.Pool, auditLogger llm.LLMAuditLogger) *DecisionEngine {
	repair, _ := llm.NewRepairPromptRenderer()
	return &DecisionEngine{
		provider:    provider,
		repo:        repo,
		pool:        pool,
		fallback:    llm.NewRuleBasedProvider(),
		auditLogger: auditLogger,
		recorder:    NewNoopSnapshotRecorder(),
		repair:      repair,
	}
}

// WithSnapshotRecorder attaches a SnapshotRecorder for persisting LLM input/output snapshots.
func (e *DecisionEngine) WithSnapshotRecorder(r SnapshotRecorder) *DecisionEngine {
	e.recorder = r
	return e
}

// WithRepairRenderer attaches a RepairPromptRenderer for validation retry.
func (e *DecisionEngine) WithRepairRenderer(r *llm.RepairPromptRenderer) *DecisionEngine {
	e.repair = r
	return e
}

func (e *DecisionEngine) providerName() string {
	switch e.provider.(type) {
	case *llm.OpenAICompatibleProvider:
		return "openai"
	case *llm.RuleBasedProvider:
		return "rule_based"
	default:
		return "unknown"
	}
}

func (e *DecisionEngine) modelName() string {
	if p, ok := e.provider.(interface{ ModelName() string }); ok {
		return p.ModelName()
	}
	return ""
}

func (e *DecisionEngine) fallbackProviderName() string {
	switch e.fallback.(type) {
	case *llm.RuleBasedProvider:
		return "rule_based"
	default:
		return "unknown"
	}
}

func (e *DecisionEngine) GenerateDecision(ctx context.Context, caseID string, context *DecisionContext) (*llm.DecisionOutput, error) {
	llmSafeContext := BuildLLMSafeContext(context)

	// Phase 2: Persist the LLM input context as a snapshot before calling the provider.
	e.persistContextSnapshot(ctx, caseID, llmSafeContext)

	provName := e.providerName()
	modName := e.modelName()

	e.auditLogger.LogDecisionRequested(ctx, caseID, provName, modName)
	e.recorder.RecordEvent(ctx, LineageEventRecord{
		CaseID:    caseID,
		EventType: LineageEventDecisionRequested,
		Actor:     "system",
	})

	start := time.Now()
	output, err := e.provider.GenerateDecision(ctx, llmSafeContext)
	latencyMs := time.Since(start).Milliseconds()

	if err != nil {
		e.auditLogger.LogDecisionFailed(ctx, caseID, provName, modName, err)
		return e.handleProviderError(ctx, caseID, llmSafeContext, err)
	}

	// Phase 3: Save raw output snapshot
	e.persistRawOutputSnapshot(ctx, caseID, output)

	// Phase 3: Save parsed output snapshot
	e.persistParsedOutputSnapshot(ctx, caseID, output)

	result := llm.ValidateDecision(output, context.AllowedActions)
	if result.Valid {
		e.auditLogger.LogDecisionCompleted(ctx, caseID, provName, modName, latencyMs, nil)
		e.recorder.RecordEvent(ctx, LineageEventRecord{
			CaseID:    caseID,
			EventType: LineageEventDecisionGenerated,
			Actor:     "system",
		})
		if saveErr := e.saveDecision(ctx, caseID, output, "valid", nil, nil); saveErr != nil {
			return output, saveErr
		}
		if updateErr := e.updateCaseStatus(ctx, caseID, "decision_generated"); updateErr != nil {
			return output, updateErr
		}
		return output, nil
	}

	e.auditLogger.LogDecisionValidationFailed(ctx, caseID, result.Errors)
	e.persistValidationResult(ctx, caseID, result)
	return e.handleInvalidOutput(ctx, caseID, llmSafeContext, output, result.Errors)
}

func (e *DecisionEngine) handleProviderError(ctx context.Context, caseID string, llmSafeContext llm.LLMSafeContext, providerErr error) (*llm.DecisionOutput, error) {
	e.recorder.RecordEvent(ctx, LineageEventRecord{
		CaseID:    caseID,
		EventType: LineageEventFallbackUsed,
		Actor:     "system",
		EventData: json.RawMessage(`{"reason":"provider_error"}`),
	})

	fallbackOutput, fallbackErr := e.fallback.GenerateDecision(ctx, llmSafeContext)
	if fallbackErr != nil {
		reason := fmt.Sprintf("provider error: %v; fallback error: %v", providerErr, fallbackErr)
		e.recorder.RecordEvent(ctx, LineageEventRecord{
			CaseID:    caseID,
			EventType: LineageEventCaseFailed,
			Actor:     "system",
		})
		e.auditLogger.LogFallbackUsed(ctx, caseID, reason)
		_ = e.saveDecision(ctx, caseID, nil, "failed", nil, &reason)
		_ = e.updateCaseStatus(ctx, caseID, "failed")
		return nil, fmt.Errorf("provider error: %w; fallback error: %v", providerErr, fallbackErr)
	}

	reason := "provider error: " + providerErr.Error()
	e.auditLogger.LogFallbackUsed(ctx, caseID, reason)
	e.auditLogger.LogDecisionCompleted(ctx, caseID, e.fallbackProviderName(), "", 0, nil)
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

	e.recorder.RecordEvent(ctx, LineageEventRecord{
		CaseID:    caseID,
		EventType: LineageEventValidationFailed,
		Actor:     "system",
	})

	// Phase 3: Repair retry — one attempt to fix the output with a repair prompt
	if e.repair != nil {
		e.recorder.RecordEvent(ctx, LineageEventRecord{
			CaseID:    caseID,
			EventType: LineageEventRepairAttempted,
			Actor:     "system",
		})
		repairOutput, repairErr := e.provider.GenerateDecision(ctx, e.buildRepairContext(llmSafeContext, validationErrors))
		if repairErr == nil && llm.ValidateDecision(repairOutput, llmSafeContext.AllowedActions).Valid {
			e.recorder.RecordEvent(ctx, LineageEventRecord{
				CaseID:    caseID,
				EventType: LineageEventRepairSucceeded,
				Actor:     "system",
			})
			e.auditLogger.LogDecisionCompleted(ctx, caseID, e.providerName(), e.modelName(), 0, nil)
			e.recorder.RecordEvent(ctx, LineageEventRecord{
				CaseID:    caseID,
				EventType: LineageEventDecisionGenerated,
				Actor:     "system",
			})
			if saveErr := e.saveDecision(ctx, caseID, repairOutput, "valid", nil, nil); saveErr != nil {
				return repairOutput, saveErr
			}
			if updateErr := e.updateCaseStatus(ctx, caseID, "decision_generated"); updateErr != nil {
				return repairOutput, updateErr
			}
			return repairOutput, nil
		}
		e.recorder.RecordEvent(ctx, LineageEventRecord{
			CaseID:    caseID,
			EventType: LineageEventRepairFailed,
			Actor:     "system",
		})
	}

	// Fallback to rule-based provider
	e.recorder.RecordEvent(ctx, LineageEventRecord{
		CaseID:    caseID,
		EventType: LineageEventFallbackUsed,
		Actor:     "system",
	})
	e.auditLogger.LogFallbackUsed(ctx, caseID, reason)

	fallbackOutput, fallbackErr := e.fallback.GenerateDecision(ctx, llmSafeContext)
	if fallbackErr != nil {
		failReason := fmt.Sprintf("validation failed: %s; fallback error: %v", reason, fallbackErr)
		e.auditLogger.LogFallbackUsed(ctx, caseID, failReason)
		_ = e.saveDecision(ctx, caseID, output, "failed", validationErrors, &failReason)
		_ = e.updateCaseStatus(ctx, caseID, "failed")
		return nil, fmt.Errorf("validation failed and fallback error: %v", fallbackErr)
	}

	e.auditLogger.LogDecisionCompleted(ctx, caseID, e.fallbackProviderName(), "", 0, nil)
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

// BuildLLMSafeContext maps a DecisionContext to an LLMSafeContext.
func BuildLLMSafeContext(dc *DecisionContext) llm.LLMSafeContext {
	llmCtx := llm.LLMSafeContext{
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

	// Map enriched objects from OAG link traversal
	if len(dc.EnrichedObjects) > 0 {
		enriched := make([]llm.EnrichedObjectData, 0, len(dc.EnrichedObjects))
		for _, eo := range dc.EnrichedObjects {
			enriched = append(enriched, llm.EnrichedObjectData{
				LinkName:   eo.LinkName,
				Depth:      eo.Depth,
				ObjectType: eo.ObjectType,
				ObjectID:   eo.ObjectID,
				Properties: eo.Properties,
			})
		}
		llmCtx.EnrichedObjects = enriched
	}

	return llmCtx
}

// persistContextSnapshot saves the LLMSafeContext as a snapshot (best-effort).
func (e *DecisionEngine) persistContextSnapshot(ctx context.Context, caseID string, safeCtx llm.LLMSafeContext) {
	data, err := json.Marshal(safeCtx)
	if err != nil {
		return
	}
	raw := json.RawMessage(data)
	e.recorder.RecordSnapshot(ctx, DataSnapshotRecord{
		CaseID:       caseID,
		SnapshotType: SnapshotTypeLLMSafeContext,
		SnapshotJSON: raw,
		RowCount:     1,
	})
}

// persistRawOutputSnapshot saves the serialized DecisionOutput as a raw snapshot (best-effort).
func (e *DecisionEngine) persistRawOutputSnapshot(ctx context.Context, caseID string, output *llm.DecisionOutput) {
	if output == nil {
		return
	}
	data, err := json.Marshal(output)
	if err != nil {
		return
	}
	raw := json.RawMessage(data)
	e.recorder.RecordSnapshot(ctx, DataSnapshotRecord{
		CaseID:       caseID,
		SnapshotType: SnapshotTypeLLMRawOutput,
		SnapshotJSON: raw,
		RowCount:     1,
	})
}

// persistParsedOutputSnapshot saves the parsed DecisionOutput as a snapshot (best-effort).
func (e *DecisionEngine) persistParsedOutputSnapshot(ctx context.Context, caseID string, output *llm.DecisionOutput) {
	if output == nil {
		return
	}
	data, err := json.Marshal(output)
	if err != nil {
		return
	}
	raw := json.RawMessage(data)
	e.recorder.RecordSnapshot(ctx, DataSnapshotRecord{
		CaseID:       caseID,
		SnapshotType: SnapshotTypeLLMParsedOutput,
		SnapshotJSON: raw,
		RowCount:     1,
	})
}

// persistValidationResult saves the validation result as a snapshot (best-effort).
func (e *DecisionEngine) persistValidationResult(ctx context.Context, caseID string, result *llm.ValidationResult) {
	data, err := json.Marshal(result)
	if err != nil {
		return
	}
	raw := json.RawMessage(data)
	e.recorder.RecordSnapshot(ctx, DataSnapshotRecord{
		CaseID:       caseID,
		SnapshotType: SnapshotTypeLLMValidation,
		SnapshotJSON: raw,
		RowCount:     1,
	})
}

// buildRepairContext creates an LLMSafeContext that includes the validation errors
// in the context for the repair retry. It keeps the original trigger/object/governance
// data unchanged and injects errors as additional evidence.
func (e *DecisionEngine) buildRepairContext(original llm.LLMSafeContext, validationErrors []llm.ValidationError) llm.LLMSafeContext {
	errMsgs := make([]string, len(validationErrors))
	for i, verr := range validationErrors {
		errMsgs[i] = verr.Field + ": " + verr.Message
	}
	repaired := original
	if repaired.GovernanceInfo.Role == "" {
		repaired.GovernanceInfo.Role = "agent_readonly"
	}
	repaired.GovernanceInfo.RepairErrors = errMsgs
	return repaired
}
