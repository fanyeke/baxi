package dto

// PipelineRunRequest is the request body for POST /api/v1/pipeline/run.
type PipelineRunRequest struct {
	// Config is the pipeline configuration name (e.g. "ingest_raw", "full").
	Config string `json:"config"`
}

// PipelineRunResponse is the response body for POST /api/v1/pipeline/run.
type PipelineRunResponse struct {
	RunID  string `json:"run_id"`
	Status string `json:"status"`
}

// PipelinePreview is the preview response for a pipeline run.
type PipelinePreview struct {
	Command           string   `json:"command"`
	PipelineType      string   `json:"pipeline_type"`
	EstimatedDuration string   `json:"estimated_duration"`
	RequiredEnvVars   []string `json:"required_env_vars"`
	Warnings          []string `json:"warnings"`
	Description       string   `json:"description"`
}

// PipelineInfo describes an available pipeline type.
type PipelineInfo struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}
