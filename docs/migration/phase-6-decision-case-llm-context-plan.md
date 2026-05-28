# Phase 6: Decision Case / LLM Context / Action Proposal

## 1. 概述

Phase 6 构建受控的决策管道，将 `ops.metric_alert` 转换为 `ai.decision_case`，构建经过治理脱敏的 LLM-safe 上下文，通过基于规则的决策引擎（LLM 默认关闭）生成结构化决策，并产生 `ai.action_proposal`（全部需要人工审核）。

Phase 6 不执行真实 LLM 调用，不执行任何操作，不生成 outbox 事件。它桥接 Phase 5（治理/本体）和 Phase 7（审核/执行）。

### 基线参考

- **前置依赖**: Phase 5 设计文档 `docs/migration/phase-5-governance-ontology-runtime-plan.md`
- **Phase 5 服务**: ObjectQueryService, GovernanceService, RedactionEngine
- **现有 Schema**: `ai.decision_case`（基础表，来自 `007_ai_tables.sql`）
- **新增 Schema**: `migrations/010_ai_tables_enhance.sql`（在 007 基础上增强，不修改原始迁移）
- **数据库 Schema**: `ai.*`（decision_case, llm_decision, action_proposal）

### 核心原则

Phase 6 是 **受控决策管道**。它创建决策案例，构建 LLM-safe 上下文，生成基于规则的决策，并产生待审核的操作提案。所有提案都需要人工审核，不会自动执行。

---

## 2. decision_case 生命周期

decision_case 经历定义的状态转换流程。每个状态代表案例处理中的一个明确阶段。

### 状态机

```
created ─→ context_built ─→ decision_generated ─→ proposal_generated ─→ review_required ─→ closed
                                                                                              │
                                                                                              └→ failed
```

### 状态定义

| 状态 | 说明 | 入口条件 | 出口条件 |
|------|------|---------|---------|
| `created` | 案例已从 alert 创建，尚未构建上下文 | alert 映射完成，case 记录写入 DB | BuildContext 调用成功 |
| `context_built` | LLM-safe 上下文已构建，已计算 context_hash | 上下文构建成功，gov snapshot 已附加 | Decide 调用成功 |
| `decision_generated` | 决策已生成（基于规则或 LLM），已验证 schema | 决策输出通过 SchemaValidator 校验 | 提案生成成功 |
| `proposal_generated` | 操作提案已从决策生成 | 所有 recommended_action 已转换为 proposal | 状态标记为 review_required |
| `review_required` | 等待人工审核（Phase 6 终点状态） | 提案已生成，requires_human_review=true | Phase 7（人工审核/执行） |
| `closed` | 案例已关闭（审核完成或手动关闭） | 人工操作 | — |
| `failed` | 案例处理失败（上下文构建失败、决策失败） | 关键流程出错 | — |

### 状态转换规则

- 状态只能向前推进（created → context_built → decision_generated → proposal_generated → review_required）
- `closed` 和 `failed` 是终止状态，不可再转换
- 只有 `created` 状态允许重试上下文构建
- 失败状态记录 `error_message`

---

## 3. alert → decision_case 映射

从 `ops.metric_alert` 到 `ai.decision_case` 的字段映射定义了告警如何触发决策流程。

### 字段映射表

| decision_case 字段 | 来源 | 说明 |
|--------------------|------|------|
| `case_id` | idgen 生成 | 格式: `dc_<timestamp>_<6char_hash>` |
| `source_type` | 固定值 `"alert"` | 来源类型，Phase 6 仅支持 alert |
| `source_id` | `metric_alert.alert_id` | 原始告警 ID |
| `object_type` | `metric_alert.object_type` | AIP 对象类型（seller, region 等） |
| `object_id` | `metric_alert.object_id` | AIP 对象标识 |
| `severity` | `metric_alert.severity` | 告警严重程度（low, medium, high, critical） |
| `status` | 固定值 `"created"` | 初始状态 |
| `context_hash` | 构建后计算 | SHA256 脱敏后上下文 JSON |
| `governance_snapshot_json` | GovernanceService | 分类/标记/血缘快照 |
| `created_by` | 调用方传入 | 触发者标识 |
| `error_message` | 流程错误时设置 | 失败原因 |
| `created_at` | NOW() | 创建时间戳 |
| `updated_at` | NOW() | 更新时间戳 |

### metric_alert 关键字段

```sql
-- ops.metric_alert (参考)
alert_id        TEXT PRIMARY KEY
rule_id         TEXT NOT NULL       -- 告警规则 ID (如 "gmv_drop")
severity        TEXT NOT NULL       -- low, medium, high, critical
object_type     TEXT NOT NULL       -- 对象类型
object_id       TEXT NOT NULL       -- 对象 ID
metric_name     TEXT NOT NULL       -- 指标名称 (如 "gmv")
current_value   DOUBLE PRECISION   -- 当前值
baseline_value  DOUBLE PRECISION   -- 基线值
change_rate     DOUBLE PRECISION   -- 变化率
```

