# Bug Fix Plan: Environment, Schema, Qoder, Pipeline, and Diagnostics

## TL;DR

> **Quick Summary**: Fix 6 critical/high/medium bugs in the baxi FastAPI + SQLite system: env var parsing (Critical), schema/migration mismatch (High), Qoder task filtering (High), Qoder diagnosis crash (Medium), pipeline dry-run semantics (Medium), and 500 logging path (Medium). Include regression tests and targeted ruff cleanup.
>
> **Deliverables**:
> - Fixed env var parsing in `api/main.py`
> - Updated `sql/schema.sql` with missing columns
> - Fixed migration `010_add_foreign_keys.sql`
> - Extended `_build_conditions` for multi-status filtering
> - Fixed Qoder diagnosis to pass real request_id
> - Fixed pipeline dry-run default to match documented behavior
> - Fixed unhandled 500 logging to use `api.error` logger (dual logging)
> - Ruff-clean modified files
> - Regression tests for all fixes
>
> **Estimated Effort**: Medium (8-10 tasks, 2 waves + final verification)
> **Parallel Execution**: YES - Wave 1: 5 tasks, Wave 2: 2 tasks
> **Critical Path**: Task 2 (Schema) → Task 7 (Tests) → Final Verification

---

## Context

### Original Request
Fix 6 findings from a security and correctness audit of the baxi v0.5.3 system.

### Interview Summary
**Key Decisions** (made by planner based on code analysis + Metis review):
- **Env vars**: Inline explicit truthy parsing (not extracting utility - too risky)
- **Schema**: Additive changes only - code is correct, schema.sql is stale
- **Migration 010**: Fix in-place, do NOT automate (structurally incompatible with `_apply_migration`'s `;` splitting)
- **Qoder status**: Extend `_build_conditions` to support `IN` clauses for list values
- **Qoder diagnosis**: Pass actual request_id from context (not just type fix)
- **Dry-run**: Change runner.py default to `True`, audit all callers, update call sites
- **Logging**: Use `"api.error"` logger (confirmed name), keep dual logging to `"api"`
- **Ruff**: Only fix files modified by bug fixes (not full repo)

### Metis Review
**Identified Gaps** (addressed in this plan):
- `diagnose_by_request_id(None)` design: Passing None always returns empty results because log entries have UUID request_ids. Fix: pass actual request_id.
- Migration 010 cannot be automated: `_apply_migration` splits on `;`, but 010 contains multiple semicolons. Fix the SQL file but don't hook into automation.
- `run_pipeline()` caller audit: Must check all callers before changing default. Included in plan.
- `_build_conditions` limitation: Single-value only. Plan includes extending for `IN` clauses.
- Ruff scope: 16,247 total issues vs 43 in target dirs. Plan scopes to modified files only.
- Error logger name: Confirmed `"api.error"` (not `"error"`).

---

## Work Objectives

### Core Objective
Fix 6 verified bugs that affect security (Swagger exposure, error detail leakage), data integrity (schema mismatch, migration failure), functionality (missing tasks, diagnosis crash), and CLI safety (destructive default).

### Concrete Deliverables
- `api/main.py`: Truthy env var parsing + correct error logger
- `sql/schema.sql`: 5 missing columns added (alert_events: 3, action_tasks: 2)
- `sql/migrations/010_add_foreign_keys.sql`: Fixed column count
- `services/qoder_service.py`: Multi-status filtering + real request_id passing
- `services/_query_utils.py` (or equivalent): `IN` clause support
- `api/schemas_qoder.py`: Handle empty diagnosis list
- `pipeline/runner.py`: `dry_run=True` default
- `scripts/run_db_pipeline.py`: `--apply` flag semantics
- All caller sites of `run_pipeline()`: Explicit `dry_run` args
- Regression tests in `tests/`
- Ruff-clean modified files

### Definition of Done
- [ ] All 6 findings fixed with evidence
- [ ] `pytest` passes: 435+ tests, coverage >= 88%
- [ ] `npm test` passes: 33 tests
- [ ] `npm run build` passes
- [ ] `ruff check` passes on all modified files
- [ ] All QA scenarios executed and evidence captured

### Must Have
- Explicit truthy parsing for ENABLE_DOCS and DEBUG
- Schema.sql includes all columns dimensional engine inserts
- Migration 010 works on DB with migration 006 applied
- Qoder context returns todo/in_progress tasks
- Qoder diagnosis passes real request_id, handles empty results
- Pipeline defaults to dry-run unless `--apply`
- Unhandled 500s appear in error.log
- Regression tests for each fix

### Must NOT Have (Guardrails)
- **MUST NOT** add new env var parsing utility (inline fix only)
- **MUST NOT** restructure migration system (no new migrations, no automation of 010)
- **MUST NOT** change `diagnose_by_request_id` signature (fix the caller only)
- **MUST NOT** modify `logging_config.py`
- **MUST NOT** add new features, endpoints, or tables
- **MUST NOT** fix bugs outside the 6 findings
- **MUST NOT** run full-repo ruff cleanup (only modified files)
- **MUST NOT** refactor `_build_conditions` into an ORM (only add `IN` support)
- **MUST NOT** reorder columns or change DEFAULTs in schema

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** - ALL verification is agent-executed.

### Test Decision
- **Infrastructure exists**: YES
- **Automated tests**: YES (Tests after implementation)
- **Framework**: pytest 9.0.3 + pytest-cov 7.1.0
- **Coverage target**: Maintain >= 88%

### QA Policy
Every task includes agent-executed QA scenarios. Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **Backend/API**: Bash (curl, pytest, sqlite3)
- **Scripts/CLI**: Bash (python3 script.py, assert exit code and output)
- **Library/Module**: Bash (python3 -c "import module; assert ...")

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Start Immediately - 5 independent tasks):
├── Task 1: Fix api/main.py env vars + logging [quick]
├── Task 2: Sync schema.sql + fix migration 010 [quick]
├── Task 3: Fix Qoder status filtering [quick]
├── Task 4: Fix Qoder diagnosis request_id passing [quick]
├── Task 5: Fix pipeline dry-run semantics [quick]

