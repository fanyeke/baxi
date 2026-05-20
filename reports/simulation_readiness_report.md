# Phase 6: Simulation Readiness Report
## Decision Scenario Prioritization for Waker Sandbox

**Report Date**: 2026-05-20
**Analysis Basis**: Phase 1-5 Comprehensive Results
**Scope**: Variable mapping, scenario design, and sandbox readiness assessment

---

## Executive Summary

Based on 5 phases of business analysis, **14 simulation variables** have been identified and **10 what-if scenarios** have been designed. This report assesses which scenarios are ready for immediate implementation in the Waker Decision Sandbox and outlines the data requirements for the remaining scenarios.

**Key Finding**: 4 scenarios (S01, S02, S07, S08) are **immediately ready** with direct data support. 4 scenarios (S03, S04, S05, S10) are **partially ready** and need cost assumptions. 2 scenarios (S06, S09) require significant additional data collection.

---

## Phase 1-5 Analysis Summary

| Phase | Focus | Key Variables Generated |
|-------|-------|------------------------|
| Phase 1 | Data Profiling | Order/item base tables |
| Phase 2 | Data Validation | Join validation, ERD |
| Phase 3 | Overall Business | category_gmv_concentration, seller_concentration, avg_order_value, repeat_purchase_rate |
| Phase 4 | Fulfillment & Experience | delivery_total_days, late_delivery_rate, approval_to_shipping_days, review_score, state_late_rate |
| Phase 5 | Marketing Funnel | marketing_channel_conversion, seller_activation_rate, channel_quality |

---

## Scenario Readiness Assessment

### Immediate Sandbox Candidates (HIGH Priority)

These 4 scenarios have **direct quantitative support** from existing analysis data:

| Scenario | Name | Data Available | Impact Quantifiable | Business Urgency | Sandbox Ready |
|----------|------|----------------|---------------------|------------------|---------------|
| **S01** | Late Delivery Optimization | ✅ Phase 4 correlation matrix | ✅ Score impact calculable | ✅ High (8%→4% late) | **YES** |
| **S02** | Delivery Time Reduction | ✅ Phase 4 correlation (-0.334) | ✅ Score uplift estimable | ✅ High (12.56→10 days) | **YES** |
| **S07** | Regional Logistics Improvement | ✅ Phase 4 state performance | ✅ Score impact by state | ✅ High (5 states >15% late) | **YES** |
| **S08** | Seller Activation Improvement | ✅ Phase 5 activation by channel | ✅ GMV uplift = Δsellers × avg GMV | ✅ Critical (45%→55%) | **YES** |

### Rationale for HIGH Priority Selection

**S01 & S02 (Fulfillment)**: Phase 4 provides direct causal evidence:
- Late delivery correlation with score: -0.267
- Total delivery correlation with score: -0.334
- Low-score customers have 33.75% late rate vs 3.50% for high-score
- Data supports direct quantification: `Δscore = correlation × Δvariable`

**S07 (Regional)**: Phase 4 identifies 5 problematic states (AL, MA, PI, CE, SE) with >15% late rate. Reducing their late rate to 8% (national average) would:
- Improve state-specific scores
- Unlock market expansion potential
- Direct state-level data available for quantification

**S08 (Activation)**: Phase 5 shows 54.9% of closed sellers never place orders. Increasing activation:
- Direct GMV impact: Additional sellers × average GMV per active seller
- Channel-specific activation rates enable targeted scenarios
- No additional cost data needed for initial impact estimation

---

### Partially Ready Scenarios (MEDIUM Priority)

These 4 scenarios need **cost/budget assumptions** for full quantification:

| Scenario | Name | Data Gap | Missing Data Type | Sandbox With Assumptions? |
|----------|------|----------|-------------------|--------------------------|
| **S03** | Paid Search Investment | Cost per click, campaign budget | Marketing spend | ⚠️ NEEDS ASSUMPTIONS |
| **S04** | Organic Search Conversion | SEO investment costs | Marketing spend | ⚠️ NEEDS ASSUMPTIONS |
| **S05** | Category Growth | Category margins, growth elasticity | Profitability data | ⚠️ NEEDS ASSUMPTIONS |
| **S10** | Approval-Speed Reduction | Seller processing capacity | Operational constraints | ⚠️ NEEDS ASSUMPTIONS |

### Assumption Framework for MEDIUM Scenarios

For Waker sandbox implementation, reasonable assumptions can be defined:

| Scenario | Suggested Assumption | Rationale |
|----------|---------------------|-----------|
| S03 | CPC = R$2-5, Budget = R$10K-50K/month | Industry standard ranges |
| S04 | SEO investment = R$5K-20K/month | Content creation costs |
| S05 | Category margin = 15-30% (varies) | E-commerce typical margins |
| S10 | Processing capacity = linear scaling | Simplified model for initial simulation |

---

### Not Ready for Sandbox (LOW Priority)

