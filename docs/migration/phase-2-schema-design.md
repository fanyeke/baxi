# Phase 2: PostgreSQL Schema Design

## 1. Introduction & Scope

### Objective

Phase 2 designs the PostgreSQL layered schema that replaces the existing SQLite database (`data/olist_ops.db`) and establishes the foundation for the Go + PostgreSQL pipeline. The design covers 7 logical schemas (raw, dwd, mart, ops, gov, ai, audit) housing 42 tables total: 16 mapped from existing SQLite tables, 11 raw Olist staging tables, and 15 new design tables created for future phases.

### What Phase 2 Covers

- Schema design documentation (this document)
- 7 goose migration files (migrations 002 through 008) containing DDL for all tables
- Schema verification script for SQLite-to-PostgreSQL parity checks
- Proper PostgreSQL type mappings replacing SQLite flexible types
- Indexes matching the 25 existing SQLite indexes
- Foreign key constraints from SQLite migration 010, with ON DELETE SET NULL

### What Phase 2 Does NOT Cover

- NO data import from SQLite or CSV files
- NO Go struct or domain model generation
- NO pipeline migration (Phase 3)
- NO LLM integration (Phase 7)
- NO triggers, views, functions, or stored procedures
- NO table partitioning
- NO UUID migration (deferred to Phase 4 or 5)
- NO changes to Python/FastAPI/React code
- NO data migration or ETL operations

### Source of Truth

The exclusive source of truth for existing SQLite table definitions is:

- **`migration_baseline/sqlite_schema.sql`** — 16 table definitions with all columns from migrations 003 through 007 applied
- **NOT `sql/schema.sql`** — this file is outdated v0.2, missing 2 tables and 10+ columns

Additional references:
- `migration_baseline/table_counts.json` — 906,526 total rows across 16 tables
- `sql/indexes.sql` — 25 existing indexes
- `sql/migrations/010_add_foreign_keys.sql` — Foreign key constraints
- `migration_baseline/configs_snapshot/` — 28 YAML config files for governance context

---

## 2. SQLite to PostgreSQL Table Mapping Matrix

The following table maps all 16 existing SQLite tables to their PostgreSQL schema and table names. Row counts are from `migration_baseline/table_counts.json`.

| SQLite Table | Rows | PostgreSQL Schema | PostgreSQL Table | Notes |
|---|---|---|---|---|
| dwd_order_level | 99,441 | dwd | order_level | TEXT PK preserved. purchase_date becomes DATE. is_late, is_cancelled become BOOLEAN. |
| dwd_item_level | 112,650 | dwd | item_level | item_key removed. New composite PK: (order_id, order_item_id). |
| metric_daily | 634 | mart | metric_daily | metric_date becomes DATE. Monetary fields become NUMERIC(18,2). |
| metric_dimension_daily | 693,602 | mart | metric_dimension_daily | 4-column composite PK preserved: (metric_date, dimension_type, dimension_value, metric_name). |
| alert_events | 36 | ops | metric_alert | event_id becomes alert_id. evidence_json becomes JSONB. |
| strategy_recommendations | 36 | ops | recommendation | TEXT PK preserved. requires_approval becomes BOOLEAN. |
| action_tasks | 36 | ops | task | TEXT PK preserved. Fields reordered logically. |
| event_outbox | 36 | ops | outbox_event | outbox_id becomes event_id. payload_json becomes JSONB. |
| review_retro | 0 | ops | review_retro | is_effective, promote_to_rule become BOOLEAN. status and feedback columns included from migration 007. |
| qoder_jobs | 0 | ops | qoder_job | job_context_json becomes JSONB. |
| governance_checkpoints | 0 | gov | governance_checkpoint | checkpoint_id becomes BIGINT GENERATED ALWAYS AS IDENTITY. metadata_json becomes JSONB. |
| governance_health_results | 0 | gov | health_check_result | result_id becomes BIGINT GENERATED ALWAYS AS IDENTITY. |
| pipeline_runs | 12 | audit | pipeline_run | input_count, output_count become BIGINT. |
| ingestion_batches | 6 | audit | ingestion_batch | row_count becomes BIGINT. |
| qoder_runs | 19 | ai | qoder_run | can_apply becomes BOOLEAN. |
| qoder_reports | 18 | ai | qoder_report | findings_json, recommended_human_actions_json, used_endpoints_json become JSONB. no_apply_performed, business_side_effect become BOOLEAN. |

### Notable Changes from SQLite

1. **item_key removed**: `dwd_item_level` had an SQLite-specific TEXT surrogate key. PostgreSQL uses the natural composite PK `(order_id, order_item_id)` instead.
2. **event_id renamed**: `alert_events.event_id` becomes `ops.metric_alert.alert_id` for semantic clarity.
3. **outbox_id renamed**: `event_outbox.outbox_id` becomes `ops.outbox_event.event_id` for consistency.
4. **TEXT dates become proper types**: All date strings become DATE or TIMESTAMPTZ.
5. **INTEGER flags become BOOLEAN**: 10 boolean fields across multiple tables.

---

## 3. 7-Schema Layered Design

### 3.1 `raw` — Raw Staging Layer

**Purpose**: Raw ingestion staging for Olist CSV files. Each CSV file gets its own table with column names matching the original CSV. No transformations, no constraints between tables. Represents the system of record for imported data.

**Tables**:

| Table | Source CSV | Rows (est.) | Description |
|---|---|---|---|
| raw.olist_customers | olist_customers_dataset.csv | 99,441 | Customer master with geolocation |
| raw.olist_orders | olist_orders_dataset.csv | 99,441 | Order header with timestamps |
| raw.olist_order_items | olist_order_items_dataset.csv | 112,650 | Line items per order |
| raw.olist_order_payments | olist_order_payments_dataset.csv | 103,886 | Payment split records |
| raw.olist_order_reviews | olist_order_reviews_dataset.csv | 99,224 | Customer reviews |
| raw.olist_products | olist_products_dataset.csv | 32,951 | Product catalog |
| raw.olist_sellers | olist_sellers_dataset.csv | 3,095 | Seller master |
| raw.olist_geolocation | olist_geolocation_dataset.csv | 1,000,163 | Zip code coordinates |
| raw.product_category_name_translation | product_category_name_translation.csv | 71 | Portuguese-to-English category mapping |
| raw.marketing_qualified_leads | olist_marketing_qualified_leads_dataset.csv | ~8,000 | Marketing qualified leads |
| raw.closed_deals | olist_closed_deals_dataset.csv | ~8,000 | Closed deal conversions |

**Key characteristics**:
- TEXT type for all Olist string IDs (no type coercion)
- NUMERIC(18,2) for monetary values, NUMERIC(10,6) for coordinates
- Every table has 4 unified ingestion tracking columns: `ingested_at`, `source_file`, `source_row_number`, `raw_hash`
- No foreign keys between raw tables
- No indexes on raw tables (added in Phase 3 after data load)

**Phase 3 dependency**: The CSV import pipeline writes to `raw.*` tables and reads from them to build `dwd.*`.

### 3.2 `dwd` — Detail Wide Data Layer

**Purpose**: Denormalized detail tables for orders and items. One row per order (`order_level`) and one row per item (`item_level`). These are the primary input tables for metric calculation and analysis.

**Tables**:

