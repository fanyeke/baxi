package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"baxi/internal/api/dto"
)

func TestPipelineService_PreviewPipelineRun_Daily(t *testing.T) {
	svc := NewPipelineService("")
	result := svc.PreviewPipelineRun("daily")

	assert.NotNil(t, result)
	assert.Equal(t, "daily", result.PipelineType)
	assert.Equal(t, "python3 scripts/run_daily_pipeline.py", result.Command)
	assert.Equal(t, "~30 seconds (daily mode, single-day simulation)", result.EstimatedDuration)
	assert.Equal(t, "8-step daily pipeline: ingest → quality → metrics → alerts → AIP → wake → Feishu", result.Description)
	assert.Equal(t, requiredEnvVars, result.RequiredEnvVars)
}

func TestPipelineService_PreviewPipelineRun_Full(t *testing.T) {
	svc := NewPipelineService("")
	result := svc.PreviewPipelineRun("full")

	assert.NotNil(t, result)
	assert.Equal(t, "full", result.PipelineType)
	assert.Equal(t, "python3 scripts/run_full_pipeline.py", result.Command)
	assert.Equal(t, "~5 minutes (full mode, all 634 days)", result.EstimatedDuration)
	assert.Equal(t, "5-step full pipeline: metrics → alerts → AIP → AI decision → Feishu", result.Description)
}

func TestPipelineService_PreviewPipelineRun_DBFull(t *testing.T) {
	svc := NewPipelineService("")
	result := svc.PreviewPipelineRun("db_full")

	assert.NotNil(t, result)
	assert.Equal(t, "db_full", result.PipelineType)
	assert.Equal(t, "python3 scripts/run_db_pipeline.py --mode full --dimensional", result.Command)
	assert.Equal(t, "~2 minutes (DB mode, all data via SQLite)", result.EstimatedDuration)
	assert.Equal(t, "5-step DB pipeline: init → ingest → metrics → rules → export", result.Description)
}

func TestPipelineService_PreviewPipelineRun_Unknown(t *testing.T) {
	svc := NewPipelineService("")
	result := svc.PreviewPipelineRun("nonexistent")

	assert.NotNil(t, result)
	assert.Equal(t, "nonexistent", result.PipelineType)
	assert.Empty(t, result.Command)
	assert.Empty(t, result.EstimatedDuration)
	assert.Len(t, result.Warnings, 1)
	assert.Contains(t, result.Warnings[0], "Unknown pipeline type")
	assert.Contains(t, result.Warnings[0], "daily")
	assert.Contains(t, result.Warnings[0], "full")
	assert.Contains(t, result.Warnings[0], "db_full")
}

func TestPipelineService_GetAvailablePipelines(t *testing.T) {
	svc := NewPipelineService("")
	pipelines := svc.GetAvailablePipelines()

	assert.Len(t, pipelines, 3)

	types := make([]string, len(pipelines))
	for i, p := range pipelines {
		types[i] = p.Type
	}
	assert.Equal(t, []string{"daily", "db_full", "full"}, types)

	descMap := make(map[string]string)
	for _, p := range pipelines {
		descMap[p.Type] = p.Description
	}
	assert.Equal(t, pipelineRegistry["daily"].Description, descMap["daily"])
	assert.Equal(t, pipelineRegistry["full"].Description, descMap["full"])
	assert.Equal(t, pipelineRegistry["db_full"].Description, descMap["db_full"])
}

func TestPipelineService_PreviewPipelineRun_WithEnvWarnings(t *testing.T) {
	t.Setenv("FEISHU_APP_ID", "")
	t.Setenv("FEISHU_APP_SECRET", "")
	t.Setenv("FEISHU_BASE_APP_TOKEN", "")
	t.Setenv("FEISHU_CHAT_ID", "")
	t.Setenv("LLM_API_KEY", "")

	svc := NewPipelineService("")
	result := svc.PreviewPipelineRun("daily")

	assert.NotNil(t, result)
	assert.Len(t, result.Warnings, 7)

	warningMap := make(map[string]bool)
	for _, w := range result.Warnings {
		warningMap[w] = true
	}

	assert.True(t, warningMap["Env var FEISHU_APP_ID not set — Feishu operations will use dry-run defaults"])
	assert.True(t, warningMap["Env var FEISHU_APP_SECRET not set — Feishu operations will use dry-run defaults"])
	assert.True(t, warningMap["Env var FEISHU_BASE_APP_TOKEN not set — Feishu operations will use dry-run defaults"])
	assert.True(t, warningMap["Env var FEISHU_CHAT_ID not set — Feishu operations will use dry-run defaults"])
	assert.True(t, warningMap["LLM_API_KEY not set — AI decision engine will use heuristic fallback"])
	assert.True(t, warningMap["Alert rules config missing: config/alert_rules.yml"])
	assert.True(t, warningMap["Metrics config missing: config/metrics.yml"])
}

