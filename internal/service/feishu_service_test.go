package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// mockBitableClient is a test double for FeishuBitableClient.
type mockBitableClient struct {
	listRecordsFunc   func(tableID string, pageSize int, filterConfig map[string]any) ([]map[string]any, error)
	upsertByKeyFunc   func(tableID string, records []map[string]any, keyField string) (created []map[string]any, updated []map[string]any, err error)
	sendMessageFunc   func(chatID, content string, dryRun bool) (string, error)
}

func (m *mockBitableClient) ListRecords(tableID string, pageSize int, filterConfig map[string]any) ([]map[string]any, error) {
	if m.listRecordsFunc != nil {
		return m.listRecordsFunc(tableID, pageSize, filterConfig)
	}
	return nil, nil
}

func (m *mockBitableClient) UpsertByKey(tableID string, records []map[string]any, keyField string) (created []map[string]any, updated []map[string]any, err error) {
	if m.upsertByKeyFunc != nil {
		return m.upsertByKeyFunc(tableID, records, keyField)
	}
	return nil, nil, nil
}

func (m *mockBitableClient) SendMessage(chatID, content string, dryRun bool) (string, error) {
	if m.sendMessageFunc != nil {
		return m.sendMessageFunc(chatID, content, dryRun)
	}
	return "", nil
}

// setupTestEnv creates a temporary project root with config and data directories.
func setupTestEnv(t *testing.T) (root string, cleanup func()) {
	t.Helper()
	root = t.TempDir()
	_ = os.MkdirAll(filepath.Join(root, "config"), 0755)
	_ = os.MkdirAll(filepath.Join(root, "data", "feishu"), 0755)
	_ = os.MkdirAll(filepath.Join(root, "data", "system"), 0755)
	return root, func() { os.RemoveAll(root) }
}

func writeYAML(t *testing.T, path string, data any) {
	t.Helper()
	b, err := yaml.Marshal(data)
	require.NoError(t, err)
	err = os.WriteFile(path, b, 0644)
	require.NoError(t, err)
}

func writeCSV(t *testing.T, path string, rows [][]string) {
	t.Helper()
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()
	for _, row := range rows {
		_, err := fmt.Fprintln(f, joinCSVRow(row))
		require.NoError(t, err)
	}
}

func joinCSVRow(row []string) string {
	return fmt.Sprintf("%s", row) // simplistic, only for test data without commas
}

// TestFeishuService_New tests constructor.
func TestFeishuService_New(t *testing.T) {
	svc := NewFeishuService(true)
	assert.True(t, svc.dryRun)
	assert.NotEmpty(t, svc.projectRoot)
}

// TestFeishuService_WithProjectRoot tests option.
func TestFeishuService_WithProjectRoot(t *testing.T) {
	root, cleanup := setupTestEnv(t)
	defer cleanup()

	svc := NewFeishuService(false, WithProjectRoot(root))
	assert.Equal(t, root, svc.projectRoot)
	assert.Equal(t, filepath.Join(root, "data", "feishu"), svc.feishuDir)
}

// TestFeishuService_WithFeishuClient tests client injection.
func TestFeishuService_WithFeishuClient(t *testing.T) {
	mock := &mockBitableClient{}
	svc := NewFeishuService(false, WithFeishuClient(mock))
	assert.Equal(t, mock, svc.client)
}

// TestFeishuService_loadConfig_FromEnv tests config loaded from env vars.
func TestFeishuService_loadConfig_FromEnv(t *testing.T) {
	root, cleanup := setupTestEnv(t)
	defer cleanup()

	t.Setenv("FEISHU_APP_ID", "env_app_id")
	t.Setenv("FEISHU_APP_SECRET", "env_secret")
	t.Setenv("FEISHU_BASE_APP_TOKEN", "env_token")
	t.Setenv("FEISHU_CHAT_ID", "env_chat")

	svc := NewFeishuService(false, WithProjectRoot(root))
	cfg := svc.loadConfig()

	assert.Equal(t, "env_app_id", cfg.appID)
	assert.Equal(t, "env_secret", cfg.appSecret)
	assert.Equal(t, "env_token", cfg.appToken)
	assert.Equal(t, "env_chat", cfg.chatID)
}

