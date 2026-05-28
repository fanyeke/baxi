# Go/PostgreSQL 完整迁移 + Python 清理

## TL;DR

> **Quick Summary**: 完成 Go/PostgreSQL 迁移（服务层 + 适配器 + 前端切换 + 数据迁移），然后彻底删除所有 Python/SQLite 代码。
> 
> **Deliverables**:
> - Go 服务层完整覆盖 Python 12 个 service 模块
> - Go 适配器完整覆盖 Python 6 个 adapter
> - 前端切换到 Go API (:8080)
> - 数据从 SQLite 迁移到 PostgreSQL（重跑 pipeline）
> - 删除 api/, services/, adapters/, sql/, scripts/, tests/（Python 代码）
> - 更新 CI/CD 和文档
> 
> **Estimated Effort**: XL（17 个任务，5 个 wave）
> **Parallel Execution**: YES - 5 waves
> **Critical Path**: Task 1 → Task 3-7 → Task 8-11 → Task 12-13 → Task 14-17

---

## Context

### Original Request
用户要求一次性完成 Go/PostgreSQL 迁移并清理迁移前的 Python 代码。

### Interview Summary
**Key Discussions**:
- 数据迁移: 重跑 Go pipeline 从原始 CSV（最干净）
- 功能对等: 核心功能优先，前端实际用到的路由先覆盖
- 清理策略: 彻底删除 Python 代码
- 飞书迁移: 完整迁移到 Go
- 测试策略: 严格，每个迁移模块都要有对应测试，覆盖率目标 70%+
- 回滚方案: 无回滚，直接切

**Research Findings**:
- 项目当前约 58% 已迁移到 Go/PostgreSQL
- Go 核心基础设施（pipeline/decision/governance/action/LLM/repository/worker）约 90% 完成
- Go API 已有 feishu.go 和 pipeline.go handler，差距比预想小
- 主要差距: Python 12 service 模块 vs Go 8 模块（缺 feishu/diagnosis/pipeline/dispatch）
- Python 适配器 6 个 vs Go 2 个 domain type（缺 CLI/Manual，Feishu/GitHub 需验证完整性）
- 现有 Go 代码有编译错误需要修复

### Metis Review
**Identified Gaps** (addressed):
- 需要验证 Go pipeline 能从 CSV 复现 SQLite 状态 → 加入 Task 2
- 需要明确 Feature Parity List → 在 plan 中列出详细映射
- 需要 "Must NOT Have" 防止范围蔓延 → 已加入
- 编译错误必须先修复 → Task 1 作为前置
- 删除顺序需要明确定义 → Task 14 中定义

---

## Work Objectives

### Core Objective
完成 Go/PostgreSQL 迁移，删除所有 Python/SQLite 代码，使项目成为纯 Go/PostgreSQL/React 架构。

### Concrete Deliverables
- Go 服务层: feishu_service.go, diagnosis_service.go, pipeline_service.go, dispatch 扩展, log_service 扩展
- Go 适配器: feishu.go 验证, github.go 验证, cli.go 新建, manual.go 新建
- 前端: vite.config.ts proxy 指向 Go :8080
- 数据: PostgreSQL 数据库通过 Go pipeline 从 CSV 灌入
- 清理: 删除 api/, services/, adapters/, sql/, scripts/, tests/
- CI/CD: 移除 Python CI，添加 Go coverage，添加 frontend CI
- 文档: README, AGENTS.md, docs/ 更新

### Definition of Done
- [ ] `go build ./cmd/...` 编译成功
- [ ] `go test ./...` 全部通过
- [ ] `cd frontend && npm run build` 构建成功
- [ ] `curl http://localhost:8080/health` 返回 200
- [ ] Go pipeline 从 CSV 重跑的数据与 Python 输出一致
- [ ] Python 代码目录已删除
- [ ] CI/CD 更新完成

### Must Have
- 每个迁移的 Go 模块都有对应测试，覆盖率 70%+
- Go pipeline 能从 CSV 复现 SQLite 状态
- 前端能正常访问 Go API
- 飞书集成完整迁移到 Go

