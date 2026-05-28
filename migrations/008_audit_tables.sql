-- +goose Up
-- +goose StatementBegin

-- audit.pipeline_run: Pipeline execution runs
-- Source: SQLite pipeline_runs (12 rows)
CREATE TABLE audit.pipeline_run (
    run_id        TEXT PRIMARY KEY,
    run_type      TEXT NOT NULL,
    mode          TEXT NOT NULL,
    status        TEXT NOT NULL,
    started_at    TIMESTAMPTZ NOT NULL,
    finished_at   TIMESTAMPTZ,
    input_count   BIGINT DEFAULT 0,
    output_count  BIGINT DEFAULT 0,
    error_message TEXT
);

-- audit.ingestion_batch: Data ingestion batches
-- Source: SQLite ingestion_batches (6 rows)
CREATE TABLE audit.ingestion_batch (
    batch_id       TEXT PRIMARY KEY,
    source_name    TEXT NOT NULL,
    ingestion_mode TEXT NOT NULL,
    date_start     DATE,
    date_end       DATE,
    source_file    TEXT,
    row_count      BIGINT DEFAULT 0,
    status         TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- audit.pipeline_step_run: Per-step pipeline tracking
CREATE TABLE audit.pipeline_step_run (
    step_run_id    TEXT PRIMARY KEY,
    pipeline_run_id TEXT,
    step_name      TEXT,
    step_order     BIGINT,
    status         TEXT,
    started_at     TIMESTAMPTZ,
    finished_at    TIMESTAMPTZ,
    input_count    BIGINT,
    output_count   BIGINT,
    error_message  TEXT
);

-- audit.api_request_log: API access logging
CREATE TABLE audit.api_request_log (
    log_id            BIGSERIAL PRIMARY KEY,
    request_id        TEXT,
    method            TEXT,
    path              TEXT,
    status_code       BIGINT,
    user_agent        TEXT,
    client_ip         TEXT,
    request_body_json  JSONB,
    response_body_json JSONB,
    duration_ms       BIGINT,
    created_at        TIMESTAMPTZ DEFAULT NOW()
);

-- audit.audit_log: Business audit trail
CREATE TABLE audit.audit_log (
    audit_id      BIGSERIAL PRIMARY KEY,
    category      TEXT,
    action        TEXT,
    actor         TEXT,
    resource_type TEXT,
    resource_id   TEXT,
    metadata      JSONB,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

-- audit.error_log: Error tracking
CREATE TABLE audit.error_log (
    error_id      BIGSERIAL PRIMARY KEY,
    request_id    TEXT,
    error_type    TEXT,
    error_message TEXT,
    stack_trace   TEXT,
    details       JSONB,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for query performance
CREATE INDEX idx_audit_pipeline_run_status  ON audit.pipeline_run (status, started_at);
CREATE INDEX idx_audit_pipeline_step_run    ON audit.pipeline_step_run (pipeline_run_id);
CREATE INDEX idx_audit_api_request_log      ON audit.api_request_log (request_id);
CREATE INDEX idx_audit_log_category         ON audit.audit_log (category, created_at);
CREATE INDEX idx_audit_error_log            ON audit.error_log (request_id, created_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS audit.idx_audit_error_log;
DROP INDEX IF EXISTS audit.idx_audit_log_category;
DROP INDEX IF EXISTS audit.idx_audit_api_request_log;
DROP INDEX IF EXISTS audit.idx_audit_pipeline_step_run;
DROP INDEX IF EXISTS audit.idx_audit_pipeline_run_status;

DROP TABLE IF EXISTS audit.error_log;
DROP TABLE IF EXISTS audit.audit_log;
DROP TABLE IF EXISTS audit.api_request_log;
DROP TABLE IF EXISTS audit.pipeline_step_run;
DROP TABLE IF EXISTS audit.ingestion_batch;
DROP TABLE IF EXISTS audit.pipeline_run;

-- +goose StatementEnd
