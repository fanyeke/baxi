# Plan: Fix All HIGH + MEDIUM Oracle Code Review Findings

## Meta

- **Date**: 2026-05-23
- **Intent**: Mid-sized task (bounded scope, 14 items with clear deliverables)
- **Execution Mode**: Ultrawork with parallel waves
- **Test Strategy**: TDD for code logic changes, direct verification for config/cleanup

## Intent Classification

**Type**: Mid-sized Task
**Confidence**: High
**Rationale**: 14 distinct findings across backend (7), frontend (5), and infra/config (2). Each has a defined scope. No new architecture needed. The key risk is scope creep — especially on SQL refactoring and error boundary design.

---

## Dependency Graph

```
Wave 0 (Pre-flight — can run first)
├── T01: Fix duplicate router tag "Logs" → "Diagnosis" [api/main.py]
├── T03: Remove empty directories [frontend/src/hooks/, frontend/src/lib/]
├── T04: Fix pipeline_service.py env var duplication [services/pipeline_service.py]
├── T11: Move migration from startup to explicit script [api/main.py + scripts/]
│   (no tests — verify `uvicorn` starts without running migration)

Wave 1 (HIGH — security critical, parallel within wave)
├── T02: SQL parameterization cleanup — 7 patterns [4 files]
│   ├── test_sql_param.py (NEW — TDD)
│   └── services/task_service.py
│   └── services/alert_service.py
│   └── services/db_service.py
│   └── api/main.py
├── T05: Subprocess args sanitization [services/feishu_service.py]
│   └── tests/test_feishu_service.py (TDD — add validation test)
├── T06: Transaction safety in dispatch_service.py [services/dispatch_service.py + api/routers/outbox.py]
│   ├── test_dispatch_transaction.py (NEW — TDD)
│   └── tests/test_dispatch_service.py (extend existing)
├── T07: Rate limiting on dispatch endpoint [api/routers/outbox.py]
│   ├── test_rate_limit.py (NEW — TDD)
│   └── api/routers/outbox.py

Wave 2 (MEDIUM — UX/code quality, parallel within wave)
├── T08: Frontend ErrorBoundary wrapper [frontend/src/pages/*]
│   └── All page tests unchanged (ErrorBoundary is transparent)
├── T09: a11y labels on frontend selects [Alerts.tsx, Tasks.tsx]
│   └── Tests: verify aria-label in rendered HTML
├── T10: Feishu config cache/singleton [services/feishu_service.py]
│   └── tests/test_feishu_service.py (extend — test caching behavior)

Wave 3 (MEDIUM — frontend refactor)
└── T12: Extract useLogQuery hook from Logs.tsx [frontend/src/pages/Logs.tsx]
    ├── frontend/src/hooks/useLogQuery.ts (NEW)
    ├── frontend/src/pages/__tests__/Logs.test.tsx (unchanged behavior)
    └── Cleanup: now hooks/ directory is used (not empty)
```

### Parallel Execution Plan

| Wave | Tasks | Run Together? | Blocked By |
|------|-------|---------------|------------|
| 0 | T01, T03, T04, T11 | YES (all independent) | None |
| 1 | T02, T05, T06, T07 | YES (all touch different files/services) | Wave 0 only for T11 (migration move) |
| 2 | T08, T09, T10 | YES (frontend vs backend independent) | None |
| 3 | T12 | Alone (depends on T09 not overwriting Logs.tsx) | T09 must NOT include Logs.tsx |

---

## Task Breakdown

### T01: Fix Duplicate Router Tag

**Category**: quick
**Skills**: []
**File**: `api/main.py` line 175
**Change**: `tags=["Logs"]` → `tags=["Diagnosis"]`
**Verification**: `grep -n 'tags=\["Logs"\]' api/main.py` returns exactly 1 match
**Test**: None needed — visual fix

### T02: SQL Parameterization Cleanup (7 patterns)

**Category**: deep
**Skills**: ["test-driven-development"]

**Current State** (from exploration):
- `task_service.py:28,49` — f-string with `{where}`, where clause built safely with `?` + params — SAFE but fragile
- `alert_service.py:31,52` — same safe pattern — SAFE but fragile
- `db_service.py:68,85` — f-string with `{table_name}` — PROTECTED by `ALLOWED_PUBLIC_TABLES` allowlist
- `api/main.py:75` — f-string with `{table}` — hardcoded names, safe

**Changes Needed**:
1. Replace f-strings with `str.format()` or `+` concatenation to eliminate the f-string SQL anti-pattern
2. Add explicit `validate_sql_identifier()` calls where table names are interpolated
3. Add a test that catches any future f-string SQL in these 4 files

