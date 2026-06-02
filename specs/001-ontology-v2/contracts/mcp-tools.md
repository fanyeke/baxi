# MCP Tool Contracts: Baxi Ontology v2

**Date**: 2026-06-02
**Status**: Complete

## 工具列表

### 1. describe_ontology

**描述**: 返回 Ontology 结构，包含 v2 扩展字段。

**输入**:
```json
{
  "include_v2": true
}
```

**输出**:
```json
{
  "object_types": [
    {
      "name": "seller",
      "display_name": "卖家",
      "description": "电商平台上的销售主体",
      "grain": "seller_id",
      "source": {
        "schema": "dwd",
        "table": "item_level",
        "primary_key": "seller_id"
      },
      "properties": [
        {
          "name": "seller_id",
          "type": "string",
          "sensitivity": "L2",
          "llm_readable": false,
          "searchable": true,
          "filterable": false,
          "is_pk": true
        },
        {
          "name": "gmv",
          "type": "float",
          "agg": "sum",
          "sensitivity": "L0",
          "llm_readable": true
        }
      ],
      "metrics": [
        {
          "name": "seller_late_delivery_rate_7d",
          "display_name": "最近 7 天延迟发货率"
        }
      ],
      "links": [
        {
          "name": "recent_orders",
          "display_name": "最近订单",
          "target_type": "order",
          "cardinality": "one_to_many"
        }
      ],
      "allowed_actions": [
        "notify_owner",
        "create_followup_task",
        "export_report"
      ],
      "governance": {
        "default_role": "agent_readonly",
        "redact_pii": true
      }
    }
  ]
}
```

---

### 2. get_object

**描述**: 获取单个对象详情。

**输入**:
```json
{
  "object_type": "seller",
  "object_id": "seller_123"
}
```

**输出**:
```json
{
  "object_type": "seller",
  "object_id": "seller_123",
  "properties": {
    "seller_id": "seller_123",
    "seller_state": "SP",
    "seller_city": "Sao Paulo",
    "gmv": 125000.50,
    "order_count": 150,
    "avg_review_score": 4.2,
    "late_delivery_rate": 0.15
  },
  "metrics": {
    "seller_late_delivery_rate_7d": {
      "current_value": 0.15,
      "baseline_value": 0.08,
      "delta": 0.07,
      "severity": "medium"
    },
    "seller_order_count_7d": {
      "current_value": 45
    },
    "seller_gmv_7d": {
      "current_value": 35000.00
    }
  },
  "available_links": [
    {
      "name": "recent_orders",
      "display_name": "最近订单",
      "target_type": "order",
      "cardinality": "one_to_many"
    },
    {
      "name": "products",
      "display_name": "关联产品",
      "target_type": "product",
      "cardinality": "one_to_many"
    }
  ],
  "available_actions": [
    "notify_owner",
    "create_followup_task",
    "export_report"
  ]
}
```

---

### 3. get_linked_objects

**描述**: 获取关联对象列表。

**输入**:
```json
{
  "object_type": "seller",
  "object_id": "seller_123",
  "link_name": "recent_orders",
  "options": {
    "limit": 10,
    "offset": 0
  }
}
```

**输出**:
```json
{
  "object_type": "seller",
  "object_id": "seller_123",
  "link_name": "recent_orders",
  "target_type": "order",
  "cardinality": "one_to_many",
  "total_count": 20,
  "objects": [
    {
      "order_id": "order_001",
      "order_status": "delivered",
      "payment_value": 150.00,
      "review_score": 4,
      "delivery_status": "on_time"
    },
    {
      "order_id": "order_002",
      "order_status": "shipped",
      "payment_value": 89.90,
      "review_score": null,
      "delivery_status": "late"
    }
  ]
}
```

---

### 4. build_context (新增)

**描述**: 按 case_id 构建带 recipe 的 LLM 上下文。

**输入**:
```json
{
  "case_id": "case_456",
  "recipe_id": "seller_late_delivery_alert"
}
```

