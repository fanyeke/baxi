# Codebase Concerns

**Analysis Date:** 2026-06-03

## Tech Debt

### Python/SQLite Migration Remnants
- **Issue:** `internal/service/pipeline_service.go` still references Python scripts (`python3 scripts/run_daily_pipeline.py`, `python3 scripts/run_full_pipeline.py`) and mentions SQLite (`"~2 minutes (DB mode, all data via SQLite)"`). The `api-compare` Makefile target calls `python3 scripts/migration/compare_api_baseline.py`. The `migration_baseline/` directory contains `sqlite_schema.sql` and Python migration scripts in `scripts/migration/`.
- **Files:** `internal/service/pipeline_service.go`, `Makefile`, `migration_baseline/sqlite_schema.sql`, `scripts/migration/*.py`
- **Impact:** Pipeline preview shows obsolete commands. Makefile depends on Python for verification. Migration artifacts bloat the repo.
- **Fix approach:** Update pipeline registry to Go commands. Remove or archive migration baseline. Convert `api-compare` to Go or remove.

### Deprecated Repository Compatibility Layer
- **Issue:** Flat `*_repository.go` files in `internal/repository/` (e.g., `governance_repository.go`, `decision_repository.go`, `ontology_repository.go`, `outbox_repository.go`, `log_repository.go`, `context_repository.go`) are marked DEPRECATED and act as compatibility shims delegating to new subpackages. They still pass `*pgxpool.Pool` as a parameter instead of using `PoolProvider`.
- **Files:** `internal/repository/governance_repository.go`, `internal/repository/decision_repository.go`, `internal/repository/ontology_repository.go`, `internal/repository/outbox_repository.go`, `internal/repository/log_repository.go`, `internal/repository/context_repository.go`
- **Impact:** Dual APIs confuse callers. The old interfaces still accept `pool` parameters, undermining the `PoolProvider` abstraction.
- **Fix approach:** Migrate all callers to subpackage repositories and delete the shim files.

### Dead/Unreachable Code
- **Issue:** `cmd/baxi-cli/llm.go` registers `llm status` and `llm metrics` subcommands, but `cmd/baxi-cli/main.go` only dispatches `pipeline`, `governance`, and `decision`. The `llm` subcommand is unreachable dead code. `internal/worker/worker.go` is a placeholder that only pings the DB and blocks — all real logic moved to `dispatch_worker.go`.
- **Files:** `cmd/baxi-cli/llm.go`, `cmd/baxi-cli/main.go`, `internal/worker/worker.go`
- **Impact:** Maintenance overhead, misleading entry points.
- **Fix approach:** Remove `llm.go` or wire it into main.go dispatch. Remove `worker.go`.

### Stubs Returning 501 Not Implemented
- **Issue:** 6 HTTP handlers in `decision.go` and 1 in `outbox.go` return `501 Not Implemented` for real API endpoints: `DecideLLM`, `Compare`, `Replay`, `ListLLMDecisions`, `ListEvals`, `HandleBatchDispatch`.
- **Files:** `internal/api/handler/decision.go:267-288`, `internal/api/handler/outbox.go:192-194`
- **Impact:** Frontend or CLI clients calling these endpoints receive unhelpful 501 errors.
- **Fix approach:** Implement or remove the endpoints. If intentionally deferred, document in API spec.

### Missing Goose Migration Sequence Numbers
- **Issue:** The migrations directory jumps from `014_add_outbox_next_retry_at.sql` to `016_config_versions.sql` (missing `015`) and from `024_context_hash_proposal.sql` to `026_action_outcome.sql` (missing `025`).
- **Files:** `migrations/`
- **Impact:** Missing sequence numbers suggest migrations were deleted or skipped. Risk of schema drift if deleted migrations contained necessary changes.
- **Fix approach:** Audit git history for deleted migrations. If intentionally removed, document why. Otherwise recreate or renumber.

