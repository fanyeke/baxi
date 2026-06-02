package decision

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func validDecisionJSON() *DecisionJSON {
	return &DecisionJSON{
		CaseID:          "dc_1234567890_abc123",
		RecipeID:        "recipe_001",
		ContextHash:     "sha256_abc123",
		Severity:        "high",
		Confidence:      0.85,
		DecisionSummary: "Test decision summary",
		EvidenceRefs:    []string{"ev_001", "ev_002"},
		RecommendedActions: []RecommendAction{
			{
				ActionType:       "ai.llm_decision",
				RiskLevel:        "medium",
				RequiresApproval: true,
				Payload:          json.RawMessage(`{"key": "value"}`),
			},
		},
	}
}

func TestValidateDecision_Valid(t *testing.T) {
	d := validDecisionJSON()
	err := ValidateDecision(d)
	assert.NoError(t, err)
}

func TestValidateDecision_Nil(t *testing.T) {
	err := ValidateDecision(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestValidateDecision_InvalidSeverity(t *testing.T) {
	d := validDecisionJSON()
	d.Severity = "unknown"
	err := ValidateDecision(d)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid severity")
}

func TestValidateDecision_ValidSeverities(t *testing.T) {
	sevs := []string{"critical", "high", "medium", "low", "info"}
	for _, sev := range sevs {
		t.Run(sev, func(t *testing.T) {
			d := validDecisionJSON()
			d.Severity = sev
			err := ValidateDecision(d)
			assert.NoError(t, err)
		})
	}
}

func TestValidateDecision_ConfidenceOutOfRange(t *testing.T) {
	tests := []struct {
		name       string
		confidence float64
	}{
		{"negative", -0.1},
		{"above_one", 1.1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := validDecisionJSON()
			d.Confidence = tt.confidence
			err := ValidateDecision(d)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "confidence out of range")
		})
	}
}

func TestValidateDecision_ConfidenceBoundary(t *testing.T) {
	tests := []struct {
		name       string
		confidence float64
	}{
		{"zero", 0},
		{"one", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := validDecisionJSON()
			d.Confidence = tt.confidence
			err := ValidateDecision(d)
			assert.NoError(t, err)
		})
	}
}

func TestValidateDecision_EmptyEvidenceRefs(t *testing.T) {
	d := validDecisionJSON()
	d.EvidenceRefs = []string{}
	err := ValidateDecision(d)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "evidence_refs must not be empty")
}

func TestValidateDecision_NilEvidenceRefs(t *testing.T) {
	d := validDecisionJSON()
	d.EvidenceRefs = nil
	err := ValidateDecision(d)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "evidence_refs must not be empty")
}

func TestValidateDecision_EmptyRecommendedActions(t *testing.T) {
	d := validDecisionJSON()
	d.RecommendedActions = []RecommendAction{}
	err := ValidateDecision(d)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "recommended_actions must not be empty")
}

func TestValidateDecision_NilRecommendedActions(t *testing.T) {
	d := validDecisionJSON()
	d.RecommendedActions = nil
	err := ValidateDecision(d)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "recommended_actions must not be empty")
}

func TestDefaultPiAgentPolicy(t *testing.T) {
	policy := DefaultPiAgentPolicy()
	assert.True(t, policy.CanWrite)
	assert.False(t, policy.CannotWrite)

	allowed := policy.AllowedTools
	assert.Contains(t, allowed, "ai.llm_decision")
	assert.Contains(t, allowed, "ai.action_proposal")
	assert.Len(t, allowed, 2)

	assert.Empty(t, policy.ForbiddenTools)
}

func TestValidateProposalTrace_Valid(t *testing.T) {
	err := ValidateProposalTrace("dec_001", []string{"ev_001"}, "hash_abc", "recipe_001")
	assert.NoError(t, err)
}

func TestValidateProposalTrace_EmptyDecisionID(t *testing.T) {
	err := ValidateProposalTrace("", []string{"ev_001"}, "hash_abc", "recipe_001")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decision_id must not be empty")
}

func TestValidateProposalTrace_EmptyEvidenceRefs(t *testing.T) {
	err := ValidateProposalTrace("dec_001", []string{}, "hash_abc", "recipe_001")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "evidence_refs must not be empty")
}

func TestValidateProposalTrace_NilEvidenceRefs(t *testing.T) {
	err := ValidateProposalTrace("dec_001", nil, "hash_abc", "recipe_001")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "evidence_refs must not be empty")
}

func TestValidateProposalTrace_EmptyContextHash(t *testing.T) {
	err := ValidateProposalTrace("dec_001", []string{"ev_001"}, "", "recipe_001")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context_hash must not be empty")
}

func TestValidateProposalTrace_EmptyRecipeID(t *testing.T) {
	err := ValidateProposalTrace("dec_001", []string{"ev_001"}, "hash_abc", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "recipe_id must not be empty")
}

func TestRecommendAction_JSONSerialization(t *testing.T) {
	action := RecommendAction{
		ActionType:       "ai.llm_decision",
		RiskLevel:        "high",
		RequiresApproval: true,
		Payload:          json.RawMessage(`{"detail": "test"}`),
	}

	data, err := json.Marshal(action)
	assert.NoError(t, err)

	var decoded RecommendAction
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, action.ActionType, decoded.ActionType)
	assert.Equal(t, action.RiskLevel, decoded.RiskLevel)
	assert.True(t, decoded.RequiresApproval)
}

func TestDecisionJSON_JSONSerialization(t *testing.T) {
	d := validDecisionJSON()

	data, err := json.Marshal(d)
	assert.NoError(t, err)

	var decoded DecisionJSON
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, d.CaseID, decoded.CaseID)
	assert.Equal(t, d.RecipeID, decoded.RecipeID)
	assert.Equal(t, d.Severity, decoded.Severity)
	assert.Equal(t, d.Confidence, decoded.Confidence)
	assert.Len(t, decoded.EvidenceRefs, len(d.EvidenceRefs))
	assert.Len(t, decoded.RecommendedActions, 1)
}

func TestAgentWritePolicy_JSONSerialization(t *testing.T) {
	policy := DefaultPiAgentPolicy()

	data, err := json.Marshal(policy)
	assert.NoError(t, err)

	var decoded AgentWritePolicy
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.True(t, decoded.CanWrite)
	assert.False(t, decoded.CannotWrite)
	assert.Len(t, decoded.AllowedTools, 2)
	assert.Empty(t, decoded.ForbiddenTools)
}

func TestDefaultPiAgentPolicy_NotNilTools(t *testing.T) {
	policy := DefaultPiAgentPolicy()
	assert.NotNil(t, policy.AllowedTools)
	assert.NotNil(t, policy.ForbiddenTools)
}
