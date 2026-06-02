# Quickstart: Baxi Ontology v2

**Date**: 2026-06-02
**Status**: Complete

## 1. 配置文件

### 1.1 创建 `config/aip_object_schema_v2.yml`

定义 v2 格式的对象 schema。示例（seller）:

```yaml
version: ontology.v2
objects:
  seller:
    display_name: 卖家
    description: 电商平台上的销售主体
    grain: seller_id
    source:
      schema: dwd
      table: item_level
      primary_key: seller_id
    properties:
      seller_id:
        type: string
        source: seller_id
        is_pk: true
        sensitivity: L2
        llm_readable: false
        searchable: true
      seller_state:
        type: string
        source: seller_state
        sensitivity: L0
        llm_readable: true
        filterable: true
      gmv:
        type: float
        source: price
        agg: sum
        sensitivity: L0
        llm_readable: true
    metrics:
      - seller_late_delivery_rate_7d
      - seller_order_count_7d
      - seller_gmv_7d
    relationships:
      recent_orders:
        display_name: 最近订单
        to: order
        cardinality: one_to_many
        strategy: reverse_lookup
        source_key: seller_id
        target:
          schema: dwd
          table: item_level
          key: seller_id
          object_id_field: order_id
        limit: 20
        sort: order_purchase_timestamp desc
    actions:
      - notify_owner
      - create_followup_task
      - export_report
    alert_fields:
      - late_delivery_rate
      - avg_review_score
      - order_count
    governance:
      default_role: agent_readonly
      redact_pii: true
```

### 1.2 创建 `config/metric_definitions.yml`

定义指标合同:

```yaml
version: metrics.v1
metrics:
  seller_late_delivery_rate_7d:
    display_name: 最近 7 天延迟发货率
    object_type: seller
    grain: seller_id
    source:
      schema: mart
      table: metric_dimension_daily
    filters:
      dimension_type: seller
      metric_name: late_delivery_rate_7d
    value_column: current_value
    baseline_column: baseline_value
    severity:
      medium: "current_value > baseline_value + 0.10"
      high: "current_value > baseline_value + 0.20"
    llm_explanation: >
      衡量卖家近期履约稳定性。显著高于基线时，可能存在库存、发货或物流问题。
```

### 1.3 创建 `config/context_recipes.yml`

定义上下文配方:

```yaml
version: context_recipes.v1
recipes:
  seller_late_delivery_alert:
    description: 卖家延迟发货异常上下文
    trigger:
      object_type: metric_alert
      rule_id: seller_late_delivery_spike
    root_object:
      type_from: alert.object_type
      id_from: alert.object_id
    include:
      root_properties:
        - seller_state
        - seller_city
        - gmv
        - order_count
        - avg_review_score
        - late_delivery_rate
      metrics:
        - seller_late_delivery_rate_7d
        - seller_order_count_7d
        - seller_gmv_7d
      links:
        recent_orders:
          limit: 10
          fields:
            - order_id
            - order_status
            - payment_value
            - review_score
            - delivery_status
      actions:
        - notify_owner
        - create_followup_task
        - export_report
    budget:
      max_link_depth: 2
      max_objects: 30
      max_tokens_hint: 4000
    governance:
      role: agent_readonly
      redact_pii: true
```

---

## 2. 代码实现

### 2.1 新增结构体 (`internal/ontology/schema_v2.go`)

定义 ObjectTypeV2、ObjectPropertyV2、ObjectLinkV2 等结构体。

### 2.2 新增解析器 (`internal/ontology/registry_v2.go`)

解析 v2 格式的 YAML 文件。

### 2.3 新增 QueryCompiler (`internal/ontology/compiler.go`)

从 v2 schema 编译 SQL 查询。

### 2.4 新增 LinkResolver (`internal/ontology/link_plan.go`)

解析对象关系（支持 one_to_many）。

### 2.5 新增 MetricResolver (`internal/ontology/metric_definition.go`)

解析 metric_definitions.yml。

### 2.6 新增 ContextRecipeResolver (`internal/ontology/context_recipe.go`)

解析 context_recipes.yml。

### 2.7 新增 ActionBindingValidator (`internal/ontology/action_binding.go`)

校验对象级动作绑定。

### 2.8 新增 V2 验证器 (`internal/ontology/validator_v2.go`)

验证 v2 schema 的完整性。

### 2.9 新增查询执行器 (`internal/repository/ontology/query_executor.go`)

执行 CompiledQuery。

### 2.10 新增关系执行器 (`internal/repository/ontology/link_executor.go`)

执行 LinkResolver 查询。

### 2.11 新增上下文构建器 (`internal/decision/context_builder_recipe.go`)

基于 ContextRecipe 构建 LLM 上下文。

### 2.12 新增 MCP 工具 (`internal/mcp/tools_context.go`)

新增 build_context MCP 工具。

---

## 3. 测试

### 3.1 Schema 验证测试 (`internal/ontology/validator_v2_test.go`)

验证 v2 schema 的完整性规则。

### 3.2 QueryCompiler 测试 (`internal/ontology/compiler_test.go`)

验证 SQL 生成的正确性。

### 3.3 LinkResolver 测试 (`internal/ontology/link_plan_test.go`)

验证关系解析的正确性。

### 3.4 ContextRecipe 测试 (`internal/ontology/context_recipe_test.go`)

验证上下文配方的解析。

### 3.5 ActionBinding 测试 (`internal/ontology/action_binding_test.go`)

验证动作绑定的校验。

### 3.6 集成测试 (`test/integration/ontology_v2_test.go`)

端到端测试 seller_late_delivery_alert 场景。

---

## 4. 运行

### 4.1 启动服务

```bash
# 启动 PostgreSQL
make up

# 运行迁移
make migrate

# 启动 API 服务
make api

# 启动 Worker
make worker
```

### 4.2 测试 MCP 工具

```bash
# 描述 Ontology
curl -X POST http://localhost:8080/mcp/tools/describe_ontology

# 获取卖家对象
curl -X POST http://localhost:8080/mcp/tools/get_object \
  -H "Content-Type: application/json" \
  -d '{"object_type": "seller", "object_id": "seller_123"}'

# 获取关联订单
curl -X POST http://localhost:8080/mcp/tools/get_linked_objects \
  -H "Content-Type: application/json" \
  -d '{"object_type": "seller", "object_id": "seller_123", "link_name": "recent_orders"}'

# 构建上下文
curl -X POST http://localhost:8080/mcp/tools/build_context \
  -H "Content-Type: application/json" \
  -d '{"case_id": "case_456"}'
```

---

## 5. 验收标准

1. ✅ seller、order、product、metric_alert 至少 4 个对象由 v2 schema 驱动
2. ✅ seller 查询不再依赖 objectTableMap
3. ✅ seller → recent_orders 关系返回多条订单
4. ✅ seller_late_delivery_alert 上下文包含完整证据链
5. ✅ 未绑定 action 被拒绝
6. ✅ 核心路径测试覆盖率 ≥80%
