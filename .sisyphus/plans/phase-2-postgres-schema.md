# Phase 2: PostgreSQL Schema Design & Migration

## TL;DR

> **Quick Summary**: Design PostgreSQL layered schema (raw/dwd/mart/ops/gov/ai/audit) and create goose migrations mapping 16 existing SQLite tables plus 11 raw Olist staging tables and 15+ new design tables. No data import, no Go models, no pipeline logic.
>
> **Deliverables**:
> - `docs/migration/phase-2-schema-design.md` — Schema design document
> - `migrations/002_raw_tables.sql` — 11 raw Olist staging tables
> - `migrations/003_dwd_tables.sql` — 2 DWD detail-wide tables
> - `migrations/004_mart_tables.sql` — 3 mart metric tables
> - `migrations/005_ops_tables.sql` — 7 ops workflow tables
> - `migrations/006_gov_tables.sql` — 7 governance tables
> - `migrations/007_ai_tables.sql` — 6 AI/LLM tables
> - `migrations/008_audit_tables.sql` — 6 audit tables
> - `scripts/migration/verify_schema.py` — Automated SQLite↔PostgreSQL parity check
>
> **Estimated Effort**: Medium (2-3 waves, ~10 tasks)
> **Parallel Execution**: YES — 3 waves (Design → Migrations → Verification)
> **Critical Path**: Task 1 (Design) → Tasks 2-8 (Migrations in parallel) → Task 9 (Verification script) → Task 10 (Final QA)

---

## Context

### Original Request
Migrate Baxi from Python + SQLite + FastAPI to Go + PostgreSQL. Phase 2 specifically designs and implements the PostgreSQL schema layer across 7 schemas, creating goose migrations for all tables. Phase 2 does NOT import data, does NOT migrate pipeline logic, does NOT generate Go models, does NOT connect LLM.

### Interview Summary
**Key Discisions**:
- qoder_runs / qoder_reports → **ai** schema (AI Agent execution artifacts)
- metric_dimension_daily → **dedicated mart table** (not merged into metric_snapshot)
- Primary keys → **TEXT** (preserve SQLite IDs, no UUID in Phase 2)
- 11 raw Olist tables → created as **empty definitions** in Phase 2
- Go domain structs → **NOT in Phase 2** (SQL migration + documentation only)

### Research Findings
- **16 SQLite tables** exist with **906,526 total rows** (migration_baseline/table_counts.json)
- **7 PostgreSQL schemas** already created via `migrations/001_init_schemas.sql`
- **Go skeleton** exists (Docker Compose, API/Worker binaries, DB pool, health endpoint)
- **Makefile** has goose targets (`make migrate`, `make migrate-down`, `make migrate-status`)
- **SQLite schema source of truth**: `migration_baseline/sqlite_schema.sql` (NOT `sql/schema.sql` which is outdated v0.2)

### Metis Review
**Identified Gaps** (addressed in this plan):
- **Source of truth**: Explicitly designated `migration_baseline/sqlite_schema.sql` as exclusive source
- **Indexes**: Included in Phase 2 (critical for 693K-row metric_dimension_daily)
- **AUTOINCREMENT**: Use `GENERATED ALWAYS AS IDENTITY` (SQL standard, Go-friendly)
- **FK constraints**: Recreate migration 010 foreign keys with `ON DELETE SET NULL`
- **Missing table mappings**: Resolved (see Table Mapping Matrix below)
- **Default values**: SQLite `datetime('now')` → PostgreSQL `NOW()`

---

## Work Objectives

### Core Objective
Create a complete PostgreSQL schema design document and 7 goose migration files that establish all tables across 7 schemas, with proper PostgreSQL types, constraints, indexes, and parity with the SQLite baseline.

### Concrete Deliverables
1. Schema design document at `docs/migration/phase-2-schema-design.md`
2. 7 goose migration files (`002` through `008`)
3. Schema verification script at `scripts/migration/verify_schema.py`
4. All migrations pass `make migrate` and `make migrate-down`

### Definition of Done
- [x] All migrations apply cleanly via `make migrate`
- [x] All migrations roll back cleanly via `make migrate-down`
- [x] `psql` query shows all expected tables in correct schemas
- [x] Verification script confirms column parity between SQLite baseline and PostgreSQL
- [x] `go test ./...` still passes (no Go code broken)
- [x] `make api` + `curl http://localhost:8080/api/v1/health` still returns `{"status":"ok"}`

### Must Have
- [x] 7 schemas populated with tables
- [x] All 16 existing SQLite tables mapped to PostgreSQL
- [x] All 11 raw Olist tables defined
- [x] Proper PostgreSQL types (TIMESTAMPTZ, NUMERIC, BOOLEAN, JSONB)
- [x] Primary keys preserved (TEXT)
- [x] Foreign keys from migration 010 recreated
- [x] Indexes for performance-critical tables
- [x] Goose Up/Down blocks for every migration

### Must NOT Have (Guardrails)
- [x] NO Go struct/model generation
- [x] NO data migration / ETL / INSERT statements
- [x] NO trigger creation
- [x] NO view creation
- [x] NO function/procedure creation
- [x] NO partitioning
- [x] NO seed data (except verification needs)
- [x] NO changing TEXT PK to UUID
- [x] NO adding columns "for future use"
- [x] NO CHECK constraints beyond SQLite parity
- [x] NO modifying Python pipeline code
- [x] NO modifying FastAPI business code
- [x] NO modifying React frontend
- [x] NO modifying existing YAML config semantics

---

## Verification Strategy

### Test Decision
- **Infrastructure exists**: YES (goose CLI, Makefile targets, PostgreSQL Docker)
- **Automated tests**: Tests-after (verification script, not TDD — schema is design-first)
- **Framework**: Python script for SQLite↔PostgreSQL parity + Bash/PSQL for schema verification

### QA Policy
Every task MUST include agent-executed QA scenarios. Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **Schema verification**: Bash (psql) — Query information_schema, compare counts
- **Migration rollback**: Bash (goose) — Run up/down/up cycle
- **Parity check**: Python script — Compare SQLite PRAGMA vs PostgreSQL information_schema

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Start Immediately — Foundation):
├── Task 1: Schema Design Document [writing]
│   └── Output: docs/migration/phase-2-schema-design.md
└── Task 2: Schema Verification Script [quick]
    └── Output: scripts/migration/verify_schema.py

Wave 2 (After Wave 1 — Migrations, MAX PARALLEL):
├── Task 3: Migration 002 — raw tables [quick]
├── Task 4: Migration 003 — dwd tables [quick]
├── Task 5: Migration 004 — mart tables [quick]
├── Task 6: Migration 005 — ops tables [quick]
├── Task 7: Migration 006 — gov tables [quick]
├── Task 8: Migration 007 — ai tables [quick]
└── Task 9: Migration 008 — audit tables [quick]

