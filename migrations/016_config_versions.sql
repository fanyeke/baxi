-- +goose Up
-- +goose StatementBegin

-- Migration 016: Config version tracking for governance YAML configs.
-- Tracks loaded config versions and their content hashes for:
--   - action_registry.yml
--   - alert_rules.yml
--   - access_policy.yml

CREATE TABLE IF NOT EXISTS ops.config_versions (
    config_name   TEXT PRIMARY KEY,
    version       TEXT NOT NULL,
    content_hash  TEXT NOT NULL,
    loaded_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    active        BOOLEAN NOT NULL DEFAULT true
);

COMMENT ON TABLE ops.config_versions IS 'Tracks loaded versions and content hashes of governance YAML configs';
COMMENT ON COLUMN ops.config_versions.config_name IS 'Config file name (e.g. action_registry.yml)';
COMMENT ON COLUMN ops.config_versions.version IS 'Version identifier (auto-generated if not specified)';
COMMENT ON COLUMN ops.config_versions.content_hash IS 'SHA-256 hex digest of config file content';
COMMENT ON COLUMN ops.config_versions.loaded_at IS 'Timestamp when config was loaded/updated';
COMMENT ON COLUMN ops.config_versions.active IS 'Whether this config version is currently active';

-- Index for fast active config lookups
CREATE INDEX IF NOT EXISTS idx_config_versions_active
    ON ops.config_versions(active)
    WHERE active = true;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS ops.idx_config_versions_active;
DROP TABLE IF EXISTS ops.config_versions CASCADE;

-- +goose StatementEnd
