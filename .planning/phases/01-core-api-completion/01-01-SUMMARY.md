---
phase: 01-core-api-completion
plan: "01"
subsystem: api
tags: [go, chi, handler, dto, decision]

requires:
  - phase: init
    provides: project structure, base handler patterns

provides:
  - Working DecideLLM handler (POST /api/v1/decisions/cases/{case_id}/decide/llm)
  - Working ListLLMDecisions handler (GET /api/v1/decisions/cases/{case_id}/llm-decisions)
  - Working ListEvals handler (GET /api/v1/decisions/cases/{case_id}/evals)
  - DTO types for LLM decision and eval list responses

affects:
  - frontend decision pages
  - API client expectations

tech-stack:
  added: []
  patterns:
    - "Handler proxy pattern: handlers delegate to service layer via narrow interfaces"
    - "Anonymous struct with json tags for direct DB-to-JSON serialization"

key-files:
  created: []
  modified:
    - internal/api/handler/decision.go
    - internal/api/dto/decision.go
    - internal/api/handler/decision_test.go
    - internal/api/handler/decision_extra_test.go

key-decisions:
  - "Service returns anonymous structs with json tags — handlers pass through directly instead of mapping to DTOs for list endpoints"
  - "DecideLLM reuses existing DecisionResponse DTO since the output shape is identical to Decide"

patterns-established:
  - "Handler list endpoints return service result directly when json tags are already correct"

requirements-completed:
  - API-01
  - API-04
  - API-05

duration: 4min
completed: "2026-06-03"
---

# Phase 01 Plan 01: Core API Completion — Decision Endpoints Summary

**Replace three 501 stubs with working handlers (DecideLLM, ListLLMDecisions, ListEvals) using existing service layer**

## Performance

- **Duration:** 4 min
- **Started:** 2026-06-03T10:41:43Z
- **Completed:** 2026-06-03T10:46:09Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments

- Extended `handler.DecisionService` interface with `DecideLLM`, `ListLLMDecisions`, `ListEvals` method signatures
- Added `LLMDecisionItem`, `EvalItem`, `LLMDecisionListResponse`, `EvalListResponse` DTO types
- Implemented `DecideLLM` handler that proxies to `svc.DecideLLM` and returns `DecisionResponse`
- Implemented `ListLLMDecisions` handler that returns service result directly as JSON array
- Implemented `ListEvals` handler that returns service result directly as JSON array
- Updated mock service in tests to implement new interface methods

## Task Commits

Each task was committed atomically:

1. **Task 1: Extend DecisionService interface and add DTO types** — `523bae8` (feat)
2. **Task 2: Implement DecideLLM, ListLLMDecisions, and ListEvals handlers** — `cc68107` (feat)

**Plan metadata:** `TBD` (docs: complete plan)

## Files Created/Modified

- `internal/api/handler/decision.go` — Extended DecisionService interface; implemented DecideLLM, ListLLMDecisions, ListEvals handlers
- `internal/api/dto/decision.go` — Added LLMDecisionItem, EvalItem, LLMDecisionListResponse, EvalListResponse types
- `internal/api/handler/decision_test.go` — Added mock methods for new interface methods
- `internal/api/handler/decision_extra_test.go` — Updated TestListEvals_InternalError to use mock function field

## Decisions Made

- **Service result passthrough for list endpoints:** The service's `ListLLMDecisions` and `ListEvals` return anonymous structs with correct `json` tags. Rather than adding mapping boilerplate, handlers pass the result directly to `httputil.JSON`. This keeps the handler layer thin and avoids duplicative struct definitions.
- **Reuse DecisionResponse for DecideLLM:** Since `DecideLLM` produces the same output shape as `Decide` (decision context + output + proposals), we reuse the existing `dto.DecisionResponse` type instead of creating a separate LLM-specific response DTO.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added mock methods for new interface methods in tests**
- **Found during:** Task 2
- **Issue:** After extending `DecisionService` interface, `go vet` failed because `mockDecisionService` in `decision_test.go` did not implement the three new methods (`DecideLLM`, `ListLLMDecisions`, `ListEvals`)
- **Fix:** Added function fields (`decideLLMFn`, `listLLMDecisionsFn`, `listEvalsFn`) to `mockDecisionService` struct and corresponding method implementations
- **Files modified:** `internal/api/handler/decision_test.go`, `internal/api/handler/decision_extra_test.go`
- **Verification:** `go vet ./internal/api/handler/` passes
- **Committed in:** `cc68107` (Task 2 commit)

**2. [Rule 3 - Blocking] Updated TestListEvals_InternalError to use explicit error mock**
- **Found during:** Task 2
- **Issue:** After adding default no-op mock methods, `TestListEvals_InternalError` began returning 200 instead of 500 because the mock no longer returned an error by default
- **Fix:** Updated the test to set `listEvalsFn` to return an explicit error, matching the test's intent
- **Files modified:** `internal/api/handler/decision_extra_test.go`
- **Verification:** `go test ./internal/api/handler/... -run TestListEvals_InternalError` passes
- **Committed in:** `cc68107` (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (2 blocking)
**Impact on plan:** Both auto-fixes necessary for compilation and test correctness. No scope creep.

## Issues Encountered

- Pre-existing test failures in `TestSandboxHandler_AddProposal_SandboxNotFound` and `TestHandleGetDetail_NotFound` — unrelated to this plan's changes

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Three 501 stubs now return 200 with valid data
- Frontend can integrate with these endpoints immediately
- Remaining 501 stubs (Compare, Replay, BatchDispatch) ready for next plans

---

*Phase: 01-core-api-completion*
*Completed: 2026-06-03*
