package decision

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"baxi/internal/governance"
	"baxi/internal/ontology"
	"baxi/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ObjectDataProvider provides object context data.
type ObjectDataProvider interface {
	BuildObjectContext(ctx context.Context, objectType, objectID string) (*ontology.ObjectContext, error)
	GetMetricAlert(ctx context.Context, alertID string) (*repository.ObjectInstance, error)
}

// ClassificationProvider provides data classification info.
type ClassificationProvider interface {
	GetFieldMarking(ctx context.Context, objectType, property string) (string, bool, bool, error)
}

// DecisionCaseDataProvider provides decision case data.
type DecisionCaseDataProvider interface {
	GetCaseByID(ctx context.Context, pool *pgxpool.Pool, caseID string) (*repository.DecisionCaseRow, error)
	GetCaseBySource(ctx context.Context, pool *pgxpool.Pool, sourceType, sourceID string) (*repository.DecisionCaseRow, error)
}

// ActionTypeProvider provides action type information for decision contexts.
type ActionTypeProvider interface {
	ListActionTypes() []string
	IsActionAllowed(actionType string) bool
	GetActionPolicy(actionType string) (ActionPolicy, bool)
}

// ActionPolicy holds the policy configuration for an action type.
type ActionPolicy struct {
	RiskLevel        string
	RequiresApproval bool
	AllowedBy        []string
}

// PolicyResult holds the evaluated policy result for a decision context.
type PolicyResult struct {
	AllowedActions          []string          `json:"allowed_actions"`
	BlockedActions          map[string]string `json:"blocked_actions"`
	RiskLevels              map[string]string `json:"risk_levels"`
	HumanApprovalRequired   bool              `json:"human_approval_required"`
	RequiresApprovalActions []string          `json:"requires_approval_actions"`
	EvidenceSources         []string          `json:"evidence_sources"`
}

// Compile-time interface checks.
var _ DecisionCaseDataProvider = (*repository.DecisionRepository)(nil)

// EnrichedObjectData holds the context of a linked object discovered via
// OAG (Object-Action-Governance) link traversal.
type EnrichedObjectData struct {
	LinkName   string                 `json:"link_name"`
	Depth      int                    `json:"depth"`
	ObjectType string                 `json:"object_type"`
	ObjectID   string                 `json:"object_id"`
	Properties map[string]interface{} `json:"properties"`
}

// DecisionContext is the full domain context for a decision case.
type DecisionContext struct {
	DecisionCaseID   string               `json:"decision_case_id"`
	SourceType       *string              `json:"source_type"`
	SourceID         *string              `json:"source_id"`
	Trigger          TriggerInfo          `json:"trigger"`
	ObjectContext    ObjectContextData    `json:"object_context"`
	Governance       GovernanceData       `json:"governance"`
	AllowedActions   []string             `json:"allowed_actions"`
	ForbiddenActions []string             `json:"forbidden_actions"`
	Policy           *PolicyResult        `json:"policy,omitempty"`
	EnrichedObjects  []EnrichedObjectData `json:"enriched_objects,omitempty"`
}

// TriggerInfo holds the alert/metric data that triggered the decision case.
type TriggerInfo struct {
	AlertID       string  `json:"alert_id"`
	RuleID        string  `json:"rule_id"`
	Severity      string  `json:"severity"`
	MetricName    string  `json:"metric_name"`
	CurrentValue  float64 `json:"current_value"`
	BaselineValue float64 `json:"baseline_value"`
	DeltaPct      float64 `json:"delta_pct"`
}

// ObjectContextData is a lightweight representation of an object for LLM context.
type ObjectContextData struct {
	ObjectType string                 `json:"object_type"`
	ObjectID   string                 `json:"object_id"`
	Properties map[string]interface{} `json:"properties"`
}

// GovernanceData holds classification, lineage, and redaction metadata.
type GovernanceData struct {
	Classification   string       `json:"classification"`
	Lineage          *LineageData `json:"lineage,omitempty"`
	RedactionApplied bool         `json:"redaction_applied"`
	RedactedFields   []string     `json:"redacted_fields"`
	Role             string       `json:"role"`
}

// LineageData holds upstream and downstream lineage.
type LineageData struct {
	Upstream   []string `json:"upstream,omitempty"`
	Downstream []string `json:"downstream,omitempty"`
}

// LLMSafeContext is the sanitized, hash-verified input to a DecisionProvider.
type LLMSafeContext struct {
	CaseID           string            `json:"case_id"`
	Trigger          TriggerInfo       `json:"trigger"`
	ObjectContext    ObjectContextData `json:"object_context"`
	GovernanceInfo   GovernanceData    `json:"governance"`
	AllowedActions   []string          `json:"allowed_actions"`
	ForbiddenActions []string          `json:"forbidden_actions"`
	ContextHash      string            `json:"context_hash"`
}

// ContextBuilder builds governed LLM-safe decision contexts by composing Phase 5 services.
type ContextBuilder struct {
	caseSvc        DecisionCaseDataProvider
	objectProvider ObjectDataProvider
	classProvider  ClassificationProvider
	pool           *pgxpool.Pool
	actionTypes    ActionTypeProvider
}

// NewContextBuilder creates a ContextBuilder backed by the given providers.
func NewContextBuilder(
	caseSvc DecisionCaseDataProvider,
	objectProvider ObjectDataProvider,
	classProvider ClassificationProvider,
	pool *pgxpool.Pool,
	actionTypes ActionTypeProvider,
) *ContextBuilder {
	return &ContextBuilder{
		caseSvc:        caseSvc,
		objectProvider: objectProvider,
		classProvider:  classProvider,
		pool:           pool,
		actionTypes:    actionTypes,
	}
}

