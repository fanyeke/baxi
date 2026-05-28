package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiagnosisService_DiagnoseByRequestID_AllSources(t *testing.T) {
	tmpDir := t.TempDir()

	errorLog := filepath.Join(tmpDir, "error.log")
	auditCSV := filepath.Join(tmpDir, "audit_dispatch.csv")
	auditFeishu := filepath.Join(tmpDir, "audit_feishu.csv")

	require.NoError(t, os.WriteFile(errorLog, []byte(`
{"request_id":"req-123","timestamp":"2024-01-01T00:00:00Z","error_code":"E001","message":"connection refused","diagnosis":"db down","suggested_action":"restart db"}
{"request_id":"req-456","timestamp":"2024-01-01T01:00:00Z","error_code":"E002","message":"timeout"}
{"request_id":"req-123","timestamp":"2024-01-01T00:01:00Z","error_code":"E003","message":"retry failed"}
`), 0644))

	require.NoError(t, os.WriteFile(auditCSV, []byte("timestamp,outbox_id,status,error,request_id\n2024-01-01T00:00:00Z,out-1,failed,conn error,req-123\n2024-01-01T00:00:00Z,out-2,success,,req-456\n"), 0644))

	require.NoError(t, os.WriteFile(auditFeishu, []byte("timestamp,action,status,request_id\n2024-01-01T00:00:00Z,send,ok,req-123\n"), 0644))

	svc := NewDiagnosisService(errorLog, auditCSV, auditFeishu)

	result, err := svc.DiagnoseByRequestID("req-123")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "req-123", result.RequestID)
	assert.Equal(t, "connection refused", result.Summary)
	assert.Equal(t, "E001", result.ErrorCode)
	assert.Equal(t, "db down", result.Diagnosis)
	assert.Equal(t, "restart db", result.SuggestedAction)
	require.Len(t, result.RelatedLogs, 4)

	assert.Equal(t, "error.log", result.RelatedLogs[0].Source)
	assert.Equal(t, "2024-01-01T00:00:00Z", result.RelatedLogs[0].Ts)
	assert.Equal(t, "E001", result.RelatedLogs[0].ErrorCode)
	assert.Equal(t, "connection refused", result.RelatedLogs[0].Message)
	assert.Equal(t, "db down", result.RelatedLogs[0].Diagnosis)

	assert.Equal(t, "error.log", result.RelatedLogs[1].Source)
	assert.Equal(t, "2024-01-01T00:01:00Z", result.RelatedLogs[1].Ts)
	assert.Equal(t, "E003", result.RelatedLogs[1].ErrorCode)

	assert.Equal(t, "audit_dispatch.csv", result.RelatedLogs[2].Source)
	assert.Equal(t, "2024-01-01T00:00:00Z", result.RelatedLogs[2].Timestamp)
	assert.Equal(t, "out-1", result.RelatedLogs[2].OutboxID)
	assert.Equal(t, "failed", result.RelatedLogs[2].Status)
	assert.Equal(t, "conn error", result.RelatedLogs[2].Error)

	assert.Equal(t, "audit_feishu.csv", result.RelatedLogs[3].Source)
	assert.Equal(t, "2024-01-01T00:00:00Z", result.RelatedLogs[3].Timestamp)
	assert.Equal(t, "send", result.RelatedLogs[3].Action)
	assert.Equal(t, "ok", result.RelatedLogs[3].Status)
}

