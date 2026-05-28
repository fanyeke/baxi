# Baxi Architecture Convergence Plan

## TL;DR

> **Quick Summary**: Fix 5 critical P0 bugs in Go/PostgreSQL decision pipeline, establish quality gates (lint/test/CI), migrate React frontend from Python to Go API, fix ontology schema drift, and add observability views. Qoder/LLM integration excluded (user handles separately).
>
> **Deliverables**:
> - 5 P0 bug fixes (outbox schema, decision case index, action whitelist, API executors, CLI auth)
> - Unified quality gates (`make lint`, `make test-all`)
> - 5 missing Go API endpoints (pipeline, 3x feishu, batch outbox dispatch)
> - Frontend base URL switched to Go API
> - Ontology repository fixed (hardcoded non-existent columns removed)
> - 3 new React observability pages (Decision Case Detail, Audit Timeline, Policy Inspector)
>
> **Estimated Effort**: Large (5 phases, 24 tasks)
> **Parallel Execution**: YES — 5-8 tasks per wave
> **Critical Path**: Phase 0 validation → Phase 1 P0 fixes → Phase 2 quality gates → Phase 3 backend endpoints → Phase 3 frontend switch → Phase 4 ontology → Phase 5 observability → Final verification

---

## Context

### Original Request
User provided a comprehensive 6-stage architecture convergence proposal. Stages 1-5 are to be planned and executed by the agent. Stage 6 (Qoder/LLM integration) is explicitly excluded — user will handle separately.

### Interview Summary
**Key Discussions**:
- Architecture convergence direction: PostgreSQL as sole source, Go API as core business API, Python as legacy gateway, React console connects to Go API
- Priority order: P0 bugs first → quality gates → frontend migration → ontology/governance → observability
- LLM integration deferred to user

**Research Findings** (all validated by parallel agents):
1. **Outbox schema drift**: `migrations/005_ops_tables.sql` creates `ops.outbox_event` without `next_retry_at`, but `internal/outbox/repository.go` expects it. Tests define inline DDL masking the drift.
2. **Decision case unique index**: `source_type`/`source_id` default to `''` (empty string), causing unique index `idx_ai_decision_case_active_source` to conflict on `('', '')`.
3. **Action whitelist bug**: `actions: {}` behaves identically to full config. Canonical actions always allowed. No way to disable via config.
4. **API nil executors**: `internal/api/server.go:408` passes `nil` executors to `NewApplyService`. Real Feishu/GitHub executors exist in worker but not injected into API.
5. **CLI hardcoded**: 5 HTTP calls in `cmd/baxi-cli/` hardcoded to `http://localhost:8080` with no auth, no timeout.
6. **Ontology drift**: `ontology_repository.go` queries columns that don't exist in migrations (`total_payment_value`, `customer_city`, `delivery_status`, `seller_city`, `product_weight_g`).
7. **Go API gaps**: Missing `/pipeline/run`, 3x `/feishu/*`, and batch `/outbox/dispatch` (Go has per-ID dispatch only).

### Metis Review
**Identified Gaps** (addressed):
- **Phase 0 validation needed**: Data audit for decision case duplicates, Feishu code search in Go, pytest hang identification
- **API shape mismatch**: Outbox batch vs per-item is a design decision, not simple endpoint addition
- **Feishu scope unknown**: Could be wrapper (if Go adapter exists) or full implementation
- **Ruff errors mostly auto-fixable**: 831 of 1528 auto-fixable, 376 from `.claude/skills/` (should exclude)
- **Go tests healthier than feared**: 75 test files compile, 473 Python tests collect cleanly

---

## Work Objectives

### Core Objective
Repair the critical decision → review → action → outbox → audit closed loop in Go/PostgreSQL, establish quality gates preventing regression, migrate frontend to Go API, fix ontology schema consistency, and add product-level observability views.

### Concrete Deliverables
- `migrations/006_add_outbox_next_retry_at.sql` — adds missing column
- `migrations/007_fix_decision_case_index.sql` — fixes unique index semantics
- `internal/action/registry.go` — fixed whitelist logic with 3-state semantics
- `internal/api/server.go` — injects real executors into API
- `cmd/baxi-cli/` — shared HTTP client with config-driven URL, auth, timeout
- `Makefile` — unified `lint`, `test-go`, `test-python`, `test-frontend`, `test-all`
- `internal/api/handler/` — 5 new endpoints (pipeline, 3x feishu, batch outbox)
- `frontend/src/api/client.ts` — switched base URL to Go API
- `internal/repository/ontology_repository.go` — fixed column references
- `frontend/src/pages/` — 3 new observability pages

### Definition of Done
- [ ] `make migrate && make test && go test ./test/integration -count=1` all pass
- [ ] `make lint` passes for Go, Python, and frontend
- [ ] Frontend successfully calls Go API for all pages
- [ ] All P0 bugs have agent-executable verification scenarios

### Must Have
1. All 5 P0 bugs fixed with verification
2. Go API has all endpoints frontend needs
3. Ontology queries work against real migration schema
4. Quality gates run in CI

### Must NOT Have (Guardrails)
1. **NO Qoder/LLM integration** — user handles separately
2. **NO production data migration** — only schema fixes, no data backfill
3. **NO changes to Python API business logic** — Python remains legacy gateway
4. **NO modification of applied migrations** — use follow-up migrations only
5. **NO frontend framework changes** — keep React 19 + Vite + TanStack Query

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed. No exceptions.

### Test Decision
- **Infrastructure exists**: YES (Go: testcontainers-go, testify; Python: pytest, ruff; Frontend: vitest)
- **Automated tests**: Tests-after (existing test patterns)
- **Framework**: Go `go test`, Python `pytest`, Frontend `vitest`
- **Agent-Executed QA**: MANDATORY for every task — curl for APIs, grep for code, Playwright for UI

### QA Policy
Every task MUST include agent-executed QA scenarios. Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.
- **API**: `curl` — send requests, assert status + response fields
- **CLI**: `bash` — run command, validate output, check exit code
- **Frontend**: `playwright` — navigate, interact, assert DOM
- **Schema**: `psql` / `goose` — verify column/index existence

---

## Execution Strategy

### Parallel Execution Waves

```
Phase 0 — VALIDATION (Foundation — must complete first):
├── Task V1: Audit decision case duplicates
├── Task V2: Search Feishu code in Go
├── Task V3: Identify pytest hang root cause
└── Task V4: Verify Go auth requirements

Phase 1 — P0 BUG FIXES (Wave 1 — foundation, MAX PARALLEL):
├── Task 1.1: Fix outbox schema drift (migration + index)
├── Task 1.2: Fix decision case unique index (migration + data fix if needed)
├── Task 1.3: Fix action whitelist semantics (3-state config)
├── Task 1.4: Fix API action execution (inject executors + response fix)
└── Task 1.5: Fix CLI hardcoded URLs (shared client + auth + timeout)

Phase 2 — QUALITY GATES (Wave 2 — parallel, after Phase 1):
├── Task 2.1: Fix Python pytest hang (timeout + root cause)
├── Task 2.2: Fix Ruff errors (auto-fix + exclude .claude/)
├── Task 2.3: Add migration contract tests (real Postgres + goose)
└── Task 2.4: Unify Makefile targets (lint, test-go, test-python, test-frontend, test-all)

Phase 3 — FRONTEND MIGRATION (Wave 3 — backend endpoints first, then frontend):
├── Task 3.1: Add POST /pipeline/run endpoint
├── Task 3.2: Add 3x Feishu endpoints (export/sync/status-import)
├── Task 3.3: Add batch POST /outbox/dispatch endpoint
├── Task 3.4: Switch frontend base URL to Go API
└── Task 3.5: Add frontend environment config (VITE_API_BACKEND)

Phase 4 — ONTOLOGY & GOVERNANCE (Wave 4 — parallel, after Phase 3 backend):
├── Task 4.1: Fix ontology repository column references
├── Task 4.2: Update ontology test DDL to match migrations
├── Task 4.3: Version governance configs (hash + loaded_at tracking)
└── Task 4.4: Include policy results in decision context builder

Phase 5 — OBSERVABILITY VIEWS (Wave 5 — parallel, after Phase 3 frontend):
├── Task 5.1: Decision Case Detail page
├── Task 5.2: Audit Timeline page
└── Task 5.3: Policy Inspector page

Wave FINAL — VERIFICATION (after ALL phases):
├── Task F1: Plan compliance audit (oracle)
├── Task F2: Code quality review (unspecified-high)
├── Task F3: End-to-end QA (unspecified-high + playwright)
└── Task F4: Scope fidelity check (deep)
-> Present results -> Get explicit user okay
```

