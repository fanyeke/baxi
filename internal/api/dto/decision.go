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
