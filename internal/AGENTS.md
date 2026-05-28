# INTERNAL: Go Backend Core

**Generated:** 2026-05-28 15:45
**Commit:** d908f6d
**Branch:** main

## OVERVIEW

Core Go backend: 23 packages, chi router, pgx/PostgreSQL, zap logger.

## STRUCTURE

**API/HTTP**: `api/server.go` (chi routes, lifecycle), `api/handler/` (9 handlers), `api/middleware/` (auth/CORS/request-id/error), `api/dto/` (request/response types), `httputil/` (JSON response, pagination).

**Pipeline**: `pipeline/` (Step interface + Runner), `pipeline/steps/` (7 impls: ingest_raw → build_dwd → build_metrics → detect_alerts → generate_recommendations → generate_tasks → create_outbox), `ingest/` (CSV loader).

**Business services**: `service/` (8 orchestrators), `decision/` (case engine + context builder), `action/` (registry + proposal + apply + executor), `review/` (domain + approval flow), `recommendation/` (generator), `alert/` (dimensional engine + rules), `governance/` (policy/classification/lineage/redaction), `worker/` (dispatch).

**Data access**: `repository/` (interfaces + pgx impls, 8 repos), `outbox/` (write repo), `db/` (pool creation).

**Infrastructure**: `config/` (env struct), `configloader/` (YAML parse/validate), `adapter/` (Feishu/GitHub domain types), `llm/` (provider abstraction + rule provider), `ontology/` (object registry + query), `audit/`, `logger/` (zap init), `testutil/` (testcontainer + fixtures).

## WHERE TO LOOK

| Task | Package |
|------|---------|
| Add API endpoint | `api/handler/` + `api/dto/` + `api/server.go` |
| Add pipeline step | `pipeline/steps/` + implement `pipeline.Step` |
| Tweak decision logic | `decision/` (engine, context_builder) |
| Add governance rule | `governance/` (classification, lineage, access_policy) |
| Add DB query | `repository/` (interface + pgx impl) |
| Add LLM provider | `llm/` (implement DecisionProvider) |
| Fix integration test | `testutil/` (testcontainer + goose) |

## KEY PATTERNS

- **Chi handler**: Handler struct with interface field, `NewXxxHandler(iface)`, methods on `*Handler`. Tests mock the interface.
- **Repository interfaces**: Defined in `repository/interfaces.go`, implemented with `pgxpool.Pool`. Callers pass `pool` explicitly.
- **Pipeline Step**: `Name() string` + `Run(ctx, tx, input) (*Output, error)`. Step gets `pgx.Tx`, runner owns commit/rollback.
- **Lazy handler init**: Server creates handlers on first call, wiring repo → service → handler chain.
- **testcontainers**: `testutil.StartPostgres()` spins postgres:15-alpine, runs goose migrations, returns conn string.
- **YAML registries**: ActionRegistry, configloader parse `config/*.yml` with whitelist enforcement.
- **Local interfaces**: Handler/engine/service packages define own narrow dependency interfaces, not cross-imports.

## ANTI-PATTERNS

- `test/` outside internal/ — E2E tests in root break `go test ./...` isolation
- `cmd/` constructs pipelines directly — should delegate to `internal/pipeline/`
- No golangci-lint config — varying style, no lint CI step
- Flat `repository/` package (17 files) mixes interface definitions with pgx implementations
- `pool` passed as parameter everywhere — no DI container or context propagation
- Package naming: `config` (struct) vs `configloader` (parser) — adjacent but not cohesive
