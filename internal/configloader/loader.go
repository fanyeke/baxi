package configloader

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ConfigLoader loads governance YAML configs and syncs them to gov.* tables.
type ConfigLoader struct {
	pool *pgxpool.Pool
}

// ConfigRegistry holds all loaded typed configs and raw config snapshots.
type ConfigRegistry struct {
	ObjectSchema       *ObjectSchemaConfig
	DataClassification *DataClassificationConfig
	AccessPolicy       *AccessPolicyConfig
	DataLineage        *DataLineageConfig
	DataMarkings       *DataMarkingsConfig
	HealthChecks       *HealthChecksConfig
	CheckpointRules    *CheckpointRulesConfig
	AlertRules         *AlertRulesConfig
	Metrics            *MetricsConfig
	RawConfigs         map[string]RawConfig
}

// RawConfig stores a single loaded YAML config with its metadata.
type RawConfig struct {
	ConfigKey   string
	ConfigType  string
	SourcePath  string
	Content     []byte
	ContentHash string
}

// NewConfigLoader creates a new ConfigLoader backed by a pgx pool.
func NewConfigLoader(pool *pgxpool.Pool) *ConfigLoader {
	return &ConfigLoader{pool: pool}
}

// LoadAll scans the given directory, loads all .yml files, computes content
// hashes, attempts typed parsing for known config types, and returns a
// ConfigRegistry populated with all results.
//
// The directory is scanned non-recursively. Files ending in .example or .yml.example
// are skipped. The returned registry includes both typed config structs (for known
// types) and RawConfig entries for every loaded file.
func (cl *ConfigLoader) LoadAll(ctx context.Context, dir string) (*ConfigRegistry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read config dir %s: %w", dir, err)
	}

	registry := &ConfigRegistry{
		RawConfigs: make(map[string]RawConfig),
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".example") || strings.Contains(entry.Name(), ".yml.example") {
			continue
		}

		configKey := strings.TrimSuffix(entry.Name(), ".yml")
		configType := detectConfigType(configKey)

		path := filepath.Join(dir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			slog.Warn("failed to read config file", "path", path, "error", err)
			continue
		}

		hash := computeHash(content)

		raw := RawConfig{
			ConfigKey:   configKey,
			ConfigType:  configType,
			SourcePath:  entry.Name(),
			Content:     content,
			ContentHash: hash,
		}
		registry.RawConfigs[configKey] = raw

		typed, err := parseConfigByType(configType, content)
		if err != nil {
			slog.Warn("failed to parse typed config",
				"config_key", configKey, "config_type", configType, "error", err)
			continue
		}

		switch v := typed.(type) {
		case *ObjectSchemaConfig:
			registry.ObjectSchema = v
		case *DataClassificationConfig:
			registry.DataClassification = v
		case *AccessPolicyConfig:
			registry.AccessPolicy = v
		case *DataLineageConfig:
			registry.DataLineage = v
		case *DataMarkingsConfig:
			registry.DataMarkings = v
		case *HealthChecksConfig:
			registry.HealthChecks = v
		case *CheckpointRulesConfig:
			registry.CheckpointRules = v
		case *AlertRulesConfig:
			registry.AlertRules = v
		case *MetricsConfig:
			registry.Metrics = v
		}
	}

	slog.Info("config loading complete",
		"dir", dir,
		"total", len(registry.RawConfigs),
	)

	return registry, nil
}

// SyncSnapshots writes all loaded configs to the gov.* database tables.
// It inserts/upserts:
//   - gov.config_snapshot — every raw config
//   - gov.object_schema — parsed object types (aip_object_schema)
//   - gov.data_classification — parsed classifications (data_classification)
//   - gov.data_lineage — parsed lineage edges (data_lineage)
//   - gov.access_policy — parsed role-action entries (access_policy)
//
// Each sync operation is independent; errors from individual table syncs are
// logged but do not prevent other syncs from proceeding.
func (cl *ConfigLoader) SyncSnapshots(ctx context.Context, registry *ConfigRegistry) error {
	tx, err := cl.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var syncErr bool

	if err := syncConfigSnapshots(ctx, tx, registry); err != nil {
		slog.Error("config snapshot sync failed", "error", err)
		syncErr = true
	}

	if err := syncObjectSchema(ctx, tx, registry); err != nil {
		slog.Error("object schema sync failed", "error", err)
		syncErr = true
	}

	if err := syncDataClassification(ctx, tx, registry); err != nil {
		slog.Error("data classification sync failed", "error", err)
		syncErr = true
	}

	if err := syncDataLineage(ctx, tx, registry); err != nil {
		slog.Error("data lineage sync failed", "error", err)
		syncErr = true
	}

	if err := syncAccessPolicy(ctx, tx, registry); err != nil {
		slog.Error("access policy sync failed", "error", err)
		syncErr = true
	}

	if syncErr {
		return fmt.Errorf("one or more sync operations failed")
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	slog.Info("config sync complete",
		"snapshots", len(registry.RawConfigs),
	)

	return nil
}

// ListConfigKeys returns a sorted list of all config keys loaded in the registry.
func ListConfigKeys(registry *ConfigRegistry) []string {
	keys := make([]string, 0, len(registry.RawConfigs))
	for k := range registry.RawConfigs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
