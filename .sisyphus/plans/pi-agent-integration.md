# Pi Agent 完整集成计划

## TL;DR

> **目标**: 修复所有已知 Bug，完善 MCP 工具，实现 Pi Agent 主动预警和完整闭环
> 
> **交付物**:
> - 修复 4 个 P0 Bug
> - 新增 8 个 MCP 工具
> - 创建 Pi 扩展实现主动预警
> - 完整端到端测试
> 
> **工作量**: 3-4 周
> **并行执行**: YES - 3 个 Wave

---

## Context

### 原始需求
用户希望实现 Pi Agent 与 Baxi 平台的完整集成，包括：
1. 修复现有 Bug
2. 完善 MCP 工具
3. 实现主动预警唤起
4. 实现项目状态了解能力
5. 端到端闭环测试

### 研究发现

**Pi 信息**:
- 版本: 0.76.0
- 安装: `/home/zzz/.npm-global/bin/pi`
- 配置: `~/.pi/agent/mcp.json`
- 已配置 9 个 directTools

**Pi MCP 集成最佳实践** (来源: 网络搜索):
1. **pi-mcp-adapter** (nicobailon): 最流行的适配器，使用代理工具模式（~200 tokens），支持 directTools
2. **directTools 模式**: 将常用工具提升为一等 Pi 工具，每个 ~150-300 tokens
3. **lazy 连接**: 默认按需连接，空闲 10 分钟后断开
4. **工具元数据缓存**: 缓存到 `~/.pi/agent/mcp-cache.json`，无需连接即可搜索
5. **环境变量插值**: 支持 `${VAR}` 在配置中解析环境变量

**Go MCP SDK 选项**:
- **mark3labs/mcp-go**: Baxi 当前使用的库，功能完整
- **官方 Go SDK**: `github.com/modelcontextprotocol/go-sdk` - 由 Google 协作维护，更活跃
- **paularlott/mcp**: 另一个流行实现，支持工具发现和并行调用

**现有 MCP 集成**:
- Server: `cmd/baxi-mcp/main.go` + `internal/mcp/`
- 传输: stdio
- 工具: 9 个 directTools (create_decision_case, decide, list_cases, get_case, list_alerts, list_proposals, check_access, get_classification, run_pipeline)

**已知 Bug**:
- P0: Migration 缺列、白名单语义、Executor 缺失、Schema 漂移
- P1: 前端未连 Go API、CLI 硬编码、501 端点、LLM 配置未生效

### 优化建议 (基于研究)

1. **工具数量控制**: 当前 9 个 directTools 已接近推荐上限 (5-20 个)，新增工具应谨慎选择
2. **使用代理模式**: 对于低频工具，考虑使用 mcp 代理工具而非 directTools
3. **考虑官方 Go SDK**: 长期可考虑迁移到官方 SDK，获得更好的维护支持
4. **添加工具描述**: 每个工具应有清晰的描述，帮助 Pi 理解何时使用

---

## Work Objectives

### 核心目标
实现 Pi Agent 与 Baxi 的完整闭环：预警 → 决策 → 审批 → 执行 → 反馈

### 具体交付物

**Wave 1: Bug 修复**
1. 修复 Migration 005 - 添加 next_retry_at 列
2. 修复 Action 白名单 - 空配置应禁用所有动作
3. 注入真实 Action Executor
4. 修复 Ontology Schema 漂移

**Wave 2: MCP 工具完善**
5. 添加 approve_proposal 工具
6. 添加 reject_proposal 工具
7. 添加 execute_proposal 工具
8. 添加 get_decision_context 工具
9. 添加 list_outbox_events 工具
10. 添加 get_pipeline_status 工具
11. 添加 get_system_status 工具
12. 添加 search_objects 工具
13. **优化 MCP 配置** - 使用混合模式 (directTools + 代理)

**Wave 3: Pi 扩展开发**
14. 创建 Baxi 扩展
15. 实现主动预警检测
16. 实现项目状态查询
17. 端到端测试

### Must Have
- 所有 P0 Bug 修复
- 审批工具 (approve/reject)
- 执行工具 (execute)
- 主动预警机制

### Must NOT Have
- 不实现 Row-Level Access Control
- 不实现 Multi-Organization Silos
- 不实现 Policy-as-Code (Cedar/OPA)
- 不修改 Pi 核心代码

---

## Verification Strategy

