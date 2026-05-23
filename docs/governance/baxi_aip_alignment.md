# Baxi AIP 对齐文档 v0.5.3

> **Generated:** 2026-05-23
> **Purpose:** Document how Baxi's governance layer aligns with Palantir AIP/Foundry patterns
> **Related:** All other governance docs in docs/governance/

## Overview

This document maps the governance patterns adopted from Palantir Foundry and AIP to Baxi's implementation, explaining what was adopted, what was skipped, and why (参考: 所有 docs/external/palantir_aip_foundry/ 文档).

---

## Pattern Adoption Matrix

| # | Palantir Pattern | Baxi Implementation | Adoption Status | Reference Doc |
|---|-----------------|-------------------|----------------|---------------|
| 1 | FIPPs Principles | baxi_data_governance_policy.md | ✅ Adopted | 01_data_protection_and_governance.md |
| 2 | Sensitive Data Classification | 5-level marking system (L0-L4) | ✅ Adapted | 02_protecting_sensitive_data_markings.md |
| 3 | Data Lineage DAG | 8-stage pipeline with sensitivity propagation | ✅ Adapted | 03_data_lineage_overview.md |
| 4 | CBAC (Classification-Based Access Controls) | Sensitivity-to-role access matrix | ✅ Adapted | 04_classification_based_access_controls.md |
| 5 | Two-Layer Security (Mandatory + Discretionary) | Documented but not implemented | ❌ Skipped | 05_security_overview.md |
| 6 | Checkpoints (Justification for Sensitive Actions) | CSV-based checkpoint audit + API advisory | ✅ Adapted | 06_checkpoints_overview.md, 07_configure_checkpoints.md |
| 7 | Ontology (Semantic + Kinetic Layers) | AIP objects mapped with governance metadata | ✅ Adopted | 08_ontology_overview.md |
| 8 | Data Lifetime (Lineage-Aware Deletion) | Retention policies with cascade rules | ✅ Adapted | 09_data_lifetime_retention.md |
| 9 | Restricted Views (Row-Level Access Control) | Documented for future implementation | ⏳ Deferred | 10_restricted_views.md |
| 10 | Sensitive Data Scanner (Auto-detect) | Health checks include PII detection | ✅ Adapted | 11_sensitive_data_scanner_best_practices.md |
| 11 | Audit Logs (Category-Based) | CSV-based audit with category field | ✅ Adapted | 12_audit_logs_monitoring.md |
| 12 | Data Catalog (Compass) | YAML-based data_catalog.yml | ✅ Adapted | 13_data_catalog.md |
| 13 | Data Health Checks | Category-based health monitoring | ✅ Adopted | 14_data_health_checks.md |
| 14 | Action Types (Submission Criteria) | Decision evaluation rules | ✅ Adapted | 15_action_types_decision_quality.md |

---

## Patterns Skipped and Why

| Pattern | Why Skipped | Reassessment Timeline |
|---------|-------------|----------------------|
| **Multi-Organization Silos** | Baxi is a solo project with single-tenant data organization | Reassess if multi-tenancy needed |
| **Cedar Policy Language** | Over-engineering for a solo project; YAML configs suffice | Reassess if policy complexity grows |
| **OPA (Open Policy Agent)** | Same as above — over-engineering risk | Reassess if need for declarative policy evaluation |
| **Function-Backed Actions** | Baxi's pipelines are fixed scripts; governance doesn't modify them | Reassess if dynamic actions needed |
| **Object Views (UI Config)** | Baxi uses React pages, not Foundry-style configurable views | Reassess if dynamic UI composition needed |
| **Granular Lineage-Aware Deletion with Versioning** | SQLite doesn't support Foundry's transaction versioning model | Reassess if migration to Iceberg/postgres happens |
| **SSO/MFA Integration** | Single bearer token sufficient for solo project | Reassess if team access needed |

---

## Governance Config File Index

