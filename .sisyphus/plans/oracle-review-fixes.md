# Fix Oracle-Identified Issues — Backend + Frontend

## TL;DR

> **Quick Summary**: Fix 12 HIGH+MEDIUM findings from Oracle code review. SQL patterns refactored for safety, rate limiting added, frontend error boundaries and a11y labels implemented. All fixes follow TDD where code logic changes.

> **Deliverables**:
> - 5 backend security fixes (SQL patterns, rate limiting, transaction rollback, subprocess sanitization)
> - 3 backend optimizations (migration command, config caching, env vars dedup)
> - 3 frontend improvements (error boundary, a11y labels, Logs.tsx hook extraction)
> - 1 config cleanup (router tag fix)

> **Estimated Effort**: Medium
> **Parallel Execution**: YES — 4 waves
> **Critical Path**: Wave 1 (config) → Waves 2-4 (parallel within each wave)

---

## Context

### Original Request
Oracle code review identified ~15 issues across HIGH and MEDIUM priority. User selected "ALL HIGH + MEDIUM" scope.

### Oracle Review Key Findings
- **SQL patterns**: 7 f-string interpolations are currently safe (hardcoded literals) but fragile — refactor to plain concatenation for auditability
- **Security gaps**: No rate limiting on dispatch, no transaction rollback, subprocess args unsanitized
- **Frontend gaps**: Missing error boundaries, missing a11y labels, duplicate query logic
- **Config issues**: Migration runs on every startup, duplicate router tags, env vars duplicated

### Metis Review
**Key Directives**:
- SQL fixes: Remove f-strings but keep same query logic (TDD verifies before/after results match)
- Rate limiting: Per-token (not per-IP) to avoid NAT false positives
- ErrorBoundary: Must be class-based (React 19 limitations)
- httpOnly cookie: **OUT OF SCOPE** — document only in OAUTH_GUIDE
- T14 (scripts JSON logging): **WONTFIX** — scripts intentionally use text format for TTY readability

---

## Work Objectives

### Core Objective
Fix all 12 actionable HIGH+MEDIUM findings from Oracle review with TDD. Zero regressions on existing test suite.

### Concrete Deliverables
- `services/task_service.py` — f-string removed from WHERE clause
- `services/alert_service.py` — f-string removed from WHERE clause
- `services/db_service.py` — f-string removed from table name queries
- `api/main.py` — f-string removed from PRAGMA; migration deferred; router tag fixed
- `services/feishu_service.py` — subprocess args sanitized; config cached
- `services/dispatch_service.py` — transaction rollback on failure
- `services/pipeline_service.py` — env vars deduplicated
- `api/routers/outbox.py` — dispatch endpoint rate limited
- `api/dependencies.py` — rate limiting middleware
- `frontend/src/components/ErrorBoundary.tsx` — new React 19 error boundary
- `frontend/src/App.tsx` — wrapped in ErrorBoundary
- `frontend/src/pages/Alerts.tsx`, `Tasks.tsx` — select a11y labels
- `frontend/src/hooks/useLogQuery.ts` — extracted from Logs.tsx
- `frontend/src/pages/Logs.tsx` — uses useLogQuery hook

### Definition of Done
- [ ] All 12 findings have passing TDD tests or verified config changes
- [ ] `pytest tests/ -v` — 281+ tests, 0 failures
- [ ] `cd frontend && npm test` — 27+ tests, 0 failures
- [ ] `grep -rn "execute(f'" services/ api/` returns 0 matches
- [ ] Rate limit test: 5 requests OK, 6th returns 429

### Must Have
- SQL f-strings removed from all 4 files
- Rate limiting on dispatch endpoint
- Transaction rollback on dispatch failure
- Error boundary on frontend
- All existing tests green

### Must NOT Have (Guardrails)
- **NO** ORM introduction
- **NO** new package dependencies (Redis, slowapi, etc.)
- **NO** abstract SQL query builder — just remove f-strings
- **NO** httpOnly cookie implementation
- **NO** changes to query semantics — only string construction method
- **NO** rate limiting on non-dispatch endpoints
- **NO** rate limiting per-IP — use per-token
- **NO** generic hook extraction beyond Logs.tsx

---

## Verification Strategy

### Test Decision
- **Infrastructure exists**: YES (pytest + vitest)
- **Automated tests**: TDD for code logic changes; direct verification for config changes
- **Framework**: pytest (backend), vitest (frontend)

