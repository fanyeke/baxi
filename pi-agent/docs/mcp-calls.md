# Baxi MCP 工具调用文档

本文档记录了 Pi Agent 可用的所有 Baxi MCP 工具，包括参数定义、示例调用、响应格式以及决策工作流中的使用场景。

---

## 目录

1. [describe_ontology](#1-describe_ontology)
2. [get_object](#2-get_object)
3. [get_linked_objects](#3-get_linked_objects)
4. [build_context](#4-build_context)
5. [get_decision_context](#5-get_decision_context)
6. [list_action_schemas](#6-list_action_schemas)

---

## 1. describe_ontology

### 描述

描述所有已注册的 AIP 对象类型，包括每个类型的属性定义、关系链接和允许的操作。相当于 Ontology v2 的元数据自省接口。

### 参数

| 参数 | 类型 | 必填 | 描述 |
|-------|------|----------|-------------|
| 无 | — | — | 该工具不接受任何参数 |

### 示例调用

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "describe_ontology"
  }
}
```

### 示例响应

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"object_types\":[{\"name\":\"seller\",\"display_name\":\"卖家\",\"grain\":\"seller_id\",\"source\":{\"schema\":\"dwd\",\"table\":\"item_level\",\"primary_key\":\"seller_id\"},\"properties\":[{\"name\":\"seller_id\",\"type\":\"string\",\"sensitivity\":\"L2\",\"llm_readable\":false,\"is_pk\":true},{\"name\":\"seller_state\",\"type\":\"string\",\"sensitivity\":\"L0\",\"llm_readable\":true,\"is_pk\":false},{\"name\":\"seller_city\",\"type\":\"string\",\"sensitivity\":\"L0\",\"llm_readable\":true,\"is_pk\":false},{\"name\":\"gmv\",\"type\":\"float\",\"sensitivity\":\"L0\",\"llm_readable\":true,\"is_pk\":false},{\"name\":\"order_count\",\"type\":\"int\",\"sensitivity\":\"L0\",\"llm_readable\":true,\"is_pk\":false},{\"name\":\"avg_review_score\",\"type\":\"float\",\"sensitivity\":\"L0\",\"llm_readable\":true,\"is_pk\":false},{\"name\":\"late_delivery_rate\",\"type\":\"float\",\"sensitivity\":\"L0\",\"llm_readable\":true,\"is_pk\":false}],\"metrics\":[\"seller_late_delivery_rate_7d\",\"seller_order_count_7d\",\"seller_gmv_7d\"],\"links\":[{\"name\":\"recent_orders\",\"target_type\":\"order\",\"via\":\"reverse_lookup:dwd.item_level.seller_id\"},{\"name\":\"products\",\"target_type\":\"product\",\"via\":\"reverse_lookup:dwd.item_level.seller_id\"}],\"allowed_actions\":[\"notify_owner\",\"create_followup_task\",\"export_report\"],\"llm_access\":{\"can_read\":true,\"can_write\":false,\"read_only\":true},\"governance\":{\"default_role\":\"agent_readonly\",\"redact_pii\":true}},{\"name\":\"order\",\"display_name\":\"订单\",\"grain\":\"order_id\",\"source\":{\"schema\":\"dwd\",\"table\":\"order_level\",\"primary_key\":\"order_id\"},\"properties\":[{\"name\":\"order_id\",\"type\":\"string\",\"sensitivity\":\"L2\",\"llm_readable\":false,\"is_pk\":true},{\"name\":\"order_status\",\"type\":\"string\",\"sensitivity\":\"L0\",\"llm_readable\":true,\"is_pk\":false},{\"name\":\"payment_value\",\"type\":\"float\",\"sensitivity\":\"L1\",\"llm_readable\":true,\"is_pk\":false},{\"name\":\"review_score\",\"type\":\"float\",\"sensitivity\":\"L0\",\"llm_readable\":true,\"is_pk\":false},{\"name\":\"delivery_status\",\"type\":\"string\",\"sensitivity\":\"L0\",\"llm_readable\":true,\"is_pk\":false}],\"links\":[{\"name\":\"customer\",\"target_type\":\"customer\",\"via\":\"query_ref\"},{\"name\":\"seller\",\"target_type\":\"seller\",\"via\":\"reverse_lookup:dwd.item_level.order_id\"},{\"name\":\"product\",\"target_type\":\"product\",\"via\":\"reverse_lookup:dwd.item_level.order_id\"},{\"name\":\"payment\",\"target_type\":\"payment\",\"via\":\"direct_key:dwd.order_level.order_id\"},{\"name\":\"shipment\",\"target_type\":\"shipment\",\"via\":\"direct_key:dwd.order_level.order_id\"},{\"name\":\"reviews\",\"target_type\":\"review\",\"via\":\"direct_key:dwd.order_level.order_id\"}],\"allowed_actions\":[\"create_followup_task\"],\"llm_access\":{\"can_read\":true,\"can_write\":false,\"read_only\":true},\"governance\":{\"default_role\":\"agent_readonly\",\"redact_pii\":false}},{\"name\":\"product\",\"display_name\":\"产品\",\"grain\":\"product_id\",\"source\":{\"schema\":\"dwd\",\"table\":\"item_level\",\"primary_key\":\"product_id\"},...}]}"
      }
    ]
  }
}
```

### 决策工作流中的使用时机

- **初始探索阶段**：当 Pi Agent 第一次连接到 MCP 服务器，需要了解可用的领域对象类型时调用
- **发现可用字段**：了解每个对象类型有哪些属性（properties）及其敏感性分级（L0/L1/L2/L3）和 LLM 可读性
- **发现关系图**：查看对象之间的链接关系（links），用于规划后续 `get_linked_objects` 调用链
- **发现允许的操作**：查看每个对象类型上定义了哪些 action 操作
- **上下文构建前**：在调用 `build_context` 或 `get_decision_context` 之前，先确认目标 case 涉及的对象类型结构

### 常见错误

1. **在不必要的时机重复调用**：`describe_ontology` 返回的是静态元数据，建议在连接时调用一次并缓存结果，而不是每次决策前都调用
2. **忽略 llm_readable 标记**：属性上标记了 `llm_readable: false` 的字段（如主键 `seller_id`）不应出现在传递给 LLM 的上下文中
3. **忽略 sensitivity 标记**：L2 和 L3 字段受治理策略约束，在决策上下文中可能被自动脱敏

---

## 2. get_object

### 描述

按对象类型和 ID 获取单个对象的详细数据，包括所有属性和可选的计算指标（metrics）。

### 参数

| 参数 | 类型 | 必填 | 描述 |
|-----------|------|----------|-------------|
| object_type | string | 是 | 要检索的对象类型名称（如 `seller`、`order`、`product`、`category`、`region`、`customer`、`marketing_lead`、`metric_alert`） |
| object_id | string | 是 | 要检索的对象 ID |

### 示例调用：获取卖家信息

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "get_object",
    "arguments": {
      "object_type": "seller",
      "object_id": "SELLER001"
    }
  }
}
```

### 示例响应

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"object_type\":\"seller\",\"object_id\":\"SELLER001\",\"properties\":{\"seller_state\":\"SP\",\"seller_city\":\"Sao Paulo\",\"gmv\":158000.50,\"order_count\":342,\"avg_review_score\":4.2,\"late_delivery_rate\":0.08},\"metrics\":{\"seller_gmv_7d\":28500.00,\"seller_order_count_7d\":45,\"seller_late_delivery_rate_7d\":0.12}}"
      }
    ]
  }
}
```

### 示例调用：获取订单信息

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "get_object",
    "arguments": {
      "object_type": "order",
      "object_id": "ORD-20240301-001"
    }
  }
}
```

### 示例响应

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"object_type\":\"order\",\"object_id\":\"ORD-20240301-001\",\"properties\":{\"order_status\":\"delivered\",\"payment_value\":199.90,\"review_score\":4.5,\"delivery_status\":\"on_time\"}}"
      }
    ]
  }
}
```