### 测试决策
- **基础设施**: 已有 Go test + testcontainers
- **测试策略**: TDD for new tools, Tests-after for bug fixes
- **框架**: go test + testify

### QA Policy
每个任务必须有可执行的验证场景

---

## Execution Strategy

### 并行执行波次

```
Wave 1 (Bug 修复 - 并行):
├── Task 1: 修复 Migration 005 [quick]
├── Task 2: 修复 Action 白名单 [quick]
├── Task 3: 注入 Action Executor [unspecified-high]
└── Task 4: 修复 Schema 漂移 [quick]

Wave 2 (MCP 工具 + 配置 - 并行):
├── Task 5-6: 审批工具 [quick]
├── Task 7-8: 执行和上下文工具 [quick]
├── Task 9-10: Outbox 和管道工具 [quick]
├── Task 11-12: 状态和搜索工具 [quick]
└── Task 13: 优化 MCP 配置 [quick]

Wave 3 (Pi 扩展 - 依赖 Wave 1+2):
├── Task 13: 优化 MCP 配置 [quick]
├── Task 14: 创建 Baxi 扩展 [unspecified-high]
├── Task 15: 实现主动预警 [unspecified-high]
├── Task 16: 实现状态查询 [unspecified-high]
└── Task 17: 端到端测试 [unspecified-high]

Wave FINAL (验证):
├── Task F1: 计划合规审计 [oracle]
├── Task F2: 代码质量审查 [unspecified-high]
├── Task F3: 真实 QA 测试 [unspecified-high]
└── Task F4: 范围保真检查 [deep]
```

### 依赖矩阵

| Task | 依赖 | 被依赖 |
|------|------|--------|
| 1-4 | 无 | 5-13, 14-17 |
| 5-12 | 1-4 | 13-17 |
| 13 | 5-12 | 14-17 |
| 14-17 | 13 | F1-F4 |

---

## TODOs

- [x] 1. 修复 Migration 005 - 添加 next_retry_at 列

  **What to do**:
  - 创建新迁移文件 `028_add_outbox_next_retry_at.sql`
  - 添加 `next_retry_at TIMESTAMPTZ` 列到 `ops.outbox_events`
  - 更新 Repository 查询使用新列
  - 运行迁移验证

  **Must NOT do**:
  - 不修改现有迁移文件
  - 不改变现有数据

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2, 3, 4)
  - **Blocks**: Tasks 5-17
  - **Blocked By**: None

  **References**:
  - `migrations/005_ops_tables.sql` - 现有 outbox_events 表结构
  - `internal/repository/outbox_repository.go` - 查询 next_retry_at 的代码
  - `test/integration/phase7_test.go:454` - 失败的测试

  **Acceptance Criteria**:
  - [ ] 迁移文件创建
  - [ ] `go test ./test/integration/... -run TestPhase7_WorkerDispatch` 通过

  **QA Scenarios**:

  ```
  Scenario: 迁移成功
    Tool: Bash
    Steps:
      1. make migrate
      2. psql -c "SELECT next_retry_at FROM ops.outbox_events LIMIT 1"
    Expected Result: 列存在，查询成功
    Evidence: .sisyphus/evidence/task-1-migration.txt

  Scenario: 集成测试通过
    Tool: Bash
    Steps:
      1. go test ./test/integration/... -run TestPhase7_WorkerDispatch -v
    Expected Result: PASS
    Evidence: .sisyphus/evidence/task-1-test.txt
  ```

  **Commit**: YES
  - Message: `fix(migration): add next_retry_at to outbox_events`
  - Files: `migrations/028_*.sql`, `internal/repository/outbox_repository.go`

---