| Table | SQLite Source | Description |
|---|---|---|
| dwd.order_level | dwd_order_level | Order-level detail with customer, payment, review, delivery info |
| dwd.item_level | dwd_item_level | Item-level detail with product, seller, category info |

**Key characteristics**:
- Composite PK on `dwd.item_level(order_id, order_item_id)` replacing SQLite `item_key`
- Proper typed columns (TIMESTAMPTZ for timestamps, DATE for purchase_date, BOOLEAN for flags)
- New tracking columns: `created_at`, `updated_at`, `pipeline_run_id`, `record_hash`

**Phase 3 dependency**: All metric calculation (daily, dimension) reads from `dwd.*`. Pipeline reads `dwd.order_level` and `dwd.item_level` as inputs.

### 3.3 `mart` — Metrics and Aggregation Layer

**Purpose**: Pre-computed metrics at daily and dimension grain. Stores aggregated results that drive alerting, dashboards, and decision recommendations.

**Tables**:

| Table | SQLite Source | Rows | Description |
|---|---|---|---|
| mart.metric_daily | metric_daily | 634 | One row per day with 12 aggregate metrics |
| mart.metric_dimension_daily | metric_dimension_daily | 693,602 | Metrics sliced by dimension (seller, category, region) |
| mart.metric_snapshot | NEW | 0 | Unified metric fact table for flexible queries |

**Key characteristics**:
- `metric_daily` has DATE PK (one metric row per day)
- `metric_dimension_daily` preserves the 4-column composite PK
- `metric_snapshot` is a new table that unifies metric storage with grain, dimension, and baseline tracking
- Monetary metrics are NUMERIC(18,2). Ratios and rates are NUMERIC(10,6). Scores are NUMERIC(4,2).

**Phase 3 dependency**: Pipeline writes metric calculation results here. Downstream alert rules read from these tables.

### 3.4 `ops` — Operations Workflow Layer

**Purpose**: Captures the operational decision pipeline output: alerts generated from metric anomalies, strategy recommendations, action tasks, outbound event dispatch, and human review. This is the core workflow layer.

**Tables**:

| Table | SQLite Source | Rows | Description |
|---|---|---|---|
| ops.metric_alert | alert_events | 36 | Anomaly alerts generated by rule engine |
| ops.recommendation | strategy_recommendations | 36 | Strategic recommendations from heuristic/LLM |
| ops.task | action_tasks | 36 | Action items assigned to roles/users |
| ops.outbox_event | event_outbox | 36 | Outbound dispatch queue (Outbox pattern) |
| ops.dispatch_attempt | NEW | 0 | Per-attempt dispatch tracking |
| ops.qoder_job | qoder_jobs | 0 | External Qoder agent job tracking |
| ops.review_retro | review_retro | 0 | Post-execution retrospection records |

**Key characteristics**:
- TEXT PKs for all migrated tables (matching SQLite IDs)
- JSONB for semi-structured payloads (evidence_json, payload_json, job_context_json)
- Foreign keys recreate SQLite migration 010 constraints with ON DELETE SET NULL
- `dispatch_attempt` is new: tracks each dispatch try with response and error info
- Indexes on status, severity, owner_role for operational queries

**Phase 3 dependency**: Pipeline writes alert/recommendation/task/outbox records. Phase 6 (Outbox Worker) reads and processes pending events.

### 3.5 `gov` — Governance Layer

**Purpose**: Stores governance metadata about data assets, classification, lineage, access control, and health checks. This layer is primarily configuration-driven, populated by loading YAML governance configs at startup or via API.

**Tables**:

| Table | SQLite Source | Rows | Description |
|---|---|---|---|
| gov.config_snapshot | NEW | 0 | YAML config snapshots with content_hash dedup |
| gov.object_schema | NEW | 0 | Ontology object definitions (customer, order, seller...) |
| gov.data_classification | NEW | 0 | Classification rules per asset and field |
| gov.data_lineage | NEW | 0 | Column-level lineage tracking |
| gov.access_policy | NEW | 0 | Role-based access control policies |
| gov.health_check_result | governance_health_results | 0 | System health check results |
| gov.governance_checkpoint | governance_checkpoints | 0 | Governance action audit records |

**Key characteristics**:
- New tables use BIGSERIAL PKs (reference data, not operational IDs)
- `config_snapshot` stores YAML configs as JSONB with content dedup via hash
- `object_schema` stores ontology definitions (from `aip_object_schema.yml`)
- `data_classification` stores classification metadata (from `data_classification.yml`)
- `data_lineage` stores column-level lineage (from `data_lineage.yml`)
- `access_policy` stores RBAC rules (from `access_policy.yml`)
- Migrated tables use `GENERATED ALWAYS AS IDENTITY` (SQL standard replacement for AUTOINCREMENT)
- JSONB for flexible metadata (schema_jsonb, conditions_jsonb, content_jsonb)

**Phase 3 dependency**: Governance tables are populated by config loaders in Phase 5. Phase 3 pipeline does not directly depend on gov tables.

### 3.6 `ai` — AI Decision Layer

**Purpose**: Stores AI/LLM execution artifacts: decision cases (alert + context), structured LLM outputs, proposed actions, and human review records. Supports the LLM decision workflow.

**Tables**:

| Table | SQLite Source | Rows | Description |
|---|---|---|---|
| ai.qoder_run | qoder_runs | 19 | Qoder agent execution runs |
| ai.qoder_report | qoder_reports | 18 | Qoder agent analysis reports |
| ai.decision_case | NEW | 0 | Alert + context aggregation for LLM |
| ai.llm_decision | NEW | 0 | Structured LLM decision output |
| ai.action_proposal | NEW | 0 | Proposed actions from LLM |
| ai.review_record | NEW | 0 | Human review of AI proposals |

**Key characteristics**:
- TEXT PKs for all tables (generated by Go application)
- JSONB for flexible AI outputs (findings, recommendations, context, output)
- BOOLEAN for AI flags (can_apply, no_apply_performed, business_side_effect)
- Indexes on status + created_at for workflow queries

**Phase 3 dependency**: AI tables are populated by Phase 7 (LLM Decision Layer). Phase 3 pipeline does not directly depend on ai tables.

### 3.7 `audit` — Audit and Observability Layer

**Purpose**: Captures all execution metadata: pipeline runs, ingestion batches, API requests, business audit trails, and error logs. This is the system's observability foundation.

**Tables**:

| Table | SQLite Source | Rows | Description |
|---|---|---|---|
| audit.pipeline_run | pipeline_runs | 12 | Pipeline execution runs |
| audit.ingestion_batch | ingestion_batches | 6 | Data ingestion batches |
| audit.pipeline_step_run | NEW | 0 | Per-step pipeline tracking |
| audit.api_request_log | NEW | 0 | API access logging |
| audit.audit_log | NEW | 0 | Business audit trail |
| audit.error_log | NEW | 0 | Error tracking |

**Key characteristics**:
- Migrated tables use TEXT PKs; new log tables use BIGSERIAL (append-only logs)
- JSONB for request/response bodies and error details
- DATE for date_start/date_end (not TIMESTAMPTZ, these are calendar date ranges)
- Indexes on pipeline run status, request_id, category+created_at

**Phase 3 dependency**: Pipeline writes pipeline_run and pipeline_step_run records during execution. Phase 4 (API migration) writes api_request_log records.

