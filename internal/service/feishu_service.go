package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"baxi/internal/feishu"
)

const (
	batchLimit = 500
)

var defaultTableNames = []string{
	"daily_metrics",
	"alert_events",
	"strategy_recommendations",
	"action_tasks",
	"review_retro",
}

// Config types
type feishuAppConfig struct {
	AppID     string `yaml:"app_id"`
	AppSecret string `yaml:"app_secret"`
	ChatID    string `yaml:"chat_id"`
}

type feishuTableIDsConfig struct {
	Tables map[string]struct {
		TableID string `yaml:"table_id"`
		Name    string `yaml:"name"`
	} `yaml:"tables"`
}

// FeishuBitableClient defines operations on Feishu bitable tables.
type FeishuBitableClient interface {
	ListRecords(tableID string, pageSize int, filterConfig map[string]any) ([]map[string]any, error)
	UpsertByKey(tableID string, records []map[string]any, keyField string) (created []map[string]any, updated []map[string]any, err error)
	SendMessage(chatID, content string, dryRun bool) (string, error)
}

// FeishuService provides Feishu bitable CRUD, data sync, export, and status import.
type FeishuService struct {
	dryRun      bool
	config      *feishuConfig
	client      FeishuBitableClient
	projectRoot string
	feishuDir   string
	systemDir   string
}

type feishuConfig struct {
	appID     string
	appSecret string
	appToken  string
	chatID    string
	tableIDs  map[string]string
}

// FeishuServiceOption configures FeishuService.
type FeishuServiceOption func(*FeishuService)

// WithFeishuClient sets a custom bitable client (used for testing).
func WithFeishuClient(client FeishuBitableClient) FeishuServiceOption {
	return func(s *FeishuService) {
		s.client = client
	}
}

// WithProjectRoot overrides the project root directory.
func WithProjectRoot(root string) FeishuServiceOption {
	return func(s *FeishuService) {
		s.projectRoot = root
		s.feishuDir = filepath.Join(root, "data", "feishu")
		s.systemDir = filepath.Join(root, "data", "system")
	}
}

