# Phase 3 Pipeline Migration Plan (Go + PostgreSQL)

## TL;DR

> **Quick Summary**: Migrate the Python + SQLite data pipeline to Go + PostgreSQL while maintaining exact output parity with the v0.5.3 baseline.
>
> **Deliverables**:
> - `docs/migration/phase-3-pipeline-migration-plan.md` - Design document
> - `cmd/baxi-cli/` - Pipeline CLI binary
> - `internal/pipeline/` - Pipeline runner, steps, and audit
> - `internal/ingest/` - CSV loading and raw table ingestion
> - `internal/alert/` - Alert rule engine (3 global + 6 dimensional rules)
> - `internal/recommendation/` - Recommendation and task generator
> - `internal/outbox/` - Outbox event generator
> - `scripts/migration/compare_pipeline_baseline.py` - Baseline comparison
> - 100% test coverage for all pipeline steps using testcontainers-go
>
> **Estimated Effort**: Large (4 waves, ~20 tasks)
> **Parallel Execution**: YES - 4-5 tasks per wave
> **Critical Path**: Wave 1 (scaffolding) → Wave 2 (dwd/mart) → Wave 3 (alerts/ops) → Wave 4 (integration)

---

## Context

### Original Request
Migrate the existing Python + SQLite pipeline (9 steps: db_init → db_ingest → db_calc_metrics → db_calc_dim_metrics → db_rule_engine → db_dim_rule_engine → db_gen_recommendations → db_export_feishu → db_trigger_simulator) to Go + PostgreSQL, maintaining exact baseline parity.

### Interview Summary
**Key Discussions**:
- **Alert Rules**: 5 global rules confirmed (GMV下降, 延迟飙升, 评分下降, 取消率上升, 卖家活跃度下降)
- **Dimensional Rules**: 6 additional dimensional rules discovered that generate 35/36 baseline alerts
- **Dead Rules**: 2 global rules (review_score_drop, seller_activation_gap) are dead code in Python - keep dead for parity
- **Test Strategy**: Write Go unit tests for each pipeline step using testcontainers-go
- **CLI**: New `cmd/baxi-cli/` binary for pipeline management

**Research Findings** (from Metis + explore):
- Go scaffold exists with pgx/v5, chi, zap
- Migrations 001-008 present (raw, dwd, mart, ops, gov, ai, audit tables)
- Audit tables (`audit.pipeline_run`, `audit.pipeline_step_run`) already in migration 008
- Baseline: 16 tables, 906,526 total rows
- Current Go codebase has 0 tests
- Python pipeline uses `INSERT OR REPLACE` / `INSERT OR IGNORE` for idempotency
- Deterministic alert IDs via SHA-256 hash
- Complex NULL handling: `safe_int`/`safe_float` return None on empty/unparseable

### Metis Review
**Identified Gaps** (addressed):
- **Gap**: Dimensional rules not in original requirements → **Resolved**: Included 6 dimensional rules in scope
- **Gap**: 2 dead rules in Python → **Resolved**: Keep dead for exact parity, document as known limitation
- **Gap**: Float precision differences Python vs Go → **Resolved**: Tolerance-based comparison (1e-9 relative, 1e-6 absolute)
- **Gap**: No test infrastructure → **Resolved**: testcontainers-go + PostgreSQL
- **Gap**: Idempotency model unclear → **Resolved**: Use `INSERT ... ON CONFLICT` matching Python's INSERT OR REPLACE/IGNORE

---

## Work Objectives

### Core Objective
Implement a Go + PostgreSQL pipeline that reproduces the exact output of the Python + SQLite v0.5.3 baseline, with agent-executed QA verifying row counts and metric values match within tolerance.

### Concrete Deliverables
1. `docs/migration/phase-3-pipeline-migration-plan.md` - Migration design document
2. `cmd/baxi-cli/main.go` - Pipeline CLI with `run` and `validate` commands
3. `internal/pipeline/runner.go` - Pipeline orchestration with audit logging
4. `internal/pipeline/step.go` - Step interface and registry
5. `internal/pipeline/audit.go` - Audit record management
6. `internal/ingest/csv_loader.go` - CSV loading with NULL handling
7. `internal/ingest/table_mapping.go` - CSV to raw table mapping
8. `internal/pipeline/steps/ingest_raw.go` - Step 1: CSV → raw tables
9. `internal/pipeline/steps/build_dwd.go` - Step 2: raw → dwd.order_level + dwd.item_level
10. `internal/pipeline/steps/build_metrics.go` - Step 3: dwd → mart.metric_daily + mart.metric_dimension_daily
11. `internal/pipeline/steps/detect_alerts.go` - Step 4: mart → ops.metric_alert (3 global + 6 dimensional)
12. `internal/pipeline/steps/generate_recommendations.go` - Step 5: alerts → ops.recommendation + ops.task
13. `internal/pipeline/steps/create_outbox.go` - Step 6: tasks/recs → ops.outbox_event
14. `internal/alert/rule.go` - Alert rule definitions and evaluation
15. `internal/alert/engine.go` - Alert rule engine
16. `internal/recommendation/generator.go` - Recommendation/task generator
17. `internal/outbox/repository.go` - Outbox event repository
18. `scripts/migration/compare_pipeline_baseline.py` - Baseline comparison script
19. Test files for each pipeline step using testcontainers-go

### Definition of Done
- [ ] All 9 old pipeline steps mapped to Go stages
- [ ] Row counts match baseline: dwd.order_level=99441, dwd.item_level=112650, metric_daily=634, metric_dimension_daily=693602, metric_alert=36, recommendation=36, task=36, outbox_event=36
- [ ] Metric values match within tolerance (1e-9 relative, 1e-6 absolute)
- [ ] `go test ./...` passes with >70% coverage for internal/pipeline/
- [ ] `make pipeline` runs full pipeline successfully
- [ ] `make pipeline-compare` outputs PASS for all tables
- [ ] API health endpoint still returns `{"status":"ok"}`
- [ ] Audit tables record every pipeline run and step

### Must Have
1. Exact row count parity for all 8 pipeline output tables
2. Metric value parity within tolerance
3. 3 working global alert rules + 6 dimensional rules
4. Dead rules remain dead (documented limitation)
5. Deterministic alert IDs (same SHA-256 algorithm as Python)
6. Idempotency via `INSERT ... ON CONFLICT`
7. Audit logging for every run and step
8. Go unit tests with testcontainers-go
9. Baseline comparison script

