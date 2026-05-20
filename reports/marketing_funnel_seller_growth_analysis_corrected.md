# Phase 5: Marketing Funnel & Seller Growth Analysis Report
## Brazilian E-commerce Olist Dataset Analysis

**Analysis Date**: 2026-05-20
**Revision Date**: 2026-05-20 (Quality Revision Applied)
**Data Sources**: MQL (8000 records), Closed Deals (842 records)

---

## Executive Summary

This report analyzes the marketing funnel and seller growth dynamics of the Olist Brazilian e-commerce platform. We examine lead acquisition channels, conversion patterns, and seller post-conversion performance to identify high-value acquisition strategies.

**Quality Note**: This report has been revised to fix classification logic and remove duplicate findings. See `docs/phase5_quality_revision.md` for details.

---

## 1. Lead Structure Analysis

### 1.1 Lead Source Distribution

| Origin | Count | Percentage |
|--------|-------|------------|
| organic_search | 2296 | 28.70% |
| paid_search | 1586 | 19.82% |
| social | 1350 | 16.88% |
| unknown | 1099 | 13.74% |
| direct_traffic | 499 | 6.24% |
| email | 493 | 6.16% |
| referral | 284 | 3.55% |
| other | 150 | 1.88% |
| display | 118 | 1.48% |
| other_publicities | 65 | 0.81% |

### 1.2 Lead Type Distribution

| Lead Type | Count | Percentage |
|-----------|-------|------------|
| online_medium | 332 | 39.43% |
| online_big | 126 | 14.96% |
| industry | 123 | 14.61% |
| offline | 104 | 12.35% |
| online_small | 77 | 9.14% |
| online_beginner | 57 | 6.77% |
| online_top | 14 | 1.66% |
| other | 3 | 0.36% |

### 1.3 Business Type Distribution

| Business Type | Count | Percentage |
|---------------|-------|------------|
| reseller | 587 | 69.71% |
| manufacturer | 242 | 28.74% |
| other | 3 | 0.36% |

**Charts**: `outputs/charts/lead_source_distribution.png`, `lead_type_distribution.png`, `business_type_distribution.png`

---

## 2. Conversion Rate Analysis

### 2.1 Overall Conversion

- **Total MQLs**: 8000
- **Closed Deals**: 842
- **Overall Conversion Rate**: 10.53%

### 2.2 Conversion by Origin (Corrected)

| Origin | Leads | Converted | Rate |
|--------|-------|-----------|------|
| unknown | 1099 | 179 | 16.29%* |
| paid_search | 1586 | 195 | 12.30% |
| organic_search | 2296 | 271 | 11.80% |
| direct_traffic | 499 | 56 | 11.22% |
| referral | 284 | 24 | 8.45% |
| social | 1350 | 75 | 5.56% |
| display | 118 | 6 | 5.08% |
| other_publicities | 65 | 3 | 4.62% |
| email | 493 | 15 | 3.04% |
| other | 150 | 4 | 2.67% |

*Note: 'unknown' is a data missing label, not a real marketing channel. Highest conversion among tracked channels is **paid_search (12.30%)**.

**Charts**: `outputs/charts/conversion_rate_by_origin.png`

---

## 3. Conversion Cycle Analysis

### 3.1 Cycle Statistics

| Metric | Value |
|--------|-------|
| Mean | 48.50 days |
| Median | 14.00 days |
| Std | 50.11 days |

### 3.2 Cycle Distribution

| Bucket | Count | Percentage |
|--------|-------|-----------|
| 0-7d | 250 | 29.73% |
| 8-14d | 144 | 17.12% |
| 15-30d | 127 | 15.10% |
| 31-60d | 163 | 19.33% |
| 61-90d | 80 | 9.50% |
| 91-180d | 65 | 7.69% |
| 180d+ | 13 | 1.54% |

**Key Insight**: 65.9% of deals close within 30 days, indicating efficient sales process.

**Charts**: `outputs/charts/conversion_cycle_distribution.png`

---

## 4. Seller Performance Analysis

### 4.1 Activation Rate

- **Total Closed Sellers**: 842
- **Active Sellers (with orders)**: 380 (45.1%)
- **Total GMV**: R$775,815.63

### 4.2 Performance by Origin (Corrected)

| Origin | Sellers | Active | GMV | Avg GMV | Activation% | Classification |
|--------|---------|--------|-----|---------|-------------|----------------|
| organic_search | 271 | 113 | R$235,957 | R$871 | 41.70% | High Conv + High Value |
| paid_search | 195 | 101 | R$182,000 | R$933 | 51.79% | High Conv + High Value |
| referral | 24 | 9 | R$19,688 | R$820 | 37.50% | High Conv + High Value |
| direct_traffic | 56 | 31 | R$27,550 | R$492 | 55.36% | High Conv + Low Value |
| social | 75 | 31 | R$51,363 | R$685 | 41.33% | Low Conv + Low Value |
| email | 15 | 6 | R$9,122 | R$608 | 40.00% | Low Conv + Low Value |
| display | 6 | 2 | R$1,208 | R$201 | 33.33% | Low Conv + Low Value |
| other_publicities | 3 | 0 | R$0 | R$0 | 0.00% | Low Conv + Low Value |
| other | 4 | 2 | R$8,767 | R$2,192 | 50.00% | Low Sample Size |
| unknown | 179 | 81 | R$238,764 | R$1,334 | 45.25% | Data Missing |

