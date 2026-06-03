# Coding Conventions

**Analysis Date:** 2026-06-03

## Naming Patterns

### Go

**Files:**
- Snake_case for test files: `decision_eval_test.go`, `agent_logs_test.go`
- Descriptive suffixes for test variants: `_test.go` (unit), `_integration_test.go` (integration), `_coverage_test.go` (coverage fill), `_extra_test.go` (additional scenarios)
- No `_test` package separation — tests live in the same package as production code

**Functions:**
- PascalCase for exported: `NewDecisionService`, `BuildDecisionContext`
- camelCase for unexported: `caseToResponse`, `structToMap`, `writeError`
- Constructor pattern: `NewXxx` prefix for constructors (`NewDecisionHandler`, `NewPoolProvider`)
- Builder pattern for optional dependencies: `WithMetrics`, `WithReplayService`, `WithRuleProvider`

**Variables:**
- Short names in tight scopes: `ctx`, `w`, `r`, `err`
- Descriptive names in broader scopes: `decisionCaseID`, `pagination`, `proposals`
- Pointer receivers named after type: `(h *DecisionHandler)`, `(s *DecisionService)`

**Types:**
- PascalCase for all exported types: `DecisionCase`, `ActionProposal`, `LLMSafeContext`
- Interface names use `-er` suffix: `DecisionProvider`, `AlertLister`, `CaseService`, `ContextBuilder`
- Struct suffixes: `Row` for DB row structs (`DecisionCaseRow`, `LLMDecisionRow`)
- Request/Response DTOs: `CreateCaseRequest`, `CaseListResponse`

**Constants:**
- PascalCase for exported string constants: `DecisionTypeMonitor`, `SeverityHigh`, `ActionTypeNotifyOwner`
- Grouped by type in const blocks with doc comments

### TypeScript/React

**Files:**
- PascalCase for component files: `Dashboard.tsx`, `CaseDetail.test.tsx`
- camelCase for utility files: `client.ts`, `governance.ts`
- Co-located tests: `PageName.test.tsx` alongside `PageName.tsx`, or in `__tests__/` subdirectory

**Components:**
- PascalCase for component names: `Dashboard`, `ConfirmApplyDialog`
- Hooks: camelCase, no `use` prefix enforced

**Variables:**
- camelCase for variables and functions
- ALL_CAPS_SNAKE_CASE for constants (env vars)

## Code Style

### Go

**Formatting:**
- `gofmt` enforced via golangci-lint (`gofmt` linter enabled)
- `goimports` enforced for import organization
- Max cyclomatic complexity: 15 (`gocyclo` linter, `.golangci.yml` line 21)

**Linting:**
- Tool: `golangci-lint` with config in `.golangci.yml`
- Enabled linters: `errcheck`, `gosimple`, `govet`, `ineffassign`, `staticcheck`, `unused`, `gocyclo`, `goimports`, `gofmt`, `revive`
- Test files skip: `gocyclo` and `errcheck` on `*_test.go`
- CI runs `go vet ./...` and `go mod tidy` check (`.github/workflows/go-ci.yml`)

**Makefile targets:**
```bash
make fmt      # go fmt ./...
make vet      # go vet ./...
make lint     # go vet ./... (golangci-lint target exists but CI uses go vet)
```

### TypeScript

**Formatting:**
- Prettier configured in `package.json`: `"format": "prettier --write \"src/**/*.{ts,tsx}\""`
- ESLint config in `frontend/eslint.config.js` using `@eslint/js`, `typescript-eslint`, `eslint-plugin-react-hooks`
- `verbatimModuleSyntax: true` in `tsconfig.json` — requires `import type` for type-only imports

**Key ESLint rules:**
- `react/react-in-jsx-scope: off` (React 19 automatic JSX runtime)
- `@typescript-eslint/no-unused-vars: warn`
- React Hooks rules enabled via plugin

## Import Organization

### Go

**Order (enforced by goimports):**
1. Standard library
2. External dependencies
3. Internal module (`baxi/internal/*`)

**Example from `internal/api/handler/decision.go`:**
```go
import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"baxi/internal/action"
	"baxi/internal/api/dto"
	"baxi/internal/api/middleware"
	"baxi/internal/decision"
	"baxi/internal/httputil"
	"baxi/internal/llm"
)
```

**Path aliases:** None in Go — full module path `baxi/internal/...`

### TypeScript

**Path aliases:**
- `@/` maps to `./src/` (configured in `tsconfig.json` and `vite.config.ts`)
- Example: `import { apiClient } from "@/api/client"`

**Import style:**
- `import type` required for type-only imports due to `verbatimModuleSyntax`
- Example: `import type { ReactElement } from "react"`

## Error Handling

### Go

**Wrapping pattern:**
```go
if err != nil {
    return fmt.Errorf("insert ai.decision_case: %w", err)
}
```
- Always wrap with context using `fmt.Errorf("...: %w", err)`
- Repository layer adds domain context: `"query ai.decision_case by id: %w"`, `"scan action_proposal row: %w"`

**Sentinel error checking:**
```go
if errors.Is(err, pgx.ErrNoRows) {
    writeError(w, r, http.StatusNotFound, middleware.NOT_FOUND, "case not found")
    return
}
```