---

## 4. New Table Designs

### 4.1 `raw.*` — 11 Olist Staging Tables

Each raw table mirrors its CSV source with Olist-native column names and types. Four unified ingestion tracking columns are appended to every table.

| Column | Type | Purpose |
|---|---|---|
| ingested_at | TIMESTAMPTZ NOT NULL DEFAULT NOW() | When this row was imported |
| source_file | TEXT | Original CSV filename |
| source_row_number | BIGINT | Row number in original CSV |
| raw_hash | TEXT | Hash of raw CSV row for dedup |

**raw.olist_customers**: customer_id (TEXT PK), customer_unique_id (TEXT), customer_zip_code_prefix (TEXT), customer_city (TEXT), customer_state (TEXT), plus ingestion tracking.

**raw.olist_orders**: order_id (TEXT PK), customer_id (TEXT), order_status (TEXT), order_purchase_timestamp (TIMESTAMPTZ), order_approved_at (TIMESTAMPTZ), order_delivered_carrier_date (TIMESTAMPTZ), order_delivered_customer_date (TIMESTAMPTZ), order_estimated_delivery_date (TIMESTAMPTZ), plus ingestion tracking.

**raw.olist_order_items**: order_id (TEXT), order_item_id (BIGINT), product_id (TEXT), seller_id (TEXT), shipping_limit_date (TIMESTAMPTZ), price (NUMERIC(18,2)), freight_value (NUMERIC(18,2)), plus ingestion tracking. PK: (order_id, order_item_id).

**raw.olist_order_payments**: order_id (TEXT), payment_sequential (BIGINT), payment_type (TEXT), payment_installments (BIGINT), payment_value (NUMERIC(18,2)), plus ingestion tracking. PK: (order_id, payment_sequential).

**raw.olist_order_reviews**: review_id (TEXT PK), order_id (TEXT), review_score (NUMERIC(4,2)), review_comment_title (TEXT), review_comment_message (TEXT), review_creation_date (TIMESTAMPTZ), review_answer_timestamp (TIMESTAMPTZ), plus ingestion tracking.

**raw.olist_products**: product_id (TEXT PK), product_category_name (TEXT), product_name_lenght (BIGINT), product_description_lenght (BIGINT), product_photos_qty (BIGINT), product_weight_g (NUMERIC(18,2)), product_length_cm (NUMERIC(18,2)), product_height_cm (NUMERIC(18,2)), product_width_cm (NUMERIC(18,2)), plus ingestion tracking.

**raw.olist_sellers**: seller_id (TEXT PK), seller_zip_code_prefix (TEXT), seller_city (TEXT), seller_state (TEXT), plus ingestion tracking.

**raw.olist_geolocation**: geolocation_zip_code_prefix (TEXT), geolocation_lat (NUMERIC(10,6)), geolocation_lng (NUMERIC(10,6)), geolocation_city (TEXT), geolocation_state (TEXT), plus ingestion tracking. PK: BIGSERIAL (no natural unique key due to duplicate lat/lng entries).

**raw.product_category_name_translation**: product_category_name (TEXT PK), product_category_name_english (TEXT), plus ingestion tracking.

**raw.marketing_qualified_leads**: mql_id (TEXT PK), first_contact_date (TIMESTAMPTZ), landing_page_id (TEXT), origin (TEXT), plus ingestion tracking.

**raw.closed_deals**: mql_id (TEXT), seller_id (TEXT), sdr_id (TEXT), sr_id (TEXT), won_date (TIMESTAMPTZ), business_segment (TEXT), lead_type (TEXT), lead_behaviour_profile (TEXT), has_company (TEXT), has_gtin (TEXT), average_stock (TEXT), business_type (TEXT), declared_product_catalog_size (TEXT), declared_monthly_revenue (TEXT), plus ingestion tracking. PK: (mql_id, seller_id).

### 4.2 `mart.metric_snapshot` — Unified Metric Fact Table

New table for flexible metric storage. Captures any metric at any grain with baseline comparison.

| Column | Type | Description |
|---|---|---|
| snapshot_id | BIGSERIAL | Auto-increment PK |
| metric_name | TEXT NOT NULL | Metric identifier (gmv, order_count, review_score...) |
| metric_value | NUMERIC(18,4) | Current metric value |
| metric_date | DATE | Date of metric |
| grain | TEXT | Aggregation grain (daily, seller, category, region) |
| dimension_type | TEXT | Dimension type when grained (seller, category, state) |
| dimension_value | TEXT | Dimension value when grained |
| baseline_value | NUMERIC(18,4) | Historical baseline for comparison |
| delta_value | NUMERIC(18,4) | Absolute change from baseline |
| delta_pct | NUMERIC(10,6) | Percentage change from baseline |
| severity_hint | TEXT | Severity indicator (normal, warning, critical) |
| created_at | TIMESTAMPTZ DEFAULT NOW() | Record creation time |
| pipeline_run_id | TEXT | Reference to pipeline run that produced this |

Unique constraint: (metric_name, metric_date, grain, dimension_type, dimension_value).

### 4.3 `ops.dispatch_attempt` — Per-Attempt Dispatch Tracking

New table for tracking each outbox dispatch attempt with response data.

| Column | Type | Description |
|---|---|---|
| attempt_id | BIGSERIAL | Auto-increment PK |
| event_id | TEXT NOT NULL | Reference to outbox_event |
| attempt_number | BIGINT | Attempt sequence number |
| status | TEXT | Dispatch status (success, failed, retry) |
| dispatched_at | TIMESTAMPTZ | When the dispatch was attempted |
| response_json | JSONB | Raw response from target channel |
| error_message | TEXT | Error details if failed |
| created_at | TIMESTAMPTZ DEFAULT NOW() | Record creation time |

### 4.4 `gov.config_snapshot` — YAML Config Snapshots

Stores governance YAML configs for version tracking and runtime lookup.

| Column | Type | Description |
|---|---|---|
| snapshot_id | BIGSERIAL | Auto-increment PK |
| config_key | TEXT NOT NULL | Config identifier (alert_rules, access_policy...) |
| config_type | TEXT | Type classification |
| source_path | TEXT | Original file path in config/ |
| content_jsonb | JSONB | Full YAML content as JSON |
| content_hash | TEXT | Hash for dedup detection |
| loaded_at | TIMESTAMPTZ DEFAULT NOW() | When this snapshot was loaded |

Unique constraint: (config_key, content_hash).

### 4.5 `gov.object_schema` — Ontology Object Definitions

Stores ontology definitions from `aip_object_schema.yml`.

| Column | Type | Description |
|---|---|---|
| object_schema_id | BIGSERIAL | Auto-increment PK |
| object_type | TEXT NOT NULL | Object type (customer, order, seller, product...) |
| object_name | TEXT | Display name |
| schema_jsonb | JSONB | Full property/relationship definitions |
| version | TEXT | Schema version |
| created_at | TIMESTAMPTZ DEFAULT NOW() | Record creation time |

Unique constraint: (object_type, version).

### 4.6 `gov.data_classification` — Data Classification Rules

Stores field-level classification from `data_classification.yml`.