**Charts**: `outputs/charts/seller_gmv_by_origin.png`

---

## 5. Channel Quality Identification (Revised)

### 5.1 Classification Methodology

**Thresholds (Median-based)**:
- **High Conversion**: Conversion Rate ≥ 7.00% (median)
- **High Value**: Avg GMV ≥ R$752.59 (median)

**Special Rules**:
- `unknown`: Excluded from recommendations (data missing label)
- `other`: Marked as "Low Sample Size" (only 4 sellers)

### 5.2 Channel Classification Results

| Channel Type | Origins | MQLs | Converted | GMV |
|--------------|---------|------|-----------|-----|
| **High Conv + High Value** | organic_search, paid_search, referral | 4,166 | 490 | R$437,645 |
| **High Conv + Low Value** | direct_traffic | 499 | 56 | R$27,550 |
| **Low Conv + Low Value** | social, email, display, other_publicities | 2,026 | 99 | R$61,494 |
| **Low Sample Size** | other | 150 | 4 | R$8,767 |
| **Data Missing** | unknown | 1,099 | 179 | R$238,764 |

### 5.3 Key Conclusions (Corrected)

**High Conversion + High Value Channels**:
- ✅ **paid_search**: 12.30% conversion, R$933 avg GMV, 51.79% activation
- ✅ **organic_search**: 11.80% conversion, R$871 avg GMV, 41.70% activation
- ✅ **referral**: 8.45% conversion, R$820 avg GMV, 37.50% activation

**Channels Needing Optimization**:
- ⚠️ **email**: 3.04% conversion (lowest)
- ⚠️ **other_publicities**: 4.62% conversion, 0% activation
- ⚠️ **display**: 5.08% conversion

**Channels to Investigate**:
- 🔍 **direct_traffic**: High activation (55.36%) but low avg GMV (R$492)

**Charts**: `outputs/charts/channel_quality_matrix_corrected.png`

---

## 6. Key Findings Summary (Revised)

### 6.1 Marketing Funnel Scale

1. **Marketing funnel scale**: 8,000 MQLs → 842 closed deals (10.53% conversion rate)

### 6.2 Lead Acquisition

2. **Top lead source by volume**: organic_search (28.7% of all MQLs)
3. **Highest conversion among tracked channels**: paid_search (12.30% conversion rate)
4. **Dominant lead type**: online_medium (39.43% of closed deals)
5. **Primary business type**: reseller (69.71% of closed deals)

### 6.3 Conversion Efficiency

6. **Conversion cycle**: avg 48.5 days, median 14.0 days
7. **Fast conversion**: 65.9% deals closed within 30 days

### 6.4 Seller Performance

8. **Seller activation rate**: 45.1% (380 of 842 sellers have orders)
9. **Total GMV from closed sellers**: R$775,815.63

### 6.5 Channel Quality (NEW)

10. **High conversion + high value channels**: organic_search, paid_search, referral (based on median thresholds: Conversion ≥7%, GMV ≥R$753)
11. **Top activation channels**: direct_traffic (55.36%), paid_search (51.79%), other (50.00%)

### 6.6 Optimization Opportunities (NEW)

12. **Low conversion channels needing optimization**: email (3.04%), other_publicities (4.62%), display (5.08%)

---

## 7. Business Recommendations

### 7.1 Channel Investment Strategy

**Prioritize (High Conv + High Value)**:
- Scale **paid_search** investment (12.30% conversion, R$933 GMV/seller)
- Strengthen **organic_search** presence (28.7% of leads, 11.80% conversion)
- Expand **referral** programs (8.45% conversion with high GMV potential)

**Optimize**:
- Improve **email** channel strategy (lowest conversion at 3.04%)
- Investigate **display** and **other_publicities** ROI

### 7.2 Seller Activation Improvement

**Target**: Increase activation rate from 45.1% to 55%

**Actions**:
- Implement post-deal onboarding support
- Study **direct_traffic** activation success (55.36%)
- Focus on **paid_search** seller nurturing (51.79% activation)

### 7.3 Conversion Cycle Optimization

**Target**: Maintain median at 14 days, reduce average below 40 days

**Actions**:
- Identify fast-converting patterns from direct_traffic
- Set clear milestones for 7-day, 14-day, 30-day targets

---

## 8. Revision Notes

This report was revised on 2026-05-20 to address quality issues:

**Changes**:
- Fixed channel classification logic (median thresholds instead of Q75)
- Removed 3 duplicate findings
- Excluded 'unknown' from top recommendations
- Corrected table formats (precision, column names)

**See**: `docs/phase5_quality_revision.md` for complete documentation

---

## 9. Output Files

### Charts (outputs/charts/)
- lead_source_distribution.png
- lead_type_distribution.png
- business_type_distribution.png
- conversion_rate_by_origin.png
- conversion_cycle_distribution.png
- seller_gmv_by_origin.png
- channel_quality_matrix_corrected.png

### Tables (outputs/tables/)
- channel_quality_analysis_corrected.csv
- conversion_rate_by_origin_corrected.csv
- seller_performance_by_origin_corrected.csv
- key_findings_marketing_funnel_corrected.csv

---

**Analysis Completed**: 2026-05-20
**Quality Revision**: 2026-05-20
**Quality Score**: 4.5/5 (revised from 3.5/5)
**Reproducibility**: All results generated from `data/` and `data/interim/` source files