# Research: Baxi Ontology v2

**Date**: 2026-06-02
**Status**: Complete

## Phase 0: Research Findings

### R1: v1/v2 共存策略

**Decision**: 按对象独立选择。v2 schema 定义了的对象走 v2（QueryCompiler/LinkResolver/ContextRecipe），未定义的走 v1 fallback（objectTableMap 硬编码）。

**Rationale**:
- 允许渐进式迁移，seller 先验证，其他对象后迁
- 避免一次性迁移所有对象的风险
- v1 代码在 v2 稳定运行 1-2 个 release 后标记 deprecated

**Alternatives Considered**:
- A) 全量迁移：风险太高，改动面太大
- B) 双写双读：复杂度高，维护成本大

---

### R2: Context Recipe 覆盖策略

**Decision**: 按对象类型回退到默认 recipe。recipe 未定义时，使用对象类型的默认 recipe（包含所有 llm_readable 属性和 metrics）。

**Rationale**:
- 避免严格模式导致新告警类型无法工作
- 默认 recipe 保证 LLM 总能获得基本上下文
- 特定 recipe 可以精确控制上下文内容

**Alternatives Considered**:
- A) 严格模式：recipe 不存在则报错 → 不够灵活
- B) 全局默认 recipe：无法针对对象类型优化

---

### R3: Metric 数据源设计

**Decision**: 分层设计。properties 中 agg/expression 用于实时 OLAP 查询（如 seller gmv=SUM(price)），metric_definitions 用于预聚合指标（如 gmv_7d 来自 mart.metric_dimension_daily）。

**Rationale**:
- 两处语义不同：实时 vs 预聚合
- properties 中的 agg 用于 QueryCompiler 生成 SQL
- metric_definitions 中的预聚合指标用于 Context Recipe

**Alternatives Considered**:
- A) 统一到 metric_definitions：无法支持实时查询
- B) 统一到 properties：无法支持预聚合指标

---

### R4: QueryCompiler 安全策略

**Decision**: 白名单校验 + 硬上限。filter 字段必须 filterable=true，sort 字段必须 searchable=true，limit 硬上限 10000。schema/table/column 必须来自 ontology config，禁止任意 SQL 注入。

**Rationale**:
- 防止 SQL 注入攻击
- 确保查询只访问授权的字段
- 限制结果集大小，防止资源耗尽

**Alternatives Considered**:
- A) 黑名单校验：容易遗漏
- B) 无限制：安全风险太高

---

### R5: v1 废弃策略

**Decision**: 渐进废弃。v2 稳定运行 1-2 个 release 后标记 v1 deprecated，再 1 个 release 后删除。过渡期同时维护两套代码。

**Rationale**:
- 给用户时间迁移
- 保持向后兼容
- 逐步减少维护成本

**Alternatives Considered**:
- A) 立即删除：破坏性太大
- B) 永久维护：维护成本高

---

### R6: 现有代码结构分析

**Decision**: 基于现有代码结构扩展，不重构。

**Rationale**:
- 现有代码结构清晰（ontology/repository/decision/mcp）
- 新增文件放在对应子包中
- 修改现有文件时保持接口兼容

**现有代码关键点**:
- `internal/ontology/schema.go`: ObjectType 结构体（v1）
- `internal/ontology/registry.go`: ObjectRegistry（从 DB 或 YAML 加载）
- `internal/repository/ontology/repository.go`: objectTableMap/metricColumns 硬编码
- `internal/decision/context_builder_v2.go`: V2 上下文构建器
- `internal/decision/context_builder_v3.go`: V3 link traversal（depth≤2）
- `internal/mcp/tools_ontology.go`: 4个MCP工具

---

### R7: 技术栈确认

**Decision**: 使用现有技术栈，不引入新依赖。

**Rationale**:
- Go 1.23 + PostgreSQL 15 + pgx/v5 + testify
- YAML 配置文件（与现有 aip_object_schema.yml 一致）
- MCP 工具协议（与现有 tools_ontology.go 一致）

**确认项**:
- YAML 解析：使用现有 `gopkg.in/yaml.v3`
- SQL 生成：手动拼接（与现有代码一致）
- 测试框架：testify（与现有代码一致）
