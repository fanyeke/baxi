-- +goose Up
-- +goose StatementBegin

-- raw: Raw Olist CSV import tables
CREATE SCHEMA IF NOT EXISTS raw;

-- dwd: Order/item-level detail wide tables
CREATE SCHEMA IF NOT EXISTS dwd;

-- mart: Metric snapshots, daily metrics, dimension metrics
CREATE SCHEMA IF NOT EXISTS mart;

-- ops: Alerts, tasks, recommendations, outbox
CREATE SCHEMA IF NOT EXISTS ops;

-- gov: Data classification, object schemas, lineage, permissions
CREATE SCHEMA IF NOT EXISTS gov;

-- ai: Decision cases, LLM decisions, action proposals
CREATE SCHEMA IF NOT EXISTS ai;

-- audit: Pipeline runs, API logs, audit logs, error logs
CREATE SCHEMA IF NOT EXISTS audit;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP SCHEMA IF EXISTS raw CASCADE;
DROP SCHEMA IF EXISTS dwd CASCADE;
DROP SCHEMA IF EXISTS mart CASCADE;
DROP SCHEMA IF EXISTS ops CASCADE;
DROP SCHEMA IF EXISTS gov CASCADE;
DROP SCHEMA IF EXISTS ai CASCADE;
DROP SCHEMA IF EXISTS audit CASCADE;

-- +goose StatementEnd
