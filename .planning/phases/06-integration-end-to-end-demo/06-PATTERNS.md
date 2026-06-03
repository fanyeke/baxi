# Phase 6: Integration & End-to-End Demo - Pattern Map

**Mapped:** 2026-06-03
**Files analyzed:** 16 (9 Go backend test files, 2 frontend API files, 2 frontend page components, 7 frontend test files)
**Analogs found:** 16 / 16

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/action/proposal_service_test.go` | test | CRUD | `internal/action/proposal_service.go` | exact |
| `internal/decision/context_builder_test.go` | test | CRUD | `internal/decision/context_builder.go` | exact |
| `internal/service/alert_service_test.go` | test | CRUD | `internal/service/alert_service.go` | exact |
| `internal/api/handler/outbox_test.go` | test | CRUD | `internal/api/handler/outbox.go` | exact |
| `internal/repository/status/repository_test.go` | test | CRUD | `internal/repository/status/repository.go` | exact |
| `frontend/src/api/governance.ts` | hooks/API | request-response | `frontend/src/api/client.ts` | role-match |
| `frontend/src/api/types.ts` | type definitions | request-response | `frontend/src/api/types.ts` (self) | exact |
| `frontend/src/pages/Governance.tsx` | component | request-response | `frontend/src/pages/Governance.tsx` (self) | exact |
| `frontend/src/pages/Pipeline.tsx` | component | request-response | `frontend/src/pages/Pipeline.tsx` (self) | exact |
| `frontend/src/pages/__tests__/DecisionReview.test.tsx` | test | request-response | `frontend/src/pages/__tests__/DecisionReview.test.tsx` (self) | exact |
| `frontend/src/pages/__tests__/PolicyInspector.test.tsx` | test | request-response | `frontend/src/pages/__tests__/PolicyInspector.test.tsx` (self) | exact |
| `frontend/src/pages/__tests__/CaseDetail.test.tsx` | test | request-response | `frontend/src/pages/__tests__/CaseDetail.test.tsx` (self) | exact |
| `frontend/src/pages/__tests__/AuditTimeline.test.tsx` | test | request-response | `frontend/src/pages/__tests__/AuditTimeline.test.tsx` (self) | exact |
| `frontend/src/pages/__tests__/AgentLogs.test.tsx` | test | request-response | `frontend/src/pages/__tests__/AgentLogs.test.tsx` (self) | exact |
| `frontend/src/pages/__tests__/SandboxCompare.test.tsx` | test | request-response | `frontend/src/pages/__tests__/SandboxCompare.test.tsx` (self) | exact |
| `frontend/src/components/__tests__/Layout.test.tsx` | test | request-response | `frontend/src/components/__tests__/Layout.test.tsx` (self) | exact |

## Pattern Assignments

### `internal/action/proposal_service_test.go` (test, CRUD)

**Analog:** `internal/action/proposal_service.go` (production code the test covers)

**Compilation Issues:**
1. **Duplicate import** (lines 10-11): `decisionRepo "baxi/internal/repository/decision"` declared twice
2. **Mock signature mismatch** — mock concrete functions include `pool *pgxpool.Pool` parameter that doesn't exist in the `ProposalRepository` interface

**Actual interface** (`internal/action/proposal_service.go` lines 34-37):
```go
type ProposalRepository interface {
    CreateProposal(ctx context.Context, row *decisionRepo.ActionProposalRow) error
    ListProposalsByCase(ctx context.Context, caseID string) ([]decisionRepo.ActionProposalRow, error)
}
```

**Actual interface** (`internal/action/proposal_service.go` lines 40-42):
```go
type CaseStatusUpdater interface {
    UpdateCaseStatus(ctx context.Context, caseID string, status string, contextJSON *json.RawMessage, contextHash *string, governanceSnapshot *json.RawMessage) error
}
```

**Fix pattern — remove `pool` from mock function types** (lines 19, 33):
```go
// BAD — has pool param not in interface:
createProposalFn      func(ctx context.Context, pool *pgxpool.Pool, row *decisionRepo.ActionProposalRow) error

