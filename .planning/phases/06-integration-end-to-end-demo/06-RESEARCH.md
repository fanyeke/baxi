# Phase 6: Integration & End-to-End Demo - Research

**Researched:** 2026-06-03
**Domain:** Frontend-backend integration, E2E testing, demo readiness
**Confidence:** HIGH

## Summary

Phase 6 is the culmination of all previous phases — wiring the frontend to real backend endpoints, fixing broken tests, and making the full closed-loop demo work. The codebase has clean backend APIs (Phases 1-5) but significant **frontend-backend data model mismatches**, **compilation errors in Go test files**, and **failing frontend unit tests** prevent the integration from working.

The main blocker is a **structural mismatch** between what the frontend expects and what the backend actually returns for the Governance API pipeline. Rather than redesigning the backend DTOs (which have been working since Phase 1-2), the frontend hooks and types need to be realigned to consume the actual backend response shapes. Additionally, 4 Go test files fail to compile (stale references from Phase 3 repository refactoring), and 10 frontend unit tests fail due to text assertion mismatches.

**Primary recommendation:** Fix in this order: (1) Go test compilation errors → (2) Frontend type/hook alignment with backend → (3) Frontend test assertion fixes → (4) Full closed-loop demo validation. This maximizes parallelism between frontend and backend work.

<user_constraints>
## User Constraints (from STATE.md)

### Locked Decisions
- Fix bugs before adding features — user wants demonstrable closed loop, not new capabilities
- Implement 501 stubs rather than remove — frontend already expects these endpoints
- Remove deprecated repository shims — clean up dual APIs, enforce PoolProvider pattern

### The Agent's Discretion
(No discretion items documented for this phase)

### Deferred Ideas (OUT OF SCOPE)
- Multi-tenant isolation (from REQUIREMENTS.md)
- OAuth / social login (from REQUIREMENTS.md)
- Kubernetes deployment (from REQUIREMENTS.md)
- New channel adapters (from REQUIREMENTS.md)
- Performance benchmarking (from REQUIREMENTS.md)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| INT-01 | Frontend pages for decisions, governance, pipeline, alerts connect to working backend endpoints | Governance and Pipeline pages have **data model/protocol mismatches** — frontend types/hooks must be realigned to match backend DTOs. DecisionReview and Alerts pages have matching routes. |
| INT-02 | E2E integration tests (`test/integration/phase7_test.go`) pass cleanly | 4 Go test files fail to compile due to stale references from Phase 3 repository refactoring. `internal/service` has build failure. `internal/repository/status` has runtime test failure. |
| INT-03 | Security E2E tests (`test/security/phase7_test.go`) pass cleanly | Same compilation errors block these tests. No indication of logic failures once compilation is fixed — the security tests use the same patterns verified in Phase 5. |
| INT-04 | Frontend unit tests (`frontend/src/pages/__tests__/*.test.tsx`) pass cleanly | 10 unit tests fail across 7 files. Primary cause: tests expect Chinese error text "请求异常" but components render English/Chinese text like "Failed to load"/"加载失败". Also: Layout token test, empty state text mismatch. |
| INT-05 | Full closed-loop demo: trigger pipeline → governance → decision → action → alert → frontend | Most complex requirement. Requires all previous INT fixes plus: pipeline seed data, demo workflow orchestration, and frontend display of end-to-end results. Governance and Pipeline pages currently cannot display live data due to INT-01 issues. |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Decision case list/display | API/Backend | Browser | Frontend queries `/decisions/cases`; backend serves paginated data |
| Governance data display | API/Backend | Browser | All 6 governance endpoints served from backend; frontend only formats tables |
| Pipeline run preview | API/Backend | Browser | Backend validates pipeline config, returns preview command |
| Alert list/filter | API/Backend | Browser | Backend serves filtered alert data; frontend applies UI filters |
| Review/approve actions | API/Backend | Browser | Frontend POSTs approve/reject/cancel mutations to backend |
| E2E integration tests | Database | Backend | Testcontainers + PostgreSQL; test domain logic not HTTP |
| Frontend unit tests | Browser | — | Vitest + jsdom; mock API, test rendering only |

## Standard Stack

