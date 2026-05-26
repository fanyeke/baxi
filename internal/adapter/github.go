package adapter

import (
	"context"
	"fmt"
	"log"

	"baxi/internal/action"
)

// GitHubAdapter dispatches action proposals to GitHub (via API).
type GitHubAdapter struct {
	config GitHubConfig
}

// NewGitHubAdapter creates a new GitHubAdapter with the given config.
func NewGitHubAdapter(config GitHubConfig) *GitHubAdapter {
	return &GitHubAdapter{config: config}
}

// Execute implements action.ActionExecutor. In dry-run mode it returns a
// successful result without making any HTTP calls. When the token is empty
// it returns an error. Otherwise it logs the dispatch and returns success.
func (a *GitHubAdapter) Execute(ctx context.Context, proposal action.ActionProposal, dryRun bool) (action.ExecutionResult, error) {
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

	if a.config.Token == "" {
		return action.ExecutionResult{}, fmt.Errorf("github token not configured")
	}

	log.Printf("[GitHubAdapter] dispatching to repo %s: proposal_id=%s action_type=%s title=%q",
		a.config.Repo, proposal.ProposalID, proposal.ActionType, proposal.Title)

	return action.ExecutionResult{
		Success:         true,
		DryRun:          false,
		DispatchPayload: payload,
	}, nil
}

// Compile-time assertion that *GitHubAdapter satisfies action.ActionExecutor.
var _ action.ActionExecutor = (*GitHubAdapter)(nil)
