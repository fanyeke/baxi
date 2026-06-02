# Data Model: Baxi Ontology v2

**Date**: 2026-06-02
**Status**: Complete

## Core Entities

### ObjectTypeV2

业务对象的完整语义定义，是查询、关系、指标、上下文和动作的统一源头。

**YAML 表示**:
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
    properties: { ... }
    metrics: [ ... ]
    relationships: { ... }
    actions: [ ... ]
    alert_fields: [ ... ]
    governance:
      default_role: agent_readonly
      redact_pii: true
```

**Go 结构体**:
```go
type ObjectTypeV2 struct {
    Name           string
    DisplayName    string
    Description    string
    Grain          string
    Source         ObjectSource
    Properties     map[string]ObjectPropertyV2
    Metrics        []string
    Links          []ObjectLinkV2
    AllowedActions []string
    LLMAccess      LLMAccessPolicy
    AlertFields    []string
    Governance     ObjectGovernancePolicy
}
```

**验证规则**:
- Name 必须唯一
- Source.Schema/Table/PrimaryKey 必须非空
- Properties 中只能有一个 is_pk=true
- Metrics 中的引用必须在 metric_definitions.yml 中存在
- AllowedActions 中的引用必须在 action_registry.yml 中存在

---

### ObjectPropertyV2

对象属性定义，支持多种来源和聚合方式。

**YAML 表示**:
```yaml
properties:
  seller_id:
    type: string
    source: seller_id
    is_pk: true
    sensitivity: L2
    llm_readable: false
    searchable: true
  gmv:
    type: float
    source: price
    agg: sum
    sensitivity: L0
    llm_readable: true
  late_delivery_rate:
    type: float
    metric_ref: seller_late_delivery_rate_7d
    sensitivity: L0
    llm_readable: true
```

**Go 结构体**:
```go
type ObjectPropertyV2 struct {
    Name        string
    Type        string      // string, int, float, bool, timestamp
    SourceField string      // 原始表字段名
    Expression  string      // SQL 表达式（如 AVG(order_level.review_score)）
    MetricRef   string      // 引用 metric_definitions 中的指标
    Sensitivity string      // L0, L1, L2, L3
    Aggregation string      // sum, count, count_distinct, avg, min, max
    LLMReadable bool        // 是否可出现在 LLM 上下文
    Searchable  bool        // 是否可搜索
    Filterable  bool        // 是否可过滤
    IsPK        bool        // 是否为主键
}
```

**聚合类型**:
- `sum`: 求和
- `count`: 计数
- `count_distinct`: 去重计数
- `avg`: 平均值
- `min`: 最小值
- `max`: 最大值

---

### ObjectLinkV2

对象关系定义，支持多种 cardinality 和解析策略。

**YAML 表示**:
```yaml
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
    fields:
      - order_id
      - order_status
      - payment_value
```

**Go 结构体**:
```go
type ObjectLinkV2 struct {
    Name        string
    DisplayName string
    TargetType  string      // 目标对象类型
    Cardinality string      // one_to_one, one_to_many, many_to_many
    Strategy    string      // direct_key, reverse_lookup, bridge_table, query_ref
    SourceKey   string      // 源对象字段名
    Target      LinkTarget
    Limit       int         // 默认限制
    Sort        string      // 默认排序
    Fields      []string    // 返回字段列表
}

type LinkTarget struct {
    Schema        string
    Table         string
    Key           string      // 目标表关联字段
    ObjectIDField string      // 目标对象 ID 字段
}
```

**解析策略**:
- `direct_key`: 源对象字段直接指向目标对象主键
- `reverse_lookup`: 目标表里有 source key
- `bridge_table`: 通过中间表关联
- `query_ref`: 使用预定义查询模板

---

### MetricDefinition

指标合同，定义指标的计算逻辑、数据源、严重级别和解释说明。

**YAML 表示**:
```yaml
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

**Go 结构体**:
```go
type MetricDefinition struct {
    Name            string
    DisplayName     string
    ObjectType      string
    Grain           string
    Source          MetricSource
    Filters         map[string]string
    ValueColumn     string
    BaselineColumn  string
    Severity        map[string]string  // medium/high 表达式
    LLMExplanation  string
}

type MetricSource struct {
    Schema string
    Table  string
}
```

---

### ContextRecipe

上下文配方，定义特定告警场景下需要包含哪些对象属性、指标、关联证据和可用操作。

**YAML 表示**:
```yaml
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
      root_properties: [ ... ]
      metrics: [ ... ]
      links: { ... }
      actions: [ ... ]
    budget:
      max_link_depth: 2
      max_objects: 30
      max_tokens_hint: 4000
    governance:
      role: agent_readonly
      redact_pii: true
```

**Go 结构体**:
```go
type ContextRecipe struct {
    Name        string
    Description string
    Trigger     RecipeTrigger
    RootObject  RecipeRootObject
    Include     RecipeInclude
    Budget      RecipeBudget
    Governance  RecipeGovernance
}

type RecipeTrigger struct {
    ObjectType string
    RuleID     string
}

type RecipeRootObject struct {
    TypeFrom string  // e.g., "alert.object_type"
    IDFrom   string  // e.g., "alert.object_id"
}

type RecipeInclude struct {
    RootProperties []string
    Metrics        []string
    Links          map[string]RecipeLinkInclude
    Actions        []string
}

type RecipeLinkInclude struct {
    Limit  int
    Fields []string
}

type RecipeBudget struct {
    MaxLinkDepth  int
    MaxObjects    int
    MaxTokensHint int
}

type RecipeGovernance struct {
    Role      string
    RedactPII bool
}
```

---

### CompiledQuery

从 Ontology schema 编译出的安全查询计划。

**Go 结构体**:
```go
type CompiledQuery struct {
    SQL        string
    Args       []any
    Columns    []string
    ObjectType string
    PrimaryKey string
}
```

---

### ActionProposal

动作提案，包含目标对象、操作类型、payload、风险级别和审批状态。

**Go 结构体**:
```go
type ActionProposal struct {
    ID              string
    ObjectType      string
    ObjectID        string
    ActionType      string
    Payload         map[string]any
    RiskLevel       string      // low, medium, high
    RequiresApproval bool
    Status          string      // pending, approved, rejected, executed, dry_run
    CreatedBy       string
    CreatedAt       time.Time
    ApprovedBy      string
    ApprovedAt      *time.Time
    ExecutedAt      *time.Time
}
```

---

## Relationships

```
ObjectTypeV2
  ├── Properties (1:N) → ObjectPropertyV2
  ├── Metrics (1:N) → MetricDefinition (via metric_ref)
  ├── Links (1:N) → ObjectLinkV2
  └── Actions (1:N) → ActionBinding (via action_registry)

ContextRecipe
  ├── Trigger → ObjectTypeV2 + RuleID
  ├── RootObject → ObjectTypeV2
  ├── Include.Properties → ObjectPropertyV2
  ├── Include.Metrics → MetricDefinition
  ├── Include.Links → ObjectLinkV2
  └── Include.Actions → ActionBinding

ActionProposal
  ├── ObjectType → ObjectTypeV2
  └── ActionType → ActionBinding (via action_registry)
```

---

## State Transitions

### ActionProposal 状态机

```
pending → approved → executed
pending → rejected
pending → dry_run
```

### Context Recipe 匹配流程

```
alert.rule_id
  → 匹配 Recipe.Trigger.RuleID
  → 确定 RootObject (type_from + id_from)
  → 加载 RootObject.Properties
  → 加载 Metrics
  → 加载 Links
  → 应用 Governance/Redaction
  → 生成 LLMSafeContextEnvelope
```