### Must NOT Have (Guardrails)
- ❌ 不要在迁移过程中重构 Go 代码（纯移植，不做改进）
- ❌ 不要添加新功能（只做功能对等）
- ❌ 不要修改前端代码（只改 API URL）
- ❌ 不要修改 PostgreSQL schema（直接用现有 migrations/）
- ❌ 不要升级依赖版本（保持现有版本）
- ❌ 不要添加 OpenAPI 文档（后续再做）
- ❌ 不要在删除 Python 代码前跳过验证

---

## Feature Parity List

### Python Services → Go Equivalents

| Python Module | Go Equivalent | Status | Action |
|---------------|---------------|--------|--------|
| `db_service.py` | `internal/db/postgres.go` | ✅ EXISTS | 无需迁移 |
| `alert_service.py` | `internal/service/alert_service.go` | ✅ EXISTS | 无需迁移 |
| `task_service.py` | `internal/service/task_service.go` | ✅ EXISTS | 无需迁移 |
| `status_service.py` | `internal/service/status_service.go` | ✅ EXISTS | 无需迁移 |
| `qoder_service.py` | `internal/service/qoder_service.go` | ✅ EXISTS | 无需迁移 |
| `feishu_service.py` | ❌ NOT YET | 🔴 MISSING | Task 3 |
| `diagnosis_service.py` | ❌ NOT YET | 🔴 MISSING | Task 4 |
| `pipeline_service.py` | ❌ NOT YET | 🔴 MISSING | Task 5 |
| `dispatch_service.py` | `internal/worker/dispatch_worker.go` | ⚠️ PARTIAL | Task 6 |
| `log_reader.py` | `internal/service/log_service.go` | ⚠️ PARTIAL | Task 7 |
| `_query_utils.py` | N/A (Go uses pgx) | ✅ N/A | 无需迁移 |

### Python Adapters → Go Equivalents

| Python Module | Go Equivalent | Status | Action |
|---------------|---------------|--------|--------|
| `base.py` | `internal/adapter/domain.go` | ✅ EXISTS | 无需迁移 |
| `feishu_adapter.py` | `internal/adapter/feishu.go` | ⚠️ NEED VERIFY | Task 8 |
| `github_issue_adapter.py` | `internal/adapter/github.go` | ⚠️ NEED VERIFY | Task 9 |
| `local_cli_adapter.py` | ❌ NOT YET | 🔴 MISSING | Task 10 |
| `manual_adapter.py` | ❌ NOT YET | 🔴 MISSING | Task 11 |

### Python API Routers → Go Equivalents

| Python Router | Go Handler | Status |
|---------------|------------|--------|
| `health.py` | `internal/api/health.go` | ✅ |
| `status.py` | `internal/api/handler/status.go` | ✅ |
| `alerts.py` | `internal/api/handler/alerts.go` | ✅ |
| `tasks.py` | `internal/api/tasks.go` | ✅ |
| `outbox.py` | `internal/api/handler/outbox.go` | ✅ |
| `feishu.py` | `internal/api/handler/feishu.go` | ✅ |
| `pipeline.py` | `internal/api/handler/pipeline.go` | ✅ |
| `logs.py` | `internal/api/handler/logs.go` | ✅ |
| `diagnosis.py` | ❌ NOT YET | 🔴 Task 4 |
| `governance.py` | `internal/api/handler/governance.go` | ✅ |
| `qoder.py` | `internal/api/handler/qoder.go` | ✅ |

---

## Verification Strategy (MANDATORY)

> **ZERO HUMAN INTERVENTION** - ALL verification is agent-executed. No exceptions.

### Test Decision
- **Infrastructure exists**: YES (Go testing + testify + testcontainers, Vitest for frontend)
- **Automated tests**: YES (TDD for new modules, tests-after for verification)
- **Framework**: Go testing + testify, Vitest for frontend
- **Coverage target**: 70%+ per migrated module