### CLI Logic Trapped in package main
- **Issue:** `cmd/baxi-cli/` contains ~919 lines of subcommand logic across 6 files (`pipeline.go`, `governance.go`, `decision.go`, `llm.go`, `client.go`) all declaring `package main`. HTTP client helpers (`apiGet`, `apiPost`, `setAuth`) live in the entry point instead of a reusable internal package.
- **Files:** `cmd/baxi-cli/*.go`
- **Impact:** Cannot unit test CLI logic. Cannot reuse the HTTP client elsewhere.
- **Fix approach:** Move subcommand logic to `internal/cli/` and the HTTP client to `internal/api/client/`.

### Generic 500 Errors Everywhere
- **Issue:** Handlers return generic `"internal server error"` for all service errors. 78 occurrences of `StatusInternalServerError` across handler files. Original error details are swallowed, making debugging impossible.
- **Files:** `internal/api/handler/decision.go`, `internal/api/handler/alerts.go`, `internal/api/handler/status.go`, `internal/api/handler/diagnosis.go`, `internal/api/handler/handler_sandbox.go`, and others.
- **Impact:** Operators and clients cannot distinguish between DB failures, validation errors, or downstream service issues.
- **Fix approach:** Map service errors to appropriate HTTP status codes (400, 404, 409, 502) and include structured error details.

## Known Bugs

### JSON Decode Errors Silently Ignored
- **Issue:** `internal/api/handler/action.go:65` ignores `json.NewDecoder(r.Body).Decode(&req)` errors with `_ =`. If the client sends malformed JSON, the handler proceeds with default values (`dryRun = true`) instead of returning 400.
- **Files:** `internal/api/handler/action.go:65`
- **Trigger:** POST malformed JSON to `/api/v1/proposals/{id}/execute`.
- **Workaround:** None — the request proceeds with safe defaults but hides the error.

### Pipeline Preview References Deleted Python Scripts
- **Issue:** `PreviewPipelineRun` returns commands like `python3 scripts/run_daily_pipeline.py`, but the Go pipeline is invoked via `go run ./cmd/baxi-cli pipeline run`. The preview is misleading.
- **Files:** `internal/service/pipeline_service.go:37`, `internal/service/pipeline_service.go:77`
- **Trigger:** Call the pipeline preview API.
- **Workaround:** Manually construct the correct Go command.

### Missing Error Wrapping in Alert Engine
- **Issue:** `internal/alert/engine.go:174`, `241`, `303` ignore `json.Marshal` errors with `evJSON, _ := json.Marshal(evidence)`. If evidence cannot be marshaled, empty evidence is stored silently.
- **Files:** `internal/alert/engine.go`
- **Trigger:** Evidence map contains unmarshalable values (e.g., cyclic references, `chan` types).
- **Workaround:** None — silently loses data.

### Feishu Page Token Type Assertion Ignores Failure
- **Issue:** `internal/feishu/client.go:124` uses `pageToken, _ = data["page_token"].(string)` — if the assertion fails, `pageToken` becomes empty string, breaking pagination silently.
- **Files:** `internal/feishu/client.go:124`
- **Trigger:** Feishu API returns non-string page_token.

## Security Considerations

### SQL Injection Risk in Dynamic Table Names
- **Issue:** `internal/repository/ontology/repository.go` uses `fmt.Sprintf` with table names derived from `objectTableMap` (e.g., `dwd.order_level`). While `sanitizeIdent` wraps column names with `pgx.Identifier.Sanitize()`, the `tableMapping.fullTableName()` method concatenates schema and table without sanitization in some paths. The V2 compiler (`compiler.go`) does sanitize schema/table via `pgx.Identifier`, but the V1 fallback path does not consistently validate that the mapped table exists before querying.
- **Files:** `internal/repository/ontology/repository.go:370-376`, `internal/repository/ontology/compiler.go:76-191`
- **Current mitigation:** `sanitizeIdent` is used for column names and primary keys.
- **Recommendations:** Always sanitize schema.table in V1 paths. Add an allowlist check that the object type exists in `objectTableMap` before building any query.

### Auth Middleware Only Supports Single Bearer Token
- **Issue:** `internal/api/middleware/auth.go` validates against a single `API_BEARER_TOKEN` env var. There is no user database, no token rotation, no JWT (despite `golang-jwt/jwt` being in go.mod), and no rate limiting per actor.
- **Files:** `internal/api/middleware/auth.go`
- **Current mitigation:** Weak token rejection, minimum 32-char length, constant-time comparison.
- **Recommendations:** Add token rotation support or implement JWT with proper claims and expiry.