- [x] 2. 修复 Action 白名单 - 空配置应禁用所有动作

  **What to do**:
  - 修改 `internal/action/registry.go` 的 `whitelistActions()`
  - 空配置 `{}` 应返回空白名单
  - 缺省配置才加载所有规范动作
  - 添加单元测试

  **Must NOT do**:
  - 不改变 CanonicalActions 定义
  - 不影响现有配置文件

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 3, 4)
  - **Blocks**: Tasks 5-17
  - **Blocked By**: None

  **References**:
  - `internal/action/registry.go:101-145` - whitelistActions() 逻辑
  - `internal/action/registry_test.go` - 现有测试
  - `test/integration/phase7_test.go` - TestPhase7_Whitelist_NonWhitelistedAction

  **Acceptance Criteria**:
  - [ ] 空配置返回空 whitelist
  - [ ] 缺省配置返回所有 canonical actions
  - [ ] 单元测试通过

  **QA Scenarios**:

  ```
  Scenario: 空配置禁用所有动作
    Tool: Bash
    Steps:
      1. 创建临时配置文件 actions: {}
      2. NewActionRegistry(path)
      3. reg.AllowedActions()
    Expected Result: 返回空切片
    Evidence: .sisyphus/evidence/task-2-empty.txt

  Scenario: 集成测试通过
    Tool: Bash
    Steps:
      1. go test ./test/integration/... -run TestPhase7_Whitelist -v
    Expected Result: PASS
    Evidence: .sisyphus/evidence/task-2-test.txt
  ```

  **Commit**: YES
  - Message: `fix(action): empty config should disable all actions`
  - Files: `internal/action/registry.go`, `internal/action/registry_test.go`

---

- [x] 3. 注入真实 Action Executor

  **What to do**:
  - 修改 `internal/api/server.go` 的 `actionHandler()` 工厂
  - 注入真实的 Feishu/GitHub Executor
  - 修复 `handleExecute` 返回 OutboxEventID
  - 添加集成测试

  **Must NOT do**:
  - 不改变 ActionExecutor 接口
  - 不影响 dry-run 模式

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2, 4)
  - **Blocks**: Tasks 5-17
  - **Blocked By**: None

  **References**:
  - `internal/api/handler_factories.go` - actionHandler() 工厂
  - `internal/action/executor.go` - ActionExecutor 接口
  - `internal/action/apply_service.go` - ApplyService 实现
  - `internal/adapter/` - Feishu/GitHub 适配器

  **Acceptance Criteria**:
  - [ ] API 执行动作时使用真实 Executor
  - [ ] 返回 OutboxEventID
  - [ ] 集成测试通过

  **QA Scenarios**:

  ```
  Scenario: API 执行动作
    Tool: curl
    Steps:
      1. 创建 proposal
      2. curl -X POST /api/v1/proposals/{id}/execute
      3. 检查响应包含 outbox_event_id
    Expected Result: 200 OK, 包含 outbox_event_id
    Evidence: .sisyphus/evidence/task-3-execute.json

  Scenario: dry-run 模式
    Tool: curl
    Steps:
      1. 设置 ACTION_APPLY_DRY_RUN=true
      2. 执行动作
      3. 验证未实际发送
    Expected Result: 200 OK, 标记为 dry-run
    Evidence: .sisyphus/evidence/task-3-dryrun.json
  ```

  **Commit**: YES
  - Message: `fix(action): inject real executors and return OutboxEventID`
  - Files: `internal/api/server.go`, `internal/api/handler_factories.go`

---

- [x] 4. 修复 Ontology Schema 漂移

  **What to do**:
  - 对比 Migration 列名与 Repository 查询列名
  - 修复不匹配的列名
  - 更新 Repository 查询
  - 添加 Schema 契约测试

  **Must NOT do**:
  - 不修改 Migration (只改查询)
  - 不添加新列

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2, 3)
  - **Blocks**: Tasks 5-17
  - **Blocked By**: None

  **References**:
  - `migrations/003_dwd_tables.sql` - DWD 表定义
  - `internal/repository/ontology_repository.go` - 查询代码
  - `test/migration/contract_test.go` - Schema 契约测试

  **Acceptance Criteria**:
  - [ ] 所有查询列名与 Migration 匹配
  - [ ] Schema 契约测试通过
  - [ ] Ontology 查询成功

  **QA Scenarios**:

  ```
  Schema 对齐验证
    Tool: Bash
    Steps:
      1. go test ./test/migration/... -run TestContract -v
      2. go test ./internal/ontology/... -v
    Expected Result: PASS
    Evidence: .sisyphus/evidence/task-4-schema.txt
  ```

  **Commit**: YES
  - Message: `fix(ontology): align repository queries with migration schema`
  - Files: `internal/repository/ontology_repository.go`

---

