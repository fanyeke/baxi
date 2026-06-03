package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// registerPipelineTools registers all pipeline-related MCP tools.
func (s *Server) registerPipelineTools() {
	// Tool: run_pipeline
	runPipelineTool := mcp.NewTool(
		"run_pipeline",
		mcp.WithDescription("Run a data pipeline with the specified configuration"),
		mcp.WithString("config", mcp.Required(), mcp.Description("The pipeline configuration name or path")),
		mcp.WithString("data_dir", mcp.Description("Directory containing CSV data files. Defaults to ./data/raw relative to the baxi project root")),
	)
	s.server.AddTool(runPipelineTool, s.handleRunPipeline)
}

// handleRunPipeline handles the run_pipeline tool.
func (s *Server) handleRunPipeline(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	config, ok := args["config"].(string)
	if !ok || config == "" {
		return mcp.NewToolResultError("config is required"), nil
	}

	dataDir := "./data/raw"
	if v, ok := args["data_dir"].(string); ok && v != "" {
		dataDir = v
	}

	resultID, err := s.pipelineRunner.Run(ctx, config, dataDir)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to run pipeline: %v", err)), nil
	}

	result := map[string]interface{}{
		"config":    config,
		"data_dir":  dataDir,
		"result_id": resultID,
		"status":    "started",
	}

	return mcp.NewToolResultJSON(result)
}
