# Phase 6: Simulation Variable Map
## Operating Model Variables for Decision Scenario Design

**Documentation Date**: 2026-05-20
**Source Phases**: 1-5 Analysis Summary

---

## Executive Summary

This document maps all variables identified from Phases 1-5 analysis that can be used for business simulation and decision scenario design. Each variable includes its current baseline value, impacted metrics, directional impact, and data availability status.

---

## Variable Categories

### 1. Fulfillment Variables (Phase 4)
- delivery_total_days
- late_delivery_rate
- approval_to_shipping_days

### 2. Customer Experience Variables (Phase 4)
- review_score
- repeat_purchase_rate

### 3. Category/Seller Variables (Phase 3, 4)
- category_gmv_concentration
- seller_concentration
- seller_activation_rate

### 4. Regional Variables (Phase 3, 4)
- state_late_rate_variation

### 5. Marketing Variables (Phase 5)
- marketing_channel_conversion

### 6. Business Metric Variables (Phase 3)
- avg_order_value

---

## Variable Detail Map

### 1. delivery_total_days (Total Delivery Time)

| Attribute | Value |
|-----------|-------|
| **Description** | Total days from order placement to customer delivery |
| **Source Phase** | Phase 4 |
| **Current Value** | Mean: 12.56 days / Median: 10.22 days |
| **Distribution** | 78% early delivery / 8.11% late |
| **Impacted Metrics** | review_score, cancel_rate, repeat_purchase |
| **Impact Direction** | Negative: Longer delivery → Lower customer satisfaction |
| **Data Support** | Direct data available from order lifecycle tracking |

---

### 2. late_delivery_rate (Late Delivery Rate)

| Attribute | Value |
|-----------|-------|
| **Description** | Percentage of orders delivered after estimated delivery date |
| **Source Phase** | Phase 4 |
| **Current Value** | 8.11% late deliveries |
| **Impact Evidence** | Low-score customers have 33.75% late rate vs 3.50% for high-score (30 point gap!) |
| **Impacted Metrics** | review_score, seller_rating, customer_retention |
| **Impact Direction** | Negative: Higher late rate → Lower scores, more churn |
| **Data Support** | Direct data available |

---

### 3. approval_to_shipping_days (Approval to Shipping Time)

| Attribute | Value |
|-----------|-------|
| **Description** | Time from order approval to carrier dispatch |
| **Source Phase** | Phase 4 |
| **Current Value** | Mean: 2.80 days (High variability: std 3.54 days) |
| **Significance** | Identified as bottleneck stage in fulfillment chain |
| **Impacted Metrics** | delivery_total_days, customer_satisfaction |
| **Impact Direction** | Negative: Longer approval→shipping → Longer total delivery |
| **Data Support** | Direct data available |

---

### 4. review_score (Customer Review Score)

| Attribute | Value |
|-----------|-------|
| **Description** | Average customer review score on 1-5 scale |
| **Source Phase** | Phase 4 |
| **Current Value** | Mean: 4.16 / Median: 5.00 |
| **Distribution** | 78.40% high scores (4-5), 12.71% low scores (1-2) |
| **Key Correlations** | delivery_days: -0.334, delay_days: -0.267, freight: -0.090 |
| **Impacted Metrics** | total_GMV, repeat_purchase, customer_retention |
| **Impact Direction** | Positive: Higher scores → More retention, better platform reputation |
| **Data Support** | Direct data available |

---

### 5. category_gmv_concentration (Category GMV Concentration)

| Attribute | Value |
|-----------|-------|
| **Description** | GMV concentration in top categories |
| **Source Phase** | Phase 3 |
| **Current Value** | Top 5 categories: 40.3% GMV; 16 categories for 80% GMV (out of 73 total) |
| **Impacted Metrics** | avg_price, order_volume, seller_specialization |
| **Impact Direction** | Mixed: Concentration enables efficiency but creates risk |
| **Top Categories** | health_beauty, watches_gifts, bed_bath_table, sports_leisure, computers_accessories |
| **Data Support** | Direct data available |

---

### 6. seller_concentration (Seller Concentration)

| Attribute | Value |
|-----------|-------|
| **Description** | Percentage of GMV driven by top sellers |
| **Source Phase** | Phase 3 |
| **Current Value** | Top 10: 13.1% GMV; 540 sellers account for 80% GMV (out of 3,095 total) |
| **Impacted Metrics** | avg_delivery_time, review_score, platform_risk |
| **Impact Direction** | Mixed: Top sellers drive efficiency but create concentration risk |
| **Problem Area** | 98 high-volume sellers with average score <4.0 |
| **Data Support** | Direct data available |

---