// GOOD — match interface exactly:
createProposalFn      func(ctx context.Context, row *decisionRepo.ActionProposalRow) error
```

**Fix pattern — remove `pool` from mock calls** (lines 25, 37):
```go
// BAD — passes pool not in interface:
return m.createProposalFn(ctx, pool, row)

// GOOD — match interface:
return m.createProposalFn(ctx, row)
```

**Fix pattern — remove `pool` from inline lambda definitions** (lines 54, 61-68, 141, 148-149, 178, 185, 214, 221, 249, 256, 319, 396, 403, 487-488, 500-501, 605-606, 619):
```go
// BAD:
createProposalFn: func(ctx context.Context, pool *pgxpool.Pool, row *decisionRepo.ActionProposalRow) error {

// GOOD:
createProposalFn: func(ctx context.Context, row *decisionRepo.ActionProposalRow) error {
```

**Also duplicate import removal** (lines 10-11):
```go
// BAD — duplicate:
import (
    ...
    decisionRepo "baxi/internal/repository/decision"
    decisionRepo "baxi/internal/repository/decision"  // DUPLICATE
    ...
)

// GOOD — single import:
import (
    ...
    decisionRepo "baxi/internal/repository/decision"
    ...
)
```

---

### `internal/decision/context_builder_test.go` (test, CRUD)

**Analog:** `internal/decision/context_builder.go` (production interface the test mocks)

**Compilation Issues:**
1. Uses `repository.DecisionCaseRow` but type is in `decisionRepo "baxi/internal/repository/decision"` subpackage
2. Mock signatures include `pool *pgxpool.Pool` parameter not in production interfaces

**Actual interface** (`internal/decision/context_builder.go` lines 29-32):
```go
type DecisionCaseDataProvider interface {
    GetCaseByID(ctx context.Context, caseID string) (*decisionRepo.DecisionCaseRow, error)
    GetCaseBySource(ctx context.Context, sourceType, sourceID string) (*decisionRepo.DecisionCaseRow, error)
}
```

**Fix pattern — add import** (after line 10):
```go
import (
    ...
    "baxi/internal/repository"
    decisionRepo "baxi/internal/repository/decision"
    ...
)
```

**Fix pattern — change type references** (28 occurrences):
```go
// BAD — type not in flat repository package:
*repository.DecisionCaseRow

// GOOD — type is in decision subpackage:
*decisionRepo.DecisionCaseRow
```

**Fix pattern — remove pool from mock method signatures** (lines 22-26):
```go
// BAD:
func (m *mockDecisionCaseDataProvider) GetCaseByID(ctx context.Context, pool *pgxpool.Pool, caseID string) (*decisionRepo.DecisionCaseRow, error) {
func (m *mockDecisionCaseDataProvider) GetCaseBySource(ctx context.Context, pool *pgxpool.Pool, sourceType, sourceID string) (*decisionRepo.DecisionCaseRow, error) {

// GOOD:
func (m *mockDecisionCaseDataProvider) GetCaseByID(ctx context.Context, caseID string) (*decisionRepo.DecisionCaseRow, error) {
func (m *mockDecisionCaseDataProvider) GetCaseBySource(ctx context.Context, sourceType, sourceID string) (*decisionRepo.DecisionCaseRow, error) {
```

**Fix pattern — remove pool from mock function field types** (lines 18-19):
```go
// BAD:
getCaseByIDFn     func(ctx context.Context, pool *pgxpool.Pool, caseID string) (*decisionRepo.DecisionCaseRow, error)
getCaseBySourceFn func(ctx context.Context, pool *pgxpool.Pool, sourceType, sourceID string) (*decisionRepo.DecisionCaseRow, error)

// GOOD:
getCaseByIDFn     func(ctx context.Context, caseID string) (*decisionRepo.DecisionCaseRow, error)
getCaseBySourceFn func(ctx context.Context, sourceType, sourceID string) (*decisionRepo.DecisionCaseRow, error)
```

**Fix pattern — remove pool from all inline lambda definitions** (21 occurrences across all test functions):
```go
// BAD:
getCaseByIDFn: func(ctx context.Context, pool *pgxpool.Pool, caseID string) (*decisionRepo.DecisionCaseRow, error) {

// GOOD:
getCaseByIDFn: func(ctx context.Context, caseID string) (*decisionRepo.DecisionCaseRow, error) {
```

**Note:** `repository.ObjectInstance` still works via type alias in `internal/repository/ontology_aware_repo.go` line 15:
```go
type ObjectInstance = ontologyRepo.ObjectInstance
```
So references to `repository.ObjectInstance` do NOT need to be changed.

---

### `internal/service/alert_service_test.go` (test, CRUD)

**Analog:** `internal/service/alert_service.go` (production code the test covers)

**Compilation Issues:**
1. `"baxi/internal/repository"` import is wrong — service uses `alertRepo "baxi/internal/repository/alert"`
2. `NewAlertService(alertRepo.NewRepository(nil), pool)` has extra `pool` parameter

**Fix pattern — replace import** (line 14):
```go
// BAD — flat package doesn't expose the right type:
import "baxi/internal/repository"

// GOOD — use the subpackage:
import alertRepo "baxi/internal/repository/alert"
```

**Fix pattern — constructor call** (lines 86, 156):
```go
// BAD — pool param not in constructor:
svc := NewAlertService(alertRepo.NewRepository(nil), pool)

// GOOD — constructor only takes repo:
svc := NewAlertService(alertRepo.NewRepository(nil))
```

---

### `internal/api/handler/outbox_test.go` (test, CRUD)

**Analog:** `internal/api/handler/outbox.go` and `internal/service/outbox_service.go`

**Compilation Issues:**
1. `service.NewOutboxService(readRepo, pool)` has extra `pool` parameter
2. Optionally unused imports if no longer referenced

**Fix pattern — constructor call** (line 90):
```go
// BAD — pool param not in current constructor:
svc := service.NewOutboxService(readRepo, pool)

// GOOD — constructor only takes repo:
svc := service.NewOutboxService(readRepo)
```

**Current constructor** (`internal/service/outbox_service.go` line 18):
```go
func NewOutboxService(repo *outboxRepo.Repository) *OutboxService {
```

---

### `internal/repository/status/repository_test.go` (test, CRUD)

**Analog:** `internal/repository/status/repository.go`

**Current status:** This file already compiles correctly. The `TestStatusGetLastPipelineRun` test (line 106-114) works as-is — it inserts a row and queries it. No fix needed.

**Pattern for reference** (lines 70-80 — test setup pattern):
```go
func setupRepo(t *testing.T) (*Repository, *common.PoolProvider) {
    t.Helper()
    pool := testutil.SetupTestPool(t)
    ctx := context.Background()
    _, err := pool.Exec(ctx, statusDDL)
    require.NoError(t, err)
    for _, tbl := range []string{"ops.metric_alert", "ops.task", "ops.outbox_event", "audit.pipeline_run"} {
        _, _ = pool.Exec(ctx, "TRUNCATE TABLE "+tbl+" CASCADE")
    }
    return NewRepository(common.NewPoolProvider(pool)), common.NewPoolProvider(pool)
}
```

---

### `frontend/src/api/governance.ts` (hooks/API, request-response)

**Analog:** `frontend/src/api/client.ts` (API client pattern)

**Required changes:**
1. Replace `CatalogAsset` interface with `CatalogObject` + `CatalogDataset`
2. Replace `CatalogResponse` to match backend `{objects, datasets}` shape
3. Keep all other hooks (Classification, Marking, Lineage, Checkpoints, Health) unchanged — they match backend DTOs

**New types to add** (replacing lines 4-17):
```typescript
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
```

**Hook pattern — keep existing** (lines 97-103, unchanged):
```typescript
export function useCatalog() {
  return useQuery<CatalogResponse>({
    queryKey: ["governance", "catalog"],
    queryFn: () => apiClient.get<CatalogResponse>("/governance/catalog"),
    staleTime: 30_000,
  })
}
```

---

### `frontend/src/api/types.ts` (type definitions, request-response)

**Analog:** self (already correct)

**No changes required.** All types already match backend DTOs based on codebase inspection. The `PipelineRunResponse` (lines 167-174) is:
```typescript
export interface PipelineRunResponse {
  command: string
  pipeline_type: string
  estimated_duration: string
  required_env_vars: string[]
  warnings: string[]
  description: string
}
```
This matches what the backend returns. Only the request body (in Pipeline.tsx) changes from `pipeline_type` to `config`.

---

### `frontend/src/pages/Governance.tsx` (component, request-response)

**Analog:** self (existing component)

**Required changes:**
1. Update `CatalogTab` to use `data.objects` instead of `data.assets`
2. Change columns from `asset_id/name/type/location/description/grain/status` to `object_type/source_dataset/primary_key/properties_count/links_count`
3. Update imports to remove `CatalogAsset`, add `CatalogObject`

**New import** (lines 10-18):
```typescript
import type {
  CatalogObject, // replaced CatalogAsset
  Classification,
  MarkingInfo,
  LineageEdge,
  CheckpointRule,
  HealthCheck,
  MonitoringView,
} from "../api/governance"
```

**Fixed CatalogTab** (lines 79-99):
```typescript
function CatalogTab({ data, isLoading, error }: ReturnType<typeof useCatalog>) {
  if (isLoading) return <LoadingSkeleton type="table" count={6} />
  if (error) return <ErrorPanel title="加载失败" message={fmtError(error)} />
  if (!data || data.objects.length === 0) return <EmptyState title="暂无数据目录" />

  return (
    <DataTable headers={["对象类型", "来源数据集", "主键", "属性数", "链接数"]}>
      {data.objects.map((o: CatalogObject, i: number) => (
        <tr key={o.primary_key + i} className="border-t hover:bg-muted/50">
          <td className="p-2 font-medium">{o.object_type}</td>
          <td className="p-2 font-mono text-xs">{o.source_dataset}</td>
          <td className="p-2 font-mono text-xs">{o.primary_key}</td>
          <td className="p-2">{o.properties_count}</td>
          <td className="p-2">{o.links_count}</td>
        </tr>
      ))}
    </DataTable>
  )
}
```

**Fixed SummaryStats** (line 353):
```typescript
// BAD:
catalogCount={catalog.data?.assets.length ?? 0}

// GOOD:
catalogCount={catalog.data?.objects.length ?? 0}
```

---

### `frontend/src/pages/Pipeline.tsx` (component, request-response)

**Analog:** self (existing component)

**Required change** (line 18 — request body field name):
```typescript
// BAD — backend expects "config" not "pipeline_type":
mutationFn: () => apiClient.post<PipelineRunResponse>("/pipeline/run", { pipeline_type: pipelineType }),

// GOOD:
mutationFn: () => apiClient.post<PipelineRunResponse>("/pipeline/run", { config: pipelineType }),
```

---

### Frontend Test Files (7 files, test, request-response)

**Shared pattern — all 7 files have the same error text assertion fix:**

**Analog:** `frontend/src/components/ErrorPanel.tsx` (the component under test renders `title="加载失败"`)

**Fix pattern** — update "请求异常" → "加载失败" in all 7 test files:

**DecisionReview.test.tsx** (line 72):
```typescript
// BAD:
expect(await screen.findByText("请求异常")).toBeInTheDocument()
// GOOD:
expect(await screen.findByText("加载失败")).toBeInTheDocument()
```

**PolicyInspector.test.tsx** (line 47):
```typescript
// BAD:
expect(await screen.findByText("请求异常")).toBeInTheDocument()
// GOOD:
expect(await screen.findByText("加载失败")).toBeInTheDocument()
```

**CaseDetail.test.tsx** (line 58):
```typescript
// BAD:
expect(await screen.findByText("请求异常")).toBeInTheDocument()
// GOOD:
expect(await screen.findByText("加载失败")).toBeInTheDocument()
```

**AuditTimeline.test.tsx, AgentLogs.test.tsx, SandboxCompare.test.tsx** — same pattern:
```typescript
// Replace all "请求异常" with "加载失败"
expect(await screen.findByText("加载失败")).toBeInTheDocument()
```

**Layout.test.tsx** (line 58) — fix token default assertion:
```typescript
// BAD — SessionStorage default is empty string, not a test token:
expect(sessionStorage.getItem("API_BEARER_TOKEN")).toBe("test-token-for-dev-...")

// GOOD — SessionStorage starts empty:
expect(sessionStorage.getItem("API_BEARER_TOKEN")).toBe("")
```

**Complete list of specific changes per test file:**

| File | Line | Change |
|------|------|--------|
| `DecisionReview.test.tsx` | 72 | `"请求异常"` → `"加载失败"` |
| `PolicyInspector.test.tsx` | 47 | `"请求异常"` → `"加载失败"` |
| `CaseDetail.test.tsx` | 58 | `"请求异常"` → `"加载失败"` |
| `AuditTimeline.test.tsx` | — | `"请求异常"` → `"加载失败"` |
| `AgentLogs.test.tsx` | — | `"请求异常"` → `"加载失败"` |
| `SandboxCompare.test.tsx` | — | `"请求异常"` → `"加载失败"` |
| `Layout.test.tsx` | 58 | token default → `""` |

---

## Shared Patterns

### Go Test Fix Pattern — Repository Subpackage Migration

**Source:** All Go test files with stale references from Phase 3 repository refactoring

**The core pattern change:**

The flat `internal/repository` package no longer re-exports all types. Types have been moved to subpackages:

```go
// OLD (flat)                // NEW (subpackage)
repository.DecisionCaseRow   → decisionRepo.DecisionCaseRow  (in internal/repository/decision/)
repository.ObjectInstance    → repository.ObjectInstance     (still works via type alias in ontology_aware_repo.go)
```

**Constructor signatures changed** — no longer pass `pool`:
```go
// OLD:
service.NewAlertService(alertRepo.NewRepository(nil), pool)
service.NewOutboxService(readRepo, pool)

// NEW:
service.NewAlertService(alertRepo.NewRepository(nil))
service.NewOutboxService(readRepo)
```

**Mock interface signatures no longer take `pool` parameter**:
```go
// OLD mock signatures:
GetCaseByID(ctx context.Context, pool *pgxpool.Pool, caseID string)
CreateProposal(ctx context.Context, pool *pgxpool.Pool, row *Row)
UpdateCaseStatus(ctx context.Context, pool *pgxpool.Pool, ...)

// NEW mock signatures (match actual interfaces):
GetCaseByID(ctx context.Context, caseID string)
CreateProposal(ctx context.Context, row *Row)
UpdateCaseStatus(ctx context.Context, caseID string, status string, ...)
```

### Frontend Test Fix Pattern — Error Text Assertion

**Source:** `frontend/src/components/ErrorPanel.tsx` line 16 (`title` hardcoded as `"加载失败"`)

**Pattern:** All frontend tests mock API failure with `mockRejectedValue()` and assert the ErrorPanel title. The ErrorPanel renders `title="加载失败"` (Chinese). Tests previously asserted `"请求异常"` (a different Chinese error text). Fix all test files to assert the actual rendered title.

```typescript
// When component renders:
<ErrorPanel title="加载失败" message={error.message} />

// Test must assert:
expect(await screen.findByText("加载失败")).toBeInTheDocument()
```

### Frontend Type Alignment Pattern

**Source:** Backend Go DTOs (source of truth)

**Pattern:** Simplify frontend types to match actual backend JSON shapes. Backend is source of truth — no backend changes.

```typescript
// Backend returns: { objects: [...], datasets: [...] }
// Frontend must match this, not the wider "assets" shape
export interface CatalogObject {
  object_type: string
  source_dataset: string
  primary_key: string
  properties_count: number
  links_count: number
}
```

---

## No Analog Found

All 16 files have close analogs — no files without matches.

| File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| All 16 | — | — | Same file or its production counterpart | exact |

---

## Metadata

**Analog search scope:** 
- `internal/action/`, `internal/decision/`, `internal/service/`, `internal/api/handler/`, `internal/repository/status/` (Go)
- `frontend/src/api/`, `frontend/src/pages/`, `frontend/src/components/` (TypeScript/React)
- `frontend/src/pages/__tests__/`, `frontend/src/components/__tests__/` (frontend tests)

**Files scanned:** ~50 files (all relevant Go backend test files and frontend source files)

**Pattern extraction date:** 2026-06-03
