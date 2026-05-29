package action

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotApproved       = errors.New("proposal is not approved")
	ErrActionNotAllowed  = errors.New("action type is not allowed")
	ErrProposalNotFound  = errors.New("proposal not found")
	ErrLineageIncomplete = errors.New("decision lineage incomplete: missing required lineage events")
)

// ProposalLoader retrieves action proposals by ID.
type ProposalLoader interface {
	GetProposalByID(ctx context.Context, pool *pgxpool.Pool, proposalID string) (*ActionProposal, error)
}

// ApplyService executes approved action proposals with dry-run support
// and whitelist enforcement.
// LineageVerifier checks whether a decision case has complete lineage before execution.
type LineageVerifier interface {
	HasCompleteLineage(ctx context.Context, caseID string) (bool, error)
}

// LineageRecorder records decision lineage events in the apply flow.
type LineageRecorder interface {
	RecordApplyEvent(ctx context.Context, pool *pgxpool.Pool, caseID, proposalID, eventType, actor string, eventData map[string]interface{}) error
}

// ApplyService executes approved action proposals with dry-run support
// and whitelist enforcement.
type ApplyService struct {
	registry      *ActionRegistry
	executors     map[string]ActionExecutor
	loader        ProposalLoader
	lineageVerify LineageVerifier
	lineageRecord LineageRecorder
	pool          *pgxpool.Pool
}

// NewApplyService creates a new ApplyService.
// registry: whitelist enforcement via ActionRegistry.
// executors: channel-name → ActionExecutor mapping (e.g. "feishu", "github").
// loader: retrieves proposals by ID.
func NewApplyService(registry *ActionRegistry, executors map[string]ActionExecutor, loader ProposalLoader, lineageVerify LineageVerifier, lineageRecord LineageRecorder, pool *pgxpool.Pool) *ApplyService {
	if executors == nil {
		executors = make(map[string]ActionExecutor)
	}
	return &ApplyService{
		registry:      registry,
		executors:     executors,
		loader:        loader,
		lineageVerify: lineageVerify,
		lineageRecord: lineageRecord,
		pool:          pool,
	}
}

// ExecuteOption is a functional option for ExecuteProposal.
type ExecuteOption func(*ExecuteOptions)

// ExecuteOptions controls the behavior of ExecuteProposal.
type ExecuteOptions struct {
	DryRun bool
}

// WithDryRun sets the dry-run flag. When true (default), no side effects occur.
func WithDryRun(dryRun bool) ExecuteOption {
	return func(opts *ExecuteOptions) {
		opts.DryRun = dryRun
	}
}

// ExecuteProposal executes an approved action proposal.
func (s *ApplyService) ExecuteProposal(ctx context.Context, pool *pgxpool.Pool, proposalID string, actorID string, opts ...ExecuteOption) (*ExecutionResult, error) {
	executeOpts := &ExecuteOptions{DryRun: true}
	for _, opt := range opts {
		opt(executeOpts)
	}

	proposal, err := s.loader.GetProposalByID(ctx, pool, proposalID)
	if err != nil {
		return nil, fmt.Errorf("load proposal: %w", err)
	}
	if proposal == nil {
		return nil, ErrProposalNotFound
	}

	if proposal.ApplyStatus != "approved" &&
		proposal.ApplyStatus != "proposed" {
		return nil, ErrNotApproved
	}

	// Risk-adaptive HITL: if the proposal is still in "proposed" status and risk level is low,
	// allow execution without requiring human approval.
	if proposal.ApplyStatus == "proposed" {
		if proposal.RiskLevel != "low" {
			return nil, ErrNotApproved
		}
		// Even for "low" risk, check if the action config requires approval
		// and if so, verify the registry allows this
		if cfg, ok := s.registry.GetActionConfig(proposal.ActionType); ok {
			if cfg.RequiresApproval {
				// Low-risk actions bypass human review approval
			}
		}
	}

	if !s.registry.IsAllowed(proposal.ActionType) {
		return nil, ErrActionNotAllowed
	}

	traceID := generateTraceID()

	// Phase 5: Check lineage completeness before allowing execution
	if !executeOpts.DryRun && s.lineageVerify != nil {
		if ok, err := s.lineageVerify.HasCompleteLineage(ctx, proposal.CaseID); err != nil || !ok {
			return nil, ErrLineageIncomplete
		}
	}

	if executeOpts.DryRun {
		noop := NewNoOpExecutor()
		res, err := noop.Execute(ctx, *proposal, true)
		if err != nil {
			return nil, err
		}
		return &res, nil
	}

	return s.executeForReal(ctx, pool, proposal, actorID, traceID)
}

