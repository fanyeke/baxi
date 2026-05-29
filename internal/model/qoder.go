package model

import "time"

// CapabilitiesResponse represents Qoder capabilities.
type CapabilitiesResponse struct {
	Mode              string
	Version           string
	CanReadStatus     bool
	CanReadAlerts     bool
	CanReadTasks      bool
	CanReadOutbox     bool
	CanReadGovernance bool
	CanReadLogs       bool
	CanWriteReports   bool
	CanExecuteActions bool
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

// ContextResponse represents Qoder decision context.
type ContextResponse struct {
	RequestID        string
	System           SystemInfo
	Summary          ContextSummary
	TopAlerts        []AlertItem
	OpenTasks        []TaskItem
	PendingOutbox    []OutboxItem
	RecentDiagnosis  []string
	AllowedActions   []string
	ForbiddenActions []string
	Ontology         OntologyInfo
	Governance       GovernanceInfo
	AgentPolicy      AgentPolicyInfo
}

// SystemInfo holds system-level information in the context response.
type SystemInfo struct {
	LastPipelineRun *PipelineRunInfo
}

// PipelineRunInfo represents the last pipeline run in the context response.
type PipelineRunInfo struct {
	RunID        string
	RunType      string
	Mode         string
	Status       string
	StartedAt    string
	FinishedAt   *string
	InputCount   int64
	OutputCount  int64
	ErrorMessage *string
}

// PipelineRunSummary represents a pipeline run for status responses.
type PipelineRunSummary struct {
	RunID        string
	RunType      string
	Mode         string
	Status       string
	StartedAt    string
	FinishedAt   *string
	InputCount   int64
	OutputCount  int64
	ErrorMessage *string
}

// ContextSummary holds aggregated counts in the context response.
type ContextSummary struct {
	TotalAlerts        int
	TotalOpenTasks     int
	TotalPendingOutbox int
}

// AlertItem is a detailed alert for context responses.
type AlertItem struct {
	EventID       string
	RuleID        string
	EventDate     string
	Severity      string
	MetricName    string
	ObjectType    string
	ObjectID      string
	CurrentValue  *float64
	BaselineValue *float64
	ChangeRate    *float64
	OwnerRole     string
	Status        string
	ImpactScore   *float64
}

// TaskItem is a detailed task for context responses.
type TaskItem struct {
	TaskID           string
	TaskTitle        string
	TaskDescription  string
	Status           string
	Priority         string
	OwnerRole        string
	OwnerUserID      *string
	DueAt            *time.Time
	CreatedAt        time.Time
	CompletedAt      *time.Time
	Feedback         *string
	RecommendationID *string
	EventID          *string
	TargetObjectType *string
	TargetObjectID   *string
}

// OutboxItem is a detailed outbox event for context responses.
type OutboxItem struct {
	OutboxID         string
	EventType        string
	SourceType       string
	SourceID         string
	TargetChannel    string
	Status           string
	CreatedAt        time.Time
	DispatchAttempts int
	LastDispatchAt   *time.Time
}

// AlertSummary is a summary of an alert for context responses.
type AlertSummary struct {
	AlertID     string
	Severity    string
	Status      string
	ImpactScore float64
}

// TaskSummary is a summary of a task for context responses.
type TaskSummary struct {
	TaskID    string
	Title     string
	Status    string
	OwnerRole string
}

// OutboxSummary is a summary of an outbox event for context responses.
type OutboxSummary struct {
	EventID   string
	EventType string
	Status    string
}

// ContextQueryParams holds parsed query parameters.
type ContextQueryParams struct {
	Severity    string
	LimitAlerts int
	LimitTasks  int
	LimitOutbox int
	IncludeLogs bool
}

// OntologyInfo describes known object types in the system.
type OntologyInfo struct {
	ObjectTypes      []string
	ObjectsAvailable bool
}

// GovernanceInfo describes the governance configuration load state.
type GovernanceInfo struct {
	ClassificationLoaded bool
	LineageLoaded        bool
	AccessPolicyLoaded   bool
	RedactionEnabled     bool
}

// AgentPolicyInfo describes the agent's permitted actions.
type AgentPolicyInfo struct {
	Role              string
	CanReadObjects    bool
	CanExecuteActions bool
	CanWriteReports   bool
}
