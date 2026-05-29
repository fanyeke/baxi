# Palantir AIP 理念集成到 Baxi 平台

## TL;DR

> **Quick Summary**: 将 Palantir AIP 理念集成到 Baxi 电商平台，实现 Ontology MCP 工具暴露、决策生命周期闭环、审核工作流增强、风险自适应 HITL、持久化提案沙箱和行动类型模式目录。目标是让 Pi Agent 能够读取本体论数据、生成决策建议、人工审核后写回执行。
> 
> **Deliverables**:
> - Ontology MCP 工具 (get_object, get_linked_objects, describe_ontology, execute_action)
> - 决策生命周期修复 (decide 创建提案, 案例写回汇聚点)
> - 审核工具补全 (cancel_proposal, get_proposal_by_id, list_review_records)
> - 3 个 MCP 桩修复 (execute_proposal, get_system_status, search_objects)
> - 行动类型模式目录 (结构化 ActionDefinition)
> - 风险自适应 HITL (自动批准低风险操作)
> - 持久化提案沙箱 (提案比较和选择)
> - OAG 链接遍历 (上下文构建器增强)
> 
> **Estimated Effort**: 4-6 周
> **Parallel Execution**: YES - 5 个波次
> **Critical Path**: Task 1-3 (桩修复) → Task 4-7 (Ontology MCP) → Task 8-11 (决策修复) → Task 12-15 (审核增强) → Task 16-20 (AIP 特性) → Task 21-24 (集成测试)

---

## Context

### Original Request
用户请求将 Palantir AIP 理念集成到 Baxi 平台：
1. 属性、关系、动作的接口需要让 Pi 可以读取
2. Pi 生成的决策需要可以让人工审核，审核之后可以写回
3. 需要补充其他的和 Palantir AIP 理念相关的设计
4. 基于 LLM 决策这一步需要完成

### Interview Summary
**Key Discussions**:
- **Pi 使用场景**: 决策辅助 — Pi 分析数据后生成决策建议，人工审核后执行
- **AIP 核心概念**: Ontology (本体论) — 把业务对象建模为可交互的数字孪生
- **审核流程**: Pi Agent 内审核 — 在 Pi Agent 对话中直接审核，通过 MCP 工具写回
- **决策场景**: 全部都要 — 告警响应 + 风险评估 + 运营优化
- **写回含义**: 两者都要 — 更新案例状态 + 触发动作执行
- **实现范围**: 完整实现 (4-6周)
- **测试策略**: TDD

**Research Findings**:
- MCP 服务器有 17 个工具，3 个是桩实现
- 决策引擎有 4 阶段 LLM 链但 MCP `decide` 绕过 ProposalService
- Ontology 包已有 ObjectType/Property/Link 但无 MCP 暴露
- 审核工作流有 approve/reject 但缺少 cancel、get、list 操作
- 10+ 后端服务存在但未暴露为 MCP 工具

### Gap Analysis (Self-Review)
**Identified Gaps** (addressed):
- 对象类型优先级: 默认暴露所有 8 种类型
- 沙箱实现方式: 新建 `ai.proposal_sandbox` 表
- 风险级别确定: 配置驱动 (action_registry.yml)
- 循环链接处理: 添加深度限制 (最多 3 跳)

---

## Work Objectives

### Core Objective
将 Baxi 平台从"LLM 生成决策"升级为"Ontology 驱动的完整决策生命周期"，实现 Pi Agent 读取本体论数据、生成决策建议、人工审核后写回执行的闭环。

### Concrete Deliverables
- `internal/mcp/tools_ontology.go` — 4 个新 MCP 工具
- `internal/mcp/tools_review.go` — 4 个增强/新工具
- `internal/mcp/tools_status.go` — 2 个桩修复
- `internal/mcp/tools_action.go` — 1 个桩修复
- `internal/action/schema.go` — 行动类型模式目录
- `internal/review/sandbox.go` — 持久化提案沙箱
- `internal/decision/case_service.go` — 案例写回汇聚点
- `internal/decision/context_builder_v3.go` — OAG 链接遍历
- `config/action_registry.yml` — 增强的行动定义
- 新增数据库迁移文件

### Definition of Done
- [ ] 所有 MCP 工具通过 Pi Agent 可调用
- [ ] 决策生命周期完整: 创建案例 → 生成决策 → 创建提案 → 人工审核 → 写回执行
- [ ] 所有桩实现替换为真实实现
- [ ] 行动类型模式目录可通过 MCP 暴露
- [ ] 风险自适应 HITL 可配置
- [ ] 持久化沙箱支持提案比较
- [ ] OAG 链接遍历在上下文构建器中工作
- [ ] 所有测试通过 (TDD)

### Must Have
- Ontology MCP 工具必须暴露所有 8 种对象类型
- 决策 MCP `decide` 必须创建可审核的提案
- 审核后必须能更新案例状态和触发动作执行
- 所有 MCP 桩必须替换为真实实现
- 行动类型必须有结构化的 JSON Schema 定义

### Must NOT Have (Guardrails)
- 不修改现有 MCP 工具的接口签名 (向后兼容)
- 不绕过治理策略 (所有操作必须经过 access_policy 检查)
- 不硬编码风险级别 (必须配置驱动)
- 不实现外部系统写回 (ERP/CRM)
- 不实现多 Agent 协调
- 不实现调度编排
- 不实现 SIEM 导出

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed. No exceptions.

### Test Decision
- **Infrastructure exists**: YES (testify, testcontainers-go)
- **Automated tests**: TDD
- **Framework**: Go testify + testcontainers-go

### QA Policy
Every task MUST include agent-executed QA scenarios.
Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **MCP Tools**: Use bash to start MCP server, send tool calls via stdin, validate JSON responses
- **Database**: Use testcontainers-go to spin up PostgreSQL, run migrations, verify data
- **API**: Use curl to test HTTP endpoints
- **Frontend**: Use Playwright to test UI interactions

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Foundation - 3 tasks, parallel):
├── Task 1: 修复 execute_proposal 桩 [quick]
├── Task 2: 修复 get_system_status 桩 [quick]
└── Task 3: 修复 search_objects 桩 [quick]

Wave 2 (Ontology MCP - 4 tasks, parallel after Wave 1):
├── Task 4: 添加 describe_ontology MCP 工具 [unspecified-high]
├── Task 5: 添加 get_object MCP 工具 [unspecified-high]
├── Task 6: 添加 get_linked_objects MCP 工具 [unspecified-high]
└── Task 7: 添加 execute_action MCP 工具 [unspecified-high]

Wave 3 (Decision Lifecycle - 4 tasks, parallel after Wave 2):
├── Task 8: 修复 MCP decide 创建提案 [deep]
├── Task 9: 添加案例写回汇聚点 [deep]
├── Task 10: 添加 cancel_proposal MCP 工具 [unspecified-high]
└── Task 11: 添加 get_proposal_by_id MCP 工具 [unspecified-high]

