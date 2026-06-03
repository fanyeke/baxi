# Phase 3: Pipeline Migration Plan

> **Note:** Historical migration document. The `migration_baseline/` directory and `scripts/migration/` scripts referenced below have been removed — migration is complete. Current pipeline runs via Go CLI (`go run ./cmd/baxi-cli pipeline run`).

## 1. Overview

This document describes how the old Python + SQLite 9-step data pipeline maps to the new Go + PostgreSQL pipeline. The goal is **exact output parity**: given the same input CSV files, the Go pipeline must produce identical row counts and near-identical values in all target tables.

### Guiding Principle

This is a parity migration, not an optimization. The Go pipeline reproduces the Python pipeline's logic faithfully, including known quirks. Any deviation must be documented as an explainable difference.

### Baseline Reference

- **Python freeze tag**: `v0.5.3-python-sqlite-freeze`
- **Legacy branch**: `legacy/python-sqlite`
- **Migration branch**: `migration/go-postgres`
- **Freeze commit**: `8a0f57e`
- **SQLite schema**: `migration_baseline/sqlite_schema.sql` (16 tables)
- **Table row counts**: `migration_baseline/table_counts.json` (906,526 total rows)
- **Pipeline CSV samples**: `migration_baseline/pipeline_outputs/` (8 CSV files)

---

## 2. Old Pipeline: 9-Step Python Execution

The existing Python pipeline (`pipeline/runner.py`) defines an ordered list of 9 steps:

```python
STEPS = [
    {"name": "db_init",                                "func": step_db_init},
    {"name": "db_ingest",                              "func": step_db_ingest},
    {"name": "db_calculate_metrics",                   "func": step_db_calculate_metrics},
    {"name": "db_calculate_dimension_metrics",          "func": step_db_calculate_dimension_metrics},
    {"name": "db_rule_engine",                         "func": step_db_rule_engine},
    {"name": "db_dimensional_rule_engine",              "func": step_db_dimensional_rule_engine},
    {"name": "db_generate_recommendations",             "func": step_db_generate_recommendations},
    {"name": "db_export_feishu",                       "func": step_db_export_feishu},
    {"name": "db_trigger_simulator",                   "func": step_db_trigger_simulator},
]
```

The `run_pipeline()` function supports a `dimensional` flag. When `False` (v0.2 mode), only 6 steps execute: `db_init`, `db_ingest`, `db_calculate_metrics`, `db_rule_engine`, `db_generate_recommendations`, `db_trigger_simulator`. When `True` (v0.3 mode), all 9 steps run.

### Step-by-Step Description

| # | Step Name | Python Source | Input | Output | Purpose |
|---|---|---|---|---|---|
| 1 | `db_init` | `scripts/db_init.py` | None | SQLite schema | Creates 14 tables + indexes via `sql/schema.sql` + `sql/indexes.sql` |
| 2 | `db_ingest` | `scripts/db_ingest.py` | 9 CSV files + config YAML | `dwd_order_level` (99,441), `dwd_item_level` (112,650), `ingestion_batches`, `pipeline_runs` | Reads CSV files, builds DWD tables via SQL JOIN, records audit |
| 3 | `db_calculate_metrics` | `scripts/db_calculate_metrics.py` | `dwd_order_level`, `dwd_item_level` | `metric_daily` (634) | Aggregates 12 daily KPIs grouped by `purchase_date` |
| 4 | `db_calculate_dimension_metrics` | `scripts/db_calculate_dimension_metrics.py` | `dwd_order_level`, `dwd_item_level` | `metric_dimension_daily` (693,602) | Computes metrics sliced by seller, category, region |
| 5 | `db_rule_engine` | `scripts/db_rule_engine.py` | `metric_daily` + `config/alert_rules.yml` | `alert_events`, `strategy_recommendations`, `action_tasks`, `event_outbox` | Evaluates 3 global rules (gmv_drop, late_delivery_spike, cancel_rate_spike) |
| 6 | `db_dimensional_rule_engine` | `scripts/db_dimensional_rule_engine.py` | `metric_dimension_daily` + `config/alert_rules.yml` | `alert_events`, `event_outbox` | Evaluates 6 dimensional rules (seller/category/region variants) |
| 7 | `db_generate_recommendations` | `scripts/db_generate_recommendations.py` | `alert_events` + YAML templates | `strategy_recommendations`, `action_tasks` | Generates Chinese-language recommendations and action tasks for each alert |
| 8 | `db_export_feishu` | `scripts/db_export_feishu.py` | All pipeline tables | CSV files in `data/processed/` | Exports tables to Feishu-sync-ready CSV files |
| 9 | `db_trigger_simulator` | `scripts/db_trigger_simulator.py` | `event_outbox` | Log file + outbox status update | Simulates dispatch for pending outbox events |

---

## 3. New Pipeline: Go Stage Mapping

The Go pipeline replaces the 9 Python steps with 7 logical stages, executed by a `runner.go` orchestrator.

### Step-to-Stage Mapping

| Old Python Step | New Go Stage | Package | Key Change |
|---|---|---|---|
| 1. `db_init` | **Removed** (schema managed by goose) | — | Schema is pre-applied via goose migration files. Pipeline assumes tables exist. |
| 2. `db_ingest` | **Stage 1: Ingest Raw** | `internal/ingest/` | CSV files load into `raw.*` tables via PostgreSQL COPY. Then `raw.*` → `dwd.*` via SQL. |
| 3. `db_calculate_metrics` | **Stage 2: Build DWD** (part of) + **Stage 3: Build Metrics** | `internal/pipeline/steps/build_dwd.go`, `build_metrics.go` | DWD build and metric calculation are separated into distinct sub-stages. |
| 4. `db_calculate_dimension_metrics` | **Stage 3: Build Metrics** (dimension pass) | `internal/pipeline/steps/build_metrics.go` | Unified metric calculation handles both daily and dimension metrics. |
| 5. `db_rule_engine` | **Stage 4: Detect Alerts** (global rules) | `internal/alert/engine.go` | Reads `mart.metric_daily`, writes `ops.metric_alert` + `ops.outbox_event`. |
| 6. `db_dimensional_rule_engine` | **Stage 4: Detect Alerts** (dimensional rules) | `internal/alert/engine.go` | Reads `mart.metric_dimension_daily`, writes `ops.metric_alert` + `ops.outbox_event`. |
| 7. `db_generate_recommendations` | **Stage 5: Generate Recommendations** | `internal/recommendation/generator.go` | Reads `ops.metric_alert`, writes `ops.recommendation` + `ops.task`. |
| 8. `db_export_feishu` | **Removed** (handled by Phase 6 Outbox Worker) | — | Feishu export becomes an outbox consumer concern, not a pipeline stage. |
| 9. `db_trigger_simulator` | **Stage 6: Create Outbox** | `internal/outbox/repository.go` | Outbox events are written during alert detection and recommendation stages. A final outbox stage finalizes pending records. |

