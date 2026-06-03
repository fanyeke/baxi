---
phase: 02-error-handling-observability
plan: 02
subsystem: api
tags: [go, chi, error-handling, pgx, sentinel-errors]

# Dependency graph
requires:
  - phase: 02-error-handling-observability
    provides: middleware constants (CONFLICT, SERVICE_UNAVAILABLE)
provides:
  - JSON decode errors now return HTTP 400 instead of silent ignore
  - 409 Conflict responses use CONFLICT error code instead of BAD_REQUEST
  - 503 Service Unavailable response uses SERVICE_UNAVAILABLE error code
  - handler_sandbox.go uses typed sentinel error (review.ErrSandboxNotFound) via errors.Is()
  - action.go HandleStatus detects pgx.ErrNoRows and returns 404
  - review/errors.go sentinel error module
affects: [api, error-handling]

# Tech tracking
tech-stack:
  added: []
  patterns: ["typed sentinel errors with errors.Is()", "pgx.ErrNoRows detection in handlers"]

key-files:
  created: [internal/review/errors.go]
  modified: [internal/api/handler/action.go, internal/api/handler/outbox.go, internal/api/handler/review.go, internal/api/handler/decision.go, internal/api/handler/handler_sandbox.go, internal/review/sandbox.go]

key-decisions:
  - "Created review/errors.go as canonical location for review package sentinel errors"
  - "All sandbox not-found errors wrap ErrSandboxNotFound via fmt.Errorf('...: %w', ErrSandboxNotFound)"

requirements-completed: [ERR-01, ERR-03]

# Metrics
duration: 8 min
completed: 2026-06-03
---

# Phase 02 Plan 02: Error Codes & JSON Decode Fix Summary

**Fixed 5 handler files: 2 silent JSON decode bugs, 4 error code misuses, and 1 string-matching anti-pattern replaced with typed sentinel errors**

## Performance

- **Duration:** 8 min
- **Started:** 2026-06-03T12:10:39Z
- **Completed:** 2026-06-03T12:18:53Z
- **Tasks:** 3
- **Files modified:** 7

## Accomplishments
- action.go HandleExecute: JSON decode failure now returns 400 instead of silently ignoring (ERR-03)
- outbox.go HandleBatchDispatch: JSON decode failure now returns 400 instead of silently defaulting to dry_run=false (ERR-03)
- outbox.go HandleDispatch: 409 conflict uses correct CONFLICT error code (ERR-01)
- outbox.go HandleCancel: 409 conflict uses correct CONFLICT error code (ERR-01)
- review.go handleReviewAction: 409 conflict uses correct CONFLICT error code (ERR-01)
- decision.go Replay: 503 uses correct SERVICE_UNAVAILABLE error code (ERR-01)
- handler_sandbox.go: replaced brittle string matching with `errors.Is(err, review.ErrSandboxNotFound)` (ERR-01)
- action.go HandleStatus: detects `pgx.ErrNoRows` from service layer and returns 404
- Created `internal/review/errors.go` as canonical location for review package sentinel errors

## Task Commits

Each task was committed atomically:

1. **Task 1: Fix silent JSON decode errors** - `aa363e7` (fix)
2. **Task 2: Fix error code misuse** - `0c4aad7` (fix)
3. **Task 3: Replace string matching with sentinel errors** - `c3b712d` (fix)

**Plan metadata:** (committed below in final metadata commit)

## Files Created/Modified
- `internal/review/errors.go` - New sentinel errors file with `ErrSandboxNotFound`
- `internal/review/sandbox.go` - Updated to wrap `ErrSandboxNotFound` via `fmt.Errorf("%w")`
- `internal/api/handler/action.go` - Fixed HandleExecute JSON decode; added HandleStatus pgx.ErrNoRows detection
- `internal/api/handler/outbox.go` - Fixed HandleBatchDispatch JSON decode; Fixed HandleDispatch/HandleCancel error codes
- `internal/api/handler/review.go` - Fixed handleReviewAction error code
- `internal/api/handler/decision.go` - Fixed Replay error code
- `internal/api/handler/handler_sandbox.go` - Replaced string matching with typed sentinel errors

## Decisions Made
- Created `internal/review/errors.go` as the canonical location for review package sentinel errors (following the existing pattern from packages like `internal/action/` and `internal/decision/`)
- All sandbox not-found errors now wrap `review.ErrSandboxNotFound` via `fmt.Errorf("...: %w", ErrSandboxNotFound)` so `errors.Is()` works through error wrapping chains
- Added `pgx/v5` import to action.go for `pgx.ErrNoRows` detection in HandleStatus

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None - all acceptance criteria passed on first attempt.

## Next Phase Readiness
- ERR-01 and ERR-03 requirements completed
- Ready for next plan in Phase 02 (Error Handling & Observability)
