# Fix 10 Audit Findings — Backend + Frontend

## TL;DR

> **Quick Summary**: Fix 8 audit findings (plus 2 discovered) across backend Python services and frontend React pages. All fixes follow TDD — write a failing test first, then implement the minimal fix. Backend uses pytest (22 existing test files), frontend uses vitest (4 existing test files).

> **Deliverables**:
> - 6 backend bug fixes (feishu filter, dispatch dry-run, task schema, deps, db counts, log reader)
> - 3 frontend bug fixes (StatusCard styling, query error handling, Pipeline POST anti-pattern)
> - 1 backend dependency consolidation (pyproject.toml)

> **Estimated Effort**: Medium
> **Parallel Execution**: YES — 3 waves
> **Critical Path**: Wave 1 (deps) → Wave 2 (all fixes parallel) → Wave 3 (all fixes parallel) → Verification

---

## Context

### Original Request
User presented 8 findings from an audit. During exploration, 2 additional issues were discovered: TaskItem schema missing `completed_at`/`feedback` fields, and Pipeline.tsx using `useQuery` for a POST mutation.

### Interview Summary
**Key Discussions**:
- **Dependency fix**: Consolidate all deps to pyproject.toml with proper version constraints; remove requirements.txt
- **get_table_counts**: Skip unknown tables with warning log instead of ValueError
- **Log reader**: True reverse seek from end of file, not read-entire-file-into-memory
- **Test strategy**: TDD — write failing test first, then implement minimal fix
- **priority ordering**: HIGH findings first, MEDIUM next, LOW last

**Research Findings**:
- Explore agent confirmed all 8 findings with exact root causes and line numbers
- Backend tests use TestClient + SQLite seed DB (integration style)
- Frontend tests use vi.mock + testing-library (mock-based)
- NO CI pipeline, NO coverage configuration
- Backend test deps NOT in pyproject.toml (part of finding #4 fix)

### Metis Review
**Identified Gaps** (addressed):
- **Feishu filter semantics**: Confirmed as table-level filtering (which tables to export/sync), not row-level
- **Dispatch counter downstream effects**: Task includes checking all callers of dispatch_attempts before fix
- **Schema field source-of-truth**: DB columns verified existing; fix is purely query/schema alignment
- **Pipeline POST urgency**: Real risk — useQuery refetches on window focus; included as finding #10
- **requirements.txt fate**: Deleted after consolidation; pyproject.toml becomes single source
- **Empty table_filter semantics**: Defined as [] = all tables (backward compatible)
- **SQLite WAL mode test isolation**: Conftest fixture ensures fresh DB per test

---

## Work Objectives

### Core Objective
Fix all 10 audit findings with TDD: write failing test → implement minimal fix → verify green. Zero regressions on existing tests.

### Concrete Deliverables
- `services/feishu_service.py` — filter propagation fixed
- `services/dispatch_service.py` — dry-run purity restored
- `services/task_service.py` — SELECT includes all schema fields
- `api/schemas.py` — TaskItem includes all DB columns
- `pyproject.toml` — all runtime deps with version constraints
- `services/db_service.py` — skip unknown tables with warning
- `services/log_reader.py` — true reverse seek tail
- `frontend/src/pages/Feishu.tsx` — StatusCard recognizes synced/imported
- `frontend/src/pages/Tasks.tsx` — error handling
- `frontend/src/pages/Logs.tsx` — error handling
- `frontend/src/pages/Pipeline.tsx` — useMutation + error handling

### Definition of Done
- [ ] All 10 findings have passing TDD tests
- [ ] All existing tests still pass (`pytest` + `cd frontend && npm test`)
- [ ] `uv pip check` reports zero conflicts
- [ ] `uv pip install -e .` succeeds from pyproject.toml alone

### Must Have
- Fix all 10 findings with TDD
- Keep existing test suite green
- Zero new dependencies beyond what already exists
- pyproject.toml as single dependency source of truth

### Must NOT Have (Guardrails)
- **NO** database schema changes (ALTER TABLE, migrations)
- **NO** new pip/npm packages added
- **NO** refactoring code outside the 10 findings
- **NO** changing Feishu API integration contract (only fix filter propagation)
- **NO** touching test framework configuration (pytest.ini, vitest.config.ts)
- **NO** modifying production log files
- **NO** dry-run must remain truly read-only (no DB writes, no counter increments)
- **NO** "while I'm here" cleanups in adjacent code

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed. No exceptions.

### Test Decision
- **Infrastructure exists**: YES (pytest + vitest)
- **Automated tests**: TDD — RED (failing test) → GREEN (minimal impl) → REFACTOR
- **Framework**: pytest (backend), vitest (frontend)

### QA Policy
Every task MUST include agent-executed QA scenarios.
Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **Backend**: Bash (curl for API endpoints, pytest for unit/integration)
- **Frontend**: Bash (vitest run) for component tests

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Start Immediately — dependency foundation):
├── Task 1: Consolidate deps to pyproject.toml [quick]

Wave 2 (After Wave 1 — all backend fixes, MAX PARALLEL):
├── Task 2: Fix feishu table filter propagation [unspecified-high]
├── Task 3: Fix dispatch dry-run purity [unspecified-high]
├── Task 4: Fix task schema+query alignment [unspecified-high]
├── Task 5: Fix get_table_counts ValueError [quick]
├── Task 6: Fix log reader true tail [unspecified-high]

Wave 3 (After Wave 1 — all frontend fixes, MAX PARALLEL):
├── Task 7: Fix StatusCard styling [quick]
├── Task 8: Fix Tasks/Logs/Pipeline error handling [unspecified-high]
├── Task 9: Fix Pipeline POST anti-pattern [quick]

