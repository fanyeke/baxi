# Phase 1: Go + PostgreSQL Foundation

## TL;DR

> **建立 Go + PostgreSQL + Docker Compose 的最小可运行工程骨架。**
>
> **Deliverables**:
> - `docker-compose.yml` with postgres, api, worker services
> - `migrations/001_init_schemas.sql` creating 7 schemas
> - `cmd/baxi-api/main.go` serving `/api/v1/health`
> - `cmd/baxi-worker/main.go` starting and connecting to PG
> - `Makefile` with up, down, migrate, api, worker, test, fmt
> - `docs/migration/phase-1-foundation.md` documentation
>
> **Estimated Effort**: Medium (new infrastructure from scratch)
> **Parallel Execution**: YES - 4 waves
> **Critical Path**: Wave 1 (infra) → Wave 2 (Go packages) → Wave 3 (binaries + Docker) → Wave FINAL (verification)

---

## Context

### Original Request
建立 Go + PostgreSQL + Docker Compose 的最小可运行工程骨架。完成后具备：PostgreSQL 可启动、Go 可编译、migration 可初始化 schema、API 可提供 health endpoint、Worker 可启动、Makefile 统一管理命令。不修改旧 Python 业务逻辑。

### Interview Summary
**Key Discussions**:
- Go module path: `baxi` (confirmed by user)
- Worker behavior: connect to DB, log, then sleep forever (confirmed)
- Health endpoint: static JSON, no DB ping in Phase 1 (confirmed)
- Tech stack: Go 1.23, PostgreSQL 16, chi, pgx, goose, zap (confirmed)

**Research Findings**:
- Existing project: Python-based (pyproject.toml, pipeline/, api/, services/, adapters/)
- No existing Go infrastructure (no go.mod, no Docker, no docker-compose, no Makefile)
- `sql/migrations/` exists for Python/SQLite — Go migrations will use root `migrations/`
- goose + pgx requires `stdlib.OpenDBFromPool()` bridge for `database/sql` compatibility

### Metis Review
**Identified Gaps** (addressed):
- **Gap**: goose + pgx stdlib bridge footgun → Resolved: plan includes `internal/db/postgres.go` with explicit bridge
- **Gap**: Port conflict with Python FastAPI → Resolved: Go API uses 8080
- **Gap**: Missing `.dockerignore` → Resolved: included in Wave 1
- **Gap**: Missing Go patterns in `.gitignore` → Resolved: included in Wave 4
- **Gap**: Worker DB connectivity validation → Resolved: worker connects to PG before sleeping
- **Gap**: Migration idempotency → Resolved: `CREATE SCHEMA IF NOT EXISTS`

---

## Work Objectives

### Core Objective
建立最小可运行的 Go + PostgreSQL + Docker Compose 工程骨架，为后续迁移阶段提供基础设施基础。

### Concrete Deliverables
- `docker-compose.yml` (postgres + api + worker)
- `Dockerfile.api` + `Dockerfile.worker`
- `.env.example` with all required environment variables
- `migrations/001_init_schemas.sql` (7 schemas)
- `go.mod` + `go.sum`
- `cmd/baxi-api/main.go`
- `cmd/baxi-worker/main.go`
- `internal/config/config.go`
- `internal/logger/logger.go`
- `internal/db/postgres.go`
- `internal/api/server.go`
- `internal/api/health.go`
- `internal/worker/worker.go`
- `Makefile`
- `.dockerignore`
- `docs/migration/phase-1-foundation.md`

### Definition of Done
- [ ] `docker compose up -d postgres` starts healthy container
- [ ] `make migrate` creates all 7 schemas
- [ ] `go test ./...` compiles without errors
- [ ] `make api` + `curl http://localhost:8080/api/v1/health` returns correct JSON
- [ ] `make worker` starts without panic and logs DB connection
- [ ] `git diff --name-only` shows only new files, no old Python modified

### Must Have
- PostgreSQL 16 via Docker Compose with healthcheck
- 7 schemas created via goose migration
- Go module initialized with module path `baxi`
- baxi-api serving `/api/v1/health`
- baxi-worker starting and connecting to PostgreSQL
- Makefile with up, down, migrate, api, worker, test, fmt
- Documentation in `docs/migration/phase-1-foundation.md`

