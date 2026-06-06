# Roadmap: Baxi Demo-Ready Platform

## Overview

This roadmap takes Baxi from a brownfield Go/PostgreSQL + React codebase with 6 broken API endpoints, generic 500 errors, dead code, and security gaps to a complete, demonstrable closed-loop governance platform. Each phase delivers an observable increment — working APIs, proper errors, clean code, stable behavior, secure access, and a fully integrated frontend-to-backend demo.

## Phases

**Phase Numbering:**

- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

## Milestone v1.0 (已完成)

- [x] **Phase 1: Core API Completion** - Implement all 6 API endpoints returning 501, plus OpenAPI schemas (completed 2026-06-03)
- [x] **Phase 2: Error Handling & Observability** - Replace generic 500s with proper HTTP status codes and structured error responses (completed 2026-06-03)
- [x] **Phase 3: Code Hygiene & Cleanup** - Remove Python/SQLite remnants, dead code, deprecated repositories (completed 2026-06-03)
- [x] **Phase 4: Bug Fixes & Stability** - Fix silently ignored errors, marshaling failures, migration gaps, SQL injection risks (completed 2026-06-03)
- [x] **Phase 5: Security Hardening** - Strengthen auth middleware, CORS validation, Docker Compose credentials (completed 2026-06-03)
- [x] **Phase 6: Integration & End-to-End Demo** - Wire frontend to backend, pass all E2E tests, run full closed-loop demo (completed 2026-06-03)

## Milestone v1.1 — MCP 信息收束

- [ ] **Phase 7: Foundation — 服务器身份泛化 & 工具名抽象** - 泛化 MCP 服务器名称和 instructions，工具按业务能力重新分组命名
- [ ] **Phase 8: Schema & Status 输出裁剪** - 移除 describe_ontology 和 get_system_status 中的架构细节泄露
- [ ] **Phase 9: 对象数据字段级过滤** - get_object 和 get_linked_objects 利用 LLMReadable 标记做字段级过滤
- [ ] **Phase 10: 输入加固 — Search & Pipeline** - search_objects 分页限制 + run_pipeline config allowlist
- [ ] **Phase 11: 兼容性验证 & 错误信息净化** - Pi Agent 兼容验证、E2E 测试扩展、错误信息脱敏

## Phase Details

### Phase 1: Core API Completion

**Goal**: All API endpoints return proper responses with documented schemas instead of 501 Not Implemented
**Depends on**: Nothing (first phase)
**Requirements**: API-01, API-02, API-03, API-04, API-05, API-06, API-07
**Success Criteria** (what must be TRUE):

  1. POST /api/v1/decisions/llm returns 200 with valid decision body containing context, validation, and repair retry
  2. POST /api/v1/decisions/compare returns 200 with structured comparison and diff visualization data
  3. POST /api/v1/decisions/replay returns 200 with new decision result from original or modified context
  4. GET /api/v1/decisions/llm returns 200 with paginated list of LLM-assisted decisions supporting filters
  5. GET /api/v1/evals returns 200 with evaluation metrics and quality scores
  6. POST /api/v1/outbox/dispatch returns 200 after processing pending outbox events in batch
  7. All implemented endpoints return response bodies matching their OpenAPI-documented schemas

**Plans**: 4 plans

Plans:

- [x] 01-01-PLAN.md — DecideLLM + ListLLMDecisions + ListEvals implementation
- [x] 01-02-PLAN.md — Compare + Replay implementation
- [x] 01-03-PLAN.md — BatchDispatch implementation
- [x] 01-04-PLAN.md — OpenAPI schema documentation

### Phase 2: Error Handling & Observability

**Goal**: API returns meaningful HTTP status codes and structured error details instead of generic 500 errors
**Depends on**: Phase 1
**Requirements**: ERR-01, ERR-02, ERR-03, ERR-04, ERR-05
**Success Criteria** (what must be TRUE):

  1. Invalid request payloads return 400 Bad Request (not 500) with field-level error details
  2. Requests for non-existent resources return 404 Not Found (not 500)
  3. All error responses include structured JSON with `code`, `message`, and optional `details` fields
  4. Malformed JSON in request bodies returns 400 with parse error details instead of being silently ignored
  5. Database connection failures return 503 Service Unavailable with retry-after guidance

**Plans**: 3 plans
Plans:
**Wave 1**

- [x] 02-01-PLAN.md — 核心错误基础设施（新常量、Details 字段、FieldError 类型、DB 连接检测）

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 02-02-PLAN.md — Bug 修复 + 错误码修正（静默 JSON 解码、冲突/服务不可用错误码、类型化哨兵错误）

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 02-03-PLAN.md — 验证错误字段级详情 + DB 503 检测（writeValidationError、writeServiceError 覆盖所有 handler）

### Phase 3: Code Hygiene & Cleanup

