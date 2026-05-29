package mcp

import (
	"context"
	"fmt"

	"baxi/internal/action"
	"github.com/mark3labs/mcp-go/mcp"
)

// registerActionTools registers all action-related MCP tools.
func (s *Server) registerActionTools() {
	// Tool: execute_proposal
	executeTool := mcp.NewTool(
		"execute_proposal",
		mcp.WithDescription("Execute an approved action proposal"),
		mcp.WithString("proposal_id", mcp.Required(), mcp.Description("The ID of the proposal to execute")),
		mcp.WithBoolean("dry_run", mcp.Description("When true (default), simulate execution without side effects")),
	)
	s.server.AddTool(executeTool, s.handleExecuteProposal)

	// Tool: get_decision_context
	contextTool := mcp.NewTool(
		"get_decision_context",
		mcp.WithDescription("Get the full decision context for a case"),
		mcp.WithString("case_id", mcp.Required(), mcp.Description("The ID of the case to get context for")),
	)
	s.server.AddTool(contextTool, s.handleGetDecisionContext)
}

// handleExecuteProposal handles the execute_proposal tool.
func (s *Server) handleExecuteProposal(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	proposalID, ok := args["proposal_id"].(string)
	if !ok || proposalID == "" {
		return mcp.NewToolResultError("proposal_id is required"), nil
	}

	dryRun := true
	if v, ok := args["dry_run"].(bool); ok {
		dryRun = v
	}

	result, err := s.executeSvc.ExecuteProposal(ctx, s.pool, proposalID, "mcp_user", action.WithDryRun(dryRun))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to execute proposal: %v", err)), nil
	}

	payload := map[string]interface{}{
		"proposal_id": proposalID,
		"success":     result.Success,
		"dry_run":     result.DryRun,
	}
	if result.OutboxEventID != "" {
		payload["outbox_event_id"] = result.OutboxEventID
	}
	if result.Error != "" {
		payload["error"] = result.Error
	}

	return mcp.NewToolResultJSON(payload)
}

// handleGetDecisionContext handles the get_decision_context tool.
func (s *Server) handleGetDecisionContext(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	caseID, ok := args["case_id"].(string)
	if !ok || caseID == "" {
		return mcp.NewToolResultError("case_id is required"), nil
	}

	decisionCtx, err := s.contextBuilder.BuildDecisionContext(ctx, caseID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to build decision context: %v", err)), nil
	}

	result := map[string]interface{}{
		"case_id":           decisionCtx.DecisionCaseID,
		"trigger":           decisionCtx.Trigger,
		"object_context":    decisionCtx.ObjectContext,
		"governance":        decisionCtx.Governance,
		"allowed_actions":   decisionCtx.AllowedActions,
		"forbidden_actions": decisionCtx.ForbiddenActions,
	}
	if decisionCtx.SourceType != nil {
		result["source_type"] = *decisionCtx.SourceType
	}
	if decisionCtx.SourceID != nil {
		result["source_id"] = *decisionCtx.SourceID
	}
	if decisionCtx.Policy != nil {
		result["policy"] = decisionCtx.Policy
	}

	return mcp.NewToolResultJSON(result)
}