Wave 3 (After Wave 2 — Integration + QA):
├── Task 10: Full migration test + parity verification [unspecified-high]
│   └── Run all migrations up → verify → down → verify → up → verify
└── Task 11: Schema introspection script + docs update [quick]
    └── Output: scripts/migration/introspect_schema.sql

Wave FINAL (After ALL tasks — 4 parallel reviews, then user okay):
├── Task F1: Plan compliance audit (oracle)
├── Task F2: Code quality review (unspecified-high)
├── Task F3: Real manual QA (unspecified-high)
└── Task F4: Scope fidelity check (deep)
-> Present results -> Get explicit user okay

Critical Path: Task 1 → Tasks 3-9 (parallel) → Task 10 → F1-F4 → user okay
Parallel Speedup: ~60% faster than sequential
Max Concurrent: 7 (Wave 2)
```

### Dependency Matrix

| Task | Depends On | Blocks |
|------|-----------|--------|
| 1 | — | 3-9 |
| 2 | — | 10 |
| 3 | 1 | 10 |
| 4 | 1 | 10 |
| 5 | 1 | 10 |
| 6 | 1 | 10 |
| 7 | 1 | 10 |
| 8 | 1 | 10 |
| 9 | 1 | 10 |
| 10 | 3-9, 2 | F1-F4 |
| 11 | 10 | F1-F4 |
| F1-F4 | 10-11 | — |

### Agent Dispatch Summary

| Wave | Tasks | Agent Categories |
|------|-------|-----------------|
| 1 | 1-2 | writing, quick |
| 2 | 3-9 | quick (7 parallel tasks) |
| 3 | 10-11 | unspecified-high, quick |
| FINAL | F1-F4 | oracle, unspecified-high, deep |

---

## TODOs

- [x] 1. Schema Design Document

  **What to do**:
  - Write comprehensive schema design document at `docs/migration/phase-2-schema-design.md`
  - Include table-to-schema mapping matrix for all 16 existing SQLite tables + 11 raw tables + new design tables
  - Include column-level type conversion table (SQLite → PostgreSQL)
  - Include index design rationale
  - Include primary key and foreign key strategy
  - Include JSONB usage strategy
  - Include Phase 3 Pipeline dependency mapping
  - Document AUTOINCREMENT → `GENERATED ALWAYS AS IDENTITY` strategy
  - Document default value translations
  - Document scope boundaries (what's NOT included)

  **Must NOT do**:
  - Do NOT include SQL DDL in the design document
  - Do NOT define Go structs or models
  - Do NOT include data migration plans
  - Do NOT go beyond table/column/index design

  **Recommended Agent Profile**:
  - **Category**: `writing`
    - Reason: Pure documentation task requiring structured technical writing
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Task 2)
  - **Parallel Group**: Wave 1
  - **Blocks**: Tasks 3-9
  - **Blocked By**: None

  **References**:
  - **Source of truth**: `migration_baseline/sqlite_schema.sql` — EXCLUSIVE source for existing table definitions. DO NOT use `sql/schema.sql` (outdated v0.2)
  - `migration_baseline/table_counts.json` — Row counts for understanding table scale
  - `docs/migration/phase-1-foundation.md` — Phase 1 context and schema purposes
  - `docs/migration/go-postgres-migration-plan.md` — High-level migration plan
  - `sql/indexes.sql` — Existing SQLite indexes (25 indexes across 12 tables)
  - `sql/migrations/010_add_foreign_keys.sql` — Foreign key constraints from migration 010
  - Olist dataset documentation (external) — For raw table field definitions

  **Acceptance Criteria**:
  - [ ] File exists: `docs/migration/phase-2-schema-design.md`
  - [ ] Document contains table-to-schema mapping matrix
  - [ ] Document contains column type conversion table
  - [ ] Document contains index strategy
  - [ ] Document contains PK/FK strategy
  - [ ] Document contains JSONB usage strategy
  - [ ] Document explicitly designates `migration_baseline/sqlite_schema.sql` as source of truth
  - [ ] Document contains "Must NOT Have" guardrails section

  **QA Scenarios**:

  ```
  Scenario: Document completeness check
    Tool: Bash
    Preconditions: Task 1 completed
    Steps:
      1. grep -c "## " docs/migration/phase-2-schema-design.md
      2. Verify document has at least 8 major sections
      3. grep -c "|" docs/migration/phase-2-schema-design.md
      4. Verify document has at least 5 tables (markdown tables)
    Expected Result: grep counts >= expected values
    Evidence: .sisyphus/evidence/task-1-doc-structure.txt
  ```

  **Commit**: YES
  - Message: `docs: add phase 2 postgres schema design`
  - Files: `docs/migration/phase-2-schema-design.md`

---

- [x] 2. Schema Verification Script

  **What to do**:
  - Create Python script at `scripts/migration/verify_schema.py`
  - Script connects to SQLite (`data/olist_ops.db`) and PostgreSQL (`$DATABASE_URL`)
  - Compares table lists between SQLite and PostgreSQL
  - Compares column names and types per table
  - Outputs PASS/FAIL report with mismatches
  - Returns exit code 0 on parity, 1 on mismatch
  - Should be runnable standalone: `python3 scripts/migration/verify_schema.py`

  **Must NOT do**:
  - Do NOT modify any database (read-only)
  - Do NOT require complex dependencies (stdlib + psycopg2-binary/sqlite3 only)
  - Do NOT validate data content (only schema)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Standalone Python script, no dependencies on other tasks
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Task 1)
  - **Parallel Group**: Wave 1
  - **Blocks**: Task 10
  - **Blocked By**: None

  **References**:
  - `migration_baseline/sqlite_schema.sql` — Table/column reference
  - `sql/indexes.sql` — Index reference
  - `scripts/migration/export_schema.py` — Existing export script (reference pattern)

  **Acceptance Criteria**:
  - [ ] File exists: `scripts/migration/verify_schema.py`
  - [ ] Script runs without errors: `python3 scripts/migration/verify_schema.py`
  - [ ] Script can connect to both SQLite and PostgreSQL
  - [ ] Script outputs structured report (table count, column count per table)
  - [ ] Script returns exit code 0 on parity

  **QA Scenarios**:

  ```
  Scenario: Script execution test
    Tool: Bash
    Preconditions: PostgreSQL running, SQLite DB exists
    Steps:
      1. python3 scripts/migration/verify_schema.py
      2. Check exit code: echo $?
    Expected Result: Exit code 0, output shows comparison results
    Evidence: .sisyphus/evidence/task-2-verify-script.txt
  ```

  **Commit**: YES (can be grouped with Task 1 or standalone)
  - Message: `chore: add schema verification script`
  - Files: `scripts/migration/verify_schema.py`

---

- [x] 3. Migration 002 — raw Tables

  **What to do**:
  - Create `migrations/002_raw_tables.sql` with goose format
  - Create 11 raw Olist staging tables as empty definitions:
    - `raw.olist_customers` — customer_id, customer_unique_id, customer_zip_code_prefix, customer_city, customer_state
    - `raw.olist_orders` — order_id, customer_id, order_status, order_purchase_timestamp, order_approved_at, order_delivered_carrier_date, order_delivered_customer_date, order_estimated_delivery_date
    - `raw.olist_order_items` — order_id, order_item_id, product_id, seller_id, shipping_limit_date, price, freight_value
    - `raw.olist_order_payments` — order_id, payment_sequential, payment_type, payment_installments, payment_value
    - `raw.olist_order_reviews` — review_id, order_id, review_score, review_comment_title, review_comment_message, review_creation_date, review_answer_timestamp
    - `raw.olist_products` — product_id, product_category_name, product_name_lenght, product_description_lenght, product_photos_qty, product_weight_g, product_length_cm, product_height_cm, product_width_cm
    - `raw.olist_sellers` — seller_id, seller_zip_code_prefix, seller_city, seller_state
    - `raw.olist_geolocation` — geolocation_zip_code_prefix, geolocation_lat, geolocation_lng, geolocation_city, geolocation_state
    - `raw.product_category_name_translation` — product_category_name, product_category_name_english
    - `raw.marketing_qualified_leads` — mql_id, first_contact_date, landing_page_id, origin
    - `raw.closed_deals` — mql_id, seller_id, sdr_id, sr_id, won_date, business_segment, lead_type, lead_behaviour_profile, has_company, has_gtin, average_stock, business_type, declared_product_catalog_size, declared_monthly_revenue
  - Add unified ingestion tracking columns to all raw tables:
    - `ingested_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
    - `source_file TEXT`
    - `source_row_number BIGINT`
    - `raw_hash TEXT`
  - Use `TEXT` for string IDs (Olist uses string hashes)
  - Use `TIMESTAMPTZ` for all timestamp fields
  - Use `NUMERIC(18,2)` for monetary fields (price, freight_value, payment_value)
  - Use `NUMERIC(10,6)` for latitude/longitude
  - Primary keys: business natural keys where available, `BIGSERIAL` for geolocation (no natural PK)

  **Must NOT do**:
  - Do NOT import any CSV data
  - Do NOT create indexes on raw tables (Phase 3 will add after data load)
  - Do NOT add foreign key constraints between raw tables
  - Do NOT add CHECK constraints

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Straightforward DDL translation from known Olist schema
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 4-9)
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 10
  - **Blocked By**: Task 1

  **References**:
  - Olist Brazilian E-Commerce Public Dataset schema (external knowledge)
  - `migration_baseline/sqlite_schema.sql` — Pattern for column naming (snake_case)
  - `docs/migration/phase-1-foundation.md` — raw schema purpose definition

  **Acceptance Criteria**:
  - [ ] File exists: `migrations/002_raw_tables.sql`
  - [ ] Goose Up creates all 11 tables in `raw` schema
  - [ ] Goose Down drops all 11 tables
  - [ ] `make migrate` applies without errors
  - [ ] `psql` query shows 11 tables in `raw` schema

  **QA Scenarios**:

  ```
  Scenario: Raw tables migration test
    Tool: Bash
    Preconditions: PostgreSQL running, goose installed
    Steps:
      1. make migrate
      2. psql "$DATABASE_URL" -c "SELECT table_name FROM information_schema.tables WHERE table_schema = 'raw' ORDER BY table_name;"
    Expected Result: 11 tables listed (olist_customers, olist_orders, olist_order_items, olist_order_payments, olist_order_reviews, olist_products, olist_sellers, olist_geolocation, product_category_name_translation, marketing_qualified_leads, closed_deals)
    Evidence: .sisyphus/evidence/task-3-raw-tables.txt

  Scenario: Raw tables rollback test
    Tool: Bash
    Preconditions: Migration 002 applied
    Steps:
      1. goose -dir migrations postgres "$DATABASE_URL" down
      2. psql "$DATABASE_URL" -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'raw';"
    Expected Result: Count = 0 (raw schema empty)
    Evidence: .sisyphus/evidence/task-3-raw-rollback.txt
  ```

  **Commit**: YES
  - Message: `feat: add raw schema empty definitions`
  - Files: `migrations/002_raw_tables.sql`

