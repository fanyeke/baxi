-- +goose Up
-- +goose StatementBegin

-- v0.5.4 Migration 009: Additional governance indexes for Phase 5 query performance

-- Composite index for config_snapshot type + key lookup
CREATE INDEX IF NOT EXISTS idx_gov_config_key_type ON gov.config_snapshot(config_key, config_type);

-- Composite index for data_classification field + level queries
CREATE INDEX IF NOT EXISTS idx_gov_class_field_level ON gov.data_classification(field_path, classification_level);

-- Composite index for data_lineage source-to-target traversal
CREATE INDEX IF NOT EXISTS idx_gov_lineage_source_target ON gov.data_lineage(source_table, target_table);

-- Composite index for access_policy resource + action lookup
CREATE INDEX IF NOT EXISTS idx_gov_policy_resource_action ON gov.access_policy(resource_type, action);

-- Created_at index for config_snapshot (time-based queries)
CREATE INDEX IF NOT EXISTS idx_gov_config_loaded_at ON gov.config_snapshot(loaded_at DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS gov.idx_gov_config_key_type;
DROP INDEX IF EXISTS gov.idx_gov_class_field_level;
DROP INDEX IF EXISTS gov.idx_gov_lineage_source_target;
DROP INDEX IF EXISTS gov.idx_gov_policy_resource_action;
DROP INDEX IF EXISTS gov.idx_gov_config_loaded_at;

-- +goose StatementEnd
