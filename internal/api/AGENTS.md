# Go HTTP API

**Generated:** 2026-05-28 15:45
**Commit:** d908f6d
**Branch:** main

## OVERVIEW
chi HTTP server on :8080, 14 handlers, 6 middlewares, 10 DTO types, serves all business endpoints.

## HANDLERS
- `action`, `agent_logs`, `alerts`, `decision`, `diagnosis`, `feishu`, `governance`, `llm`, `logs`, `outbox`, `pipeline`, `qoder`, `review`, `status`
- Each exposes interface (e.g. `AlertLister`) for mock-injectable testing
- Lazy initialization in `handler_factories.go` (`server.go` is lifecycle-only after refactoring): `initXxxHandler()` called per route group

## MIDDLEWARE
- **auth**: bearer token via `ConstantTimeCompare`, min 32 chars, known-weak-token rejection set
- **CORS**: comma-separated origins, no wildcard
- **error**: 5-field JSON (`request_id`, `error_code`, `message`, `diagnosis`, `suggested_action`) + panic recovery
- **request_id**: propagate or generate `req_<ts>_<8chr>`

## RESPONSE FORMAT
- `PaginatedResponse[T]` with `httputil.PaginationMeta` (limit, offset, total)
- Error responses follow pre-existing error format for frontend compatibility

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| MCP agent query | `handler/agent_logs.go` | Exposed via MCP: Pi Agent queries activity logs through this handler |

## ANTI-PATTERNS
- `httputil.ParsePagination` returns hardcoded default (limit=100) — callers can't distinguish "not provided" from "default"
- ~~CORS middleware uses string split on comma — no trimming, fragile on whitespace~~ ✅ Fixed in `parseOrigins()` with `strings.TrimSpace`
