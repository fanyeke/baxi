package decision

import (
	"context"
	"encoding/json"
	"time"
)

// LineageEventType represents the type of decision lineage event.
type LineageEventType string

const (
	LineageEventCaseCreated       LineageEventType = "case_created"
	LineageEventContextBuilt      LineageEventType = "context_built"
	LineageEventDecisionRequested LineageEventType = "decision_requested"
	LineageEventDecisionGenerated LineageEventType = "decision_generated"
	LineageEventProposalCreated   LineageEventType = "proposal_created"
	LineageEventProposalApproved  LineageEventType = "proposal_approved"
	LineageEventProposalRejected  LineageEventType = "proposal_rejected"
	LineageEventActionApplying    LineageEventType = "action_applying"
	LineageEventActionApplied     LineageEventType = "action_applied"
	LineageEventActionFailed      LineageEventType = "action_failed"
	LineageEventCaseClosed        LineageEventType = "case_closed"
	LineageEventCaseFailed        LineageEventType = "case_failed"
	LineageEventFallbackUsed      LineageEventType = "fallback_used"
	LineageEventValidationFailed  LineageEventType = "validation_failed"
	LineageEventRepairAttempted   LineageEventType = "repair_attempted"
	LineageEventRepairSucceeded   LineageEventType = "repair_succeeded"
	LineageEventRepairFailed      LineageEventType = "repair_failed"
	LineageEventDispatchSucceeded LineageEventType = "dispatch_succeeded"
	LineageEventDispatchFailed    LineageEventType = "dispatch_failed"
)

// SnapshotType represents the type of data snapshot.
type SnapshotType string

const (
	SnapshotTypeAlertContext    SnapshotType = "alert_context"
	SnapshotTypeObjectContext   SnapshotType = "object_context"
	SnapshotTypeGovernance      SnapshotType = "governance"
	SnapshotTypeDecisionInput   SnapshotType = "decision_input"
	SnapshotTypeDecisionOutput  SnapshotType = "decision_output"
	SnapshotTypeProposalPayload SnapshotType = "proposal_payload"
	// Phase 2: LLM audit snapshots
	SnapshotTypeLLMSafeContext   SnapshotType = "llm_safe_context"
	SnapshotTypeLLMRawOutput     SnapshotType = "llm_raw_output"
	SnapshotTypeLLMParsedOutput  SnapshotType = "llm_parsed_output"
	SnapshotTypeLLMValidation    SnapshotType = "llm_validation_result"
	SnapshotTypeLLMRepairAttempt SnapshotType = "llm_repair_attempt"
)

// DecisionLineageEvent represents a single event in the decision lineage chain.
type DecisionLineageEvent struct {
	EventID        string           `json:"event_id"`
	CaseID         string           `json:"case_id"`
	EventType      LineageEventType `json:"event_type"`
	EventTimestamp time.Time        `json:"event_timestamp"`
	Actor          string           `json:"actor,omitempty"`
	EventData      json.RawMessage  `json:"event_data,omitempty"`
	ContextHash    string           `json:"context_hash,omitempty"`
	ConfigHash     string           `json:"config_hash,omitempty"`
}

// DecisionDataSnapshot represents a point-in-time data capture for a decision case.
type DecisionDataSnapshot struct {
	SnapshotID   string          `json:"snapshot_id"`
	CaseID       string          `json:"case_id"`
	SnapshotType SnapshotType    `json:"snapshot_type"`
	SnapshotJSON json.RawMessage `json:"snapshot_json,omitempty"`
	SourceTable  string          `json:"source_table,omitempty"`
	RowCount     int             `json:"row_count"`
	CapturedAt   time.Time       `json:"captured_at"`
}

// LineageEventRecord holds the parameters for recording a lineage event.
type LineageEventRecord struct {
	CaseID      string           `json:"case_id"`
	EventType   LineageEventType `json:"event_type"`
	Actor       string           `json:"actor,omitempty"`
	EventData   json.RawMessage  `json:"event_data,omitempty"`
	ContextHash string           `json:"context_hash,omitempty"`
	ConfigHash  string           `json:"config_hash,omitempty"`
}

// DataSnapshotRecord holds the parameters for recording a data snapshot.
type DataSnapshotRecord struct {
	CaseID       string          `json:"case_id"`
	SnapshotType SnapshotType    `json:"snapshot_type"`
	SnapshotJSON json.RawMessage `json:"snapshot_json,omitempty"`
	SourceTable  string          `json:"source_table,omitempty"`
	RowCount     int             `json:"row_count"`
}

// DecisionLineageChain holds the full lineage chain for a decision case.
type DecisionLineageChain struct {
	CaseID    string                 `json:"case_id"`
	Events    []DecisionLineageEvent `json:"events"`
	Snapshots []DecisionDataSnapshot `json:"snapshots"`
}

// ContextLineage holds the lineage information needed for context building.
type ContextLineage struct {
	CaseID         string                 `json:"case_id"`
	UpstreamTables []string               `json:"upstream_tables"`
	ConfigVersions map[string]string      `json:"config_versions"`
	Snapshots      []DecisionDataSnapshot `json:"snapshots"`
}

// DecisionLineageService provides decision-specific lineage tracking.
type DecisionLineageService interface {
	// GetDecisionLineage returns the full lineage chain for a decision case,
	// including all events and data snapshots in chronological order.
	GetDecisionLineage(ctx context.Context, caseID string) (*DecisionLineageChain, error)

	// GetContextLineage returns lineage information needed for context building,
	// including upstream tables and relevant data snapshots.
	GetContextLineage(ctx context.Context, caseID string) (*ContextLineage, error)

	// RecordDecisionLineage records a lineage event for a decision case.
	RecordDecisionLineage(ctx context.Context, record LineageEventRecord) error

	// RecordDataSnapshot records a point-in-time data snapshot for a decision case.
	RecordDataSnapshot(ctx context.Context, record DataSnapshotRecord) error
}