### 决策工作流中的使用时机

- **获取触发告警的对象详情**：当告警涉及某个 seller/order/product 时，用 `get_object` 获取其详细信息
- **检查关键指标**：返回的 `metrics` 字段包含预聚合指标（如 7 天 GMV、订单量），用于判断异常严重程度
- **作为上下文构建的补充**：调用 `build_context` 或 `get_decision_context` 后，如果发现某些数据缺失，可以手动用 `get_object` 补充
- **验证治理脱敏**：确认脱敏规则是否正确应用到敏感字段

### 常见错误

1. **混淆 object_type 名称**：类型名称是英文小写，如 `seller` 而非 `sellers`；注意 `metric_alert` 而非 `alert`
2. **忽略 ID 格式**：不同对象类型的 ID 格式可能不同（如 `SELLER001` vs `ORD-...`），需与告警数据中的格式一致
3. **忽略敏感性标记**：标记为 `llm_readable: false` 的属性（如主键）仍会返回，但不应作为决策依据传入 LLM

---

## 3. get_linked_objects

### 描述

获取与指定对象通过关系（relationships）相关联的对象。支持按链接名称过滤和控制遍历深度，用于在对象关系图中进行上下文探索。

### 参数

| 参数 | 类型 | 必填 | 描述 |
|-----------|------|----------|-------------|
| object_type | string | 是 | 源对象的类型名称 |
| object_id | string | 是 | 源对象的 ID |
| link_name | string | 否 | 按链接名称过滤（如 `recent_orders`、`products`、`seller`），省略时返回所有链接 |
| max_depth | number | 否 | 关系遍历深度（默认：1，最大：3） |