**Goal**: Clean, buildable codebase with no Python/SQLite migration artifacts or dead code
**Depends on**: Phase 2
**Requirements**: HYG-01, HYG-02, HYG-03, HYG-04, HYG-05, HYG-06, HYG-07
**Success Criteria** (what must be TRUE):

  1. Pipeline preview command (`make pipeline` or baxi-cli) displays Go commands, not Python scripts
  2. Makefile has no references to Python scripts for verification or pipeline tasks
  3. Deprecated repository shim files are removed from the codebase
  4. All callers use subpackage repositories with PoolProvider — no references to deprecated shims remain
  5. Dead CLI subcommand `cmd/baxi-cli/llm.go` is either removed or properly wired into `main.go` dispatch
  6. Placeholder `internal/worker/worker.go` is removed
  7. Migration baseline directory (`migration_baseline/`) is archived or removed (no sqlite_schema.sql or Python scripts)

**Plans**: 3 plans

Plans:

- [x] 03-01-PLAN.md — 快速清理：pipeline 预览、Makefile、llm.go、worker.go (HYG-01, HYG-02, HYG-05, HYG-06)
- [x] 03-02-PLAN.md — 仓库调用方迁移 (HYG-04)
- [x] 03-03-PLAN.md — 最终清理：删除 shim + migration_baseline + 文档更新 (HYG-03, HYG-07)

### Phase 4: Bug Fixes & Stability

**Goal**: Known bugs are fixed and the system handles edge cases gracefully without silent failures
**Depends on**: Phase 3
**Requirements**: BUG-01, BUG-02, BUG-03, BUG-04, BUG-05
**Success Criteria** (what must be TRUE):

  1. Invalid JSON in `internal/api/handler/action.go` request body returns 400 instead of proceeding with zero-value defaults
  2. `internal/alert/engine.go` handles JSON marshal errors explicitly with logging (no silent data loss)
  3. `internal/feishu/client.go` handles `page_token` type assertion failures with proper error propagation
  4. Goose migration sequence is continuous with no missing migration numbers (audit and fix any gaps like 015, 025)
  5. Ontology repository queries use allowlist validation before interpolating `schema.table` identifiers (SQL injection eliminated)

**Plans**: 2 plans

Plans:

- [x] 04-01-PLAN.md — 迁移占位文件 + Feishu page_token 修复 (BUG-03, BUG-04)
- [x] 04-02-PLAN.md — 告警引擎 JSON marshal 修复 + Ontology SQL 注入加固 (BUG-02, BUG-05)

### Phase 5: Security Hardening

**Goal**: CORS origin check validates the scheme explicitly (`http` vs `https`) before allowing requests
**Depends on**: Phase 4
**Requirements**: SEC-02
**Success Criteria** (what must be TRUE):

   1. CORS origin check validates the scheme explicitly (`http` vs `https`) — SEC-01 (JWT/token rotation) and SEC-03 (Docker Compose credentials) skipped per D-03/D-04
   2. Port normalization works correctly: `http://localhost` matches `http://localhost:80`, `https://example.com` matches `https://example.com:443`
   3. Unparseable Origin headers are rejected (fail closed)
   4. `CORS_ALLOWED_ORIGINS` comma-separated format remains unchanged

**Plans**: 1 plan

### Phase 6: Integration & End-to-End Demo

**Goal**: Frontend connects to all backend features, all tests pass, and the full closed-loop demo runs successfully
**Depends on**: Phase 5
**Requirements**: INT-01, INT-02, INT-03, INT-04, INT-05
**Success Criteria** (what must be TRUE):

  1. Frontend pages for decisions, governance, pipeline, and alerts all load and display live data from backend endpoints
  2. E2E integration tests (`test/integration/phase7_test.go`) pass cleanly with no failures
  3. Security E2E tests (`test/security/phase7_test.go`) pass cleanly with no failures
  4. Frontend unit tests (`frontend/src/pages/__tests__/*.test.tsx`) pass cleanly
  5. Full closed-loop demo works end-to-end: trigger pipeline → governance rules fire → decision created → action executed → alert sent → result visible in frontend

**Plans**: 1 plan
Plans:

- [x] 05-01-PLAN.md — CORS scheme 验证（parseOrigins + isOriginAllowed 重构 + 单元测试）

**UI hint**: yes

### Phase 6: Integration & End-to-End Demo

**Plans**: 4 plans

Plans:

- [x] 06-01-PLAN.md — Go 测试编译修复（proposal_service, context_builder, alert_service, outbox）
- [x] 06-02-PLAN.md — 前端类型对齐（Governance、Pipeline 页面）
- [x] 06-03-PLAN.md — 前端测试断言修复（7 个测试文件）
- [x] 06-04-PLAN.md — 演示验证（全测试套件 + 闭环确认）

### Phase 7: Foundation — 服务器身份泛化 & 工具名抽象

**Goal**: MCP 服务器暴露不可识别项目身份的通用身份信息，工具按业务能力命名（抹掉 internal 包映射）
**Depends on**: Nothing (first phase of v1.1 milestone)
**Requirements**: MCP-01, MCP-02
**Success Criteria** (what must be TRUE):

  1. `server.info` 中的服务器名称和 instructions 不包含 "Baxi"、项目名或任何可识别项目身份的文本
  2. 所有 31+ MCP 工具使用业务能力名称（如 `describe_schema` 替代 `describe_ontology`），无 `internal/` 包映射痕迹
  3. 通过 `MCP_ENABLE_LEGACY_TOOLS=true` 环境变量可启用旧名称兼容模式，旧工具名仍可调用
  4. 默认工具列表仅显示新名称，不暴露旧名称

