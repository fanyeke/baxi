# Baxi Decision Evaluation Policy v0.5.3

> **Generated:** 2026-05-23
> **Purpose:** Define quality evaluation rules for strategy recommendations, action tasks, and decision outputs
> **Config:** config/decision_eval_rules.yml
> **Related:** baxi_ontology_model.md, baxi_access_control_model.md, config/action_registry.yml

## Overview

This document defines evaluation criteria for Baxi's decision outputs, following the Palantir action type submission criteria pattern (参考: 15_action_types_decision_quality.md). Each decision artifact must pass validation before it can proceed to the next pipeline stage.

### Evaluation Philosophy

- **Prevent garbage downstream**: Validate before propagation, not after
- **Fail fast, fail loud**: Critical validation failures produce immediately visible errors
- **Non-breaking for v0.5.3**: v0.5.3 evaluation is advisory — it flags issues but doesn't block the pipeline
- **v0.5.4+ enforcement**: Failed validation will block propagation to the next stage

---

## Evaluation Rules

### 1. Strategy Recommendation Validation

| Rule ID | Rule Name | Target | Condition | Severity | Failure Action |
|---------|-----------|--------|-----------|----------|----------------|
| SEV001 | Strategy Type Required | strategy_recommendations | strategy_type IS NOT NULL AND NOT EMPTY | critical | Flag as invalid |
| SEV002 | Confidence Score Range | strategy_recommendations | 0.0 ≤ confidence_score ≤ 1.0 | critical | Flag as invalid |
| SEV003 | Alert Reference Valid | strategy_recommendations | alert_id exists in alert_events | warning | Flag as orphaned |
| SEV004 | Strategy Text Not Empty | strategy_recommendations | LENGTH(strategy_text) > 10 | warning | Flag as insufficient detail |
| SEV005 | Confidence Above Threshold | strategy_recommendations | confidence_score ≥ 0.3 | info | Flag as low confidence |
| SEV006 | No Duplicate Strategies | strategy_recommendations | Unique (alert_id, strategy_type) per batch | warning | Flag as duplicate |

### 2. Action Task Validation

| Rule ID | Rule Name | Target | Condition | Severity | Failure Action |
|---------|-----------|--------|-----------|----------|----------------|
| TEV001 | Risk Level Valid | action_tasks | risk_level ∈ {low, medium, high} | critical | Flag as invalid |
| TEV002 | Status Valid | action_tasks | status ∈ status_enums.yml valid values | critical | Flag as invalid |
| TEV003 | Strategy Reference Valid | action_tasks | strategy_id exists in strategy_recommendations | warning | Flag as orphaned |
| TEV004 | Action Type Registered | action_tasks | action_type exists in config/action_registry.yml | warning | Flag as unregistered |
| TEV005 | Risk Matches Action Registry | action_tasks | Risk level consistent with action_registry.yml risk_level for action_type | warning | Flag as mismatch |
| TEV006 | Owner Role Valid | action_tasks | assigned_role exists in config/owner_mapping.yml | critical | Flag as unmapped |
| TEV007 | Target Object Valid | action_tasks | target_object_type is one of 9 AIP objects | warning | Flag as unknown target |
| TEV008 | High-Risk Approval | action_tasks | If risk_level = high and requires_approval=true, check for approval record | critical | Flag as unapproved |

### 3. Alert Event Validation

| Rule ID | Rule Name | Target | Condition | Severity | Failure Action |
|---------|-----------|--------|-----------|----------|----------------|
| AEV001 | Metric Not Empty | alert_events | metric IS NOT NULL AND NOT EMPTY | critical | Flag as invalid |
| AEV002 | Severity Valid | alert_events | severity ∈ {critical, warning, info} | critical | Flag as invalid |
| AEV003 | Status Valid | alert_events | status ∈ status_enums.yml alert_event values | critical | Flag as invalid |
| AEV004 | Rule Reference Valid | alert_events | rule_id exists in alert_rules.yml or dimensional_alert_rules.yml | warning | Flag as orphaned |
| AEV005 | Owner Role Valid | alert_events | owner_role exists in owner_mapping.yml | warning | Flag as unmapped |
| AEV006 | Current Value Present | alert_events | current_value IS NOT NULL | warning | Flag as missing data |
| AEV007 | Deviation Reasonable | alert_events | ABS(deviation_pct) < 10000% | info | Flag as anomalous |
| AEV008 | No Duplicate Alerts | alert_events | No exact duplicates within 1h window | warning | Flag as duplicate alert |

