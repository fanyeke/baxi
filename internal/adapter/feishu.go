package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"baxi/internal/action"
)

// FeishuAdapter dispatches action proposals to Feishu (Lark) via open API.
type FeishuAdapter struct {
	config FeishuConfig
	client feishuHTTPClient
}

// NewFeishuAdapter creates a new FeishuAdapter with the given config.
func NewFeishuAdapter(config FeishuConfig) *FeishuAdapter {
	var client feishuHTTPClient
	if config.AppID != "" && config.AppSecret != "" {
		client = newRealFeishuClient(config.AppID, config.AppSecret, false)
	}
	return &FeishuAdapter{config: config, client: client}
}

// NewFeishuAdapterWithClient creates a new FeishuAdapter with a custom HTTP client (for testing).
func NewFeishuAdapterWithClient(config FeishuConfig, client feishuHTTPClient) *FeishuAdapter {
	return &FeishuAdapter{config: config, client: client}
}

// Execute implements action.ActionExecutor.
func (a *FeishuAdapter) Execute(ctx context.Context, proposal action.ActionProposal, dryRun bool) (action.ExecutionResult, error) {
	message, err := a.formatMessage(proposal)
	if err != nil {
		return action.ExecutionResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	payload := map[string]interface{}{
		"proposal_id": proposal.ProposalID,
		"case_id":     proposal.CaseID,
		"action_type": proposal.ActionType,
		"channel":     ActionChannel(proposal.ActionType),
		"dry_run":     dryRun,
		"message":     message,
	}

	if dryRun {
		return action.ExecutionResult{
			Success:         true,
			DryRun:          true,
			DispatchPayload: payload,
		}, nil
	}

	chatID := a.getChatID()
	if chatID == "" {
		return action.ExecutionResult{
			Success: false,
			Error:   "no chat_id configured (set FEISHU_CHAT_ID in env or feishu_app.yml)",
		}, nil
	}

	if a.client == nil {
		return action.ExecutionResult{
			Success: false,
			Error:   "feishu client not initialized (app_id and app_secret required)",
		}, nil
	}

	msgID, err := a.client.sendMessage(chatID, message, "text")
	if err != nil {
		return action.ExecutionResult{
			Success:         false,
			Error:           err.Error(),
			DispatchPayload: payload,
		}, nil
	}

	payload["external_ref"] = msgID
	return action.ExecutionResult{
		Success:         true,
		DryRun:          false,
		DispatchPayload: payload,
	}, nil
}

// formatMessage builds the human-readable alert text from the proposal payload.
func (a *FeishuAdapter) formatMessage(proposal action.ActionProposal) (string, error) {
	p := proposal.Payload
	if p == nil {
		p = make(map[string]interface{})
	}

	ruleID := getString(p, "rule_id", "unknown")
	metric := getString(p, "metric_name", "unknown")
	lines := []string{fmt.Sprintf("[Alert] %s: %s", ruleID, metric)}

	if current := getString(p, "current_value", ""); current != "" {
		lines = append(lines, fmt.Sprintf("Current: %s", current))
	}
	if baseline := getString(p, "baseline_value", ""); baseline != "" {
		lines = append(lines, fmt.Sprintf("Baseline: %s", baseline))
	}
	if severity := getString(p, "severity", ""); severity != "" {
		lines = append(lines, fmt.Sprintf("Severity: %s", severity))
	}
	if owner := getString(p, "owner_role", ""); owner != "" {
		lines = append(lines, fmt.Sprintf("Owner: %s", owner))
	}

	return strings.Join(lines, "\n"), nil
}

// getChatID returns the configured chat ID, preferring the explicit config over payload.
func (a *FeishuAdapter) getChatID() string {
	if a.config.ChatID != "" {
		return a.config.ChatID
	}
	return ""
}

// ParsePayloadJSON parses a JSON string into a map. Returns nil on failure.
func ParsePayloadJSON(payloadJSON string) map[string]interface{} {
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(payloadJSON), &result); err != nil {
		return nil
	}
	return result
}

// Compile-time assertion that *FeishuAdapter satisfies action.ActionExecutor.
var _ action.ActionExecutor = (*FeishuAdapter)(nil)