Wave 2 (After Wave 1 - tests + cleanup):
├── Task 6: Add regression tests for all fixes [unspecified-high]
├── Task 7: Fix ruff issues on modified files [quick]

Wave FINAL (After ALL tasks - 4 parallel reviews):
├── Task F1: Plan compliance audit (oracle)
├── Task F2: Code quality review (unspecified-high)
├── Task F3: Real manual QA (unspecified-high)
├── Task F4: Scope fidelity check (deep)
-> Present results -> Get explicit user okay
```

### Dependency Matrix

| Task | Blocked By | Blocks |
|------|-----------|--------|
| 1 | - | 6, 7 |
| 2 | - | 6 |
| 3 | - | 6 |
| 4 | - | 6 |
| 5 | - | 6 |
| 6 | 1, 2, 3, 4, 5 | F1-F4 |
| 7 | 1, 2, 3, 4, 5 | F1-F4 |
| F1-F4 | 6, 7 | - |

### Agent Dispatch Summary

- **Wave 1**: **5** tasks → `quick` category
- **Wave 2**: **2** tasks → Task 6: `unspecified-high`, Task 7: `quick`
- **FINAL**: **4** tasks → F1: `oracle`, F2: `unspecified-high`, F3: `unspecified-high`, F4: `deep`

---

## TODOs

- [x] 1. **Fix api/main.py - Env var truthy parsing + error logger**

  **What to do**:
  - Fix `ENABLE_DOCS` check at ~line 270: Replace `if os.environ.get("ENABLE_DOCS"):` with explicit truthy parsing that handles `"0"`, `"false"`, `"False"`, `"FALSE"`, `"no"`, `"NO"`, `""`, and `None` as disabled. Only `"1"`, `"true"`, `"True"`, `"TRUE"`, `"yes"`, `"YES"` should enable.
  - Fix `DEBUG` check at ~line 375: Same pattern. When `DEBUG=0` or `DEBUG=false`, 500 responses must NOT include exception traceback. When `DEBUG=1` or `DEBUG=true`, include traceback.
  - Fix unhandled 500 logging at ~line 365: Change `logging.getLogger("api").error(...)` to `logging.getLogger("api.error").error(...)`. ALSO keep logging to `"api"` logger for dual visibility (add a second line or use both loggers).
  - Add clear comments explaining why `.lower() in (...)` is used instead of bare `os.environ.get`.

  **Must NOT do**:
  - Extract a utility function for env var parsing
  - Modify `logging_config.py`
  - Change any other env var parsing (e.g., `CORS_ORIGINS`, `TRUSTED_PROXY_IPS`)
  - Remove logging to `api` logger entirely

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - Reason: Straightforward changes in a single file with clear acceptance criteria.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: Task 6 (tests), Task 7 (ruff)
  - **Blocked By**: None

  **References**:
  - `api/main.py:270` - Current ENABLE_DOCS check: `if os.environ.get("ENABLE_DOCS"):`
  - `api/main.py:375` - Current DEBUG check: `if os.environ.get("DEBUG"):`
  - `api/main.py:365` - Current error logging: `logging.getLogger("api").error(...)`
  - `logging_config.py:73` - Logger registration: `"api.error"` is the error logger name
  - `.env.example:29` - Example showing `ENABLE_DOCS=0` and `DEBUG=0`

  **Acceptance Criteria**:
  - [ ] `ENABLE_DOCS=0` → `/docs` returns 404 (or docs endpoint disabled)
  - [ ] `ENABLE_DOCS=false` → `/docs` returns 404
  - [ ] `ENABLE_DOCS=1` → `/docs` returns 200
  - [ ] `ENABLE_DOCS` unset → `/docs` returns 404 (default disabled)
  - [ ] `DEBUG=0` → 500 response body contains `"Internal server error"` without traceback
  - [ ] `DEBUG=1` → 500 response body contains traceback details
  - [ ] `DEBUG=true` (lowercase) works correctly
  - [ ] Unhandled 500 entries appear in `logs/api/error.log`
  - [ ] Unhandled 500 entries ALSO appear in `logs/api/api.log` (dual logging)

  **QA Scenarios**:

  ```
  Scenario: ENABLE_DOCS=0 disables Swagger docs
    Tool: Bash (curl)
    Preconditions: Set env ENABLE_DOCS=0, start server
    Steps:
      1. curl -s -o /dev/null -w "%{http_code}" http://localhost:8765/docs
    Expected Result: HTTP 404
    Evidence: .sisyphus/evidence/task-1-docs-disabled.txt

  Scenario: DEBUG=0 hides exception details in 500 responses
    Tool: Bash (curl + python)
    Preconditions: Set env DEBUG=0, trigger an unhandled exception endpoint
    Steps:
      1. Find or create an endpoint that raises an unhandled exception
      2. curl the endpoint and capture response body
    Expected Result: Response contains "Internal server error" without traceback
    Evidence: .sisyphus/evidence/task-1-debug-zero.txt

  Scenario: DEBUG=1 shows exception details in 500 responses
    Tool: Bash (curl + python)
    Preconditions: Set env DEBUG=1, trigger same unhandled exception endpoint
    Steps:
      1. curl the endpoint and capture response body
    Expected Result: Response contains traceback / exception type / line numbers
    Evidence: .sisyphus/evidence/task-1-debug-one.txt

  Scenario: Unhandled 500s appear in error.log
    Tool: Bash
    Preconditions: Trigger an unhandled 500
    Steps:
      1. Check logs/api/error.log contains the error entry
      2. Check logs/api/api.log also contains the error entry
    Expected Result: Both files have the error
    Evidence: .sisyphus/evidence/task-1-error-log.txt
  ```

  **Commit**: YES
  - Message: `fix(api): explicit truthy parsing for ENABLE_DOCS and DEBUG, fix error logger`
  - Files: `api/main.py`

- [x] 2. **Sync schema.sql + fix migration 010**

  **What to do**:
  - Update `sql/schema.sql`:
    - In `alert_events` table (~line 101): Add `affected_orders INTEGER, affected_gmv REAL, impact_score REAL` after existing columns
    - In `action_tasks` table (~line 142): Add `target_object_type TEXT, target_object_id TEXT` after existing columns
    - These columns are inserted by `scripts/db_dimensional_rule_engine.py` at lines 140 and 216
  - Fix `sql/migrations/010_add_foreign_keys.sql`:
    - The `CREATE TABLE action_tasks_new (...)` at ~line 8 is missing `target_object_type` and `target_object_id`
    - Add these two columns to `action_tasks_new` definition
    - This fixes the "14 columns but 16 values were supplied" error when migration 006 has been applied
  - Do NOT automate migration 010 (it's structurally incompatible with `_apply_migration`'s semicolon splitting)

  **Must NOT do**:
  - Reorder existing columns
  - Change DEFAULT values
  - Add new tables
  - Create new migration files
  - Modify `_apply_migration` function

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - Reason: DDL changes in SQL files, straightforward column additions.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: Task 6 (tests), Task 7 (ruff)
  - **Blocked By**: None

  **References**:
  - `sql/schema.sql:101` - `alert_events` CREATE TABLE (missing 3 columns)
  - `sql/schema.sql:142` - `action_tasks` CREATE TABLE (missing 2 columns)
  - `scripts/db_dimensional_rule_engine.py:140` - INSERT INTO alert_events with affected_orders/gmv/impact_score
  - `scripts/db_dimensional_rule_engine.py:216` - INSERT INTO action_tasks with target_object_type/id
  - `sql/migrations/010_add_foreign_keys.sql:8` - action_tasks_new CREATE TABLE (14 cols, needs 16)
  - `sql/migrations/006_add_action_target_columns.sql` - Migration that adds target_object_type/id to action_tasks

  **Acceptance Criteria**:
  - [ ] `sqlite3 :memory: < sql/schema.sql` succeeds without error
  - [ ] `alert_events` table has 14+ columns including `affected_orders`, `affected_gmv`, `impact_score`
  - [ ] `action_tasks` table has 16+ columns including `target_object_type`, `target_object_id`
  - [ ] `scripts/db_dimensional_rule_engine.py` can insert into fresh DB without "no such column" error
  - [ ] Migration 010 runs successfully on DB with migrations 005+006+007 applied
  - [ ] Migration 010 runs successfully on DB with only migrations 005+007 applied (skip 006)

  **QA Scenarios**:

  ```
  Scenario: Fresh DB from schema.sql supports dimensional engine inserts
    Tool: Bash (sqlite3)
    Preconditions: Clean environment
    Steps:
      1. sqlite3 /tmp/test_schema.db < sql/schema.sql
      2. python3 -c "import sqlite3; conn = sqlite3.connect('/tmp/test_schema.db'); cursor = conn.cursor(); cursor.execute('INSERT INTO alert_events (alert_id, event_type, affected_orders, affected_gmv, impact_score) VALUES (\\\"test\\\", \\\"test\\\", 1, 1.0, 1.0)'); conn.commit(); print('OK')"
    Expected Result: "OK" (no "no such column" error)
    Evidence: .sisyphus/evidence/task-2-schema-fresh.txt

  Scenario: Migration 010 works with migration 006 applied
    Tool: Bash (sqlite3)
    Preconditions: Apply migrations 005, 006, 007 in order
    Steps:
      1. sqlite3 /tmp/test_mig.db < sql/schema.sql
      2. Apply migrations 005, 006, 007
      3. Apply migration 010
    Expected Result: Migration 010 completes without column count error
    Evidence: .sisyphus/evidence/task-2-migration-010.txt
  ```

  **Commit**: YES
  - Message: `fix(sql): add missing columns to schema and migration 010`
  - Files: `sql/schema.sql`, `sql/migrations/010_add_foreign_keys.sql`

- [x] 3. **Fix Qoder task status filtering**

  **What to do**:
  - In `services/qoder_service.py` ~line 98: Change `status="open"` to filter for both `"todo"` and `"in_progress"` statuses
  - The schema default is `'todo'` (`sql/schema.sql:153`), and generators store `"todo"` and `"in_progress"` (`scripts/db_generate_recommendations.py:186`)
  - Extend `_build_conditions` in `services/_query_utils.py` to support list values → generates `WHERE status IN (?, ?)`
  - Alternative if extending is complex: call `get_tasks_with_count` twice (once for "todo", once for "in_progress") and merge results
  - Also check if existing DB has tasks with `status='open'` - if so, include "open" in the filter too to avoid hiding existing tasks

  **Must NOT do**:
  - Turn `_build_conditions` into an ORM
  - Change schema default status values
  - Rename status columns
  - Drop the "open" filter without checking existing data

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - Reason: Filter logic change in service layer, may need small utility extension.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: Task 6 (tests), Task 7 (ruff)
  - **Blocked By**: None

  **References**:
  - `services/qoder_service.py:98` - Current filter: `status="open"`
  - `services/_query_utils.py:8-31` - `_build_conditions` function (single-value equality only)
  - `sql/schema.sql:153` - action_tasks.status DEFAULT 'todo'
  - `scripts/db_generate_recommendations.py:186` - Status values used: "todo", "in_progress"

  **Acceptance Criteria**:
  - [ ] `/qoder/context` returns tasks with `status` in `["todo", "in_progress"]` (and optionally "open")
  - [ ] `build_context()` no longer reports 0 tasks when DB has todo/in_progress tasks
  - [ ] `_build_conditions` supports list values (generates `IN` clause) OR merge strategy works
  - [ ] Existing tasks with `status='open'` are not hidden

  **QA Scenarios**:

  ```
  Scenario: Qoder context returns todo and in_progress tasks
    Tool: Bash (python3 + sqlite3)
    Preconditions: DB with tasks having status "todo" and "in_progress"
    Steps:
      1. Create test DB with schema
      2. Insert tasks with status "todo" (33) and "in_progress" (3)
      3. Call qoder_service.build_context() or /qoder/context endpoint
    Expected Result: Returns 36 tasks total, not 0
    Evidence: .sisyphus/evidence/task-3-qoder-tasks.txt

  Scenario: Qoder context handles include_logs=true without 500
    Tool: Bash (curl)
    Preconditions: Server running with Qoder tasks present
    Steps:
      1. curl "http://localhost:8765/qoder/context?include_logs=true"
    Expected Result: HTTP 200, JSON response with recent_diagnosis field
    Evidence: .sisyphus/evidence/task-3-qoder-logs.txt
  ```

  **Commit**: YES
  - Message: `fix(qoder): filter tasks by todo/in_progress status`
  - Files: `services/qoder_service.py`, `services/_query_utils.py` (if modified)

- [x] 4. **Fix Qoder diagnosis - pass real request_id**

  **What to do**:
  - In `services/qoder_service.py` ~line 126: Instead of `diagnose_by_request_id(request_id=None)`, pass the actual request_id from the current request context
  - The request_id is available via `get_request_id()` (imported at line 108 in qoder_service.py, or available from `core.request_context`)
  - If `get_request_id()` returns None or empty string, call `diagnose_by_request_id` with `limit=N` to fetch recent entries, OR return `[]` (empty list)
  - In `api/schemas_qoder.py` ~line 133: Ensure `recent_diagnosis: List[dict]` can handle empty list `[]`
  - The fix must ensure `include_logs=true` actually returns useful diagnosis data, not just avoid the 500

  **Must NOT do**:
  - Change `diagnose_by_request_id` function signature
  - Modify `services/diagnosis_service.py`
  - Return None instead of empty list

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - Reason: Caller-side fix in qoder_service.py, minor schema adjustment.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: Task 6 (tests), Task 7 (ruff)
  - **Blocked By**: None

  **References**:
  - `services/qoder_service.py:126` - Current: `diagnose_by_request_id(request_id=None)`
  - `services/qoder_service.py:108` - `get_request_id()` import
  - `api/schemas_qoder.py:133` - `recent_diagnosis: List[dict]`
  - `services/diagnosis_service.py:10` - Reads error.log and audit CSV
  - `core/request_context.py` - Request ID management

  **Acceptance Criteria**:
  - [ ] `/qoder/context?include_logs=true` returns HTTP 200 (not 500)
  - [ ] `recent_diagnosis` is always a `List[dict]` (never None)
  - [ ] When logs contain matching request_id entries, they appear in `recent_diagnosis`
  - [ ] When no matching entries exist, `recent_diagnosis` is `[]`
  - [ ] Pydantic validation passes for all response shapes

  **QA Scenarios**:

  ```
  Scenario: Qoder context with include_logs=true returns 200
    Tool: Bash (curl)
    Preconditions: Server running, some request history exists
    Steps:
      1. curl "http://localhost:8765/qoder/context?include_logs=true"
    Expected Result: HTTP 200, valid JSON, recent_diagnosis is a list
    Evidence: .sisyphus/evidence/task-4-logs-200.txt

  Scenario: Qoder context without logs returns empty diagnosis
    Tool: Bash (curl)
    Preconditions: Server running
    Steps:
      1. curl "http://localhost:8765/qoder/context"
    Expected Result: HTTP 200, recent_diagnosis is [] or absent
    Evidence: .sisyphus/evidence/task-4-no-logs.txt

  Scenario: Diagnosis returns actual entries when matching request_id exists
    Tool: Bash (python3)
    Preconditions: Error log has entries with request_ids
    Steps:
      1. python3 -c "from services.qoder_service import build_context; ctx = build_context(..., include_logs=True); print(len(ctx.recent_diagnosis))"
    Expected Result: Returns non-empty list when matching entries exist
    Evidence: .sisyphus/evidence/task-4-diagnosis-entries.txt
  ```

  **Commit**: YES
  - Message: `fix(qoder): pass real request_id to diagnosis service`
  - Files: `services/qoder_service.py`, `api/schemas_qoder.py`

- [x] 5. **Fix pipeline dry-run semantics**

  **What to do**:
  - **Step 1 - Audit all callers**: Search entire codebase for `run_pipeline(` calls. Check each call site for explicit `dry_run` argument.
  - **Step 2 - Change default**: In `pipeline/runner.py` ~line 38, change default from `dry_run=False` to `dry_run=True`
  - **Step 3 - Update callers**: For any caller that relied on `dry_run=False` default, add explicit `dry_run=False` to preserve current behavior (OR if the caller should be safe by default, leave it as `dry_run=True`)
  - **Step 4 - Fix CLI wrapper**: In `scripts/run_db_pipeline.py` ~line 22, ensure `--apply` flag correctly passes `dry_run=False` to `run_pipeline()`. The help text says dry-run is default - verify the argparse setup matches.
  - **Step 5 - Verify simulator**: In `scripts/db_trigger_simulator.py` ~line 91, confirm it defaults to dry-run unless `--apply` is passed (should already be correct)

  **Must NOT do**:
  - Change `run_pipeline` function signature (only change default value)
  - Modify pipeline step logic
  - Remove the `--apply` flag

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - Reason: Default value change + caller audit, mostly grep and small edits.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: Task 6 (tests), Task 7 (ruff)
  - **Blocked By**: None

  **References**:
  - `pipeline/runner.py:38` - `def run_pipeline(..., dry_run=False)`
  - `scripts/run_db_pipeline.py:22` - CLI argument parsing for --dry-run / --apply
  - `scripts/db_trigger_simulator.py:91` - Original simulator main() with --dry-run / --apply

  **Acceptance Criteria**:
  - [ ] `python3 scripts/run_db_pipeline.py` (no args) does NOT update event_outbox status
  - [ ] `python3 scripts/run_db_pipeline.py --apply` DOES update event_outbox status
  - [ ] `run_pipeline()` called with no dry_run arg defaults to dry_run=True
  - [ ] `run_pipeline(dry_run=False)` explicit call still applies triggers
  - [ ] All existing callers that need mutations have explicit `dry_run=False`

  **QA Scenarios**:

  ```
  Scenario: Default pipeline run is dry-run (no DB mutations)
    Tool: Bash (python3 + sqlite3)
    Preconditions: DB with pending event_outbox entries
    Steps:
      1. Record current event_outbox.status values
      2. Run: python3 scripts/run_db_pipeline.py
      3. Check event_outbox.status values again
    Expected Result: Status values unchanged (still 'pending' or original value)
    Evidence: .sisyphus/evidence/task-5-dry-run-default.txt

  Scenario: --apply flag allows mutations
    Tool: Bash (python3 + sqlite3)
    Preconditions: DB with pending event_outbox entries
    Steps:
      1. Run: python3 scripts/run_db_pipeline.py --apply
      2. Check event_outbox.status values
    Expected Result: Status values changed (e.g., to 'simulated')
    Evidence: .sisyphus/evidence/task-5-apply-flag.txt

  Scenario: Explicit dry_run=False still works
    Tool: Bash (python3)
    Preconditions: Import run_pipeline in Python
    Steps:
      1. python3 -c "from pipeline.runner import run_pipeline; # call with dry_run=False explicitly"
    Expected Result: Triggers are applied
    Evidence: .sisyphus/evidence/task-5-explicit-false.txt
  ```

  **Commit**: YES
  - Message: `fix(pipeline): default to dry-run, require --apply for mutations`
  - Files: `pipeline/runner.py`, `scripts/run_db_pipeline.py` (+ any caller files found in audit)

- [x] 6. **Add regression tests**

  **What to do**:
  - Add tests that verify each bug fix:
    - Test env var parsing: `ENABLE_DOCS=0`, `DEBUG=0`, `ENABLE_DOCS=1`, `DEBUG=1`, case variations
    - Test schema: verify fresh DB has all required columns
    - Test Qoder status: verify tasks with "todo" and "in_progress" are returned
    - Test Qoder diagnosis: verify `include_logs=true` returns 200, verify empty diagnosis is `[]`
    - Test pipeline dry-run: verify default is dry-run, verify `--apply` mutates
    - Test error logging: verify unhandled 500s go to error.log
  - Reuse existing fixtures (`in_memory_db`, `temp_db_path` from `conftest.py`)
  - No new integration tests - use unit tests with mocks/fixtures

  **Must NOT do**:
  - Create new fixtures
  - Add integration tests (skip `-m integration`)
  - Test unrelated functionality
  - Reduce existing test coverage

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: [`test-driven-development`]
  - Reason: Writing comprehensive tests for multiple bug fixes requires understanding of the test patterns.

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Wave 1)
  - **Parallel Group**: Wave 2
  - **Blocks**: Final Verification (F1-F4)
  - **Blocked By**: Tasks 1, 2, 3, 4, 5

  **References**:
  - `tests/conftest.py` - Existing fixtures: `in_memory_db`, `temp_db_path`
  - `tests/test_qoder_context.py` - Existing Qoder tests (if exists)
  - `tests/test_pipeline_api.py` - Existing pipeline tests
  - `tests/test_db_schema.py` - Existing schema tests

  **Acceptance Criteria**:
  - [ ] New tests exist for all 6 bug fixes
  - [ ] `pytest tests/` passes with 435+ tests
  - [ ] Coverage remains >= 88%
  - [ ] Each new test fails before the fix, passes after the fix

  **QA Scenarios**:

  ```
  Scenario: All new regression tests pass
    Tool: Bash (pytest)
    Preconditions: All fixes applied
    Steps:
      1. pytest tests/ -v --tb=short
    Expected Result: All tests pass, coverage >= 88%
    Evidence: .sisyphus/evidence/task-6-pytest-results.txt
  ```

  **Commit**: YES
  - Message: `test: add regression tests for bug fixes`
  - Files: `tests/test_*.py` (new or modified test files)

- [x] 7. **Fix ruff issues on modified files**

  **What to do**:
  - Run `ruff check --fix-only` on ONLY the files modified by Tasks 1-5
  - Do NOT run `ruff check .` (full repo has 16,247 issues)
  - Target files: `api/main.py`, `sql/schema.sql`, `sql/migrations/010_add_foreign_keys.sql`, `services/qoder_service.py`, `services/_query_utils.py`, `api/schemas_qoder.py`, `pipeline/runner.py`, `scripts/run_db_pipeline.py`, `tests/test_*.py` (new tests)
  - If isort (I) rules cause cascading changes, skip them: use `--select=E,F,W,UP,N` instead of full ruleset
  - Focus on: unused imports, undefined names, syntax errors, line length

  **Must NOT do**:
  - Fix ruff issues in unmodified files
  - Run `ruff check .`
  - Reformat entire codebase
  - Introduce isort changes in unrelated imports

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - Reason: Linting fixes on known file set, straightforward.

  **Parallelization**:
  - **Can Run In Parallel**: YES (after Wave 1)
  - **Parallel Group**: Wave 2
  - **Blocks**: Final Verification (F1-F4)
  - **Blocked By**: Tasks 1, 2, 3, 4, 5

  **Acceptance Criteria**:
  - [ ] `ruff check $(git diff --name-only)` exits with code 0
  - [ ] `ruff format --check $(git diff --name-only)` exits with code 0 (or format is applied)
  - [ ] No functional changes introduced by ruff fixes

  **QA Scenarios**:

  ```
  Scenario: Ruff passes on modified files
    Tool: Bash
    Preconditions: All fixes applied
    Steps:
      1. ruff check $(git diff --name-only)
    Expected Result: Exit code 0
    Evidence: .sisyphus/evidence/task-7-ruff.txt
  ```

  **Commit**: YES
  - Message: `style: ruff cleanup on modified files`
  - Files: All files modified in Tasks 1-6

---

## Final Verification Wave

> 4 review agents run in PARALLEL. ALL must APPROVE. Present results to user.

- [x] F1. **Plan Compliance Audit** — `oracle`
  Verify all 6 findings are fixed. Check evidence files exist. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT`

- [x] F2. **Code Quality Review** — `unspecified-high`
  Run `pytest`, `ruff check` on modified files. Review for `as any`, empty catches, `console.log`, commented-out code.
  Output: `Build [PASS/FAIL] | Lint [PASS/FAIL] | Tests [N pass/N fail] | VERDICT`

- [x] F3. **Real Manual QA** — `unspecified-high`
  Execute every QA scenario from every task. Test edge cases. Save evidence.
  Output: `Scenarios [N/N pass] | Edge Cases [N tested] | VERDICT`

- [x] F4. **Scope Fidelity Check** — `deep`
  Verify each task's "What to do" matches actual diff. Check for scope creep.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N] | VERDICT`

---

## Commit Strategy

- **Wave 1 commits** (one per task, or group if same file):
  - `fix(api): explicit truthy parsing for ENABLE_DOCS and DEBUG`
  - `fix(sql): add missing columns to schema and migration 010`
  - `fix(qoder): filter tasks by todo/in_progress status`
  - `fix(qoder): pass real request_id to diagnosis service`
  - `fix(pipeline): default to dry-run, require --apply for mutations`
- **Wave 2 commits**:
  - `test: add regression tests for bug fixes`
  - `style: ruff cleanup on modified files`

---

## Success Criteria

### Verification Commands
```bash
# Run all tests
pytest tests/ --tb=short

# Run tests with coverage
pytest tests/ --cov=api --cov=services --cov=adapters --cov=core --cov=pipeline

# Check ruff on modified files
ruff check $(git diff --name-only)

# Verify npm tests (frontend)
npm test

# Verify frontend build
npm run build
```

### Final Checklist
- [ ] All 6 findings fixed
- [ ] `pytest` passes (435+ tests, coverage >= 88%)
- [ ] `npm test` passes (33 tests)
- [ ] `npm run build` passes
- [ ] `ruff check` passes on modified files
- [ ] All evidence files in `.sisyphus/evidence/`
- [ ] No scope creep detected
- [ ] User explicit "okay" received