---

- [x] 4. Migration 003 — dwd Tables

  **What to do**:
  - Create `migrations/003_dwd_tables.sql` with goose format
  - Create 2 DWD detail-wide tables mapping from SQLite baseline:
    - `dwd.order_level` (from `dwd_order_level`)
      - PK: `order_id TEXT PRIMARY KEY`
      - Fields: customer_id, customer_unique_id, order_status, order_purchase_timestamp (TIMESTAMPTZ), purchase_date (DATE), customer_state, payment_type, payment_installments, payment_value (NUMERIC(18,2)), review_score (NUMERIC(4,2)), delivered_customer_date (TIMESTAMPTZ), estimated_delivery_date (TIMESTAMPTZ), delivery_days (NUMERIC(10,6)), delay_days (NUMERIC(10,6)), is_late (BOOLEAN), is_cancelled (BOOLEAN)
      - Tracking: ingestion_batch_id, loaded_at (TIMESTAMPTZ)
      - New tracking: created_at (TIMESTAMPTZ DEFAULT NOW()), updated_at (TIMESTAMPTZ), pipeline_run_id TEXT, record_hash TEXT
    - `dwd.item_level` (from `dwd_item_level`)
      - PK: `order_id TEXT, order_item_id BIGINT, PRIMARY KEY (order_id, order_item_id)`
      - Fields: product_id, seller_id, product_category_name, product_category_name_english, seller_state, price (NUMERIC(18,2)), freight_value (NUMERIC(18,2))
      - Tracking: ingestion_batch_id, loaded_at (TIMESTAMPTZ)
      - New tracking: created_at (TIMESTAMPTZ DEFAULT NOW()), updated_at (TIMESTAMPTZ), pipeline_run_id TEXT, record_hash TEXT
  - Note: SQLite `item_key` (composite TEXT PK) replaced with natural composite PK `(order_id, order_item_id)`
  - Remove `item_key` column (was SQLite-specific surrogate key)

  **Must NOT do**:
  - Do NOT create foreign keys to raw tables (avoid circular deps)
  - Do NOT add data
  - Do NOT create indexes (add in Phase 3 after data load)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Direct column-for-column translation with type mapping
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 3, 5-9)
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 10
  - **Blocked By**: Task 1

  **References**:
  - `migration_baseline/sqlite_schema.sql` — `dwd_order_level` and `dwd_item_level` definitions
  - `migration_baseline/table_counts.json` — 99,441 + 112,650 rows (scale context)

  **Acceptance Criteria**:
  - [ ] File exists: `migrations/003_dwd_tables.sql`
  - [ ] `dwd.order_level` has 18+ columns
  - [ ] `dwd.item_level` has 13+ columns
  - [ ] Composite PK on `dwd.item_level(order_id, order_item_id)`
  - [ ] `is_late` and `is_cancelled` are BOOLEAN type
  - [ ] `payment_value`, `price`, `freight_value` are NUMERIC(18,2)

  **QA Scenarios**:

  ```
  Scenario: DWD table structure verification
    Tool: Bash (psql)
    Preconditions: Migration 003 applied
    Steps:
      1. psql "$DATABASE_URL" -c "\d dwd.order_level"
      2. psql "$DATABASE_URL" -c "\d dwd.item_level"
    Expected Result: Both tables exist with correct column types. order_level has TEXT PK. item_level has composite PK.
    Evidence: .sisyphus/evidence/task-4-dwd-structure.txt
  ```

  **Commit**: YES
  - Message: `feat: add dwd schema tables`
  - Files: `migrations/003_dwd_tables.sql`

