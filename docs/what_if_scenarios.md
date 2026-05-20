# Phase 6: What-If Scenario Catalog
## Decision Scenarios for Business Simulation

**Documentation Date**: 2026-05-20
**Design Basis**: Phase 1-5 Analysis Results

---

## Scenario Design Methodology

Each scenario includes:
- **Scenario ID**: Unique identifier (S01-S10)
- **Category**: Fulfillment / Marketing / Category / Regional / Seller Quality / Activation / Retention
- **Input Variable**: The lever to adjust in simulation
- **Input Assumption**: The proposed change from baseline
- **Impacted Metrics**: What business metrics this change affects
- **Data Support**: Whether current data supports quantitative simulation
- **Priority for Sandbox**: HIGH / MEDIUM / LOW for Waker decision sandbox prioritization

---

## HIGH Priority Scenarios (Ready for Immediate Sandbox)

### S01: Late Delivery Optimization

| Attribute | Value |
|-----------|-------|
| **Scenario ID** | S01 |
| **Category** | Fulfillment |
| **Input Variable** | late_delivery_rate |
| **Input Assumption** | Reduce late delivery rate from 8.11% to 4% |
| **Impacted Metrics** | review_score, repeat_purchase_rate, customer_retention |
| **Impact Direction** | Positive: Reduced late rate → Higher satisfaction |
| **Historical Data Basis** | Phase 4: Low-score customers have 33.75% late rate vs 3.50% for high-score (30 point gap!) |
| **Current Data Support** | ✅ YES - Direct correlation between late rate and score (correlation_heatmap shows delay_days -0.267) |
| **Needed Extra Data** | None for initial simulation; Marketing cost for ROI in advanced version |
| **Quantification Approach** | Using score_vs_metrics.csv data, estimate score improvement from late rate reduction |
| **Business Case** | Reducing late rate to 4% could significantly reduce the 12.71% low-score customer segment |

---

### S02: Delivery Time Reduction

| Attribute | Value |
|-----------|-------|
| **Scenario ID** | S02 |
| **Category** | Fulfillment |
| **Input Variable** | delivery_total_days |
| **Input Assumption** | Reduce mean delivery time from 12.56 to 10.0 days (-20%) |
| **Impacted Metrics** | review_score, cancel_rate, seller_rating, repeat_purchase |
| **Impact Direction** | Positive: Faster delivery → Higher satisfaction |
| **Historical Data Basis** | Phase 4: Correlation between delivery days and score is -0.334; Low-score avg 20.21 days vs high-score 11.09 days |
| **Current Data Support** | ✅ YES - Direct correlation data available |
| **Needed Extra Data** | Logistics cost per day reduction for ROI analysis |
| **Quantification Approach** | Using correlation -0.334, estimate delta score from -2.56 day reduction |
| **Business Case** | Closing the 9.12-day gap between low/high score delivery times would improve satisfaction |

---

### S07: Regional Logistics Improvement

| Attribute | Value |
|-----------|-------|
| **Scenario ID** | S07 |
| **Category** | Regional |
| **Input Variable** | state_late_rate_variation |
| **Input Assumption** | Reduce worst states (AL/MA/PI/CE/SE) late rate from 15%+ to 8% (align with average) |
| **Impacted Metrics** | state_review_score, seller_activation_cost, market_expansion_potential |
| **Impact Direction** | Positive: Regional equity → Better overall satisfaction, unlock new markets |
| **Historical Data Basis** | Phase 4: 5 states with >15% late rate; delivery time varies 8.76-29.39 days by state |
| **Current Data Support** | ✅ YES - Direct state-level performance data in state_delivery_performance.csv |
| **Needed Extra Data** | Regional logistics investment required for improvement |
| **Quantification Approach** | Score improvement from late rate reduction using national correlation patterns |
| **Business Case** | Geographic expansion beyond São Paulo requires addressing regional delivery gaps |

---

### S08: Seller Activation Rate Improvement