- [x] 5-6. 添加审批工具 (approve_proposal, reject_proposal)
- [x] 7-8. 添加执行和上下文工具
- [x] 9-10. 添加 Outbox 和管道工具
- [x] 11-12. 添加状态和搜索工具

  **What to do**:
  - 实现 get_system_status 工具
  - 实现 search_objects 工具
  - 注册到 MCP server
  - 添加测试

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2
  - **Blocks**: Tasks 14-17
  - **Blocked By**: Tasks 1-4

  **References**:
  - `internal/repository/status_repository.go` - 状态查询
  - `internal/ontology/query_service.go` - 对象搜索

  **Commit**: YES
  - Message: `feat(mcp): add get_system_status and search_objects tools`
  - Files: `internal/mcp/tools_status.go`, `internal/mcp/server.go`

---

- [x] 13. 优化 MCP 配置 - 混合模式

  **What to do**:
  - 更新 `~/.pi/agent/mcp.json` 配置
  - 将高频工具设为 directTools (approve, reject, decide, list_alerts)
  - 低频工具使用 mcp 代理工具
  - 添加工具描述优化
  - 测试配置加载

  **Why**: 当前 9 个 directTools 已接近推荐上限，新增工具会消耗过多上下文 tokens

  **配置示例**:
  ```json
  {
    "settings": {
      "toolPrefix": "server",
      "idleTimeout": 10,
      "directTools": [
        "approve_proposal",
        "reject_proposal", 
        "decide",
        "list_alerts",
        "get_case",
        "create_decision_case"
      ]
    },
    "mcpServers": {
      "baxi": {
        "command": "/tmp/baxi-mcp",
        "env": {
          "DATABASE_URL": "${DATABASE_URL}",
          "API_BEARER_TOKEN": "${API_BEARER_TOKEN}"
        },
        "lifecycle": "lazy"
      }
    }
  }
  ```

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2
  - **Blocks**: Tasks 14-17
  - **Blocked By**: Tasks 1-13

  **References**:
  - `~/.pi/agent/mcp.json` - 现有配置
  - pi-mcp-adapter 文档 - directTools 最佳实践

  **Acceptance Criteria**:
  - [ ] 配置文件更新
  - [ ] Pi 启动时加载配置
  - [ ] directTools 可用
  - [ ] 代理工具可搜索

  **QA Scenarios**:

  ```
  配置加载测试
    Tool: pi
    Steps:
      1. pi -p "列出可用工具"
      2. 验证 directTools 出现
      3. pi -p "搜索 MCP 工具"
      4. 验证代理工具可发现
    Expected Result: 工具列表正确
    Evidence: .sisyphus/evidence/task-13-config.txt
  ```

  **Commit**: NO (Pi 配置文件)

---

- [x] 14. 创建 Baxi 扩展

  **What to do**:
  - 创建 `~/.pi/agent/extensions/baxi.ts`
  - 实现 session_start 事件 - 加载项目状态
  - 注册 /baxi-status 命令
  - 注册 /baxi-alerts 命令
  - 测试扩展加载

  **Must NOT do**:
  - 不修改 Pi 核心代码
  - 不影响其他扩展

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 14, 15, 16)
  - **Blocks**: None
  - **Blocked By**: Tasks 1-13

  **References**:
  - `docs/mcp-integration/pi-agent-framework.md` - 扩展开发指南
  - `~/.pi/agent/mcp.json` - 现有 MCP 配置

  **Acceptance Criteria**:
  - [ ] 扩展文件创建
  - [ ] Pi 启动时加载扩展
  - [ ] /baxi-status 命令可用
  - [ ] /baxi-alerts 命令可用

  **QA Scenarios**:

  ```
  扩展加载测试
    Tool: pi
    Steps:
      1. pi --extension ~/.pi/agent/extensions/baxi.ts -p "列出可用命令"
      2. 验证 /baxi-status 和 /baxi-alerts 出现
    Expected Result: 命令列表包含 baxi 命令
    Evidence: .sisyphus/evidence/task-13-extension.txt
  ```

  **Commit**: NO (Pi 配置文件)

---

- [x] 14. 实现主动预警检测

  **What to do**:
  - 在扩展中实现定时轮询 (session_start 时启动)
  - 检测新告警 (severity >= high)
  - 注入告警上下文到对话
  - 实现可配置的轮询间隔

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3
  - **Blocks**: None
  - **Blocked By**: Tasks 1-13

  **References**:
  - `internal/alert/engine.go` - 告警引擎
  - Pi Extension API - pi.on(), pi.sendMessage()

  **Acceptance Criteria**:
  - [ ] 定时轮询新告警
  - [ ] 高严重度告警触发通知
  - [ ] 告警上下文注入对话

  **QA Scenarios**:

  ```
  主动预警测试
    Tool: pi
    Steps:
      1. 创建高严重度告警
      2. 等待轮询间隔
      3. 验证 Pi 通知
    Expected Result: Pi 主动通知告警
    Evidence: .sisyphus/evidence/task-14-alert.txt
  ```

  **Commit**: NO

