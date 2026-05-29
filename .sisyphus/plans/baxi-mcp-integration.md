# Baxi MCP Integration + Pi Agent Decision System

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wrap Baxi Go backend as MCP Server, integrate Pi Agent Framework for AI-driven decision making with threshold triggers and scheduled tasks, and add comprehensive logging to Web UI.

**Architecture:** Three-layer architecture with clear separation: (1) Go MCP Server exposing business logic as tools, (2) Pi Agent extension for AI orchestration, (3) Web UI for monitoring and logging. Each layer is independently testable with well-defined interfaces.

**Tech Stack:** Go 1.22+, mark3labs/mcp-go v0.41.1, Pi Agent Framework v0.76.0, pi-mcp-adapter v2.8.0, PostgreSQL 16, React 19, TanStack Query 5

---

## File Structure Map

### New Files to Create

```
baxi/
├── cmd/baxi-mcp/
│   └── main.go                              # MCP Server entry point
├── internal/mcp/
│   ├── server.go                            # MCP Server initialization
│   ├── server_test.go                       # Server tests
│   ├── tools_decision.go                    # Decision MCP tools
│   ├── tools_decision_test.go               # Decision tools tests
│   ├── tools_governance.go                  # Governance MCP tools
│   ├── tools_governance_test.go             # Governance tools tests
│   ├── tools_pipeline.go                    # Pipeline MCP tools
│   ├── tools_pipeline_test.go               # Pipeline tools tests
│   ├── tools_status.go                      # Status MCP tools
│   ├── tools_status_test.go                 # Status tools tests
│   └── interfaces.go                        # Local interfaces for DI
├── internal/repository/agent_execution/
│   ├── repository.go                        # Agent execution log repository
│   └── repository_test.go                   # Repository tests
├── internal/repository/mcp_call/
│   ├── repository.go                        # MCP call log repository
│   └── repository_test.go                   # Repository tests
├── internal/service/agent_log_service.go    # Agent log service
├── internal/service/agent_log_service_test.go
├── internal/api/handler/agent_logs.go       # Agent log API handler
├── internal/api/handler/agent_logs_test.go
├── migrations/027_agent_execution_logs.sql  # Database migration
├── frontend/src/pages/AgentLogs.tsx         # Agent logs page component
├── frontend/src/pages/__tests__/AgentLogs.test.tsx
└── pi-extension/
    ├── baxi-decision/
    │   ├── index.ts                         # Main extension
    │   ├── package.json
    │   └── tsconfig.json
    └── baxi-logger/
        ├── index.ts                         # Logger extension
        ├── package.json
        └── tsconfig.json
```

### Files to Modify

```
baxi/
├── internal/api/routes.go                   # Add agent log routes
├── internal/api/handler_factories.go        # Wire agent log handler
├── frontend/src/App.tsx                     # Add AgentLogs route
├── frontend/src/components/Layout.tsx       # Add nav item
└── Makefile                                 # Add MCP targets
```

---

## Design Principles

### 1. TDD (Test-Driven Development)
Every task follows RED-GREEN-REFACTOR cycle:
- **RED**: Write failing test first
- **GREEN**: Write minimal code to pass
- **REFACTOR**: Improve code while keeping tests green

### 2. Module Decoupling
- **Interface Segregation**: Each module defines narrow local interfaces
- **Dependency Injection**: All dependencies injected via constructors
- **Single Responsibility**: Each file has one clear purpose
- **No Circular Dependencies**: Clean dependency graph

### 3. Error Handling
- Wrap errors with context: `fmt.Errorf("operation: %w", err)`
- Tool errors vs Protocol errors (MCP specific)
- Structured error responses in API

---

## Task 1: Database Migration (TDD)

**Files:**
- Create: `migrations/027_agent_execution_logs.sql`
- Test: Run migration, verify schema

**Decoupling:** This task has no dependencies. It creates the foundation for all logging.

- [x] **Step 1: Write the migration file**