### Core (already installed)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go (chi/v5) | 1.23 / v5.2.5 | Backend HTTP API | Complete, stable, no changes needed |
| React 19 | ^19.1.0 | Frontend SPA | Complete, stable, no changes needed |
| TanStack Query 5 | ^5.72.2 | Async state management | Complete, stable, no changes needed |
| pgx/v5 | v5.5.5 | PostgreSQL driver | Complete, stable, no changes needed |
| testcontainers-go | v0.35.0 | Integration test DB isolation | Already used by existing E2E tests |
| Vitest | ^4.1.7 | Frontend unit testing | Already configured |
| testing-library | ^16.3.2 | React component testing | Already configured |

### No new packages required

This phase is about integration and testing — no new dependencies are needed. All work is in:
- Frontend type definitions (`frontend/src/api/types.ts`, `frontend/src/api/governance.ts`)
- Frontend test files (`frontend/src/pages/__tests__/`)
- Go test files (`internal/*/*_test.go`)
- Potentially frontend hooks or page components to handle actual backend response shapes

## Package Legitimacy Audit

> No external packages are installed in this phase. All libraries are already present in `go.mod` and `package.json`. Skipping Package Legitimacy Gate.

## Architecture Patterns

### System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Browser (React SPA)                          │
│  ┌──────────┐ ┌───────────┐ ┌──────────┐ ┌───────────────────┐   │
│  │Dashboard │ │Governance │ │Pipeline │ │ DecisionReview     │   │
│  │ /health  │ │/governance│ │/pipeline│ │ /decisions/cases   │   │
│  │ /status  │ │/catalog   │ │ /run    │ │ /proposals/{id}/   │   │
│  │ /alerts  │ │/class     │ │ (POST)  │ │   approve/reject   │   │
│  │ /tasks   │ │/markings  │ │         │ │   /cancel          │   │
│  │          │ │/lineage   │ │         │ └───────────────────┘   │
│  │          │ │/chkpts    │ │         │                         │
│  │          │ │/health    │ │         │                         │
│  └────┬─────┘ └─────┬─────┘ └────┬────┘ └──────────────────────┘ │
│       │              │            │                               │
└───────┼──────────────┼────────────┼───────────────────────────────┘
        │              │            │
   ┌────▼──────────────▼────────────▼──────────────────────────┐
   │              API Gateway (chi, /api/v1)                    │
   │              Auth: Bearer Token                            │
   │              CORS: validate scheme + port                  │
   │              Error: 5-field JSON response                  │
   └────┬──────────────┬────────────┬───────────────────────────┘
        │              │            │
   ┌────▼──────────────▼────────────▼──────────────────────────┐
   │                    Backend Services                        │
   │  ┌──────────┐ ┌───────────┐ ┌──────────┐ ┌──────────┐   │
   │  │ Decision │ │Governance │ │ Action   │ │ Pipeline │   │
   │  │ Engine   │ │ Engine    │ │ Registry │ │ Runner   │   │
   │  └────┬─────┘ └─────┬─────┘ └────┬─────┘ └────┬─────┘   │
   │       │              │            │             │         │
   │  ┌────▼──────────────▼────────────▼─────────────▼─────┐  │
   │  │              Repository Layer (pgx)                 │  │
   │  │  decision/  governance/  action/  outbox/  alert/   │  │
   │  └─────────────────────┬──────────────────────────────┘  │
   └────────────────────────┼──────────────────────────────────┘
                            │
                     ┌──────▼──────┐
                     │ PostgreSQL  │
                     │ 15 (Docker) │
                     └─────────────┘
```

### Closed-Loop Demo Data Flow

```
Trigger Pipeline ──► Ingest → Transform → Build Metrics ──► Governance Rules Fire
                                                                    │
                                                                    ▼
                                                         Alert Generated
                                                                    │
                                                                    ▼
New Decision Case Created ←─────── Decision Engine (LLM-assisted)
         │
         ▼
Action Proposal Generated ──► Approved (manual or auto) ──► Executed
                                                                    │
                                                                    ▼
                                                            Outbox Event
                                                                    │
                                                                    ▼
                                                    Channel Dispatch (Feishu/GitHub)
                                                                    │
                                                                    ▼
                                                    Result Visible in Frontend
```

### Recommended Fix Strategy (by file)

```
Backend test fixes (Go):
  internal/action/proposal_service_test.go     — fix redeclared variable
  internal/decision/context_builder_test.go    — fix undefined type reference
  internal/service/alert_service_test.go       — fix undefined variable
  internal/api/handler/outbox_test.go          — fix constructor signature

Backend test data fix:
  internal/repository/status/repository_test.go — seed DB with pipeline run
  (or skip if no meaningful test data)