### Go Package Structure

```
internal/
  pipeline/
    runner.go       - Pipeline orchestration (step interface, sequential execution)
    step.go         - Step interface definition
    audit.go        - Pipeline run and step run audit records
    steps/
      ingest_raw.go           - Stage 1a: CSV → raw.* tables
      build_dwd.go            - Stage 1b: raw.* → dwd.* tables
      build_metrics.go        - Stage 2: dwd.* → mart.* (daily + dimension)
      detect_alerts.go        - Stage 3: mart.* → ops.metric_alert
      generate_recommendations.go - Stage 4: ops.metric_alert → ops.recommendation + ops.task
      create_outbox.go        - Stage 5: Finalize ops.outbox_event records
  ingest/
    csv_loader.go    - PostgreSQL COPY from CSV
    table_mapping.go - CSV filename → raw table name mapping
  alert/
    rule.go          - Rule definition struct + YAML loading
    engine.go        - Rule evaluation logic
  recommendation/
    generator.go     - Recommendation + task generation logic
  outbox/
    repository.go    - Outbox event CRUD
cmd/
  baxi-cli/
    main.go          - CLI entry point
    pipeline.go      - Pipeline command handler
```

### Pipeline Execution Flow

```
┌─────────────────────────────────────────────────────────────────────┐
│                         runner.go orchestration                     │
│                                                                     │
│  Stage 1a: Ingest Raw (TRUNCATE + COPY)                             │
│  └─ internal/ingest/csv_loader.go → raw.olist_* (11 tables)        │
│                                                                     │
│  Stage 1b: Build DWD (INSERT ... ON CONFLICT)                      │
│  └─ internal/pipeline/steps/build_dwd.go                            │
│     ├─ dwd.order_level (raw.orders + customers + payments + reviews)│
│     └─ dwd.item_level (raw.order_items + products + sellers + cat)  │
│                                                                     │
│  Stage 2: Build Metrics (TRUNCATE + INSERT)                         │
│  └─ internal/pipeline/steps/build_metrics.go                        │
│     ├─ mart.metric_daily (daily aggregation from dwd.order_level)   │
│     └─ mart.metric_dimension_daily (dimension slice from dwd.*)     │
│                                                                     │
│  Stage 3: Detect Alerts (INSERT ... ON CONFLICT)                    │
│  └─ internal/pipeline/steps/detect_alerts.go                        │
│     └─ internal/alert/engine.go → ops.metric_alert                  │
│                                                                     │
│  Stage 4: Generate Recommendations (INSERT ... ON CONFLICT)         │
│  └─ internal/pipeline/steps/generate_recommendations.go             │
│     └─ internal/recommendation/generator.go → ops.recommendation    │
│                                                                     │
│  Stage 5: Create Outbox (INSERT ... ON CONFLICT)                    │
│  └─ internal/pipeline/steps/create_outbox.go                        │
│     └─ internal/outbox/repository.go → ops.outbox_event             │
│                                                                     │
│  Audit: pipeline_run + pipeline_step_run written at each stage      │
│  └─ internal/pipeline/audit.go → audit.pipeline_run/step_run       │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 4. CSV → Raw Table Mapping

### 4.1 Source CSVs

The Python pipeline reads 9 CSV files from `data/raw/`. The Go pipeline reads the same files via PostgreSQL COPY, loading into 11 `raw.*` staging tables.

| # | CSV File | Raw Table | Rows (est.) | PK |
|---|---|---|---|---|
| 1 | `olist_customers_dataset.csv` | `raw.olist_customers` | 99,441 | `customer_id` |
| 2 | `olist_orders_dataset.csv` | `raw.olist_orders` | 99,441 | `order_id` |
| 3 | `olist_order_items_dataset.csv` | `raw.olist_order_items` | 112,650 | `(order_id, order_item_id)` |
| 4 | `olist_order_payments_dataset.csv` | `raw.olist_order_payments` | 103,886 | `(order_id, payment_sequential)` |
| 5 | `olist_order_reviews_dataset.csv` | `raw.olist_order_reviews` | 99,224 | `review_id` |
| 6 | `olist_products_dataset.csv` | `raw.olist_products` | 32,951 | `product_id` |
| 7 | `olist_sellers_dataset.csv` | `raw.olist_sellers` | 3,095 | `seller_id` |
| 8 | `olist_geolocation_dataset.csv` | `raw.olist_geolocation` | 1,000,163 | `id` (BIGSERIAL) |
| 9 | `product_category_name_translation.csv` | `raw.product_category_name_translation` | 71 | `product_category_name` |
| 10 | `olist_marketing_qualified_leads_dataset.csv` | `raw.marketing_qualified_leads` | ~8,000 | `mql_id` |
| 11 | `olist_closed_deals_dataset.csv` | `raw.closed_deals` | ~8,000 | `(mql_id, seller_id)` |

### 4.2 Ingestion Strategy

The Python pipeline reads CSVs into Python, transforms them in memory, and inserts directly into `dwd_order_level` and `dwd_item_level` via SQLite INSERT. The Go pipeline adds a raw staging layer:

1. **TRUNCATE** each `raw.*` table (full reload, matching Python's `DELETE FROM dwd_*` before insert)
2. **COPY** each CSV into its corresponding `raw.*` table using `pgx.CopyFrom`
3. **Track** each batch in `audit.ingestion_batch` with source filename, row count, and status

### 4.3 Column Mapping

All raw table columns map 1:1 from CSV file headers. Four unified tracking columns are appended to every raw table:

| Column | Type | Source |
|---|---|---|
| `ingested_at` | `TIMESTAMPTZ DEFAULT NOW()` | Go pipeline |
| `source_file` | `TEXT` | CSV filename |
| `source_row_number` | `BIGINT` | Row number during COPY |
| `raw_hash` | `TEXT` | MD5 hash of raw CSV row (for dedup) |

---

## 5. Raw → DWD: Order Level

### 5.1 Target Table

`dwd.order_level` — one row per order (99,441 rows expected).

### 5.2 SQL Strategy

The Python pipeline (`db_ingest.py`) builds `dwd_order_level` via a SQL JOIN across 4 raw tables. The Go pipeline uses an equivalent SQL statement:

```sql
INSERT INTO dwd.order_level (
    order_id, customer_id, customer_unique_id, order_status,
    order_purchase_timestamp, purchase_date, customer_state,
    payment_type, payment_installments, payment_value,
    review_score, delivered_customer_date, estimated_delivery_date,
    delivery_days, delay_days, is_late, is_cancelled,
    ingestion_batch_id, loaded_at, pipeline_run_id, record_hash
)
SELECT
    o.order_id,
    o.customer_id,
    c.customer_unique_id,
    o.order_status,
    o.order_purchase_timestamp,
    o.order_purchase_timestamp::DATE AS purchase_date,
    c.customer_state,
    pa.payment_type,
    pa.payment_installments,
    pa.payment_value,
    r.review_score,
    o.order_delivered_customer_date,
    o.order_estimated_delivery_date,
    -- delivery_days: calendar days between purchase and delivery
    EXTRACT(EPOCH FROM (o.order_delivered_customer_date - o.order_purchase_timestamp)) / 86400 AS delivery_days,
    -- delay_days: negative means early, positive means late
    EXTRACT(EPOCH FROM (o.order_delivered_customer_date - o.order_estimated_delivery_date)) / 86400 AS delay_days,
    -- is_late: delivered late or not yet delivered and past estimated date
    CASE WHEN o.order_delivered_customer_date > o.order_estimated_delivery_date
         OR (o.order_delivered_customer_date IS NULL AND o.order_estimated_delivery_date < NOW())
         THEN TRUE ELSE FALSE END AS is_late,
    -- is_cancelled
    CASE WHEN o.order_status = 'canceled' THEN TRUE ELSE FALSE END AS is_cancelled,
    :batch_id AS ingestion_batch_id,
    NOW() AS loaded_at,
    :pipeline_run_id AS pipeline_run_id,
    MD5(o.order_id || :pipeline_run_id) AS record_hash
