# Baxi Architecture & Code Quality Cleanup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor Baxi architecture to eliminate anti-patterns (flat repository package, pool parameter passing, DTO reverse dependencies) while maintaining 100% test pass rate.

**Architecture:** 
1. Restructure `internal/repository/` into domain-based subpackages with pool injection
2. Create `internal/model/` package to decouple service layer from API DTOs
3. Split large files (`server.go` 691 lines, `feishu_service.go` 967 lines) into focused modules
4. Add golangci-lint configuration for code quality enforcement

**Tech Stack:** Go 1.22, PostgreSQL (pgx), chi router, testify, testcontainers

**Estimated Duration:** ~5 weeks (24 days)

---

## Phase 0: Pre-Flight Check (Current State Assessment)

### Task 0.1: Document Current Test Failures

**Files:**
- Read: `internal/api/health_test.go`, `internal/api/integration_test.go`
- Read: `internal/config/config_test.go`
- Read: `internal/repository/decision_repository_test.go`
- Read: `internal/action/registry_test.go`
- Read: `test/integration/phase7_test.go`

- [ ] **Step 1: Run full test suite and capture failures**

Run:
```bash
cd /home/zzz/project/baxi
go test ./... 2>&1 | tee /tmp/test_failures.txt
echo "Test run complete. Check /tmp/test_failures.txt"
```

Expected: Multiple FAIL entries for packages: `internal/api`, `internal/config`, `internal/repository`, `internal/action`, `internal/adapter`, `test/integration`

- [ ] **Step 2: Categorize build failures vs runtime failures**

Build failures (compilation errors):
- `internal/api`: `New` function signature mismatch (needs 3 args, tests pass 2)
- `internal/config`: `LLMMaxRetries` field missing
- `internal/repository`: Method signature mismatch in `GetCaseBySource`

Runtime failures:
- `internal/action`: Test data assertions
- `internal/adapter`: Configuration assertions
- `test/integration`: Worker dispatch test timeout

- [ ] **Step 3: Create fix priority list**

Priority 1 (blocking):
1. Fix API server signature in tests
2. Fix Config structure
3. Fix Repository method signatures

Priority 2 (can delay):
4. Fix Action tests
5. Fix Adapter tests
6. Fix Integration tests

---

## Phase 1: Foundation Repair (Week 1)

### Task 1.1: Fix API Server Constructor Signature

**Problem:** Tests call `New(nil, nil)` but signature requires `New(logger, pool, cfg)`

**Files:**
- Modify: `internal/api/health_test.go:14-16`
- Modify: `internal/api/health_test.go:31-33`
- Modify: `internal/api/health_test.go:49-51`
- Modify: `internal/api/health_test.go:62-64`
- Modify: `internal/api/integration_test.go:35-37`
- Modify: `internal/api/integration_test.go:63-65`
- Modify: `internal/api/integration_test.go:75-77`
- Modify: `internal/api/integration_test.go:90-92`
- Modify: `internal/api/integration_test.go:106-108`
- Modify: `internal/api/integration_test.go:128-130`

- [x] **Step 1: Add required imports**

```go
// internal/api/health_test.go
import (
    "github.com/jackc/pgx/v5/pgxpool"
    "go.uber.org/zap"
    "baxi/internal/config"
)
```

- [x] **Step 2: Create test helper function**

Add at top of `health_test.go`:
```go
func newTestServer(t *testing.T) *Server {
    logger := zap.NewNop()
    cfg := &config.Config{
        APIBearerToken: "test-token",
    }
    // pool can be nil for health tests
    return New(logger, nil, cfg)
}
```

- [x] **Step 3: Replace all New(nil, nil) calls**

Replace:
```go
// OLD
server := New(nil, nil)

// NEW
server := newTestServer(t)
```

- [x] **Step 4: Run tests to verify**

Run:
```bash
cd /home/zzz/project/baxi
go test ./internal/api -run TestHealth -v
```

Expected: Tests compile and pass

- [x] **Step 5: Commit**

```bash
git add internal/api/health_test.go internal/api/integration_test.go
git commit -m "fix(api): update test constructors to match New signature"
```

---

### Task 1.2: Fix Config LLMMaxRetries Field

**Problem:** Tests reference `cfg.LLMMaxRetries` but field doesn't exist in struct

**Files:**
- Read: `internal/config/config.go`
- Modify: `internal/config/config.go`
- Test: `internal/config/config_test.go`