// TestFeishuService_loadConfig_FromYAML tests config fallback from YAML.
func TestFeishuService_loadConfig_FromYAML(t *testing.T) {
	root, cleanup := setupTestEnv(t)
	defer cleanup()

	writeYAML(t, filepath.Join(root, "config", "feishu_app.yml"), map[string]string{
		"app_id":     "yml_app_id",
		"app_secret": "yml_secret",
		"chat_id":    "yml_chat",
	})

	writeYAML(t, filepath.Join(root, "config", "feishu_table_ids.yml"), map[string]any{
		"tables": map[string]any{
			"daily_metrics": map[string]string{"table_id": "tbl1", "name": "Metrics"},
		},
	})

	svc := NewFeishuService(false, WithProjectRoot(root))
	cfg := svc.loadConfig()

	assert.Equal(t, "yml_app_id", cfg.appID)
	assert.Equal(t, "yml_secret", cfg.appSecret)
	assert.Equal(t, "tbl1", cfg.tableIDs["daily_metrics"])
}

// TestFeishuService_loadConfig_EnvOverridesYAML tests env vars override YAML.
func TestFeishuService_loadConfig_EnvOverridesYAML(t *testing.T) {
	root, cleanup := setupTestEnv(t)
	defer cleanup()

	t.Setenv("FEISHU_APP_ID", "env_app_id")

	writeYAML(t, filepath.Join(root, "config", "feishu_app.yml"), map[string]string{
		"app_id":     "yml_app_id",
		"app_secret": "yml_secret",
	})

	svc := NewFeishuService(false, WithProjectRoot(root))
	cfg := svc.loadConfig()

	assert.Equal(t, "env_app_id", cfg.appID)
	assert.Equal(t, "yml_secret", cfg.appSecret)
}

// TestFeishuService_isConfigured tests configuration detection.
func TestFeishuService_isConfigured(t *testing.T) {
	t.Run("not configured", func(t *testing.T) {
		root, cleanup := setupTestEnv(t)
		defer cleanup()
		svc := NewFeishuService(false, WithProjectRoot(root))
		assert.False(t, svc.isConfigured())
	})

	t.Run("configured", func(t *testing.T) {
		root, cleanup := setupTestEnv(t)
		defer cleanup()
		writeYAML(t, filepath.Join(root, "config", "feishu_app.yml"), map[string]string{
			"app_id":     "app_id",
			"app_secret": "secret",
		})
		svc := NewFeishuService(false, WithProjectRoot(root))
		assert.True(t, svc.isConfigured())
	})
}