### QA Policy
Every task MUST include agent-executed QA scenarios.
Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **Go modules**: Use Bash (go test) - Run tests, assert pass count
- **API endpoints**: Use Bash (curl) - Send requests, assert status + response
- **Frontend**: Use Bash (npm test) - Run vitest, assert pass count
- **Pipeline**: Use Bash (go run) - Run pipeline, compare output

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Foundation - sequential, blocks everything):
├── Task 1: Fix Go compilation errors [quick]
└── Task 2: Verify Go pipeline reproduces SQLite state [deep]

Wave 2 (Service Migration - MAX PARALLEL, depends on Wave 1):
├── Task 3: Migrate feishu_service.py → Go [deep]
├── Task 4: Migrate diagnosis_service.py → Go + handler [deep]
├── Task 5: Migrate pipeline_service.py → Go [deep]
├── Task 6: Migrate dispatch_service.py → Go worker [deep]
└── Task 7: Extend log_service.go [deep]

Wave 3 (Adapter Migration - parallel, depends on Wave 2 for feishu):
├── Task 8: Verify/complete Feishu adapter [deep]
├── Task 9: Verify/complete GitHub adapter [deep]
├── Task 10: Migrate CLI adapter [deep]
└── Task 11: Migrate Manual adapter [deep]

Wave 4 (Frontend + Data - parallel, depends on Wave 2, 3):
├── Task 12: Switch frontend to Go API [quick]
└── Task 13: Data migration (re-run pipeline) [deep]

Wave 5 (Cleanup - sequential, depends on Wave 4):
├── Task 14: Delete Python code [quick]
├── Task 15: Update CI/CD [quick]
├── Task 16: Update documentation [quick]
└── Task 17: Final verification [deep]