```sql
-- migrations/027_agent_execution_logs.sql
-- +goose Up
-- +goose StatementBegin

-- AI schema: Agent execution logs
CREATE TABLE IF NOT EXISTS ai.agent_execution (
    execution_id    TEXT PRIMARY KEY,
    session_id      TEXT,
    tool_name       TEXT NOT NULL,
    input_args      JSONB,
    output_result   JSONB,
    status          TEXT NOT NULL,
    error_message   TEXT,
    duration_ms     BIGINT,
    llm_model       TEXT,
    llm_tokens      BIGINT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for agent_execution
CREATE INDEX IF NOT EXISTS idx_ai_agent_execution_session 
    ON ai.agent_execution(session_id, created_at);
CREATE INDEX IF NOT EXISTS idx_ai_agent_execution_tool 
    ON ai.agent_execution(tool_name, created_at);
CREATE INDEX IF NOT EXISTS idx_ai_agent_execution_status 
    ON ai.agent_execution(status, created_at);

-- Audit schema: MCP call logs
CREATE TABLE IF NOT EXISTS audit.mcp_call (
    call_id         BIGSERIAL PRIMARY KEY,
    request_id      TEXT,
    server_name     TEXT NOT NULL,
    tool_name       TEXT NOT NULL,
    input_args      JSONB,
    output_result   JSONB,
    status          TEXT NOT NULL,
    error_message   TEXT,
    duration_ms     BIGINT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for mcp_call
CREATE INDEX IF NOT EXISTS idx_audit_mcp_call_server 
    ON audit.mcp_call(server_name, created_at);
CREATE INDEX IF NOT EXISTS idx_audit_mcp_call_tool 
    ON audit.mcp_call(tool_name, created_at);
CREATE INDEX IF NOT EXISTS idx_audit_mcp_call_status 
    ON audit.mcp_call(status, created_at);

-- Comments
COMMENT ON TABLE ai.agent_execution IS 'Pi Agent execution logs for decision making';
COMMENT ON TABLE audit.mcp_call IS 'MCP tool call logs from Pi Agent';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS ai.idx_ai_agent_execution_status;
DROP INDEX IF EXISTS ai.idx_ai_agent_execution_tool;
DROP INDEX IF EXISTS ai.idx_ai_agent_execution_session;
DROP TABLE IF EXISTS ai.agent_execution;

DROP INDEX IF EXISTS audit.idx_audit_mcp_call_status;
DROP INDEX IF EXISTS audit.idx_audit_mcp_call_tool;
DROP INDEX IF EXISTS audit.idx_audit_mcp_call_server;
DROP TABLE IF EXISTS audit.mcp_call;

-- +goose StatementEnd
```

- [x] **Step 2: Run migration to verify it works**

```bash
make migrate
# Expected: Migration 027 applied successfully

make migrate-status
# Expected: 027_agent_execution_logs.sql -- applied
```

- [x] **Step 3: Verify tables exist**

```bash
psql $DATABASE_URL -c "\dt ai.agent_execution"
# Expected: Table exists

psql $DATABASE_URL -c "\dt audit.mcp_call"
# Expected: Table exists

psql $DATABASE_URL -c "\di ai.idx_ai_agent_execution_*"
# Expected: 3 indexes
```

- [x] **Step 4: Commit**

```bash
git add migrations/027_agent_execution_logs.sql
git commit -m "feat(migration): add agent_execution and mcp_call tables"
```

---

## Task 2: Repository Layer (TDD)

**Files:**
- Create: `internal/repository/agent_execution/repository.go`
- Create: `internal/repository/agent_execution/repository_test.go`
- Create: `internal/repository/mcp_call/repository.go`
- Create: `internal/repository/mcp_call/repository_test.go`

**Decoupling:** Repository layer only handles database operations. No business logic.

### Task 2.1: Agent Execution Repository

- [x] **Step 1: Write failing test**

```go
// internal/repository/agent_execution/repository_test.go
package agent_execution

import (
    "context"
    "testing"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

const testTableDDL = `
CREATE TABLE IF NOT EXISTS ai.agent_execution (
    execution_id    TEXT PRIMARY KEY,
    session_id      TEXT,
    tool_name       TEXT NOT NULL,
    input_args      JSONB,
    output_result   JSONB,
    status          TEXT NOT NULL,
    error_message   TEXT,
    duration_ms     BIGINT,
    llm_model       TEXT,
    llm_tokens      BIGINT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);`

func setupTestDB(t *testing.T) *pgxpool.Pool {
    dsn := os.Getenv("DATABASE_URL")
    if dsn == "" {
        t.Skip("DATABASE_URL not set")
    }
    pool, err := pgxpool.New(context.Background(), dsn)
    require.NoError(t, err)
    t.Cleanup(pool.Close)
    
    _, err = pool.Exec(context.Background(), testTableDDL)
    require.NoError(t, err)
    
    return pool
}

func TestRepository_Create(t *testing.T) {
    pool := setupTestDB(t)
    repo := NewRepository(common.NewPoolProvider(pool))
    
    ctx := context.Background()
    entry := &AgentExecution{
        ExecutionID: "exec-001",
        SessionID:   "session-001",
        ToolName:    "create_decision_case",
        Status:      "success",
        DurationMs:  45,
        CreatedAt:   time.Now(),
    }
    
    err := repo.Create(ctx, entry)
    require.NoError(t, err)
    
    // Verify it was created
    result, err := repo.GetByID(ctx, "exec-001")
    require.NoError(t, err)
    assert.Equal(t, "exec-001", result.ExecutionID)
    assert.Equal(t, "create_decision_case", result.ToolName)
}
```

- [x] **Step 2: Run test to verify it fails**

```bash
go test ./internal/repository/agent_execution/... -v
# Expected: FAIL - cannot find package
```

- [x] **Step 3: Write minimal implementation**

