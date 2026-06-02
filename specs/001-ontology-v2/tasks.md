# Tasks: Baxi Ontology v2 — Executable Semantic Layer

**Input**: Design documents from `/specs/001-ontology-v2/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: The spec explicitly requests tests (FR-009), so test tasks are included.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Go backend**: `internal/`, `config/`, `test/`
- Paths below follow Baxi project structure from plan.md

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: 配置文件创建，为所有用户故事提供 YAML schema 定义

- [X] T001 [P] 创建 v2 对象 schema 配置文件 `config/aip_object_schema_v2.yml`（包含 seller、order、product、metric_alert 4个对象定义）
- [X] T002 [P] 创建指标合同配置文件 `config/metric_definitions.yml`（包含 seller_late_delivery_rate_7d、seller_order_count_7d、seller_gmv_7d 3个指标）
- [X] T003 [P] 创建上下文配方配置文件 `config/context_recipes.yml`（包含 seller_late_delivery_alert 配方）

**Checkpoint**: 配置文件就绪，可被后续 Go 代码加载

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: 核心结构体和解析器，所有用户故事的基础依赖

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T004 创建 v2 schema 结构体定义 `internal/ontology/schema_v2.go`（ObjectTypeV2、ObjectSource、ObjectPropertyV2、ObjectLinkV2、LinkTarget、ObjectGovernancePolicy）
- [X] T005 创建指标合同结构体定义 `internal/ontology/metric_definition.go`（MetricDefinition、MetricSource）
- [X] T006 创建上下文配方结构体定义 `internal/ontology/context_recipe.go`（ContextRecipe、RecipeTrigger、RecipeRootObject、RecipeInclude、RecipeLinkInclude、RecipeBudget、RecipeGovernance）
- [X] T007 创建查询计划结构体定义 `internal/ontology/query_plan.go`（CompiledQuery）
- [X] T008 创建动作提案结构体定义 `internal/ontology/action_binding.go`（ActionProposal、ActionBinding）
- [X] T009 创建 v2 YAML 解析器 `internal/ontology/registry_v2.go`（解析 aip_object_schema_v2.yml、metric_definitions.yml、context_recipes.yml）
- [X] T010 创建 v2 schema 验证器 `internal/ontology/validator_v2.go`（验证 source 完整性、唯一主键、metric_ref 存在性、action binding 存在性）
- [X] T011 扩展现有 ObjectRegistry 支持 v2 加载 `internal/ontology/registry.go`（添加 v2 schema 加载逻辑）
- [X] T012 扩展现有 Validator 支持 v2 验证 `internal/ontology/validator.go`（添加 v2 验证规则）
- [X] T013 编写 v2 验证器测试 `internal/ontology/validator_v2_test.go`（测试 schema 完整性、重复主键、metric 缺失、action 缺失）

**Checkpoint**: 基础结构体和解析器就绪，可开始用户故事实现

---

## Phase 3: User Story 1 — 运营分析师查看卖家异常上下文 (Priority: P1) 🎯 MVP

**Goal**: 按 context recipe 构建精确的 LLM 上下文，包含 seller 属性、指标、最近订单和可用操作

**Independent Test**: 通过 seller_late_delivery_alert case，验证上下文包含 6 个 seller 属性 + 3 个指标 + 10 笔订单 + 3 个可用操作 + 治理摘要

### Tests for User Story 1

- [X] T014 [P] [US1] 编写 ContextRecipe 解析测试 `internal/ontology/context_recipe_test.go`（测试 recipe 加载、trigger 匹配、默认 recipe 回退）
- [X] T015 [P] [US1] 编写 recipe-driven 上下文构建器测试 `internal/decision/context_builder_recipe_test.go`（测试完整上下文构建、脱敏、evidence 生成）

### Implementation for User Story 1

- [X] T016 [P] [US1] 创建查询执行器 `internal/repository/ontology/query_executor.go`（执行 CompiledQuery，替代 objectTableMap）
- [X] T017 [P] [US1] 创建关系执行器 `internal/repository/ontology/link_executor.go`（执行 LinkResolver 查询，支持 one_to_many）
- [X] T018 [US1] 创建 MetricResolver `internal/ontology/metric_resolver.go`（从 metric_definitions 加载指标、查询预聚合值）
- [X] T019 [US1] 创建 recipe-driven 上下文构建器 `internal/decision/context_builder_recipe.go`（匹配 recipe → 加载 properties → 加载 metrics → 加载 links → 应用 governance → 生成 evidence → 注入 actions）
- [X] T020 [US1] 扩展现有 MCP tools_ontology.go 增强 describe_ontology 输出 `internal/mcp/tools_ontology.go`（添加 v2 扩展字段：source、metrics、links、governance）
- [X] T021 [US1] 新增 build_context MCP 工具 `internal/mcp/tools_context.go`（按 case_id + recipe_id 构建上下文）
- [X] T022 [US1] 注册新 MCP 工具 `internal/mcp/server.go`（注册 build_context 工具）

**Checkpoint**: User Story 1 完成，seller_late_delivery_alert 场景可端到端运行

---

## Phase 4: User Story 2 — 通过 Ontology Schema 动态查询卖家 (Priority: P1)

**Goal**: QueryCompiler 从 v2 schema 动态编译 SQL，替代硬编码 objectTableMap

**Independent Test**: 查询 seller 对象，验证 SQL 由 schema 动态生成（含 SELECT 列、聚合、WHERE、GROUP BY、LIMIT）

### Tests for User Story 2

- [X] T023 [P] [US2] 编写 QueryCompiler 测试 `internal/ontology/compiler_test.go`（测试 SQL 生成、filterable 校验、searchable 校验、limit 硬上限）

### Implementation for User Story 2

- [X] T024 [US2] 创建 QueryCompiler `internal/ontology/compiler.go`（CompileGetObject、CompileSearchObjects、CompileObjectMetrics）
- [X] T025 [US2] 扩展现有 repository 添加 v2 fallback 逻辑 `internal/repository/ontology/repository.go`（v2 schema 存在时使用 QueryCompiler，否则回退 objectTableMap）
- [X] T026 [US2] 扩展现有 MCP get_object 支持 v2 查询 `internal/mcp/tools_ontology.go`（使用 QueryCompiler 获取对象详情和指标）

**Checkpoint**: User Story 2 完成，seller 查询不再依赖 objectTableMap

---

## Phase 5: User Story 3 — 通过关系链路查看卖家的最近订单 (Priority: P2)

**Goal**: LinkResolver 支持一对多关系，seller → recent_orders 返回订单列表

**Independent Test**: 查询 seller_123 的 recent_orders，验证返回最多 20 条订单记录（而非单个对象）

### Tests for User Story 3

- [X] T027 [P] [US3] 编写 LinkResolver 测试 `internal/ontology/link_plan_test.go`（测试 reverse_lookup 策略、limit/sort 生效、无效 link_name 报错、max_depth 生效）

### Implementation for User Story 3

- [X] T028 [US3] 创建 LinkResolver `internal/ontology/link_plan.go`（GetLinkedObjects 接口，支持 direct_key/reverse_lookup/bridge_table/query_ref 四种策略）
- [X] T029 [US3] 扩展现有 MCP get_linked_objects 支持 v2 关系 `internal/mcp/tools_ontology.go`（使用 LinkResolver 获取关联对象列表）

**Checkpoint**: User Story 3 完成，seller → recent_orders 返回一对多列表

---

## Phase 6: User Story 4 — 安全的对象级动作执行 (Priority: P2)

**Goal**: 对象级 Action Binding 校验，propose_action 创建动作提案

**Independent Test**: 对 seller 执行 notify_owner，验证对象级绑定、payload 校验、dry-run 行为

### Tests for User Story 4

- [X] T030 [P] [US4] 编写 ActionBinding 校验测试 `internal/ontology/action_binding_test.go`（测试未绑定 action 被拒绝、action_registry 禁用被拒绝、payload 校验、requires_approval 校验、dry-run 行为）

### Implementation for User Story 4

- [X] T031 [US4] 创建 ActionBinding 校验器 `internal/ontology/action_binding_validator.go`（按顺序校验：object_type 存在 → action 绑定 → action 启用 → role 权限 → payload schema → approval 状态）
- [X] T032 [US4] 新增 propose_action MCP 工具 `internal/mcp/tools_context.go`（创建 ActionProposal，不直接执行）
- [X] T033 [US4] 扩展现有 execute_action MCP 工具支持 v2 校验 `internal/mcp/tools_ontology.go`（默认 dry-run，内部转成 proposal）

**Checkpoint**: User Story 4 完成，seller 上的 notify_owner 通过对象级 action binding 控制

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: 集成测试、性能优化、文档更新

- [X] T034 [P] 编写集成测试 `test/integration/ontology_v2_test.go`（端到端测试 seller_late_delivery_alert 场景：case_id → LLMSafeContextEnvelope）
- [X] T035 [P] 扩展 MCP describe_ontology 完整输出 `internal/mcp/tools_ontology.go`（返回所有 v2 对象的完整结构）
- [X] T036 更新 AGENTS.md 文档 `internal/ontology/AGENTS.md`（记录 v2 schema 结构、新组件、配置文件）
- [X] T037 运行 quickstart.md 验证（按 quickstart.md 步骤验证端到端流程）

**Checkpoint**: Ontology v2 阶段完成，所有验收标准满足

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-6)**: All depend on Foundational phase completion
  - US1 (P1) 和 US2 (P1) 可并行
  - US3 (P2) 和 US4 (P2) 可并行
  - US3/US4 依赖 US1/US2 的部分组件（QueryCompiler、LinkResolver）
- **Polish (Phase 7)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - 依赖 T016/T017（query_executor/link_executor）
- **User Story 2 (P1)**: Can start after Foundational (Phase 2) - 依赖 T024（QueryCompiler）
- **User Story 3 (P2)**: Can start after Foundational (Phase 2) - 依赖 T028（LinkResolver）
- **User Story 4 (P2)**: Can start after Foundational (Phase 2) - 依赖 T031（ActionBindingValidator）

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- Models before services
- Services before endpoints
- Core implementation before integration
- Story complete before moving to next priority

### Parallel Opportunities

- T001/T002/T003: 配置文件创建可并行
- T004-T008: 结构体定义可并行
- T014/T015: US1 测试可并行
- T016/T017: US1 执行器可并行
- T023: US2 测试可并行
- T027: US3 测试可并行
- T030: US4 测试可并行
- T034/T035: 集成测试可并行

---

## Parallel Example: User Story 1

```bash
# Launch all tests for User Story 1 together:
Task: "编写 ContextRecipe 解析测试 internal/ontology/context_recipe_test.go"
Task: "编写 recipe-driven 上下文构建器测试 internal/decision/context_builder_recipe_test.go"