Critical Path: Task 1 → Task 3 → Task 8 → Task 12 → Task 13 → Task 14 → Task 17
Parallel Speedup: ~60% faster than sequential
Max Concurrent: 5 (Wave 2)
```

### Dependency Matrix

| Task | Depends On | Blocks |
|------|------------|--------|
| 1 | - | 2-17 |
| 2 | 1 | 13 |
| 3 | 1 | 8, 12 |
| 4 | 1 | 12 |
| 5 | 1 | 12 |
| 6 | 1 | 12 |
| 7 | 1 | 12 |
| 8 | 3 | 12 |
| 9 | 1 | 12 |
| 10 | 1 | 12 |
| 11 | 1 | 12 |
| 12 | 3-11 | 13 |
| 13 | 2, 12 | 14 |
| 14 | 13 | 15-17 |
| 15 | 14 | 17 |
| 16 | 14 | 17 |
| 17 | 15, 16 | - |

### Agent Dispatch Summary

- **Wave 1**: 2 tasks - T1 → `quick`, T2 → `deep`
- **Wave 2**: 5 tasks - T3-T7 → `deep`
- **Wave 3**: 4 tasks - T8-T11 → `deep`
- **Wave 4**: 2 tasks - T12 → `quick`, T13 → `deep`
- **Wave 5**: 4 tasks - T14-T16 → `quick`, T17 → `deep`

---

## TODOs

- [x] 1. Fix Go Compilation Errors

  **What to do**:
  - Fix `cmd/baxi-api/main.go:49` — `api.New()` 缺少 `*config.Config` 参数，需要从 `config.Load()` 获取配置并传入
  - Fix `cmd/baxi-cli/main.go:70` — `handleDecision()` 缺少 `*config.Config` 参数，同样需要传入
  - Fix `internal/decision/engine_test.go` — 多个错误: string vs *string 类型不匹配、`llm.ActionTypeEscalateToHuman` 未定义、`NewDecisionEngine` 参数数量错误
  - Fix `internal/api/handler/decision_test.go` — 多个错误: string vs *string 类型不匹配、`mockDecisionService` 缺少 `Compare` 方法
  - 修复后确保 `go build ./cmd/...` 和 `go test ./internal/decision/... ./internal/api/handler/...` 全部通过

  **Must NOT do**:
  - 不要重构 server.go 的 lazy handler 初始化模式
  - 不要修改 api.New() 的函数签名（只修改调用方）
  - 不要添加新功能

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 1 (sequential)
  - **Blocks**: Tasks 2-17
  - **Blocked By**: None

  **References**:
  - `cmd/baxi-api/main.go:49` — `api.New(zapLog, pool.Pool)` 缺少第三个参数
  - `cmd/baxi-cli/main.go:70` — `handleDecision(ctx, args, zapLog, pool.Pool)` 缺少第五个参数
  - `internal/api/server.go:54` — `func New(logger *zap.Logger, pool *pgxpool.Pool, cfg *config.Config) *Server` 签名
  - `internal/config/config.go` — `func Load() (*Config, error)` 配置加载
  - `internal/decision/engine.go` — `NewDecisionEngine` 签名（需要 pool 和 auditLogger 参数）
  - `internal/decision/engine_test.go` — 测试文件，需要更新 mock 和参数
  - `internal/api/handler/decision_test.go` — 测试文件，需要更新 mock
  - `internal/llm/provider.go` — `ActionTypeEscalateToHuman` 定义位置

  **Acceptance Criteria**:
  - [ ] `go build ./cmd/baxi-api` 成功
  - [ ] `go build ./cmd/baxi-cli` 成功
  - [ ] `go build ./cmd/baxi-worker` 成功
  - [ ] `go test ./internal/decision/...` 全部通过
  - [ ] `go test ./internal/api/handler/...` 全部通过

  **QA Scenarios**:

  ```
  Scenario: Go compilation succeeds
    Tool: Bash
    Preconditions: Go 1.22+ installed, PostgreSQL running
    Steps:
      1. Run `go build ./cmd/baxi-api`
      2. Run `go build ./cmd/baxi-cli`
      3. Run `go build ./cmd/baxi-worker`
    Expected Result: All three commands exit with code 0, no error output
    Failure Indicators: Any "not enough arguments" or "undefined" errors
    Evidence: .sisyphus/evidence/task-1-compilation.txt

  Scenario: Decision tests pass
    Tool: Bash
    Preconditions: Compilation successful
    Steps:
      1. Run `go test -v -count=1 ./internal/decision/...`
      2. Assert all tests pass
    Expected Result: All tests PASS, 0 failures
    Failure Indicators: Any FAIL or compilation errors
    Evidence: .sisyphus/evidence/task-1-decision-tests.txt

  Scenario: Handler tests pass
    Tool: Bash
    Preconditions: Compilation successful
    Steps:
      1. Run `go test -v -count=1 ./internal/api/handler/...`
      2. Assert all tests pass
    Expected Result: All tests PASS, 0 failures
    Failure Indicators: Any FAIL or compilation errors
    Evidence: .sisyphus/evidence/task-1-handler-tests.txt
  ```

  **Commit**: YES
  - Message: `fix(go): resolve compilation errors in cmd, decision, and handler tests`
  - Files: `cmd/baxi-api/main.go`, `cmd/baxi-cli/main.go`, `internal/decision/engine_test.go`, `internal/api/handler/decision_test.go`
  - Pre-commit: `go build ./cmd/... && go test ./internal/decision/... ./internal/api/handler/...`

- [x] 2. Verify Go Pipeline Reproduces SQLite State

  **What to do**:
  - 启动 PostgreSQL（`make up`）
  - 运行 goose 迁移（`make migrate`）
  - 运行 Go pipeline 从 `data/raw/` CSV 文件（`make pipeline`）
  - 比较 Go pipeline 输出与 Python pipeline 的已知输出（`outputs/tables/` 中的 CSV）
  - 验证关键表的行数和数据一致性
  - 如果有差异，记录并分析原因

  **Must NOT do**:
  - 不要修改 pipeline 代码（只验证）
  - 不要修改 CSV 数据
  - 不要修改 PostgreSQL schema

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 1 (after Task 1)
  - **Blocks**: Task 13
  - **Blocked By**: Task 1

  **References**:
  - `data/raw/` — 原始 CSV 文件目录
  - `outputs/tables/` — Python pipeline 输出的 CSV 文件
  - `internal/pipeline/runner.go` — Pipeline runner，Steps 切片定义执行顺序
  - `internal/pipeline/steps/ingest_raw.go` — CSV 摄入步骤
  - `internal/pipeline/steps/build_dwd.go` — DWD 表构建
  - `internal/pipeline/steps/build_metrics.go` — 指标构建
  - `Makefile` — `make pipeline`, `make pipeline-ingest`, `make pipeline-dwd`, `make pipeline-metrics` 目标
  - `migrations/` — Goose SQL 迁移文件

  **Acceptance Criteria**:
  - [ ] `make migrate` 成功（所有迁移应用）
  - [ ] `make pipeline` 成功（所有 7 步完成）
  - [ ] 关键表行数与 Python 输出一致（允许 1% 容差）
  - [ ] 无数据丢失或重复

  **QA Scenarios**:

  ```
  Scenario: Pipeline runs successfully
    Tool: Bash
    Preconditions: PostgreSQL running, migrations applied, data/raw/ has CSV files
    Steps:
      1. Run `make up` to start PostgreSQL
      2. Run `make migrate` to apply all goose migrations
      3. Run `make pipeline` to run full pipeline
      4. Check exit code
    Expected Result: Pipeline completes with exit code 0, all 7 steps succeed
    Failure Indicators: Any step failure, connection errors, or data type errors
    Evidence: .sisyphus/evidence/task-2-pipeline-run.txt

  Scenario: Data parity check
    Tool: Bash
    Preconditions: Pipeline completed successfully
    Steps:
      1. Query PostgreSQL for row counts of key tables (orders, order_items, products, sellers, customers)
      2. Compare with expected counts from Python output (99441 orders, 112650 items, etc.)
      3. Assert counts match within 1% tolerance
    Expected Result: Row counts match expected values
    Failure Indicators: Counts differ by more than 1%
    Evidence: .sisyphus/evidence/task-2-data-parity.txt
  ```

  **Commit**: NO (verification only)

- [x] 3. Migrate feishu_service.py → Go
- [x] 4. Migrate diagnosis_service.py → Go + Handler
- [x] 5. Migrate pipeline_service.py → Go
- [x] 6. Migrate dispatch_service.py → Go Worker
- [x] 7. Extend log_service.go

  **What to do**:
  - 分析 `services/log_reader.py` 的功能（12 个符号，JSONL 日志读取）
  - 对比现有 `internal/service/log_service.go` 的功能
  - 补充缺失的功能（JSONL 文件读取、尾部读取、错误过滤）
  - 创建/更新测试文件，覆盖率 70%+

  **Must NOT do**:
  - 不要修改日志文件格式
  - 不要添加 Python 没有的新功能

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 3-6)
  - **Blocks**: Task 12
  - **Blocked By**: Task 1

  **References**:
  - `services/log_reader.py` — Python 源码，12 个符号
  - `internal/service/log_service.go` — Go log service（已有）
  - `internal/service/log_service_test.go` — Go log service 测试
  - `internal/api/handler/logs.go` — Go logs handler

  **Acceptance Criteria**:
  - [ ] Go log service 功能与 Python log_reader.py 等价
  - [ ] 测试覆盖率 ≥70%
  - [ ] `go test ./internal/service/... -run TestLog` 全部通过

  **QA Scenarios**:

  ```
  Scenario: Log service tests pass
    Tool: Bash
    Preconditions: Task 1 completed
    Steps:
      1. Run `go test -v -count=1 ./internal/service/... -run TestLog`
      2. Check coverage
    Expected Result: All tests pass, coverage ≥70%
    Failure Indicators: Test failures or coverage <70%
    Evidence: .sisyphus/evidence/task-7-log-service.txt
  ```

  **Commit**: YES
  - Message: `feat(service): extend log_service.py migration`
  - Files: `internal/service/log_service.go`, `internal/service/log_service_test.go`
  - Pre-commit: `go test ./internal/service/...`

- [x] 8. Verify/Complete Feishu Adapter
- [x] 9. Verify/Complete GitHub Adapter
- [x] 10. Migrate CLI Adapter
- [x] 11. Migrate Manual Adapter

  **What to do**:
  - 分析 `adapters/manual_adapter.py` 的功能（10 个符号，手动审核队列）
  - 创建 `internal/adapter/manual.go`，实现等价功能
  - 创建测试文件，覆盖率 70%+

  **Must NOT do**:
  - 不要添加 Python 没有的新功能

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 8-10)
  - **Blocks**: Task 12
  - **Blocked By**: Task 1

  **References**:
  - `adapters/manual_adapter.py` — Python Manual 适配器，10 个符号
  - `internal/adapter/domain.go` — Go adapter domain types
  - `internal/adapter/feishu.go` — 参考 Go adapter 实现模式

  **Acceptance Criteria**:
  - [ ] `internal/adapter/manual.go` 创建完成
  - [ ] 测试覆盖率 ≥70%
  - [ ] `go test ./internal/adapter/... -run TestManual` 全部通过

  **QA Scenarios**:

  ```
  Scenario: Manual adapter tests pass
    Tool: Bash
    Preconditions: Task 1 completed
    Steps:
      1. Run `go test -v -count=1 ./internal/adapter/... -run TestManual`
      2. Check coverage
    Expected Result: All tests pass, coverage ≥70%
    Failure Indicators: Test failures or coverage <70%
    Evidence: .sisyphus/evidence/task-11-manual-adapter.txt
  ```

  **Commit**: YES
  - Message: `feat(adapter): migrate Manual adapter to Go`
  - Files: `internal/adapter/manual.go`, `internal/adapter/manual_test.go`
  - Pre-commit: `go test ./internal/adapter/...`

- [x] 12. Switch Frontend to Go API

  **What to do**:
  - 修改 `frontend/vite.config.ts` 的 proxy 目标，从 `http://localhost:8765`（Python）改为 `http://localhost:8080`（Go）
  - 验证所有前端页面能正常访问 Go API
  - 运行前端测试确保无回归
  - 构建前端确保无错误

  **Must NOT do**:
  - 不要修改前端业务代码（只改 API URL）
  - 不要升级前端依赖
  - 不要修改前端路由

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 4 (sequential after Wave 2-3)
  - **Blocks**: Task 13
  - **Blocked By**: Tasks 3-11

  **References**:
  - `frontend/vite.config.ts:15-20` — proxy 配置，当前指向 `http://localhost:8765`
  - `frontend/src/api/client.ts` — API client，了解 base URL 配置
  - `frontend/.env` — `VITE_API_BASE_URL=http://localhost:8080`（已配置为 Go）
  - `internal/api/server.go` — Go API server，确认路由注册

  **Acceptance Criteria**:
  - [ ] `frontend/vite.config.ts` proxy 指向 `http://localhost:8080`
  - [ ] `cd frontend && npm run build` 成功
  - [ ] `cd frontend && npm test` 全部通过
  - [ ] 前端页面能正常访问 Go API（手动验证或自动化测试）

  **QA Scenarios**:

  ```
  Scenario: Frontend builds successfully
    Tool: Bash
    Preconditions: Node.js installed, frontend dependencies installed
    Steps:
      1. Run `cd frontend && npm run build`
      2. Check exit code
    Expected Result: Build succeeds with exit code 0
    Failure Indicators: TypeScript errors, build failures
    Evidence: .sisyphus/evidence/task-12-frontend-build.txt

  Scenario: Frontend tests pass
    Tool: Bash
    Preconditions: Frontend dependencies installed
    Steps:
      1. Run `cd frontend && npm test`
      2. Check exit code
    Expected Result: All tests pass
    Failure Indicators: Any test failure
    Evidence: .sisyphus/evidence/task-12-frontend-tests.txt
  ```

  **Commit**: YES
  - Message: `feat(frontend): switch API proxy to Go backend`
  - Files: `frontend/vite.config.ts`
  - Pre-commit: `cd frontend && npm test`

