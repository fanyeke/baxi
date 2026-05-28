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

	resultID, err := s.pipelineRunner.Run(ctx, config)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to run pipeline: %v", err)), nil
	}

	result := map[string]interface{}{
		"config":    config,
		"result_id": resultID,
		"status":    "started",
	}

	return mcp.NewToolResultJSON(result)
}
