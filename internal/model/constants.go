package model

// Task priorities
const (
	PriorityLow      = "low"
	PriorityMedium   = "medium"
	PriorityHigh     = "high"
	PriorityCritical = "critical"
)

// Task statuses
const (
	StatusTodo       = "todo"
	StatusInProgress = "in_progress"
	StatusDone       = "done"
	StatusBlocked    = "blocked"
	StatusCancelled  = "cancelled"
)

// Alert statuses
const (
	AlertStatusNew           = "new"
	AlertStatusInvestigating = "investigating"
	AlertStatusResolved      = "resolved"
	AlertStatusIgnored       = "ignored"
)

// Alert severities
const (
	SeverityLow      = "low"
	SeverityMedium   = "medium"
	SeverityHigh     = "high"
	SeverityCritical = "critical"
)

// Default values
const (
	DefaultPriority = PriorityMedium
	DefaultStatus   = StatusTodo
)
