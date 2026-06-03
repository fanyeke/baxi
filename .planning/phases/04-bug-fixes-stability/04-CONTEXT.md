# Phase 04: Bug Fixes & Stability - Context

**Gathered:** 2026-06-03
**Status:** Ready for planning

<domain>
## Phase Boundary

Phase 4 修复已知的静默错误、序列化失败、迁移序列不完整和 SQL 注入风险。让系统在面对边缘情况时优雅降级而非静默失败。

**在范围内（从 REQUIREMENTS.md）：**
- BUG-01: action.go JSON 解码错误处理（已在 Phase 2 修复，确认移除）
- BUG-02: alert/engine.go JSON marshal 错误从静默忽略改为记录日志 + 优雅降级
- BUG-03: feishu/client.go page_token 类型断言失败从静默忽略改为记录 + 中断分页
- BUG-04: 迁移序列缺口（缺少 015、025）补充空占位迁移
- BUG-05: ontology/repository.go SQL 注入风险——添加允许列表 + pgx.Identifier 消毒

**不在范围内：**
- 安全加固（Phase 5）
- E2E 测试和前后端集成（Phase 6）
- 新功能或新 endpoint
- CLI 架构重构
- 性能优化

</domain>

<decisions>
## Implementation Decisions

### BUG-01: action.go JSON 解码（已在 Phase 2 修复）
- **D-01:** BUG-01 确认已在 Phase 2（commit `aa363e7`）修复。当前 `internal/api/handler/action.go:69-72` 正确返回 400 + 错误信息。从 Phase 4 范围中移除。

### BUG-02: 告警引擎 JSON Marshal 错误处理
- **D-02:** `json.Marshal(evidence)` 失败时，使用 `zap.Logger.Error()` 记录错误，包含证据键名清单和 marshal 错误信息。**不记录完整证据内容**（可能包含敏感数据）。
- **D-03:** 记录日志后，使用空证据 `evJSON` 继续创建 `AlertResult`。不跳过告警——Message 字段仍包含可读信息，告警不能完全丢失。
- **Rationale:** 告警系统的首要职责是发出告警，证据是辅助信息。Marshal 失败不应导致告警丢失。

### BUG-03: 飞书 page_token 类型断言失败
- **D-04:** `pageToken, _ = data["page_token"].(string)` 失败时，使用 `zap.Logger.Error()` 记录错误，然后 `break` 退出分页循环。
- **D-05:** 不向调用方返回错误——返回已获取的 `allRecords` 和 `nil error`。调用方获得部分数据，不会因为分页问题而完全失败。
- **Rationale:** 分页中断是部分故障，不是完全故障。已获取的数据仍然有效。

### BUG-04: 迁移序列缺口
- **D-06:** 添加空占位迁移文件 `015` 和 `025`，包含注释说明编号被有意跳过。不对 goose 功能产生任何影响。
- **Rationale:** Git 历史中从未存在过 015 或 025——是创建时跳过了编号，并非被删除。Goose 对缺口不敏感。
- **占位文件内容:** 每个文件包含 `-- +goose Up` / `-- +goose Down` 和注释 `-- Migration [NUM]: intentionally skipped — no content was removed`.

### BUG-05: Ontology SQL 注入加固
- **D-07:** 在 V1 回退路径（`fullTableName()`）中使用 `pgx.Identifier{Schema, Table}.Sanitize()` 构建安全标识符，替换当前的直接字符串拼接。
- **D-08:** 在 `objectTableMap` 查找后添加显式允许列表检查——确保 `objectType` 是预定义映射中的有效键，在构建任何查询前验证。
- **D-09:** 在修复的 V1 回退路径代码旁添加 `// GODEPRECATED: use V2 compiler instead` 注释。
- **Rationale:** `objectTableMap` 是编译期常量映射，但缺少防御性消毒。V2 compiler 已经在 `compiler.go` 中使用 `pgx.Identifier` 消毒——V1 路径也应遵循相同标准。