| Column | Type | Description |
|---|---|---|
| classification_id | BIGSERIAL | Auto-increment PK |
| field_path | TEXT NOT NULL | Asset/field reference (e.g., raw_customers.customer_unique_id) |
| classification_level | TEXT | Classification level (pii, sensitive, internal, public) |
| sensitivity_score | NUMERIC(4,2) | Optional numeric sensitivity score |
| description | TEXT | Rationale for classification |
| created_at | TIMESTAMPTZ DEFAULT NOW() | Record creation time |

### 4.7 `gov.data_lineage` — Data Lineage Records

Stores column-level lineage from `data_lineage.yml`.

| Column | Type | Description |
|---|---|---|
| lineage_id | BIGSERIAL | Auto-increment PK |
| source_table | TEXT | Source table name |
| source_column | TEXT | Source column name |
| target_table | TEXT | Target table name |
| target_column | TEXT | Target column name |
| transformation_logic | TEXT | Description of transformation |
| confidence | NUMERIC(4,2) | Lineage confidence score |
| created_at | TIMESTAMPTZ DEFAULT NOW() | Record creation time |

### 4.8 `gov.access_policy` — Access Control Policies

Stores RBAC policies from `access_policy.yml`.

| Column | Type | Description |
|---|---|---|
| policy_id | BIGSERIAL | Auto-increment PK |
| policy_name | TEXT NOT NULL | Policy identifier |
| resource_type | TEXT | Type of resource (table, api, field) |
| resource_pattern | TEXT | Resource pattern for matching |
| action | TEXT | Allowed action (read, write, admin) |
| principal_type | TEXT | Principal type (role, user) |
| principal_pattern | TEXT | Principal pattern for matching |
| effect | TEXT | Policy effect (allow, deny) |
| conditions_jsonb | JSONB | Additional conditions |
| created_at | TIMESTAMPTZ DEFAULT NOW() | Record creation time |

### 4.9 `ai.decision_case` — Alert + Context Aggregation

Aggregates alert data with context for LLM consumption.

| Column | Type | Description |
|---|---|---|
| case_id | TEXT PK | Unique case identifier (Go-generated) |
| alert_id | TEXT | Source alert reference |
| case_type | TEXT | Case classification |
| status | TEXT DEFAULT 'open' | Workflow status (open, in_review, resolved) |
| context_json | JSONB | Aggregated context (metrics, trends, history) |
| created_at | TIMESTAMPTZ DEFAULT NOW() | When the case was created |
| resolved_at | TIMESTAMPTZ | When the case was resolved |

### 4.10 `ai.llm_decision` — Structured LLM Decision Output

Captures structured output from LLM decision calls.

| Column | Type | Description |
|---|---|---|
| decision_id | TEXT PK | Unique decision identifier (Go-generated) |
| case_id | TEXT | Reference to decision_case |
| model_version | TEXT | LLM model identifier |
| prompt_hash | TEXT | Hash of input prompt for reproducibility |
| output_json | JSONB | Structured LLM response |
| confidence | NUMERIC(4,2) | Model confidence score |
| created_at | TIMESTAMPTZ DEFAULT NOW() | Decision timestamp |

### 4.11 `ai.action_proposal` — Proposed Actions from LLM

Stores actionable proposals extracted from LLM decisions.

| Column | Type | Description |
|---|---|---|
| proposal_id | TEXT PK | Unique proposal identifier (Go-generated) |
| case_id | TEXT | Reference to decision_case |
| decision_id | TEXT | Reference to llm_decision |
| action_type | TEXT | Type of action (notify, recommend, create_task) |
| payload | JSONB | Action parameters and context |
| apply_status | TEXT DEFAULT 'pending' | Execution status (pending, applied, rejected) |
| created_at | TIMESTAMPTZ DEFAULT NOW() | Creation timestamp |
| applied_at | TIMESTAMPTZ | When the action was applied |
| applied_by | TEXT | Who applied the action |

### 4.12 `ai.review_record` — Human Review Records

Captures human feedback on AI-generated proposals.

| Column | Type | Description |
|---|---|---|
| review_id | TEXT PK | Unique review identifier (Go-generated) |
| proposal_id | TEXT | Reference to action_proposal |
| reviewer_id | TEXT | Who performed the review |
| verdict | TEXT | Review outcome (approved, rejected, modified) |
| feedback | TEXT | Human feedback text |
| reviewed_at | TIMESTAMPTZ DEFAULT NOW() | Review timestamp |

### 4.13 `audit.pipeline_step_run` — Per-Step Pipeline Tracking

Tracks individual steps within a pipeline run.

| Column | Type | Description |
|---|---|---|
| step_run_id | TEXT PK | Unique step run identifier (Go-generated) |
| pipeline_run_id | TEXT | Reference to pipeline_run |
| step_name | TEXT | Step name (csv_ingest, dwd_build, metric_calc...) |
| step_order | BIGINT | Execution order within pipeline |
| status | TEXT | Step status (running, completed, failed) |
| started_at | TIMESTAMPTZ | When the step started |
| finished_at | TIMESTAMPTZ | When the step finished |
| input_count | BIGINT | Rows/records processed as input |
| output_count | BIGINT | Rows/records produced as output |
| error_message | TEXT | Error details if failed |

### 4.14 `audit.api_request_log` — API Access Logging

Captures all API requests for audit and debugging.

| Column | Type | Description |
|---|---|---|
| log_id | BIGSERIAL | Auto-increment PK |
| request_id | TEXT | Correlation ID for the request |
| method | TEXT | HTTP method |
| path | TEXT | Request path |
| status_code | BIGINT | HTTP response status |
| user_agent | TEXT | Client user agent |
| client_ip | TEXT | Client IP address |
| request_body_json | JSONB | Request body snapshot |
| response_body_json | JSONB | Response body snapshot |
| duration_ms | BIGINT | Request duration in milliseconds |
| created_at | TIMESTAMPTZ DEFAULT NOW() | Request timestamp |

### 4.15 `audit.audit_log` — Business Audit Trail

Captures business-level audit events.

| Column | Type | Description |
|---|---|---|
| audit_id | BIGSERIAL | Auto-increment PK |
| category | TEXT | Audit category (dispatch, governance, decision) |
| action | TEXT | Audit action (created, updated, dispatched, reviewed) |
| actor | TEXT | Who performed the action |
| resource_type | TEXT | Type of resource affected |
| resource_id | TEXT | Identifier of resource affected |
| metadata | JSONB | Additional audit context |
| created_at | TIMESTAMPTZ DEFAULT NOW() | Audit timestamp |

### 4.16 `audit.error_log` — Error Tracking

Captures application errors for monitoring and debugging.

| Column | Type | Description |
|---|---|---|
| error_id | BIGSERIAL | Auto-increment PK |
| request_id | TEXT | Correlation ID for the request |
| error_type | TEXT | Error classification |
| error_message | TEXT | Human-readable error message |
| stack_trace | TEXT | Stack trace for debugging |
| details | JSONB | Additional error context |
| created_at | TIMESTAMPTZ DEFAULT NOW() | Error timestamp |

---

## 5. Field Type Conversion Strategy

The following table documents the SQLite-to-PostgreSQL type mapping applied across all migrated tables.

