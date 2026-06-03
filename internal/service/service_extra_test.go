package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogService_ReadAuditLogs_ZeroLimit_Extra(t *testing.T) {
	svc := NewLogService(nil)
	entries, err := svc.ReadAuditLogs("/nonexistent", nil, nil, 0)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestLogService_ReadLogErrors_ZeroLimit_Extra(t *testing.T) {
	svc := NewLogService(nil)
	entries, err := svc.ReadLogErrors("/nonexistent", nil, 0)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestLogService_ReadLogRecent_ZeroLimit_Extra(t *testing.T) {
	svc := NewLogService(nil)
	entries, err := svc.ReadLogRecent("/nonexistent", 0)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestLogService_ReadAuditLogs_EmptyRequestID_Extra(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.csv")
	content := "timestamp,outbox_id,status\n2024-01-01T00:00:00Z,ob-1,sent\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	svc := NewLogService(nil)
	emptyID := ""
	entries, err := svc.ReadAuditLogs(path, &emptyID, nil, 10)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestLogService_ReadLogErrors_EmptyRequestID_Extra(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "error.log")
	content := `{"request_id":"req-1","msg":"error1"}` + "\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	svc := NewLogService(nil)
	emptyID := ""
	entries, err := svc.ReadLogErrors(path, &emptyID, 10)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestTailJSONL_SingleLineNoNewline_Extra(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "single.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(`{"msg":"only"}`), 0644))

	entries, err := tailJSONL(path, 10)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "only", entries[0]["msg"])
}

func TestTailJSONL_LargeFile_Extra(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large.jsonl")
	var content string
	for i := 0; i < 1000; i++ {
		content += `{"msg":"line"}` + "\n"
	}
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	entries, err := tailJSONL(path, 5)
	require.NoError(t, err)
	assert.Len(t, entries, 5)
}

func TestTailJSONL_NegativeLimit_Extra(t *testing.T) {
	entries, err := tailJSONL("/nonexistent", -1)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestReadAuditLogs_HeaderOnly_Extra(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "header_only.csv")
	require.NoError(t, os.WriteFile(path, []byte("timestamp,outbox_id,status\n"), 0644))

	svc := NewLogService(nil)
	entries, err := svc.ReadAuditLogs(path, nil, nil, 10)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestLogService_NewLogService_Extra(t *testing.T) {
	svc := NewLogService(nil)
	assert.NotNil(t, svc)
}

func TestPipelineService_PreviewPipelineRun_AllTypes_Extra(t *testing.T) {
	svc := NewPipelineService("")
	for _, pType := range []string{"daily", "full", "db_full"} {
		t.Run(pType, func(t *testing.T) {
			result := svc.PreviewPipelineRun(pType)
			assert.NotNil(t, result)
			assert.NotEmpty(t, result.Command)
			assert.NotEmpty(t, result.EstimatedDuration)
			assert.Equal(t, pType, result.PipelineType)
		})
	}
}

func TestPipelineService_GetAvailablePipelines_Sorted_Extra(t *testing.T) {
	svc := NewPipelineService("")
	pipelines := svc.GetAvailablePipelines()
	require.Len(t, pipelines, 3)

	for i := 1; i < len(pipelines); i++ {
		assert.LessOrEqual(t, pipelines[i-1].Type, pipelines[i].Type)
	}
}

func TestPipelineService_PreviewPipelineRun_CustomConfigDir_Extra(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "alert_rules.yml"), []byte("rules: []"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "metrics.yml"), []byte("metrics: []"), 0644))

	t.Setenv("FEISHU_APP_ID", "test")
	t.Setenv("FEISHU_APP_SECRET", "test")
	t.Setenv("FEISHU_BASE_APP_TOKEN", "test")
	t.Setenv("FEISHU_CHAT_ID", "test")
	t.Setenv("LLM_API_KEY", "test")

	svc := NewPipelineService(tmpDir)
	result := svc.PreviewPipelineRun("daily")
	assert.Empty(t, result.Warnings)
}

func TestAllowedSorts_AllKeys(t *testing.T) {
	expected := map[string]bool{
		"created_at_desc": true,
		"created_at_asc":  true,
		"severity_desc":   true,
	}
	for key, val := range expected {
		got, ok := allowedSorts[key]
		assert.True(t, ok, "expected sort key %s", key)
		assert.Equal(t, val, got)
	}
	assert.Len(t, allowedSorts, 3)
}
