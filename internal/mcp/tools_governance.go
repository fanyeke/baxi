package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// registerGovernanceTools registers all governance-related MCP tools.
func (s *Server) registerGovernanceTools() {
	// Tool: check_access
	checkAccessTool := mcp.NewTool(
		"check_access",
		mcp.WithDescription("Check if a role has access to perform an action on an object type"),
		mcp.WithString("role", mcp.Required(), mcp.Description("The role to check access for")),
		mcp.WithString("object_type", mcp.Required(), mcp.Description("The type of object to check access on")),
		mcp.WithString("action", mcp.Required(), mcp.Description("The action to check (e.g., 'read', 'write', 'delete')")),
	)
	s.server.AddTool(checkAccessTool, s.handleCheckAccess)

	// Tool: get_classification
	getClassificationTool := mcp.NewTool(
		"get_classification",
		mcp.WithDescription("Get classification information for a field path"),
		mcp.WithString("field_path", mcp.Required(), mcp.Description("The field path to get classification for (e.g., 'user.email')")),
	)
	s.server.AddTool(getClassificationTool, s.handleGetClassification)
}

// handleCheckAccess handles the check_access tool.
func (s *Server) handleCheckAccess(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	role, ok := args["role"].(string)
	if !ok || role == "" {
		return mcp.NewToolResultError("role is required"), nil
	}

	objectType, ok := args["object_type"].(string)
	if !ok || objectType == "" {
		return mcp.NewToolResultError("object_type is required"), nil
	}

	action, ok := args["action"].(string)
	if !ok || action == "" {
		return mcp.NewToolResultError("action is required"), nil
	}

	accessDecision, err := s.govSvc.CheckAccess(ctx, role, objectType, action)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to check access: %v", err)), nil
	}

	result := map[string]interface{}{
		"role":         role,
		"object_type":  objectType,
		"action":       action,
		"access":       string(*accessDecision),
	}

	return mcp.NewToolResultJSON(result)
}

// handleGetClassification handles the get_classification tool.
func (s *Server) handleGetClassification(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	fieldPath, ok := args["field_path"].(string)
	if !ok || fieldPath == "" {
		return mcp.NewToolResultError("field_path is required"), nil
	}

	classification, err := s.govSvc.GetClassification(ctx, fieldPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get classification: %v", err)), nil
	}

	result := map[string]interface{}{
		"field_path": fieldPath,
	}

	if classification.Levels != nil {
		result["levels"] = classification.Levels
	}
	if classification.Resources != nil {
		resources := make([]map[string]interface{}, len(classification.Resources))
		for i, r := range classification.Resources {
			resources[i] = map[string]interface{}{
				"resource":       r.Resource,
				"classification": r.Classification,
			}
		}
		result["resources"] = resources
	}

	return mcp.NewToolResultJSON(result)
}
