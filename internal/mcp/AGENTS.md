# mcp: MCP Server — AI Agent 集成层

**Generated:** 2026-05-29
**Commit:** d908f6d
**Branch:** main

## OVERVIEW

MCP (Model Context Protocol) 服务器，通过 stdio 传输协议暴露 Baxi 核心能力。基于 `github.com/mark3labs/mcp-go` 框架，支持 Pi Agent 等 MCP 兼容客户端连接。注册 17 个工具，按 8 个业务域分组。

## STRUCTURE

11 个文件：

| 文件 | 职责 |
|------|------|
| `server.go` | `Server` 结构体（14 个依赖字段）、`NewServer` 构造函数、`Run()` stdio 启动、工具注册编排 |
| `interfaces.go` | 10 个服务接口定义（`DecisionService`、`DecisionEngine`、`ContextBuilder` 等） |
| `server_test.go` | Mock 实现 12 个 + 2 个测试（构造验证 + 工具注册清单验证） |
| `tools_decision.go` | 5 个决策域工具：create_decision_case, decide, list_cases, get_case, list_proposals |
| `tools_alert.go` | 1 个告警域工具：list_alerts |
| `tools_governance.go` | 2 个治理域工具：check_access, get_classification |
| `tools_pipeline.go` | 1 个管道域工具：run_pipeline |
| `tools_review.go` | 2 个审核域工具：approve_proposal, reject_proposal |
| `tools_action.go` | 2 个执行域工具：execute_proposal, get_decision_context |
| `tools_outbox.go` | 2 个出站事件域工具：list_outbox_events, get_pipeline_status |
| `tools_status.go` | 2 个状态域工具：get_system_status, search_objects |

## WHERE TO LOOK

| 任务 | 文件 | 说明 |
|------|------|------|
| 添加新工具 | `tools_*.go` + `server.go` + `interfaces.go` | 4 步模式：定义 handler → 注册函数 → server.go 注册 → interfaces.go 加接口（如需） |
| 修改接口定义 | `interfaces.go` | 添加/修改服务接口，保持与 `cmd/baxi-mcp/main.go` 适配器一致 |
| 修改 Server 构造 | `server.go` | `NewServer` 构造函数 + `register*Tools` 调用 + `Server` 结构体字段 |
| 修改 main.go 注入 | `cmd/baxi-mcp/main.go` | 依赖装配，共 13 个参数传给 NewServer |
| 查找某工具 handler | `tools_*.go` | 按域分组，每个文件对应一个 register*Tools + handler 对 |
| 验证工具注册 | `server_test.go` | `TestServerToolRegistration` 维护 17 个工具名白名单 |

## 工具清单

| 工具名 | 组 | 参数 | 说明 |
|--------|-----|------|------|
| create_decision_case | decision | alert_id(必填), created_by | 从告警创建决策 Case |
| decide | decision | case_id(必填) | 为 Case 生成决策 |
| list_cases | decision | source_type, source_id, status, severity, limit, offset | 分页查询 Case 列表 |
| get_case | decision | case_id(必填) | 查询单个 Case 详情 |
| list_proposals | decision | case_id(必填) | 列出 Case 下的 Action 提案 |
| list_alerts | alert | severity, status, object_type, rule_id, sort, limit, offset | 分页查询告警列表 |
| check_access | governance | role(必填), object_type(必填), action(必填) | 检查角色权限 |
| get_classification | governance | field_path(必填) | 查询字段分类级别 |
| run_pipeline | pipeline | config(必填) | 运行数据管道 |
| approve_proposal | review | proposal_id(必填), reviewer_id(必填), feedback | 审批提案 |
| reject_proposal | review | proposal_id(必填), reviewer_id(必填), feedback | 驳回提案 |
| execute_proposal | action | proposal_id(必填), dry_run | 执行已审批提案 |
| get_decision_context | action | case_id(必填) | 获取决策上下文 |
| list_outbox_events | outbox | status, limit, offset | 查询出站事件 |
| get_pipeline_status | outbox | (无参数) | 获取管道运行状态 |
| get_system_status | status | (无参数) | 系统概览 |
| search_objects | status | object_type(必填), query(必填), limit, offset | 搜索对象 |

## KEY PATTERNS

- **4 步工具模式**: ① 在 `tools_*.go` 定义 `mcp.NewTool` + handler 方法 ② 同文件内 `register*Tools()` 调用 `s.server.AddTool` ③ `NewServer` 调用 `srv.register*Tools()` ④ `interfaces.go` 定义服务接口，`cmd/baxi-mcp/main.go` 提供实现
- **接口定义在 mcp 包，实现在 cmd**: `interfaces.go` 只声明服务接口，具体实现在 `cmd/baxi-mcp/main.go`（含 adapter 包装），main.go 将业务服务适配为 mcp 包的接口
- **传输方式**: stdio，通过 `server.ServeStdio(s.server)` 启动
- **返回值格式**: 所有 handler 统一使用 `mcp.NewToolResultJSON(map)` 返回 JSON，错误使用 `mcp.NewToolResultError`
- **参数解析模式**: 统一从 `req.GetArguments()` 读取，用 `ok` 模式做强类型断言，可选参数检查空字符串

## ANTI-PATTERNS

- **interfaces.go 和 server.go 紧耦合**: 新增工具需要修改 `server.go`（Server 结构体 + NewServer）+ `interfaces.go`（新接口）+ `tools_*.go`，总共改 3 个文件
- **无 HTTP/SSE 传输支持**: 当前只支持 stdio，无法远程连接
- **status/action 服务为桩实现**: `executeServiceAdapter`、`statusServiceAdapter`、`searchServiceAdapter` 在 `cmd/baxi-mcp/main.go` 中返回空结果，未对接真实业务逻辑
- **server_test.go 工具注册测试硬编码清单**: `expectedTools` 切片需要手动同步新增/删除的工具，容易遗漏更新
- **Mock 代码冗长**: `server_test.go` 中 12 个 Mock 结构体（共 ~190 行），每个需要按接口方法一一映射，重复代码多