### 示例调用：获取卖家的最近订单

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "tools/call",
  "params": {
    "name": "get_linked_objects",
    "arguments": {
      "object_type": "seller",
      "object_id": "SELLER001",
      "link_name": "recent_orders",
      "max_depth": 1
    }
  }
}
```

### 示例响应

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"object_type\":\"seller\",\"object_id\":\"SELLER001\",\"links\":[{\"link_name\":\"recent_orders\",\"target_type\":\"order\",\"objects\":[{\"object_type\":\"order\",\"object_id\":\"ORD-20240315-003\",\"properties\":{\"order_status\":\"delivered\",\"payment_value\":299.00,\"review_score\":2.0,\"delivery_status\":\"late\"}},{\"object_type\":\"order\",\"object_id\":\"ORD-20240314-001\",\"properties\":{\"order_status\":\"delivered\",\"payment_value\":150.00,\"review_score\":4.0,\"delivery_status\":\"on_time\"}}]}]}"
      }
    ]
  }
}
```

### 示例调用：获取订单的所有关联对象（客户、卖家和产品）

```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "method": "tools/call",
  "params": {
    "name": "get_linked_objects",
    "arguments": {
      "object_type": "order",
      "object_id": "ORD-20240301-001",
      "max_depth": 1
    }
  }
}
```

### 示例响应

```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"object_type\":\"order\",\"object_id\":\"ORD-20240301-001\",\"links\":[{\"link_name\":\"seller\",\"target_type\":\"seller\",\"objects\":[{\"object_type\":\"seller\",\"object_id\":\"SELLER001\",\"properties\":{\"seller_state\":\"SP\",\"seller_city\":\"Sao Paulo\",\"gmv\":158000.50,\"avg_review_score\":4.2}}]},{\"link_name\":\"customer\",\"target_type\":\"customer\",\"objects\":[{\"object_type\":\"customer\",\"object_id\":\"CUST001\",\"properties\":{\"customer_state\":\"SP\",\"customer_city\":\"Sao Paulo\",\"order_count_90d\":5,\"gmv_90d\":1250.00}}]},{\"link_name\":\"product\",\"target_type\":\"product\",\"objects\":[{\"object_type\":\"product\",\"object_id\":\"PROD-001\",\"properties\":{\"product_category_name\":\"eletronicos\",\"product_category_name_english\":\"electronics\",\"price\":199.90,\"avg_review_score\":4.5}}]}]}"
      }
    ]
  }
}
```

