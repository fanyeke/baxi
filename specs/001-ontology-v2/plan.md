# Implementation Plan: Baxi Ontology v2 — Executable Semantic Layer

**Branch**: `001-ontology-v2` | **Date**: 2026-06-02 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-ontology-v2/spec.md`

## Summary

将 Baxi Ontology 从对象描述层升级为可执行语义层。核心改造包括：
1. 扩展 Ontology schema v2（支持 source/properties/metrics/relationships/actions/governance）
2. 新增 QueryCompiler 替代硬编码 objectTableMap
3. 新增 LinkResolver 支持一对多关系（reverse_lookup/bridge_table/query_ref）
4. 新增 Metric Contract（metric_definitions.yml）替代硬编码 metricColumns
5. 新增 Context Recipe 驱动上下文构建
6. 新增对象级 Action Binding

以 seller_late_delivery_alert 场景跑通最小闭环。

## Technical Context

**Language/Version**: Go 1.23 (golang:1.23-alpine)
**Primary Dependencies**: chi (router), pgx/v5 (PostgreSQL driver), testify (testing)
**Storage**: PostgreSQL 15 (docker compose), goose migrations
**Testing**: go test + testify
**Target Platform**: Linux server (Docker)
**Project Type**: Go backend + React frontend (baxi-api, baxi-worker, baxi-cli, baxi-mcp)
**Performance Goals**: LLM context 构建 ≤2s, 查询性能与硬编码方案差异 ≤10%
**Constraints**: 保持 v1 向后兼容，渐进式迁移
**Scale/Scope**: 8+ 对象类型，4 个 MCP 工具增强

**现有代码结构**:
- `internal/ontology/` — 15个Go文件（schema.go, registry.go, query_service.go, validator.go 等）
- `internal/repository/ontology/repository.go` — 硬编码 objectTableMap/metricColumns
- `internal/decision/context_builder_v2.go` — V2 上下文构建器
- `internal/decision/context_builder_v3.go` — V3 link traversal（depth≤2）
- `internal/mcp/tools_ontology.go` — 4个MCP工具
- `config/action_registry.yml` — 4个canonical actions
- `config/aip_object_schema.yml` — 8个对象类型定义

**关键差距**:
- ObjectLink 仅 Name/TargetType/Via，无法表达一对多
- objectTableMap/metricColumns 硬编码与 YAML schema 分离
- 全局 action binding，无对象级显式绑定
- Context 构建取全部非脱敏字段，无场景化 recipe

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**状态**: Constitution 未配置（模板状态），无特定 gates 需要检查。项目遵循 AGENTS.md 中的约定。

## Project Structure

### Documentation (this feature)

```text
specs/001-ontology-v2/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
# 新增文件
config/
├── aip_object_schema_v2.yml    # 扩展的 v2 schema（4个对象）
├── metric_definitions.yml      # 指标合同定义
└── context_recipes.yml         # 上下文配方

internal/ontology/
├── schema_v2.go                # ObjectTypeV2/ObjectPropertyV2/ObjectLinkV2 等结构体
├── registry_v2.go              # V2 YAML 解析器
├── compiler.go                 # QueryCompiler 接口和实现
├── query_plan.go               # CompiledQuery 结构体
├── link_plan.go                # LinkResolver 接口和实现
├── metric_definition.go        # MetricDefinition 结构体和解析
├── context_recipe.go           # ContextRecipe 结构体和解析
├── action_binding.go           # ActionBinding 结构体和校验
└── validator_v2.go             # V2 schema 验证器

internal/repository/ontology/
├── query_executor.go           # CompiledQuery 执行器（替代 objectTableMap）
└── link_executor.go            # LinkResolver 执行器

internal/decision/
└── context_builder_recipe.go   # Recipe-driven 上下文构建器

internal/mcp/
├── tools_context.go            # 新增 build_context 工具
└── tools_ontology.go           # 增强 describe_ontology/get_linked_objects

# 修改文件
internal/ontology/
├── registry.go                 # 扩展支持 v2 schema 加载
└── validator.go                # 扩展支持 v2 验证

internal/repository/ontology/
└── repository.go               # 添加 v2 fallback 逻辑

internal/mcp/
└── server.go                   # 注册新工具
```

**Structure Decision**: 使用 Go backend 结构，新增文件放在对应 internal 子包中。配置文件放在 config/ 目录。测试文件与实现文件同目录。

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

无 violations。所有设计决策都有明确的技术理由：
- 新增 v2 schema 文件而非修改 v1：保持向后兼容
- 新增 compiler/link_executor 替代硬编码：提升可维护性
- 分离 metric_definitions/context_recipes：关注点分离