// TestFeishuService_getTableNames tests table name resolution.
func TestFeishuService_getTableNames(t *testing.T) {
	root, cleanup := setupTestEnv(t)
	defer cleanup()

	writeYAML(t, filepath.Join(root, "config", "feishu_table_ids.yml"), map[string]any{
		"tables": map[string]any{
			"daily_metrics": map[string]string{"table_id": "tbl1"},
			"alert_events":  map[string]string{"table_id": "tbl2"},
		},
	})

	svc := NewFeishuService(false, WithProjectRoot(root))

	t.Run("default returns all", func(t *testing.T) {
		names, err := svc.getTableNames(nil)
		require.NoError(t, err)
		assert.Len(t, names, 2)
	})

	t.Run("specific valid", func(t *testing.T) {
		names, err := svc.getTableNames([]string{"daily_metrics"})
		require.NoError(t, err)
		assert.Equal(t, []string{"daily_metrics"}, names)
	})

	t.Run("unknown table", func(t *testing.T) {
		_, err := svc.getTableNames([]string{"unknown"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown table names")
	})
}

// TestFeishuService_getTableNames_DefaultFallback tests fallback when no YAML.
func TestFeishuService_getTableNames_DefaultFallback(t *testing.T) {
	root, cleanup := setupTestEnv(t)
	defer cleanup()
	svc := NewFeishuService(false, WithProjectRoot(root))

	names, err := svc.getTableNames(nil)
	require.NoError(t, err)
	assert.Equal(t, defaultTableNames, names)
}

// TestFeishuService_getPrimaryKey tests primary key mapping.
func TestFeishuService_getPrimaryKey(t *testing.T) {
	assert.Equal(t, "simulated_date", getPrimaryKey("daily_metrics"))
	assert.Equal(t, "event_id", getPrimaryKey("alert_events"))
	assert.Equal(t, "recommendation_id", getPrimaryKey("strategy_recommendations"))
	assert.Equal(t, "task_id", getPrimaryKey("action_tasks"))
	assert.Equal(t, "review_id", getPrimaryKey("review_retro"))
	assert.Equal(t, "record_id", getPrimaryKey("unknown"))
}

// TestFeishuService_ExportTables_NotConfigured tests unconfigured export.
func TestFeishuService_ExportTables_NotConfigured(t *testing.T) {
	root, cleanup := setupTestEnv(t)
	defer cleanup()
	svc := NewFeishuService(false, WithProjectRoot(root))

	result, err := svc.ExportTables(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, "not_configured", result.Status)
	assert.Empty(t, result.Tables)
}

// TestFeishuService_ExportTables_DryRun tests dry-run export.
func TestFeishuService_ExportTables_DryRun(t *testing.T) {
	root, cleanup := setupTestEnv(t)
	defer cleanup()
	writeYAML(t, filepath.Join(root, "config", "feishu_app.yml"), map[string]string{
		"app_id":     "id",
		"app_secret": "secret",
	})

	svc := NewFeishuService(true, WithProjectRoot(root))
	result, err := svc.ExportTables(context.Background(), []string{"daily_metrics"})
	require.NoError(t, err)
	assert.Equal(t, "preview", result.Status)
	require.Len(t, result.Tables, 1)
	assert.Equal(t, "daily_metrics", result.Tables[0].Name)
	assert.Equal(t, "preview", result.Tables[0].Status)
}

// TestFeishuService_ExportTables_Real tests actual export with CSV files.
func TestFeishuService_ExportTables_Real(t *testing.T) {
	root, cleanup := setupTestEnv(t)
	defer cleanup()
	writeYAML(t, filepath.Join(root, "config", "feishu_app.yml"), map[string]string{
		"app_id":     "id",
		"app_secret": "secret",
	})
	writeYAML(t, filepath.Join(root, "config", "feishu_table_ids.yml"), map[string]any{
		"tables": map[string]any{
			"daily_metrics": map[string]string{"table_id": "tbl1"},
		},
	})

	csvPath := filepath.Join(root, "data", "feishu", "daily_metrics_for_feishu.csv")
	f, err := os.Create(csvPath)
	require.NoError(t, err)
	fmt.Fprintln(f, "date,gmv")
	fmt.Fprintln(f, "2024-01-01,100")
	fmt.Fprintln(f, "2024-01-02,200")
	f.Close()

	svc := NewFeishuService(false, WithProjectRoot(root))
	result, err := svc.ExportTables(context.Background(), []string{"daily_metrics"})
	require.NoError(t, err)
	assert.Equal(t, "exported", result.Status)
	require.Len(t, result.Tables, 1)
	assert.Equal(t, "daily_metrics", result.Tables[0].Name)
	assert.Equal(t, 2, result.Tables[0].Rows)
	assert.Equal(t, csvPath, result.Tables[0].File)
}

// TestFeishuService_ExportTables_MissingCSV tests export when CSV is missing.
func TestFeishuService_ExportTables_MissingCSV(t *testing.T) {
	root, cleanup := setupTestEnv(t)
	defer cleanup()
	writeYAML(t, filepath.Join(root, "config", "feishu_app.yml"), map[string]string{
		"app_id":     "id",
		"app_secret": "secret",
	})

	svc := NewFeishuService(false, WithProjectRoot(root))
	result, err := svc.ExportTables(context.Background(), []string{"daily_metrics"})
	require.NoError(t, err)
	assert.Equal(t, "exported", result.Status)
	require.Len(t, result.Tables, 1)
	assert.Equal(t, 0, result.Tables[0].Rows)
}

// TestFeishuService_SyncToFeishu_NotConfigured tests unconfigured sync.
func TestFeishuService_SyncToFeishu_NotConfigured(t *testing.T) {
	root, cleanup := setupTestEnv(t)
	defer cleanup()
	svc := NewFeishuService(false, WithProjectRoot(root))

	result, err := svc.SyncToFeishu(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, "not_configured", result.Status)
}

// TestFeishuService_SyncToFeishu_DryRun tests dry-run sync.
func TestFeishuService_SyncToFeishu_DryRun(t *testing.T) {
	root, cleanup := setupTestEnv(t)
	defer cleanup()
	writeYAML(t, filepath.Join(root, "config", "feishu_app.yml"), map[string]string{
		"app_id":     "id",
		"app_secret": "secret",
	})

	svc := NewFeishuService(true, WithProjectRoot(root))
	result, err := svc.SyncToFeishu(context.Background(), []string{"daily_metrics"})
	require.NoError(t, err)
	assert.Equal(t, "preview", result.Status)
	require.Len(t, result.Tables, 1)
	assert.Equal(t, 0, result.Tables[0].Created)
}

// TestFeishuService_SyncToFeishu_Real tests actual sync with mock client.
func TestFeishuService_SyncToFeishu_Real(t *testing.T) {
	root, cleanup := setupTestEnv(t)
	defer cleanup()
	writeYAML(t, filepath.Join(root, "config", "feishu_app.yml"), map[string]string{
		"app_id":     "id",
		"app_secret": "secret",
	})
	writeYAML(t, filepath.Join(root, "config", "feishu_table_ids.yml"), map[string]any{
		"tables": map[string]any{
			"daily_metrics": map[string]string{"table_id": "tbl1"},
		},
	})

	csvPath := filepath.Join(root, "data", "feishu", "daily_metrics_for_feishu.csv")
	f, err := os.Create(csvPath)
	require.NoError(t, err)
	fmt.Fprintln(f, "simulated_date,gmv")
	fmt.Fprintln(f, "2024-01-01,100")
	fmt.Fprintln(f, "2024-01-02,200")
	f.Close()

	mock := &mockBitableClient{
		upsertByKeyFunc: func(tableID string, records []map[string]any, keyField string) (created []map[string]any, updated []map[string]any, err error) {
			assert.Equal(t, "tbl1", tableID)
			assert.Equal(t, "simulated_date", keyField)
			return records, nil, nil
		},
	}

	svc := NewFeishuService(false, WithProjectRoot(root), WithFeishuClient(mock))
	result, err := svc.SyncToFeishu(context.Background(), []string{"daily_metrics"})
	require.NoError(t, err)
	assert.Equal(t, "synced", result.Status)
	require.Len(t, result.Tables, 1)
	assert.Equal(t, 2, result.Tables[0].Created)
	assert.Equal(t, "synced", result.Tables[0].Status)
}

// TestFeishuService_SyncToFeishu_NoTableID tests sync when table ID is missing.
func TestFeishuService_SyncToFeishu_NoTableID(t *testing.T) {
	root, cleanup := setupTestEnv(t)
	defer cleanup()
	writeYAML(t, filepath.Join(root, "config", "feishu_app.yml"), map[string]string{
		"app_id":     "id",
		"app_secret": "secret",
	})

	svc := NewFeishuService(false, WithProjectRoot(root))
	result, err := svc.SyncToFeishu(context.Background(), []string{"daily_metrics"})
	require.NoError(t, err)
	assert.Equal(t, "synced", result.Status)
	require.Len(t, result.Tables, 1)
	assert.Equal(t, "skipped", result.Tables[0].Status)
}

// TestFeishuService_SyncToFeishu_EmptyCSV tests sync with empty CSV.
func TestFeishuService_SyncToFeishu_EmptyCSV(t *testing.T) {
	root, cleanup := setupTestEnv(t)
	defer cleanup()
	writeYAML(t, filepath.Join(root, "config", "feishu_app.yml"), map[string]string{
		"app_id":     "id",
		"app_secret": "secret",
	})
	writeYAML(t, filepath.Join(root, "config", "feishu_table_ids.yml"), map[string]any{
		"tables": map[string]any{
			"daily_metrics": map[string]string{"table_id": "tbl1"},
		},
	})

	csvPath := filepath.Join(root, "data", "feishu", "daily_metrics_for_feishu.csv")
	f, err := os.Create(csvPath)
	require.NoError(t, err)
	fmt.Fprintln(f, "simulated_date,gmv")
	f.Close()

	svc := NewFeishuService(false, WithProjectRoot(root))
	result, err := svc.SyncToFeishu(context.Background(), []string{"daily_metrics"})
	require.NoError(t, err)
	require.Len(t, result.Tables, 1)
	assert.Equal(t, "skipped", result.Tables[0].Status)
}

// TestFeishuService_SyncToFeishu_ClientError tests sync when client fails.
func TestFeishuService_SyncToFeishu_ClientError(t *testing.T) {
	root, cleanup := setupTestEnv(t)
	defer cleanup()
	writeYAML(t, filepath.Join(root, "config", "feishu_app.yml"), map[string]string{
		"app_id":     "id",
		"app_secret": "secret",
	})
	writeYAML(t, filepath.Join(root, "config", "feishu_table_ids.yml"), map[string]any{
		"tables": map[string]any{
			"daily_metrics": map[string]string{"table_id": "tbl1"},
		},
	})

	csvPath := filepath.Join(root, "data", "feishu", "daily_metrics_for_feishu.csv")
	f, err := os.Create(csvPath)
	require.NoError(t, err)
	fmt.Fprintln(f, "simulated_date,gmv")
	fmt.Fprintln(f, "2024-01-01,100")
	f.Close()

	mock := &mockBitableClient{
		upsertByKeyFunc: func(string, []map[string]any, string) ([]map[string]any, []map[string]any, error) {
			return nil, nil, fmt.Errorf("api error")
		},
	}

	svc := NewFeishuService(false, WithProjectRoot(root), WithFeishuClient(mock))
	result, err := svc.SyncToFeishu(context.Background(), []string{"daily_metrics"})
	require.NoError(t, err)
	require.Len(t, result.Tables, 1)
	assert.Equal(t, "failed", result.Tables[0].Status)
}

// TestFeishuService_ImportStatusFromFeishu_NotConfigured tests unconfigured import.
func TestFeishuService_ImportStatusFromFeishu_NotConfigured(t *testing.T) {
	root, cleanup := setupTestEnv(t)
	defer cleanup()
	svc := NewFeishuService(false, WithProjectRoot(root))

	result, err := svc.ImportStatusFromFeishu(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, "not_configured", result.Status)
}

// TestFeishuService_ImportStatusFromFeishu_DryRun tests dry-run import.
func TestFeishuService_ImportStatusFromFeishu_DryRun(t *testing.T) {
	root, cleanup := setupTestEnv(t)
	defer cleanup()
	writeYAML(t, filepath.Join(root, "config", "feishu_app.yml"), map[string]string{
		"app_id":     "id",
		"app_secret": "secret",
	})

	svc := NewFeishuService(true, WithProjectRoot(root))
	result, err := svc.ImportStatusFromFeishu(context.Background(), []string{"action_tasks"})
	require.NoError(t, err)
	assert.Equal(t, "preview", result.Status)
	require.Len(t, result.Tables, 1)
	assert.Equal(t, 0, result.Tables[0].Pulled)
}

// TestFeishuService_ImportStatusFromFeishu_Real tests actual import with mock client.
func TestFeishuService_ImportStatusFromFeishu_Real(t *testing.T) {
	root, cleanup := setupTestEnv(t)
	defer cleanup()
	writeYAML(t, filepath.Join(root, "config", "feishu_app.yml"), map[string]string{
		"app_id":     "id",
		"app_secret": "secret",
	})
	writeYAML(t, filepath.Join(root, "config", "feishu_table_ids.yml"), map[string]any{
		"tables": map[string]any{
			"action_tasks": map[string]string{"table_id": "tbl_tasks"},
			"review_retro": map[string]string{"table_id": "tbl_review"},
		},
	})

	mock := &mockBitableClient{
		listRecordsFunc: func(tableID string, pageSize int, filterConfig map[string]any) ([]map[string]any, error) {
			if tableID == "tbl_tasks" {
				return []map[string]any{
					{"task_id": "t1", "status": "done"},
					{"task_id": "t2", "status": "todo"},
				}, nil
			}
			return []map[string]any{
				{"review_id": "r1", "outcome": "good"},
			}, nil
		},
	}

	svc := NewFeishuService(false, WithProjectRoot(root), WithFeishuClient(mock))
	result, err := svc.ImportStatusFromFeishu(context.Background(), []string{"action_tasks", "review_retro"})
	require.NoError(t, err)
	assert.Equal(t, "imported", result.Status)
	require.Len(t, result.Tables, 2)

	var taskResult, reviewResult *FeishuImportTableResult
	for i := range result.Tables {
		if result.Tables[i].Name == "action_tasks" {
			taskResult = &result.Tables[i]
		} else {
			reviewResult = &result.Tables[i]
		}
	}
	require.NotNil(t, taskResult)
	require.NotNil(t, reviewResult)
	assert.Equal(t, 2, taskResult.Pulled)
	assert.Equal(t, 1, reviewResult.Pulled)

	// Verify snapshot file was written
	snapshotPath := filepath.Join(root, "data", "ops", "action_task_status_snapshot.csv")
	_, statErr := os.Stat(snapshotPath)
	assert.NoError(t, statErr)
}

// TestFeishuService_ImportStatusFromFeishu_NonPullTable tests import for non-pull tables.
func TestFeishuService_ImportStatusFromFeishu_NonPullTable(t *testing.T) {
	root, cleanup := setupTestEnv(t)
	defer cleanup()
	writeYAML(t, filepath.Join(root, "config", "feishu_app.yml"), map[string]string{
		"app_id":     "id",
		"app_secret": "secret",
	})
	writeYAML(t, filepath.Join(root, "config", "feishu_table_ids.yml"), map[string]any{
		"tables": map[string]any{
			"daily_metrics": map[string]string{"table_id": "tbl1"},
		},
	})

	svc := NewFeishuService(false, WithProjectRoot(root))
	result, err := svc.ImportStatusFromFeishu(context.Background(), []string{"daily_metrics"})
	require.NoError(t, err)
	assert.Equal(t, "imported", result.Status)
	require.Len(t, result.Tables, 1)
	assert.Equal(t, 0, result.Tables[0].Pulled)
}

// TestFeishuService_loadCSVRecords tests CSV loading.
func TestFeishuService_loadCSVRecords(t *testing.T) {
	root, cleanup := setupTestEnv(t)
	defer cleanup()
	_ = os.MkdirAll(filepath.Join(root, "data", "feishu"), 0755)

	csvPath := filepath.Join(root, "data", "feishu", "test_for_feishu.csv")
	f, err := os.Create(csvPath)
	require.NoError(t, err)
	fmt.Fprintln(f, "id,name,value")
	fmt.Fprintln(f, "1,foo,10")
	fmt.Fprintln(f, "2,bar,20")
	f.Close()

	svc := NewFeishuService(false, WithProjectRoot(root))
	records, err := svc.loadCSVRecords("test")
	require.NoError(t, err)
	require.Len(t, records, 2)
	assert.Equal(t, "1", records[0]["id"])
	assert.Equal(t, "foo", records[0]["name"])
}

// TestFeishuService_loadCSVRecords_MissingFile tests missing CSV.
func TestFeishuService_loadCSVRecords_MissingFile(t *testing.T) {
	root, cleanup := setupTestEnv(t)
	defer cleanup()
	svc := NewFeishuService(false, WithProjectRoot(root))
	_, err := svc.loadCSVRecords("missing")
	require.Error(t, err)
}

// TestFeishuService_countCSVLines tests line counting.
func TestFeishuService_countCSVLines(t *testing.T) {
	assert.Equal(t, 3, countCSVLines("a\nb\nc"))
	assert.Equal(t, 2, countCSVLines("a\n\nb"))
	assert.Equal(t, 0, countCSVLines(""))
	assert.Equal(t, 1, countCSVLines("only"))
}

// TestFeishuService_ExportTables_InvalidTable tests export with invalid table.
func TestFeishuService_ExportTables_InvalidTable(t *testing.T) {
	root, cleanup := setupTestEnv(t)
	defer cleanup()
	writeYAML(t, filepath.Join(root, "config", "feishu_app.yml"), map[string]string{
		"app_id":     "id",
		"app_secret": "secret",
	})

	svc := NewFeishuService(false, WithProjectRoot(root))
	_, err := svc.ExportTables(context.Background(), []string{"nonexistent"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown table names")
}

// TestFeishuService_ExportTables_AllAvailable tests export of all tables.
func TestFeishuService_ExportTables_AllAvailable(t *testing.T) {
	root, cleanup := setupTestEnv(t)
	defer cleanup()
	writeYAML(t, filepath.Join(root, "config", "feishu_app.yml"), map[string]string{
		"app_id":     "id",
		"app_secret": "secret",
	})

	svc := NewFeishuService(false, WithProjectRoot(root))
	result, err := svc.ExportTables(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, "exported", result.Status)
	assert.Len(t, result.Tables, len(defaultTableNames))
}

// TestFeishuHTTPClient_SendMessage tests message sending.
func TestFeishuHTTPClient_SendMessage(t *testing.T) {
	client := newFeishuHTTPClient("id", "secret", "token", true)
	msgID, err := client.SendMessage("chat1", "hello", false)
	require.NoError(t, err)
	assert.Contains(t, msgID, "dry_run_message")
}

// TestFeishuHTTPClient_SendMessage_DryRunOverride tests dry-run override.
func TestFeishuHTTPClient_SendMessage_DryRunOverride(t *testing.T) {
	client := newFeishuHTTPClient("id", "secret", "token", false)
	msgID, err := client.SendMessage("chat1", "hello", true)
	require.NoError(t, err)
	assert.Contains(t, msgID, "dry_run_message")
}

// TestFeishuHTTPClient_getTenantAccessToken_DryRun tests token in dry-run mode.
func TestFeishuHTTPClient_getTenantAccessToken_DryRun(t *testing.T) {
	client := newFeishuHTTPClient("id", "secret", "token", true)
	token, err := client.getTenantAccessToken()
	require.NoError(t, err)
	assert.Equal(t, "dry_run_token", token)
}

// TestFeishuHTTPClient_UpsertByKey_DryRun tests upsert in dry-run mode.
func TestFeishuHTTPClient_UpsertByKey_DryRun(t *testing.T) {
	client := newFeishuHTTPClient("id", "secret", "token", true)
	records := []map[string]any{{"id": "1", "name": "test"}}
	created, updated, err := client.UpsertByKey("tbl1", records, "id")
	require.NoError(t, err)
	assert.Len(t, created, 1)
	assert.Len(t, updated, 0)
}

// TestFeishuHTTPClient_ListRecords_DryRun tests list in dry-run mode.
func TestFeishuHTTPClient_ListRecords_DryRun(t *testing.T) {
	client := newFeishuHTTPClient("id", "secret", "token", true)
	records, err := client.ListRecords("tbl1", 50, nil)
	require.NoError(t, err)
	assert.Empty(t, records)
}

// TestParseInt tests integer parsing helper.
func TestParseInt(t *testing.T) {
	assert.Equal(t, 42, parseInt(42))
	assert.Equal(t, 42, parseInt(int64(42)))
	assert.Equal(t, 42, parseInt(float64(42)))
	assert.Equal(t, 42, parseInt("42"))
	assert.Equal(t, 0, parseInt(nil))
	assert.Equal(t, 0, parseInt("not_a_number"))
}

// TestParseFloat tests float parsing helper.
func TestParseFloat(t *testing.T) {
	assert.Equal(t, 42.5, parseFloat(42.5))
	assert.Equal(t, 42.0, parseFloat(42))
	assert.Equal(t, 42.5, parseFloat("42.5"))
	assert.Equal(t, 0.0, parseFloat(nil))
	assert.Equal(t, 0.0, parseFloat("not_a_number"))
}

// TestFeishuService_writeImportSnapshot tests snapshot writing.
func TestFeishuService_writeImportSnapshot(t *testing.T) {
	root, cleanup := setupTestEnv(t)
	defer cleanup()
	svc := NewFeishuService(false, WithProjectRoot(root))

	records := map[string][]map[string]any{
		"action_tasks": {
			{"task_id": "t1", "status": "done"},
		},
	}
	err := svc.writeImportSnapshot(records)
	require.NoError(t, err)

	path := filepath.Join(root, "data", "ops", "action_task_status_snapshot.csv")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "task_id")
	assert.Contains(t, content, "t1")
	assert.Contains(t, content, "action_tasks")
}

// TestFeishuService_ImportStatusFromFeishu_InvalidTable tests import with invalid table.
func TestFeishuService_ImportStatusFromFeishu_InvalidTable(t *testing.T) {
	root, cleanup := setupTestEnv(t)
	defer cleanup()
	writeYAML(t, filepath.Join(root, "config", "feishu_app.yml"), map[string]string{
		"app_id":     "id",
		"app_secret": "secret",
	})

	svc := NewFeishuService(false, WithProjectRoot(root))
	_, err := svc.ImportStatusFromFeishu(context.Background(), []string{"nonexistent"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown table names")
}

// TestFeishuHTTPClient_doRequest_GET tests successful GET request.
func TestFeishuHTTPClient_doRequest_GET(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/bitable/v1/apps/app1/tables/tbl1/records", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{
				"items": []map[string]any{
					{"record_id": "rec1", "fields": map[string]any{"name": "test"}},
				},
				"has_more": false,
			},
		})
	}))
	defer server.Close()

	client := newFeishuHTTPClient("id", "secret", "app1", false)
	client.baseURL = server.URL
	client.accessToken = "token"
	client.tokenExpiry = time.Now().Add(time.Hour)

	resp, err := client.doRequest("GET", "/bitable/v1/apps/app1/tables/tbl1/records", map[string]any{"page_size": 50}, false)
	require.NoError(t, err)
	data, _ := resp["data"].(map[string]any)
	require.NotNil(t, data)
}

