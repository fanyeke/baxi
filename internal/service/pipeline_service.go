package service

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"baxi/internal/model"
)

// pipelineDefinition holds metadata for a pipeline type.
type pipelineDefinition struct {
	Script      string
	Args        string
	Duration    string
	Description string
}

var pipelineRegistry = map[string]pipelineDefinition{
	"daily": {
		Script:      "run_daily_pipeline.py",
		Args:        "",
		Duration:    "~30 seconds (daily mode, single-day simulation)",
		Description: "8-step daily pipeline: ingest → quality → metrics → alerts → AIP → wake → Feishu",
	},
	"full": {
		Script:      "run_full_pipeline.py",
		Args:        "",
		Duration:    "~5 minutes (full mode, all 634 days)",
		Description: "5-step full pipeline: metrics → alerts → AIP → AI decision → Feishu",
	},
	"db_full": {
		Script:      "run_db_pipeline.py",
		Args:        "--mode full --dimensional",
		Duration:    "~2 minutes (DB mode, all data via SQLite)",
		Description: "5-step DB pipeline: init → ingest → metrics → rules → export",
	},
}

var requiredEnvVars = []string{
	"API_BEARER_TOKEN",
	"FEISHU_APP_ID",
	"FEISHU_APP_SECRET",
	"FEISHU_BASE_APP_TOKEN",
	"FEISHU_CHAT_ID",
}

// PipelineService provides pipeline preview functionality.
type PipelineService struct {
	configDir string
}

// NewPipelineService creates a new PipelineService.
// configDir is the directory containing YAML config files (e.g. "config").
func NewPipelineService(configDir string) *PipelineService {
	if configDir == "" {
		configDir = "config"
	}
	return &PipelineService{configDir: configDir}
}

// PreviewPipelineRun returns a preview of a pipeline run without executing.
func (s *PipelineService) PreviewPipelineRun(pipelineType string) *model.PipelinePreview {
	pipe, ok := pipelineRegistry[pipelineType]
	if !ok {
		validTypes := s.validPipelineTypes()
		return &model.PipelinePreview{
			Command:      "",
			PipelineType: pipelineType,
			Warnings:     []string{fmt.Sprintf("Unknown pipeline type: '%s'. Valid: %s", pipelineType, validTypes)},
		}
	}

	scriptPath := filepath.Join("scripts", pipe.Script)
	command := fmt.Sprintf("python3 %s", scriptPath)
	if pipe.Args != "" {
		command = fmt.Sprintf("%s %s", command, pipe.Args)
	}

	return &model.PipelinePreview{
		Command:           command,
		PipelineType:      pipelineType,
		EstimatedDuration: pipe.Duration,
		RequiredEnvVars:   requiredEnvVars,
		Warnings:          s.checkEnvWarnings(),
		Description:       pipe.Description,
	}
}

// GetAvailablePipelines returns a list of available pipeline types with descriptions.
func (s *PipelineService) GetAvailablePipelines() []model.PipelineInfo {
	types := make([]string, 0, len(pipelineRegistry))
	for t := range pipelineRegistry {
		types = append(types, t)
	}
	sort.Strings(types)

	result := make([]model.PipelineInfo, 0, len(types))
	for _, t := range types {
		result = append(result, model.PipelineInfo{
			Type:        t,
			Description: pipelineRegistry[t].Description,
		})
	}
	return result
}

func (s *PipelineService) validPipelineTypes() string {
	types := make([]string, 0, len(pipelineRegistry))
	for t := range pipelineRegistry {
		types = append(types, t)
	}
	sort.Strings(types)
	var b strings.Builder
	for i, t := range types {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(t)
	}
	return b.String()
}

func (s *PipelineService) checkEnvWarnings() []string {
	var warnings []string

	for _, v := range []string{"FEISHU_APP_ID", "FEISHU_APP_SECRET", "FEISHU_BASE_APP_TOKEN", "FEISHU_CHAT_ID"} {
		if os.Getenv(v) == "" {
			warnings = append(warnings, fmt.Sprintf("Env var %s not set — Feishu operations will use dry-run defaults", v))
		}
	}

	if os.Getenv("LLM_API_KEY") == "" {
		warnings = append(warnings, "LLM_API_KEY not set — AI decision engine will use heuristic fallback")
	}

	alertRulesFile := filepath.Join(s.configDir, "alert_rules.yml")
	if _, err := os.Stat(alertRulesFile); os.IsNotExist(err) {
		warnings = append(warnings, fmt.Sprintf("Alert rules config missing: %s", alertRulesFile))
	}

	metricsFile := filepath.Join(s.configDir, "metrics.yml")
	if _, err := os.Stat(metricsFile); os.IsNotExist(err) {
		warnings = append(warnings, fmt.Sprintf("Metrics config missing: %s", metricsFile))
	}

	return warnings
}