### Must NOT Have (Guardrails)
- No modification to existing Python files (pipeline/, api/, services/, adapters/, config/*.yml)
- No React frontend changes
- No CSV pipeline migration
- No SQLite table migration
- No business table creation beyond schemas
- No LLM integration
- No FastAPI endpoint rewrite
- No graceful shutdown logic (Phase 1 keep minimal)
- No multi-stage Docker builds (single-stage for speed)

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** - ALL verification is agent-executed. No exceptions.

### Test Decision
- **Infrastructure exists**: NO (Go project has no tests yet)
- **Automated tests**: None (Phase 1 is infrastructure scaffolding; tests added in Phase 2+)
- **Framework**: N/A
- **Agent-Executed QA**: ALWAYS - Every task includes executable QA scenarios

### QA Policy
Every task MUST include agent-executed QA scenarios. Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **Docker/Infra**: Use Bash (docker compose, psql)
- **API**: Use Bash (curl) - Send requests, assert status + response fields
- **Go compilation**: Use Bash (go build, go vet, go test)
- **Worker**: Use Bash (timeout + process check)

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Start Immediately - Docker + PostgreSQL foundation):
├── Task 1: docker-compose.yml + .env.example + .dockerignore
├── Task 2: Migration SQL (7 schemas)
└── Task 3: .gitignore Go patterns

Wave 2 (After Wave 1 - Go internal packages, MAX PARALLEL):
├── Task 4: go.mod + dependencies
├── Task 5: internal/config/config.go
├── Task 6: internal/logger/logger.go
├── Task 7: internal/db/postgres.go (pgx + goose bridge)
├── Task 8: internal/api/server.go + health.go
└── Task 9: internal/worker/worker.go

Wave 3 (After Wave 2 - Entry points + containerization):
├── Task 10: cmd/baxi-api/main.go
├── Task 11: cmd/baxi-worker/main.go
├── Task 12: Dockerfile.api + Dockerfile.worker
└── Task 13: Makefile

Wave 4 (After Wave 3 - Documentation + cleanup):
├── Task 14: docs/migration/phase-1-foundation.md
└── Task 15: Final formatting and tidy

Wave FINAL (After ALL tasks - 4 parallel reviews, then user okay):
├── Task F1: Plan compliance audit (oracle)
├── Task F2: Code quality review (unspecified-high)
├── Task F3: Real manual QA (unspecified-high)
├── Task F4: Scope fidelity check (deep)
-> Present results -> Get explicit user okay
```

### Dependency Matrix

| Task | Depends On | Blocks |
|------|-----------|--------|
| 1 (docker-compose) | - | F1-F4 |
| 2 (migration) | - | 4 (goose in go.mod) |
| 3 (.gitignore) | - | F1-F4 |
| 4 (go.mod) | - | 5, 6, 7, 8, 9, 10, 11 |
| 5 (config) | 4 | 10, 11 |
| 6 (logger) | 4 | 7, 8, 9, 10, 11 |
| 7 (db) | 4, 6 | 10, 11 |
| 8 (api) | 4, 6 | 10 |
| 9 (worker) | 4, 6, 7 | 11 |
| 10 (api main) | 4, 5, 6, 7, 8 | F1-F4 |
| 11 (worker main) | 4, 5, 6, 7, 9 | F1-F4 |
| 12 (Dockerfiles) | 10, 11 | F1-F4 |
| 13 (Makefile) | 1, 2, 10, 11, 12 | F1-F4 |
| 14 (docs) | ALL | F1-F4 |
| 15 (tidy) | ALL | F1-F4 |

### Agent Dispatch Summary

- **Wave 1**: 3 tasks → `quick` (infrastructure files)
- **Wave 2**: 6 tasks → `quick` to `unspecified-high` (Go package development)
- **Wave 3**: 4 tasks → `quick` (entry points, Dockerfiles, Makefile)
- **Wave 4**: 2 tasks → `writing` (documentation, cleanup)
- **FINAL**: 4 tasks → `oracle`, `unspecified-high`, `unspecified-high`, `deep`

---

## TODOs

- [x] 1. Docker Compose + Environment Configuration

  **What to do**:
  - Create `docker-compose.yml` with three services: postgres, api, worker
  - postgres: image postgres:16, container_name baxi-postgres, port 5432:5432, named volume, healthcheck
  - api: build from Dockerfile.api, depends_on postgres condition: service_healthy, port 8080:8080
  - worker: build from Dockerfile.worker, depends_on postgres condition: service_healthy
  - Create `.env.example` with all required environment variables (POSTGRES_HOST, POSTGRES_PORT, POSTGRES_DB, POSTGRES_USER, POSTGRES_PASSWORD, DATABASE_URL, API_PORT, LOG_LEVEL)
  - Create `.dockerignore` to exclude Python code, venv, node_modules, .git, etc.
  - Set TZ=UTC on all services

  **Must NOT do**:
  - Do not create Go source files in this task
  - Do not modify existing Python files
  - Do not add business logic to docker-compose

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Infrastructure configuration files with well-defined structure
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2, 3)
  - **Blocks**: Tasks F1-F4
  - **Blocked By**: None

  **References**:
  - `docker-compose.yml` spec: standard Docker Compose v3.8+ format
  - PostgreSQL 16 official image: `postgres:16`

  **Acceptance Criteria**:
  - [ ] `docker-compose.yml` exists and is valid YAML
  - [ ] `.env.example` contains all required variables
  - [ ] `.dockerignore` excludes Python/Node artifacts

  **QA Scenarios**:

  ```
  Scenario: Docker Compose syntax validation
    Tool: Bash
    Preconditions: docker compose CLI available
    Steps:
      1. Run: docker compose config
    Expected Result: Exit code 0, no errors, all services parsed correctly
    Failure Indicators: Exit code != 0, syntax errors
    Evidence: .sisyphus/evidence/task-1-compose-config.txt

  Scenario: PostgreSQL container starts healthy
    Tool: Bash
    Preconditions: Docker daemon running
    Steps:
      1. Run: docker compose up -d postgres
      2. Wait 10 seconds
      3. Run: docker compose ps | grep baxi-postgres
    Expected Result: Container shows "healthy" status
    Failure Indicators: Container shows "unhealthy" or "restarting"
    Evidence: .sisyphus/evidence/task-1-postgres-healthy.txt
  ```

  **Evidence to Capture**:
  - [ ] task-1-compose-config.txt
  - [ ] task-1-postgres-healthy.txt

  **Commit**: YES
  - Message: `chore: add docker postgres foundation`
  - Files: `docker-compose.yml`, `.env.example`, `.dockerignore`

- [x] 2. Database Migration - Schema Initialization

  **What to do**:
  - Create `migrations/001_init_schemas.sql`
  - Content: `CREATE SCHEMA IF NOT EXISTS` for 7 schemas: raw, dwd, mart, ops, gov, ai, audit
  - Ensure idempotency with `IF NOT EXISTS`
  - Install goose CLI (or use `go install` approach)

  **Must NOT do**:
  - Do not create business tables (no raw.olist_orders, etc.)
  - Do not add seed data
  - Do not modify existing `sql/migrations/` directory

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple SQL file creation and migration execution
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 3)
  - **Blocks**: Task 4 (go.mod includes goose)
  - **Blocked By**: None

  **References**:
  - goose documentation: https://github.com/pressly/goose
  - PostgreSQL schema creation syntax

  **Acceptance Criteria**:
  - [ ] `migrations/001_init_schemas.sql` exists with 7 CREATE SCHEMA statements
  - [ ] Running `make migrate` (or goose equivalent) creates all 7 schemas
  - [ ] Running migration twice does not error (idempotent)

  **QA Scenarios**:

  ```
  Scenario: Migration creates all schemas
    Tool: Bash (psql)
    Preconditions: PostgreSQL container running (from Task 1)
    Steps:
      1. Export DATABASE_URL="postgres://baxi:baxi_dev@localhost:5432/baxi?sslmode=disable"
      2. Run: goose -dir migrations postgres "$DATABASE_URL" up
      3. Run: psql "$DATABASE_URL" -c "SELECT schema_name FROM information_schema.schemata WHERE schema_name IN ('raw','dwd','mart','ops','gov','ai','audit') ORDER BY schema_name;"
    Expected Result: All 7 schemas listed, one per line
    Failure Indicators: Missing schemas, SQL errors
    Evidence: .sisyphus/evidence/task-2-schemas-created.txt

  Scenario: Migration idempotency
    Tool: Bash (psql)
    Preconditions: Migration already applied
    Steps:
      1. Run: goose -dir migrations postgres "$DATABASE_URL" up
    Expected Result: Exit code 0, no errors, no duplicate schema creation errors
    Failure Indicators: "schema already exists" errors or non-zero exit
    Evidence: .sisyphus/evidence/task-2-migration-idempotent.txt
  ```

  **Evidence to Capture**:
  - [ ] task-2-schemas-created.txt
  - [ ] task-2-migration-idempotent.txt

  **Commit**: YES (grouped with Task 1)
  - Message: `chore: add docker postgres foundation`
  - Files: `migrations/001_init_schemas.sql`

- [x] 3. Update .gitignore for Go Patterns

  **What to do**:
  - Add Go-specific patterns to existing `.gitignore`:
    - `*.exe`, `*.dll`, `*.so`, `*.dylib`
    - `/baxi-api`, `/baxi-worker` (compiled binaries)
    - `*.test`, `*.out`
    - `vendor/`
  - Keep all existing Python/Node patterns intact

  **Must NOT do**:
  - Do not remove existing ignore patterns
  - Do not add IDE-specific patterns (keep minimal)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Single file edit
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2)
  - **Blocks**: F1-F4
  - **Blocked By**: None

  **References**:
  - Standard Go .gitignore patterns

  **Acceptance Criteria**:
  - [ ] `.gitignore` contains Go binary patterns
  - [ ] `git check-ignore -v baxi-api` returns match
  - [ ] Existing patterns still present

  **QA Scenarios**:

  ```
  Scenario: Go binaries are ignored
    Tool: Bash
    Preconditions: .gitignore updated
    Steps:
      1. Run: touch baxi-api baxi-worker
      2. Run: git check-ignore -v baxi-api
      3. Run: rm baxi-api baxi-worker
    Expected Result: git check-ignore shows .gitignore as matching rule
    Failure Indicators: No match found, binary would be tracked
    Evidence: .sisyphus/evidence/task-3-gitignore.txt
  ```

  **Evidence to Capture**:
  - [ ] task-3-gitignore.txt

  **Commit**: NO (group with Task 15 or docs commit)
  - Files: `.gitignore`

- [x] 4. Initialize Go Module and Dependencies

  **What to do**:
  - Run `go mod init baxi` to create `go.mod`
  - Add dependencies via `go get`:
    - `github.com/go-chi/chi/v5`
    - `github.com/jackc/pgx/v5`
    - `github.com/pressly/goose/v3`
    - `go.uber.org/zap`
    - `github.com/joho/godotenv` (optional, for .env file loading)
  - Run `go mod tidy` to resolve and lock dependencies
  - Verify `go.sum` is generated

  **Must NOT do**:
  - Do not add unnecessary dependencies
  - Do not use replace directives unless absolutely needed

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Standard Go module initialization
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 5, 6, 7, 8, 9)
  - **Blocks**: Tasks 5, 6, 7, 8, 9, 10, 11
  - **Blocked By**: None (can start immediately)

  **References**:
  - `go.mod` spec: https://go.dev/ref/mod

  **Acceptance Criteria**:
  - [ ] `go.mod` exists with module path `baxi`
  - [ ] `go.sum` exists with checksums
  - [ ] `go mod verify` passes
  - [ ] `go list -m all` shows all expected dependencies

  **QA Scenarios**:

  ```
  Scenario: Go module is valid
    Tool: Bash
    Preconditions: Go 1.23+ installed
    Steps:
      1. Run: go mod verify
      2. Run: go mod tidy && git diff --exit-code go.mod go.sum
    Expected Result: Both commands exit 0, no diff after tidy
    Failure Indicators: Non-zero exit, checksum failures, untidy dependencies
    Evidence: .sisyphus/evidence/task-4-go-mod.txt
  ```

  **Evidence to Capture**:
  - [ ] task-4-go-mod.txt

  **Commit**: YES (group with Wave 2)
  - Message: `feat: add go api and worker skeleton`
  - Files: `go.mod`, `go.sum`

- [x] 5. Configuration Package (internal/config)

  **What to do**:
  - Create `internal/config/config.go`
  - Define `Config` struct with fields:
    - `DatabaseURL string`
    - `APIPort string`
    - `LogLevel string`
  - Implement `Load() (*Config, error)` function:
    - Read from environment variables
    - Use sensible defaults (DATABASE_URL default, API_PORT=8080, LOG_LEVEL=info)
    - Validate required fields (DatabaseURL must not be empty)
  - Keep it minimal — no Viper, no complex validation

  **Must NOT do**:
  - Do not add config file parsing (YAML/JSON/TOML)
  - Do not add secret management (vault, etc.)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple struct + env var parsing
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 4, 6, 7, 8, 9)
  - **Blocks**: Tasks 10, 11
  - **Blocked By**: Task 4

  **References**:
  - `os.Getenv` standard library
  - Go struct initialization patterns

  **Acceptance Criteria**:
  - [ ] `internal/config/config.go` compiles
  - [ ] `config.Load()` returns valid config with correct defaults
  - [ ] Missing required env var returns error

  **QA Scenarios**:

  ```
  Scenario: Config loads with defaults
    Tool: Bash (go test)
    Preconditions: Go module initialized
    Steps:
      1. Run: go test ./internal/config/... (if tests exist)
      2. Or verify compilation: go build ./internal/config
    Expected Result: Compilation succeeds
    Failure Indicators: Compilation errors
    Evidence: .sisyphus/evidence/task-5-config-compile.txt
  ```

  **Evidence to Capture**:
  - [ ] task-5-config-compile.txt

  **Commit**: YES (group with Wave 2)
  - Files: `internal/config/config.go`

- [x] 6. Logger Package (internal/logger)

  **What to do**:
  - Create `internal/logger/logger.go`
  - Initialize zap logger with configurable log level
  - Provide `New(level string) (*zap.Logger, error)` constructor
  - Return structured JSON logger
  - Support levels: debug, info, warn, error
  - Default to info if invalid level provided

  **Must NOT do**:
  - Do not use global logger (zap.L()) — dependency injection only
  - Do not add log rotation or file output (stdout only)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Standard zap initialization
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 4, 5, 7, 8, 9)
  - **Blocks**: Tasks 7, 8, 9, 10, 11
  - **Blocked By**: Task 4

  **References**:
  - zap documentation: https://pkg.go.dev/go.uber.org/zap

  **Acceptance Criteria**:
  - [ ] `internal/logger/logger.go` compiles
  - [ ] `logger.New("info")` returns valid logger
  - [ ] `logger.New("invalid")` defaults to info (no panic)

  **QA Scenarios**:

  ```
  Scenario: Logger initialization
    Tool: Bash (go test)
    Preconditions: Go module initialized
    Steps:
      1. Run: go build ./internal/logger
    Expected Result: Compilation succeeds
    Failure Indicators: Compilation errors
    Evidence: .sisyphus/evidence/task-6-logger-compile.txt
  ```

  **Evidence to Capture**:
  - [ ] task-6-logger-compile.txt

  **Commit**: YES (group with Wave 2)
  - Files: `internal/logger/logger.go`

- [x] 7. Database Package (internal/db)

  **What to do**:
  - Create `internal/db/postgres.go`
  - Implement `NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error)`
    - Parse config with `pgxpool.ParseConfig(databaseURL)`
    - Create pool with `pgxpool.NewWithConfig(ctx, config)`
    - Ping database to verify connection
  - Implement `NewStdDB(pool *pgxpool.Pool) (*sql.DB, error)`
    - Use `stdlib.OpenDBFromPool(pool)` for goose compatibility
    - This is the critical bridge: pgx pool → database/sql
  - Provide `Close()` method on a wrapper struct or document cleanup

  **Must NOT do**:
  - Do not use `database/sql` as primary interface — only for goose bridge
  - Do not add connection pool tuning (defaults are fine for Phase 1)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Critical infrastructure, goose+pgx bridge is a known footgun
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 4, 5, 6, 8, 9)
  - **Blocks**: Tasks 10, 11
  - **Blocked By**: Tasks 4, 6

  **References**:
  - pgx pool documentation: https://pkg.go.dev/github.com/jackc/pgx/v5/pgxpool
  - pgx stdlib bridge: `github.com/jackc/pgx/v5/stdlib`
  - goose requires `*sql.DB`: https://github.com/pressly/goose

  **Acceptance Criteria**:
  - [ ] `internal/db/postgres.go` compiles
  - [ ] `NewPool()` returns connected pool
  - [ ] `NewStdDB()` returns valid `*sql.DB` via stdlib bridge

  **QA Scenarios**:

  ```
  Scenario: Database package compiles
    Tool: Bash
    Preconditions: Go module initialized, PostgreSQL running
    Steps:
      1. Run: go build ./internal/db
    Expected Result: Compilation succeeds
    Failure Indicators: Compilation errors, missing imports
    Evidence: .sisyphus/evidence/task-7-db-compile.txt

  Scenario: DB connection works
    Tool: Bash (go run small test)
    Preconditions: PostgreSQL running with baxi database
    Steps:
      1. Create temporary test program that calls db.NewPool and pool.Ping
      2. Run with DATABASE_URL set
    Expected Result: Program exits 0, no connection errors
    Failure Indicators: Connection refused, auth errors
    Evidence: .sisyphus/evidence/task-7-db-connect.txt
  ```

  **Evidence to Capture**:
  - [ ] task-7-db-compile.txt
  - [ ] task-7-db-connect.txt

  **Commit**: YES (group with Wave 2)
  - Files: `internal/db/postgres.go`

- [x] 8. API Server and Health Handler (internal/api)

  **What to do**:
  - Create `internal/api/server.go`
    - Define `Server` struct with `chi.Router`, `*zap.Logger`, `*pgxpool.Pool`
    - Implement `New(logger, pool)` constructor
    - Implement `SetupRoutes()` — add middleware (request ID, logger)
    - Implement `Start(addr string) error`
  - Create `internal/api/health.go`
    - Define `HealthResponse` struct: `Status string`, `Service string`
    - Implement `handleHealth(w, r)` — return static JSON `{"status":"ok","service":"baxi-api"}`
    - Register at `GET /api/v1/health`
  - Do NOT connect to DB in health handler (Phase 1 static response)

  **Must NOT do**:
  - Do not add authentication middleware
  - Do not add CORS (not needed for Phase 1)
  - Do not add request logging middleware (keep minimal)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Standard chi router setup
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 4, 5, 6, 7, 9)
  - **Blocks**: Task 10
  - **Blocked By**: Tasks 4, 6

  **References**:
  - chi documentation: https://go-chi.io/#/README
  - chi middleware: `middleware.Heartbeat` (for Docker healthcheck, not API health)

  **Acceptance Criteria**:
  - [ ] `internal/api/server.go` and `health.go` compile
  - [ ] `GET /api/v1/health` returns correct JSON structure
  - [ ] HTTP status is 200

  **QA Scenarios**:

  ```
  Scenario: Health endpoint returns correct JSON
    Tool: Bash (curl)
    Preconditions: API server running
    Steps:
      1. Start API: go run ./cmd/baxi-api (in background)
      2. Wait 1 second
      3. Run: curl -s http://localhost:8080/api/v1/health
      4. Stop API
    Expected Result: Response body is {"status":"ok","service":"baxi-api"}
    Failure Indicators: Wrong JSON structure, wrong status code, connection refused
    Evidence: .sisyphus/evidence/task-8-health-endpoint.txt
  ```

  **Evidence to Capture**:
  - [ ] task-8-health-endpoint.txt

  **Commit**: YES (group with Wave 2)
  - Files: `internal/api/server.go`, `internal/api/health.go`

- [x] 9. Worker Package (internal/worker)

  **What to do**:
  - Create `internal/worker/worker.go`
  - Define `Worker` struct with `*zap.Logger`, `*pgxpool.Pool`
  - Implement `New(logger, pool)` constructor
  - Implement `Run(ctx context.Context) error`:
    - Log "baxi-worker started" with structured fields
    - Verify DB connection with `pool.Ping(ctx)`
    - Log "connected to database" on success
    - Block on `<-ctx.Done()` — sleep forever until signal
  - Do NOT implement outbox polling, pipeline execution, or any business logic

  **Must NOT do**:
  - Do not add outbox poller
  - Do not add adapter dispatch
  - Do not add retry logic
  - Do not add dead letter queue

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple struct + context wait
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 4, 5, 6, 7, 8)
  - **Blocks**: Task 11
  - **Blocked By**: Tasks 4, 6, 7

  **References**:
  - Context cancellation patterns in Go

  **Acceptance Criteria**:
  - [ ] `internal/worker/worker.go` compiles
  - [ ] `Run()` starts and blocks on context
  - [ ] Logs show startup message

  **QA Scenarios**:

  ```
  Scenario: Worker package compiles
    Tool: Bash
    Preconditions: Go module initialized
    Steps:
      1. Run: go build ./internal/worker
    Expected Result: Compilation succeeds
    Failure Indicators: Compilation errors
    Evidence: .sisyphus/evidence/task-9-worker-compile.txt
  ```

  **Evidence to Capture**:
  - [ ] task-9-worker-compile.txt

  **Commit**: YES (group with Wave 2)
  - Files: `internal/worker/worker.go`

- [x] 10. API Entry Point (cmd/baxi-api/main.go)

  **What to do**:
  - Create `cmd/baxi-api/main.go`
  - Main flow:
    1. Create context with signal handling (os.Interrupt, syscall.SIGTERM)
    2. Load config via `internal/config`
    3. Initialize logger via `internal/logger`
    4. Connect to PostgreSQL via `internal/db`
    5. Create API server via `internal/api`
    6. Start server in goroutine
    7. Block on signal, then graceful shutdown (server.Shutdown with timeout)
  - Log startup and shutdown messages
  - Exit with code 0 on clean shutdown, 1 on error

  **Must NOT do**:
  - Do not add CLI flags (keep minimal)
  - Do not add systemd/socket activation

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Wiring existing packages together
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 11, 12, 13)
  - **Blocks**: F1-F4
  - **Blocked By**: Tasks 4, 5, 6, 7, 8

  **References**:
  - `internal/config/config.go`
  - `internal/logger/logger.go`
  - `internal/db/postgres.go`
  - `internal/api/server.go`

  **Acceptance Criteria**:
  - [ ] `cmd/baxi-api/main.go` compiles
  - [ ] `go run ./cmd/baxi-api` starts server on port 8080
  - [ ] Server responds to `/api/v1/health`

  **QA Scenarios**:

  ```
  Scenario: API binary compiles and runs
    Tool: Bash
    Preconditions: All dependencies available, PostgreSQL running
    Steps:
      1. Run: go build -o baxi-api ./cmd/baxi-api
      2. Run: DATABASE_URL="postgres://baxi:baxi_dev@localhost:5432/baxi?sslmode=disable" ./baxi-api &
      3. Wait 2 seconds
      4. Run: curl -s http://localhost:8080/api/v1/health
      5. Run: kill %1
      6. Run: rm baxi-api
    Expected Result: curl returns {"status":"ok","service":"baxi-api"}
    Failure Indicators: Compilation error, connection refused, wrong response
    Evidence: .sisyphus/evidence/task-10-api-binary.txt
  ```

  **Evidence to Capture**:
  - [ ] task-10-api-binary.txt

  **Commit**: YES (group with Wave 3)
  - Message: `feat: add go api and worker skeleton`
  - Files: `cmd/baxi-api/main.go`

- [x] 11. Worker Entry Point (cmd/baxi-worker/main.go)

  **What to do**:
  - Create `cmd/baxi-worker/main.go`
  - Main flow:
    1. Create context with signal handling
    2. Load config via `internal/config`
    3. Initialize logger via `internal/logger`
    4. Connect to PostgreSQL via `internal/db`
    5. Create worker via `internal/worker`
    6. Call `worker.Run(ctx)`
    7. Block until signal, then cancel context
  - Log startup message: "baxi-worker started"
  - Exit with code 0 on clean shutdown

  **Must NOT do**:
  - Do not add CLI flags
  - Do not add background goroutines beyond worker.Run

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Wiring existing packages together
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 10, 12, 13)
  - **Blocks**: F1-F4
  - **Blocked By**: Tasks 4, 5, 6, 7, 9

  **References**:
  - `internal/config/config.go`
  - `internal/logger/logger.go`
  - `internal/db/postgres.go`
  - `internal/worker/worker.go`

  **Acceptance Criteria**:
  - [ ] `cmd/baxi-worker/main.go` compiles
  - [ ] `go run ./cmd/baxi-worker` starts and logs "baxi-worker started"
  - [ ] Process exits cleanly on SIGINT

  **QA Scenarios**:

  ```
  Scenario: Worker binary compiles and runs
    Tool: Bash
    Preconditions: All dependencies available, PostgreSQL running
    Steps:
      1. Run: go build -o baxi-worker ./cmd/baxi-worker
      2. Run: timeout 5s DATABASE_URL="postgres://baxi:baxi_dev@localhost:5432/baxi?sslmode=disable" ./baxi-worker
      3. Run: rm baxi-worker
    Expected Result: Output contains "baxi-worker started", exits 0 within 5s (timeout kills it)
    Failure Indicators: Compilation error, panic, missing log message
    Evidence: .sisyphus/evidence/task-11-worker-binary.txt
  ```

  **Evidence to Capture**:
  - [ ] task-11-worker-binary.txt

  **Commit**: YES (group with Wave 3)
  - Files: `cmd/baxi-worker/main.go`

- [x] 12. Dockerfiles for API and Worker

  **What to do**:
  - Create `Dockerfile.api`:
    - Base image: `golang:1.23-alpine`
    - Set `CGO_ENABLED=0`
    - Copy go.mod, go.sum, download dependencies
    - Copy source code
    - Build: `go build -o /app/baxi-api ./cmd/baxi-api`
    - Final stage (optional single-stage for Phase 1): `FROM alpine:latest`
    - Or keep single-stage for simplicity
    - EXPOSE 8080
    - CMD ["/app/baxi-api"]
  - Create `Dockerfile.worker`:
    - Similar structure
    - Build: `go build -o /app/baxi-worker ./cmd/baxi-worker`
    - No exposed ports
    - CMD ["/app/baxi-worker"]
  - Both should use `.dockerignore`

  **Must NOT do**:
  - Do not add multi-stage builds (keep single-stage for Phase 1)
  - Do not add healthcheck in Dockerfile (docker-compose handles it)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Standard Go Docker patterns
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 10, 11, 13)
  - **Blocks**: F1-F4
  - **Blocked By**: Tasks 10, 11

  **References**:
  - Go Docker best practices
  - `.dockerignore` from Task 1

  **Acceptance Criteria**:
  - [ ] `Dockerfile.api` builds successfully
  - [ ] `Dockerfile.worker` builds successfully
  - [ ] Both images run without errors

  **QA Scenarios**:

  ```
  Scenario: API Docker image builds and runs
    Tool: Bash
    Preconditions: Docker daemon running
    Steps:
      1. Run: docker build -f Dockerfile.api -t baxi-api:test .
      2. Run: docker run --rm -e DATABASE_URL="postgres://baxi:baxi_dev@host.docker.internal:5432/baxi?sslmode=disable" -p 8080:8080 baxi-api:test &
      3. Wait 3 seconds
      4. Run: curl -s http://localhost:8080/api/v1/health
      5. Run: docker stop $(docker ps -q --filter ancestor=baxi-api:test)
      6. Run: docker rmi baxi-api:test
    Expected Result: curl returns correct health JSON
    Failure Indicators: Build failure, runtime error, wrong response
    Evidence: .sisyphus/evidence/task-12-docker-api.txt

  Scenario: Worker Docker image builds
    Tool: Bash
    Preconditions: Docker daemon running
    Steps:
      1. Run: docker build -f Dockerfile.worker -t baxi-worker:test .
      2. Run: docker rmi baxi-worker:test
    Expected Result: Build succeeds
    Failure Indicators: Build failure
    Evidence: .sisyphus/evidence/task-12-docker-worker.txt
  ```

  **Evidence to Capture**:
  - [ ] task-12-docker-api.txt
  - [ ] task-12-docker-worker.txt

  **Commit**: YES (group with Wave 3)
  - Files: `Dockerfile.api`, `Dockerfile.worker`

- [x] 13. Makefile with Standard Commands

  **What to do**:
  - Create `Makefile` with targets:
    - `up`: `docker compose up -d postgres`
    - `down`: `docker compose down`
    - `migrate`: `goose -dir migrations postgres "$DATABASE_URL" up`
    - `api`: `go run ./cmd/baxi-api`
    - `worker`: `go run ./cmd/baxi-worker`
    - `test`: `go test ./...`
    - `fmt`: `go fmt ./...`
    - `build`: build both binaries
    - `vet`: `go vet ./...`
    - `tidy`: `go mod tidy`
  - Set `DATABASE_URL` default: `postgres://baxi:baxi_dev@localhost:5432/baxi?sslmode=disable`
  - All targets should be `.PHONY`

  **Must NOT do**:
  - Do not add complex build scripts
  - Do not add cross-compilation targets
  - Do not modify existing Python Makefile patterns

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Standard Makefile patterns
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 10, 11, 12)
  - **Blocks**: F1-F4
  - **Blocked By**: Tasks 1, 2, 10, 11, 12

  **References**:
  - GNU Make documentation
  - User specification section 九

  **Acceptance Criteria**:
  - [ ] `Makefile` exists with all required targets
  - [ ] `make up` starts PostgreSQL
  - [ ] `make test` compiles and runs tests
  - [ ] `make fmt` formats all Go files

  **QA Scenarios**:

  ```
  Scenario: Makefile targets work
    Tool: Bash
    Preconditions: PostgreSQL running (from make up)
    Steps:
      1. Run: make fmt
      2. Run: make test
      3. Run: make vet
    Expected Result: All commands exit 0
    Failure Indicators: Non-zero exit, missing targets
    Evidence: .sisyphus/evidence/task-13-makefile.txt
  ```

  **Evidence to Capture**:
  - [ ] task-13-makefile.txt

  **Commit**: YES (group with Wave 3)
  - Files: `Makefile`

- [x] 14. Phase 1 Documentation

  **What to do**:
  - Create `docs/migration/phase-1-foundation.md`
  - Document:
    1. Phase 1 goals and scope
    2. Directory structure added
    3. Local PostgreSQL startup (`make up`)
    4. Migration execution (`make migrate`)
    5. API startup (`make api`)
    6. Worker startup (`make worker`)
    7. Acceptance commands (docker ps, psql, curl)
    8. What's NOT included (CSV pipeline, SQLite, business tables, LLM, FastAPI rewrite, React changes)
    9. Commit history
    10. Next steps (Phase 2 preview)

  **Must NOT do**:
  - Do not document future phases in detail
  - Do not modify existing docs

  **Recommended Agent Profile**:
  - **Category**: `writing`
    - Reason: Documentation creation
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 4 (with Task 15)
  - **Blocks**: F1-F4
  - **Blocked By**: ALL previous tasks

  **References**:
  - User specification section 十一

  **Acceptance Criteria**:
  - [ ] `docs/migration/phase-1-foundation.md` exists
  - [ ] Contains all required sections
  - [ ] Markdown renders correctly

  **QA Scenarios**:

  ```
  Scenario: Documentation exists and is valid
    Tool: Bash
    Preconditions: File created
    Steps:
      1. Run: test -f docs/migration/phase-1-foundation.md
      2. Run: head -n 5 docs/migration/phase-1-foundation.md
    Expected Result: File exists, starts with heading
    Failure Indicators: Missing file, wrong content
    Evidence: .sisyphus/evidence/task-14-docs.txt
  ```

  **Evidence to Capture**:
  - [ ] task-14-docs.txt

  **Commit**: YES
  - Message: `docs: add phase 1 foundation guide`
  - Files: `docs/migration/phase-1-foundation.md`

- [x] 15. Final Formatting and Dependency Tidy

  **What to do**:
  - Run `go fmt ./...` to format all Go files
  - Run `go mod tidy` to clean dependencies
  - Run `go vet ./...` to check for issues
  - Verify `.gitignore` includes Go patterns (from Task 3)
  - Stage `.gitignore` if not already staged
  - Verify no binaries or temp files are tracked

  **Must NOT do**:
  - Do not add new code in this task
  - Do not modify business logic

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Cleanup and formatting
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 4 (with Task 14)
  - **Blocks**: F1-F4
  - **Blocked By**: ALL previous tasks

  **Acceptance Criteria**:
  - [ ] `go fmt ./...` produces no diff
  - [ ] `go mod tidy` produces no diff
  - [ ] `go vet ./...` passes
  - [ ] No untracked binaries in git status

  **QA Scenarios**:

  ```
  Scenario: Code is formatted and clean
    Tool: Bash
    Preconditions: All Go files written
    Steps:
      1. Run: go fmt ./...
      2. Run: git diff --exit-code
      3. Run: go vet ./...
      4. Run: go mod tidy && git diff --exit-code go.mod go.sum
    Expected Result: All commands exit 0
    Failure Indicators: Formatting diffs, vet warnings, untidy dependencies
    Evidence: .sisyphus/evidence/task-15-tidy.txt
  ```

  **Evidence to Capture**:
  - [ ] task-15-tidy.txt

  **Commit**: YES (group with Task 3 .gitignore)
  - Message: `chore: format code and update gitignore`
  - Files: `.gitignore` (if not committed), any formatting fixes

---

## Final Verification Wave

> 4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.
>
> **Do NOT auto-proceed after verification. Wait for user's explicit approval before marking work complete.**

- [x] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, curl endpoint, run command). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist in `.sisyphus/evidence/`. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [x] F2. **Code Quality Review** — `unspecified-high`
  Run `go vet ./...`, `go fmt ./...`, `go mod tidy` (verify no diff), `go test ./...`. Review all changed files for: `as any` (Go equivalent: interface{} abuse), empty catches, `fmt.Println` in prod code, commented-out code, unused imports. Check AI slop: excessive comments, over-abstraction, generic names.
  Output: `Vet [PASS/FAIL] | Fmt [PASS/FAIL] | Tidy [PASS/FAIL] | Tests [N pass/N fail] | Files [N clean/N issues] | VERDICT`

- [x] F3. **Real Manual QA** — `unspecified-high`
  Start from clean state. Execute EVERY QA scenario from EVERY task — follow exact steps, capture evidence. Test cross-task integration (API + DB + migration working together, not isolation). Test edge cases: empty DATABASE_URL, invalid log level, rapid restarts. Save to `.sisyphus/evidence/final-qa/`.
  Output: `Scenarios [N/N pass] | Integration [N/N] | Edge Cases [N tested] | VERDICT`

- [x] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual diff (`git diff --name-only` and contents). Verify 1:1 — everything in spec was built (no missing), nothing beyond spec was built (no creep). Check "Must NOT do" compliance. Detect cross-task contamination. Flag unaccounted changes. Verify no Python files modified.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | Python Modified [NO/YES] | VERDICT`

---

## Commit Strategy

### Commit 1: Docker + PostgreSQL Foundation
```bash
git add docker-compose.yml .env.example migrations/ .dockerignore
git commit -m "chore: add docker postgres foundation"
```

### Commit 2: Go API + Worker Skeleton
```bash
git add go.mod go.sum cmd/ internal/ Dockerfile.api Dockerfile.worker Makefile
git commit -m "feat: add go api and worker skeleton"
```

### Commit 3: Documentation + Cleanup
```bash
git add docs/migration/phase-1-foundation.md .gitignore
git commit -m "docs: add phase 1 foundation guide"
# If formatting fixes needed:
git add -A
git commit -m "chore: format code and update gitignore"
```

---

## Success Criteria

### Verification Commands
```bash
# Infrastructure
docker compose up -d postgres
docker compose ps | grep baxi-postgres | grep healthy

# Migration
export DATABASE_URL="postgres://baxi:baxi_dev@localhost:5432/baxi?sslmode=disable"
make migrate
psql "$DATABASE_URL" -c "SELECT schema_name FROM information_schema.schemata WHERE schema_name IN ('raw','dwd','mart','ops','gov','ai','audit') ORDER BY schema_name;"

# Go compilation
go test ./...
go vet ./...

# API
make api &
curl http://localhost:8080/api/v1/health
# Expected: {"status":"ok","service":"baxi-api"}

# Worker
timeout 5s make worker
# Expected: log contains "baxi-worker started"

# Scope check
git diff --name-only
# Expected: only new files, no Python modifications
```

### Final Checklist
- [ ] All "Must Have" present
- [ ] All "Must NOT Have" absent
- [ ] All tests pass (or no tests exist — compilation only)
- [ ] All evidence files captured in `.sisyphus/evidence/`
- [ ] 3 commits created as specified
- [ ] No Python files modified
- [ ] No existing config/*.yml modified

