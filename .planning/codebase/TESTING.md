# Testing Patterns

**Analysis Date:** 2026-06-03

## Test Framework

### Go

**Runner:** `go test` (stdlib)
- Go version: 1.23 (module `go.mod`)

**Assertion Library:** `github.com/stretchr/testify v1.9.0`
- `assert` — non-fatal assertions (continue test on failure)
- `require` — fatal assertions (stop test immediately on failure)

**Run Commands:**
```bash
# All tests (includes integration tests that need DB)
go test ./... -v -count=1

# Unit tests only (skip DB-dependent)
go test ./... -short -count=1

# Specific packages
make test-pipeline    # pipeline/ ingest/ alert/ recommendation/ outbox/
make test-governance  # configloader/ ontology/ governance/

# Coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# Race detection (CI)
go test -v -count=1 -race -coverprofile=coverage.out ./...
```

### TypeScript/React

**Runner:** `vitest` v4.1.7 with `jsdom` environment
- Config: `frontend/vitest.config.ts`
- Globals enabled (no explicit `import { describe, it, expect }` needed in test files, though they do import)

**Assertion Library:** `@testing-library/jest-dom` v6.9.1
- DOM matchers: `toBeInTheDocument()`, `toHaveTextContent()`

**Testing utilities:**
- `@testing-library/react` v16.3.2 — component rendering
- `@testing-library/user-event` v14.6.1 — user interactions

**Run Commands:**
```bash
cd frontend && npm test           # vitest run
cd frontend && npm run test       # same as above
cd frontend && npm run e2e        # playwright test
```

## Test File Organization

### Go

**Location:** Co-located with production code
- `decision.go` → `decision_test.go` (same directory)
- No separate `tests/` directory within `internal/`

**Naming:**
- `*_test.go` — standard unit tests
- `*_integration_test.go` — integration tests with DB
- `*_coverage_test.go` — coverage fill tests for edge cases
- `*_extra_test.go` — additional scenario tests
- `*_edge_test.go` — edge case tests
- `*_pure_test.go` — pure function tests (no external deps)
- `*_mock_test.go` — mock-related tests

**Build tags:** Integration tests use build constraints
```go
//go:build integration

package governance
```
- Tagged tests compile only with `-tags integration`
- CI runs integration tests separately with postgres service
- `testing.Short()` check at test start for DB tests:
```go
if testing.Short() {
    t.Skip("skipping integration test")
}
```

### TypeScript

**Location:** Co-located
- `src/pages/__tests__/Dashboard.test.tsx`
- `src/components/__tests__/Layout.test.tsx`
- `src/api/__tests__/governance.test.tsx`

**Naming:** `*.test.tsx` (or `*.test.ts`)

## Test Structure

### Go

**Table-driven tests:**
```go
func TestResolveLevel(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"pii", "L3"},
        {"sensitive", "L3"},
        {"internal", "L2"},
        {"unknown_level", "L2"},
        {"", "L2"},
    }

    for _, tc := range tests {
        t.Run(tc.input, func(t *testing.T) {
            result := ResolveLevel(tc.input)
            assert.Equal(t, tc.expected, result)
        })
    }
}
```

**Subtests with `t.Run`:**
```go
func TestAgentLogHandler_List(t *testing.T) { ... }
func TestAgentLogHandler_List_Error(t *testing.T) { ... }
func TestAgentLogHandler_List_Empty(t *testing.T) { ... }
func TestAgentLogHandler_List_BadPagination(t *testing.T) { ... }
```

**Setup/teardown patterns:**
- `testutil.SetupTestPool(t)` — starts testcontainer, applies migrations, returns pool
- `t.Cleanup()` for deferred cleanup:
```go
pg, err := testutil.StartPostgres(ctx)
require.NoError(t, err)
t.Cleanup(func() { _ = pg.Terminate(ctx) })
```

**Inline DDL for unit tests:**
```go
const govServiceDDL = `
CREATE SCHEMA IF NOT EXISTS gov;
CREATE TABLE IF NOT EXISTS gov.config_snapshot (...);
`
_, err = pool.Exec(ctx, govServiceDDL)
```

**Helper functions marked with `t.Helper()`:**
```go
func setupGovServiceDB(t *testing.T) *pgxpool.Pool {
    t.Helper()
    // ...
}
```

### TypeScript

**Test wrapper pattern:**
```typescript
function renderWithQueryClient(ui: ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>
    </MemoryRouter>
  )
}
```
- Defined in `frontend/src/test-setup.tsx`
- Wraps components with QueryClientProvider and MemoryRouter
- Disables query retries for predictable tests