- [x] **Step 1: Check current Config struct definition**

Read `internal/config/config.go` and find Config struct definition.

- [x] **Step 2: Add missing LLMMaxRetries field**

```go
// internal/config/config.go
type Config struct {
    // ... existing fields ...
    
    LLMMaxRetries int `env:"LLM_MAX_RETRIES" envDefault:"3"`
}
```

- [x] **Step 3: Verify tests compile**

Run:
```bash
go test ./internal/config -v
```

Expected: Tests compile and pass

- [x] **Step 4: Commit**

```bash
git add internal/config/config.go
git commit -m "fix(config): add missing LLMMaxRetries field"
```

---

### Task 1.3: Fix Repository GetCaseBySource Method Signature

**Problem:** Tests pass `*string` but method expects `string`

**Files:**
- Read: `internal/repository/decision_repository.go` (find GetCaseBySource signature)
- Modify: `internal/repository/decision_repository_test.go:196,205`

- [x] **Step 1: Check actual method signature**

```go
// Look for:
func (r *DecisionRepository) GetCaseBySource(ctx context.Context, pool *pgxpool.Pool, sourceType, sourceID string) (*DecisionCaseRow, error)
```

- [x] **Step 2: Fix test calls**

```go
// OLD (in test)
alertStr := "alert-1"
src42Str := "src-42"
row, err := repo.GetCaseBySource(ctx, pool, &alertStr, &src42Str)  // ❌ *string

// NEW
alertStr := "alert-1"
src42Str := "src-42"
row, err := repo.GetCaseBySource(ctx, pool, alertStr, src42Str)   // ✅ string
```

- [x] **Step 3: Find and fix all occurrences**

Search for `GetCaseBySource` in test file and fix all call sites.

- [x] **Step 4: Run tests**

```bash
go test ./internal/repository -run TestGetCaseBySource -v
```

Expected: Tests pass

- [x] **Step 5: Commit**

```bash
git add internal/repository/decision_repository_test.go
git commit -m "fix(repository): correct GetCaseBySource test call signatures"
```

---

### Task 1.4: Fix Action Registry Tests

**Files:**
- Read: `internal/action/registry_test.go`
- Read: `config/action_registry.yml`
- Modify: `internal/action/registry_test.go` or `config/action_registry.yml`

- [x] **Step 1: Analyze test failures**

Run:
```bash
go test ./internal/action -v 2>&1 | head -100
```

Identify which tests are failing and why.

- [x] **Step 2: Check if action types changed**

Compare test expectations with `config/action_registry.yml`:
```go
// Tests expect:
"create_followup_task", "notify_owner", "export_report", "create_outbox_message"

// Check if config.yml contains these
```

- [x] **Step 3: Fix test data or config**

If config changed, update tests:
```go
// Update expected action types in tests
expectedActions := []string{
    "create_followup_task",
    "notify_owner",
    "export_report",
    "create_outbox_message",
}
```

- [x] **Step 4: Run tests**

```bash
go test ./internal/action -v
```

Expected: Tests pass

- [x] **Step 5: Commit**

```bash
git add internal/action/registry_test.go  # or config/action_registry.yml
git commit -m "fix(action): update test expectations to match registry config"
```

---

### Task 1.5: Fix Adapter Tests

**Files:**
- Read: `internal/adapter/feishu_test.go`
- Read: `internal/adapter/github_test.go`
- Modify: Test files as needed

- [x] **Step 1: Analyze failures**

Run:
```bash
go test ./internal/adapter -v 2>&1 | head -50
```

- [x] **Step 2: Fix test data**

Example fix for Feishu adapter:
```go
// Check if WebhookURL is expected to be non-empty
if cfg.WebhookURL == "" {
    // Test should expect specific behavior for empty URL
}
```

- [x] **Step 3: Run tests**

```bash
go test ./internal/adapter -v
```

- [x] **Step 4: Commit**

```bash
git add internal/adapter/
git commit -m "fix(adapter): update test expectations"
```

---

### Task 1.6: Phase 1 Verification

- [x] **Step 1: Run full test suite**

```bash
cd /home/zzz/project/baxi
go test ./... 2>&1 | grep -E "^(ok|FAIL)"
```

Expected: All packages except `test/integration` should pass

- [x] **Step 2: Verify build succeeds**

```bash
go build ./...
```

Expected: No compilation errors