Wave FINAL (After ALL tasks):
├── Task F1: Plan compliance audit (oracle)
├── Task F2: Code quality review (unspecified-high)
├── Task F3: Real QA — run all tests (unspecified-high)
├── Task F4: Scope fidelity check (deep)
-> Present results → Get explicit user okay

Critical Path: Task 1 → (Tasks 2-9 in parallel) → Verification Wave
Parallel Speedup: ~85% vs sequential
Max Concurrent: 8 (Wave 2-3)
```

### Dependency Matrix

- **1**: — 2–9, 1
- **2**: 1 — FINAL, — (parallel with 3–9)
- **3**: 1 — FINAL, — (parallel with 2, 4–9)
- **4**: 1 — FINAL, — (parallel with 2–3, 5–9)
- **5**: 1 — FINAL, — (parallel with 2–4, 6–9)
- **6**: 1 — FINAL, — (parallel with 2–5, 7–9)
- **7**: — FINAL, — (parallel with 1–6, 8–9)
- **8**: — FINAL, — (parallel with 1–7, 9)
- **9**: — FINAL, — (parallel with 1–8)
- **FINAL**: 1–9 — done, —

### Agent Dispatch Summary

- **Wave 1**: **1** task — T1 → `quick`
- **Wave 2**: **5** tasks — T2 → `unspecified-high`, T3 → `unspecified-high`, T4 → `unspecified-high`, T5 → `quick`, T6 → `unspecified-high`
- **Wave 3**: **3** tasks — T7 → `quick`, T8 → `unspecified-high`, T9 → `quick`
- **FINAL**: **4** tasks — F1 → `oracle`, F2 → `unspecified-high`, F3 → `unspecified-high`, F4 → `deep`

---

## TODOs

- [x] 1. **Consolidate Python dependencies to pyproject.toml**

  **What to do** (TDD — write test first):
  - **RED**: Create test that checks `uv pip install -e .` succeeds and `uv pip check` reports zero conflicts. Also test that `import fastapi; import httpx; import starlette` works with installed versions.
  - **GREEN**: Add `[project.dependencies]` to pyproject.toml with ALL runtime deps from requirements.txt. Update version constraints: FastAPI `>=0.115.0` (remove `<0.120`), Starlette `>=0.38.0` (remove `<1.0`), httpx `>=0.27.0` (remove `<0.28`). Add `[project.optional-dependencies]` with `dev` group including pytest, pytest-cov. Verify `uv pip install -e ".[dev]"` works. Delete `requirements.txt`.
  - **REFACTOR**: Ensure pyproject.toml package discovery works (`[tool.setuptools.packages.find]` if needed).

  **Must NOT do**:
  - Do NOT change any dependency version to a different major version
  - Do NOT add new dependencies
  - Do NOT modify pytest.ini or any test config

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Single-file config change with clear constraints
  - **Skills**: []
  - **Skills Evaluated but Omitted**: N/A

  **Parallelization**:
  - **Can Run In Parallel**: NO (foundation for all other tasks)
  - **Parallel Group**: Wave 1 (standalone)
  - **Blocks**: Tasks 2–9
  - **Blocked By**: None

  **References**:
  - `pyproject.toml:1-15` — Current state: name/version/requires-python only, no deps
  - `requirements.txt:13-16` — Current constraints: fastapi>=0.115.0,<0.120.0; starlette>=0.38.0,<1.0.0; httpx>=0.27.0,<0.28.0
  - `requirements.txt:1-17` — Full dependency list to migrate
  - **WHY**: Understand current constraint violations (installed FastAPI 0.136.1 violates <0.120) and complete dependency list

  **Acceptance Criteria**:
  - [ ] `uv pip install -e ".[dev]"` exits 0
  - [ ] `uv pip check` exits 0 (no conflicts)
  - [ ] `python -c "import fastapi; import starlette; import httpx; import uvicorn"` exits 0
  - [ ] `requirements.txt` no longer exists

  **QA Scenarios**:

  ```
  Scenario: Fresh install from pyproject.toml succeeds
    Tool: Bash
    Preconditions: Clean venv (create temp venv)
    Steps:
      1. uv venv /tmp/test-venv && source /tmp/test-venv/bin/activate
      2. uv pip install -e ".[dev]"
      3. uv pip check
      4. python -c "from fastapi import FastAPI; from starlette.testclient import TestClient; import httpx"
    Expected Result: All commands exit 0, no error output
    Failure Indicators: Version conflict errors, import errors, non-zero exit
    Evidence: .sisyphus/evidence/task-1-install-verify.txt

  Scenario: FastAPI 0.136.x is accepted (no upper bound rejection)
    Tool: Bash
    Preconditions: fastapi 0.136.1 installed (current environment)
    Steps:
      1. uv pip check 2>&1 | grep -i "fastapi"
    Expected Result: No fastapi-related conflict output
    Failure Indicators: "fastapi 0.136.1 does not satisfy constraint <0.120"
    Evidence: .sisyphus/evidence/task-1-fastapi-version.txt
  ```

  **Evidence to Capture**:
  - [ ] task-1-install-verify.txt — output of uv pip install + check
  - [ ] task-1-fastapi-version.txt — version conflict check output

  **Commit**: YES
  - Message: `fix(deps): consolidate Python dependencies to pyproject.toml`
  - Files: `pyproject.toml`, `requirements.txt` (deleted)

---

- [x] 2. **Fix Feishu table filter propagation**

  **What to do** (TDD):
  - **RED**: Write tests in `tests/test_feishu_service.py`:
    - `test_export_tables_respects_filter`: Mock `_run_script`; call `export_tables(table_names=["orders", "products"])`; assert `_run_script` received args containing both table names, NOT `--all`
    - `test_sync_to_feishu_respects_filter`: Mock `_run_script`; call `sync_to_feishu(table_names=["orders", "products"])`; assert `_run_script` received `--table orders --table products`, not `--table orders` alone (resolved[0] bug)
    - `test_import_status_per_table_counts`: Mock `_run_script` to return `{"tables": {"orders": {"imported": 5, "skipped": 2}, "products": {"imported": 3, "skipped": 1}}}`; assert per-table counts differ
  - **GREEN** (services/feishu_service.py):
    - **export_tables (line 164)**: Build args from `resolved` list: if `len(resolved) == len(all_available)` use `--all`, else pass `["--table", t]` for each t in resolved
    - **sync_to_feishu (line 200-201)**: When not syncing all, pass ALL resolved tables: `args = ["--apply"] + [a for t in resolved for a in ("--table", t)]`
    - **import_status_from_feishu (lines 242-243)**: Check if `import_result` has per-table data; extract per-table imported/skipped instead of using global values
  - **REFACTOR**: No refactoring — minimal fix only

  **Must NOT do**:
  - Do NOT change the Feishu API integration contract
  - Do NOT add row-level filtering (table-level filtering only)
  - Do NOT change `_run_script` or `_get_table_names`

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Multi-function fix in a single file, requires understanding of script argument passing
  - **Skills**: []
  - **Skills Evaluated but Omitted**: N/A

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 3–6)
  - **Parallel Group**: Wave 2
  - **Blocks**: None
  - **Blocked By**: Task 1

  **References**:
  - `services/feishu_service.py:147-177` — `export_tables()`: Always passes `["--all"]` at line 164, ignoring `table_names` param
  - `services/feishu_service.py:179-214` — `sync_to_feishu()`: Line 200 passes only `resolved[0]` instead of all resolved tables
  - `services/feishu_service.py:216-249` — `import_status_from_feishu()`: Lines 242-243 use same global counts for every table
  - `services/feishu_service.py:70-90` (approx) — `_get_table_names()`: How `resolved` list is built from `table_names` param
  - **WHY**: Understand exact filter propagation bug mechanics and how `_run_script` args are constructed

  **Acceptance Criteria**:
  - [ ] `pytest tests/test_feishu_service.py -v -k "test_export_tables_respects_filter or test_sync_to_feishu_respects_filter or test_import_status_per_table_counts"` — PASS
  - [ ] `pytest tests/ -v` — all existing tests still pass

  **QA Scenarios**:

  ```
  Scenario: export_tables with specific table filter
    Tool: Bash
    Preconditions: FEISHU_DRY_RUN=true in env, mock _run_script
    Steps:
      1. pytest tests/test_feishu_service.py::test_export_tables_respects_filter -v
    Expected Result: Test passes — _run_script called with table-specific args, not --all
    Failure Indicators: Test fails because --all still passed unconditionally
    Evidence: .sisyphus/evidence/task-2-export-filter.txt

  Scenario: sync_to_feishu with multiple specific tables
    Tool: Bash
    Preconditions: Same as above
    Steps:
      1. pytest tests/test_feishu_service.py::test_sync_to_feishu_respects_filter -v
    Expected Result: Test passes — all 2+ tables passed, not just resolved[0]
    Failure Indicators: Only first table passed to script
    Evidence: .sisyphus/evidence/task-2-sync-filter.txt

  Scenario: import_status returns per-table (not global) counts
    Tool: Bash
    Preconditions: Mock import_result with per-table data
    Steps:
      1. pytest tests/test_feishu_service.py::test_import_status_per_table_counts -v
    Expected Result: Different tables show different imported/skipped counts
    Failure Indicators: All tables show identical global counts
    Evidence: .sisyphus/evidence/task-2-import-status.txt
  ```

  **Evidence to Capture**:
  - [ ] task-2-export-filter.txt
  - [ ] task-2-sync-filter.txt
  - [ ] task-2-import-status.txt

  **Commit**: YES
  - Message: `fix(feishu): propagate table filter to export/sync/import`
  - Files: `services/feishu_service.py`, `tests/test_feishu_service.py`

---

- [x] 3. **Fix dispatch dry-run purity**

  **What to do** (TDD):
  - **RED**: Write test in `tests/test_dispatch_service.py`:
    - `test_dry_run_does_not_increment_attempts`: Seed outbox event with `dispatch_attempts=0`. Call `dispatch_one(is_dry_run=True)`. Query DB: assert `dispatch_attempts` is still 0, `last_dispatch_at` is still NULL.
    - `test_dry_run_does_not_change_status`: Same setup; assert `status` remains unchanged.
    - `test_real_dispatch_does_increment`: Call `dispatch_one(is_dry_run=False)`. Assert `dispatch_attempts` is 1, `last_dispatch_at` is set.
  - **GREEN** (services/dispatch_service.py):
    - **write_result (lines 116-133)**: When `is_dry_run=True`, skip the UPDATE entirely (or use a conditional UPDATE that doesn't modify `dispatch_attempts`/`last_dispatch_at`). The `db_status = "pending"` assignment is kept for reporting but not written to DB.
    - **dispatch_one (line 203)**: Only call `write_result()` when `not is_dry_run`. Or refactor `write_result` to accept a no-op flag.
  - **REFACTOR**: Optionally extract dry-run guard into a separate early-return path in `write_result`.

  **Must NOT do**:
  - Do NOT change `MAX_ATTEMPTS` or retry logic
  - Do NOT change how `dispatch_one` claims events
  - Do NOT remove dry-run reporting (the `return` dict still needs adapter name etc.)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Carefully scoped fix in stateful dispatch logic; must preserve all non-dry-run behavior
  - **Skills**: []
  - **Skills Evaluated but Omitted**: N/A

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 2, 4–6)
  - **Parallel Group**: Wave 2
  - **Blocks**: None
  - **Blocked By**: Task 1

  **References**:
  - `services/dispatch_service.py:111-135` — `write_result()`: Line 132 always increments dispatch_attempts; line 117 sets db_status=pending for dry-run but still writes
  - `services/dispatch_service.py:156-211` — `dispatch_one()`: Lines 200-203 call write_result regardless of is_dry_run
  - `services/dispatch_service.py:1-20` — `MAX_ATTEMPTS` constant definition
  - **WHY**: Understand exact write path and where to add the dry-run guard

  **Acceptance Criteria**:
  - [ ] `pytest tests/test_dispatch_service.py -v -k "test_dry_run"` — all dry-run tests PASS
  - [ ] `pytest tests/test_dispatch_service.py -v` — all dispatch tests PASS
  - [ ] `pytest tests/ -v` — no regressions

  **QA Scenarios**:

  ```
  Scenario: Dry-run does not increment dispatch_attempts
    Tool: Bash
    Preconditions: Seed SQLite with outbox event (dispatch_attempts=0, last_dispatch_at=NULL)
    Steps:
      1. pytest tests/test_dispatch_service.py::test_dry_run_does_not_increment_attempts -v
    Expected Result: dispatch_attempts=0 after dry-run, last_dispatch_at=NULL
    Failure Indicators: Counter incremented, timestamp set
    Evidence: .sisyphus/evidence/task-3-dry-run-counter.txt

  Scenario: Dry-run does not change event status
    Tool: Bash
    Preconditions: Seed outbox event with status="pending"
    Steps:
      1. pytest tests/test_dispatch_service.py::test_dry_run_does_not_change_status -v
    Expected Result: status still "pending" after dry-run
    Failure Indicators: Status changed to anything else
    Evidence: .sisyphus/evidence/task-3-dry-run-status.txt

  Scenario: Real dispatch still increments counter
    Tool: Bash
    Preconditions: Seed outbox event (dispatch_attempts=0)
    Steps:
      1. pytest tests/test_dispatch_service.py::test_real_dispatch_does_increment -v
    Expected Result: dispatch_attempts=1, last_dispatch_at is set
    Failure Indicators: Counter not incremented (regression)
    Evidence: .sisyphus/evidence/task-3-real-dispatch.txt
  ```

  **Evidence to Capture**:
  - [ ] task-3-dry-run-counter.txt
  - [ ] task-3-dry-run-status.txt
  - [ ] task-3-real-dispatch.txt

  **Commit**: YES
  - Message: `fix(dispatch): make dry-run truly read-only`
  - Files: `services/dispatch_service.py`, `tests/test_dispatch_service.py`

---

- [x] 4. **Fix task schema+query alignment**

  **What to do** (TDD):
  - **RED**: Write test in `tests/test_task_schema.py`:
    - `test_task_item_includes_all_db_fields`: Query `action_tasks` table via task_service. Assert returned dict keys include all 15 fields: task_id, task_title, task_description, status, priority, owner_role, owner_user_id, due_at, created_at, completed_at, feedback, recommendation_id, event_id, target_object_type, target_object_id
    - `test_task_list_response_matches_schema`: Call `GET /tasks` via TestClient. Assert each item has all 15 fields, none missing
    - `test_completed_at_and_feedback_present`: Verify completed_at and feedback appear in API response (not silently dropped)
  - **GREEN**:
    - **services/task_service.py (lines 28-31)**: Add recommendation_id, event_id, owner_user_id, target_object_type, target_object_id to SELECT. Confirm columns exist with `PRAGMA table_info(action_tasks)` first.
    - **api/schemas.py (TaskItem)**: Add `completed_at: Optional[str] = None` and `feedback: Optional[str] = None`. Ensure all 15 fields declared.
  - **REFACTOR**: Align field ordering between SELECT and schema

  **Must NOT do**:
  - Do NOT ALTER TABLE or add/remove columns
  - Do NOT change field defaults beyond what's necessary
  - Do NOT change API response shape beyond adding missing fields

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Schema+query coordination across two files; requires DB schema verification
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 2–3, 5–6)
  - **Parallel Group**: Wave 2
  - **Blocks**: None
  - **Blocked By**: Task 1

  **References**:
  - `api/schemas.py:56-71` — TaskItem model: declares 13 fields; missing completed_at, feedback
  - `services/task_service.py:26-31` — SELECT returns 10 columns; missing 5 declared fields
  - **WHY**: Schema declares fields SELECT doesn't return → API returns default nulls. Also SELECT returns fields schema doesn't have → silently dropped.

  **Acceptance Criteria**:
  - [ ] `pytest tests/test_task_schema.py -v` — all 3 tests PASS
  - [ ] `curl -s localhost:8000/tasks | python3 -c "import sys,json; d=json.load(sys.stdin); assert 'completed_at' in d['items'][0]"` exits 0
  - [ ] `pytest tests/ -v` — no regressions

  **QA Scenarios**:

  ```
  Scenario: Task list response includes all 15 declared fields
    Tool: Bash (curl)
    Preconditions: Server running, at least 1 task in DB
    Steps:
      1. curl -s http://localhost:8000/tasks | python3 -c "
