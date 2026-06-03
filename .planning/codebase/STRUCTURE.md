# Codebase Structure

**Analysis Date:** 2026-06-03

## Directory Layout

```
baxi/
├── cmd/                    # Go entry points
│   ├── baxi-api/           # HTTP API server main.go
│   ├── baxi-cli/           # CLI tool subcommands
│   ├── baxi-mcp/           # MCP server main.go
│   └── baxi-worker/        # Background worker main.go
├── internal/               # Go backend core (44 packages)
│   ├── action/             # Action registry, proposals, execution
│   ├── adapter/            # Channel adapters (Feishu, GitHub, CLI, Manual)
│   ├── alert/              # Dimensional alert engine + rules
│   ├── api/                # chi HTTP API
│   │   ├── dto/            # Request/response types
│   │   ├── handler/        # 14 HTTP handlers
│   │   └── middleware/     # Auth, CORS, request-id, error recovery
│   ├── audit/              # Audit log integration
│   ├── config/             # Env-based config struct
│   ├── configloader/       # YAML config parse/validate
│   ├── db/                 # PostgreSQL pool creation
│   ├── decision/           # Decision engine, case service, context builders
│   ├── eval/               # Decision evaluation, replay, metrics
│   ├── feature/            # Feature flags
│   ├── feishu/             # Feishu OpenAPI client
│   ├── governance/         # Classification, lineage, access policy, redaction
│   ├── httputil/           # JSON response helpers, pagination
│   ├── ingest/             # CSV loader, table mapping
│   ├── llm/                # LLM provider abstraction, prompts, schema validator
│   │   └── prompts/        # LLM prompt templates
│   ├── logger/             # zap logger initialization
│   ├── model/              # Shared domain types
│   ├── mcp/                # MCP server (31 tools, 11 domains)
│   ├── ontology/           # AIP object schema V1/V2, registry, query compiler
│   │   └── testdata/       # Ontology test fixtures
│   ├── outbox/             # Outbox write repository
│   ├── pipeline/           # Pipeline runner + step interface
│   │   └── steps/          # 7 pipeline step implementations
│   ├── recommendation/     # Recommendation generator
│   ├── repository/         # Data access layer
│   │   ├── agent_execution/# Agent execution log persistence
│   │   ├── alert/          # Alert query repository
│   │   ├── common/         # PoolProvider infrastructure
│   │   ├── context/        # Qoder context queries
│   │   ├── decision/       # Decision case + LLM decision queries
│   │   ├── governance/     # Governance config queries
│   │   ├── log/            # Log query repository
│   │   ├── mcp_call/       # MCP call record persistence
│   │   ├── ontology/       # Object queries across dwd/mart/ops
│   │   ├── outbox/         # Outbox read queries
│   │   ├── status/         # System status queries
│   │   └── task/           # Task query repository
│   ├── review/             # Review/approval flow + sandbox
│   ├── service/            # Business orchestration services (11 files)
│   ├── testutil/           # Testcontainers + fixtures
│   └── worker/             # Dispatch worker + background worker
├── frontend/               # React 19 SPA
│   ├── src/
│   │   ├── api/            # API client, typed endpoints
│   │   ├── components/     # Shared UI components
│   │   ├── hooks/          # Custom React hooks (sparse)
│   │   ├── lib/            # Utilities (sparse)
│   │   ├── pages/          # 13 route pages
│   │   └── __tests__/      # Vitest setup
│   ├── e2e/                # Playwright E2E tests
│   └── test-results/       # Test artifacts
├── config/                 # YAML governance configs (28 files)
├── migrations/             # Goose SQL migrations (29 files)
├── test/                   # Go E2E tests (integration, migration, security)
├── scripts/                # Utility scripts
│   ├── backup/             # PostgreSQL backup/restore
│   ├── migration/          # Migration helpers
│   ├── migration_baseline/ # Baseline generation
│   ├── rollback/           # Rollback scripts
│   └── verification/       # Phase verification scripts
├── docs/                   # Documentation
│   ├── assets/             # Diagrams and images
│   ├── external/           # External API docs
│   ├── governance/         # Governance rule docs
│   ├── mcp-integration/    # MCP integration guides
│   ├── migration/          # Migration plans
│   ├── openapi/            # OpenAPI specs
│   └── plan/               # Architecture plans
├── data/                   # Raw and processed data
│   ├── ads/                # Ads platform data
│   ├── aip/                # AIP data
│   ├── feishu/             # Feishu exports
│   ├── interim/            # Intermediate processing files
│   ├── ops/                # Operations data
│   ├── processed/          # Processed outputs
│   ├── raw/                # Raw CSV inputs
│   └── system/             # System-generated files
├── pi-extension/           # Pi Agent TypeScript extensions
│   ├── baxi-decision/      # Decision extension
│   ├── baxi-logger/        # Logger extension
│   ├── baxi-operations/    # Operations extension
│   └── baxi-sandbox/       # Sandbox extension
├── pi-agent/               # Pi Agent prompts, schemas, golden cases
├── outputs/                # Generated outputs
├── logs/                   # Runtime logs
└── diagrams/               # Generated architecture diagrams
```

