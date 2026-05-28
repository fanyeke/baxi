package adapter

import (
	"context"
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"baxi/internal/action"
	"github.com/stretchr/testify/require"
)

func TestCLIAdapter_DryRunTrue(t *testing.T) {
	ctx := context.Background()
	adapter := NewCLIAdapter(CLIConfig{
		LogPath: "/tmp/test_cli_dispatch_log.csv",
		Enabled: true,
	})

	proposal := action.ActionProposal{
		ProposalID: "prop-001",
		CaseID:     "case-001",
		ActionType: "notify_owner",
		Payload: map[string]interface{}{
			"rule_id": "rule_123",
		},
	}

	result, err := adapter.Execute(ctx, proposal, true)
	require.NoError(t, err)
	require.True(t, result.Success)
	require.True(t, result.DryRun)
	require.NotNil(t, result.DispatchPayload)
	require.Equal(t, "rule_123", result.DispatchPayload["rule_id"])
	require.Equal(t, "python3 scripts/run_alert_detection.py --rule rule_123 --investigate", result.DispatchPayload["command"])
	require.Equal(t, "feishu", result.DispatchPayload["channel"])
}

func TestCLIAdapter_EmptyLogPath(t *testing.T) {
	ctx := context.Background()
	adapter := NewCLIAdapter(CLIConfig{
		LogPath: "",
		Enabled: true,
	})

	proposal := action.ActionProposal{
		ProposalID: "prop-002",
		CaseID:     "case-002",
		ActionType: "export_report",
		Payload: map[string]interface{}{
			"rule_id": "rule_456",
		},
	}

	result, err := adapter.Execute(ctx, proposal, false)
	require.Error(t, err)
	require.Equal(t, "cli log path not configured", err.Error())
	require.False(t, result.Success)
}

func TestCLIAdapter_ExecuteSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "local_cli_dispatch_log.csv")

	ctx := context.Background()
	adapter := NewCLIAdapter(CLIConfig{
		LogPath: logPath,
		Enabled: true,
	})

	proposal := action.ActionProposal{
		ProposalID: "prop-003",
		CaseID:     "case-003",
		ActionType: "create_outbox_message",
		Title:      "Test CLI dispatch",
		Payload: map[string]interface{}{
			"rule_id": "my_rule-01",
		},
	}

	result, err := adapter.Execute(ctx, proposal, false)
	require.NoError(t, err)
	require.True(t, result.Success)
	require.False(t, result.DryRun)
	require.NotNil(t, result.DispatchPayload)
	require.Equal(t, logPath, result.OutboxEventID)
	require.Equal(t, "my_rule-01", result.DispatchPayload["rule_id"])

	// Verify CSV content
	records := readCSV(t, logPath)
	require.Len(t, records, 2) // header + 1 row
	require.Equal(t, []string{"timestamp", "outbox_id", "command", "rule_id", "status"}, records[0])
	require.Equal(t, "prop-003", records[1][1])
	require.Equal(t, "python3 scripts/run_alert_detection.py --rule my_rule-01 --investigate", records[1][2])
	require.Equal(t, "my_rule-01", records[1][3])
	require.Equal(t, "dispatched", records[1][4])
}

func TestCLIAdapter_MultipleDispatchesAppend(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "local_cli_dispatch_log.csv")

	ctx := context.Background()
	adapter := NewCLIAdapter(CLIConfig{
		LogPath: logPath,
		Enabled: true,
	})

	for i := 1; i <= 3; i++ {
		proposal := action.ActionProposal{
			ProposalID: "prop-00" + string(rune('0'+i)),
			CaseID:     "case-00" + string(rune('0'+i)),
			ActionType: "notify_owner",
			Payload: map[string]interface{}{
				"rule_id": "rule_" + string(rune('0'+i)),
			},
		}
		_, err := adapter.Execute(ctx, proposal, false)
		require.NoError(t, err)
	}

	records := readCSV(t, logPath)
	require.Len(t, records, 4) // header + 3 rows
	require.Equal(t, []string{"timestamp", "outbox_id", "command", "rule_id", "status"}, records[0])
	require.Equal(t, "rule_1", records[1][3])
	require.Equal(t, "rule_2", records[2][3])
	require.Equal(t, "rule_3", records[3][3])
}

func TestCLIAdapter_MissingRuleID(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "local_cli_dispatch_log.csv")

	ctx := context.Background()
	adapter := NewCLIAdapter(CLIConfig{
		LogPath: logPath,
		Enabled: true,
	})

	proposal := action.ActionProposal{
		ProposalID: "prop-004",
		CaseID:     "case-004",
		ActionType: "export_report",
		Payload:    map[string]interface{}{},
	}

	result, err := adapter.Execute(ctx, proposal, false)
	require.NoError(t, err)
	require.True(t, result.Success)
	require.Equal(t, "unknown", result.DispatchPayload["rule_id"])

	records := readCSV(t, logPath)
	require.Len(t, records, 2)
	require.Equal(t, "unknown", records[1][3])
	require.Equal(t, "python3 scripts/run_alert_detection.py --rule unknown --investigate", records[1][2])
}