### 幂等性保证

决策案例对同一来源的告警是幂等的。部分唯一索引确保活跃案例不会重复：

```sql
CREATE UNIQUE INDEX idx_ai_decision_case_active_source
ON ai.decision_case(source_type, source_id)
WHERE status NOT IN ('closed', 'failed');
```

当 `CreateCaseFromAlert` 被重复调用时，如果存在 `source_type='alert'` 且 `source_id=alert_id` 的活跃案例，则返回现有案例（类似 HTTP 200 而非 409）。

---

## 4. decision context schema

决策上下文是送入 LLM（或规则引擎）的结构化数据包，包含触发信息、对象上下文、治理快照和操作边界。

### 完整上下文 JSON 示例

```json
{
  "decision_case_id": "dc_1717000000_a1b2c3",
  "source_type": "alert",
  "source_id": "alert_xxx",
  "trigger": {
    "alert_id": "alert_xxx",
    "rule_id": "gmv_drop",
    "severity": "high",
    "metric_name": "gmv",
    "current_value": 125000,
    "baseline_value": 160000,
    "delta_pct": -0.218
  },
  "object_context": {
    "object_type": "seller",
    "object_id": "seller_xxx",
    "properties": {
      "seller_id": "seller_xxx",
      "seller_city": "sao_paulo",
      "seller_state": "SP",
      "order_count": 45,
      "gmv_total": 125000,
      "avg_review_score": 4.2
    }
  },
  "governance": {
    "classification": "L2",
    "classification_label": "internal",
    "redaction_applied": true,
    "redacted_fields": [
      "seller_contact_name",
      "seller_email"
    ],
    "data_markings": [
      "OPERATIONAL_INTERNAL"
    ],
    "role": "agent_readonly"
  },
  "allowed_actions": [
    "create_followup_task",
    "notify_owner",
    "export_report",
    "escalate_to_human"
  ],
  "forbidden_actions": [
    "execute_dispatch",
    "modify_raw_data",
    "write_dwd",
    "write_mart"
  ]
}
```

### 字段说明

| 字段路径 | 类型 | 说明 |
|----------|------|------|
| `trigger` | object | 告警触发信息，来自 `ops.metric_alert` |
| `trigger.alert_id` | string | 原始告警 ID |
| `trigger.rule_id` | string | 告警规则标识 |
| `trigger.severity` | string | 严重程度枚举 |
| `trigger.metric_name` | string | 指标名称 |
| `trigger.current_value` | number | 当前指标值 |
| `trigger.baseline_value` | number | 基线值 |
| `trigger.delta_pct` | number | 变化百分比（负值表示下降） |
| `object_context` | object | AIP 对象上下文数据 |
| `object_context.object_type` | string | 对象类型 |
| `object_context.object_id` | string | 对象 ID |
| `object_context.properties` | object | 对象属性键值对（已治理脱敏） |
| `governance` | object | 治理快照 |
| `governance.classification` | string | L0-L4 分类级别 |
| `governance.redaction_applied` | boolean | 是否已应用脱敏 |
| `governance.redacted_fields` | string[] | 被脱敏的字段列表 |
| `allowed_actions` | string[] | LLM/规则引擎允许执行的操作 |
| `forbidden_actions` | string[] | LLM/规则引擎禁止执行的操作 |

---

## 5. LLM-safe 上下文策略

LLM-safe 上下文确保送入决策引擎的数据已经过治理层脱敏，不包含敏感字段。

### 上下文构建流程

```
BuildContext(caseID)
  │
  ├── 1. 从 repository 获取 case（object_type, object_id, source_id）
  │
  ├── 2. 从 ops.metric_alert 获取告警触发信息
  │
  ├── 3. 调用 ObjectQueryService.BuildObjectContext(objectType, objectID)
  │       └── 获取对象属性数据（只读，无直接 SQL）
  │
  ├── 4. 调用 GovernanceService.GetClassification(objectType, objectID)
  │       └── 获取 L0-L4 分类级别
  │
  ├── 5. 调用 GovernanceService.GetLineage(objectType, objectID)
  │       └── 获取对象血缘信息（可选）
  │
  ├── 6. 调用 RedactionEngine.Redact(objectContext, "agent_readonly")
  │       └── 脱敏 L3/L4 字段，记录脱敏日志
  │
  ├── 7. 组装 DecisionContext（trigger + object + governance + actions）
  │
  ├── 8. 计算最终脱敏上下文 JSON 的 SHA256 哈希
  │
  └── 9. 返回 DecisionContext + context_hash
```