func TestDiagnosisService_DiagnoseByRequestID_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	errorLog := filepath.Join(tmpDir, "error.log")
	auditCSV := filepath.Join(tmpDir, "audit_dispatch.csv")
	auditFeishu := filepath.Join(tmpDir, "audit_feishu.csv")

	require.NoError(t, os.WriteFile(errorLog, []byte("{}\n"), 0644))
	require.NoError(t, os.WriteFile(auditCSV, []byte("a,b\n1,2\n"), 0644))
	require.NoError(t, os.WriteFile(auditFeishu, []byte("a,b\n1,2\n"), 0644))

	svc := NewDiagnosisService(errorLog, auditCSV, auditFeishu)

	result, err := svc.DiagnoseByRequestID("nonexistent")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestDiagnosisService_DiagnoseByRequestID_MissingFiles(t *testing.T) {
	tmpDir := t.TempDir()

	svc := NewDiagnosisService(
		filepath.Join(tmpDir, "missing.log"),
		filepath.Join(tmpDir, "missing.csv"),
		filepath.Join(tmpDir, "missing2.csv"),
	)

	result, err := svc.DiagnoseByRequestID("req-123")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestDiagnosisService_DiagnoseByRequestID_OnlyErrorLog(t *testing.T) {
	tmpDir := t.TempDir()

	errorLog := filepath.Join(tmpDir, "error.log")
	auditCSV := filepath.Join(tmpDir, "audit_dispatch.csv")
	auditFeishu := filepath.Join(tmpDir, "audit_feishu.csv")

	require.NoError(t, os.WriteFile(errorLog, []byte(`{"request_id":"req-123","timestamp":"2024-01-01T00:00:00Z","error_code":"E001","message":"connection refused"}`+"\n"), 0644))

	svc := NewDiagnosisService(errorLog, auditCSV, auditFeishu)

	result, err := svc.DiagnoseByRequestID("req-123")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "req-123", result.RequestID)
	assert.Equal(t, "connection refused", result.Summary)
	assert.Equal(t, "E001", result.ErrorCode)
	require.Len(t, result.RelatedLogs, 1)
	assert.Equal(t, "error.log", result.RelatedLogs[0].Source)
}

func TestDiagnosisService_DiagnoseByRequestID_OnlyAuditCSV(t *testing.T) {
	tmpDir := t.TempDir()

	errorLog := filepath.Join(tmpDir, "error.log")
	auditCSV := filepath.Join(tmpDir, "audit_dispatch.csv")
	auditFeishu := filepath.Join(tmpDir, "audit_feishu.csv")

	require.NoError(t, os.WriteFile(auditCSV, []byte("timestamp,outbox_id,status,error,request_id\n2024-01-01T00:00:00Z,out-1,failed,conn error,req-123\n"), 0644))

	svc := NewDiagnosisService(errorLog, auditCSV, auditFeishu)

	result, err := svc.DiagnoseByRequestID("req-123")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "req-123", result.RequestID)
	assert.Equal(t, "No error message recorded", result.Summary)
	assert.Empty(t, result.ErrorCode)
	require.Len(t, result.RelatedLogs, 1)
	assert.Equal(t, "audit_dispatch.csv", result.RelatedLogs[0].Source)
}

func TestDiagnosisService_DiagnoseByRequestID_OnlyFeishuCSV(t *testing.T) {
	tmpDir := t.TempDir()

	errorLog := filepath.Join(tmpDir, "error.log")
	auditCSV := filepath.Join(tmpDir, "audit_dispatch.csv")
	auditFeishu := filepath.Join(tmpDir, "audit_feishu.csv")

	require.NoError(t, os.WriteFile(auditFeishu, []byte("timestamp,action,status,request_id\n2024-01-01T00:00:00Z,send,ok,req-123\n"), 0644))

	svc := NewDiagnosisService(errorLog, auditCSV, auditFeishu)

	result, err := svc.DiagnoseByRequestID("req-123")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "req-123", result.RequestID)
	assert.Equal(t, "No error message recorded", result.Summary)
	require.Len(t, result.RelatedLogs, 1)
	assert.Equal(t, "audit_feishu.csv", result.RelatedLogs[0].Source)
}

func TestDiagnosisService_searchJSONL_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	errorLog := filepath.Join(tmpDir, "error.log")

	require.NoError(t, os.WriteFile(errorLog, []byte(`invalid json line
{"request_id":"req-123","message":"ok"}
`), 0644))

	svc := NewDiagnosisService(errorLog, "", "")
	result := svc.searchJSONL(errorLog, "req-123")

	require.Len(t, result, 1)
	assert.Equal(t, "ok", result[0]["message"])
}

func TestDiagnosisService_searchCSV_MissingHeader(t *testing.T) {
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "audit.csv")

	require.NoError(t, os.WriteFile(csvPath, []byte(""), 0644))

	svc := NewDiagnosisService("", csvPath, "")
	result := svc.searchCSV(csvPath, "req-123")
	assert.Empty(t, result)
}

func TestDiagnosisService_getString_FallbackKeys(t *testing.T) {
	m := map[string]interface{}{
		"ts": "2024-01-01T00:00:00Z",
	}
	assert.Equal(t, "2024-01-01T00:00:00Z", getString(m, "timestamp", "ts"))
	assert.Equal(t, "", getString(m, "nonexistent"))
}