func TestCLIAdapter_NilPayload(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "local_cli_dispatch_log.csv")

	ctx := context.Background()
	adapter := NewCLIAdapter(CLIConfig{
		LogPath: logPath,
		Enabled: true,
	})

	proposal := action.ActionProposal{
		ProposalID: "prop-005",
		CaseID:     "case-005",
		ActionType: "export_report",
		Payload:    nil,
	}

	result, err := adapter.Execute(ctx, proposal, false)
	require.NoError(t, err)
	require.True(t, result.Success)
	require.Equal(t, "unknown", result.DispatchPayload["rule_id"])
}

func TestCLIAdapter_InvalidRuleIDTooLong(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "local_cli_dispatch_log.csv")

	ctx := context.Background()
	adapter := NewCLIAdapter(CLIConfig{
		LogPath: logPath,
		Enabled: true,
	})

	proposal := action.ActionProposal{
		ProposalID: "prop-006",
		CaseID:     "case-006",
		ActionType: "export_report",
		Payload: map[string]interface{}{
			"rule_id": strings.Repeat("a", 65),
		},
	}

	result, err := adapter.Execute(ctx, proposal, false)
	require.NoError(t, err)
	require.True(t, result.Success)
	require.Equal(t, "unknown", result.DispatchPayload["rule_id"])
}

func TestCLIAdapter_InvalidRuleIDSpecialChars(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "local_cli_dispatch_log.csv")

	ctx := context.Background()
	adapter := NewCLIAdapter(CLIConfig{
		LogPath: logPath,
		Enabled: true,
	})

	proposal := action.ActionProposal{
		ProposalID: "prop-007",
		CaseID:     "case-007",
		ActionType: "export_report",
		Payload: map[string]interface{}{
			"rule_id": "rule@123",
		},
	}

	result, err := adapter.Execute(ctx, proposal, false)
	require.NoError(t, err)
	require.True(t, result.Success)
	require.Equal(t, "unknown", result.DispatchPayload["rule_id"])
}

func TestCLIAdapter_EmptyRuleID(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "local_cli_dispatch_log.csv")

	ctx := context.Background()
	adapter := NewCLIAdapter(CLIConfig{
		LogPath: logPath,
		Enabled: true,
	})

	proposal := action.ActionProposal{
		ProposalID: "prop-008",
		CaseID:     "case-008",
		ActionType: "export_report",
		Payload: map[string]interface{}{
			"rule_id": "",
		},
	}

	result, err := adapter.Execute(ctx, proposal, false)
	require.NoError(t, err)
	require.True(t, result.Success)
	require.Equal(t, "unknown", result.DispatchPayload["rule_id"])
}

func TestCLIAdapter_RuleIDNotString(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "local_cli_dispatch_log.csv")

	ctx := context.Background()
	adapter := NewCLIAdapter(CLIConfig{
		LogPath: logPath,
		Enabled: true,
	})

	proposal := action.ActionProposal{
		ProposalID: "prop-009",
		CaseID:     "case-009",
		ActionType: "export_report",
		Payload: map[string]interface{}{
			"rule_id": 123,
		},
	}

	result, err := adapter.Execute(ctx, proposal, false)
	require.NoError(t, err)
	require.True(t, result.Success)
	require.Equal(t, "unknown", result.DispatchPayload["rule_id"])
}

func TestCLIAdapter_DryRunNoLogWritten(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "should_not_exist.csv")

	ctx := context.Background()
	adapter := NewCLIAdapter(CLIConfig{
		LogPath: logPath,
		Enabled: true,
	})

	proposal := action.ActionProposal{
		ProposalID: "prop-010",
		CaseID:     "case-010",
		ActionType: "export_report",
		Payload: map[string]interface{}{
			"rule_id": "rule_010",
		},
	}

	result, err := adapter.Execute(ctx, proposal, true)
	require.NoError(t, err)
	require.True(t, result.Success)
	require.True(t, result.DryRun)

	_, err = os.Stat(logPath)
	require.True(t, os.IsNotExist(err), "log file should not be created in dry-run mode")
}

func TestCLIAdapter_UnknownActionType(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "local_cli_dispatch_log.csv")

	ctx := context.Background()
	adapter := NewCLIAdapter(CLIConfig{
		LogPath: logPath,
		Enabled: true,
	})

	proposal := action.ActionProposal{
		ProposalID: "prop-011",
		CaseID:     "case-011",
		ActionType: "some_unknown_action",
		Payload: map[string]interface{}{
			"rule_id": "rule_011",
		},
	}

	result, err := adapter.Execute(ctx, proposal, false)
	require.NoError(t, err)
	require.True(t, result.Success)
	require.Equal(t, "unknown", result.DispatchPayload["channel"])
}

func TestCLIAdapter_ValidateRuleID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "unknown"},
		{"valid_rule", "valid_rule"},
		{"rule-123", "rule-123"},
		{"rule_456", "rule_456"},
		{"Rule789", "Rule789"},
		{strings.Repeat("a", 64), strings.Repeat("a", 64)},
		{strings.Repeat("a", 65), "unknown"},
		{"rule@bad", "unknown"},
		{"rule.bad", "unknown"},
		{"rule bad", "unknown"},
		{"rule/bad", "unknown"},
		{"rule:bad", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := validateRuleID(tt.input)
			require.Equal(t, tt.expected, got)
		})
	}
}

func readCSV(t *testing.T, path string) [][]string {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	require.NoError(t, err)
	return records
}