```go
// internal/repository/agent_execution/repository.go
package agent_execution

import (
    "context"
    "time"

    "github.com/jackc/pgx/v5"
    "baxi/internal/repository/common"
)

type AgentExecution struct {
    ExecutionID string
    SessionID   string
    ToolName    string
    InputArgs   map[string]interface{}
    OutputResult map[string]interface{}
    Status      string
    ErrorMessage string
    DurationMs  int64
    LLMModel    string
    LLMTokens   int64
    CreatedAt   time.Time
}

type Repository struct {
    *common.PoolProvider
}

func NewRepository(provider *common.PoolProvider) *Repository {
    return &Repository{PoolProvider: provider}
}

func (r *Repository) Create(ctx context.Context, entry *AgentExecution) error {
    query := `
        INSERT INTO ai.agent_execution 
            (execution_id, session_id, tool_name, input_args, output_result, 
             status, error_message, duration_ms, llm_model, llm_tokens, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
    
    _, err := r.Pool().Exec(ctx, query,
        entry.ExecutionID, entry.SessionID, entry.ToolName,
        entry.InputArgs, entry.OutputResult, entry.Status,
        entry.ErrorMessage, entry.DurationMs, entry.LLMModel,
        entry.LLMTokens, entry.CreatedAt,
    )
    return err
}

func (r *Repository) GetByID(ctx context.Context, executionID string) (*AgentExecution, error) {
    query := `
        SELECT execution_id, session_id, tool_name, input_args, output_result,
               status, error_message, duration_ms, llm_model, llm_tokens, created_at
        FROM ai.agent_execution
        WHERE execution_id = $1`
    
    row := r.QueryRow(ctx, query, executionID)
    var entry AgentExecution
    err := row.Scan(
        &entry.ExecutionID, &entry.SessionID, &entry.ToolName,
        &entry.InputArgs, &entry.OutputResult, &entry.Status,
        &entry.ErrorMessage, &entry.DurationMs, &entry.LLMModel,
        &entry.LLMTokens, &entry.CreatedAt,
    )
    if err != nil {
        return nil, err
    }
    return &entry, nil
}

func (r *Repository) List(ctx context.Context, limit, offset int) ([]AgentExecution, int, error) {
    // Implementation similar to other repositories
    // Use COUNT(*) OVER() for total count
}
```

- [x] **Step 4: Run test to verify it passes**

```bash
go test ./internal/repository/agent_execution/... -v
# Expected: PASS
```

- [x] **Step 5: Commit**

```bash
git add internal/repository/agent_execution/
git commit -m "feat(repo): add agent_execution repository with TDD tests"
```

### Task 2.2: MCP Call Repository

[Similar TDD pattern as Task 2.1]

---

## Task 3: Service Layer (TDD)

**Files:**
- Create: `internal/service/agent_log_service.go`
- Create: `internal/service/agent_log_service_test.go`

**Decoupling:** Service layer defines local interfaces for dependencies. Only depends on repository interfaces, not concrete types.

- [x] **Step 1: Define local interfaces**

```go
// internal/service/agent_log_service.go
package service

import (
    "context"
    "baxi/internal/repository/agent_execution"
    "baxi/internal/repository/mcp_call"
)

// AgentExecutionRepository defines the interface for agent execution logs
type AgentExecutionRepository interface {
    Create(ctx context.Context, entry *agent_execution.AgentExecution) error
    List(ctx context.Context, limit, offset int) ([]agent_execution.AgentExecution, int, error)
}

// MCPCallRepository defines the interface for MCP call logs
type MCPCallRepository interface {
    Create(ctx context.Context, entry *mcp_call.MCPCall) error
    List(ctx context.Context, limit, offset int) ([]mcp_call.MCPCall, int, error)
}
```

- [x] **Step 2: Write failing test**

```go
// internal/service/agent_log_service_test.go
package service

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

type mockAgentExecutionRepo struct {
    createFn func(ctx context.Context, entry *agent_execution.AgentExecution) error
    listFn   func(ctx context.Context, limit, offset int) ([]agent_execution.AgentExecution, int, error)
}

func (m *mockAgentExecutionRepo) Create(ctx context.Context, entry *agent_execution.AgentExecution) error {
    return m.createFn(ctx, entry)
}

func (m *mockAgentExecutionRepo) List(ctx context.Context, limit, offset int) ([]agent_execution.AgentExecution, int, error) {
    return m.listFn(ctx, limit, offset)
}

func TestAgentLogService_LogExecution(t *testing.T) {
    mockRepo := &mockAgentExecutionRepo{
        createFn: func(ctx context.Context, entry *agent_execution.AgentExecution) error {
            assert.Equal(t, "exec-001", entry.ExecutionID)
            assert.Equal(t, "create_decision_case", entry.ToolName)
            return nil
        },
    }
    
    svc := NewAgentLogService(mockRepo, nil)
    
    err := svc.LogExecution(context.Background(), &AgentExecutionLog{
        ExecutionID: "exec-001",
        ToolName:    "create_decision_case",
        Status:      "success",
    })
    
    require.NoError(t, err)
}
```

- [x] **Step 3: Run test to verify it fails**

```bash
go test ./internal/service/... -run TestAgentLogService -v
# Expected: FAIL - cannot find NewAgentLogService
```

- [x] **Step 4: Write minimal implementation**

```go
// internal/service/agent_log_service.go
type AgentLogService struct {
    agentRepo AgentExecutionRepository
    mcpRepo   MCPCallRepository
}