### 安全保证

- **无直接 SQL**: 所有上下文数据来自 `ObjectQueryService` 和 `GovernanceService`，不构造或传递原始 SQL 查询
- **只读访问**: 上下文构建过程中没有任何写入操作
- **脱敏后哈希**: `context_hash` 基于**脱敏后**的最终 JSON 计算，确保哈希值反映的是 LLM 实际看到的內容
- **字段过滤**: `allowed_actions` 和 `forbidden_actions` 限制决策输出范围
- **L3/L4 字段排除**: 敏感度 L3（sensitive）和 L4（pii）的字段在上下文中不可见

### context_hash 生成

```go
// SHA256 计算步骤
data, _ := json.Marshal(redactedContext)     // 脱敏后的完整上下文
hash := sha256.Sum256(data)                   // SHA256 哈希
contextHash := hex.EncodeToString(hash[:])    // 64 字符十六进制字符串
```

---

## 6. Governance/Redaction 使用策略

Phase 6 组合 Phase 5 的三个治理服务来保证上下文的合规性。

### 服务组合

| Phase 5 服务 | Phase 6 职责 | 输入 | 输出 |
|-------------|-------------|------|------|
| `ObjectQueryService` | 获取对象上下文数据 | object_type, object_id | 对象属性映射（含敏感度标记） |
| `GovernanceService` | 获取治理分类和血缘信息 | object_type, object_id | 分类级别、标记列表、血缘摘要 |
| `RedactionEngine` | 脱敏 L3/L4 敏感字段 | 对象上下文, 角色 | 脱敏后上下文 + 脱敏日志 |

### 角色定义

Phase 6 为决策上下文硬编码角色为 `agent_readonly`：

| 角色 | can_access | 说明 |
|------|-----------|------|
| `agent_readonly` | L0-L2 字段 | AI 代理只读角色，不能看到 L3/L4 敏感字段 |

### 脱敏行为

| 分类级别 | 脱敏行为 |
|----------|---------|
| L0 (public_internal) | 始终可见 |
| L1 (internal) | 始终可见 |
| L2 (derived_sensitive) | 可见（聚合衍生指标） |
| L3 (sensitive) | **脱敏**。从 object_context.properties 中移除 |
| L4 (pii) | **脱敏**。从 object_context.properties 中移除 |

### Redacted Fields 记录

脱敏的字段记录在上下文 `governance.redacted_fields` 列表中，以及 `gov.redaction_log` 审计表中：

```json
{
  "governance": {
    "redaction_applied": true,
    "redacted_fields": ["seller_contact_name", "seller_email"]
  }
}
```

---

## 7. Rule-Based Fallback 策略

基于规则的决策提供器（`RuleBasedProvider`）是 Phase 6 的默认决策引擎。当 LLM 被禁用或 LLM 返回无效决策时触发回退。

### Severity 到 Action 映射

| Severity | Decision Type | Recommended Actions | Priority | Owner |
|----------|--------------|-------------------|----------|-------|
| `critical` | `escalate_to_human` | escalate_to_human, notify_owner | high | ops |
| `high` | `escalate_to_human` | escalate_to_human, notify_owner | high | ops |
| `medium` | `investigate` | notify_owner, create_followup_task | medium | analyst |
| `low` | `monitor_only` | create_followup_task | low | analyst |
| 未知/默认 | `investigate` | notify_owner | medium | analyst |

### Confidence 映射

| Severity | Confidence |
|----------|-----------|
| critical | 0.95 |
| high | 0.85 |
| medium | 0.72 |
| low | 0.60 |
| 默认 | 0.50 |

### 实现逻辑

```go
func (p *RuleBasedProvider) GenerateDecision(ctx context.Context, input LLMSafeContext) (*DecisionOutput, error) {
    severity := input.Trigger.Severity

    output := &DecisionOutput{
        RequiresHumanReview: true,       // Phase 6 硬约束
        RecommendedActions:  []RecommendedAction{},
    }

    switch severity {
    case "critical", "high":
        output.DecisionType = "escalate_to_human"
        output.Confidence = 0.95  // critical
        output.RecommendedActions = []RecommendedAction{
            {ActionType: "escalate_to_human", Priority: "high", OwnerRole: "ops"},
            {ActionType: "notify_owner", Priority: "high", OwnerRole: "ops"},
        }
    case "medium":
        output.DecisionType = "investigate"
        output.Confidence = 0.72
        output.RecommendedActions = []RecommendedAction{
            {ActionType: "notify_owner", Priority: "medium", OwnerRole: "analyst"},
            {ActionType: "create_followup_task", Priority: "medium", OwnerRole: "analyst"},
        }
    case "low":
        output.DecisionType = "monitor_only"
        output.Confidence = 0.60
        output.RecommendedActions = []RecommendedAction{
            {ActionType: "create_followup_task", Priority: "low", OwnerRole: "analyst"},
        }
    default:
        output.DecisionType = "investigate"
        output.Confidence = 0.50
        output.RecommendedActions = []RecommendedAction{
            {ActionType: "notify_owner", Priority: "medium", OwnerRole: "analyst"},
        }
    }

    return output, nil
}
```