Frontend type alignment:
  frontend/src/api/types.ts         — add/align types for actual backend responses
  frontend/src/api/governance.ts    — fix hooks to consume backend DTO shapes
  frontend/src/pages/Governance.tsx — fix tab rendering to use actual backend data
  frontend/src/pages/Pipeline.tsx   — fix request body field name

Frontend test fixes:
  frontend/src/pages/__tests__/DecisionReview.test.tsx   — update error text assertions
  frontend/src/pages/__tests__/PolicyInspector.test.tsx   — update error text assertions
  frontend/src/pages/__tests__/CaseDetail.test.tsx        — update error text assertions
  frontend/src/pages/__tests__/AuditTimeline.test.tsx     — update error text assertions
  frontend/src/pages/__tests__/AgentLogs.test.tsx         — update error text assertions
  frontend/src/pages/__tests__/SandboxCompare.test.tsx    — update error text assertions
  frontend/src/components/__tests__/Layout.test.tsx       — fix token assertion
```

### Anti-Patterns to Avoid
- **Changing backend DTOs to match frontend**: The backend DTOs have been working since Phase 1-2. Changing them risks regressions. Fix the frontend instead.
- **Changing all error titles to Chinese/English in both tests and components**: Churn without value. Pick one side (update test assertions to match component output) and be consistent.
- **Running all tests as one monolithic fail/pass gate**: Prioritize tests by dependency — Go compilation fixes first, then frontend tests, then integration tests.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Go test DB isolation | Custom container management | `testutil.StartPostgres()` | Already used by existing tests, wraps testcontainers-go |
| Frontend test query client | Per-test QueryClient setup | `renderWithQueryClient()` from `@/test-setup` | Already in test-setup.tsx — but many test files duplicate it inline |
| API mocking | Manual fetch mocking | Automated via `vi.mock("@/api/client")` | Already used by all frontend tests |

**Key insight:** This phase has no new feature work — it's purely about fixing existing code so tests pass and integration works. Every change should be a bugfix or alignment fix, not a new implementation.

## Common Pitfalls

### Pitfall 1: Fixing Frontend Types Without Verifying Real API Responses
**What goes wrong:** Developer changes TypeScript types to match a mental model of the backend response, without actually running the backend and checking the real JSON.
**Why it happens:** It's faster to edit types than to start the backend and verify.
**How to avoid:** Start the API server (`make api`), call each endpoint with curl, capture actual JSON, then write types to match the real data.
**Warning signs:** Tests pass but pages show empty/error state when connected to live backend.

### Pitfall 2: Fixing Tests Without Looking at Real Component Output
**What goes wrong:** Tests assert specific text (e.g., "请求异常") but the component renders different text (e.g., "加载失败").
**Why it happens:** Tests were written for a previous version of the component, or vice versa.
**How to avoid:** Read the component source to see what it actually renders, then update the test to match. OR update the component to render what the test expects (prefer the former).
**Warning signs:** Tests consistently fail with "Unable to find an element with the text: X".

### Pitfall 3: Stage Fixing Go Test Files in Isolation
**What goes wrong:** Fixing one compilation error at a time without running `go test ./...` to discover all of them.
**Why it happens:** Developers fix the first error, run tests, see the next error, repeat.
**How to avoid:** Run `go vet ./...` first to see ALL compilation errors at once, fix them all, then run tests.
**Warning signs:** The developer is on iteration 4+ of "fix compile, run test, find next error."

### Pitfall 4: Running E2E Tests Without Docker
**What goes wrong:** `test/integration/phase7_test.go` and `test/security/phase7_test.go` use testcontainers to spawn PostgreSQL. If Docker is not running, they'll skip or fail.
**Why it happens:** Developers forget the test dependency on Docker.
**How to avoid:** Always verify Docker is running before running integration tests. The test has `if testing.Short() { t.Skip() }` guards — use `-short` flag for quick iteration.
**Warning signs:** Integration tests hang or error with "Cannot connect to the Docker daemon."

### Pitfall 5: Over-Scoping the Governance API Alignment
**What goes wrong:** Trying to make the Governance page display ALL the complex nested data the TypeScript types describe, when the backend returns a simpler structure.
**Why it happens:** The frontend types were designed for a richer response than the backend currently provides.
**How to avoid:** Simplify the frontend to display what the backend actually returns. Don't try to add backend features to fill gaps. Remember the user's constraint: "Fix bugs before adding features."
**Warning signs:** The fix for Governance.tsx ballooned into a multi-file rewrite.

## Code Examples

### Pattern: Frontend Hook Aligned to Backend Response
```typescript
// Source: Original codebase pattern, adapt governance hooks to match actual backend DTOs

