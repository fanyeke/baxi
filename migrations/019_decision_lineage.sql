-- +goose Up
-- +goose StatementBegin

-- Migration 019: Add decision lineage tables for tracing decision flow
-- Adds config versioning columns to ai.decision_case and creates
-- ai.decision_lineage_event + ai.decision_data_snapshot for full audit trail.

-- ============================================================
-- 1. ADD COLUMNS to ai.decision_case (config versioning + snapshots)
-- ============================================================
ALTER TABLE ai.decision_case
    ADD COLUMN IF NOT EXISTS alert_rules_version TEXT,
    ADD COLUMN IF NOT EXISTS alert_rules_hash TEXT,
    ADD COLUMN IF NOT EXISTS action_registry_version TEXT,
    ADD COLUMN IF NOT EXISTS action_registry_hash TEXT,
    ADD COLUMN IF NOT EXISTS context_snapshot_json JSONB,
    ADD COLUMN IF NOT EXISTS data_snapshot_json JSONB;

-- ============================================================
-- 2. CREATE ai.decision_lineage_event (event sourcing for decision flow)
-- ============================================================
CREATE TABLE IF NOT EXISTS ai.decision_lineage_event (
    event_id TEXT PRIMARY KEY,
    case_id TEXT NOT NULL REFERENCES ai.decision_case(case_id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    event_timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    actor TEXT,
    event_data JSONB,
    context_hash TEXT,
    config_hash TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================================
-- 3. CREATE ai.decision_data_snapshot (point-in-time data captures)
-- ============================================================
CREATE TABLE IF NOT EXISTS ai.decision_data_snapshot (
    snapshot_id TEXT PRIMARY KEY,
    case_id TEXT NOT NULL REFERENCES ai.decision_case(case_id) ON DELETE CASCADE,
    snapshot_type TEXT NOT NULL,
    snapshot_json JSONB,
    source_table TEXT,
    row_count INT,
    captured_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================================
-- 4. INDEXES
-- ============================================================

-- Lineage event: lookup by case
CREATE INDEX IF NOT EXISTS idx_ai_decision_lineage_event_case
    ON ai.decision_lineage_event(case_id);

-- Lineage event: filter by event type
CREATE INDEX IF NOT EXISTS idx_ai_decision_lineage_event_type
    ON ai.decision_lineage_event(event_type);

-- Lineage event: chronological ordering per case
CREATE INDEX IF NOT EXISTS idx_ai_decision_lineage_event_ts
    ON ai.decision_lineage_event(case_id, event_timestamp);

-- Data snapshot: lookup by case
CREATE INDEX IF NOT EXISTS idx_ai_decision_data_snapshot_case
    ON ai.decision_data_snapshot(case_id);

-- Data snapshot: filter by snapshot type
CREATE INDEX IF NOT EXISTS idx_ai_decision_data_snapshot_type
    ON ai.decision_data_snapshot(snapshot_type);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- ============================================================
-- REVERSE: Drop indexes
-- ============================================================
DROP INDEX IF EXISTS ai.idx_ai_decision_data_snapshot_type;
DROP INDEX IF EXISTS ai.idx_ai_decision_data_snapshot_case;
DROP INDEX IF EXISTS ai.idx_ai_decision_lineage_event_ts;
DROP INDEX IF EXISTS ai.idx_ai_decision_lineage_event_type;
DROP INDEX IF EXISTS ai.idx_ai_decision_lineage_event_case;

-- ============================================================
-- REVERSE: Drop tables (cascade removes FK deps)
-- ============================================================
DROP TABLE IF EXISTS ai.decision_data_snapshot;
DROP TABLE IF EXISTS ai.decision_lineage_event;

-- ============================================================
-- REVERSE: Drop added columns from ai.decision_case
-- ============================================================
ALTER TABLE ai.decision_case DROP COLUMN IF EXISTS data_snapshot_json;
ALTER TABLE ai.decision_case DROP COLUMN IF EXISTS context_snapshot_json;
ALTER TABLE ai.decision_case DROP COLUMN IF EXISTS action_registry_hash;
ALTER TABLE ai.decision_case DROP COLUMN IF EXISTS action_registry_version;
ALTER TABLE ai.decision_case DROP COLUMN IF EXISTS alert_rules_hash;
ALTER TABLE ai.decision_case DROP COLUMN IF EXISTS alert_rules_version;

-- +goose StatementEnd