FROM raw.olist_orders o
LEFT JOIN raw.olist_customers c ON o.customer_id = c.customer_id
LEFT JOIN (
    -- Payment aggregation: one row per order (first payment row)
    SELECT DISTINCT ON (order_id)
        order_id, payment_type, payment_installments, payment_value
    FROM raw.olist_order_payments
    ORDER BY order_id, payment_sequential
) pa ON o.order_id = pa.order_id
LEFT JOIN (
    -- Review aggregation: one row per order (latest review score)
    SELECT DISTINCT ON (order_id)
        order_id, review_score
    FROM raw.olist_order_reviews
    ORDER BY order_id, review_answer_timestamp DESC NULLS LAST
) r ON o.order_id = r.order_id
ON CONFLICT (order_id) DO NOTHING;
```

### 5.3 Key Translation Details

| Column | Python Logic | Go/PostgreSQL Equivalent | Parity |
|---|---|---|---|
| `delivery_days` | Python `(delivered - purchase).days` as float | `EXTRACT(EPOCH ...) / 86400` as `NUMERIC(10,6)` | Near-identical. Python returns integer days; PostgreSQL returns fractional days. |
| `delay_days` | Python `(delivered - estimated).days` | `EXTRACT(EPOCH ...) / 86400` | See above. |
| `is_late` | `delivered > estimated OR (NULL AND past_estimate)` | `CASE` expression | Exact match. |
| `is_cancelled` | `order_status == 'canceled'` | `CASE WHEN order_status = 'canceled'` | Exact match. |
| `payment_type` / `installments` / `value` | First payment row per order (ordered by `payment_sequential`) | `DISTINCT ON (order_id) ... ORDER BY payment_sequential` | Exact match. |
| `review_score` | Latest review per order (ordered by `review_answer_timestamp DESC NULLS LAST`) | `DISTINCT ON (order_id) ... ORDER BY review_answer_timestamp DESC NULLS LAST` | Exact match. |

### 5.4 `record_hash` Strategy

The Python pipeline does not compute record hashes. The Go pipeline adds `record_hash` as a `MD5(order_id || pipeline_run_id)` for change detection. This is a one-way addition; existing SQLite data does not have this field.

---

## 6. Raw → DWD: Item Level

### 6.1 Target Table

`dwd.item_level` — one row per order item (112,650 rows expected).

### 6.2 SQL Strategy

```sql
INSERT INTO dwd.item_level (
    order_id, order_item_id, product_id, seller_id,
    product_category_name, product_category_name_english,
    seller_state, price, freight_value,
    ingestion_batch_id, loaded_at, pipeline_run_id, record_hash
)
SELECT
    oi.order_id,
    oi.order_item_id,
    oi.product_id,
    oi.seller_id,
    p.product_category_name,
    COALESCE(pct.product_category_name_english, p.product_category_name) AS product_category_name_english,
    s.seller_state,
    oi.price,
    oi.freight_value,
    :batch_id AS ingestion_batch_id,
    NOW() AS loaded_at,
    :pipeline_run_id AS pipeline_run_id,
    MD5(oi.order_id || oi.order_item_id::TEXT || :pipeline_run_id) AS record_hash
FROM raw.olist_order_items oi
LEFT JOIN raw.olist_products p ON oi.product_id = p.product_id
LEFT JOIN raw.olist_sellers s ON oi.seller_id = s.seller_id
LEFT JOIN raw.product_category_name_translation pct
    ON p.product_category_name = pct.product_category_name
