package model

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
	AllowedEndpoints  []string
	ForbiddenActions  []string
}

// ContextResponse represents Qoder decision context.
type ContextResponse struct {
	RequestID        string
	System           SystemInfo
	Summary          ContextSummary
	TopAlerts        []AlertSummary
	OpenTasks        []TaskSummary
	PendingOutbox    []OutboxSummary
	RecentDiagnosis  []string
	AllowedActions   []string
	ForbiddenActions []string
}

// SystemInfo holds system-level information in the context response.
type SystemInfo struct {
	LastPipelineRun *PipelineRunSummary
}

// PipelineRunSummary represents the last pipeline run in the context response.
type PipelineRunSummary struct {
	RunID        string
	RunType      string
	Mode         string
	Status       string
	StartedAt    string
	FinishedAt   string
	InputCount   int
	OutputCount  int
	ErrorMessage *string
}

// ContextSummary holds aggregated counts in the context response.
type ContextSummary struct {
	TotalAlerts        int
	TotalOpenTasks     int
	TotalPendingOutbox int
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