- [x] **Step 3: Document remaining integration test failure**

Integration test failure is expected (Phase 7) and will be addressed later.

- [x] **Step 4: Phase 1 Summary Commit**

```bash
git log --oneline -10
# Verify all Phase 1 commits are present
```

---

## Phase 2: Repository Restructuring (Week 2-3)

### Task 2.1: Create Repository Subpackage Structure

**Files:**
- Create: `internal/repository/governance/`
- Create: `internal/repository/decision/`
- Create: `internal/repository/task/`
- Create: `internal/repository/alert/`
- Create: `internal/repository/outbox/`
- Create: `internal/repository/log/`
- Create: `internal/repository/common/`

- [ ] **Step 1: Create directory structure**

```bash
cd /home/zzz/project/baxi
mkdir -p internal/repository/{governance,decision,task,alert,outbox,log,common}
touch internal/repository/{governance,decision,task,alert,outbox,log,common}/.gitkeep
```

- [ ] **Step 2: Create common pool injection base**

Create `internal/repository/common/pool.go`:
```go
// Package common provides shared repository infrastructure.
package common

import (
    "context"
    
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
)

// PoolProvider provides access to database connections.
type PoolProvider struct {
    pool *pgxpool.Pool
}

// NewPoolProvider creates a new PoolProvider.
func NewPoolProvider(pool *pgxpool.Pool) *PoolProvider {
    return &PoolProvider{pool: pool}
}

// Query executes a query and returns rows.
func (p *PoolProvider) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
    return p.pool.Query(ctx, sql, args...)
}

// QueryRow executes a query and returns a single row.
func (p *PoolProvider) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
    return p.pool.QueryRow(ctx, sql, args...)
}

// Begin starts a transaction.
func (p *PoolProvider) Begin(ctx context.Context) (pgx.Tx, error) {
    return p.pool.Begin(ctx)
}

// Pool returns the underlying pool for direct access if needed.
func (p *PoolProvider) Pool() *pgxpool.Pool {
    return p.pool
}
```

- [ ] **Step 3: Commit structure**

```bash
git add internal/repository/
git commit -m "chore(repository): create subpackage structure for domain separation"
```

---

### Task 2.2: Migrate Governance Repository ✅ DONE

**Files:**
- Create: `internal/repository/governance/snapshot.go`
- Create: `internal/repository/governance/repository.go`
- Read: `internal/repository/governance_repository.go`
- Read: `internal/repository/interfaces.go` (governance interfaces)

**Migration Strategy:**
1. Create new repository with pool injection
2. Keep old file as compatibility layer
3. Migrate tests gradually
4. Remove old file when complete

- [ ] **Step 1: Create new GovernanceRepository with pool injection**

Create `internal/repository/governance/repository.go`:
```go
// Package governance provides repository access for governance domain.
package governance

import (
    "context"
    "fmt"
    
    "github.com/jackc/pgx/v5"
    "baxi/internal/repository/common"
)

// Repository provides data access for governance configuration.
type Repository struct {
    *common.PoolProvider
}

// NewRepository creates a new Governance repository.
func NewRepository(poolProvider *common.PoolProvider) *Repository {
    return &Repository{PoolProvider: poolProvider}
}

// GetConfigSnapshots retrieves all config snapshots.
func (r *Repository) GetConfigSnapshots(ctx context.Context) ([]ConfigSnapshotRow, error) {
    query := `
        SELECT config_key, config_type, source_path, content_jsonb, content_hash, created_at, updated_at
        FROM gov.config_snapshot
        ORDER BY config_key
    `
    
    rows, err := r.Query(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("query config snapshots: %w", err)
    }
    defer rows.Close()
    
    var results []ConfigSnapshotRow
    for rows.Next() {
        var row ConfigSnapshotRow
        if err := rows.Scan(
            &row.ConfigKey,
            &row.ConfigType,
            &row.SourcePath,
            &row.ContentJSONB,
            &row.ContentHash,
            &row.CreatedAt,
            &row.UpdatedAt,
        ); err != nil {
            return nil, fmt.Errorf("scan config snapshot: %w", err)
        }
        results = append(results, row)
    }
    
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("iterate config snapshots: %w", err)
    }
    
    return results, nil
}

// CountObjectSchemas returns the count of object schemas.
func (r *Repository) CountObjectSchemas(ctx context.Context) int {
    query := `SELECT COUNT(*) FROM gov.object_schema`
    
    var count int
    err := r.QueryRow(ctx, query).Scan(&count)
    if err != nil {
        return 0
    }
    return count
}

// ConfigSnapshotRow represents a config snapshot.
type ConfigSnapshotRow struct {
    ConfigKey    string
    ConfigType   string
    SourcePath   string
    ContentJSONB []byte
    ContentHash  string
    CreatedAt    string
    UpdatedAt    string
}
```