### Dependency Matrix

- **V1-V4**: None (can start immediately)
- **1.1-1.5**: V1 (for 1.2), V2 (for 1.4), V4 (for 1.5)
- **2.1-2.4**: V3 (for 2.1)
- **3.1-3.3**: 1.4 (API executors fixed), V2 (Feishu scope known)
- **3.4-3.5**: 3.1-3.3 (endpoints exist)
- **4.1-4.4**: None (can parallel with Phase 3)
- **5.1-5.3**: 3.4 (frontend base URL switched), 4.4 (policy in context)
- **F1-F4**: ALL above tasks

---

## TODOs

- [x] V1. **Audit decision case duplicates**

  **What to do**:
  - Run SQL query to check for duplicate active decision cases with empty source_type/source_id
  - If duplicates found, document count and affected case IDs
  - If no duplicates, confirm clean state
  **RESULT**: ✅ 0 duplicates — clean state

  **Must NOT do**:
  - Do NOT modify any data — this is read-only audit
  - Do NOT create migration based on audit results (that happens in Task 1.2)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Phase 0 (with V2, V3, V4)
  - **Blocks**: Task 1.2 (if duplicates found, adds data dedup subtask)
  - **Blocked By**: None

  **References**:
  - `migrations/010_ai_tables_enhance.sql:69-71` — unique partial index definition
  - `internal/decision/case_service.go:85-120` — CreateCaseFromAlert hardcodes source_type="alert"

  **Acceptance Criteria**:
  - [ ] Query executed: `SELECT source_type, source_id, COUNT(*) FROM ai.decision_case WHERE status NOT IN ('closed','failed') GROUP BY source_type, source_id HAVING COUNT(*) > 1`
  - [ ] Results documented in `.sisyphus/evidence/v1-duplicate-audit.txt`
  - [ ] If duplicates > 0: list affected case IDs and counts

  **QA Scenarios**:
  ```
  Scenario: Audit query returns clean result
    Tool: Bash (psql)
    Steps:
      1. psql -c "SELECT ..." → capture output
    Expected Result: 0 rows or documented duplicates
    Evidence: .sisyphus/evidence/v1-duplicate-audit.txt
  ```

  **Commit**: NO (read-only audit)

- [x] V2. **Search Feishu code in Go**

  **What to do**:
  - Search all Go packages for existing Feishu/Lark implementation
  - Check `internal/adapter/`, `internal/api/`, `cmd/` for any Feishu-related code
  - Document findings: existing adapter, partial implementation, or zero code
  **RESULT**: ✅ FeishuAdapter exists in `internal/adapter/feishu.go`, but NO HTTP handlers in API. Task 3.2 = wrapper work.

  **Must NOT do**:
  - Do NOT implement anything — this is scope discovery only

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Phase 0 (with V1, V3, V4)
  - **Blocks**: Task 3.2 (Feishu endpoint scope)
  - **Blocked By**: None

  **References**:
  - `cmd/baxi-worker/main.go:78-92` — worker correctly creates FeishuAdapter
  - `internal/adapter/feishu.go` — FeishuAdapter implements ActionExecutor

  **Acceptance Criteria**:
  - [ ] `grep -ri "feishu\|lark" internal/ cmd/ --include="*.go"` executed
  - [ ] Results saved to `.sisyphus/evidence/v2-feishu-search.txt`
  - [ ] Clear verdict: "Go has X Feishu files" or "Zero Feishu code found"

  **QA Scenarios**:
  ```
  Scenario: Search completes with documented results
    Tool: Bash (grep)
    Steps:
      1. grep -ri "feishu" internal/ cmd/ --include="*.go"
    Expected Result: File list or "no matches"
    Evidence: .sisyphus/evidence/v2-feishu-search.txt
  ```

  **Commit**: NO

- [x] V3. **Identify pytest hang root cause**

  **What to do**:
  - Run pytest with timeout to identify which test(s) hang
  - Check if it's: specific test file, all integration tests, fixture issue, or async loop
  - Document root cause
  **RESULT**: ✅ 473 tests pass in 17.20s with pytest-timeout. Root cause = missing timeout. Task 2.1 = add pytest-timeout to config.

  **Must NOT do**:
  - Do NOT fix the hang in this task (that is Task 2.1)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Phase 0 (with V1, V2, V4)
  - **Blocks**: Task 2.1
  - **Blocked By**: None

  **References**:
  - `pytest.ini` — testpaths=tests, no timeout configured
  - `tests/conftest.py` — in_memory_db fixture, auth_headers fixture

  **Acceptance Criteria**:
  - [ ] `pytest --timeout=30 -x` executed (fails fast on first hang)
  - [ ] Hanging test file/name identified
  - [ ] Results saved to `.sisyphus/evidence/v3-pytest-hang.txt`

  **QA Scenarios**:
  ```
  Scenario: Identify hanging test with timeout
    Tool: Bash (pytest)
    Steps:
      1. pytest --timeout=30 -x -v
    Expected Result: Timeout exception pointing to specific test
    Evidence: .sisyphus/evidence/v3-pytest-hang.txt
  ```

  **Commit**: NO