---

- [x] 5. Migration 004 — mart Tables

  **What to do**:
  - Create `migrations/004_mart_tables.sql` with goose format
  - Create 3 mart tables:
    - `mart.metric_snapshot` (NEW design — unified metric fact table)
      - PK: `snapshot_id BIGSERIAL PRIMARY KEY`
      - Fields: metric_name TEXT NOT NULL, metric_value NUMERIC(18,4), metric_date DATE, grain TEXT, dimension_type TEXT, dimension_value TEXT, baseline_value NUMERIC(18,4), delta_value NUMERIC(18,4), delta_pct NUMERIC(10,6), severity_hint TEXT, created_at TIMESTAMPTZ DEFAULT NOW(), pipeline_run_id TEXT
      - Unique: `(metric_name, metric_date, grain, dimension_type, dimension_value)`
    - `mart.metric_daily` (from `metric_daily`)
      - PK: `metric_date DATE PRIMARY KEY`
      - Fields: gmv (NUMERIC(18,2)), order_count (BIGINT), customer_count (BIGINT), seller_count (BIGINT), avg_order_value (NUMERIC(18,2)), freight_value (NUMERIC(18,2)), avg_review_score (NUMERIC(4,2)), low_review_rate (NUMERIC(10,6)), late_delivery_rate (NUMERIC(10,6)), cancel_rate (NUMERIC(10,6)), payment_installment_rate (NUMERIC(10,6)), marketing_seller_share (NUMERIC(10,6)), created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    - `mart.metric_dimension_daily` (from `metric_dimension_daily`)
      - PK: `metric_date DATE, dimension_type TEXT, dimension_value TEXT, metric_name TEXT, PRIMARY KEY (metric_date, dimension_type, dimension_value, metric_name)`
      - Fields: metric_value NUMERIC(18,4), sample_size BIGINT, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
  - Preserve composite PK on `metric_dimension_daily` (4-column natural key)

  **Must NOT do**:
  - Do NOT add data
  - Do NOT create views
  - Do NOT add materialized view logic

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Mix of existing table mapping + new table design
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 3-4, 6-9)
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 10
  - **Blocked By**: Task 1

  **References**:
  - `migration_baseline/sqlite_schema.sql` — `metric_daily` and `metric_dimension_daily`
  - `migration_baseline/table_counts.json` — 634 + 693,602 rows (693K dimension table needs indexes)
  - `sql/indexes.sql` — Existing indexes on metric tables

  **Acceptance Criteria**:
  - [ ] File exists: `migrations/004_mart_tables.sql`
  - [ ] 3 tables created in `mart` schema
  - [ ] `metric_dimension_daily` has 4-column composite PK
  - [ ] `metric_snapshot` has unique constraint on natural key
  - [ ] `metric_daily` has DATE PK
  - [ ] Monetary fields are NUMERIC(18,2) or NUMERIC(18,4)

  **QA Scenarios**:

  ```
  Scenario: Mart table verification
    Tool: Bash (psql)
    Preconditions: Migration 004 applied
    Steps:
      1. psql "$DATABASE_URL" -c "SELECT table_name FROM information_schema.tables WHERE table_schema = 'mart' ORDER BY table_name;"
      2. psql "$DATABASE_URL" -c "\d mart.metric_dimension_daily"
    Expected Result: 3 tables exist. metric_dimension_daily shows 4-column composite PK.
    Evidence: .sisyphus/evidence/task-5-mart-structure.txt
  ```

  **Commit**: YES
  - Message: `feat: add mart schema tables`
  - Files: `migrations/004_mart_tables.sql`

---