### the agent's Discretion
- 告警引擎中 zap logger 的具体注入方式（Engine 结构体字段 vs 参数传递）
- 占位迁移文件中注释的具体措辞
- 允许列表检查的具体实现细节
- 不涉及自由裁量——所有 bug 都有明确的修复方向

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Bug Affected Files
- `internal/alert/engine.go:174,241,303` — 3 处 `json.Marshal` 错误被静默忽略（BUG-02）
- `internal/feishu/client.go:124` — `page_token` 类型断言失败（BUG-03）
- `migrations/` — 缺少 `015_*.sql` 和 `025_*.sql`（BUG-04）
- `internal/repository/ontology/repository.go:227-228,370-377` — `fullTableName()` 无 SQL 消毒 + V1 回退路径直接拼接查询（BUG-05）
- `internal/repository/ontology/compiler.go:76-191` — V2 compiler（参考——已正确使用 pgx.Identifier 消毒）

### Confirmed Fixed
- `internal/api/handler/action.go:69-72` — BUG-01 已在 Phase 2 修复

### Requirements & Roadmap
- `.planning/REQUIREMENTS.md` §BUG-01~BUG-05 — 本阶段 5 个需求（BUG-01 已移除）
- `.planning/ROADMAP.md` §Phase 4 — Phase 4 目标和成功标准

### Prior Phase Context
- `.planning/phases/02-error-handling-observability/02-PLAN.md` — Phase 2 的错误处理模式（参考——BUG-02/03 遵循类似模式）
- `.planning/phases/01-core-api-completion/01-CONTEXT.md` — 现有 handler 和 service 结构

### Codebase Maps
- `.planning/codebase/ARCHITECTURE.md` — 分层架构概览
- `.planning/codebase/CONCERNS.md` §Known Bugs — bug 的详细描述
- `.planning/codebase/CONCERNS.md` §Security Considerations — SQL 注入风险细节

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/logger/` — zap.Logger 初始化，可直接注入到 alert Engine
- `internal/repository/ontology/compiler.go` — V2 compiler 使用的 `pgx.Identifier.Sanitize()` 模式，可供 V1 路径参考
- Phase 2 的错误处理模式（类型化哨兵错误、writeValidationError、writeServiceError）——BUG-02/03 可复用相同模式

### Established Patterns
- **错误处理模式:** handler 层使用 `writeError(w, r, status, code, message)` 统一错误响应。但 engine 和 client 层不会返回 HTTP 错误——它们应使用 zap 记录日志
- **日志模式:** 项目使用 `go.uber.org/zap` 的 JSON 编码结构化日志，`alert/engine.go` 和 `feishu/client.go` 使用标准库 `log/slog`——需要升级到 zap

### Integration Points
- BUG-02 影响 3 条告警规则：`evaluateGMVDrop`、`evaluateLateDeliverySpike`、`evaluateCancelRateSpike`
- BUG-03 影响 `feishu/client.go` 的 `FetchAllRecords` 分页循环
- BUG-05 影响 4 个 V1 回退函数：`QueryByObjectType`、`QueryByID`、`QueryObjectMetrics`、`SearchObjects`

</code_context>

<specifics>
## Specific Ideas

- BUG-02 的 zap logger 可以通过 `Engine` 结构体注入（新增 `logger *zap.Logger` 字段），也可以在函数内部使用 `logger.L()` 全局 logger——由 planner 决定
- BUG-05 的 `fullTableName()` 改为 `pgx.Identifier{m.Schema, m.Table}.Sanitize()` 即可——这个函数很简单（当前只有一行返回）
- 占位迁移文件可以只包含 `-- +goose Up` / `-- +goose Down` 和一行注释

</specifics>

<deferred>
## Deferred Ideas

- 无——讨论保持在阶段范围内

</deferred>

---

*Phase: 04-Bug-Fixes-Stability*
*Context gathered: 2026-06-03*
