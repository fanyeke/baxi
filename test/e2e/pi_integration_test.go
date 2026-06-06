//go:build integration

package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"

	"baxi/internal/testutil"
)

// mcpBinaryPath returns the path to the baxi-mcp binary.
// It first checks the BAXI_MCP_BINARY env var, then falls back to building from source.
func mcpBinaryPath(t *testing.T) string {
	t.Helper()

	if p := os.Getenv("BAXI_MCP_BINARY"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// Fall back to /tmp/baxi-mcp (pre-built)
	if _, err := os.Stat("/tmp/baxi-mcp"); err == nil {
		return "/tmp/baxi-mcp"
	}

	t.Skip("baxi-mcp binary not found; set BAXI_MCP_BINARY or run: go build -o /tmp/baxi-mcp ./cmd/baxi-mcp")
	return ""
}

// migrationsDir returns the absolute path to the migrations directory.
func migrationsDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller failed")
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "migrations")
}

// setupMCPTest starts Postgres, runs migrations, launches baxi-mcp as a subprocess,
// and returns an initialized MCP client with cleanup function.
func setupMCPTest(t *testing.T) (*mcpclient.Client, func()) {
	t.Helper()
	ctx := context.Background()

	// 1. Start Postgres container
	pg, err := testutil.StartPostgres(ctx)
	require.NoError(t, err, "start postgres container")

	// 2. Run migrations
	err = pg.RunMigrations(ctx, migrationsDir(t))
	require.NoError(t, err, "run migrations")

	// 3. Launch baxi-mcp subprocess with stdio transport
	binary := mcpBinaryPath(t)
	connStr := pg.ConnectionString()
	env := append(os.Environ(), fmt.Sprintf("DATABASE_URL=%s", connStr))

	mcpClient, err := mcpclient.NewStdioMCPClient(binary, env)
	require.NoError(t, err, "start MCP server subprocess")

	// 4. Initialize the MCP connection
	initCtx, initCancel := context.WithTimeout(ctx, 30*time.Second)
	defer initCancel()

	initResult, err := mcpClient.Initialize(initCtx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			Capabilities:    mcp.ClientCapabilities{},
			ClientInfo: mcp.Implementation{
				Name:    "pi-integration-test",
				Version: "1.0.0",
			},
		},
	})
	require.NoError(t, err, "MCP initialize")
	require.NotNil(t, initResult, "initialize result")
	t.Logf("MCP server: %s v%s (protocol %s)",
		initResult.ServerInfo.Name, initResult.ServerInfo.Version, initResult.ProtocolVersion)

	cleanup := func() {
		_ = mcpClient.Close()
		_ = pg.Terminate(ctx)
	}
	return mcpClient, cleanup
}

// mcpCallTool is a helper that calls an MCP tool and returns the result.
func mcpCallTool(t *testing.T, client *mcpclient.Client, toolName string, args map[string]interface{}) *mcp.CallToolResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := client.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		},
	})
	require.NoError(t, err, "call tool %s", toolName)
	require.NotNil(t, result, "tool %s result", toolName)
	return result
}

// extractJSON extracts the first text content from a CallToolResult and unmarshals it.
func extractJSON(t *testing.T, result *mcp.CallToolResult, dest interface{}) {
	t.Helper()
	require.NotEmpty(t, result.Content, "result should have content")

	textContent, ok := mcp.AsTextContent(result.Content[0])
	require.True(t, ok, "first content should be TextContent")
	require.False(t, result.IsError, "tool returned error: %s", textContent.Text)

	err := json.Unmarshal([]byte(textContent.Text), dest)
	require.NoError(t, err, "unmarshal tool result JSON: %s", textContent.Text)
}

// ---------------------------------------------------------------------------
// Test: MCP Server startup, tool listing, and basic tool calls
// ---------------------------------------------------------------------------