Wave 4 (Review Enhancement - 4 tasks, parallel after Wave 3):
├── Task 12: 添加 list_review_records MCP 工具 [unspecified-high]
├── Task 13: 创建行动类型模式目录 [deep]
├── Task 14: 实现风险自适应 HITL [deep]
└── Task 15: 创建持久化提案沙箱 [deep]

Wave 5 (AIP Features - 5 tasks, parallel after Wave 4):
├── Task 16: 实现 OAG 链接遍历 [deep]
├── Task 17: 添加前端决策审核页面 [visual-engineering]
├── Task 18: 添加前端沙箱比较页面 [visual-engineering]
├── Task 19: 集成测试 - 决策生命周期 [unspecified-high]
└── Task 20: 集成测试 - Ontology MCP [unspecified-high]

Wave FINAL (After ALL tasks — 4 parallel reviews, then user okay):
├── Task F1: Plan compliance audit (oracle)
├── Task F2: Code quality review (unspecified-high)
├── Task F3: Real manual QA (unspecified-high)
└── Task F4: Scope fidelity check (deep)
-> Present results -> Get explicit user okay

Critical Path: Task 1-3 → Task 4-7 → Task 8-11 → Task 12-15 → Task 16-20 → F1-F4 → user okay
Parallel Speedup: ~60% faster than sequential
Max Concurrent: 5 (Wave 5)
```

### Dependency Matrix

| Task | Depends On | Blocks |
|------|------------|--------|
| 1-3 | - | 4-7 |
| 4-7 | 1-3 | 8-11, 16 |
| 8-11 | 4-7 | 12-15, 19 |
| 12-15 | 8-11 | 16-20 |
| 16-20 | 12-15 | F1-F4 |

### Agent Dispatch Summary

- **Wave 1**: 3 × `quick` — 桩修复
- **Wave 2**: 4 × `unspecified-high` — MCP 工具
- **Wave 3**: 2 × `deep` + 2 × `unspecified-high` — 决策修复
- **Wave 4**: 3 × `deep` + 1 × `unspecified-high` — 审核增强
- **Wave 5**: 2 × `deep` + 2 × `visual-engineering` + 2 × `unspecified-high` — AIP 特性
- **FINAL**: 4 × `oracle`/`unspecified-high`/`deep` — 验证

---

## TODOs

### Wave 1: Foundation — 修复 MCP 桩实现

- [x] 1. 修复 execute_proposal MCP 桩

  **What to do**:
  - 将 `cmd/baxi-mcp/main.go` 中的 `executeServiceAdapter` 替换为真实的 `action.ApplyService` 调用
  - 在 `main.go` 中装配 `ApplyService` (需要 ActionRegistry, ReviewRepository, OutboxService, AdapterRegistry)
  - 更新 `internal/mcp/interfaces.go` 中的 `ExecuteService` 接口以匹配 `ApplyService.ExecuteProposal` 签名
  - 添加 dry_run 支持: 当 dry_run=true 时使用 NoOpExecutor
  - TDD: 先写测试验证 execute_proposal 真实执行流程

  **Must NOT do**:
  - 不修改 `tools_action.go` 的 handler 接口 (只修改 main.go 适配器)
  - 不绕过治理策略检查

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2, 3)
  - **Blocks**: Tasks 4-7
  - **Blocked By**: None

  **References**:
  - `cmd/baxi-mcp/main.go:executeServiceAdapter` — 当前桩实现 (第 180-190 行)
  - `internal/action/apply_service.go:ApplyService.ExecuteProposal` — 真实执行逻辑
  - `internal/action/executor.go:NoOpExecutor` — dry_run 执行器
  - `internal/mcp/interfaces.go:ExecuteService` — 接口定义
  - `internal/mcp/tools_action.go:handleExecuteProposal` — handler (不修改)

  **Acceptance Criteria**:
  - [ ] `execute_proposal` MCP 工具调用真实 ApplyService
  - [ ] dry_run=true 返回 NoOpExecutor 结果
  - [ ] dry_run=false 实际执行动作并创建 outbox 事件
  - [ ] 测试通过

  **QA Scenarios**:
  ```
  Scenario: execute_proposal with dry_run
    Tool: bash (MCP stdio)
    Preconditions: 一个 approved 状态的 proposal 存在
    Steps:
      1. 启动 MCP server
      2. 调用 execute_proposal(proposal_id=..., dry_run=true)
      3. 验证返回 success=true, dry_run=true
      4. 验证数据库中 proposal 状态未改变
    Expected Result: dry_run 不产生副作用
    Evidence: .sisyphus/evidence/task-1-execute-dry-run.json

  Scenario: execute_proposal with real execution
    Tool: bash (MCP stdio)
    Preconditions: 一个 approved 状态的 proposal 存在
    Steps:
      1. 启动 MCP server
      2. 调用 execute_proposal(proposal_id=..., dry_run=false)
      3. 验证返回 success=true, outbox_event_id 非空
      4. 验证数据库中 proposal 状态变为 "applied"
      5. 验证 outbox.outbox_event 记录已创建
    Expected Result: 真实执行产生 outbox 事件
    Evidence: .sisyphus/evidence/task-1-execute-real.json
  ```

  **Commit**: YES
  - Message: `fix(mcp): connect execute_proposal to real ApplyService`
  - Files: `cmd/baxi-mcp/main.go`, `internal/mcp/interfaces.go`
  - Pre-commit: `go test ./internal/mcp/... ./internal/action/...`

---

- [x] 2. 修复 get_system_status MCP 桩

  **What to do**:
  - 将 `cmd/baxi-mcp/main.go` 中的 `statusServiceAdapter` 替换为真实的数据库查询
  - 实现 `SystemStatusService.GetStatus()`: 查询 alert_count, pipeline_run, table_counts, recent_errors
  - 使用 `pgxpool.Pool` 直接查询 `ops.metric_alert`, `pipeline_runs`, 信息架构表
  - TDD: 先写测试验证 GetStatus 返回正确数据

  **Must NOT do**:
  - 不修改 `tools_status.go` 的 handler 接口
  - 不引入新的 service 包 (直接在 main.go 适配器中实现)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 3)
  - **Blocks**: Tasks 4-7
  - **Blocked By**: None

  **References**:
  - `cmd/baxi-mcp/main.go:statusServiceAdapter` — 当前桩实现
  - `internal/model/status.go:SystemStatus` — 返回类型定义
  - `internal/service/status_service.go` — 如果存在，参考真实实现
  - `internal/mcp/tools_status.go:handleGetSystemStatus` — handler (不修改)

  **Acceptance Criteria**:
  - [ ] `get_system_status` MCP 工具返回真实数据
  - [ ] alert_count 反映 ops.metric_alert 表中的实际记录数
  - [ ] table_counts 包含所有主要表的行数
  - [ ] recent_errors 返回最近的错误日志
  - [ ] 测试通过

  **QA Scenarios**:
  ```
  Scenario: get_system_status returns real data
    Tool: bash (MCP stdio)
    Preconditions: 数据库有告警和管道运行记录
    Steps:
      1. 启动 MCP server
      2. 调用 get_system_status()
      3. 验证返回 alert_count > 0
      4. 验证返回 table_counts 非空
      5. 验证返回 pipeline_run 包含 last_run 信息
    Expected Result: 返回真实系统状态
    Evidence: .sisyphus/evidence/task-2-system-status.json
  ```

  **Commit**: YES (groups with 1, 3)
  - Message: `fix(mcp): connect get_system_status to real queries`
  - Files: `cmd/baxi-mcp/main.go`
  - Pre-commit: `go test ./internal/mcp/...`

---

- [x] 3. 修复 search_objects MCP 桩

  **What to do**:
  - 将 `cmd/baxi-mcp/main.go` 中的 `searchServiceAdapter` 连接到 `ontology.ObjectQueryService`
  - `objectSvc` 已经在 main.go 中实例化，但搜索适配器未使用它
  - 更新适配器以调用 `objectSvc.SearchObjects(ctx, objectType, query, limit, offset)`
  - TDD: 先写测试验证 search_objects 返回真实数据

  **Must NOT do**:
  - 不修改 `tools_status.go` 的 handler 接口
  - 不修改 `ontology.ObjectQueryService` 的实现

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2)
  - **Blocks**: Tasks 4-7
  - **Blocked By**: None

  **References**:
  - `cmd/baxi-mcp/main.go:searchServiceAdapter` — 当前桩实现
  - `cmd/baxi-mcp/main.go:objectSvc` — 已实例化的 ObjectQueryService
  - `internal/ontology/query_service.go:ObjectQueryService.SearchObjects` — 真实查询逻辑
  - `internal/mcp/tools_status.go:handleSearchObjects` — handler (不修改)

  **Acceptance Criteria**:
  - [ ] `search_objects` MCP 工具调用真实 ObjectQueryService
  - [ ] 返回匹配的对象列表
  - [ ] 支持 object_type 和 query 参数过滤
  - [ ] 测试通过

  **QA Scenarios**:
  ```
  Scenario: search_objects returns real results
    Tool: bash (MCP stdio)
    Preconditions: 数据库有订单和卖家数据
    Steps:
      1. 启动 MCP server
      2. 调用 search_objects(object_type="order", query="delayed")
      3. 验证返回 items 非空
      4. 验证每个 item 包含 object_type, object_id, properties
    Expected Result: 返回匹配的订单对象
    Evidence: .sisyphus/evidence/task-3-search-objects.json
  ```

  **Commit**: YES (groups with 1, 2)
  - Message: `fix(mcp): connect search_objects to ontology ObjectQueryService`
  - Files: `cmd/baxi-mcp/main.go`
  - Pre-commit: `go test ./internal/mcp/... ./internal/ontology/...`

---

### Wave 2: Ontology MCP — 暴露本体论给 Pi Agent

- [x] 4. 添加 describe_ontology MCP 工具

  **What to do**:
  - 创建 `internal/mcp/tools_ontology.go` 文件
  - 实现 `describe_ontology` MCP 工具: 列出所有注册的对象类型、属性、链接、允许的动作
  - 调用 `ontology.ObjectRegistry.ListTypes()` 获取所有类型
  - 返回每个类型的: name, display_name, grain, properties[], links[], allowed_actions[]
  - 注册到 `server.go` 的 `registerOntologyTools()` 方法
  - TDD: 先写测试验证 describe_ontology 返回完整本体论结构

  **Must NOT do**:
  - 不修改现有 ontology/ 包的实现
  - 不暴露敏感属性 (LLMReadable=false 的属性)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 5, 6, 7)
  - **Blocks**: Tasks 8-11, 16
  - **Blocked By**: Tasks 1-3

  **References**:
  - `internal/ontology/schema.go:ObjectType` — 对象类型定义
  - `internal/ontology/schema.go:ObjectProperty` — 属性定义 (含 LLMReadable 标志)
  - `internal/ontology/schema.go:ObjectLink` — 链接定义
  - `internal/ontology/registry.go:ObjectRegistry.ListTypes` — 获取所有类型
  - `internal/mcp/tools_governance.go` — 参考现有工具模式
  - `internal/mcp/server.go:registerGovernanceTools` — 参考注册模式

  **Acceptance Criteria**:
  - [ ] `describe_ontology` MCP 工具返回所有注册的对象类型
  - [ ] 每个类型包含 name, display_name, grain, properties, links, allowed_actions
  - [ ] LLMReadable=false 的属性被过滤
  - [ ] 测试通过

  **QA Scenarios**:
  ```
  Scenario: describe_ontology returns all types
    Tool: bash (MCP stdio)
    Preconditions: aip_object_schema.yml 已配置 8 种对象类型
    Steps:
      1. 启动 MCP server
      2. 调用 describe_ontology()
      3. 验证返回 object_types 数组
      4. 验证包含 order, seller, product, customer 等类型
      5. 验证每个类型有 properties 和 links
    Expected Result: 返回完整的本体论结构
    Evidence: .sisyphus/evidence/task-4-describe-ontology.json

  Scenario: describe_ontology filters non-LLM-readable properties
    Tool: bash (MCP stdio)
    Preconditions: 有 LLMReadable=false 的属性
    Steps:
      1. 启动 MCP server
      2. 调用 describe_ontology()
      3. 验证返回的 properties 中不包含 LLMReadable=false 的属性
    Expected Result: 敏感属性被过滤
    Evidence: .sisyphus/evidence/task-4-describe-ontology-filter.json
  ```

  **Commit**: YES
  - Message: `feat(mcp): add describe_ontology tool`
  - Files: `internal/mcp/tools_ontology.go`, `internal/mcp/server.go`, `internal/mcp/interfaces.go`
  - Pre-commit: `go test ./internal/mcp/...`

---

- [x] 5. 添加 get_object MCP 工具

  **What to do**:
  - 在 `internal/mcp/tools_ontology.go` 中添加 `get_object` MCP 工具
  - 实现通过 object_type + object_id 检索单个对象的完整信息
  - 调用 `ontology.ObjectQueryService.GetOrder/GetSeller/GetProduct/...` 等类型化方法
  - 返回对象的: type, id, properties, linked_objects (with types and IDs)
  - 支持 depth 参数控制链接遍历深度 (默认 1, 最大 3)
  - TDD: 先写测试验证 get_object 返回完整对象信息

  **Must NOT do**:
  - 不暴露 LLMReadable=false 的属性
  - 不支持深度超过 3 的链接遍历 (防止循环)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 4, 6, 7)
  - **Blocks**: Tasks 8-11, 16
  - **Blocked By**: Tasks 1-3

  **References**:
  - `internal/ontology/query_service.go:ObjectQueryService` — 类型化查询方法
  - `internal/ontology/schema.go:ObjectProperty` — 属性定义
  - `internal/ontology/schema.go:ObjectLink` — 链接定义
  - `internal/governance/redaction.go` — 属性过滤逻辑

  **Acceptance Criteria**:
  - [ ] `get_object` MCP 工具通过 type+id 检索对象
  - [ ] 返回完整属性和链接
  - [ ] depth 参数控制遍历深度
  - [ ] depth > 3 被拒绝
  - [ ] 测试通过

  **QA Scenarios**:
  ```
  Scenario: get_object returns order with links
    Tool: bash (MCP stdio)
    Preconditions: 数据库有订单数据，订单有 seller 和 customer 链接
    Steps:
      1. 启动 MCP server
      2. 调用 get_object(object_type="order", object_id="ORD-001", depth=1)
      3. 验证返回 type=order, id=ORD-001
      4. 验证返回 properties 包含 order_date, total_amount 等
      5. 验证返回 linked_objects 包含 seller 和 customer
    Expected Result: 返回完整订单对象
    Evidence: .sisyphus/evidence/task-5-get-object.json

  Scenario: get_object rejects depth > 3
    Tool: bash (MCP stdio)
    Steps:
      1. 启动 MCP server
      2. 调用 get_object(object_type="order", object_id="ORD-001", depth=5)
      3. 验证返回错误信息
    Expected Result: 拒绝过深的遍历
    Evidence: .sisyphus/evidence/task-5-get-object-depth-error.json
  ```

  **Commit**: YES (groups with 4, 6, 7)
  - Message: `feat(mcp): add get_object tool`
  - Files: `internal/mcp/tools_ontology.go`
  - Pre-commit: `go test ./internal/mcp/...`

---

- [x] 6. 添加 get_linked_objects MCP 工具

  **What to do**:
  - 在 `internal/mcp/tools_ontology.go` 中添加 `get_linked_objects` MCP 工具
  - 实现从源对象遍历链接到关联对象
  - 调用 `ontology.ObjectQueryService` 的链接遍历方法
  - 支持 link_type 过滤 (只返回特定类型的链接)
  - 支持 depth 参数 (默认 1, 最大 3)
  - TDD: 先写测试验证链接遍历

  **Must NOT do**:
  - 不支持深度超过 3 的链接遍历
  - 不暴露 LLMReadable=false 的属性

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 4, 5, 7)
  - **Blocks**: Tasks 8-11, 16
  - **Blocked By**: Tasks 1-3

  **References**:
  - `internal/ontology/schema.go:ObjectLink` — 链接定义
  - `internal/ontology/query_service.go` — 链接遍历方法
  - `internal/ontology/registry.go:ObjectRegistry.GetLinks` — 获取类型链接定义

  **Acceptance Criteria**:
  - [ ] `get_linked_objects` MCP 工具遍历链接
  - [ ] 支持 link_type 过滤
  - [ ] depth 参数控制遍历深度
  - [ ] 测试通过

  **QA Scenarios**:
  ```
  Scenario: get_linked_objects traverses order links
    Tool: bash (MCP stdio)
    Preconditions: 订单 ORD-001 链接到 seller SELL-001 和 customer CUST-001
    Steps:
      1. 启动 MCP server
      2. 调用 get_linked_objects(object_type="order", object_id="ORD-001", depth=1)
      3. 验证返回 linked_objects 包含 seller 和 customer
      4. 验证每个 linked_object 有 type, id, properties
    Expected Result: 返回订单的关联对象
    Evidence: .sisyphus/evidence/task-6-get-linked-objects.json

  Scenario: get_linked_objects with link_type filter
    Tool: bash (MCP stdio)
    Steps:
      1. 启动 MCP server
      2. 调用 get_linked_objects(object_type="order", object_id="ORD-001", link_type="seller")
      3. 验证只返回 seller 类型的链接
    Expected Result: 过滤生效
    Evidence: .sisyphus/evidence/task-6-get-linked-objects-filter.json
  ```

  **Commit**: YES (groups with 4, 5, 7)
  - Message: `feat(mcp): add get_linked_objects tool`
  - Files: `internal/mcp/tools_ontology.go`
  - Pre-commit: `go test ./internal/mcp/...`

---

- [x] 7. 添加 execute_action MCP 工具

  **What to do**:
  - 在 `internal/mcp/tools_ontology.go` 中添加 `execute_action` MCP 工具
  - 实现通过 MCP 调用治理化的动作执行
  - 调用 `action.ApplyService.ExecuteProposal` 或 `action.ActionRegistry.GetAction` 获取动作定义
  - 支持 object_type, action_type, params 参数
  - 执行前验证: 动作在注册表中、参数符合 schema、通过治理策略检查
  - TDD: 先写测试验证动作执行

  **Must NOT do**:
  - 不绕过治理策略检查
  - 不支持未注册的动作类型

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 4, 5, 6)
  - **Blocks**: Tasks 8-11, 16
  - **Blocked By**: Tasks 1-3

  **References**:
  - `internal/action/registry.go:ActionRegistry.GetAction` — 获取动作定义
  - `internal/action/apply_service.go:ApplyService.ExecuteProposal` — 执行逻辑
  - `internal/action/contract.go:ActionContract` — 动作契约类型
  - `config/action_registry.yml` — 动作配置

  **Acceptance Criteria**:
  - [ ] `execute_action` MCP 工具执行治理化动作
  - [ ] 验证动作在注册表中
  - [ ] 验证参数符合 schema
  - [ ] 通过治理策略检查
  - [ ] 测试通过

  **QA Scenarios**:
  ```
  Scenario: execute_action creates outbox event
    Tool: bash (MCP stdio)
    Preconditions: notify_owner 动作已注册
    Steps:
      1. 启动 MCP server
      2. 调用 execute_action(object_type="order", action_type="notify_owner", params={"order_id": "ORD-001", "message": "..."})
      3. 验证返回 success=true
      4. 验证 outbox.outbox_event 记录已创建
    Expected Result: 动作执行成功
    Evidence: .sisyphus/evidence/task-7-execute-action.json

  Scenario: execute_action rejects unregistered action
    Tool: bash (MCP stdio)
    Steps:
      1. 启动 MCP server
      2. 调用 execute_action(object_type="order", action_type="unknown_action", params={})
      3. 验证返回错误信息
    Expected Result: 拒绝未注册的动作
    Evidence: .sisyphus/evidence/task-7-execute-action-error.json
  ```

  **Commit**: YES (groups with 4, 5, 6)
  - Message: `feat(mcp): add execute_action tool`
  - Files: `internal/mcp/tools_ontology.go`
  - Pre-commit: `go test ./internal/mcp/... ./internal/action/...`

---

### Wave 3: Decision Lifecycle — 修复决策闭环

- [x] 8. 修复 MCP decide 创建提案

  **What to do**:
  - 修改 `internal/mcp/tools_decision.go` 中的 `handleDecide`
  - 当前实现直接调用 `s.decisionEngine.GenerateDecision()`，绕过 `ProposalService.GenerateProposals()`
  - 修改为调用 `s.decisionSvc.Decide()` (如果存在) 或在 GenerateDecision 后调用 GenerateProposals
  - 更新 `internal/mcp/interfaces.go` 中的 `DecisionService` 接口以包含 `Decide` 方法
  - TDD: 先写测试验证 decide 创建提案

  **Must NOT do**:
  - 不修改 DecisionEngine 的内部逻辑
  - 不绕过验证/修复/回退链

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 9, 10, 11)
  - **Blocks**: Tasks 12-15, 19
  - **Blocked By**: Tasks 4-7

  **References**:
  - `internal/mcp/tools_decision.go:handleDecide` — 当前实现 (绕过 ProposalService)
  - `internal/service/decision_service.go:DecisionService.Decide` — 完整生命周期编排
  - `internal/action/proposal_service.go:ProposalService.GenerateProposals` — 提案创建
  - `internal/decision/engine.go:DecisionEngine.GenerateDecision` — 决策生成
  - `internal/mcp/interfaces.go:DecisionService` — 接口定义

  **Acceptance Criteria**:
  - [ ] MCP `decide` 工具创建提案
  - [ ] 提案 apply_status 为 "proposed"
  - [ ] 案例状态更新为 "proposal_generated"
  - [ ] 测试通过

  **QA Scenarios**:
  ```
  Scenario: decide creates proposals
    Tool: bash (MCP stdio)
    Preconditions: 一个 created 状态的 case 存在
    Steps:
      1. 启动 MCP server
      2. 调用 create_decision_case(alert_id=...)
      3. 调用 decide(case_id=...)
      4. 验证返回 decision_type, severity, summary 等字段
      5. 调用 list_proposals(case_id=...)
      6. 验证返回 proposals 数组非空
      7. 验证每个 proposal 的 apply_status 为 "proposed"
    Expected Result: decide 创建可审核的提案
    Evidence: .sisyphus/evidence/task-8-decide-creates-proposals.json
  ```

  **Commit**: YES
  - Message: `fix(decision): MCP decide now creates proposals via DecisionService`
  - Files: `internal/mcp/tools_decision.go`, `internal/mcp/interfaces.go`, `cmd/baxi-mcp/main.go`
  - Pre-commit: `go test ./internal/mcp/... ./internal/decision/...`

---

- [x] 9. 添加案例写回汇聚点

  **What to do**:
  - 在 `internal/decision/case_service.go` 中添加 `ResolveCase` 方法
  - 实现逻辑: 检查所有提案是否已审核 (approved/rejected) → 更新案例状态为 "resolved" → 设置 resolved_at
  - 在 `internal/review/service.go` 的审核流程中调用 ResolveCase
  - 添加 MCP 工具 `resolve_case` 或在 approve/reject 后自动检查
  - TDD: 先写测试验证案例状态自动更新

  **Must NOT do**:
  - 不修改现有审核流程的接口
  - 不强制要求所有提案都 approved (只要有审核记录即可)

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 8, 10, 11)
  - **Blocks**: Tasks 12-15, 19
  - **Blocked By**: Tasks 4-7

  **References**:
  - `internal/decision/case_service.go:CaseService` — 案例服务
  - `internal/review/service.go:ReviewService.ApproveProposal` — 审核流程
  - `internal/review/repository.go:ReviewRepository.GetReviewsByProposal` — 获取审核记录
  - `internal/model/constants.go:CaseStatus` — 案例状态常量

  **Acceptance Criteria**:
  - [ ] ResolveCase 检查所有提案的审核状态
  - [ ] 案例状态自动更新为 "resolved"
  - [ ] resolved_at 时间戳被设置
  - [ ] 测试通过

  **QA Scenarios**:
  ```
  Scenario: case auto-resolves when all proposals reviewed
    Tool: bash (MCP stdio)
    Preconditions: 案例有 2 个提案，状态为 proposed
    Steps:
      1. 启动 MCP server
      2. 调用 approve_proposal(proposal_id=1, reviewer_id="user1")
      3. 调用 approve_proposal(proposal_id=2, reviewer_id="user1")
      4. 调用 get_case(case_id=...)
      5. 验证案例状态为 "resolved"
      6. 验证 resolved_at 非空
    Expected Result: 案例自动解决
    Evidence: .sisyphus/evidence/task-9-case-auto-resolve.json
  ```

  **Commit**: YES
  - Message: `feat(decision): add case resolution after all proposals reviewed`
  - Files: `internal/decision/case_service.go`, `internal/review/service.go`
  - Pre-commit: `go test ./internal/decision/... ./internal/review/...`

---

- [x] 10. 添加 cancel_proposal MCP 工具

  **What to do**:
  - 在 `internal/mcp/tools_review.go` 中添加 `cancel_proposal` MCP 工具
  - 调用 `ReviewService.CancelProposal(ctx, proposalID, reviewerID, feedback)`
  - 更新提案 apply_status 为 "cancelled"
  - 插入审核记录 (verdict=cancelled)
  - TDD: 先写测试验证取消流程

  **Must NOT do**:
  - 不修改 ReviewService.CancelProposal 的实现
  - 不允许取消已执行的提案

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 8, 9, 11)
  - **Blocks**: Tasks 12-15
  - **Blocked By**: Tasks 4-7

  **References**:
  - `internal/review/service.go:ReviewService.CancelProposal` — 已存在但未暴露
  - `internal/mcp/tools_review.go` — 参考 approve/reject 模式
  - `internal/review/domain.go:ReviewRecord` — 审核记录类型

  **Acceptance Criteria**:
  - [ ] `cancel_proposal` MCP 工具可用
  - [ ] 提案状态更新为 "cancelled"
  - [ ] 审核记录已插入
  - [ ] 测试通过

  **QA Scenarios**:
  ```
  Scenario: cancel_proposal works
    Tool: bash (MCP stdio)
    Preconditions: 一个 proposed 状态的提案存在
    Steps:
      1. 启动 MCP server
      2. 调用 cancel_proposal(proposal_id=..., reviewer_id="user1", feedback="不再需要")
      3. 验证返回 verdict="cancelled"
      4. 调用 list_proposals(case_id=...)
      5. 验证提案状态为 "cancelled"
    Expected Result: 取消成功
    Evidence: .sisyphus/evidence/task-10-cancel-proposal.json
  ```

  **Commit**: YES
  - Message: `feat(review): add cancel_proposal MCP tool`
  - Files: `internal/mcp/tools_review.go`
  - Pre-commit: `go test ./internal/mcp/... ./internal/review/...`

---

- [x] 11. 添加 get_proposal_by_id MCP 工具

  **What to do**:
  - 在 `internal/mcp/tools_review.go` 中添加 `get_proposal` MCP 工具
  - 调用 `ReviewService.GetProposalByID(ctx, proposalID)`
  - 返回提案的完整信息: id, case_id, action_type, title, description, risk_level, apply_status, created_at, applied_at 等
  - TDD: 先写测试验证查询

  **Must NOT do**:
  - 不修改 ReviewService.GetProposalByID 的实现

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 8, 9, 10)
  - **Blocks**: Tasks 12-15
  - **Blocked By**: Tasks 4-7

  **References**:
  - `internal/review/service.go:ReviewService.GetProposalByID` — 已存在但未暴露
  - `internal/mcp/tools_review.go` — 参考 approve/reject 模式

  **Acceptance Criteria**:
  - [ ] `get_proposal` MCP 工具可用
  - [ ] 返回完整提案信息
  - [ ] 测试通过

  **QA Scenarios**:
  ```
  Scenario: get_proposal returns full details
    Tool: bash (MCP stdio)
    Preconditions: 一个提案存在
    Steps:
      1. 启动 MCP server
      2. 调用 get_proposal(proposal_id=...)
      3. 验证返回 id, case_id, action_type, title, description
      4. 验证返回 risk_level, apply_status, created_at
    Expected Result: 返回完整提案
    Evidence: .sisyphus/evidence/task-11-get-proposal.json
  ```

  **Commit**: YES
  - Message: `feat(review): add get_proposal MCP tool`
  - Files: `internal/mcp/tools_review.go`
  - Pre-commit: `go test ./internal/mcp/... ./internal/review/...`

---

---

### Wave 4: Review Enhancement — 审核工作流增强

- [x] 12. 添加 list_review_records MCP 工具

  **What to do**:
  - 在 `internal/mcp/tools_review.go` 中添加 `list_review_records` MCP 工具
  - 调用 `ReviewRepository.GetReviewsByProposal(ctx, proposalID)`
  - 返回审核记录列表: id, proposal_id, verdict, reviewer_id, feedback, created_at
  - TDD: 先写测试验证查询

  **Must NOT do**:
  - 不修改 ReviewRepository.GetReviewsByProposal 的实现

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 4 (with Tasks 13, 14, 15)
  - **Blocks**: Tasks 16-20
  - **Blocked By**: Tasks 8-11

  **References**:
  - `internal/review/repository.go:ReviewRepository.GetReviewsByProposal` — 已存在但未暴露
  - `internal/review/domain.go:ReviewRecord` — 审核记录类型
  - `internal/mcp/tools_review.go` — 参考 approve/reject 模式

  **Acceptance Criteria**:
  - [ ] `list_review_records` MCP 工具可用
  - [ ] 返回审核记录列表
  - [ ] 测试通过

  **QA Scenarios**:
  ```
  Scenario: list_review_records returns history
    Tool: bash (MCP stdio)
    Preconditions: 提案有审核记录
    Steps:
      1. 启动 MCP server
      2. 调用 list_review_records(proposal_id=...)
      3. 验证返回 records 数组
      4. 验证每个 record 有 id, verdict, reviewer_id, feedback
    Expected Result: 返回审核历史
    Evidence: .sisyphus/evidence/task-12-list-reviews.json
  ```

  **Commit**: YES
  - Message: `feat(review): add list_review_records MCP tool`
  - Files: `internal/mcp/tools_review.go`
  - Pre-commit: `go test ./internal/mcp/... ./internal/review/...`

---

- [x] 13. 创建行动类型模式目录

  **What to do**:
  - 创建 `internal/action/schema.go` 文件
  - 实现 `ActionDefinition` 结构体: Type, InputSchema (JSON Schema), RiskLevel, Prerequisites, SideEffects, AuditCategory
  - 从 `config/action_registry.yml` 加载配置
  - 添加 JSON Schema 验证方法
  - 更新 `action/registry.go` 以使用 ActionDefinition
  - TDD: 先写测试验证模式加载和验证

  **Must NOT do**:
  - 不修改现有 ActionRegistry 的接口
  - 不硬编码风险级别

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 4 (with Tasks 12, 14, 15)
  - **Blocks**: Tasks 16-20
  - **Blocked By**: Tasks 8-11

  **References**:
  - `internal/action/registry.go:ActionRegistry` — 现有注册表
  - `config/action_registry.yml` — 动作配置
  - `internal/llm/schema_validator.go` — 参考 JSON Schema 验证模式

  **Acceptance Criteria**:
  - [ ] ActionDefinition 结构体定义完整
  - [ ] 从 YAML 加载配置
  - [ ] JSON Schema 验证工作
  - [ ] 测试通过

  **QA Scenarios**:
  ```
  Scenario: ActionDefinition loads from YAML
    Tool: bash (go test)
    Preconditions: action_registry.yml 有 4 种动作类型
    Steps:
      1. 加载配置
      2. 验证返回 4 个 ActionDefinition
      3. 验证每个有 Type, InputSchema, RiskLevel
      4. 验证 JSON Schema 验证方法工作
    Expected Result: 模式目录完整
    Evidence: .sisyphus/evidence/task-13-action-schema.json
  ```

  **Commit**: YES
  - Message: `feat(action): add ActionDefinition schema catalog`
  - Files: `internal/action/schema.go`, `internal/action/registry.go`, `config/action_registry.yml`
  - Pre-commit: `go test ./internal/action/...`

---

- [x] 14. 实现风险自适应 HITL

  **What to do**:
  - 在 `internal/review/service.go` 中添加风险评估逻辑
  - 根据 ActionProposal 的 risk_level 决定审核流程:
    - low: 自动批准 (跳过人工审核)
    - medium: 单一审核者
    - high: 需要 2 名审核者
    - critical: 需要副总裁级审核 + 人工执行
  - 添加配置文件 `config/risk_adaptive_hitl.yml`
  - TDD: 先写测试验证风险路由

  **Must NOT do**:
  - 不绕过治理策略检查
  - 不硬编码风险级别

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 4 (with Tasks 12, 13, 15)
  - **Blocks**: Tasks 16-20
  - **Blocked By**: Tasks 8-11

  **References**:
  - `internal/review/service.go:ReviewService` — 审核服务
  - `internal/action/schema.go:ActionDefinition` — 行动定义 (Task 13)
  - `config/access_policy.yml` — 访问策略配置

  **Acceptance Criteria**:
  - [ ] 低风险操作自动批准
  - [ ] 中风险操作需要单一审核者
  - [ ] 高风险操作需要 2 名审核者
  - [ ] 关键操作需要副总裁级审核
  - [ ] 配置可驱动
  - [ ] 测试通过

  **QA Scenarios**:
  ```
  Scenario: low risk auto-approves
    Tool: bash (MCP stdio)
    Preconditions: notify_owner 动作 risk_level=low
    Steps:
      1. 启动 MCP server
      2. 创建案例并生成决策
      3. 验证低风险提案自动批准
    Expected Result: 无需人工审核
    Evidence: .sisyphus/evidence/task-14-auto-approve.json

  Scenario: high risk requires 2 reviewers
    Tool: bash (MCP stdio)
    Preconditions: cancel_order 动作 risk_level=high
    Steps:
      1. 启动 MCP server
      2. 创建案例并生成决策
      3. 调用 approve_proposal(reviewer_id="user1")
      4. 验证提案状态仍为 "proposed" (需要第二个审核者)
      5. 调用 approve_proposal(reviewer_id="user2")
      6. 验证提案状态变为 "approved"
    Expected Result: 需要 2 名审核者
    Evidence: .sisyphus/evidence/task-14-two-reviewers.json
  ```

  **Commit**: YES
  - Message: `feat(review): implement risk-adaptive HITL`
  - Files: `internal/review/service.go`, `config/risk_adaptive_hitl.yml`
  - Pre-commit: `go test ./internal/review/...`

---

- [x] 15. 创建持久化提案沙箱

  **What to do**:
  - 创建 `internal/review/sandbox.go` 文件
  - 实现 `ProposalSandbox` 结构体: SandboxID, CaseID, Proposals[], Status, CreatedAt
  - 添加数据库表 `ai.proposal_sandbox` 和迁移
  - 实现沙箱操作: CreateSandbox, AddProposal, CompareProposals, MergeToCase
  - 支持提案比较: 并排显示多个提案的差异
  - TDD: 先写测试验证沙箱操作

  **Must NOT do**:
  - 不修改现有提案表结构
  - 不支持跨案例的沙箱

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 4 (with Tasks 12, 13, 14)
  - **Blocks**: Tasks 16-20
  - **Blocked By**: Tasks 8-11

  **References**:
  - `internal/review/domain.go` — 审核领域类型
  - `internal/review/repository.go` — 数据访问
  - `migrations/` — 参考现有迁移模式

  **Acceptance Criteria**:
  - [ ] ProposalSandbox 结构体定义完整
  - [ ] 数据库迁移成功
  - [ ] CreateSandbox, AddProposal, CompareProposals, MergeToCase 操作工作
  - [ ] 测试通过

  **QA Scenarios**:
  ```
  Scenario: sandbox supports proposal comparison
    Tool: bash (go test)
    Preconditions: 案例有 3 个提案
    Steps:
      1. 创建沙箱
      2. 添加 3 个提案到沙箱
      3. 调用 CompareProposals
      4. 验证返回差异视图
      5. 选择一个提案并 MergeToCase
      6. 验证案例状态更新
    Expected Result: 沙箱支持比较和选择
    Evidence: .sisyphus/evidence/task-15-sandbox-compare.json
  ```

  **Commit**: YES
  - Message: `feat(review): add persistent proposal sandbox`
  - Files: `internal/review/sandbox.go`, `migrations/xxx_add_proposal_sandbox.sql`
  - Pre-commit: `go test ./internal/review/...`

---

### Wave 5: AIP Features — 高级特性

- [x] 16. 实现 OAG 链接遍历

  **What to do**:
  - 创建 `internal/decision/context_builder_v3.go` 文件
  - 实现本体论增强生成 (OAG): 给定一个告警，遍历链接找到相关的订单、卖家、产品
  - 调用 `ontology.ObjectQueryService` 的链接遍历方法
  - 将遍历结果注入 LLM 上下文
  - 添加深度限制 (最多 3 跳)
  - TDD: 先写测试验证链接遍历

  **Must NOT do**:
  - 不支持深度超过 3 的链接遍历
  - 不暴露 LLMReadable=false 的属性

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 5 (with Tasks 17, 18, 19, 20)
  - **Blocks**: F1-F4
  - **Blocked By**: Tasks 12-15

  **References**:
  - `internal/decision/context_builder_v2.go` — 现有上下文构建器
  - `internal/ontology/query_service.go` — 链接遍历方法
  - `internal/ontology/schema.go:ObjectLink` — 链接定义
  - `internal/llm/provider.go:LLMSafeContext` — LLM 上下文类型

  **Acceptance Criteria**:
  - [ ] OAG 链接遍历工作
  - [ ] 深度限制 3 跳
  - [ ] LLM 上下文包含遍历结果
  - [ ] 测试通过

  **QA Scenarios**:
  ```
  Scenario: OAG traverses alert to order to seller
    Tool: bash (go test)
    Preconditions: 告警链接到订单，订单链接到卖家
    Steps:
      1. 构建决策上下文
      2. 验证上下文包含告警、订单、卖家信息
      3. 验证链接关系正确
    Expected Result: 上下文包含完整链路
    Evidence: .sisyphus/evidence/task-16-oag-traversal.json
  ```

  **Commit**: YES
  - Message: `feat(decision): implement OAG link traversal in context builder`
  - Files: `internal/decision/context_builder_v3.go`
  - Pre-commit: `go test ./internal/decision/... ./internal/ontology/...`

---

- [x] 17. 添加前端决策审核页面

  **What to do**:
  - 创建 `frontend/src/pages/DecisionReview.tsx` 页面
  - 实现功能: 案例列表、决策详情、提案列表、审核操作 (approve/reject/cancel)
  - 添加路由 `/decision-review`
  - 使用 TanStack Query 获取数据
  - 使用 Radix UI 组件
  - TDD: 先写测试验证页面渲染

  **Must NOT do**:
  - 不修改现有页面的路由
  - 不引入新的状态管理库

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 5 (with Tasks 16, 18, 19, 20)
  - **Blocks**: F1-F4
  - **Blocked By**: Tasks 12-15

  **References**:
  - `frontend/src/pages/Alerts.tsx` — 参考现有页面模式
  - `frontend/src/api/client.ts` — API 客户端
  - `frontend/src/components/` — 现有组件
  - `frontend/src/router.tsx` — 路由配置

  **Acceptance Criteria**:
  - [ ] 决策审核页面可访问
  - [ ] 案例列表显示
  - [ ] 决策详情显示
  - [ ] 审核操作可用
  - [ ] 测试通过

  **QA Scenarios**:
  ```
  Scenario: decision review page loads
    Tool: Playwright
    Preconditions: 有案例和提案数据
    Steps:
      1. 导航到 /decision-review
      2. 验证案例列表显示
      3. 点击案例查看详情
      4. 验证提案列表显示
      5. 点击 approve 按钮
      6. 验证审核成功
    Expected Result: 页面功能完整
    Evidence: .sisyphus/evidence/task-17-decision-review.png
  ```

  **Commit**: YES
  - Message: `feat(frontend): add decision review page`
  - Files: `frontend/src/pages/DecisionReview.tsx`, `frontend/src/router.tsx`
  - Pre-commit: `cd frontend && npm test`

---

- [x] 18. 添加前端沙箱比较页面

  **What to do**:
  - 创建 `frontend/src/pages/ProposalSandbox.tsx` 页面
  - 实现功能: 沙箱列表、提案比较视图、选择和合并操作
  - 添加路由 `/proposal-sandbox`
  - 使用 TanStack Query 获取数据
  - 使用 Radix UI 组件
  - TDD: 先写测试验证页面渲染

  **Must NOT do**:
  - 不修改现有页面的路由
  - 不引入新的状态管理库

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 5 (with Tasks 16, 17, 19, 20)
  - **Blocks**: F1-F4
  - **Blocked By**: Tasks 12-15

  **References**:
  - `frontend/src/pages/DecisionReview.tsx` — 参考 (Task 17)
  - `frontend/src/api/client.ts` — API 客户端
  - `frontend/src/components/` — 现有组件

  **Acceptance Criteria**:
  - [ ] 沙箱比较页面可访问
  - [ ] 提案并排比较视图
  - [ ] 选择和合并操作可用
  - [ ] 测试通过

  **QA Scenarios**:
  ```
  Scenario: sandbox comparison page loads
    Tool: Playwright
    Preconditions: 沙箱有多个提案
    Steps:
      1. 导航到 /proposal-sandbox
      2. 验证沙箱列表显示
      3. 点击沙箱查看详情
      4. 验证提案并排比较视图
      5. 选择一个提案并合并
      6. 验证合并成功
    Expected Result: 沙箱功能完整
    Evidence: .sisyphus/evidence/task-18-sandbox-compare.png
  ```

  **Commit**: YES
  - Message: `feat(frontend): add proposal sandbox comparison page`
  - Files: `frontend/src/pages/ProposalSandbox.tsx`, `frontend/src/router.tsx`
  - Pre-commit: `cd frontend && npm test`

---

- [x] 19. 集成测试 — 决策生命周期

  **What to do**:
  - 创建 `test/integration/decision_lifecycle_test.go` 文件
  - 实现端到端测试: 创建案例 → 生成决策 → 创建提案 → 人工审核 → 写回执行
  - 使用 testcontainers-go 管理 PostgreSQL
  - 验证所有状态转换和数据库记录
  - TDD: 先写测试框架

  **Must NOT do**:
  - 不修改现有集成测试
  - 不使用 mock (使用真实数据库)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 5 (with Tasks 16, 17, 18, 20)
  - **Blocks**: F1-F4
  - **Blocked By**: Tasks 12-15

  **References**:
  - `test/integration/phase7_test.go` — 参考现有集成测试模式
  - `internal/testutil/` — 测试工具
  - `migrations/` — 数据库迁移

  **Acceptance Criteria**:
  - [ ] 端到端测试通过
  - [ ] 所有状态转换正确
  - [ ] 数据库记录完整
  - [ ] 测试覆盖完整生命周期

  **QA Scenarios**:
  ```
  Scenario: full decision lifecycle
    Tool: bash (go test)
    Preconditions: 数据库已初始化
    Steps:
      1. 创建案例
      2. 生成决策
      3. 验证提案已创建
      4. 审核提案
      5. 执行提案
      6. 验证案例状态为 resolved
      7. 验证 outbox 事件已创建
    Expected Result: 完整生命周期通过
    Evidence: .sisyphus/evidence/task-19-lifecycle-test.log
  ```

  **Commit**: YES
  - Message: `test(integration): add decision lifecycle integration test`
  - Files: `test/integration/decision_lifecycle_test.go`
  - Pre-commit: `go test -tags integration ./test/integration/...`

---

- [x] 20. 集成测试 — Ontology MCP

  **What to do**:
  - 创建 `test/integration/ontology_mcp_test.go` 文件
  - 实现端到端测试: describe_ontology, get_object, get_linked_objects, execute_action
  - 使用 testcontainers-go 管理 PostgreSQL
  - 验证所有 MCP 工具的输入输出
  - TDD: 先写测试框架

  **Must NOT do**:
  - 不修改现有集成测试
  - 不使用 mock (使用真实数据库)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 5 (with Tasks 16, 17, 18, 19)
  - **Blocks**: F1-F4
  - **Blocked By**: Tasks 12-15

  **References**:
  - `test/integration/phase7_test.go` — 参考现有集成测试模式
  - `internal/mcp/tools_ontology.go` — MCP 工具 (Task 4-7)
  - `internal/ontology/` — 本体论包

  **Acceptance Criteria**:
  - [ ] 所有 Ontology MCP 工具测试通过
  - [ ] 输入输出验证正确
  - [ ] 错误处理测试通过

  **QA Scenarios**:
  ```
  Scenario: ontology MCP tools work end-to-end
    Tool: bash (go test)
    Preconditions: 数据库有本体论数据
    Steps:
      1. 调用 describe_ontology
      2. 验证返回所有对象类型
      3. 调用 get_object(type=order, id=ORD-001)
      4. 验证返回完整对象
      5. 调用 get_linked_objects(type=order, id=ORD-001)
      6. 验证返回关联对象
      7. 调用 execute_action(type=order, action=notify_owner)
      8. 验证执行成功
    Expected Result: 所有工具工作正常
    Evidence: .sisyphus/evidence/task-20-ontology-mcp-test.log
  ```

  **Commit**: YES
  - Message: `test(integration): add ontology MCP integration test`
  - Files: `test/integration/ontology_mcp_test.go`
  - Pre-commit: `go test -tags integration ./test/integration/...`

---

> 4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.

- [ ] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, curl endpoint, run command). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist in .sisyphus/evidence/. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [ ] F2. **Code Quality Review** — `unspecified-high`
  Run `go test ./...` + `golangci-lint`. Review all changed files for: `as any`/`@ts-ignore`, empty catches, console.log in prod, commented-out code, unused imports. Check AI slop: excessive comments, over-abstraction, generic names (data/result/item/temp).
  Output: `Build [PASS/FAIL] | Lint [PASS/FAIL] | Tests [N pass/N fail] | Files [N clean/N issues] | VERDICT`

- [ ] F3. **Real Manual QA** — `unspecified-high`
  Start from clean state. Execute EVERY QA scenario from EVERY task — follow exact steps, capture evidence. Test cross-task integration (features working together, not isolation). Test edge cases: empty state, invalid input, rapid actions. Save to `.sisyphus/evidence/final-qa/`.
  Output: `Scenarios [N/N pass] | Integration [N/N] | Edge Cases [N tested] | VERDICT`

- [ ] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual diff (git log/diff). Verify 1:1 — everything in spec was built (no missing), nothing beyond spec was built (no creep). Check "Must NOT do" compliance. Detect cross-task contamination: Task N touching Task M's files. Flag unaccounted changes.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

- **Wave 1**: `fix(mcp): connect stub implementations` — tools_status.go, tools_action.go, cmd/baxi-mcp/main.go
- **Wave 2**: `feat(mcp): add ontology tools` — tools_ontology.go, interfaces.go, server.go
- **Wave 3**: `fix(decision): complete lifecycle with proposal creation` — tools_decision.go, case_service.go
- **Wave 4**: `feat(review): enhance workflow with sandbox and risk-adaptive HITL` — tools_review.go, sandbox.go, schema.go
- **Wave 5**: `feat(aip): OAG traversal and frontend review UI` — context_builder_v3.go, frontend pages

---

## Success Criteria

### Verification Commands
```bash
make test                    # Expected: all tests pass
make build                   # Expected: builds without errors
go run ./cmd/baxi-mcp       # Expected: MCP server starts, responds to tool calls
curl http://localhost:8080/api/health  # Expected: 200 OK
```

### Final Checklist
- [ ] All "Must Have" present
- [ ] All "Must NOT Have" absent
- [ ] All MCP tools respond correctly
- [ ] Decision lifecycle complete end-to-end
- [ ] All tests pass (TDD)
- [ ] No stub implementations remaining
- [ ] Action type schema catalog exposed via MCP
- [ ] Risk-adaptive HITL configurable
- [ ] Persistent sandbox functional
- [ ] OAG link traversal working