- [ ] **Step 2: Create compatibility layer**

Modify `internal/repository/governance_repository.go` to use new repository:
```go
// This file serves as a compatibility layer during migration.
// TODO: Remove this file after all call sites migrate.

package repository

import (
    "context"
    "fmt"
    
    "github.com/jackc/pgx/v5/pgxpool"
    
    "baxi/internal/repository/common"
    governanceRepo "baxi/internal/repository/governance"
)

// GovernanceRepository provides data access for governance (DEPRECATED: use governance.Repository).
type GovernanceRepository struct {
    inner *governanceRepo.Repository
}

// NewGovernanceRepository creates a new repository (DEPRECATED).
// Use governance.NewRepository(common.NewPoolProvider(pool)) instead.
func NewGovernanceRepository() *GovernanceRepository {
    return &GovernanceRepository{}
}

// SetPool sets the pool provider. Must be called before other methods.
func (r *GovernanceRepository) SetPool(pool *pgxpool.Pool) {
    r.inner = governanceRepo.NewRepository(common.NewPoolProvider(pool))
}

// GetConfigSnapshots retrieves config snapshots (DEPRECATED).
func (r *GovernanceRepository) GetConfigSnapshots(ctx context.Context, pool *pgxpool.Pool) ([]ConfigSnapshotRow, error) {
    if r.inner == nil {
        r.SetPool(pool)
    }
    
    rows, err := r.inner.GetConfigSnapshots(ctx)
    if err != nil {
        return nil, err
    }
    
    // Convert types
    var results []ConfigSnapshotRow
    for _, row := range rows {
        results = append(results, ConfigSnapshotRow{
            ConfigKey:    row.ConfigKey,
            ConfigType:   row.ConfigType,
            SourcePath:   row.SourcePath,
            ContentJSONB: row.ContentJSONB,
            ContentHash:  row.ContentHash,
            CreatedAt:    row.CreatedAt,
            UpdatedAt:    row.UpdatedAt,
        })
    }
    return results, nil
}

// CountObjectSchemas returns schema count (DEPRECATED).
func (r *GovernanceRepository) CountObjectSchemas(ctx context.Context, pool *pgxpool.Pool) int {
    if r.inner == nil {
        r.SetPool(pool)
    }
    return r.inner.CountObjectSchemas(ctx)
}
```

- [ ] **Step 3: Update GovernanceService to use new pattern**

Modify `internal/service/governance_service.go`:
```go
import (
    // ... existing imports ...
    "baxi/internal/repository/common"
    governanceRepo "baxi/internal/repository/governance"
)

// NewGovernanceService creates a new GovernanceService.
func NewGovernanceService(pool *pgxpool.Pool) *GovernanceService {
    poolProvider := common.NewPoolProvider(pool)
    return &GovernanceService{
        repo: governanceRepo.NewRepository(poolProvider),
        // ... other fields
    }
}

// Update method signatures to not pass pool:
func (s *GovernanceService) GetStatus(ctx context.Context) (*dto.GovernanceStatusResponse, error) {
    configs, err := s.repo.GetConfigSnapshots(ctx)  // No pool parameter
    // ...
}
```

- [ ] **Step 4: Run governance tests**

```bash
go test ./internal/repository/governance -v
go test ./internal/service -run TestGovernance -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/repository/governance/
git add internal/repository/governance_repository.go
git add internal/service/governance_service.go
git commit -m "refactor(repository): migrate governance to subpackage with pool injection"
```

---

### Task 2.3: Migrate Decision Repository

**Files:**
- Create: `internal/repository/decision/repository.go`
- Create: `internal/repository/decision/case.go`
- Read: `internal/repository/decision_repository.go`

- [ ] **Step 1: Create decision case repository**

