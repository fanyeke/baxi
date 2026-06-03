---
phase: 01-core-api-completion
plan: 02
subsystem: api
tags: [go, chi, dto, handler, eval, comparison, replay]

requires:
  - phase: 01-01
    provides: DecisionService interface extension foundation

provides:
  - Working POST /api/v1/decisions/cases/{case_id}/compare endpoint
  - Working POST /api/v1/decisions/cases/{case_id}/replay endpoint
  - DecisionService interface with Compare and Replay methods
  - DTO types for CompareResponse, ReplayRequest, ReplayResponse, DiffItem, ReplayDiff, CompareMeta

affects:
  - 01-03 (batch-dispatch)
  - frontend decision pages
  - internal/eval package consumers

tech-stack:
  added: []
  patterns:
    - "Handler → Service → eval package delegation for comparison and replay"
    - "DTO transformation from domain eval types to API response types"
    - "Structured diff arrays (added/removed/changed) for decision comparison"

key-files:
  created: []
  modified:
    - internal/api/handler/decision.go
    - internal/api/dto/decision.go
    - internal/api/dto/sandbox.go
    - internal/api/handler/handler_sandbox.go
    - internal/api/handler/decision_test.go

key-decisions:
  - "Renamed existing DiffItem to SandboxDiffItem to avoid package-level type conflict with new decision comparison DiffItem"
  - "ReplayRequest parses model/temperature/context_overrides for future use while only passing dry_run to service layer"

patterns-established:
  - "DTO structs with snake_case JSON tags for all API response fields"
  - "Empty array initialization (not nil) for consistent API response shapes"
  - "503 Service Unavailable for optional service not configured scenarios"

requirements-completed:
  - API-02
  - API-03

duration: 12min
completed: 2026-06-03
---

# Phase 01 Plan 02: Compare and Replay Endpoint Implementation Summary

**Implemented working Compare and Replay HTTP handlers replacing 501 stubs, with structured DTOs and proper error handling**

## Performance

- **Duration:** 12 min
- **Started:** 2026-06-03T10:45:00Z
- **Completed:** 2026-06-03T10:57:00Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- Extended DecisionService interface with Compare and Replay method signatures
- Added 6 new DTO types (DiffItem, CompareMeta, CompareResponse, ReplayRequest, ReplayDiff, ReplayResponse) for structured API responses
- Implemented Compare handler returning structured diff with added/removed/changed arrays and metadata
- Implemented Replay handler supporting dry-run mode and optional parameter parsing
- Added 503 Service Unavailable handling for unconfigured replay service
- Updated mock service and resolved type naming conflict with existing sandbox DiffItem

## Task Commits

Each task was committed atomically:

1. **Task 1: Extend interface and add Compare/Replay DTOs** - `4d6a414` (feat)
2. **Task 2: Implement Compare and Replay handlers** - `5e5f947` (feat)

**Plan metadata:** `TBD` (docs: complete plan)

## Files Created/Modified
- `internal/api/handler/decision.go` - Extended DecisionService interface; implemented Compare and Replay handlers
- `internal/api/dto/decision.go` - Added DiffItem, CompareMeta, CompareResponse, ReplayRequest, ReplayDiff, ReplayResponse types
- `internal/api/dto/sandbox.go` - Renamed DiffItem to SandboxDiffItem to avoid package conflict
- `internal/api/handler/handler_sandbox.go` - Updated to use SandboxDiffItem
- `internal/api/handler/decision_test.go` - Added Compare and Replay methods to mockDecisionService

## Decisions Made
- Renamed existing sandbox DiffItem to SandboxDiffItem rather than merging field sets, preserving distinct semantics for sandbox comparison vs decision comparison
- Chose to parse but ignore model/temperature/context_overrides in ReplayRequest, documenting future use with a code comment per CONTEXT.md deferred items

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Resolved type naming conflict with existing DiffItem**
- **Found during:** Task 1
- **Issue:** dto/decision.go could not declare DiffItem because dto/sandbox.go already had a DiffItem with different fields (Value1/Value2 vs Before/After/ChangeType)
- **Fix:** Renamed existing DiffItem to SandboxDiffItem in sandbox.go, updated handler_sandbox.go usage, kept new DiffItem in decision.go per plan specification
- **Files modified:** internal/api/dto/sandbox.go, internal/api/handler/handler_sandbox.go
- **Verification:** go build ./internal/api/... exits with code 0
- **Committed in:** 4d6a414 (Task 1 commit)

**2. [Rule 3 - Blocking] Updated mockDecisionService to implement extended interface**
- **Found during:** Task 2
- **Issue:** go vet failed because mockDecisionService in decision_test.go lacked Compare and Replay methods after interface extension
- **Fix:** Added compareFn/replayFn fields and Compare/Replay method implementations to mockDecisionService
- **Files modified:** internal/api/handler/decision_test.go
- **Verification:** go vet ./internal/api/handler/ exits with code 0
- **Committed in:** 5e5f947 (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (2 blocking)
**Impact on plan:** Both auto-fixes necessary for compilation. No scope creep.

## Issues Encountered
- Pre-existing test failures in TestSandboxHandler_AddProposal_SandboxNotFound and TestHandleGetDetail_NotFound (unrelated to this plan; sandbox and outbox handlers)

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Compare and Replay endpoints ready for frontend integration
- Plan 01-03 (batch-dispatch) can proceed independently
- Remaining 501 stubs to implement: DecideLLM, ListLLMDecisions, ListEvals (planned in subsequent plans)

---
*Phase: 01-core-api-completion*
*Completed: 2026-06-03*
