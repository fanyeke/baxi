---
phase: 02-error-handling-observability
plan: 03
subsystem: api
tags: [handler, validation, error-handling, db-503, field-level-errors, writeServiceError]
requires:
  - phase: 02-02
    provides: error_helper.go with writeServiceError, writeValidationError, classifyError
provides:
  - Field-level validation errors in all 6 decision/action/outbox/pipeline/review/sandbox handlers
  - DB connection failure 503 detection in all handler default error branches
affects: []

tech-stack:
  added: []
  patterns:
    - All handler validation failures use writeValidationError producing {fields: [{field, message, code}]}
    - All handler default error branches use writeServiceError for automatic DB 503 classification

key-files:
  created: []
  modified:
    - internal/api/handler/decision.go
    - internal/api/handler/action.go
    - internal/api/handler/outbox.go
    - internal/api/handler/pipeline.go
    - internal/api/handler/review.go
    - internal/api/handler/handler_sandbox.go
    - internal/api/handler/error_helper_test.go
    - internal/api/handler/action_test.go
    - internal/api/handler/decision_test.go
    - internal/api/handler/review_test.go
    - internal/api/handler/handler_sandbox_test.go

key-decisions:
  - "HandleDispatch switch missing default case: added default writeServiceError branch"
  - "HandleGetDetail lacked isNotFound check: added before writeServiceError to return 404"

requirements-completed: [ERR-01, ERR-04, ERR-05]

duration: 7min
completed: 2026-06-03
---

# Phase 2: Error Handling & Observability — Plan 3 Summary

**Field-level validation errors in all 6 handlers + DB 503 connection failure detection in default error branches**

## Performance

- **Duration:** 7 min
- **Started:** 2026-06-03T12:20:06Z
- **Completed:** 2026-06-03T12:27:21Z
- **Tasks:** 3
- **Files modified:** 11

## Accomplishments

- decision.go CreateCase: split into separate field-level errors for source_type/source_id
- action.go HandleExecute/HandleStatus: empty proposalID returns writeValidationError
- outbox.go HandleDispatch/HandleCancel/HandleGetDetail: empty eventID returns writeValidationError
- pipeline.go HandleRun: empty config returns writeValidationError
- review.go handleReviewAction: empty reviewer_id returns writeValidationError
- handler_sandbox.go HandleCreate/HandleAddProposal/HandleCompare: empty fields return writeValidationError
- ALL handler default error branches replaced writeError(500) with writeServiceError for automatic DB 503 classification
- HandleDispatch switch gained default case (was missing — bug fix per deviation rules)
- HandleGetDetail gained isNotFound check before writeServiceError call
- Added 4 new test functions covering writeDatabaseError, wrapped DB errors, and writeServiceError DB detection
- Fixed 5 test assertions and 2 sandbox test mocks to match new error response format

## Task Commits

Each task was committed atomically:

1. **Task 1: Upgrade decision.go + action.go validation to field-level errors** - `f426d86` (feat)
2. **Task 2: Upgrade remaining handler validations to field-level errors** - `e56a392` (feat)
3. **Task 3: Add DB 503 detection to handler default error branches + tests** - `668a136` (feat)

## Files Created/Modified

- `internal/api/handler/decision.go` - Field-level CreateCase validation; 12 default error branches use writeServiceError
- `internal/api/handler/action.go` - HandleExecute/HandleStatus use writeValidationError and writeServiceError
- `internal/api/handler/outbox.go` - HandleDispatch/HandleCancel/HandleGetDetail use writeValidationError; all default branches use writeServiceError; Dispatch switch default case added; GetDetail isNotFound check added
- `internal/api/handler/pipeline.go` - HandleRun uses writeValidationError
- `internal/api/handler/review.go` - handleReviewAction uses writeValidationError; default branches use writeServiceError
- `internal/api/handler/handler_sandbox.go` - HandleCreate/HandleAddProposal/HandleCompare use writeValidationError; all default branches use writeServiceError
- `internal/api/handler/error_helper_test.go` - 4 new DB 503 and wrapped error tests
- `internal/api/handler/action_test.go` - Updated assertion for writeValidationError message
- `internal/api/handler/decision_test.go` - Updated assertion for writeValidationError message
- `internal/api/handler/review_test.go` - Updated assertion for writeValidationError message
- `internal/api/handler/handler_sandbox_test.go` - Fixed sandbox not-found mocks to return proper sentinel errors

## Decisions Made

- HandleDispatch switch had no default case (bug) — added writeServiceError default
- HandleGetDetail lacked isNotFound check for GetEvent errors — added before writeServiceError to return proper 404

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] HandleDispatch switch missing default case**
- **Found during:** Task 3 (test run)
- **Issue:** HandleDispatch's error switch had no default case — generic errors silently returned 200
- **Fix:** Added `default: writeServiceError(w, r, err, "internal server error")` to the switch
- **Files modified:** outbox.go
- **Verification:** TestHandleDispatch_GenericError now passes (expects 500)
- **Committed in:** 668a136 (Task 3 commit)

**2. [Rule 2 - Missing Critical] HandleGetDetail lacked isNotFound detection on GetEvent error**
- **Found during:** Task 3 (test run)
- **Issue:** HandleGetDetail called writeServiceError for all GetEvent errors, including not-found — should return 404 for ErrEventNotFound
- **Fix:** Added `if isNotFound(err)` check before writeServiceError call
- **Files modified:** outbox.go
- **Verification:** TestHandleGetDetail_NotFound now passes (expects 404)
- **Committed in:** 668a136 (Task 3 commit)

---

**Total deviations:** 2 auto-fixed (2 Rule 2 - Missing Critical)
**Impact on plan:** Both fixes were necessary for correctness of the outbox handler's error handling. No scope creep.

## Issues Encountered

- 5 test assertions across action_test.go, decision_test.go, review_test.go needed updating from old error message "required"/"is required" to new "validation failed" message format (field details now in details.fields object)
- 2 sandbox test mocks returned `errors.New("sandbox nonexistent not found")` instead of `review.ErrSandboxNotFound` sentinel — causing 500 instead of expected 404

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- All 6 target handlers upgraded with field-level validation errors (ERR-05)
- All handler default branches use writeServiceError for DB 503 classification (ERR-04)
- Error helper tests extended with wrapped DB error and writeServiceError coverage (ERR-01)
- Ready for next plan in the error handling and observability phase

---

*Phase: 02-error-handling-observability*
*Completed: 2026-06-03*

## Self-Check: PASSED

- ✅ All 3 tasks executed and committed
- ✅ SUMMARY.md created at `.planning/phases/02-error-handling-observability/02-03-SUMMARY.md`
- ✅ No modifications to shared orchestrator artifacts (STATE.md, ROADMAP.md)
- ✅ Build passes: `go build ./internal/api/...`
- ✅ Vet passes: `go vet ./internal/api/handler/`
- ✅ All handler tests pass: `go test ./internal/api/handler/`
- ✅ 4 commits: 3 feat + 1 docs