- [x] 1.1. **Fix outbox schema drift — add next_retry_at**

  **What to do**:
  - Create `migrations/006_add_outbox_next_retry_at.sql` with `ALTER TABLE ops.outbox_event ADD COLUMN next_retry_at TIMESTAMPTZ;`
  - Add partial index on `(status, next_retry_at, created_at)` to optimize `GetPendingEvents` query
  - Update `internal/outbox/repository_test.go` to use real migration schema (remove inline DDL with fake column)
  - Add integration test that runs `goose.Up` on real Postgres then verifies repository queries work

  **Must NOT do**:
  - Do NOT modify `migrations/005_ops_tables.sql` (already applied)
  - Do NOT change repository query logic (it already handles NULL correctly)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Phase 1 (with 1.2-1.5)
  - **Blocks**: None
  - **Blocked By**: None

  **References**:
  - `migrations/005_ops_tables.sql:83-98` — original outbox_event schema (missing next_retry_at)
  - `internal/outbox/repository.go:88-121` — GetPendingEvents queries next_retry_at
  - `internal/outbox/repository.go:182-197` — SetNextRetryAt UPDATE
  - `internal/worker/dispatch_worker.go:241-243` — handleFailed calls SetNextRetryAt
  - `internal/outbox/repository_test.go:28` — test DDL incorrectly includes next_retry_at
  - `internal/testutil/db.go` — StartPostgres + RunMigrations helpers

  **Acceptance Criteria**:
  - [ ] Migration file `006_add_outbox_next_retry_at.sql` created
  - [ ] `goose up` applies successfully
  - [ ] `psql -c "SELECT column_name FROM information_schema.columns WHERE table_name='outbox_event' AND column_name='next_retry_at'"` returns 1 row
  - [ ] Integration test passes: `go test ./internal/outbox -run TestRepository_MigrationContract`

  **QA Scenarios**:
  ```
  Scenario: Migration adds column successfully
    Tool: Bash (goose + psql)
    Preconditions: Postgres running with migrations up to 005
    Steps:
      1. goose up
      2. psql -c "SELECT column_name FROM information_schema.columns WHERE table_name='outbox_event' AND column_name='next_retry_at'"
    Expected Result: 1 row returned
    Evidence: .sisyphus/evidence/task-11-migration-up.txt

  Scenario: Repository query works after migration
    Tool: Bash (go test)
    Preconditions: Postgres with all migrations applied
    Steps:
      1. go test ./internal/outbox -run TestRepository_MigrationContract -v
    Expected Result: PASS
    Evidence: .sisyphus/evidence/task-11-contract-test.txt
  ```

  **Commit**: YES
  - Message: `fix(outbox): add next_retry_at column via migration 006`
  - Files: `migrations/006_add_outbox_next_retry_at.sql`, `internal/outbox/repository_test.go`

- [x] 1.2. **Fix decision case unique index**

  **What to do**:
  - Create `migrations/007_fix_decision_case_index.sql`:
    - Drop existing partial unique index
    - Recreate with `WHERE source_type IS NOT NULL AND source_id IS NOT NULL AND status NOT IN ('closed', 'failed')`
    - OR add CHECK constraint `source_type != '' AND source_id != ''`
  - If V1 audit found duplicates, add data deduplication logic (mark duplicates as 'failed' or merge)
  - Change `source_type`/`source_id` columns from `NOT NULL DEFAULT ''` to `DEFAULT NULL`
  - Update repository struct to use `*string` for nullable fields

  **Must NOT do**:
  - Do NOT modify `010_ai_tables_enhance.sql`
  - Do NOT break existing cases with valid source_type/source_id

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Phase 1 (with 1.1, 1.3-1.5)
  - **Blocks**: None
  - **Blocked By**: V1 (duplicate audit results)

  **References**:
  - `migrations/010_ai_tables_enhance.sql:10-12` — column definitions with DEFAULT ''
  - `migrations/010_ai_tables_enhance.sql:69-71` — existing partial unique index
  - `internal/repository/decision_repository.go:22-23` — DecisionCaseRow struct fields
  - `internal/decision/case_service.go:105-106` — hardcoded source_type="alert"
  - `internal/api/handler/decision.go:72` — handler passes source_id as alertID

  **Acceptance Criteria**:
  - [ ] Migration `007_fix_decision_case_index.sql` created and applies successfully
  - [ ] `psql -c "SELECT indexdef FROM pg_indexes WHERE indexname='idx_ai_decision_case_active_source'"` shows updated WHERE clause
  - [ ] Two cases with NULL source_type/source_id can coexist without conflict
  - [ ] Two cases with same non-null source CANNOT coexist (unique constraint enforced)

  **QA Scenarios**:
  ```
  Scenario: NULL sources don't conflict
    Tool: Bash (psql)
    Steps:
      1. INSERT INTO ai.decision_case (case_id, status) VALUES ('c1', 'open')
      2. INSERT INTO ai.decision_case (case_id, status) VALUES ('c2', 'open')
    Expected Result: Both inserts succeed
    Evidence: .sisyphus/evidence/task-12-null-conflict.txt

  Scenario: Non-null duplicate sources blocked
    Tool: Bash (psql)
    Steps:
      1. INSERT INTO ai.decision_case (case_id, source_type, source_id, status) VALUES ('c3', 'alert', 'alert-1', 'open')
      2. INSERT INTO ai.decision_case (case_id, source_type, source_id, status) VALUES ('c4', 'alert', 'alert-1', 'open')
    Expected Result: Second insert fails with unique_violation
    Evidence: .sisyphus/evidence/task-12-unique-block.txt
  ```

  **Commit**: YES
  - Message: `fix(decision): fix unique index to handle NULL sources`
  - Files: `migrations/007_fix_decision_case_index.sql`, `internal/repository/decision_repository.go`

- [x] 1.3. **Fix action whitelist semantics**

  **What to do**:
  - Modify `whitelistActions()` in `internal/action/registry.go` to differentiate 3 states:
    1. **No config / file missing**: Load all canonical actions with defaults (backward compatible)
    2. **Explicit `actions: {}`**: Allow NO actions (empty whitelist)
    3. **Explicit populated config**: Only allow actions listed in config
  - Add `enabled` field to `ActionConfig` struct (default true)
  - Update `IsAllowed()` to check both `whitelist` map AND `config.Actions` entry existence
  - Add tests for all 3 config states
  - Fix `server.go` to not swallow registry errors (lines 143, 406)

  **Must NOT do**:
  - Do NOT change canonical action list
  - Do NOT change ActionConfig struct shape beyond `enabled` field

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Phase 1 (with 1.1-1.2, 1.4-1.5)
  - **Blocks**: None
  - **Blocked By**: None

  **References**:
  - `internal/action/registry.go:14-19` — CanonicalActions list
  - `internal/action/registry.go:55-62` — whitelist map initialization
  - `internal/action/registry.go:87-108` — whitelistActions() with the bug
  - `internal/action/registry.go:111-115` — IsAllowed() check
  - `internal/action/registry_test.go` — existing tests (no empty config test)
  - `config/action_registry.yml` — current config format

  **Acceptance Criteria**:
  - [ ] Test passes: `actions: {}` → `IsAllowed("notify_owner")` returns false
  - [ ] Test passes: no config file → `IsAllowed("notify_owner")` returns true
  - [ ] Test passes: config with only `create_followup_task` → `IsAllowed("notify_owner")` returns false
  - [ ] `go test ./internal/action -run TestWhitelist` passes

  **QA Scenarios**:
  ```
  Scenario: Empty config blocks all actions
    Tool: Bash (go test)
    Steps:
      1. Create temp config with actions: {}
      2. Load registry
      3. Call IsAllowed for each canonical action
    Expected Result: All return false
    Evidence: .sisyphus/evidence/task-13-empty-config.txt

  Scenario: Full config allows only listed actions
    Tool: Bash (go test)
    Steps:
      1. Create temp config with only create_followup_task
      2. Load registry
      3. IsAllowed("create_followup_task") → true
      4. IsAllowed("notify_owner") → false
    Expected Result: Only listed action allowed
    Evidence: .sisyphus/evidence/task-13-partial-config.txt
  ```

  **Commit**: YES
  - Message: `fix(action): fix whitelist semantics for 3 config states`
  - Files: `internal/action/registry.go`, `internal/action/registry_test.go`, `config/action_registry.yml`

