package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// registerSandboxTools registers all simulation MCP tools.
func (s *Server) registerSandboxTools() {
	// Tool: create_simulation (was create_sandbox)
	createTool := mcp.NewTool(
		ToolCreateSimulation,
		mcp.WithDescription("Create a new simulation"),
		mcp.WithString("case_id", mcp.Required(), mcp.Description("The ID of the decision case")),
	)
	s.server.AddTool(createTool, s.handleCreateSandbox)

	if isLegacyToolsEnabled() {
		legacyCreateSimulationTool := mcp.NewTool(
			LegacyCreateSandbox,
			mcp.WithDescription("Create a new proposal sandbox"),
			mcp.WithString("case_id", mcp.Required(), mcp.Description("The ID of the decision case")),
		)
		s.server.AddTool(legacyCreateSimulationTool, s.handleCreateSandbox)
	}

	// Tool: add_to_simulation (was add_to_sandbox)
	addTool := mcp.NewTool(
		ToolAddToSimulation,
		mcp.WithDescription("Add a proposal to an existing simulation"),
		mcp.WithString("simulation_id", mcp.Required(), mcp.Description("The ID of the simulation")),
		mcp.WithString("proposal_id", mcp.Required(), mcp.Description("The ID of the proposal to add")),
	)
	s.server.AddTool(addTool, s.handleAddToSandbox)

	if isLegacyToolsEnabled() {
		legacyAddToSimulationTool := mcp.NewTool(
			LegacyAddToSandbox,
			mcp.WithDescription("Add a proposal to an existing sandbox"),
			mcp.WithString("sandbox_id", mcp.Required(), mcp.Description("The ID of the sandbox")),
			mcp.WithString("proposal_id", mcp.Required(), mcp.Description("The ID of the proposal to add")),
		)
		s.server.AddTool(legacyAddToSimulationTool, s.handleAddToSandbox)
	}

	// Tool: compare_simulations (was compare_sandboxes)
	compareTool := mcp.NewTool(
		ToolCompareSimulations,
		mcp.WithDescription("Compare two simulations and return differences"),
		mcp.WithString("simulation_id_1", mcp.Required(), mcp.Description("The first simulation ID")),
		mcp.WithString("simulation_id_2", mcp.Required(), mcp.Description("The second simulation ID")),
	)
	s.server.AddTool(compareTool, s.handleCompareSandboxes)

	if isLegacyToolsEnabled() {
		legacyCompareSimulationsTool := mcp.NewTool(
			LegacyCompareSandboxes,
			mcp.WithDescription("Compare two sandboxes and return differences"),
			mcp.WithString("sandbox_id_1", mcp.Required(), mcp.Description("The first sandbox ID")),
			mcp.WithString("sandbox_id_2", mcp.Required(), mcp.Description("The second sandbox ID")),
		)
		s.server.AddTool(legacyCompareSimulationsTool, s.handleCompareSandboxes)
	}

	// Tool: get_simulation (was get_sandbox)
	getTool := mcp.NewTool(
		ToolGetSimulation,
		mcp.WithDescription("Get simulation details by ID"),
		mcp.WithString("simulation_id", mcp.Required(), mcp.Description("The ID of the simulation")),
	)
	s.server.AddTool(getTool, s.handleGetSandbox)

	if isLegacyToolsEnabled() {
		legacyGetSimulationTool := mcp.NewTool(
			LegacyGetSandbox,
			mcp.WithDescription("Get sandbox details by ID"),
			mcp.WithString("sandbox_id", mcp.Required(), mcp.Description("The ID of the sandbox")),
		)
		s.server.AddTool(legacyGetSimulationTool, s.handleGetSandbox)
	}
}

// handleCreateSandbox handles the create_sandbox tool.
func (s *Server) handleCreateSandbox(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	caseID, ok := args["case_id"].(string)
	if !ok || caseID == "" {
		return mcp.NewToolResultError("case_id is required"), nil
	}

	// Parse optional initial data from raw_data if provided
	data := make(map[string]interface{})

	sandboxID, err := s.sandboxSvc.CreateSandbox(ctx, caseID, data)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create sandbox: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"sandbox_id": sandboxID,
		"case_id":    caseID,
		"status":     "draft",
	})
}

// handleAddToSandbox handles the add_to_sandbox tool.
func (s *Server) handleAddToSandbox(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	sandboxID, ok := args["sandbox_id"].(string)
	if !ok || sandboxID == "" {
		return mcp.NewToolResultError("sandbox_id is required"), nil
	}

	proposalID, ok := args["proposal_id"].(string)
	if !ok || proposalID == "" {
		return mcp.NewToolResultError("proposal_id is required"), nil
	}

	if err := s.sandboxSvc.AddProposalToSandbox(ctx, sandboxID, proposalID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to add proposal to sandbox: %v", err)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"sandbox_id":  sandboxID,
		"proposal_id": proposalID,
		"status":      "added",
	})
}

// handleCompareSandboxes handles the compare_sandboxes tool.
func (s *Server) handleCompareSandboxes(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	sandboxID1, ok := args["sandbox_id_1"].(string)
	if !ok || sandboxID1 == "" {
		return mcp.NewToolResultError("sandbox_id_1 is required"), nil
	}

	sandboxID2, ok := args["sandbox_id_2"].(string)
	if !ok || sandboxID2 == "" {
		return mcp.NewToolResultError("sandbox_id_2 is required"), nil
	}

	result, err := s.sandboxSvc.CompareSandbox(ctx, sandboxID1, sandboxID2)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to compare sandboxes: %v", err)), nil
	}

	diffs := make([]map[string]interface{}, 0, len(result.Differences))
	for _, d := range result.Differences {
		diffs = append(diffs, map[string]interface{}{
			"field":   d.Field,
			"value_1": d.Value1,
			"value_2": d.Value2,
		})
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"sandbox_1_id": result.Sandbox1ID,
		"sandbox_2_id": result.Sandbox2ID,
		"differences":  diffs,
	})
}

// handleGetSandbox handles the get_sandbox tool.
func (s *Server) handleGetSandbox(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	sandboxID, ok := args["sandbox_id"].(string)
	if !ok || sandboxID == "" {
		return mcp.NewToolResultError("sandbox_id is required"), nil
	}

	sandbox, err := s.sandboxSvc.GetSandbox(ctx, sandboxID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get sandbox: %v", err)), nil
	}
	if sandbox == nil {
		return mcp.NewToolResultError(fmt.Sprintf("sandbox %q not found", sandboxID)), nil
	}

	return mcp.NewToolResultJSON(map[string]interface{}{
		"sandbox_id":    sandbox.SandboxID,
		"case_id":       sandbox.CaseID,
		"proposal_id":   sandbox.ProposalID,
		"data":          sandbox.Data,
		"status":        sandbox.Status,
		"compared_with": sandbox.ComparedWith,
		"created_at":    sandbox.CreatedAt,
	})
}