### QA Policy
- **Backend**: Bash (curl for rate limiting, pytest for SQL/compat)
- **Frontend**: Bash (vitest run) for component tests

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Quick config wins — MAX PARALLEL):
├── Task 1: Fix router tag + migration deferred [quick]
├── Task 2: Remove empty frontend directories [quick]
├── Task 3: JSON logger config [quick]

Wave 2 (Backend security — MAX PARALLEL):
├── Task 4: SQL f-string removal (4 files) [unspecified-high]
├── Task 5: Rate limiting on dispatch [unspecified-high]
├── Task 6: Transaction rollback [unspecified-high]
├── Task 7: Subprocess args sanitization [quick]

Wave 3 (Backend optimization — MAX PARALLEL):
├── Task 8: Feishu config caching [quick]
├── Task 9: Pipeline env vars dedup [quick]

Wave 4 (Frontend — MAX PARALLEL):
├── Task 10: Error boundary [unspecified-high]
├── Task 11: A11y select labels [quick]
├── Task 12: Extract useLogQuery hook [quick]

Wave FINAL:
├── Task F1: Plan compliance audit (oracle)
├── Task F2: Code quality review (unspecified-high)
├── Task F3: Real QA — run all tests (unspecified-high)
├── Task F4: Scope fidelity check (deep)
```

### Agent Dispatch Summary
- **Wave 1**: 3 tasks — all `quick`
- **Wave 2**: 4 tasks — T4→unspecified-high, T5→unspecified-high, T6→unspecified-high, T7→quick
- **Wave 3**: 2 tasks — both `quick`
- **Wave 4**: 3 tasks — T10→unspecified-high, T11→quick, T12→quick
- **FINAL**: 4 tasks — F1→oracle, F2→unspecified-high, F3→unspecified-high, F4→deep

---

- [ ] 1. **Fix duplicate router tag + defer migration to explicit command**

  **What to do**:
  - `api/main.py:175`: Change `tags=["Logs"]` to `tags=["Diagnosis"]` for diagnosis router
  - `api/main.py:160-162`: Remove `_migrate_if_needed()` call from startup event. Add a `if __name__ == "__main__":` block that prints a warning if not migrated. Document migration command: `python3 -m api.main --migrate`.

  **Must NOT do**: Do NOT remove migration code itself — only defer it
  **Recommended Agent Profile**: `quick`
  **Parallelization**: Wave 1, parallel with Tasks 2-3, blocked by: none

  **Acceptance Criteria**:
  - [ ] `python3 -c "import api.main"` does NOT trigger migration
  - [ ] OpenAPI docs show "Diagnosis" tag (not "Logs")
  - [ ] `pytest tests/ -v` — no regressions

  **QA Scenarios**:
  ```
  Scenario: Importing app does not trigger migration
    Tool: Bash
    Steps: python3 -c "from api.main import app; print('OK')" 2>&1
    Expected: "OK" only, no migration output
    Evidence: .sisyphus/evidence/task-1-no-migration.txt
  ```

  **Commit**: `fix(api): fix router tag and defer migration`

---

- [ ] 2. **Remove empty frontend directories**

  **What to do**: Delete empty directories: `frontend/src/hooks/` and `frontend/src/lib/`. Note: if hooks/ already has files from Task 12, only delete lib/.

  **Must NOT do**: Do NOT delete any directories with files in them
  **Recommended Agent Profile**: `quick`
  **Parallelization**: Wave 1, parallel with Tasks 1, 3, blocked by: none

  **Acceptance Criteria**:
  - [ ] `test -d frontend/src/lib` returns 1 (does not exist)
  - [ ] `cd frontend && npm test` — no regressions

  **Commit**: `chore(frontend): remove empty directories`

---

- [ ] 3. **Add JSON formatter to Python logging config**

  **What to do**: In `api/logging_config.py`, add `python-json-logger` formatter to the default logging config. Only add JSON formatter if the library is importable (guard with try/except). Keep text formatter as fallback.

  **Must NOT do**: Do NOT add new pip dependency; do NOT change log output format for existing handlers
  **Recommended Agent Profile**: `quick`
  **Parallelization**: Wave 1, parallel with Tasks 1-2, blocked by: none

  **Acceptance Criteria**:
  - [ ] JSON logging config added to logging_config.py
  - [ ] `python3 -c "from api.logging_config import *"` exits 0

  **Commit**: `fix(logging): add JSON formatter config`

---

- [ ] 4. **Remove f-string from SQL query construction (4 files)**

  **What to do** (TDD):
  - **RED**: Add tests verifying query results unchanged after refactoring
  - **GREEN**: Replace `f"SELECT ... FROM {table} WHERE {where}"` with plain concat:
    - `services/task_service.py:28,49`: `"SELECT ... FROM action_tasks " + where + " ORDER BY ... LIMIT ?"`
    - `services/alert_service.py:31,52`: Same pattern for alert_events
    - `services/db_service.py:68,85`: `"SELECT COUNT(*) FROM " + table_name`
    - `api/main.py:75`: `"PRAGMA table_info(" + table + ")"`

  **Must NOT do**: Do NOT change query semantics; do NOT introduce ORM
  **Recommended Agent Profile**: `unspecified-high`
  **Parallelization**: Wave 2, parallel with Tasks 5-7, blocked by: none

  **Acceptance Criteria**:
  - [ ] `grep -rn "execute(f'" services/ api/` returns 0 matches
  - [ ] `pytest tests/ -v` — 281+ tests, 0 failures

  **Commit**: `fix(sql): remove f-string from query construction`

---

- [ ] 5. **Add rate limiting to dispatch endpoint**

  **What to do** (TDD):
  - **RED**: Test POSTs to `/v1/dispatch` 5 times (OK), 6th returns 429
  - **GREEN**: Add in-memory rate limiter in `api/dependencies.py` using token-based keys. 5 req/60s per token. Apply to dispatch endpoint only.

  **Must NOT do**: Do NOT add new deps; per-token NOT per-IP; only dispatch endpoint
  **Recommended Agent Profile**: `unspecified-high`
  **Parallelization**: Wave 2, parallel with Tasks 4, 6-7, blocked by: none

  **Acceptance Criteria**:
  - [ ] 6 curl requests → first 5 return 2xx, 6th returns 429
  - [ ] `pytest tests/ -v` — no regressions

  **Commit**: `feat(api): add rate limiting to dispatch endpoint`

---

- [ ] 6. **Add transaction rollback to dispatch**

  **What to do** (TDD):
  - **RED**: Test seeds 3 events, mocks failure on 2nd, asserts 0 changed rows (rolled back)
  - **GREEN**: In `write_result()`, wrap UPDATE in explicit BEGIN/COMMIT. On exception, ROLLBACK.

  **Must NOT do**: Do NOT change dispatch logic or retry behavior
  **Recommended Agent Profile**: `unspecified-high`
  **Parallelization**: Wave 2, parallel with Tasks 4-5, 7, blocked by: none

  **Acceptance Criteria**:
  - [ ] `pytest tests/test_dispatch_service.py -v` — rollback test passes
  - [ ] `pytest tests/ -v` — 281+ tests, 0 failures

  **Commit**: `fix(dispatch): add transaction rollback`

---

- [ ] 7. **Sanitize subprocess args in feishu_service**

  **What to do**: In `_run_script()`, reject args containing `;`, `|`, `&`, `$`, `` ` ``. Ensure `shell=False`.
  **Recommended Agent Profile**: `quick`
  **Parallelization**: Wave 2, parallel with Tasks 4-6, blocked by: none

  **Acceptance Criteria**:
  - [ ] `pytest tests/test_feishu_service.py -v` — sanitization test passes
  - [ ] Feishu service still functions correctly

  **Commit**: `fix(feishu): sanitize subprocess args`

