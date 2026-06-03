---
phase: 01-core-api-completion
plan: 03
subsystem: api
tags: [outbox, batch-dispatch, chi, handler]

requires:
  - phase: 01-core-api-completion
    provides: Outbox service interface with List, GetEvent, DispatchEvent, CancelEvent
provides:
  - POST /api/v1/outbox/dispatch endpoint returning BatchDispatchResponse
  - BatchDispatchRequest DTO with DryRun field
  - Batch dispatch handler processing up to 1000 pending events
affects:
  - 01-core-api-completion

tech-stack:
  added: []
  patterns:
    - "Handler defines narrow service interface for testability"
    - "DTOs separate from model types"
    - "Fire-and-forget batch operations with partial failure handling"

key-files:
  created: []
  modified:
    - internal/api/handler/outbox.go
    - internal/api/dto/outbox.go

key-decisions:
  - "Used existing OutboxService.List + DispatchEvent rather than adding a new service method — keeps handler logic transparent and testable"
  - "Empty/invalid request body defaults to dry_run=false instead of returning 400 — aligns with fire-and-forget batch operation semantics"
  - "Event IDs collected for ALL events including failures — caller can see which events were attempted"

requirements-completed:
  - API-06

duration: 5min
completed: 2026-06-03
---

# Phase 01 Plan 03: BatchDispatch Endpoint Summary

**Batch dispatch handler for outbox events with dry-run support and partial-failure resilience**

## Performance

- **Duration:** 5 min
- **Started:** 2026-06-03T10:47:00Z
- **Completed:** 2026-06-03T10:52:00Z
- **Tasks:** 1
- **Files modified:** 2

## Accomplishments
- Replaced 501 Not Implemented stub with working batch dispatch handler
- Added BatchDispatchRequest DTO for request body parsing
- Implemented dry-run mode for safe testing of batch operations
- Added partial failure handling — individual dispatch errors don't abort the batch

## Task Commits

Each task was committed atomically:

1. **Task 1: Add BatchDispatch request DTO and implement handler** - `3f921de` (feat)

**Plan metadata:** `3f921de` (docs: complete plan)

## Files Created/Modified
- `internal/api/dto/outbox.go` - Added BatchDispatchRequest struct with DryRun field
- `internal/api/handler/outbox.go` - Replaced HandleBatchDispatch 501 stub with full implementation that queries pending events, dispatches each in a loop, and returns counts

## Decisions Made
- Used existing OutboxService.List + DispatchEvent rather than adding a new service method — keeps handler logic transparent and testable
- Empty/invalid request body defaults to dry_run=false instead of returning 400 — aligns with fire-and-forget batch operation semantics
- Event IDs collected for ALL events including failures — caller can see which events were attempted

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- Pre-existing test failure in `TestHandleGetDetail_NotFound` (unrelated to this change — expects 404 but gets 500). All outbox-related tests pass.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Ready for next plan in Phase 01
- BatchDispatch endpoint is functional and can be called via POST /api/v1/outbox/dispatch

## Self-Check: PASSED

- [x] internal/api/handler/outbox.go contains HandleBatchDispatch implementation
- [x] internal/api/dto/outbox.go contains BatchDispatchRequest with DryRun field
- [x] Handler constructs OutboxFilters with Status="pending"
- [x] Handler calls h.svc.List and iterates over resp.Items
- [x] Handler returns BatchDispatchResponse with Dispatched, Failed, EventIDs
- [x] go build ./internal/api/handler/ exits 0
- [x] go vet ./internal/api/handler/ exits 0
- [x] Route exists in internal/api/routes.go: r.Post("/outbox/dispatch", ...)

---
*Phase: 01-core-api-completion*
*Completed: 2026-06-03*