---

## 8. LLM Feature Flag 策略

Phase 6 的 LLM 集成通过特性开关控制。默认情况下 LLM 处于禁用状态。

### Feature Flag

| 环境变量 | 默认值 | 说明 |
|----------|--------|------|
| `LLM_ENABLED` | `false` | 是否启用 LLM 决策提供器 |
| `LLM_PROVIDER` | `disabled` | LLM 提供器名称（disabled, rule） |
| `LLM_MODEL` | `""` | LLM 模型名称（Phase 7） |
| `LLM_TIMEOUT_SECONDS` | `30` | LLM 请求超时秒数 |

### DecisionProvider 接口

```go
type DecisionProvider interface {
    GenerateDecision(ctx context.Context, input LLMSafeContext) (*DecisionOutput, error)
}
```

### 提供器注册

| Provider | LLM_ENABLED | 行为 |
|----------|------------|------|
| `DisabledProvider` | `false` | 返回错误 "LLM is disabled" |
| `RuleBasedProvider` | `false`（默认） | 基于 severity 的规则映射 |
| `LLMProvider`（Phase 7） | `true` | 调用真实 LLM API |

### 策略决议

```
DecisionEngine.GenerateDecision()
  │
  ├── LLM_ENABLED=true  → 调用已注册的 LLM DecisionProvider
  │                         └── 如果 LLM 返回无效决策 → 回退到 RuleBasedProvider
  │
  └── LLM_ENABLED=false → 直接使用 RuleBasedProvider
                            └── DisabledProvider 仅用于框架未配置时的提示
```

---

## 9. Decision Output Validation

决策输出必须通过 SchemaValidator 的校验，确保输出结构完整、值在允许范围内。

### DecisionOutput Schema

```json
{
  "decision_type": "investigate",
  "severity": "high",
  "summary": "High severity alert for seller gmv. GMV dropped 21.8% from baseline.",
  "rationale": [
    "Severity is high, triggering escalation protocol",
    "Metric gmv shows significant negative delta (-21.8%)"
  ],
  "confidence": 0.85,
  "requires_human_review": true,
  "recommended_actions": [
    {
      "action_type": "notify_owner",
      "priority": "high",
      "owner_role": "ops",
      "payload": {
        "subject": "GMV alert for seller_xxx",
        "channel": "feishu"
      }
    }
  ]
}
```

### 校验规则

| # | 规则 | 错误条件 |
|---|------|---------|
| 1 | `decision_type` ∈ 枚举 | 不在 `{monitor_only, investigate, optimize, intervention, experiment}` 中 |
| 2 | `severity` ∈ 枚举 | 不在 `{low, medium, high, critical}` 中 |
| 3 | `confidence` ∈ [0.0, 1.0] | `confidence < 0.0` 或 `confidence > 1.0` |
| 4 | `requires_human_review` == true | Phase 6 不允许自动执行 |
| 5 | `recommended_actions` 非空 | 没有行动建议 |
| 6 | 每个 `action_type` ∈ 枚举 | 不在 `{create_followup_task, notify_owner, export_report, escalate_to_human}` 中 |
| 7 | `action_type` 属于 `allowed_actions` | action 不在上下文的允许列表中 |
| 8 | `payload` 是有效 JSON | 无法序列化为 map |

### 校验失败处理

当校验失败时，DecisionEngine 记录验证错误并回退到 RuleBasedProvider：

```json
{
  "status": "fallback",
  "fallback_reason": "LLM provider returned invalid confidence value: 1.5",
  "validation_errors": [
    "confidence must be in [0.0, 1.0], got 1.5"
  ]
}
```

---

## 10. Action Proposal 生成策略

决策输出的每个 `recommended_action` 转换为一个 `ai.action_proposal` 记录。

### Proposal JSON 示例

```json
{
  "proposal_id": "ap_1717000000_x9y8z7",
  "decision_case_id": "dc_1717000000_a1b2c3",
  "action_type": "notify_owner",
  "title": "notify_owner: GMV alert for seller_xxx",
  "description": "GMV dropped 21.8% from baseline. Notify the seller owner for investigation.",
  "priority": "high",
  "risk_level": "high",
  "owner_role": "ops",
  "requires_human_review": true,
  "apply_status": "proposed",
  "payload": {
    "subject": "GMV alert for seller_xxx",
    "channel": "feishu"
  }
}
```

### 生成逻辑