- [x] 1.4. **Fix API action execution — inject real executors**

  **What to do**:
  - In `internal/api/server.go` `actionHandler()`, inject real executors (FeishuAdapter, GitHubAdapter) same pattern as worker
  - Pass `cfg.FeishuWebhookURL` and `cfg.GitHubToken` to adapter constructors
  - Update `NewApplyService` call to pass executors map instead of `nil`
  - Fix `HandleExecute` in `internal/api/handler/action.go` to populate `OutboxEventID` in response
  - Ensure dry-run path still works (NoOpExecutor when DryRun=true)

  **Must NOT do**:
  - Do NOT change ApplyService.ExecuteProposal logic
  - Do NOT change executor interface

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Phase 1 (with 1.1-1.3, 1.5)
  - **Blocks**: Task 3.2 (Feishu endpoints need working adapters)
  - **Blocked By**: None

  **References**:
  - `internal/api/server.go:408` — `NewApplyService(reg, nil, loader)`
  - `internal/api/server.go:79-83` — actionExecutors() only returns noop
  - `cmd/baxi-worker/main.go:78-92` — correct executor injection pattern
  - `internal/adapter/feishu.go` — FeishuAdapter
  - `internal/adapter/github.go` — GitHubAdapter
  - `internal/api/handler/action.go:89-101` — response missing OutboxEventID

  **Acceptance Criteria**:
  - [ ] `grep -n "NewApplyService" internal/api/server.go` shows non-nil executors
  - [ ] `curl -X POST /api/v1/proposals/{id}/execute -d '{"dry_run":false}'` returns `outbox_event_id` field
  - [ ] Dry-run still works: `curl -X POST ... -d '{"dry_run":true}'` returns success with empty outbox_event_id

  **QA Scenarios**:
  ```
  Scenario: Real execution returns outbox_event_id
    Tool: Bash (curl)
    Preconditions: API running with real executors, valid proposal exists
    Steps:
      1. curl -X POST http://localhost:8080/api/v1/proposals/{id}/execute -H "Authorization: Bearer $TOKEN" -d '{"dry_run":false}'
    Expected Result: 200 OK, JSON contains outbox_event_id with non-empty UUID
    Evidence: .sisyphus/evidence/task-14-real-execute.txt

  Scenario: Dry-run works without outbox_event_id
    Tool: Bash (curl)
    Steps:
      1. curl -X POST ... -d '{"dry_run":true}'
    Expected Result: 200 OK, outbox_event_id is empty or omitted
    Evidence: .sisyphus/evidence/task-14-dry-run.txt
  ```

  **Commit**: YES
  - Message: `fix(api): inject real executors and return outbox_event_id`
  - Files: `internal/api/server.go`, `internal/api/handler/action.go`

- [x] 1.5. **Fix CLI hardcoded URLs — shared HTTP client**

  **What to do**:
  - Create shared HTTP client helper in `cmd/baxi-cli/`:
    - Reads base URL from `BAXI_API_BASE_URL` env var (default `http://localhost:8080`)
    - Reads bearer token from `API_BEARER_TOKEN` env var
    - Uses `http.Client` with 30-second timeout
    - Drains response body
  - Replace all 5 hardcoded `http.Get`/`http.Post` calls in `decision.go` and `llm.go`
  - Add `--api-url` and `--api-token` CLI flags as overrides
  - Pass `cfg` from `main.go` to `handleLLM()` and decision subcommands

  **Must NOT do**:
  - Do NOT change CLI command structure or flags beyond --api-url/--api-token
  - Do NOT add interactive prompts

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Phase 1 (with 1.1-1.4)
  - **Blocks**: None
  - **Blocked By**: V4 (auth requirements known)

  **References**:
  - `cmd/baxi-cli/decision.go:199,235,269` — 3 hardcoded HTTP calls
  - `cmd/baxi-cli/llm.go:34,57` — 2 hardcoded HTTP calls
  - `cmd/baxi-cli/main.go` — entry point, loads config
  - `internal/config/config.go` — Config struct with APIPort, APIBearerToken
  - `internal/api/middleware/auth.go` — Bearer token auth requirements

  **Acceptance Criteria**:
  - [ ] `grep -r "localhost:8080" cmd/baxi-cli/` returns 0 matches
  - [ ] New shared client file exists with timeout, auth header, env var support
  - [ ] CLI runs successfully against authenticated API: `BAXI_API_BASE_URL=http://localhost:8080 API_BEARER_TOKEN=xxx baxi-cli decision compare --id=xxx`
  - [ ] Error output includes status code and body summary on failure

  **QA Scenarios**:
  ```
  Scenario: CLI with env vars connects to API
    Tool: Bash
    Steps:
      1. BAXI_API_BASE_URL=http://localhost:8080 API_BEARER_TOKEN=$(cat .env | grep API_BEARER_TOKEN | cut -d= -f2) baxi-cli llm status
    Expected Result: 0 exit code, valid JSON output
    Evidence: .sisyphus/evidence/task-15-cli-success.txt

  Scenario: CLI without auth fails with clear error
    Tool: Bash
    Steps:
      1. BAXI_API_BASE_URL=http://localhost:8080 baxi-cli llm status
    Expected Result: Non-zero exit, error contains "401" and "Unauthorized"
    Evidence: .sisyphus/evidence/task-15-cli-auth-fail.txt

  Scenario: CLI with timeout shows error on slow API
    Tool: Bash
    Steps:
      1. BAXI_API_BASE_URL=http://slow-endpoint baxi-cli llm status
    Expected Result: Error contains "timeout" or "deadline exceeded"
    Evidence: .sisyphus/evidence/task-15-cli-timeout.txt
  ```

  **Commit**: YES
  - Message: `fix(cli): shared HTTP client with config-driven URL, auth, timeout`
  - Files: `cmd/baxi-cli/client.go` (new), `cmd/baxi-cli/decision.go`, `cmd/baxi-cli/llm.go`, `cmd/baxi-cli/main.go`

- [x] 2.1. **Fix Python pytest hang**

  **What to do**:
  - Install `pytest-timeout` plugin
  - Add `--timeout=60` to `pytest.ini`
  - Fix root cause identified by V3 (likely: async fixture cleanup, DB connection leak, or FastAPI event loop issue)
  - If fixture issue: fix `conftest.py` fixtures to properly close connections
  - If specific test: mark with `@pytest.mark.skip` temporarily and file follow-up issue

  **Must NOT do**:
  - Do NOT disable all integration tests permanently
  - Do NOT remove pytest coverage

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Phase 2 (with 2.2-2.4)
  - **Blocks**: None
  - **Blocked By**: V3 (hang root cause identified)

  **References**:
  - `pytest.ini` — current pytest config
  - `tests/conftest.py` — in_memory_db fixture (SQLite connection)
  - `pyproject.toml` — deps and ruff config
  - V3 results: `.sisyphus/evidence/v3-pytest-hang.txt`

  **Acceptance Criteria**:
  - [ ] `pytest --timeout=60` completes in <120s
  - [ ] All 473 tests pass or skip with clear reason
  - [ ] No hangs or infinite loops

  **QA Scenarios**:
  ```
  Scenario: pytest completes without hang
    Tool: Bash (pytest)
    Steps:
      1. pytest --timeout=60 -q
    Expected Result: Exits in <120s, shows test count and pass/fail summary
    Evidence: .sisyphus/evidence/task-21-pytest-pass.txt
  ```

  **Commit**: YES
  - Message: `fix(tests): add pytest-timeout and fix hang root cause`
  - Files: `pytest.ini`, `tests/conftest.py`, `pyproject.toml`