### Must NOT Have (Guardrails)
1. **No LLM integration** — rule_based only
2. **No Ontology Runtime**
3. **No React frontend changes**
4. **No full Go API** — only existing health endpoint
5. **No real Feishu/GitHub dispatch** — outbox_event in pending state only
6. **No auto-scheduling**
7. **No Python code changes**
8. **No YAML governance semantic changes**
9. **No business rule rewrites** — exact threshold parity
10. **No schema changes** — migrations 001-008 are frozen
11. **No dead rule fixes** — review_score_drop and seller_activation_gap stay dead
12. **No performance optimization** — baseline parity, not speed
13. **No new metrics/dimensions**
14. **No integration tests for existing API/worker**

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** - ALL verification is agent-executed.

### Test Decision
- **Infrastructure exists**: NO (0 existing Go tests)
- **Automated tests**: YES (Tests after implementation)
- **Framework**: testcontainers-go + PostgreSQL 15 + Go testing
- **Test data**: Baseline CSV samples loaded into test DB
- **Coverage target**: >70% for internal/pipeline/

### QA Policy
Every task MUST include agent-executed QA scenarios.
Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **Backend/DB**: Use Bash (psql) — Query tables, assert counts and values
- **CLI**: Use Bash — Run commands, assert exit codes and output
- **Tests**: Use Bash (go test) — Run test suites, assert PASS

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Foundation - can all start immediately):
├── Task 1: Migration design document [writing]
├── Task 2: Test infrastructure (testcontainers-go) [quick]
├── Task 3: Pipeline runner framework [quick]
├── Task 4: CLI scaffolding [quick]
└── Task 5: CSV loader infrastructure [quick]

Wave 2 (Core data transformations - depends on Wave 1):
├── Task 6: ingest_raw step + tests [unspecified-high]
├── Task 7: build_dwd_order_level step + tests [unspecified-high]
├── Task 8: build_dwd_item_level step + tests [unspecified-high]
├── Task 9: build_metric_daily step + tests [unspecified-high]
└── Task 10: build_metric_dimension_daily step + tests [unspecified-high]

Wave 3 (Alerts and ops - depends on Wave 2):
├── Task 11: detect_global_alerts step + tests [unspecified-high]
├── Task 12: detect_dimension_alerts step + tests [unspecified-high]
├── Task 13: generate_recommendations step + tests [unspecified-high]
├── Task 14: generate_tasks step + tests [unspecified-high]
└── Task 15: create_outbox_events step + tests [unspecified-high]

Wave 4 (Integration and validation - depends on Wave 3):
├── Task 16: Full pipeline integration test [unspecified-high]
├── Task 17: Baseline comparison script [quick]
└── Task 18: Makefile targets + documentation [quick]

Wave FINAL (After ALL tasks - 4 parallel reviews):
├── Task F1: Plan compliance audit (oracle)
├── Task F2: Code quality review (unspecified-high)
├── Task F3: Real manual QA (unspecified-high)
└── Task F4: Scope fidelity check (deep)
-> Present results -> Get explicit user okay