**输出**:
```json
{
  "case_id": "case_456",
  "recipe_id": "seller_late_delivery_alert",
  "context_hash": "abc123def456",
  "object_context": {
    "object_type": "seller",
    "object_id": "seller_123",
    "properties": {
      "seller_state": "SP",
      "seller_city": "Sao Paulo",
      "gmv": 125000.50,
      "order_count": 150,
      "avg_review_score": 4.2,
      "late_delivery_rate": 0.15
    }
  },
  "metrics": [
    {
      "name": "seller_late_delivery_rate_7d",
      "display_name": "最近 7 天延迟发货率",
      "current_value": 0.15,
      "baseline_value": 0.08,
      "delta": 0.07,
      "severity": "medium",
      "llm_explanation": "衡量卖家近期履约稳定性。显著高于基线时，可能存在库存、发货或物流问题。"
    }
  ],
  "evidence": [
    {
      "type": "linked_objects",
      "link_name": "recent_orders",
      "target_type": "order",
      "count": 10,
      "objects": [ ... ]
    }
  ],
  "allowed_actions": [
    {
      "action_type": "notify_owner",
      "display_name": "通知负责人",
      "risk_level": "low",
      "requires_approval": false
    }
  ],
  "governance": {
    "role": "agent_readonly",
    "redacted_fields": ["seller_id"],
    "redacted_count": 1,
    "total_fields": 7
  }
}
```

---

### 5. propose_action (新增)

**描述**: 创建动作提案，不直接执行。

**输入**:
```json
{
  "object_type": "seller",
  "object_id": "seller_123",
  "action_type": "notify_owner",
  "payload": {
    "owner_role": "seller_manager",
    "message": "卖家延迟发货率异常升高",
    "channel": "feishu"
  },
  "dry_run": true
}
```

**输出**:
```json
{
  "proposal_id": "proposal_789",
  "object_type": "seller",
  "object_id": "seller_123",
  "action_type": "notify_owner",
  "status": "dry_run",
  "risk_level": "low",
  "requires_approval": false,
  "validation": {
    "valid": true,
    "checks": [
      {"check": "object_exists", "passed": true},
      {"check": "action_bound", "passed": true},
      {"check": "action_enabled", "passed": true},
      {"check": "role_authorized", "passed": true},
      {"check": "payload_valid", "passed": true}
    ]
  },
  "dry_run_result": {
    "would_send_notification": true,
    "target_channel": "feishu",
    "target_role": "seller_manager"
  },
  "created_at": "2026-06-02T10:30:00Z"
}
```

---

### 6. execute_action (保留，增强)

**描述**: 执行动作（默认 dry-run）。

**输入**:
```json
{
  "object_type": "seller",
  "object_id": "seller_123",
  "action_type": "notify_owner",
  "payload": {
    "owner_role": "seller_manager",
    "message": "卖家延迟发货率异常升高",
    "channel": "feishu"
  },
  "dry_run": true
}
```

**输出**:
```json
{
  "execution_id": "exec_101",
  "object_type": "seller",
  "object_id": "seller_123",
  "action_type": "notify_owner",
  "status": "dry_run_success",
  "dry_run": true,
  "result": {
    "would_send_notification": true,
    "target_channel": "feishu",
    "target_role": "seller_manager"
  },
  "executed_at": "2026-06-02T10:30:00Z"
}
```

---

## 错误响应格式

所有工具统一错误格式:

```json
{
  "error": {
    "code": "OBJECT_NOT_FOUND",
    "message": "Object type 'unknown_type' not found in ontology",
    "details": {
      "object_type": "unknown_type"
    }
  }
}
```

**错误码**:
- `OBJECT_NOT_FOUND`: 对象类型不存在
- `OBJECT_ID_NOT_FOUND`: 对象 ID 不存在
- `LINK_NOT_FOUND`: 关系不存在
- `RECIPE_NOT_FOUND`: 配方不存在
- `ACTION_NOT_BOUND`: 动作未绑定到对象
- `ACTION_DISABLED`: 动作在 registry 中禁用
- `ACTION_REQUIRES_APPROVAL`: 动作需要审批
- `PAYLOAD_INVALID`: payload 校验失败
- `FILTER_NOT_ALLOWED`: 字段不允许过滤
- `SORT_NOT_ALLOWED`: 字段不允许排序
- `LIMIT_EXCEEDED`: 超过最大限制