- [x] 2.2. **Fix Ruff errors**

  **What to do**:
  - Add `exclude = [".claude/"]` to `pyproject.toml` ruff config (eliminates ~376 invalid-syntax errors)
  - Run `ruff check --fix api services adapters core` for auto-fixable errors (~831)
  - Fix remaining manual errors (likely: unused imports, line length, type annotations)
  - Run `ruff check api services adapters core` until 0 errors

  **Must NOT do**:
  - Do NOT fix `.claude/`, `scripts/`, or generated files
  - Do NOT change business logic to fix lint

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Phase 2 (with 2.1, 2.3-2.4)
  - **Blocks**: None
  - **Blocked By**: None

  **References**:
  - `pyproject.toml` — ruff config section
  - `pytest.ini` — test paths
  - AGENTS.md: Python conventions (Ruff line-length=100, rules E/F/I/N/W/UP)

  **Acceptance Criteria**:
  - [ ] `ruff check api services adapters core` returns 0 errors
  - [ ] `ruff check .` still shows errors in excluded dirs (expected)

  **QA Scenarios**:
  ```
  Scenario: Ruff passes on production code
    Tool: Bash (ruff)
    Steps:
      1. ruff check api services adapters core
    Expected Result: "All checks passed!"
    Evidence: .sisyphus/evidence/task-22-ruff-clean.txt
  ```

  **Commit**: YES
  - Message: `style: fix Ruff errors in production code`
  - Files: `pyproject.toml`, various Python files in api/services/adapters/core

- [x] 2.3. **Add migration contract tests**

  **What to do**:
  - Create `internal/repository/migration_contract_test.go`:
    - Uses `testutil.StartPostgres()` + `testutil.RunMigrations()` (real goose migrations)
    - Tests that each repository can query its table after migrations
    - Covers: outbox, decision_case, ontology (dwd tables), governance configs
  - Remove inline DDL from tests that mask schema drift (e.g., `ontology_repository_test.go`)
  - Add CI step to run contract tests

  **Must NOT do**:
  - Do NOT modify migration files themselves
  - Do NOT remove unit tests that use in-memory SQLite

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Phase 2 (with 2.1-2.2, 2.4)
  - **Blocks**: None
  - **Blocked By**: 1.1 (outbox migration must exist first for contract test)

  **References**:
  - `internal/testutil/db.go` — StartPostgres, RunMigrations helpers
  - `migrations/` — all goose migration files
  - `internal/outbox/repository_test.go` — example of inline DDL masking drift
  - `internal/repository/ontology_repository_test.go` — inline DDL with non-existent columns

  **Acceptance Criteria**:
  - [ ] `go test ./internal/repository -run TestMigrationContract` passes
  - [ ] Test uses real migrations (not inline DDL)
  - [ ] All repository queries succeed against migrated schema

  **QA Scenarios**:
  ```
  Scenario: Contract test validates schema alignment
    Tool: Bash (go test)
    Steps:
      1. go test ./internal/repository -run TestMigrationContract -v
    Expected Result: PASS (all repositories query successfully)
    Evidence: .sisyphus/evidence/task-23-contract-test.txt
  ```

  **Commit**: YES
  - Message: `test: add migration contract tests against real Postgres`
  - Files: `internal/repository/migration_contract_test.go` (new)

- [x] 2.4. **Unify Makefile targets**

  **What to do**:
  - Add to `Makefile`:
    - `make lint`: runs `go vet ./...`, `ruff check api services adapters core`, `cd frontend && npm run lint` (if exists)
    - `make test-go`: runs `go test ./... -count=1`
    - `make test-python`: runs `pytest --timeout=60 -q`
    - `make test-frontend`: runs `cd frontend && npm test -- --run`
    - `make test-all`: runs all three test targets
  - Update existing `make test` to run only Go unit tests (exclude integration tests with `-short`)
  - Add `make test-integration`: runs `go test ./test/... -count=1`

  **Must NOT do**:
  - Do NOT remove existing targets (backward compatibility)
  - Do NOT change `make api` or `make worker` behavior

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Phase 2 (with 2.1-2.3)
  - **Blocks**: None
  - **Blocked By**: None

  **References**:
  - `Makefile` — existing targets
  - AGENTS.md: Commands reference

  **Acceptance Criteria**:
  - [ ] `make lint` runs all linters
  - [ ] `make test-go` runs Go tests
  - [ ] `make test-python` runs Python tests with timeout
  - [ ] `make test-frontend` runs vitest
  - [ ] `make test-all` runs all three

  **QA Scenarios**:
  ```
  Scenario: Unified test targets work
    Tool: Bash (make)
    Steps:
      1. make lint
      2. make test-go
      3. make test-python
      4. make test-frontend
      5. make test-all
    Expected Result: All targets complete without error
    Evidence: .sisyphus/evidence/task-24-make-targets.txt
  ```

  **Commit**: YES
  - Message: `chore: unify Makefile with lint and test targets`
  - Files: `Makefile`

- [x] 3.1. **Add POST /pipeline/run endpoint**

  **What to do**:
  - Create `internal/api/handler/pipeline.go` with `POST /api/v1/pipeline/run` handler
  - Accept JSON body with `config` field (pipeline configuration)
  - Delegate to existing pipeline runner (`internal/pipeline/runner.go`)
  - Return pipeline run ID and status
  - Follow existing handler patterns (local interface, lazy init)

  **Must NOT do**:
  - Do NOT reimplement pipeline logic — delegate to existing runner
  - Do NOT add new pipeline steps

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with 3.2-3.3)
  - **Parallel Group**: Phase 3 backend
  - **Blocks**: Task 3.4 (frontend needs endpoint)
  - **Blocked By**: None

  **References**:
  - `internal/pipeline/runner.go` — PipelineRunner orchestration
  - `internal/pipeline/steps.py` — step definitions (adapt to Go)
  - `internal/api/handler/decision.go` — handler pattern to follow
  - Python API: `api/routers/pipeline.py` — request/response shape

  **Acceptance Criteria**:
  - [ ] `curl -X POST /api/v1/pipeline/run -d '{"config":"test"}'` returns 200 with run_id
  - [ ] Endpoint protected by auth middleware
  - [ ] Error responses follow 5-field JSON format

  **QA Scenarios**:
  ```
  Scenario: Pipeline run endpoint works
    Tool: Bash (curl)
    Steps:
      1. curl -X POST http://localhost:8080/api/v1/pipeline/run -H "Authorization: Bearer $TOKEN" -d '{"config":"ingest_raw"}'
    Expected Result: 200 OK, JSON with run_id and status
    Evidence: .sisyphus/evidence/task-31-pipeline-run.txt
  ```

  **Commit**: YES
  - Message: `feat(api): add POST /pipeline/run endpoint`
  - Files: `internal/api/handler/pipeline.go` (new), `internal/api/server.go`

- [x] 3.2. **Add 3x Feishu endpoints**

  **What to do**:
  - Create `internal/api/handler/feishu.go` with 3 endpoints:
    - `POST /api/v1/feishu/export` — export report to Feishu
    - `POST /api/v1/feishu/sync` — sync data to Feishu
    - `POST /api/v1/feishu/status/import` — import status from Feishu
  - Reuse existing `FeishuAdapter` (`internal/adapter/feishu.go`) if possible
  - If no Go Feishu client exists, create minimal wrapper using `lark-oapi` SDK or HTTP client
  - Follow Python API request/response shapes for compatibility

  **Must NOT do**:
  - Do NOT implement full Feishu SDK — minimal HTTP wrapper only
  - Do NOT add new Feishu features beyond what Python API has

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with 3.1, 3.3)
  - **Parallel Group**: Phase 3 backend
  - **Blocks**: Task 3.4 (frontend needs endpoints)
  - **Blocked By**: V2 (Feishu scope known)

  **References**:
  - `internal/adapter/feishu.go` — FeishuAdapter (ActionExecutor interface)
  - Python API: `api/routers/feishu.py` — request/response shapes
  - `internal/api/handler/decision.go` — handler pattern

  **Acceptance Criteria**:
  - [ ] `curl -X POST /api/v1/feishu/export -d '{"type":"report"}'` returns 200
  - [ ] `curl -X POST /api/v1/feishu/sync` returns 200
  - [ ] `curl -X POST /api/v1/feishu/status/import -d '{"data":[]}'` returns 200

  **QA Scenarios**:
  ```
  Scenario: Feishu export endpoint works
    Tool: Bash (curl)
    Steps:
      1. curl -X POST http://localhost:8080/api/v1/feishu/export -H "Authorization: Bearer $TOKEN" -d '{"type":"report"}'
    Expected Result: 200 OK, JSON with success status
    Evidence: .sisyphus/evidence/task-32-feishu-export.txt
  ```

  **Commit**: YES
  - Message: `feat(api): add Feishu export/sync/import endpoints`
  - Files: `internal/api/handler/feishu.go` (new), `internal/api/server.go`