```
从 DecisionOutput 的 recommended_actions 循环：
  │
  ├── 生成 proposal_id（格式: ap_<timestamp>_<6char_hash>）
  ├── title = "{action_type}: {decision.summary}"
  ├── description = decision.rationale 拼接
  ├── action_type, priority, owner_role, payload → 从 action 拷贝
  ├── risk_level = low/medium/high（对应 decision.severity）
  │     critical/high → high
  │     medium       → medium
  │     low          → low
  ├── requires_human_review = true（Phase 6 硬约束）
  ├── apply_status = "proposed"（Phase 7 之前不会变化）
  │
  └── 保存到 ai.action_proposal
```

### apply_status 枚举

| 值 | 说明 | Phase |
|----|------|-------|
| `proposed` | 提案已生成，等待审核 | Phase 6 |
| `approved` | 人工审核通过（待执行） | Phase 7 |
| `rejected` | 人工审核拒绝 | Phase 7 |

---

## 11. Human Review 边界

Phase 6 专注于提案生成而非执行。人工审核的完整流程在 Phase 7 实现。

### Phase 6 边界

| 职责 | Phase 6 | Phase 7 |
|------|---------|---------|
| 生成提案 | ✅ `requires_human_review=true` | — |
| 返回提案列表 | ✅ `GET /proposals` | — |
| 审核/批准提案 | ❌ | ✅ 审核 UI |
| 拒绝提案 | ❌ | ✅ 拒绝流程 |
| 执行操作 | ❌ | ✅ 自动执行 |
| 更新 apply_status | ❌（始终 `proposed`） | ✅ approved / rejected |

### 关键约束

- `requires_human_review` 在所有决策输出和提案中始终为 `true`
- `apply_status` 始终为 `proposed`
- Phase 6 不包含批准/拒绝端点（`POST /approve`, `POST /reject`）
- Phase 6 不执行任何操作（不产生 outbox 事件）
- Phase 6 不修改任何业务数据（ops.*, dwd.*, mart.*, audit.*）

### 从 Phase 6 到 Phase 7 的桥接

```
Phase 6 产出                  Phase 7 消费
─────────────────            ─────────────────
ai.decision_case             → 审核工作台 (review UI)
ai.llm_decision              → 决策回溯分析
ai.action_proposal           → 审核/批准/拒绝面板
（requires_human_review=true）  → 人工确认后自动执行
```

---

## 12. API Endpoint Reference

Phase 6 公开 6 个 HTTP API 端点，全部位于 `/api/v1/decisions` 路径下。

### 端点摘要

| # | Method | Path | Handler | 说明 |
|---|--------|------|---------|------|
| D01 | POST | `/api/v1/decisions/cases` | `CreateCase` | 从 alert 创建决策案例 |
| D02 | GET | `/api/v1/decisions/cases` | `ListCases` | 列出决策案例（支持筛选） |
| D03 | GET | `/api/v1/decisions/cases/{case_id}` | `GetCase` | 获取单个案例详情 |
| D04 | POST | `/api/v1/decisions/cases/{case_id}/context` | `BuildContext` | 构建 LLM-safe 上下文 |
| D05 | POST | `/api/v1/decisions/cases/{case_id}/decide` | `Decide` | 生成决策 + 提案 |
| D06 | GET | `/api/v1/decisions/cases/{case_id}/proposals` | `ListProposals` | 列出案例的提案列表 |

### 端点详情

#### D01: POST /api/v1/decisions/cases

从 alert 创建决策案例。幂等，同一活跃 alert 返回已有案例。

**请求**:
```json
{
  "source_type": "alert",
  "source_id": "alert_xxx",
  "created_by": "system"
}
```

**响应** (201 Created):
```json
{
  "case_id": "dc_1717000000_a1b2c3",
  "source_type": "alert",
  "source_id": "alert_xxx",
  "status": "created",
  "object_type": "seller",
  "object_id": "seller_xxx",
  "severity": "high",
  "created_at": "2026-05-26T10:00:00Z",
  "updated_at": "2026-05-26T10:00:00Z"
}
```

#### D02: GET /api/v1/decisions/cases

列出决策案例，支持状态筛选和分页。

**查询参数**:
- `status` (optional): 按状态筛选
- `source_type` (optional): 按来源类型筛选
- `object_type` (optional): 按对象类型筛选
- `limit` (optional, default 20): 每页数量
- `offset` (optional, default 0): 偏移量

**响应** (200 OK):
```json
{
  "items": [
    {
      "case_id": "dc_1717000000_a1b2c3",
      "source_type": "alert",
      "source_id": "alert_xxx",
      "status": "created",
      "severity": "high",
      "created_at": "2026-05-26T10:00:00Z"
    }
  ],
  "total": 42,
  "limit": 20,
  "offset": 0
}
```