| SQLite Type | PostgreSQL Type | Examples |
|---|---|---|
| TEXT (identifier) | TEXT | order_id, customer_id, product_id, seller_id, task_id |
| TEXT (status/enum) | TEXT | order_status, payment_type, severity, status, mode |
| TEXT (name/description) | TEXT | strategy_title, strategy_detail, description, feedback |
| INTEGER (count) | BIGINT | payment_installments, sample_size, order_count, dispatch_attempts |
| INTEGER (sequential) | BIGINT | order_item_id, payment_sequential, attempt_number |
| INTEGER (0/1 boolean) | BOOLEAN | is_late, is_cancelled, requires_approval, is_effective, promote_to_rule, can_apply, no_apply_performed, business_side_effect |
| INTEGER (AUTOINCREMENT PK) | BIGINT GENERATED ALWAYS AS IDENTITY | checkpoint_id, result_id |
| REAL (monetary) | NUMERIC(18,2) | price, freight_value, payment_value, gmv, avg_order_value |
| REAL (ratios/rates) | NUMERIC(10,6) | change_rate, delivery_days, delay_days, low_review_rate, late_delivery_rate, cancel_rate, payment_installment_rate, marketing_seller_share, delta_pct |
| REAL (scores) | NUMERIC(4,2) | review_score, confidence, impact_score, sensitivity_score |
| REAL (metrics) | NUMERIC(18,4) | metric_value, baseline_value, current_value, delta_value |
| REAL (coordinates) | NUMERIC(10,6) | geolocation_lat, geolocation_lng |
| TEXT (date only) | DATE | metric_date, purchase_date, event_date, date_start, date_end |
| TEXT (datetime) | TIMESTAMPTZ | created_at, loaded_at, started_at, finished_at, order_purchase_timestamp, delivered_customer_date |
| TEXT (JSON) | JSONB | evidence_json, payload_json, context_json, output_json, findings_json |
| TEXT (hash/ref) | TEXT | ingestion_batch_id, external_ref, request_id, record_hash, content_hash |

### Default Value Translations

| SQLite Default | PostgreSQL Default |
|---|---|
| `datetime('now')` | `NOW()` |
| `0` (for INTEGER count) | `0` (for BIGINT count) |
| `'new'` / `'pending'` / `'draft'` / `'todo'` | `'new'` / `'pending'` / `'draft'` / `'todo'` (unchanged TEXT) |
| `0` (for INTEGER boolean) | `FALSE` (for BOOLEAN) |
| `1` (for INTEGER boolean true) | `TRUE` (for BOOLEAN) |
| `'heuristic_strategy'` | `'heuristic_strategy'` (unchanged TEXT) |
| `'medium'` | `'medium'` (unchanged TEXT) |
| `'global'` | `'global'` (unchanged TEXT) |
| `'simulated'` | `'simulated'` (unchanged TEXT) |
| `'hindsight_rule'` | `'hindsight_rule'` (unchanged TEXT) |
| `'dry_run'` | `'dry_run'` (unchanged TEXT) |
| `'recorded'` | `'recorded'` (unchanged TEXT) |
| `'qoder'` | `'qoder'` (unchanged TEXT) |
| `'read_only'` | `'read_only'` (unchanged TEXT) |

---

## 6. Primary Key Strategy

### Guiding Principles

1. **Preserve SQLite TEXT PKs** for all migrated tables. IDs like `order_id`, `task_id`, `event_id` are already meaningful business identifiers. Changing them to UUID or BIGSERIAL would break API contracts and cross-references.
2. **New operational tables** use TEXT PKs generated by Go application code (e.g., `decision_case.case_id`, `action_proposal.proposal_id`).
3. **New reference / log tables** use BIGSERIAL auto-increment PKs (e.g., `gov.config_snapshot`, `audit.api_request_log`, `ops.dispatch_attempt`). These are append-only tables where the PK has no business meaning.
4. **Composite PKs preserved** from SQLite where they represent natural keys.
5. **No UUID in Phase 2**. UUID migration is deferred to Phase 4 or 5 if needed.

### PK Summary by Table

| PostgreSQL Table | PK Strategy | PK Column(s) |
|---|---|---|
| raw.olist_customers | TEXT | customer_id |
| raw.olist_orders | TEXT | order_id |
| raw.olist_order_items | Composite TEXT + BIGINT | (order_id, order_item_id) |
| raw.olist_order_payments | Composite TEXT + BIGINT | (order_id, payment_sequential) |
| raw.olist_order_reviews | TEXT | review_id |
| raw.olist_products | TEXT | product_id |
| raw.olist_sellers | TEXT | seller_id |
| raw.olist_geolocation | BIGSERIAL | geolocation_id (no natural PK) |
| raw.product_category_name_translation | TEXT | product_category_name |
| raw.marketing_qualified_leads | TEXT | mql_id |
| raw.closed_deals | Composite TEXT | (mql_id, seller_id) |
| dwd.order_level | TEXT | order_id |
| dwd.item_level | Composite TEXT + BIGINT | (order_id, order_item_id) |
| mart.metric_daily | DATE | metric_date |
| mart.metric_dimension_daily | Composite | (metric_date, dimension_type, dimension_value, metric_name) |
| mart.metric_snapshot | BIGSERIAL | snapshot_id |
| ops.metric_alert | TEXT | alert_id |
| ops.recommendation | TEXT | recommendation_id |
| ops.task | TEXT | task_id |
| ops.outbox_event | TEXT | event_id |
| ops.dispatch_attempt | BIGSERIAL | attempt_id |
| ops.qoder_job | TEXT | job_id |
| ops.review_retro | TEXT | review_id |
| gov.config_snapshot | BIGSERIAL | snapshot_id |
| gov.object_schema | BIGSERIAL | object_schema_id |
| gov.data_classification | BIGSERIAL | classification_id |
| gov.data_lineage | BIGSERIAL | lineage_id |
| gov.access_policy | BIGSERIAL | policy_id |
| gov.health_check_result | GENERATED ALWAYS AS IDENTITY | result_id |
| gov.governance_checkpoint | GENERATED ALWAYS AS IDENTITY | checkpoint_id |
| ai.qoder_run | TEXT | run_id |
| ai.qoder_report | TEXT | report_id |
| ai.decision_case | TEXT (Go-generated) | case_id |
| ai.llm_decision | TEXT (Go-generated) | decision_id |
| ai.action_proposal | TEXT (Go-generated) | proposal_id |
| ai.review_record | TEXT (Go-generated) | review_id |
| audit.pipeline_run | TEXT | run_id |
| audit.ingestion_batch | TEXT | batch_id |
| audit.pipeline_step_run | TEXT (Go-generated) | step_run_id |
| audit.api_request_log | BIGSERIAL | log_id |
| audit.audit_log | BIGSERIAL | audit_id |
| audit.error_log | BIGSERIAL | error_id |

### AUTOINCREMENT Migration

SQLite tables `governance_checkpoints` and `governance_health_results` use `INTEGER PRIMARY KEY AUTOINCREMENT`. In PostgreSQL, this becomes:

```sql
checkpoint_id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY
```

This is the SQL-standard way to create auto-incrementing columns and is compatible with Go's pgx driver.

---

## 7. Index Strategy

The following indexes are created in Phase 2, mapped from the 25 existing SQLite indexes in `sql/indexes.sql` with additions for new tables.

### dwd schema

