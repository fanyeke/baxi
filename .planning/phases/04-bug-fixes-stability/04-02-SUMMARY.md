---
phase: 04-bug-fixes-stability
plan: 02
subsystem: alert-engine, ontology
tags: [go, zap, pgx, json-marshal, sql-injection, ontology]

requires:
  - phase: 01-core-api-completion
    provides: Go API and service layer patterns
  - phase: 02-error-handling-observability
    provides: error handling and logging patterns (zap)
provides:
  - Alert engine json.Marshal error handling with zap logging
  - Ontology V1 fallback SQL identifier sanitization via pgx.Identifier
affects: []

tech-stack:
  added: []
  patterns:
    - "Alert engine: engineLogger package-level variable for rule function logging"
    - "Alert engine: WithLogger() setter on Engine struct for optional logger injection"
    - "Ontology: fullTableName() uses pgx.Identifier.Sanitize() for safe SQL identifiers"
    - "Ontology: GODEPRECATED comments on V1 fallback paths marking migration targets"

key-files:
  created: []
  modified:
    - internal/alert/engine.go
    - internal/repository/ontology/repository.go

key-decisions:
  - "Alert engine marshal: use engineLogger package-level var (not Engine param) because global rule functions are package-level, not methods (per D-02, D-03)"
  - "Marshal fallback: use {} empty JSON object instead of empty string, alerts never skipped (per D-03)"
  - "Log evidence keys only, not full evidence content, avoiding sensitive data leaks (per D-02)"
  - "Ontology SQL sanitization: pgx.Identifier.Sanitize() matches V2 compiler pattern in compiler.go (per D-07, D-09)"
  - "Allowlist checks already existed in all 4 V1 fallback paths — GODEPRECATED comments added alongside (per D-08)"

requirements-completed: [BUG-02, BUG-05]

duration: 1 min
completed: 2026-06-03
---

# Phase 04: Bug Fixes & Stability — Plan 02 Summary

**Alert engine json.Marshal silent error ignores fixed with zap logging + Ontology V1 fallback SQL injection risk eliminated via pgx.Identifier.Sanitize()**

## Performance

- **Duration:** ~1 min (commit-to-commit; actual execution ~5 min including context loading)
- **Started:** 2026-06-03T14:15:51Z
- **Completed:** 2026-06-03T14:17:06Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- **BUG-02:** Alert engine `evaluateGMVDrop`, `evaluateLateDeliverySpike`, and `evaluateCancelRateSpike` no longer silently ignore `json.Marshal(evidence)` errors. On marshal failure: zap structured log with evidence key names (not full content), empty `{}` object fallback, alert is never skipped.
- **BUG-05:** Ontology V1 fallback `fullTableName()` uses `pgx.Identifier{m.Schema, m.Table}.Sanitize()` instead of raw `+` concatenation. 4 V1 fallback paths marked with `GODEPRECATED` comments at both the objectTableMap lookup and fullTableName() call sites.

## Task Commits

Each task was committed atomically:

1. **Task 1: Fix alert engine 3x json.Marshal silent error ignores** — `bddee15` (fix)
2. **Task 2: Fix Ontology V1 fallback SQL injection risk** — `73ddacc` (fix)

**Plan metadata:** No metadata commit (orchestrator owns state updates).

## Files Created/Modified

- `internal/alert/engine.go` — Engine struct now has `logger *zap.Logger` field, `WithLogger()` setter, package-level `engineLogger` var, 3 marshal error checks with zap logging
- `internal/repository/ontology/repository.go` — `fullTableName()` uses `pgx.Identifier.Sanitize()`, 4 V1 fallback paths have `GODEPRECATED` comments

## Decisions Made

- Used package-level `engineLogger` var (not Engine parameter passing) because global rule functions are package-level functions, not Engine methods. `EvaluateGlobalRules` injects the logger and defers cleanup.
- Marshal fallback uses `[]byte("{}")` — empty JSON object — so `Message` field still provides readable context. Alerts are never lost on marshal failure.
- Evidence keys logged as `zap.Strings(...)` to avoid leaking sensitive evidence content.
- `pgx.Identifier.Sanitize()` pattern matches V2 compiler in `internal/ontology/compiler.go`.
- Allowlist checks (`if !ok { return error }`) were already present in all 4 V1 fallback paths; `GODEPRECATED` comments added alongside to document migration intent.

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered

None.

## Self-Check: PASSED

- ✅ `internal/alert/engine.go` — exists, builds
- ✅ `internal/repository/ontology/repository.go` — exists, builds
- ✅ Commit `bddee15` — Task 1
- ✅ Commit `73ddacc` — Task 2
- ✅ `go build ./internal/alert/...` — clean
- ✅ `go build ./internal/repository/ontology/...` — clean
- ✅ `go build ./...` — clean

## Next Phase Readiness

- Alert engine safe under marshal failures; evidence logging complete.
- Ontology V1 fallback paths now SQL-injection-safe with proper sanitization.
- Next plan in this phase can address remaining bug fixes or stability improvements.