- [x] 6. Migration 005 — ops Tables

  **What to do**:
  - Create `migrations/005_ops_tables.sql` with goose format
  - Create 7 ops workflow tables mapping from SQLite baseline + new design:
    - `ops.metric_alert` (from `alert_events`)
      - PK: `alert_id TEXT PRIMARY KEY` (maps from event_id)
      - Fields: rule_id, event_date (DATE), severity, metric_name, object_type, object_id, current_value (NUMERIC(18,4)), baseline_value (NUMERIC(18,4)), change_rate (NUMERIC(10,6)), sample_size (BIGINT), affected_orders (BIGINT), affected_gmv (NUMERIC(18,2)), impact_score (NUMERIC(10,6)), evidence_json (JSONB), description, owner_role, status, created_at (TIMESTAMPTZ NOT NULL DEFAULT NOW())
    - `ops.recommendation` (from `strategy_recommendations`)
      - PK: `recommendation_id TEXT PRIMARY KEY`
      - Fields: alert_id, decision_source, rule_id, strategy_title, strategy_detail, target_object_type, target_object_id, expected_impact, risk_level, confidence, requires_approval (BOOLEAN DEFAULT FALSE), approval_status, execution_status, owner_role, success_metric, created_at (TIMESTAMPTZ NOT NULL DEFAULT NOW())
    - `ops.task` (from `action_tasks`)
      - PK: `task_id TEXT PRIMARY KEY`
      - Fields: recommendation_id, alert_id, task_title, task_description, target_object_type, target_object_id, task_source, owner_role, owner_user_id, priority, due_at (TIMESTAMPTZ), status, feedback, completed_at (TIMESTAMPTZ), created_at (TIMESTAMPTZ NOT NULL DEFAULT NOW())
    - `ops.outbox_event` (from `event_outbox`)
      - PK: `event_id TEXT PRIMARY KEY`
      - Fields: event_type, source_type, source_id, payload_json (JSONB), target_channel, status, dispatch_attempts (BIGINT DEFAULT 0), last_dispatch_at (TIMESTAMPTZ), external_ref, adapter_name, created_at (TIMESTAMPTZ NOT NULL DEFAULT NOW()), processed_at (TIMESTAMPTZ), error_message
    - `ops.dispatch_attempt` (NEW — tracks each dispatch try)
      - PK: `attempt_id BIGSERIAL PRIMARY KEY`
      - Fields: event_id, attempt_number (BIGINT), status, dispatched_at (TIMESTAMPTZ), response_json (JSONB), error_message, created_at (TIMESTAMPTZ DEFAULT NOW())
    - `ops.qoder_job` (from `qoder_jobs`)
      - PK: `job_id TEXT PRIMARY KEY`
      - Fields: trigger_event_id, job_type, job_title, job_context_json (JSONB), dispatch_channel, dispatch_status, external_ref, created_at (TIMESTAMPTZ NOT NULL DEFAULT NOW()), dispatched_at (TIMESTAMPTZ), completed_at (TIMESTAMPTZ)
    - `ops.review_retro` (from `review_retro`)
      - PK: `review_id TEXT PRIMARY KEY`
      - Fields: recommendation_id, task_id, review_type, review_source, actual_result, actual_impact, is_effective (BOOLEAN), lesson_learned, promote_to_rule (BOOLEAN DEFAULT FALSE), status, feedback, reviewed_at (TIMESTAMPTZ)
  - Add indexes for ops tables (operational queries need performance):
    - `ops.metric_alert(status, severity, created_at)`
    - `ops.metric_alert(object_type, object_id)`
    - `ops.task(status, owner_role, priority)`
    - `ops.outbox_event(status, next_attempt_at)` — but next_attempt_at doesn't exist, use status + created_at
    - `ops.outbox_event(status)`
    - `ops.dispatch_attempt(event_id)`
    - `ops.qoder_job(dispatch_status, created_at)`
    - `ops.review_retro(recommendation_id)`
  - Add foreign key constraints from migration 010:
    - `ops.task.recommendation_id` → `ops.recommendation.recommendation_id` ON DELETE SET NULL
    - `ops.qoder_job.trigger_event_id` → `ops.metric_alert.alert_id` ON DELETE SET NULL
    - `ops.review_retro.recommendation_id` → `ops.recommendation.recommendation_id` ON DELETE SET NULL

  **Must NOT do**:
  - Do NOT add data
  - Do NOT create triggers for outbox event status changes

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Largest migration file but still straightforward DDL
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 3-5, 7-9)
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 10
  - **Blocked By**: Task 1

  **References**:
  - `migration_baseline/sqlite_schema.sql` — alert_events, strategy_recommendations, action_tasks, event_outbox, qoder_jobs, review_retro
  - `sql/indexes.sql` — Existing indexes on ops tables
  - `sql/migrations/010_add_foreign_keys.sql` — FK constraints to recreate
  - `sql/migrations/005_dispatch_adapters.sql` — outbox columns added in migration 005
  - `sql/migrations/006_api_schema_fix.sql` — alert columns added in migration 006
  - `sql/migrations/007_review_retro_status_feedback.sql` — review_retro columns added in migration 007

  **Acceptance Criteria**:
  - [ ] File exists: `migrations/005_ops_tables.sql`
  - [ ] 7 tables created in `ops` schema
  - [ ] FK constraints exist with ON DELETE SET NULL
  - [ ] Indexes created on performance-critical columns
  - [ ] BOOLEAN fields use BOOLEAN type (is_effective, promote_to_rule, requires_approval)
  - [ ] JSONB fields use JSONB type (evidence_json, payload_json, job_context_json)

  **QA Scenarios**:

  ```
  Scenario: Ops tables with constraints
    Tool: Bash (psql)
    Preconditions: Migration 005 applied
    Steps:
      1. psql "$DATABASE_URL" -c "SELECT table_name FROM information_schema.tables WHERE table_schema = 'ops' ORDER BY table_name;"
      2. psql "$DATABASE_URL" -c "SELECT conname, conrelid::regclass, confrelid::regclass FROM pg_constraint WHERE connamespace = 'ops'::regnamespace AND contype = 'f';"
      3. psql "$DATABASE_URL" -c "SELECT indexname FROM pg_indexes WHERE schemaname = 'ops';"
    Expected Result: 7 tables, 3 FK constraints, 6+ indexes
    Evidence: .sisyphus/evidence/task-6-ops-structure.txt
  ```

  **Commit**: YES
  - Message: `feat: add ops schema tables`
  - Files: `migrations/005_ops_tables.sql`

---