### 决策工作流中的使用时机

- **根因分析**：当告警涉及 seller 的 `late_delivery_rate` 异常时，调用 `get_linked_objects` 获取 `recent_orders` 查看哪些具体订单延迟
- **上下文丰富**：在 `get_decision_context` 返回的上下文不足以支持决策时，手动获取关联对象补充证据
- **级联影响分析**：从 seller 出发 —> recent_orders —> order.seller —> seller，追踪影响范围
- **分步探索**：对复杂 case，先用 `max_depth=1` 获取直接关联，再基于结果中的目标对象 ID 分步探索

### 常见错误

1. **过度使用 max_depth=3**：深度 2 和 3 会增加数据库查询量并可能返回大量数据，建议大部分场景用默认深度 1，仅在确实需要跨级查询时使用深度 2
2. **不指定 link_name**：如果只想获取某一类关联对象（如 `recent_orders`），务必指定 `link_name` 来减少返回数据量
3. **忽略 Cardinality**：`one_to_many` 关系的链接可能返回多个对象，而 `many_to_one`（如 `order.seller`）通常只返回一个，不要假设数量

---

## 4. build_context

### 描述

为决策 case 构建一个 LLM-safe 上下文信封（context envelope）。该信封是版本化、可审计的封装，包含脱敏后的对象上下文、证据项、治理元数据、脱敏摘要、提示版本和配置版本。这是将原始运营数据转化为安全、可审计的 LLM 输入的桥梁。

### 参数

| 参数 | 类型 | 必填 | 描述 |
|-----------|------|----------|-------------|
| case_id | string | 是 | 要构建上下文的决策 case ID |
| recipe_id | string | 否 | 指定使用的 ContextRecipe ID（未指定时按 rule_id 自动匹配） |

### 示例调用

```json
{
  "jsonrpc": "2.0",
  "id": 6,
  "method": "tools/call",
  "params": {
    "name": "build_context",
    "arguments": {
      "case_id": "case-abc123",
      "recipe_id": "seller_late_delivery_recipe"
    }
  }
}
```

### 示例响应

```json
{
  "jsonrpc": "2.0",
  "id": 6,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"case_id\":\"case-abc123\",\"alert_id\":\"alert-001\",\"schema_version\":\"1.0\",\"context_hash\":\"a1b2c3d4e5f6...\",\"built_at\":\"2026-06-02T10:30:00Z\",\"trigger\":{\"alert_id\":\"alert-001\",\"rule_id\":\"seller_late_delivery_rate_7d\",\"severity\":\"high\",\"metric_name\":\"seller_late_delivery_rate_7d\",\"current_value\":0.12,\"baseline_value\":0.05,\"delta_pct\":140.0},\"object_context\":{\"object_type\":\"seller\",\"object_id\":\"SELLER001\",\"properties\":{\"seller_state\":\"SP\",\"seller_city\":\"Sao Paulo\",\"gmv\":158000.50,\"order_count\":342,\"avg_review_score\":4.2,\"late_delivery_rate\":0.08}},\"evidence\":[{\"type\":\"metric\",\"key\":\"seller_gmv_7d\",\"value\":28500.0},{\"type\":\"metric\",\"key\":\"seller_order_count_7d\",\"value\":45}],\"allowed_actions\":[\"create_followup_task\",\"notify_owner\",\"export_report\"],\"forbidden_actions\":[\"execute_dispatch\",\"modify_raw_data\"],\"governance\":{\"classification\":\"L2\",\"redaction_applied\":false,\"redacted_fields\":[],\"role\":\"agent_readonly\"},\"redaction_summary\":{\"total_fields\":7,\"redacted_count\":1,\"redacted_list\":[\"seller_id\"],\"applied_role\":\"agent_readonly\"},\"prompt_version\":\"v2.1\",\"config_versions\":{\"ontology\":\"v2\",\"action_registry\":\"v1\",\"recipe\":\"seller_late_delivery_recipe\"}}"
      }
    ]
  }
}
```