// TestFeishuHTTPClient_doRequest_POST tests successful POST request.
func TestFeishuHTTPClient_doRequest_POST(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{"message_id": "msg123"},
		})
	}))
	defer server.Close()

	client := newFeishuHTTPClient("id", "secret", "app1", false)
	client.baseURL = server.URL
	client.accessToken = "token"
	client.tokenExpiry = time.Now().Add(time.Hour)

	resp, err := client.doRequest("POST", "/im/v1/messages", map[string]any{"text": "hello"}, false)
	require.NoError(t, err)
	data, _ := resp["data"].(map[string]any)
	require.NotNil(t, data)
}

// TestFeishuHTTPClient_doRequest_APIError tests API error response.
func TestFeishuHTTPClient_doRequest_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 999,
			"msg":  "bad request",
		})
	}))
	defer server.Close()

	client := newFeishuHTTPClient("id", "secret", "app1", false)
	client.baseURL = server.URL
	client.accessToken = "token"
	client.tokenExpiry = time.Now().Add(time.Hour)

	_, err := client.doRequest("GET", "/test", nil, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad request")
}

// TestFeishuHTTPClient_ListRecords_Real tests list records via mock server.
func TestFeishuHTTPClient_ListRecords_Real(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{
				"items": []map[string]any{
					{"record_id": "rec1", "fields": map[string]any{"task_id": "t1"}},
					{"record_id": "rec2", "fields": map[string]any{"task_id": "t2"}},
				},
				"has_more": false,
			},
		})
	}))
	defer server.Close()

	client := newFeishuHTTPClient("id", "secret", "app1", false)
	client.baseURL = server.URL
	client.accessToken = "token"
	client.tokenExpiry = time.Now().Add(time.Hour)

	records, err := client.ListRecords("tbl1", 50, nil)
	require.NoError(t, err)
	assert.Len(t, records, 2)
}