### 7. state_late_rate_variation (Regional Late Rate Variation)

| Attribute | Value |
|-----------|-------|
| **Description** | Late delivery rate variation across customer states |
| **Source Phase** | Phase 4 |
| **Current Value** | Highest late states: AL, MA, PI, CE, SE (all >15%) |
| **Delivery Time Range** | 8.76 to 29.39 days average by state |
| **Impacted Metrics** | state_score, activation_cost, market_expansion |
| **Impact Direction** | Negative: Regional disparities limit expansion |
| **Data Support** | Direct data available |

---

### 8. marketing_channel_conversion (Marketing Channel Conversion)

| Attribute | Value |
|-----------|-------|
| **Description** | Conversion rate from MQL to closed deal by lead origin |
| **Source Phase** | Phase 5 |
| **Current Value** | paid_search: 12.3%; organic_search: 11.8%; referral: 8.45%; others lower |
| **Lead Volume Share** | organic_search: 28.7%; paid_search: 19.82%; social: 16.88% |
| **Impacted Metrics** | seller_activation, total_GMV, acquisition_cost |
| **Impact Direction** | Positive: Higher conversion → More sellers activated, better ROI |
| **Data Support** | Direct data available |

---

### 9. seller_activation_rate (Seller Activation Rate)

| Attribute | Value |
|-----------|-------|
| **Description** | Percentage of closed deals with subsequent orders on platform |
| **Source Phase** | Phase 5 |
| **Current Value** | 45.1% (380 out of 842 closed sellers have orders) |
| **By Channel** | direct_traffic: 55.36%; paid_search: 51.79%; email: 40% |
| **Impacted Metrics** | total_GMV, platform_revenue, channel_ROI |
| **Impact Direction** | Positive: Higher activation → More GMV from same acquisition spend |
| **Data Support** | Direct data available |

---

### 10. avg_order_value (Average Order Value)

| Attribute | Value |
|-----------|-------|
| **Description** | Mean payment value per order transaction |
| **Source Phase** | Phase 3 |
| **Current Value** | Mean: R$160.99 / Median: R$105.29 (right-skewed) |
| **Impacted Metrics** | total_GMV, repeat_purchase, shipping_efficiency |
| **Impact Direction** | Positive: Higher AOV drives revenue and efficiency |
| **Data Support** | Direct data available |

---

### 11. repeat_purchase_rate (Repeat Purchase Rate)

| Attribute | Value |
|-----------|-------|
| **Description** | Percentage of customers with multiple orders |
| **Source Phase** | Phase 3 |
| **Current Value** | 3.4% (low repeat rate indicates acquisition dependency) |
| **Impacted Metrics** | customer_lifetime_value, GMV_stability, CAC_reduction |
| **Impact Direction** | Positive: Higher retention → Lower acquisition cost dependency |
| **Data Support** | Direct data available |

---

## Variables Needing Extra Assumptions

### 12. channel_activation_cost (Channel Activation Cost)

| Attribute | Value |
|-----------|-------|
| **Description** | Estimated marketing cost per activated seller by channel |
| **Source Phase** | Phase 5 |
| **Current Value** | N/A - Requires marketing spend data |
| **Extra Data Needed** | Channel marketing budget, cost per lead, cost per closed deal |
| **Use Case** | ROI calculation for channel investment decisions |
| **Data Support** | ❌ NEEDS EXTRA ASSUMPTION |

---

### 13. category_profit_margin (Category Profit Margin)

| Attribute | Value |
|-----------|-------|
| **Description** | Profit margin by product category |
| **Source Phase** | Phase 3 |
| **Current Value** | N/A - Requires product cost data |
| **Extra Data Needed** | Product costs, platform take rate, category margins |
| **Use Case** | Revenue optimization and category prioritization |
| **Data Support** | ❌ NEEDS EXTRA ASSUMPTION |

---

## Variable Impact Summary

| Variable | Direct Data Impact on Review Score | Direct Data Impact on GMV | Simulation Readiness |
|----------|-----------------------------------|--------------------------|---------------------|
| delivery_total_days | ✓ Correlation -0.334 | Via satisfaction | ✅ HIGH |
| late_delivery_rate | ✓ Low-score 33.75% late | Via retention | ✅ HIGH |
| review_score | ✓ Direct metric | ✓ Driver of repeat | ✅ HIGH |
| marketing_channel_conversion | ✗ Indirect | ✓ Via activation | ✅ HIGH |
| seller_activation_rate | ✗ Indirect | ✓ Direct driver | ✅ HIGH |
| category_profit_margin | ✗ Indirect | Via pricing | ❌ NEEDS DATA |

---

**All results reproducible from Phase 3-5 analysis outputs**
