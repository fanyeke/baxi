package eval

import (
	"encoding/json"
	"math"
	"time"

	"baxi/internal/llm"
)

// DecisionComparison holds the result of comparing two decisions.
type DecisionComparison struct {
	DecisionCaseID     string          `json:"decision_case_id"`
	LLMDecisionType    string          `json:"llm_decision_type"`
	RuleDecisionType   string          `json:"rule_decision_type"`
	DecisionTypeMatch  bool            `json:"decision_type_match"`
	SeverityMatch      bool            `json:"severity_match"`
	ActionOverlap      float64         `json:"action_overlap"` // Jaccard index
	LLMValid           bool            `json:"llm_valid"`
	RuleValid          bool            `json:"rule_valid"`
	ConfidenceDiff     float64         `json:"confidence_diff"`
	LLMRequiresReview  bool            `json:"llm_requires_review"`
	RuleRequiresReview bool            `json:"rule_requires_review"`
	ComparisonJSON     json.RawMessage `json:"comparison_json"`
	CreatedAt          time.Time       `json:"created_at"`
}

// Compare compares two decision outputs across multiple dimensions.
func Compare(caseID string, llmDecision, ruleDecision *llm.DecisionOutput) *DecisionComparison {
	c := &DecisionComparison{
		DecisionCaseID:    caseID,
		DecisionTypeMatch: llmDecision.DecisionType == ruleDecision.DecisionType,
		SeverityMatch:     llmDecision.Severity == ruleDecision.Severity,
		ActionOverlap:     jaccardIndex(llmDecision.RecommendedActions, ruleDecision.RecommendedActions),
		LLMValid:          true,
		RuleValid:         true,
		ConfidenceDiff:    math.Abs(llmDecision.Confidence - ruleDecision.Confidence),
		LLMRequiresReview:  llmDecision.RequiresHumanReview,
		RuleRequiresReview: ruleDecision.RequiresHumanReview,
		CreatedAt:         time.Now(),
	}

	if llmDecision.DecisionType != "" {
		c.LLMDecisionType = llmDecision.DecisionType
	}
	if ruleDecision.DecisionType != "" {
		c.RuleDecisionType = ruleDecision.DecisionType
	}

	// Marshal comparison fields to JSON
	c.ComparisonJSON, _ = json.Marshal(c)

	return c
}

// jaccardIndex computes the Jaccard index (intersection over union) of two action slices.
func jaccardIndex(a, b []llm.RecommendedAction) float64 {
	setA := make(map[string]bool)
	setB := make(map[string]bool)
	for _, action := range a {
		setA[action.ActionType] = true
	}
	for _, action := range b {
		setB[action.ActionType] = true
	}

	intersection := 0
	for k := range setA {
		if setB[k] {
			intersection++
		}
	}

	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 1.0 // both empty = identical
	}

	return float64(intersection) / float64(union)
}