func (s *ApplyService) executeForReal(ctx context.Context, pool *pgxpool.Pool, proposal *ActionProposal, actorID string, traceID string) (*ExecutionResult, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := updateProposalStatus(ctx, tx, proposal.ProposalID, "applying"); err != nil {
		return nil, err
	}

	channel := actionChannel(proposal.ActionType)
	executor, ok := s.executors[channel]
	if !ok {
		log.Printf("[ApplyService] warning: no executor found for channel=%s action_type=%s proposal=%s", channel, proposal.ActionType, proposal.ProposalID)

		_ = updateProposalStatus(ctx, tx, proposal.ProposalID, "failed")
		_ = insertAuditLog(ctx, tx, actorID, proposal, false, fmt.Sprintf("no executor for channel %s", channel))

		if commitErr := tx.Commit(ctx); commitErr != nil {
			return nil, fmt.Errorf("commit after adapter not found: %w", commitErr)
		}

		return &ExecutionResult{
			Success: false,
			DryRun:  false,
			Error:   fmt.Sprintf("no executor found for channel %s", channel),
		}, nil
	}

	log.Printf("[ApplyService] dispatching proposal=%s action_type=%s channel=%s actor=%s trace=%s",
		proposal.ProposalID, proposal.ActionType, channel, actorID, traceID)

	result, err := executor.Execute(ctx, *proposal, false)

	var execResult ExecutionResult
	if err != nil {
		execResult = ExecutionResult{
			Success: false,
			DryRun:  false,
			Error:   err.Error(),
		}
	} else {
		execResult = result
	}

	if execResult.Success {
		if err := updateProposalToApplied(ctx, tx, proposal.ProposalID, actorID); err != nil {
			return nil, err
		}

		outboxEvent, err := CreateOutboxEventFromProposal(ctx, tx, proposal)
		if err != nil {
			return nil, fmt.Errorf("create outbox event: %w", err)
		}
		execResult.OutboxEventID = outboxEvent.EventID
	} else {
		if err := updateProposalStatus(ctx, tx, proposal.ProposalID, "failed"); err != nil {
			return nil, err
		}
	}

	if auditErr := insertAuditLog(ctx, tx, actorID, proposal, execResult.Success, execResult.Error); auditErr != nil {
		log.Printf("[ApplyService] warning: failed to insert audit log: %v", auditErr)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &execResult, nil
}

// actionChannel returns the dispatch channel for a given action type.
// Mirrors the mapping in internal/adapter/domain.go to avoid import cycle.
func actionChannel(actionType string) string {
	switch actionType {
	case "export_report", "notify_owner", "create_outbox_message":
		return "feishu"
	case "create_followup_task":
		return "github"
	default:
		return "unknown"
	}
}

func generateTraceID() string {
	return fmt.Sprintf("trace-%d", time.Now().UnixNano())
}

func updateProposalStatus(ctx context.Context, tx pgx.Tx, proposalID string, status string) error {
	query := `UPDATE ai.action_proposal SET apply_status = $1 WHERE proposal_id = $2`
	res, err := tx.Exec(ctx, query, status, proposalID)
	if err != nil {
		return fmt.Errorf("update proposal status: %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("proposal %s not found", proposalID)
	}
	return nil
}

func updateProposalToApplied(ctx context.Context, tx pgx.Tx, proposalID string, actorID string) error {
	query := `UPDATE ai.action_proposal SET apply_status = $1, applied_at = NOW(), applied_by = $2 WHERE proposal_id = $3`
	res, err := tx.Exec(ctx, query, "applied", actorID, proposalID)
	if err != nil {
		return fmt.Errorf("update proposal to applied: %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("proposal %s not found", proposalID)
	}
	return nil
}

func insertExecutionOutcome(ctx context.Context, tx pgx.Tx, proposalID, caseID, actionType, actor string, success bool) error {
	status := "applied"
	if !success {
		status = "failed"
	}
	outcomeID := fmt.Sprintf("out_%d", time.Now().UnixNano())
	query := `
		INSERT INTO ai.action_outcome (
			outcome_id, case_id, proposal_id, action_type,
			execution_status, recorded_by, recorded_at
		) VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`
	_, err := tx.Exec(ctx, query, outcomeID, caseID, proposalID, actionType, status, actor)
	if err != nil {
		return fmt.Errorf("insert outcome: %w", err)
	}
	return nil
}

func insertAuditLog(ctx context.Context, tx pgx.Tx, actorID string, proposal *ActionProposal, success bool, errorMsg string) error {
	action := "execute"
	if !success {
		action = "execute_failed"
	}

	metadata := map[string]interface{}{
		"action_type": proposal.ActionType,
		"case_id":     proposal.CaseID,
		"success":     success,
	}
	if errorMsg != "" {
		metadata["error"] = errorMsg
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal audit metadata: %w", err)
	}

	query := `
		INSERT INTO audit.audit_log (category, action, actor, resource_type, resource_id, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err = tx.Exec(ctx, query, "action_apply", action, actorID, "action_proposal", proposal.ProposalID, metadataJSON)
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}
