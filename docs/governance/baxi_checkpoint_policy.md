# Baxi Checkpoint Policy v0.5.3

> **Generated:** 2026-05-23
> **Purpose:** Define checkpoint/justification requirements for sensitive operations
> **Config:** config/checkpoint_rules.yml
> **Related:** baxi_access_control_model.md, baxi_data_marking_policy.md

## Overview

The checkpoint system requires users to provide justification before performing sensitive operations. It follows Palantir Foundry's Checkpoints pattern (参考: 06_checkpoints_overview.md, 07_configure_checkpoints.md).

### v0.5.3 Scope

Checkpoints are **advisory-only** in v0.5.3:
- API endpoints log checkpoint prompts in the response
- UI displays checkpoint requirements before sensitive actions
- Checkpoint records are stored for audit (not enforced)

### v0.5.4+ (Planned)

- Checkpoint middleware will BLOCK operations without valid justification
- UI will show modal dialogs requiring justification before proceeding

## Checkpoint Trigger Endpoints

The following operations require checkpoint justification:

| Rule ID | Trigger Endpoint | Operation | Justification Type | Required Roles |
|---------|-----------------|-----------|-------------------|----------------|
| CP001 | POST /api/v1/outbox/dispatch | **Apply-mode dispatch** (real external send) | dropdown + acknowledgment | all |
| CP002 | POST /api/v1/pipeline/run | **Trigger pipeline execution** | acknowledgment | all |
| CP003 | GET /api/v1/governance/catalog | **View PII-marked fields** | acknowledgment | admin |
| CP004 | GET /api/v1/feishu | **View Feishu sync data** (contains mapped business data) | acknowledgment | owner_role |
| CP005 | Export governance report | **Export full governance data** | response (free-text reason) | admin |

**Note**: GET /api/v1/outbox/dispatch with `apply=false` (dry-run) does NOT require a checkpoint. Only apply-mode triggers require justification.

---

## Justification Types

Following Palantir's checkpoint types pattern (参考: 07_configure_checkpoints.md):

### Type 1: Acknowledgment (Checkbox)

User checks a box to confirm policy awareness before proceeding.

**Example for CP001 (Outbox dispatch):**
```
⚠️ 您正在执行真实的数据分发操作。
  This will send data to external systems (Feishu).
  
  ☑ 我已确认此操作用于合法业务目的
    (I confirm this operation is for a legitimate business purpose)
```

### Type 2: Dropdown (Predefined Values)

User selects from predefined values.

**Example for CP001:**
```
分发原因 / Dispatch Reason:
  [ ] 管道测试 (Pipeline test)
  [ ] 策略验证 (Strategy validation)
  [ ] 业务运营 (Business operations)
  [ ] 数据导出 (Data export)
  [ ] 其他 (Other - please specify in notes)
```

### Type 3: Response (Free-Text)

User provides a free-text justification.

**Example for CP005 (Export governance report):**
```
导出治理报告 / Export Governance Report:
  Reason: [________________________]
  
  Note: Your justification will be recorded and
        reviewable by administrators.
```

### Type 4: Reauthentication (Future)

For highly sensitive operations requiring identity verification. Not implemented in v0.5.3.

---

## Checkpoint Record Format

All checkpoint records are stored in `system_dir/governance_checkpoints.csv`:

```csv
request_id,timestamp,user_id,rule_id,endpoint,justification_type,justification_value,status
uuid-1,2026-05-23T10:30:00Z,user1,CP001,/api/v1/outbox/dispatch,dropdown,business_ops,approved
uuid-2,2026-05-23T10:31:00Z,user2,CP002,/api/v1/pipeline/run,acknowledgment,acknowledged,approved
uuid-3,2026-05-23T10:32:00Z,admin1,CP005,/api/v1/governance/export,response,"需要审核本月策略",approved
```

### Field Definitions