ON CONFLICT (order_id, order_item_id) DO NOTHING;
```

### 6.3 Key Translation Details

| Column | Python Logic | Go/PostgreSQL Equivalent | Parity |
|---|---|---|---|
| `item_key` | SQLite `TEXT` surrogate key: `order_id || '-' || order_item_id` | **Removed**. Replaced by composite PK `(order_id, order_item_id)`. | Intentional difference (Phase 2 decision). |
| `product_category_name_english` | `COALESCE(translation, original)` | `COALESCE(pct.product_category_name_english, p.product_category_name)` | Exact match. |

---

## 7. DWD → Mart: Daily Metrics

### 7.1 Target Table

`mart.metric_daily` — one row per date with 12 aggregate metrics (634 rows expected).

### 7.2 SQL Strategy

The Python script `db_calculate_metrics.py` performs: `DELETE FROM metric_daily` (full mode), then:

```sql
-- Main aggregation (full mode trivially uses min/max date range)
INSERT INTO mart.metric_daily (
    metric_date, gmv, order_count, customer_count, seller_count,
    avg_order_value, freight_value, avg_review_score,
    low_review_rate, late_delivery_rate, cancel_rate,
    payment_installment_rate, marketing_seller_share, created_at
)
SELECT
    o.purchase_date,
    ROUND(COALESCE(SUM(o.payment_value), 0)::numeric, 2) AS gmv,
    COUNT(DISTINCT o.order_id) AS order_count,
    COUNT(DISTINCT o.customer_unique_id) AS customer_count,
    COALESCE(sc.seller_count, 0) AS seller_count,
    ROUND((COALESCE(SUM(o.payment_value), 0) / NULLIF(COUNT(DISTINCT o.order_id), 0))::numeric, 2) AS avg_order_value,
    ROUND(COALESCE(SUM(i.freight_value), 0)::numeric, 2) AS freight_value,
    ROUND(COALESCE(AVG(o.review_score), 0)::numeric, 4) AS avg_review_score,
    ROUND(
        COALESCE(
            SUM(CASE WHEN o.review_score IS NOT NULL AND o.review_score <= 2 THEN 1 ELSE 0 END)::numeric
            / NULLIF(COUNT(CASE WHEN o.review_score IS NOT NULL THEN 1 END), 0), 0
        ), 4
    ) AS low_review_rate,
    ROUND(
        COALESCE(
            SUM(CASE WHEN o.order_status = 'delivered' AND o.is_late THEN 1 ELSE 0 END)::numeric
            / NULLIF(COUNT(CASE WHEN o.order_status = 'delivered' THEN 1 END), 0), 0
        ), 4
    ) AS late_delivery_rate,
    ROUND(
        COALESCE(
            SUM(CASE WHEN o.is_cancelled THEN 1 ELSE 0 END)::numeric
            / NULLIF(COUNT(o.order_id), 0), 0
        ), 4
    ) AS cancel_rate,
    ROUND(
        COALESCE(
            SUM(CASE WHEN o.payment_installments > 1 THEN 1 ELSE 0 END)::numeric
            / NULLIF(COUNT(o.order_id), 0), 0
        ), 4
    ) AS payment_installment_rate,
    0 AS marketing_seller_share,  -- placeholder: requires marketing dataset join
    NOW() AS created_at
FROM dwd.order_level o
LEFT JOIN LATERAL (
    SELECT COALESCE(SUM(i.freight_value), 0) AS freight_value
    FROM dwd.item_level i
    WHERE i.order_id = o.order_id
) i ON TRUE
LEFT JOIN LATERAL (
    SELECT COUNT(DISTINCT i2.seller_id) AS seller_count
    FROM dwd.item_level i2
    WHERE i2.order_id = o.order_id
) sc ON TRUE
WHERE o.purchase_date IS NOT NULL
  AND o.purchase_date BETWEEN :date_start AND :date_end
GROUP BY o.purchase_date, sc.seller_count
ORDER BY o.purchase_date;
```

### 7.3 Metric Formulas

| Metric | Formula | Python Rounding | Go/PostgreSQL Rounding |
|---|---|---|---|
| `gmv` | `SUM(payment_value)` | `round(x, 2)` | `ROUND(x, 2)` |
| `order_count` | `COUNT(DISTINCT order_id)` | integer | `BIGINT` |
| `customer_count` | `COUNT(DISTINCT customer_unique_id)` | integer | `BIGINT` |
| `seller_count` | Subquery from `dwd.item_level` | integer | `BIGINT` (via `LATERAL`) |
| `avg_order_value` | `SUM(payment_value) / COUNT(DISTINCT order_id)` | `round(x, 2)` | `ROUND(x, 2)` |
| `freight_value` | `SUM(item_level.freight_value)` by date | `round(x, 2)` | `ROUND(x, 2)` |
| `avg_review_score` | `AVG(review_score)` | `round(x, 4)` | `ROUND(x, 4)` |
| `low_review_rate` | `COUNT(score <= 2) / COUNT(score NOT NULL)` | `round(x, 4)` | `ROUND(x, 4)` |
| `late_delivery_rate` | `COUNT(delivered AND late) / COUNT(delivered)` | `round(x, 4)` | `ROUND(x, 4)` |
| `cancel_rate` | `COUNT(is_cancelled) / COUNT(all orders)` | `round(x, 4)` | `ROUND(x, 4)` |
| `payment_installment_rate` | `COUNT(installments > 1) / COUNT(all)` | `round(x, 4)` | `ROUND(x, 4)` |
| `marketing_seller_share` | `0` (placeholder) | `0` | `0` |

### 7.4 Date Range Strategy

- **Full mode**: `MIN(purchase_date)` to `MAX(purchase_date)`. TRUNCATE + full reload.
- **Range mode**: Specific date range. DELETE + INSERT for affected dates only.

---

## 8. DWD → Mart: Dimension Daily Metrics

### 8.1 Target Table

`mart.metric_dimension_daily` — metrics sliced by seller, category, and region (693,602 rows expected).

### 8.2 Dimension Types

| `dimension_type` | Source | `dimension_value` | Aggregation |
|---|---|---|---|
| `seller` | `dwd.item_level.seller_id` | Seller UUID | Items grouped by seller |
| `category` | `dwd.item_level.product_category_name_english` | Category name | Items grouped by category |
| `region` | `dwd.order_level.customer_state` | State code (SP, RJ, MG, etc.) | Orders grouped by state |

### 8.3 SQL Strategy

The Python pipeline (`db_calculate_dimension_metrics.py`) computes per-dimension metrics by joining `dwd.order_level` with `dwd.item_level` and grouping by dimension. The Go pipeline replicates this:

```sql
-- Seller dimension metrics
INSERT INTO mart.metric_dimension_daily (
    metric_date, dimension_type, dimension_value,
    metric_name, metric_value, sample_size, created_at
)
SELECT
    o.purchase_date,
    'seller' AS dimension_type,
    i.seller_id AS dimension_value,
    'gmv' AS metric_name,
    ROUND(SUM(i.price)::numeric, 4) AS metric_value,
    COUNT(DISTINCT o.order_id) AS sample_size,
    NOW() AS created_at