**New Test**: `tests/test_sql_param.py`
```python
# Test: No f-string SQL patterns in service files
# Verify parameterized query patterns work correctly for all 4 services
# Test validate_table_name rejects malicious input
def test_validate_table_name_rejects_malicious():
    with pytest.raises(ValueError):
        validate_table_name("event_outbox; DROP TABLE users;--")

def test_build_task_conditions_uses_placeholders():
    # Verify _build_task_conditions produces "column = ?" patterns
    conditions, params = _build_task_conditions(status="active", priority="high")
    assert "?" in conditions
    assert "active" in params
    assert "high" in params
```

**TDD Steps**:
1. Write test that imports services and checks query construction uses `?` placeholders
2. Write test that `validate_table_name` rejects SQL injection attempts
3. Run tests → fail
4. Refactor f-strings → pass
5. Run grep verification: no `f"SELECT`, `f"UPDATE`, `f"DELETE`, `f"INSERT` in service files

**MUST NOT**:
- Introduce an ORM (SQLAlchemy, etc.)
- Create a generic query builder abstraction
- Change query logic — only change string construction method
- Touch any SQL patterns outside the 4 identified files

### T03: Remove Empty Directories

**Category**: quick
**Skills**: []
**Files**: `frontend/src/hooks/`, `frontend/src/lib/`
**Note**: T12 will need `hooks/` for `useLogQuery.ts`, so DO NOT delete `hooks/`. Only delete `lib/`.
**Verification**: `find frontend/src -type d -empty` returns no matches

### T04: Fix pipeline_service.py Env Var Duplication

**Category**: quick
**Skills**: []
**File**: `services/pipeline_service.py`
**Change**: Derive `_check_env_warnings` from `_REQUIRED_ENV_VARS` (filter Feishu vars) instead of duplicating the list
**Verification**: 
```python
# The warning-check list should be derived, not hardcoded
_FEISHU_ENV_VARS = [v for v in _REQUIRED_ENV_VARS if v.startswith("FEISHU_")]
```
**Test**: None needed — internal refactoring

### T05: Subprocess Args Sanitization

**Category**: quick
**Skills**: ["test-driven-development"]
**File**: `services/feishu_service.py`
**Current State**: Already safe (list-based, no shell=True, validated table names)
**Change**: Add an explicit allowlist for script names and validate args against known types. Add a test confirming no shell=True usage.

**New Test** (add to `tests/test_feishu_service.py`):
```python
def test_subprocess_no_shell_true():
    # Verify _run_script never uses shell=True
    with patch("subprocess.run") as mock_run:
        svc = FeishuService(dry_run=True)
        svc.run_script("db_export_feishu.py", ["--json"])
        mock_run.assert_called_once()
        call_kwargs = mock_run.call_args.kwargs
        assert call_kwargs.get("shell") is not True
```

### T06: Transaction Safety in Dispatch Service

**Category**: deep
**Skills**: ["test-driven-development"]

**Current State**:
- `dispatch_service.py` has NO `conn.rollback()`, NO `conn.commit()`, NO `BEGIN`  
- `outbox.py` calls `conn.commit()` only on success path (line 108)
- If `dispatch_one()` fails mid-loop, partial UPDATEs persist

**Changes**:

**api/routers/outbox.py** — Add rollback on exception:
```python
@router.post("/outbox/dispatch", response_model=DispatchResponse)
def dispatch_outbox(body: DispatchRequest, conn=Depends(get_db)):
    ...
    try:
        for event in pending:
            ...
        _write_api_audit(request_id, audit_entries)
        if not is_dry_run:
            conn.commit()
    except Exception:
        conn.rollback()
        raise
```

**Test** (new `tests/test_dispatch_transaction.py`):
```python
def test_rollback_on_dispatch_failure():
    conn = _make_conn()
    _seed_multiple_events(conn, count=3)
    with patch("dispatch_one", side_effect=[success, Exception("boom"), success]):
        with pytest.raises(Exception, match="boom"):
            dispatch_outbox(...)
    # Verify NO events had their status changed (all rolled back)
    assert _count_changed_events(conn) == 0
```

**TDD Steps**:
1. Write test: dispatch failure → rollback → no partial updates
2. Write test: dispatch success → commit → all updates persist
3. Run → fail (no rollback exists)
4. Add `conn.rollback()` in except block → pass
5. Verify existing `test_dispatch_service.py` tests still pass

**MUST NOT**:
- Change dispatch logic
- Introduce savepoints or nested transactions
- Modify claim_event() atomicity
- Touch non-dispatch routers