| Table | Index | Columns | Purpose |
|---|---|---|---|
| dwd.order_level | idx_dwd_order_purchase_date | purchase_date | Date-range metric filtering |
| dwd.order_level | idx_dwd_order_customer | customer_unique_id | Customer analysis |
| dwd.order_level | idx_dwd_order_status | order_status | Status-based queries |
| dwd.order_level | idx_dwd_order_batch | ingestion_batch_id | Batch tracking |
| dwd.order_level | idx_dwd_order_cancelled | is_cancelled | Cancellation analysis |
| dwd.order_level | idx_dwd_order_late | is_late | Late delivery analysis |
| dwd.item_level | idx_dwd_item_order | order_id | FK-style lookups |
| dwd.item_level | idx_dwd_item_seller | seller_id | Seller performance |
| dwd.item_level | idx_dwd_item_category | product_category_name | Category analysis |
| dwd.item_level | idx_dwd_item_batch | ingestion_batch_id | Batch tracking |

### mart schema

| Table | Index | Columns | Purpose |
|---|---|---|---|
| mart.metric_daily | idx_metric_date | metric_date | Rolling window queries |
| mart.metric_dimension_daily | idx_metric_dim_date_type | metric_date, dimension_type | Date + dimension filtering |
| mart.metric_dimension_daily | idx_metric_dim_value | dimension_type, dimension_value | Dimension-specific lookups |

Indexes on `metric_dimension_daily` (693,602 rows) are critical for performance. The 4-column composite PK already covers (metric_date, dimension_type, dimension_value, metric_name) lookups. Additional indexes cover the most common query patterns: date-range filtering by dimension, and dimension-specific aggregation.

### ops schema

| Table | Index | Columns | Purpose |
|---|---|---|---|
| ops.metric_alert | idx_alert_event_date | event_date | Date-range alert queries |
| ops.metric_alert | idx_alert_rule | rule_id | Rule-specific analysis |
| ops.metric_alert | idx_alert_severity | severity | Severity-based filtering |
| ops.metric_alert | idx_alert_status | status | Status-based workflows |
| ops.metric_alert | idx_alert_owner | owner_role | Owner assignment queries |
| ops.recommendation | idx_strategy_event | alert_id | Alert-to-recommendation lookup |
| ops.recommendation | idx_strategy_status | execution_status | Status-based filtering |
| ops.recommendation | idx_strategy_source | decision_source | Source analysis |
| ops.task | idx_action_status | status | Status-based task queries |
| ops.task | idx_action_priority | priority | Priority-based sorting |
| ops.task | idx_action_owner | owner_role | Owner assignment queries |
| ops.task | idx_action_due | due_at | Due-date monitoring |
| ops.outbox_event | idx_outbox_status_channel | status, target_channel | Queue polling by channel |
| ops.outbox_event | idx_outbox_status | status | Queue polling |
| ops.dispatch_attempt | idx_dispatch_event | event_id | Event-to-attempt lookup |
| ops.qoder_job | idx_qoder_status | dispatch_status | Status-based job tracking |
| ops.qoder_job | idx_qoder_channel | dispatch_channel | Channel-specific queries |
| ops.review_retro | idx_retro_recommendation | recommendation_id | Recommendation retro lookup |

### gov schema

| Table | Index | Columns | Purpose |
|---|---|---|---|
| gov.config_snapshot | idx_config_key | config_key | Config key lookups |

### ai schema

| Table | Index | Columns | Purpose |
|---|---|---|---|
| ai.decision_case | idx_case_status | status, created_at | Case workflow queries |
| ai.decision_case | idx_case_alert | alert_id | Alert-to-case lookup |
| ai.llm_decision | idx_decision_case | case_id | Case-to-decision lookup |
| ai.action_proposal | idx_proposal_case_status | case_id, apply_status | Proposal workflow queries |
| ai.review_record | idx_review_proposal | proposal_id | Proposal-to-review lookup |

### audit schema

| Table | Index | Columns | Purpose |
|---|---|---|---|
| audit.pipeline_run | idx_pipeline_type | run_type | Type-based queries |
| audit.pipeline_run | idx_pipeline_status | status | Status-based queries |
| audit.ingestion_batch | idx_ingest_source | source_name | Source-based queries |
| audit.ingestion_batch | idx_ingest_status | status | Status-based queries |
| audit.pipeline_step_run | idx_step_pipeline | pipeline_run_id | Pipeline-to-step lookup |
| audit.api_request_log | idx_api_request | request_id | Request correlation |
| audit.audit_log | idx_audit_category_time | category, created_at | Audit trail queries |
| audit.error_log | idx_error_request_time | request_id, created_at | Error correlation |

### Index Design Notes

- **Raw tables have no indexes**: Phase 3 will add indexes after data load when query patterns are known.
- **Existing indexes preserved**: All 25 indexes from `sql/indexes.sql` are recreated in PostgreSQL, mapped to new table names.
- **New indexes on ops tables**: Operational queries (status filtering, dispatch polling, owner assignment) need indexes for the system to function at scale.
- **Pipeline run indexes**: These tables are small (12-36 rows) in the current system but will grow significantly with scheduled execution.

---

## 8. Foreign Key Strategy

### Constraints from SQLite Migration 010

The three foreign key constraints from `sql/migrations/010_add_foreign_keys.sql` are recreated with `ON DELETE SET NULL` instead of SQLite's default `NO ACTION`.

| Constraint | Source | Target | On Delete |
|---|---|---|---|
| FK_task_recommendation | ops.task.recommendation_id | ops.recommendation.recommendation_id | SET NULL |
| FK_qoderjob_alert | ops.qoder_job.trigger_event_id | ops.metric_alert.alert_id | SET NULL |
| FK_retro_recommendation | ops.review_retro.recommendation_id | ops.recommendation.recommendation_id | SET NULL |

### FK Policy by Schema

| Schema | FK Policy | Rationale |
|---|---|---|
| raw | NO FKs | Raw staging tables are independent; no referential guarantees needed |
| dwd | NO FKs to raw | Avoid circular dependencies; dwd tables loaded independently |
| mart | NO FKs | Metric tables are write-once, read-many; FK overhead not justified |
| ops | YES (3 FKs from migration 010) | Operational workflow integrity requires referential consistency |
| gov | NO FKs | Governance tables are configuration-loaded, not transactionally linked |
| ai | NO FKs in Phase 2 | FK relationships (case→alert, decision→case) will be added when LLM logic is implemented (Phase 7) |
| audit | NO FKs | Audit tables are append-only logs; FK overhead not justified |

### Rationale for ON DELETE SET NULL

- Alerts, recommendations, and tasks can be deleted independently (e.g., cleanup of stale alerts).
- SET NULL preserves the dependent records for historical analysis without requiring the parent to exist.
- This matches the current SQLite behavior where FKs are declared but rarely enforced at the application level.

---

## 9. JSONB Usage Strategy

JSONB is used for semi-structured data where the schema is variable, evolves frequently, or is defined externally (e.g., LLM outputs, YAML configs, API payloads).

### JSONB Columns by Table