### 决策工作流中的使用时机

- **将决策上下文输入 LLM 之前**：`build_context` 的输出是经过治理脱敏、版本化签名的 LLM 安全上下文，是 LLM decision provider 的标准输入格式
- **审计和重放**：`context_hash` 可用于验证上下文是否被篡改，`built_at` 时间戳和 `config_versions` 提供审计线索
- **需要完整证据清单时**：`evidence` 数组包含构建决策所需的所有定量证据
- **需要治理信息时**：`governance` 块告知 LLM 当前的数据治理分类和脱敏状态

### 常见错误

1. **忽视 context_hash 验证**：每次接收 LLM 返回的决策时，应对比 `context_hash` 确保 LLM 看到的上下文的完整性和一致性
2. **不指定 recipe_id 时的匹配不确定性**：省略 `recipe_id` 时系统按 rule_id 自动匹配，如果有多个 recipe 匹配同一个 rule_id，结果不确定
3. **混淆 build_context 与 get_decision_context**：`build_context` 返回的是结构化、版本化的 LLM 上下文信封（含 evidence、config_versions）；`get_decision_context` 返回的是领域决策上下文（含 policy、enriched_objects），两者用途不同
4. **忽略脱敏摘要检查**：`redaction_summary` 指示了被脱敏的字段，如果关键字段被脱敏，LLM 的决策可能基于不完整信息

---

## 5. get_decision_context

### 描述

获取一个决策 case 的完整领域上下文。包括触发信息、治理数据、允许/禁止的操作、策略评估结果和丰富对象（通过 OAG 链路遍历发现的关联对象）。与 `build_context` 不同，此工具返回的是面向领域分析的决策上下文，而非 LLM 输入信封。

### 参数

| 参数 | 类型 | 必填 | 描述 |
|-----------|------|----------|-------------|
| case_id | string | 是 | 要获取上下文的决策 case ID |

### 示例调用

```json
{
  "jsonrpc": "2.0",
  "id": 7,
  "method": "tools/call",
  "params": {
    "name": "get_decision_context",
    "arguments": {
      "case_id": "case-abc123"
    }
  }
}
```

### 示例响应

```json
{
  "jsonrpc": "2.0",
  "id": 7,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"case_id\":\"case-abc123\",\"trigger\":{\"alert_id\":\"alert-001\",\"rule_id\":\"seller_late_delivery_rate_7d\",\"severity\":\"high\",\"metric_name\":\"seller_late_delivery_rate_7d\",\"current_value\":0.12,\"baseline_value\":0.05,\"delta_pct\":140.0},\"object_context\":{\"object_type\":\"seller\",\"object_id\":\"SELLER001\",\"properties\":{\"seller_state\":\"SP\",\"seller_city\":\"Sao Paulo\",\"gmv\":158000.50,\"order_count\":342,\"avg_review_score\":4.2,\"late_delivery_rate\":0.08}},\"governance\":{\"classification\":\"L2\",\"redaction_applied\":false,\"redacted_fields\":[],\"role\":\"agent_readonly\"},\"allowed_actions\":[\"create_followup_task\",\"notify_owner\",\"export_report\",\"escalate_to_human\"],\"forbidden_actions\":[\"execute_dispatch\",\"modify_raw_data\",\"write_dwd\",\"write_mart\"],\"source_type\":\"metric_alert\",\"source_id\":\"alert-001\"}"
      }
    ]
  }
}
```