### CORS Origin Check Does Not Validate Scheme
- **Issue:** `internal/api/middleware/cors.go` checks `Origin` header against a comma-separated list but does not validate the scheme (`http://` vs `https://`). A configured origin of `example.com` would match both `http://example.com` and `https://example.com`.
- **Files:** `internal/api/middleware/cors.go`
- **Current mitigation:** Origins must be explicitly listed.
- **Recommendations:** Parse and compare scheme + host + port explicitly.

### Action Handler Defaults to Dry Run but Proceeds on Decode Failure
- **Issue:** As noted above, malformed JSON in action execution requests causes the handler to default to `dryRun=true`. While this is safe, it hides client errors and could confuse automation.
- **Files:** `internal/api/handler/action.go:64-71`

### Docker Compose Exposes Postgres Without SSL
- **Issue:** `docker-compose.yml` sets `sslmode=disable` and binds postgres to `127.0.0.1:5432`. The compose file also hardcodes credentials (`baxi_dev`).
- **Files:** `docker-compose.yml`
- **Current mitigation:** Only localhost binding.
- **Recommendations:** Use env files for credentials. Document that production must use TLS.

## Performance Bottlenecks

### Pipeline Runner Has No Idempotency Keys
- **Issue:** `internal/pipeline/runner.go` re-runs steps on retry without idempotency keys, risking duplicate row inserts in `raw.*` and `dwd.*` tables.
- **Files:** `internal/pipeline/runner.go`
- **Cause:** Step interface does not receive a run-scoped idempotency key.
- **Improvement path:** Add `RunID` as an idempotency key to each step. Make INSERT statements idempotent with `ON CONFLICT`.

### Feishu Client Synchronous Pagination
- **Issue:** `internal/feishu/client.go` fetches records page-by-page in a single goroutine with `time.Sleep(1 * time.Second)` between pages.
- **Files:** `internal/feishu/client.go:222`
- **Cause:** Sequential page fetching with artificial delay.
- **Improvement path:** Batch or parallelize page fetching. Reduce/remove sleep if rate limits are handled via backoff.

### Ontology V1 Fallback Mutex Contention
- **Issue:** `internal/repository/ontology/repository.go:277` uses `sync.Mutex` for V1 fallback tracking. All object queries contend on this mutex when V2 compiler fails.
- **Files:** `internal/repository/ontology/repository.go:277`
- **Cause:** Global mutex for fallback metrics.
- **Improvement path:** Use `sync.RWMutex` or atomic counters.

### Alert Engine Queries Same Metrics Repeatedly
- **Issue:** `evaluateLateDeliverySpike` and similar functions query `order_count` separately even though `queryMetricSeries` may have already queried related data.
- **Files:** `internal/alert/engine.go`
- **Cause:** Each rule makes independent SQL queries.
- **Improvement path:** Cache metric series per evaluation batch.

## Fragile Areas

### E2E Test Fragility (test/ Outside Module)
- **Issue:** The `test/` directory is outside `internal/` and imports `baxi/internal/*` by full module path. Any internal package rename or relocation breaks E2E tests. Three test files (`test/integration/phase7_test.go`, `test/migration/contract_test.go`, `test/security/phase7_test.go`) all duplicate the same `migrationsDir()` directory-walking function.
- **Files:** `test/integration/phase7_test.go`, `test/migration/contract_test.go`, `test/security/phase7_test.go`
- **Why fragile:** Imports by module path are brittle. Directory walking assumes CWD depth.
- **Safe modification:** Only add new test cases; do not rename internal packages without updating all three files.
- **Test coverage:** E2E tests run only with `-tags=integration` and Docker. `go test ./...` skips them silently.

### Decision Engine Swallows DB Errors
- **Issue:** `internal/decision/engine.go` ignores `saveDecision` and `updateCaseStatus` errors during fallback paths with `_ = e.saveDecision(...)`. If the DB is down, the engine returns a fallback decision to the caller without persisting anything, making the system appear healthy when it is not.
- **Files:** `internal/decision/engine.go:156-157`, `229-230`
- **Why fragile:** Silent data loss on DB failures.
- **Safe modification:** Do not ignore persistence errors in fallback paths — return them to the caller.

