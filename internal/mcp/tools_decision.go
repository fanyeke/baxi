package mcp

import (
	"context"
	"fmt"

	"baxi/internal/decision"
	"github.com/mark3labs/mcp-go/mcp"
)

// registerDecisionTools registers all decision-related MCP tools.
func (s *Server) registerDecisionTools() {
	// Tool: create_decision_case
	createCaseTool := mcp.NewTool(
		"create_decision_case",
		mcp.WithDescription("Create a new decision case from an alert"),
		mcp.WithString("alert_id", mcp.Required(), mcp.Description("The ID of the alert to create a case from")),
		mcp.WithString("created_by", mcp.Description("The user or system creating the case")),
	)
	s.server.AddTool(createCaseTool, s.handleCreateDecisionCase)

	// Tool: decide
	decideTool := mcp.NewTool(
		"decide",
		mcp.WithDescription("Generate a decision for a case, persisting action proposals"),
		mcp.WithString("case_id", mcp.Required(), mcp.Description("The ID of the case to generate a decision for")),
	)
	s.server.AddTool(decideTool, s.handleDecide)

	// Tool: resolve_case
	resolveCaseTool := mcp.NewTool(
		"resolve_case",
		mcp.WithDescription("Resolve a decision case with a resolution and optional comment"),
		mcp.WithString("case_id", mcp.Required(), mcp.Description("The ID of the case to resolve")),
		mcp.WithString("resolution", mcp.Required(), mcp.Description("Resolution type: approved, rejected, escalated")),
		mcp.WithString("comment", mcp.Description("Optional comment about the resolution")),
	)
	s.server.AddTool(resolveCaseTool, s.handleResolveCase)

	// Tool: list_cases
	listCasesTool := mcp.NewTool(
		"list_cases",
		mcp.WithDescription("List decision cases with optional filtering"),
		mcp.WithString("source_type", mcp.Description("Filter by source type (e.g., 'alert')")),
		mcp.WithString("source_id", mcp.Description("Filter by source ID")),
		mcp.WithString("status", mcp.Description("Filter by case status (e.g., 'created', 'decision_generated', 'closed')")),
		mcp.WithString("severity", mcp.Description("Filter by severity (e.g., 'low', 'medium', 'high', 'critical')")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of cases to return (default 20)")),
		mcp.WithNumber("offset", mcp.Description("Offset for pagination (default 0)")),
	)
	s.server.AddTool(listCasesTool, s.handleListCases)

	// Tool: get_case
	getCaseTool := mcp.NewTool(
		"get_case",
		mcp.WithDescription("Get details of a specific decision case"),
		mcp.WithString("case_id", mcp.Required(), mcp.Description("The ID of the case to retrieve")),
	)
	s.server.AddTool(getCaseTool, s.handleGetCase)

	// Tool: list_proposals
	listProposalsTool := mcp.NewTool(
		"list_proposals",
		mcp.WithDescription("List action proposals for a case"),
		mcp.WithString("case_id", mcp.Required(), mcp.Description("The ID of the case to list proposals for")),
	)
	s.server.AddTool(listProposalsTool, s.handleListProposals)
}

// handleCreateDecisionCase handles the create_decision_case tool.
func (s *Server) handleCreateDecisionCase(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	alertID, ok := args["alert_id"].(string)
	if !ok || alertID == "" {
		return mcp.NewToolResultError("alert_id is required"), nil
	}

	createdBy := "mcp_user"
	if v, ok := args["created_by"].(string); ok && v != "" {
		createdBy = v
	}

	decisionCase, err := s.decisionSvc.CreateCaseFromAlert(ctx, alertID, createdBy)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create decision case: %v", err)), nil
	}

	result := map[string]interface{}{
		"case_id":     decisionCase.CaseID,
		"alert_id":    decisionCase.AlertID,
		"status":      decisionCase.Status,
		"object_type": decisionCase.ObjectType,
		"object_id":   decisionCase.ObjectID,
		"severity":    decisionCase.Severity,
		"created_at":  decisionCase.CreatedAt,
	}

	return mcp.NewToolResultJSON(result)
}

// handleDecide handles the decide tool.
func (s *Server) handleDecide(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	caseID, ok := args["case_id"].(string)
	if !ok || caseID == "" {
		return mcp.NewToolResultError("case_id is required"), nil
	}

	// Generate the decision and persist proposals via DecisionService.Decide
	proposals, err := s.decisionSvc.Decide(ctx, caseID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to generate decision: %v", err)), nil
	}

	proposalList := make([]map[string]interface{}, len(proposals))
	for i, p := range proposals {
		proposalList[i] = map[string]interface{}{
			"proposal_id":           p.ProposalID,
			"case_id":               p.CaseID,
			"decision_id":           p.DecisionID,
			"action_type":           p.ActionType,
			"title":                 p.Title,
			"description":           p.Description,
			"risk_level":            p.RiskLevel,
			"requires_human_review": p.RequiresHumanReview,
			"apply_status":          p.ApplyStatus,
			"created_at":            p.CreatedAt,
		}
		if p.Payload != nil {
			proposalList[i]["payload"] = p.Payload
		}
	}

	result := map[string]interface{}{
		"case_id":   caseID,
		"proposals": proposalList,
		"count":     len(proposals),
	}

	return mcp.NewToolResultJSON(result)
}

