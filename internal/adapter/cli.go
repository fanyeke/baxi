package adapter

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"baxi/internal/action"
)

var ruleIDRe = regexp.MustCompile("^[a-zA-Z0-9_-]+$")

func validateRuleID(ruleID string) string {
	if ruleID == "" {
		return "unknown"
	}
	if len(ruleID) > 64 {
		return "unknown"
	}
	if !ruleIDRe.MatchString(ruleID) {
		return "unknown"
	}
	return ruleID
}

func buildCommand(ruleID string) string {
	return fmt.Sprintf("python3 scripts/run_alert_detection.py --rule %s --investigate", ruleID)
}

// CLIAdapter dispatches action proposals to a local CLI audit log.
type CLIAdapter struct {
	config CLIConfig
}

// NewCLIAdapter creates a new CLIAdapter with the given config.
func NewCLIAdapter(config CLIConfig) *CLIAdapter {
	return &CLIAdapter{config: config}
}

// Execute implements action.ActionExecutor. In dry-run mode it returns a
// successful preview result without writing to the log. When the log path is
// empty it returns an error. Otherwise it appends an audit entry to the CSV
// log and returns success.
func (a *CLIAdapter) Execute(ctx context.Context, proposal action.ActionProposal, dryRun bool) (action.ExecutionResult, error) {
	ruleID := "unknown"
	if proposal.Payload != nil {
		if v, ok := proposal.Payload["rule_id"].(string); ok {
			ruleID = validateRuleID(v)
		}
	}

	command := buildCommand(ruleID)

	payload := map[string]interface{}{
		"proposal_id": proposal.ProposalID,
		"case_id":     proposal.CaseID,
		"action_type": proposal.ActionType,
		"channel":     ActionChannel(proposal.ActionType),
		"dry_run":     dryRun,
		"rule_id":     ruleID,
		"command":     command,
	}

	if dryRun {
		return action.ExecutionResult{
			Success:         true,
			DryRun:          true,
			DispatchPayload: payload,
		}, nil
	}

	if a.config.LogPath == "" {
		return action.ExecutionResult{}, fmt.Errorf("cli log path not configured")
	}

	if err := a.writeLog(proposal.ProposalID, command, ruleID); err != nil {
		return action.ExecutionResult{}, fmt.Errorf("cli log write failed: %w", err)
	}

	return action.ExecutionResult{
		Success:         true,
		DryRun:          false,
		DispatchPayload: payload,
		OutboxEventID:   a.config.LogPath,
	}, nil
}

func (a *CLIAdapter) writeLog(outboxID, command, ruleID string) error {
	dir := filepath.Dir(a.config.LogPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	writeHeader := false
	if _, err := os.Stat(a.config.LogPath); os.IsNotExist(err) {
		writeHeader = true
	}

	f, err := os.OpenFile(a.config.LogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	if writeHeader {
		if err := writer.Write([]string{"timestamp", "outbox_id", "command", "rule_id", "status"}); err != nil {
			return err
		}
	}

	record := []string{
		time.Now().Format(time.RFC3339),
		outboxID,
		command,
		ruleID,
		"dispatched",
	}
	return writer.Write(record)
}

// Compile-time assertion that *CLIAdapter satisfies action.ActionExecutor.
var _ action.ActionExecutor = (*CLIAdapter)(nil)