func TestPiIntegration_Basic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	client, cleanup := setupMCPTest(t)
	defer cleanup()

	ctx := context.Background()

	// --- Step 1: List tools and verify all expected tools exist ---
	listResult, err := client.ListTools(ctx, mcp.ListToolsRequest{})
	require.NoError(t, err, "list tools")
	require.NotNil(t, listResult, "list tools result")

	registeredTools := make(map[string]bool)
	for _, tool := range listResult.Tools {
		registeredTools[tool.Name] = true
		t.Logf("registered tool: %s", tool.Name)
	}

	expectedTools := []string{
		"evaluate_case",
		"decide",
		"list_cases",
		"get_case",
		"list_proposals",
		"list_alerts",
		"check_permission",
		"get_classification",
		"process_data",
		"approve_proposal",
		"reject_proposal",
		"execute_proposal",
		"get_decision_context",
		"get_system_health",
		"search_records",
		"list_outbox_events",
		"get_pipeline_status",
	}

	for _, name := range expectedTools {
		require.True(t, registeredTools[name], "expected tool %q not found", name)
	}
	require.Equal(t, len(expectedTools), len(listResult.Tools),
		"tool count mismatch: expected %d, got %d", len(expectedTools), len(listResult.Tools))

	// --- Step 2: Call list_alerts — expect empty result (no data loaded) ---
	alertResult := mcpCallTool(t, client, "list_alerts", nil)
	var alertResp struct {
		Alerts []interface{} `json:"alerts"`
		Total  int           `json:"total"`
	}
	extractJSON(t, alertResult, &alertResp)
	require.Equal(t, 0, alertResp.Total, "list_alerts should return 0 total with no data")
	t.Logf("list_alerts: total=%d", alertResp.Total)

	// --- Step 3: Call get_system_health — verify it returns data ---
	statusResult := mcpCallTool(t, client, "get_system_health", nil)
	var statusResp map[string]interface{}
	extractJSON(t, statusResult, &statusResp)
	require.Contains(t, statusResp, "alert_count", "get_system_health should contain alert_count")
	t.Logf("get_system_health: %v", statusResp)

	// --- Step 4: Call list_outbox_events — expect empty result ---
	outboxResult := mcpCallTool(t, client, "list_outbox_events", nil)
	var outboxResp struct {
		Events []interface{} `json:"events"`
		Total  int           `json:"total"`
	}
	extractJSON(t, outboxResult, &outboxResp)
	require.Equal(t, 0, outboxResp.Total, "list_outbox_events should return 0 total with no data")
	t.Logf("list_outbox_events: total=%d", outboxResp.Total)

	// --- Step 5: Call get_pipeline_status — expect null last_run ---
	pipelineResult := mcpCallTool(t, client, "get_pipeline_status", nil)
	var pipelineResp map[string]interface{}
	extractJSON(t, pipelineResult, &pipelineResp)
	t.Logf("get_pipeline_status: %v", pipelineResp)
}

// ---------------------------------------------------------------------------
// Test: Tool parameter validation — missing required params should return error
// ---------------------------------------------------------------------------

func TestPiIntegration_ParameterValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	client, cleanup := setupMCPTest(t)
	defer cleanup()

	ctx := context.Background()

	// Test missing required param for evaluate_case (needs alert_id)
	result, err := client.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "evaluate_case",
			Arguments: map[string]interface{}{},
		},
	})
	require.NoError(t, err, "call should not error at protocol level")
	require.True(t, result.IsError, "should return isError for missing required param")
	textContent, ok := mcp.AsTextContent(result.Content[0])
	require.True(t, ok)
	t.Logf("evaluate_case missing alert_id: %s", textContent.Text)

	// Test missing required param for get_case (needs case_id)
	result, err = client.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get_case",
			Arguments: map[string]interface{}{},
		},
	})
	require.NoError(t, err)
	require.True(t, result.IsError, "should return isError for missing case_id")
	textContent, ok = mcp.AsTextContent(result.Content[0])
	require.True(t, ok)
	t.Logf("get_case missing case_id: %s", textContent.Text)

	// Test missing required param for approve_proposal (needs proposal_id + reviewer_id)
	result, err = client.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "approve_proposal",
			Arguments: map[string]interface{}{},
		},
	})
	require.NoError(t, err)
	require.True(t, result.IsError, "should return isError for missing proposal_id")
	textContent, ok = mcp.AsTextContent(result.Content[0])
	require.True(t, ok)
	t.Logf("approve_proposal missing params: %s", textContent.Text)

	// Test missing required param for execute_proposal (needs proposal_id)
	result, err = client.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "execute_proposal",
			Arguments: map[string]interface{}{},
		},
	})
	require.NoError(t, err)
	require.True(t, result.IsError, "should return isError for missing proposal_id")
	textContent, ok = mcp.AsTextContent(result.Content[0])
	require.True(t, ok)
	t.Logf("execute_proposal missing proposal_id: %s", textContent.Text)

	// Test missing required param for search_records (needs object_type + query)
	result, err = client.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "search_records",
			Arguments: map[string]interface{}{},
		},
	})
	require.NoError(t, err)
	require.True(t, result.IsError, "should return isError for missing object_type")
	textContent, ok = mcp.AsTextContent(result.Content[0])
	require.True(t, ok)
	t.Logf("search_records missing params: %s", textContent.Text)
}

// ---------------------------------------------------------------------------
// Test: Tool descriptions are present and non-empty
// ---------------------------------------------------------------------------

func TestPiIntegration_ToolDescriptions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	client, cleanup := setupMCPTest(t)
	defer cleanup()

	ctx := context.Background()

	listResult, err := client.ListTools(ctx, mcp.ListToolsRequest{})
	require.NoError(t, err, "list tools")

	for _, tool := range listResult.Tools {
		require.NotEmpty(t, tool.Description, "tool %s should have a description", tool.Name)
		t.Logf("tool: %s — %s", tool.Name, tool.Description)
	}
}