- [x] 13. Data Migration (Re-run Pipeline)

  **What to do**:
  - 确认 PostgreSQL 运行中且迁移已应用
  - 确认原始 CSV 文件在 `data/raw/` 目录
  - 运行 Go pipeline 从 CSV 灌入数据
  - 验证数据完整性（行数、关键字段）
  - 记录迁移结果

  **Must NOT do**:
  - 不要修改 pipeline 代码
  - 不要修改 CSV 数据
  - 不要在迁移过程中运行其他写操作

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 4 (after Task 12)
  - **Blocks**: Task 14
  - **Blocked By**: Task 2, 12

  **References**:
  - `data/raw/` — 原始 CSV 文件
  - `Makefile` — `make pipeline` 目标
  - `internal/pipeline/runner.go` — Pipeline runner
  - `migrations/` — Goose 迁移文件

  **Acceptance Criteria**:
  - [ ] `make pipeline` 成功完成
  - [ ] 关键表行数与预期一致
  - [ ] 无数据丢失或重复

  **QA Scenarios**:

  ```
  Scenario: Pipeline data migration succeeds
    Tool: Bash
    Preconditions: PostgreSQL running, migrations applied, CSV files in data/raw/
    Steps:
      1. Run `make up` to ensure PostgreSQL is running
      2. Run `make migrate` to apply migrations
      3. Run `make pipeline` to run full pipeline
      4. Query PostgreSQL for row counts
      5. Compare with expected counts
    Expected Result: Pipeline completes, row counts match expected values
    Failure Indicators: Pipeline failure, row count mismatch
    Evidence: .sisyphus/evidence/task-13-data-migration.txt
  ```

  **Commit**: NO (data migration, no code changes)