FROM dwd.order_level o
JOIN dwd.item_level i ON o.order_id = i.order_id
WHERE o.purchase_date IS NOT NULL
  AND o.purchase_date BETWEEN :date_start AND :date_end
  AND o.is_cancelled = FALSE
GROUP BY o.purchase_date, i.seller_id

UNION ALL

SELECT
    o.purchase_date,
    'seller' AS dimension_type,
    i.seller_id AS dimension_value,
    'order_count' AS metric_name,
    COUNT(DISTINCT o.order_id)::numeric AS metric_value,
    COUNT(DISTINCT o.order_id) AS sample_size,
    NOW() AS created_at
FROM dwd.order_level o
JOIN dwd.item_level i ON o.order_id = i.order_id
WHERE o.purchase_date IS NOT NULL
  AND o.purchase_date BETWEEN :date_start AND :date_end
  AND o.is_cancelled = FALSE
GROUP BY o.purchase_date, i.seller_id

-- ... (similar for avg_review_score, late_delivery_rate, cancel_rate per dimension)
ON CONFLICT (metric_date, dimension_type, dimension_value, metric_name)
DO UPDATE SET metric_value = EXCLUDED.metric_value,
              sample_size = EXCLUDED.sample_size;
```

### 8.4 Dimension Metrics Per Type

| Dimension | Metrics Computed | Python Logic |
|---|---|---|
| `seller` | `gmv`, `order_count`, `avg_review_score`, `late_delivery_rate`, `cancel_rate` | Group by `seller_id` |
| `category` | `gmv`, `order_count`, `avg_review_score`, `low_review_rate`, `late_delivery_rate` | Group by `product_category_name_english` |
| `region` | `gmv`, `order_count`, `avg_review_score`, `late_delivery_rate`, `cancel_rate` | Group by `customer_state` |

---

## 9. Mart → Ops: Alert Rule Engine

### 9.1 Rule Configuration

Rules are defined in `config/alert_rules.yml`. The Go pipeline loads the same YAML file and evaluates the same conditions.

### 9.2 Global Rules (3 Working)

| Rule ID | Metric | Condition | Threshold | Severity | Owner |
|---|---|---|---|---|---|
| `gmv_drop` | `gmv` | `current_7d_avg < prev_14d_avg * 0.85` | 15% drop | `high` | `business_ops` |
| `late_delivery_spike` | `late_delivery_rate` | `value > 0.25 AND order_count >= 20` | 25% rate, 20+ samples | `high` | `logistics_ops` |
| `cancel_rate_spike` | `cancel_rate` | `change_rate > 0.5 AND value > 0.05` | 50% increase, 5% rate | `medium` | `logistics_ops` |

### 9.3 Global Rule Evaluation Logic

The Python `db_rule_engine.py` evaluates these rules by:
1. Loading the `metric_daily` time series for the target metric
2. Computing 7-day rolling average (`last_7` days)
3. Computing 14-day rolling average (`-21 to -7` days = 14 days before the 7-day window)
4. Applying the rule-specific condition
5. Writing to `alert_events`, `strategy_recommendations`, `action_tasks`, and `event_outbox`

The Go `internal/alert/engine.go` replicates this:

```go
type RollingWindow struct {
    Current7dAvg  float64 // avg of last 7 days
    Prev14dAvg    float64 // avg of 14 days before the 7-day window
    CurrentValue  float64 // latest single-day value
    TotalSamples  int     // total data points
    Window7dCount int     // 7 (or fewer if insufficient data)
    Window14dCount int    // 14 (or fewer if insufficient data)
}

