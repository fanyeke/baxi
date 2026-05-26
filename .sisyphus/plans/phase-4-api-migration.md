# Phase 4: Go API Migration — Read-Side API from FastAPI/SQLite to Go/PostgreSQL

## TL;DR

> Migrate 11 core read-side API endpoints from old FastAPI + SQLite to Go + PostgreSQL, maintaining response compatibility with Phase 0 baseline snapshots. Implement Bearer token auth, request_id propagation, unified error responses, CORS, and pagination. Add automated API baseline comparison.
>
> **Deliverables**:
> - 11 Go API endpoints (health enhanced + 10 new)
> - Auth, error, CORS, request_id, pagination middleware
> - Repository + service + DTO layers for 7+ tables
> - `docs/migration/phase-4-api-migration-plan.md`
> - `scripts/migration/compare_api_baseline.py`
> - Comprehensive Go tests (unit + integration)
>
> **Estimated Effort**: Large
> **Parallel Execution**: YES — 3 waves + final verification
> **Critical Path**: Foundation (Wave 1) → Core Endpoints (Wave 2) → Composite Endpoints (Wave 3) → Verification (Wave FINAL)

---

## Context

### Original Request
User provided a comprehensive Phase 4 migration specification covering endpoint inventory, data source mappings, response compatibility requirements, auth strategy, request_id strategy, error format, pagination, and baseline comparison.

### Interview Summary
**Key Discussions**:
- **Pagination**: Old-compatible — default limit=100, max=1000, offset=0
- **Auth env var**: API_BEARER_TOKEN (primary), optional fallback to API_TOKEN
- **CORS**: Implement with CORS_ALLOWED_ORIGINS, default dev origins: localhost:5173, localhost:3000
- **Rate limiting**: Stub/no-op only, not blocking acceptance
- **Qoder capabilities**: Preserve old object format (can_read_status, can_read_alerts, etc.)
- **Compatibility priority**: API compatibility with Phase 0 baseline > cleaner new models

**Research Findings**:
- Go project has only 1 endpoint: GET /api/v1/health (static response, no DB ping)
- Chi router with basic middleware (RequestID, RealIP, Logger, Recoverer, Timeout)
- No auth, error, CORS middleware; no pagination helpers
- Only 1 repository: internal/outbox/repository.go (stateless, tx passed explicitly)
- PostgreSQL connection pool: internal/db/postgres.go (pgxpool)
- 7 baseline JSON snapshots exist in migration_baseline/api_responses/
- Makefile has `api` and `pipeline-compare`, missing `api-compare`
- No Phase 4 doc exists yet
- Old FastAPI has 24 endpoints with Bearer auth, structured errors, rate limiting, CORS
- Config only has DatabaseURL, APIPort, LogLevel — missing API_BEARER_TOKEN, CORS_ALLOWED_ORIGINS

### Metis Review
**Identified Gaps** (addressed in this plan):
- **Health endpoint discrepancy**: Current Go returns `{"status":"ok","service":"baxi-api"}`, baseline expects `{"status":"ok","version":"0.6.0","db_connected":true}`. Must fix.
- **Missing config fields**: API_BEARER_TOKEN and CORS_ALLOWED_ORIGINS must be added to internal/config/config.go
- **DTO location**: User specified internal/api/dto/ — follow this convention
- **Repository pattern**: Must follow existing stateless pattern (tx passed explicitly, no pool stored)
- **Null handling**: Must use pointer types for nullable fields to match Pydantic Optional[T]
- **ORDER BY**: Every list query must have explicit ORDER BY for deterministic array ordering
- **Datetime format**: May need custom time.Time marshaling if baseline uses `+00:00` instead of `Z`
- **Trailing slashes**: May need middleware.StripSlashes if old API supported trailing slash redirects
- **Validation errors**: Old FastAPI returns 422; Go typically returns 400. Must match baseline behavior.

---

## Work Objectives

### Core Objective
Implement 11 read-side API endpoints in Go that produce responses compatible with Phase 0 FastAPI baseline snapshots, reading from existing PostgreSQL schema (no schema changes).

### Concrete Deliverables
- `docs/migration/phase-4-api-migration-plan.md` — migration design document
- `internal/config/config.go` — add API_BEARER_TOKEN, CORS_ALLOWED_ORIGINS
- `internal/api/middleware/auth.go` — Bearer token auth with constant-time compare
- `internal/api/middleware/error.go` — unified JSON error response middleware
- `internal/api/middleware/cors.go` — CORS middleware with configurable origins
- `internal/api/middleware/request_id.go` — request_id from header or generated
- `internal/api/pagination.go` — limit/offset parsing and clamping
- `internal/api/response.go` — shared response helpers
- `internal/api/dto/` — DTO types for all endpoints
- `internal/api/handler/` — HTTP handlers for all endpoints
- `internal/service/` — business logic / aggregation services
- `internal/repository/` — PostgreSQL query repositories (stateless, tx passed)
- `scripts/migration/compare_api_baseline.py` — baseline comparison script
- Makefile `api-compare` target
- Comprehensive tests for all new code

### Definition of Done
- [x] All 11 endpoints return 200 with correct Bearer token
- [x] All 11 endpoints return 401 without token
- [x] `make api-compare` returns PASS or accepted WARN
- [x] `make pipeline-compare` still returns PASS with accepted WARN
- [x] `go test ./...` passes
- [x] No modifications to old Python/React/Pipeline code
- [x] CORS preflight works for configured origins
- [x] request_id propagated correctly

### Must Have
- 11 read endpoints fully functional
- Bearer token auth on all endpoints except health
- request_id generation and propagation
- Unified error response format
- Pagination with limit/offset
- CORS support
- API baseline comparison script
- Go tests passing