- [x] 3.3. **Add batch POST /outbox/dispatch endpoint**

  **What to do**:
  - Add `POST /api/v1/outbox/dispatch` (batch) alongside existing `POST /outbox/{id}/dispatch` (per-item)
  - Accept body: `{ "dry_run": bool, "channel": string, "limit": int }`
  - Query pending outbox events matching criteria
  - Dispatch each via existing executor logic
  - Return summary: `{ "dispatched": int, "failed": int, "event_ids": []string }`

  **Must NOT do**:
  - Do NOT remove per-item dispatch endpoint
  - Do NOT change outbox repository interface

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with 3.1-3.2)
  - **Parallel Group**: Phase 3 backend
  - **Blocks**: Task 3.4 (frontend needs batch dispatch)
  - **Blocked By**: 1.4 (executors must be injected)

  **References**:
  - `internal/api/handler/outbox.go` — existing outbox handlers
  - `internal/outbox/repository.go` — GetPendingEvents, GetEventByID
  - `internal/worker/dispatch_worker.go` — dispatch logic to reuse
  - Python API: `api/routers/outbox.py` — batch dispatch shape

  **Acceptance Criteria**:
  - [ ] `curl -X POST /api/v1/outbox/dispatch -d '{"dry_run":true}'` returns 200 with summary
  - [ ] Batch endpoint respects `channel` and `limit` filters
  - [ ] Dry-run mode does not actually dispatch

  **QA Scenarios**:
  ```
  Scenario: Batch dispatch with dry_run
    Tool: Bash (curl)
    Steps:
      1. curl -X POST http://localhost:8080/api/v1/outbox/dispatch -H "Authorization: Bearer $TOKEN" -d '{"dry_run":true,"channel":"feishu","limit":10}'
    Expected Result: 200 OK, dispatched=0 (dry run), event_ids listed
    Evidence: .sisyphus/evidence/task-33-batch-dispatch.txt
  ```

  **Commit**: YES
  - Message: `feat(api): add batch outbox dispatch endpoint`
  - Files: `internal/api/handler/outbox.go`, `internal/api/server.go`

- [x] 3.4. **Switch frontend base URL to Go API**

  **What to do**:
  - Update `frontend/src/api/client.ts` base URL from Python API (`localhost:8765`) to Go API (`localhost:8080`)
  - Update `frontend/vite.config.ts` proxy target from `:8765` to `:8080`
  - Ensure all existing API calls work against Go endpoints
  - Fix any response shape mismatches between Python and Go APIs

  **Must NOT do**:
  - Do NOT change frontend framework or routing
  - Do NOT add new pages (that is Phase 5)

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with 3.5)
  - **Parallel Group**: Phase 3 frontend
  - **Blocks**: Task 5.1-5.3 (observability pages need working API)
  - **Blocked By**: 3.1-3.3 (all backend endpoints must exist)

  **References**:
  - `frontend/src/api/client.ts` — API client base URL
  - `frontend/vite.config.ts` — Vite proxy config
  - `frontend/src/api/types.ts` — response type definitions

  **Acceptance Criteria**:
  - [ ] `npm run dev` starts frontend successfully
  - [ ] Frontend fetches data from Go API (verify via browser Network tab or curl)
  - [ ] All existing pages load without 404/500 errors

  **QA Scenarios**:
  ```
  Scenario: Frontend calls Go API
    Tool: Playwright
    Steps:
      1. Start Go API on :8080
      2. Start frontend: cd frontend && npm run dev
      3. Open http://localhost:5173
      4. Navigate to Alerts page
    Expected Result: Alerts load from Go API, no 404 errors in console
    Evidence: .sisyphus/evidence/task-34-frontend-switch.png
  ```

  **Commit**: YES
  - Message: `feat(frontend): switch API base URL to Go API`
  - Files: `frontend/src/api/client.ts`, `frontend/vite.config.ts`

- [x] 3.5. **Add frontend environment config**

  **What to do**:
  - Add `.env.example` with `VITE_API_BACKEND=go` (or `python` for fallback)
  - Update `frontend/src/api/client.ts` to read `VITE_API_BASE_URL` env var
  - Support switching between Go and Python API for gradual migration
  - Document env var usage in README

  **Must NOT do**:
  - Do NOT make Python API the default (Go should be default)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with 3.4)
  - **Parallel Group**: Phase 3 frontend
  - **Blocks**: None
  - **Blocked By**: None

  **References**:
  - `frontend/src/api/client.ts` — current hardcoded base URL
  - Vite env var docs: `import.meta.env.VITE_*`

  **Acceptance Criteria**:
  - [ ] `VITE_API_BASE_URL=http://localhost:8080 npm run dev` connects to Go API
  - [ ] `VITE_API_BASE_URL=http://localhost:8765 npm run dev` connects to Python API
  - [ ] Default (no env var) connects to Go API

  **QA Scenarios**:
  ```
  Scenario: Env var controls API backend
    Tool: Bash
    Steps:
      1. grep "import.meta.env.VITE_API_BASE_URL" frontend/src/api/client.ts
    Expected Result: Code reads env var with fallback to Go API
    Evidence: .sisyphus/evidence/task-35-env-config.txt
  ```

  **Commit**: YES
  - Message: `feat(frontend): add VITE_API_BASE_URL environment config`
  - Files: `frontend/.env.example`, `frontend/src/api/client.ts`