### 决策工作流中的使用时机

- **在决策引擎（LLM）生成决策前**：为决策引擎提供完整的领域上下文，包括触发指标、目标对象属性和治理约束
- **判断可执行的操作范围**：`allowed_actions` 和 `forbidden_actions` 明确指示了在当前治理策略下可以/不可以执行哪些操作
- **查看脱敏效果**：`governance.redacted_fields` 指示哪些字段被脱敏，帮助评估决策信息是否充分
- **获取触发源信息**：`source_type` 和 `source_id` 指示该 case 由哪个告警触发
- **内部决策引擎**：当不使用 LLM decision provider，而是使用规则引擎或内部决策逻辑时使用

### 常见错误

1. **混淆 get_decision_context 与 build_context**：`get_decision_context` 返回的是领域上下文，不包含 `context_hash` 签名和 `evidence` 数组；`build_context` 提供了 LLM-safe 的上下文信封
2. **在 LLM decision provider 之前使用错误的上下文**：如果系统配置了基于 recipe 的 LLM provider，应使用 `build_context` 而非 `get_decision_context`
3. **忽略 trigger 数据**：`trigger` 中的 `delta_pct`（变化百分比）是判断异常严重程度的关键指标，不应忽略
4. **假设 allowed_actions 固定不变**：允许的操作由治理策略动态评估，不同 case 的 `allowed_actions` 可能不同
5. **混淆 forbidden_actions 与不可用的 action schema**：`forbidden_actions` 表示当前治理策略禁止的操作，而 `list_action_schemas` 列出的是系统支持的所有 action schema

---

## 6. list_action_schemas

### 描述

列出所有可用的 action schema（操作模式定义）。每个 schema 包含操作名称、描述、风险等级、payload 结构定义（JSON Schema）、允许的角色和适配器信息。它是系统支持的操作目录。

### 参数

| 参数 | 类型 | 必填 | 描述 |
|-------|------|----------|-------------|
| 无 | — | — | 该工具不接受任何参数 |

### 示例调用

```json
{
  "jsonrpc": "2.0",
  "id": 8,
  "method": "tools/call",
  "params": {
    "name": "list_action_schemas"
  }
}
```

### 示例响应

```json
{
  "jsonrpc": "2.0",
  "id": 8,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"schemas\":[{\"name\":\"create_followup_task\",\"description\":\"创建跟进调查任务\",\"risk_level\":\"medium\",\"payload_schema\":{\"properties\":{\"task_title\":{\"type\":\"string\",\"maxLength\":200},\"task_description\":{\"type\":\"string\",\"maxLength\":2000},\"owner_role\":{\"type\":\"string\",\"enum\":[\"analyst\",\"marketing_ops\",\"seller_ops\",\"category_ops\",\"logistics_ops\"]},\"priority\":{\"type\":\"string\",\"enum\":[\"low\",\"medium\",\"high\"],\"default\":\"medium\"}},\"required\":[\"task_title\",\"owner_role\"]},\"allowed_by\":[\"analyst\",\"manager\"],\"adapter\":\"github\"},{\"name\":\"notify_owner\",\"description\":\"通知负责人处理异常\",\"risk_level\":\"medium\",\"payload_schema\":{\"properties\":{\"alert_id\":{\"type\":\"string\"},\"owner_role\":{\"type\":\"string\",\"enum\":[\"analyst\",\"marketing_ops\",\"seller_ops\",\"category_ops\",\"logistics_ops\",\"admin\"]},\"message\":{\"type\":\"string\",\"maxLength\":2000}},\"required\":[\"alert_id\",\"owner_role\",\"message\"]},\"allowed_by\":[\"analyst\"],\"adapter\":\"feishu\"},{\"name\":\"export_report\",\"description\":\"导出报告\",\"risk_level\":\"medium\",\"payload_schema\":{\"properties\":{\"report_type\":{\"type\":\"string\",\"enum\":[\"alert_summary\",\"decision_rationale\",\"impact_analysis\"]},\"format\":{\"type\":\"string\",\"enum\":[\"markdown\",\"csv\",\"pdf\"],\"default\":\"markdown\"},\"include_evidence\":{\"type\":\"boolean\",\"default\":true}},\"required\":[\"report_type\"]},\"allowed_by\":[\"analyst\",\"manager\"],\"adapter\":\"feishu\"},{\"name\":\"create_outbox_message\",\"description\":\"创建出站消息\",\"risk_level\":\"medium\",\"payload_schema\":{\"properties\":{\"target_channel\":{\"type\":\"string\",\"enum\":[\"feishu\",\"webhook\"]},\"message\":{\"type\":\"string\",\"maxLength\":4000},\"priority\":{\"type\":\"string\",\"enum\":[\"low\",\"medium\",\"high\"],\"default\":\"medium\"}},\"required\":[\"target_channel\",\"message\"]},\"allowed_by\":[\"manager\"],\"adapter\":\"feishu\"}]}"
      }
    ]
  }
}
```