- [x] 7. Migration 006 — gov Tables

  **What to do**:
  - Create `migrations/006_gov_tables.sql` with goose format
  - Create 7 governance tables:
    - `gov.config_snapshot` (NEW — stores YAML config snapshots)
      - PK: `snapshot_id BIGSERIAL PRIMARY KEY`
      - Fields: config_key TEXT NOT NULL, config_type TEXT, source_path TEXT, content_jsonb JSONB, content_hash TEXT, loaded_at TIMESTAMPTZ DEFAULT NOW()
      - Unique: `(config_key, content_hash)`
    - `gov.object_schema` (NEW — ontology object definitions)
      - PK: `object_schema_id BIGSERIAL PRIMARY KEY`
      - Fields: object_type TEXT NOT NULL, object_name TEXT, schema_jsonb JSONB, version TEXT, created_at TIMESTAMPTZ DEFAULT NOW()
      - Unique: `(object_type, version)`
    - `gov.data_classification` (NEW — data classification rules)
      - PK: `classification_id BIGSERIAL PRIMARY KEY`
      - Fields: field_path TEXT NOT NULL, classification_level TEXT, sensitivity_score NUMERIC(4,2), description TEXT, created_at TIMESTAMPTZ DEFAULT NOW()
    - `gov.data_lineage` (NEW — data lineage records)
      - PK: `lineage_id BIGSERIAL PRIMARY KEY`
      - Fields: source_table TEXT, source_column TEXT, target_table TEXT, target_column TEXT, transformation_logic TEXT, confidence NUMERIC(4,2), created_at TIMESTAMPTZ DEFAULT NOW()
    - `gov.access_policy` (NEW — access control policies)
      - PK: `policy_id BIGSERIAL PRIMARY KEY`
      - Fields: policy_name TEXT NOT NULL, resource_type TEXT, resource_pattern TEXT, action TEXT, principal_type TEXT, principal_pattern TEXT, effect TEXT, conditions_jsonb JSONB, created_at TIMESTAMPTZ DEFAULT NOW()
    - `gov.health_check_result` (from `governance_health_results`)
      - PK: `result_id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY`
      - Fields: check_id TEXT NOT NULL, check_type TEXT NOT NULL, status TEXT NOT NULL, detail TEXT, checked_at TIMESTAMPTZ DEFAULT NOW()
    - `gov.governance_checkpoint` (from `governance_checkpoints`)
      - PK: `checkpoint_id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY`
      - Fields: action_type TEXT NOT NULL, endpoint TEXT NOT NULL, actor TEXT NOT NULL, request_id TEXT, justification TEXT, mode TEXT NOT NULL DEFAULT 'dry_run', status TEXT NOT NULL DEFAULT 'recorded', metadata_json JSONB, created_at TIMESTAMPTZ DEFAULT NOW()
  - Use `GENERATED ALWAYS AS IDENTITY` for AUTOINCREMENT migration (governance_checkpoint, health_check_result)

  **Must NOT do**:
  - Do NOT import YAML configs as data
  - Do NOT create complex policy evaluation logic

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Mix of existing table mapping + new design tables
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 3-6, 8-9)
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 10
  - **Blocked By**: Task 1

  **References**:
  - `migration_baseline/sqlite_schema.sql` — governance_checkpoints, governance_health_results
  - `migration_baseline/configs_snapshot/` — 28 YAML files for context on config_snapshot design
  - `docs/governance/baxi_data_governance_policy.md` — Governance context

  **Acceptance Criteria**:
  - [ ] File exists: `migrations/006_gov_tables.sql`
  - [ ] 7 tables created in `gov` schema
  - [ ] `governance_checkpoint` uses `GENERATED ALWAYS AS IDENTITY`
  - [ ] `health_check_result` uses `GENERATED ALWAYS AS IDENTITY`
  - [ ] JSONB fields use JSONB type (content_jsonb, schema_jsonb, conditions_jsonb, metadata_json)

  **QA Scenarios**:

  ```
  Scenario: Gov tables with identity columns
    Tool: Bash (psql)
    Preconditions: Migration 006 applied
    Steps:
      1. psql "$DATABASE_URL" -c "SELECT table_name, column_name, column_default FROM information_schema.columns WHERE table_schema = 'gov' AND is_identity = 'YES';"
      2. psql "$DATABASE_URL" -c "SELECT table_name FROM information_schema.tables WHERE table_schema = 'gov' ORDER BY table_name;"
    Expected Result: 7 tables. governance_checkpoint and health_check_result show identity columns.
    Evidence: .sisyphus/evidence/task-7-gov-structure.txt
  ```

  **Commit**: YES
  - Message: `feat: add gov schema tables`
  - Files: `migrations/006_gov_tables.sql`

---

- [x] 8. Migration 007 — ai Tables

  **What to do**:
  - Create `migrations/007_ai_tables.sql` with goose format
  - Create 6 AI/LLM tables (2 from SQLite baseline + 4 new design):
    - `ai.qoder_run` (from `qoder_runs`)
      - PK: `run_id TEXT PRIMARY KEY`
      - Fields: run_type TEXT NOT NULL, mode TEXT NOT NULL DEFAULT 'read_only', status TEXT NOT NULL, started_at TIMESTAMPTZ NOT NULL, finished_at TIMESTAMPTZ, request_id TEXT, actor TEXT DEFAULT 'qoder', can_apply (BOOLEAN DEFAULT FALSE), error_message TEXT
    - `ai.qoder_report` (from `qoder_reports`)
      - PK: `report_id TEXT PRIMARY KEY`
      - Fields: run_id TEXT, run_type TEXT NOT NULL, summary TEXT NOT NULL, findings_json JSONB, recommended_human_actions_json JSONB, risk_level TEXT, used_endpoints_json JSONB, no_apply_performed (BOOLEAN NOT NULL DEFAULT TRUE), business_side_effect (BOOLEAN NOT NULL DEFAULT FALSE), created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), request_id TEXT
    - `ai.decision_case` (NEW — aggregates alert + context for LLM)
      - PK: `case_id TEXT PRIMARY KEY`
      - Fields: alert_id TEXT, case_type TEXT, status TEXT DEFAULT 'open', context_json JSONB, created_at TIMESTAMPTZ DEFAULT NOW(), resolved_at TIMESTAMPTZ
    - `ai.llm_decision` (NEW — structured LLM output)
      - PK: `decision_id TEXT PRIMARY KEY`
      - Fields: case_id TEXT, model_version TEXT, prompt_hash TEXT, output_json JSONB, confidence NUMERIC(4,2), created_at TIMESTAMPTZ DEFAULT NOW()
    - `ai.action_proposal` (NEW — proposed actions from LLM)
      - PK: `proposal_id TEXT PRIMARY KEY`
      - Fields: case_id TEXT, decision_id TEXT, action_type TEXT, payload JSONB, apply_status TEXT DEFAULT 'pending', created_at TIMESTAMPTZ DEFAULT NOW(), applied_at TIMESTAMPTZ, applied_by TEXT
    - `ai.review_record` (NEW — human review of proposals)
      - PK: `review_id TEXT PRIMARY KEY`
      - Fields: proposal_id TEXT, reviewer_id TEXT, verdict TEXT, feedback TEXT, reviewed_at TIMESTAMPTZ DEFAULT NOW()
  - Add indexes:
    - `ai.decision_case(status, created_at)`
    - `ai.decision_case(alert_id)`
    - `ai.llm_decision(case_id)`
    - `ai.action_proposal(case_id, apply_status)`
    - `ai.review_record(proposal_id)`

  **Must NOT do**:
  - Do NOT add LLM integration logic
  - Do NOT add trigger for case status changes

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Mix of existing mapping + forward-looking design
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 3-7, 9)
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 10
  - **Blocked By**: Task 1

  **References**:
  - `migration_baseline/sqlite_schema.sql` — qoder_runs, qoder_reports
  - `migration_baseline/table_counts.json` — qoder_runs (19 rows), qoder_reports (18 rows)
  - `migration_baseline/api_responses/qoder_context.json` — Qoder API response format

  **Acceptance Criteria**:
  - [ ] File exists: `migrations/007_ai_tables.sql`
  - [ ] 6 tables created in `ai` schema
  - [ ] `qoder_run` and `qoder_report` map correctly from SQLite baseline
  - [ ] BOOLEAN fields use BOOLEAN (can_apply, no_apply_performed, business_side_effect)
  - [ ] JSONB fields use JSONB (findings_json, recommended_human_actions_json, used_endpoints_json, context_json, output_json, payload)

  **QA Scenarios**:

  ```
  Scenario: AI tables verification
    Tool: Bash (psql)
    Preconditions: Migration 007 applied
    Steps:
      1. psql "$DATABASE_URL" -c "SELECT table_name FROM information_schema.tables WHERE table_schema = 'ai' ORDER BY table_name;"
      2. psql "$DATABASE_URL" -c "\d ai.qoder_run"
      3. psql "$DATABASE_URL" -c "\d ai.decision_case"
    Expected Result: 6 tables. qoder_run has TEXT PK with BOOLEAN can_apply. decision_case has JSONB context_json.
    Evidence: .sisyphus/evidence/task-8-ai-structure.txt
  ```

  **Commit**: YES
  - Message: `feat: add ai schema tables`
  - Files: `migrations/007_ai_tables.sql`

