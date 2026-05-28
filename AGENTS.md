# PROJECT KNOWLEDGE BASE

**Generated:** 2026-05-28 15:45
**Commit:** d908f6d
**Branch:** main

## OVERVIEW

Multi-language governance + analytics platform (Go pipeline backend, Go chi API, React frontend). Migrated from Python/SQLite to Go/PostgreSQL. All Python code has been removed.

## STRUCTURE

```
baxi/
├── cmd/            # Go entry points (baxi-api, baxi-cli, baxi-worker)
├── internal/       # Go core: 23 packages (pipeline, decision, governance, etc.)
├── frontend/       # React 19 SPA (Vite, TanStack Query, Radix UI)
├── config/         # YAML governance configs (29 files)
├── migrations/     # Goose SQL migrations (Go → PostgreSQL)
├── test/           # Go integration + security E2E tests
├── scripts/        # Utility scripts (frozen analysis scripts)
├── docs/           # Governance docs + migration plans
└── data/           # Raw CSVs + intermediate data
```

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Pipeline orchestration | `internal/pipeline/` | Go, 13 step files |
| Governance rules | `internal/governance/` + `config/` | Go engine + YAML configs |
| Decision engine | `internal/decision/` | Go case engine + context builder |
| API handlers | `internal/api/handler/` | 9 Go handler files |
| React pages | `frontend/src/pages/` | 8 pages + co-located tests |
| Channel adapters | `internal/adapter/` | Go strategy pattern (Feishu, GitHub, CLI, Manual) |
| DB repository layer | `internal/repository/` | Go interfaces + implementations |
| YAML configs | `config/*.yml` | 29 governance/alert/metric configs |
| Background workers | `internal/worker/` | Dispatch worker |
| Action execution | `internal/action/` | Registry + proposal + apply |

## CONVENTIONS

- **Go**: chi router, pgx/PostgreSQL, goose migrations, testify for tests. No golangci-lint config.
- **TypeScript**: `verbatimModuleSyntax` (requires `import type`), `@/` path alias, permissive unused vars.
- **Env vars**: ALL_CAPS_SNAKE_CASE, grouped by domain. `API_BEARER_TOKEN` shared between Go services.
- **Docker**: Multi-stage golang:1.23-alpine→alpine, CGO_ENABLED=0, static binaries.

## ANTI-PATTERNS (THIS PROJECT)

- **Two test roots**: `test/` at root inside module vs `internal/` tests — Go E2E tests in `test/` break `go test ./...` isolation.
- **Committed Go binaries**: `baxi-api`, `baxi-cli`, `baxi-worker` in git — should be in `.gitignore`.
- **`test/` outside `internal/`**: E2E tests in root `test/` directory use internal packages by name, fragile to refactoring.
- **No golangci-lint config**: varying style, no lint CI step.
- **Flat `internal/repository/` package**: 17 files mixing interface definitions with pgx implementations.
- **`pool` passed as parameter everywhere**: no DI container or context propagation.
- **Package naming**: `internal/config` (struct) vs `internal/configloader` (parser) — adjacent but not cohesive.

## COMMANDS

```bash
make up              # docker compose up postgres
make api             # go run ./cmd/baxi-api
make worker          # go run ./cmd/baxi-worker
make build           # go build both binaries
make pipeline        # Go CLI pipeline run
make migrate         # goose migrations up
make test            # go test ./... (Go only)
cd frontend && npm run dev  # React dev server :5173
```