| Table | JSONB Column | Content | Rationale |
|---|---|---|---|
| ops.metric_alert | evidence_json | Alert evidence context | Variable structure per alert rule |
| ops.outbox_event | payload_json | Dispatch payload | Varies by event type and channel |
| ops.dispatch_attempt | response_json | Channel response | Unpredictable channel response format |
| ops.qoder_job | job_context_json | Qoder job context | Variable per job type |
| gov.config_snapshot | content_jsonb | YAML config content | Arbitrary YAML structure |
| gov.governance_checkpoint | metadata_json | Checkpoint metadata | Variable per action type |
| gov.object_schema | schema_jsonb | Object property definitions | Schema structure varies by object type |
| gov.access_policy | conditions_jsonb | Policy conditions | Variable condition expressions |
| ai.qoder_report | findings_json | Analysis findings | LLM output, no fixed schema |
| ai.qoder_report | recommended_human_actions_json | Suggested actions | Variable per analysis type |
| ai.qoder_report | used_endpoints_json | API endpoints used | Variable endpoint lists |
| ai.decision_case | context_json | Aggregated context | Varies by case type |
| ai.llm_decision | output_json | Structured LLM output | Schema depends on LLM prompt |
| ai.action_proposal | payload | Action parameters | Variable per action type |
| audit.api_request_log | request_body_json | HTTP request body | Variable per endpoint |
| audit.api_request_log | response_body_json | HTTP response body | Variable per endpoint |
| audit.audit_log | metadata | Business audit context | Variable per audit category |
| audit.error_log | details | Error details | Variable per error type |

### When NOT to Use JSONB

- **Core metrics** (gmv, order_count, review_score): These are well-defined structured fields that benefit from typed columns, constraints, and index support.
- **Monetary values** (price, freight_value, payment_value): Must be NUMERIC(18,2) for precise arithmetic and aggregation.
- **Status fields** (status, severity, mode): These are enums with fixed values; TEXT columns with application-level validation.
- **Identifiers** (order_id, customer_id, task_id): Must be TEXT for PK/FK relationships and index performance.
- **Boolean flags** (is_late, is_cancelled, can_apply): Distinct semantics deserve dedicated BOOLEAN columns.

### Benefits of JSONB in This Design

1. **LLM output flexibility**: AI layers produce variable-output JSON. JSONB avoids schema migration churn.
2. **Config governance**: YAML configs have arbitrary depth. JSONB preserves the full structure without flattening.
3. **Audit completeness**: Request/response bodies and error details are captured without needing to parse.
4. **Query capability**: JSONB supports indexing (GIN) and path queries when needed in later phases.

---

## 10. Guardrails (Must NOT Have)

| Rule | Rationale |
|---|---|
| NO SQL DDL in this document | Design document is prose-only. DDL belongs in migration files. |
| NO Go struct or model generation | Domain models will be designed in Phase 3 using the schema as reference. |
| NO data import from SQLite or CSV | Schema only. Data migration is a separate phase with its own verification. |
| NO triggers, views, functions, or procedures | Business logic belongs in Go application code, not the database. |
| NO table partitioning | Partitioning adds operational complexity. Not justified at current data volume (<1M rows). |
| NO CHECK constraints beyond SQLite parity | Application-layer validation is preferred; the Go codebase is the validation source of truth. |
| NO changing TEXT PKs to UUID | Preserving SQLite string IDs maintains API compatibility. UUID migration deferred. |
| NO adding columns "for future use" | Every column must map to an existing SQLite column or have a documented purpose in this design. |
| NO modifying Python, FastAPI, or React code | Phase 2 is schema-only. Pipeline and API migration happen in Phase 3 and 4. |
| NO modifying YAML governance config semantics | Config files remain as-is. The gov schema is designed to ingest them without changes. |
| NO seed or sample data | Migrations create empty tables. Data loading is a separate operation. |
| NO database users, roles, or permissions | Access control is handled by the Go application layer, not PostgreSQL roles. |

---

## 11. Phase 3 Pipeline Dependencies

The following table documents which tables Phase 3 (Pipeline Migration) reads from and writes to.

### Phase 3 Read Sources

| Table | Read By | Purpose |
|---|---|---|
| raw.olist_customers | CSV ingest step | Source for customer dimensions |
| raw.olist_orders | CSV ingest step | Source for order headers |
| raw.olist_order_items | CSV ingest step | Source for line items |
| raw.olist_order_payments | CSV ingest step | Source for payment data |
| raw.olist_order_reviews | CSV ingest step | Source for review scores |
| raw.olist_products | CSV ingest step | Source for product catalog |
| raw.olist_sellers | CSV ingest step | Source for seller master |
| raw.olist_geolocation | CSV ingest step | Source for location data |
| raw.product_category_name_translation | CSV ingest step | Source for category names |
| raw.marketing_qualified_leads | CSV ingest step | Source for MQL data |
| raw.closed_deals | CSV ingest step | Source for closed deals |
| dwd.order_level | Metric calculation step | Input for daily metrics and dimension metrics |
| dwd.item_level | Metric calculation step | Input for daily metrics and dimension metrics |

### Phase 3 Write Targets

| Table | Written By | Purpose |
|---|---|---|
| dwd.order_level | DWD build step | Denormalized order-level data |
| dwd.item_level | DWD build step | Denormalized item-level data |
| mart.metric_daily | Metric calculation step | Daily aggregate metrics |
| mart.metric_dimension_daily | Dimension calculation step | Dimension-level metrics |
| mart.metric_snapshot | Metric calculation step | Unified metric records |
| ops.metric_alert | Alert rule engine | Anomaly detection output |
| ops.recommendation | Recommendation generator | Strategic recommendations |
| ops.task | Task generator | Action items |
| ops.outbox_event | Outbox writer | Dispatch queue |
| ops.qoder_job | Qoder trigger | External agent jobs |
| ops.review_retro | Review step | Post-execution analysis |
| audit.pipeline_run | Pipeline runner | Pipeline execution tracking |
| audit.ingestion_batch | CSV ingest step | Batch ingestion tracking |
| audit.pipeline_step_run | Pipeline runner | Per-step execution tracking |

### Pipeline Data Flow

```
raw.* ──CSV Ingest──► dwd.* ──Metric Calc──► mart.* ──Alert Engine──► ops.* ──Outbox──► External
                         │                                                │
                         └── audit.pipeline_run ◄── Pipeline Runner ──────┘
                         └── audit.ingestion_batch ◄── CSV Ingest ─────────┘
```

Phase 3 does NOT interact with `gov.*` or `ai.*` tables. Governance tables are populated in Phase 5 (config loaders). AI tables are populated in Phase 7 (LLM decision layer).

---

## Appendix A: Table Count Summary by Schema

| Schema | Migrated Tables | New Tables | Total Tables |
|---|---|---|---|
| raw | 0 | 11 | 11 |
| dwd | 2 | 0 | 2 |
| mart | 2 | 1 | 3 |
| ops | 6 | 1 | 7 |
| gov | 2 | 5 | 7 |
| ai | 2 | 4 | 6 |
| audit | 2 | 4 | 6 |
| **Total** | **16** | **26** | **42** |

## Appendix B: Boolean Fields Inventory

The following fields change from SQLite INTEGER (0/1) to PostgreSQL BOOLEAN:

| Table | Boolean Fields |
|---|---|
| dwd.order_level | is_late, is_cancelled |
| ops.metric_alert | (none in SQLite; all TEXT) |
| ops.recommendation | requires_approval |
| ops.review_retro | is_effective, promote_to_rule |
| ai.qoder_run | can_apply |
| ai.qoder_report | no_apply_performed, business_side_effect |