---

- [x] 9. Migration 008 — audit Tables

  **What to do**:
  - Create `migrations/008_audit_tables.sql` with goose format
  - Create 6 audit tables (2 from SQLite baseline + 4 new design):
    - `audit.pipeline_run` (from `pipeline_runs`)
      - PK: `run_id TEXT PRIMARY KEY`
      - Fields: run_type TEXT NOT NULL, mode TEXT NOT NULL, status TEXT NOT NULL, started_at TIMESTAMPTZ NOT NULL, finished_at TIMESTAMPTZ, input_count (BIGINT DEFAULT 0), output_count (BIGINT DEFAULT 0), error_message TEXT
    - `audit.ingestion_batch` (from `ingestion_batches`)
      - PK: `batch_id TEXT PRIMARY KEY`
      - Fields: source_name TEXT NOT NULL, ingestion_mode TEXT NOT NULL, date_start DATE, date_end DATE, source_file TEXT, row_count (BIGINT DEFAULT 0), status TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    - `audit.pipeline_step_run` (NEW — individual step tracking)
      - PK: `step_run_id TEXT PRIMARY KEY`
      - Fields: pipeline_run_id TEXT, step_name TEXT, step_order BIGINT, status TEXT, started_at TIMESTAMPTZ, finished_at TIMESTAMPTZ, input_count BIGINT, output_count BIGINT, error_message TEXT
    - `audit.api_request_log` (NEW — API access logging)
      - PK: `log_id BIGSERIAL PRIMARY KEY`
      - Fields: request_id TEXT, method TEXT, path TEXT, status_code BIGINT, user_agent TEXT, client_ip TEXT, request_body_json JSONB, response_body_json JSONB, duration_ms BIGINT, created_at TIMESTAMPTZ DEFAULT NOW()
    - `audit.audit_log` (NEW — business audit trail)
      - PK: `audit_id BIGSERIAL PRIMARY KEY`
      - Fields: category TEXT, action TEXT, actor TEXT, resource_type TEXT, resource_id TEXT, metadata JSONB, created_at TIMESTAMPTZ DEFAULT NOW()
    - `audit.error_log` (NEW — error tracking)
      - PK: `error_id BIGSERIAL PRIMARY KEY`
      - Fields: request_id TEXT, error_type TEXT, error_message TEXT, stack_trace TEXT, details JSONB, created_at TIMESTAMPTZ DEFAULT NOW()
  - Add indexes:
    - `audit.pipeline_run(status, started_at)`
    - `audit.pipeline_step_run(pipeline_run_id)`
    - `audit.api_request_log(request_id)`
    - `audit.audit_log(category, created_at)`
    - `audit.error_log(request_id, created_at)`

  **Must NOT do**:
  - Do NOT add audit triggers (defer to Phase 5)
  - Do NOT add data

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Final migration, mix of existing + new tables
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 3-8)
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 10
  - **Blocked By**: Task 1

  **References**:
  - `migration_baseline/sqlite_schema.sql` — pipeline_runs, ingestion_batches
  - `migration_baseline/table_counts.json` — pipeline_runs (12), ingestion_batches (6)

  **Acceptance Criteria**:
  - [ ] File exists: `migrations/008_audit_tables.sql`
  - [ ] 6 tables created in `audit` schema
  - [ ] `pipeline_run` and `ingestion_batch` map correctly from SQLite baseline
  - [ ] `date_start` and `date_end` use DATE type (not TIMESTAMPTZ)
  - [ ] Indexes created for query patterns

  **QA Scenarios**:

  ```
  Scenario: Audit tables verification
    Tool: Bash (psql)
    Preconditions: Migration 008 applied
    Steps:
      1. psql "$DATABASE_URL" -c "SELECT table_name FROM information_schema.tables WHERE table_schema = 'audit' ORDER BY table_name;"
      2. psql "$DATABASE_URL" -c "SELECT indexname FROM pg_indexes WHERE schemaname = 'audit';"
    Expected Result: 6 tables, 5 indexes
    Evidence: .sisyphus/evidence/task-9-audit-structure.txt
  ```

  **Commit**: YES
  - Message: `feat: add audit schema tables`
  - Files: `migrations/008_audit_tables.sql`

---

- [x] 10. Full Migration Test + Parity Verification

  **What to do**:
  - Run complete migration cycle: `make migrate` (apply all 002-008)
  - Verify all tables exist in correct schemas via `psql`
  - Run verification script (`scripts/migration/verify_schema.py`) to compare SQLite baseline vs PostgreSQL
  - Run rollback test: `make migrate-down` (roll back all 008-002)
  - Verify all tables dropped
  - Re-apply: `make migrate` again
  - Verify schema is back
  - Check `go test ./...` still passes
  - Check `make api` + health endpoint still works
  - Document any parity mismatches

  **Must NOT do**:
  - Do NOT fix parity issues by modifying SQLite baseline
  - Do NOT modify Go code beyond what's needed for compilation

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Multi-step integration test requiring careful verification
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 3 (sequential)
  - **Blocks**: Task 11, F1-F4
  - **Blocked By**: Tasks 3-9, Task 2

  **References**:
  - `scripts/migration/verify_schema.py` — Parity check script (from Task 2)
  - `Makefile` — migrate, migrate-down, migrate-status targets
  - `migration_baseline/sqlite_schema.sql` — Baseline reference

  **Acceptance Criteria**:
  - [ ] `make migrate` applies all 8 migrations (001-008) without errors
  - [ ] `make migrate-down` rolls back all migrations without errors
  - [ ] `make migrate` re-applies successfully after rollback
  - [ ] Verification script shows schema parity (column names match)
- [x] `go test ./...` passes
  - [ ] `curl http://localhost:8080/api/v1/health` returns `{"status":"ok"}`

  **QA Scenarios**:

  ```
  Scenario: Full migration cycle
    Tool: Bash
    Preconditions: All migrations 002-008 exist
    Steps:
      1. make migrate
      2. make migrate-status
      3. python3 scripts/migration/verify_schema.py
      4. make migrate-down
      5. make migrate
      6. make migrate-status
    Expected Result: All migrations show Applied. Verification script exits 0. Rollback and re-apply succeed.
    Evidence: .sisyphus/evidence/task-10-migration-cycle.txt

  Scenario: Go compilation and API health
    Tool: Bash
    Preconditions: Migrations applied
    Steps:
      1. go test ./...
      2. make api &
      3. sleep 2
      4. curl -s http://localhost:8080/api/v1/health
      5. kill %1
    Expected Result: go test passes. curl returns {"status":"ok","service":"baxi-api"}
    Evidence: .sisyphus/evidence/task-10-go-health.txt
  ```

  **Commit**: NO (verification only, no new files)

