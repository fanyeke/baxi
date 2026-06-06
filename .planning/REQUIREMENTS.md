# Requirements: Baxi MCP 信息收束

**Defined:** 2026-06-06
**Core Value:** Agent 通过 MCP 接触项目时，无法通过工具名称、服务器自述、返回数据拼凑出项目架构，也无法获取不应暴露的业务数据

## v1 Requirements (已关闭)

### API Completeness ✓

- [x] **API-01**: `POST /api/v1/decisions/llm` — DecideLLM endpoint (Phase 1, completed 2026-06-03)
- [x] **API-02**: `POST /api/v1/decisions/compare` — Compare endpoint (Phase 1, completed 2026-06-03)
- [x] **API-03**: `POST /api/v1/decisions/replay` — Replay endpoint (Phase 1, completed 2026-06-03)
- [x] **API-04**: `GET /api/v1/decisions/llm` — ListLLMDecisions endpoint (Phase 1, completed 2026-06-03)
- [x] **API-05**: `GET /api/v1/evals` — ListEvals endpoint (Phase 1, completed 2026-06-03)
- [x] **API-06**: `POST /api/v1/outbox/dispatch` — HandleBatchDispatch endpoint (Phase 1, completed 2026-06-03)
- [x] **API-07**: All implemented endpoints return OpenAPI-documented response schemas (Phase 1, completed 2026-06-03)

### Error Handling ✓

- [x] **ERR-01**: Service errors map to appropriate HTTP status codes (Phase 2, completed 2026-06-03)
- [x] **ERR-02**: Error responses include structured JSON (Phase 2, completed 2026-06-03)
- [x] **ERR-03**: Malformed JSON returns 400 with parse error details (Phase 2, completed 2026-06-03)
- [x] **ERR-04**: Database connection failures return 503 with retry-after guidance (Phase 2, completed 2026-06-03)
- [x] **ERR-05**: Validation errors return 400 with field-level error details (Phase 2, completed 2026-06-03)

### Code Hygiene ✓

- [x] **HYG-01**: Pipeline preview returns correct Go commands (Phase 3, completed 2026-06-03)
- [x] **HYG-02**: Makefile targets no longer reference Python scripts (Phase 3, completed 2026-06-03)
- [x] **HYG-03**: Deprecated repository shim files removed (Phase 3, completed 2026-06-03)
- [x] **HYG-04**: All callers migrated to subpackage repositories with PoolProvider (Phase 3, completed 2026-06-03)
- [x] **HYG-05**: Dead CLI subcommand removed or wired (Phase 3, completed 2026-06-03)
- [x] **HYG-06**: Placeholder worker.go removed (Phase 3, completed 2026-06-03)
- [x] **HYG-07**: Migration baseline directory archived (Phase 3, completed 2026-06-03)

### Bug Fixes ✓

- [x] **BUG-01**: action.go returns 400 on JSON decode failure (Phase 4, completed 2026-06-03)
- [x] **BUG-02**: alert engine handles JSON marshal errors explicitly (Phase 4, completed 2026-06-03)
- [x] **BUG-03**: feishu client handles page_token type assertion failure (Phase 4, completed 2026-06-03)
- [x] **BUG-04**: Goose migration sequence is continuous (Phase 4, completed 2026-06-03)
- [x] **BUG-05**: SQL injection risk in ontology eliminated (Phase 4, completed 2026-06-03)

### Security ✓

- [x] **SEC-02**: CORS origin check validates scheme explicitly (Phase 5, completed 2026-06-03)
- [ ] **SEC-01**: Auth middleware supports token rotation or JWT with claims and expiry (deferred)
- [ ] **SEC-03**: Docker Compose does not hardcode credentials in plain text (deferred)

### Integration & Testing ✓

- [x] **INT-01**: Frontend pages connect to working backend endpoints (Phase 6, completed 2026-06-03)
- [x] **INT-02**: E2E integration tests pass cleanly (Phase 6, completed 2026-06-03)
- [x] **INT-03**: Security E2E tests pass cleanly (Phase 6, completed 2026-06-03)
- [x] **INT-04**: Frontend unit tests pass cleanly (Phase 6, completed 2026-06-03)
- [x] **INT-05**: Full closed-loop demo works end-to-end (Phase 6, completed 2026-06-03)

## v1.1 Requirements (当前里程碑)

### Layer 1 — 意图收束 (Intention Containment)

- [ ] **MCP-01**: MCP 服务器身份泛化 — `NewMCPServer` 调用使用的服务器名称和 instructions 改为无法识别项目身份的通用描述
- [ ] **MCP-02**: MCP 工具按业务能力重新分组命名 — 12 组注册函数取消与 `internal/` 包的命名映射，使用通用业务能力名称
- [ ] **MCP-03**: `describe_ontology` 输出裁剪 — 移除 SourceDescriptor（schema/table/primary_key），属性描述仅保留精简字段，不暴露底层架构细节
- [ ] **MCP-04**: `get_system_status` 输出裁剪 — 移除 `table_counts`（表名+行数），仅返回聚合级别的健康状态（如 alert_count 总量）

### Layer 2 — 数据暴露防护 (Data Exposure Control)

- [ ] **MCP-05**: `get_object` 字段级过滤 — 在 handler 层利用 ontology 已有的 `LLMReadable` 和 `sensitivity` 标记做字段级过滤，仅返回允许 Agent 读取的属性
- [ ] **MCP-06**: `get_linked_objects` 遍历控制 — 确认 max_depth ≤ 1 为默认值，或按 link_name 做 allowlist 限制；应用字段级过滤规则
- [ ] **MCP-07**: `search_objects` 结果安全加固 — 设置分页上限，搜索结果仅返回 ID/type/精简摘要，不返回完整对象
- [ ] **MCP-08**: `run_pipeline` 输入加固 — config 参数改为枚举 allowlist（只能选预定义的 pipeline 类型），data_dir 参数移除或固定

### 兼容性

- [ ] **MCP-09**: 所有改造不破坏 Pi Agent 现有集成（工具功能等价，只是输入输出名称/结构可能变化）

## Out of Scope

| Feature | Reason |
|---------|--------|
| SEC-01: JWT/token rotation | Deferred from v1 — single bearer token sufficient for demo |
| SEC-03: Docker Compose credentials | Deferred from v1 — not related to MCP scope |
| 新的 MCP 工具 | 当前只做信息收束，不新增能力 |
| API HTTP handler 改造 | 本里程碑只关注 MCP 层，不涉及 REST API |
| MCP 鉴权机制 | 当前通过 BAXI_MCP_USER_ID/ROLE 环境变量控制，不做额外引入 |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| MCP-01 | Phase 7 | Pending |
| MCP-02 | Phase 7 | Pending |
| MCP-03 | Phase 8 | Pending |
| MCP-04 | Phase 8 | Pending |
| MCP-05 | Phase 9 | Pending |
| MCP-06 | Phase 9 | Pending |
| MCP-07 | Phase 10 | Pending |
| MCP-08 | Phase 10 | Pending |
| MCP-09 | Phase 11 | Pending |

**Coverage:**
- v1.1 requirements: 9 total
- Mapped to phases: 9/9 ✓

---
*Requirements defined: 2026-06-06*
*Last updated: 2026-06-06 after v1.1 milestone initialization*
