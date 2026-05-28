# PROJECT KNOWLEDGE BASE

**Generated:** 2026-05-28 15:45
**Commit:** d908f6d
**Branch:** main

## OVERVIEW

Multi-language governance + analytics platform (Go pipeline backend, Python FastAPI gateway, React frontend). Migrates from Python/SQLite to Go/PostgreSQL.

## STRUCTURE

```
baxi/
├── cmd/            # Go entry points (baxi-api, baxi-cli, baxi-worker)
├── internal/       # Go core: 23 packages (pipeline, decision, governance, etc.)
├── api/            # Python FastAPI gateway (SQLite, port 8765)
├── services/       # Python business services
├── adapters/       # Python channel adapters (Feishu, GitHub, CLI, Manual)
├── frontend/       # React 19 SPA (Vite, TanStack Query, Radix UI)
├── config/         # YAML governance configs (29 files)
├── migrations/     # Goose SQL migrations (Go → PostgreSQL)
├── sql/            # SQLite schema + Python migrations
├── tests/          # Python pytest suite
├── test/           # Go integration + security E2E tests
├── scripts/        # Python scripts (many frozen/broken, see _FROZEN.md)
├── docs/           # Governance docs + migration plans
└── data/           # Raw CSVs + intermediate data
```

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Pipeline orchestration | `internal/pipeline/` | Go, 13 step files |
| Governance rules | `internal/governance/` + `config/` | Go engine + YAML configs |
| Decision engine | `internal/decision/` | Go case engine + context builder |
| FastAPI routes | `api/routers/` | 11 router files |
| React pages | `frontend/src/pages/` | 7 pages + co-located tests |
| Channel adapters | `adapters/` | Strategy pattern (Feishu, GitHub, CLI, Manual) |
| DB repository layer | `internal/repository/` | Go interfaces + implementations |
| YAML configs | `config/*.yml` | 29 governance/alert/metric configs |

## CONVENTIONS

- **Python**: Ruff (line-length=100, rules: E/F/I/N/W/UP). `import type`-style via py39.
- **TypeScript**: `verbatimModuleSyntax` (requires `import type`), `@/` path alias, permissive unused vars.
- **Go**: chi router, pgx/PostgreSQL, goose migrations, testify for tests. No golangci-lint config.
- **SQLite**: All timestamps TEXT (ISO 8601). PKs TEXT (UUID) except governance tables use INTEGER AUTOINCREMENT.
- **Env vars**: ALL_CAPS_SNAKE_CASE, grouped by domain. `API_BEARER_TOKEN` shared between Go and Python.
- **Docker**: Multi-stage golang:1.23-alpine→alpine, CGO_ENABLED=0, static binaries.
- **Coverage**: Python only (source=api/services/adapters, CI enforces ≥60%). Go/frontend have no coverage config.

## ANTI-PATTERNS (THIS PROJECT)

- **Two test roots**: `tests/` (Python) vs `test/` (Go) — confusing. Go E2E tests in `test/` should use build tags inline.
- **Two migration dirs**: `migrations/` (Go/goose) vs `sql/migrations/` (Python) — uncoordinated.
- **Two API servers**: Python FastAPI (8765) + Go chi (8080) — overlapping roles, frontend only talks to Python.
- **Flat Python packages**: `api/`, `services/`, `adapters/` are top-level (no `baxi.` namespace).
- **Committed Go binaries**: `baxi-api`, `baxi-cli`, `baxi-worker` in git — should be in `.gitignore`.
- **CI only tests Python**: Go and frontend tests never run in CI.
- **Frozen scripts**: 14 `phaseXX_*.py` scripts in `scripts/` have broken hardcoded paths.
- **f-string SQL**: 19 locations in Python services/scripts — parameterize or whitelist.

## COMMANDS

```bash
make up              # docker compose up postgres
make api             # go run ./cmd/baxi-api
make worker          # go run ./cmd/baxi-worker
make build           # go build both binaries
make pipeline        # Go CLI pipeline run
make migrate         # goose migrations up
make test            # go test ./... (Go only)
pytest               # Python tests (from root)
cd frontend && npm run dev  # React dev server :5173
python3 scripts/run_api.py  # Python API :8765
```
