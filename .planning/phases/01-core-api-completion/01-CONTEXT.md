# Phase 01: core-api-completion - Context

**Gathered:** 2026-06-03
**Status:** Ready for planning

<domain>
## Phase Boundary

Phase 1 实现 6 个当前返回 501 Not Implemented 的 API 端点，使所有核心决策和分发接口可用。范围包括：DecideLLM、Compare、Replay、ListLLMDecisions、ListEvals、BatchDispatch。

不包括：错误处理重构（Phase 2）、代码清理（Phase 3）、前端页面开发（Phase 6）。

</domain>

<decisions>
## Implementation Decisions

### DecideLLM 行为
- **D-01:** DecideLLM 直接代理到现有的 `Decide` 方法，走完全相同的 LLM 决策流程。不创建独立的 LLM 专用流程。
- **Rationale:** 现有 `Decide` 已经调用 LLM provider 并返回决策结果。DecideLLM 作为其别名可减少重复代码，保持行为一致。

### Compare 数据格式
- **D-02:** Compare 端点在后端计算结构化 diff，返回 `{added, removed, changed}` 数组。
- **Rationale:** 后端有完整的决策数据（输出 JSON、提案列表、状态），可以精确比较。前端只需渲染，降低前端复杂度。
- **Diff 维度:** 比较两个决策的 `output_json`、`proposals`（按 ProposalID 匹配）、`status`、`confidence`。

### Replay 灵活性
- **D-03:** Replay 端点允许客户端传入修改后的参数重新决策。
- **Rationale:** 演示场景需要展示不同参数下的决策差异。
- **可覆盖参数:** `model`（模型名称）、`temperature`（温度）、`context_overrides`（上下文字段覆盖）。
- **不可修改:** `case_id`（必须匹配原始 case）、`source_type`/`source_id`（保持溯源）。

### BatchDispatch 范围
- **D-04:** BatchDispatch 处理所有 `status='pending'` 的 outbox 事件，逐个尝试分发。
- **Rationale:** 最简单直接，符合演示需求。不需要筛选条件或自动重试失败事件（属于 Phase 2/3 的增强功能）。
- **返回格式:** 返回 `{dispatched: N, failed: M, event_ids: [...]}`，与现有 `BatchDispatchResponse` 结构一致。

### List 端点实现
- **D-05:** ListLLMDecisions 和 ListEvals 直接复用 `DecisionService` 中已实现的 `ListLLMDecisions` 和 `ListEvals` 方法。只需在 handler 中添加调用逻辑和 DTO 转换。
- **Rationale:** Service 层已实现数据库查询逻辑，handler 只需补全 HTTP 层封装。

### the agent's Discretion
- Compare diff 算法的具体实现细节（如 JSON 深度比较策略）由 planner/executor 决定。
- Replay 参数验证规则（温度范围、允许的上下文字段）由 planner/executor 决定。
- BatchDispatch 的错误处理策略（部分失败时是否继续）由 planner/executor 决定。

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### API Handler Patterns
- `internal/api/handler/decision.go` — 现有 decision handler 实现（含 5 个 501 stub）
- `internal/api/handler/outbox.go` — 现有 outbox handler 实现（含 1 个 501 stub）
- `internal/api/routes.go` — 路由定义（第 31、59-63 行）
- `internal/api/dto/` — DTO 类型定义
- `internal/httputil/` — JSON 响应、分页工具

### Service Layer
- `internal/service/decision_service.go` — DecisionService 实现（含 ListLLMDecisions、ListEvals）
- `internal/service/outbox_service.go` — OutboxService 实现

### Decision Engine
- `internal/decision/engine.go` — DecisionEngine 结构和方法
- `internal/decision/context_builder_v2.go` — 上下文构建
- `internal/decision/lineage_service.go` — 决策事件追踪

### LLM Provider
- `internal/llm/provider.go` — DecisionProvider 接口
- `internal/llm/openai_provider.go` — OpenAI 兼容 provider

### Outbox
- `internal/model/outbox.go` — Outbox 模型定义
- `internal/worker/dispatch_worker.go` — 事件分发逻辑

### Repository
- `internal/repository/decision/` — 决策相关数据库操作
- `internal/repository/outbox/` — Outbox 相关数据库操作

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `DecisionService.ListLLMDecisions()` — 已实现数据库查询，返回 `ai.llm_decision` 表数据
- `DecisionService.ListEvals()` — 已实现数据库查询，返回 `ai.decision_eval_result` 表数据
- `DecisionHandler.Decide()` — 现有决策流程可直接复用于 DecideLLM
- `OutboxService.List()` — 已有带筛选的分页查询，可用于 BatchDispatch 前获取 pending 事件
- `httputil.ParsePagination()` — 分页参数解析工具
- `caseToResponse()` / `proposalsToDTO()` — 现有 DTO 转换函数

### Established Patterns
- Handler 结构：`*Handler` struct + `NewXxxHandler(iface)` + 方法接收 `http.ResponseWriter, *http.Request`
- Service 接口：handler 定义 narrow interface（如 `DecisionService`），便于 mock 测试
- 错误处理：`writeError(w, r, status, code, message)` 统一错误响应格式
- DTO 转换：service 层返回 domain 类型，handler 层转换为 DTO
- 路由注册：在 `routes.go` 中使用 chi 的 `r.Post()` / `r.Get()`

### Integration Points
- DecideLLM → `DecisionService.Decide()` → `DecisionEngine` → LLM provider
- Compare → `DecisionService.GetCase()` × 2 → 比较逻辑 → diff 响应
- Replay → `DecisionService.Decide()` → 允许参数覆盖
- ListLLMDecisions → `DecisionService.ListLLMDecisions()` → DTO 转换
- ListEvals → `DecisionService.ListEvals()` → DTO 转换
- BatchDispatch → `OutboxService.List()`（筛选 pending）→ `OutboxService.DispatchEvent()` 逐个调用

</code_context>

<specifics>
## Specific Ideas

- Compare diff 应包含字段级变更标记（如 `field: "proposals"`, `change_type: "added"`, `before: null, after: [...]`）
- Replay 支持 `dry_run: true` 参数，只生成决策不保存（便于预览）
- BatchDispatch 返回结果中，`failed` 事件应附带错误信息摘要

</specifics>

<deferred>
## Deferred Ideas

- Compare 端点的可视化渲染细节（属于 Phase 6 前端工作）
- Replay 的历史记录保存（当前只返回结果，不创建新 case）
- BatchDispatch 的筛选条件和自动重试（属于 Phase 2/3 增强）
- OpenAPI 文档自动生成工具（属于 Phase 1 API-07，但工具选型可延后）

</deferred>

---

*Phase: 01-core-api-completion*
*Context gathered: 2026-06-03*
