// Package dto provides data transfer objects for API responses.
package dto

// StatusResponse is the system status response aggregating table counts,
// pipeline run info, and database connectivity.
type StatusResponse struct {
	Database        DatabaseInfo    `json:"database"`
	LastPipelineRun *PipelineRun   `json:"last_pipeline_run"`
	Version         string         `json:"version"`
}

// DatabaseInfo holds database connection metadata and table counts.
type DatabaseInfo struct {
	Path   string         `json:"path"`
	Exists bool           `json:"exists"`
	Tables map[string]int `json:"tables"`
}

// PipelineRun represents a single pipeline execution record.
type PipelineRun struct {
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