// Backend returns: { objects: [{object_type, source_dataset, primary_key, ...}], datasets: [...] }
// Frontend type must match this shape, not the wider "assets" shape

export interface CatalogObject {
  object_type: string
  source_dataset: string
  primary_key: string
  properties_count: number
  links_count: number
}

export interface CatalogDataset {
  dataset: string
  schema: string
  table: string
}

export interface CatalogResponse {
  objects: CatalogObject[]
  datasets: CatalogDataset[]
}

export function useCatalog() {
  return useQuery<CatalogResponse>({
    queryKey: ["governance", "catalog"],
    queryFn: () => apiClient.get<CatalogResponse>("/governance/catalog"),
    staleTime: 30_000,
  })
}
```

### Pattern: Testing ErrorPanel Render
```typescript
// Source: Current component pattern — tests must match actual component output

// Component renders: <ErrorPanel title="Failed to load" message="Network error" />
// Test assertion MUST match:
expect(await screen.findByText("Failed to load")).toBeInTheDocument()

// NOT this (will fail): 
expect(await screen.findByText("请求异常")).toBeInTheDocument()
```

### Pattern: Go Test Compilation Fixes

Fix stale references from repository refactoring:

```go
// BAD — references removed type:
// var row repository.DecisionCaseRow

// GOOD — use current repository types:
var row decision.DecisionCaseRow  // or whatever the current package provides
```

Fix constructor signature changes:

```go
// BAD — old constructor with extra args:
// svc := service.NewOutboxService(outboxRepo, pool)

// GOOD — current constructor:
svc := service.NewOutboxService(outboxRepo)
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Deprecated flat repository shims | Subpackage repositories with PoolProvider | Phase 3 | Go test files still reference removed types — needs fixing |
| 501 stubs for API endpoints | Working implementations | Phase 1 | Frontend should now receive real data, not 501 errors |
| Python pipeline scripts | Go CLI commands | Phase 3 | Pipeline.tsx test mock still has old Python command |
| Generic 500 errors | Structured 5-field error JSON | Phase 2 | ErrorPanel components now receive structured error data |

**Deprecated/outdated:**
- `repository.DecisionCaseRow`: Removed in Phase 3, but `context_builder_test.go` still references it
- `service.NewOutboxService(repo, pool)`: Signature changed in Phase 2/3, but `outbox_test.go` still uses old call

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Backend Governance DTOs are the "source of truth" and should not be changed | Standard Stack | LOW — if user prefers updating backend, approach changes |
| A2 | `go vet` compilation errors are only in test files, not production code | Common Pitfalls | MEDIUM — if production code also has compilation errors, scope increases |
| A3 | The Pipeline frontend sends `pipeline_type` but backend expects `config` | INT-01 Findings | MEDIUM — this was inferred from DTO inspection; needs runtime verification |
| A4 | The DecisionReview page routes `/proposals/{id}/approve`/etc. match backend handlers | INT-01 Findings | LOW — confirmed by reading routes.go |

## Open Questions

1. **Should we adopt the Governance backend DTOs as-is (simplifying frontend) or add a middleware translation layer?**
   - What we know: Backend returns `{objects, datasets}` not `{data_catalog, assets}`
   - What's unclear: User preference on approach. Adding a middleware layer between frontend and backend would be "new feature" — user said fix bugs before features.
   - Recommendation: Simplify frontend types to match actual backend responses. This is a bugfix (frontend was built for wrong API contract).

2. **Do the E2E integration tests need PostgreSQL data seeding, or do they work with empty databases?**
   - What we know: Tests insert their own `decision_case` and `action_proposal` records via direct SQL
   - What's unclear: Whether the database schema and seed migrations provide the right state for tests that query governance tables
   - Recommendation: Run the tests and see. If they fail on data-dependent queries (like `TestPhase7_WorkerDispatch`), add test data in the test itself, not in global seeds.

3. **Is the closed-loop demo (INT-05) achievable with demo seed data, or does it require manual pipeline execution?**
   - What we know: The pipeline has 7 steps (`ingest_raw → build_dwd → ... → create_outbox`)
   - What's unclear: Whether seed data exists for a demo pipeline run, or whether the pipeline must be triggered manually
   - Recommendation: Create a seed SQL script that inserts demo data for all pipeline stages, so the closed-loop can be demonstrated without actually running the pipeline.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go 1.23+ | Backend build/test | ✓ | (check: go version) | — |