### Context.Background() in Handler Factories
- **Issue:** `internal/api/handler_factories.go:228` uses `context.Background()` when creating `ObjectRegistry` during lazy handler initialization. This bypasses request-scoped deadlines and cancellation.
- **Files:** `internal/api/handler_factories.go:228`
- **Why fragile:** Registry initialization can hang indefinitely if YAML files or DB are slow.
- **Safe modification:** Pass a timeout context or initialize at startup, not on first request.

### Dispatch Worker Ignores Transaction Rollback Errors
- **Issue:** `internal/worker/dispatch_worker.go` uses `defer tx.Rollback(ctx) //nolint:errcheck` in multiple places. If rollback fails (e.g., connection lost), the worker continues as if cleanup succeeded.
- **Files:** `internal/worker/dispatch_worker.go:365`, `386`, `408`, `429`
- **Why fragile:** Failed rollbacks can leave connections in aborted state.
- **Safe modification:** Check rollback errors or rely on connection pool cleanup.

## Scaling Limits

### Single Bearer Token = No Multi-Tenant Auth
- **Issue:** The auth system supports only one global bearer token. There is no per-user or per-tenant token. All API clients share the same identity.
- **Current capacity:** 1 token.
- **Limit:** Cannot support multiple users, service accounts, or key rotation.
- **Scaling path:** Implement JWT or API key table with per-key permissions.

### Pipeline Runner Single-Threaded
- **Issue:** `internal/pipeline/runner.go` executes steps sequentially in a single goroutine with a hardcoded 30-minute timeout.
- **Current capacity:** 1 run at a time per process.
- **Limit:** Long-running steps block all subsequent steps. No parallel step execution.
- **Scaling path:** Implement a DAG executor with parallel branches.

### Alert Suppression Hardcoded at 50
- **Issue:** `internal/alert/engine.go` caps suppressed alerts at 50 per run. No configuration to raise this limit.
- **Current capacity:** 50 alerts/run.
- **Limit:** High-volume days may drop alerts silently.
- **Scaling path:** Make suppression limit configurable per rule.

## Dependencies at Risk

### pgx v5.5.5 (PostgreSQL Driver)
- **Risk:** v5.5.5 is not the latest. Newer versions contain connection pool fixes and prepared statement improvements.
- **Impact:** Potential connection leaks under high load.
- **Migration plan:** Upgrade to pgx v5.6.x or later. Test with integration suite.

### testcontainers-go v0.35.0
- **Risk:** Testcontainers is heavy and requires Docker. Tests fail if Docker is unavailable. The module adds 40+ indirect dependencies (Docker client, containerd, etc.).
- **Impact:** Slow test startup, CI complexity, potential Docker socket security issues.
- **Migration plan:** Consider lightweight in-memory PostgreSQL alternatives (e.g., `pgxmock` for unit tests, Docker only for E2E).

### OpenAI Go SDK v1.12.0
- **Risk:** The LLM provider abstraction exists but `internal/llm/openai_provider.go` may lag behind OpenAI API changes.
- **Impact:** New API features unavailable; potential breakage on API deprecation.
- **Migration plan:** Keep SDK updated. Add integration tests against OpenAI sandbox.

## Missing Critical Features

### No golangci-lint Configuration
- **Issue:** The `Makefile` `lint` target runs `go vet ./...` only. There is no `.golangci.yml` and no automated linting in CI.
- **Blocks:** Enforcing consistent code style, catching common bugs (e.g., ignored errors, shadowed variables).
- **Priority:** Medium.

### No Frontend ESLint/Prettier Configuration
- **Issue:** `frontend/package.json` references `eslint` and `prettier` in scripts, but no ESLint config file (`.eslintrc.*` or `eslint.config.*`) exists. The `frontend/AGENTS.md` incorrectly states "No ESLint, no Prettier — TypeScript compiler only" while package.json includes both.
- **Blocks:** Consistent frontend code style.
- **Priority:** Low.

