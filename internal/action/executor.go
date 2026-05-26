package action

import (
	"context"
)

// ExecutionResult represents the outcome of executing an action.
type ExecutionResult struct {
	Success         bool                   `json:"success"`
	DryRun          bool                   `json:"dry_run"`
	DispatchPayload map[string]interface{} `json:"dispatch_payload,omitempty"`
	Error           string                 `json:"error,omitempty"`
	OutboxEventID   string                 `json:"outbox_event_id,omitempty"`
}

// ExecutionContext provides runtime context for action execution.
type ExecutionContext struct {
	ActorID string // Who is executing this action
	TraceID string // Request tracing identifier
}

// ActionExecutor defines the interface for executing action proposals.
// Implementations include NoOpExecutor (dry-run), FeishuAdapter, GitHubAdapter, etc.
type ActionExecutor interface {
	// Execute executes the given action proposal. If dryRun is true,
	// the implementation should log what it would do without side effects.
	Execute(ctx context.Context, proposal ActionProposal, dryRun bool) (ExecutionResult, error)
}

// NoOpExecutor is a dry-run executor that logs what it would do.
// Used when ACTION_APPLY_DRY_RUN=true or when no adapter is configured.
type NoOpExecutor struct{}

// NewNoOpExecutor creates a new NoOpExecutor.
func NewNoOpExecutor() *NoOpExecutor {
	return &NoOpExecutor{}
}

// Execute implements ActionExecutor. It returns a successful result with
// the dryRun flag preserved and a dispatch payload containing metadata.
func (e *NoOpExecutor) Execute(ctx context.Context, proposal ActionProposal, dryRun bool) (ExecutionResult, error) {
	return ExecutionResult{
		Success: true,
		DryRun:  dryRun,
		DispatchPayload: map[string]interface{}{
			"action_type": proposal.ActionType,
			"proposal_id": proposal.ProposalID,
			"case_id":     proposal.CaseID,
			"dry_run":     dryRun,
		},
	}, nil
}

// Compile-time assertion that *NoOpExecutor satisfies ActionExecutor.
var _ ActionExecutor = (*NoOpExecutor)(nil)
