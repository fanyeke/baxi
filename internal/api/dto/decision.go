// Package dto provides data transfer objects for API responses.
package dto

import "encoding/json"

// CreateCaseRequest is the request body for creating a decision case.
type CreateCaseRequest struct {
	SourceType string `json:"source_type"`
	SourceID   string `json:"source_id"`
}

// CreateCaseResponse is the response for case creation.
type CreateCaseResponse struct {
	DecisionCaseID string `json:"decision_case_id"`
	SourceType     string `json:"source_type"`
	SourceID       string `json:"source_id"`
	Status         string `json:"status"`
}

// DecisionCaseResponse represents a full decision case.
type DecisionCaseResponse struct {
	DecisionCaseID string `json:"decision_case_id"`
	SourceType     string `json:"source_type"`
	SourceID       string `json:"source_id"`
	ObjectType     string `json:"object_type,omitempty"`
	ObjectID       string `json:"object_id,omitempty"`
	Severity       string `json:"severity,omitempty"`
	Status         string `json:"status"`
	ContextHash    string `json:"context_hash,omitempty"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at,omitempty"`
}

// CaseListResponse is the paginated list response.
type CaseListResponse struct {
	Items []DecisionCaseResponse `json:"items"`
	Total int                    `json:"total"`
}

// DecisionContextResponse contains trigger and governance info.
type DecisionContextResponse struct {
	DecisionCaseID string                 `json:"decision_case_id"`
	Status         string                 `json:"status"`
	Trigger        map[string]interface{} `json:"trigger"`
	ObjectContext  map[string]interface{} `json:"object_context"`
	Governance     map[string]interface{} `json:"governance"`
	AllowedActions []string               `json:"allowed_actions"`
}

// DecisionResponse contains the generated decision.
type DecisionResponse struct {
	DecisionCaseID string                 `json:"decision_case_id"`
	Status         string                 `json:"status"`
	Decision       map[string]interface{} `json:"decision"`
	Proposals      []ProposalItem         `json:"proposals"`
}

// ProposalItem represents a proposal in API responses.
type ProposalItem struct {
	ProposalID          string `json:"proposal_id"`
	ActionType          string `json:"action_type"`
	Title               string `json:"title"`
	RiskLevel           string `json:"risk_level"`
	RequiresHumanReview bool   `json:"requires_human_review"`
	ApplyStatus         string `json:"apply_status"`
	CreatedAt           string `json:"created_at"`
}

// ProposalListResponse is the list of proposals.
type ProposalListResponse struct {
	Items []ProposalItem `json:"items"`
}

// LLMDecisionItem represents a single LLM decision record in API responses.
type LLMDecisionItem struct {
	DecisionID       string          `json:"decision_id"`
	CaseID           string          `json:"case_id"`
	Provider         string          `json:"provider"`
	Model            string          `json:"model"`
	Confidence       float64         `json:"confidence"`
	ValidationStatus string          `json:"validation_status"`
	FallbackUsed     bool            `json:"fallback_used"`
	OutputJSON       json.RawMessage `json:"output_json,omitempty"`
	CreatedAt        string          `json:"created_at"`
}

// EvalItem represents a single evaluation result in API responses.
type EvalItem struct {
	EvalID        string          `json:"eval_id"`
	LLMDecisionID string          `json:"llm_decision_id"`
	CaseID        string          `json:"decision_case_id"`
	EvalRuleID    string          `json:"eval_rule_id"`
	EvalStatus    string          `json:"eval_status"`
	Score         float64         `json:"score,omitempty"`
	DetailsJSON   json.RawMessage `json:"details_json,omitempty"`
	CreatedAt     string          `json:"created_at"`
}

// LLMDecisionListResponse is the paginated list response for LLM decisions.
type LLMDecisionListResponse struct {
	Items []LLMDecisionItem `json:"items"`
	Total int               `json:"total"`
}

// EvalListResponse is the paginated list response for eval results.
type EvalListResponse struct {
	Items []EvalItem `json:"items"`
	Total int        `json:"total"`
}

// CaseFilter represents query filter parameters.
type CaseFilter struct {
	SourceType string
	Status     string
	Severity   string
	Limit      int
	Offset     int
}

// DiffItem represents a single field difference in a decision comparison.
type DiffItem struct {
	Field      string      `json:"field"`
	Before     interface{} `json:"before,omitempty"`
	After      interface{} `json:"after,omitempty"`
	ChangeType string      `json:"change_type"`
}

// CompareMeta contains metadata about a decision comparison.
type CompareMeta struct {
	DecisionTypeMatch bool    `json:"decision_type_match"`
	SeverityMatch     bool    `json:"severity_match"`
	ActionOverlap     float64 `json:"action_overlap"`
	ConfidenceDiff    float64 `json:"confidence_diff"`
	CreatedAt         string  `json:"created_at"`
}

// CompareResponse is the response for the compare endpoint.
type CompareResponse struct {
	DecisionCaseID string      `json:"decision_case_id"`
	Added          []DiffItem  `json:"added"`
	Removed        []DiffItem  `json:"removed"`
	Changed        []DiffItem  `json:"changed"`
	Metadata       CompareMeta `json:"metadata"`
}

// ReplayRequest is the request body for replaying a decision.
type ReplayRequest struct {
	DryRun           bool                   `json:"dry_run"`
	Model            string                 `json:"model,omitempty"`
	Temperature      float64                `json:"temperature,omitempty"`
	ContextOverrides map[string]interface{} `json:"context_overrides,omitempty"`
}

// ReplayDiff contains the diff between original and replayed decisions.
type ReplayDiff struct {
	DecisionTypeMatch bool    `json:"decision_type_match"`
	SeverityMatch     bool    `json:"severity_match"`
	ConfidenceDiff    float64 `json:"confidence_diff"`
	ActionOverlap     float64 `json:"action_overlap"`
	SummaryChanged    bool    `json:"summary_changed"`
	RationaleChanged  bool    `json:"rationale_changed"`
}

// ReplayResponse is the response for the replay endpoint.
type ReplayResponse struct {
	OriginalDecision map[string]interface{} `json:"original_decision"`
	ReplayedDecision map[string]interface{} `json:"replayed_decision,omitempty"`
	Diff             *ReplayDiff            `json:"diff,omitempty"`
	ContextHash      string                 `json:"context_hash"`
	Model            string                 `json:"model"`
	DryRun           bool                   `json:"dry_run"`
}
