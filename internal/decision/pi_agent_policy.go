package decision

import (
	"encoding/json"
	"fmt"
)

// DecisionJSON represents a Pi Agent decision for write-back infrastructure.
// It captures the decision output along with contextual metadata required for
// traceability, governance, and automated action recommendation.
type DecisionJSON struct {
	CaseID             string              `json:"case_id"`
	RecipeID           string              `json:"recipe_id"`
	ContextHash        string              `json:"context_hash"`
	Severity           string              `json:"severity"`
	Confidence         float64             `json:"confidence"`
	DecisionSummary    string              `json:"decision_summary"`
	EvidenceRefs       []string            `json:"evidence_refs"`
	RecommendedActions []RecommendAction   `json:"recommended_actions"`
}

// RecommendAction is a single action suggested by a Pi Agent decision.
// Each action carries a risk level and approval gate for governance enforcement.
type RecommendAction struct {
	ActionType       string          `json:"action_type"`
	RiskLevel        string          `json:"risk_level"`
	RequiresApproval bool            `json:"requires_approval"`
	Payload          json.RawMessage `json:"payload"`
}

// validSeverities is the set of allowed decision severity values.
var validSeverities = map[string]bool{
	"critical": true,
	"high":     true,
	"medium":   true,
	"low":      true,
	"info":     true,
}

// ValidateDecision validates a DecisionJSON against schema rules.
// It checks that severity is a recognised enum value, confidence falls within
// [0,1], and both evidence_refs and recommended_actions are non-empty.
func ValidateDecision(d *DecisionJSON) error {
	if d == nil {
		return fmt.Errorf("decision is nil")
	}

	if !validSeverities[d.Severity] {
		return fmt.Errorf("invalid severity: %s (must be one of: critical, high, medium, low, info)", d.Severity)
	}

	if d.Confidence < 0 || d.Confidence > 1 {
		return fmt.Errorf("confidence out of range [0,1]: %f", d.Confidence)
	}

	if len(d.EvidenceRefs) == 0 {
		return fmt.Errorf("evidence_refs must not be empty")
	}

	if len(d.RecommendedActions) == 0 {
		return fmt.Errorf("recommended_actions must not be empty")
	}

	return nil
}

// AgentWritePolicy defines which write operations a Pi Agent is permitted to perform.
// It encodes both an allowlist and a denylist of tool names for governance enforcement.
type AgentWritePolicy struct {
	CanWrite       bool     `json:"can_write"`
	CannotWrite    bool     `json:"cannot_write"`
	AllowedTools   []string `json:"allowed_tools"`
	ForbiddenTools []string `json:"forbidden_tools"`
}

// DefaultPiAgentPolicy returns an AgentWritePolicy that permits only
// ai.llm_decision and ai.action_proposal write operations. All other tools
// are implicitly forbidden by the absence of an allowlist entry. An explicit
// ForbiddenTools slice provides an extension point for future restrictions.
func DefaultPiAgentPolicy() AgentWritePolicy {
	return AgentWritePolicy{
		CanWrite:    true,
		CannotWrite: false,
		AllowedTools: []string{
			"ai.llm_decision",
			"ai.action_proposal",
		},
		ForbiddenTools: []string{},
	}
}

// ValidateProposalTrace validates a proposal trace by ensuring every required
// field is non-empty. This is used to enforce completeness on write-back records
// before they are persisted or forwarded.
func ValidateProposalTrace(decisionID string, evidenceRefs []string, contextHash string, recipeID string) error {
	if decisionID == "" {
		return fmt.Errorf("decision_id must not be empty")
	}
	if len(evidenceRefs) == 0 {
		return fmt.Errorf("evidence_refs must not be empty")
	}
	if contextHash == "" {
		return fmt.Errorf("context_hash must not be empty")
	}
	if recipeID == "" {
		return fmt.Errorf("recipe_id must not be empty")
	}
	return nil
}