**Mock pattern with vi:**
```typescript
vi.mock("@/api/client", () => ({
  apiClient: {
    get: vi.fn(),
  },
}))

beforeEach(() => {
  vi.clearAllMocks()
})
```

**Async testing:**
```typescript
it("renders stat cards with data", async () => {
  vi.mocked(apiClient.get).mockImplementation(async (path: string) => {
    if (path === "/health") return { status: "ok" }
    // ...
  })
  renderWithQueryClient(<Dashboard />)
  expect(await screen.findByText("OK")).toBeInTheDocument()
})
```

## Mocking

### Go

**Pattern: Local interfaces + inline mock structs**
```go
// In handler file — interface defined locally
type AlertLister interface {
    ListAlerts(ctx context.Context, filters model.AlertFilters, sort string, limit, offset int) (*model.AlertListResponse, error)
}

// In test file — inline mock
type mockAgentLogService struct {
    listFn func(ctx context.Context, limit, offset int) (*service.AgentLogListResponse, error)
}
func (m *mockAgentLogService) ListAgentLogs(...) (...) {
    return m.listFn(...)
}
```
- No mocking framework (no gomock, no testify/mock)
- Handlers define narrow interfaces for their dependencies
- Tests implement those interfaces with inline structs

**HTTP test server:**
```go
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(201)
    json.NewEncoder(w).Encode(map[string]interface{}{"html_url": "..."})
}))
defer server.Close()
```

### TypeScript

**Pattern: `vi.mock()` for module mocking**
```typescript
vi.mock("@/api/client", () => ({
  apiClient: { get: vi.fn(), post: vi.fn() },
}))
```

**Component isolation:**
- Mock API client, not components
- Test renders full component tree with QueryClientProvider

## Fixtures and Factories

### Go

**Testcontainers for PostgreSQL:**
- Package: `internal/testutil/`
- Image: `postgres:15-alpine`
- Timeout: 120 seconds startup

**Key helpers:**
```go
// testutil/db.go
func StartPostgres(ctx context.Context) (*PostgresContainer, error)
func (c *PostgresContainer) RunMigrations(ctx context.Context, migrationsDir string) error
func SetupTestPool(t *testing.T) *pgxpool.Pool
```

**Fixture insertion helpers:**
```go
func insertClassification(t *testing.T, pool *pgxpool.Pool, fieldPath, level string, score float64)
func insertLineage(t *testing.T, pool *pgxpool.Pool, sourceTable, sourceCol, targetTable, targetCol string, confidence float64)
```

**Test data factories:**
```go
func validDecisionOutput() *llm.DecisionOutput {
    return &llm.DecisionOutput{
        DecisionType: llm.DecisionTypeInvestigate,
        Severity:     llm.SeverityMedium,
        // ...
    }
}
```

### TypeScript

**No formal fixture system** — mock data inline per test
```typescript
vi.mocked(apiClient.get).mockImplementation(async (path: string) => {
  if (path === "/health") return { status: "ok", version: "1.0.0" }
  if (path === "/alerts?limit=1") return { items: [], total: 42 }
  return {}
})
```

## Coverage

### Go

**Target:** No explicit percentage target enforced
- Coverage profiles generated in CI: `-coverprofile=coverage.out`
- Multiple `.out` files observed in repo root (development artifacts)

**View coverage:**
```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

**Coverage fill strategy:**
- `*_coverage_test.go` files exist specifically to cover edge cases
- `*_extra_test.go` files add scenarios for uncovered branches
- Examples: `db_coverage_test.go`, `governance_coverage_test.go`, `eval_coverage_test.go`

### TypeScript

**Tool:** `@vitest/coverage-v8` v4.1.7 (installed but no config observed in `vitest.config.ts`)
- No coverage threshold configured

## Test Types

### Unit Tests

**Go:**
- Scope: Pure functions, individual methods
- No external dependencies
- Table-driven where applicable
- Example: `internal/model/model_test.go` — struct construction tests
- Example: `internal/eval/decision_eval_test.go` — evaluator logic tests

**TypeScript:**
- Scope: Component rendering, hook behavior
- Mocked API client
- Example: `frontend/src/pages/__tests__/Dashboard.test.tsx`

### Integration Tests

**Go:**
- Scope: Database + repository + service layers
- Requires PostgreSQL
- Build tag: `//go:build integration`
- Location: Co-located in packages (`internal/governance/governance_test.go`) or in `test/integration/`

**Key integration test suites:**
- `test/integration/phase7_test.go` — Full pipeline + governance workflow (497 lines)
- `test/integration/phase8_aip_test.go` — AIP integration tests
- `test/integration/ontology_v2_e2e_test.go` — Ontology E2E
- `internal/api/integration_test.go` — API handler integration
- `internal/audit/integration_test.go` — Audit integration