Create `internal/repository/decision/case.go`:
```go
// Package decision provides repository access for decision domain.
package decision

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "time"
    
    "github.com/jackc/pgx/v5"
    "baxi/internal/repository/common"
)

// CaseRepository provides data access for decision cases.
type CaseRepository struct {
    *common.PoolProvider
}

// NewCaseRepository creates a new case repository.
func NewCaseRepository(provider *common.PoolProvider) *CaseRepository {
    return &CaseRepository{PoolProvider: provider}
}

// DecisionCaseRow represents a decision case.
type DecisionCaseRow struct {
    CaseID                 string
    AlertID                *string
    CaseType               *string
    Status                 string
    ContextJSON            *json.RawMessage
    CreatedAt              time.Time
    ResolvedAt             *time.Time
    SourceType             *string
    SourceID               *string
    ObjectType             *string
    ObjectID               *string
    Severity               *string
    ContextHash            *string
    GovernanceSnapshotJSON *json.RawMessage
    CreatedBy              *string
    ErrorMessage           *string
    UpdatedAt              *time.Time
    AlertRulesVersion      *string
    AlertRulesHash         *string
    ActionRegistryVersion  *string
    ActionRegistryHash     *string
    ContextSnapshotJSON    *json.RawMessage
    DataSnapshotJSON       *json.RawMessage
}

// GetCaseByID retrieves a case by ID.
func (r *CaseRepository) GetCaseByID(ctx context.Context, caseID string) (*DecisionCaseRow, error) {
    query := `
        SELECT case_id, alert_id, case_type, status, context_jsonb, created_at, resolved_at,
               source_type, source_id, object_type, object_id, severity, context_hash,
               governance_snapshot_jsonb, created_by, error_message, updated_at,
               alert_rules_version, alert_rules_hash, action_registry_version, action_registry_hash,
               context_snapshot_jsonb, data_snapshot_jsonb
        FROM ai.decision_case
        WHERE case_id = $1
    `
    
    row := r.QueryRow(ctx, query, caseID)
    
    var result DecisionCaseRow
    err := row.Scan(
        &result.CaseID,
        &result.AlertID,
        &result.CaseType,
        &result.Status,
        &result.ContextJSON,
        &result.CreatedAt,
        &result.ResolvedAt,
        &result.SourceType,
        &result.SourceID,
        &result.ObjectType,
        &result.ObjectID,
        &result.Severity,
        &result.ContextHash,
        &result.GovernanceSnapshotJSON,
        &result.CreatedBy,
        &result.ErrorMessage,
        &result.UpdatedAt,
        &result.AlertRulesVersion,
        &result.AlertRulesHash,
        &result.ActionRegistryVersion,
        &result.ActionRegistryHash,
        &result.ContextSnapshotJSON,
        &result.DataSnapshotJSON,
    )
    if err != nil {
        if err == pgx.ErrNoRows {
            return nil, nil
        }
        return nil, fmt.Errorf("get case by id: %w", err)
    }
    
    return &result, nil
}

// GetCaseBySource retrieves a case by source.
func (r *CaseRepository) GetCaseBySource(ctx context.Context, sourceType, sourceID string) (*DecisionCaseRow, error) {
    query := `
        SELECT case_id, alert_id, case_type, status, context_jsonb, created_at, resolved_at,
               source_type, source_id, object_type, object_id, severity, context_hash,
               governance_snapshot_jsonb, created_by, error_message, updated_at,
               alert_rules_version, alert_rules_hash, action_registry_version, action_registry_hash,
               context_snapshot_jsonb, data_snapshot_jsonb
        FROM ai.decision_case
        WHERE source_type = $1 AND source_id = $2
        ORDER BY created_at DESC
        LIMIT 1
    `
    
    row := r.QueryRow(ctx, query, sourceType, sourceID)
    
    var result DecisionCaseRow
    err := row.Scan(
        &result.CaseID,
        &result.AlertID,
        &result.CaseType,
        &result.Status,
        &result.ContextJSON,
        &result.CreatedAt,
        &result.ResolvedAt,
        &result.SourceType,
        &result.SourceID,
        &result.ObjectType,
        &result.ObjectID,
        &result.Severity,
        &result.ContextHash,
        &result.GovernanceSnapshotJSON,
        &result.CreatedBy,
        &result.ErrorMessage,
        &result.UpdatedAt,
        &result.AlertRulesVersion,
        &result.AlertRulesHash,
        &result.ActionRegistryVersion,
        &result.ActionRegistryHash,
        &result.ContextSnapshotJSON,
        &result.DataSnapshotJSON,
    )
    if err != nil {
        if err == pgx.ErrNoRows {
            return nil, nil
        }
        return nil, fmt.Errorf("get case by source: %w", err)
    }
    
    return &result, nil
}
```

