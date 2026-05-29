package decision

import (
	"context"
	"fmt"
	"time"

	"baxi/internal/governance"
	"baxi/internal/llm"
	"baxi/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ContextBuilderV2 builds governed LLM-safe decision contexts using the new
// OntologyAwareRepo, MarkingService, and DecisionLineageService interfaces.
// It replaces the per-field classification provider with the unified MarkingService
// and adds decision-specific lineage tracking via DecisionLineageService.
type ContextBuilderV2 struct {
	caseSvc      DecisionCaseDataProvider
	ontologyRepo repository.OntologyAwareRepo
	markingSvc   governance.MarkingService
	lineageSvc   DecisionLineageService
	pool         *pgxpool.Pool
	actionTypes  ActionTypeProvider
}

// NewContextBuilderV2 creates a ContextBuilderV2 backed by the new interfaces.
func NewContextBuilderV2(
	caseSvc DecisionCaseDataProvider,
	ontologyRepo repository.OntologyAwareRepo,
	markingSvc governance.MarkingService,
	lineageSvc DecisionLineageService,
	pool *pgxpool.Pool,
	actionTypes ActionTypeProvider,
) *ContextBuilderV2 {
	return &ContextBuilderV2{
		caseSvc:      caseSvc,
		ontologyRepo: ontologyRepo,
		markingSvc:   markingSvc,
		lineageSvc:   lineageSvc,
		pool:         pool,
		actionTypes:  actionTypes,
	}
}

// BuildDecisionContext constructs a full DecisionContext for the given case ID
// using the new ontology-aware repo, marking service, and lineage service.
func (b *ContextBuilderV2) BuildDecisionContext(ctx context.Context, caseID string) (*DecisionContext, error) {
	caseRow, err := b.caseSvc.GetCaseByID(ctx, b.pool, caseID)
	if err != nil {
		return nil, fmt.Errorf("fetch case %s: %w", caseID, err)
	}

	objectType := derefString(caseRow.ObjectType)
	objectID := derefString(caseRow.ObjectID)
	severity := derefString(caseRow.Severity)

	trigger, err := b.buildTriggerV2(ctx, caseRow, severity)
	if err != nil {
		return nil, err
	}

	objectInst, err := b.ontologyRepo.GetObjectByID(ctx, b.pool, objectType, objectID)
	if err != nil {
		return nil, fmt.Errorf("get object %s/%s: %w", objectType, objectID, err)
	}

	classifications, err := b.buildClassifications(ctx, objectType, objectInst.Properties)
	if err != nil {
		return nil, fmt.Errorf("build classifications for %s: %w", objectType, err)
	}

	markings := make(map[string]string)
	policy := governance.RedactionPolicy{Role: "agent_readonly"}
	redactionResult := governance.RedactObjectContext(objectInst.Properties, classifications, markings, policy)

	redactedFieldNames := make([]string, len(redactionResult.RedactedFields))
	for i, rf := range redactionResult.RedactedFields {
		redactedFieldNames[i] = rf.Field
	}

	overallClassification := resolveOverallClassification(classifications)

	lineage, err := b.buildLineage(ctx, caseID)
	if err != nil {
		lineage = nil
	}

	govData := GovernanceData{
		Classification:   overallClassification,
		Lineage:          lineage,
		RedactionApplied: len(redactedFieldNames) > 0,
		RedactedFields:   redactedFieldNames,
		Role:             "agent_readonly",
	}

	allowedActions := b.actionTypes.ListActionTypes()
	forbiddenActions := []string{
		"execute",
		"apply",
		"dispatch",
	}

	policyResult := b.buildPolicyResult()

	decisionCtx := &DecisionContext{
		DecisionCaseID: caseRow.CaseID,
		SourceType:     caseRow.SourceType,
		SourceID:       caseRow.SourceID,
		Trigger:        trigger,
		ObjectContext: ObjectContextData{
			ObjectType: objectInst.ObjectType,
			ObjectID:   objectInst.ID,
			Properties: redactionResult.Properties,
		},
		Governance:       govData,
		AllowedActions:   allowedActions,
		ForbiddenActions: forbiddenActions,
		Policy:           policyResult,
	}

	return decisionCtx, nil
}

// buildTriggerV2 fetches alert data using OntologyAwareRepo and builds TriggerInfo.
func (b *ContextBuilderV2) buildTriggerV2(ctx context.Context, caseRow *repository.DecisionCaseRow, severity string) (TriggerInfo, error) {
	trigger := TriggerInfo{
		Severity: severity,
	}

	if caseRow.AlertID == nil || *caseRow.AlertID == "" {
		return trigger, nil
	}

	alert, err := b.ontologyRepo.GetObjectByID(ctx, b.pool, "metric_alert", *caseRow.AlertID)
	if err != nil {
		return trigger, fmt.Errorf("fetch alert %s: %w", *caseRow.AlertID, err)
	}

	trigger.AlertID = *caseRow.AlertID
	trigger.RuleID = getStringProp(alert.Properties, "rule_id")
	trigger.MetricName = getStringProp(alert.Properties, "metric_name")
	trigger.CurrentValue = getFloatProp(alert.Properties, "current_value")
	trigger.BaselineValue = getFloatProp(alert.Properties, "baseline_value")
	trigger.DeltaPct = getFloatProp(alert.Properties, "delta_pct")

	if trigger.Severity == "" {
		trigger.Severity = getStringProp(alert.Properties, "severity")
	}

	return trigger, nil
}

// buildClassifications uses MarkingService to get unified field markings
// and maps them to the classification format expected by the redaction engine.
func (b *ContextBuilderV2) buildClassifications(ctx context.Context, objectType string, properties map[string]interface{}) (map[string]string, error) {
	classifications := make(map[string]string, len(properties))

	for field := range properties {
		marking, err := b.markingSvc.GetFieldMarking(ctx, objectType, field)
		if err != nil {
			classifications[field] = "internal"
			continue
		}
		classifications[field] = mapClassification(marking.Classification, marking.PII)
	}

	return classifications, nil
}

// buildLineage fetches context lineage from the DecisionLineageService
// and maps it to the LineageData struct for the governance section.
func (b *ContextBuilderV2) buildLineage(ctx context.Context, caseID string) (*LineageData, error) {
	if b.lineageSvc == nil {
		return nil, nil
	}

	contextLineage, err := b.lineageSvc.GetContextLineage(ctx, caseID)
	if err != nil {
		return nil, fmt.Errorf("get context lineage for %s: %w", caseID, err)
	}

	if contextLineage == nil || len(contextLineage.UpstreamTables) == 0 {
		return nil, nil
	}

	return &LineageData{
		Upstream: contextLineage.UpstreamTables,
	}, nil
}

// buildPolicyResult evaluates the action type provider and returns a PolicyResult.
func (b *ContextBuilderV2) buildPolicyResult() *PolicyResult {
	actionTypes := b.actionTypes.ListActionTypes()

	allowedActions := make([]string, 0, len(actionTypes))
	blockedActions := make(map[string]string)
	riskLevels := make(map[string]string)
	requiresApprovalActions := make([]string, 0)
	evidenceSources := make([]string, 0)
	humanApprovalRequired := false

	seenEvidence := make(map[string]struct{})

	for _, a := range actionTypes {
		policy, ok := b.actionTypes.GetActionPolicy(a)
		if !ok {
			if b.actionTypes.IsActionAllowed(a) {
				allowedActions = append(allowedActions, a)
			} else {
				blockedActions[a] = "not configured in action registry"
			}
			continue
		}

		if b.actionTypes.IsActionAllowed(a) {
			allowedActions = append(allowedActions, a)
		} else {
			blockedActions[a] = "disabled by action registry policy"
			continue
		}

		if policy.RiskLevel != "" {
			riskLevels[a] = policy.RiskLevel
		}

		if policy.RequiresApproval {
			humanApprovalRequired = true
			requiresApprovalActions = append(requiresApprovalActions, a)
		}

		for _, source := range policy.AllowedBy {
			if _, seen := seenEvidence[source]; !seen {
				evidenceSources = append(evidenceSources, source)
				seenEvidence[source] = struct{}{}
			}
		}
	}

	if len(allowedActions) == 0 {
		allowedActions = nil
	}
	if len(blockedActions) == 0 {
		blockedActions = nil
	}
	if len(riskLevels) == 0 {
		riskLevels = nil
	}
	if len(requiresApprovalActions) == 0 {
		requiresApprovalActions = nil
	}
	if len(evidenceSources) == 0 {
		evidenceSources = nil
	}

	return &PolicyResult{
		AllowedActions:          allowedActions,
		BlockedActions:          blockedActions,
		RiskLevels:              riskLevels,
		HumanApprovalRequired:   humanApprovalRequired,
		RequiresApprovalActions: requiresApprovalActions,
		EvidenceSources:         evidenceSources,
	}
}

// BuildEnvelope constructs an LLMSafeContextEnvelope for the given case.
// This is the versioned, auditable wrapper that gets persisted as a snapshot
// before the LLM call, enabling replay and audit.
func (b *ContextBuilderV2) BuildEnvelope(ctx context.Context, caseID string, promptVersion string) (*llm.LLMSafeContextEnvelope, error) {
	decisionCtx, err := b.BuildDecisionContext(ctx, caseID)
	if err != nil {
		return nil, fmt.Errorf("build decision context: %w", err)
	}

	llmSafeCtx := BuildLLMSafeContext(decisionCtx)
	contextHash, err := ComputeContextHash(llmSafeCtx)
	if err != nil {
		return nil, fmt.Errorf("compute context hash: %w", err)
	}

	alertID := ""
	if decisionCtx.SourceType != nil && *decisionCtx.SourceType == "alert" && decisionCtx.SourceID != nil {
		alertID = *decisionCtx.SourceID
	}

	evidence := buildEvidenceItems(decisionCtx)

	redactionSummary := llm.RedactionSummary{
		TotalFields:   len(decisionCtx.ObjectContext.Properties) + len(decisionCtx.Governance.RedactedFields),
		RedactedCount: len(decisionCtx.Governance.RedactedFields),
		RedactedList:  decisionCtx.Governance.RedactedFields,
		AppliedRole:   decisionCtx.Governance.Role,
	}

	configVersions := make(map[string]string)
	if decisionCtx.SourceType != nil {
		configVersions["source_type"] = *decisionCtx.SourceType
	}

	envelope := &llm.LLMSafeContextEnvelope{
		SchemaVersion:    "llm_safe_context.v1",
		CaseID:           caseID,
		AlertID:          alertID,
		ContextHash:      contextHash,
		BuiltAt:          time.Now(),
		Trigger:          llmSafeCtx.Trigger,
		ObjectContext:    llmSafeCtx.ObjectContext,
		Evidence:         evidence,
		AllowedActions:   llmSafeCtx.AllowedActions,
		ForbiddenActions: llmSafeCtx.ForbiddenActions,
		Governance:       llmSafeCtx.GovernanceInfo,
		RedactionSummary: redactionSummary,
		PromptVersion:    promptVersion,
		ConfigVersions:   configVersions,
	}

	return envelope, nil
}

// buildEvidenceItems extracts evidence items from a DecisionContext for inclusion in the envelope.
func buildEvidenceItems(dc *DecisionContext) []llm.EvidenceItem {
	var items []llm.EvidenceItem

	if dc.Trigger.AlertID != "" {
		items = append(items, llm.EvidenceItem{Type: "alert", Key: "alert_id", Value: dc.Trigger.AlertID})
	}
	if dc.Trigger.RuleID != "" {
		items = append(items, llm.EvidenceItem{Type: "alert", Key: "rule_id", Value: dc.Trigger.RuleID})
	}
	if dc.Trigger.MetricName != "" {
		items = append(items, llm.EvidenceItem{Type: "metric", Key: "metric_name", Value: dc.Trigger.MetricName})
		items = append(items, llm.EvidenceItem{Type: "metric", Key: "current_value", Value: dc.Trigger.CurrentValue})
		items = append(items, llm.EvidenceItem{Type: "metric", Key: "baseline_value", Value: dc.Trigger.BaselineValue})
		items = append(items, llm.EvidenceItem{Type: "metric", Key: "delta_pct", Value: dc.Trigger.DeltaPct})
	}
	if dc.Governance.Classification != "" {
		items = append(items, llm.EvidenceItem{Type: "classification", Key: "overall_level", Value: dc.Governance.Classification})
	}

	if items == nil {
		items = []llm.EvidenceItem{}
	}
	return items
}

var _ interface {
	BuildDecisionContext(ctx context.Context, caseID string) (*DecisionContext, error)
} = (*ContextBuilderV2)(nil)