func TestPipelineService_PreviewPipelineRun_NoWarnings(t *testing.T) {
	t.Setenv("FEISHU_APP_ID", "test-app-id")
	t.Setenv("FEISHU_APP_SECRET", "test-secret")
	t.Setenv("FEISHU_BASE_APP_TOKEN", "test-token")
	t.Setenv("FEISHU_CHAT_ID", "test-chat-id")
	t.Setenv("LLM_API_KEY", "test-llm-key")

	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "alert_rules.yml"), []byte("rules: []"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "metrics.yml"), []byte("metrics: []"), 0644)

	svc := NewPipelineService(tmpDir)
	result := svc.PreviewPipelineRun("daily")

	assert.NotNil(t, result)
	assert.Empty(t, result.Warnings)
}

func TestPipelineService_PreviewPipelineRun_PartialWarnings(t *testing.T) {
	t.Setenv("FEISHU_APP_ID", "test-app-id")
	t.Setenv("FEISHU_APP_SECRET", "")
	t.Setenv("FEISHU_BASE_APP_TOKEN", "")
	t.Setenv("FEISHU_CHAT_ID", "test-chat-id")
	t.Setenv("LLM_API_KEY", "test-llm-key")

	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "alert_rules.yml"), []byte("rules: []"), 0644)

	svc := NewPipelineService(tmpDir)
	result := svc.PreviewPipelineRun("daily")

	assert.NotNil(t, result)
	assert.Len(t, result.Warnings, 3)

	warningMap := make(map[string]bool)
	for _, w := range result.Warnings {
		warningMap[w] = true
	}

	assert.True(t, warningMap["Env var FEISHU_APP_SECRET not set — Feishu operations will use dry-run defaults"])
	assert.True(t, warningMap["Env var FEISHU_BASE_APP_TOKEN not set — Feishu operations will use dry-run defaults"])
	assert.True(t, warningMap["Metrics config missing: "+filepath.Join(tmpDir, "metrics.yml")])
}

func TestPipelineService_NewPipelineService_DefaultConfigDir(t *testing.T) {
	svc := NewPipelineService("")
	assert.Equal(t, "config", svc.configDir)
}

func TestPipelineService_NewPipelineService_CustomConfigDir(t *testing.T) {
	svc := NewPipelineService("custom/config")
	assert.Equal(t, "custom/config", svc.configDir)
}

func TestPipelineService_GetAvailablePipelines_EmptyRegistry(t *testing.T) {
	svc := NewPipelineService("")
	pipelines := svc.GetAvailablePipelines()

	assert.NotNil(t, pipelines)
	assert.Len(t, pipelines, 3)
}

func TestPipelineService_PreviewPipelineRun_AllTypesHaveWarnings(t *testing.T) {
	t.Setenv("FEISHU_APP_ID", "")
	t.Setenv("LLM_API_KEY", "")

	for pType := range pipelineRegistry {
		t.Run(pType, func(t *testing.T) {
			svc := NewPipelineService("")
			result := svc.PreviewPipelineRun(pType)
			assert.NotNil(t, result)
			assert.Equal(t, pType, result.PipelineType)
			assert.NotEmpty(t, result.Command)
			assert.NotEmpty(t, result.EstimatedDuration)
			assert.NotEmpty(t, result.Description)
			assert.Equal(t, requiredEnvVars, result.RequiredEnvVars)
			assert.NotNil(t, result.Warnings)
		})
	}
}

func TestPipelineService_PreviewPipelineRun_ResultStructure(t *testing.T) {
	t.Setenv("FEISHU_APP_ID", "test")
	t.Setenv("FEISHU_APP_SECRET", "test")
	t.Setenv("FEISHU_BASE_APP_TOKEN", "test")
	t.Setenv("FEISHU_CHAT_ID", "test")
	t.Setenv("LLM_API_KEY", "test")

	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "alert_rules.yml"), []byte("rules: []"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "metrics.yml"), []byte("metrics: []"), 0644)

	svc := NewPipelineService(tmpDir)
	result := svc.PreviewPipelineRun("daily")

	assert.IsType(t, &dto.PipelinePreview{}, result)
	assert.Equal(t, "python3 scripts/run_daily_pipeline.py", result.Command)
	assert.Equal(t, "daily", result.PipelineType)
	assert.Equal(t, "~30 seconds (daily mode, single-day simulation)", result.EstimatedDuration)
	assert.Equal(t, requiredEnvVars, result.RequiredEnvVars)
	assert.Empty(t, result.Warnings)
	assert.Equal(t, "8-step daily pipeline: ingest → quality → metrics → alerts → AIP → wake → Feishu", result.Description)
}

func TestPipelineService_GetAvailablePipelines_ReturnsDTOs(t *testing.T) {
	svc := NewPipelineService("")
	pipelines := svc.GetAvailablePipelines()

	for _, p := range pipelines {
		assert.NotEmpty(t, p.Type)
		assert.NotEmpty(t, p.Description)
	}
}
