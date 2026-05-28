package model

// PipelinePreview shows what a pipeline run would do.
type PipelinePreview struct {
	Command           string
	PipelineType      string
	EstimatedDuration string
	RequiredEnvVars   []string
	Warnings          []string
	Description       string
}

// PipelineInfo describes an available pipeline type.
type PipelineInfo struct {
	Type        string
	Description string
}