## 12. Schema Introspection

### Running the Introspection

```bash
psql "$DATABASE_URL" -f scripts/migration/introspect_schema.sql
```

The script outputs 6 sections: Schema Overview, Table Inventory, Column Type Distribution, Index Inventory, Foreign Key Constraints, and Summary. It inspects all 7 schemas (raw, dwd, mart, ops, gov, ai, audit) and reports on every table, column, index, and foreign key constraint.

### Introspection Results (2026-05-25)

#### Schema Overview

| Schema | Tables |
|--------|--------|
| raw | 11 |
| dwd | 2 |
| mart | 3 |
| ops | 7 |
| gov | 7 |
| ai | 6 |
| audit | 6 |
| **Total** | **42** |

#### Table Inventory

| Schema | Table | Columns |
|--------|-------|---------|
| raw | closed_deals | 18 |
| raw | marketing_qualified_leads | 8 |
| raw | olist_customers | 9 |
| raw | olist_geolocation | 10 |
| raw | olist_order_items | 11 |
| raw | olist_order_payments | 9 |
| raw | olist_order_reviews | 11 |
| raw | olist_orders | 12 |
| raw | olist_products | 13 |
| raw | olist_sellers | 8 |
| raw | product_category_name_translation | 6 |
| dwd | item_level | 15 |
| dwd | order_level | 23 |
| mart | metric_daily | 14 |
| mart | metric_dimension_daily | 7 |
| mart | metric_snapshot | 13 |
| ops | dispatch_attempt | 8 |
| ops | metric_alert | 19 |
| ops | outbox_event | 14 |
| ops | qoder_job | 11 |
| ops | recommendation | 17 |
| ops | review_retro | 13 |
| ops | task | 16 |
| gov | access_policy | 10 |
| gov | config_snapshot | 7 |
| gov | data_classification | 6 |
| gov | data_lineage | 8 |
| gov | governance_checkpoint | 10 |
| gov | health_check_result | 6 |
| gov | object_schema | 6 |
| ai | action_proposal | 9 |
| ai | decision_case | 7 |
| ai | llm_decision | 7 |
| ai | qoder_report | 12 |
| ai | qoder_run | 11 |
| ai | review_record | 6 |
| audit | api_request_log | 11 |
| audit | audit_log | 8 |
| audit | error_log | 7 |
| audit | ingestion_batch | 9 |
| audit | pipeline_run | 9 |
| audit | pipeline_step_run | 10 |

#### Expected Table Count Matrix

| Schema | Tables | Columns | Indexes | FKs |
|--------|--------|---------|---------|-----|
| raw | 11 | 115 | 11 | 0 |
| dwd | 2 | 38 | 2 | 0 |
| mart | 3 | 34 | 7 | 0 |
| ops | 7 | 98 | 15 | 3 |
| gov | 7 | 53 | 25 | 0 |
| ai | 6 | 52 | 11 | 0 |
| audit | 6 | 54 | 11 | 0 |
| **Total** | **42** | **444** | **82** | **3** |

> **Note on index counts**: The reported index counts include both user-defined indexes (from migration files, matching the Index Strategy in Section 7) and system-created indexes automatically generated by PostgreSQL for PRIMARY KEY and UNIQUE constraints. Raw schema tables show indexes because PostgreSQL creates implicit indexes for their primary key columns.

#### Column Type Distribution

The introspection reveals 39 distinct schema-type combinations across the 7 schemas:

| Schema | Data Types Used |
|--------|----------------|
| raw | bigint, boolean, date, numeric, text, timestamp with time zone |
| dwd | bigint, boolean, date, numeric, text, timestamp with time zone |
| mart | bigint, date, numeric, text, timestamp with time zone |
| ops | bigint, boolean, date, jsonb, numeric, text, timestamp with time zone |
| gov | bigint, jsonb, numeric, text, timestamp with time zone |
| ai | boolean, jsonb, numeric, text, timestamp with time zone |
| audit | bigint, date, jsonb, text, timestamp with time zone |

Dominant types: `text` (240 columns, 54.1%), `timestamp with time zone` (70, 15.8%), `bigint` (52, 11.7%), `numeric` (36, 8.1%), `jsonb` (18, 4.1%), `boolean` (10, 2.3%), `date` (9, 2.0%), and various domain-specific types.

#### Foreign Key Constraints

The 3 expected foreign key constraints from `sql/migrations/010_add_foreign_keys.sql` are confirmed present:

| Constraint Name | Source Table | Referenced Table |
|----------------|-------------|------------------|
| task_recommendation_id_fkey | ops.task | ops.recommendation |
| qoder_job_trigger_event_id_fkey | ops.qoder_job | ops.metric_alert |
| review_retro_recommendation_id_fkey | ops.review_retro | ops.recommendation |

All 3 FKs reside in the `ops` schema with `ON DELETE SET NULL`, matching the design intent.

### Known Deviations from SQLite Baseline

- `dwd.item_level`: `item_key` column removed, replaced with composite PK `(order_id, order_item_id)`
- `ops.metric_alert`: `event_id` renamed to `alert_id`
- `ops.outbox_event`: `outbox_id` renamed to `event_id`
- All tables: Added `created_at`, `updated_at` tracking columns
- SQLite `INTEGER AUTOINCREMENT` → PostgreSQL `GENERATED ALWAYS AS IDENTITY`

### Verification Commands

```bash
# Quick schema health check
psql "$DATABASE_URL" -c "SELECT table_schema, COUNT(*) FROM information_schema.tables WHERE table_schema IN ('raw','dwd','mart','ops','gov','ai','audit') GROUP BY table_schema ORDER BY table_schema;"

# Full introspection (this script)
psql "$DATABASE_URL" -f scripts/migration/introspect_schema.sql

# Run parity with SQLite baseline
python3 scripts/migration/verify_schema.py
```

---

## Appendix C: Default Value Translation Reference

| SQLite Expression | Translation Target | Used In |
|---|---|---|
| `datetime('now')` | `NOW()` | governance_checkpoints.created_at, governance_health_results.checked_at |
| `0` (boolean false) | `FALSE` | is_late, is_cancelled, requires_approval, can_apply |
| `1` (boolean true) | `TRUE` | no_apply_performed, business_side_effect |
| `0` (count default) | `0` (BIGINT) | dispatch_attempts, row_count, input_count, output_count |
| `'new'` | `'new'` | alert_events.status |
| `'pending'` | `'pending'` | event_outbox.status, qoder_jobs.dispatch_status |
| `'draft'` | `'draft'` | strategy_recommendations.approval_status, review_retro.status |
| `'todo'` | `'todo'` | action_tasks.status |
| `'global'` | `'global'` | alert_events.object_type, alert_events.object_id |
| `'heuristic_strategy'` | `'heuristic_strategy'` | action_tasks.task_source |
| `'medium'` | `'medium'` | action_tasks.priority |
| `'qoder'` | `'qoder'` | qoder_runs.actor |
| `'read_only'` | `'read_only'` | qoder_runs.mode |
| `'simulated'` | `'simulated'` | review_retro.review_type |
| `'hindsight_rule'` | `'hindsight_rule'` | review_retro.review_source |
| `'dry_run'` | `'dry_run'` | governance_checkpoints.mode |
| `'recorded'` | `'recorded'` | governance_checkpoints.status |
