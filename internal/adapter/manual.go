package adapter

import (
	"context"
	"fmt"
	"log"

	"baxi/internal/action"
)

// ManualAdapter queues action proposals for human review instead of dispatching
// them to an external system.
type ManualAdapter struct {
	config ManualConfig
}

// NewManualAdapter creates a new ManualAdapter with the given config.
func NewManualAdapter(config ManualConfig) *ManualAdapter {
	return &ManualAdapter{config: config}
}

// Execute implements action.ActionExecutor. In dry-run mode it returns a
// preview result. In normal mode it logs the event as queued for manual review.
func (a *ManualAdapter) Execute(ctx context.Context, proposal action.ActionProposal, dryRun bool) (action.ExecutionResult, error) {
	ruleID := extractRuleID(proposal.Payload)
	msg := fmt.Sprintf("Event queued for manual review: rule=%s", ruleID)

	payload := map[string]interface{}{
		"proposal_id": proposal.ProposalID,
		"case_id":     proposal.CaseID,
		"action_type": proposal.ActionType,
		"channel":     "manual",
		"dry_run":     dryRun,
		"rule_id":     ruleID,
		"message":     msg,
	}

	if dryRun {
		return action.ExecutionResult{
			Success:         true,
			DryRun:          true,
			DispatchPayload: payload,
		}, nil
	}

	log.Printf("[ManualAdapter] %s", msg)

	return action.ExecutionResult{
		Success:         true,
		DryRun:          false,
		DispatchPayload: payload,
	}, nil
}

// extractRuleID extracts the rule_id from a payload map. If the payload is nil
// or does not contain a string rule_id, it returns "unknown".
func extractRuleID(payload map[string]interface{}) string {
	if payload == nil {
		return "unknown"
	}
	v, ok := payload["rule_id"].(string)
	if !ok || v == "" {
		return "unknown"
	}
	return v
}

// Compile-time assertion that *ManualAdapter satisfies action.ActionExecutor.
var _ action.ActionExecutor = (*ManualAdapter)(nil)
