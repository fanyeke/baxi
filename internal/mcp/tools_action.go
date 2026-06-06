package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"baxi/internal/action"
	"baxi/internal/decision"
	"baxi/internal/llm"
	"baxi/internal/ontology"
	"github.com/mark3labs/mcp-go/mcp"
)

// registerActionTools registers all action-related MCP tools.
func (s *Server) registerActionTools() {
	// Tool: execute_action (was execute_proposal)
	executeTool := mcp.NewTool(
		ToolExecuteAction,
		mcp.WithDescription("Execute an action"),
		mcp.WithString("proposal_id", mcp.Required(), mcp.Description("The ID of the proposal to execute")),
		mcp.WithBoolean("dry_run", mcp.Description("When true (default), simulate execution without side effects")),
	)
	s.server.AddTool(executeTool, s.handleExecuteProposal)

	if isLegacyToolsEnabled() {
		legacyExecuteTool := mcp.NewTool(
			LegacyExecuteProposal,
			mcp.WithDescription("Execute an approved action proposal"),
			mcp.WithString("proposal_id", mcp.Required(), mcp.Description("The ID of the proposal to execute")),
			mcp.WithBoolean("dry_run", mcp.Description("When true (default), simulate execution without side effects")),
		)
		s.server.AddTool(legacyExecuteTool, s.handleExecuteProposal)
	}

	// Tool: suggest_action (was propose_action)
	proposeTool := mcp.NewTool(
		ToolProposeAction,
		mcp.WithDescription("Propose an action on an object"),
		mcp.WithString("object_type", mcp.Required(), mcp.Description("The type of the target object")),
		mcp.WithString("object_id", mcp.Required(), mcp.Description("The ID of the target object")),
		mcp.WithString("action_type", mcp.Required(), mcp.Description("The action type to propose")),
		mcp.WithString("params", mcp.Description("Optional JSON-encoded parameters for the action")),
		mcp.WithString("evidence_refs", mcp.Description("Optional JSON string array of evidence reference IDs")),
		mcp.WithString("context_hash", mcp.Description("Optional hash of the context used for the decision")),
		mcp.WithString("recipe_id", mcp.Description("Optional ID of the recipe that triggered the decision")),
		mcp.WithString("decision_json", mcp.Description("Optional full agent decision JSON (creates an LLM decision record)")),
	)
	s.server.AddTool(proposeTool, s.handleProposeAction)

	if isLegacyToolsEnabled() {
		legacyProposeTool := mcp.NewTool(
			LegacyProposeAction,
			mcp.WithDescription("Propose an action on an object (creates a proposal for approval)"),
			mcp.WithString("object_type", mcp.Required(), mcp.Description("The type of the target object")),
			mcp.WithString("object_id", mcp.Required(), mcp.Description("The ID of the target object")),
			mcp.WithString("action_type", mcp.Required(), mcp.Description("The action type to propose")),
			mcp.WithString("params", mcp.Description("Optional JSON-encoded parameters for the action")),
			mcp.WithString("evidence_refs", mcp.Description("Optional JSON string array of evidence reference IDs")),
			mcp.WithString("context_hash", mcp.Description("Optional hash of the context used for the decision")),
			mcp.WithString("recipe_id", mcp.Description("Optional ID of the recipe that triggered the decision")),
			mcp.WithString("decision_json", mcp.Description("Optional full agent decision JSON (creates an LLM decision record)")),
		)
		s.server.AddTool(legacyProposeTool, s.handleProposeAction)
	}

	// Tool: get_decision_context
	contextTool := mcp.NewTool(
		ToolGetDecisionContext,
		mcp.WithDescription("Get the full decision context for a case, including trigger, object, governance, evidence, and decision guidance recommendations"),
		mcp.WithString("case_id", mcp.Required(), mcp.Description("The ID of the case to get context for")),
	)
	s.server.AddTool(contextTool, s.handleGetDecisionContext)

	if isLegacyToolsEnabled() {
		legacyContextTool := mcp.NewTool(
			LegacyGetDecisionContext,
			mcp.WithDescription("Get the full decision context for a case"),
			mcp.WithString("case_id", mcp.Required(), mcp.Description("The ID of the case to get context for")),
		)
		s.server.AddTool(legacyContextTool, s.handleGetDecisionContext)
	}
}