## Directory Purposes

**`cmd/`: Entry Points**
- Purpose: Thin `main.go` wrappers for each binary
- Contains: 4 subdirectories, each with `package main`
- Key files: `cmd/baxi-api/main.go`, `cmd/baxi-worker/main.go`, `cmd/baxi-mcp/main.go`, `cmd/baxi-cli/main.go`

**`internal/`: Go Backend Core**
- Purpose: All business logic, data access, and infrastructure
- Contains: 44 packages organized by domain
- Key files: `internal/api/server.go`, `internal/service/decision_service.go`, `internal/decision/engine.go`

**`internal/api/`: HTTP API**
- Purpose: chi router, handlers, middleware, DTOs
- Contains: 3 subdirectories + flat files
- Key files: `routes.go`, `server.go`, `handler_factories.go`

**`internal/repository/`: Data Access**
- Purpose: PostgreSQL queries via pgx
- Contains: 12 domain subpackages + flat compatibility files
- Key files: `interfaces.go`, `common/pool.go`, subpackage `repository.go` files

**`internal/pipeline/`: ETL Pipeline**
- Purpose: Data pipeline orchestration
- Contains: Runner, step interface, audit recorder, 7 step implementations
- Key files: `runner.go`, `step.go`, `steps/*.go`

**`internal/mcp/`: MCP Server**
- Purpose: AI agent integration via Model Context Protocol
- Contains: 14 files, 31 tools across 11 domains
- Key files: `server.go`, `interfaces.go`, `tools_*.go`

**`frontend/`: React SPA**
- Purpose: Console UI for governance + analytics
- Contains: Vite project with React 19, TanStack Query, Tailwind v4
- Key files: `src/main.tsx`, `src/App.tsx`, `src/api/client.ts`

**`config/`: YAML Configs**
- Purpose: Governance rules, alert rules, action registry, ontology schema
- Contains: 28 YAML files
- Key files: `action_registry.yml`, `aip_object_schema_v2.yml`, `context_recipes.yml`

**`migrations/`: Database Migrations**
- Purpose: Goose-managed PostgreSQL schema evolution
- Contains: 29 sequential SQL files
- Key files: `001_init_schemas.sql` through `029_add_proposal_sandbox.sql`

**`test/`: E2E Tests**
- Purpose: Integration, migration contract, and security E2E tests
- Contains: 4 subdirectories, each its own package
- Key files: `integration/phase7_test.go`, `migration/contract_test.go`, `security/phase7_test.go`

**`pi-extension/`: Pi Agent Extensions**
- Purpose: TypeScript extensions for Pi Agent integration
- Contains: 4 extension packages
- Key files: Extension source files per package

## Key File Locations

**Entry Points:**
- `cmd/baxi-api/main.go`: HTTP API bootstrap (~78 lines)
- `cmd/baxi-worker/main.go`: Worker bootstrap (~111 lines)
- `cmd/baxi-mcp/main.go`: MCP server bootstrap (~1287 lines — heavy wiring)
- `cmd/baxi-cli/main.go`: CLI dispatcher (~95 lines)

**Configuration:**
- `internal/config/config.go`: Env var config struct
- `go.mod`: Module dependencies (Go 1.23)
- `docker-compose.yml`: PostgreSQL 16 + API + Worker services
- `Makefile`: Build targets, pipeline commands, governance commands

**Core Logic:**
- `internal/api/routes.go`: All API route definitions
- `internal/api/handler_factories.go`: Lazy handler initialization with full dependency wiring
- `internal/pipeline/runner.go`: Pipeline orchestration with per-step transactions
- `internal/decision/engine.go`: LLM decision generation with fallback chain
- `internal/service/decision_service.go`: Decision business orchestration
- `internal/action/registry.go`: Action whitelist + YAML config
- `internal/ontology/registry.go`: AIP object schema registry (DB + YAML fallback)

**Testing:**
- `internal/testutil/db.go`: Testcontainer postgres:15-alpine setup
- `frontend/vitest.config.ts`: Frontend test config
- `test/integration/phase7_test.go`: Full pipeline+governance E2E (~485 lines)

## Naming Conventions

**Files:**
- Go: `snake_case.go` for implementation, `snake_case_test.go` for tests
- Go handler files: `handler_*.go` (e.g. `handler_sandbox.go`) or domain name (e.g. `decision.go`)
- Go service files: `{domain}_service.go`
- Go repository files: `repository.go` inside subpackages, `{domain}_repository.go` for flat compat
- TypeScript: `PascalCase.tsx` for components/pages, `camelCase.ts` for utilities
- YAML configs: `snake_case.yml`
- SQL migrations: `NNN_description.sql` (zero-padded sequential)

