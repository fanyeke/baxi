# Discussion Log: Phase 01 — Core API Completion

**Date:** 2026-06-03
**Phase:** 01 — Core API Completion

---

## Areas Discussed

### 1. DecideLLM 行为
**Options presented:**
- 直接代理到 Decide（最简单，复用现有代码）
- 独立 LLM 专用流程（额外灵活性）
- 增强版 Decide（返回元数据）

**Decision:** 直接代理到 Decide。DecideLLM 作为 Decide 的别名，走完全相同的 LLM 决策流程。

### 2. Compare 数据格式
**Options presented:**
- 后端计算结构化 diff（前端只需渲染）
- 返回原始数据让前端比较（后端最简单）
- 混合模式（后端标记 + 前端计算）

**Decision:** 后端计算结构化 diff。后端比较输出 JSON、提案列表、状态差异，返回 `{added, removed, changed}` 数组。

### 3. Replay 灵活性
**Options presented：**
- 固定重放（完全复用原始上下文）
- 允许修改参数（客户端传入覆盖值）
- 固定重放 + 可选覆盖（平衡方案）

**Decision:** 允许修改参数。客户端可传入 `model`、`temperature`、`context_overrides` 重新决策。`case_id`、`source_type`、`source_id` 不可修改。

### 4. BatchDispatch 范围
**Options presented：**
- 处理所有 pending 事件（最简单）
- 支持筛选条件（按 channel/type/时间）
- 处理 pending + 自动重试失败事件（最完整）

**Decision:** 处理所有 pending 事件。找到所有 `status='pending'` 的事件，逐个尝试分发。

---

## Deferred Ideas

- Compare 可视化渲染细节 → Phase 6（前端）
- Replay 历史记录保存 → 未来增强
- BatchDispatch 筛选和自动重试 → Phase 2/3
- OpenAPI 文档生成工具 → Phase 1 内实现，但工具选型可延后

---

## Scope Creep Redirected

无。所有讨论均在 Phase 1 范围内。

---

*Logged: 2026-06-03*