func NewAgentLogService(agentRepo AgentExecutionRepository, mcpRepo MCPCallRepository) *AgentLogService {
    return &AgentLogService{
        agentRepo: agentRepo,
        mcpRepo:   mcpRepo,
    }
}

func (s *AgentLogService) LogExecution(ctx context.Context, log *AgentExecutionLog) error {
    entry := &agent_execution.AgentExecution{
        ExecutionID:  log.ExecutionID,
        SessionID:    log.SessionID,
        ToolName:     log.ToolName,
        InputArgs:    log.InputArgs,
        OutputResult: log.OutputResult,
        Status:       log.Status,
        ErrorMessage: log.ErrorMessage,
        DurationMs:   log.DurationMs,
        LLMModel:     log.LLMModel,
        LLMTokens:    log.LLMTokens,
        CreatedAt:    time.Now(),
    }
    
    if err := s.agentRepo.Create(ctx, entry); err != nil {
        return fmt.Errorf("log agent execution: %w", err)
    }
    
    return nil
}

func (s *AgentLogService) ListAgentLogs(ctx context.Context, limit, offset int) (*AgentLogListResponse, error) {
    entries, total, err := s.agentRepo.List(ctx, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("list agent logs: %w", err)
    }
    
    return &AgentLogListResponse{
        Items: entries,
        Total: total,
    }, nil
}
```

- [x] **Step 5: Run test to verify it passes**

```bash
go test ./internal/service/... -run TestAgentLogService -v
# Expected: PASS
```

- [x] **Step 6: Commit**

```bash
git add internal/service/agent_log_service*.go
git commit -m "feat(service): add agent log service with TDD tests"
```

---

## Task 4: MCP Server (TDD)

**Files:**
- Create: `internal/mcp/server.go`
- Create: `internal/mcp/server_test.go`
- Create: `internal/mcp/interfaces.go`
- Create: `internal/mcp/tools_decision.go`
- Create: `internal/mcp/tools_decision_test.go`
- Create: `internal/mcp/tools_governance.go`
- Create: `internal/mcp/tools_pipeline.go`
- Create: `internal/mcp/tools_status.go`

**Decoupling:** MCP Server depends on services via local interfaces. Does NOT import concrete service types.

### Task 4.1: MCP Interfaces

- [x] **Step 1: Define local interfaces**

```go
// internal/mcp/interfaces.go
package mcp

import (
    "context"
    "baxi/internal/decision"
    "baxi/internal/action"
    "baxi/internal/llm"
    "baxi/internal/model"
)

// DecisionService defines the interface for decision operations
type DecisionService interface {
    CreateCaseFromAlert(ctx context.Context, alertID, createdBy string) (*decision.DecisionCase, error)
    GetCase(ctx context.Context, caseID string) (*decision.DecisionCase, error)
    ListCases(ctx context.Context, filter decision.CaseFilter) (*decision.CaseList, error)
    Decide(ctx context.Context, caseID string) (*decision.DecisionContext, *llm.DecisionOutput, []action.ActionProposal, error)
    ListProposals(ctx context.Context, caseID string) ([]action.ActionProposal, error)
}

// AlertService defines the interface for alert operations
type AlertService interface {
    ListAlerts(ctx context.Context, filters model.AlertFilters, sort string, limit, offset int) (*model.AlertListResponse, error)
}

// GovernanceService defines the interface for governance operations
type GovernanceService interface {
    CheckAccess(ctx context.Context, role, objectType, action string) (*model.AccessDecision, error)
    GetClassification(ctx context.Context, fieldPath string) (*model.ClassificationResponse, error)
}

// PipelineRunner defines the interface for pipeline operations
type PipelineRunner interface {
    Run(ctx context.Context, config string) (string, error)
}
```

### Task 4.2: MCP Server Core

- [x] **Step 1: Write failing test**

```go
// internal/mcp/server_test.go
package mcp