### T07: Rate Limiting on Dispatch Endpoint

**Category**: deep
**Skills**: ["test-driven-development"]

**Constraints**:
- NO new dependencies (no slowapi, no Redis)
- Simple in-memory counter per IP (or per Bearer token)
- Must be FastAPI-compatible

**Approach**: Custom FastAPI middleware/deps using a simple token-bucket or sliding-window counter:

```python
# api/dependencies.py
from collections import defaultdict
import time

_rate_limits: dict[str, list[float]] = defaultdict(list)

def check_rate_limit(key: str = "default", max_requests: int = 5, window: int = 60):
    """Simple in-memory rate limiter. max_requests per window seconds per key."""
    now = time.time()
    _rate_limits[key] = [t for t in _rate_limits[key] if now - t < window]
    if len(_rate_limits[key]) >= max_requests:
        from fastapi import HTTPException
        raise HTTPException(status_code=429, detail="Rate limit exceeded")
    _rate_limits[key].append(now)
```

Add to dispatch endpoint:
```python
def dispatch_outbox(..., user=Depends(get_current_user)):
    check_rate_limit(key=user, max_requests=5, window=60)
```

**Test** (new `tests/test_rate_limit.py`):
```python
def test_dispatch_rate_limit_blocks_after_n_requests():
    client = TestClient(app)
    for _ in range(5):
        resp = client.post("/api/v1/outbox/dispatch", json={"apply": False}, headers=auth_headers)
        assert resp.status_code in (200, 422)  # 422 from validation is fine
    resp = client.post("/api/v1/outbox/dispatch", json={"apply": False}, headers=auth_headers)
    assert resp.status_code == 429
    assert "Rate limit" in resp.json()["detail"]

def test_rate_limit_resets_after_window(client: TestClient):
    client = TestClient(app)
    # Exhaust limit
    for _ in range(5):
        client.post("/api/v1/outbox/dispatch", json={"apply": False}, headers=auth_headers)
    # Wait for window to expire
    # Clear the rate limit store to simulate time passage
    from api.dependencies import _rate_limits
    _rate_limits.clear()
    # Should succeed again
    resp = client.post("/api/v1/outbox/dispatch", json={"apply": False}, headers=auth_headers)
    assert resp.status_code != 429

def test_different_users_have_separate_limits(client: TestClient):
    client = TestClient(app)
    # User A exhausts limit
    for _ in range(5):
        client.post("/api/v1/outbox/dispatch", json={"apply": False}, 
                    headers={"Authorization": "Bearer user-a-token"})
    # User B should still be able to dispatch
    resp = client.post("/api/v1/outbox/dispatch", json={"apply": False},
                       headers={"Authorization": "Bearer user-b-token"})
    assert resp.status_code != 429
```

**MUST NOT**:
- Add slowapi, Redis, or external rate limiting
- Implement distributed rate limiting
- Add rate limiting to non-dispatch endpoints
- Use IP-based rate limiting (shared NAT would block legit users — use token-based instead)

### T08: Frontend ErrorBoundary

**Category**: quick
**Skills**: []
**Files**: `frontend/src/pages/*.tsx`
**React version**: 19.1.0 — ErrorBoundary must be class component (no hook-based ErrorBoundary in stable React 19)

**Change**: Create `frontend/src/components/ErrorBoundary.tsx` and wrap all page routes:
```tsx
// class-based ErrorBoundary for React 19
class ErrorBoundary extends React.Component<{children: ReactNode}, {hasError: boolean}> {
  state = { hasError: false };
  static getDerivedStateFromError() { return { hasError: true }; }
  render() {
    if (this.state.hasError) {
      return <div className="p-8 text-center">Something went wrong. Please refresh.</div>;
    }
    return this.props.children;
  }
}
```

Wrap each page in `App.tsx` or router:
```tsx
<Route path="/alerts" element={<ErrorBoundary><Alerts /></ErrorBoundary>} />
```

