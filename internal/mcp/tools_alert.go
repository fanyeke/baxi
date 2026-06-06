package mcp

import (
	"context"

	"baxi/internal/model"
	"github.com/mark3labs/mcp-go/mcp"
)

// registerAlertTools registers all alert-related MCP tools.
func (s *Server) registerAlertTools() {
	// Tool: list_alerts
	listAlertsTool := mcp.NewTool(
		ToolListAlerts,
		mcp.WithDescription("List alerts with optional filtering and sorting"),
		mcp.WithString("severity", mcp.Description("Filter by severity (e.g., 'low', 'medium', 'high', 'critical')")),
		mcp.WithString("status", mcp.Description("Filter by status (e.g., 'open', 'acknowledged', 'resolved')")),
		mcp.WithString("object_type", mcp.Description("Filter by object type")),
		mcp.WithString("rule_id", mcp.Description("Filter by rule ID")),
		mcp.WithString("sort", mcp.Description("Sort field (default: 'created_at desc')")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of alerts to return (default 20)")),
		mcp.WithNumber("offset", mcp.Description("Offset for pagination (default 0)")),
	)
	s.server.AddTool(listAlertsTool, s.handleListAlerts)
	if isLegacyToolsEnabled() && LegacyListAlerts != ToolListAlerts {
		legacyTool := mcp.NewTool(
			LegacyListAlerts,
			mcp.WithDescription("List alerts with optional filtering and sorting"),
			mcp.WithString("severity", mcp.Description("Filter by severity (e.g., 'low', 'medium', 'high', 'critical')")),
			mcp.WithString("status", mcp.Description("Filter by status (e.g., 'open', 'acknowledged', 'resolved')")),
			mcp.WithString("object_type", mcp.Description("Filter by object type")),
			mcp.WithString("rule_id", mcp.Description("Filter by rule ID")),
			mcp.WithString("sort", mcp.Description("Sort field (default: 'created_at desc')")),
			mcp.WithNumber("limit", mcp.Description("Maximum number of alerts to return (default 20)")),
			mcp.WithNumber("offset", mcp.Description("Offset for pagination (default 0)")),
		)
		s.server.AddTool(legacyTool, s.handleListAlerts)
	}
}

// handleListAlerts handles the list_alerts tool.
func (s *Server) handleListAlerts(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	filters := model.AlertFilters{}
	args := req.GetArguments()

	if v, ok := args["severity"].(string); ok && v != "" {
		filters.Severity = v
	}
	if v, ok := args["status"].(string); ok && v != "" {
		filters.Status = v
	}
	if v, ok := args["object_type"].(string); ok && v != "" {
		filters.ObjectType = v
	}
	if v, ok := args["rule_id"].(string); ok && v != "" {
		filters.RuleID = v
	}

	sort := "created_at desc"
	if v, ok := args["sort"].(string); ok && v != "" {
		sort = v
	}

	limit := 20
	if v, ok := args["limit"].(float64); ok && v > 0 {
		limit = int(v)
	}

	offset := 0
	if v, ok := args["offset"].(float64); ok && v >= 0 {
		offset = int(v)
	}

	alertList, err := s.alertSvc.ListAlerts(ctx, filters, sort, limit, offset)
	if err != nil {
		return mcp.NewToolResultError(SanitizeErrorf("Failed to list alerts: %v", err)), nil
	}

	alerts := make([]map[string]interface{}, len(alertList.Items))
	for i, a := range alertList.Items {
		alerts[i] = map[string]interface{}{
			"event_id":    a.EventID,
			"rule_id":     a.RuleID,
			"event_date":  a.EventDate,
			"severity":    a.Severity,
			"metric_name": a.MetricName,
			"object_type": a.ObjectType,
			"object_id":   a.ObjectID,
			"status":      a.Status,
		}
		if a.CurrentValue != nil {
			alerts[i]["current_value"] = *a.CurrentValue
		}
		if a.BaselineValue != nil {
			alerts[i]["baseline_value"] = *a.BaselineValue
		}
		if a.ChangeRate != nil {
			alerts[i]["change_rate"] = *a.ChangeRate
		}
		if a.ImpactScore != nil {
			alerts[i]["impact_score"] = *a.ImpactScore
		}
	}

	result := map[string]interface{}{
		"alerts": alerts,
		"total":  alertList.Total,
	}

	return mcp.NewToolResultJSON(result)
}
