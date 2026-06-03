---
phase: 05-security-hardening
plan: 01
subsystem: api
tags: [cors, security, middleware, scheme-validation]
requires: []
provides:
  - CORS scheme-aware origin validation using url.Parse
  - Port normalization (http:80 / https:443)
  - Fail-closed on unparseable origins
affects: [05-security-hardening]

tech-stack:
  added: []
  patterns:
    - "CORS origin validation parses scheme+host+port for exact matching"
    - "Port normalization assigns default ports (80/443) when omitted"

key-files:
  created: []
  modified:
    - internal/api/middleware/cors.go
    - internal/api/middleware/cors_test.go

key-decisions:
  - "Use url.Parse to compare scheme+host+port instead of string exact match"
  - "Fail closed on unparseable origins (log error, reject request)"
  - "Default port normalization: http→80, https→443"
  - "CORS_ALLOWED_ORIGINS comma-separated format unchanged"
  - "Use log.Printf for parse errors (no new dependencies)"

requirements-completed: [SEC-02]

duration: 1 min
completed: 2026-06-03
---

# Phase 5 Plan 1: CORS Scheme Validation Summary

**CORS middleware updated to validate scheme+host+port using url.Parse, with port normalization and fail-closed behavior for unparseable origins**

## Performance

- **Duration:** 1 min
- **Started:** 2026-06-03T14:40:38Z
- **Completed:** 2026-06-03T14:41:13Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- CORS middleware now parses both config origins and request Origin headers with `url.Parse()` for precise scheme+host+port comparison
- Port normalization: origins without explicit ports get default ports (80 for http, 443 for https) — `http://localhost` matches `http://localhost:80`, `https://example.com` matches `https://example.com:443`
- Fail-closed: unparseable request origins are rejected (no CORS headers set, request passes through as if origin not allowed)
- Invalid config entries are logged and skipped without breaking middleware initialization
- 6 new test functions cover scheme mismatch, port normalization (both directions), different port/host rejection, invalid origin rejection, and http→https scheme rejection

## Task Commits

Each task was committed atomically:

1. **Task 1: CORS scheme validation in cors.go** - `4546a0e` (fix)
2. **Task 2: CORS scheme validation tests** - `cf7f402` (test)

**Plan metadata:** Pending (orchestrator-led)

## Files Created/Modified
- `internal/api/middleware/cors.go` — `parseOrigins()` returns `[]url.URL`, `isOriginAllowed()` compares scheme+host+port, added `normalizeHostPort()` helper
- `internal/api/middleware/cors_test.go` — Added 6 test functions (132 lines): `SchemeMismatch`, `DefaultPortNormalization` (table-driven, 4 cases), `DifferentPortRejected`, `DifferentHostRejected`, `InvalidOrigin_Rejected`, `SchemeRejection_HttpVsHttps`

## Decisions Made
- Used `url.Parse()` to extract scheme+host+port from origins, rather than string manipulation or regex
- Port normalization: `url.Parse("http://example.com")` → `Host: "example.com:80"`; `url.Parse("https://example.com")` → `Host: "example.com:443"`
- Invalid config entries (e.g., `not-a-url`) are skipped with `log.Printf` warning — the middleware still initializes with valid entries
- Unparseable request Origin headers are rejected (fail closed) — no CORS headers set, no information leaked
- No new external dependencies: `net/url`, `net`, `log` are all Go standard library

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered
None

## Next Phase Readiness
- CORS scheme validation complete (SEC-02). Ready for next security-hardening tasks or phase completion.
- All 9 existing CORS tests continue to pass with zero regressions.
- All 15 CORS tests (9 existing + 6 new) pass.

---

*Phase: 05-security-hardening*
*Completed: 2026-06-03*