// TestFeishuHTTPClient_SendMessage_Real tests send message via mock server.
func TestFeishuHTTPClient_SendMessage_Real(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{"message_id": "msg123"},
		})
	}))
	defer server.Close()

	client := newFeishuHTTPClient("id", "secret", "app1", false)
	client.baseURL = server.URL
	client.accessToken = "token"
	client.tokenExpiry = time.Now().Add(time.Hour)

	msgID, err := client.SendMessage("chat1", "hello", false)
	require.NoError(t, err)
	assert.Equal(t, "msg123", msgID)
}

// TestFeishuHTTPClient_updateRecord_Real tests update record via mock server.
func TestFeishuHTTPClient_updateRecord_Real(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{"record_id": "rec1", "fields": map[string]any{"name": "updated"}},
		})
	}))
	defer server.Close()

	client := newFeishuHTTPClient("id", "secret", "app1", false)
	client.baseURL = server.URL
	client.accessToken = "token"
	client.tokenExpiry = time.Now().Add(time.Hour)

	rec, err := client.updateRecord("tbl1", "rec1", map[string]any{"name": "updated"})
	require.NoError(t, err)
	require.NotNil(t, rec)
	assert.Equal(t, "rec1", rec["record_id"])
}

// TestFeishuHTTPClient_batchCreate_Real tests batch create via mock server.
func TestFeishuHTTPClient_batchCreate_Real(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{
				"records": []map[string]any{
					{"record_id": "rec1"},
					{"record_id": "rec2"},
				},
			},
		})
	}))
	defer server.Close()

	client := newFeishuHTTPClient("id", "secret", "app1", false)
	client.baseURL = server.URL
	client.accessToken = "token"
	client.tokenExpiry = time.Now().Add(time.Hour)

	records := []map[string]any{{"name": "a"}, {"name": "b"}}
	created := client.batchCreate("tbl1", records)
	assert.Len(t, created, 2)
}