---

- [x] 11. Schema Introspection Script + Documentation

  **What to do**:
  - Create `scripts/migration/introspect_schema.sql` — PSQL script to introspect all 7 schemas
  - Script outputs: table count per schema, column count per table, index count per schema, FK count per schema
  - Add schema introspection section to `docs/migration/phase-2-schema-design.md`
  - Document any deviations from SQLite baseline (e.g., item_key removed, new tracking columns added)
  - Document table count matrix (expected vs actual)

  **Must NOT do**:
  - Do NOT modify migration files
  - Do NOT add new tables

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Script + documentation updates
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 3
  - **Blocks**: F1-F4
  - **Blocked By**: Task 10

  **References**:
  - PostgreSQL `information_schema` and `pg_catalog` system catalogs
  - `docs/migration/phase-2-schema-design.md` — Document to update

  **Acceptance Criteria**:
  - [ ] File exists: `scripts/migration/introspect_schema.sql`
  - [ ] Script runs without errors and produces structured output
  - [ ] Design document updated with schema introspection section
  - [ ] Table count matrix documented

  **QA Scenarios**:

  ```
  Scenario: Introspection script execution
    Tool: Bash (psql)
    Preconditions: All migrations applied
    Steps:
      1. psql "$DATABASE_URL" -f scripts/migration/introspect_schema.sql
    Expected Result: Output shows 7 schemas with table counts, column counts, index counts
    Evidence: .sisyphus/evidence/task-11-introspection.txt
  ```

  **Commit**: YES
  - Message: `docs: add schema introspection script and verification docs`
  - Files: `scripts/migration/introspect_schema.sql`, `docs/migration/phase-2-schema-design.md`

---

## Final Verification Wave

> 4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.

- [x] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, curl endpoint, run command). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist in `.sisyphus/evidence/`. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [x] F2. **Code Quality Review** — `unspecified-high`
  Run `tsc --noEmit` + linter + `bun test`. Review all changed files for: `as any`/`@ts-ignore`, empty catches, console.log in prod, commented-out code, unused imports. Check AI slop: excessive comments, over-abstraction, generic names (data/result/item/temp).
  Output: `Build [PASS/FAIL] | Lint [PASS/FAIL] | Tests [N pass/N fail] | Files [N clean/N issues] | VERDICT`

- [x] F3. **Real Manual QA** — `unspecified-high`
  Start from clean state. Execute EVERY QA scenario from EVERY task — follow exact steps, capture evidence. Test cross-task integration (features working together, not isolation). Test edge cases: empty state, invalid input, rapid actions. Save to `.sisyphus/evidence/final-qa/`.
  Output: `Scenarios [N/N pass] | Integration [N/N] | Edge Cases [N tested] | VERDICT`

- [x] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual diff (git log/diff). Verify 1:1 — everything in spec was built (no missing), nothing beyond spec was built (no creep). Check "Must NOT do" compliance. Detect cross-task contamination: Task N touching Task M's files. Flag unaccounted changes.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

| Commit | Content | Files |
|--------|---------|-------|
| 1 | `docs: add phase 2 postgres schema design` | `docs/migration/phase-2-schema-design.md` |
| 2 | `chore: add schema verification script` | `scripts/migration/verify_schema.py` |
| 3 | `feat: add raw schema empty definitions` | `migrations/002_raw_tables.sql` |
| 4 | `feat: add dwd schema tables` | `migrations/003_dwd_tables.sql` |
| 5 | `feat: add mart schema tables` | `migrations/004_mart_tables.sql` |
| 6 | `feat: add ops schema tables` | `migrations/005_ops_tables.sql` |
| 7 | `feat: add gov schema tables` | `migrations/006_gov_tables.sql` |
| 8 | `feat: add ai schema tables` | `migrations/007_ai_tables.sql` |
| 9 | `feat: add audit schema tables` | `migrations/008_audit_tables.sql` |
| 10 | `docs: add schema introspection script and verification docs` | `scripts/migration/introspect_schema.sql`, `docs/migration/phase-2-schema-design.md` |

---

## Success Criteria

### Verification Commands
```bash
# 1. Start PostgreSQL
docker compose up -d postgres

# 2. Run all migrations
export DATABASE_URL="postgres://baxi:baxi_dev@localhost:5432/baxi?sslmode=disable"
make migrate

# 3. Verify schema counts
psql "$DATABASE_URL" -c "
SELECT table_schema, COUNT(*) AS table_count
FROM information_schema.tables
WHERE table_schema IN ('raw','dwd','mart','ops','gov','ai','audit')
GROUP BY table_schema
ORDER BY table_schema;
"
# Expected:
# raw   | 11
# dwd   | 2
# mart  | 3
# ops   | 7
# gov   | 7
# ai    | 6
# audit | 6

# 4. Verify key tables exist
psql "$DATABASE_URL" -c "
SELECT table_schema, table_name
FROM information_schema.tables
WHERE table_schema IN ('raw','dwd','mart','ops','gov','ai','audit')
ORDER BY table_schema, table_name;
"
# Must include:
# raw.olist_orders
# raw.olist_order_items
# dwd.order_level
# dwd.item_level
# mart.metric_snapshot
# mart.metric_daily
# mart.metric_dimension_daily
# ops.metric_alert
# ops.outbox_event
# gov.config_snapshot
# gov.governance_checkpoint
# ai.decision_case
# ai.qoder_run
# ai.action_proposal
# audit.pipeline_run
# audit.ingestion_batch

# 5. Run verification script
python3 scripts/migration/verify_schema.py
# Expected: Exit 0, parity report shows all 16 existing tables mapped

# 6. Test rollback
make migrate-down
make migrate

# 7. Go compilation
go test ./...

# 8. API health
make api &
curl http://localhost:8080/api/v1/health
# Expected: {"status":"ok","service":"baxi-api"}
```

### Final Checklist
- [x] All 7 schemas populated with tables
- [x] All 16 existing SQLite tables mapped to PostgreSQL
- [x] All 11 raw Olist tables defined
- [x] Proper PostgreSQL types used (TIMESTAMPTZ, NUMERIC, BOOLEAN, JSONB)
- [x] Primary keys preserved (TEXT)
- [x] Foreign keys from migration 010 recreated
- [x] Indexes created for performance-critical tables
- [x] Goose Up/Down blocks for every migration
- [x] All "Must NOT Have" guardrails respected
- [x] Verification script passes
- [x] `go test ./...` passes
- [x] API health endpoint still works
- [x] No Python/FastAPI/React code modified