**Directories:**
- Go packages: lowercase, single word (e.g. `decision`, `governance`, `ontology`)
- Repository subpackages: match domain name (e.g. `repository/decision/`, `repository/governance/`)
- Frontend: lowercase for directories (`pages/`, `components/`, `api/`)

**Types:**
- Go interfaces: noun describing capability (e.g. `DecisionService`, `ActionExecutor`, `Querier`)
- Go structs: PascalCase, often with domain prefix (e.g. `DecisionEngine`, `PipelineRunner`)
- Go test mocks: `mock_test.go` in each package with mock implementations
- TypeScript components: PascalCase (e.g. `Dashboard`, `DecisionReview`)

## Where to Add New Code

**New API Endpoint:**
- Route: `internal/api/routes.go` — add to `/api/v1` group
- Handler: `internal/api/handler/{domain}.go` — implement handler struct + methods
- DTO: `internal/api/dto/{domain}.go` — request/response types
- Factory: `internal/api/handler_factories.go` — add lazy init method
- Tests: `internal/api/handler/{domain}_test.go`

**New Service:**
- Implementation: `internal/service/{domain}_service.go`
- Interface: define at top of the same file (local interface pattern)
- Tests: `internal/service/{domain}_service_test.go`

**New Repository Query:**
- Interface: `internal/repository/interfaces.go` — add method signature
- Implementation: `internal/repository/{domain}/repository.go` — implement with PoolProvider
- Tests: `internal/repository/{domain}/repository_test.go`
- Mock: `internal/repository/{domain}/mock_test.go` — add mock methods

**New Pipeline Step:**
- Implementation: `internal/pipeline/steps/{step_name}.go` — implement `pipeline.Step`
- Registration: `internal/api/handler_factories.go` — add to pipelineSteps slice
- Tests: `internal/pipeline/steps/{step_name}_test.go`

**New MCP Tool:**
- Handler: `internal/mcp/tools_{domain}.go` — implement `mcp.NewTool` + handler
- Registration: `internal/mcp/server.go` — add `register*Tools()` call
- Interface: `internal/mcp/interfaces.go` — add service interface if needed
- Wiring: `cmd/baxi-mcp/main.go` — wire dependency into `NewServer`
- Tests: `internal/mcp/server_test.go` — update expectedTools whitelist

**New Governance Rule:**
- Config: `config/{rule_type}.yml` — add YAML definition
- Loader: `internal/configloader/` — extend if new config type
- Engine: `internal/governance/` — add evaluation logic
- Tests: `internal/governance/{rule}_test.go`

**New Frontend Page:**
- Page: `frontend/src/pages/{PageName}.tsx`
- Route: `frontend/src/App.tsx` — add `<Route>`
- API: `frontend/src/api/{domain}.ts` — add endpoint function
- Tests: `frontend/src/pages/__tests__/{PageName}.test.tsx`

**New Adapter (Channel):**
- Implementation: `internal/adapter/{channel}.go` — implement `action.ActionExecutor`
- Config: `internal/adapter/domain.go` — add Config struct
- Tests: `internal/adapter/{channel}_test.go`

## Special Directories

**`internal/repository/*/mock_test.go`:**
- Purpose: Mock implementations for repository interfaces
- Generated: No — hand-written
- Committed: Yes

**`internal/pipeline/steps/`:**
- Purpose: Pipeline step implementations
- Generated: No
- Committed: Yes
- Note: Step order is hardcoded in `handler_factories.go` pipelineSteps slice

**`config/`:**
- Purpose: YAML governance/alert/action configs
- Generated: No
- Committed: Yes
- Note: Parsed at startup by `configloader` and `action.NewActionRegistry`

**`migrations/`:**
- Purpose: Goose SQL migrations
- Generated: No
- Committed: Yes
- Note: Sequential numbering. Use `goose create` for new migrations.

**`test/`:**
- Purpose: E2E tests with build constraints
- Generated: No
- Committed: Yes
- Note: Requires `-tags=integration` to run. Uses testcontainers.

**`pi-extension/`:**
- Purpose: Pi Agent TypeScript extensions
- Generated: No
- Committed: Yes
- Note: Separate npm package from frontend. Built independently.

**`data/raw/`:**
- Purpose: Raw CSV inputs for pipeline
- Generated: No
- Committed: No (gitignored)
- Note: Pipeline expects specific CSV schemas. See `internal/ingest/csv_loader.go`.

**`logs/`:**
- Purpose: Runtime log output
- Generated: Yes (by application)
- Committed: No (gitignored)

**`outputs/`:**
- Purpose: Generated charts, tables, validation reports
- Generated: Yes (by pipeline)
- Committed: No (gitignored)

---

*Structure analysis: 2026-06-03*