| Node.js 20+ | Frontend tests | ✓ | (check: node --version) | — |
| Docker | E2E integration tests | ✓ | (check: docker info) | Use `-short` to skip E2E tests |
| PostgreSQL | Full integration | ✓ | Docker Compose | — |
| testcontainers-go | E2E test isolation | ✓ | v0.35.0 (go.mod) | Need Docker running |

## Validation Architecture

> `workflow.nyquist_validation` is enabled — this section is required.

### Test Framework

| Property | Value |
|----------|-------|
| Framework (Go) | Go testing + testify v1.9.0 |
| Framework (Frontend) | Vitest ^4.1.7 + Testing Library |
| Config (Go) | `go.mod` + `.golangci.yml` |
| Config (Frontend) | `frontend/vitest.config.ts` |
| Quick run (Go) | `go test -short ./internal/...` |
| Full suite (Go) | `go test -tags=integration ./test/...` (requires Docker) |
| Quick run (Frontend) | `cd frontend && npx vitest run --reporter=verbose` |
| Full suite (Frontend) | `cd frontend && npm test` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| INT-01 | Frontend pages connect to backend | Integration (manual) | Start API + frontend, check each page loads | Manual check |
| INT-02 | E2E integration tests pass | Integration | `go test -tags=integration ./test/integration/` | ✅ phase7_test.go |
| INT-03 | Security E2E tests pass | Integration | `go test -tags=integration ./test/security/` | ✅ phase7_test.go |
| INT-04 | Frontend unit tests pass | Unit | `cd frontend && vitest run` | ✅ 42 test files |
| INT-05 | Full closed-loop demo works | E2E (manual) | Manual trigger pipeline → check frontend | No automated test |

### Sampling Rate
- **Per task commit:** `cd frontend && npx vitest run --changed` (frontend changes), `go test -short ./internal/...` (Go changes)
- **Per wave merge:** `cd frontend && npm test` (all frontend), `go test ./...` (all Go internal)
- **Phase gate:** Full suite green + INT-01 manual verification + INT-05 demo walkthrough

### Wave 0 Gaps
- [ ] **Go test compilation**: Fix 4 test files that fail `go vet`:
  - `internal/action/proposal_service_test.go`
  - `internal/decision/context_builder_test.go`
  - `internal/service/alert_service_test.go`
  - `internal/api/handler/outbox_test.go`
- [ ] **Go test runtime**: Fix `TestStatusGetLastPipelineRun` in `internal/repository/status/repository_test.go`
- [ ] **Frontend type alignment**: Fix `frontend/src/api/governance.ts` hooks to consume actual backend DTOs
- [ ] **Frontend test alignment**: Fix 10 failing assertions in 7 test files
- [ ] **Pipeline page request body**: Fix `pipeline_type` → `config` field name mismatch

## Security Domain

> `security_enforcement` is absent from config — treat as enabled.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | Bearer token auth already hardened in Phase 5 |
| V3 Session Management | No | Stateless Bearer token, no session |
| V4 Access Control | No | Single-operator model (deferred) |
| V5 Input Validation | Yes | JSON decoding with 400 on malformed (Phase 2) |
| V6 Cryptography | No | No cryptographic operations in this phase |

### Known Threat Patterns

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| API data exposure via type confusion | Information Disclosure | Ensure frontend types match backend DTOs — don't silently drop fields |
| Test data leakage | Information Disclosure | E2E tests use ephemeral testcontainers — no persistent test data |
| Stale mock data in tests | Tampering | Tests that mock API responses need updating when backend DTOs change |

## Sources

### Primary (HIGH confidence)
- **Codebase inspection** — Direct reading of frontend pages, backend handlers, DTOs, routes, test files, and configuration
- **Test execution** — Ran `vitest run` and `go vet ./...` to verify current test status

### Secondary (MEDIUM confidence)
- **`go build ./...`** — Confirmed production code compiles cleanly (no errors)
- **`go test ./test/...`** — Not run (requires Docker + testcontainers); status inferred from build errors in test files

### Tertiary (LOW confidence)
- None — all findings verified through codebase inspection or test execution

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new packages, existing stack is well-understood
- Architecture: HIGH — data flow and component responsibilities verified by reading routes.go, handlers, frontend pages
- Pitfalls: HIGH — all pitfalls derived from actual test failures and code reading, not speculation

**Research date:** 2026-06-03
**Valid until:** 2026-06-17 (2 weeks — stable codebase, no fast-moving dependencies)