# Launch all executors for User Story 1 together:
Task: "创建查询执行器 internal/repository/ontology/query_executor.go"
Task: "创建关系执行器 internal/repository/ontology/link_executor.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (配置文件)
2. Complete Phase 2: Foundational (核心结构体)
3. Complete Phase 3: User Story 1 (Context Recipe)
4. **STOP and VALIDATE**: Test seller_late_delivery_alert 场景
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Deploy/Demo (MVP!)
3. Add User Story 2 → Test independently → Deploy/Demo
4. Add User Story 3 → Test independently → Deploy/Demo
5. Add User Story 4 → Test independently → Deploy/Demo
6. Polish → Integration tests → Documentation

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 (Context Recipe)
   - Developer B: User Story 2 (QueryCompiler)
3. Then:
   - Developer A: User Story 3 (LinkResolver)
   - Developer B: User Story 4 (ActionBinding)
4. Stories complete and integrate independently

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Verify tests fail before implementing
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence

---

## Summary

| Phase | Tasks | Description |
|-------|-------|-------------|
| Phase 1: Setup | T001-T003 | 3 tasks (配置文件) |
| Phase 2: Foundational | T004-T013 | 10 tasks (核心结构体和解析器) |
| Phase 3: US1 (P1) | T014-T022 | 9 tasks (Context Recipe) |
| Phase 4: US2 (P1) | T023-T026 | 4 tasks (QueryCompiler) |
| Phase 5: US3 (P2) | T027-T029 | 3 tasks (LinkResolver) |
| Phase 6: US4 (P2) | T030-T033 | 4 tasks (ActionBinding) |
| Phase 7: Polish | T034-T037 | 4 tasks (集成测试) |
| **Total** | **37 tasks** | |

**MVP Scope**: Phase 1 + Phase 2 + Phase 3 (22 tasks)
**Full Delivery**: All 37 tasks