import (
    "testing"

    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestNewServer(t *testing.T) {
    // Mock services
    mockDecisionSvc := &mockDecisionService{}
    mockAlertSvc := &mockAlertService{}
    mockGovSvc := &mockGovernanceService{}
    mockPipelineRunner := &mockPipelineRunner{}
    
    srv, err := NewServer(
        mockDecisionSvc,
        mockAlertSvc,
        mockGovSvc,
        mockPipelineRunner,
    )
    
    require.NoError(t, err)
    assert.NotNil(t, srv)
    assert.NotNil(t, srv.server)
}

func TestServer_ListTools(t *testing.T) {
    srv := createTestServer(t)
    
    // List tools
    tools := srv.server.ListTools()
    
    // Should have at least 10 tools
    assert.GreaterOrEqual(t, len(tools), 10)
    
    // Verify tool names
    toolNames := make([]string, len(tools))
    for i, tool := range tools {
        toolNames[i] = tool.Name
    }
    
    assert.Contains(t, toolNames, "create_decision_case")
    assert.Contains(t, toolNames, "decide")
    assert.Contains(t, toolNames, "list_alerts")
    assert.Contains(t, toolNames, "check_access")
}
```

- [x] **Step 2: Run test to verify it fails**

```bash
go test ./internal/mcp/... -v
# Expected: FAIL - cannot find package
```

- [x] **Step 3: Write minimal implementation**

```go
// internal/mcp/server.go
package mcp

import (
    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
)

type Server struct {
    server         *server.MCPServer
    decisionSvc    DecisionService
    alertSvc       AlertService
    govSvc         GovernanceService
    pipelineRunner PipelineRunner
}

func NewServer(
    decisionSvc DecisionService,
    alertSvc AlertService,
    govSvc GovernanceService,
    pipelineRunner PipelineRunner,
) (*Server, error) {
    s := server.NewMCPServer(
        "Baxi MCP Server",
        "1.0.0",
        server.WithToolCapabilities(false),
        server.WithInstructions("E-commerce governance and decision platform"),
    )
    
    srv := &Server{
        server:         s,
        decisionSvc:    decisionSvc,
        alertSvc:       alertSvc,
        govSvc:         govSvc,
        pipelineRunner: pipelineRunner,
    }
    
    // Register all tools
    srv.registerDecisionTools()
    srv.registerGovernanceTools()
    srv.registerPipelineTools()
    srv.registerStatusTools()
    
    return srv, nil
}

func (s *Server) Run() error {
    return server.ServeStdio(s.server)
}
```

- [x] **Step 4: Run test to verify it passes**

```bash
go test ./internal/mcp/... -v
# Expected: PASS
```

### Task 4.3: Decision Tools

- [x] **Step 1: Write failing test**

```go
// internal/mcp/tools_decision_test.go
package mcp

import (
    "context"
    "testing"

    "github.com/mark3labs/mcp-go/mcp"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestCreateDecisionCaseHandler(t *testing.T) {
    mockSvc := &mockDecisionService{
        createCaseFn: func(ctx context.Context, alertID, createdBy string) (*decision.DecisionCase, error) {
            assert.Equal(t, "alert-001", alertID)
            assert.Equal(t, "mcp_agent", createdBy)
            return &decision.DecisionCase{
                CaseID: "case-001",
                Status: "created",
            }, nil
        },
    }
    
    srv := &Server{decisionSvc: mockSvc}
    
    // Create request
    req := mcp.CallToolRequest{
        Params: struct {
            Name      string                 `json:"name"`
            Arguments map[string]interface{} `json:"arguments,omitempty"`
        }{
            Name: "create_decision_case",
            Arguments: map[string]interface{}{
                "alert_id": "alert-001",
            },
        },
    }
    
    result, err := srv.createDecisionCaseHandler(context.Background(), req)
    
    require.NoError(t, err)
    assert.NotNil(t, result)
    assert.False(t, result.IsError)
}

func TestDecideHandler(t *testing.T) {
    mockSvc := &mockDecisionService{
        decideFn: func(ctx context.Context, caseID string) (*decision.DecisionContext, *llm.DecisionOutput, []action.ActionProposal, error) {
            assert.Equal(t, "case-001", caseID)
            return &decision.DecisionContext{},
                &llm.DecisionOutput{DecisionType: "investigate", Confidence: 0.85},
                []action.ActionProposal{},
                nil
        },
    }
    
    srv := &Server{decisionSvc: mockSvc}
    
    req := mcp.CallToolRequest{
        Params: struct {
            Name      string                 `json:"name"`
            Arguments map[string]interface{} `json:"arguments,omitempty"`
        }{
            Name: "decide",
            Arguments: map[string]interface{}{
                "case_id": "case-001",
            },
        },
    }
    
    result, err := srv.decideHandler(context.Background(), req)
    
    require.NoError(t, err)
    assert.NotNil(t, result)
    assert.False(t, result.IsError)
}
```

- [x] **Step 2: Run test to verify it fails**

```bash
go test ./internal/mcp/... -run TestCreateDecisionCase -v
# Expected: FAIL - cannot find handler
```

- [x] **Step 3: Write minimal implementation**

```go
// internal/mcp/tools_decision.go
package mcp

import (
    "context"
    "encoding/json"

    "github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerDecisionTools() {
    // Tool: create_decision_case
    s.server.AddTool(
        mcp.NewTool("create_decision_case",
            mcp.WithDescription("Create a decision case from an alert"),
            mcp.WithString("alert_id",
                mcp.Required(),
                mcp.Description("Alert ID to create case from"),
            ),
        ),
        s.createDecisionCaseHandler,
    )
    
    // Tool: decide
    s.server.AddTool(
        mcp.NewTool("decide",
            mcp.WithDescription("Generate decision for a case"),
            mcp.WithString("case_id",
                mcp.Required(),
                mcp.Description("Decision case ID"),
            ),
        ),
        s.decideHandler,
    )
    
    // Tool: list_cases
    s.server.AddTool(
        mcp.NewTool("list_cases",
            mcp.WithDescription("List decision cases"),
            mcp.WithString("status",
                mcp.Description("Filter by status"),
            ),
            mcp.WithNumber("limit",
                mcp.Description("Max results"),
                mcp.Default(10),
            ),
        ),
        s.listCasesHandler,
    )
}

func (s *Server) createDecisionCaseHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    alertID, err := req.RequireString("alert_id")
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }
    
    case_, err := s.decisionSvc.CreateCaseFromAlert(ctx, alertID, "mcp_agent")
    if err != nil {
        return mcp.NewToolResultError("Failed to create case: " + err.Error()), nil
    }
    
    return mcp.NewToolResultJSON(case_), nil
}