// handleProposeAction handles the propose_action tool.
// Supports optional trace fields: evidence_refs, context_hash, recipe_id, and decision_json.
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
			return mcp.NewToolResultError(SanitizeErrorf("invalid JSON in params: %v", err)), nil
		}
	}

	// Parse optional trace fields
	evidenceRefs, _ := args["evidence_refs"].(string)
	contextHash, _ := args["context_hash"].(string)
	recipeID, _ := args["recipe_id"].(string)
	decisionJSON, _ := args["decision_json"].(string)

	trace := ProposeActionTrace{
		EvidenceRefs: evidenceRefs,
		ContextHash:  contextHash,
		RecipeID:     recipeID,
	}

	// When decision_json is provided, parse it, validate it, and create an LLM decision record.
	var decisionSeverity string
	if decisionJSON != "" {
		decisionID, caseID, ds, err := s.handleDecisionJSON(ctx, decisionJSON, &trace)
		if err != nil {
			return mcp.NewToolResultError(SanitizeErrorf("invalid decision_json: %v", err)), nil
		}
		trace.DecisionID = decisionID
		trace.CaseID = caseID
		decisionSeverity = ds
	}

	// ──── Decision guidance resolution (advisory) ────────────────────────────
	guidanceSeverity := decisionSeverity
	var guidanceWarnings []string
	var guidanceRecommendation string

	if recipeID != "" && s.recipes != nil {
		if recipe, ok := s.recipes[recipeID]; ok {
			// If no severity from decision_json, try the recipe's default (first level).
			if guidanceSeverity == "" && len(recipe.DecisionGuidance.Levels) > 0 {
				guidanceSeverity = recipe.DecisionGuidance.Levels[0].Severity
			}
			if guidanceSeverity != "" {
				if level, err := ontology.ResolveGuidance(recipe, guidanceSeverity); err == nil {
					guidanceRecommendation = level.Recommendation
					// Check if the proposed action is recommended.
					recommended := false
					for _, a := range level.Actions {
						if a == actionType {
							recommended = true
							break
						}
					}
					if !recommended {
						warning := SanitizeErrorf("action %q is not in the recommended actions for severity %q (recommended: %v)",
							actionType, guidanceSeverity, level.Actions)
						guidanceWarnings = append(guidanceWarnings, warning)
					}
				}
			}
		}
	}
	// ─────────────────────────────────────────────────────────────────────────

	result, err := s.ontologySvc.ProposeAction(ctx, objectType, objectID, actionType, params, trace)
	if err != nil {
		return mcp.NewToolResultError(SanitizeErrorf("Failed to propose action: %v", err)), nil
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
	if trace.DecisionID != "" {
		res["decision_id"] = trace.DecisionID
	}
	if guidanceRecommendation != "" {
		res["guidance_recommendation"] = guidanceRecommendation
	}
	if len(guidanceWarnings) > 0 {
		res["guidance_warnings"] = guidanceWarnings
	}
	return mcp.NewToolResultJSON(res)
}

// handleDecisionJSON parses, validates, and persists an agent decision JSON blob.
// It extracts case_id, severity, confidence, and evidence_refs from the JSON,
// creates an LLM decision record, and returns the generated decision ID, case ID, and severity.
func (s *Server) handleDecisionJSON(ctx context.Context, rawJSON string, trace *ProposeActionTrace) (string, string, string, error) {
	// Parse into DecisionOutput for schema validation.
	var decisionOutput llm.DecisionOutput
	if err := json.Unmarshal([]byte(rawJSON), &decisionOutput); err != nil {
		return "", "", "", fmt.Errorf("parse decision_json: %w", err)
	}

	// Validate against the decision schema.
	// Use a broad allowed actions list since we are recording a pre-made decision.
	allowedActions := []string{
		llm.ActionTypeCreateFollowupTask,
		llm.ActionTypeNotifyOwner,
		llm.ActionTypeExportReport,
		llm.ActionTypeCreateOutboxMessage,
		llm.ActionTypeEscalateToHuman,
	}
	if vr := llm.ValidateDecision(&decisionOutput, allowedActions); !vr.Valid {
		errMsg := "validation failed"
		if len(vr.Errors) > 0 {
			errMsg = vr.Errors[0].Error()
		}
		return "", "", "", fmt.Errorf("decision schema %s", errMsg)
	}

	// Extract case_id and evidence_refs from the raw JSON (not part of DecisionOutput).
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(rawJSON), &raw); err != nil {
		return "", "", "", fmt.Errorf("parse decision_json fields: %w", err)
	}

	caseID, _ := raw["case_id"].(string)
	if caseID == "" {
		return "", "", "", fmt.Errorf("decision_json must contain a non-empty 'case_id' field")
	}

	// If evidence_refs was not provided as a top-level argument, extract from decision_json.
	if trace.EvidenceRefs == "" {
		if refsRaw, ok := raw["evidence_refs"]; ok {
			refsJSON, err := json.Marshal(refsRaw)
			if err == nil {
				trace.EvidenceRefs = string(refsJSON)
			}
		}
	}

	// Marshal the validated decision output to JSON for storage.
	outputJSON, err := json.Marshal(decisionOutput)
	if err != nil {
		return "", "", "", fmt.Errorf("marshal decision output: %w", err)
	}
	outputRaw := json.RawMessage(outputJSON)
	confidence := decisionOutput.Confidence
	severity := decisionOutput.Severity
	now := time.Now()
	status := "recorded"
	decisionID := decision.GenerateDecisionID()

	// Insert LLM decision record via the pool.
	_, err = s.pool.Exec(ctx, `
		INSERT INTO ai.llm_decision (
			decision_id, case_id, output_json, confidence, created_at,
			status, recipe_id, context_hash, severity
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)`,
		decisionID, caseID, &outputRaw, confidence, now,
		status, nullIfEmpty(trace.RecipeID), nullIfEmpty(trace.ContextHash), severity,
	)
	if err != nil {
		return "", "", "", fmt.Errorf("create LLM decision: %w", err)
	}

	return decisionID, caseID, severity, nil
}

