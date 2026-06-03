# PROJECT KNOWLEDGE BASE

**Generated:** 2026-05-28 15:45
**Commit:** d908f6d
**Branch:** main

## OVERVIEW

Multi-language governance + analytics platform (Go pipeline backend, Go chi API, React frontend). Migrated from Python/SQLite to Go/PostgreSQL. All Python code has been removed.

## STRUCTURE

```
baxi/
├── cmd/            # Go entry points (baxi-api, baxi-cli, baxi-worker, baxi-mcp)
├── internal/       # Go core: 29 packages (pipeline, decision, governance, mcp, etc.)
├── frontend/       # React 19 SPA (Vite, TanStack Query, Radix UI)
├── config/         # YAML governance configs (28 files)
├── migrations/     # Goose SQL migrations (Go → PostgreSQL)
├── test/           # Go integration + security E2E tests
├── scripts/        # Utility scripts (frozen analysis scripts)
├── docs/           # Governance docs + migration plans
├── data/           # Raw CSVs + intermediate data
└── pi-extension/    # Pi Agent TypeScript extensions
```

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Pipeline orchestration | `internal/pipeline/` | Go, 13 step files |
| Governance rules | `internal/governance/` + `config/` | Go engine + YAML configs |
| Decision engine | `internal/decision/` | Go case engine + context builder + lineage |
| API handlers | `internal/api/handler/` | 14 Go handler files |
| React pages | `frontend/src/pages/` | 11 pages + co-located tests |
| Channel adapters | `internal/adapter/` | Go strategy pattern (Feishu, GitHub, CLI, Manual) |
| DB repository layer | `internal/repository/` | Go interfaces + implementations |
| YAML configs | `config/*.yml` | 28 governance/alert/metric configs |
| LLM providers | `internal/llm/` | OpenAI + rule-based fallback |
| Action execution | `internal/action/` | Registry + proposal + apply |
| Shared types | `internal/model/` | Domain types (service layer, not API DTOs) |
| Background workers | `internal/worker/` | Dispatch worker |
| MCP Server + Pi Agent | `internal/mcp/` + `pi-extension/` | Go MCP server (17 tools) + Pi Agent TypeScript extensions |

## CONVENTIONS

- **Go**: chi router, pgx/PostgreSQL, goose migrations, testify for tests. golangci-lint configured.
- **TypeScript**: `verbatimModuleSyntax` (requires `import type`), `@/` path alias, permissive unused vars.
- **Env vars**: ALL_CAPS_SNAKE_CASE, grouped by domain. `API_BEARER_TOKEN` shared between Go services.
- **Docker**: Multi-stage golang:1.23-alpine→alpine, CGO_ENABLED=0, static binaries.

## ANTI-PATTERNS (THIS PROJECT)

- **Two test roots**: `test/` at root inside module vs `internal/` tests — Go E2E tests in `test/` break `go test ./...` isolation.
- **`test/` outside `internal/`**: E2E tests in root `test/` directory use internal packages by name, fragile to refactoring.
- **No golangci-lint config**: varying style, no lint CI step.
- **Package naming**: `internal/config` (struct) vs `internal/configloader` (parser) — adjacent but not cohesive.

### ✅ Resolved Anti-Patterns

- **`internal/repository/` flat package**: Now organized into 10 domain subpackages with clean separation between interface and implementation.
- **`pool` passed as parameter everywhere**: `PoolProvider` interface now injected across all subpackages — standardized, mockable, and no more raw `pgxpool.Pool` passing.
- **Committed Go binaries**: `baxi-api`, `baxi-cli`, `baxi-worker` in git — now excluded via `.gitignore`.

<!-- cmd/ deep-dive -->

## cmd/

Three Go entry points. baxi-api and baxi-worker are thin main.go wrappers. baxi-cli is the outlier — 6 files all declaring `package main` with ~919 lines of subcommand logic that belongs in `internal/cli/`.

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

- **CLI logic in package main**: baxi-cli's 6 files (~919 lines) keep subcommand logic in `package main` instead of delegating to `internal/cli/`. Only baxi-api and baxi-worker follow the thin-main.go pattern.
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

## DOCUMENTATION

All documentation must be updated to reflect the current Go/PostgreSQL architecture.
Python/SQLite references are no longer valid. When adding features or refactoring,
update the affected AGENTS.md files and README.md to match.

