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
- **Package naming**: `internal/config` (struct) vs `internal/configloader` (parser) — adjacent but not cohesive.

### ✅ Resolved Anti-Patterns

- **`internal/repository/` flat package**: Now organized into 9 domain subpackages with clean separation between interface and implementation.
- **`pool` passed as parameter everywhere**: `PoolProvider` interface now injected across all subpackages — standardized, mockable, and no more raw `pgxpool.Pool` passing.

<!-- cmd/ deep-dive -->

## cmd/

Three Go entry points. baxi-api and baxi-worker are thin main.go wrappers. baxi-cli is the outlier — 6 files all declaring `package main` with ~820 lines of subcommand logic that belongs in `internal/cli/`.

### WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| API bootstrap | `cmd/baxi-api/main.go` | Thin wrapper, ~74 lines |
| Worker bootstrap | `cmd/baxi-worker/main.go` | Two workers (dispatch + main), ~111 lines |
| CLI pipeline commands | `cmd/baxi-cli/pipeline.go` | run/validate + step definitions + baseline validation |
| CLI governance commands | `cmd/baxi-cli/governance.go` | load/check YAML configs into database |
| CLI decision commands | `cmd/baxi-cli/decision.go` | 7 subcommands, ~295 lines, heavy logic |
| CLI HTTP helpers | `cmd/baxi-cli/client.go` | apiGet/apiPost/auth used by llm + decision subcommands |

### ANTI-PATTERNS

- **CLI logic in package main**: baxi-cli's 6 files (~820 lines) keep subcommand logic in `package main` instead of delegating to `internal/cli/`. Only baxi-api and baxi-worker follow the thin-main.go pattern.
- **HTTP client in cmd/**: `cmd/baxi-cli/client.go` defines shared API call helpers (apiGet, apiPost, auth) inside the entry point instead of a reusable internal package.
- **Decision subcommands hit live API**: `compare`, `replay`, and `evals` in decision.go make HTTP calls to the baxi-api server rather than importing internal packages directly.
- **Dead subcommand**: `cmd/baxi-cli/llm.go` registers `llm status/metrics` handlers, but main.go only dispatches pipeline/governance/decision — llm is unreachable.

<!-- test/ deep-dive -->

## test/

Root-level E2E test suite (3 subdirs). Runs separately in CI, not under `go test ./...`. Spins postgres:15-alpine via testcontainers.

### WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Integration E2E | `test/integration/phase7_test.go` | ~485 lines, full pipeline+governance workflow |
| Schema contract | `test/migration/contract_test.go` | ~503 lines, goose schema vs repo struct alignment |
| Security E2E | `test/security/phase7_test.go` | ~316 lines, auth/RBAC contract tests |

### CONVENTIONS

- **External packages**: Each subdir is its own package (`integration`, `migration`, `security`) — these are outside `internal/`, so they import `baxi/internal/*` by full module path
- **Build constraint**: All files start with `//go:build integration` — requires `-tags=integration` to compile, so `go test ./...` silently skips them (what it does is skip them, not break on the import isolation issue)
- **Testcontainers**: `testutil.StartPostgres()` from `baxi/internal/testutil` manages container lifecycle (postgres:15-alpine, pgxpool connection)
- **CI isolation**: CI invokes these via `go test -tags integration ./test/...` with Docker running

### ANTI-PATTERNS

- **Fragile import path**: Imports `baxi/internal/*` by name from outside the module tree — any internal package rename or relocation breaks E2E tests silently
- **Duplicated helper**: All 3 files reimplement the same `migrationsDir()` directory-walking function instead of sharing a common helper

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
