package mcp

import (
	"context"
	"encoding/json"
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

	// Tool: propose_action
	proposeTool := mcp.NewTool(
		"propose_action",
		mcp.WithDescription("Propose an action on an object (creates a proposal for approval)"),
		mcp.WithString("object_type", mcp.Required(), mcp.Description("The type of the target object")),
		mcp.WithString("object_id", mcp.Required(), mcp.Description("The ID of the target object")),
		mcp.WithString("action_type", mcp.Required(), mcp.Description("The action type to propose")),
		mcp.WithString("params", mcp.Description("Optional JSON-encoded parameters for the action")),
	)
	s.server.AddTool(proposeTool, s.handleProposeAction)

	// Tool: get_decision_context
	contextTool := mcp.NewTool(
		"get_decision_context",
		mcp.WithDescription("Get the full decision context for a case"),
		mcp.WithString("case_id", mcp.Required(), mcp.Description("The ID of the case to get context for")),
	)
	s.server.AddTool(contextTool, s.handleGetDecisionContext)
}

// handleProposeAction handles the propose_action tool.
func (s *Server) handleProposeAction(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	objectType, ok := args["object_type"].(string)
	if !ok || objectType == "" {
		return mcp.NewToolResultError("object_type is required"), nil
	}

	objectID, ok := args["object_id"].(string)
	if !ok || objectID == "" {
		return mcp.NewToolResultError("object_id is required"), nil
	}

	actionType, ok := args["action_type"].(string)
	if !ok || actionType == "" {
		return mcp.NewToolResultError("action_type is required"), nil
	}

	var params map[string]interface{}
	if paramsRaw, ok := args["params"].(string); ok && paramsRaw != "" {
		if err := json.Unmarshal([]byte(paramsRaw), &params); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid JSON in params: %v", err)), nil
		}
	}

	result, err := s.ontologySvc.ProposeAction(ctx, objectType, objectID, actionType, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to propose action: %v", err)), nil
	}

	res := map[string]interface{}{
		"success":     result.Success,
		"action_type": result.ActionType,
		"object_type": result.ObjectType,
		"object_id":   result.ObjectID,
	}
	if result.Result != nil {
		if proposalID, ok := result.Result["proposal_id"].(string); ok {
			res["proposal_id"] = proposalID
		}
		if status, ok := result.Result["status"].(string); ok {
			res["status"] = status
		}
		if msg, ok := result.Result["message"].(string); ok {
			res["message"] = msg
		}
	}
	return mcp.NewToolResultJSON(res)
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
