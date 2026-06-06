package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerContextTools() {
	buildContextTool := mcp.NewTool(
		ToolAnalyzeSituation,
		mcp.WithDescription("Build an LLM-safe context envelope for analyzing a situation using a ContextRecipe"),
		mcp.WithString("case_id", mcp.Required(), mcp.Description("The ID of the evaluation to build context for")),
		mcp.WithString("recipe_id", mcp.Description("Optional: specific recipe ID to use (otherwise matched by rule_id)")),
	)
	s.server.AddTool(buildContextTool, s.handleBuildContext)
	if isLegacyToolsEnabled() {
		legacyTool := mcp.NewTool(
			LegacyBuildContext,
			mcp.WithDescription("Build an LLM-safe context envelope for a decision case using a ContextRecipe"),
			mcp.WithString("case_id", mcp.Required(), mcp.Description("The ID of the decision case to build context for")),
			mcp.WithString("recipe_id", mcp.Description("Optional: specific recipe ID to use (otherwise matched by rule_id)")),
		)
		s.server.AddTool(legacyTool, s.handleBuildContext)
	}
}

func (s *Server) handleBuildContext(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	caseID, ok := args["case_id"].(string)
	if !ok || caseID == "" {
		return mcp.NewToolResultError("case_id is required"), nil
	}

	recipeID, _ := args["recipe_id"].(string)

	if s.buildContextSvc == nil {
		return mcp.NewToolResultError("build_context service is not available"), nil
	}

	envelope, err := s.buildContextSvc.BuildEnvelope(ctx, caseID, recipeID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to build context: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"case_id":           envelope.CaseID,
		"alert_id":          envelope.AlertID,
		"schema_version":    envelope.SchemaVersion,
		"context_hash":      envelope.ContextHash,
		"built_at":          envelope.BuiltAt,
		"trigger":           envelope.Trigger,
		"object_context":    envelope.ObjectContext,
		"evidence":          envelope.Evidence,
		"allowed_actions":   envelope.AllowedActions,
		"forbidden_actions": envelope.ForbiddenActions,
		"governance":        envelope.Governance,
		"redaction_summary": envelope.RedactionSummary,
		"config_versions":   envelope.ConfigVersions,
	})
}