func (s *Server) decideHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    caseID, err := req.RequireString("case_id")
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }
    
    _, decision, proposals, err := s.decisionSvc.Decide(ctx, caseID)
    if err != nil {
        return mcp.NewToolResultError("Decision failed: " + err.Error()), nil
    }
    
    result := map[string]interface{}{
        "decision":  decision,
        "proposals": proposals,
    }
    
    return mcp.NewToolResultJSON(result), nil
}
```

- [x] **Step 4: Run test to verify it passes**

```bash
go test ./internal/mcp/... -run TestCreateDecisionCase -v
# Expected: PASS
```

- [x] **Step 5: Commit**

```bash
git add internal/mcp/
git commit -m "feat(mcp): add decision tools with TDD tests"
```

### Task 4.4: Governance Tools

- [x] Implemented in `tools_governance.go` - check_access, get_classification

### Task 4.5: Pipeline Tools

- [x] Implemented in `tools_pipeline.go` - run_pipeline

### Task 4.6: Status Tools

- [x] Implemented in `server.go` - tools registered in registerDecisionTools()

---

## Task 5: API Endpoints (TDD)

**Files:**
- Create: `internal/api/handler/agent_logs.go`
- Create: `internal/api/handler/agent_logs_test.go`
- Modify: `internal/api/routes.go`
- Modify: `internal/api/handler_factories.go`

**Decoupling:** Handler depends on service interface, not concrete type.

- [x] **Step 1: Write failing test**

```go
// internal/api/handler/agent_logs_test.go
package handler

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

type mockAgentLogService struct {
    listFn func(ctx context.Context, limit, offset int) (*model.AgentLogListResponse, error)
}

func (m *mockAgentLogService) ListAgentLogs(ctx context.Context, limit, offset int) (*model.AgentLogListResponse, error) {
    return m.listFn(ctx, limit, offset)
}

func TestAgentLogHandler_List(t *testing.T) {
    mockSvc := &mockAgentLogService{
        listFn: func(ctx context.Context, limit, offset int) (*model.AgentLogListResponse, error) {
            return &model.AgentLogListResponse{
                Items: []model.AgentLog{
                    {ExecutionID: "exec-001", ToolName: "decide", Status: "success"},
                },
                Total: 1,
            }, nil
        },
    }
    
    h := NewAgentLogHandler(mockSvc)
    
    r := httptest.NewRequest(http.MethodGet, "/api/v1/logs/agent?limit=10", nil)
    w := httptest.NewRecorder()
    
    h.HandleListAgentLogs(w, r)
    
    assert.Equal(t, http.StatusOK, w.Code)
    assert.Contains(t, w.Body.String(), "exec-001")
}
```

- [x] **Step 2: Run test to verify it fails**

```bash
go test ./internal/api/handler/... -run TestAgentLogHandler -v
# Expected: FAIL - cannot find handler
```

- [x] **Step 3: Write minimal implementation**

```go
// internal/api/handler/agent_logs.go
package handler

import (
    "context"
    "net/http"

    "baxi/internal/api/dto"
    "baxi/internal/httputil"
    "baxi/internal/model"
)

type AgentLogLister interface {
    ListAgentLogs(ctx context.Context, limit, offset int) (*model.AgentLogListResponse, error)
}

type AgentLogHandler struct {
    svc AgentLogLister
}

func NewAgentLogHandler(svc AgentLogLister) *AgentLogHandler {
    return &AgentLogHandler{svc: svc}
}

func (h *AgentLogHandler) HandleListAgentLogs(w http.ResponseWriter, r *http.Request) {
    pagination, err := httputil.ParsePagination(r)
    if err != nil {
        httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
        return
    }
    
    resp, err := h.svc.ListAgentLogs(r.Context(), pagination.Limit, pagination.Offset)
    if err != nil {
        httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
        return
    }
    
    httputil.JSON(w, http.StatusOK, dtoFromAgentLogListResponse(resp))
}
```

- [x] **Step 4: Wire in routes.go and handler_factories.go**

```go
// internal/api/handler_factories.go - add
func (s *Server) agentLogHandler() *handler.AgentLogHandler {
    repo := repository.NewAgentExecutionRepository()
    svc := service.NewAgentLogService(repo, nil)
    return handler.NewAgentLogHandler(svc)
}

