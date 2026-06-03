---
phase: 02-error-handling-observability
plan: 01
subsystem: api
tags: [error-handling, middleware, dto, validation, database, observability]

requires: []
provides:
  - Structured error handling: CONFLICT, SERVICE_UNAVAILABLE error codes
  - Field-level validation error types: FieldError, ValidationError
  - DB connection error detection and 503 mapping
  - writeValidationError and writeDatabaseError helper functions
affects: [02-error-handling-observability, all-api-phases]

tech-stack:
  added: []
  patterns:
    - "FieldError + ValidationError DTO pattern for field-level validation responses"
    - "isDatabaseConnectionError for standardized DB failure detection"
    - "Retry-After header on 503 responses for transient DB failures"

key-files:
  created:
    - internal/api/dto/error.go
  modified:
    - internal/api/middleware/error.go
    - internal/api/middleware/error_test.go
    - internal/api/handler/error_helper.go
    - internal/api/handler/error_helper_test.go

key-decisions:
  - "CONFLICT (409) and SERVICE_UNAVAILABLE (503) error codes added as middleware constants"
  - "Details field on APIError uses omitempty — nil details omitted from JSON"
  - "WriteErrorWithDetails reuses same JSON encoding pattern as WriteError with extra details param"
  - "isDatabaseConnectionError checks known error substrings (connection refused, pool closed, etc.)"
  - "writeDatabaseError sets Retry-After: 5 header for connection errors, falls back to 500 for others"
  - "CONFLICT and SERVICE_UNAVAILABLE added to defaultDiagnosis and defaultAction switch"

patterns-established:
  - "FieldError/ValidationError: standard DTO types for field-level validation, used by writeValidationError"
  - "Database error classification: isDatabaseConnectionError → classifyError → writeDatabaseError"
  - "Extensible error code system: add codes to middleware error.go constants, then add diagnosis/action cases"

requirements-completed: [ERR-01, ERR-02, ERR-04]

duration: 12 min
completed: 2026-06-03
---

# Phase 2 Plan 1: Error Handling & Observability Summary

**Structured error handling infrastructure: CONFLICT/SERVICE_UNAVAILABLE error codes, APIError details field, FieldError/ValidationError DTO types, DB connection error detection, and helper functions**

## Performance

- **Duration:** 12 min
- **Started:** 2026-06-03T12:00:00Z (approx)
- **Completed:** 2026-06-03T12:12:00Z (approx)
- **Tasks:** 3
- **Files modified/created:** 5

## Accomplishments
- Added CONFLICT and SERVICE_UNAVAILABLE error code constants in middleware/error.go
- Added optional Details field to APIError struct with JSON omitempty tag
- Implemented WriteErrorWithDetails for structured error responses with extra details
- Created dto/error.go with FieldError and ValidationError types for field-level validation errors
- Implemented isDatabaseConnectionError detecting "connection refused", "pool closed", "no such host", etc.
- Enhanced classifyError to return 503 SERVICE_UNAVAILABLE for DB connection errors
- Added writeValidationError helper for 400 responses with field-level validation details
- Added writeDatabaseError helper with Retry-After: 5 header for DB connection errors
- Added CONFLICT and SERVICE_UNAVAILABLE cases to defaultDiagnosis and defaultAction
- All new functions covered by unit tests

## Task Commits

Each task was committed atomically:

1. **Task 1: Extend middleware/error.go** — `a84869e` (feat)
   - CONFLICT/SERVICE_UNAVAILABLE constants, Details field, WriteErrorWithDetails
2. **Task 2: Add dto/error.go** — `0e98368` (feat)
   - FieldError and ValidationError types
3. **Task 3: Enhance handler/error_helper.go** — `a06332b` (feat)
   - DB connection detection, writeValidationError, writeDatabaseError

## Files Created/Modified
- `internal/api/middleware/error.go` — Added CONFLICT, SERVICE_UNAVAILABLE constants; Details field on APIError; WriteErrorWithDetails function
- `internal/api/middleware/error_test.go` — Tests for new constants, WriteErrorWithDetails, Details omission
- `internal/api/dto/error.go` — New file: FieldError and ValidationError types
- `internal/api/handler/error_helper.go` — Added isDatabaseConnectionError, writeValidationError, writeDatabaseError; enhanced classifyError; updated defaultDiagnosis/defaultAction
- `internal/api/handler/error_helper_test.go` — Tests for DB error detection, writeValidationError, new diagnosis/action codes

## Decisions Made
- **Details as interface{}:** Kept flexible — can carry validation errors, retry info, or any structured data
- **omitempty on Details:** nil details omitted from JSON to avoid bloating responses when not used
- **String matching for DB errors:** Works without importing database driver types, covers both pgx and standard lib errors
- **No new dependencies:** All changes use only standard library and existing pgx imports

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- Two pre-existing test failures (TestSandboxHandler_AddProposal_SandboxNotFound, TestHandleGetDetail_NotFound) — both exist independently of this plan's changes, expected 404 but getting 500 due to error wrapping patterns

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Core error handling infrastructure complete
- Ready for Phase 2 Plan 2: Observer pattern and logging improvements
- All downstream API changes can use the new error codes, details field, and helper functions

## Self-Check: PASSED

- All 5 files exist on disk
- All 4 commits present in git history (a84869e, 0e98368, a06332b, a5c6d60)
- `go build ./internal/api/...` passes
- All new tests pass: middleware (4), dto (all existing), handler (9 new)
- Pre-existing failures (2 unrelated tests) unchanged — not caused by this plan
- No unexpected file deletions
- No task-generated untracked files