// BuildDecisionContext constructs a full DecisionContext for the given case ID.
func (b *ContextBuilder) BuildDecisionContext(ctx context.Context, caseID string) (*DecisionContext, error) {
	caseRow, err := b.caseSvc.GetCaseByID(ctx, b.pool, caseID)
	if err != nil {
		return nil, fmt.Errorf("fetch case %s: %w", caseID, err)
	}

	objectType := derefString(caseRow.ObjectType)
	objectID := derefString(caseRow.ObjectID)
	severity := derefString(caseRow.Severity)

	trigger, err := b.buildTrigger(ctx, caseRow, severity)
	if err != nil {
		return nil, err
	}

	objectCtx, err := b.objectProvider.BuildObjectContext(ctx, objectType, objectID)
	if err != nil {
		return nil, fmt.Errorf("build object context for %s/%s: %w", objectType, objectID, err)
	}

	classifications := make(map[string]string, len(objectCtx.Properties))
	for field := range objectCtx.Properties {
		level, isPII, _, err := b.classProvider.GetFieldMarking(ctx, objectType, field)
		if err != nil {
			continue
		}
		classifications[field] = mapClassification(level, isPII)
	}

	markings := make(map[string]string)
	policy := governance.RedactionPolicy{Role: "agent_readonly"}
	redactionResult := governance.RedactObjectContext(objectCtx.Properties, classifications, markings, policy)

	redactedFieldNames := make([]string, len(redactionResult.RedactedFields))
	for i, rf := range redactionResult.RedactedFields {
		redactedFieldNames[i] = rf.Field
	}

	overallClassification := resolveOverallClassification(classifications)

	govData := GovernanceData{
		Classification:   overallClassification,
		RedactionApplied: len(redactedFieldNames) > 0,
		RedactedFields:   redactedFieldNames,
		Role:             "agent_readonly",
	}

	allowedActions := []string{
		"create_followup_task",
		"notify_owner",
		"export_report",
		"escalate_to_human",
	}
	forbiddenActions := []string{
		"execute_dispatch",
		"modify_raw_data",
		"write_dwd",
		"write_mart",
	}

	decisionCtx := &DecisionContext{
		DecisionCaseID: caseRow.CaseID,
		SourceType:     caseRow.SourceType,
		SourceID:       caseRow.SourceID,
		Trigger:        trigger,
		ObjectContext: ObjectContextData{
			ObjectType: objectCtx.ObjectType,
			ObjectID:   objectCtx.ObjectID,
			Properties: redactionResult.Properties,
		},
		Governance:       govData,
		AllowedActions:   allowedActions,
		ForbiddenActions: forbiddenActions,
	}

	return decisionCtx, nil
}

// BuildLLMSafeContext maps a DecisionContext to an LLMSafeContext and adds a SHA256 context hash.
func (b *ContextBuilder) BuildLLMSafeContext(ctx context.Context, decisionContext *DecisionContext) (*LLMSafeContext, error) {
	llmCtx := &LLMSafeContext{
		CaseID:           decisionContext.DecisionCaseID,
		Trigger:          decisionContext.Trigger,
		ObjectContext:    decisionContext.ObjectContext,
		GovernanceInfo:   decisionContext.Governance,
		AllowedActions:   decisionContext.AllowedActions,
		ForbiddenActions: decisionContext.ForbiddenActions,
	}

	hash, err := ComputeContextHash(llmCtx)
	if err != nil {
		return nil, fmt.Errorf("compute context hash: %w", err)
	}
	llmCtx.ContextHash = hash

	return llmCtx, nil
}

// ComputeContextHash returns the hex-encoded SHA256 hash of the JSON-serialized context.
func ComputeContextHash(context interface{}) (string, error) {
	data, err := json.Marshal(context)
	if err != nil {
		return "", fmt.Errorf("marshal context: %w", err)
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

func (b *ContextBuilder) buildTrigger(ctx context.Context, caseRow *repository.DecisionCaseRow, severity string) (TriggerInfo, error) {
	trigger := TriggerInfo{
		Severity: severity,
	}

	if caseRow.AlertID == nil || *caseRow.AlertID == "" {
		return trigger, nil
	}

	alert, err := b.objectProvider.GetMetricAlert(ctx, *caseRow.AlertID)
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

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func getStringProp(props map[string]interface{}, key string) string {
	if v, ok := props[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getFloatProp(props map[string]interface{}, key string) float64 {
	if v, ok := props[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case float32:
			return float64(n)
		case int:
			return float64(n)
		case int64:
			return float64(n)
		case int32:
			return float64(n)
		}
	}
	return 0
}

func mapClassification(level string, isPII bool) string {
	if isPII {
		return "pii"
	}
	switch level {
	case "L3":
		return "sensitive"
	case "L2":
		return "internal"
	case "L1":
		return "public_internal"
	default:
		return "internal"
	}
}

func resolveOverallClassification(classifications map[string]string) string {
	overall := "L1"
	for _, class := range classifications {
		switch class {
		case "pii", "sensitive":
			return "L3"
		case "internal", "derived_sensitive":
			if overall != "L3" {
				overall = "L2"
			}
		}
	}
	return overall
}
