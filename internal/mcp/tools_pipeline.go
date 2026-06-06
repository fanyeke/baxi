package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

// validPipelineConfigs defines the allowed pipeline configuration names.
// Any config value not in this list is rejected (Phase 10 containment).
var validPipelineConfigs = map[string]bool{
	"full":                   true,
	"ingest_raw":             true,
	"build_dwd":              true,
	"build_metrics":          true,
	"detect_alerts":          true,
	"generate_recommendations": true,
	"generate_tasks":         true,
	"create_outbox":          true,
}

// registerPipelineTools registers all pipeline-related MCP tools.
func (s *Server) registerPipelineTools() {
	// Tool: process_data
	runPipelineTool := mcp.NewTool(
		ToolProcessData,
		mcp.WithDescription("Process data with the specified configuration"),
		mcp.WithString("config", mcp.Required(), mcp.Description("Pipeline configuration name")),
	)
	s.server.AddTool(runPipelineTool, s.handleRunPipeline)

	if isLegacyToolsEnabled() {
		legacyTool := mcp.NewTool(
			LegacyRunPipeline,
			mcp.WithDescription("Run a data pipeline with the specified configuration"),
			mcp.WithString("config", mcp.Required(), mcp.Description("The pipeline configuration name or path")),
			mcp.WithString("data_dir", mcp.Description("Directory containing CSV data files. Defaults to ./data/raw")),
		)
		s.server.AddTool(legacyTool, s.handleRunPipeline)
	}
}

// handleRunPipeline handles the run_pipeline tool.
func (s *Server) handleRunPipeline(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	config, ok := args["config"].(string)
	if !ok || config == "" {
		return mcp.NewToolResultError("config is required"), nil
	}

	// Validate config against allowlist (Phase 10 containment)
	if !validPipelineConfigs[config] {
		valid := make([]string, 0, len(validPipelineConfigs))
		for c := range validPipelineConfigs {
			valid = append(valid, c)
		}
		return mcp.NewToolResultError(SanitizeErrorf("invalid config %q. Valid options: %v", config, valid)), nil
	}

	// data_dir is fixed to the built-in path (not user-specifiable)
	dataDir := "./data/raw"

	resultID, err := s.pipelineRunner.Run(ctx, config, dataDir)
	if err != nil {
		return mcp.NewToolResultError(SanitizeErrorf("Failed to run pipeline: %v", err)), nil
	}

	result := map[string]interface{}{
		"config":    config,
		"result_id": resultID,
		"status":    "started",
	}

	return mcp.NewToolResultJSON(result)
}
