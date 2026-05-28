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