import sys, json
data = json.load(sys.stdin)
item = data['items'][0]
required = ['task_id','task_title','task_description','status','priority',
            'owner_role','owner_user_id','due_at','created_at','completed_at',
            'feedback','recommendation_id','event_id','target_object_type','target_object_id']
missing = [f for f in required if f not in item]
assert not missing, f'Missing fields: {missing}'
print('All 15 fields present')
"
    Expected Result: "All 15 fields present"
    Failure Indicators: AssertionError listing missing fields
    Evidence: .sisyphus/evidence/task-4-schema-fields.json

  Scenario: recommendation_id is populated from DB
    Tool: Bash (curl)
    Preconditions: Task with recommendation_id='rec-123' in DB
    Steps:
      1. curl -s http://localhost:8000/tasks | python3 -c "
import sys, json
data = json.load(sys.stdin)
items_with_rec = [i for i in data['items'] if i.get('recommendation_id')]
assert items_with_rec, 'No items have recommendation_id'
print(f'{len(items_with_rec)} items have recommendation_id')
"
    Expected Result: At least 1 item has non-null recommendation_id
    Failure Indicators: All items null (default empty)
    Evidence: .sisyphus/evidence/task-4-recommendation-id.txt
  ```

  **Evidence to Capture**:
  - [ ] task-4-schema-fields.json
  - [ ] task-4-recommendation-id.txt

  **Commit**: YES
  - Message: `fix(tasks): align schema fields with DB query columns`
  - Files: `api/schemas.py`, `services/task_service.py`, `tests/test_task_schema.py`

---

- [x] 5. **Fix get_table_counts ValueError for unknown tables**

  **What to do** (TDD):
  - **RED**: Write test in `tests/test_db_service.py`:
    - `test_get_table_counts_skips_unknown_tables`: Temp SQLite DB with whitelist table + non-whitelist table (e.g., temp_cache, sqlite_sequence). Call get_table_counts(). Assert returns counts for whitelist table, does NOT raise ValueError.
    - `test_get_table_counts_logs_warning`: Assert warning logged (caplog) for unknown tables.
  - **GREEN** (services/db_service.py line 53): Replace `validate_table_name(table_name)` (raises ValueError) with try/except or conditional: `logger.warning("Skipping unknown table: %s", table_name)` + `continue`.
  - **REFACTOR**: No refactoring needed

  **Must NOT do**:
  - Do NOT expand the table whitelist
  - Do NOT change behavior for known tables
  - Do NOT suppress the warning

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Single-line behavior change (raise → log+skip), well-scoped
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 2–4, 6)
  - **Parallel Group**: Wave 2
  - **Blocks**: None
  - **Blocked By**: Task 1

  **References**:
  - `services/db_service.py:44-64` — get_table_counts(): line 53 calls validate_table_name() which raises ValueError
  - `services/db_service.py:20-42` — validate_table_name(): which tables are whitelisted
  - **WHY**: Current behavior causes /status 500; fix is log+skip

  **Acceptance Criteria**:
  - [ ] `pytest tests/test_db_service.py::test_get_table_counts_skips_unknown_tables -v` — PASS
  - [ ] `pytest tests/test_db_service.py::test_get_table_counts_logs_warning -v` — PASS
  - [ ] Warning log contains "Skipping unknown table"

  **QA Scenarios**:

  ```
  Scenario: Unknown table skipped gracefully
    Tool: Bash (pytest)
    Preconditions: Temp SQLite DB with known + unknown tables
    Steps:
      1. pytest tests/test_db_service.py::test_get_table_counts_skips_unknown_tables -v
    Expected Result: Known table counts returned, no ValueError
    Failure Indicators: ValueError raised
    Evidence: .sisyphus/evidence/task-5-skip-unknown.txt

  Scenario: Warning logged for skipped tables
    Tool: Bash (pytest)
    Preconditions: Same DB setup
    Steps:
      1. pytest tests/test_db_service.py::test_get_table_counts_logs_warning -v -s 2>&1 | grep "Skipping unknown table"
    Expected Result: Warning message found
    Failure Indicators: No warning (silent skip)
    Evidence: .sisyphus/evidence/task-5-warning-log.txt
  ```

  **Evidence to Capture**:
  - [ ] task-5-skip-unknown.txt
  - [ ] task-5-warning-log.txt

  **Commit**: YES
  - Message: `fix(db): skip unknown tables in get_table_counts with warning`
  - Files: `services/db_service.py`, `tests/test_db_service.py`

---

- [x] 6. **Fix log reader true tail with reverse seek**

  **What to do** (TDD):
  - **RED**: Write test in `tests/test_log_reader.py`:
    - `test_tail_jsonl_returns_correct_entries`: 20-line JSONL file, tail 5 → assert last 5 lines in reverse chronological order
    - `test_tail_jsonl_memory_bounded`: 10MB+ JSONL file, tail 100 → assert function completes without loading entire file (use tracemalloc or verify no OOM)
    - `test_tail_jsonl_empty_file`: Assert returns []
    - `test_tail_jsonl_malformed_lines`: Mix of valid/invalid JSON → valid lines returned, malformed skipped with warning
  - **GREEN** (services/log_reader.py line 40): Replace `reversed(list(f))` with reverse seek: `f.seek(0, os.SEEK_END)`, read backwards in 8KB chunks, split on newlines, parse JSON, collect until limit reached. Remove double reversal. Handle edge case: file smaller than chunk size.
  - **REFACTOR**: Clean up double-reversal pattern

  **Must NOT do**:
  - Do NOT change function signature or return type
  - Do NOT change malformed JSON handling (skip+log)
  - Do NOT add new dependencies

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Algorithm change with memory guarantees; careful edge case handling
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 2–5)
  - **Parallel Group**: Wave 2
  - **Blocks**: None
  - **Blocked By**: Task 1

  **References**:
  - `services/log_reader.py:18-56` — _tail_jsonl(): line 40 reversed(list(f)) reads entire file into memory
  - **WHY**: O(file_size) memory → O(chunk_size + limit)

  **Acceptance Criteria**:
  - [ ] `pytest tests/test_log_reader.py -v` — all 4 tests PASS
  - [ ] Memory usage for 100MB file under 10MB during tail
  - [ ] `pytest tests/ -v` — no regressions

  **QA Scenarios**:

  ```
  Scenario: Correct last N lines returned
    Tool: Bash (pytest)
    Preconditions: 20-line JSONL file
    Steps:
      1. pytest tests/test_log_reader.py::test_tail_jsonl_returns_correct_entries -v
    Expected Result: Last 5 lines in correct order
    Failure Indicators: Wrong lines/order/all lines
    Evidence: .sisyphus/evidence/task-6-correct-tail.txt

  Scenario: Large file memory bounded
    Tool: Bash (pytest)
    Preconditions: 10MB+ JSONL file
    Steps:
      1. pytest tests/test_log_reader.py::test_tail_jsonl_memory_bounded -v
    Expected Result: Completes without OOM
    Failure Indicators: MemoryError, timeout
    Evidence: .sisyphus/evidence/task-6-memory-bounded.txt

  Scenario: Malformed JSON skipped
    Tool: Bash (pytest)
    Preconditions: Mix of valid/invalid JSON lines
    Steps:
      1. pytest tests/test_log_reader.py::test_tail_jsonl_malformed_lines -v
    Expected Result: Valid lines returned, warnings for malformed
    Failure Indicators: json.loads ValueError propagates
    Evidence: .sisyphus/evidence/task-6-malformed.txt
  ```

  **Evidence to Capture**:
  - [ ] task-6-correct-tail.txt
  - [ ] task-6-memory-bounded.txt
  - [ ] task-6-malformed.txt

  **Commit**: YES
  - Message: `fix(logs): use reverse seek for tail instead of full file read`
  - Files: `services/log_reader.py`, `tests/test_log_reader.py`

---

- [x] 7. **Fix StatusCard OK-status recognition (Feishu.tsx)**

  **What to do** (TDD):
  - **RED**: Write test in `frontend/src/pages/__tests__/Feishu.test.tsx`:
    - `test_status_card_shows_ok_for_synced`: Render StatusCard with `result={{ status: "synced" }}`. Assert it renders with success styling (green/check icon), NOT error styling (red/alert icon).
    - `test_status_card_shows_ok_for_imported`: Same for `status: "imported"`.
    - `test_status_card_shows_error_for_failed`: Render with `status: "failed"`; assert error styling.
  - **GREEN** (frontend/src/pages/Feishu.tsx line 20): Update `isOk` condition from `result.status === "preview" || result.status === "not_configured" || result.status === "exported"` to also include `"synced"` and `"imported"`. Better: use a set: `const okStatuses = new Set(["preview", "not_configured", "exported", "synced", "imported"]); const isOk = okStatuses.has(result.status);`
  - **REFACTOR**: Use Set for clarity and extensibility.

  **Must NOT do**:
  - Do NOT change StatusCard component structure or props
  - Do NOT change what "not_configured" means
  - Do NOT add new status values beyond what Feishu API returns

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: One-line condition change with clear test coverage
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 8–9)
  - **Parallel Group**: Wave 3
  - **Blocks**: None
  - **Blocked By**: None (frontend; no backend dependency)

  **References**:
  - `frontend/src/pages/Feishu.tsx:20` — StatusCard isOk condition: only preview/not_configured/exported recognized as OK
  - **WHY**: synced and imported are success states but rendered as errors

  **Acceptance Criteria**:
  - [ ] `cd frontend && npx vitest run src/pages/__tests__/Feishu.test.tsx` — all 3 tests PASS
  - [ ] `cd frontend && npm test` — no regressions

  **QA Scenarios**:

  ```
  Scenario: synced status renders as success
    Tool: Bash (vitest)
    Preconditions: None
    Steps:
      1. cd frontend && npx vitest run -t "test_status_card_shows_ok_for_synced"
    Expected Result: Test PASS — synced renders with success styling
    Failure Indicators: Test fails — synced still renders as error
    Evidence: .sisyphus/evidence/task-7-synced-status.txt

  Scenario: failed status still renders as error
    Tool: Bash (vitest)
    Preconditions: None
    Steps:
      1. cd frontend && npx vitest run -t "test_status_card_shows_error_for_failed"
    Expected Result: Test PASS — failed still renders as error
    Failure Indicators: Regression — failed renders as success
    Evidence: .sisyphus/evidence/task-7-failed-status.txt
  ```

  **Evidence to Capture**:
  - [ ] task-7-synced-status.txt
  - [ ] task-7-failed-status.txt

  **Commit**: YES
  - Message: `fix(frontend): recognize synced/imported as OK status in StatusCard`
  - Files: `frontend/src/pages/Feishu.tsx`, `frontend/src/pages/__tests__/Feishu.test.tsx`

---

- [x] 8. **Add query error handling to Tasks, Logs, Pipeline pages**

  **What to do** (TDD):
  - **RED**: Write tests (3 new or updated test files):
    - `frontend/src/pages/__tests__/Tasks.test.tsx`: Mock apiClient.get to reject with Error("Network error"). Assert error message/component rendered.
    - `frontend/src/pages/__tests__/Logs.test.tsx`: Same pattern for logs endpoint.
    - `frontend/src/pages/__tests__/Pipeline.test.tsx`: Same for pipeline preview endpoint.
  - **GREEN** (frontend/src/pages/Tasks.tsx, Logs.tsx, Pipeline.tsx):
    - Each page: Destructure `error` from useQuery: `const { data, isLoading, error } = useQuery(...)`
    - Add error rendering: `if (error) return <ErrorPanel message={error.message} />` (or similar existing ErrorPanel component)
    - Tasks.tsx line 15: Add `error` destructuring + error branch
    - Logs.tsx line 39: Same in all 3 tab components
    - Pipeline.tsx line 18: Same for query error
  - **REFACTOR**: If ErrorPanel already exists in components, use it consistently across all pages

  **Must NOT do**:
  - Do NOT change the success rendering path
  - Do NOT add error handling to pages not in findings (Alerts, Dashboard, Outbox, etc.)
  - Do NOT create a new error component if ErrorPanel already exists

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: 3 pages × consistent pattern changes; requires understanding of each page's query setup
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 7, 9)
  - **Parallel Group**: Wave 3
  - **Blocks**: None
  - **Blocked By**: None (frontend; no backend dependency)

  **References**:
  - `frontend/src/pages/Tasks.tsx:15-17` — useQuery without error destructuring
  - `frontend/src/pages/Logs.tsx:38-42` — useQuery without error handling in all tabs
  - `frontend/src/pages/Pipeline.tsx:18-22` — useQuery without error handling
  - `frontend/src/components/ErrorPanel.tsx` — Check if this component exists and its props
  - **WHY**: Query failures show blank/generic failure instead of actionable error info

  **Acceptance Criteria**:
  - [ ] `cd frontend && npx vitest run src/pages/__tests__/Tasks.test.tsx` — error test PASS
  - [ ] `cd frontend && npx vitest run src/pages/__tests__/Logs.test.tsx` — error test PASS
  - [ ] `cd frontend && npm test` — no regressions

  **QA Scenarios**:

  ```
  Scenario: Tasks page shows error on query failure
    Tool: Bash (vitest)
    Preconditions: Mock apiClient.get → reject
    Steps:
      1. cd frontend && npx vitest run src/pages/__tests__/Tasks.test.tsx -t "error" -v
    Expected Result: Error message/panel rendered
    Failure Indicators: Blank page, loading skeleton only
    Evidence: .sisyphus/evidence/task-8-tasks-error.txt

  Scenario: Logs page shows error on query failure
    Tool: Bash (vitest)
    Preconditions: Mock apiClient.get → reject
    Steps:
      1. cd frontend && npx vitest run src/pages/__tests__/Logs.test.tsx -t "error" -v
    Expected Result: Error message rendered
    Failure Indicators: Blank or infinite loading
    Evidence: .sisyphus/evidence/task-8-logs-error.txt

  Scenario: Pipeline page shows error on query failure
    Tool: Bash (vitest)
    Preconditions: Mock apiClient.post → reject
    Steps:
      1. cd frontend && npx vitest run src/pages/__tests__/Pipeline.test.tsx -t "error" -v
    Expected Result: Error message rendered
    Failure Indicators: Blank state
    Evidence: .sisyphus/evidence/task-8-pipeline-error.txt
  ```

  **Evidence to Capture**:
  - [ ] task-8-tasks-error.txt
  - [ ] task-8-logs-error.txt
  - [ ] task-8-pipeline-error.txt

  **Commit**: YES
  - Message: `fix(frontend): add error handling to Tasks/Logs/Pipeline pages`
  - Files: `frontend/src/pages/Tasks.tsx`, `frontend/src/pages/Logs.tsx`, `frontend/src/pages/Pipeline.tsx`, `frontend/src/pages/__tests__/Tasks.test.tsx`, `frontend/src/pages/__tests__/Logs.test.tsx`

---

- [x] 9. **Fix Pipeline.tsx POST anti-pattern (useQuery → useMutation)**

  **What to do** (TDD):
  - **RED**: Write/update test in `frontend/src/pages/__tests__/Pipeline.test.tsx`:
    - `test_pipeline_uses_mutation_not_query`: Assert the component uses useMutation (not useQuery) for the /pipeline/run endpoint. Verify that page refresh (component remount) does NOT trigger pipeline run.
    - `test_pipeline_shows_result_on_success`: Mock mutation success; assert results rendered.
  - **GREEN** (frontend/src/pages/Pipeline.tsx lines 18-22):
    - Replace `useQuery` with `useMutation`:
      ```tsx
      const mutation = useMutation({
        mutationFn: () => apiClient.post<PipelineRunResponse>("/pipeline/run", { pipeline_type: pipelineType }),
      })
      ```
    - Replace `data`/`isLoading` references: `mutation.data`, `mutation.isPending`
    - Update trigger: `onClick={() => mutation.mutate()}` instead of `setTriggered(true)`
    - Remove `triggered` state variable (no longer needed with useMutation)
    - Also add error handling (Task 8 covers this; coordinate)
  - **REFACTOR**: Remove the `triggered` state + `enabled: triggered` pattern

  **Must NOT do**:
  - Do NOT change the pipeline endpoint or request shape
  - Do NOT change the pipeline types (daily, full, db_full)
  - Do NOT remove the pipeline type selector

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Hook replacement with clear pattern; limited scope to one file
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 7–8)
  - **Parallel Group**: Wave 3
  - **Blocks**: None
  - **Blocked By**: None (frontend; no backend dependency)
  - **⚠️ Coordination**: Task 8 also modifies Pipeline.tsx (error handling). Execute Task 9 first, then Task 8 adds error handling on top.

  **References**:
  - `frontend/src/pages/Pipeline.tsx:18-22` — useQuery used for POST /pipeline/run with `enabled: triggered`
  - `frontend/src/pages/Pipeline.tsx:14-16` — `triggered` state + `setTriggered` for enabling query
  - **WHY**: useQuery for POST causes re-trigger on page refresh/stale-while-revalidate

  **Acceptance Criteria**:
  - [ ] `cd frontend && npx vitest run src/pages/__tests__/Pipeline.test.tsx -t "mutation" -v` — PASS
  - [ ] `cd frontend && npx vitest run src/pages/__tests__/Pipeline.test.tsx -t "pipeline" -v` — ALL pipeline tests PASS
  - [ ] `cd frontend && npm test` — no regressions
  - [ ] Component no longer imports `useQuery` for pipeline run; uses `useMutation` instead

  **QA Scenarios**:

  ```
  Scenario: Pipeline uses useMutation (not useQuery)
    Tool: Bash (vitest)
    Preconditions: Mock apiClient
    Steps:
      1. cd frontend && npx vitest run -t "test_pipeline_uses_mutation_not_query"
    Expected Result: useMutation used; no query-triggered re-fetch on mount
    Failure Indicators: Still uses useQuery with enabled flag
    Evidence: .sisyphus/evidence/task-9-mutation.txt

  Scenario: Pipeline run shows results on success
    Tool: Bash (vitest)
    Preconditions: Mock mutation success
    Steps:
      1. cd frontend && npx vitest run -t "test_pipeline_shows_result_on_success"
    Expected Result: Pipeline result data rendered
    Failure Indicators: No result shown, mutation not called
    Evidence: .sisyphus/evidence/task-9-results.txt

  Scenario: Page refresh does not re-trigger pipeline
    Tool: Bash (vitest)
    Preconditions: Component mounted without user clicking Run
    Steps:
      1. Verify apiClient.post NOT called on initial render
    Expected Result: apiClient.post call count = 0 before user action
    Failure Indicators: apiClient.post called on mount (useQuery auto-fetch)
    Evidence: .sisyphus/evidence/task-9-no-re-trigger.txt
  ```

  **Evidence to Capture**:
  - [ ] task-9-mutation.txt
  - [ ] task-9-results.txt
  - [ ] task-9-no-re-trigger.txt

  **Commit**: YES
  - Message: `fix(frontend): switch Pipeline page from useQuery to useMutation`
  - Files: `frontend/src/pages/Pipeline.tsx`, `frontend/src/pages/__tests__/Pipeline.test.tsx`

---

## Final Verification Wave (MANDATORY — after ALL implementation tasks)

> 4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.
>
> **Do NOT auto-proceed after verification. Wait for user's explicit approval before marking work complete.**

- [x] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, curl endpoint, run command). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist in `.sisyphus/evidence/`. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [x] F2. **Code Quality Review** — `unspecified-high`
  Run `ruff check .` + `cd frontend && npx tsc -b --noEmit` + `pytest` + `cd frontend && npm test`. Review all changed files for: `as any`/`@ts-ignore`, empty catches, console.log, commented-out code, unused imports. Check AI slop: excessive comments, over-abstraction, generic names.
  Output: `Build [PASS/FAIL] | Lint [PASS/FAIL] | Tests [N pass/N fail] | Files [N clean/N issues] | VERDICT`

- [x] F3. **Real Manual QA** — `unspecified-high`
  Start from clean state. Execute EVERY QA scenario from EVERY task — follow exact steps, capture evidence. Test cross-task integration. Test edge cases: empty input, error states, concurrent operations. Save to `.sisyphus/evidence/final-qa/`.
  Output: `Scenarios [N/N pass] | Integration [N/N] | Edge Cases [N tested] | VERDICT`

- [x] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual diff (git log/diff). Verify 1:1 — everything in spec was built (no missing), nothing beyond spec was built (no creep). Check "Must NOT do" compliance. Detect cross-task contamination. Flag unaccounted changes.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

- **1**: `fix(deps): consolidate Python dependencies to pyproject.toml` — pyproject.toml, requirements.txt (deleted)
- **2**: `fix(feishu): propagate table filter to export/sync/import` — services/feishu_service.py, tests/test_feishu_service.py
- **3**: `fix(dispatch): make dry-run truly read-only` — services/dispatch_service.py, tests/test_dispatch_service.py
- **4**: `fix(tasks): align schema fields with DB query columns` — api/schemas.py, services/task_service.py, tests/test_task_schema.py
- **5**: `fix(db): skip unknown tables in get_table_counts with warning` — services/db_service.py, tests/test_db_service.py
- **6**: `fix(logs): use reverse seek for tail instead of full file read` — services/log_reader.py, tests/test_log_reader.py
- **7**: `fix(frontend): recognize synced/imported as OK status` — frontend/src/pages/Feishu.tsx, frontend/src/pages/__tests__/Feishu.test.tsx
- **8**: `fix(frontend): add error handling to Tasks/Logs/Pipeline pages` — frontend/src/pages/Tasks.tsx, Logs.tsx, Pipeline.tsx, __tests__/*
- **9**: `fix(frontend): switch Pipeline page from useQuery to useMutation` — frontend/src/pages/Pipeline.tsx, __tests__/Pipeline.test.tsx

---

## Success Criteria

### Verification Commands
```bash
# Backend tests
pytest tests/ -v

# Frontend tests
cd frontend && npm test

# Dependency check
uv pip check

# Install from pyproject.toml
uv pip install -e .

# Check no requirements.txt remains
test ! -f requirements.txt || echo "FAIL: requirements.txt still present"
```

### Final Checklist
- [ ] All 10 findings have TDD tests passing
- [ ] All existing tests still green
- [ ] `uv pip check` returns zero conflicts
- [ ] pyproject.toml is the single dependency source
- [ ] requirements.txt deleted
- [ ] All "Must Have" present
- [ ] All "Must NOT Have" absent