### E2E Tests

**Go:**
- Location: `test/e2e/`
- Packages: `package e2e`
- Tests full system flows via API

**Files:**
- `test/e2e/api_lifecycle_test.go`
- `test/e2e/decision_lifecycle_test.go`
- `test/e2e/pipeline_to_dispatch_test.go`
- `test/e2e/sandbox_comparison_test.go`
- `test/e2e/pi_integration_test.go`

**Security E2E:**
- `test/security/phase7_test.go` — Auth/RBAC contract tests (316 lines)

### Migration Tests

**Schema contract tests:**
- `test/migration/contract_test.go` — Goose schema vs repository struct alignment (503 lines)
- Ensures DB migrations match Go struct definitions

## CI/CD Test Pipeline

**Workflow:** `.github/workflows/go-ci.yml`

**Jobs:**
1. **Lint** (go vet, go mod tidy check)
2. **Unit Tests** — runs packages that don't need DB:
   ```bash
   go test -v -count=1 -race -coverprofile=coverage.out \
     ./internal/llm/... ./internal/action/... ./internal/governance/... \
     ./internal/ontology/... ./internal/eval/... ./internal/feature/... \
     ./internal/config/... ./internal/alert/... ./internal/recommendation/... \
     ./internal/httputil/... ./internal/api/middleware/... \
     ./internal/decision/... ./internal/adapter/...
   ```
3. **Integration Tests** — runs with postgres:15-alpine service container:
   ```bash
   go test -v -count=1 -race -timeout 300s -coverprofile=coverage.out \
     ./internal/repository/... ./internal/service/... ./internal/outbox/... \
     ./internal/worker/... ./internal/api/handler/... ./internal/review/...
   ```
4. **Frontend Tests** — Node 20, npm ci, `npm test`

**Environment for integration tests:**
```yaml
DATABASE_URL: postgres://baxi:baxi_test@localhost:5432/baxi_test?sslmode=disable
API_BEARER_TOKEN: test-token-for-ci-at-least-32-chars-long
```

## Common Patterns

### Go

**Async/concurrent testing:**
```go
var wg sync.WaitGroup
results := make(chan error, numProposals)

for i := 0; i < numProposals; i++ {
    wg.Add(1)
    go func(idx int) {
        defer wg.Done()
        _, err := reviewSvc.ApproveProposal(ctx, proposalIDs[idx], ...)
        results <- err
    }(i)
}

wg.Wait()
close(results)
```

**Error testing:**
```go
require.Error(t, err)
require.ErrorIs(t, err, action.ErrNotApproved)
```

**HTTP handler testing:**
```go
r := httptest.NewRequest(http.MethodGet, "/api/v1/logs/agent", nil)
w := httptest.NewRecorder()
h.HandleListAgentLogs(w, r)
require.Equal(t, http.StatusOK, w.Code)
```

**Context with timeout/cancel:**
```go
workerCtx, cancel := context.WithCancel(ctx)
done := make(chan struct{})
go func() {
    _ = w.Run(workerCtx)
    close(done)
}()
time.Sleep(500 * time.Millisecond)
cancel()
<-done
```

### TypeScript

**Loading state testing:**
```typescript
it("shows loading skeleton when queries are loading", async () => {
  vi.mocked(apiClient.get).mockImplementation(() => new Promise(() => {}))
  renderWithQueryClient(<Dashboard />)
  const skeletons = document.querySelectorAll(".animate-pulse")
  expect(skeletons.length).toBeGreaterThan(0)
})
```

**Error state testing:**
```typescript
it("shows error panel when queries fail", async () => {
  vi.mocked(apiClient.get).mockRejectedValue(new Error("Network error"))
  renderWithQueryClient(<Dashboard />)
  expect(await screen.findByText("连接失败")).toBeInTheDocument()
})
```

## Anti-Patterns Observed

1. **Two test roots**: `test/` at root vs `internal/` tests — E2E tests in `test/` break `go test ./...` isolation
2. **`test/` outside `internal/`**: E2E tests import `baxi/internal/*` by full module path, fragile to refactoring
3. **Duplicated helper**: All 3 `test/` subdirs reimplement `migrationsDir()` function
4. **No coverage config in vitest.config.ts**: Coverage tool installed but not configured
5. **renderWithQueryClient duplicated**: AGENTS.md notes this helper is duplicated inline across test files (though `test-setup.tsx` exists)

---

*Testing analysis: 2026-06-03*
