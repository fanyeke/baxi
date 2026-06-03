# External Integrations

**Analysis Date:** 2026-06-03

## APIs & External Services

**Feishu / Lark (ByteDance):**
- **What it's used for:** Action dispatch (alert messages to chat), bitable record sync, status import/export
- **SDK/Client:** Custom HTTP client in `internal/feishu/client.go` (no official SDK)
- **Auth:** Tenant access token via `app_id` + `app_secret` (cached with expiry)
- **Endpoints used:**
  - `POST /auth/v3/tenant_access_token/internal` — token acquisition
  - `GET/POST /bitable/v1/apps/{app_token}/tables/{table_id}/records` — bitable CRUD
  - `POST /im/v1/messages` — chat message sending
- **Config env vars:** `FEISHU_APP_ID`, `FEISHU_APP_SECRET`, `FEISHU_BASE_APP_TOKEN`, `FEISHU_CHAT_ID`, `FEISHU_WEBHOOK_URL`
- **Files:** `internal/feishu/client.go`, `internal/adapter/feishu.go`, `internal/api/handler/feishu.go`
- **YAML configs:** `config/feishu_app.yml`, `config/feishu_base_schema.yml`, `config/feishu_field_mapping.yml`, `config/feishu_table_ids.yml`, `config/feishu_user_mapping.yml`

**GitHub:**
- **What it's used for:** Creating issues from action proposals, adding labels and comments
- **SDK/Client:** Custom REST client (`internal/adapter/github.go`)
- **Auth:** Personal access token via `Authorization: Bearer` header
- **Endpoints used:**
  - `POST /repos/{repo}/issues` — issue creation
  - `POST /repos/{repo}/issues/{number}/labels` — label addition
  - `POST /repos/{repo}/issues/{number}/comments` — comment addition
- **Config env var:** `GITHUB_TOKEN`
- **Files:** `internal/adapter/github.go`

**OpenAI / OpenAI-Compatible:**
- **What it's used for:** LLM-based decision generation (`decide` tool, decision engine)
- **SDK/Client:** `github.com/openai/openai-go` `v1.12.0` (`internal/llm/openai_provider.go`)
- **Auth:** API key (`LLM_API_KEY`)
- **Features:** Chat completions with structured JSON output, temperature/max_tokens/seed control, timeout handling
- **Config env vars:** `LLM_API_KEY`, `LLM_API_BASE` (optional, for custom base URL), `LLM_MODEL`, `LLM_TEMPERATURE`, `LLM_MAX_TOKENS`, `LLM_TIMEOUT_SECONDS`
- **Fallback:** Rule-based provider when LLM disabled or fails (`internal/llm/rule_provider.go`)
- **Files:** `internal/llm/openai_provider.go`, `internal/llm/provider_factory.go`, `internal/llm/rule_provider.go`

**MCP (Model Context Protocol):**
- **What it's used for:** Pi Agent and other MCP clients connect to Baxi via stdio transport
- **SDK/Client:** `github.com/mark3labs/mcp-go` `v0.41.1`
- **Transport:** stdio only (`server.ServeStdio` in `cmd/baxi-mcp/main.go`)
- **Tools exposed:** 31 tools across 11 domains (decision, alert, governance, pipeline, review, action, outbox, status, ontology, sandbox, schema)
- **Files:** `internal/mcp/server.go`, `internal/mcp/tools_*.go`, `cmd/baxi-mcp/main.go`
- **Pi Agent extensions:** `pi-extension/` (TypeScript, `@earendil-works/pi-coding-agent`)

## Data Storage

**Databases:**
- **PostgreSQL 15/16**
  - Connection: `DATABASE_URL` env var
  - Client: `pgx/v5` (`pgxpool.Pool` throughout codebase)
  - Migrations: Goose (`migrations/`, 21 SQL files)
  - Schemas: `raw`, `dwd`, `metric`, `ops`, `gov`, `ai`, `audit`
  - Files: `internal/db/postgres.go`, `migrations/*.sql`

**File Storage:**
- Local filesystem only — raw CSVs in `data/raw/`, intermediate data in `data/`
- No cloud object storage integration

**Caching:**
- None detected — no Redis or in-memory cache
- Feishu access token cached in-memory with expiry (`internal/feishu/client.go:52`)

## Authentication & Identity