| Attribute | Value |
|-----------|-------|
| **Scenario ID** | S08 |
| **Category** | Activation |
| **Input Variable** | seller_activation_rate |
| **Input Assumption** | Increase activation from 45.1% to 55% (+10 percentage points) |
| **Current State** | 380 out of 842 closed sellers have orders on platform |
| **Best Performers** | direct_traffic: 55.36%, paid_search: 51.79% (already above 50%) |
| **Impacted Metrics** | total_GMV, platform_revenue, marketing_channel_ROI |
| **Impact Direction** | Positive: Higher activation → More GMV from same acquisition |
| **Historical Data Basis** | Phase 5: 54.9% of closed sellers never place an order (activation opportunity) |
| **Current Data Support** | ✅ YES - Direct activation data by channel in seller_performance_by_origin_corrected.csv |
| **Needed Extra Data** | Activation program cost (onboarding, incentives, support) |
| **Quantification Approach** | GMV uplift = (Additional activated sellers) × (Average GMV per active seller from closed sellers) |
| **Business Case** | Activating 84 additional sellers (55% - 45.1%) could drive significant incremental GMV |

---

## MEDIUM Priority Scenarios (Partial Data Support)

### S03: Paid Search Investment Expansion

| Attribute | Value |
|-----------|-------|
| **Scenario ID** | S03 |
| **Category** | Marketing |
| **Input Variable** | marketing_channel_conversion (paid_search) |
| **Input Assumption** | Increase paid_search conversion from 12.3% to 15% |
| **Current State** | paid_search: 1,586 leads, 195 closed, 12.3% conversion; High Conv + High Value channel |
| **Impacted Metrics** | seller_activation_rate, total_GMV, channel_ROI |
| **Impact Direction** | Positive: Higher conversion → More closed deals, better ROI |
| **Historical Data Basis** | Phase 5: paid_search is High Conv + High Value channel (conversion ≥7%, GMV ≥R$753) |
| **Current Data Support** | ⚠️ PARTIAL - Conversion data available; lacks cost-per-click/campaign budget |
| **Needed Extra Data** | Paid search campaign budget, CPC, cost per closed deal |
| **Quantification Approach** | Additional deals = Leads × (New Rate - Old Rate); Revenue impact needs cost data |
| **Business Case** | paid_search already converts well (12.3%); optimization could yield more efficiently |

---

### S04: Organic Search Conversion Optimization

| Attribute | Value |
|-----------|-------|
| **Scenario ID** | S04 |
| **Category** | Marketing |
| **Input Variable** | marketing_channel_conversion (organic_search) |
| **Input Assumption** | Increase organic_search conversion from 11.8% to 14% |
| **Current State** | organic_search: 2,296 leads (28.7% of all MQLs); 11.8% conversion; High Conv + High Value |
| **Impacted Metrics** | seller_activation_rate, total_GMV, CAC_reduction |
| **Impact Direction** | Positive: Higher conversion + free traffic → Lower CAC |
| **Historical Data Basis** | Phase 5: Highest volume source (28.7%), already classified as High Conv + High Value |
| **Current Data Support** | ⚠️ PARTIAL - Conversion and volume data available; SEO investment costs not tracked |
| **Needed Extra Data** | SEO/content investment, lead quality by source |
| **Quantification Approach** | 2,296 leads × (14% - 11.8%) = ~50 additional deals if conversion improves |
| **Business Case** | Organic has highest volume; even small conversion improvement yields meaningful gains |

---

### S05: High-Potential Category Growth

| Attribute | Value |
|-----------|-------|
| **Scenario ID** | S05 |
| **Category** | Category |
| **Input Variable** | category_gmv_concentration |
| **Input Assumption** | Increase top 5 category GMV by 20% |
| **Current State** | Top 5 categories: health_beauty, watches_gifts, bed_bath_table, sports_leisure, computers_accessories (40.3% GMV) |
| **Impacted Metrics** | total_GMV, avg_price, seller_specialization |
| **Impact Direction** | Positive: Category growth → Revenue increase |
| **Historical Data Basis** | Phase 3: 16 categories drive 80% GMV; Pareto distribution identified |
| **Current Data Support** | ⚠️ PARTIAL - GMV by category available; category margins and growth elasticity not tracked |
| **Needed Extra Data** | Category profit margin, growth investment required, cross-category cannibalization |
| **Quantification Approach** | GMV uplift = Current top 5 GMV × 20%; Needs margin for profit impact |
| **Business Case** | Focus on top 5 categories could drive 8% overall GMV growth (40.3% × 20%) |

---

### S10: Approval-to-Shipping Speed Improvement