**Plans**: TBD

### Phase 8: Schema & Status 输出裁剪

**Goal**: `describe_ontology` 和 `get_system_status` 不再暴露数据库架构细节
**Depends on**: Phase 7
**Requirements**: MCP-03, MCP-04
**Success Criteria** (what must be TRUE):

  1. `describe_ontology` 响应中不包含 SourceDescriptor（无 schema/table/primary_key 字段）
  2. `describe_ontology` 中属性描述仅保留精简的 LLMReadable 字段，不暴露底层数据模型结构
  3. `get_system_status` 响应中不包含 `table_counts`（无表名或行数信息）
  4. `get_system_status` 仅展示聚合级别健康状态（alert_count 总量等），不暴露 schema 级细节
  5. 创建 `output_filter.go` 集中管理所有过滤函数，包含 `FilterProperty()` / `FilterOntologyDescriptor()` / `FilterSystemStatus()` / `FilterSearchResult()`

**Research flag**: 需要审计所有 31 个 handler 的输出结构，确保 LLMReadable 覆盖完整。某些属性可能未设置标记。

**Plans**: TBD

### Phase 9: 对象数据字段级过滤

**Goal**: `get_object` 和 `get_linked_objects` 只返回允许 Agent 读取的属性
**Depends on**: Phase 8 (复用 output_filter.go 的 FilterProperties)
**Requirements**: MCP-05, MCP-06
**Success Criteria** (what must be TRUE):

  1. `get_object` 响应仅包含标记了 `LLMReadable: true` 的属性字段
  2. `get_linked_objects` 默认 max_depth ≤ 1，防止深度遍历泄露关联数据
  3. `get_linked_objects` 同对象字段级过滤与 get_object 一致，不会因关联查询跳过过滤
  4. 通过 E2E 测试验证非 LLMReadable 属性的确不存在于响应中

**Plans**: TBD

### Phase 10: 输入加固 — Search & Pipeline

**Goal**: `search_objects` 和 `run_pipeline` 只接受受约束和验证过的输入参数
**Depends on**: Phase 8 (复用 output_filter.go 的 FilterSearchResult)
**Requirements**: MCP-07, MCP-08
**Success Criteria** (what must be TRUE):

  1. `search_objects` 强制最大结果数限制和分页上限，超过时截断或拒绝
  2. `search_objects` 搜索结果仅返回 ID/type/精简摘要，不返回完整对象
  3. `run_pipeline` 的 config 参数仅接受预定义的 pipeline 类型枚举（allowlist），拒绝任意字符串
  4. `run_pipeline` 的 data_dir 参数已移除或固定为内置路径，用户无法指定任意目录

**Plans**: TBD

### Phase 11: 兼容性验证 & 错误信息净化

**Goal**: 所有改造已验证与 Pi Agent 兼容；错误路径不泄露内部架构细节
**Depends on**: Phases 7-10
**Requirements**: MCP-09
**Success Criteria** (what must be TRUE):

  1. 完整的 E2E 测试套件在新工具名称下通过（覆盖全部 31+ 工具）
  2. Pi Agent 集成测试在旧名称兼容模式下通过
  3. Pi 扩展 (`pi-extension/baxi-decision/index.ts`) 中的工具提示更新为意图描述（不再引用旧工具名）
  4. 所有工具的 `NewToolResultError` 调用点完成错误信息脱敏——无 SQL、schema、架构细节泄露
  5. 创建 `sanitizeError()` 辅助函数，统一处理错误文本中的 `schema.table` 模式和 SQL 关键词脱敏

**Research flag**: 需要追溯 `internal/mcp/` 和 `cmd/baxi-mcp/main.go` 中所有 `fmt.Errorf` 和 `NewToolResultError` 调用点，约 15-20 处。

**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 → 9 → 10 → 11
*Note: Phases 8, 9, 10 all depend on Phase 7 but are independent of each other — can be parallelized if needed.*

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Core API Completion | 4/4 | Complete | 2026-06-03 |
| 2. Error Handling & Observability | 3/3 | Complete | 2026-06-03 |
| 3. Code Hygiene & Cleanup | 3/3 | Complete | 2026-06-03 |
| 4. Bug Fixes & Stability | 2/2 | Complete | 2026-06-03 |
| 5. Security Hardening | 1/1 | Complete | 2026-06-03 |
| 6. Integration & End-to-End Demo | 4/4 | Complete | 2026-06-03 |
| 7. Foundation — 身份 & 命名 | 0/TBD | Not started | - |
| 8. Schema & Status 输出裁剪 | 0/TBD | Not started | - |
| 9. 对象数据字段级过滤 | 0/TBD | Not started | - |
| 10. 输入加固 — Search & Pipeline | 0/TBD | Not started | - |
| 11. 兼容性验证 & 错误净化 | 0/TBD | Not started | - |