**Verification**: `grep -rl 'ErrorBoundary' frontend/src/` shows it wraps all 7 pages
**Test**: Behavior unchanged for happy path; no tests needed for ErrorBoundary itself (it's transparent)

### T09: a11y Labels on Select Elements

**Category**: quick
**Skills**: []
**Files**: `frontend/src/pages/Alerts.tsx`, `frontend/src/pages/Tasks.tsx`

**Changes** (add `aria-label` to each select):
- `Alerts.tsx:25` — `aria-label="告警等级"`
- `Alerts.tsx:31` — `aria-label="告警状态"`
- `Tasks.tsx:25` — `aria-label="任务状态"`
- `Tasks.tsx:31` — `aria-label="任务优先级"`

**Test**: Add assertions to existing `Alerts.test.tsx` and `Tasks.test.tsx`:
```tsx
// In Alerts.test.tsx
test("severity select has aria-label", () => {
  render(<Alerts />);
  expect(screen.getByRole("combobox", { name: /告警等级/ })).toBeInTheDocument();
});
```

### T10: Feishu Config Cache/Singleton

**Category**: quick
**Skills**: []
**File**: `services/feishu_service.py`
**Current State**: Already uses lazy init with `self._config` caching (lines 29-52). The finding is about loading config every instance.

**Change**: Make config a module-level singleton instead of per-instance:
```python
_module_config: dict | None = None

def _get_feishu_config() -> dict:
    global _module_config
    if _module_config is not None:
        return _module_config
    # Load once
    _module_config = _load_feishu_config_impl()
    return _module_config
```

**Test** (extend `tests/test_feishu_service.py`):
```python
def test_config_is_cached_across_instances():
    # First instance loads config
    svc1 = FeishuService()
    svc1._load_config()
    # Second instance should reuse the cached config
    svc2 = FeishuService()
    config2 = svc2._load_config()
    assert config2 is _module_config  # Same object reference
```

### T11: Move Migration from Startup to Explicit Command

**Category**: quick
**Skills**: []
**Files**: `api/main.py`, `scripts/run_api.py`
**Current State**: `_apply_migration(DB_PATH)` is called at module import time in `api/main.py` lines 160-162

**Change**: 
1. Remove `_apply_migration` call from `api/main.py` startup code
2. Add a new script: `scripts/run_migration.py` with explicit migration command
3. Update `scripts/run_api.py` to NOT call migration automatically

**Verification**:
```bash
# Migration does NOT run on API start
python -c "from api.main import app; print('API loaded')" 2>&1 | grep -v migration

# Migration DOES run on explicit script
python scripts/run_migration.py  # should show migration output
```

**Test**: None needed — structural change

### T12: Extract useLogQuery Hook from Logs.tsx

**Category**: quick (frontend refactor)
**Skills**: []
**Files**: `frontend/src/pages/Logs.tsx` → `frontend/src/hooks/useLogQuery.ts`

**Current State**: Three tab components (`ErrorsTab`, `AuditTab`, `RecentTab`) each have identical:
- `useQuery` setup
- loading/error/empty state checks
- `<table>` rendering with identical structure

**Change**: Extract shared logic into `useLogQuery` hook and a `LogTable` component:
```tsx
// frontend/src/hooks/useLogQuery.ts
function useLogQuery<T>(endpoint: string, queryKey: string) {
  return useQuery({
    queryKey: [queryKey],
    queryFn: () => apiClient.get<T[]>(endpoint).then(r => r.data),
  });
}

// Use in tab components:
function ErrorsTab() {
  const { data, isLoading, error } = useLogQuery<LogError>("/logs/errors", "log-errors");
  if (isLoading) return <Loader />;
  if (error) return <ErrorMessage />;
  return <LogTable columns={ERROR_COLUMNS} items={data ?? []} />;
}
```

**Verification**: `frontend/src/hooks/` now contains `useLogQuery.ts` (no longer empty)
**Test**: Existing `Logs.test.tsx` should pass unchanged (behavior identical)

### T14: Scripts JSON Logging Config

**Category**: quick
**Skills**: []
**Current State**: `python-json-logger` is configured in `api/logging_config.py` for API file handlers. Scripts use `logging.basicConfig()` with plain text.
**Finding**: "python-json-logger without JSON config" — this is a FINDING, not a bug. The API already has JSON logging. Scripts intentionally use text format for TTY readability.
**Decision**: **CLOSE as WONTFIX**. Document in the findings tracker that scripts intentionally use text logging. No change needed.

---

## Scope Lock

### Must Have (14 items → 13 after T14 wontfix)
1. ✅ T01: Fix duplicate router tag
2. ✅ T02: SQL parameterization cleanup
3. (WONTFIX) T14: Scripts JSON logging — document as intentional design
4. ✅ T03: Remove empty `lib/` directory (keep `hooks/` for T12)
5. ✅ T04: Fix pipeline env var duplication
6. ✅ T05: Subprocess args validation test
7. ✅ T06: Transaction rollback on dispatch failure
8. ✅ T07: Rate limiting on dispatch endpoint
9. ✅ T08: ErrorBoundary wrapper on all pages
10. ✅ T09: a11y labels on 4 select elements
11. ✅ T10: Feishu config module-level singleton
12. ✅ T11: Move migration to explicit command
13. ✅ T12: Extract useLogQuery hook

### Must NOT Have
- **NO** ORM introduction
- **NO** Redis or external rate limiting
- **NO** error reporting/tracking in ErrorBoundary
- **NO** Sentry, logging, or stack traces in ErrorBoundary
- **NO** generic query builder
- **NO** httpOnly cookie implementation (MEDIUM — document only)
- **NO** redesign of select/dropdown components
- **NO** refactoring of hooks beyond Logs.tsx duplicate logic
- **NO** changes to scripts' text logging format
- **NO** touching any SQL patterns outside the 4 identified files
- **NO** new package dependencies (unless rate limiting requires it, and even then, avoid)

---

## Verification Commands

### Before Starting (Baseline)
```bash
# Backend tests
pytest -v --tb=short

# Frontend tests
cd frontend && npm test

# Count f-string SQL patterns (should be 7 in the 4 target files)
grep -rn 'f"' services/task_service.py services/alert_service.py services/db_service.py api/main.py | grep -E '(SELECT|UPDATE|DELETE|INSERT)'
```

### After Each Wave
```bash
# Wave 0: Direct verification
grep -n 'tags=\["Logs"\]' api/main.py  # should return 1
find frontend/src -type d -empty       # should be 0 (after T12 completes hooks/)
python -c "from api.main import app"   # should NOT show migration output (after T11)

# Wave 1: Run all tests
pytest -v --tb=short tests/test_sql_param.py
pytest -v --tb=short tests/test_feishu_service.py
pytest -v --tb=short tests/test_dispatch_transaction.py
pytest -v --tb=short tests/test_rate_limit.py
pytest -v --tb=short tests/test_dispatch_service.py  # existing tests must still pass

# Wave 2: Run all tests
cd frontend && npm test  # all existing tests must pass

# Wave 3: Run all tests
cd frontend && npm test  # Logs.test.tsx must pass unchanged
```

### Final Verification
```bash
# No f-string SQL in target files (T02)
grep -rn 'execute(f"' services/task_service.py services/alert_service.py services/db_service.py api/main.py | wc -l

# Transaction rollback exists (T06)
grep -n 'rollback' api/routers/outbox.py

# Rate limit exists (T07)
grep -n 'rate_limit\|429' api/routers/outbox.py api/dependencies.py

# ErrorBoundary wraps all pages (T08)
grep -c 'ErrorBoundary' frontend/src/App.tsx  # should match number of page routes

# All 4 selects have aria-label (T09)
grep -c 'aria-label' frontend/src/pages/Alerts.tsx  # should be 2
grep -c 'aria-label' frontend/src/pages/Tasks.tsx   # should be 2

# Hooks directory has content (T12)
ls frontend/src/hooks/  # should show useLogQuery.ts
```

---

## Risk Register

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| SQL refactoring breaks query semantics | Medium | High | TDD: write tests first, verify query results match before/after |
| Rate limiting breaks existing clients | Medium | Medium | Rate limit only applies to dispatch POST, not GET; start with generous limits (5/60s) |
| Transaction rollback loses data on partial failure | Low | High | Use rollback only on uncaught exceptions; commit on success path |
| ErrorBoundary hides errors in development | Medium | Low | Show error details in development, generic message in production |
| Migration move breaks deployments | Low | High | Document that CI must run `python scripts/run_migration.py` before API restart |
| useLogQuery extraction changes rendering | Low | Medium | Existing Logs.test.tsx must pass unchanged |

---

## Commit Strategy (Atomic Commits)

Each task gets its own commit. No squashing until PR creation:

```
T01: fix(api): change diagnosis router tag from "Logs" to "Diagnosis"
T03: chore(frontend): remove empty src/lib directory
T04: refactor(services): derive Feishu env warnings from _REQUIRED_ENV_VARS
T05: test(services): add subprocess shell=False verification for feishu_service
T06: fix(services): add transaction rollback on dispatch failure in outbox router
T07: feat(api): add per-token rate limiting to dispatch endpoint
T02: fix(services): replace f-string SQL patterns with parameterized concatenation
T08: feat(frontend): add ErrorBoundary wrapper to all page routes
T09: fix(frontend): add aria-label to select elements in Alerts and Tasks
T10: refactor(services): move Feishu config to module-level singleton
T11: chore(api): move migration from startup to explicit script
T12: refactor(frontend): extract useLogQuery hook from Logs.tsx tabs
```

**Commit order**: Wave 0 → Wave 1 → Wave 2 → Wave 3
- Waves processed in parallel
- Commits within waves can be ordered by risk (highest risk first)
- If a wave's task fails verification, skip to next wave; fix later
