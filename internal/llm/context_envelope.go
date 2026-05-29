package llm

import "time"

// LLMSafeContextEnvelope is the versioned, auditable wrapper around an LLMSafeContext.
// It captures everything needed to replay or audit an LLM decision request:
// the redacted context, evidence items, governance metadata, redaction summary,
// prompt version, and config versions.
type LLMSafeContextEnvelope struct {
	SchemaVersion    string            `json:"schema_version"`
	CaseID           string            `json:"case_id"`
	AlertID          string            `json:"alert_id"`
	ContextHash      string            `json:"context_hash"`
	BuiltAt          time.Time         `json:"built_at"`
	Trigger          TriggerInfo       `json:"trigger"`
	ObjectContext    ObjectContext     `json:"object_context"`
	Evidence         []EvidenceItem    `json:"evidence"`
	AllowedActions   []string          `json:"allowed_actions"`
	ForbiddenActions []string          `json:"forbidden_actions"`
	Governance       GovernanceInfo    `json:"governance"`
	RedactionSummary RedactionSummary  `json:"redaction_summary"`
	PromptVersion    string            `json:"prompt_version"`
	ConfigVersions   map[string]string `json:"config_versions"`
}

// EvidenceItem is a single piece of evidence included in the LLM context.
type EvidenceItem struct {
	Type  string      `json:"type"`
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

// RedactionSummary summarizes what was redacted from the object context
// before it was sent to the LLM.
type RedactionSummary struct {
	TotalFields   int      `json:"total_fields"`
	RedactedCount int      `json:"redacted_count"`
	RedactedList  []string `json:"redacted_list"`
	AppliedRole   string   `json:"applied_role"`
}