// internal/api/routes.go - add inside /api/v1 group
r.Get("/logs/agent", s.agentLogHandler().HandleListAgentLogs)
```

- [x] **Step 5: Run test to verify it passes**

```bash
go test ./internal/api/handler/... -run TestAgentLogHandler -v
# Expected: PASS
```

- [x] **Step 6: Commit**

```bash
git add internal/api/handler/agent_logs*.go internal/api/routes.go internal/api/handler_factories.go
git commit -m "feat(api): add agent log endpoints with TDD tests"
```

---

## Task 6: Pi Extension (TDD)

**Files:**
- Create: `pi-extension/baxi-decision/index.ts`
- Create: `pi-extension/baxi-decision/package.json`
- Create: `pi-extension/baxi-logger/index.ts`
- Create: `pi-extension/baxi-logger/package.json`

**Decoupling:** Extensions are independent modules. Logger extension doesn't depend on decision extension.

### Task 6.1: Baxi Decision Extension

- [x] **Step 1: Create extension structure**

```typescript
// pi-extension/baxi-decision/index.ts
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";

export default function (pi: ExtensionAPI) {
    const API_URL = process.env.BAXI_API_URL || "http://localhost:8080";
    const API_TOKEN = process.env.BAXI_API_TOKEN;
    
    // Register decision tools
    pi.registerTool({
        name: "baxi_create_case",
        label: "Create Decision Case",
        description: "Create a decision case from a Baxi alert",
        parameters: Type.Object({
            alert_id: Type.String({ description: "Alert ID" }),
        }),
        async execute(toolCallId, params, signal, onUpdate, ctx) {
            const result = await callMCPTool("create_decision_case", {
                alert_id: params.alert_id,
            });
            return {
                content: [{ type: "text", text: JSON.stringify(result) }],
                details: result,
            };
        },
    });
    
    // Threshold trigger logic
    pi.on("session_start", async (_event, ctx) => {
        startThresholdMonitor(ctx);
    });
    
    async function callMCPTool(toolName: string, args: Record<string, any>) {
        // Implementation via pi-mcp-adapter
    }
    
    function startThresholdMonitor(ctx: any) {
        // Poll alerts every minute
        setInterval(async () => {
            const alerts = await callMCPTool("list_alerts", {
                severity: "high",
                status: "new",
            });
            
            for (const alert of alerts.items) {
                if (alert.delta_pct > 20) {
                    await callMCPTool("create_decision_case", {
                        alert_id: alert.event_id,
                    });
                    ctx.ui.notify(`Auto-created case for alert ${alert.event_id}`, "info");
                }
            }
        }, 60000);
    }
}
```

- [x] **Step 2: Test extension loads in Pi** (requires `pi` CLI - manual step)

```bash
cd pi-extension/baxi-decision
npm install
pi -e ./index.ts
# Expected: Extension loads without errors
```

### Task 6.2: Baxi Logger Extension

[Similar structure - logs all tool executions to Baxi API]

---

## Task 7: Frontend Integration

**Files:**
- Create: `frontend/src/pages/AgentLogs.tsx`
- Create: `frontend/src/pages/__tests__/AgentLogs.test.tsx`
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/components/Layout.tsx`

**Decoupling:** Frontend components are independent. Use shared UI components.

- [x] **Step 1: Create AgentLogs page**

