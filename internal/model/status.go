package model

// StatusResponse is the system status response aggregating table counts,
// pipeline run info, and database connectivity.
type StatusResponse struct {
	Database        DatabaseInfo
	LastPipelineRun *PipelineRun
	Version         string
}

// DatabaseInfo holds database connection metadata and table counts.
type DatabaseInfo struct {
	Path   string
	Exists bool
	Tables map[string]int
}

// PipelineRun represents a single pipeline execution record.
type PipelineRun struct {
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

// SystemStatus represents the overall system health and operational status.
type SystemStatus struct {
	PipelineRun  *PipelineRun
	AlertCount   int
	TableCounts  []TableCount
	RecentErrors []string
}

// TableCount maps a table name to its row count.
type TableCount struct {
	TableName string
	RowCount  int
}

// SearchResult holds the result of a paginated object search.
type SearchResult struct {
	Items []map[string]interface{}
	Total int
}