#### D03: GET /api/v1/decisions/cases/{case_id}

获取单个决策案例的完整详情。

**响应** (200 OK): 同 D01 响应格式，包含 `context_hash`、`governance_snapshot_json`（如果已构建）等字段。

#### D04: POST /api/v1/decisions/cases/{case_id}/context

为案例构建 LLM-safe 上下文。

**请求**: 无 body

**响应** (200 OK):
```json
{
  "case_id": "dc_1717000000_a1b2c3",
  "status": "context_built",
  "context_hash": "a1b2c3d4...",
  "context": {
    "trigger": { ... },
    "object_context": { ... },
    "governance": {
      "classification": "L2",
      "redaction_applied": true,
      "redacted_fields": ["seller_contact_name"]
    },
    "allowed_actions": ["create_followup_task", "notify_owner", "export_report", "escalate_to_human"],
    "forbidden_actions": ["execute_dispatch", "modify_raw_data", "write_dwd", "write_mart"]
  }
}
```

#### D05: POST /api/v1/decisions/cases/{case_id}/decide

生成决策并从决策产出提案。

**请求**: 无 body（决策引擎自动使用已构建的上下文和新状态）

**响应** (200 OK):
```json
{
  "case_id": "dc_1717000000_a1b2c3",
  "status": "proposal_generated",
  "decision": {
    "decision_id": "de_1717000000_f1e2d3",
    "decision_type": "escalate_to_human",
    "severity": "high",
    "summary": "High severity alert for seller gmv. GMV dropped 21.8%.",
    "confidence": 0.85,
    "requires_human_review": true,
    "status": "generated"
  },
  "proposals": [
    {
      "proposal_id": "ap_1717000000_x9y8z7",
      "action_type": "escalate_to_human",
      "title": "escalate_to_human: GMV alert for seller_xxx",
      "priority": "high",
      "risk_level": "high",
      "requires_human_review": true,
      "apply_status": "proposed"
    }
  ]
}
```

#### D06: GET /api/v1/decisions/cases/{case_id}/proposals

列出案例的所有操作提案。

**响应** (200 OK):
```json
{
  "items": [
    {
      "proposal_id": "ap_1717000000_x9y8z7",
      "action_type": "escalate_to_human",
      "title": "escalate_to_human: GMV alert for seller_xxx",
      "priority": "high",
      "risk_level": "high",
      "requires_human_review": true,
      "apply_status": "proposed",
      "created_at": "2026-05-26T10:00:00Z"
    }
  ],
  "total": 2
}
```

### 认证

所有端点需要 Bearer Token 认证：

```
Authorization: Bearer <token>
```

未认证请求返回 HTTP 401。

---

## 13. 非目标 (Non-Goals)

以下内容明确不属于 Phase 6 的范围：

### 无真实 LLM 调用

Phase 6 不进行任何真实 LLM API 调用。`DisabledProvider` 返回错误信息。即使设置 `LLM_ENABLED=true`，也不会产生 HTTP 出站请求到 LLM 服务商。真实 LLM 集成（OpenAI、Claude 等）在 Phase 7 实现。

### 无操作执行

Phase 6 不执行任何操作。不会调用 adapter 发送飞书消息、创建 GitHub Issue、或执行本地 CLI 命令。所有操作停留在 `ai.action_proposal` 表中，`apply_status` 固定为 `proposed`。

### 无 Outbox 事件

Phase 6 不产生 `audit.event_outbox` 记录。Outbox 事件生成在 Phase 7 的操作执行阶段处理。

### 无 Pipeline 逻辑变更

Phase 6 不修改 pipeline 步骤、不改变 `dwd.*` / `mart.*` / `ops.*` 层的计算逻辑。所有 pipeline 计算保持不变。

### 无 Python 代码修改

Phase 6 不改动 `api/*`、`services/*`、`scripts/*` 中的 Python 文件。Phase 6 是纯 Go 实现。

### 无 React 前端修改

Phase 6 不修改 `frontend/*` 目录中的 React 代码。决策案例的 UI 展示在 Phase 7 实现。

### 无 LLM 直接 SQL 访问

上下文构建不允许将原始 SQL 查询传递给 LLM。所有数据通过 `ObjectQueryService` / `GovernanceService` 获取。

### 无 LLM 直接数据库写入

LLM 没有数据库写入权限。所有持久化操作通过明确定义的 Repository 方法进行。

### 无自动操作应用

Phase 6 没有任何逻辑会自动执行操作提案。操作应用仅在 Phase 7 人工审核后发生。

---

## 14. 验收标准 (Acceptance Criteria)

### AC1: Migration

