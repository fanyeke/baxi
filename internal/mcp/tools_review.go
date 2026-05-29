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
		mcp.WithString("reviewer_id", mcp.Required(), mcp.Description("The ID of the reviewer")),
		mcp.WithString("feedback", mcp.Description("Optional feedback for the approval")),
	)
	s.server.AddTool(approveTool, s.handleApproveProposal)

	// Tool: reject_proposal
	rejectTool := mcp.NewTool(
		"reject_proposal",
		mcp.WithDescription("Reject an action proposal"),
		mcp.WithString("proposal_id", mcp.Required(), mcp.Description("The ID of the proposal to reject")),
		mcp.WithString("reviewer_id", mcp.Required(), mcp.Description("The ID of the reviewer")),
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
}

// handleApproveProposal handles the approve_proposal tool.
func (s *Server) handleApproveProposal(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	proposalID, ok := args["proposal_id"].(string)
	if !ok || proposalID == "" {
		return mcp.NewToolResultError("proposal_id is required"), nil
	}

	reviewerID, ok := args["reviewer_id"].(string)
	if !ok || reviewerID == "" {
		return mcp.NewToolResultError("reviewer_id is required"), nil
	}

	feedback := ""
	if v, ok := args["feedback"].(string); ok {
		feedback = v
	}

	record, err := s.reviewSvc.ApproveProposal(ctx, proposalID, reviewerID, feedback)
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

	reviewerID, ok := args["reviewer_id"].(string)
	if !ok || reviewerID == "" {
		return mcp.NewToolResultError("reviewer_id is required"), nil
	}

	feedback := ""
	if v, ok := args["feedback"].(string); ok {
		feedback = v
	}

	record, err := s.reviewSvc.RejectProposal(ctx, proposalID, reviewerID, feedback)
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

	if err := s.reviewSvc.CancelProposal(ctx, proposalID, reason); err != nil {
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
