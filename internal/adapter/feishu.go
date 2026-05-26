package adapter

import (
	"context"
	"fmt"
	"log"

	"baxi/internal/action"
)

// FeishuAdapter dispatches action proposals to Feishu (Lark) via webhook.
type FeishuAdapter struct {
	config FeishuConfig
}

// NewFeishuAdapter creates a new FeishuAdapter with the given config.
func NewFeishuAdapter(config FeishuConfig) *FeishuAdapter {
	return &FeishuAdapter{config: config}
}

// Execute implements action.ActionExecutor. In dry-run mode it returns a
// successful result without making any HTTP calls. When the webhook URL is
// empty it returns an error. Otherwise it logs the dispatch and returns success.
func (a *FeishuAdapter) Execute(ctx context.Context, proposal action.ActionProposal, dryRun bool) (action.ExecutionResult, error) {
	payload := map[string]interface{}{
		"proposal_id": proposal.ProposalID,
		"case_id":     proposal.CaseID,
		"action_type": proposal.ActionType,
		"channel":     ActionChannel(proposal.ActionType),
		"dry_run":     dryRun,
	}

	if dryRun {
		return action.ExecutionResult{
			Success:         true,
			DryRun:          true,
			DispatchPayload: payload,
		}, nil
	}

	if a.config.WebhookURL == "" {
		return action.ExecutionResult{}, fmt.Errorf("feishu webhook not configured")
	}

	log.Printf("[FeishuAdapter] dispatching to %s: proposal_id=%s action_type=%s title=%q",
		a.config.WebhookURL, proposal.ProposalID, proposal.ActionType, proposal.Title)

	return action.ExecutionResult{
		Success:         true,
		DryRun:          false,
		DispatchPayload: payload,
	}, nil
}

// Compile-time assertion that *FeishuAdapter satisfies action.ActionExecutor.
var _ action.ActionExecutor = (*FeishuAdapter)(nil)
