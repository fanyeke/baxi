package mcp

// Tool name constants for MCP information containment.
//
// All tool names are renamed from domain-specific names (e.g., "run_pipeline",
// "describe_ontology") to generic business-capability names. This prevents
// AI agents from mapping tool names to internal package structure.
//
// Legacy names are preserved via MCP_ENABLE_LEGACY_TOOLS for Pi Agent
// backward compatibility.

// New (contained) tool names — business capability oriented.
const (
	// Decision / Evaluation
	ToolEvaluateCase          = "evaluate_case"
	ToolGetEvaluation         = "get_evaluation"
	ToolListEvaluations       = "list_evaluations"
	ToolResolveEvaluation     = "resolve_evaluation"
	ToolAnalyzeSituation      = "analyze_situation"
	ToolGetAnalysis           = "get_analysis"
	ToolGenerateRecommendation = "generate_recommendation"
	ToolListRecommendations   = "list_recommendations"

	// Alerts / Monitoring
	ToolListAlerts = "list_alerts"

	// Governance / Classification
	ToolCheckPermission       = "check_permission"
	ToolGetDataClassification = "get_data_classification"

	// Data Processing
	ToolProcessData        = "process_data"
	ToolGetProcessingStatus = "get_processing_status"

	// Events
	ToolListEvents = "list_events"

	// Approvals / Reviews
	ToolApproveAction   = "approve_action"
	ToolRejectAction    = "reject_action"
	ToolCancelAction    = "cancel_action"
	ToolGetActionProposal = "get_action_proposal"
	ToolListReviews     = "list_reviews"

	// Action Execution
	ToolExecuteAction     = "execute_action"
	ToolProposeAction     = "suggest_action"
	ToolGetDecisionContext = "get_decision_context"
	ToolGetSystemHealth   = "get_system_health"
	ToolSearchRecords     = "search_records"

	// Schema / Data Model
	ToolDescribeSchema   = "describe_schema"
	ToolGetRecord        = "get_record"
	ToolGetRelatedRecords = "get_related_records"
	ToolApplyAction      = "apply_action"

	// Sandbox / Simulation
	ToolCreateSimulation  = "create_simulation"
	ToolAddToSimulation   = "add_to_simulation"
	ToolCompareSimulations = "compare_simulations"
	ToolGetSimulation     = "get_simulation"

	// Action Types
	ToolListActionTypes = "list_action_types"
	ToolGetActionType   = "get_action_type"
)

// Legacy tool names — original domain-specific names.
const (
	LegacyCreateDecisionCase   = "create_decision_case"
	LegacyDecide               = "decide"
	LegacyResolveCase          = "resolve_case"
	LegacyListCases            = "list_cases"
	LegacyGetCase              = "get_case"
	LegacyListProposals        = "list_proposals"
	LegacyListAlerts           = "list_alerts"
	LegacyCheckAccess          = "check_access"
	LegacyGetClassification    = "get_classification"
	LegacyRunPipeline          = "run_pipeline"
	LegacyApproveProposal      = "approve_proposal"
	LegacyRejectProposal       = "reject_proposal"
	LegacyCancelProposal       = "cancel_proposal"
	LegacyGetProposalByID      = "get_proposal_by_id"
	LegacyListReviewRecords    = "list_review_records"
	LegacyExecuteProposal      = "execute_proposal"
	LegacyGetDecisionContext   = "get_decision_context"
	LegacyListOutboxEvents     = "list_outbox_events"
	LegacyGetPipelineStatus    = "get_pipeline_status"
	LegacyGetSystemStatus      = "get_system_status"
	LegacySearchObjects        = "search_objects"
	LegacyDescribeOntology     = "describe_ontology"
	LegacyGetObject            = "get_object"
	LegacyGetLinkedObjects     = "get_linked_objects"
	LegacyExecuteAction        = "execute_action"
	LegacyCreateSandbox        = "create_sandbox"
	LegacyAddToSandbox         = "add_to_sandbox"
	LegacyCompareSandboxes     = "compare_sandboxes"
	LegacyGetSandbox           = "get_sandbox"
	LegacyListActionSchemas    = "list_action_schemas"
	LegacyGetActionSchema      = "get_action_schema"
	LegacyBuildContext         = "build_context"
	LegacyProposeAction        = "propose_action"
)