### Missing Unit Tests for QoderService
- **Issue:** `internal/service/qoder_service.go` (393 lines, substantial business logic) has no dedicated `qoder_service_test.go`. It is the only service file without unit tests.
- **Blocks:** Safe refactoring of context query logic.
- **Priority:** Medium.

### No Database Connection Retry Logic
- **Issue:** `cmd/baxi-api/main.go` and other entry points fail immediately if the database is unavailable at startup. There is no backoff retry for initial connection.
- **Blocks:** Resilient deployments in container orchestration where DB may start after the app.
- **Priority:** Medium.

### No Structured Logging in Dispatch Worker
- **Issue:** `internal/worker/dispatch_worker.go` uses `log.Printf` (standard library) instead of the project's `zap` logger. This breaks structured log aggregation and log level filtering.
- **Blocks:** Operational observability.
- **Priority:** Low.

## Test Coverage Gaps

### E2E Tests Skip in Short Mode
- **Issue:** Nearly all integration tests in `internal/review/`, `internal/service/`, and `test/` directories call `t.Skip("skipping integration test in short mode")` or `t.Skip("skipping in short mode")`. This means `go test ./... -short` skips ~40+ test cases, leaving most DB-interacting code untested in standard CI runs.
- **Files:** `internal/review/*_test.go`, `internal/service/*_integration_test.go`, `test/integration/phase7_test.go`
- **Risk:** DB schema changes or query regressions go undetected in fast test runs.
- **Priority:** High.

### Test/ Directory Breaks `go test ./...` Isolation
- **Issue:** `test/` is at the repo root, inside the Go module. Running `go test ./...` includes `test/integration`, `test/migration`, and `test/security` packages. These packages require `-tags=integration` and Docker, so they fail or are skipped in standard runs, but they still compile. If they ever import non-existent packages, `go test ./...` breaks.
- **Files:** `test/`
- **Risk:** `go test ./...` is not a reliable CI command.
- **Priority:** Medium.

### Handler Tests Use Generic 500 Assertions
- **Issue:** Handler tests assert on `StatusInternalServerError` without verifying error codes or messages. This masks regressions in error classification.
- **Files:** `internal/api/handler/*_test.go`
- **Risk:** Error contract changes break clients silently.
- **Priority:** Low.

### No Race Detector in CI
- **Issue:** `Makefile` and CI do not run `go test -race`. The dispatch worker uses `sync.Mutex` and goroutines, and the API server uses lazy handler init with `sync.Mutex`.
- **Files:** `Makefile`, `internal/worker/dispatch_worker.go`, `internal/api/server.go`
- **Risk:** Data races in handler caching or worker batch processing.
- **Priority:** Medium.

### Frontend Test Coverage Unknown
- **Issue:** `frontend/vitest.config.ts` has no coverage configuration. The `frontend/` directory has 51 source files but only 22 test files. `src/hooks/` and `src/lib/` are empty directories.
- **Files:** `frontend/src/hooks/`, `frontend/src/lib/`, `frontend/vitest.config.ts`
- **Risk:** Frontend logic is largely untested.
- **Priority:** Low.

## Additional Concerns

### Package Naming Inconsistency
- **Issue:** `internal/config` (env struct) and `internal/configloader` (YAML parser) are adjacent but not cohesive. The naming does not describe the distinction.
- **Files:** `internal/config/`, `internal/configloader/`
- **Recommendation:** Rename `internal/config` to `internal/env` or merge into `internal/configloader`.

### YAML Config Drift Risk
- **Issue:** 28 YAML config files in `config/` are parsed manually. No automated validation ensures that `config/aip_object_schema_v2.yml` stays in sync with `config/aip_object_schema.yml` or with the database schema.
- **Files:** `config/*.yml`
- **Recommendation:** Add a CI step that validates YAML schemas against Go structs.

### V1/V2 Ontology Parallel Hierarchies
- **Issue:** `ObjectType` (v1) and `ObjectTypeV2` (v2) coexist with overlapping fields. Some code paths fall back from V2 to V1 silently.
- **Files:** `internal/ontology/schema.go`, `internal/ontology/schema_v2.go`, `internal/ontology/registry.go`
- **Recommendation:** Deprecate V1 types and migrate all callers to V2.

---

*Concerns audit: 2026-06-03*
