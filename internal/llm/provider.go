package llm

import "context"

// DecisionProvider generates structured decisions from LLM-safe context.
type DecisionProvider interface {
	GenerateDecision(ctx context.Context, input LLMSafeContext) (*DecisionOutput, error)
}

// LLMSafeContext is the governance-redacted context fed to decision providers.
type LLMSafeContext struct {
	CaseID           string         `json:"case_id"`
	Trigger          TriggerInfo    `json:"trigger"`
	ObjectContext    ObjectContext  `json:"object_context"`
	GovernanceInfo   GovernanceInfo `json:"governance"`
	AllowedActions   []string       `json:"allowed_actions"`
	ForbiddenActions []string       `json:"forbidden_actions"`
}

// TriggerInfo describes the event that triggered a decision request.
type TriggerInfo struct {
	AlertID      string  `json:"alert_id"`
	RuleID       string  `json:"rule_id"`
	Severity     string  `json:"severity"`
	MetricName   string  `json:"metric_name"`
	CurrentValue float64 `json:"current_value"`
	BaselineValue float64 `json:"baseline_value"`
	DeltaPct     float64 `json:"delta_pct"`
}

// ObjectContext describes the target object of a decision.
type ObjectContext struct {
	ObjectType string                 `json:"object_type"`
	ObjectID   string                 `json:"object_id"`
	Properties map[string]interface{} `json:"properties"`
}

// GovernanceInfo carries redaction and access-control metadata.
type GovernanceInfo struct {
	Classification   string   `json:"classification"`
	RedactionApplied bool     `json:"redaction_applied"`
	RedactedFields   []string `json:"redacted_fields"`
	Role             string   `json:"role"`
}

// DecisionType values.
const (
	DecisionTypeMonitor      = "monitor_only"
	DecisionTypeInvestigate  = "investigate"
	DecisionTypeOptimize     = "optimize"
	DecisionTypeIntervention = "intervention"
	DecisionTypeExperiment   = "experiment"
)

// DecisionSeverity values.
const (
	SeverityLow      = "low"
	SeverityMedium   = "medium"
	SeverityHigh     = "high"
	SeverityCritical = "critical"
)

// ActionType values.
const (
	ActionTypeCreateFollowupTask = "create_followup_task"
	ActionTypeNotifyOwner        = "notify_owner"
	ActionTypeExportReport       = "export_report"
	ActionTypeCreateOutboxMessage = "create_outbox_message"
)

// DecisionOutput is the structured result from a DecisionProvider.
type DecisionOutput struct {
	SchemaVersion      string               `json:"schema_version"`        // "decision_output.v1" or empty for legacy
	DecisionType       string               `json:"decision_type"`
	Severity           string               `json:"severity"`
	Summary            string               `json:"summary"`
	Rationale          []string             `json:"rationale"`
	RecommendedActions []RecommendedAction  `json:"recommended_actions"`
	Confidence         float64              `json:"confidence"`
	RequiresHumanReview bool                `json:"requires_human_review"`
}

// RecommendedAction is a single action suggested by a decision.
type RecommendedAction struct {
	ActionType string                 `json:"action_type"`
	Priority   string                 `json:"priority"`
	OwnerRole  string                 `json:"owner_role"`
	Payload    map[string]interface{} `json:"payload"`
}