Critical Path: Wave 1 → Wave 2 → Wave 3 → Wave 4 → F1-F4 → user okay
Parallel Speedup: ~60% faster than sequential
Max Concurrent: 5 (Wave 2)
```

### Dependency Matrix

| Task | Blocked By | Blocks |
|------|-----------|--------|
| 1 (Design doc) | - | 2-18 (references) |
| 2 (Test infra) | - | 6-16 |
| 3 (Runner) | - | 6-16 |
| 4 (CLI) | - | 16-18 |
| 5 (CSV loader) | - | 6 |
| 6 (ingest_raw) | 2, 3, 5 | 7, 8 |
| 7 (dwd_order) | 2, 3, 6 | 9, 10 |
| 8 (dwd_item) | 2, 3, 6 | 9, 10 |
| 9 (metric_daily) | 2, 3, 7, 8 | 11, 12 |
| 10 (metric_dim) | 2, 3, 7, 8 | 11, 12 |
| 11 (global_alerts) | 2, 3, 9, 10 | 13, 14 |
| 12 (dim_alerts) | 2, 3, 9, 10 | 13, 14 |
| 13 (recommendations) | 2, 3, 11, 12 | 15 |
| 14 (tasks) | 2, 3, 11, 12 | 15 |
| 15 (outbox) | 2, 3, 13, 14 | 16 |
| 16 (Integration) | 4, 15 | 17, 18 |
| 17 (Compare script) | 16 | - |
| 18 (Makefile/docs) | 16 | - |

### Agent Dispatch Summary

- **Wave 1**: T1 → `writing`, T2-T5 → `quick`
- **Wave 2**: T6-T10 → `unspecified-high` (each with testcontainers tests)
- **Wave 3**: T11-T15 → `unspecified-high`
- **Wave 4**: T16-T18 → `unspecified-high` (T17-T18 → `quick`)
- **FINAL**: F1 → `oracle`, F2 → `unspecified-high`, F3 → `unspecified-high`, F4 → `deep`

---

## TODOs

- [x] 1. Migration Design Document

  **What to do**:
  - Create `docs/migration/phase-3-pipeline-migration-plan.md`
  - Document old 9-step pipeline to new Go stage mapping
  - Document CSV → raw table mapping for all 11 tables
  - Document raw → dwd.order_level SQL strategy
  - Document raw → dwd.item_level SQL strategy
  - Document dwd → mart.metric_daily calculation logic
  - Document dwd/mart → mart.metric_dimension_daily logic
  - Document mart → ops.metric_alert rule migration (3 global + 6 dimensional)
  - Document ops.metric_alert → recommendation/task generation
  - Document outbox_event generation logic
  - Document baseline comparison method
  - Document which fields allow explainable differences
  - Document known limitations (2 dead rules)

  **Must NOT do**:
  - Do not modify any code
  - Do not change any SQL migrations
  - Do not add new requirements beyond what's in this plan

  **Recommended Agent Profile**:
  - **Category**: `writing`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2-5)
  - **Blocks**: Tasks 6-18 (as reference)
  - **Blocked By**: None

  **References**:
  - `migration_baseline/table_counts.json` - Baseline row counts
  - `migration_baseline/pipeline_outputs/*.csv` - Baseline CSV samples
  - `migration_baseline/sqlite_schema.sql` - Old SQLite schema
  - `docs/migration/phase-2-schema-design.md` - Schema design decisions
  - `migrations/002_raw_tables.sql` - Raw table schema
  - `migrations/003_dwd_tables.sql` - DWD table schema
  - `migrations/004_mart_tables.sql` - Mart table schema
  - `migrations/005_ops_tables.sql` - Ops table schema
  - `pipeline/steps.py` - Old Python pipeline steps
  - `pipeline/runner.py` - Old Python pipeline runner

  **Acceptance Criteria**:
  - [ ] File exists at `docs/migration/phase-3-pipeline-migration-plan.md`
  - [ ] Contains all 12 required sections
  - [ ] References actual baseline files
  - [ ] Documents the 2 dead rules as known limitations

  **QA Scenarios**:

  ```
  Scenario: Design document completeness
    Tool: Bash
    Preconditions: None
    Steps:
      1. ls docs/migration/phase-3-pipeline-migration-plan.md
      2. grep -c "old 9" docs/migration/phase-3-pipeline-migration-plan.md
      3. grep -c "dwd.order_level" docs/migration/phase-3-pipeline-migration-plan.md
    Expected Result: File exists, contains pipeline mapping and dwd strategy
    Evidence: .sisyphus/evidence/task-1-design-doc-exists.txt
  ```

  **Commit**: YES
  - Message: `docs: add phase 3 pipeline migration plan`
  - Files: `docs/migration/phase-3-pipeline-migration-plan.md`

---

- [x] 2. Test Infrastructure (testcontainers-go)

  **What to do**:
  - Add `testcontainers-go` dependency to `go.mod`
  - Create `internal/testutil/db.go` - Test database helper
  - Create `internal/testutil/fixtures.go` - Load baseline CSV fixtures
  - Create `internal/testutil/tx.go` - Transaction wrapper for test isolation
  - Ensure tests can start PostgreSQL container, run migrations, load fixtures

  **Must NOT do**:
  - Do not write actual pipeline tests yet (that's Tasks 6-15)
  - Do not modify production code
  - Do not use sqlmock (decided: testcontainers-go)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 3-5)
  - **Blocks**: Tasks 6-16
  - **Blocked By**: None

  **References**:
  - `internal/db/postgres.go` - Existing pgxpool connection pattern
  - `go.mod` - Current dependencies
  - Migrations 001-008 in `migrations/`

  **Acceptance Criteria**:
  - [ ] `go mod tidy` succeeds
  - [ ] `go test ./internal/testutil/...` passes (at least one smoke test)
  - [ ] Test helper can start PostgreSQL container
  - [ ] Test helper can run Goose migrations
  - [ ] Test helper can load CSV fixtures into test DB

  **QA Scenarios**:

  ```
  Scenario: Test infrastructure smoke test
    Tool: Bash
    Preconditions: Docker available
    Steps:
      1. go test ./internal/testutil/... -v -count=1
    Expected Result: Tests pass, PostgreSQL container starts and stops
    Failure Indicators: Container fails to start, migrations fail
    Evidence: .sisyphus/evidence/task-2-test-infra-pass.txt
  ```

  **Commit**: YES (grouped with T3-T5)
  - Message: `test: add testcontainers-go infrastructure`
  - Files: `internal/testutil/`, `go.mod`, `go.sum`

---

- [x] 3. Pipeline Runner Framework

  **What to do**:
  - Create `internal/pipeline/runner.go`:
    - `Runner` struct with `DB *pgxpool.Pool`, `Steps []Step`, `Logger *zap.Logger`
    - `Run(ctx context.Context, input RunInput) error` method
    - Creates `audit.pipeline_run` record at start
    - Executes steps sequentially in a transaction
    - Writes `audit.pipeline_step_run` for each step
    - Handles failures: marks step and run as failed, rolls back
  - Create `internal/pipeline/step.go`:
    - `Step` interface: `Name() string`, `Run(ctx context.Context, tx pgx.Tx, input StepInput) error`
    - `StepInput` struct with pipeline run context
    - `StepOutput` struct with row counts
  - Create `internal/pipeline/audit.go`:
    - `AuditRecorder` for pipeline_run and pipeline_step_run
    - Generate deterministic run_id (UUID or timestamp-based)

  **Must NOT do**:
  - Do not implement actual step logic (Tasks 6-15)
  - Do not add scheduling or concurrency
  - Do not modify audit table schema

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1-2, 4-5)
  - **Blocks**: Tasks 6-16
  - **Blocked By**: None

  **References**:
  - `migrations/008_audit_tables.sql` - Audit table schema
  - `internal/db/postgres.go` - pgxpool usage
  - `internal/logger/logger.go` - Zap logger
  - `pipeline/runner.py` - Old Python runner for behavior reference

  **Acceptance Criteria**:
  - [ ] `go test ./internal/pipeline/... -run TestRunner` passes
  - [ ] Runner creates pipeline_run record
  - [ ] Runner creates pipeline_step_run records
  - [ ] Failed step marks run as failed
  - [ ] Success marks run as completed

  **QA Scenarios**:

  ```
  Scenario: Runner creates audit records
    Tool: Bash
    Preconditions: testcontainers DB running
    Steps:
      1. go test ./internal/pipeline/... -run TestRunnerAudit -v
    Expected Result: Test passes, audit tables populated
    Evidence: .sisyphus/evidence/task-3-runner-audit.txt

  Scenario: Runner handles step failure
    Tool: Bash
    Preconditions: testcontainers DB running
    Steps:
      1. go test ./internal/pipeline/... -run TestRunnerFailure -v
    Expected Result: Test passes, step and run marked failed
    Evidence: .sisyphus/evidence/task-3-runner-failure.txt
  ```

  **Commit**: YES (grouped with T2, T4-T5)
  - Message: `feat: add pipeline runner and audit framework`
  - Files: `internal/pipeline/runner.go`, `step.go`, `audit.go`

---

- [x] 4. CLI Scaffolding

  **What to do**:
  - Create `cmd/baxi-cli/main.go`:
    - Use `flag` or `cobra` for CLI parsing
    - Command: `pipeline run [--step <name>] [--data-dir <path>]`
    - Command: `pipeline validate` (compare against baseline)
    - Read DATABASE_URL from environment
    - Initialize pgxpool, logger, runner
    - Execute pipeline and print results
  - Create `cmd/baxi-cli/pipeline.go`:
    - Pipeline command handlers
    - Step registry mapping names to Step implementations

  **Must NOT do**:
  - Do not add commands beyond pipeline run/validate
  - Do not implement real step logic (just wiring)
  - Do not modify baxi-api or baxi-worker

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1-3, 5)
  - **Blocks**: Task 16-18
  - **Blocked By**: None

  **References**:
  - `cmd/baxi-api/main.go` - Existing CLI pattern
  - `internal/config/config.go` - Config loading
  - `internal/db/postgres.go` - DB connection

  **Acceptance Criteria**:
  - [ ] `go build ./cmd/baxi-cli` succeeds
  - [ ] `./baxi-cli --help` shows pipeline commands
  - [ ] `./baxi-cli pipeline run --step=nonexistent` returns error

  **QA Scenarios**:

  ```
  Scenario: CLI builds and shows help
    Tool: Bash
    Preconditions: None
    Steps:
      1. go build -o /tmp/baxi-cli ./cmd/baxi-cli
      2. /tmp/baxi-cli --help
    Expected Result: Exit code 0, help text contains pipeline commands
    Evidence: .sisyphus/evidence/task-4-cli-help.txt
  ```

  **Commit**: YES (grouped with T2-T3, T5)
  - Message: `feat: add baxi-cli pipeline command scaffold`
  - Files: `cmd/baxi-cli/main.go`, `cmd/baxi-cli/pipeline.go`

---

- [x] 5. CSV Loader Infrastructure

  **What to do**:
  - Create `internal/ingest/csv_loader.go`:
    - `CSVLoder` struct
    - `LoadCSV(path string, tableName string, tx pgx.Tx) error` method
    - Use PostgreSQL `COPY ... FROM STDIN` via pgx
    - Handle NULL values (empty strings → NULL, matching Python safe_int/safe_float)
    - Support configurable delimiter and encoding
  - Create `internal/ingest/table_mapping.go`:
    - Map CSV filenames to raw table names
    - Define column mappings for each table
    - List of 11 CSV files and their target raw tables:
      - olist_customers_dataset.csv → raw.olist_customers
      - olist_orders_dataset.csv → raw.olist_orders
      - olist_order_items_dataset.csv → raw.olist_order_items
      - olist_order_payments_dataset.csv → raw.olist_order_payments
      - olist_order_reviews_dataset.csv → raw.olist_order_reviews
      - olist_products_dataset.csv → raw.olist_products
      - olist_sellers_dataset.csv → raw.olist_sellers
      - olist_geolocation_dataset.csv → raw.olist_geolocation
      - product_category_name_translation.csv → raw.product_category_name_translation
      - olist_marketing_qualified_leads_dataset.csv → raw.marketing_qualified_leads
      - olist_closed_deals_dataset.csv → raw.closed_deals

  **Must NOT do**:
  - Do not implement TRUNCATE + COPY logic yet (that's Task 6)
  - Do not handle incremental ingestion (full reload only)
  - Do not add CSV validation beyond what Python did

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1-4)
  - **Blocks**: Task 6
  - **Blocked By**: None

  **References**:
  - `migrations/002_raw_tables.sql` - Raw table schemas
  - `pipeline/steps.py` - Old Python ingestion logic
  - `internal/db/postgres.go` - pgx usage

  **Acceptance Criteria**:
  - [ ] `go test ./internal/ingest/...` passes
  - [ ] Can load a test CSV into a test table
  - [ ] NULL handling matches Python behavior

  **QA Scenarios**:

  ```
  Scenario: CSV loader handles NULLs correctly
    Tool: Bash
    Preconditions: testcontainers DB running
    Steps:
      1. go test ./internal/ingest/... -run TestCSVLoader -v
    Expected Result: Test passes, empty strings loaded as NULL
    Evidence: .sisyphus/evidence/task-5-csv-loader.txt
  ```

  **Commit**: YES (grouped with T2-T4)
  - Message: `feat: add CSV loader and table mapping`
  - Files: `internal/ingest/csv_loader.go`, `table_mapping.go`

---

- [ ] 6. ingest_raw Step

  **What to do**:
  - Create `internal/pipeline/steps/ingest_raw.go`:
    - Implement `Step` interface
    - For each of 11 CSV files:
      1. Verify file exists in data-dir
      2. TRUNCATE target raw table
      3. COPY CSV via `internal/ingest` loader
      4. Count loaded rows
    - Return row counts in StepOutput
  - Write tests:
    - Test with testcontainers DB
    - Load sample CSVs
    - Assert row counts match expected

  **Must NOT do**:
  - Do not implement incremental ingestion
  - Do not add data validation beyond Python's behavior
  - Do not skip missing files (fail fast)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO (depends on Wave 1)
  - **Parallel Group**: Wave 2 (with Tasks 7-10)
  - **Blocks**: Tasks 7, 8
  - **Blocked By**: Tasks 2, 3, 5

  **References**:
  - `internal/ingest/csv_loader.go` - CSV loading
  - `internal/ingest/table_mapping.go` - Table mappings
  - `internal/pipeline/step.go` - Step interface
  - `migrations/002_raw_tables.sql` - Raw table schemas

  **Acceptance Criteria**:
  - [ ] Test passes: `go test ./internal/pipeline/steps/... -run TestIngestRaw`
  - [ ] All 11 tables loaded with correct row counts
  - [ ] NULL handling matches Python

  **QA Scenarios**:

  ```
  Scenario: Ingest raw tables from CSV
    Tool: Bash
    Preconditions: testcontainers DB, sample CSVs in testdata/
    Steps:
      1. go test ./internal/pipeline/steps/... -run TestIngestRaw -v
    Expected Result: Test passes, raw tables populated
    Evidence: .sisyphus/evidence/task-6-ingest-raw.txt

  Scenario: Row count parity for raw tables
    Tool: Bash (psql)
    Preconditions: Pipeline run completed
    Steps:
      1. psql $DATABASE_URL -c "SELECT COUNT(*) FROM raw.olist_orders"
      2. psql $DATABASE_URL -c "SELECT COUNT(*) FROM raw.olist_order_items"
    Expected Result: olist_orders=99441, olist_order_items=112650
    Evidence: .sisyphus/evidence/task-6-raw-counts.txt
  ```

  **Commit**: YES
  - Message: `feat: add ingest_raw pipeline step`
  - Files: `internal/pipeline/steps/ingest_raw.go`

---

- [ ] 7. build_dwd_order_level Step

  **What to do**:
  - Create `internal/pipeline/steps/build_dwd.go` (or separate file):
    - Implement SQL INSERT for dwd.order_level
    - Join: raw.olist_orders + raw.olist_customers + payment_agg + review_agg
    - Calculate delivery_days, is_late
    - Standardize order_status
    - Aggregate payments (payment_value, payment_installments)
    - Aggregate reviews (review_score, review_count)
    - Row count target: 99,441
  - Write tests with testcontainers

  **Must NOT do**:
  - Do not change calculation logic (exact parity)
  - Do not add new columns
  - Do not modify dwd schema

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Task 8)
  - **Parallel Group**: Wave 2 (with Tasks 6, 8-10)
  - **Blocks**: Tasks 9, 10
  - **Blocked By**: Tasks 2, 3, 6

  **References**:
  - `migrations/003_dwd_tables.sql` - dwd schema
  - `migration_baseline/pipeline_outputs/dwd_order_level_sample.csv` - Baseline
    - Actual baseline is: `data/interim/order_level_base.csv` (99,441 rows, 22 cols)
  - `pipeline/steps.py` - Old Python dwd logic

  **Acceptance Criteria**:
  - [ ] Test passes: `go test ./internal/pipeline/steps/... -run TestBuildDWDOrder`
  - [ ] Row count = 99,441
  - [ ] Sample values match baseline within tolerance

  **QA Scenarios**:

  ```
  Scenario: DWD order_level row count parity
    Tool: Bash (psql)
    Preconditions: ingest_raw completed
    Steps:
      1. psql $DATABASE_URL -c "SELECT COUNT(*) FROM dwd.order_level"
    Expected Result: 99441
    Evidence: .sisyphus/evidence/task-7-dwd-order-count.txt

  Scenario: DWD order_level sample values
    Tool: Bash (psql)
    Preconditions: dwd.order_level populated
    Steps:
      1. psql $DATABASE_URL -c "SELECT order_id, delivery_days, is_late FROM dwd.order_level LIMIT 5"
    Expected Result: Values match baseline sample
    Evidence: .sisyphus/evidence/task-7-dwd-order-sample.txt
  ```

  **Commit**: YES
  - Message: `feat: add build_dwd_order_level step`
  - Files: `internal/pipeline/steps/build_dwd.go`

---

- [ ] 8. build_dwd_item_level Step

  **What to do**:
  - Create SQL INSERT for dwd.item_level:
    - Join: raw.olist_order_items + raw.olist_orders + raw.olist_products + raw.olist_sellers + raw.product_category_name_translation
    - Calculate price, freight_value
    - Calculate shipping_limit vs delivered relationship
    - Granularity: order_id + order_item_id
    - Row count target: 112,650
  - Write tests with testcontainers

  **Must NOT do**:
  - Do not change join logic
  - Do not add new columns
  - Do not modify schema

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Task 7)
  - **Parallel Group**: Wave 2 (with Tasks 6-7, 9-10)
  - **Blocks**: Tasks 9, 10
  - **Blocked By**: Tasks 2, 3, 6

  **References**:
  - `migrations/003_dwd_tables.sql` - dwd schema
  - `data/interim/item_level_base.csv` - Baseline (112,650 rows)
  - `pipeline/steps.py` - Old Python item_level logic

  **Acceptance Criteria**:
  - [ ] Test passes: `go test ./internal/pipeline/steps/... -run TestBuildDWDItem`
  - [ ] Row count = 112,650
  - [ ] Sample values match baseline

  **QA Scenarios**:

  ```
  Scenario: DWD item_level row count parity
    Tool: Bash (psql)
    Preconditions: ingest_raw completed
    Steps:
      1. psql $DATABASE_URL -c "SELECT COUNT(*) FROM dwd.item_level"
    Expected Result: 112650
    Evidence: .sisyphus/evidence/task-8-dwd-item-count.txt
  ```

  **Commit**: YES (grouped with T7)
  - Message: `feat: add build_dwd_item_level step`
  - Files: `internal/pipeline/steps/build_dwd.go`

---

- [ ] 9. build_metric_daily Step

  **What to do**:
  - Create SQL INSERT for mart.metric_daily:
    - Aggregate dwd.order_level and dwd.item_level by date
    - Calculate: orders_count, delivered_orders_count, cancelled_orders_count, gmv, payment_value, avg_review_score, avg_delivery_days, late_delivery_rate, cancel_rate, active_sellers, active_customers, items_count, freight_value
    - Row count target: 634
    - Use COALESCE, NULLIF matching Python patterns
  - Write tests

  **Must NOT do**:
  - Do not change aggregation logic
  - Do not add new metrics
  - Do not change date granularity

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO (depends on dwd steps)
  - **Parallel Group**: Wave 2 (with Tasks 6-8, 10)
  - **Blocks**: Tasks 11, 12
  - **Blocked By**: Tasks 2, 3, 7, 8

  **References**:
  - `migrations/004_mart_tables.sql` - mart schema
  - `migration_baseline/pipeline_outputs/metric_daily_sample.csv` - Baseline
  - `pipeline/steps.py` - Old Python metric calculation

  **Acceptance Criteria**:
  - [ ] Test passes: `go test ./internal/pipeline/steps/... -run TestBuildMetricDaily`
  - [ ] Row count = 634
  - [ ] GMV total matches baseline within tolerance

  **QA Scenarios**:

  ```
  Scenario: metric_daily row count and GMV parity
    Tool: Bash (psql)
    Preconditions: dwd steps completed
    Steps:
      1. psql $DATABASE_URL -c "SELECT COUNT(*) FROM mart.metric_daily"
      2. psql $DATABASE_URL -c "SELECT SUM(gmv) FROM mart.metric_daily"
    Expected Result: Count=634, GMV matches baseline within 1e-6
    Evidence: .sisyphus/evidence/task-9-metric-daily.txt
  ```

  **Commit**: YES
  - Message: `feat: add build_metric_daily step`
  - Files: `internal/pipeline/steps/build_metrics.go`

---

- [ ] 10. build_metric_dimension_daily Step

  **What to do**:
  - Create SQL INSERT for mart.metric_dimension_daily:
    - Aggregate by date + dimension (seller, category, region, product, customer_segment)
    - Row count target: 693,602
    - This is the largest table - SQL must be efficient
  - Write tests (may use smaller fixture for speed)

  **Must NOT do**:
  - Do not change dimension definitions
  - Do not add new dimensions
  - Do not merge into metric_snapshot (keep separate table)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO (depends on dwd steps)
  - **Parallel Group**: Wave 2 (with Tasks 6-9)
  - **Blocks**: Tasks 11, 12
  - **Blocked By**: Tasks 2, 3, 7, 8

  **References**:
  - `migrations/004_mart_tables.sql` - mart schema
  - `migration_baseline/pipeline_outputs/metric_dimension_daily_sample.csv` - Baseline
  - `pipeline/steps.py` - Old Python dimension metric calculation

  **Acceptance Criteria**:
  - [ ] Test passes: `go test ./internal/pipeline/steps/... -run TestBuildMetricDim`
  - [ ] Row count = 693,602
  - [ ] Dimension counts match baseline

  **QA Scenarios**:

  ```
  Scenario: metric_dimension_daily row count parity
    Tool: Bash (psql)
    Preconditions: dwd steps completed
    Steps:
      1. psql $DATABASE_URL -c "SELECT COUNT(*) FROM mart.metric_dimension_daily"
    Expected Result: 693602
    Evidence: .sisyphus/evidence/task-10-metric-dim-count.txt
  ```

  **Commit**: YES (grouped with T9)
  - Message: `feat: add build_metric_dimension_daily step`
  - Files: `internal/pipeline/steps/build_metrics.go`

---

- [ ] 11. detect_global_alerts Step

  **What to do**:
  - Create `internal/alert/rule.go` - Rule definitions
  - Create `internal/alert/engine.go` - Rule evaluation engine
  - Implement 3 working global rules:
    1. gmv_drop (GMV下降)
    2. late_delivery_spike (延迟飙升)
    3. cancel_rate_spike (取消率上升)
  - Keep 2 dead rules defined but never triggered:
    4. review_score_drop (评分下降) - dead
    5. seller_activation_gap (卖家活跃度下降) - dead
  - Generate exactly 1 global alert (matching baseline)
  - Use deterministic SHA-256 IDs (same algorithm as Python)
  - INSERT ... ON CONFLICT for idempotency
  - Write tests

  **Must NOT do**:
  - Do not fix dead rules (exact parity)
  - Do not change rule thresholds
  - Do not add new rules

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Task 12)
  - **Parallel Group**: Wave 3 (with Tasks 12-15)
  - **Blocks**: Tasks 13, 14
  - **Blocked By**: Tasks 2, 3, 9, 10

  **References**:
  - `migrations/005_ops_tables.sql` - ops schema
  - `migration_baseline/pipeline_outputs/alert_events_sample.csv` - Baseline
  - `config/alert_rules.yml` - Alert rule YAML configs
  - `services/alert_service.py` - Old Python alert logic

  **Acceptance Criteria**:
  - [ ] Test passes: `go test ./internal/alert/... -run TestGlobalAlerts`
  - [ ] Exactly 1 global alert generated
  - [ ] Alert severity and rule_id match baseline

  **QA Scenarios**:

  ```
  Scenario: Global alert count parity
    Tool: Bash (psql)
    Preconditions: metric steps completed
    Steps:
      1. psql $DATABASE_URL -c "SELECT COUNT(*) FROM ops.metric_alert WHERE object_type='global'"
    Expected Result: 1
    Evidence: .sisyphus/evidence/task-11-global-alerts.txt
  ```

  **Commit**: YES
  - Message: `feat: add global alert detection`
  - Files: `internal/alert/rule.go`, `internal/alert/engine.go`

---

- [ ] 12. detect_dimension_alerts Step

  **What to do**:
  - Implement 6 dimensional rules:
    1. seller_late_delivery_spike (seller维度, >25%)
    2. seller_review_score_drop (seller维度, <3.5)
    3. category_gmv_drop (category维度, 下降>20%)
    4. category_low_review_cluster (category维度, >15%)
    5. region_cancel_rate_spike (region维度, >5%)
    6. region_late_delivery_spike (region维度, >20%)
  - Implement alert suppression:
    - Max 50 alerts per run
    - Max 5 per dimension value
  - Implement impact scoring: severity_weight × sample_size
  - Generate 35 dimensional alerts (matching baseline)
  - Use same deterministic ID algorithm as Python
  - Write tests

  **Must NOT do**:
  - Do not change suppression logic
  - Do not change impact scoring
  - Do not add new dimensional rules

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Task 11)
  - **Parallel Group**: Wave 3 (with Tasks 11, 13-15)
  - **Blocks**: Tasks 13, 14
  - **Blocked By**: Tasks 2, 3, 9, 10

  **References**:
  - `migrations/005_ops_tables.sql` - ops schema
  - `migration_baseline/pipeline_outputs/alert_events_sample.csv` - Baseline
  - `config/alert_rules.yml` - Dimensional rule configs
  - `pipeline/steps.py` - Dimensional rule engine logic

  **Acceptance Criteria**:
  - [ ] Test passes: `go test ./internal/alert/... -run TestDimensionAlerts`
  - [ ] Exactly 35 dimensional alerts generated
  - [ ] Alert distribution by dimension matches baseline

  **QA Scenarios**:

  ```
  Scenario: Dimensional alert count parity
    Tool: Bash (psql)
    Preconditions: metric steps completed
    Steps:
      1. psql $DATABASE_URL -c "SELECT COUNT(*) FROM ops.metric_alert WHERE object_type!='global'"
      2. psql $DATABASE_URL -c "SELECT object_type, COUNT(*) FROM ops.metric_alert GROUP BY object_type"
    Expected Result: 35 non-global alerts, distribution matches baseline
    Evidence: .sisyphus/evidence/task-12-dim-alerts.txt
  ```

  **Commit**: YES (grouped with T11)
  - Message: `feat: add dimensional alert detection`
  - Files: `internal/alert/engine.go`, `internal/pipeline/steps/detect_alerts.go`

---

- [ ] 13. generate_recommendations Step

  **What to do**:
  - Create `internal/recommendation/generator.go`:
    - Read ops.metric_alert
    - Generate ops.recommendation records
    - source_type = 'rule_based'
    - Use template-based generation (same as Python)
    - Row count target: 36
  - Write tests

  **Must NOT do**:
  - Do not call LLM
  - Do not change template logic
  - Do not add new recommendation types

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Task 14)
  - **Parallel Group**: Wave 3 (with Tasks 11-12, 14-15)
  - **Blocks**: Task 15
  - **Blocked By**: Tasks 2, 3, 11, 12

  **References**:
  - `migrations/005_ops_tables.sql` - ops schema
  - `migration_baseline/pipeline_outputs/recommendations_sample.csv` - Baseline
  - `pipeline/steps.py` - Recommendation generation logic

  **Acceptance Criteria**:
  - [ ] Test passes: `go test ./internal/recommendation/...`
  - [ ] Row count = 36
  - [ ] source_type = 'rule_based' for all

  **QA Scenarios**:

  ```
  Scenario: Recommendation count parity
    Tool: Bash (psql)
    Preconditions: alert steps completed
    Steps:
      1. psql $DATABASE_URL -c "SELECT COUNT(*) FROM ops.recommendation"
    Expected Result: 36
    Evidence: .sisyphus/evidence/task-13-recommendations.txt
  ```

  **Commit**: YES
  - Message: `feat: add recommendation generation`
  - Files: `internal/recommendation/generator.go`

---

- [ ] 14. generate_tasks Step

  **What to do**:
  - Generate ops.task records from alerts/recommendations
  - Row count target: 36
  - Status: pending
  - Write tests

  **Must NOT do**:
  - Do not add task assignment logic
  - Do not add priority scoring

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Task 13)
  - **Parallel Group**: Wave 3 (with Tasks 11-13, 15)
  - **Blocks**: Task 15
  - **Blocked By**: Tasks 2, 3, 11, 12

  **References**:
  - `migrations/005_ops_tables.sql` - ops schema
  - `migration_baseline/pipeline_outputs/tasks_sample.csv` - Baseline
  - `pipeline/steps.py` - Task generation logic

  **Acceptance Criteria**:
  - [ ] Test passes
  - [ ] Row count = 36

  **QA Scenarios**:

  ```
  Scenario: Task count parity
    Tool: Bash (psql)
    Preconditions: alert steps completed
    Steps:
      1. psql $DATABASE_URL -c "SELECT COUNT(*) FROM ops.task"
    Expected Result: 36
    Evidence: .sisyphus/evidence/task-14-tasks.txt
  ```

  **Commit**: YES (grouped with T13)
  - Message: `feat: add task generation`
  - Files: `internal/recommendation/generator.go`

---

- [ ] 15. create_outbox_events Step

  **What to do**:
  - Create `internal/outbox/repository.go`:
    - Read ops.task and ops.recommendation
    - Generate ops.outbox_event records
    - Status: pending
    - payload as JSONB
    - Row count target: 36
  - Write tests

  **Must NOT do**:
  - Do not implement real worker dispatch
  - Do not call Feishu/GitHub APIs
  - Do not change outbox schema

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO (depends on tasks/recs)
  - **Parallel Group**: Wave 3 (with Tasks 11-14)
  - **Blocks**: Task 16
  - **Blocked By**: Tasks 2, 3, 13, 14

  **References**:
  - `migrations/005_ops_tables.sql` - ops schema
  - `migration_baseline/pipeline_outputs/outbox_sample.csv` - Baseline
  - `pipeline/steps.py` - Outbox generation logic

  **Acceptance Criteria**:
  - [ ] Test passes: `go test ./internal/outbox/...`
  - [ ] Row count = 36
  - [ ] All records have status='pending'

  **QA Scenarios**:

  ```
  Scenario: Outbox event count parity
    Tool: Bash (psql)
    Preconditions: task/rec steps completed
    Steps:
      1. psql $DATABASE_URL -c "SELECT COUNT(*) FROM ops.outbox_event"
      2. psql $DATABASE_URL -c "SELECT status, COUNT(*) FROM ops.outbox_event GROUP BY status"
    Expected Result: Count=36, all status='pending'
    Evidence: .sisyphus/evidence/task-15-outbox.txt
  ```

  **Commit**: YES
  - Message: `feat: add outbox event generation`
  - Files: `internal/outbox/repository.go`, `internal/pipeline/steps/create_outbox.go`

---

- [ ] 16. Full Pipeline Integration Test

  **What to do**:
  - Create integration test that runs all steps end-to-end
  - Use testcontainers-go with fresh DB
  - Load real CSV fixtures (or representative subset)
  - Execute full pipeline
  - Assert all row counts match baseline
  - Assert audit tables populated

  **Must NOT do**:
  - Do not use production database
  - Do not skip steps

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO (depends on all steps)
  - **Parallel Group**: Wave 4 (with Tasks 17-18)
  - **Blocks**: Tasks 17, 18
  - **Blocked By**: Tasks 4, 11-15

  **References**:
  - All previous tasks
  - `migration_baseline/table_counts.json`

  **Acceptance Criteria**:
  - [ ] Test passes: `go test ./... -run TestFullPipeline`
  - [ ] All 8 output tables have correct row counts
  - [ ] audit.pipeline_run has 1 completed record
  - [ ] audit.pipeline_step_run has 9 completed records

  **QA Scenarios**:

  ```
  Scenario: Full pipeline execution
    Tool: Bash
    Preconditions: testcontainers available
    Steps:
      1. go test ./... -run TestFullPipeline -v -timeout 10m
    Expected Result: Test passes in < 5 minutes
    Evidence: .sisyphus/evidence/task-16-full-pipeline.txt
  ```

  **Commit**: YES
  - Message: `test: add full pipeline integration test`
  - Files: `internal/pipeline/integration_test.go`

---

- [ ] 17. Baseline Comparison Script

  **What to do**:
  - Create `scripts/migration/compare_pipeline_baseline.py`:
    - Read `migration_baseline/table_counts.json`
    - Query PostgreSQL for actual row counts
    - Compare old_count vs new_count
    - Output PASS/FAIL for each table
    - Optional: Read baseline CSVs and compare sample values
  - Or implement as Go command: `baxi-cli pipeline validate`

  **Must NOT do**:
  - Do not modify baseline files
  - Do not require manual inspection

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Task 18)
  - **Parallel Group**: Wave 4 (with Tasks 16, 18)
  - **Blocks**: None
  - **Blocked By**: Task 16

  **References**:
  - `migration_baseline/table_counts.json`
  - `migration_baseline/pipeline_outputs/*.csv`

  **Acceptance Criteria**:
  - [ ] Script runs successfully
  - [ ] All 8 tables show PASS
  - [ ] Outputs detailed diff on failure

  **QA Scenarios**:

  ```
  Scenario: Baseline comparison
    Tool: Bash
    Preconditions: Pipeline has run
    Steps:
      1. make pipeline-compare
    Expected Result: All tables PASS
    Evidence: .sisyphus/evidence/task-17-baseline-compare.txt
  ```

  **Commit**: YES
  - Message: `chore: add pipeline baseline comparison script`
  - Files: `scripts/migration/compare_pipeline_baseline.py`

---

- [ ] 18. Makefile Targets and Documentation

  **What to do**:
  - Add to `Makefile`:
    - `pipeline` - run full pipeline
    - `pipeline-ingest` - run ingest step
    - `pipeline-dwd` - run dwd step
    - `pipeline-metrics` - run metrics step
    - `pipeline-compare` - compare baseline
    - `test-pipeline` - run Go pipeline tests
  - Update README with pipeline usage
  - Ensure `make api` still works

  **Must NOT do**:
  - Do not break existing Makefile targets
  - Do not modify Python targets

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Task 17)
  - **Parallel Group**: Wave 4 (with Tasks 16-17)
  - **Blocks**: None
  - **Blocked By**: Task 16

  **References**:
  - `Makefile` - Existing targets
  - `README.md` - Existing documentation

  **Acceptance Criteria**:
  - [ ] `make pipeline` runs successfully
  - [ ] `make pipeline-compare` outputs PASS
  - [ ] `make api` still works (health endpoint returns ok)
  - [ ] `go test ./...` passes

  **QA Scenarios**:

  ```
  Scenario: Makefile targets work
    Tool: Bash
    Preconditions: DB running
    Steps:
      1. make pipeline DATA_DIR=./data/raw
      2. make pipeline-compare
    Expected Result: Both succeed, compare shows PASS
    Evidence: .sisyphus/evidence/task-18-makefile.txt
  ```

  **Commit**: YES
  - Message: `chore: add pipeline Makefile targets`
  - Files: `Makefile`, `README.md`

---

## Final Verification Wave

> **4 review agents run in PARALLEL. ALL must APPROVE.**
> Present consolidated results to user and get explicit "okay" before completing.

- [ ] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, curl endpoint, run command). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist in .sisyphus/evidence/. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [ ] F2. **Code Quality Review** — `unspecified-high`
  Run `go test ./...`, `go vet ./...`, check for `as any`, empty catches, `fmt.Println` in prod, commented-out code, unused imports. Check AI slop: excessive comments, over-abstraction, generic names.
  Output: `Build [PASS/FAIL] | Lint [PASS/FAIL] | Tests [N pass/N fail] | Files [N clean/N issues] | VERDICT`

- [ ] F3. **Real Manual QA** — `unspecified-high`
  Start from clean state. Execute EVERY QA scenario from EVERY task — follow exact steps, capture evidence. Test cross-task integration. Test edge cases: empty state, invalid input.
  Output: `Scenarios [N/N pass] | Integration [N/N] | Edge Cases [N tested] | VERDICT`

- [ ] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual diff (git log/diff). Verify 1:1 — everything in spec was built, nothing beyond spec was built. Check "Must NOT do" compliance. Detect cross-task contamination.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

| Commit | Message | Files |
|--------|---------|-------|
| 1 | `docs: add phase 3 pipeline migration plan` | `docs/migration/phase-3-pipeline-migration-plan.md` |
| 2 | `test: add testcontainers-go infrastructure` | `internal/testutil/`, `go.mod` |
| 3 | `feat: add pipeline runner and audit framework` | `internal/pipeline/runner.go`, `step.go`, `audit.go` |
| 4 | `feat: add baxi-cli pipeline command scaffold` | `cmd/baxi-cli/main.go`, `pipeline.go` |
| 5 | `feat: add CSV loader and table mapping` | `internal/ingest/csv_loader.go`, `table_mapping.go` |
| 6 | `feat: add ingest_raw pipeline step` | `internal/pipeline/steps/ingest_raw.go` |
| 7 | `feat: add build_dwd_order_level step` | `internal/pipeline/steps/build_dwd.go` |
| 8 | `feat: add build_dwd_item_level step` | `internal/pipeline/steps/build_dwd.go` |
| 9 | `feat: add build_metric_daily step` | `internal/pipeline/steps/build_metrics.go` |
| 10 | `feat: add build_metric_dimension_daily step` | `internal/pipeline/steps/build_metrics.go` |
| 11 | `feat: add global alert detection` | `internal/alert/rule.go`, `engine.go` |
| 12 | `feat: add dimensional alert detection` | `internal/alert/engine.go`, `steps/detect_alerts.go` |
| 13 | `feat: add recommendation generation` | `internal/recommendation/generator.go` |
| 14 | `feat: add task generation` | `internal/recommendation/generator.go` |
| 15 | `feat: add outbox event generation` | `internal/outbox/repository.go`, `steps/create_outbox.go` |
| 16 | `test: add full pipeline integration test` | `internal/pipeline/integration_test.go` |
| 17 | `chore: add pipeline baseline comparison script` | `scripts/migration/compare_pipeline_baseline.py` |
| 18 | `chore: add pipeline Makefile targets` | `Makefile`, `README.md` |

---

## Success Criteria

### Verification Commands
```bash
# 1. Full pipeline run
make pipeline DATA_DIR=./data/raw

# 2. Baseline comparison
make pipeline-compare
# Expected: All tables PASS

# 3. Go tests
go test ./...
# Expected: PASS (coverage > 70% for internal/pipeline/)

# 4. API health check
make api
# In another terminal:
curl http://localhost:8080/api/v1/health
# Expected: {"status":"ok","service":"baxi-api"}

# 5. Row count parity
psql $DATABASE_URL -c "SELECT COUNT(*) FROM dwd.order_level"        # 99441
psql $DATABASE_URL -c "SELECT COUNT(*) FROM dwd.item_level"         # 112650
psql $DATABASE_URL -c "SELECT COUNT(*) FROM mart.metric_daily"      # 634
psql $DATABASE_URL -c "SELECT COUNT(*) FROM mart.metric_dimension_daily"  # 693602
psql $DATABASE_URL -c "SELECT COUNT(*) FROM ops.metric_alert"       # 36
psql $DATABASE_URL -c "SELECT COUNT(*) FROM ops.recommendation"     # 36
psql $DATABASE_URL -c "SELECT COUNT(*) FROM ops.task"               # 36
psql $DATABASE_URL -c "SELECT COUNT(*) FROM ops.outbox_event"       # 36
```

### Final Checklist
- [ ] All 18 tasks completed
- [ ] All "Must Have" present and verified
- [ ] All "Must NOT Have" absent
- [ ] All tests pass with >70% coverage
- [ ] Baseline comparison shows PASS for all 8 tables
- [ ] API health endpoint still works
- [ ] No modifications to Python code, React frontend, or YAML configs
- [ ] 2 dead rules documented as known limitations
- [ ] Audit tables record every run and step

