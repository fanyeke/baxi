package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// registerStatusTools registers all system status related MCP tools.
func (s *Server) registerStatusTools() {
	// Tool: get_system_health
	getStatusTool := mcp.NewTool(
		ToolGetSystemHealth,
		mcp.WithDescription("Get the current system health including pipeline state, alert counts, and recent events"),
	)
	s.server.AddTool(getStatusTool, s.handleGetSystemStatus)

	if isLegacyToolsEnabled() {
		legacyGetStatusTool := mcp.NewTool(
			LegacyGetSystemStatus,
			mcp.WithDescription("Get the current system status including pipeline state, alert counts, table row counts, and recent errors"),
		)
		s.server.AddTool(legacyGetStatusTool, s.handleGetSystemStatus)
	}

	// Tool: search_records
	searchObjectsTool := mcp.NewTool(
		ToolSearchRecords,
		mcp.WithDescription("Search for objects by type and query string"),
		mcp.WithString("object_type", mcp.Required(), mcp.Description("The type of object to search for (e.g., 'order', 'seller', 'category')")),
		mcp.WithString("query", mcp.Required(), mcp.Description("The search query string")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results to return (default 20)")),
		mcp.WithNumber("offset", mcp.Description("Offset for pagination (default 0)")),
	)
	s.server.AddTool(searchObjectsTool, s.handleSearchObjects)

	if isLegacyToolsEnabled() {
		legacySearchObjectsTool := mcp.NewTool(
			LegacySearchObjects,
			mcp.WithDescription("Search for objects by type and query string"),
			mcp.WithString("object_type", mcp.Required(), mcp.Description("The type of object to search for (e.g., 'order', 'seller', 'category')")),
			mcp.WithString("query", mcp.Required(), mcp.Description("The search query string")),
			mcp.WithNumber("limit", mcp.Description("Maximum number of results to return (default 20)")),
			mcp.WithNumber("offset", mcp.Description("Offset for pagination (default 0)")),
		)
		s.server.AddTool(legacySearchObjectsTool, s.handleSearchObjects)
	}
}

// handleGetSystemStatus handles the get_system_status tool.
func (s *Server) handleGetSystemStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	status, err := s.statusSvc.GetStatus(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get system status: %v", err)), nil
	}

	result := map[string]interface{}{
		"alert_count": status.AlertCount,
	}

	if status.PipelineRun != nil {
		pipelineMap := map[string]interface{}{
			"run_id":       status.PipelineRun.RunID,
			"run_type":     status.PipelineRun.RunType,
			"mode":         status.PipelineRun.Mode,
			"status":       status.PipelineRun.Status,
			"started_at":   status.PipelineRun.StartedAt,
			"input_count":  status.PipelineRun.InputCount,
			"output_count": status.PipelineRun.OutputCount,
		}
		if status.PipelineRun.FinishedAt != nil {
			pipelineMap["finished_at"] = *status.PipelineRun.FinishedAt
		}
		if status.PipelineRun.ErrorMessage != nil {
			pipelineMap["error_message"] = *status.PipelineRun.ErrorMessage
		}
		result["pipeline_run"] = pipelineMap
	}

	if len(status.TableCounts) > 0 {
		tables := make([]map[string]interface{}, len(status.TableCounts))
		for i, tc := range status.TableCounts {
			tables[i] = map[string]interface{}{
				"table_name": tc.TableName,
				"row_count":  tc.RowCount,
			}
		}
		result["table_counts"] = tables
	}

	if len(status.RecentErrors) > 0 {
		result["recent_errors"] = status.RecentErrors
	}

	return mcp.NewToolResultJSON(result)
}

// handleSearchObjects handles the search_objects tool.
func (s *Server) handleSearchObjects(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	objectType, ok := args["object_type"].(string)
	if !ok || objectType == "" {
		return mcp.NewToolResultError("object_type is required"), nil
	}

	query, ok := args["query"].(string)
	if !ok || query == "" {
		return mcp.NewToolResultError("query is required"), nil
	}

	limit := 20
	if v, ok := args["limit"].(float64); ok && v > 0 {
		limit = int(v)
	}

	offset := 0
	if v, ok := args["offset"].(float64); ok && v >= 0 {
		offset = int(v)
	}

	searchResult, err := s.searchSvc.SearchObjects(ctx, objectType, query, limit, offset)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to search objects: %v", err)), nil
	}

	if searchResult.Items == nil {
		searchResult.Items = []map[string]interface{}{}
	}

	result := map[string]interface{}{
		"items": searchResult.Items,
		"total": searchResult.Total,
	}

	return mcp.NewToolResultJSON(result)
}