- [ ] **Step 2: Create compatibility layer**

Update `internal/repository/decision_repository.go` to delegate to new repository.

- [ ] **Step 3: Run tests**

```bash
go test ./internal/repository/decision -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/repository/decision/
git commit -m "refactor(repository): migrate decision case repository to subpackage"
```

---

### Task 2.4: Migrate Remaining Repositories (Task, Alert, Outbox, Log)

Repeat the same pattern for:
- TaskRepository → `internal/repository/task/`
- AlertRepository → `internal/repository/alert/`
- OutboxRepository → `internal/repository/outbox/`
- LogRepository → `internal/repository/log/`

Each should follow the same pattern:
1. Create new repository with pool injection
2. Create compatibility layer
3. Update service to use new repository
4. Run tests
5. Commit

---

### Task 2.5: Update Repository Interfaces

**Files:**
- Modify: `internal/repository/interfaces.go`

- [ ] **Step 1: Update interfaces to use new types**

Update interfaces to reference new subpackage types or define local interfaces.

- [ ] **Step 2: Run all repository tests**

```bash
go test ./internal/repository/... -v
```

Expected: All pass

- [ ] **Step 3: Commit**

```bash
git add internal/repository/
git commit -m "refactor(repository): update interfaces for new subpackage structure"
```

---

## Phase 3: Service Layer Refactoring (Week 4)

### Task 3.1: Create Internal Model Package

**Files:**
- Create: `internal/model/task.go`
- Create: `internal/model/alert.go`
- Create: `internal/model/decision.go`
- Create: `internal/model/governance.go`
- Create: `internal/model/outbox.go`

- [ ] **Step 1: Create model types**

Create `internal/model/task.go`:
```go
// Package model provides domain models shared across layers.
package model

import "time"

// Task represents a task in the system.
type Task struct {
    TaskID           string
    TaskTitle        string
    TaskDescription  string
    Status           string
    Priority         string
    OwnerRole        string
    OwnerUserID      *string
    DueAt            *time.Time
    CreatedAt        time.Time
    CompletedAt      *time.Time
    Feedback         *string
    RecommendationID *string
    AlertID          *string
    TargetObjectType *string
    TargetObjectID   *string
}

// TaskFilters provides filter options for listing tasks.
type TaskFilters struct {
    Status   *string
    Priority *string
    Owner    *string
}

// TaskListResponse is the result of listing tasks.
type TaskListResponse struct {
    Items []Task
    Total int
}
```

Create similar files for alert, decision, governance, outbox.

- [ ] **Step 2: Commit model package**

```bash
git add internal/model/
git commit -m "feat(model): create internal model package for domain decoupling"
```

---

### Task 3.2: Refactor TaskService

**Files:**
- Read: `internal/service/task_service.go`
- Modify: `internal/service/task_service.go`
- Modify: `internal/api/handler/task.go`

- [ ] **Step 1: Update TaskService to use model**

```go
// OLD
import "baxi/internal/api/dto"

func (s *TaskService) ListTasks(ctx context.Context, filters dto.TaskFilters, limit, offset int) (*dto.TaskListResponse, error)

// NEW
import "baxi/internal/model"

func (s *TaskService) ListTasks(ctx context.Context, filters model.TaskFilters, limit, offset int) (*model.TaskListResponse, error)
```

- [ ] **Step 2: Add DTO conversion in handler**

```go
// internal/api/handler/task.go

import "baxi/internal/model"

func (h *TaskHandler) HandleList(w http.ResponseWriter, r *http.Request) {
    // Parse request
    var req dto.TaskListRequest
    // ... parse ...
    
    // Convert to model filters
    modelFilters := model.TaskFilters{
        Status:   req.Status,
        Priority: req.Priority,
        Owner:    req.Owner,
    }
    
    // Call service with model types
    result, err := h.svc.ListTasks(r.Context(), modelFilters, limit, offset)
    if err != nil {
        // handle error
    }
    
    // Convert model result to DTO
    resp := dto.TaskListResponse{
        Total: result.Total,
    }
    for _, t := range result.Items {
        resp.Items = append(resp.Items, dto.TaskItem{
            TaskID:   t.TaskID,
            TaskTitle: t.TaskTitle,
            // ... map fields ...
        })
    }
    
    httputil.JSON(w, http.StatusOK, resp)
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/service -run TestTask -v
go test ./internal/api/handler -run TestTask -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/service/task_service.go
git add internal/api/handler/task.go
git add internal/model/task.go
git commit -m "refactor(service): migrate TaskService to model package"
```