// nullIfEmpty returns nil for an empty string, or the string pointer otherwise.
func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
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

	result, err := s.executeSvc.ExecuteProposal(ctx, s.pool, proposalID, s.mcpUserID, action.WithDryRun(dryRun))
	if err != nil {
		return mcp.NewToolResultError(SanitizeErrorf("Failed to execute proposal: %v", err)), nil
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
		return mcp.NewToolResultError(SanitizeErrorf("Failed to build decision context: %v", err)), nil
	}

	// Build evidence items from the trigger data.
	var evidence []llm.EvidenceItem
	if decisionCtx.Trigger.AlertID != "" {
		evidence = append(evidence, llm.EvidenceItem{Type: "alert", Key: "alert_id", Value: decisionCtx.Trigger.AlertID})
	}
	if decisionCtx.Trigger.RuleID != "" {
		evidence = append(evidence, llm.EvidenceItem{Type: "alert", Key: "rule_id", Value: decisionCtx.Trigger.RuleID})
	}
	if decisionCtx.Trigger.MetricName != "" {
		evidence = append(evidence, llm.EvidenceItem{Type: "metric", Key: "metric_name", Value: decisionCtx.Trigger.MetricName})
		evidence = append(evidence, llm.EvidenceItem{Type: "metric", Key: "current_value", Value: decisionCtx.Trigger.CurrentValue})
		evidence = append(evidence, llm.EvidenceItem{Type: "metric", Key: "baseline_value", Value: decisionCtx.Trigger.BaselineValue})
		evidence = append(evidence, llm.EvidenceItem{Type: "metric", Key: "delta_pct", Value: decisionCtx.Trigger.DeltaPct})
	}
	if decisionCtx.Governance.Classification != "" {
		evidence = append(evidence, llm.EvidenceItem{Type: "classification", Key: "overall_level", Value: decisionCtx.Governance.Classification})
	}
	if evidence == nil {
		evidence = []llm.EvidenceItem{}
	}

	result := map[string]interface{}{
		"case_id":           decisionCtx.DecisionCaseID,
		"trigger":           decisionCtx.Trigger,
		"object_context":    decisionCtx.ObjectContext,
		"governance":        decisionCtx.Governance,
		"allowed_actions":   decisionCtx.AllowedActions,
		"forbidden_actions": decisionCtx.ForbiddenActions,
		"evidence":          evidence,
		"rendered_evidence": decisionCtx.RenderedEvidence,
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

	// ──── Decision guidance resolution ──────────────────────────────────────
	// Look up the recipe by matching the trigger's rule_id against available recipes.
	if decisionCtx.Trigger.RuleID != "" && s.recipes != nil {
		for _, recipe := range s.recipes {
			if recipe.Trigger.RuleID != "" && (recipe.Trigger.RuleID == decisionCtx.Trigger.RuleID ||
				strings.Contains(decisionCtx.Trigger.RuleID, recipe.Trigger.RuleID)) {
				severity := decisionCtx.Trigger.Severity
				if severity == "" && len(recipe.DecisionGuidance.Levels) > 0 {
					severity = recipe.DecisionGuidance.Levels[0].Severity
				}
				if severity != "" {
					if level, err := ontology.ResolveGuidance(recipe, severity); err == nil {
						result["guidance_recommendation"] = level.Recommendation
						result["guidance_recommended_actions"] = level.Actions
						result["guidance_prompt_fragment"] = ontology.GuidanceToPromptFragment(level)
					}
				}
				break
			}
		}
	}
	// ─────────────────────────────────────────────────────────────────────────

	return mcp.NewToolResultJSON(result)
}