---

- [x] 15. 实现项目状态查询

  **What to do**:
  - 实现 /baxi-status 命令处理
  - 查询系统状态 (表计数、最近管道运行、告警统计)
  - 格式化输出
  - 实现 /baxi-case <id> 命令查看案例详情

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3
  - **Blocks**: None
  - **Blocked By**: Tasks 1-13

  **References**:
  - `internal/service/status_service.go` - 状态服务
  - MCP get_system_status 工具

  **Acceptance Criteria**:
  - [ ] /baxi-status 显示系统状态
  - [ ] /baxi-case <id> 显示案例详情

  **QA Scenarios**:

  ```
  状态查询测试
    Tool: pi
    Steps:
      1. pi -p "/baxi-status"
      2. 验证输出包含表计数、告警统计
    Expected Result: 系统状态正确显示
    Evidence: .sisyphus/evidence/task-15-status.txt
  ```

  **Commit**: NO

---

- [x] 16. 端到端测试

  **What to do**:
  - 编写完整闭环测试脚本
  - 测试: 告警 → 决策 → 审批 → 执行 → 反馈
  - 测试异常场景 (执行失败、审批拒绝)
  - 测试并发场景
  - 生成测试报告

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3
  - **Blocks**: F1-F4
  - **Blocked By**: Tasks 1-15

  **References**:
  - `test/integration/phase7_test.go` - 现有集成测试
  - MCP 工具文档

  **Acceptance Criteria**:
  - [ ] 完整闭环测试通过
  - [ ] 异常场景测试通过
  - [ ] 测试报告生成

  **QA Scenarios**:

  ```
  完整闭环测试
    Tool: Bash
    Steps:
      1. 运行端到端测试脚本
      2. 检查测试报告
    Expected Result: 所有测试通过
    Evidence: .sisyphus/evidence/task-16-e2e.txt
  ```

  **Commit**: YES
  - Message: `test(e2e): add Pi Agent integration end-to-end tests`
  - Files: `test/e2e/pi_integration_test.go`

---

## Final Verification Wave

- [x] F1. **计划合规审计** — `oracle`
- [x] F2. **代码质量审查** — `unspecified-high`

- [ ] F3. **真实 QA 测试** — `unspecified-high`
  使用 Pi 执行完整闭环流程

- [ ] F4. **范围保真检查** — `deep`
  验证实现与计划一致，无范围蔓延

---

## Commit Strategy

| Task | Commit Message | Files |
|------|----------------|-------|
| 1 | `fix(migration): add next_retry_at to outbox_events` | migrations/028_*.sql |
| 2 | `fix(action): empty config should disable all actions` | internal/action/registry.go |
| 3 | `fix(action): inject real executors` | internal/api/server.go |
| 4 | `fix(ontology): align schema with migration` | internal/repository/ontology_repository.go |
| 5-6 | `feat(mcp): add approval tools` | internal/mcp/tools_review.go |
| 7-8 | `feat(mcp): add execution tools` | internal/mcp/tools_action.go |
| 9-10 | `feat(mcp): add outbox tools` | internal/mcp/tools_outbox.go |
| 11-12 | `feat(mcp): add status tools` | internal/mcp/tools_status.go |
| 16 | `test(e2e): add Pi integration tests` | test/e2e/pi_integration_test.go |

---

## Success Criteria

### 验证命令

```bash
# 1. 所有测试通过
go test ./... -v

# 2. MCP Server 启动
go run ./cmd/baxi-mcp

# 3. Pi 连接测试
pi -p "使用 baxi MCP 列出最近告警"

# 4. 完整闭环测试
go test ./test/e2e/... -run TestPiIntegration -v
```

### 最终检查清单

- [ ] 所有 P0 Bug 修复
- [ ] 所有 MCP 工具可用
- [ ] Pi 扩展加载成功
- [ ] 主动预警工作正常
- [ ] 完整闭环测试通过
- [ ] 文档更新