func computeWindows(series []MetricPoint) RollingWindow {
    n := len(series)
    if n < 7 {
        return RollingWindow{TotalSamples: n}
    }
    // 7-day avg: last 7 points
    last7 := series[n-7:]
    avg7 := average(last7)
    // 14-day avg: 14 points before the last 7
    if n >= 21 {
        prev14 := series[n-21 : n-7]
        avg14 := average(prev14)
        return RollingWindow{...}
    }
    // ... handle partial windows
}
```

### 9.4 Dimensional Rules (6 Working)

| Rule ID | Dimension | Condition | Threshold | Severity | Owner |
|---|---|---|---|---|---|
| `seller_late_delivery_spike` | seller | `value > 0.25 AND order_count >= 20` | 25% late, 20+ orders | `high` | `seller_ops` |
| `seller_review_score_drop` | seller | `value < 3.5 AND order_count >= 20` | **DEAD RULE** | `medium` | `seller_ops` |
| `category_gmv_drop` | category | `change_rate < -0.20 AND order_count >= 30` | 20% GMV drop, 30+ orders | `medium` | `category_ops` |
| `category_low_review_cluster` | category | `avg_review_score < 3.0 AND order_count >= 20` | **DEAD RULE** | `medium` | `category_ops` |
| `region_cancel_rate_spike` | region | `value > 0.05 AND order_count >= 30` | 5% cancel, 30+ orders | `medium` | `logistics_ops` |
| `region_late_delivery_spike` | region | `value > 0.20 AND order_count >= 30` | 20% late, 30+ orders | `high` | `logistics_ops` |

### 9.5 Dimensional Rule Evaluation

The Python `db_dimensional_rule_engine.py` reads `metric_dimension_daily` and evaluates each rule per dimension value (seller UUID, category name, state code). The Go pipeline replicates this:

```go
func evaluateDimensionRule(rule Rule, series []MetricPoint, dimensionValue string) *Alert {
    // Same rolling window logic as global rules, but applied to
    // dimension-specific time series (filtered by dimension_type + dimension_value)
    windows := computeWindows(series)
    // ... condition matching
}
```

### 9.6 Idempotency

Both pipelines use idempotent inserts:
- **Python**: `SELECT 1 FROM alert_events WHERE event_id = ?` before INSERT
- **Go**: `INSERT INTO ops.metric_alert (...) VALUES (...) ON CONFLICT (alert_id) DO NOTHING`

---

## 10. Ops: Recommendation and Task Generation

### 10.1 Recommendation Generation

The Python `db_generate_recommendations.py` reads `alert_events` rows that do not yet have a corresponding `strategy_recommendations` entry, then generates:

1. **Strategy recommendation**: Uses YAML templates from `config/action_registry.yml` and `config/action_templates.yml` to render Chinese-language strategy descriptions
2. **Action task**: Created for each recommendation with priority, due date, and owner role

The Go `internal/recommendation/generator.go` replicates this:

```go
type RecommendationGenerator struct {
    templates   map[string]ActionTemplate // loaded from YAML
    ownerMapping map[string]string        // loaded from YAML
}

func (g *RecommendationGenerator) Generate(ctx context.Context, alert ops.MetricAlert) error {
    // 1. Determine rule_id → template_key mapping
    templateKey := ruleToTemplate[alert.RuleID]
    // 2. Look up template from YAML
    tpl := g.templates[templateKey]
    // 3. Render template with alert values
    title := renderTemplate(tpl.StrategyTitle, alert)
    detail := renderTemplate(tpl.StrategyDetailTemplate, alert)
    // 4. INSERT INTO ops.recommendation
    // 5. INSERT INTO ops.task (if not exists)
}
```

### 10.2 Template Rule Mapping

```go
var ruleToTemplate = map[string]string{
    "seller_late_delivery_spike": "investigate_seller_delivery",
    "seller_review_score_drop":   "investigate_seller_review",
    "category_gmv_drop":          "investigate_category_gmv",
    "category_low_review_cluster":"investigate_category_review",
    "region_cancel_rate_spike":   "investigate_region_cancel",
    "region_late_delivery_spike": "investigate_region_delivery",
}
```

### 10.3 Confidence Calculation

```go
func computeConfidence(sampleSize, minSampleSize int) string {
    switch {
    case sampleSize > minSampleSize*2:
        return "high"
    case sampleSize > minSampleSize:
        return "medium"
    default:
        return "low"
    }
}
```

---

## 11. Outbox Event Generation

### 11.1 Event Types

Outbox events are generated during two pipeline stages:

| Source Stage | `event_type` | `source_type` | `target_channel` |
|---|---|---|---|
| Global Rule Engine | `alert` | `rule_engine` | `local_cli` |
| Dimensional Rule Engine | `dimensional_alert` | `dimensional_rule_engine` | `feishu_cli` / `manual` / `local_cli` |
| Recommendation Generator | `recommendation` | `recommendation_generator` | `local_cli` |
| Task Generator | `task_created` | `task_generator` | `local_cli` |

### 11.2 Channel Assignment Logic

The Python pipeline assigns target channels based on rule_id and severity:

| Condition | Channel |
|---|---|
| Global rule alert | `local_cli` |
| Dimensional with `severity=high` | `feishu_cli` |
| Dimensional `region_cancel_rate_spike` or low severity | `manual` |
| Dimensional `seller_review_score_drop` | `local_cli` |

The Go pipeline replicates this in `internal/outbox/repository.go`:

```go
func resolveTargetChannel(alert ops.MetricAlert) string {
    if alert.ObjectType == "global" {
        return "local_cli"
    }
    switch {
    case alert.RuleID == "region_cancel_rate_spike":
        return "manual"
    case alert.Severity == "high":
        return "feishu_cli"
    default:
        return "local_cli"
    }
}
```

### 11.3 Payload Composition

Outbox payloads are JSONB documents with fields matching the Python output:

```json
{
  "rule_id": "gmv_drop",
  "event_id": "gmv_drop_2018-10-17",
  "metric_name": "gmv",
  "current_value": 89.71,
  "baseline_value": 1991.9836,
  "change_rate": -0.9252,
  "severity": "high",
  "owner_role": "business_ops"
}
```

---

## 12. Baseline Comparison Method

### 12.1 Verification Tiers

Three tiers of verification ensure parity:

#### Tier 1: Row Count Parity

Compare row counts between SQLite baseline and PostgreSQL target tables.

```bash
# Python side (baseline)
python3 -c "
import json
with open('migration_baseline/table_counts.json') as f:
    counts = json.load(f)
    for k, v in counts.items():
        print(f'{k}: {v}')
"

# PostgreSQL side (after Go pipeline run)
psql "$DATABASE_URL" -c "
SELECT 'dwd_order_level' as tbl, COUNT(*) FROM dwd.order_level
UNION ALL
SELECT 'dwd_item_level', COUNT(*) FROM dwd.item_level
UNION ALL
SELECT 'metric_daily', COUNT(*) FROM mart.metric_daily
UNION ALL
SELECT 'metric_dimension_daily', COUNT(*) FROM mart.metric_dimension_daily
UNION ALL
SELECT 'metric_alert', COUNT(*) FROM ops.metric_alert
UNION ALL
SELECT 'recommendation', COUNT(*) FROM ops.recommendation
UNION ALL
SELECT 'task', COUNT(*) FROM ops.task
UNION ALL
SELECT 'outbox_event', COUNT(*) FROM ops.outbox_event
"
```

**Expected counts**:

| Table | Baseline Rows | Tolerance |
|---|---|---|
| `dwd.order_level` | 99,441 | Exact |
| `dwd.item_level` | 112,650 | Exact |
| `mart.metric_daily` | 634 | Exact |
| `mart.metric_dimension_daily` | 693,602 | Exact |
| `ops.metric_alert` | 36 | Exact |
| `ops.recommendation` | 36 | Exact |
| `ops.task` | 36 | Exact |
| `ops.outbox_event` | 36 | Exact |

#### Tier 2: CSV Sample Comparison

For each pipeline output table, compare sorted CSV samples:

```bash
# Export from PostgreSQL
psql "$DATABASE_URL" -c "\COPY (SELECT * FROM mart.metric_daily ORDER BY metric_date) TO '/tmp/pg_metric_daily.csv' CSV HEADER"