// NewFeishuService creates a new FeishuService.
func NewFeishuService(dryRun bool, opts ...FeishuServiceOption) *FeishuService {
	cwd, _ := os.Getwd()
	s := &FeishuService{
		dryRun:      dryRun,
		projectRoot: cwd,
		feishuDir:   filepath.Join(cwd, "data", "feishu"),
		systemDir:   filepath.Join(cwd, "data", "system"),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// loadConfig loads Feishu credentials and table IDs from config files.
func (s *FeishuService) loadConfig() *feishuConfig {
	if s.config != nil {
		return s.config
	}

	cfg := &feishuConfig{
		appID:     os.Getenv("FEISHU_APP_ID"),
		appSecret: os.Getenv("FEISHU_APP_SECRET"),
		appToken:  os.Getenv("FEISHU_BASE_APP_TOKEN"),
		chatID:    os.Getenv("FEISHU_CHAT_ID"),
		tableIDs:  make(map[string]string),
	}

	appConfigPath := filepath.Join(s.projectRoot, "config", "feishu_app.yml")
	if data, err := os.ReadFile(appConfigPath); err == nil {
		var appCfg feishuAppConfig
		if err := yaml.Unmarshal(data, &appCfg); err == nil {
			if cfg.appID == "" {
				cfg.appID = appCfg.AppID
			}
			if cfg.appSecret == "" {
				cfg.appSecret = appCfg.AppSecret
			}
			if cfg.chatID == "" {
				cfg.chatID = appCfg.ChatID
			}
		}
	}

	tableIDsPath := filepath.Join(s.projectRoot, "config", "feishu_table_ids.yml")
	if data, err := os.ReadFile(tableIDsPath); err == nil {
		var tids feishuTableIDsConfig
		if err := yaml.Unmarshal(data, &tids); err == nil {
			for name, info := range tids.Tables {
				cfg.tableIDs[name] = info.TableID
			}
		}
	}

	s.config = cfg
	return cfg
}

// isConfigured returns true when app_id and app_secret are set.
func (s *FeishuService) isConfigured() bool {
	cfg := s.loadConfig()
	return cfg.appID != "" && cfg.appSecret != ""
}

// getClient lazily initializes the Feishu bitable client.
func (s *FeishuService) getClient() FeishuBitableClient {
	if s.client != nil {
		return s.client
	}
	cfg := s.loadConfig()
	s.client = feishu.NewClient(cfg.appID, cfg.appSecret, cfg.appToken, s.dryRun)
	return s.client
}

// getTableNames validates and resolves table names.
func (s *FeishuService) getTableNames(tableNames []string) ([]string, error) {
	cfg := s.loadConfig()
	available := make([]string, 0, len(cfg.tableIDs))
	for name := range cfg.tableIDs {
		available = append(available, name)
	}
	if len(available) == 0 {
		available = append([]string(nil), defaultTableNames...)
	}

	if len(tableNames) == 0 {
		return available, nil
	}

	availableSet := make(map[string]bool, len(available))
	for _, a := range available {
		availableSet[a] = true
	}

	var unknown []string
	for _, t := range tableNames {
		if !availableSet[t] {
			unknown = append(unknown, t)
		}
	}
	if len(unknown) > 0 {
		return nil, fmt.Errorf("unknown table names: %s. Available: %s",
			strings.Join(unknown, ", "), strings.Join(available, ", "))
	}
	return tableNames, nil
}

// getPrimaryKey returns the primary key field for a table.
func getPrimaryKey(tableName string) string {
	mapping := map[string]string{
		"daily_metrics":            "simulated_date",
		"alert_events":             "event_id",
		"strategy_recommendations": "recommendation_id",
		"action_tasks":             "task_id",
		"review_retro":             "review_id",
	}
	if pk, ok := mapping[tableName]; ok {
		return pk
	}
	return "record_id"
}

// Export

type FeishuExportTableResult struct {
	Name   string `json:"name"`
	Rows   int    `json:"rows"`
	File   string `json:"file"`
	Status string `json:"status"`
}

type FeishuExportResult struct {
	Status  string                    `json:"status"`
	Message string                    `json:"message"`
	Tables  []FeishuExportTableResult `json:"tables"`
}

// ExportTables exports local data to CSV files for Feishu tables.
func (s *FeishuService) ExportTables(ctx context.Context, tableNames []string) (*FeishuExportResult, error) {
	if !s.isConfigured() {
		return &FeishuExportResult{
			Status:  "not_configured",
			Message: "Feishu credentials not configured",
			Tables:  []FeishuExportTableResult{},
		}, nil
	}

	resolved, err := s.getTableNames(tableNames)
	if err != nil {
		return nil, err
	}

	if s.dryRun {
		tables := make([]FeishuExportTableResult, len(resolved))
		for i, t := range resolved {
			tables[i] = FeishuExportTableResult{Name: t, Rows: 0, File: "", Status: "preview"}
		}
		return &FeishuExportResult{
			Status:  "preview",
			Message: "Dry-run: no files written",
			Tables:  tables,
		}, nil
	}

	tables := make([]FeishuExportTableResult, 0, len(resolved))
	for _, name := range resolved {
		csvPath := filepath.Join(s.feishuDir, fmt.Sprintf("%s_for_feishu.csv", name))
		rows := 0
		if data, err := os.ReadFile(csvPath); err == nil {
			rows = countCSVLines(string(data)) - 1
			if rows < 0 {
				rows = 0
			}
		}
		tables = append(tables, FeishuExportTableResult{
			Name:   name,
			Rows:   rows,
			File:   csvPath,
			Status: "exported",
		})
	}

	return &FeishuExportResult{
		Status:  "exported",
		Message: "",
		Tables:  tables,
	}, nil
}

func countCSVLines(data string) int {
	lines := strings.Split(data, "\n")
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

// Sync

type FeishuSyncTableResult struct {
	Name    string `json:"name"`
	Created int    `json:"created"`
	Updated int    `json:"updated"`
	Status  string `json:"status"`
}

type FeishuSyncResult struct {
	Status  string                  `json:"status"`
	Message string                  `json:"message"`
	Tables  []FeishuSyncTableResult `json:"tables"`
}

// SyncToFeishu syncs local CSV data to Feishu bitable tables.
func (s *FeishuService) SyncToFeishu(ctx context.Context, tableNames []string) (*FeishuSyncResult, error) {
	if !s.isConfigured() {
		return &FeishuSyncResult{
			Status:  "not_configured",
			Message: "Feishu credentials not configured",
			Tables:  []FeishuSyncTableResult{},
		}, nil
	}

	resolved, err := s.getTableNames(tableNames)
	if err != nil {
		return nil, err
	}

	if s.dryRun {
		tables := make([]FeishuSyncTableResult, len(resolved))
		for i, t := range resolved {
			tables[i] = FeishuSyncTableResult{Name: t, Created: 0, Updated: 0, Status: "preview"}
		}
		return &FeishuSyncResult{
			Status:  "preview",
			Message: "Dry-run: no Feishu API calls",
			Tables:  tables,
		}, nil
	}

	cfg := s.loadConfig()
	client := s.getClient()

	tables := make([]FeishuSyncTableResult, 0, len(resolved))
	for _, name := range resolved {
		tableID := cfg.tableIDs[name]
		if tableID == "" {
			tables = append(tables, FeishuSyncTableResult{
				Name:    name,
				Created: 0,
				Updated: 0,
				Status:  "skipped",
			})
			continue
		}

		records, err := s.loadCSVRecords(name)
		if err != nil {
			tables = append(tables, FeishuSyncTableResult{
				Name:    name,
				Created: 0,
				Updated: 0,
				Status:  "failed",
			})
			continue
		}

		if len(records) == 0 {
			tables = append(tables, FeishuSyncTableResult{
				Name:    name,
				Created: 0,
				Updated: 0,
				Status:  "skipped",
			})
			continue
		}

		pk := getPrimaryKey(name)
		created, updated, err := client.UpsertByKey(tableID, records, pk)
		if err != nil {
			tables = append(tables, FeishuSyncTableResult{
				Name:    name,
				Created: 0,
				Updated: 0,
				Status:  "failed",
			})
			continue
		}

		tables = append(tables, FeishuSyncTableResult{
			Name:    name,
			Created: len(created),
			Updated: len(updated),
			Status:  "synced",
		})
	}

	return &FeishuSyncResult{
		Status:  "synced",
		Message: "",
		Tables:  tables,
	}, nil
}

func (s *FeishuService) loadCSVRecords(tableName string) ([]map[string]any, error) {
	csvPath := filepath.Join(s.feishuDir, fmt.Sprintf("%s_for_feishu.csv", tableName))
	f, err := os.Open(csvPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	headers, err := reader.Read()
	if err != nil {
		return nil, err
	}

	var records []map[string]any
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		rec := make(map[string]any, len(headers))
		for i, h := range headers {
			if i < len(row) {
				rec[h] = row[i]
			}
		}
		records = append(records, rec)
	}
	return records, nil
}

// Import

type FeishuImportTableResult struct {
	Name     string `json:"name"`
	Pulled   int    `json:"pulled"`
	Imported int    `json:"imported"`
	Skipped  int    `json:"skipped"`
	Status   string `json:"status"`
}

type FeishuImportResult struct {
	Status  string                     `json:"status"`
	Message string                     `json:"message"`
	Tables  []FeishuImportTableResult  `json:"tables"`
}

// ImportStatusFromFeishu pulls status from Feishu and imports to local storage.
func (s *FeishuService) ImportStatusFromFeishu(ctx context.Context, tableNames []string) (*FeishuImportResult, error) {
	if !s.isConfigured() {
		return &FeishuImportResult{
			Status:  "not_configured",
			Message: "Feishu credentials not configured",
			Tables:  []FeishuImportTableResult{},
		}, nil
	}

	resolved, err := s.getTableNames(tableNames)
	if err != nil {
		return nil, err
	}

	if s.dryRun {
		tables := make([]FeishuImportTableResult, len(resolved))
		for i, t := range resolved {
			tables[i] = FeishuImportTableResult{Name: t, Pulled: 0, Imported: 0, Skipped: 0, Status: "preview"}
		}
		return &FeishuImportResult{
			Status:  "preview",
			Message: "Dry-run: no Feishu API calls",
			Tables:  tables,
		}, nil
	}

	cfg := s.loadConfig()
	client := s.getClient()

	pullTables := []string{"action_tasks", "review_retro"}
	pullSet := make(map[string]bool)
	for _, t := range pullTables {
		pullSet[t] = true
	}

	allRecords := make(map[string][]map[string]any)
	for _, name := range resolved {
		if !pullSet[name] {
			continue
		}
		tableID := cfg.tableIDs[name]
		if tableID == "" {
			continue
		}
		records, err := client.ListRecords(tableID, 500, nil)
		if err != nil {
			slog.Warn("failed to pull records from Feishu", "table", name, "error", err)
			continue
		}
		allRecords[name] = records
	}

	if len(allRecords) > 0 {
		_ = s.writeImportSnapshot(allRecords)
	}

	tables := make([]FeishuImportTableResult, 0, len(resolved))
	for _, name := range resolved {
		records := allRecords[name]
		tables = append(tables, FeishuImportTableResult{
			Name:     name,
			Pulled:   len(records),
			Imported: len(records),
			Skipped:  0,
			Status:   "imported",
		})
	}

	return &FeishuImportResult{
		Status:  "imported",
		Message: "",
		Tables:  tables,
	}, nil
}

func (s *FeishuService) writeImportSnapshot(allRecords map[string][]map[string]any) error {
	opDir := filepath.Join(s.projectRoot, "data", "ops")
	if err := os.MkdirAll(opDir, 0755); err != nil {
		return err
	}
	outputPath := filepath.Join(opDir, "action_task_status_snapshot.csv")

	allKeys := make(map[string]bool)
	allKeys["_table"] = true
	allKeys["_record_id"] = true
	allKeys["_pulled_at"] = true
	for _, records := range allRecords {
		for _, rec := range records {
			for k := range rec {
				allKeys[k] = true
			}
		}
	}

	var headers []string
	for k := range allKeys {
		headers = append(headers, k)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	_ = writer.Write(headers)

	now := time.Now().UTC().Format(time.RFC3339)
	for tableName, records := range allRecords {
		for _, rec := range records {
			row := make([]string, len(headers))
			for i, h := range headers {
				switch h {
				case "_table":
					row[i] = tableName
				case "_pulled_at":
					row[i] = now
				case "_record_id":
					if v, ok := rec["task_id"]; ok {
						row[i] = fmt.Sprint(v)
					} else if v, ok := rec["review_id"]; ok {
						row[i] = fmt.Sprint(v)
					} else if v, ok := rec["record_id"]; ok {
						row[i] = fmt.Sprint(v)
					} else {
						row[i] = ""
					}
				default:
					if v, ok := rec[h]; ok {
						row[i] = fmt.Sprint(v)
					} else {
						row[i] = ""
					}
				}
			}
			_ = writer.Write(row)
		}
	}
	writer.Flush()
	return writer.Error()
}

// Helper functions

func parseInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case string:
		i, _ := strconv.Atoi(n)
		return i
	default:
		return 0
	}
}

func parseFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case string:
		f, _ := strconv.ParseFloat(n, 64)
		return f
	default:
		return 0
	}
}