// TestBatchToFields tests the batchToFields helper.
func TestBatchToFields(t *testing.T) {
	records := []map[string]any{{"id": "1"}, {"id": "2"}}
	result := batchToFields(records)
	require.Len(t, result, 2)
	assert.Equal(t, map[string]any{"fields": map[string]any{"id": "1"}}, result[0])
}

// TestFeishuHTTPClient_getTenantAccessToken_Real tests token fetch via mock server.
func TestFeishuHTTPClient_getTenantAccessToken_Real(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/auth/v3/tenant_access_token/internal", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code":                0,
			"tenant_access_token": "test_token_123",
			"expire":              7200,
		})
	}))
	defer server.Close()

	client := newFeishuHTTPClient("id", "secret", "app1", false)
	client.baseURL = server.URL

	token, err := client.getTenantAccessToken()
	require.NoError(t, err)
	assert.Equal(t, "test_token_123", token)
	assert.Equal(t, "test_token_123", client.accessToken)
}

// TestFeishuHTTPClient_UpsertByKey_Real tests upsert via mock server.
func TestFeishuHTTPClient_UpsertByKey_Real(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/records") && !strings.Contains(r.URL.Path, "/batch") {
			json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{
					"items": []map[string]any{
						{"record_id": "rec1", "fields": map[string]any{"task_id": "t1", "status": "old"}},
					},
					"has_more": false,
				},
			})
		} else if strings.Contains(r.URL.Path, "/batch_create") {
			json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{
					"records": []map[string]any{{"record_id": "rec2"}},
				},
			})
		} else if strings.Contains(r.URL.Path, "/records/") {
			json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{"record_id": "rec1", "fields": map[string]any{"task_id": "t1", "status": "updated"}},
			})
		}
	}))
	defer server.Close()

	client := newFeishuHTTPClient("id", "secret", "app1", false)
	client.baseURL = server.URL
	client.accessToken = "token"
	client.tokenExpiry = time.Now().Add(time.Hour)

	records := []map[string]any{
		{"task_id": "t1", "status": "updated"},
		{"task_id": "t2", "status": "new"},
	}
	created, updated, err := client.UpsertByKey("tbl1", records, "task_id")
	require.NoError(t, err)
	assert.Len(t, updated, 1)
	assert.Len(t, created, 1)
}

// TestFeishuHTTPClient_doRequest_RateLimitRetry tests retry on rate limit.
func TestFeishuHTTPClient_doRequest_RateLimitRetry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"code": 170002,
				"msg":  "rate limited",
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{"ok": true},
		})
	}))
	defer server.Close()

	client := newFeishuHTTPClient("id", "secret", "app1", false)
	client.baseURL = server.URL
	client.accessToken = "token"
	client.tokenExpiry = time.Now().Add(time.Hour)

	resp, err := client.doRequest("GET", "/test", nil, false)
	require.NoError(t, err)
	assert.Equal(t, 2, attempts)
	data, _ := resp["data"].(map[string]any)
	require.NotNil(t, data)
}