- [ ] `migrations/010_ai_tables_enhance.sql` 存在且可被 goose 应用
- [ ] `make migrate` 应用 010 迁移成功
- [ ] `ai.decision_case` 表包含 `source_type`、`source_id`、`object_type`、`object_id`、`severity`、`context_hash`、`governance_snapshot_json`、`created_by`、`error_message`、`updated_at` 列
- [ ] `ai.action_proposal` 表包含 `title`、`description`、`risk_level`、`requires_human_review` 列
- [ ] `ai.llm_decision` 表包含 `status`、`fallback_reason`、`validation_errors` 列
- [ ] 部分唯一索引 `idx_ai_decision_case_active_source` 存在
- [ ] CHECK 约束正确设置（status 枚举、action_type 枚举、apply_status 枚举）
- [ ] `make migrate-down` 能正常回退 010 迁移

### AC2: 决策案例创建

- [ ] `POST /api/v1/decisions/cases` 从 alert 创建案例并返回 201
- [ ] `CreateCaseFromAlert` 为同一 alert 返回已有案例（幂等）
- [ ] 案例初始状态为 `created`
- [ ] 案例包含正确的 `source_type`、`source_id`、`object_type`、`object_id`、`severity`

### AC3: LLM-Safe 上下文

- [ ] `POST /api/v1/decisions/cases/{case_id}/context` 构建上下文并返回 200
- [ ] 上下文包含 `trigger`（告警触发信息）
- [ ] 上下文包含 `object_context`（对象属性）
- [ ] 上下文包含 `governance`（分类级别、脱敏标记）
- [ ] 上下文包含 `allowed_actions` 和 `forbidden_actions`
- [ ] 上下文包含 `context_hash`（SHA256 十六进制字符串）
- [ ] L3/L4 敏感字段不在上下文中
- [ ] 脱敏字段记录在 `redacted_fields` 中

### AC4: 决策生成

- [ ] `POST /api/v1/decisions/cases/{case_id}/decide` 生成决策并返回 200
- [ ] critical/high 严重级别生成 `escalate_to_human` 决策类型
- [ ] medium 严重级别生成 `investigate` 决策类型
- [ ] low 严重级别生成 `monitor_only` 决策类型
- [ ] 所有决策的 `requires_human_review` 为 `true`
- [ ] `confidence` 值在 [0.0, 1.0] 范围内
- [ ] 无效决策触发回退到 RuleBasedProvider
- [ ] 决策保存到 `ai.llm_decision` 表

### AC5: 操作提案

- [ ] 决策生成后自动创建提案
- [ ] 所有提案的 `requires_human_review` 为 `true`
- [ ] 所有提案的 `apply_status` 为 `proposed`
- [ ] `GET /api/v1/decisions/cases/{case_id}/proposals` 返回提案列表
- [ ] 提案有正确的 `action_type`、`priority`、`risk_level`、`owner_role`

### AC6: API 端点

- [ ] 所有 6 个端点需要 Bearer Token 认证（无 token 返回 401）
- [ ] `POST /api/v1/decisions/cases` 返回 201
- [ ] `GET /api/v1/decisions/cases` 支持分页（limit/offset）
- [ ] `GET /api/v1/decisions/cases/{case_id}` 返回 200（存在）或 404（不存在）
- [ ] `POST /api/v1/decisions/cases/{case_id}/context` 返回 200
- [ ] `POST /api/v1/decisions/cases/{case_id}/decide` 返回 200
- [ ] `GET /api/v1/decisions/cases/{case_id}/proposals` 返回 200
- [ ] 端点响应使用统一 JSON 格式

### AC7: 回归和安全

- [ ] 无真实 LLM API 调用（LLM_ENABLED=false）
- [ ] 无操作执行
- [ ] 无 outbox 事件生成
- [ ] 所有现有测试通过（`go test ./...`）
- [ ] Phase 4 和 Phase 5 端点不受影响
- [ ] `go build ./cmd/baxi-api` 和 `go build ./cmd/baxi-cli` 编译成功

---

## 附录 A: 数据库 Schema 增强

### ai.decision_case 新增列

```sql
ALTER TABLE ai.decision_case ADD COLUMN source_type TEXT NOT NULL DEFAULT '';
ALTER TABLE ai.decision_case ADD COLUMN source_id TEXT NOT NULL DEFAULT '';
ALTER TABLE ai.decision_case ADD COLUMN object_type TEXT;
ALTER TABLE ai.decision_case ADD COLUMN object_id TEXT;
ALTER TABLE ai.decision_case ADD COLUMN severity TEXT;
ALTER TABLE ai.decision_case ADD COLUMN context_hash TEXT;
ALTER TABLE ai.decision_case ADD COLUMN governance_snapshot_json JSONB;
ALTER TABLE ai.decision_case ADD COLUMN created_by TEXT;
ALTER TABLE ai.decision_case ADD COLUMN error_message TEXT;
ALTER TABLE ai.decision_case ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- CHECK 约束
ALTER TABLE ai.decision_case ADD CONSTRAINT chk_decision_case_status
    CHECK (status IN ('created', 'context_built', 'decision_generated',
                      'proposal_generated', 'review_required', 'closed', 'failed'));

-- 部分唯一索引（幂等性保证）
CREATE UNIQUE INDEX idx_ai_decision_case_active_source
    ON ai.decision_case(source_type, source_id)
    WHERE status NOT IN ('closed', 'failed');
```