- **AGENTS.md**: Update per-package knowledge base when package structure changes
- **README.md**: Update when project structure, counts, or commands change
- **docs/**: Keep pipeline and config docs in sync with Go implementation

### AGENTS.md Hierarchy (15 files)

```
./AGENTS.md                     # Root project knowledge
├── config/AGENTS.md            # YAML governance configs
├── frontend/AGENTS.md          # React 19 SPA
└── internal/
    ├── AGENTS.md               # Go backend core (29+ packages)
    ├── action/AGENTS.md        # Action registry + execution
    ├── adapter/AGENTS.md       # Channel adapters (Feishu/GitHub/CLI/Manual)
    ├── api/AGENTS.md           # chi HTTP API
    ├── decision/AGENTS.md      # Decision engine + case management
    ├── governance/AGENTS.md    # Data governance
    ├── llm/AGENTS.md           # LLM provider abstraction
    ├── model/AGENTS.md         # Shared domain types
    ├── pipeline/AGENTS.md      # Data pipeline (7 steps)
    ├── repository/AGENTS.md    # Repository layer (10 subpackages)
    ├── service/AGENTS.md       # Business orchestration services
    └── mcp/AGENTS.md           # MCP Server (17 tools)
```

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

<!-- GSD:project-start source:PROJECT.md -->

## Project

**Baxi — Governance + Analytics Platform**

Baxi is a multi-language data governance and analytics platform that runs data pipelines, enforces governance rules, and makes LLM-assisted decisions with full audit trails. It connects to external channels (Feishu/Lark, GitHub) for alerting and action execution, and exposes an MCP server for Pi Agent integration.

**Core Value:** A complete, demonstrable closed-loop system: data flows through pipelines, governance rules catch issues, decisions are made with context, actions are executed, and results feed back — all observable through API, CLI, and web frontend.

### Constraints

- **Tech stack**: Fixed — Go 1.23 backend, React 19 frontend, PostgreSQL 15, Docker Compose
- **Timeline**: Demo-ready — focus on completeness and bug fixes, not new features
- **Dependencies**: Feishu/Lark and GitHub integrations must remain functional
- **Compatibility**: MCP server contract must not break Pi Agent integration
- **Security**: Fix SQL injection risks and auth gaps before demo

<!-- GSD:project-end -->

<!-- GSD:stack-start source:codebase/STACK.md -->

## Technology Stack

## Languages

- **Go 1.23** — Backend API, pipeline engine, MCP server, CLI, worker, and all business logic (`internal/`, `cmd/`, `test/`)
- **TypeScript** — React 19 SPA frontend (`frontend/`), Pi Agent extensions (`pi-extension/`)
- **SQL** — Goose migrations (`migrations/`, 21 migration files)
- **YAML** — Governance configs (`config/`, 28 files), docker-compose, GitHub Actions workflows
- **Shell** — Makefile, backup/restore/verification scripts (`scripts/`)

## Runtime

- Go 1.23 (module `baxi`)
- Node.js 20 (frontend CI), local dev uses Vite dev server
- Go modules (`go.mod`, `go.sum`)
- npm (`frontend/package.json`, `frontend/package-lock.json`)
- Lockfile: present for both Go and frontend

## Frameworks

- **chi/v5** `v5.2.5` — HTTP router for Go API (`internal/api/server.go`, `internal/api/routes.go`)
- **React 19** `^19.1.0` — Frontend SPA (`frontend/src/`)
- **Vite 6** `^6.3.5` — Frontend build tool and dev server (`frontend/vite.config.ts`)
- **TanStack Query 5** `^5.72.2` — Async state management (`frontend/src/api/`)
- **Tailwind CSS v4** `^4.1.6` — Utility-first styling (`frontend/vite.config.ts`)
- **Radix UI** — Headless accessible primitives (Dialog, Tabs, etc.)
- **testify** `v1.9.0` — Go assertions and test suites
- **testcontainers-go** `v0.35.0` + `modules/postgres` `v0.35.0` — Integration test database isolation
- **Vitest** `^4.1.7` — Frontend unit testing (`frontend/vitest.config.ts`)
- **Playwright** `^1.60.0` — Frontend E2E testing
- **jsdom** `^29.1.1` — Frontend test environment
- **Testing Library** (`@testing-library/react`, `@testing-library/jest-dom`, `@testing-library/user-event`) — React component testing
- **Goose** `v3.20.0` — SQL migration runner (`Makefile`, `internal/testutil/db.go`)
- **Docker** — Multi-stage builds (`Dockerfile.api`, `Dockerfile.worker`)
- **golangci-lint** — Go linting (`.golangci.yml`)
- **ESLint 10** + `typescript-eslint` + `eslint-plugin-react-hooks` — Frontend linting (`frontend/eslint.config.js`)
- **Prettier** `^3.8.3` — Frontend formatting
- **Zap** `v1.28.0` — Structured logging (`internal/logger/`, `go.uber.org/zap`)

## Key Dependencies

- **pgx/v5** `v5.5.5` — PostgreSQL driver and connection pool (`internal/db/postgres.go`, `internal/repository/`)
- **openai-go** `v1.12.0` — OpenAI-compatible LLM API client (`internal/llm/openai_provider.go`)
- **mcp-go** `v0.41.1` — MCP (Model Context Protocol) server framework (`internal/mcp/`)
- **golang-jwt/jwt/v5** `v5.3.1` — JWT parsing for API auth middleware (`internal/api/middleware/auth.go`)
- **uuid** `v1.6.0` — UUID generation
- **yaml.v3** `v3.0.1` — YAML config parsing (`internal/configloader/`, `config/`)
- **goose/v3** `v3.20.0` — Database migrations
- **testcontainers-go** `v0.35.0` — Docker-based test infrastructure
- **react-router-dom** `^7.6.1` — SPA routing
- **tailwindcss-animate** `^1.0.7` — Tailwind animation utilities
- **lucide-react** — Icon library (referenced in AGENTS.md, not in package.json — verify installed)
- **clsx** / **tailwind-merge** — Conditional class merging (referenced in AGENTS.md conventions)

## Configuration

- All configuration loaded from environment variables (`internal/config/config.go`)
- `.env` file present (not committed, `.env.example` committed as template)
- `frontend/.env` — Vite env vars (`VITE_API_BASE_URL`, `VITE_API_BACKEND`)
- `DATABASE_URL` — PostgreSQL connection string (required)
- `API_BEARER_TOKEN` — API auth token, minimum 32 chars (required)
- `API_PORT` — defaults to 8080
- `LOG_LEVEL` — defaults to `info`
- `CORS_ALLOWED_ORIGINS` — comma-separated, defaults to localhost dev origins
- `go.mod` / `go.sum` — Go dependency management
- `frontend/package.json` — Node dependency management
- `frontend/vite.config.ts` — Vite build config with proxy to `:8080`
- `frontend/tsconfig.json` — TypeScript config (`verbatimModuleSyntax`, `@/` alias)
- `.golangci.yml` — Lint config (18 linters enabled, gocyclo min-complexity 15)
- `docker-compose.yml` — Local orchestration (postgres:16, api, worker)

## Platform Requirements

- Go 1.23+
- Node.js 20+ (frontend)
- Docker & Docker Compose (for postgres and local orchestration)
- PostgreSQL 15+ (local dev via `docker compose up postgres`)
- Make (for Makefile targets)
- Docker multi-stage build: `golang:1.23-alpine` → `alpine:latest`
- CGO_ENABLED=0 static binaries
- PostgreSQL 15/16
- Target port: 8080 (API), stdio (MCP)

<!-- GSD:stack-end -->

<!-- GSD:conventions-start source:CONVENTIONS.md -->

## Conventions

## Naming Patterns

### Go

- Snake_case for test files: `decision_eval_test.go`, `agent_logs_test.go`
- Descriptive suffixes for test variants: `_test.go` (unit), `_integration_test.go` (integration), `_coverage_test.go` (coverage fill), `_extra_test.go` (additional scenarios)
- No `_test` package separation — tests live in the same package as production code
- PascalCase for exported: `NewDecisionService`, `BuildDecisionContext`
- camelCase for unexported: `caseToResponse`, `structToMap`, `writeError`
- Constructor pattern: `NewXxx` prefix for constructors (`NewDecisionHandler`, `NewPoolProvider`)
- Builder pattern for optional dependencies: `WithMetrics`, `WithReplayService`, `WithRuleProvider`
- Short names in tight scopes: `ctx`, `w`, `r`, `err`
- Descriptive names in broader scopes: `decisionCaseID`, `pagination`, `proposals`
- Pointer receivers named after type: `(h *DecisionHandler)`, `(s *DecisionService)`
- PascalCase for all exported types: `DecisionCase`, `ActionProposal`, `LLMSafeContext`
- Interface names use `-er` suffix: `DecisionProvider`, `AlertLister`, `CaseService`, `ContextBuilder`
- Struct suffixes: `Row` for DB row structs (`DecisionCaseRow`, `LLMDecisionRow`)
- Request/Response DTOs: `CreateCaseRequest`, `CaseListResponse`
- PascalCase for exported string constants: `DecisionTypeMonitor`, `SeverityHigh`, `ActionTypeNotifyOwner`
- Grouped by type in const blocks with doc comments

### TypeScript/React

- PascalCase for component files: `Dashboard.tsx`, `CaseDetail.test.tsx`
- camelCase for utility files: `client.ts`, `governance.ts`
- Co-located tests: `PageName.test.tsx` alongside `PageName.tsx`, or in `__tests__/` subdirectory
- PascalCase for component names: `Dashboard`, `ConfirmApplyDialog`
- Hooks: camelCase, no `use` prefix enforced
- camelCase for variables and functions
- ALL_CAPS_SNAKE_CASE for constants (env vars)

## Code Style

### Go

- `gofmt` enforced via golangci-lint (`gofmt` linter enabled)
- `goimports` enforced for import organization
- Max cyclomatic complexity: 15 (`gocyclo` linter, `.golangci.yml` line 21)
- Tool: `golangci-lint` with config in `.golangci.yml`
- Enabled linters: `errcheck`, `gosimple`, `govet`, `ineffassign`, `staticcheck`, `unused`, `gocyclo`, `goimports`, `gofmt`, `revive`
- Test files skip: `gocyclo` and `errcheck` on `*_test.go`
- CI runs `go vet ./...` and `go mod tidy` check (`.github/workflows/go-ci.yml`)

### TypeScript

- Prettier configured in `package.json`: `"format": "prettier --write \"src/**/*.{ts,tsx}\""`
- ESLint config in `frontend/eslint.config.js` using `@eslint/js`, `typescript-eslint`, `eslint-plugin-react-hooks`
- `verbatimModuleSyntax: true` in `tsconfig.json` — requires `import type` for type-only imports
- `react/react-in-jsx-scope: off` (React 19 automatic JSX runtime)
- `@typescript-eslint/no-unused-vars: warn`
- React Hooks rules enabled via plugin

## Import Organization

### Go

### TypeScript

- `@/` maps to `./src/` (configured in `tsconfig.json` and `vite.config.ts`)
- Example: `import { apiClient } from "@/api/client"`
- `import type` required for type-only imports due to `verbatimModuleSyntax`
- Example: `import type { ReactElement } from "react"`

## Error Handling

### Go

- Always wrap with context using `fmt.Errorf("...: %w", err)`
- Repository layer adds domain context: `"query ai.decision_case by id: %w"`, `"scan action_proposal row: %w"`

### TypeScript

## Logging

- JSON encoding to stdout
- ISO8601 timestamps
- Short caller encoding
- Levels: debug, info, warn, error (default: info)
- `log` package used for startup warnings
- `zap.Logger` used for structured application logging (injected via config)

## Comments

### Go

### TypeScript

- No JSDoc convention observed
- Component props typed via interfaces (inferred)

## Function Design

### Go

- `ctx context.Context` as first parameter
- Pointer receivers on structs: `(h *DecisionHandler)`
- Functional options pattern for configuration: `action.WithDryRun(true)`
- `(result, error)` pattern
- Named return values rarely used
- Nil result + error on failure
- Local narrow interfaces defined in consuming packages
- Example: `AlertLister` in `handler/alerts.go`, `DecisionService` in `handler/decision.go`
- Compile-time interface checks:

### TypeScript

## Module Design

### Go

- PascalCase = exported
- No explicit `export` keyword — visibility by case
- Domain-driven subpackages: `internal/repository/decision/`, `internal/repository/alert/`
- Flat package for services: `internal/service/` (all services in one package)
- Handler package: `internal/api/handler/` (all handlers in one package)

### TypeScript

- `frontend/src/components/index.ts` exports all shared components
- `frontend/src/api/governance.ts` exports typed API functions

## Environment Variables

- `DATABASE_URL` — PostgreSQL connection string
- `API_BEARER_TOKEN` — Shared auth token (min 32 chars)
- LLM: `LLM_API_KEY`, `LLM_API_BASE`, `LLM_MODEL`, `LLM_TEMPERATURE`, `LLM_MAX_Tokens`, `LLM_TIMEOUT_SECONDS`, `LLM_ENABLED`, `LLM_PROVIDER`, `LLM_FALLBACK_ENABLED`, `LLM_STORE_RAW_OUTPUT`, `LLM_MAX_RETRIES`
- Worker: `WORKER_BATCH_SIZE`, `WORKER_TICK_INTERVAL`
- Action: `ACTION_APPLY_DRY_RUN`, `FEISHU_WEBHOOK_URL`, `GITHUB_TOKEN`
- CORS: `CORS_ALLOWED_ORIGINS`
- `os.Getenv()` for required, `getEnv(key, defaultValue)` for optional
- Bool parsing: explicit string comparison `v == "true"`
- Numeric parsing with `_` for ignored errors (uses defaults)

## TypeScript-Specific Conventions

- Tailwind CSS v4 with `@tailwindcss/vite` plugin
- `tailwindcss-animate` for animations
- Radix UI primitives for accessible components
- `lucide-react` for icons (convention from AGENTS.md)
- Singleton `apiClient` with `get<T>()` and `post<T>()` methods
- AbortController with configurable timeout (10s default, 120s for Feishu)
- Bearer token from `sessionStorage`

<!-- GSD:conventions-end -->

<!-- GSD:architecture-start source:ARCHITECTURE.md -->

## Architecture

## System Overview

```text

