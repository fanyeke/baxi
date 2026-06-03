# Technology Stack

**Analysis Date:** 2026-06-03

## Languages

**Primary:**
- **Go 1.23** — Backend API, pipeline engine, MCP server, CLI, worker, and all business logic (`internal/`, `cmd/`, `test/`)
- **TypeScript** — React 19 SPA frontend (`frontend/`), Pi Agent extensions (`pi-extension/`)
- **SQL** — Goose migrations (`migrations/`, 21 migration files)

**Secondary:**
- **YAML** — Governance configs (`config/`, 28 files), docker-compose, GitHub Actions workflows
- **Shell** — Makefile, backup/restore/verification scripts (`scripts/`)

## Runtime

**Environment:**
- Go 1.23 (module `baxi`)
- Node.js 20 (frontend CI), local dev uses Vite dev server

**Package Manager:**
- Go modules (`go.mod`, `go.sum`)
- npm (`frontend/package.json`, `frontend/package-lock.json`)
- Lockfile: present for both Go and frontend

## Frameworks

**Core:**
- **chi/v5** `v5.2.5` — HTTP router for Go API (`internal/api/server.go`, `internal/api/routes.go`)
- **React 19** `^19.1.0` — Frontend SPA (`frontend/src/`)
- **Vite 6** `^6.3.5` — Frontend build tool and dev server (`frontend/vite.config.ts`)
- **TanStack Query 5** `^5.72.2` — Async state management (`frontend/src/api/`)
- **Tailwind CSS v4** `^4.1.6` — Utility-first styling (`frontend/vite.config.ts`)
- **Radix UI** — Headless accessible primitives (Dialog, Tabs, etc.)

**Testing:**
- **testify** `v1.9.0` — Go assertions and test suites
- **testcontainers-go** `v0.35.0` + `modules/postgres` `v0.35.0` — Integration test database isolation
- **Vitest** `^4.1.7` — Frontend unit testing (`frontend/vitest.config.ts`)
- **Playwright** `^1.60.0` — Frontend E2E testing
- **jsdom** `^29.1.1` — Frontend test environment
- **Testing Library** (`@testing-library/react`, `@testing-library/jest-dom`, `@testing-library/user-event`) — React component testing

**Build/Dev:**
- **Goose** `v3.20.0` — SQL migration runner (`Makefile`, `internal/testutil/db.go`)
- **Docker** — Multi-stage builds (`Dockerfile.api`, `Dockerfile.worker`)
- **golangci-lint** — Go linting (`.golangci.yml`)
- **ESLint 10** + `typescript-eslint` + `eslint-plugin-react-hooks` — Frontend linting (`frontend/eslint.config.js`)
- **Prettier** `^3.8.3` — Frontend formatting
- **Zap** `v1.28.0` — Structured logging (`internal/logger/`, `go.uber.org/zap`)

## Key Dependencies

**Critical:**
- **pgx/v5** `v5.5.5` — PostgreSQL driver and connection pool (`internal/db/postgres.go`, `internal/repository/`)
- **openai-go** `v1.12.0` — OpenAI-compatible LLM API client (`internal/llm/openai_provider.go`)
- **mcp-go** `v0.41.1` — MCP (Model Context Protocol) server framework (`internal/mcp/`)
- **golang-jwt/jwt/v5** `v5.3.1` — JWT parsing for API auth middleware (`internal/api/middleware/auth.go`)
- **uuid** `v1.6.0` — UUID generation
- **yaml.v3** `v3.0.1` — YAML config parsing (`internal/configloader/`, `config/`)

**Infrastructure:**
- **goose/v3** `v3.20.0` — Database migrations
- **testcontainers-go** `v0.35.0` — Docker-based test infrastructure

**Frontend:**
- **react-router-dom** `^7.6.1` — SPA routing
- **tailwindcss-animate** `^1.0.7` — Tailwind animation utilities
- **lucide-react** — Icon library (referenced in AGENTS.md, not in package.json — verify installed)
- **clsx** / **tailwind-merge** — Conditional class merging (referenced in AGENTS.md conventions)

## Configuration

**Environment:**
- All configuration loaded from environment variables (`internal/config/config.go`)
- `.env` file present (not committed, `.env.example` committed as template)
- `frontend/.env` — Vite env vars (`VITE_API_BASE_URL`, `VITE_API_BACKEND`)

**Key configs required:**
- `DATABASE_URL` — PostgreSQL connection string (required)
- `API_BEARER_TOKEN` — API auth token, minimum 32 chars (required)
- `API_PORT` — defaults to 8080
- `LOG_LEVEL` — defaults to `info`
- `CORS_ALLOWED_ORIGINS` — comma-separated, defaults to localhost dev origins

**Build:**
- `go.mod` / `go.sum` — Go dependency management
- `frontend/package.json` — Node dependency management
- `frontend/vite.config.ts` — Vite build config with proxy to `:8080`
- `frontend/tsconfig.json` — TypeScript config (`verbatimModuleSyntax`, `@/` alias)
- `.golangci.yml` — Lint config (18 linters enabled, gocyclo min-complexity 15)
- `docker-compose.yml` — Local orchestration (postgres:16, api, worker)

## Platform Requirements

**Development:**
- Go 1.23+
- Node.js 20+ (frontend)
- Docker & Docker Compose (for postgres and local orchestration)
- PostgreSQL 15+ (local dev via `docker compose up postgres`)
- Make (for Makefile targets)

**Production:**
- Docker multi-stage build: `golang:1.23-alpine` → `alpine:latest`
- CGO_ENABLED=0 static binaries
- PostgreSQL 15/16
- Target port: 8080 (API), stdio (MCP)

---

*Stack analysis: 2026-06-03*