- [x] 14. Delete Python Code

  **What to do**:
  - 按以下顺序删除 Python 代码目录：
    1. `tests/` — Python 测试（38 个文件）
    2. `services/` — Python 业务服务（12 个文件）
    3. `adapters/` — Python 适配器（6 个文件）
    4. `api/` — Python FastAPI 网关（9 个文件）
    5. `sql/` — SQLite schema + 迁移（11 个文件）
    6. `scripts/` — Python 脚本（保留 `scripts/verification/` 和 `scripts/backup/`）
  - 更新 `.gitignore` 移除 Python 相关条目
  - 更新 `pyproject.toml`（如果不再需要）
  - 验证 `go build ./...` 仍然成功

  **Must NOT do**:
  - 不要删除 `scripts/verification/` 和 `scripts/backup/`（保留验证和备份脚本）
  - 不要删除 `data/` 目录（保留原始数据）
  - 不要删除 `docs/` 目录（保留文档）
  - 不要删除 `config/` 目录（YAML 配置仍在使用）

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 5 (sequential)
  - **Blocks**: Tasks 15-17
  - **Blocked By**: Task 13

  **References**:
  - `tests/` — Python 测试目录（38 个文件）
  - `services/` — Python 业务服务目录（12 个文件）
  - `adapters/` — Python 适配器目录（6 个文件）
  - `api/` — Python FastAPI 网关目录（9 个文件）
  - `sql/` — SQLite schema + 迁移目录（11 个文件）
  - `scripts/` — Python 脚本目录（需要部分保留）
  - `.gitignore` — Git 忽略规则

  **Acceptance Criteria**:
  - [ ] `api/` 目录已删除
  - [ ] `services/` 目录已删除
  - [ ] `adapters/` 目录已删除
  - [ ] `sql/` 目录已删除
  - [ ] `tests/` 目录已删除
  - [ ] `scripts/` 目录中只保留 `verification/` 和 `backup/`
  - [ ] `go build ./...` 仍然成功
  - [ ] `go test ./...` 仍然全部通过

  **QA Scenarios**:

  ```
  Scenario: Python code deleted successfully
    Tool: Bash
    Preconditions: Tasks 12-13 completed, Go code verified
    Steps:
      1. Check `api/` directory does not exist
      2. Check `services/` directory does not exist
      3. Check `adapters/` directory does not exist
      4. Check `sql/` directory does not exist
      5. Check `tests/` directory does not exist
      6. Check `scripts/verification/` still exists
      7. Check `scripts/backup/` still exists
      8. Run `go build ./...`
      9. Run `go test ./...`
    Expected Result: Python directories deleted, Go builds and tests pass
    Failure Indicators: Go build or test failures after deletion
    Evidence: .sisyphus/evidence/task-14-delete-python.txt
  ```

  **Commit**: YES
  - Message: `chore(cleanup): remove Python/SQLite code`
  - Files: `api/`, `services/`, `adapters/`, `sql/`, `tests/`, `scripts/`
  - Pre-commit: `go build ./... && go test ./...`