| Attribute | Value |
|-----------|-------|
| **Scenario ID** | S10 |
| **Category** | Fulfillment |
| **Input Variable** | approval_to_shipping_days |
| **Input Assumption** | Reduce from 2.80 to 2.00 days (-0.80 days, -29%) |
| **Current State** | Mean 2.80 days, Median 1.82 days, Std 3.54 (high variability) |
| **Impacted Metrics** | delivery_total_days, customer_satisfaction |
| **Impact Direction** | Positive: Faster approval→shipping → Shorter total cycle |
| **Historical Data Basis** | Phase 4: Identified as bottleneck stage with highest variability (std 3.54 days) |
| **Current Data Support** | ⚠️ PARTIAL - Time data available; seller processing capacity constraints not modeled |
| **Needed Extra Data** | Seller processing capacity, approval automation cost |
| **Quantification Approach** | Direct time reduction contribution to total delivery (-0.80 days) |
| **Business Case** | 0.80-day reduction contributes directly to delivery time improvement goal |

---

## LOW Priority Scenarios (Need Significant Additional Data)

### S06: High-Quality Seller Proportion Increase

| Attribute | Value |
|-----------|-------|
| **Scenario ID** | S06 |
| **Category** | Seller Quality |
| **Input Variable** | seller_quality_distribution |
| **Input Assumption** | Increase high-quality sellers (avg score ≥4.5) by 30% |
| **Current State** | 98 high-volume sellers identified with average score <4.0 (problem sellers) |
| **Impacted Metrics** | avg_delivery_time, review_score, platform_risk |
| **Impact Direction** | Positive: Better sellers → Better platform reputation |
| **Historical Data Basis** | Phase 4: 98 high-volume low-score sellers; Phase 3: 540 sellers drive 80% GMV |
| **Current Data Support** | ⚠️ PARTIAL - Seller scores available; recruitment timeline and replacement rate not modeled |
| **Needed Extra Data** | Seller recruitment pipeline, training program effectiveness, replacement timelines |
| **Quantification Approach** | Complex: Requires seller lifecycle model; simulation not directly feasible with current data |
| **Business Case** | Long-term platform quality depends on seller quality, not immediate impact |

---

### S09: Repeat Purchase Enhancement

| Attribute | Value |
|-----------|-------|
| **Scenario ID** | S09 |
| **Category** | Retention |
| **Input Variable** | repeat_purchase_rate |
| **Input Assumption** | Increase from 3.4% to 8% |
| **Current State** | 3.4% repeat rate (96,096 unique buyers, 99,441 orders = very low retention) |
| **Impacted Metrics** | customer_lifetime_value, GMV_stability, CAC_reduction |
| **Impact Direction** | Positive: Higher retention → Lower acquisition dependency, stable GMV |
| **Historical Data Basis** | Phase 3: Low repeat rate identified (3.4%); high acquisition dependency |
| **Current Data Support** | ⚠️ PARTIAL - Baseline available; retention program costs and effectiveness not modeled |
| **Needed Extra Data** | Retention program types (loyalty, personalized offers, email campaigns), cost per retention, expected uplift |
| **Quantification Approach** | Complex: Requires customer behavior model; not directly feasible |
| **Business Case** | Low current rate suggests significant improvement potential; long-term priority |

---

## Scenario Implementation Priority Ranking

| Priority | Scenario IDs | Rationale |
|----------|--------------|-----------|
| **HIGH** | S01, S02, S07, S08 | Direct data support; clear business impact; immediate actionable |
| **MEDIUM** | S03, S04, S05, S10 | Partial data; needs marketing/cost assumptions for full quantification |
| **LOW** | S06, S09 | Significant additional data needed; complex modeling required |

---

## Data Requirements Summary

### ✅ Available for Quantitative Simulation
- late_delivery_rate and its impact on review_score (Phase 4 correlation matrix)
- delivery_total_days and its impact metrics (Phase 4 fulfillment stats)
- seller_activation_rate by channel (Phase 5 channel performance data)
- state_late_rate_variation (Phase 4 regional analysis)
- marketing_channel_conversion rates (Phase 5 funnel analysis)

### ⚠️ Needs Cost/Budget Assumptions
- Marketing spend by channel (for ROI calculation)
- Logistics improvement costs
- Seller activation program costs
- SEO/investment costs

### ❌ Not Available for Simulation
- Category profit margins
- Customer retention program costs
- Seller recruitment timelines
- Product-level cost data

---

## Next Steps for Waker Decision Sandbox

1. **Start with S01, S02, S07, S08** - These have direct data support
2. **Add cost assumptions for MEDIUM tier** - Enable channel investment scenarios
3. **Model S06, S09 as "future roadmap"** - Design data collection strategy

---

**All scenarios designed from Phase 1-5 analysis results**
