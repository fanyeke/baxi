# Dispatch Dry-Run Purity Fix — Findings

## Date: 2026-05-23

### Problem
`write_result()` in `services/dispatch_service.py` always ran the DB UPDATE, incrementing `dispatch_attempts` and setting `last_dispatch_at` even when `is_dry_run=True`. After too many dry-runs, real dispatches would fail because MAX_ATTEMPTS was exhausted.

### Root Cause
The function had a branch `if is_dry_run: db_status = "pending"` but the `conn.execute(UPDATE ...)` ran unconditionally after the if/elif/else block.

### Fix
Added early return `if is_dry_run: return` at the top of `write_result()`, before any DB mutation. Removed the dead `if is_dry_run: db_status = "pending"` branch since it's unreachable now.

### Tests Added
`tests/test_dispatch_service.py` — 4 tests using `unittest.mock.patch` on `resolve_adapter`:
- `test_dry_run_does_not_increment_attempts`: DB untouched, dispatch_attempts stays 0
- `test_dry_run_does_not_change_status`: status stays "pending"  
- `test_dry_run_does_not_set_last_dispatch_at`: last_dispatch_at stays NULL
- `test_real_dispatch_does_increment_attempts`: dispatch_attempts=1, status="dispatched"

### Key Pattern
Tests use `unittest.mock.patch("services.dispatch_service.resolve_adapter", ...)` to inject a dummy adapter without needing the real YAML registry or importlib-based module loading. This keeps tests fast and isolated.

### Verification
- All 4 new tests pass
- Full suite: 272 tests pass, zero regressions
- LSP diagnostics: clean

## 2026-05-28: ontology_repository.go migration

- Migrated `internal/repository/ontology_repository.go` to `internal/repository/ontology/` subpackage
- Pattern: new subpackage defines its own duplicate types (to avoid circular import with `repository` parent). The compat layer converts between ontology.ObjectInstance and repository.ObjectInstance via helper functions (toObjectInstanceSlice, toObjectQueryResult, etc.)
- The subpackage methods use `r.Query()` / `r.QueryRow()` from embedded `*common.PoolProvider` instead of `pool.Query()` / `pool.QueryRow()`
- `WithRole` re-exported from the compat layer (backward compat for `internal/ontology/query_service.go`)
- Follows same pattern as governance_repository.go and decision_repository.go compat layers
- `ontology_aware_adapter.go` and `ontology_aware_repo.go` unchanged - they only define interfaces using types from `interfaces.go` still in the `repository` package