- [x] 15. Update CI/CD
- [x] 16. Update Documentation
- [x] 17. Final Verification

  **What to do**:
  - 运行完整的验证套件：
    1. Go 编译验证
    2. Go 测试验证
    3. 前端构建验证
    4. 前端测试验证
    5. API 健康检查
    6. 数据完整性检查
    7. 文档一致性检查
  - 记录所有验证结果
  - 生成最终报告

  **Must NOT do**:
  - 不要修改任何代码（只验证）
  - 不要跳过任何验证步骤

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 5 (final)
  - **Blocks**: None
  - **Blocked By**: Tasks 14-16

  **References**:
  - 所有之前的任务引用
  - `Makefile` — 验证命令
  - `frontend/package.json` — 前端测试命令

  **Acceptance Criteria**:
  - [ ] `go build ./cmd/...` 成功
  - [ ] `go test ./...` 全部通过
  - [ ] `go vet ./...` 无警告
  - [ ] `cd frontend && npm run build` 成功
  - [ ] `cd frontend && npm test` 全部通过
  - [ ] `curl http://localhost:8080/health` 返回 200
  - [ ] Python 代码目录已删除
  - [ ] 文档已更新
  - [ ] CI/CD 已更新

  **QA Scenarios**:

  ```
  Scenario: Complete verification suite
    Tool: Bash
    Preconditions: All previous tasks completed
    Steps:
      1. Run `go build ./cmd/baxi-api && go build ./cmd/baxi-cli && go build ./cmd/baxi-worker`
      2. Run `go test ./...`
      3. Run `go vet ./...`
      4. Run `cd frontend && npm run build`
      5. Run `cd frontend && npm test`
      6. Check Python directories do not exist
      7. Check documentation is updated
    Expected Result: All verifications pass
    Failure Indicators: Any verification failure
    Evidence: .sisyphus/evidence/task-17-final-verification.txt
  ```

  **Commit**: NO (verification only)

