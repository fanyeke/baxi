---
phase: 06-integration-end-to-end-demo
plan: 01
subsystem: testing
tags: [go, test-compilation, repository-subpackage-migration, pool-removal]

requires:
  - phase: 03-code-hygiene-cleanup
    provides: repository subpackage reorganization, PoolProvider pattern
provides:
  - Go test files in 4 packages compile without errors via go vet
  - Mock signatures match production interfaces (no pool-as-param)
  - Import paths use subpackage aliases (decisionRepo, taskRepo, etc.)
affects: [06-integration-end-to-end-demo]

tech-stack:
  added: []
  patterns: [repository subpackage imports, common.NewPoolProvider for test repos]

key-files:
  created: []
  modified:
    - internal/action/proposal_service_test.go
    - internal/action/deep_coverage_test.go
    - internal/api/handler/outbox_test.go
    - internal/decision/context_builder_test.go
    - internal/decision/context_builder_v2_test.go
    - internal/decision/engine_test.go
    - internal/decision/engine_helpers_test.go
    - internal/decision/lineage_service_test.go
    - internal/decision/lineage_adapter_test.go
    - internal/decision/case_service_test.go
    - internal/decision/decision_coverage_test.go
    - internal/service/alert_service_test.go
    - internal/service/log_service_test.go
    - internal/service/outbox_service_test.go
    - internal/service/status_service_test.go
    - internal/service/task_service_test.go
    - internal/service/service_extra_test.go
    - internal/service/service_extra3_test.go

key-decisions:
  - "All pool *pgxpool.Pool param removed from mock struct definitions across 4+ packages to match production interfaces"
  - "Flat repository package imports replaced with subpackage aliases (decisionRepo, taskRepo, alertRepo, etc.)"
  - "repository.DecisionCaseRow and repository.LLMDecisionRow changed to subpackage types"
  - "common.NewPoolProvider(pool) used in place of deprecated repo.SetPool(pool) pattern"
  - "pgxpool imports removed where no longer needed (constructor NewPoolProvider infers pool type)"
  - "NewContextBuilderV2 calls updated from 5 to 6 params (added pool parameter)"
  - "NewDecisionEngine calls updated from 4 to 3 params (removed intermediate nil)"
  - "NewProposalService calls updated from 4 to 3 params (removed extra nil)"

requirements-completed: [INT-02, INT-03]

duration: 28min
completed: 2026-06-03
---

# Phase 6 Plan 1: 修复 Go 测试编译错误 Summary

**Removed stale pool params, outdated repository type references, and wrong import paths across 18 test files in 4 packages, enabling go vet ./internal/... to pass cleanly**

## Performance

- **Duration:** 28 min
- **Started:** 2026-06-03
- **Completed:** 2026-06-03
- **Tasks:** 4
- **Files modified:** 18

## Accomplishments

- Fixed `internal/action/` package: 2 test files — removed duplicate import, removed `pool *pgxpool.Pool` from all mock struct fields/methods/lambdas, replaced `repository.NewDecisionRepository()` with `decisionRepo.NewRepository(common.NewPoolProvider(pool))`, fixed `NewProposalService` calls (4→3 args)
- Fixed `internal/api/handler/` package: 1 test file — removed extra pool arg from `NewOutboxService` call, fixed `testOutboxAdapter` field type from deprecated `*repository.OutboxRepository` to `*outboxRepo.Repository`, removed pool arg from `GetDetail` calls
- Fixed `internal/decision/` package: 8 test files — replaced `*repository.DecisionCaseRow` with `*decisionRepo.DecisionCaseRow`, removed pool params from all mock definitions, updated `NewContextBuilderV2` calls (5→6 args), updated `NewDecisionEngine` calls (4→3 args), fixed `NewContextBuilder` calls, fixed `NewDecisionLineageAdapter` calls
- Fixed `internal/service/` package: 7 test files — replaced flat `repository` imports with subpackage aliases (`alertRepo`, `logRepo`, `outboxRepo`, `statusRepo`, `taskRepo`), used `common.NewPoolProvider(pool)` instead of `repo.SetPool(pool)`, removed extra pool args from constructor calls

## Task Commits

Each task was committed atomically:

1. **Task 1: Fix proposal_service_test.go** - `505dc35` (fix: remove pool params from action test mocks, fix import paths)
2. **Task 2: Fix outbox_test.go** - `64896d3` (fix: fix outbox_test.go adapter types and constructor call)
3. **Task 3: Fix decision package test files** - `9236b12` (fix: fix decision package test compilation errors)
4. **Task 4: Fix service package test files** - `cf799c3` (fix: fix service package test compilation errors)

**Plan metadata:** (committed below with SUMMARY)

## Files Created/Modified

