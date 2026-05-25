-- +goose Up
-- +goose StatementBegin

-- v0.5.3 Migration 006: Governance tables in gov schema

-- 1. Governance checkpoint audit trail (from SQLite governance_checkpoints)
CREATE TABLE IF NOT EXISTS gov.governance_checkpoint (
    checkpoint_id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    action_type TEXT NOT NULL,
    endpoint TEXT NOT NULL,
    actor TEXT NOT NULL,
    request_id TEXT,
    justification TEXT,
    mode TEXT NOT NULL DEFAULT 'dry_run',
    status TEXT NOT NULL DEFAULT 'recorded',
    metadata_json JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Health check results (from SQLite governance_health_results)
CREATE TABLE IF NOT EXISTS gov.health_check_result (
    result_id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    check_id TEXT NOT NULL,
    check_type TEXT NOT NULL,
    status TEXT NOT NULL,
    detail TEXT,
    checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. YAML config snapshots for governance configuration history
CREATE TABLE IF NOT EXISTS gov.config_snapshot (
    snapshot_id BIGSERIAL PRIMARY KEY,
    config_key TEXT NOT NULL,
    config_type TEXT,
    source_path TEXT,
    content_jsonb JSONB,
    content_hash TEXT,
    loaded_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_config_key_hash UNIQUE (config_key, content_hash)
);

-- 4. Ontology object schema definitions
CREATE TABLE IF NOT EXISTS gov.object_schema (
    object_schema_id BIGSERIAL PRIMARY KEY,
    object_type TEXT NOT NULL,
    object_name TEXT,
    schema_jsonb JSONB,
    version TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_object_type_version UNIQUE (object_type, version)
);

-- 5. Data classification rules
CREATE TABLE IF NOT EXISTS gov.data_classification (
    classification_id BIGSERIAL PRIMARY KEY,
    field_path TEXT NOT NULL,
    classification_level TEXT,
    sensitivity_score NUMERIC(4,2),
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 6. Data lineage records
CREATE TABLE IF NOT EXISTS gov.data_lineage (
    lineage_id BIGSERIAL PRIMARY KEY,
    source_table TEXT,
    source_column TEXT,
    target_table TEXT,
    target_column TEXT,
    transformation_logic TEXT,
    confidence NUMERIC(4,2),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 7. Access control policies
CREATE TABLE IF NOT EXISTS gov.access_policy (
    policy_id BIGSERIAL PRIMARY KEY,
    policy_name TEXT NOT NULL,
    resource_type TEXT,
    resource_pattern TEXT,
    action TEXT,
    principal_type TEXT,
    principal_pattern TEXT,
    effect TEXT,
    conditions_jsonb JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for governance_checkpoint
CREATE INDEX IF NOT EXISTS idx_gov_checkpoint_action_type ON gov.governance_checkpoint(action_type);
CREATE INDEX IF NOT EXISTS idx_gov_checkpoint_actor ON gov.governance_checkpoint(actor);
CREATE INDEX IF NOT EXISTS idx_gov_checkpoint_created_at ON gov.governance_checkpoint(created_at DESC);

-- Indexes for health_check_result
CREATE INDEX IF NOT EXISTS idx_gov_health_check_id ON gov.health_check_result(check_id);
CREATE INDEX IF NOT EXISTS idx_gov_health_check_type ON gov.health_check_result(check_type);
CREATE INDEX IF NOT EXISTS idx_gov_health_checked_at ON gov.health_check_result(checked_at DESC);

-- Indexes for config_snapshot
CREATE INDEX IF NOT EXISTS idx_gov_config_key ON gov.config_snapshot(config_key);
CREATE INDEX IF NOT EXISTS idx_gov_config_type ON gov.config_snapshot(config_type);

-- Indexes for object_schema
CREATE INDEX IF NOT EXISTS idx_gov_object_type ON gov.object_schema(object_type);

-- Indexes for data_classification
CREATE INDEX IF NOT EXISTS idx_gov_classification_level ON gov.data_classification(classification_level);
CREATE INDEX IF NOT EXISTS idx_gov_classification_field_path ON gov.data_classification(field_path);

-- Indexes for data_lineage
CREATE INDEX IF NOT EXISTS idx_gov_lineage_source ON gov.data_lineage(source_table, source_column);
CREATE INDEX IF NOT EXISTS idx_gov_lineage_target ON gov.data_lineage(target_table, target_column);

-- Indexes for access_policy
CREATE UNIQUE INDEX IF NOT EXISTS idx_gov_policy_name ON gov.access_policy(policy_name);
CREATE INDEX IF NOT EXISTS idx_gov_policy_resource_type ON gov.access_policy(resource_type);
CREATE INDEX IF NOT EXISTS idx_gov_policy_effect ON gov.access_policy(effect);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS gov.access_policy;
DROP TABLE IF EXISTS gov.data_lineage;
DROP TABLE IF EXISTS gov.data_classification;
DROP TABLE IF EXISTS gov.object_schema;
DROP TABLE IF EXISTS gov.config_snapshot;
DROP TABLE IF EXISTS gov.health_check_result;
DROP TABLE IF EXISTS gov.governance_checkpoint;

-- +goose StatementEnd
