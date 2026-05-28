// Package dto provides data transfer objects for API responses.
package dto

// CapabilitiesResponse represents the Qoder capability matrix.
// This is a static read-only response — no DB queries involved.
type CapabilitiesResponse struct {
	Mode              string `json:"mode"`
	Version           string `json:"version"`
	CanReadStatus     bool   `json:"can_read_status"`
	CanReadAlerts     bool   `json:"can_read_alerts"`
	CanReadTasks      bool   `json:"can_read_tasks"`
	CanReadOutbox     bool   `json:"can_read_outbox"`
	CanReadGovernance bool   `json:"can_read_governance"`
	CanReadLogs       bool   `json:"can_read_logs"`
	CanWriteReports   bool   `json:"can_write_reports"`
	CanExecuteActions bool   `json:"can_execute_actions"`
}

// StaticCapabilities returns the default read-only capability matrix.
func StaticCapabilities() CapabilitiesResponse {
	return CapabilitiesResponse{
		Mode:              "read_only",
		Version:           "0.6.0",
		CanReadStatus:     true,
		CanReadAlerts:     true,
		CanReadTasks:      true,
		CanReadOutbox:     true,
		CanReadGovernance: true,
		CanReadLogs:       true,
		CanWriteReports:   false,
		CanExecuteActions: false,
	}
}

// AllowedActions derives the list of allowed action names from capabilities.
func (c CapabilitiesResponse) AllowedActions() []string {
	var actions []string
	if c.CanReadStatus {
		actions = append(actions, "read_status")
	}
	if c.CanReadAlerts {
		actions = append(actions, "read_alerts")
	}
	if c.CanReadTasks {
		actions = append(actions, "read_tasks")
	}
	if c.CanReadOutbox {
		actions = append(actions, "read_outbox")
	}
	if c.CanReadGovernance {
		actions = append(actions, "read_governance")
	}
	if c.CanReadLogs {
		actions = append(actions, "read_logs")
	}
	return actions
}

// ForbiddenActions derives the list of forbidden action names from capabilities.
func (c CapabilitiesResponse) ForbiddenActions() []string {
	var actions []string
	if !c.CanWriteReports {
		actions = append(actions, "write_reports")
	}
	if !c.CanExecuteActions {
		actions = append(actions, "execute_actions")
	}
	return actions
}

// PipelineRunInfo represents the last pipeline run in the context response.
type PipelineRunInfo struct {
	RunID        string  `json:"run_id"`
	RunType      string  `json:"run_type"`
	Mode         string  `json:"mode"`
	Status       string  `json:"status"`
	StartedAt    string  `json:"started_at"`
	FinishedAt   *string `json:"finished_at"`
	InputCount   int64   `json:"input_count"`
	OutputCount  int64   `json:"output_count"`
	ErrorMessage *string `json:"error_message"`
}

// SystemInfo holds system-level information in the context response.
type SystemInfo struct {
	LastPipelineRun *PipelineRunInfo `json:"last_pipeline_run,omitempty"`
}

// ContextSummary holds aggregated counts in the context response.
type ContextSummary struct {
	TotalAlerts       int `json:"total_alerts"`
	TotalOpenTasks    int `json:"total_open_tasks"`
	TotalPendingOutbox int `json:"total_pending_outbox"`
}

// ContextParams holds query parameters for the context endpoint.
type ContextParams struct {
	Severity    string
	LimitAlerts int
	LimitTasks  int
	LimitOutbox int
	IncludeLogs bool
}

// OntologyInfo describes known object types in the system.
type OntologyInfo struct {
	ObjectTypes      []string `json:"object_types"`
	ObjectsAvailable bool     `json:"objects_available"`
}

// GovernanceInfo describes the governance configuration load state.
type GovernanceInfo struct {
	ClassificationLoaded bool `json:"classification_loaded"`
	LineageLoaded        bool `json:"lineage_loaded"`
	AccessPolicyLoaded   bool `json:"access_policy_loaded"`
	RedactionEnabled     bool `json:"redaction_enabled"`
}

// AgentPolicyInfo describes the agent's permitted actions.
type AgentPolicyInfo struct {
	Role              string `json:"role"`
	CanReadObjects    bool   `json:"can_read_objects"`
	CanExecuteActions bool   `json:"can_execute_actions"`
	CanWriteReports   bool   `json:"can_write_reports"`
}

// ContextResponse is the composite aggregation response for GET /api/v1/qoder/context.
type ContextResponse struct {
	RequestID       string          `json:"request_id"`
	System          SystemInfo      `json:"system"`
	Summary         ContextSummary  `json:"summary"`
	TopAlerts       []AlertItem     `json:"top_alerts"`
	OpenTasks       []TaskItem      `json:"open_tasks"`
	PendingOutbox   []OutboxItem    `json:"pending_outbox"`
	RecentDiagnosis []interface{}   `json:"recent_diagnosis"`
	AllowedActions  []string        `json:"allowed_actions"`
	ForbiddenActions []string       `json:"forbidden_actions"`
	// Enrichment fields for agent context
	Ontology    OntologyInfo    `json:"ontology"`
	Governance  GovernanceInfo  `json:"governance"`
	AgentPolicy AgentPolicyInfo `json:"agent_policy"`
}

// ContextQueryParams holds parsed query parameters.
type ContextQueryParams struct {
	Severity    string
	LimitAlerts int
	LimitTasks  int
	LimitOutbox int
	IncludeLogs bool
}