| Field | Type | Description |
|-------|------|-------------|
| request_id | UUID (string) | Unique identifier for the API request |
| timestamp | ISO 8601 | When the checkpoint was satisfied |
| user_id | string | User from auth token (same token in v0.5.3) |
| rule_id | string | Checkpoint rule ID (CP001, CP002, etc.) |
| endpoint | string | API endpoint that triggered the checkpoint |
| justification_type | string | Type: acknowledgment/dropdown/response |
| justification_value | string | Value provided by user |
| status | string | approved/denied/skipped (all approved in v0.5.3) |

---

## Checkpoint Configuration

### CP001: Outbox Apply Dispatch

```yaml
rule_id: CP001
name: "Outbox Apply Dispatch"
trigger:
  endpoint: /api/v1/outbox/dispatch
  condition: "body.apply == true"
prompt:
  title: "⚠️ 真实分发现确认 / Dispatch Confirmation"
  message: "您正在将数据分发到外部系统。此操作将产生真实影响。"
  description: "Your justification will be recorded for audit purposes."
justification:
  - type: dropdown
    label: "分发原因 / Dispatch Reason"
    values:
      - pipeline_test
      - strategy_validation
      - business_operations
      - data_export
      - other
    require_text_for: [other]
  - type: acknowledgment
    label: "我已确认此操作用于合法业务目的"
    required: true
audit_fields: [request_id, timestamp, user_id, justification_type, justification_value]
```

### CP002: Pipeline Run

```yaml
rule_id: CP002
name: "Pipeline Run Confirmation"
trigger:
  endpoint: /api/v1/pipeline/run
  method: POST
prompt:
  title: "⚠️ 管道执行确认 / Pipeline Run"
  message: "您正在触发数据管道重新执行。此操作将覆盖现有数据。"
justification:
  - type: acknowledgment
    label: "我已确认管道需要重新执行"
    required: true
```

### CP003: View PII Fields

```yaml
rule_id: CP003
name: "View PII Marked Fields"
trigger:
  endpoint: /api/v1/governance/catalog
  condition: "response contains L4 (PII) markings"
prompt:
  title: "⚠️ 查看敏感字段 / Viewing Sensitive Data"
  message: "以下数据包含个人身份信息(PII)。请确认您有合法理由查看。"
justification:
  - type: acknowledgment
    label: "我确认有权查看个人身份信息"
    required: true
```

### CP004: View Feishu Sync Data

```yaml
rule_id: CP004
name: "View Feishu Sync Data"
trigger:
  endpoint: /api/v1/feishu
  method: GET
prompt:
  title: "📊 飞书同步数据"
  message: "此数据包含已发送到飞书的业务信息。"
justification:
  - type: acknowledgment
    label: "我已了解此操作会查看飞书同步数据"
    required: true
```

### CP005: Export Governance Report

```yaml
rule_id: CP005
name: "Export Full Governance Report"
trigger:
  endpoint: /api/v1/governance/export
  method: GET
prompt:
  title: "📥 导出治理报告"
  message: "请提供导出治理报告的合法业务原因。"
justification:
  - type: response
    label: "导出原因 / Export Reason"
    min_length: 10
    max_length: 500
```

---

## Review Process

Checkpoint records are reviewable via:

1. **API Endpoint**: GET /api/v1/governance/checkpoints
   - Returns paginated checkpoint records
   - Filterable by rule_id, user_id, timestamp range

2. **Governance Center UI**: Checkpoint History tab
   - Displays recent checkpoint records
   - Shows justification values
   - Searchable by user, rule, and date

3. **Audit File**: system_dir/governance_checkpoints.csv
   - Raw CSV format, append-only
   - Same retention as other audit files (2555 days / 7 years)

---

## Checkpoint Frequency

v0.5.3: Every sensitive operation triggers its checkpoint.

v0.5.4+ (Planned):
- Same-user, same-rule caching: Don't re-prompt within 24 hours
- Exemption: Operators can bypass checkpoint within 1 hour of previous justification