- `internal/action/proposal_service_test.go` — Removed duplicate import, pool params, fixed lambda defs, fixed LLMDecision repo ref
- `internal/action/deep_coverage_test.go` — Fixed repo type refs, pooled params, and constructor calls
- `internal/api/handler/outbox_test.go` — Fixed adapter type, removed extra pool arg from service constructor
- `internal/decision/context_builder_test.go` — Subpackage types, removed pool, removed pgxpool import
- `internal/decision/context_builder_v2_test.go` — Added decisionRepo import, removed pool from lambdas, fixed NewContextBuilderV2 calls
- `internal/decision/engine_test.go` — Subpackage types, removed pool, fixed NewDecisionEngine calls
- `internal/decision/engine_helpers_test.go` — Fixed NewDecisionEngine calls
- `internal/decision/lineage_service_test.go` — Subpackage types, removed pool, fixed adapter method signatures
- `internal/decision/lineage_adapter_test.go` — Removed pool field assertions, fixed NewDecisionLineageAdapter calls, removed pgxpool import
- `internal/decision/case_service_test.go` — Removed unused pgxpool import
- `internal/decision/decision_coverage_test.go` — Subpackage types, removed pool, fixed constructor call
- `internal/service/alert_service_test.go` — Replaced import, removed extra pool arg
- `internal/service/log_service_test.go` — Replaced import, removed old pool var
- `internal/service/outbox_service_test.go` — Replaced import, removed extra pool arg
- `internal/service/status_service_test.go` — common.NewPoolProvider, fixed NewStatusService calls
- `internal/service/task_service_test.go` — common.NewPoolProvider, fixed NewTaskService calls
- `internal/service/service_extra_test.go` — Fixed NewLogService calls
- `internal/service/service_extra3_test.go` — Fixed imports, struct field assertions, subpackage types

## Decisions Made

- Fixed all test files rather than only the 4 files in the plan, since all files within a package must pass `go vet` for the verification to succeed
- Used `common.NewPoolProvider(pool)` pattern instead of `SetPool(pool)` for test repo construction, matching the production PoolProvider pattern
- Kept nil pool providers for unit tests that don't need DB access, passing `common.NewPoolProvider(pool)` only in integration tests

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] deep_coverage_test.go had same pool/repo type errors**
- **Found during:** Task 1 (proposal_service_test.go verification)
- **Issue:** Compilation errors in additional file `internal/action/deep_coverage_test.go` blocked `go vet ./internal/action/...`
- **Fix:** Applied same pattern: subpackage types, removed pool params, fixed constructor calls
- **Files modified:** internal/action/deep_coverage_test.go
- **Verification:** go vet ./internal/action/... passes
- **Committed in:** 505dc35

**2. [Rule 1 - Bug] 7 additional decision test files had compile errors**
- **Found during:** Task 3 verification (go vet ./internal/decision/...)
- **Issue:** Files `engine_test.go`, `lineage_service_test.go`, `case_service_test.go`, `context_builder_v2_test.go`, `decision_coverage_test.go`, `engine_helpers_test.go`, `lineage_adapter_test.go` had the same pool/repo type issues
- **Fix:** Applied subpackage type replacements, removed pool params, fixed constructor signatures
- **Files modified:** 8 files in internal/decision/
- **Verification:** go vet ./internal/decision/... passes
- **Committed in:** 9236b12

**3. [Rule 1 - Bug] 6 additional service test files had compile errors**
- **Found during:** Task 4 verification (go vet ./internal/service/...)
- **Issue:** Files `log_service_test.go`, `outbox_service_test.go`, `status_service_test.go`, `task_service_test.go`, `service_extra_test.go`, `service_extra3_test.go` had flat import and pool issues
- **Fix:** Replaced flat repository imports with subpackage aliases, used common.NewPoolProvider, fixed constructor calls
- **Files modified:** 7 files in internal/service/
- **Verification:** go vet ./internal/service/... passes
- **Committed in:** cf799c3

**4. [Rule 3 - Blocking] pgxpool import became unused in several files**
- **Found during:** After removing pool params from mock definitions
- **Issue:** Several test files had unused `"github.com/jackc/pgx/v5/pgxpool"` imports which block compilation
- **Fix:** Removed unused imports where no direct *pgxpool.Pool reference remained
- **Verification:** go vet passes without unused import errors
- **Committed in:** Various commits

---

**Total deviations:** 4 auto-fixed (3 bug fixes, 1 blocking issue)
**Impact on plan:** All auto-fixes necessary for compilation. No scope creep — simply extended the same fix pattern to all files within the affected packages, as required by the `go vet ./package/...` verification command.

## Issues Encountered

- The plan only listed 4 target files, but the `go vet ./package/...` verification command requires ALL files in a package to compile. This caused cascading discovery of errors in 14 additional test files across 3 packages. All resolved with the same fix patterns (subpackage types, pool param removal).

## Self-Check

- [x] `go vet ./internal/action/...` → passes
- [x] `go vet ./internal/api/handler/...` → passes
- [x] `go vet ./internal/decision/...` → passes
- [x] `go vet ./internal/service/...` → passes
- [x] `go vet ./...` → passes
- [x] No production code modified (only `*_test.go` files)
- [x] Each task committed atomically (4 commits for 4 plan tasks)

## Next Phase Readiness

- All test compilation blockers for INT-02 and INT-03 resolved
- Ready for phase 06 follow-up plans (E2E integration tests, security tests)
- No infrastructure or configuration changes needed

---

*Phase: 06-integration-end-to-end-demo*
*Completed: 2026-06-03*