---

### Task 3.3: Refactor Remaining Services

Repeat for:
- AlertService
- DecisionService
- GovernanceService
- OutboxService
- LogService

Each service:
1. Change imports from `api/dto` to `model`
2. Update method signatures
3. Update handler to convert between model and DTO
4. Run tests
5. Commit

---

## Phase 4: File Splitting (Week 5)

### Task 4.1: Split server.go

**Files:**
- Create: `internal/api/routes.go`
- Create: `internal/api/handlers.go`
- Modify: `internal/api/server.go` (reduce to ~100 lines)

- [ ] **Step 1: Extract routes to routes.go**

Create `internal/api/routes.go`:
```go
package api

import (
    "github.com/go-chi/chi/v5"
)

// setupRoutes configures all HTTP routes.
func (s *Server) setupRoutes() {
    s.router.Route("/api/v1", func(r chi.Router) {
        // Health check (public)
        r.Get("/health", s.handleHealth)
        
        // Protected routes
        r.Group(func(r chi.Router) {
            r.Use(apimw.NewAuthMiddleware(s.bearerToken))
            
            // Status
            r.Get("/status", s.statusHandler().HandleStatus)
            
            // Alerts
            r.Get("/alerts", s.alertHandler().HandleList)
            
            // Tasks
            r.Get("/tasks", s.taskHandler().HandleList)
            
            // ... other routes ...
        })
    })
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    // Health check handler
}
```

- [ ] **Step 2: Extract handler factories to handlers.go**

Create `internal/api/handlers.go`:
```go
package api

import (
    "baxi/internal/api/handler"
    "baxi/internal/repository"
    "baxi/internal/service"
)

// statusHandler creates the status handler.
func (s *Server) statusHandler() *handler.StatusHandler {
    repo := repository.NewStatusRepository()
    svc := service.NewStatusService(repo, s.pool)
    return handler.NewStatusHandler(svc)
}

// alertHandler creates the alert handler.
func (s *Server) alertHandler() *handler.AlertHandler {
    repo := repository.NewAlertRepository()
    svc := service.NewAlertService(repo, s.pool)
    return handler.NewAlertHandler(svc)
}

// ... other handler factories ...
```

- [ ] **Step 3: Simplify server.go**

Reduce `server.go` to ~100 lines:
```go
package api

import (
    "net/http"
    "github.com/go-chi/chi/v5"
    "github.com/jackc/pgx/v5/pgxpool"
    "go.uber.org/zap"
    "baxi/internal/config"
)

// Server represents the HTTP API server.
type Server struct {
    router chi.Router
    logger *zap.Logger
    pool   *pgxpool.Pool
    cfg    *config.Config
}

// New creates a new API server instance.
func New(logger *zap.Logger, pool *pgxpool.Pool, cfg *config.Config) *Server {
    s := &Server{
        router: chi.NewRouter(),
        logger: logger,
        pool:   pool,
        cfg:    cfg,
    }
    s.setupMiddleware()
    s.setupRoutes()
    return s
}

// setupMiddleware configures global middleware.
func (s *Server) setupMiddleware() {
    s.router.Use(apimw.RequestIDMiddleware)
    s.router.Use(middleware.RealIP)
    s.router.Use(middleware.Logger)
    // ... etc ...
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/api -v
go build ./...
```

- [ ] **Step 5: Commit**

```bash
git add internal/api/server.go
git add internal/api/routes.go
git add internal/api/handlers.go
git commit -m "refactor(api): split server.go into focused modules"
```

---

### Task 4.2: Split FeishuService

**Files:**
- Create: `internal/feishu/client.go`
- Create: `internal/feishu/export.go`
- Create: `internal/feishu/sync.go`
- Modify: `internal/service/feishu_service.go` (reduce to ~200 lines)

- [ ] **Step 1: Create feishu package**

Create `internal/feishu/client.go` with HTTP client logic.
Create `internal/feishu/export.go` with CSV export logic.
Create `internal/feishu/sync.go` with data sync logic.