### Must NOT Have (Guardrails)
- **NO** write endpoints (POST outbox/dispatch, feishu/*, pipeline/run, qoder/reports)
- **NO** LLM integration
- **NO** Ontology Runtime
- **NO** real outbox dispatch
- **NO** React frontend changes
- **NO** old FastAPI code deletion
- **NO** Pipeline calculation logic changes
- **NO** YAML governance config semantic changes
- **NO** automatic scheduling
- **NO** write-side apply operations
- **NO** new Go module dependencies (chi, pgx, zap are sufficient)
- **NO** caching layer
- **NO** metrics/tracing/APM
- **NO** OpenAPI/Swagger generation
- **NO** GraphQL/gRPC/WebSocket
- **NO** schema changes
- **NO** changes to existing middleware stack behavior

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed.

### Test Decision
- **Infrastructure exists**: YES — Go project with testcontainers support
- **Automated tests**: Tests-after (implementation first, tests alongside)
- **Framework**: `go test` with testcontainers for integration tests
- **Pattern**: Each endpoint gets table-driven tests: happy path, auth failure, empty state

### QA Policy
Every task MUST include agent-executed QA scenarios. Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **API/Backend**: Use Bash (curl) — Send requests, assert status + response fields
- **Go tests**: Use Bash (`go test ./...`)
- **Baseline comparison**: Use Bash (`make api-compare`)

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Foundation — can start immediately, 7 parallel tasks):
├── Task 1: Phase 4A design document
├── Task 2: Config additions (API_BEARER_TOKEN, CORS_ALLOWED_ORIGINS)
├── Task 3: Auth middleware + Bearer token validation
├── Task 4: Error response middleware
├── Task 5: request_id middleware
├── Task 6: CORS middleware
├── Task 7: Pagination helpers + shared response utilities

Wave 2 (Core List Endpoints — after Wave 1, 4 parallel tasks):
├── Task 8: Health endpoint fix (add DB ping, version, db_connected)
├── Task 9: Alerts endpoint (handler + service + repository + DTO + tests)
├── Task 10: Tasks endpoint (handler + service + repository + DTO + tests)
├── Task 11: Outbox endpoint (handler + service + repository + DTO + tests)

Wave 3 (Logs + Governance + Qoder — after Wave 1, 5 parallel tasks):
├── Task 12: Logs endpoints (recent, errors, audit)
├── Task 13: Governance status endpoint
├── Task 14: Qoder capabilities endpoint
├── Task 15: Qoder context endpoint
├── Task 16: Status endpoint (composite, depends on understanding of other endpoints)

Wave 4 (Integration + Baseline Compare — after Waves 2-3, 2 parallel tasks):
├── Task 17: API baseline comparison script + Makefile target
├── Task 18: Integration tests + route registration in server.go

Wave FINAL (After ALL tasks — 4 parallel reviews):
├── Task F1: Plan compliance audit (oracle)
├── Task F2: Code quality review (unspecified-high)
├── Task F3: Real manual QA (unspecified-high)
├── Task F4: Scope fidelity check (deep)
-> Present results -> Get explicit user okay

Critical Path: Wave 1 (7 tasks) → Wave 2 (4 tasks) + Wave 3 (5 tasks in parallel) → Wave 4 (2 tasks) → Wave FINAL
Parallel Speedup: ~60% faster than sequential
Max Concurrent: 7 (Wave 1)
```

### Dependency Matrix

| Task | Depends On | Blocks |
|------|-----------|--------|
| 1 | — | — |
| 2 | — | 3, 6 |
| 3 | 2 | 8-18 |
| 4 | — | 8-18 |
| 5 | — | 8-18 |
| 6 | 2 | 8-18 |
| 7 | — | 8-18 |
| 8 | 1-7 | 17-18 |
| 9 | 1-7 | 16, 17-18 |
| 10 | 1-7 | 16, 17-18 |
| 11 | 1-7 | 16, 17-18 |
| 12 | 1-7 | 16, 17-18 |
| 13 | 1-7 | 16, 17-18 |
| 14 | 1-7 | 17-18 |
| 15 | 1-7, 9-14 | 17-18 |
| 16 | 1-7, 9-14 | 17-18 |
| 17 | 8-16 | F1-F4 |
| 18 | 8-16 | F1-F4 |
| F1-F4 | 17-18 | — |

### Agent Dispatch Summary

- **Wave 1**: 7 tasks → `quick` (config, middleware, helpers)
- **Wave 2**: 4 tasks → `unspecified-high` (endpoints with DB queries)
- **Wave 3**: 5 tasks → `unspecified-high` (endpoints with DB queries)
- **Wave 4**: 2 tasks → `quick` + `unspecified-high` (script + integration)
- **Wave FINAL**: 4 tasks → `oracle`, `unspecified-high`, `unspecified-high`, `deep`

---

## TODOs

- [x] 1. **Phase 4A: Write API Migration Design Document**

  **What to do**:
  - Create `docs/migration/phase-4-api-migration-plan.md`
  - Document: endpoint inventory (24 old, 11 migrated, 13 deferred)
  - Document: migration scope and non-goals
  - Document: handler/service/repository/DTO分层设计
  - Document: endpoint → PostgreSQL source table mapping
  - Document: response compatibility strategy (preserve field names, types, nullability)
  - Document: auth strategy (Bearer token, API_BEARER_TOKEN env)
  - Document: request_id strategy (header or generated, returned in response)
  - Document: unified error response strategy (request_id, error_code, message, diagnosis, suggested_action)
  - Document: pagination/filtering/sorting strategy (limit default 100, max 1000, offset 0)
  - Document: CORS strategy (CORS_ALLOWED_ORIGINS env)
  - Document: baseline comparison strategy (scripts/migration/compare_api_baseline.py)
  - Document: acceptance criteria

  **Must NOT do**:
  - Do not design write endpoints
  - Do not design LLM integration
  - Do not modify existing docs

  **Recommended Agent Profile**:
  - **Category**: `writing`
  - **Skills**: []
  - Reason: Documentation task, no code changes

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: None
  - **Blocked By**: None

  **References**:
  - `docs/migration/go-postgres-migration-plan.md` — existing migration plan structure to follow
  - `docs/migration/phase-3-pipeline-migration-plan.md` — Phase 3 doc format
  - `migration_baseline/api_responses/` — baseline snapshots to reference
  - `api/routers/` — old FastAPI endpoint implementations
  - `api/schemas.py` — old Pydantic models

  **Acceptance Criteria**:
  - [ ] File exists: `docs/migration/phase-4-api-migration-plan.md`
  - [ ] File contains all 13 required sections
  - [ ] File references actual table names and endpoint paths

  **QA Scenarios**:
  ```
  Scenario: Document completeness
    Tool: Bash
    Preconditions: None
    Steps:
      1. grep -c "endpoint inventory" docs/migration/phase-4-api-migration-plan.md
      2. grep -c "acceptance criteria" docs/migration/phase-4-api-migration-plan.md
    Expected Result: Both grep commands return count >= 1
    Evidence: .sisyphus/evidence/task-1-doc-complete.txt
  ```

  **Commit**: YES
  - Message: `docs: add phase 4 api migration plan`
  - Files: `docs/migration/phase-4-api-migration-plan.md`

---

- [x] 2. **Add Config Fields: API_BEARER_TOKEN and CORS_ALLOWED_ORIGINS**

  **What to do**:
  - Modify `internal/config/config.go` to add:
    - `APIBearerToken string` (required, min 32 chars)
    - `CORSAllowedOrigins string` (optional, default: "http://localhost:5173,http://localhost:3000")
  - Update `.env.example` with new variables
  - Ensure config loads from environment variables
  - Add validation: APIBearerToken must not be empty in production

  **Must NOT do**:
  - Do not change existing config fields (DatabaseURL, APIPort, LogLevel)
  - Do not add file-based config

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - Reason: Simple config addition, no complex logic

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: Task 3 (auth middleware), Task 6 (CORS middleware)
  - **Blocked By**: None

  **References**:
  - `internal/config/config.go` — existing config structure
  - `.env.example` — existing env template
  - `api/auth.py` — old auth token validation (min 32 chars, weak token rejection)

  **Acceptance Criteria**:
  - [ ] `go test ./internal/config/...` passes
  - [ ] Config struct has APIBearerToken and CORSAllowedOrigins fields
  - [ ] `.env.example` updated

  **QA Scenarios**:
  ```
  Scenario: Config loads correctly
    Tool: Bash
    Preconditions: .env has API_BEARER_TOKEN=test_token_32_chars_min_length
    Steps:
      1. go run ./cmd/baxi-api (or test program)
      2. Verify config.APIBearerToken == "test_token_32_chars_min_length"
    Expected Result: Config loads without error
    Evidence: .sisyphus/evidence/task-2-config-load.txt
  ```

  **Commit**: NO (groups with Wave 1 infrastructure commit)

---

- [x] 3. **Implement Bearer Token Auth Middleware**

  **What to do**:
  - Create `internal/api/middleware/auth.go`
  - Implement `AuthMiddleware(next http.Handler) http.Handler`
  - Extract `Authorization: Bearer <token>` header
  - Use `subtle.ConstantTimeCompare` for token comparison
  - Reject tokens < 32 chars
  - Reject known weak tokens ("test-token", "changeme", "admin", "password", "REPLACE_ME")
  - Skip auth for public endpoints (health only)
  - Return 401 with unified error format for missing/invalid token
  - Store authenticated actor in request context (default "qoder", fallback "unknown")

  **Must NOT do**:
  - Do not implement session-based auth
  - Do not implement JWT
  - Do not add per-user scoping
  - Do not store token in DB

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - Reason: Straightforward middleware, well-defined requirements

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: All endpoint handlers (Tasks 8-18)
  - **Blocked By**: Task 2 (config fields)

  **References**:
  - `api/auth.py` — old auth implementation (hmac.compare_digest, weak token checks)
  - `api/dependencies.py` — old DI flow for auth
  - `internal/api/server.go` — middleware mounting location

  **Acceptance Criteria**:
  - [ ] `go test ./internal/api/middleware/...` passes
  - [ ] curl without Authorization header → 401
  - [ ] curl with wrong token → 401
  - [ ] curl with correct token → 200
  - [ ] curl to /health without token → 200

  **QA Scenarios**:
  ```
  Scenario: Auth rejects missing token
    Tool: Bash (curl)
    Preconditions: API server running
    Steps:
      1. curl -s -w "\n%{http_code}" http://localhost:8080/api/v1/status
    Expected Result: HTTP 401, body contains "UNAUTHORIZED"
    Evidence: .sisyphus/evidence/task-3-auth-missing.txt

  Scenario: Auth accepts valid token
    Tool: Bash (curl)
    Preconditions: API server running with API_BEARER_TOKEN=dev_token_32_chars_minimum_length
    Steps:
      1. curl -s -w "\n%{http_code}" -H "Authorization: Bearer dev_token_32_chars_minimum_length" http://localhost:8080/api/v1/status
    Expected Result: HTTP 200 (or other success, not 401)
    Evidence: .sisyphus/evidence/task-3-auth-valid.txt

  Scenario: Auth allows public health
    Tool: Bash (curl)
    Preconditions: API server running
    Steps:
      1. curl -s -w "\n%{http_code}" http://localhost:8080/api/v1/health
    Expected Result: HTTP 200
    Evidence: .sisyphus/evidence/task-3-auth-public.txt
  ```

  **Commit**: NO (groups with Wave 1 infrastructure commit)

---

- [x] 4. **Implement Unified Error Response Middleware**

  **What to do**:
  - Create `internal/api/middleware/error.go`
  - Define `APIError` struct:
    ```go
    type APIError struct {
        RequestID       string `json:"request_id"`
        ErrorCode       string `json:"error_code"`
        Message         string `json:"message"`
        Diagnosis       string `json:"diagnosis,omitempty"`
        SuggestedAction string `json:"suggested_action,omitempty"`
    }
    ```
  - Define error codes: UNAUTHORIZED, FORBIDDEN, BAD_REQUEST, NOT_FOUND, DB_QUERY_FAILED, INTERNAL_ERROR, VALIDATION_FAILED
  - Implement `WriteError(w http.ResponseWriter, req *http.Request, status int, code string, message string, diagnosis string, action string)`
  - Implement panic recovery that returns INTERNAL_ERROR with request_id
  - Return request_id from context in all error responses
  - Match old FastAPI error format exactly

  **Must NOT do**:
  - Do not add new error fields beyond baseline
  - Do not use Go's default error format

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - Reason: Well-defined struct and middleware pattern

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: All endpoint handlers (Tasks 8-18)
  - **Blocked By**: None

  **References**:
  - `api/errors.py` — old error format and error codes
  - `migration_baseline/api_responses/` — baseline error responses if any

  **Acceptance Criteria**:
  - [ ] `go test ./internal/api/middleware/...` passes
  - [ ] Error response contains all required fields
  - [ ] Error response includes request_id

  **QA Scenarios**:
  ```
  Scenario: Error response format
    Tool: Bash (curl)
    Preconditions: API server running
    Steps:
      1. curl -s http://localhost:8080/api/v1/status
      2. jq -e '.request_id and .error_code and .message' response.json
    Expected Result: jq returns true
    Evidence: .sisyphus/evidence/task-4-error-format.txt
  ```

  **Commit**: NO (groups with Wave 1 infrastructure commit)

---

- [x] 5. **Implement Request ID Middleware**

  **What to do**:
  - Create `internal/api/middleware/request_id.go`
  - Check for `X-Request-ID` header; if present, use it
  - If not present, generate request_id (format: `req_<timestamp>_<random>` or UUID)
  - Store request_id in request context
  - Return `X-Request-ID` in response headers
  - Ensure error responses include request_id from context

  **Must NOT do**:
  - Do not write to audit.api_request_log in Phase 4 (deferred)
  - Do not change existing chi middleware.RequestID behavior (keep it for logging)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - Reason: Simple middleware, existing chi RequestID as reference

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: All endpoint handlers (Tasks 8-18)
  - **Blocked By**: None

  **References**:
  - `api/main.py` — old request_id generation (UUID via ContextVar)
  - `internal/api/server.go` — existing chi middleware.RequestID

  **Acceptance Criteria**:
  - [ ] `go test ./internal/api/middleware/...` passes
  - [ ] Custom X-Request-ID propagated to response
  - [ ] Generated request_id returned in response header

  **QA Scenarios**:
  ```
  Scenario: Request ID propagation
    Tool: Bash (curl)
    Preconditions: API server running
    Steps:
      1. curl -s -D - -H "X-Request-ID: test-req-123" http://localhost:8080/api/v1/health
      2. grep -i "X-Request-ID" headers.txt
    Expected Result: Response contains "X-Request-ID: test-req-123"
    Evidence: .sisyphus/evidence/task-5-request-id.txt
  ```

  **Commit**: NO (groups with Wave 1 infrastructure commit)

---

- [x] 6. **Implement CORS Middleware**

  **What to do**:
  - Create `internal/api/middleware/cors.go`
  - Read allowed origins from `CORS_ALLOWED_ORIGINS` env (comma-separated)
  - Default origins: `http://localhost:5173,http://localhost:3000`
  - Handle preflight OPTIONS requests with correct headers:
    - Access-Control-Allow-Origin
    - Access-Control-Allow-Methods: GET, POST, OPTIONS
    - Access-Control-Allow-Headers: Authorization, Content-Type, X-Request-ID
  - Return 204 for preflight
  - Add CORS headers to actual responses

  **Must NOT do**:
  - Do not allow all origins (*)
  - Do not add credentials support unless required

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - Reason: Well-defined middleware with clear requirements

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: All endpoint handlers (Tasks 8-18)
  - **Blocked By**: Task 2 (config fields)

  **References**:
  - `api/main.py` — old CORS configuration (CORS_ORIGINS env, explicit methods)

  **Acceptance Criteria**:
  - [ ] `go test ./internal/api/middleware/...` passes
  - [ ] Preflight OPTIONS returns 204 with correct headers
  - [ ] Actual responses include Access-Control-Allow-Origin

  **QA Scenarios**:
  ```
  Scenario: CORS preflight
    Tool: Bash (curl)
    Preconditions: API server running
    Steps:
      1. curl -s -X OPTIONS -D - -H "Origin: http://localhost:5173" -H "Access-Control-Request-Method: GET" http://localhost:8080/api/v1/health
      2. grep -i "Access-Control-Allow-Origin" headers.txt
    Expected Result: Contains "Access-Control-Allow-Origin: http://localhost:5173" and HTTP 204
    Evidence: .sisyphus/evidence/task-6-cors-preflight.txt
  ```

  **Commit**: NO (groups with Wave 1 infrastructure commit)

---

- [x] 7. **Implement Pagination Helpers and Shared Response Utilities**

  **What to do**:
  - Create `internal/api/pagination.go`:
    - `PaginationParams` struct with Limit and Offset
    - `ParsePagination(r *http.Request) (PaginationParams, error)`
    - Default limit=100, max limit=1000, min limit=1
    - Default offset=0, min offset=0
    - Clamp limit to [1, 1000]
    - Clamp offset to >= 0
  - Create `internal/api/response.go`:
    - `PaginatedResponse[T]` struct with Items and Pagination metadata
    - `JSON(w http.ResponseWriter, status int, data interface{})` helper
    - Standard pagination metadata: limit, offset, total
  - Support sorting via whitelist map (string → SQL clause)

  **Must NOT do**:
  - Do not allow arbitrary SQL in sort parameters
  - Do not allow negative limit/offset

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - Reason: Helper functions, no external dependencies

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: All list endpoints (Tasks 9-12)
  - **Blocked By**: None

  **References**:
  - Old FastAPI list endpoints — limit/offset behavior
  - `api/routers/alerts.py` — old pagination parameters

  **Acceptance Criteria**:
  - [ ] `go test ./internal/api/...` passes
  - [ ] limit=0 clamps to 100
  - [ ] limit=9999 clamps to 1000
  - [ ] offset=-1 clamps to 0
  - [ ] Pagination metadata included in responses

  **QA Scenarios**:
  ```
  Scenario: Pagination clamping
    Tool: Bash (go test)
    Preconditions: None
    Steps:
      1. go test ./internal/api/... -run TestPagination
    Expected Result: PASS
    Evidence: .sisyphus/evidence/task-7-pagination-test.txt
  ```

  **Commit**: NO (groups with Wave 1 infrastructure commit)
  - Message: `feat: add api middleware and response foundation`
  - Files: `internal/api/middleware/*`, `internal/api/response.go`, `internal/api/pagination.go`, `internal/config/config.go`, `.env.example`

---

- [x] 8. **Fix Health Endpoint (Add DB Ping, Version, db_connected)**

  **What to do**:
  - Modify `internal/api/health.go` to:
    - Ping PostgreSQL using `internal/db/postgres.go` pool
    - Return `{"status":"ok","version":"0.6.0","db_connected":true/false}`
    - Remove `service` field (not in baseline)
    - Add version constant (match baseline `0.6.0`)
  - Add `Version` field to config or as package constant
  - Ensure health endpoint remains public (no auth)

  **Must NOT do**:
  - Do not change the endpoint path
  - Do not add auth to health
  - Do not return non-baseline fields

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - Reason: Small modification to existing endpoint

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 9-11)
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 17 (api-compare)
  - **Blocked By**: Tasks 1-7 (foundation)

  **References**:
  - `migration_baseline/api_responses/health.json` — baseline snapshot
  - `internal/api/health.go` — current implementation
  - `internal/db/postgres.go` — pool for DB ping

  **Acceptance Criteria**:
  - [ ] Response matches baseline: `{"status":"ok","version":"0.6.0","db_connected":true}`
  - [ ] No `service` field in response
  - [ ] DB ping actually checks connectivity
  - [ ] `go test ./internal/api/...` passes

  **QA Scenarios**:
  ```
  Scenario: Health matches baseline
    Tool: Bash (curl)
    Preconditions: API server running with DB up
    Steps:
      1. curl -s http://localhost:8080/api/v1/health
      2. jq -e '.status == "ok" and .version == "0.6.0" and .db_connected == true and has("service") | not'
    Expected Result: jq returns true
    Evidence: .sisyphus/evidence/task-8-health-baseline.txt

  Scenario: Health without auth
    Tool: Bash (curl)
    Preconditions: API server running
    Steps:
      1. curl -s -w "\n%{http_code}" http://localhost:8080/api/v1/health
    Expected Result: HTTP 200
    Evidence: .sisyphus/evidence/task-8-health-public.txt
  ```

  **Commit**: YES
  - Message: `feat: fix health endpoint to match baseline`
  - Files: `internal/api/health.go`, `internal/config/config.go` (if Version added)

---

- [x] 9. **Implement GET /api/v1/alerts**

  **What to do**:
  - Create `internal/api/dto/alert.go` — AlertItem, AlertListResponse DTOs
  - Create `internal/repository/alert_repository.go` — stateless repository
    - `ListAlerts(ctx context.Context, tx pgx.Tx, filters AlertFilters, pagination PaginationParams) ([]AlertItem, int, error)`
    - Source table: `ops.metric_alert`
    - Support filters: severity, status, object_type, rule_id
    - Support sort: created_at_desc, created_at_asc, severity_desc
    - Add explicit `ORDER BY created_at DESC` default
  - Create `internal/service/alert_service.go` — business logic layer
    - `ListAlerts(ctx context.Context, filters AlertFilters, pagination PaginationParams) (*AlertListResponse, error)`
    - Maps repository data to DTO, handles empty states
  - Create `internal/api/handler/alerts.go` — HTTP handler
    - Parse query params (severity, status, object_type, rule_id, limit, offset, sort)
    - Call service, return paginated response
  - Register route in `internal/api/routes.go` (or `server.go`)
  - Add tests

  **Must NOT do**:
  - Do not implement POST /alerts
  - Do not implement GET /alerts/{id}
  - Do not modify alert rule logic

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
  - Reason: Multiple files, DB queries, needs to match baseline exactly

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 8, 10, 11)
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 15 (qoder/context), Task 16 (status), Task 17-18
  - **Blocked By**: Tasks 1-7 (foundation)

  **References**:
  - `migration_baseline/api_responses/alerts.json` — baseline snapshot
  - `api/routers/alerts.py` — old implementation
  - `api/schemas.py` — AlertEvent Pydantic model
  - `internal/outbox/repository.go` — repository pattern to follow

  **Acceptance Criteria**:
  - [ ] Response structure matches baseline (items[], total)
  - [ ] Pagination works (limit, offset)
  - [ ] Filtering works (severity, status)
  - [ ] Empty result returns `{"items":[],"pagination":{"limit":100,"offset":0,"total":0}}`
  - [ ] `go test ./...` passes

  **QA Scenarios**:
  ```
  Scenario: Alerts list with auth
    Tool: Bash (curl)
    Preconditions: API server running, DB has alerts data
    Steps:
      1. curl -s -H "Authorization: Bearer $API_TOKEN" http://localhost:8080/api/v1/alerts
      2. jq -e '.items | type == "array" and .pagination.total >= 0'
    Expected Result: jq returns true, HTTP 200
    Evidence: .sisyphus/evidence/task-9-alerts-list.txt

  Scenario: Alerts filtering
    Tool: Bash (curl)
    Preconditions: API server running
    Steps:
      1. curl -s -H "Authorization: Bearer $API_TOKEN" "http://localhost:8080/api/v1/alerts?severity=high"
      2. jq -e '.items | all(.severity == "high")'
    Expected Result: jq returns true (or empty array if no high severity alerts)
    Evidence: .sisyphus/evidence/task-9-alerts-filter.txt

  Scenario: Alerts without auth
    Tool: Bash (curl)
    Preconditions: API server running
    Steps:
      1. curl -s -w "\n%{http_code}" http://localhost:8080/api/v1/alerts
    Expected Result: HTTP 401
    Evidence: .sisyphus/evidence/task-9-alerts-auth.txt
  ```

  **Commit**: YES
  - Message: `feat: add alerts read api`
  - Files: `internal/api/dto/alert.go`, `internal/repository/alert_repository.go`, `internal/service/alert_service.go`, `internal/api/handler/alerts.go`, `internal/api/handler/alerts_test.go`

---

- [x] 10. **Implement GET /api/v1/tasks**

  **What to do**:
  - Create `internal/api/dto/task.go` — TaskItem, TaskListResponse DTOs
  - Create `internal/repository/task_repository.go` — stateless repository
    - `ListTasks(ctx context.Context, tx pgx.Tx, filters TaskFilters, pagination PaginationParams) ([]TaskItem, int, error)`
    - Source tables: `ops.task`, `ops.recommendation`, `ops.metric_alert` (join for related data)
    - Support filters: status, priority, owner
    - Add explicit `ORDER BY created_at DESC` default
  - Create `internal/service/task_service.go`
  - Create `internal/api/handler/tasks.go` — HTTP handler
    - Parse query params (status, priority, owner, limit, offset)
  - Register route
  - Add tests

  **Must NOT do**:
  - Do not implement POST /tasks
  - Do not implement task assignment/approval

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
  - Reason: Multiple files, DB queries, join across tables

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 8, 9, 11)
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 15 (qoder/context), Task 16 (status), Task 17-18
  - **Blocked By**: Tasks 1-7 (foundation)

  **References**:
  - `migration_baseline/api_responses/tasks.json` — baseline snapshot
  - `api/routers/tasks.py` — old implementation
  - `api/schemas.py` — ActionTask Pydantic model

  **Acceptance Criteria**:
  - [ ] Response structure matches baseline
  - [ ] Pagination works
  - [ ] Filtering works (status, priority, owner)
  - [ ] `go test ./...` passes

  **QA Scenarios**:
  ```
  Scenario: Tasks list with auth
    Tool: Bash (curl)
    Preconditions: API server running, DB has tasks data
    Steps:
      1. curl -s -H "Authorization: Bearer $API_TOKEN" http://localhost:8080/api/v1/tasks
      2. jq -e '.items | type == "array"'
    Expected Result: jq returns true, HTTP 200
    Evidence: .sisyphus/evidence/task-10-tasks-list.txt
  ```

  **Commit**: YES
  - Message: `feat: add tasks read api`
  - Files: `internal/api/dto/task.go`, `internal/repository/task_repository.go`, `internal/service/task_service.go`, `internal/api/handler/tasks.go`, `internal/api/handler/tasks_test.go`

---

- [x] 11. **Implement GET /api/v1/outbox**

  **What to do**:
  - Create `internal/api/dto/outbox.go` — OutboxItem, OutboxListResponse DTOs
  - Create `internal/repository/outbox_repository.go` (or extend existing)
    - `ListOutboxEvents(ctx context.Context, tx pgx.Tx, filters OutboxFilters, pagination PaginationParams) ([]OutboxItem, int, error)`
    - Source tables: `ops.outbox_event`, `ops.dispatch_attempt`
    - Support filters: status, channel, event_type
    - Add explicit `ORDER BY created_at DESC` default
  - Create `internal/service/outbox_service.go`
  - Create `internal/api/handler/outbox.go` — HTTP handler
    - Parse query params (status, channel, event_type, limit, offset)
  - Register route
  - Add tests

  **Must NOT do**:
  - Do not implement POST /outbox/dispatch
  - Do not implement outbox worker consumption

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
  - Reason: Multiple files, DB queries, existing repository to extend

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 8, 9, 10)
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 15 (qoder/context), Task 16 (status), Task 17-18
  - **Blocked By**: Tasks 1-7 (foundation)

  **References**:
  - `migration_baseline/api_responses/outbox.json` — baseline snapshot
  - `api/routers/outbox.py` — old implementation
  - `api/schemas.py` — EventOutbox Pydantic model
  - `internal/outbox/repository.go` — existing outbox repository

  **Acceptance Criteria**:
  - [ ] Response structure matches baseline
  - [ ] Pagination works
  - [ ] Filtering works (status, channel, event_type)
  - [ ] `go test ./...` passes

  **QA Scenarios**:
  ```
  Scenario: Outbox list with auth
    Tool: Bash (curl)
    Preconditions: API server running, DB has outbox data
    Steps:
      1. curl -s -H "Authorization: Bearer $API_TOKEN" http://localhost:8080/api/v1/outbox
      2. jq -e '.items | type == "array"'
    Expected Result: jq returns true, HTTP 200
    Evidence: .sisyphus/evidence/task-11-outbox-list.txt
  ```

  **Commit**: YES
  - Message: `feat: add outbox read api`
  - Files: `internal/api/dto/outbox.go`, `internal/repository/outbox_repository.go`, `internal/service/outbox_service.go`, `internal/api/handler/outbox.go`, `internal/api/handler/outbox_test.go`

---

- [x] 12. **Implement Logs Endpoints (recent, errors, audit)**

  **What to do**:
  - Create `internal/api/dto/logs.go` — LogItem, LogListResponse DTOs
  - Create `internal/repository/log_repository.go` — stateless repository
    - `ListRecentLogs(ctx, tx, pagination)` — from `audit.api_request_log`, `audit.pipeline_run`, `audit.pipeline_step_run`
    - `ListErrorLogs(ctx, tx, pagination)` — from `audit.error_log`, `audit.pipeline_step_run`
    - `ListAuditLogs(ctx, tx, pagination)` — from `audit.audit_log`
    - Add explicit `ORDER BY created_at DESC` default
  - Create `internal/service/log_service.go`
  - Create `internal/api/handler/logs.go` — HTTP handler with 3 handlers:
    - `GET /api/v1/logs/recent`
    - `GET /api/v1/logs/errors`
    - `GET /api/v1/logs/audit`
  - Register routes
  - Add tests

  **Must NOT do**:
  - Do not implement POST /logs
  - Do not implement complex log aggregation beyond simple list
  - Do not fail if tables are empty (return empty arrays)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
  - Reason: Multiple endpoints, multiple source tables, empty-state handling

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 13-16)
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 15 (qoder/context), Task 16 (status), Task 17-18
  - **Blocked By**: Tasks 1-7 (foundation)

  **References**:
  - `api/routers/logs.py` — old implementation
  - `api/schemas.py` — Log schemas
  - `migration_baseline/api_responses/` — may not have separate log snapshots; use old schemas as reference

  **Acceptance Criteria**:
  - [ ] All 3 endpoints return 200 with valid token
  - [ ] Empty tables return `{"items":[],"pagination":{"limit":100,"offset":0,"total":0}}`
  - [ ] Response structure matches old schemas
  - [ ] `go test ./...` passes

  **QA Scenarios**:
  ```
  Scenario: Logs recent with auth
    Tool: Bash (curl)
    Preconditions: API server running
    Steps:
      1. curl -s -H "Authorization: Bearer $API_TOKEN" http://localhost:8080/api/v1/logs/recent
      2. jq -e '.items | type == "array"'
    Expected Result: jq returns true, HTTP 200
    Evidence: .sisyphus/evidence/task-12-logs-recent.txt

  Scenario: Logs errors empty state
    Tool: Bash (curl)
    Preconditions: API server running
    Steps:
      1. curl -s -H "Authorization: Bearer $API_TOKEN" http://localhost:8080/api/v1/logs/errors
      2. jq -e '.items == [] and .pagination.total == 0'
    Expected Result: jq returns true
    Evidence: .sisyphus/evidence/task-12-logs-errors-empty.txt
  ```

  **Commit**: YES
  - Message: `feat: add logs read api`
  - Files: `internal/api/dto/logs.go`, `internal/repository/log_repository.go`, `internal/service/log_service.go`, `internal/api/handler/logs.go`, `internal/api/handler/logs_test.go`

---

- [x] 13. **Implement GET /api/v1/governance/status**

  **What to do**:
  - Create `internal/api/dto/governance.go` — GovernanceStatusResponse DTO
  - Create `internal/repository/governance_repository.go` — stateless repository
    - `GetConfigCount(ctx, tx)` — from `gov.config_snapshot`
    - `GetSchemaAvailability(ctx, tx)` — from `gov.object_schema`
    - `GetClassificationAvailability(ctx, tx)` — from `gov.data_classification`
    - `GetLineageAvailability(ctx, tx)` — from `gov.data_lineage`
    - `GetHealthCheckStatus(ctx, tx)` — from `gov.health_check_result`
  - Create `internal/service/governance_service.go`
    - Aggregate data from multiple gov.* tables
    - Return simplified status (not full Governance Runtime)
  - Create `internal/api/handler/governance.go` — HTTP handler
    - `GET /api/v1/governance/status`
  - Register route
  - Add tests

  **Must NOT do**:
  - Do not implement governance catalog, classification, markings, lineage, checkpoints, health detail endpoints
  - Do not implement full Governance Runtime
  - Do not modify YAML governance config semantics

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
  - Reason: Composite endpoint, multiple source tables, aggregation logic

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 12, 14, 15, 16)
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 15 (qoder/context), Task 16 (status), Task 17-18
  - **Blocked By**: Tasks 1-7 (foundation)

  **References**:
  - `migration_baseline/api_responses/governance_status.json` — baseline snapshot
  - `api/routers/governance.py` — old implementation
  - `api/schemas.py` — Governance schemas

  **Acceptance Criteria**:
  - [ ] Response structure matches baseline
  - [ ] Returns aggregation from gov.* tables
  - [ ] Handles missing data gracefully (return false/unknown)
  - [ ] `go test ./...` passes

  **QA Scenarios**:
  ```
  Scenario: Governance status with auth
    Tool: Bash (curl)
    Preconditions: API server running
    Steps:
      1. curl -s -H "Authorization: Bearer $API_TOKEN" http://localhost:8080/api/v1/governance/status
      2. jq -e '.status == "ok" or .governance_layer == "active"'
    Expected Result: jq returns true, HTTP 200
    Evidence: .sisyphus/evidence/task-13-governance-status.txt
  ```

  **Commit**: YES
  - Message: `feat: add governance status read api`
  - Files: `internal/api/dto/governance.go`, `internal/repository/governance_repository.go`, `internal/service/governance_service.go`, `internal/api/handler/governance.go`, `internal/api/handler/governance_test.go`

---

- [x] 14. **Implement GET /api/v1/qoder/capabilities**

  **What to do**:
  - Create `internal/api/dto/qoder.go` — CapabilitiesResponse DTO with old object format
  - Implement as static response (no DB query needed for Phase 4)
    ```go
    type CapabilitiesResponse struct {
        Mode                string `json:"mode"`
        Version             string `json:"version"`
        CanReadStatus       bool   `json:"can_read_status"`
        CanReadAlerts       bool   `json:"can_read_alerts"`
        CanReadTasks        bool   `json:"can_read_tasks"`
        CanReadOutbox       bool   `json:"can_read_outbox"`
        CanReadGovernance   bool   `json:"can_read_governance"`
        CanReadLogs         bool   `json:"can_read_logs"`
        CanWriteReports     bool   `json:"can_write_reports"`
        CanExecuteActions   bool   `json:"can_execute_actions"`
    }
    ```
  - Values: mode="read_only", version="0.6.0", read capabilities=true, write/execute=false
  - Create `internal/api/handler/qoder.go` — HTTP handler
  - Register route
  - Add tests

  **Must NOT do**:
  - Do not switch to array-based capability list
  - Do not read from DB (static for Phase 4)
  - Do not implement POST /qoder/reports

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - Reason: Static endpoint, no DB queries

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 12, 13, 15, 16)
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 17-18
  - **Blocked By**: Tasks 1-7 (foundation)

  **References**:
  - Old FastAPI response format (object with can_read_* fields)
  - `api/routers/qoder.py` — old capabilities endpoint

  **Acceptance Criteria**:
  - [ ] Response matches old object format (not array)
  - [ ] All read capabilities are true
  - [ ] All write/execute capabilities are false
  - [ ] `go test ./...` passes

  **QA Scenarios**:
  ```
  Scenario: Capabilities response format
    Tool: Bash (curl)
    Preconditions: API server running
    Steps:
      1. curl -s -H "Authorization: Bearer $API_TOKEN" http://localhost:8080/api/v1/qoder/capabilities
      2. jq -e '.mode == "read_only" and .can_read_status == true and .can_execute_actions == false'
    Expected Result: jq returns true, HTTP 200
    Evidence: .sisyphus/evidence/task-14-capabilities-format.txt
  ```

  **Commit**: YES
  - Message: `feat: add qoder capabilities read api`
  - Files: `internal/api/dto/qoder.go`, `internal/api/handler/qoder.go`, `internal/api/handler/qoder_test.go`

---

- [x] 15. **Implement GET /api/v1/qoder/context**

  **What to do**:
  - Extend `internal/api/dto/qoder.go` — ContextResponse DTO
  - Extend `internal/service/qoder_service.go` (or create)
    - Aggregate data from multiple sources:
      - System status: from status service
      - Alerts summary: from alert service
      - Tasks summary: from task service
      - Outbox summary: from outbox service
      - Governance summary: from governance service
    - Build composite response matching baseline
  - Extend `internal/api/handler/qoder.go`
    - `GET /api/v1/qoder/context`
    - Parse query params: severity, limit_alerts, limit_tasks, limit_outbox, include_logs
  - Add tests

  **Must NOT do**:
  - Do not call LLM
  - Do not generate action proposals
  - Do not write decision_case
  - Do not implement write operations

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
  - Reason: Most complex endpoint, aggregates data from multiple services

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 12-14, 16) — but depends on understanding of other services
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 17-18
  - **Blocked By**: Tasks 1-7 (foundation), Tasks 9-14 (understanding of other endpoints)

  **References**:
  - `migration_baseline/api_responses/qoder_context.json` — baseline snapshot
  - `api/routers/qoder.py` — old context endpoint
  - `api/schemas_qoder.py` — QoderContextResponse model

  **Acceptance Criteria**:
  - [ ] Response structure matches baseline
  - [ ] Aggregates data from alerts, tasks, outbox, governance
  - [ ] No LLM calls
  - [ ] `go test ./...` passes

  **QA Scenarios**:
  ```
  Scenario: Qoder context with auth
    Tool: Bash (curl)
    Preconditions: API server running with data in DB
    Steps:
      1. curl -s -H "Authorization: Bearer $API_TOKEN" http://localhost:8080/api/v1/qoder/context
      2. jq -e '.system.status == "ok" and .alerts.total >= 0 and .tasks.total >= 0'
    Expected Result: jq returns true, HTTP 200
    Evidence: .sisyphus/evidence/task-15-qoder-context.txt
  ```

  **Commit**: YES
  - Message: `feat: add qoder context read api`
  - Files: `internal/api/dto/qoder.go`, `internal/service/qoder_service.go`, `internal/api/handler/qoder.go`, `internal/api/handler/qoder_test.go`

---

- [x] 16. **Implement GET /api/v1/status**

  **What to do**:
  - Create `internal/api/dto/status.go` — StatusResponse DTO
  - Create `internal/repository/status_repository.go` — stateless repository
    - `GetTableCounts(ctx, tx)` — counts from dwd.*, mart.*, ops.* tables
    - `GetLastPipelineRun(ctx, tx)` — from `audit.pipeline_run`
    - `GetPipelineStepStatus(ctx, tx)` — from `audit.pipeline_step_run`
  - Create `internal/service/status_service.go`
    - Aggregate table counts, pipeline run info
    - Build response matching baseline
  - Create `internal/api/handler/status.go` — HTTP handler
  - Register route
  - Add tests

  **Must NOT do**:
  - Do not modify pipeline run logic
  - Do not add new status fields beyond baseline

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
  - Reason: Composite endpoint, multiple table counts, pipeline run aggregation

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Tasks 12-15)
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 17-18
  - **Blocked By**: Tasks 1-7 (foundation), Tasks 9-14 (understanding of data sources)

  **References**:
  - `migration_baseline/api_responses/status.json` — baseline snapshot
  - `api/routers/status.py` — old implementation
  - `api/schemas.py` — StatusResponse model

  **Acceptance Criteria**:
  - [ ] Response structure matches baseline
  - [ ] Table counts match actual DB counts
  - [ ] Pipeline run info correct
  - [ ] `go test ./...` passes

  **QA Scenarios**:
  ```
  Scenario: Status with auth
    Tool: Bash (curl)
    Preconditions: API server running with pipeline data
    Steps:
      1. curl -s -H "Authorization: Bearer $API_TOKEN" http://localhost:8080/api/v1/status
      2. jq -e '.database != null and .last_pipeline_run != null'
    Expected Result: jq returns true, HTTP 200
    Evidence: .sisyphus/evidence/task-16-status.txt
  ```

  **Commit**: YES
  - Message: `feat: add status read api`
  - Files: `internal/api/dto/status.go`, `internal/repository/status_repository.go`, `internal/service/status_service.go`, `internal/api/handler/status.go`, `internal/api/handler/status_test.go`

---

- [x] 17. **Create API Baseline Comparison Script + Makefile Target**

  **What to do**:
  - Create `scripts/migration/compare_api_baseline.py`:
    - Read baseline JSON files from `migration_baseline/api_responses/`
    - Make HTTP requests to new Go API endpoints
    - Compare responses using semantic JSON comparison (field-order-agnostic)
    - Check core fields exist and have correct types
    - Check items count/total roughly match
    - Accept known differences: request_id, timestamps, Phase 3 parity differences (36 vs 37)
    - Output PASS / WARN / FAIL with details
  - Add `api-compare` target to Makefile
  - Document usage in script header

  **Must NOT do**:
  - Do not fail on minor field ordering differences
  - Do not fail on accepted Phase 3 parity differences

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
  - Reason: Python script, well-defined comparison logic

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Task 18)
  - **Parallel Group**: Wave 4
  - **Blocks**: Wave FINAL
  - **Blocked By**: Tasks 8-16 (all endpoints must exist)

  **References**:
  - `migration_baseline/api_responses/` — baseline snapshots
  - `scripts/migration/` — existing scripts for reference
  - `Makefile` — add target

  **Acceptance Criteria**:
  - [ ] Script exists and is executable
  - [ ] `make api-compare` runs successfully
  - [ ] Script returns PASS or accepted WARN
  - [ ] Script handles missing baseline files gracefully

  **QA Scenarios**:
  ```
  Scenario: API comparison script
    Tool: Bash
    Preconditions: API server running with data
    Steps:
      1. make api-compare
    Expected Result: Exit code 0, output shows PASS or accepted WARN
    Evidence: .sisyphus/evidence/task-17-api-compare.txt
  ```

  **Commit**: YES
  - Message: `chore: add api baseline comparison`
  - Files: `scripts/migration/compare_api_baseline.py`, `Makefile`

---

- [x] 18. **Register All Routes in Server + Integration Tests**

  **What to do**:
  - Modify `internal/api/server.go` (or `internal/api/routes.go`):
    - Register all new endpoints with correct middleware
    - Ensure auth middleware applied to protected routes
    - Ensure public routes (health) skip auth
    - Mount CORS, error, request_id middleware in correct order
  - Create integration tests:
    - Test all 11 endpoints with testcontainers PostgreSQL
    - Test auth failure scenarios
    - Test CORS preflight
    - Test pagination edge cases
    - Test empty database states
  - Ensure `go test ./...` passes

  **Must NOT do**:
  - Do not change existing middleware stack behavior
  - Do not add new routes beyond the 11 planned

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
  - Reason: Integration wiring, multiple moving parts

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Task 17)
  - **Parallel Group**: Wave 4
  - **Blocks**: Wave FINAL
  - **Blocked By**: Tasks 8-16 (all endpoints)

  **References**:
  - `internal/api/server.go` — existing server setup
  - `internal/testutil/db.go` — testcontainers PostgreSQL for tests

  **Acceptance Criteria**:
  - [ ] All 11 routes registered and accessible
  - [ ] Auth middleware applied correctly
  - [ ] CORS middleware applied correctly
  - [ ] Integration tests pass
  - [ ] `go test ./...` passes

  **QA Scenarios**:
  ```
  Scenario: Full integration test
    Tool: Bash
    Preconditions: PostgreSQL running with migrations and data
    Steps:
      1. go test ./... -v
    Expected Result: All tests pass
    Evidence: .sisyphus/evidence/task-18-integration-test.txt
  ```

  **Commit**: YES
  - Message: `feat: register all routes and add integration tests`
  - Files: `internal/api/server.go`, `internal/api/routes.go`, `internal/api/integration_test.go`

---

## Final Verification Wave (MANDATORY — after ALL implementation tasks)

> 4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.

- [x] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, curl endpoint, run command). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist in .sisyphus/evidence/. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [x] F2. **Code Quality Review** — `unspecified-high`
  Run `go test ./...`, `go vet ./...`, check for `as any` or `@ts-ignore` (N/A for Go), check for unused imports, check for AI slop (excessive comments, over-abstraction, generic names). Verify no new module dependencies added.
  Output: `Tests [PASS/FAIL] | Vet [PASS/FAIL] | Files [N clean/N issues] | VERDICT`

- [x] F3. **Real Manual QA** — `unspecified-high`
  Start from clean state. Execute EVERY QA scenario from EVERY task — follow exact steps, capture evidence. Test cross-task integration (features working together, not isolation). Test edge cases: empty state, invalid input, auth failures.
  Save to `.sisyphus/evidence/final-qa/`.
  Output: `Scenarios [N/N pass] | Integration [N/N] | Edge Cases [N tested] | VERDICT`

- [x] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual diff (git log/diff). Verify 1:1 — everything in spec was built (no missing), nothing beyond spec was built (no creep). Check "Must NOT do" compliance. Detect cross-task contamination: Task N touching Task M's files. Flag unaccounted changes.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

### Commit Grouping (6 commits as requested by user)

1. **docs: add phase 4 api migration plan**
   - `docs/migration/phase-4-api-migration-plan.md`

2. **feat: add api middleware and response foundation**
   - `internal/config/config.go` (API_BEARER_TOKEN, CORS_ALLOWED_ORIGINS)
   - `.env.example`
   - `internal/api/middleware/auth.go`
   - `internal/api/middleware/error.go`
   - `internal/api/middleware/request_id.go`
   - `internal/api/middleware/cors.go`
   - `internal/api/response.go`
   - `internal/api/pagination.go`

3. **feat: add health, status and alerts api**
   - `internal/api/health.go` (fixed)
   - `internal/api/dto/status.go`, `internal/api/dto/alert.go`
   - `internal/repository/status_repository.go`, `internal/repository/alert_repository.go`
   - `internal/service/status_service.go`, `internal/service/alert_service.go`
   - `internal/api/handler/status.go`, `internal/api/handler/alerts.go`
   - Tests for all above

4. **feat: add tasks, outbox and logs api**
   - `internal/api/dto/task.go`, `internal/api/dto/outbox.go`, `internal/api/dto/logs.go`
   - `internal/repository/task_repository.go`, `internal/repository/outbox_repository.go`, `internal/repository/log_repository.go`
   - `internal/service/task_service.go`, `internal/service/outbox_service.go`, `internal/service/log_service.go`
   - `internal/api/handler/tasks.go`, `internal/api/handler/outbox.go`, `internal/api/handler/logs.go`
   - Tests for all above

5. **feat: add governance and qoder read api**
   - `internal/api/dto/governance.go`, `internal/api/dto/qoder.go`
   - `internal/repository/governance_repository.go`
   - `internal/service/governance_service.go`, `internal/service/qoder_service.go`
   - `internal/api/handler/governance.go`, `internal/api/handler/qoder.go`
   - Tests for all above

6. **chore: add api baseline comparison and integration**
   - `scripts/migration/compare_api_baseline.py`
   - `Makefile` (api-compare target)
   - `internal/api/server.go` (route registration)
   - `internal/api/integration_test.go`

---

## Success Criteria

### Verification Commands
```bash
# Start PostgreSQL
docker compose up -d postgres

# Set env
export DATABASE_URL="postgres://baxi:baxi_dev@localhost:5432/baxi?sslmode=disable"
export API_BEARER_TOKEN="dev_token_32_chars_minimum_length"

# Run migrations
make migrate

# Run pipeline to populate data
make pipeline DATA_DIR=./data/raw

# Verify pipeline still works
make pipeline-compare

# Start API
make api

# Test endpoints
curl http://localhost:8080/api/v1/health
curl -H "Authorization: Bearer $API_BEARER_TOKEN" http://localhost:8080/api/v1/status
curl -H "Authorization: Bearer $API_BEARER_TOKEN" http://localhost:8080/api/v1/alerts
curl -H "Authorization: Bearer $API_BEARER_TOKEN" http://localhost:8080/api/v1/tasks
curl -H "Authorization: Bearer $API_BEARER_TOKEN" http://localhost:8080/api/v1/outbox
curl -H "Authorization: Bearer $API_BEARER_TOKEN" http://localhost:8080/api/v1/logs/recent
curl -H "Authorization: Bearer $API_BEARER_TOKEN" http://localhost:8080/api/v1/logs/errors
curl -H "Authorization: Bearer $API_BEARER_TOKEN" http://localhost:8080/api/v1/logs/audit
curl -H "Authorization: Bearer $API_BEARER_TOKEN" http://localhost:8080/api/v1/governance/status
curl -H "Authorization: Bearer $API_BEARER_TOKEN" http://localhost:8080/api/v1/qoder/capabilities
curl -H "Authorization: Bearer $API_BEARER_TOKEN" http://localhost:8080/api/v1/qoder/context

# Run baseline comparison
make api-compare

# Run Go tests
go test ./...

# Verify no old code modified
git diff --name-only | grep -E "^(api/|web/|pipeline/)" && echo "WARNING: Old code modified" || echo "OK"
```

### Final Checklist
- [ ] All 11 endpoints return 200 with valid Bearer token
- [ ] `/health` works without auth
- [ ] All protected endpoints return 401 without token
- [ ] `make api-compare` returns PASS or accepted WARN
- [ ] `make pipeline-compare` still returns PASS with accepted WARN
- [ ] `go test ./...` passes
- [ ] CORS preflight works for configured origins
- [ ] request_id propagated in response headers
- [ ] No modifications to old Python/React/Pipeline code
- [ ] No LLM integration
- [ ] No write endpoints implemented
- [ ] Documentation complete