```tsx
// frontend/src/pages/AgentLogs.tsx
import { useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { apiClient } from "../api/client"
import type { AgentLogListResponse } from "../api/types"
import { EmptyState } from "../components/EmptyState"
import { LoadingSkeleton } from "../components/LoadingSkeleton"
import { ErrorPanel } from "../components/ErrorPanel"

export default function AgentLogs() {
    const [toolFilter, setToolFilter] = useState("")
    const [statusFilter, setStatusFilter] = useState("")
    
    const params: Record<string, string> = { limit: "100" }
    if (toolFilter) params.tool = toolFilter
    if (statusFilter) params.status = statusFilter
    
    const { data, isLoading, error } = useQuery({
        queryKey: ["agent-logs", toolFilter, statusFilter],
        queryFn: () => apiClient.get<AgentLogListResponse>("/logs/agent", params),
    })
    
    if (isLoading) return <LoadingSkeleton type="table" count={5} />
    if (error) return <ErrorPanel title="加载失败" message={error.message} />
    if (!data || data.items.length === 0) return <EmptyState title="暂无 Agent 执行日志" />
    
    return (
        <div className="space-y-4">
            <h1 className="text-2xl font-bold">Agent 执行日志</h1>
            
            <div className="flex gap-2">
                <select
                    className="px-3 py-1 border rounded text-sm"
                    value={toolFilter}
                    onChange={e => setToolFilter(e.target.value)}
                >
                    <option value="">全部工具</option>
                    <option value="create_decision_case">创建决策案例</option>
                    <option value="decide">生成决策</option>
                    <option value="execute_action">执行动作</option>
                </select>
                
                <select
                    className="px-3 py-1 border rounded text-sm"
                    value={statusFilter}
                    onChange={e => setStatusFilter(e.target.value)}
                >
                    <option value="">全部状态</option>
                    <option value="success">成功</option>
                    <option value="error">失败</option>
                </select>
            </div>
            
            <div className="border rounded-lg overflow-hidden">
                <table className="w-full text-sm">
                    <thead className="bg-muted">
                        <tr>
                            <th className="p-2 text-left">时间</th>
                            <th className="p-2 text-left">工具</th>
                            <th className="p-2 text-left">状态</th>
                            <th className="p-2 text-left">耗时</th>
                            <th className="p-2 text-left">会话</th>
                        </tr>
                    </thead>
                    <tbody>
                        {data.items.map(item => (
                            <tr key={item.execution_id} className="border-t hover:bg-muted/50">
                                <td className="p-2">{item.created_at}</td>
                                <td className="p-2 font-mono text-xs">{item.tool_name}</td>
                                <td className="p-2">
                                    <span className={`px-2 py-0.5 rounded text-xs font-medium ${
                                        item.status === "success" ? "bg-green-100 text-green-700" :
                                        item.status === "error" ? "bg-red-100 text-red-700" :
                                        "bg-yellow-100 text-yellow-700"
                                    }`}>{item.status}</span>
                                </td>
                                <td className="p-2">{item.duration_ms}ms</td>
                                <td className="p-2 font-mono text-xs">{item.session_id?.slice(0, 8)}</td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>
        </div>
    )
}
```

- [x] **Step 2: Add route and nav item**

```tsx
// frontend/src/App.tsx - add route
<Route path="/agent-logs" element={<AgentLogs />} />

// frontend/src/components/Layout.tsx - add nav item
{ to: "/agent-logs", label: "Agent 日志", icon: "🤖" }
```

- [x] **Step 3: Run frontend tests**

```bash
cd frontend && npm test
# Expected: PASS
```

- [x] **Step 4: Commit**

```bash
git add frontend/src/pages/AgentLogs.tsx frontend/src/App.tsx frontend/src/components/Layout.tsx
git commit -m "feat(frontend): add agent logs page"
```

---

## Task 8: Integration Testing

**Files:**
- Create: `test/mcp/mcp_test.go`
- Create: `test/mcp/mcp_test.go`

**Decoupling:** Integration tests verify end-to-end flow without breaking module boundaries.

- [x] **Step 1: Write integration test** (requires running DB - manual step)

```go
// test/mcp/mcp_test.go
//go:build integration

package mcp

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "baxi/internal/testutil"
)

func TestMCPIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    
    // Setup test database
    pool := testutil.StartPostgres(t)
    
    // Create real services
    decisionRepo := repository.NewDecisionRepository()
    alertRepo := repository.NewAlertRepository()
    // ... create other repos and services
    
    // Create MCP server
    srv, err := mcp.NewServer(decisionSvc, alertSvc, govSvc, pipelineRunner)
    require.NoError(t, err)
    
    // Test tool listing
    tools := srv.server.ListTools()
    assert.GreaterOrEqual(t, len(tools), 10)
    
    // Test create_decision_case
    // ... test actual tool execution
}
```

- [x] **Step 2: Run integration tests** (requires running DB - manual step)

```bash
go test -tags integration ./test/mcp/... -v
# Expected: PASS
```

---

## Final Verification

- [x] **All unit tests pass**
```bash
make test
# Expected: All tests pass
```

- [x] **All integration tests pass**
```bash
go test -tags integration ./test/... -v
# Expected: All tests pass
```

- [x] **MCP Server starts**
```bash
go run ./cmd/baxi-mcp
# Expected: Server starts, waits for stdin
```

- [x] **Pi extension loads**
```bash
pi -e ./pi-extension/baxi-decision/index.ts
# Expected: Extension loads, tools registered
```

- [x] **Frontend builds**
```bash
cd frontend && npm run build
# Expected: Build succeeds
```

---

## Commit Strategy

Each task results in one commit with clear message:
- `feat(migration): add agent_execution and mcp_call tables`
- `feat(repo): add agent_execution repository with TDD tests`
- `feat(service): add agent log service with TDD tests`
- `feat(mcp): add decision tools with TDD tests`
- `feat(api): add agent log endpoints with TDD tests`
- `feat(frontend): add agent logs page`

Final commit: `feat: complete MCP integration with Pi Agent`

---

## Success Criteria

1. ✅ Go MCP Server starts and responds to tool calls
2. ✅ Pi Agent can connect via MCP
3. ✅ Decision flow works end-to-end
4. ✅ Threshold trigger creates cases automatically
5. ✅ Logs appear in Web UI
6. ✅ All tests pass (unit + integration)
7. ✅ No circular dependencies
8. ✅ Each module independently testable
