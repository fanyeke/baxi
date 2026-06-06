# Phase 7 Summary: Foundation — 服务器身份泛化 & 工具名抽象

**Status:** ✅ Complete
**Date:** 2026-06-06
**Requirements:** MCP-01, MCP-02

## Deliverables

### New Files
- **`internal/mcp/server_identity.go`** — 服务器身份泛化（env 驱动）
  - `getServerName()` / `getServerVersion()` / `getServerInstructions()`
  - Env vars: `MCP_SERVER_NAME`, `MCP_SERVER_VERSION`, `MCP_SERVER_INSTRUCTIONS`
  - Defaults: "Data Processing Server" / "1.0.0" / "Platform for data analysis and action management"
  - `isLegacyToolsEnabled()` — 控制旧名兼容模式

- **`internal/mcp/tool_names.go`** — 34 个工具名常量 + 33 个旧名常量
  - 所有工具改为业务能力命名（如 `run_pipeline`→`process_data`, `describe_ontology`→`describe_schema`）
  - `legacyToolMap` 实现新旧名称双向映射
  - 条件编译：`MCP_ENABLE_LEGACY_TOOLS=true`（默认）启用旧名兼容

### Modified Files
- **`internal/mcp/server.go`** — `NewServer` 使用 `getServerName()`/`getServerVersion()`/`getServerInstructions()`，抹掉 "Baxi MCP Server"
- **12 个 `tools_*.go` 文件** — 全部改名 + 条件注册旧别名
- **`internal/mcp/server_test.go`** — 更新 expectedTools 为新旧常量
- **`test/e2e/pi_integration_test.go`** — 更新工具名引用
- **`test/e2e/decision_lifecycle_test.go`** — 更新工具名引用
- **`test/integration/ontology_v2_e2e_test.go`** — 更新工具名引用
- **`pi-extension/baxi-decision/`** — 更新 hint 文本
- **`pi-extension/baxi-operations/`** — 更新工具名引用

### Verification
- ✅ `go build ./...` — 全量编译通过
- ✅ `go test ./internal/mcp/...` — 4/4 测试通过
- ✅ 基线 Pi 测试 — 新名称正常响应，旧名称兼容可用
- ✅ 旧名兼容 — `MCP_ENABLE_LEGACY_TOOLS=false` 仅注册新名（33 个工具）

## Tool Name Mapping (完整)

| 旧名 | 新名 | 说明 |
|------|------|------|
| create_decision_case | evaluate_case | 评估 |
| decide | generate_recommendation | 推荐 |
| resolve_case | resolve_evaluation | 解决评估 |
| list_cases | list_evaluations | 列出评估 |
| get_case | get_evaluation | 获取评估 |
| list_proposals | list_recommendations | 列出推荐 |
| build_context | analyze_situation | 场景分析 |
| propose_action | suggest_action | 建议动作 |
| approve_proposal | approve_action | 批准动作 |
| reject_proposal | reject_action | 拒绝动作 |
| cancel_proposal | cancel_action | 取消动作 |
| get_proposal_by_id | get_action_proposal | 获取动作提案 |
| list_review_records | list_reviews | 列出审核记录 |
| execute_proposal | execute_action | 执行动作 |
| get_decision_context | get_decision_context | (同名保留) |
| check_access | check_permission | 检查权限 |
| get_classification | get_data_classification | 数据分类 |
| run_pipeline | process_data | 数据处理 |
| get_pipeline_status | get_processing_status | 处理状态 |
| get_system_status | get_system_health | 系统健康 |
| search_objects | search_records | 搜索记录 |
| describe_ontology | describe_schema | 描述模型 |
| get_object | get_record | 获取记录 |
| get_linked_objects | get_related_records | 关联记录 |
| execute_action | apply_action | 执行操作 |
| create_sandbox | create_simulation | 创建模拟 |
| add_to_sandbox | add_to_simulation | 加入模拟 |
| compare_sandboxes | compare_simulations | 比较模拟 |
| get_sandbox | get_simulation | 获取模拟 |
| list_action_schemas | list_action_types | 动作类型列表 |
| get_action_schema | get_action_type | 获取动作类型 |

## 测试验证

Pi 非交互模式测试确认：
1. **服务器身份** — 不再暴露版本号、技术栈、项目描述
2. **工具命名** — 旧名/新名同时可用，Agent 看到的工具名改为主干能力描述
3. **兼容性** — `MCP_ENABLE_LEGACY_TOOLS=true`（默认）保证所有旧工具名仍然可用
4. **Pi 扩展** — `baxi-decision` 和 `baxi-operations` 的提示文本已更新

## 下一步

Phase 8 (Output: Schema & Status 裁剪) 在这个基础上展开：
- 利用 `output_filter.go` 实现 describe_ontology / get_system_status 输出过滤
- 移除 SourceDescriptor 和 table_counts