- [x] 4.1. **Fix ontology repository column references**

  **What to do**:
  - Update `internal/repository/ontology_repository.go` `objectTableMap`:
    - `total_payment_value` → `payment_value` (all object types)
    - Remove `customer_city` from customer columns (doesn't exist in `dwd.order_level`)
    - Remove `delivery_status` from order columns (doesn't exist)
    - Remove `seller_state` from region columns (not in `dwd.order_level`)
    - Remove `seller_city` from seller columns (not in `dwd.item_level`)
    - Remove `product_weight_g` from product columns (not in `dwd.item_level`)
    - Remove `review_score` from seller/product/category columns (only in `dwd.order_level`)
  - Update `metricColumns` if needed (already uses `payment_value` ✅)

  **Must NOT do**:
  - Do NOT add new columns to migrations
  - Do NOT change table names

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with 4.2-4.4)
  - **Parallel Group**: Phase 4
  - **Blocks**: None
  - **Blocked By**: None

  **References**:
  - `internal/repository/ontology_repository.go:26-54` — objectTableMap with hardcoded columns
  - `migrations/003_dwd_tables.sql` — actual DWD table schemas
  - `config/aip_object_schema.yml` — YAML also references non-existent columns

  **Acceptance Criteria**:
  - [ ] `grep -r "total_payment_value\|customer_city\|delivery_status" internal/repository/` returns 0 matches in non-test code
  - [ ] `go test ./internal/repository -run TestOntology` passes against real migrations

  **QA Scenarios**:
  ```
  Scenario: Ontology queries work against real schema
    Tool: Bash (go test)
    Steps:
      1. go test ./internal/repository -run TestOntology_Queries -v
    Expected Result: PASS, no "column does not exist" errors
    Evidence: .sisyphus/evidence/task-41-ontology-fix.txt
  ```

  **Commit**: YES
  - Message: `fix(ontology): remove hardcoded non-existent column references`
  - Files: `internal/repository/ontology_repository.go`

- [x] 4.2. **Update ontology test DDL to match migrations**

  **What to do**:
  - Fix `internal/repository/ontology_repository_test.go` `ontologyTableDDL`:
    - Use exact column names from `migrations/003_dwd_tables.sql`
    - Use `NUMERIC(18,2)` instead of `DOUBLE PRECISION`
    - Remove columns that don't exist in migrations
  - Ensure test DDL matches production DDL exactly

  **Must NOT do**:
  - Do NOT add columns to test DDL that don't exist in migrations

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with 4.1, 4.3-4.4)
  - **Parallel Group**: Phase 4
  - **Blocks**: None
  - **Blocked By**: 4.1 (column names must be finalized)

  **References**:
  - `internal/repository/ontology_repository_test.go:17-69` — test DDL
  - `migrations/003_dwd_tables.sql` — production DDL

  **Acceptance Criteria**:
  - [ ] Test DDL and migration DDL have identical column names
  - [ ] `go test ./internal/repository` passes

  **QA Scenarios**:
  ```
  Scenario: Test DDL matches migration
    Tool: Bash (diff)
    Steps:
      1. Extract columns from test DDL and migration DDL
      2. diff the two lists
    Expected Result: No differences
    Evidence: .sisyphus/evidence/task-42-ddl-match.txt
  ```

  **Commit**: YES
  - Message: `test: align ontology test DDL with production migrations`
  - Files: `internal/repository/ontology_repository_test.go`

- [x] 4.3. **Version governance configs**

  **What to do**:
  - Add config versioning table: `ops.config_versions`:
    - `config_name TEXT PRIMARY KEY`
    - `version TEXT` (semantic version or git sha)
    - `content_hash TEXT` (SHA-256 of file contents)
    - `loaded_at TIMESTAMPTZ`
    - `active BOOLEAN`
  - Create `internal/governance/config_version.go`:
    - Load config file, compute hash
    - Insert/update config_versions record
    - Provide `GetActiveConfig(name)` helper
  - Track versions for: action_registry.yml, alert_rules.yml, access_policy.yml

  **Must NOT do**:
  - Do NOT change config file formats
  - Do NOT require manual version bumping

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with 4.1-4.2, 4.4)
  - **Parallel Group**: Phase 4
  - **Blocks**: Task 5.3 (Policy Inspector needs config versions)
  - **Blocked By**: None

  **References**:
  - `config/` directory — 29 YAML config files
  - `internal/governance/` — governance engine

  **Acceptance Criteria**:
  - [ ] `ops.config_versions` table exists
  - [ ] Config hash computed and stored on load
  - [ ] `GetActiveConfig("action_registry")` returns current version

  **QA Scenarios**:
  ```
  Scenario: Config version tracked
    Tool: Bash (psql)
    Steps:
      1. psql -c "SELECT config_name, content_hash FROM ops.config_versions WHERE config_name='action_registry'"
    Expected Result: 1 row with non-null hash
    Evidence: .sisyphus/evidence/task-43-config-version.txt
  ```

  **Commit**: YES
  - Message: `feat(governance): add config version tracking`
  - Files: `migrations/008_config_versions.sql`, `internal/governance/config_version.go`

- [x] 4.4. **Include policy results in decision context**

  **What to do**:
  - Update `internal/decision/context_builder.go` to include:
    - Allowed actions (from whitelist)
    - Blocked actions (with reasons)
    - Risk levels per action
    - Human approval requirements
    - Evidence sources
  - Add `PolicyResult` struct to decision DTO
  - Return policy results in `GET /api/v1/decisions/cases/{id}` response

  **Must NOT do**:
  - Do NOT change LLM integration (excluded from this plan)
  - Do NOT modify decision engine core logic

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with 4.1-4.3)
  - **Parallel Group**: Phase 4
  - **Blocks**: Task 5.1 (Decision Case Detail needs policy results)
  - **Blocked By**: 1.3 (whitelist must be fixed first)

  **References**:
  - `internal/decision/context_builder.go` — context building logic
  - `internal/action/registry.go` — whitelist and action config
  - `internal/api/dto/decision.go` — response DTOs

  **Acceptance Criteria**:
  - [ ] `GET /api/v1/decisions/cases/{id}` response includes `policy_results` array
  - [ ] Policy results show allowed/blocked actions with reasons

  **QA Scenarios**:
  ```
  Scenario: Case detail includes policy results
    Tool: Bash (curl)
    Steps:
      1. curl http://localhost:8080/api/v1/decisions/cases/123 -H "Authorization: Bearer $TOKEN"
    Expected Result: JSON contains policy_results array with action rules
    Evidence: .sisyphus/evidence/task-44-policy-context.txt
  ```

  **Commit**: YES
  - Message: `feat(decision): include policy results in case context`
  - Files: `internal/decision/context_builder.go`, `internal/api/dto/decision.go`

- [x] 5.1. **Decision Case Detail page**

  **What to do**:
  - Create `frontend/src/pages/CaseDetail.tsx`:
    - Display case metadata (source, status, severity)
    - Show current metrics
    - Display LLM recommendation (if available)
    - Show evidence list
    - Display risk flags
    - Show approval status
    - Show action status with outbox event linkage
  - Add route `/cases/:id` in `App.tsx`
  - Use existing TanStack Query patterns

  **Must NOT do**:
  - Do NOT implement LLM integration (excluded)
  - Do NOT add write operations (read-only view)

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with 5.2-5.3)
  - **Parallel Group**: Phase 5
  - **Blocks**: None
  - **Blocked By**: 3.4 (frontend switched to Go API), 4.4 (policy in context)

  **References**:
  - `frontend/src/pages/` — existing page patterns
  - `frontend/src/api/types.ts` — decision case types
  - `internal/api/handler/decision.go` — case detail endpoint

  **Acceptance Criteria**:
  - [ ] Page renders at `/cases/:id`
  - [ ] All case fields displayed
  - [ ] Policy results visible
  - [ ] Links to outbox event if action executed

  **QA Scenarios**:
  ```
  Scenario: Case detail page renders
    Tool: Playwright
    Steps:
      1. Navigate to /cases/123
      2. Assert case ID visible
      3. Assert policy results section exists
    Expected Result: All sections rendered, no 404
    Evidence: .sisyphus/evidence/task-51-case-detail.png
  ```

  **Commit**: YES
  - Message: `feat(frontend): add Decision Case Detail page`
  - Files: `frontend/src/pages/CaseDetail.tsx`, `frontend/src/App.tsx`