- [ ] **Step 2: Update service to use feishu package**

```go
// internal/service/feishu_service.go
import feishuPkg "baxi/internal/feishu"

type FeishuService struct {
    client *feishuPkg.Client
    exporter *feishuPkg.Exporter
    syncer *feishuPkg.Syncer
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/feishu -v
go test ./internal/service -run TestFeishu -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/feishu/
git add internal/service/feishu_service.go
git commit -m "refactor(feishu): extract into separate package"
```

---

## Phase 5: Code Quality (Week 6)

### Task 5.1: Add golangci-lint Configuration

**Files:**
- Create: `.golangci.yml`
- Modify: `.github/workflows/go-ci.yml`

- [ ] **Step 1: Create golangci-lint config**

Create `.golangci.yml`:
```yaml
run:
  timeout: 5m
  skip-dirs:
    - vendor

linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused
    - gocyclo
    - goimports
    - gofmt
    - revive

linters-settings:
  gocyclo:
    min-complexity: 15
  
  errcheck:
    check-blank: true

issues:
  exclude-rules:
    # Exclude some linters from test files
    - path: _test\.go
      linters:
        - gocyclo
        - errcheck
```

- [ ] **Step 2: Update CI workflow**

Add lint job to `.github/workflows/go-ci.yml`:
```yaml
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      - name: golangci-lint
        uses: golangci/golangci-lint-action@v3
        with:
          version: latest
          args: --timeout=5m
```

- [ ] **Step 3: Run linter locally**

```bash
golangci-lint run
```

- [ ] **Step 4: Commit**

```bash
git add .golangci.yml
git add .github/workflows/go-ci.yml
git commit -m "chore: add golangci-lint configuration"
```

---

### Task 5.2: Fix Linter Issues

- [ ] **Step 1: Fix errcheck issues**

```bash
golangci-lint run --fix
```

- [ ] **Step 2: Fix goimports**

```bash
goimports -w internal/
```

- [ ] **Step 3: Fix gofmt**

```bash
gofmt -w internal/
```

- [ ] **Step 4: Commit fixes**

```bash
git add .
git commit -m "style: fix linter issues"
```

---

### Task 5.3: Extract Constants

**Files:**
- Create: `internal/model/constants.go`
- Modify: Services using magic strings

- [ ] **Step 1: Create constants file**

Create `internal/model/constants.go`:
```go
package model

// Task priorities
const (
    PriorityLow    = "low"
    PriorityMedium = "medium"
    PriorityHigh   = "high"
    PriorityCritical = "critical"
)

// Task statuses
const (
    StatusTodo       = "todo"
    StatusInProgress = "in_progress"
    StatusDone       = "done"
    StatusBlocked    = "blocked"
    StatusCancelled  = "cancelled"
)

// Default values
const (
    DefaultPriority = PriorityMedium
    DefaultStatus   = StatusTodo
)
```

- [ ] **Step 2: Replace magic strings in services**

```go
// OLD
if priority == "" {
    priority = "medium"
}

// NEW
import "baxi/internal/model"

if priority == "" {
    priority = model.DefaultPriority
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./...
```

- [ ] **Step 4: Commit**

```bash
git add internal/model/constants.go
git add internal/service/
git commit -m "refactor: extract magic strings to constants"
```

---

## Final Verification

### Task 6.1: Final Test Run

- [ ] **Step 1: Run full test suite**

```bash
cd /home/zzz/project/baxi
go test ./... 2>&1 | tee /tmp/final_test_results.txt
```

Expected: All tests pass

- [ ] **Step 2: Run linter**

```bash
golangci-lint run
```

Expected: No errors

- [ ] **Step 3: Verify build**

```bash
go build ./...
```

Expected: Success

- [ ] **Step 4: Check file sizes**

```bash
wc -l internal/api/server.go
# Expected: < 200 lines

wc -l internal/service/feishu_service.go
# Expected: < 300 lines
```

---

## Summary

This plan refactors Baxi to eliminate key architectural anti-patterns:

1. **Repository restructuring**: Flat → Domain subpackages with pool injection
2. **Service decoupling**: api/dto → model package
3. **File splitting**: server.go 691 lines → ~100 lines, feishu_service.go 967 lines → focused modules
4. **Code quality**: golangci-lint, constants extraction

**Total estimated time**: ~5 weeks
**Test guarantee**: Each task includes test steps; plan maintains 100% test pass rate