# Diff with baseline
python3 scripts/migration/compare_csv.py \
  --baseline migration_baseline/pipeline_outputs/metric_daily_sample.csv \
  --actual /tmp/pg_metric_daily.csv \
  --key metric_date \
  --float-tolerance 1e-9 \
  --ignore-columns created_at,pipeline_run_id,record_hash
```

#### Tier 3: Value-Level Verification

For float comparisons, the Python verification script (`scripts/migration/compare_csv.py`) uses:

| Metric Type | Tolerance | Rule |
|---|---|---|
| Monetary (gmv, price, freight) | Relative 1e-9 | `abs(a-b) <= max(1e-9*abs(a), 1e-9*abs(b))` |
| Ratio/rate (change_rate, delivery_rate) | Relative 1e-9 | Same as above |
| Score (review_score) | Absolute 1e-6 | `abs(a-b) <= 1e-6` |
| Count (order_count, sample_size) | Exact | `a == b` |
| Boolean (is_late, is_cancelled) | Exact | `a == b` |

### 12.2 Explainable Differences

The following fields are expected to differ between Python and Go outputs:

| Field | Reason for Difference | Mitigation |
|---|---|---|
| `created_at` | Different pipeline execution timestamps | Exclude from comparison |
| `pipeline_run_id` | Different run IDs (Go generates new UUIDs) | Exclude from comparison |
| `record_hash` | New field in Go pipeline | Not present in baseline |
| `ingested_at` / `loaded_at` | Different `NOW()` values | Exclude from comparison |
| `updated_at` | New tracking column in PostgreSQL | Not present in baseline |
| JSON key order | Python `json.dumps` vs Go `json.Marshal` | Compare parsed JSON, not raw strings |
| Float precision | Python `float` vs PostgreSQL `NUMERIC` | Tolerance 1e-9 relative, 1e-6 absolute |
| `item_key` | Removed in Phase 2 (composite PK replaces it) | Not present in PostgreSQL |
| `event_id` → `alert_id` | Column renamed in `ops.metric_alert` | Map in comparison script |
| `outbox_id` → `event_id` | Column renamed in `ops.outbox_event` | Map in comparison script |
| Fractional `delivery_days` / `delay_days` | PostgreSQL computes fractional days vs Python integer days | Tolerance-based comparison |

---

## 13. Known Limitations

### 13.1 Dead Rules (2)

The following rules exist in `config/alert_rules.yml` with status `active` but produce no alerts in the current dataset. They are preserved for structural completeness but are not expected to generate output.

| Rule ID | Dimension | Reason for Inactivity |
|---|---|---|
| `review_score_drop` | seller | Threshold (`avg_review_score < 3.5`) is too strict. Most sellers have scores above 3.5. The rule is structurally correct but produces no alerts in the 2016-2018 dataset. |
| `seller_activation_gap` | seller | This rule checks for sellers with no recent orders. The current `metric_dimension_daily` calculation only includes dates where a seller has orders, so there are no gaps to detect. |

**Pipeline impact**: The Go pipeline must include these rules in the dimensional engine for structural parity, but verification tests should not expect output from them.

### 13.2 Empty Tables (4)

The following tables have schema but zero rows in the baseline and are expected to remain empty:

| Table | Schema | Baseline Rows |
|---|---|---|
| `governance_checkpoints` → `gov.governance_checkpoint` | Migration 007 | 0 |
| `governance_health_results` → `gov.health_check_result` | Migration 007 | 0 |
| `review_retro` → `ops.review_retro` | Migration 005 | 0 |
| `qoder_jobs` → `ops.qoder_job` | Migration 005 | 0 |

These tables are populated by other systems (Go governance loaders, Qoder agent), not the Phase 3 pipeline.

### 13.3 Placeholder Metrics

- `marketing_seller_share` in `mart.metric_daily` is always `0` in the baseline. This metric requires the Marketing Funnel dataset (marketing_qualified_leads + closed_deals), which is not yet integrated into the pipeline. The Go pipeline preserves this as `0`.

---

## 14. Idempotency and Transaction Strategy

### 14.1 Idempotent Operations

| Stage | Strategy | SQLite Equivalent |
|---|---|---|
| Raw ingest | `TRUNCATE` + `COPY` (full reload) | `DELETE FROM dwd_*` |
| DWD build | `INSERT ... ON CONFLICT DO NOTHING` | `INSERT OR IGNORE` |
| Metric daily | `TRUNCATE` + `INSERT` (full), or `DELETE` + `INSERT` (range) | `DELETE FROM metric_daily` + `INSERT` |
| Metric dimension daily | `TRUNCATE` + `INSERT` (full), or `DELETE` + `INSERT` (range) | `DELETE FROM metric_dimension_daily` + `INSERT` |
| Alert detection | `INSERT ... ON CONFLICT DO NOTHING` | `SELECT 1 WHERE EXISTS` guard |
| Recommendation | `INSERT ... ON CONFLICT DO NOTHING` | `SELECT 1 WHERE EXISTS` guard |
| Outbox | `INSERT ... ON CONFLICT DO NOTHING` | `SELECT 1 WHERE EXISTS` guard |

### 14.2 Transaction Boundaries

Each stage runs in its own PostgreSQL transaction. The pipeline runner commits after each successful stage and rolls back on failure, writing a `failed` status to `audit.pipeline_step_run`.

```go
type StepResult struct {
    StepName string
    Status   string // "completed", "failed"
    Duration time.Duration
    Error    error
}