- [x] 5.2. **Audit Timeline page**

  **What to do**:
  - Create `frontend/src/pages/AuditTimeline.tsx`:
    - Display chronological timeline of case events
    - Events: case created, context built, LLM called, proposal created, review approved/rejected, outbox event created, dispatched/succeeded/failed
    - Use timeline component with timestamps and status icons
  - Add route `/cases/:id/timeline` in `App.tsx`

  **Must NOT do**:
  - Do NOT create new backend endpoints (use existing audit log)

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with 5.1, 5.3)
  - **Parallel Group**: Phase 5
  - **Blocks**: None
  - **Blocked By**: 3.4 (frontend switched to Go API)

  **References**:
  - `internal/api/handler/logs.go` — audit log endpoint
  - `frontend/src/pages/` — existing page patterns

  **Acceptance Criteria**:
  - [ ] Timeline renders at `/cases/:id/timeline`
  - [ ] Events in chronological order
  - [ ] Status icons for each event type

  **QA Scenarios**:
  ```
  Scenario: Audit timeline renders
    Tool: Playwright
    Steps:
      1. Navigate to /cases/123/timeline
      2. Assert timeline container exists
      3. Assert at least 2 events visible
    Expected Result: Timeline with events rendered
    Evidence: .sisyphus/evidence/task-52-audit-timeline.png
  ```

  **Commit**: YES
  - Message: `feat(frontend): add Audit Timeline page`
  - Files: `frontend/src/pages/AuditTimeline.tsx`, `frontend/src/App.tsx`

- [x] 5.3. **Policy Inspector page**

  **What to do**:
  - Create `frontend/src/pages/PolicyInspector.tsx`:
    - Display current action whitelist status
    - Show which actions are allowed/blocked
    - Display which policy config controls each action
    - Show required roles for approval
    - Explain why human approval is needed
  - Add route `/policies/:id` in `App.tsx`

  **Must NOT do**:
  - Do NOT add policy editing (read-only)

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with 5.1-5.2)
  - **Parallel Group**: Phase 5
  - **Blocks**: None
  - **Blocked By**: 4.3 (config versioning)

  **References**:
  - `internal/governance/config_version.go` — config versioning
  - `internal/action/registry.go` — action config
  - `frontend/src/pages/` — existing page patterns

  **Acceptance Criteria**:
  - [ ] Page renders at `/policies/:id`
  - [ ] Action whitelist visible
  - [ ] Config version displayed

  **QA Scenarios**:
  ```
  Scenario: Policy inspector renders
    Tool: Playwright
    Steps:
      1. Navigate to /policies/action_registry
      2. Assert whitelist table exists
      3. Assert config version visible
    Expected Result: Policy details rendered
    Evidence: .sisyphus/evidence/task-53-policy-inspector.png
  ```

  **Commit**: YES
  - Message: `feat(frontend): add Policy Inspector page`
  - Files: `frontend/src/pages/PolicyInspector.tsx`, `frontend/src/App.tsx`

---

## Final Verification Wave

> 4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.

- [x] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, curl endpoint, run command). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist in .sisyphus/evidence/. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [x] F2. **Code Quality Review** — `unspecified-high`
  Run `go vet ./...` + `ruff check api services adapters core` + `cd frontend && npm run build`. Review all changed files for: `as any`/`@ts-ignore`, empty catches, console.log in prod, commented-out code, unused imports. Check AI slop: excessive comments, over-abstraction, generic names.
  Output: `Build [PASS/FAIL] | Lint [PASS/FAIL] | Go Vet [PASS/FAIL] | Files [N clean/N issues] | VERDICT`

- [x] F3. **End-to-End QA** — `unspecified-high` (+ `playwright` skill)
  Start from clean state. Execute EVERY QA scenario from EVERY task — follow exact steps, capture evidence. Test cross-task integration (features working together, not isolation). Test edge cases: empty state, invalid input, rapid actions. Save to `.sisyphus/evidence/final-qa/`.
  Output: `Scenarios [N/N pass] | Integration [N/N] | Edge Cases [N tested] | VERDICT`

- [x] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual diff (git log/diff). Verify 1:1 — everything in spec was built (no missing), nothing beyond spec was built (no creep). Check "Must NOT do" compliance. Detect cross-task contamination.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

- **Phase 0**: No commits (read-only validation)
- **Phase 1**: 1 bug = 1 commit
  - `fix(outbox): add next_retry_at column via migration 006`
  - `fix(decision): fix unique index to handle NULL sources`
  - `fix(action): fix whitelist semantics for 3 config states`
  - `fix(api): inject real executors and return outbox_event_id`
  - `fix(cli): shared HTTP client with config-driven URL, auth, timeout`
- **Phase 2**: 1 infra change = 1 commit
  - `fix(tests): add pytest-timeout and fix hang root cause`
  - `style: fix Ruff errors in production code`
  - `test: add migration contract tests against real Postgres`
  - `chore: unify Makefile with lint and test targets`
- **Phase 3**: 1 endpoint = 1 commit
  - `feat(api): add POST /pipeline/run endpoint`
  - `feat(api): add Feishu export/sync/import endpoints`
  - `feat(api): add batch outbox dispatch endpoint`
  - `feat(frontend): switch API base URL to Go API`
  - `feat(frontend): add VITE_API_BASE_URL environment config`
- **Phase 4**: 1 concern = 1 commit
  - `fix(ontology): remove hardcoded non-existent column references`
  - `test: align ontology test DDL with production migrations`
  - `feat(governance): add config version tracking`
  - `feat(decision): include policy results in case context`
- **Phase 5**: 1 view = 1 commit
  - `feat(frontend): add Decision Case Detail page`
  - `feat(frontend): add Audit Timeline page`
  - `feat(frontend): add Policy Inspector page`

---

## Success Criteria

### Verification Commands
```bash
# Phase 1 P0 fixes
make migrate                          # Expected: all migrations applied
psql -c "SELECT column_name FROM information_schema.columns WHERE table_name='outbox_event' AND column_name='next_retry_at'"  # Expected: 1 row
curl -X POST http://localhost:8080/api/v1/proposals/{id}/execute -H "Authorization: Bearer $TOKEN" -d '{"dry_run":false}'  # Expected: 200 with outbox_event_id
BAXI_API_BASE_URL=http://localhost:8080 API_BEARER_TOKEN=xxx baxi-cli llm status  # Expected: 0 exit code

# Phase 2 quality gates
make lint                             # Expected: all pass
make test-all                         # Expected: all pass
pytest --timeout=60 -q                # Expected: completes in <120s

# Phase 3 frontend migration
curl -X POST http://localhost:8080/api/v1/pipeline/run -H "Authorization: Bearer $TOKEN" -d '{"config":"test"}'  # Expected: 200
curl -X POST http://localhost:8080/api/v1/outbox/dispatch -H "Authorization: Bearer $TOKEN" -d '{"dry_run":true}'  # Expected: 200 with summary
cd frontend && npm run build          # Expected: build succeeds

# Phase 4 ontology
go test ./internal/repository -run TestOntology_Queries  # Expected: PASS
psql -c "SELECT config_name FROM ops.config_versions"  # Expected: rows for action_registry, alert_rules, access_policy

# Phase 5 observability
cd frontend && npm run dev &          # Start dev server
# (Playwright tests for new pages)
```

### Final Checklist
- [ ] All 5 P0 bugs fixed with evidence
- [ ] `make migrate && make test && go test ./test/integration -count=1` all pass
- [ ] `make lint` passes for Go, Python, and frontend
- [ ] Frontend successfully calls Go API for all pages
- [ ] All "Must Have" present
- [ ] All "Must NOT Have" absent
- [ ] All evidence files in `.sisyphus/evidence/`
- [ ] User explicit "okay" on final verification

