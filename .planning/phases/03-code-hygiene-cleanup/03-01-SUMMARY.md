---
phase: 03-code-hygiene-cleanup
plan: 01
subsystem: misc
tags: pipeline, makefile, cli, cleanup, dead-code
requires:
  - phase: 00-initialization
    provides: project initialization and decision records
provides:
  - Pipeline preview with Go command instead of Python scripts
  - Makefile without Python script references
  - Removal of dead CLI subcommand (llm.go)
  - Removal of deprecated placeholder worker.go
affects: 03-code-hygiene-cleanup (Plans 02-03)
tech-stack:
  added: []
  patterns:
    - "Pipeline commands: use 'go run ./cmd/baxi-cli pipeline run' instead of python3 scripts"
key-files:
  created: []
  modified:
    - internal/service/pipeline_service.go
    - cmd/baxi-worker/main.go
    - Makefile
    - cmd/baxi-cli/main.go
  deleted:
    - internal/worker/worker.go
    - cmd/baxi-cli/llm.go
key-decisions:
  - "Deleted internal/worker/worker.go instead of keeping it — real logic is in dispatch_worker.go"
  - "Updated cmd/baxi-worker/main.go to remove reference to deleted worker.New (deviation from plan — necessary for clean build)"
  - "Removed api-compare, llm-status, llm-metrics Makefile targets referencing Python scripts or deleted llm.go"
patterns-established:
  - "Pipeline preview: always returns 'go run ./cmd/baxi-cli pipeline run' regardless of pipeline type"
requirements-completed:
  - HYG-01
  - HYG-02
  - HYG-05
  - HYG-06
duration: 1min
completed: 2026-06-03
---

# Phase 03: Code Hygiene & Cleanup — Plan 01 Summary

**Pipeline preview uses Go commands, Makefile has no Python references, dead CLI subcommand and placeholder worker removed**

## Performance

- **Duration:** 1 min
- **Started:** 2026-06-03T12:57:27Z
- **Completed:** 2026-06-03T12:59:26Z
- **Tasks:** 2
- **Files modified:** 4 (2 deleted)

## Accomplishments

- **Pipeline preview (HYG-01):** `PreviewPipelineRun` now returns `go run ./cmd/baxi-cli pipeline run` instead of Python script paths
- **Makefile cleanup (HYG-02):** Removed `api-compare` target (called `python3`), removed `llm-status`/`llm-metrics` targets, updated `.PHONY` list
- **Dead CLI subcommand (HYG-05):** Deleted unreachable `cmd/baxi-cli/llm.go`; removed `"llm"` case from `main.go` switch, dispatch, and help text
- **Placeholder worker (HYG-06):** Deleted deprecated `internal/worker/worker.go`; removed `worker.New` reference from `cmd/baxi-worker/main.go`

## Task Commits

Each task was committed atomically:

1. **Task 1: Fix Pipeline Preview + Delete worker.go** — `11a9f2d` (fix)
2. **Task 2: Clean Makefile Python refs + Delete llm.go** — `53446a8` (fix)

## Files Created/Modified

- `internal/service/pipeline_service.go` — PreviewPipelineRun now returns Go command instead of Python
- `cmd/baxi-worker/main.go` — Removed reference to deleted `worker.New`/`Worker`
- `Makefile` — Removed `api-compare`, `llm-status`, `llm-metrics` targets and `python3` references
- `cmd/baxi-cli/main.go` — Removed `"llm"` case, dispatch, and help text
- `internal/worker/worker.go` — **Deleted** (deprecated placeholder, real logic in dispatch_worker.go)
- `cmd/baxi-cli/llm.go` — **Deleted** (dead CLI subcommand, unreachable from main.go)

## Decisions Made

- Deleted `worker.go` despite `cmd/baxi-worker/main.go` referencing `worker.New` — updated main.go in same commit to keep build clean (deviation Rule 3)
- Kept `client.go` untouched (still used by decision.go) per D-07
- Did not modify `decision.go`, `pipeline.go`, `governance.go` per D-08

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Updated cmd/baxi-worker/main.go when deleting worker.go**
- **Found during:** Task 1 (Delete worker.go)
- **Issue:** `cmd/baxi-worker/main.go` imported `"baxi/internal/worker"` and called `worker.New(zapLog, pool.Pool)` — deleting `worker.go` would break the build
- **Fix:** Removed the deprecated worker goroutine (lines 56-61) from main.go. The dispatch worker (`worker.NewDispatchWorker`) runs independently and handles all actual work
- **Files modified:** `cmd/baxi-worker/main.go`
- **Verification:** `go build ./...` passes cleanly
- **Committed in:** `11a9f2d` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Required for clean build. The removed goroutine was a no-op placeholder that only pinged DB and blocked until context cancellation — dispatch worker already handles real work.

## Issues Encountered

None — all changes applied cleanly, build passes.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Ready for Plan 02 (repository shim migration) and Plan 03 (migration baselines cleanup)
- Pipeline preview, Makefile, and CLI are clean of Python/dead-code references

---

*Phase: 03-code-hygiene-cleanup*
*Completed: 2026-06-03*