### 决策工作流中的使用时机

- **了解可执行的操作**：在生成决策前，调用此工具了解系统支持哪些操作类型及其 payload 要求
- **构造 action 参数**：通过 `payload_schema` 了解每个 action 需要的字段、类型、枚举值和必填项，正确构造 `execute_action` 或 `propose_action` 的参数
- **角色权限检查**：`allowed_by` 指示哪些角色可以执行该操作，用于判断当前 agent 是否有权限提议该操作
- **选择适配器**：`adapter` 指示操作将通过哪个适配器执行（如 `feishu`、`github`），用于判断执行结果的形式
- **风险等级评估**：`risk_level` 帮助判断操作是否需要人工审批

### 常见错误

1. **混淆 allowed_by 与当前 case 的 allowed_actions**：`list_action_schemas` 返回的是全局的 schema 定义，不表示当前 case 的治理策略允许了哪些操作。当前 case 允许的操作应查询 `get_decision_context` 的 `allowed_actions` 字段
2. **忽略 payload_schema 中的 required 字段**：调用 `propose_action` 或 `execute_action` 时，如果未提供 `required` 字段，验证会失败
3. **不检查 enum 值**：payload 字段的 `enum` 约束定义了合法值，使用非法值会导致验证错误
4. **重复调用**：action schemas 在系统运行期间不变化，建议缓存一次调用结果

---

## 决策工作流中的工具编排

以下是典型的 Pi Agent 决策循环中这些工具的推荐调用顺序：

```
Step 1: describe_ontology
        → 了解对象类型、属性、关系和可用操作
        （连接时调用一次，结果缓存）

Step 2: list_action_schemas
        → 了解支持的 action 类型及其参数模式
        （连接时调用一次，结果缓存）

Step 3: get_decision_context (或 build_context)
        → 获取当前 case 的领域决策上下文
        → 了解触发告警、目标对象、治理限制
        → 了解当前 case 允许/禁止的操作

Step 4: get_object (按需)
        → 补充获取特定对象的详细信息
        → 当决策上下文未包含足够数据时使用

Step 5: get_linked_objects (按需)
        → 深入探索关联对象进行根因分析
        → 获取与目标对象相关的影响面数据

Step 6: propose_action / execute_action
        → 基于以上上下文做出决策动作
        → 使用 `propose_action` 提议需要审批的操作
        → 使用 `execute_action --dry-run` 模拟验证
```

**关键原则**：
- `describe_ontology` 和 `list_action_schemas` 是元数据工具，应在连接时各调用一次并缓存
- `get_decision_context` 和 `build_context` 是主要的上下文入口，每次决策调用一次
- `get_object` 和 `get_linked_objects` 是补充探索工具，仅在上下文信息不足时使用