**Early return style:**
```go
if err != nil {
    writeError(w, r, http.StatusInternalServerError, middleware.INTERNAL_ERROR, "internal server error")
    return
}
httputil.JSON(w, http.StatusOK, resp)
```

**No panics in production code** — handlers recover via middleware panic recovery.

### TypeScript

**API error pattern:**
```typescript
class ApiClientError extends Error {
  constructor(
    public status: number,
    public apiError: ApiError,
  ) {
    super(apiError.message)
  }
}
```

## Logging

**Framework:** `go.uber.org/zap` (JSON structured logging)

**Configuration:** `internal/logger/logger.go`
- JSON encoding to stdout
- ISO8601 timestamps
- Short caller encoding
- Levels: debug, info, warn, error (default: info)

**Usage pattern:**
```go
log.Printf("WARNING: API_BEARER_TOKEN is set to a known weak/placeholder value")
```
- `log` package used for startup warnings
- `zap.Logger` used for structured application logging (injected via config)

## Comments

### Go

**Package comments:** Required, above package declaration
```go
// Package decision provides repository access for the decision domain.
// This is a domain subpackage of the repository layer with pool injection.
package decision
```

**Interface comments:** Explain purpose and testing strategy
```go
// DecisionService defines the business operations needed by DecisionHandler.
// Tests substitute a mock without importing the service package.
type DecisionService interface { ... }
```

**Function comments:** For exported functions, explain what it does
```go
// Decide orchestrates the full decision workflow: get case, build context,
// generate decision, and create action proposals.
func (s *DecisionService) Decide(...) { ... }
```

**Section separators:**
```go
// --- DTO mapping helpers ---
```

### TypeScript

**Minimal commenting** — code is expected to be self-documenting
- No JSDoc convention observed
- Component props typed via interfaces (inferred)

## Function Design

### Go

**Size:** Functions typically 10-40 lines. Complex handlers split DTO mapping into helper functions.

**Parameters:**
- `ctx context.Context` as first parameter
- Pointer receivers on structs: `(h *DecisionHandler)`
- Functional options pattern for configuration: `action.WithDryRun(true)`

**Return values:**
- `(result, error)` pattern
- Named return values rarely used
- Nil result + error on failure

**Interface design:**
- Local narrow interfaces defined in consuming packages
- Example: `AlertLister` in `handler/alerts.go`, `DecisionService` in `handler/decision.go`
- Compile-time interface checks:
```go
var (
    _ CaseService     = (*decision.CaseService)(nil)
    _ ContextBuilder  = (*decision.ContextBuilder)(nil)
)
```

### TypeScript

**Component pattern:** Function components with typed props
**Hook pattern:** Custom hooks return `[data, loading, error]` tuples

## Module Design

### Go

**Exports:**
- PascalCase = exported
- No explicit `export` keyword — visibility by case

**Barrel files:** Not used in Go

**Package structure:**
- Domain-driven subpackages: `internal/repository/decision/`, `internal/repository/alert/`
- Flat package for services: `internal/service/` (all services in one package)
- Handler package: `internal/api/handler/` (all handlers in one package)

### TypeScript

**Barrel exports:**
- `frontend/src/components/index.ts` exports all shared components
- `frontend/src/api/governance.ts` exports typed API functions

## Environment Variables

**Naming:** ALL_CAPS_SNAKE_CASE

**Required vars:**
- `DATABASE_URL` — PostgreSQL connection string
- `API_BEARER_TOKEN` — Shared auth token (min 32 chars)

**Domain-grouped vars:**
- LLM: `LLM_API_KEY`, `LLM_API_BASE`, `LLM_MODEL`, `LLM_TEMPERATURE`, `LLM_MAX_Tokens`, `LLM_TIMEOUT_SECONDS`, `LLM_ENABLED`, `LLM_PROVIDER`, `LLM_FALLBACK_ENABLED`, `LLM_STORE_RAW_OUTPUT`, `LLM_MAX_RETRIES`
- Worker: `WORKER_BATCH_SIZE`, `WORKER_TICK_INTERVAL`
- Action: `ACTION_APPLY_DRY_RUN`, `FEISHU_WEBHOOK_URL`, `GITHUB_TOKEN`
- CORS: `CORS_ALLOWED_ORIGINS`

**Loading pattern:** `internal/config/config.go`
- `os.Getenv()` for required, `getEnv(key, defaultValue)` for optional
- Bool parsing: explicit string comparison `v == "true"`
- Numeric parsing with `_` for ignored errors (uses defaults)

**Frontend token storage:** `sessionStorage.getItem("API_BEARER_TOKEN")`

## TypeScript-Specific Conventions

**Strict mode:** Enabled (`"strict": true` in `tsconfig.json`)

**Permissive unused vars:**
```json
"noUnusedLocals": false,
"noUnusedParameters": false
```

**Styling:**
- Tailwind CSS v4 with `@tailwindcss/vite` plugin
- `tailwindcss-animate` for animations
- Radix UI primitives for accessible components
- `lucide-react` for icons (convention from AGENTS.md)

**API client pattern:**
- Singleton `apiClient` with `get<T>()` and `post<T>()` methods
- AbortController with configurable timeout (10s default, 120s for Feishu)
- Bearer token from `sessionStorage`

---

*Convention analysis: 2026-06-03*