func (r *Runner) Execute(ctx context.Context) bool {
    for _, step := range r.steps {
        tx, _ := r.pool.Begin(ctx)
        err := step.Execute(ctx, tx)
        if err != nil {
            tx.Rollback(ctx)
            r.recordStepRun(step.Name(), "failed", err)
            return false
        }
        tx.Commit(ctx)
        r.recordStepRun(step.Name(), "completed", nil)
    }
    return true
}
```

---

## 15. Testing Strategy

### 15.1 Unit Tests

| Package | Focus | Framework |
|---|---|---|
| `internal/alert/` | Rule evaluation logic, window computation | `testing` + testcontainers-go |
| `internal/recommendation/` | Template rendering, confidence calculation | `testing` |
| `internal/ingest/` | CSV parsing, table mapping | `testing` |
| `internal/outbox/` | Payload composition, channel resolution | `testing` |

### 15.2 Integration Tests

Use `testcontainers-go` to spin up a PostgreSQL container for each test run:

```go
func TestPipelineEndToEnd(t *testing.T) {
    postgres, err := testcontainers.RunContainer(
        ctx,
        testcontainers.WithImage("postgres:16-alpine"),
        postgres.WithDatabase("baxi_test"),
    )
    // Run migrations
    // Copy test CSVs
    // Execute pipeline
    // Assert row counts match baseline
    // Assert metric values within tolerance
}
```

### 15.3 Comparison Script

A Python comparison script (`scripts/migration/compare_csv.py`) lives outside the Go codebase to perform independent verification:

```bash
python3 scripts/migration/compare_csv.py \
  --baseline-dir migration_baseline/pipeline_outputs/ \
  --actual-dir /tmp/pipeline_outputs/ \
  --float-tolerance 1e-9 \
  --ignore-columns created_at,pipeline_run_id,record_hash
```

---

## 16. Rollout Plan

| Phase | Action | Verification |
|---|---|---|
| 3a | Raw ingest + DWD build | Row count parity for `dwd.*` (99,441 + 112,650) |
| 3b | Metric calculation (daily + dimension) | Row count parity for `mart.*` (634 + 693,602) + value tolerance |
| 3c | Alert engine (global + dimensional) | Row count parity for `ops.metric_alert` (36) |
| 3d | Recommendation + task generation | Row count parity for `ops.recommendation`, `ops.task`, `ops.outbox_event` (36 each) |
| 3e | Full end-to-end pipeline | All tiers pass (row count + CSV sample + value tolerance) |

---

## Appendix A: SQLite-to-PostgreSQL Type Mapping for Pipeline Tables

| SQLite Column Type | PostgreSQL Column Type | Used In |
|---|---|---|
| `TEXT` (PK, IDs) | `TEXT` | All PK columns |
| `TEXT` (datetime) | `TIMESTAMPTZ` | `created_at`, `loaded_at`, `processed_at` |
| `TEXT` (date) | `DATE` | `metric_date`, `purchase_date`, `event_date` |
| `REAL` (monetary) | `NUMERIC(18,2)` | `gmv`, `payment_value`, `price`, `freight_value` |
| `REAL` (rate/ratio) | `NUMERIC(10,6)` | `change_rate`, `delivery_days`, `delay_days` |
| `REAL` (score) | `NUMERIC(4,2)` | `review_score` |
| `REAL` (metric) | `NUMERIC(18,4)` | `metric_value`, `current_value`, `baseline_value` |
| `INTEGER` (count) | `BIGINT` | `order_count`, `sample_size`, `payment_installments` |
| `INTEGER` (0/1) | `BOOLEAN` | `is_late`, `is_cancelled` |
| `TEXT` (JSON) | `JSONB` | `evidence_json`, `payload_json` |

---

## Appendix B: Pipeline Run Records

The Python pipeline writes to `pipeline_runs` (12 rows in baseline) and `ingestion_batches` (6 rows). The Go pipeline writes to:

- `audit.pipeline_run` — one row per pipeline execution
- `audit.pipeline_step_run` — one row per completed/failed step within a pipeline run
- `audit.ingestion_batch` — one row per CSV ingestion batch

**Baseline row counts**:

| Table | Rows | Notes |
|---|---|---|
| `audit.pipeline_run` (SQLite: `pipeline_runs`) | 12 | Each pipeline invocation |
| `audit.ingestion_batch` (SQLite: `ingestion_batches`) | 6 | CSV ingestion events |
| `audit.pipeline_step_run` | NEW | Not present in SQLite baseline |

The `audit.pipeline_step_run` table is a new addition that provides per-step execution tracking. It has no baseline equivalent and is excluded from parity comparison.

---

## Appendix C: Go Pipeline Configuration

The Go pipeline reads the following external configurations (same files as Python):

| Config File | Format | Loaded By | Purpose |
|---|---|---|---|
| `config/alert_rules.yml` | YAML | `internal/alert/rule.go` | Rule definitions and thresholds |
| `config/action_registry.yml` | YAML | `internal/recommendation/generator.go` | Global action templates |
| `config/action_templates.yml` | YAML | `internal/recommendation/generator.go` | Dimensional action templates |
| `config/owner_mapping.yml` | YAML | `internal/recommendation/generator.go` | Role-to-owner mapping |

---

## Appendix D: CSV Data Files

The Go pipeline ingests all 9 Olist CSV files plus 2 Marketing Funnel CSVs:

```bash
data/raw/
├── olist_customers_dataset.csv
├── olist_orders_dataset.csv
├── olist_order_items_dataset.csv
├── olist_order_payments_dataset.csv
├── olist_order_reviews_dataset.csv
├── olist_products_dataset.csv
├── olist_sellers_dataset.csv
├── olist_geolocation_dataset.csv
├── product_category_name_translation.csv
├── olist_marketing_qualified_leads_dataset.csv
└── olist_closed_deals_dataset.csv
```

The first 9 CSVs are actively ingested by the pipeline. The Marketing Funnel CSVs (10-11) are staged in `raw.*` tables but not yet consumed by DWD or metric calculations (`marketing_seller_share` remains `0`).