```

## Component Responsibilities

| Component | Responsibility | File |
|-----------|----------------|------|
| API Server | chi HTTP router, handler factories, middleware | `internal/api/server.go` |
| Pipeline Runner | Orchestrates 7 sequential ETL steps with per-step transactions | `internal/pipeline/runner.go` |
| Decision Engine | LLM-driven decisions with validation, repair retry, rule-based fallback | `internal/decision/engine.go` |
| Context Builder | Assembles LLMSafeContext from alerts, governance, ontology | `internal/decision/context_builder_v2.go` |
| Action Registry | Whitelist-enforced action config (4 canonical types) | `internal/action/registry.go` |
| Governance Engine | Classification, lineage, access policy, redaction | `internal/governance/` |
| Alert Engine | Dimensional anomaly detection rules | `internal/alert/engine.go` |
| MCP Server | 31 tools over stdio for Pi Agent integration | `internal/mcp/server.go` |
| Outbox Worker | Polls and dispatches pending events to channel adapters | `internal/worker/dispatch_worker.go` |
| Ontology V2 | AIP semantic object schema, link resolution, action binding | `internal/ontology/` |
| Feishu Adapter | Feishu webhook + OpenAPI integration | `internal/adapter/feishu.go` |
| GitHub Adapter | GitHub issue creation with labels | `internal/adapter/github.go` |

## Pattern Overview

- Interface-driven design: handlers define narrow service interfaces, services define narrow repository interfaces
- Lazy initialization: API server handlers are created on first route access (`handler_factories.go`)
- Per-step transactions: Pipeline runner wraps each step in its own `pgx.Tx`
- Feature flags: `internal/feature/flags.go` gates v2 context builder and ontology behavior
- Config-driven governance: 28 YAML files in `config/` loaded at startup
- Provider pattern: LLM providers (OpenAI, rule-based) are swappable via factory

## Layers

- Purpose: HTTP request handling, routing, middleware, DTO serialization
- Location: `internal/api/`
- Contains: chi router setup (`routes.go`), 14 handlers (`handler/`), 6 middlewares (`middleware/`), DTOs (`dto/`)
- Depends on: service layer, repository layer (for direct reads in adapters)
- Used by: `cmd/baxi-api/main.go`
- Purpose: Business orchestration, composing repositories and domain logic
- Location: `internal/service/`
- Contains: 11 service files (decision, governance, log, alert, pipeline, task, outbox, status, qoder, feishu, agent_log)
- Depends on: repository interfaces, decision engine, action registry
- Used by: API handlers, MCP server
- Purpose: Core business logic, decision generation, pipeline steps, governance rules
- Location: `internal/decision/`, `internal/pipeline/`, `internal/governance/`, `internal/action/`, `internal/alert/`, `internal/review/`, `internal/ontology/`
- Contains: DecisionEngine, ContextBuilder, Pipeline Runner, ActionRegistry, Governance services, Ontology V2
- Depends on: repository layer, LLM providers, config loader
- Used by: Service layer, MCP server, CLI
- Purpose: Data access with pgx/PostgreSQL
- Location: `internal/repository/`
- Contains: 12 domain subpackages + flat compatibility files, interfaces in `interfaces.go`
- Depends on: `internal/db/postgres.go`, `internal/repository/common/PoolProvider`
- Used by: Service layer, domain engines, handlers
- Purpose: Database connections, logging, config loading, adapters
- Location: `internal/db/`, `internal/logger/`, `internal/config/`, `internal/configloader/`, `internal/adapter/`, `internal/llm/`, `internal/testutil/`
- Contains: pgx pool wrapper, zap logger, env config struct, YAML config parser, channel adapters, LLM provider abstraction
- Used by: All layers above
- Purpose: React 19 SPA for console UI
- Location: `frontend/src/`
- Contains: 13 pages, 5 shared components, API client layer, TanStack Query hooks
- Depends on: Go HTTP API at `/api/v1`

## Data Flow

### Primary Request Path (HTTP API)

### Pipeline Execution Path

### Decision Flow (LLM → Action)

### MCP Tool Invocation Path

- No in-memory state across requests — all state in PostgreSQL
- `pgxpool.Pool` is the only shared mutable resource, managed per-process
- Feature flags read from env at startup (`internal/feature/flags.go`)
- ObjectRegistry caches ontology schema in memory with RWMutex

## Key Abstractions

- Purpose: Pipeline step contract
- Definition: `pipeline/step.go:28`
- Pattern: `Name() string` + `Run(ctx, tx, input) (*StepOutput, error)`
- Purpose: Swappable LLM provider (OpenAI, rule-based, disabled)
- Definition: `internal/llm/provider.go`
- Pattern: `GenerateDecision(ctx, LLMSafeContext) (*DecisionOutput, error)`
- Purpose: Channel-agnostic action dispatch
- Definition: `internal/action/executor.go`
- Pattern: `Execute(ctx, ActionProposal, dryRun) (*ExecutionResult, error)`
- Implementations: FeishuAdapter, GitHubAdapter, CLIAdapter, ManualAdapter, NoOpExecutor
- Purpose: Replaces raw `*pgxpool.Pool` passing with injected querier
- Definition: `internal/repository/common/pool.go:25`
- Pattern: Embeddable struct with Query/QueryRow/Exec/Begin methods
- Used by: all repository subpackages
- Purpose: Structured input for LLM decision generation
- Definition: `internal/llm/provider.go`
- Contains: Trigger info, object context, governance info, allowed/forbidden actions, enriched objects
- Purpose: Represents a proposed action from a decision
- Definition: `internal/action/proposal_service.go`
- Lifecycle: draft → pending_review → approved → executing → completed

## Entry Points

- Location: `cmd/baxi-api/main.go`
- Triggers: OS startup, `make api`, Docker container
- Responsibilities: Load config, connect DB, start chi HTTP server on :8080, graceful shutdown
- Location: `cmd/baxi-worker/main.go`
- Triggers: OS startup, `make worker`, Docker container
- Responsibilities: Load config, connect DB, run dispatch worker + background worker, handle signals
- Location: `cmd/baxi-mcp/main.go`
- Triggers: MCP client invocation (stdio), `make mcp`
- Responsibilities: Wire all 13+ dependencies into MCP server, serve stdio protocol
- Location: `cmd/baxi-cli/main.go`
- Triggers: Manual shell invocation
- Responsibilities: Pipeline run/validate, governance load/check, decision subcommands (create/context/decide/compare/replay/evals)
- Anti-pattern: ~919 lines of subcommand logic in package main, should delegate to `internal/cli/`

## Architectural Constraints

- **Threading:** Single-threaded event loop per process. No goroutine pools or worker queues beyond the dispatch worker.
- **Global state:** `feature.LoadFlags()` reads env once at startup. `action.NewActionRegistry("")` parses YAML at init. No mutable package-level vars.
- **Circular imports:** Prevented via local interfaces. `review/service.go` defines `LineageRecorder` interface to avoid importing `decision` package. `action/apply_service.go` uses interface adapter to avoid importing `decision`.
- **Database coupling:** All business logic assumes PostgreSQL 15+ with specific schemas (raw, dwd, mart, ops, gov, ai, audit). No abstraction over SQL dialect.
- **Transaction boundaries:** Pipeline steps receive `pgx.Tx` and must not commit/rollback. Services typically manage their own transactions. Handlers generally do not start transactions.

## Anti-Patterns

### Pool-as-Parameter in Interfaces

### CLI Logic in package main

### V1/V2 Context Builder Parallel Hierarchies

### E2E Tests Outside internal/

## Error Handling

- Domain packages return Go errors with `fmt.Errorf("...: %w", err)` wrapping
- Handler errors use `middleware.BAD_REQUEST`, `middleware.INTERNAL_ERROR` codes
- API error responses: 5-field JSON (`request_id`, `error_code`, `message`, `diagnosis`, `suggested_action`)
- Panic recovery via `apimw.RecoveryMiddleware`
- Pipeline step failures abort the entire run (no partial success)

## Cross-Cutting Concerns

- Framework: zap (`go.uber.org/zap`)
- Pattern: Structured JSON logging. Each step gets a scoped logger with `zap.String("step", name)`.
- Audit logs: Separate `audit.audit_log` table for business events. CSV audit logs for dispatch worker.
- Input validation: Handlers decode JSON and check required fields manually
- Schema validation: `internal/llm/schema_validator.go` validates LLM outputs against action schemas
- Governance validation: `internal/governance/` enforces classification and access policies
- HTTP API: Bearer token via `API_BEARER_TOKEN` env var, constant-time comparison, min 32 chars
- MCP: No auth at transport level; `BAXI_MCP_USER_ID` and `BAXI_MCP_ROLE` env vars set caller identity
- No RBAC enforcement in middleware beyond token validation
- Decision lineage: `decision/lineage_service.go` records all state transitions
- Snapshot recorder: Persists LLM input/output/validation per decision
- Request IDs: `apimw.RequestIDMiddleware` propagates or generates `req_<ts>_<8chr>`

<!-- GSD:architecture-end -->

<!-- GSD:skills-start source:skills/ -->

## Project Skills

| Skill | Description | Path |
|-------|-------------|------|
| ab-test-analysis | "Analyze A/B test results with statistical significance, sample size validation, confidence intervals, and ship/extend/stop recommendations. Use when evaluating experiment results, checking if a test reached significance, interpreting split test data, or deciding whether to ship a variant." | `.claude/skills/ab-test-analysis/SKILL.md` |
| analyze-feature-requests | "Analyze and prioritize a list of feature requests by theme, strategic alignment, impact, effort, and risk. Use when reviewing customer feature requests, triaging a backlog, or making prioritization decisions." | `.claude/skills/analyze-feature-requests/SKILL.md` |
| ansoff-matrix | "Generate an Ansoff Matrix analysis mapping growth strategies across market penetration, market development, product development, and diversification. Use when considering growth options, planning market expansion, or evaluating strategic growth paths." | `.claude/skills/ansoff-matrix/SKILL.md` |
| ckm:banner-design | "Design banners for social media, ads, website heroes, creative assets, and print. Multiple art direction options with AI-generated visuals. Actions: design, create, generate banner. Platforms: Facebook, Twitter/X, LinkedIn, YouTube, Instagram, Google Display, website hero, print. Styles: minimalist, gradient, bold typography, photo-based, illustrated, geometric, retro, glassmorphism, 3D, neon, duotone, editorial, collage. Uses ui-ux-pro-max, frontend-design, ai-artist, ai-multimodal skills." | `.claude/skills/banner-design/SKILL.md` |
| beachhead-segment | "Identify the first beachhead market segment for a product launch. Evaluates segments against burning pain, willingness to pay, winnable market share, and referral potential. Use when choosing a first market, targeting an initial customer segment, or planning market entry strategy." | `.claude/skills/beachhead-segment/SKILL.md` |
| brainstorm-experiments-existing | "Design experiments to test assumptions for an existing product — prototypes, A/B tests, spikes, and other low-effort validation methods. Use when validating assumptions, testing feature ideas cheaply, or planning product experiments." | `.claude/skills/brainstorm-experiments-existing/SKILL.md` |
| brainstorm-experiments-new | "Design lean startup experiments (pretotypes) for a new product. Creates XYZ hypotheses and suggests low-effort validation methods like landing pages, explainer videos, and pre-orders. Use when validating a new product idea, creating pretotypes, or testing market demand." | `.claude/skills/brainstorm-experiments-new/SKILL.md` |
| brainstorm-ideas-existing | "Brainstorm product ideas for an existing product using multi-perspective ideation from PM, Designer, and Engineer viewpoints. Use when generating new feature ideas, brainstorming solutions for an identified opportunity, or ideating with a product trio." | `.claude/skills/brainstorm-ideas-existing/SKILL.md` |
| brainstorm-ideas-new | "Brainstorm feature ideas for a new product in initial discovery from PM, Designer, and Engineer perspectives. Use when starting product discovery for a new product, exploring features for a startup idea, or doing initial ideation." | `.claude/skills/brainstorm-ideas-new/SKILL.md` |
| brainstorm-okrs | "Brainstorm team-level OKRs aligned with company objectives — qualitative objectives with measurable key results. Use when setting quarterly OKRs, aligning team goals with company strategy, drafting objectives, or learning how to write effective OKRs." | `.claude/skills/brainstorm-okrs/SKILL.md` |
| brainstorming | "You MUST use this before any creative work - creating features, building components, adding functionality, or modifying behavior. Explores user intent, requirements and design before implementation." | `.claude/skills/brainstorming/SKILL.md` |
| ckm:brand | Brand voice, visual identity, messaging frameworks, asset management, brand consistency. Activate for branded content, tone of voice, marketing assets, brand compliance, style guides. | `.claude/skills/brand/SKILL.md` |
| business-model | "Generate a Business Model Canvas with all 9 building blocks. Use when creating a business model, documenting how a business creates value, or analyzing an existing business model." | `.claude/skills/business-model/SKILL.md` |
| capture-screen | Programmatic screenshot capture on macOS. Find window IDs with Swift CGWindowListCopyWindowInfo, control application windows via AppleScript (zoom, scroll, select), and capture with screencapture. Use when automating screenshots, capturing application windows for documentation, or building multi-shot visual workflows. | `.claude/skills/capture-screen/SKILL.md` |
| cli-demo-generator | Generates professional animated CLI demos as GIFs using VHS terminal recordings. Handles tape file creation, self-bootstrapping demos with hidden setup, output noise filtering, post-processing speed-up, and frame-level verification. Use when users want to create terminal demos, record CLI workflows as GIFs, generate animated documentation, build demo tapes for README files, or need to showcase any command-line tool visually. Also triggers on "record terminal", "VHS tape", "demo GIF", "animate my CLI", or any request to visually demonstrate shell commands. | `.claude/skills/cli-demo-generator/SKILL.md` |
| cohort-analysis | "Perform cohort analysis on user engagement data — retention curves, feature adoption trends, and segment-level insights. Use when analyzing user retention by cohort, studying feature adoption over time, investigating churn patterns, or identifying engagement trends." | `.claude/skills/cohort-analysis/SKILL.md` |
| competitive-battlecard | "Create sales-ready competitive battlecards comparing your product against a specific competitor — positioning, feature comparison, objection handling, and win/loss patterns. Use when preparing sales teams, creating competitive materials, or responding to 'why not competitor X?'" | `.claude/skills/competitive-battlecard/SKILL.md` |
| competitor-analysis | "Analyze competitors with strengths, weaknesses, and differentiation opportunities. Identifies direct competitors and maps the competitive landscape. Use when doing competitive research, preparing a competitive brief, or finding differentiation opportunities." | `.claude/skills/competitor-analysis/SKILL.md` |
| competitors-analysis | Analyze competitor repositories with evidence-based approach. Use when tracking competitors, creating competitor profiles, or generating competitive analysis. CRITICAL - all analysis must be based on actual cloned code, never assumptions. Triggers include "analyze competitor", "add competitor", "competitive analysis", or "竞品分析". | `.claude/skills/competitors-analysis/SKILL.md` |
| create-prd | "Create a Product Requirements Document using a comprehensive 8-section template covering problem, objectives, segments, value propositions, solution, and release planning. Use when writing a PRD, documenting product requirements, preparing a feature spec, or reviewing an existing PRD." | `.claude/skills/create-prd/SKILL.md` |
| customer-journey-map | "Create an end-to-end customer journey map with stages, touchpoints, emotions, pain points, and opportunities. Use when mapping the customer experience, identifying friction points, improving onboarding, or visualizing the user journey." | `.claude/skills/customer-journey-map/SKILL.md` |
| debugging-network-issues | Evidence-driven investigation for network, streaming, and protocol-layer bugs. Use when debugging connection resets (ECONNRESET, HTTP/2 RST_STREAM, INTERNAL_ERROR), SSE or long-polling stalls, fixed-time connection drops, CDN/proxy/CGNAT idle timeouts, or any incident where symptoms do not match the obvious cause. Applies falsification-first methodology — layered isolation experiments to pin down the responsible network layer, env-gated runtime instrumentation for non-invasive observation, and counter-review agent teams to challenge single-cause assumptions. Strongly trigger on "socket closed unexpectedly", "stream interrupted", "ECONNRESET", "HTTP/2 INTERNAL_ERROR", "fails after N seconds", "works sometimes but not always", "upstream silent for X seconds", or any scenario where the investigator might jump to conclusions before evidence. Generalizes to any multi-layer system investigation where assumption-first thinking is the failure mode. | `.claude/skills/debugging-network-issues/SKILL.md` |
| deep-research | \| Generate format-controlled research reports with evidence tracking, citations, source governance, and multi-pass synthesis. This skill should be used when users request a research report, literature review, market or industry analysis, competitive landscape, policy or technical brief. Triggers: "帮我调研一下", "深度研究", "综述报告", "深入分析", "research this topic", "write a report on", "survey the literature on", "competitive analysis of", "技术选型分析", "竞品研究", "政策分析", "行业报告". V6 adds: source-type governance, AS_OF freshness checks, mandatory counter-review, and citation registry. V6.1 adds: source accessibility (circular verification forbidden, exclusive advantage encouraged). | `.claude/skills/deep-research/SKILL.md` |
| ckm:design | "Comprehensive design skill: brand identity, design tokens, UI styling, logo generation (55 styles, Gemini AI), corporate identity program (50 deliverables, CIP mockups), HTML presentations (Chart.js), banner design (22 styles, social/ads/web/print), icon design (15 styles, SVG, Gemini 3.1 Pro), social photos (HTML→screenshot, multi-platform). Actions: design logo, create CIP, generate mockups, build slides, design banner, generate icon, create social photos, social media images, brand identity, design system. Platforms: Facebook, Twitter, LinkedIn, YouTube, Instagram, Pinterest, TikTok, Threads, Google Ads." | `.claude/skills/design/SKILL.md` |
| ckm:design-system | Token architecture, component specifications, and slide generation. Three-layer tokens (primitive→semantic→component), CSS variables, spacing/typography scales, component specs, strategic slide creation. Use for design tokens, systematic design, brand-compliant presentations. | `.claude/skills/design-system/SKILL.md` |
| dispatching-parallel-agents | Use when facing 2+ independent tasks that can be worked on without shared state or sequential dependencies | `.claude/skills/dispatching-parallel-agents/SKILL.md` |
| docx | "Use this skill whenever the user wants to create, read, edit, or manipulate Word documents (.docx files). Triggers include: any mention of 'Word doc', 'word document', '.docx', or requests to produce professional documents with formatting like tables of contents, headings, page numbers, or letterheads. Also use when extracting or reorganizing content from .docx files, inserting or replacing images in documents, performing find-and-replace in Word files, working with tracked changes or comments, or converting content into a polished Word document. If the user asks for a 'report', 'memo', 'letter', 'template', or similar deliverable as a Word or .docx file, use this skill. Do NOT use for PDFs, spreadsheets, Google Docs, or general coding tasks unrelated to document generation." | `.claude/skills/docx/SKILL.md` |
| draft-nda | "Draft a detailed Non-Disclosure Agreement between two parties covering information types, jurisdiction, and clauses needing legal review. Use when creating confidentiality agreements or preparing an NDA for a partnership." | `.claude/skills/draft-nda/SKILL.md` |
| dummy-dataset | "Generate realistic dummy datasets for testing with customizable columns, constraints, and output formats (CSV, JSON, SQL, Python script). Use when creating test data, building mock datasets, or generating sample data for development and demos." | `.claude/skills/dummy-dataset/SKILL.md` |
| excel-automation | Create, parse, and control Excel files on macOS. Professional formatting with openpyxl, complex xlsm parsing with stdlib zipfile+xml for investment bank financial models, and Excel window control via AppleScript. Use when creating formatted Excel reports, parsing financial models that openpyxl cannot handle, or automating Excel on macOS. | `.claude/skills/excel-automation/SKILL.md` |
| executing-plans | Use when you have a written implementation plan to execute in a separate session with review checkpoints | `.claude/skills/executing-plans/SKILL.md` |
| fact-checker | Verifies factual claims in documents using web search and official sources, then proposes corrections with user confirmation. Use when the user asks to fact-check, verify information, validate claims, check accuracy, or update outdated information in documents. Supports AI model specs, technical documentation, statistics, and general factual statements. | `.claude/skills/fact-checker/SKILL.md` |
| finishing-a-development-branch | Use when implementation is complete, all tests pass, and you need to decide how to integrate the work - guides completion of development work by presenting structured options for merge, PR, or cleanup | `.claude/skills/finishing-a-development-branch/SKILL.md` |
| grammar-check | "Identify grammar, logical, and flow errors in text and suggest targeted fixes without rewriting the entire text. Use when proofreading content, checking writing quality, or reviewing a draft." | `.claude/skills/grammar-check/SKILL.md` |
| growth-loops | "Identify growth loops (flywheels) for sustainable traction. Evaluates 5 loop types: Viral, Usage, Collaboration, User-Generated, and Referral. Use when designing growth mechanisms, building product-led traction, or understanding how growth loops work." | `.claude/skills/growth-loops/SKILL.md` |
| gtm-motions | "Identify the best GTM motions and tools across 7 motion types: Inbound, Outbound, Paid Digital, Community, Partners, ABM, and PLG. Use when selecting marketing channels, choosing between inbound and outbound strategy, or planning cross-channel campaigns." | `.claude/skills/gtm-motions/SKILL.md` |
| gtm-strategy | "Create a go-to-market strategy covering marketing channels, messaging, success metrics, and launch timeline. Use when planning a product launch, creating a GTM plan from scratch, or defining a launch strategy for a new market." | `.claude/skills/gtm-strategy/SKILL.md` |
| ideal-customer-profile | "Identify the Ideal Customer Profile (ICP) from research data with demographics, behaviors, JTBD, and needs. Use when defining your ICP, analyzing PMF survey data, or understanding who your best customers are." | `.claude/skills/ideal-customer-profile/SKILL.md` |
| identify-assumptions-existing | "Identify risky assumptions for a feature idea in an existing product across Value, Usability, Viability, and Feasibility. Uses multi-perspective devil's advocate thinking. Use when stress-testing a feature idea, doing risk assessment, or preparing for assumption mapping." | `.claude/skills/identify-assumptions-existing/SKILL.md` |
| identify-assumptions-new | "Identify risky assumptions for a new product idea across 8 risk categories including Go-to-Market, Strategy, and Team. Use when evaluating startup risks, assessing a new product concept, or mapping assumptions for a new venture." | `.claude/skills/identify-assumptions-new/SKILL.md` |
| interview-script | "Create a structured customer interview script with JTBD probing questions, warm-up, core exploration, and wrap-up sections. Follows The Mom Test principles — no leading questions, no pitching, focus on past behavior. Use when preparing for user interviews, creating interview guides, or planning discovery research." | `.claude/skills/interview-script/SKILL.md` |
| job-stories | "Create job stories using the 'When [situation], I want to [motivation], so I can [outcome]' format with detailed acceptance criteria. Use when writing job stories, creating JTBD-style backlog items, or expressing user situations and motivations." | `.claude/skills/job-stories/SKILL.md` |
| lark-approval | "飞书审批 API：审批实例、审批任务管理。" | `.claude/skills/lark-approval/SKILL.md` |
| lark-attendance | "飞书考勤打卡：查询自己的考勤打卡记录" | `.claude/skills/lark-attendance/SKILL.md` |
| lark-base | "当需要用 lark-cli 操作飞书多维表格（Base）时调用：搜索 Base、建表、字段管理、记录读写、记录分享链接、视图配置、历史查询，以及角色/表单/仪表盘管理/工作流；也适用于把旧的 +table / +field / +record 写法改成当前命令写法。涉及字段设计、公式字段、查找引用、跨表计算、行级派生指标、数据分析需求时也必须使用本 skill。" | `.claude/skills/lark-base/SKILL.md` |
| lark-calendar | "飞书日历（calendar）：提供日历与日程（会议）的全面管理能力。核心场景包括：查看/搜索日程、创建/更新日程、管理参会人、查询忙闲状态及推荐空闲时段、查询/搜索与预定会议室。注意：涉及【预约日程/会议】或【查询/预定会议室】时，必须先读取 references/lark-calendar-schedule-meeting.md 工作流！高频操作请优先使用 Shortcuts：+agenda（快速概览今日/近期行程）、+create（创建日程并按需邀请参会人及预定会议室）、+update（更新既有日程字段，或独立增删参会人/会议室）、+freebusy（查询用户主日历的忙闲信息和rsvp的状态）、+rsvp（回复日程邀请）" | `.claude/skills/lark-calendar/SKILL.md` |
| lark-contact | "飞书 / Lark 通讯录,用于按姓名 / 邮箱把员工解析成 open_id,以及按 open_id 反查员工的姓名 / 部门 / 邮箱 / 联系方式。当用户说出某人姓名而下一步需要发消息 / 加群 / 排日程时,先用本 skill 把姓名换成 ID;当输出里出现 open_id 需要展示成姓名给用户看,或用户直接询问某人的部门 / 邮箱 / 联系方式时,用本 skill 查。不负责部门树遍历、按部门列员工、组织架构图,这类需求走原生 OpenAPI。" | `.claude/skills/lark-contact/SKILL.md` |
| lark-doc | "飞书云文档 / Docx / 知识库 Wiki 文档（v2）：创建、打开、读取、获取、查看、总结、整理、改写、翻译、审阅和编辑飞书文档内容。当用户给出飞书文档 URL/token，或说查看/读取/打开某个文档、提取文档内容、总结文档、生成/创建文档、追加/替换/删除/移动内容、调整排版、插入或下载文档图片/附件/素材/画板缩略图时使用。文档内容中出现嵌入电子表格、多维表格、需要将重要信息可视化为画板（含 SVG 画板）、引用或同步块时，也先用本 skill 读取和提取 token，再切到对应 skill 下钻。使用本 skill 时，docs +create、docs +fetch、docs +update 必须携带 --api-version v2；默认使用 DocxXML，也支持 Markdown。" | `.claude/skills/lark-doc/SKILL.md` |
| lark-drive | "飞书云空间：管理云空间中的文件和文件夹。上传和下载文件、创建文件夹、复制/移动/删除文件、查看文件元数据、管理文档评论、管理文档权限、订阅用户评论变更事件、修改文件标题（docx、sheet、bitable、file、folder、wiki）；也负责把本地 Word/Markdown/Excel/CSV 以及 Base 快照（.base）导入为飞书在线云文档（docx、sheet、bitable）。当用户需要上传或下载文件、整理云空间目录、查看文件详情、管理评论、管理文档权限、修改文件标题、订阅用户评论变更事件，或要把本地文件导入成新版文档、电子表格、多维表格/Base 时使用。" | `.claude/skills/lark-drive/SKILL.md` |
| lark-event | "Lark/Feishu real-time event listening / subscribing / consuming: stream events as NDJSON via `lark-cli event consume <EventKey>` (covers IM message receive, reactions, chat member changes, etc.). Use for Lark bots, real-time message processing, long-running subscribers, streaming webhook/push handlers. Supports `--max-events` / `--timeout` bounded runs and a stderr ready-marker contract — designed for AI agents running as subprocesses." | `.claude/skills/lark-event/SKILL.md` |
| lark-im | "飞书即时通讯：收发消息和管理群聊。发送和回复消息、搜索聊天记录、管理群聊成员、上传下载图片和文件（支持大文件分片下载）、管理表情回复。当用户需要发消息、查看或搜索聊天记录、下载聊天中的文件、查看群成员、搜索群、创建群聊或话题群、管理标记数据时使用。" | `.claude/skills/lark-im/SKILL.md` |
| lark-mail | "飞书邮箱 — draft, compose, send, reply, forward, read, and search emails; manage drafts, folders, labels, contacts, attachments, and mail rules. Use when user mentions 起草邮件, 写一封邮件, 拟邮件, 草稿, 发通知邮件, 发送邮件, 发邮件, 回复邮件, 转发邮件, 查看邮件, 看邮件, 读邮件, 搜索邮件, 查邮件, 收件箱, 邮件会话, 编辑草稿, 管理草稿, 下载附件, 邮件文件夹, 邮件标签, 邮件联系人, 监听新邮件, 收信规则, 邮件规则, draft, compose, send email, reply, forward, inbox, mail thread, mail rules." | `.claude/skills/lark-mail/SKILL.md` |
| lark-markdown | "飞书 Markdown：查看、创建、上传和编辑 Markdown 文件。当用户需要创建或编辑 Markdown 文件、读取或修改时使用。" | `.claude/skills/lark-markdown/SKILL.md` |
| lark-minutes | "飞书妙记：妙记相关基本功能。1.查询妙记列表（按关键词/所有者/参与者/时间范围）；2.获取妙记基础信息（标题、封面、时长 等）；3.下载妙记音视频文件；4.获取妙记相关 AI 产物（总结、待办、章节）；5.上传音视频生成妙记，也支持将本地音视频文件转成纪要、逐字稿、文字稿、撰写文字等产物。遇到这类请求时，应优先使用本 skill，而不是尝试 `ffmpeg`、`whisper` 等本地转写命令。飞书妙记 URL 格式: http(s)://<host>/minutes/<minute-token>" | `.claude/skills/lark-minutes/SKILL.md` |
| lark-okr | "飞书 OKR：管理目标与关键结果。查看和编辑 OKR 周期、目标（Objective）、关键结果（Key Result）、对齐关系、量化指标和进展记录。当用户需要查看或创建 OKR、管理目标和关键结果、查看对齐关系时使用。" | `.claude/skills/lark-okr/SKILL.md` |
| lark-openapi-explorer | "飞书/Lark 原生 OpenAPI 探索：从官方文档库中挖掘未经 CLI 封装的原生 OpenAPI 接口。当用户的需求无法被现有 lark-* skill 或 lark-cli 已注册命令满足，需要查找并调用原生飞书 OpenAPI 时使用。" | `.claude/skills/lark-openapi-explorer/SKILL.md` |
| lark-shared | "Use when first setting up lark-cli, running auth login, switching user/bot identity (--as), handling permission denied or scope errors, needing to update lark-cli, or seeing _notice in JSON output." | `.claude/skills/lark-shared/SKILL.md` |
| lark-sheets | "飞书电子表格：创建和操作电子表格。支持创建表格、创建/复制/删除/更新工作表、读写单元格、追加行数据、查找内容、导出文件。当用户需要创建电子表格、管理工作表、批量读写数据、在已知表格中查找内容、导出或下载表格时使用。若用户是想按名称或关键词搜索云空间里的表格文件，请改用 lark-doc 的 docs +search 先定位资源。" | `.claude/skills/lark-sheets/SKILL.md` |
| lark-skill-maker | "创建 lark-cli 的自定义 Skill。当用户需要把飞书 API 操作封装成可复用的 Skill（包装原子 API 或编排多步流程）时使用。" | `.claude/skills/lark-skill-maker/SKILL.md` |
| lark-slides | "飞书幻灯片：创建和编辑幻灯片，接口通过 XML 协议通信。创建演示文稿、读取幻灯片内容、管理幻灯片页面（创建、删除、读取、局部替换）。当用户需要创建或编辑幻灯片、读取或修改单个页面时使用。" | `.claude/skills/lark-slides/SKILL.md` |
| lark-task | "飞书任务：管理任务、清单和任务智能体。创建待办任务、查看和更新任务状态、拆分子任务、组织任务清单、分配协作成员、上传任务附件、注册或注销任务智能体、更新任务智能体的主页数据、写入智能体任务记录。当用户需要创建待办事项、查看任务列表、跟踪任务进度、管理项目清单或给他人分配任务、为任务上传附件文件、注册注销任务智能体、更新智能体主页数据、写入任务记录时使用。" | `.claude/skills/lark-task/SKILL.md` |
| lark-vc | "飞书视频会议：搜索历史会议、查询会议纪要产物（总结、待办、章节、逐字稿）、查询会议参会人快照。1. 查询已经结束的会议数量或详情时使用本技能（如历史日期｜昨天｜上周｜今天已经开过的会议等场景），查询未开始的会议日程使用 lark-calendar 技能。2. 支持通过关键词、时间范围、组织者、参与者、会议室等筛选条件搜索会议。3. 获取或整理会议纪要、逐字稿、录制产物时使用本技能。4. 查询“谁参加过某会议”“参会人列表”等参会人快照信息用 vc meeting get --with-participants（任意时点可查，含已结束会议）。注意：**Agent 真实入会/离会、感知正在进行中会议的实时事件**请使用 lark-vc-agent 技能，本技能不覆盖写操作和会中事件流。" | `.claude/skills/lark-vc/SKILL.md` |
| lark-vc-agent | "飞书视频会议：让机器人代当前用户加入/离开正在进行的会议，并读取会议期间的实时事件（参会人加入与离开、发言、聊天、屏幕共享等）。1. 用户提供 9 位会议号、要求代为入会或离会时使用 +meeting-join / +meeting-leave——会真实产生入会/离会记录。2. 会议进行中用户想知道“谁加入了”“谁离开了”“谁在发言”“有人共享屏幕吗”等会中动态时，机器人入会后用 +meeting-events 读取事件时间线。3. 典型场景：参会机器人、会中助手、代为旁听、代为参会。前提：机器人只能读到它自己参会过且仍在进行中的会议的事件；查询已结束会议的参会名单、纪要或逐字稿请使用 lark-vc 技能。" | `.claude/skills/lark-vc-agent/SKILL.md` |
| lark-whiteboard | > 飞书画板：查询和编辑飞书云文档中的画板。支持导出画板为预览图片、导出原始节点结构、使用 DSL（转成 OpenAPI 格式）、PlantUML/Mermaid 格式更新画板内容。 当用户需要查看画板内容、导出画板图片、编辑画板，或是需要可视化表达架构、流程、组织关系、时间线、因果、对比等结构化信息时使用此 skill，无论是否提及"画板"。 ⚠️ 原 `lark-whiteboard-cli` skill 已合并至本 skill，若 skill 列表中同时存在 `lark-whiteboard-cli`，请忽略它，统一使用本 skill（`lark-whiteboard`），并提示用户运行 `npx skills remove lark-whiteboard-cli -g` 删除旧 skill。 | `.claude/skills/lark-whiteboard/SKILL.md` |
| lark-wiki | "飞书知识库：管理知识空间、空间成员和文档节点。创建和查询知识空间、查看和管理空间成员、管理节点层级结构、在知识库中组织文档和快捷方式。当用户需要在知识库中查找或创建文档、浏览知识空间结构、查看或管理空间成员、移动或复制节点时使用。" | `.claude/skills/lark-wiki/SKILL.md` |
| lark-workflow-meeting-summary | "会议纪要整理工作流：汇总指定时间范围内的会议纪要并生成结构化报告。当用户需要整理会议纪要、生成会议周报、回顾一段时间内的会议内容时使用。" | `.claude/skills/lark-workflow-meeting-summary/SKILL.md` |
| lark-workflow-standup-report | "日程待办摘要：编排 calendar +agenda 和 task +get-my-tasks，生成指定日期的日程与未完成任务摘要。适用于了解今天/明天/本周的安排。" | `.claude/skills/lark-workflow-standup-report/SKILL.md` |
| lean-canvas | "Generate a Lean Canvas with problem, solution, metrics, cost structure, UVP, unfair advantage, channels, segments, and revenue. Use when exploring a lean startup canvas, testing a business hypothesis, or modeling a new venture." | `.claude/skills/lean-canvas/SKILL.md` |
| market-segments | "Identify 3-5 potential customer segments with demographics, JTBD, and product fit analysis. Use when exploring market segments, identifying target audiences, evaluating new markets, or learning how to segment a market." | `.claude/skills/market-segments/SKILL.md` |
| market-sizing | "Estimate market size using TAM, SAM, and SOM with top-down and bottom-up approaches. Use when sizing a market opportunity, estimating addressable market, preparing for investor pitches, or evaluating market entry." | `.claude/skills/market-sizing/SKILL.md` |
| marketing-ideas | "Generate 5 creative, cost-effective marketing ideas with channels, messaging, and engagement rationale. Use when brainstorming marketing campaigns, planning product promotion, or looking for creative marketing tactics." | `.claude/skills/marketing-ideas/SKILL.md` |
| metrics-dashboard | "Define and design a product metrics dashboard with key metrics, data sources, visualization types, and alert thresholds. Use when creating a metrics dashboard, defining KPIs, setting up product analytics, or building a data monitoring plan." | `.claude/skills/metrics-dashboard/SKILL.md` |
| monetization-strategy | "Brainstorm 3-5 monetization strategies with audience fit, risks, and validation experiments. Use when exploring revenue models, evaluating pricing strategies, or deciding how to monetize a product." | `.claude/skills/monetization-strategy/SKILL.md` |
| north-star-metric | "Define a North Star Metric and 3-5 supporting input metrics that form a metrics constellation. Classify the business game (Attention, Transaction, Productivity) and validate against 7 criteria for an effective North Star. Use when choosing a North Star Metric, setting up a metrics framework, learning about the North Star Framework, or deciding what to measure." | `.claude/skills/north-star-metric/SKILL.md` |
| opportunity-solution-tree | "Build an Opportunity Solution Tree (OST) to structure product discovery — map a desired outcome to opportunities, solutions, and experiments. Based on Teresa Torres' Continuous Discovery Habits. Use when structuring discovery work, mapping opportunities to solutions, or deciding what to build next." | `.claude/skills/opportunity-solution-tree/SKILL.md` |
| outcome-roadmap | "Transform an output-focused roadmap into an outcome-focused one that communicates strategic intent. Rewrites initiatives as outcome statements reflecting user and business impacts. Use when shifting to outcome roadmaps, making a roadmap more strategic, or rewriting feature lists as outcomes." | `.claude/skills/outcome-roadmap/SKILL.md` |
| pdf | Use this skill whenever the user wants to do anything with PDF files. This includes reading or extracting text/tables from PDFs, combining or merging multiple PDFs into one, splitting PDFs apart, rotating pages, adding watermarks, creating new PDFs, filling PDF forms, encrypting/decrypting PDFs, extracting images, and OCR on scanned PDFs to make them searchable. If the user mentions a .pdf file or asks to produce one, use this skill. | `.claude/skills/pdf/SKILL.md` |
| pestle-analysis | "Perform a PESTLE analysis covering Political, Economic, Social, Technological, Legal, and Environmental factors. Use when assessing the macro environment, doing strategic planning, or evaluating external factors affecting your business." | `.claude/skills/pestle-analysis/SKILL.md` |
| porters-five-forces | "Perform Porter's Five Forces analysis — competitive rivalry, supplier power, buyer power, threat of substitutes, and threat of new entrants. Use when analyzing industry dynamics, assessing competitive forces, or evaluating market attractiveness." | `.claude/skills/porters-five-forces/SKILL.md` |
| positioning-ideas | "Brainstorm product positioning ideas differentiated from competitors. Identifies top competitors and generates positioning statements with rationale. Use when developing product positioning, differentiating from competitors, or crafting brand positioning strategy." | `.claude/skills/positioning-ideas/SKILL.md` |
| pptx | "Use this skill any time a .pptx file is involved in any way — as input, output, or both. This includes: creating slide decks, pitch decks, or presentations; reading, parsing, or extracting text from any .pptx file (even if the extracted content will be used elsewhere, like in an email or summary); editing, modifying, or updating existing presentations; combining or splitting slide files; working with templates, layouts, speaker notes, or comments. Trigger whenever the user mentions \"deck,\" \"slides,\" \"presentation,\" or references a .pptx filename, regardless of what they plan to do with the content afterward. If a .pptx file needs to be opened, created, or touched, use this skill." | `.claude/skills/pptx/SKILL.md` |
| pre-mortem | "Run a pre-mortem risk analysis on a PRD or launch plan. Categorizes risks as Tigers (real problems), Paper Tigers (overblown concerns), and Elephants (unspoken worries), then classifies as launch-blocking, fast-follow, or track. Use when preparing for launch, stress-testing a product plan, or identifying what could go wrong." | `.claude/skills/pre-mortem/SKILL.md` |
| pricing-strategy | "Analyze and design pricing strategies including pricing models, competitive pricing analysis, willingness-to-pay estimation, and price elasticity. Use when setting prices, evaluating pricing models, preparing for a pricing change, or comparing freemium vs paid approaches." | `.claude/skills/pricing-strategy/SKILL.md` |
| prioritization-frameworks | "Reference guide to 9 prioritization frameworks with formulas, when-to-use guidance, and templates — RICE, ICE, Kano, MoSCoW, Opportunity Score, and more. Use when selecting a prioritization method, comparing frameworks like RICE vs ICE, or learning how different prioritization approaches work." | `.claude/skills/prioritization-frameworks/SKILL.md` |
| prioritize-assumptions | "Prioritize assumptions using an Impact × Risk matrix and suggest experiments for each. Use when triaging a list of assumptions, deciding what to test first, or applying the assumption prioritization canvas." | `.claude/skills/prioritize-assumptions/SKILL.md` |
| prioritize-features | "Prioritize a backlog of feature ideas based on impact, effort, risk, and strategic alignment with top 5 recommendations. Use when prioritizing a feature backlog, making scope decisions, or ranking product ideas." | `.claude/skills/prioritize-features/SKILL.md` |
| privacy-policy | "Draft a detailed privacy policy covering data types, jurisdiction, GDPR and compliance considerations, and clauses needing legal review. Use when creating a privacy policy, updating data protection documentation, or preparing for compliance." | `.claude/skills/privacy-policy/SKILL.md` |
| product-analysis | Multi-path parallel product analysis with cross-model test-time compute scaling. Spawns parallel agents (Claude Code agent teams + Codex CLI) to explore product from multiple perspectives, then synthesizes findings into actionable optimization plans. Can invoke competitors-analysis for competitive benchmarking. Use when "product audit", "self-review", "发布前审查", "产品分析", "analyze our product", "UX audit", or "信息架构审计". | `.claude/skills/product-analysis/SKILL.md` |
| product-name | "Brainstorm 5 unique, memorable product names with rationale aligned to brand values and target audience. Use when naming a new product, rebranding, or exploring product name ideas." | `.claude/skills/product-name/SKILL.md` |
| product-strategy | "Create a comprehensive product strategy using the 9-section Product Strategy Canvas — vision, segments, costs, value propositions, trade-offs, metrics, growth, capabilities, and defensibility. Use when building a product strategy, creating a strategic plan, or defining product direction." | `.claude/skills/product-strategy/SKILL.md` |
| product-vision | "Brainstorm an inspiring, achievable, and emotional product vision that motivates teams and aligns stakeholders. Use when defining or refining a product vision, creating a vision statement, or aligning the team around a shared direction." | `.claude/skills/product-vision/SKILL.md` |
| prompt-optimizer | Transform vague prompts into precise, well-structured specifications using EARS (Easy Approach to Requirements Syntax) methodology. This skill should be used when users provide loose requirements, ambiguous feature descriptions, or need to enhance prompts for AI-generated code, products, or documents. Triggers include requests to "optimize my prompt", "improve this requirement", "make this more specific", or when raw requirements lack detail and structure. | `.claude/skills/prompt-optimizer/SKILL.md` |
| promptfoo-evaluation | Configures and runs LLM evaluation using Promptfoo framework. Use when setting up prompt testing, creating evaluation configs (promptfooconfig.yaml), writing Python custom assertions, implementing llm-rubric for LLM-as-judge, or managing few-shot examples in prompts. Triggers on keywords like "promptfoo", "eval", "LLM evaluation", "prompt testing", or "model comparison". | `.claude/skills/promptfoo-evaluation/SKILL.md` |
| qa-expert | This skill should be used when establishing comprehensive QA testing processes for any software project. Use when creating test strategies, writing test cases following Google Testing Standards, executing test plans, tracking bugs with P0-P4 classification, calculating quality metrics, or generating progress reports. Includes autonomous execution capability via master prompts and complete documentation templates for third-party QA team handoffs. Implements OWASP security testing and achieves 90% coverage targets. | `.claude/skills/qa-expert/SKILL.md` |
| receiving-code-review | Use when receiving code review feedback, before implementing suggestions, especially if feedback seems unclear or technically questionable - requires technical rigor and verification, not performative agreement or blind implementation | `.claude/skills/receiving-code-review/SKILL.md` |
| release-notes | "Generate user-facing release notes from tickets, PRDs, or changelogs. Creates clear, engaging summaries organized by category (new features, improvements, fixes). Use when writing release notes, creating changelogs, announcing product updates, or summarizing what shipped." | `.claude/skills/release-notes/SKILL.md` |
| requesting-code-review | Use when completing tasks, implementing major features, or before merging to verify work meets requirements | `.claude/skills/requesting-code-review/SKILL.md` |
| retro | "Facilitate a structured sprint retrospective — what went well, what didn't, and prioritized action items with owners and deadlines. Use when running a retrospective, reflecting on a sprint, creating action items from team feedback, or learning how to run effective retros." | `.claude/skills/retro/SKILL.md` |
| review-resume | "Comprehensive PM resume review and tailoring against 10 best practices including XYZ+S formula, keyword optimization, job-specific tailoring, and structure. Use when reviewing a PM resume, preparing for job applications, or improving resume impact." | `.claude/skills/review-resume/SKILL.md` |
| scrapling-skill | Install, troubleshoot, and use Scrapling CLI to extract HTML, Markdown, or text from webpages. Use this skill whenever the user mentions Scrapling, `uv tool install scrapling`, `scrapling extract`, WeChat/mp.weixin articles, browser-backed page fetching, or needs help deciding between static and dynamic extraction. | `.claude/skills/scrapling-skill/SKILL.md` |
| sentiment-analysis | "Analyze user feedback data to identify segments with sentiment scores, JTBD, and product satisfaction insights. Use when analyzing user feedback at scale, running sentiment analysis on reviews or surveys, or identifying satisfaction patterns." | `.claude/skills/sentiment-analysis/SKILL.md` |
| ckm:slides | Create strategic HTML presentations with Chart.js, design tokens, responsive layouts, copywriting formulas, and contextual slide strategies. | `.claude/skills/slides/SKILL.md` |
| slides-creator | Narrative-first slide deck creation. Guides users through structured narrative design (ABCDEFG model), then delegates visual generation to baoyu-slide-deck. Triggers on "create slides", "make a presentation", "generate deck", "slide deck", "PPT", or when user needs to turn content into visual slides. | `.claude/skills/slides-creator/SKILL.md` |
| "speckit-analyze" | "Perform a non-destructive cross-artifact consistency and quality analysis across spec.md, plan.md, and tasks.md after task generation." | `.claude/skills/speckit-analyze/SKILL.md` |
| "speckit-checklist" | "Generate a custom checklist for the current feature based on user requirements." | `.claude/skills/speckit-checklist/SKILL.md` |
| "speckit-clarify" | "Identify underspecified areas in the current feature spec by asking up to 5 highly targeted clarification questions and encoding answers back into the spec." | `.claude/skills/speckit-clarify/SKILL.md` |
| "speckit-constitution" | "Create or update the project constitution from interactive or provided principle inputs, ensuring all dependent templates stay in sync." | `.claude/skills/speckit-constitution/SKILL.md` |
| speckit-git-commit | Auto-commit changes after a Spec Kit command completes | `.claude/skills/speckit-git-commit/SKILL.md` |
| speckit-git-feature | Create a feature branch with sequential or timestamp numbering | `.claude/skills/speckit-git-feature/SKILL.md` |
| speckit-git-initialize | Initialize a Git repository with an initial commit | `.claude/skills/speckit-git-initialize/SKILL.md` |
| speckit-git-remote | Detect Git remote URL for GitHub integration | `.claude/skills/speckit-git-remote/SKILL.md` |
| speckit-git-validate | Validate current branch follows feature branch naming conventions | `.claude/skills/speckit-git-validate/SKILL.md` |
| "speckit-implement" | "Execute the implementation plan by processing and executing all tasks defined in tasks.md" | `.claude/skills/speckit-implement/SKILL.md` |
| "speckit-plan" | "Execute the implementation planning workflow using the plan template to generate design artifacts." | `.claude/skills/speckit-plan/SKILL.md` |
| "speckit-specify" | "Create or update the feature specification from a natural language feature description." | `.claude/skills/speckit-specify/SKILL.md` |
| "speckit-tasks" | "Generate an actionable, dependency-ordered tasks.md for the feature based on available design artifacts." | `.claude/skills/speckit-tasks/SKILL.md` |
| "speckit-taskstoissues" | "Convert existing tasks into actionable, dependency-ordered GitHub issues for the feature based on available design artifacts." | `.claude/skills/speckit-taskstoissues/SKILL.md` |
| sprint-plan | "Plan a sprint with capacity estimation, story selection, dependency mapping, and risk identification. Use when preparing for sprint planning, estimating team capacity, selecting stories, or balancing sprint scope against velocity." | `.claude/skills/sprint-plan/SKILL.md` |
| sql-queries | "Generate SQL queries from natural language descriptions. Supports BigQuery, PostgreSQL, MySQL, and other dialects. Reads database schemas from uploaded diagrams or documentation. Use when writing SQL, building data reports, exploring databases, or translating business questions into queries." | `.claude/skills/sql-queries/SKILL.md` |
| stakeholder-map | "Build a stakeholder map using a power/interest grid, identify communication strategies per quadrant, and generate a communication plan. Use when managing stakeholders, preparing for a launch, aligning cross-functional teams, or planning stakeholder engagement." | `.claude/skills/stakeholder-map/SKILL.md` |
| startup-canvas | "Generate a Startup Canvas combining Product Strategy (9 sections) and Business Model (costs + revenue) for a new product. An alternative to BMC and Lean Canvas that separates strategy from business model. Use when launching a new product or evaluating a startup concept." | `.claude/skills/startup-canvas/SKILL.md` |
| subagent-driven-development | Use when executing implementation plans with independent tasks in the current session | `.claude/skills/subagent-driven-development/SKILL.md` |
| summarize-interview | "Summarize a customer interview transcript into a structured template with JTBD, satisfaction signals, and action items. Use when processing interview recordings or transcripts, synthesizing discovery interviews, or creating interview summaries." | `.claude/skills/summarize-interview/SKILL.md` |
| summarize-meeting | "Summarize a meeting transcript into structured notes with date, participants, topic, key decisions, summary points, and action items. Use when processing meeting recordings, creating meeting notes, writing meeting minutes, or recapping discussions." | `.claude/skills/summarize-meeting/SKILL.md` |
| swot-analysis | "Perform a detailed SWOT analysis — strengths, weaknesses, opportunities, and threats with actionable recommendations. Use when doing strategic assessment, competitive analysis, or evaluating a product or business position." | `.claude/skills/swot-analysis/SKILL.md` |
| systematic-debugging | Use when encountering any bug, test failure, or unexpected behavior, before proposing fixes | `.claude/skills/systematic-debugging/SKILL.md` |
| test-driven-development | Use when implementing any feature or bugfix, before writing implementation code | `.claude/skills/test-driven-development/SKILL.md` |
| test-scenarios | "Create comprehensive test scenarios from user stories with test objectives, starting conditions, user roles, step-by-step actions, and expected outcomes. Use when writing QA test cases, creating test plans, defining acceptance tests, or preparing for feature validation." | `.claude/skills/test-scenarios/SKILL.md` |
| ui-designer | Extract design systems from reference UI images and generate implementation-ready UI design prompts. Use when users provide UI screenshots/mockups and want to create consistent designs, generate design systems, or build MVP UIs matching reference aesthetics. | `.claude/skills/ui-designer/SKILL.md` |
| ckm:ui-styling | Create beautiful, accessible user interfaces with shadcn/ui components (built on Radix UI + Tailwind), Tailwind CSS utility-first styling, and canvas-based visual designs. Use when building user interfaces, implementing design systems, creating responsive layouts, adding accessible components (dialogs, dropdowns, forms, tables), customizing themes and colors, implementing dark mode, generating visual designs and posters, or establishing consistent styling patterns across applications. | `.claude/skills/ui-styling/SKILL.md` |
| ui-ux-pro-max | "UI/UX design intelligence for web and mobile. Includes 50+ styles, 161 color palettes, 57 font pairings, 161 product types, 99 UX guidelines, and 25 chart types across 10 stacks (React, Next.js, Vue, Svelte, SwiftUI, React Native, Flutter, Tailwind, shadcn/ui, and HTML/CSS). Actions: plan, build, create, design, implement, review, fix, improve, optimize, enhance, refactor, and check UI/UX code. Projects: website, landing page, dashboard, admin panel, e-commerce, SaaS, portfolio, blog, and mobile app. Elements: button, modal, navbar, sidebar, card, table, form, and chart. Styles: glassmorphism, claymorphism, minimalism, brutalism, neumorphism, bento grid, dark mode, responsive, skeuomorphism, and flat design. Topics: color systems, accessibility, animation, layout, typography, font pairing, spacing, interaction states, shadow, and gradient. Integrations: shadcn/ui MCP for component search and examples." | `.claude/skills/ui-ux-pro-max/SKILL.md` |
| user-personas | "Create refined user personas from research data — 3 personas with JTBD, pains, gains, and unexpected insights. Use when building personas from survey data, creating user profiles from research, or segmenting users for product decisions." | `.claude/skills/user-personas/SKILL.md` |
| user-segmentation | "Segment users from feedback data based on behavior, JTBD, and needs. Identifies at least 3 distinct user segments. Use when segmenting a user base, analyzing diverse user feedback, or building a segmentation model." | `.claude/skills/user-segmentation/SKILL.md` |
| user-stories | "Create user stories following the 3 C's (Card, Conversation, Confirmation) and INVEST criteria with descriptions, design links, and acceptance criteria. Use when writing user stories, breaking down features into backlog items, or defining acceptance criteria." | `.claude/skills/user-stories/SKILL.md` |
| using-git-worktrees | Use when starting feature work that needs isolation from current workspace or before executing implementation plans - ensures an isolated workspace exists via native tools or git worktree fallback | `.claude/skills/using-git-worktrees/SKILL.md` |
| using-superpowers | Use when starting any conversation - establishes how to find and use skills, requiring Skill tool invocation before ANY response including clarifying questions | `.claude/skills/using-superpowers/SKILL.md` |
| value-prop-statements | "Generate value proposition statements for marketing, sales, and onboarding from existing value propositions. Use when writing marketing copy, creating sales messaging, or crafting onboarding messages." | `.claude/skills/value-prop-statements/SKILL.md` |
| value-proposition | "Design a detailed value proposition using a 6-part JTBD template — Who, Why, What before, How, What after, Alternatives. Use when creating a value proposition, analyzing customer value delivery, or articulating why customers should choose your product." | `.claude/skills/value-proposition/SKILL.md` |
| verification-before-completion | Use when about to claim work is complete, fixed, or passing, before committing or creating PRs - requires running verification commands and confirming output before making any success claims; evidence before assertions always | `.claude/skills/verification-before-completion/SKILL.md` |
| video-comparer | This skill should be used when comparing two videos to analyze compression results or quality differences. Generates interactive HTML reports with quality metrics (PSNR, SSIM) and frame-by-frame visual comparisons. Triggers when users mention "compare videos", "video quality", "compression analysis", "before/after compression", or request quality assessment of compressed videos. | `.claude/skills/video-comparer/SKILL.md` |
| writing-plans | Use when you have a spec or requirements for a multi-step task, before touching code | `.claude/skills/writing-plans/SKILL.md` |
| writing-skills | Use when creating new skills, editing existing skills, or verifying skills work before deployment | `.claude/skills/writing-skills/SKILL.md` |
| wwas | "Create product backlog items in Why-What-Acceptance format — independent, valuable, testable items with strategic context. Use when writing structured backlog items, breaking features into work items, or using the WWA format." | `.claude/skills/wwas/SKILL.md` |
| xlsx | "Use this skill any time a spreadsheet file is the primary input or output. This means any task where the user wants to: open, read, edit, or fix an existing .xlsx, .xlsm, .csv, or .tsv file (e.g., adding columns, computing formulas, formatting, charting, cleaning messy data); create a new spreadsheet from scratch or from other data sources; or convert between tabular file formats. Trigger especially when the user references a spreadsheet file by name or path — even casually (like \"the xlsx in my downloads\") — and wants something done to it or produced from it. Also trigger for cleaning or restructuring messy tabular data files (malformed rows, misplaced headers, junk data) into proper spreadsheets. The deliverable must be a spreadsheet file. Do NOT trigger when the primary deliverable is a Word document, HTML report, standalone Python script, database pipeline, or Google Sheets API integration, even if tabular data is involved." | `.claude/skills/xlsx/SKILL.md` |
<!-- GSD:skills-end -->

<!-- GSD:workflow-start source:GSD defaults -->

## GSD Workflow Enforcement

Before using Edit, Write, or other file-changing tools, start work through a GSD command so planning artifacts and execution context stay in sync.

Use these entry points:

- `/gsd-quick` for small fixes, doc updates, and ad-hoc tasks
- `/gsd-debug` for investigation and bug fixing
- `/gsd-execute-phase` for planned phase work

Do not make direct repo edits outside a GSD workflow unless the user explicitly asks to bypass it.
<!-- GSD:workflow-end -->

<!-- GSD:profile-start -->

## Developer Profile

> Profile not yet configured. Run `/gsd-profile-user` to generate your developer profile.
> This section is managed by `generate-claude-profile` -- do not edit manually.
<!-- GSD:profile-end -->