---

- [ ] 8. **Cache Feishu config**

  **What to do**: In `services/feishu_service.py`, add class-level `_config_cache`. Load config once on first access. Reset cache when env vars change.
  **Recommended Agent Profile**: `quick`
  **Parallelization**: Wave 3, parallel with Task 9, blocked by: none

  **Acceptance Criteria**:
  - [ ] `pytest tests/test_feishu_service.py -v` — cache test passes
  - [ ] Config loaded only once per instance

  **Commit**: `perf(feishu): cache config on instance`

---

- [ ] 9. **Deduplicate pipeline env vars**

  **What to do**: In `services/pipeline_service.py`, extract hardcoded env var list to `config.py` as a constant. Reference from both pipeline_service and any other consumers.
  **Recommended Agent Profile**: `quick`
  **Parallelization**: Wave 3, parallel with Task 8, blocked by: none

  **Acceptance Criteria**:
  - [ ] `pytest tests/test_pipeline_api.py -v` — all pass
  - [ ] Only 1 source of truth for pipeline env var list

  **Commit**: `refactor(pipeline): deduplicate env var list`

---

- [ ] 10. **Add ErrorBoundary to frontend**

  **What to do**: Create `frontend/src/components/ErrorBoundary.tsx` as class component (React 19 requires class for error boundaries). Wrap App.tsx routes in ErrorBoundary. Show fallback UI on uncaught errors.
  **Recommended Agent Profile**: `unspecified-high`
  **Parallelization**: Wave 4, parallel with Tasks 11-12, blocked by: none

  **Acceptance Criteria**:
  - [ ] `cd frontend && npm test` — error boundary test passes
  - [ ] React crash shows fallback UI, not blank screen

  **Commit**: `feat(frontend): add error boundary`