**Auth Provider:**
- Custom bearer token auth (not OAuth/OIDC)
- **API auth:** `API_BEARER_TOKEN` env var, minimum 32 characters, compared with `subtle.ConstantTimeCompare`
- **JWT:** `golang-jwt/jwt/v5` used for token parsing (verify exact usage in middleware)
- **MCP auth:** `BAXI_MCP_USER_ID` and `BAXI_MCP_ROLE` env vars set caller identity
- **Files:** `internal/api/middleware/auth.go`

## Monitoring & Observability

**Error Tracking:**
- None detected — no Sentry, Rollbar, or similar service

**Logs:**
- **Zap** structured logging (`go.uber.org/zap`)
- Log level controlled by `LOG_LEVEL` env var
- Files: `internal/logger/`

**Audit:**
- Custom audit logging to `audit.audit_log` table
- LLM decision audit logging to `ai.llm_audit_log` table
- Files: `internal/audit/`, `internal/llm/audit.go`

## CI/CD & Deployment

**Hosting:**
- Docker-based deployment (multi-stage Alpine images)
- `docker-compose.yml` for local orchestration
- No Kubernetes manifests detected

**CI Pipeline:**
- **GitHub Actions** (`.github/workflows/go-ci.yml`)
  - Go 1.22 in CI (note: project uses Go 1.23)
  - Jobs: lint (`go vet`, `go mod tidy` check), unit tests, integration tests (postgres:15-alpine service), frontend tests
  - No deployment job detected

**Local Development:**
- `make up` — starts postgres via docker compose
- `make api` / `make worker` / `make mcp` — runs services locally
- `make build` — builds static Go binaries

## Environment Configuration

**Required env vars:**
| Variable | Required | Purpose |
|----------|----------|---------|
| `DATABASE_URL` | Yes | PostgreSQL connection |
| `API_BEARER_TOKEN` | Yes | API authentication |
| `API_PORT` | No | API server port (default 8080) |
| `LOG_LEVEL` | No | Logging level (default info) |
| `CORS_ALLOWED_ORIGINS` | No | CORS origins |
| `FEISHU_APP_ID` | No | Feishu integration |
| `FEISHU_APP_SECRET` | No | Feishu integration |
| `FEISHU_BASE_APP_TOKEN` | No | Feishu bitable |
| `FEISHU_CHAT_ID` | No | Feishu chat target |
| `FEISHU_WEBHOOK_URL` | No | Feishu webhook |
| `GITHUB_TOKEN` | No | GitHub issue creation |
| `LLM_ENABLED` | No | LLM toggle (default false) |
| `LLM_API_KEY` | No | OpenAI API key |
| `LLM_API_BASE` | No | Custom OpenAI base URL |
| `LLM_MODEL` | No | Model name (default gpt-4o-mini) |
| `LLM_PROVIDER` | No | Provider selection (default disabled) |
| `LLM_TIMEOUT_SECONDS` | No | LLM timeout |
| `ACTION_EXECUTION_ENABLED` | No | Action execution toggle |
| `OUTBOX_DISPATCH_ENABLED` | No | Outbox dispatch toggle |
| `WORKER_TICK_INTERVAL` | No | Worker poll interval (default 30s) |
| `WORKER_BATCH_SIZE` | No | Worker batch size (default 10) |
| `BAXI_MCP_USER_ID` | No | MCP caller identity |
| `BAXI_MCP_ROLE` | No | MCP caller role |

**Secrets location:**
- `.env` file (gitignored, `.env.example` committed as template)
- `frontend/.env` (gitignored, `frontend/.env.example` committed)
- Never commit real `.env` files

## Webhooks & Callbacks

**Incoming:**
- Feishu webhook URL support (`FEISHU_WEBHOOK_URL` env var, referenced but not actively used in current adapter logic)
- No HTTP webhook handlers detected for GitHub, Slack, or other services
- MCP stdio transport acts as a "callback" channel for Pi Agent

**Outgoing:**
- Feishu chat messages (`internal/adapter/feishu.go`)
- GitHub issue creation (`internal/adapter/github.go`)
- OpenAI chat completions (`internal/llm/openai_provider.go`)

## Frontend ↔ Backend Integration

**API Base:**
- Frontend Vite dev server proxies `/api` to `http://localhost:8080`
- Production: `VITE_API_BASE_URL` env var

**API Client:**
- Typed API client in `frontend/src/api/client.ts`
- Endpoint modules: `frontend/src/api/governance.ts`, `frontend/src/api/types.ts`

---

*Integration audit: 2026-06-03*
