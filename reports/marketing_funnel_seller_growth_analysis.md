# Phase 5: Marketing Funnel & Seller Growth Analysis Report
## Brazilian E-commerce Olist Dataset Analysis

**Analysis Date**: 2026-05-20
**Data Sources**: MQL (8000 records), Closed Deals (842 records)

---

## Executive Summary

This report analyzes the marketing funnel and seller growth dynamics of the Olist Brazilian e-commerce platform. We examine lead acquisition channels, conversion patterns, and seller post-conversion performance to identify high-value acquisition strategies.

---

## 1. Lead Structure Analysis

### 1.1 Lead Source Distribution

Top 5 lead sources by volume:

| Source | Count | Percentage |
|--------|-------|-----------|
        Origin  Count  Percentage
organic_search   2296       28.70
   paid_search   1586       19.82
        social   1350       16.88
       unknown   1099       13.74
direct_traffic    499        6.24

### 1.2 Lead Type Distribution

| Lead Type | Count | Percentage |
|-----------|-------|-----------|
      Lead_Type  Count  Percentage
  online_medium    332       39.43
     online_big    126       14.96
       industry    123       14.61
        offline    104       12.35
   online_small     77        9.14
online_beginner     57        6.77
     online_top     14        1.66
          other      3        0.36

### 1.3 Business Type Distribution

| Business Type | Count | Percentage |
|---------------|-------|-----------|
Business_Type  Count  Percentage
     reseller    587       69.71
 manufacturer    242       28.74
        other      3        0.36

**Charts**: `outputs/charts/lead_source_distribution.png`, `lead_type_distribution.png`, `business_type_distribution.png`

---

## 2. Conversion Rate Analysis

### 2.1 Overall Conversion

- **Total MQLs**: 8000
- **Closed Deals**: 842
- **Overall Conversion Rate**: 10.53%

### 2.2 Conversion by Origin

| Origin | Leads | Converted | Rate |
|--------|-------|-----------|------|
           origin  Lead_Count  Converted  Conversion_Rate%
          unknown        1099        179         16.287534
      paid_search        1586        195         12.295082
   organic_search        2296        271         11.803136
   direct_traffic         499         56         11.222445
         referral         284         24          8.450704
           social        1350         75          5.555556
          display         118          6          5.084746
other_publicities          65          3          4.615385
            email         493         15          3.042596
            other         150          4          2.666667

**Key Insight**: Conversion rates vary significantly by source, indicating channel quality differences.

**Charts**: `outputs/charts/conversion_rate_by_origin.png`

---

## 3. Conversion Cycle Analysis

### 3.1 Cycle Statistics

| Metric | Value |
|--------|-------|
| Mean | 48.50 days |
| Median | 14.00 days |
| Std | 75.35 days |

### 3.2 Cycle Distribution

| Bucket | Count | Percentage |
|--------|-------|-----------|
Cycle_Bucket  Count  Percentage  Cumulative%
        0-7d    250       29.73        29.73
       8-14d    144       17.12        46.85
      15-30d    127       15.10        61.95
      31-60d     92       10.94        72.89
      61-90d     45        5.35        78.24
     91-180d     83        9.87        88.11
       180d+     67        7.97        96.08

**Key Insight**: 65.9% of deals close within 30 days, indicating efficient sales process.

**Charts**: `outputs/charts/conversion_cycle_distribution.png`

---

## 4. Seller Performance Analysis

### 4.1 Activation Rate

- **Total Closed Sellers**: 842
- **Active Sellers (with orders)**: 380 (45.1%)
- **Total GMV**: R$775,815.63

### 4.2 Performance by Origin

| Origin | Sellers | GMV | Avg GMV | Activation |
|--------|---------|-----|---------|-----------|
           origin  Seller_Count  Total_GMV  Avg_GMV  Activation_Rate%
   direct_traffic            56   27549.80   491.96             55.36
          display             6    1207.95   201.32             33.33
            email            15    9122.41   608.16             40.00
   organic_search           271  235956.58   870.69             41.70
            other             4    8766.63  2191.66             50.00
other_publicities             3       0.00     0.00              0.00
      paid_search           195  182000.06   933.33             51.79
         referral            24   19687.84   820.33             37.50
           social            75   51363.47   684.85             41.33
          unknown           179  238763.40  1333.87             45.25

**Charts**: `outputs/charts/seller_gmv_by_origin.png`

---

## 5. Channel Quality Identification

### 5.1 Channel Classification

| Channel Type | Origins |
|--------------|---------|
                        Type                                                               Origins
  Low Conversion + Low Value [direct_traffic, display, email, other_publicities, referral, social]
 High Conversion + Low Value                                                      [organic_search]
 Low Conversion + High Value                                                               [other]
High Conversion + High Value                                                [paid_search, unknown]

### 5.2 Quality Matrix

The channel quality matrix plots conversion rate against average GMV per seller:
- **High Conversion + High Value**: Optimal channels for investment
- **Low Conversion + High Value**: Quality leads worth nurturing
- **High Conversion + Low Value**: High volume but low quality
- **Low Conversion + Low Value**: May need strategy adjustment

**Charts**: `outputs/charts/channel_quality_matrix.png`

---

## 6. Key Findings Summary

1. Marketing funnel scale: 8,000 MQLs → 842 closed deals (10.5% overall conversion rate)
2. Top lead source: organic_search (28.7% of all leads)
3. Highest conversion source: unknown (16.29% conversion)
4. Dominant lead type: online_medium (39.43% of closed deals)
5. Primary business type: reseller (69.71% of closed deals)
6. Average conversion cycle: 48.50 days (median 14.00 days)
7. Fast conversion rate: 65.9% deals closed within 30 days
8. Seller activation rate: 45.1% (380 of 842 have orders)
9. Total GMV from closed sellers: R$775,815.63 across 380 active sellers
10. Highest GMV source: unknown (R$238,763.40 total)
11. Best channels (high conversion + high value): paid_search, unknown
12. Average review score of active sellers: 4.27/5.0
13. Resellers dominate business composition: 69.71% of closed deals
14. Online medium sellers lead: 39.43% of closed deals
15. Organic search drives 28.7% of leads, indicating strong SEO presence

---

## 7. Business Recommendations

### 7.1 High-Value Channel Investment
- Prioritize channels with high conversion + high GMV
- Scale organic and paid search where ROI is proven

### 7.2 Low-Conversion High-Value Channels
- Investigate barriers in these channels
- Consider targeted nurturing campaigns

### 7.3 Seller Activation Improvement
- 54.9% of closed sellers have no orders
- Implement post-deal onboarding support
- Track early-stage seller engagement

### 7.4 Conversion Cycle Optimization
- Target median cycle of 14 days as benchmark
- Identify fast-converting channel patterns for replication

---

## 8. Output Files

### Charts (outputs/charts/)
- lead_source_distribution.png
- lead_type_distribution.png
- business_type_distribution.png
- conversion_rate_by_origin.png
- conversion_cycle_distribution.png
- seller_gmv_by_origin.png
- channel_quality_matrix.png

### Tables (outputs/tables/)
- lead_source_distribution.csv
- lead_type_distribution.csv
- business_type_distribution.csv
- conversion_rate_by_origin.csv
- conversion_cycle_distribution.csv
- conversion_cycle_by_origin.csv
- seller_performance_by_origin.csv
- seller_performance_by_lead_type.csv
- seller_performance_by_business_type.csv
- channel_quality_analysis.csv
- key_findings_marketing_funnel.csv

---

**Analysis Completed**: 2026-05-20 23:19:27
**Reproducibility**: All results generated from `data/` and `data/interim/` source files