// handleListCases handles the list_cases tool.
func (s *Server) handleListCases(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	filter := decision.CaseFilter{}
	args := req.GetArguments()

	if v, ok := args["source_type"].(string); ok && v != "" {
		filter.SourceType = &v
	}
	if v, ok := args["source_id"].(string); ok && v != "" {
		filter.SourceID = &v
	}
	if v, ok := args["status"].(string); ok && v != "" {
		filter.Status = &v
	}
	if v, ok := args["severity"].(string); ok && v != "" {
		filter.Severity = &v
	}
	if v, ok := args["limit"].(float64); ok && v > 0 {
		filter.Limit = int(v)
	} else {
		filter.Limit = 20
	}
	if v, ok := args["offset"].(float64); ok && v >= 0 {
		filter.Offset = int(v)
	}

	caseList, err := s.decisionSvc.ListCases(ctx, filter)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list cases: %v", err)), nil
	}

	cases := make([]map[string]interface{}, len(caseList.Cases))
	for i, c := range caseList.Cases {
		cases[i] = map[string]interface{}{
			"case_id":     c.CaseID,
			"status":      c.Status,
			"object_type": c.ObjectType,
			"object_id":   c.ObjectID,
			"severity":    c.Severity,
			"created_at":  c.CreatedAt,
		}
	}

	result := map[string]interface{}{
		"cases": cases,
		"total": caseList.Total,
	}

	return mcp.NewToolResultJSON(result)
}

// handleGetCase handles the get_case tool.
func (s *Server) handleGetCase(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	caseID, ok := args["case_id"].(string)
	if !ok || caseID == "" {
		return mcp.NewToolResultError("case_id is required"), nil
	}

	decisionCase, err := s.decisionSvc.GetCase(ctx, caseID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get case: %v", err)), nil
	}

	result := map[string]interface{}{
		"case_id":     decisionCase.CaseID,
		"alert_id":    decisionCase.AlertID,
		"status":      decisionCase.Status,
		"case_type":   decisionCase.CaseType,
		"object_type": decisionCase.ObjectType,
		"object_id":   decisionCase.ObjectID,
		"severity":    decisionCase.Severity,
		"created_at":  decisionCase.CreatedAt,
		"created_by":  decisionCase.CreatedBy,
	}

	return mcp.NewToolResultJSON(result)
}

// handleResolveCase handles the resolve_case tool.
func (s *Server) handleResolveCase(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	caseID, ok := args["case_id"].(string)
	if !ok || caseID == "" {
		return mcp.NewToolResultError("case_id is required"), nil
	}

	resolution, ok := args["resolution"].(string)
	if !ok || resolution == "" {
		return mcp.NewToolResultError("resolution is required"), nil
	}

	comment := ""
	if v, ok := args["comment"].(string); ok {
		comment = v
	}

	if err := s.decisionSvc.ResolveCase(ctx, caseID, resolution, comment); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to resolve case: %v", err)), nil
	}

	result := map[string]interface{}{
		"case_id":    caseID,
		"status":     "closed",
		"resolution": resolution,
	}

	return mcp.NewToolResultJSON(result)
}

// handleListProposals handles the list_proposals tool.
func (s *Server) handleListProposals(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	caseID, ok := args["case_id"].(string)
	if !ok || caseID == "" {
		return mcp.NewToolResultError("case_id is required"), nil
	}

	proposals, err := s.proposalSvc.ListProposals(ctx, caseID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list proposals: %v", err)), nil
	}

	proposalList := make([]map[string]interface{}, len(proposals))
	for i, p := range proposals {
		proposalList[i] = map[string]interface{}{
			"proposal_id":           p.ProposalID,
			"case_id":               p.CaseID,
			"decision_id":           p.DecisionID,
			"action_type":           p.ActionType,
			"title":                 p.Title,
			"description":           p.Description,
			"risk_level":            p.RiskLevel,
			"requires_human_review": p.RequiresHumanReview,
			"apply_status":          p.ApplyStatus,
			"created_at":            p.CreatedAt,
		}
		if p.Payload != nil {
			proposalList[i]["payload"] = p.Payload
		}
	}

	result := map[string]interface{}{
		"case_id":   caseID,
		"proposals": proposalList,
		"count":     len(proposals),
	}

	return mcp.NewToolResultJSON(result)
}
