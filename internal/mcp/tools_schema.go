package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// registerSchemaTools registers all action type MCP tools.
func (s *Server) registerSchemaTools() {
	// Tool: list_action_types (was list_action_schemas)
	listTool := mcp.NewTool(
		ToolListActionTypes,
		mcp.WithDescription("List all available action types"),
	)
	s.server.AddTool(listTool, s.handleListActionSchemas)

	if isLegacyToolsEnabled() {
		legacyListActionTypesTool := mcp.NewTool(
			LegacyListActionSchemas,
			mcp.WithDescription("List all available action schemas"),
		)
		s.server.AddTool(legacyListActionTypesTool, s.handleListActionSchemas)
	}

	// Tool: get_action_type (was get_action_schema)
	getTool := mcp.NewTool(
		ToolGetActionType,
		mcp.WithDescription("Get the details for a specific action type"),
		mcp.WithString("action_type", mcp.Required(), mcp.Description("The action type to retrieve")),
	)
	s.server.AddTool(getTool, s.handleGetActionSchema)

	if isLegacyToolsEnabled() {
		legacyGetActionTypeTool := mcp.NewTool(
			LegacyGetActionSchema,
			mcp.WithDescription("Get the schema for a specific action type"),
			mcp.WithString("action_type", mcp.Required(), mcp.Description("The action type to retrieve the schema for")),
		)
		s.server.AddTool(legacyGetActionTypeTool, s.handleGetActionSchema)
	}
}

// handleListActionSchemas handles the list_action_schemas tool.
func (s *Server) handleListActionSchemas(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	schemas, err := s.schemaSvc.ListActionSchemas(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list action schemas: %v", err)), nil
	}

	items := make([]map[string]interface{}, 0, len(schemas))
	for _, sc := range schemas {
		items = append(items, map[string]interface{}{
			"name":           sc.Name,
			"description":    sc.Description,
			"risk_level":     sc.RiskLevel,
			"payload_schema": sc.PayloadSchema,
			"allowed_by":     sc.AllowedBy,
			"adapter":        sc.Adapter,
		})
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"schemas": items,
	})
}

// handleGetActionSchema handles the get_action_schema tool.
func (s *Server) handleGetActionSchema(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	actionType, ok := args["action_type"].(string)
	if !ok || actionType == "" {
		return mcp.NewToolResultError("action_type is required"), nil
	}

	schema, err := s.schemaSvc.GetActionSchema(ctx, actionType)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get action schema: %v", err)), nil
	}
	if schema == nil {
		return mcp.NewToolResultError(fmt.Sprintf("action type %q not found", actionType)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"name":           schema.Name,
		"description":    schema.Description,
		"risk_level":     schema.RiskLevel,
		"payload_schema": schema.PayloadSchema,
		"allowed_by":     schema.AllowedBy,
		"adapter":        schema.Adapter,
	})
}
