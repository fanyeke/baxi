package action

// ActionContract is the LLM-visible representation of an action type.
// It is a subset of ActionConfig, containing only what the LLM needs to know
// to generate valid action proposals.
type ActionContract struct {
	ActionType      string                 `json:"action_type"`
	Description     string                 `json:"description"`
	RequiredPayload []string               `json:"required_payload"`
	PayloadSchema   map[string]interface{} `json:"payload_schema"`
	RiskLevel       string                 `json:"risk_level"`
	RequiresReview  bool                   `json:"requires_human_review"`
	Adapter         string                 `json:"adapter"`
}