### ai.action_proposal 新增列

```sql
ALTER TABLE ai.action_proposal ADD COLUMN title TEXT NOT NULL DEFAULT '';
ALTER TABLE ai.action_proposal ADD COLUMN description TEXT;
ALTER TABLE ai.action_proposal ADD COLUMN risk_level TEXT;
ALTER TABLE ai.action_proposal ADD COLUMN requires_human_review BOOLEAN NOT NULL DEFAULT TRUE;

-- apply_status CHECK 约束
ALTER TABLE ai.action_proposal ADD CONSTRAINT chk_action_proposal_apply_status
    CHECK (apply_status IN ('proposed', 'approved', 'rejected'));

-- action_type CHECK 约束
ALTER TABLE ai.action_proposal ADD CONSTRAINT chk_action_proposal_action_type
    CHECK (action_type IN ('create_followup_task', 'notify_owner', 'export_report', 'escalate_to_human'));
```

### ai.llm_decision 新增列

```sql
ALTER TABLE ai.llm_decision ADD COLUMN status TEXT;
ALTER TABLE ai.llm_decision ADD COLUMN fallback_reason TEXT;
ALTER TABLE ai.llm_decision ADD COLUMN validation_errors JSONB;
```

## 附录 B: DecisionProvider 接口定义

```go
package llm

// DecisionProvider is the interface for decision generation.
// Phase 6 implements two providers: DisabledProvider and RuleBasedProvider.
// Phase 7 adds LLMProvider for real LLM API calls.
type DecisionProvider interface {
    GenerateDecision(ctx context.Context, input LLMSafeContext) (*DecisionOutput, error)
}

// LLMSafeContext is the governance-redacted context sent to the decision provider.
type LLMSafeContext struct {
    DecisionCaseID string            `json:"decision_case_id"`
    Trigger        TriggerContext    `json:"trigger"`
    ObjectContext  ObjectContext     `json:"object_context"`
    Governance     GovernanceContext `json:"governance"`
    AllowedActions []string          `json:"allowed_actions"`
}

// DecisionOutput is the structured decision from the provider.
type DecisionOutput struct {
    DecisionType        string               `json:"decision_type"`
    Severity            string               `json:"severity"`
    Summary             string               `json:"summary"`
    Rationale           []string             `json:"rationale"`
    Confidence          float64              `json:"confidence"`
    RequiresHumanReview bool                 `json:"requires_human_review"`
    RecommendedActions  []RecommendedAction  `json:"recommended_actions"`
}

// RecommendedAction is a single proposed action from the decision.
type RecommendedAction struct {
    ActionType string                 `json:"action_type"`
    Priority   string                 `json:"priority"`
    OwnerRole  string                 `json:"owner_role"`
    Payload    map[string]interface{} `json:"payload,omitempty"`
}
```

## 附录 C: 决策上下文 Schema 定义

```go
package decision

// DecisionContext is the full context for a decision case.
type DecisionContext struct {
    DecisionCaseID string             `json:"decision_case_id"`
    SourceType     string             `json:"source_type"`
    SourceID       string             `json:"source_id"`
    Trigger        TriggerContext     `json:"trigger"`
    ObjectContext  ObjectContext      `json:"object_context"`
    Governance     GovernanceContext  `json:"governance"`
    AllowedActions []string           `json:"allowed_actions"`
    ForbiddenActions []string         `json:"forbidden_actions"`
    ContextHash    string             `json:"context_hash,omitempty"`
}

type TriggerContext struct {
    AlertID       string  `json:"alert_id"`
    RuleID        string  `json:"rule_id"`
    Severity      string  `json:"severity"`
    MetricName    string  `json:"metric_name"`
    CurrentValue  float64 `json:"current_value"`
    BaselineValue float64 `json:"baseline_value"`
    DeltaPct      float64 `json:"delta_pct"`
}

type ObjectContext struct {
    ObjectType string            `json:"object_type"`
    ObjectID   string            `json:"object_id"`
    Properties map[string]interface{} `json:"properties"`
}

type GovernanceContext struct {
    Classification   string   `json:"classification"`
    RedactionApplied bool     `json:"redaction_applied"`
    RedactedFields   []string `json:"redacted_fields"`
    Role             string   `json:"role"`
}
```
