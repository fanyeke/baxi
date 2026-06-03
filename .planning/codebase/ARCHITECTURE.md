# Architecture

**Analysis Date:** 2026-06-03

## System Overview

Baxi is a multi-language governance + analytics platform with a Go backend (API, pipeline, decision engine, MCP server) and a React 19 frontend SPA. The system migrated from Python/SQLite to Go/PostgreSQL. All Python code has been removed.

```text
┌─────────────────────────────────────────────────────────────┐
│                      React 19 Frontend                       │
│              (Vite, TanStack Query, Radix UI)               │
│                    `frontend/src/`                           │
├──────────────────┬──────────────────┬───────────────────────┤
│   HTTP API       │   MCP Server     │    CLI Tools          │
│  `cmd/baxi-api`  │  `cmd/baxi-mcp`  │   `cmd/baxi-cli`      │
│   `:8080`        │   stdio          │   subcommands         │
└────────┬─────────┴────────┬─────────┴──────────┬────────────┘
         │                  │                     │
         ▼                  ▼                     ▼
┌─────────────────────────────────────────────────────────────┐
│                    Go Backend Core (`internal/`)             │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐   │
│  │  API     │ │ Service  │ │ Decision │ │   Pipeline   │   │
│  │Handlers  │ │  Layer   │ │  Engine  │ │   Runner     │   │
│  │`api/`    │ │`service/ │ │`decision/│ │ `pipeline/`  │   │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └──────┬───────┘   │
│       │            │            │              │            │
│       └────────────┴─────┬──────┴──────────────┘            │
│                          ▼                                  │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐   │
│  │Repository│ │  Action  │ │Governance│ │   Ontology   │   │
│  │  Layer   │ │ Registry │ │  Rules   │ │    V2        │   │
│  │`repository/`│`action/` │ │`governance/`│`ontology/`  │   │
│  └────┬─────┘ └──────────┘ └──────────┘ └──────────────┘   │
│       │                                                     │
│       ▼                                                     │
│  ┌─────────────────────────────────────────────────────┐   │
│  │         PostgreSQL 15 (pgx/v5, goose migrations)     │   │
│  │         `migrations/` — 29 SQL migration files        │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
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

**Overall:** Layered architecture with clean-ish separation between API, service, and repository layers. Heavy use of Go interfaces for testability.

**Key Characteristics:**
- Interface-driven design: handlers define narrow service interfaces, services define narrow repository interfaces
- Lazy initialization: API server handlers are created on first route access (`handler_factories.go`)
- Per-step transactions: Pipeline runner wraps each step in its own `pgx.Tx`
- Feature flags: `internal/feature/flags.go` gates v2 context builder and ontology behavior
- Config-driven governance: 28 YAML files in `config/` loaded at startup
- Provider pattern: LLM providers (OpenAI, rule-based) are swappable via factory

## Layers

**API/HTTP Layer:**
- Purpose: HTTP request handling, routing, middleware, DTO serialization
- Location: `internal/api/`
- Contains: chi router setup (`routes.go`), 14 handlers (`handler/`), 6 middlewares (`middleware/`), DTOs (`dto/`)
- Depends on: service layer, repository layer (for direct reads in adapters)
- Used by: `cmd/baxi-api/main.go`

**Service Layer:**
- Purpose: Business orchestration, composing repositories and domain logic
- Location: `internal/service/`
- Contains: 11 service files (decision, governance, log, alert, pipeline, task, outbox, status, qoder, feishu, agent_log)
- Depends on: repository interfaces, decision engine, action registry
- Used by: API handlers, MCP server

**Domain/Engine Layer:**
- Purpose: Core business logic, decision generation, pipeline steps, governance rules
- Location: `internal/decision/`, `internal/pipeline/`, `internal/governance/`, `internal/action/`, `internal/alert/`, `internal/review/`, `internal/ontology/`
- Contains: DecisionEngine, ContextBuilder, Pipeline Runner, ActionRegistry, Governance services, Ontology V2
- Depends on: repository layer, LLM providers, config loader
- Used by: Service layer, MCP server, CLI

**Repository Layer:**
- Purpose: Data access with pgx/PostgreSQL
- Location: `internal/repository/`
- Contains: 12 domain subpackages + flat compatibility files, interfaces in `interfaces.go`
- Depends on: `internal/db/postgres.go`, `internal/repository/common/PoolProvider`
- Used by: Service layer, domain engines, handlers

**Infrastructure Layer:**
- Purpose: Database connections, logging, config loading, adapters
- Location: `internal/db/`, `internal/logger/`, `internal/config/`, `internal/configloader/`, `internal/adapter/`, `internal/llm/`, `internal/testutil/`
- Contains: pgx pool wrapper, zap logger, env config struct, YAML config parser, channel adapters, LLM provider abstraction
- Used by: All layers above

**Frontend Layer:**
- Purpose: React 19 SPA for console UI
- Location: `frontend/src/`
- Contains: 13 pages, 5 shared components, API client layer, TanStack Query hooks
- Depends on: Go HTTP API at `/api/v1`

## Data Flow

### Primary Request Path (HTTP API)

1. Request enters via chi router (`internal/api/routes.go:20`)
2. Auth middleware validates bearer token (`internal/api/middleware/auth.go`)
3. Handler extracts DTO from request body (e.g. `internal/api/handler/decision.go:43`)
4. Handler calls service method via interface (e.g. `DecisionService.CreateCaseFromAlert`)
5. Service composes repository calls and domain logic (`internal/service/decision_service.go:95`)
6. Repository executes SQL via pgxpool (`internal/repository/decision/repository.go`)
7. Response flows back through handler → middleware → HTTP response

### Pipeline Execution Path

1. `POST /api/v1/pipeline/run` triggers `pipelineHandler().HandleRun` (`routes.go:73`)
2. Handler delegates to `pipelineRunService.Run()` (`handler_factories.go:645`)
3. `pipeline.Runner.Run()` creates a pipeline_run audit record (`pipeline/runner.go:41`)
4. For each of 7 steps: begin tx → step.Run() → commit/rollback → audit log (`runner.go:68`)
5. Steps: ingest_raw → build_dwd → build_metrics → detect_alerts → generate_recommendations → generate_tasks → create_outbox
6. Final step writes outbox events for dispatch worker to consume

### Decision Flow (LLM → Action)

1. `POST /api/v1/decisions/cases/{id}/decide` triggers `decisionHandler().Decide` (`routes.go:56`)
2. `DecisionService.Decide()` builds context via ContextBuilder (`service/decision_service.go`)
3. `DecisionEngine.GenerateDecision()` calls LLM provider (`decision/engine.go:86`)
4. If valid: save decision, generate proposals, update case status
5. If invalid: repair retry once with same provider, then rule-based fallback
6. Proposals go through review (approve/reject) then action execution
7. Action execution dispatches to channel adapters (Feishu/GitHub/CLI/Manual)

### MCP Tool Invocation Path

1. MCP client (Pi Agent) sends JSON-RPC over stdio to `cmd/baxi-mcp/main.go`
2. `mcp-go` server routes to registered tool handler (`internal/mcp/tools_*.go`)
3. Handler calls service via interface defined in `internal/mcp/interfaces.go`
4. Service/repository layer same as HTTP API path
5. Response serialized as JSON tool result

**State Management:**
- No in-memory state across requests — all state in PostgreSQL
- `pgxpool.Pool` is the only shared mutable resource, managed per-process
- Feature flags read from env at startup (`internal/feature/flags.go`)
- ObjectRegistry caches ontology schema in memory with RWMutex

## Key Abstractions

**Step Interface:**
- Purpose: Pipeline step contract
- Definition: `pipeline/step.go:28`
- Pattern: `Name() string` + `Run(ctx, tx, input) (*StepOutput, error)`

**DecisionProvider Interface:**
- Purpose: Swappable LLM provider (OpenAI, rule-based, disabled)
- Definition: `internal/llm/provider.go`
- Pattern: `GenerateDecision(ctx, LLMSafeContext) (*DecisionOutput, error)`

**ActionExecutor Interface:**
- Purpose: Channel-agnostic action dispatch
- Definition: `internal/action/executor.go`
- Pattern: `Execute(ctx, ActionProposal, dryRun) (*ExecutionResult, error)`
- Implementations: FeishuAdapter, GitHubAdapter, CLIAdapter, ManualAdapter, NoOpExecutor

**PoolProvider:**
- Purpose: Replaces raw `*pgxpool.Pool` passing with injected querier
- Definition: `internal/repository/common/pool.go:25`
- Pattern: Embeddable struct with Query/QueryRow/Exec/Begin methods
- Used by: all repository subpackages

**LLMSafeContext:**
- Purpose: Structured input for LLM decision generation
- Definition: `internal/llm/provider.go`
- Contains: Trigger info, object context, governance info, allowed/forbidden actions, enriched objects

**ActionProposal:**
- Purpose: Represents a proposed action from a decision
- Definition: `internal/action/proposal_service.go`
- Lifecycle: draft → pending_review → approved → executing → completed

## Entry Points

**baxi-api:**
- Location: `cmd/baxi-api/main.go`
- Triggers: OS startup, `make api`, Docker container
- Responsibilities: Load config, connect DB, start chi HTTP server on :8080, graceful shutdown

**baxi-worker:**
- Location: `cmd/baxi-worker/main.go`
- Triggers: OS startup, `make worker`, Docker container
- Responsibilities: Load config, connect DB, run dispatch worker + background worker, handle signals

**baxi-mcp:**
- Location: `cmd/baxi-mcp/main.go`
- Triggers: MCP client invocation (stdio), `make mcp`
- Responsibilities: Wire all 13+ dependencies into MCP server, serve stdio protocol

**baxi-cli:**
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

**What happens:** `repository/interfaces.go` still defines interface methods that take `pool *pgxpool.Pool` as a parameter, even though subpackage implementations use `PoolProvider`.
**Why it's wrong:** Creates dual identity — some callers pass pool explicitly, others use PoolProvider. The flat `*_repository.go` compatibility files delegate to subpackages via `ensureInitialized(pool)` lazy init.
**Do this instead:** Migrate all interface definitions to accept `Querier` or remove the flat compatibility layer entirely. Use `PoolProvider` consistently.

### CLI Logic in package main

**What happens:** `cmd/baxi-cli/` has 6 files (~919 lines) all declaring `package main` with subcommand logic (pipeline, governance, decision, client HTTP helpers).
**Why it's wrong:** Not reusable, not testable in isolation, violates the thin-main.go pattern used by baxi-api and baxi-worker.
**Do this instead:** Move subcommand logic to `internal/cli/` and keep `cmd/baxi-cli/main.go` as a thin wrapper.

### V1/V2 Context Builder Parallel Hierarchies

**What happens:** `decision/context_builder.go` (v1), `decision/context_builder_v2.go`, `decision/context_builder_v3.go`, and `decision/context_builder_recipe.go` all coexist. A `SwitchableContextBuilder` selects between them.
**Why it's wrong:** Dead code risk — v1 may be unused but is still compiled and maintained. Multiple builders with overlapping concerns.
**Do this instead:** Deprecate v1 once v2+recipe is stable. Merge v3 into recipe builder if they serve the same purpose.

### E2E Tests Outside internal/

**What happens:** `test/integration/`, `test/migration/`, `test/security/` are root-level packages that import `baxi/internal/*` by full module path.
**Why it's wrong:** Breaks `go test ./...` isolation. Fragile to internal package renames. Build constraints (`//go:build integration`) required to skip them.
**Do this instead:** Either move E2E tests inside `internal/` as `_test` packages, or make them external black-box tests that use only public APIs.

## Error Handling

**Strategy:** Return errors up the call stack. Handlers convert to structured HTTP error responses.

**Patterns:**
- Domain packages return Go errors with `fmt.Errorf("...: %w", err)` wrapping
- Handler errors use `middleware.BAD_REQUEST`, `middleware.INTERNAL_ERROR` codes
- API error responses: 5-field JSON (`request_id`, `error_code`, `message`, `diagnosis`, `suggested_action`)
- Panic recovery via `apimw.RecoveryMiddleware`
- Pipeline step failures abort the entire run (no partial success)

## Cross-Cutting Concerns

**Logging:**
- Framework: zap (`go.uber.org/zap`)
- Pattern: Structured JSON logging. Each step gets a scoped logger with `zap.String("step", name)`.
- Audit logs: Separate `audit.audit_log` table for business events. CSV audit logs for dispatch worker.

**Validation:**
- Input validation: Handlers decode JSON and check required fields manually
- Schema validation: `internal/llm/schema_validator.go` validates LLM outputs against action schemas
- Governance validation: `internal/governance/` enforces classification and access policies

**Authentication:**
- HTTP API: Bearer token via `API_BEARER_TOKEN` env var, constant-time comparison, min 32 chars
- MCP: No auth at transport level; `BAXI_MCP_USER_ID` and `BAXI_MCP_ROLE` env vars set caller identity
- No RBAC enforcement in middleware beyond token validation

**Tracing/Lineage:**
- Decision lineage: `decision/lineage_service.go` records all state transitions
- Snapshot recorder: Persists LLM input/output/validation per decision
- Request IDs: `apimw.RequestIDMiddleware` propagates or generates `req_<ts>_<8chr>`

---

*Architecture analysis: 2026-06-03*
