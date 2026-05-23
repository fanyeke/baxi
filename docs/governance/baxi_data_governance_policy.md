# Baxi 数据治理政策 v0.5.3

> **Generated:** 2026-05-23
> **Purpose:** Define the master data governance framework for Baxi (电商分析与决策平台)
> **Scope:** All data assets including SQLite tables, CSV files, API endpoints, Feishu exports, and log files
> **Related Configs:** config/data_classification.yml, config/data_markings.yml, config/checkpoint_rules.yml, config/access_policy.yml

## 1. Purpose and Scope

This document establishes the data governance framework for Baxi — an e-commerce analytics and decision platform built on the Olist dataset. The governance layer provides:

- **Classification and visibility**: Every data asset is classified by sensitivity, owned by a responsible role, and catalogued for discovery
- **Auditability**: Every sensitive operation is logged with request context and justification
- **Protection**: Sensitive data fields are marked and their propagation through the pipeline is tracked
- **Accountability**: Data owners are responsible for the quality, appropriateness, and lifecycle of their data domains

**Version scope**: v0.5.3 is **advisory-only**. Governance rules are defined in YAML configs, exposed via API endpoints, and displayed in the Governance Center UI. No runtime enforcement occurs at this stage. v0.5.4+ will introduce active enforcement: checkpoints will block sensitive operations without justification, access control will reject unauthorized requests, and retention policies will automatically trigger deletions.

**Exclusions**: The scripts/ directory is FROZEN and will not be modified. Governance wraps around existing operations; it does not alter them.

## 2. Governance Principles (Based on FIPPs)

Baxi's governance model is built on the **Fair Information Practice Principles (FIPPs)** as operationalized by Palantir Foundry (参考: 01_data_protection_and_governance.md):

| Principle | Baxi Implementation | Config File |
|-----------|-------------------|-------------|
| **Collection Limitation** | Raw CSV data is collected only from Kaggle Olist dataset for analysis purposes | data_classification.yml |
| **Data Quality** | data_quality_rules.yml defines 5 validation rules for source tables | data_quality_rules.yml, health_checks.yml |
| **Purpose Specification** | Each AIP object has a defined business purpose and owner_role | aip_object_schema.yml, access_policy.yml |
| **Use Limitation** | Access control model restricts data access to authorized roles | access_policy.yml |
| **Security Safeguards** | Data markings propagate through lineage; checkpoint records log sensitive operations | data_markings.yml, checkpoint_rules.yml |
| **Openness** | Governance API exposes all catalog, lineage, and health data via GET endpoints | governance API (new) |
| **Individual Participation** | Customer data (PII) is subject to deletion upon retention policy expiry | retention_policies.yml |
| **Accountability** | Data owners (business_ops, seller_ops, etc.) are mapped to business domains | owner_mapping.yml |

## 3. Roles and Responsibilities

| Role | Description | Associated Users |
|------|-------------|-----------------|
| **data_owner** | Accountable for data quality, appropriateness, and lifecycle within their domain | Mapped via owner_mapping.yml (business_ops, seller_ops, category_ops, logistics_ops, marketing_ops) |
| **governance_lead** | Configures and maintains governance rules, reviews checkpoint records, defines retention policies | Platform administrator (solo dev in v0.5.3) |
| **data_user** | Consumes data through API endpoints, console pages, and Feishu exports | All authenticated users (same bearer token in v0.5.3) |

### Role Responsibilities

**Data Owner** (per domain):
- Define data access requirements for their domain
- Approve or reject data export justification requests (future)
- Define retention requirements for their data
- Validate data quality rules for their domain

**Governance Lead**:
- Create and update governance YAML configs
- Review checkpoint audit records
- Monitor health check results
- Respond to governance violations

**Data User**:
- Follow data access policies
- Provide justification for sensitive operations (when triggered)
- Report data quality issues
- Respect retention and deletion policies

## 4. Data Lifecycle Management

Baxi's data follows this lifecycle:

```
Ingestion → Processing → Decision → Dispatch → Retention → Deletion
```

| Stage | Description | Primary Tables | Governance Concern |
|-------|-------------|----------------|-------------------|
| **Ingestion** | Raw CSV from Kaggle → DWD tables | dwd_order_level, dwd_item_level | Data quality checks, sensitivity marking of PII fields |
| **Processing** | DWD → aggregated metrics | metric_daily, metric_dimension_daily | Column-level lineage tracking, sensitivity propagation |
| **Decision** | Metrics → alerts → strategies → tasks | alert_events, strategy_recommendations, action_tasks | Decision evaluation, owner role validation |
| **Dispatch** | Event outbox → Feishu export | event_outbox | Checkpoint justification, rate limiting, field mapping |
| **Retention** | Data stored per policy | All tables | Retention period enforcement, grace period monitoring |
| **Deletion** | Soft-delete → hard-delete after grace period | All tables | Lineage-aware cascading, audit logging of deletions |

## 5. Enforcement Strategy

### v0.5.3 (Current) — Advisory Only

- Governance rules are defined in YAML configs and loaded on demand
- API endpoints return governance metadata (catalog, lineage, health status)
- UI displays governance information in read-only mode
- NO runtime enforcement: sensitive operations proceed without checkpoint validation
- CSV audit files continue to record operations independently

### v0.5.4+ (Planned) — Active Enforcement

- Checkpoint middleware: block sensitive API calls without valid justification
- Access control middleware: reject requests based on role + sensitivity mismatch
- Retention enforcement: cron job triggers soft-deletion of expired data
- Health check alerts: Feishu notification when health checks fail

## 6. Related Documentation

| Document | Description | Config File |
|----------|-------------|-------------|
| baxi_ontology_model.md | Governance-annotated AIP object definitions | aip_object_schema.yml |
| baxi_data_lineage_model.md | Pipeline stage lineage and sensitivity propagation | data_lineage.yml |
| baxi_data_marking_policy.md | Sensitivity classification for all data fields | data_markings.yml |
| baxi_checkpoint_policy.md | Justification requirements for sensitive operations | checkpoint_rules.yml |
| baxi_retention_policy.md | Data retention periods and deletion workflow | retention_policies.yml |
| baxi_data_health_checks.md | Data health monitoring rules and thresholds | health_checks.yml |
| baxi_decision_eval_policy.md | Strategy and task quality evaluation rules | decision_eval_rules.yml |
| baxi_access_control_model.md | Role-based access control definitions | access_policy.yml |
| baxi_aip_alignment.md | Palantir pattern adaptation documentation | — |

## 7. References

- Palantir Foundry Data Protection and Governance: docs/external/palantir_aip_foundry/01_data_protection_and_governance.md
- Palantir Foundry Security Overview: docs/external/palantir_aip_foundry/05_security_overview.md
- Palantir Foundry Checkpoints: docs/external/palantir_aip_foundry/06_checkpoints_overview.md
- OECD Guidelines on Privacy: https://www.oecd.org/sti/ieconomy/oecdguidelinesontheprotectionofprivacy.htm
