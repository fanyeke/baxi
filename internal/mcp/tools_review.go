package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// registerReviewTools registers all review-related MCP tools.
func (s *Server) registerReviewTools() {
	// Tool: approve_proposal
	approveTool := mcp.NewTool(
		"approve_proposal",
		mcp.WithDescription("Approve an action proposal"),
		mcp.WithString("proposal_id", mcp.Required(), mcp.Description("The ID of the proposal to approve")),
		mcp.WithString("feedback", mcp.Description("Optional feedback for the approval")),
	)
	s.server.AddTool(approveTool, s.handleApproveProposal)

	// Tool: reject_proposal
	rejectTool := mcp.NewTool(
		"reject_proposal",
		mcp.WithDescription("Reject an action proposal"),
		mcp.WithString("proposal_id", mcp.Required(), mcp.Description("The ID of the proposal to reject")),
		mcp.WithString("feedback", mcp.Description("Optional feedback for the rejection")),
	)
	s.server.AddTool(rejectTool, s.handleRejectProposal)

	// Tool: cancel_proposal
	cancelTool := mcp.NewTool(
		"cancel_proposal",
		mcp.WithDescription("Cancel an action proposal"),
		mcp.WithString("proposal_id", mcp.Required(), mcp.Description("The ID of the proposal to cancel")),
		mcp.WithString("reason", mcp.Description("Optional reason for the cancellation")),
	)
	s.server.AddTool(cancelTool, s.handleCancelProposal)

	// Tool: get_proposal_by_id
	getProposalTool := mcp.NewTool(
		"get_proposal_by_id",
		mcp.WithDescription("Get full details of an action proposal by ID"),
		mcp.WithString("proposal_id", mcp.Required(), mcp.Description("The ID of the proposal to retrieve")),
	)
	s.server.AddTool(getProposalTool, s.handleGetProposalByID)

	// Tool: list_review_records
	listReviewTool := mcp.NewTool(
		"list_review_records",
		mcp.WithDescription("List review records for a proposal with pagination"),
		mcp.WithString("proposal_id", mcp.Required(), mcp.Description("The ID of the proposal")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of records to return (default 50)")),
		mcp.WithNumber("offset", mcp.Description("Number of records to skip (default 0)")),
	)
	s.server.AddTool(listReviewTool, s.handleListReviewRecords)
}

// handleApproveProposal handles the approve_proposal tool.
func (s *Server) handleApproveProposal(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	proposalID, ok := args["proposal_id"].(string)
	if !ok || proposalID == "" {
		return mcp.NewToolResultError("proposal_id is required"), nil
	}

	feedback := ""
	if v, ok := args["feedback"].(string); ok {
		feedback = v
	}

	record, err := s.reviewSvc.ApproveProposal(ctx, proposalID, s.mcpUserID, feedback)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to approve proposal: %v", err)), nil
	}

	result := map[string]interface{}{
		"record_id":   record.RecordID,
		"proposal_id": record.ProposalID,
		"verdict":     string(record.Verdict),
		"feedback":    record.Feedback,
		"created_at":  record.CreatedAt,
	}

	return mcp.NewToolResultJSON(result)
}

// handleRejectProposal handles the reject_proposal tool.
func (s *Server) handleRejectProposal(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	proposalID, ok := args["proposal_id"].(string)
	if !ok || proposalID == "" {
		return mcp.NewToolResultError("proposal_id is required"), nil
	}

	feedback := ""
	if v, ok := args["feedback"].(string); ok {
		feedback = v
	}

	record, err := s.reviewSvc.RejectProposal(ctx, proposalID, s.mcpUserID, feedback)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to reject proposal: %v", err)), nil
	}

	result := map[string]interface{}{
		"record_id":   record.RecordID,
		"proposal_id": record.ProposalID,
		"verdict":     string(record.Verdict),
		"feedback":    record.Feedback,
		"created_at":  record.CreatedAt,
	}

	return mcp.NewToolResultJSON(result)
}

// handleCancelProposal handles the cancel_proposal tool.
func (s *Server) handleCancelProposal(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	proposalID, ok := args["proposal_id"].(string)
	if !ok || proposalID == "" {
		return mcp.NewToolResultError("proposal_id is required"), nil
	}

	reason := ""
	if v, ok := args["reason"].(string); ok {
		reason = v
	}

	if err := s.reviewSvc.CancelProposal(ctx, proposalID, s.mcpUserID, reason); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to cancel proposal: %v", err)), nil
	}

	result := map[string]interface{}{
		"proposal_id": proposalID,
		"status":      "cancelled",
	}

	return mcp.NewToolResultJSON(result)
}

// handleGetProposalByID handles the get_proposal_by_id tool.
func (s *Server) handleGetProposalByID(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	proposalID, ok := args["proposal_id"].(string)
	if !ok || proposalID == "" {
		return mcp.NewToolResultError("proposal_id is required"), nil
	}

	proposal, err := s.reviewSvc.GetProposalByID(ctx, proposalID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get proposal: %v", err)), nil
	}
	if proposal == nil {
		return mcp.NewToolResultError("proposal not found"), nil
	}

	result := map[string]interface{}{
		"proposal_id":           proposal.ProposalID,
		"case_id":               proposal.CaseID,
		"decision_id":           proposal.DecisionID,
		"action_type":           proposal.ActionType,
		"title":                 proposal.Title,
		"description":           proposal.Description,
		"risk_level":            proposal.RiskLevel,
		"requires_human_review": proposal.RequiresHumanReview,
		"apply_status":          proposal.ApplyStatus,
		"created_at":            proposal.CreatedAt,
	}
	if proposal.Payload != nil {
		result["payload"] = proposal.Payload
	}

	return mcp.NewToolResultJSON(result)
}

// handleListReviewRecords handles the list_review_records tool.
func (s *Server) handleListReviewRecords(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	proposalID, ok := args["proposal_id"].(string)
	if !ok || proposalID == "" {
		return mcp.NewToolResultError("proposal_id is required"), nil
	}

	limit := 50
	if v, ok := args["limit"].(float64); ok {
		limit = int(v)
	}

	offset := 0
	if v, ok := args["offset"].(float64); ok {
		offset = int(v)
	}

	records, total, err := s.reviewSvc.ListReviewRecords(ctx, proposalID, limit, offset)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list review records: %v", err)), nil
	}

	items := make([]map[string]interface{}, 0, len(records))
	for _, r := range records {
		items = append(items, map[string]interface{}{
			"record_id":   r.RecordID,
			"proposal_id": r.ProposalID,
			"verdict":     string(r.Verdict),
			"feedback":    r.Feedback,
			"reviewer_id": r.ReviewerID,
			"created_at":  r.CreatedAt,
		})
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"items":  items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}