---

- [ ] 11. **Add a11y labels to select elements**

  **What to do**: In `Alerts.tsx` and `Tasks.tsx`, add `aria-label` to all `<select>` elements. Use descriptive labels like "Filter by status", "Filter by priority".
  **Recommended Agent Profile**: `quick`
  **Parallelization**: Wave 4, parallel with Tasks 10, 12, blocked by: none

  **Acceptance Criteria**:
  - [ ] `grep -c "aria-label" frontend/src/pages/Alerts.tsx` returns >= 2
  - [ ] `cd frontend && npm test` — no regressions

  **Commit**: `fix(frontend): add a11y labels to selects`

---

- [ ] 12. **Extract useLogQuery hook**

  **What to do**: Create `frontend/src/hooks/useLogQuery.ts` extracting the repeated query logic from Logs.tsx's 3 tab components. Each tab currently duplicates `useQuery({ queryKey: [...], queryFn: () => apiClient.get(...) })`. Extract to `useLogQuery(endpoint: string, params: object)`.
  **Recommended Agent Profile**: `quick`
  **Parallelization**: Wave 4, parallel with Tasks 10-11, blocked by: none

  **Acceptance Criteria**:
  - [ ] `cd frontend && npm test` — existing Logs tests pass
  - [ ] 3 tabs use `useLogQuery()` instead of duplicated `useQuery`

  **Commit**: `refactor(frontend): extract useLogQuery hook`

---

## Final Verification Wave (MANDATORY — after ALL implementation tasks)

- [ ] F1. **Plan Compliance Audit** — `oracle`
- [ ] F2. **Code Quality Review** — `unspecified-high`
- [ ] F3. **Real QA — run all tests** — `unspecified-high`
- [ ] F4. **Scope Fidelity Check** — `deep`

---

## Commit Strategy

- **1**: `fix(api): fix router tag and defer migration` — api/main.py
- **2**: `chore(frontend): remove empty directories` — rm -rf hooks/ lib/
- **3**: `fix(logging): add JSON formatter config` — api/logging_config.py
- **4**: `fix(sql): remove f-string from query construction` — 4 files
- **5**: `feat(api): add rate limiting to dispatch endpoint` — api/dependencies.py, api/routers/outbox.py
- **6**: `fix(dispatch): add transaction rollback` — services/dispatch_service.py
- **7**: `fix(feishu): sanitize subprocess args` — services/feishu_service.py
- **8**: `perf(feishu): cache config on instance` — services/feishu_service.py
- **9**: `refactor(pipeline): deduplicate env var list` — services/pipeline_service.py
- **10**: `feat(frontend): add error boundary` — ErrorBoundary.tsx, App.tsx
- **11**: `fix(frontend): add a11y labels to selects` — Alerts.tsx, Tasks.tsx
- **12**: `refactor(frontend): extract useLogQuery hook` — Logs.tsx, useLogQuery.ts

---

## Success Criteria

### Verification Commands
```bash
# No f-string SQL patterns remain
grep -rn "execute(f'" services/ api/

# All tests pass
pytest tests/ -v
cd frontend && npm test

# Rate limit test
for i in $(seq 1 6); do curl -s -o /dev/null -w "%{http_code}\n" localhost:8000/v1/dispatch; done
```