// legacyToolMap maps old tool names to new tool names.
// Used by registerLegacyAliases to provide backward compatibility.
var legacyToolMap = map[string]string{
	LegacyCreateDecisionCase: ToolEvaluateCase,
	LegacyDecide:             ToolGenerateRecommendation,
	LegacyResolveCase:        ToolResolveEvaluation,
	LegacyListCases:          ToolListEvaluations,
	LegacyGetCase:            ToolGetEvaluation,
	LegacyListProposals:      ToolListRecommendations,
	LegacyBuildContext:       ToolAnalyzeSituation,
	LegacyProposeAction:      ToolProposeAction,
	LegacyListAlerts:         ToolListAlerts,
	LegacyCheckAccess:        ToolCheckPermission,
	LegacyGetClassification:  ToolGetDataClassification,
	LegacyRunPipeline:        ToolProcessData,
	LegacyApproveProposal:    ToolApproveAction,
	LegacyRejectProposal:     ToolRejectAction,
	LegacyCancelProposal:     ToolCancelAction,
	LegacyGetProposalByID:    ToolGetActionProposal,
	LegacyListReviewRecords:  ToolListReviews,
	LegacyExecuteProposal:    ToolExecuteAction,
	LegacyGetDecisionContext: ToolGetDecisionContext,
	LegacyListOutboxEvents:   ToolListEvents,
	LegacyGetPipelineStatus:  ToolGetProcessingStatus,
	LegacyGetSystemStatus:    ToolGetSystemHealth,
	LegacySearchObjects:      ToolSearchRecords,
	LegacyDescribeOntology:   ToolDescribeSchema,
	LegacyGetObject:          ToolGetRecord,
	LegacyGetLinkedObjects:   ToolGetRelatedRecords,
	LegacyExecuteAction:      ToolApplyAction,
	LegacyCreateSandbox:      ToolCreateSimulation,
	LegacyAddToSandbox:       ToolAddToSimulation,
	LegacyCompareSandboxes:   ToolCompareSimulations,
	LegacyGetSandbox:         ToolGetSimulation,
	LegacyListActionSchemas:  ToolListActionTypes,
	LegacyGetActionSchema:    ToolGetActionType,
}

// newToolNames returns all new (contained) tool names.
func newToolNames() []string {
	return []string{
		ToolEvaluateCase,
		ToolGetEvaluation,
		ToolListEvaluations,
		ToolResolveEvaluation,
		ToolAnalyzeSituation,
		ToolGetAnalysis,
		ToolGenerateRecommendation,
		ToolListRecommendations,
		ToolListAlerts,
		ToolCheckPermission,
		ToolGetDataClassification,
		ToolProcessData,
		ToolGetProcessingStatus,
		ToolListEvents,
		ToolApproveAction,
		ToolRejectAction,
		ToolCancelAction,
		ToolGetActionProposal,
		ToolListReviews,
		ToolExecuteAction,
		ToolProposeAction,
		ToolGetDecisionContext,
		ToolGetSystemHealth,
		ToolSearchRecords,
		ToolDescribeSchema,
		ToolGetRecord,
		ToolGetRelatedRecords,
		ToolApplyAction,
		ToolCreateSimulation,
		ToolAddToSimulation,
		ToolCompareSimulations,
		ToolGetSimulation,
		ToolListActionTypes,
		ToolGetActionType,
	}
}

// legacyToolNames returns all old (legacy) tool names.
func legacyToolNames() []string {
	names := make([]string, 0, len(legacyToolMap))
	for name := range legacyToolMap {
		names = append(names, name)
	}
	return names
}