### 4. Feishu Export Validation

| Rule ID | Rule Name | Target | Condition | Severity | Failure Action |
|---------|-----------|--------|-----------|----------|----------------|
| FEV001 | Field Mapping Complete | event_outbox → Feishu | All required fields mapped per feishu_field_mapping.yml | critical | Block dispatch |
| FEV002 | Rate Limit Compliance | Feishu API calls | dispatches per minute ≤ rate limit | critical | Throttle and retry |
| FEV003 | PII Not Exported | Feishu payload | No L4 (PII) fields in payload | critical | Strip PII and log |
| FEV004 | Payload Size Limit | Feishu payload | payload_json size < 10KB | warning | Truncate summary |
| FEV005 | Target Channel Valid | event_outbox | target_channel exists in channel_routing_rules.yml | critical | Route to fallback |

---

## Evaluation Scoring

Each decision artifact receives an evaluation score:

| Score | Criteria | Action |
|-------|----------|--------|
| **PASS (100%)** | All critical + warning rules pass | Proceed to next stage |
| **PASS_WITH_WARNINGS (≥80%)** | All critical pass, some warnings flagged | Proceed with warning annotation |
| **FAIL (<80%)** | At least one critical rule fails | Block from next stage (v0.5.4+) |

### Scoring Formula

```
score = (passed_critical / total_critical × 60) + (passed_warning / total_warning × 30) + (passed_info / total_info × 10)
```

Weighting: Critical = 60%, Warning = 30%, Info = 10%

---

## Failed Evaluation Handling

### v0.5.3 (Current — Advisory)
- Failed items are flagged in the Governance Center UI
- evaluation_result column is written to the respective table
- Pipeline continues (no blocking)
- Warning/Info failures are logged but not surfaced prominently

### v0.5.4+ (Planned — Enforcement)
- **Critical failure**: Block propagation, move to quarantine table (e.g., strategy_recommendations_quarantine)
- **Warning failure**: Propagate with warning annotation, require acknowledgment for high-risk items
- **Info failure**: Propagate normally, log for analytics

### Quarantine Pattern (v0.5.4+)
```sql
-- When a strategy fails critical validation:
INSERT INTO strategy_recommendations_quarantine 
  SELECT *, evaluation_reason FROM strategy_recommendations 
  WHERE evaluation_score < 80;
  
DELETE FROM strategy_recommendations 
  WHERE evaluation_score < 80;
```

---

## Configuration

```yaml
decision_eval_rules:
  strategy_validation:
    - rule_id: SEV001
      name: "策略类型必填"
      target_table: strategy_recommendations
      field: strategy_type
      operator: not_null
      severity: critical
      failure_message: "策略类型不能为空"
      
    - rule_id: SEV002
      name: "置信度范围验证"
      target_table: strategy_recommendations
      field: confidence_score
      operator: range
      min: 0.0
      max: 1.0
      severity: critical
      failure_message: "置信度必须在0.0到1.0之间"
      
  task_validation:
    - rule_id: TEV001
      name: "风险等级验证"
      target_table: action_tasks
      field: risk_level
      operator: in
      allowed_values: [low, medium, high]
      severity: critical
      
  alert_validation:
    - rule_id: AEV002
      name: "异常等级验证"
      target_table: alert_events
      field: severity
      operator: in
      allowed_values: [critical, warning, info]
      severity: critical
      
  feishu_export_validation:
    - rule_id: FEV003
      name: "PII字段过滤"
      target: feishu_payload
      operator: no_pii_fields
      severity: critical
      
  scoring:
    critical_weight: 60
    warning_weight: 30
    info_weight: 10
    pass_threshold: 100
    pass_with_warnings_threshold: 80
```
