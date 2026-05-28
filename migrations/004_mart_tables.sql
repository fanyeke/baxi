-- +goose Up
-- +goose StatementBegin

-- mart.metric_snapshot: Unified metric fact table for flexible queries.
-- Stores any metric at any grain with baseline comparison.
CREATE TABLE mart.metric_snapshot (
    snapshot_id     BIGSERIAL PRIMARY KEY,
    metric_name     TEXT NOT NULL,
    metric_value    NUMERIC(18,4),
    metric_date     DATE,
    grain           TEXT,
    dimension_type  TEXT,
    dimension_value TEXT,
    baseline_value  NUMERIC(18,4),
    delta_value     NUMERIC(18,4),
    delta_pct       NUMERIC(10,6),
    severity_hint   TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    pipeline_run_id TEXT,
    CONSTRAINT uq_metric_snapshot UNIQUE (metric_name, metric_date, grain, dimension_type, dimension_value)
);

-- mart.metric_daily: One row per day with 12 aggregate metrics.
CREATE TABLE mart.metric_daily (
    metric_date              DATE PRIMARY KEY,
    gmv                      NUMERIC(18,2),
    order_count              BIGINT,
    customer_count           BIGINT,
    seller_count             BIGINT,
    avg_order_value          NUMERIC(18,2),
    freight_value            NUMERIC(18,2),
    avg_review_score         NUMERIC(4,2),
    low_review_rate          NUMERIC(10,6),
    late_delivery_rate       NUMERIC(10,6),
    cancel_rate              NUMERIC(10,6),
    payment_installment_rate NUMERIC(10,6),
    marketing_seller_share   NUMERIC(10,6),
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- mart.metric_dimension_daily: Metrics sliced by dimension (seller, category, region).
CREATE TABLE mart.metric_dimension_daily (
    metric_date      DATE NOT NULL,
    dimension_type   TEXT NOT NULL,
    dimension_value  TEXT NOT NULL,
    metric_name      TEXT NOT NULL,
    metric_value     NUMERIC(18,4),
    sample_size      BIGINT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (metric_date, dimension_type, dimension_value, metric_name)
);

CREATE INDEX idx_metric_date ON mart.metric_daily (metric_date);
CREATE INDEX idx_metric_dim_date_type ON mart.metric_dimension_daily (metric_date, dimension_type);
CREATE INDEX idx_metric_dim_value ON mart.metric_dimension_daily (dimension_type, dimension_value);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS mart.metric_dimension_daily;
DROP TABLE IF EXISTS mart.metric_daily;
DROP TABLE IF EXISTS mart.metric_snapshot;

-- +goose StatementEnd
