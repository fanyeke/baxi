# INTERNAL: Go Backend Core

**Generated:** 2026-05-28 15:45
**Commit:** d908f6d
**Branch:** main

## OVERVIEW

Core Go backend: 44 packages, chi router, pgx/PostgreSQL, zap logger.

## STRUCTURE

**API/HTTP**: `api/server.go` (chi routes, lifecycle), `api/handler/` (13 handlers), `api/middleware/` (auth/CORS/request-id/error), `api/dto/` (request/response types), `httputil/` (JSON response, pagination).

**MCP Server**: `mcp/` (14 files, 31 tools over 11 domains), `cmd/baxi-mcp/` (server bootstrap)

**Pipeline**: `pipeline/` (Step interface + Runner), `pipeline/steps/` (7 impls: ingest_raw → build_dwd → build_metrics → detect_alerts → generate_recommendations → generate_tasks → create_outbox), `ingest/` (CSV loader).

**Business services**: `service/` (8 orchestrators), `decision/` (case engine + context builder), `action/` (registry + proposal + apply + executor), `review/` (domain + approval flow), `recommendation/` (generator), `alert/` (dimensional engine + rules), `governance/` (policy/classification/lineage/redaction), `worker/` (dispatch).

**Data access**: `repository/` (interfaces + pgx impls, 12 repository subpackages), `outbox/` (write repo), `db/` (pool creation).

**Shared types**: `model/` (domain types shared across service, handler, and repository layers).

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
| Connect AI Agent (Pi) | `mcp/` + `cmd/baxi-mcp/main.go` | MCP server via stdio |

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
- ~~golangci-lint config exists (.golangci.yml) but Makefile lint target ignores it~~ ✅ Fixed: Makefile lint target now uses golangci-lint with go vet fallback
- Repository organized into 12 domain subpackages with PoolProvider injection
- pool passed as parameter in interfaces.go and flat compatibility files (subpackages use PoolProvider)
- Package naming: `config` (struct) vs `configloader` (parser) — adjacent but not cohesive