These 2 scenarios require significant additional data collection:

| Scenario | Name | Why Not Ready | Data Collection Needed |
|----------|------|---------------|----------------------|
| **S06** | Seller Quality Increase | Complex lifecycle model needed | Recruitment pipeline, training effectiveness, replacement timeline |
| **S09** | Repeat Purchase Enhancement | Customer behavior model needed | Retention program costs, expected uplift, customer segmentation |

---

## Variable-to-Scenario Mapping

| Variable | Current Value | Scenarios Using Variable | Priority Level |
|----------|--------------|-------------------------|----------------|
| late_delivery_rate | 8.11% | S01 | HIGH |
| delivery_total_days | 12.56 days mean | S02, S10 | HIGH, MEDIUM |
| state_late_rate | >15% (worst states) | S07 | HIGH |
| seller_activation_rate | 45.1% | S08 | HIGH |
| marketing_channel_conversion | 2.67-16.29% varies | S03, S04 | MEDIUM |
| category_gmv_concentration | Top 5 = 40.3% | S05 | MEDIUM |
| seller_quality | 98 low-score sellers | S06 | LOW |
| repeat_purchase_rate | 3.4% | S09 | LOW |

---

## Waker Sandbox Implementation Roadmap

### Phase 1: Immediate Implementation (Week 1-2)

**Focus**: S01, S02, S07, S08

These scenarios can be implemented immediately with existing data:

| Scenario | Input Variable | Input Range | Expected Impact Metric |
|----------|---------------|-------------|----------------------|
| S01 | late_delivery_rate | 4-8% | review_score improvement |
| S02 | delivery_total_days | 10-13 days | review_score improvement |
| S07 | state_late_rate (AL/MA/PI/CE/SE) | 8-15% | state_score improvement |
| S08 | seller_activation_rate | 45-55% | GMV uplift |

### Phase 2: Assumption-Based Implementation (Week 3-4)

**Focus**: S03, S04, S05, S10

Add cost assumptions and implement with sensitivity analysis:

| Scenario | Key Assumption | Sensitivity Range |
|----------|---------------|-------------------|
| S03 | CPC, conversion elasticity | CPC: R$1-5, Conversion delta: 2-5% |
| S04 | SEO investment, time-to-impact | R$5K-20K/month, 3-6 month lag |
| S05 | Category margin, growth cost | Margin: 15-30%, Marketing: 5-10% of GMV |
| S10 | Processing cost per day saved | R$50-200 per day reduction |

### Phase 3: Data Collection for Future Scenarios (Month 2+)

**Focus**: S06, S09

Design data collection strategy to enable these scenarios in future:

| Data Needed | Collection Method | Estimated Timeline |
|-------------|------------------|-------------------|
| Seller recruitment pipeline | CRM integration | 2-4 weeks |
| Retention program effectiveness | A/B test design | 4-8 weeks |
| Category profitability | Accounting data collection | 4-6 weeks |

---

## Risk Assessment for Sandbox Implementation

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Correlation ≠ Causation | Simulation output may overestimate impact | Acknowledge in scenario documentation; validate with business teams |
| Assumption Accuracy | Cost assumptions may not reflect reality | Use sensitivity analysis; range-based inputs |
| Data Freshness | Phase 3-4 data may be outdated | Update interim data when new data becomes available |
| Variable Interdependence | Changing multiple variables simultaneously | Start with single-variable scenarios; add multi-variable later |

---

## Output Files Summary

| File | Location | Content |
|------|----------|---------|
| **simulation_variable_map.md** | `docs/` | Complete variable descriptions with current values and impacted metrics |
| **what_if_scenarios.md** | `docs/` | Detailed scenario designs with all 10 scenarios |
| **simulation_variable_mapping.csv** | `outputs/tables/` | Machine-readable variable mapping table |
| **scenario_design_catalog.csv** | `outputs/tables/` | Machine-readable scenario catalog |
| **simulation_readiness_report.md** | `reports/` | This report |

---

## Quick Reference: Scenario Priority Matrix

| | HIGH Impact | MEDIUM Impact | LOW Impact |
|--|-------------|---------------|-----------|
| **HIGH Data Support** | S01, S02, S07, S08 ← **START HERE** | - | - |
| **PARTIAL Data Support** | - | S03, S04, S05, S10 | - |
| **LOW Data Support** | - | - | S06, S09 |

---

**Conclusion**: Begin with **S01 (Late Delivery)**, **S02 (Delivery Time)**, **S07 (Regional Logistics)**, and **S08 (Seller Activation)** for the Waker Decision Sandbox. These four scenarios have direct data support, clear business impact, and quantifiable outputs.

**Next Actions**:
1. Implement HIGH priority scenarios in Waker sandbox
2. Define cost assumptions for MEDIUM priority scenarios
3. Design data collection strategy for LOW priority scenarios

---

**All analysis reproducible from Phase 1-5 outputs**
