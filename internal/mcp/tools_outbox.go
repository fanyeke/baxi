package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

// registerOutboxTools registers outbox and pipeline status MCP tools.
func (s *Server) registerOutboxTools() {
	// Tool: list_events
	listTool := mcp.NewTool(
		ToolListEvents,
		mcp.WithDescription("List outbox events with optional status filter"),
		mcp.WithString("status", mcp.Description("Filter by status (e.g., 'pending', 'dispatched', 'failed')")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of events to return (default 20)")),
		mcp.WithNumber("offset", mcp.Description("Offset for pagination (default 0)")),
	)
	s.server.AddTool(listTool, s.handleListOutboxEvents)

	if isLegacyToolsEnabled() {
		legacyListTool := mcp.NewTool(
			LegacyListOutboxEvents,
			mcp.WithDescription("List outbox events with optional status filter"),
			mcp.WithString("status", mcp.Description("Filter by status (e.g., 'pending', 'dispatched', 'failed')")),
			mcp.WithNumber("limit", mcp.Description("Maximum number of events to return (default 20)")),
			mcp.WithNumber("offset", mcp.Description("Offset for pagination (default 0)")),
		)
		s.server.AddTool(legacyListTool, s.handleListOutboxEvents)
	}

	// Tool: get_processing_status
	statusTool := mcp.NewTool(
		ToolGetProcessingStatus,
		mcp.WithDescription("Get pipeline status including last run and recent runs"),
	)
	s.server.AddTool(statusTool, s.handleGetPipelineStatus)

	if isLegacyToolsEnabled() {
		legacyStatusTool := mcp.NewTool(
			LegacyGetPipelineStatus,
			mcp.WithDescription("Get pipeline status including last run and recent runs"),
		)
		s.server.AddTool(legacyStatusTool, s.handleGetPipelineStatus)
	}
}

// handleListOutboxEvents handles the list_outbox_events tool.
func (s *Server) handleListOutboxEvents(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	status := ""
	if v, ok := args["status"].(string); ok {
		status = v
	}

	limit := 20
	if v, ok := args["limit"].(float64); ok && v > 0 {
		limit = int(v)
	}

	offset := 0
	if v, ok := args["offset"].(float64); ok && v >= 0 {
		offset = int(v)
	}

	events, total, err := s.outboxSvc.ListOutboxEvents(ctx, status, limit, offset)
	if err != nil {
		return mcp.NewToolResultError(SanitizeErrorf("Failed to list outbox events: %v", err)), nil
	}

	eventList := make([]map[string]interface{}, len(events))
	for i, e := range events {
		eventList[i] = map[string]interface{}{
			"event_id":          e.OutboxID,
			"event_type":        e.EventType,
			"status":            e.Status,
			"source_type":       e.SourceType,
			"created_at":        e.CreatedAt,
			"dispatch_attempts": e.DispatchAttempts,
		}
	}

	result := map[string]interface{}{
		"events": eventList,
		"total":  total,
	}

	return mcp.NewToolResultJSON(result)
}

// handleGetPipelineStatus handles the get_pipeline_status tool.
func (s *Server) handleGetPipelineStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	lastRun, err := s.pipelineInfoSvc.GetLastRunStatus(ctx)
	if err != nil {
		return mcp.NewToolResultError(SanitizeErrorf("Failed to get last run status: %v", err)), nil
	}

	runs, err := s.pipelineInfoSvc.ListRuns(ctx, 10)
	if err != nil {
		return mcp.NewToolResultError(SanitizeErrorf("Failed to list runs: %v", err)), nil
	}

	runList := make([]map[string]interface{}, len(runs))
	for i, r := range runs {
		item := map[string]interface{}{
			"run_id":      r.RunID,
			"run_type":    r.RunType,
			"mode":        r.Mode,
			"status":      r.Status,
			"started_at":  r.StartedAt,
			"input_count": r.InputCount,
		}
		if r.FinishedAt != nil {
			item["finished_at"] = *r.FinishedAt
		}
		if r.ErrorMessage != nil {
			item["error_message"] = *r.ErrorMessage
		}
		runList[i] = item
	}

	result := map[string]interface{}{}
	if lastRun != nil {
		lastRunMap := map[string]interface{}{
			"run_id":      lastRun.RunID,
			"run_type":    lastRun.RunType,
			"mode":        lastRun.Mode,
			"status":      lastRun.Status,
			"started_at":  lastRun.StartedAt,
			"input_count": lastRun.InputCount,
		}
		if lastRun.FinishedAt != nil {
			lastRunMap["finished_at"] = *lastRun.FinishedAt
		}
		if lastRun.ErrorMessage != nil {
			lastRunMap["error_message"] = *lastRun.ErrorMessage
		}
		result["last_run"] = lastRunMap
	} else {
		result["last_run"] = nil
	}
	result["runs"] = runList

	return mcp.NewToolResultJSON(result)
}