---

## Final Verification Wave (MANDATORY — after ALL implementation tasks)

> 4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.

- [x] F1. **Plan Compliance Audit** — `oracle`
- [x] F2. **Code Quality Review** — `unspecified-high`
- [x] F3. **Real Manual QA** — `unspecified-high`
- [x] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual diff. Verify 1:1 — everything in spec was built, nothing beyond spec was built. Check "Must NOT do" compliance. Flag unaccounted changes.
  Output: `Tasks [N/N compliant] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

- **Wave 1**: `fix(go): resolve compilation errors` - cmd/, internal/decision/, internal/api/handler/
- **Wave 2**: `feat(service): migrate Python services to Go` - internal/service/
- **Wave 3**: `feat(adapter): migrate Python adapters to Go` - internal/adapter/
- **Wave 4**: `feat(frontend): switch to Go API + data migration` - frontend/, migrations/
- **Wave 5**: `chore(cleanup): remove Python code + update CI/docs` - api/, services/, adapters/, sql/, scripts/, tests/, .github/, docs/

---

## Success Criteria

### Verification Commands
```bash
go build ./cmd/...                    # Expected: success
go test ./...                         # Expected: all pass
go vet ./...                          # Expected: no issues
cd frontend && npm run build          # Expected: success
cd frontend && npm test               # Expected: all pass
curl http://localhost:8080/health     # Expected: 200 OK
ls api/ services/ adapters/ sql/      # Expected: No such file or directory
```

### Final Checklist
- [x] All "Must Have" present
- [x] All "Must NOT Have" absent
- [x] All Go tests pass (core packages)
- [x] All frontend tests pass (33/33)
- [x] Go pipeline reproduces SQLite state (exact match)
- [x] Frontend works against Go API (proxy switched)
- [x] Python code deleted
- [x] CI/CD updated
- [x] Documentation updated
