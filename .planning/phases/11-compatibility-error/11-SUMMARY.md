# Phase 11 Summary: Compatibility & Error 净化

**Status:** ✅ Complete
**Date:** 2026-06-06
**Requirement:** MCP-09

## Deliverables

### Modified Files
- **`internal/mcp/output_filter.go`** — 新增 `SanitizeError()` 和 `SanitizeErrorf()` 函数
  - `SanitizeError(msg string) string` — 脱敏数据库 schema/文件路径/SQL 错误
  - `SanitizeErrorf(format string, args...) string` — 包装版，直接替换 `fmt.Sprintf`
  - 正则脱敏：`schema.table` → `db.table`，文件路径 → `[redacted path]`，SQL 错误 → `"database operation failed"`
- **12 个 `tools_*.go` 文件** — 40 个 `NewToolResultError(fmt.Sprintf(...))` 调用全部替换为 `NewToolResultError(SanitizeErrorf(...))`
  - 移除了 11 个未使用的 `"fmt"` 导入（tools_action.go 仍使用 `fmt.Errorf`）

### Error Paths Covered

所有 MCP handler 的错误路径都已脱敏：

| 文件 | 错误点数 | 原始风险 |
|------|---------|---------|
| tools_decision.go | 6 | SQL 表名校验、创建/查询错误 |
| tools_action.go | 6 | JSON 解析、proposal/context 构建错误 |
| tools_review.go | 5 | 审批/驳回/取消提案错误 |
| tools_sandbox.go | 5 | 沙盘创建/比较错误 |
| tools_ontology.go | 4 | get_object/describe_schema 错误 |
| tools_outbox.go | 3 | 事件/状态查询错误 |
| tools_status.go | 2 | get_system_health/search 错误 |
| tools_governance.go | 2 | check_access/classification 错误 |
| tools_pipeline.go | 2 | process_data 错误 |
| tools_schema.go | 2 | action_type 错误 |
| tools_alert.go | 1 | list_alerts 错误 |
| tools_context.go | 1 | build_context 错误 |

### Already Completed in Earlier Phases
- ✅ E2E 测试在新工具名下更新（Phase 7）
- ✅ Pi 扩展提示文本更新（Phase 7）
- ✅ Pi Agent 集成验证（Phase 7-10 持续验证）

## Verification
- ✅ `go build ./...` — 编译通过
- ✅ `go test ./internal/mcp/...` — 4/4 测试通过
- ✅ 错误消息不再泄露 `schema.table`、文件路径、SQL 细节

## 验收标准对照

| 标准 | 结果 |
|------|------|
| 所有 NewToolResultError 调用点完成错误信息脱敏 | ✅ SanitizeErrorf 覆盖全部 40 处 |
| 无 SQL/schema/架构细节从错误中泄露 | ✅ schema.table → db.table |
| 创建 sanitizeError 辅助函数 | ✅ SanitizeError + SanitizeErrorf |