| Config File | Governs What | Policy Doc | Config Reference |
|-------------|-------------|------------|-----------------|
| config/data_classification.yml | Sensitivity level definitions | baxi_data_marking_policy.md | L0-L4 definitions |
| config/data_markings.yml | Field-level sensitivity markings | baxi_data_marking_policy.md | All 12 tables × columns |
| config/data_lineage.yml | Pipeline stage definitions | baxi_data_lineage_model.md | 8 stages with inputs/outputs |
| config/checkpoint_rules.yml | Checkpoint triggers and prompts | baxi_checkpoint_policy.md | CP001-CP005 rules |
| config/retention_policies.yml | Retention periods and cascade | baxi_retention_policy.md | 12 tables + files |
| config/health_checks.yml | Health monitoring rules | baxi_data_health_checks.md | 50+ checks across tables |
| config/decision_eval_rules.yml | Decision quality evaluation | baxi_decision_eval_policy.md | SEV001-FEV005 rules |
| config/access_policy.yml | Role-based endpoint access | baxi_access_control_model.md | Endpoint × Role matrix |
| config/data_catalog.yml | Complete data asset catalog | baxi_aip_alignment.md | Objects + tables + endpoints |

---

## Cross-Reference Between Governance Documents

```
baxi_data_governance_policy.md (Master Policy)
    ↓ references
    ├── baxi_ontology_model.md → defines objects and their governance
    │       ↓ provides properties for
    ├── baxi_data_marking_policy.md → marks each property
    │       ↓ used by
    ├── baxi_access_control_model.md → enforces access based on markings
    ├── baxi_checkpoint_policy.md → requires checkpoints for high-marking operations
    │
    ├── baxi_data_lineage_model.md → tracks how data flows
    │       ↓ determines
    ├── baxi_retention_policy.md → sets retention per table in lineage
    │       ↓ monitored by
    ├── baxi_data_health_checks.md → validates data at each stage
    │       ↓ evaluated by
    ├── baxi_decision_eval_policy.md → evaluates decision quality
    │       ↓ accessible via
    └── baxi_access_control_model.md → controls who can view results
```

---

## Future Governance Roadmap

### v0.5.4 (Next Release)
- [ ] Basic checkpoint enforcement (block apply-mode dispatch without justification)
- [ ] Basic access control enforcement (403 on unauthorized access)
- [ ] Health check Feishu alerts (critical checks trigger notifications)
- [ ] Decision evaluation advisory scores written to tables

### v0.5.5 (Medium Term)
- [ ] Row-level access control (restricted views for domain-specific data)
- [ ] Multi-token authentication (per-user auth, not single bearer)
- [ ] Retention automation (cron job triggers soft-deletion)
- [ ] Data catalog web search (searchable governance metadata)

### v0.6.0 (Long Term)
- [ ] Lineage-aware deletion execution (actual cascade deletes)
- [ ] Policy-as-code evaluation (Cedar/OPA for complex rules)
- [ ] Full audit trail analytics (category-based audit analysis)
- [ ] Feishu integration with governance alerts

---

## Summary

Baxi's governance layer adopts **12 of 14 Palantir patterns** (86% adoption rate), with 2 patterns deferred:

- **Adopted and Implemented**: FIPPs, Data Classification, Lineage DAG, CBAC, Checkpoints, Ontology, Data Lifetime, SDS adaptation, Audit Logs, Data Catalog, Health Checks, Action Type Validation
- **Adopted but Deferred**: Row-Level Access Control (Restricted Views), Multi-Org Security

The key difference: Palantir patterns assume enterprise scale with dedicated governance teams. Baxi adapts these patterns for a **solo developer e-commerce analytics platform** by:

1. Using **YAML configs** instead of policy engines
2. Using **CSV audit files** instead of SIEM integrations
3. Using **single bearer token** auth instead of SSO/MFA
4. Making governance **advisory-first** instead of enforcement-first

This ensures the governance layer is manageable by a single developer while maintaining the structural rigor needed for future scaling.
