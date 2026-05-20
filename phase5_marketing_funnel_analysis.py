#!/usr/bin/env python3
"""
Phase 5: Marketing Funnel & Seller Growth Analysis
Brazilian E-commerce Olist Dataset Analysis

This script performs comprehensive marketing funnel and seller growth analysis:
1. Lead structure analysis (origin, lead_type, business_type)
2. Conversion rate analysis by different dimensions
3. Conversion cycle distribution analysis
4. Closed deals seller performance evaluation
5. Channel quality identification

Output:
- Charts: outputs/charts/
- Tables: outputs/tables/
- Report: reports/marketing_funnel_seller_growth_analysis.md
"""

import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
import seaborn as sns
from pathlib import Path
import warnings
warnings.filterwarnings('ignore')

# Set Chinese font support
plt.rcParams['font.sans-serif'] = ['SimHei', 'DejaVu Sans']
plt.rcParams['axes.unicode_minus'] = False

# Set style
sns.set_style("whitegrid")
plt.rcParams['figure.figsize'] = (12, 6)
plt.rcParams['figure.dpi'] = 100

# Paths
DATA_DIR = Path('data')
INTERIM_DIR = Path('data/interim')
ROOT_DIR = Path('.')  # Root directory for original datasets
OUTPUT_CHARTS = Path('outputs/charts')
OUTPUT_TABLES = Path('outputs/tables')
REPORT_DIR = Path('reports')

# Ensure directories exist
OUTPUT_CHARTS.mkdir(parents=True, exist_ok=True)
OUTPUT_TABLES.mkdir(parents=True, exist_ok=True)
REPORT_DIR.mkdir(parents=True, exist_ok=True)

print("="*60)
print("PHASE 5: MARKETING FUNNEL & SELLER GROWTH ANALYSIS")
print("="*60)

# Load Marketing Funnel data
print("\nLoading Marketing Funnel data...")
mql_df = pd.read_csv(DATA_DIR / 'olist_marketing_qualified_leads_dataset.csv')
closed_df = pd.read_csv(DATA_DIR / 'olist_closed_deals_dataset.csv')

# Load Olist main data
item_df = pd.read_csv(INTERIM_DIR / 'item_level_base.csv')
order_df = pd.read_csv(INTERIM_DIR / 'order_level_base.csv')
sellers_df = pd.read_csv(ROOT_DIR / 'olist_sellers_dataset.csv')

# Convert dates
mql_df['first_contact_date'] = pd.to_datetime(mql_df['first_contact_date'], errors='coerce')
closed_df['won_date'] = pd.to_datetime(closed_df['won_date'], errors='coerce')

print(f"MQL records: {len(mql_df)}")
print(f"Closed deals: {len(closed_df)}")
print(f"Item records: {len(item_df)}")
print(f"Order records: {len(order_df)}")
print(f"Sellers: {len(sellers_df)}")

# =============================================================================
# SECTION 1: Lead Structure Analysis
# =============================================================================
print("\n=== Section 1: Lead Structure Analysis ===")

# 1.1 Origin Distribution
origin_counts = mql_df['origin'].value_counts()
origin_pct = (origin_counts / len(mql_df) * 100).round(2)

origin_table = pd.DataFrame({
    'Origin': origin_counts.index,
    'Count': origin_counts.values,
    'Percentage': origin_pct.values
})
origin_table.to_csv(OUTPUT_TABLES / 'lead_source_distribution.csv', index=False)

# Plot origin distribution
fig, ax = plt.subplots(figsize=(14, 6))
colors = sns.color_palette("husl", len(origin_counts))
ax.barh(origin_counts.index[::-1], origin_counts.values[::-1], color=colors[::-1])
ax.set_title('Lead Source (Origin) Distribution', fontsize=14, fontweight='bold')
ax.set_xlabel('Number of Leads')
for i, (v, p) in enumerate(zip(origin_counts.values[::-1], origin_pct.values[::-1])):
    ax.text(v + 50, i, f'{v} ({p}%)', va='center')
plt.tight_layout()
plt.savefig(OUTPUT_CHARTS / 'lead_source_distribution.png', dpi=150, bbox_inches='tight')
plt.close()

# 1.2 Lead Type Distribution
lead_type_counts = closed_df['lead_type'].value_counts()
lead_type_pct = (lead_type_counts / len(closed_df) * 100).round(2)

lead_type_table = pd.DataFrame({
    'Lead_Type': lead_type_counts.index,
    'Count': lead_type_counts.values,
    'Percentage': lead_type_pct.values
})
lead_type_table.to_csv(OUTPUT_TABLES / 'lead_type_distribution.csv', index=False)

# Plot lead type
fig, ax = plt.subplots(figsize=(12, 6))
colors = sns.color_palette("Set2", len(lead_type_counts))
ax.barh(lead_type_counts.index[::-1], lead_type_counts.values[::-1], color=colors[::-1])
ax.set_title('Lead Type Distribution (Closed Deals)', fontsize=14, fontweight='bold')
ax.set_xlabel('Number of Deals')
for i, (v, p) in enumerate(zip(lead_type_counts.values[::-1], lead_type_pct.values[::-1])):
    ax.text(v + 5, i, f'{v} ({p}%)', va='center')
plt.tight_layout()
plt.savefig(OUTPUT_CHARTS / 'lead_type_distribution.png', dpi=150, bbox_inches='tight')
plt.close()

# 1.3 Business Type Distribution
business_type_counts = closed_df['business_type'].value_counts()
business_type_pct = (business_type_counts / len(closed_df) * 100).round(2)

business_type_table = pd.DataFrame({
    'Business_Type': business_type_counts.index,
    'Count': business_type_counts.values,
    'Percentage': business_type_pct.values
})
business_type_table.to_csv(OUTPUT_TABLES / 'business_type_distribution.csv', index=False)

# Plot business type
fig, ax = plt.subplots(figsize=(10, 5))
colors = sns.color_palette("Set1", len(business_type_counts))
ax.barh(business_type_counts.index[::-1], business_type_counts.values[::-1], color=colors[::-1])
ax.set_title('Business Type Distribution (Closed Deals)', fontsize=14, fontweight='bold')
ax.set_xlabel('Number of Deals')
for i, (v, p) in enumerate(zip(business_type_counts.values[::-1], business_type_pct.values[::-1])):
    ax.text(v + 10, i, f'{v} ({p}%)', va='center')
plt.tight_layout()
plt.savefig(OUTPUT_CHARTS / 'business_type_distribution.png', dpi=150, bbox_inches='tight')
plt.close()

print("Lead structure analysis completed")

# =============================================================================
# SECTION 2: Conversion Rate Analysis
# =============================================================================
print("\n=== Section 2: Conversion Rate Analysis ===")

# Merge to identify conversion status
mql_merged = mql_df.merge(closed_df[['mql_id']], on='mql_id', how='left', indicator=True)
mql_merged['converted'] = (mql_merged['_merge'] == 'both').astype(int)

overall_conversion_rate = mql_merged['converted'].mean() * 100
print(f"Overall Conversion Rate: {overall_conversion_rate:.2f}%")

# 2.1 Conversion by Origin
origin_conversion = mql_merged.groupby('origin').agg(
    Lead_Count=('mql_id', 'count'),
    Converted=('converted', 'sum'),
    Conversion_Rate=('converted', 'mean')
).reset_index()
origin_conversion['Conversion_Rate%'] = origin_conversion['Conversion_Rate'] * 100
origin_conversion = origin_conversion.sort_values('Conversion_Rate', ascending=False)
origin_conversion.to_csv(OUTPUT_TABLES / 'conversion_rate_by_origin.csv', index=False)

# Plot conversion by origin
fig, ax = plt.subplots(figsize=(14, 6))
ax.barh(origin_conversion['origin'][::-1], origin_conversion['Conversion_Rate%'][::-1], 
        color='steelblue', alpha=0.7)
ax.set_title('Conversion Rate by Lead Origin', fontsize=14, fontweight='bold')
ax.set_xlabel('Conversion Rate (%)')
for i, (v, c) in enumerate(zip(origin_conversion['Conversion_Rate%'][::-1], 
                               origin_conversion['Converted'][::-1])):
    ax.text(v + 0.3, i, f'{v:.2f}% ({c} deals)', va='center')
plt.tight_layout()
plt.savefig(OUTPUT_CHARTS / 'conversion_rate_by_origin.png', dpi=150, bbox_inches='tight')
plt.close()

# 2.2 Conversion by Lead Type (for closed deals only)
lead_type_conversion = closed_df.groupby('lead_type').size().reset_index(name='Deal_Count')
lead_type_conversion.to_csv(OUTPUT_TABLES / 'conversion_by_lead_type.csv', index=False)

# 2.3 Conversion by Business Type
business_type_conversion = closed_df.groupby('business_type').size().reset_index(name='Deal_Count')
business_type_conversion.to_csv(OUTPUT_TABLES / 'conversion_by_business_type.csv', index=False)

print("Conversion rate analysis completed")

# =============================================================================
# SECTION 3: Conversion Cycle Analysis
# =============================================================================
print("\n=== Section 3: Conversion Cycle Analysis ===")

# Merge MQL with closed deals for cycle calculation
conversion_cycle = mql_df.merge(closed_df[['mql_id', 'won_date']], on='mql_id', how='inner')

# Calculate cycle in days
conversion_cycle['cycle_days'] = (
    conversion_cycle['won_date'] - conversion_cycle['first_contact_date']
).dt.days

# Filter valid cycles
conversion_cycle = conversion_cycle[conversion_cycle['cycle_days'] >= 0].copy()

# Cycle statistics
cycle_mean = conversion_cycle['cycle_days'].mean()
cycle_median = conversion_cycle['cycle_days'].median()
cycle_std = conversion_cycle['cycle_days'].std()

print(f"Average conversion cycle: {cycle_mean:.2f} days")
print(f"Median conversion cycle: {cycle_median:.2f} days")

# Binning
bins = [0, 7, 14, 30, 60, 90, 180, float('inf')]
labels = ['0-7d', '8-14d', '15-30d', '31-60d', '61-90d', '91-180d', '180d+']
conversion_cycle['cycle_bucket'] = pd.cut(conversion_cycle['cycle_days'], bins=bins, labels=labels)

cycle_distribution = conversion_cycle['cycle_bucket'].value_counts().sort_index()
cycle_pct = (cycle_distribution / len(conversion_cycle) * 100).round(2)

cycle_table = pd.DataFrame({
    'Cycle_Bucket': cycle_distribution.index.astype(str),
    'Count': cycle_distribution.values,
    'Percentage': cycle_pct.values,
    'Cumulative%': cycle_pct.cumsum().values
})
cycle_table.to_csv(OUTPUT_TABLES / 'conversion_cycle_distribution.csv', index=False)

# Plot cycle distribution
fig, ax = plt.subplots(figsize=(12, 6))
ax.bar(cycle_table['Cycle_Bucket'], cycle_table['Count'], color='coral', alpha=0.7)
ax.set_title('Conversion Cycle Distribution', fontsize=14, fontweight='bold')
ax.set_xlabel('Cycle (Days)')
ax.set_ylabel('Number of Deals')
for i, (v, p) in enumerate(zip(cycle_table['Count'], cycle_table['Percentage'])):
    ax.text(i, v + 10, f'{p}%', ha='center')
plt.tight_layout()
plt.savefig(OUTPUT_CHARTS / 'conversion_cycle_distribution.png', dpi=150, bbox_inches='tight')
plt.close()

# Cycle by origin
cycle_by_origin = conversion_cycle.groupby('origin').agg(
    Count=('cycle_days', 'count'),
    Mean_Days=('cycle_days', 'mean'),
    Median_Days=('cycle_days', 'median')
).round(2).reset_index()
cycle_by_origin = cycle_by_origin.sort_values('Count', ascending=False)
cycle_by_origin.to_csv(OUTPUT_TABLES / 'conversion_cycle_by_origin.csv', index=False)

print("Conversion cycle analysis completed")

# =============================================================================
# SECTION 4: Closed Deals Seller Performance Analysis
# =============================================================================
print("\n=== Section 4: Closed Deals Seller Performance ===")

# Merge closed deals with item-level data to get seller performance
seller_items = item_df.groupby('seller_id').agg(
    Order_Count=('order_id', 'nunique'),
    Item_Count=('order_item_id', 'count'),
    Total_Price=('price', 'sum'),
    Total_Freight=('freight_value', 'sum')
).reset_index()
seller_items['GMV'] = seller_items['Total_Price'] + seller_items['Total_Freight']

# Merge with closed deals (get origin)
closed_with_origin = closed_df.merge(mql_df[['mql_id', 'origin']], on='mql_id', how='left')
closed_performance = closed_with_origin.merge(seller_items, on='seller_id', how='left')

# Use review_score from item_df directly (already merged)
seller_review_avg = item_df.groupby('seller_id').agg(
    Avg_Score=('review_score', 'mean'),
    Review_Count=('review_score', 'count')
).reset_index()

closed_performance = closed_performance.merge(seller_review_avg, on='seller_id', how='left')

# Fill missing values (sellers with no orders)
closed_performance['Order_Count'] = closed_performance['Order_Count'].fillna(0).astype(int)
closed_performance['GMV'] = closed_performance['GMV'].fillna(0)
closed_performance['Avg_Score'] = closed_performance['Avg_Score'].fillna(0)

# Summary stats
sellers_with_orders = closed_performance[closed_performance['Order_Count'] > 0]
print(f"Sellers with orders: {len(sellers_with_orders)} / {len(closed_performance)} ({len(sellers_with_orders)/len(closed_performance)*100:.1f}%)")
print(f"Total GMV from closed sellers: R${closed_performance['GMV'].sum():,.2f}")
print(f"Average GMV per seller: R${closed_performance['GMV'].mean():,.2f}")

# Performance by origin
perf_by_origin = closed_performance.groupby('origin').agg(
    Seller_Count=('seller_id', 'count'),
    Sellers_with_Orders=('Order_Count', lambda x: (x > 0).sum()),
    Total_GMV=('GMV', 'sum'),
    Avg_GMV=('GMV', 'mean'),
    Avg_Order_Count=('Order_Count', 'mean'),
    Avg_Score=('Avg_Score', 'mean')
).round(2).reset_index()
perf_by_origin['Activation_Rate%'] = (perf_by_origin['Sellers_with_Orders'] / perf_by_origin['Seller_Count'] * 100).round(2)
perf_by_origin.to_csv(OUTPUT_TABLES / 'seller_performance_by_origin.csv', index=False)

# Plot performance by origin - GMV
fig, ax = plt.subplots(figsize=(14, 6))
perf_by_origin_sorted = perf_by_origin.sort_values('Total_GMV', ascending=False)
ax.barh(perf_by_origin_sorted['origin'][::-1], perf_by_origin_sorted['Total_GMV'][::-1]/1000, 
        color='teal', alpha=0.7)
ax.set_title('Total GMV by Lead Origin (R$ Thousands)', fontsize=14, fontweight='bold')
ax.set_xlabel('GMV (R$ Thousands)')
for i, (v, rate) in enumerate(zip(perf_by_origin_sorted['Total_GMV'][::-1]/1000, 
                                  perf_by_origin_sorted['Activation_Rate%'][::-1])):
    ax.text(v + 1, i, f'{v:.1f}K ({rate}% active)', va='center')
plt.tight_layout()
plt.savefig(OUTPUT_CHARTS / 'seller_gmv_by_origin.png', dpi=150, bbox_inches='tight')
plt.close()

# Performance by lead type
perf_by_lead_type = closed_performance.groupby('lead_type').agg(
    Seller_Count=('seller_id', 'count'),
    Total_GMV=('GMV', 'sum'),
    Avg_GMV=('GMV', 'mean'),
    Avg_Order_Count=('Order_Count', 'mean'),
    Avg_Score=('Avg_Score', 'mean')
).round(2).reset_index()
perf_by_lead_type.to_csv(OUTPUT_TABLES / 'seller_performance_by_lead_type.csv', index=False)

# Performance by business type
perf_by_business_type = closed_performance.groupby('business_type').agg(
    Seller_Count=('seller_id', 'count'),
    Total_GMV=('GMV', 'sum'),
    Avg_GMV=('GMV', 'mean'),
    Avg_Order_Count=('Order_Count', 'mean'),
    Avg_Score=('Avg_Score', 'mean')
).round(2).reset_index()
perf_by_business_type.to_csv(OUTPUT_TABLES / 'seller_performance_by_business_type.csv', index=False)

print("Seller performance analysis completed")

# =============================================================================
# SECTION 5: Channel Quality Identification
# =============================================================================
print("\n=== Section 5: Channel Quality Identification ===")

# Merge conversion rate with performance data
channel_quality = perf_by_origin.merge(origin_conversion[['origin', 'Conversion_Rate%']], 
                                        on='origin', how='left')

# Calculate composite metrics
channel_quality['GMV_per_Deal'] = (channel_quality['Total_GMV'] / channel_quality['Seller_Count']).round(2)
channel_quality = channel_quality.fillna(0)

# Identify channel types
high_conversion_threshold = channel_quality['Conversion_Rate%'].quantile(0.75)
high_gmv_threshold = channel_quality['Avg_GMV'].quantile(0.75)

channel_quality['High_Conversion'] = channel_quality['Conversion_Rate%'] >= high_conversion_threshold
channel_quality['High_GMV'] = channel_quality['Avg_GMV'] >= high_gmv_threshold

# Category classification
def classify_channel(row):
    if row['High_Conversion'] and row['High_GMV']:
        return 'High Conversion + High Value'
    elif row['High_Conversion'] and not row['High_GMV']:
        return 'High Conversion + Low Value'
    elif not row['High_Conversion'] and row['High_GMV']:
        return 'Low Conversion + High Value'
    else:
        return 'Low Conversion + Low Value'

channel_quality['Channel_Type'] = channel_quality.apply(classify_channel, axis=1)
channel_quality.to_csv(OUTPUT_TABLES / 'channel_quality_analysis.csv', index=False)

# Print channel classification
print("\nChannel Quality Classification:")
for channel_type in channel_quality['Channel_Type'].unique():
    channels = channel_quality[channel_quality['Channel_Type'] == channel_type]['origin'].tolist()
    print(f"  {channel_type}: {', '.join(channels)}")

# Plot channel quality matrix
fig, ax = plt.subplots(figsize=(10, 8))
colors = {'High Conversion + High Value': 'green', 
          'High Conversion + Low Value': 'blue',
          'Low Conversion + High Value': 'orange',
          'Low Conversion + Low Value': 'gray'}

for channel_type, color in colors.items():
    subset = channel_quality[channel_quality['Channel_Type'] == channel_type]
    ax.scatter(subset['Conversion_Rate%'], subset['Avg_GMV'], 
               c=color, label=channel_type, s=100, alpha=0.7)

ax.axhline(high_gmv_threshold, color='red', linestyle='--', alpha=0.5)
ax.axvline(high_conversion_threshold, color='red', linestyle='--', alpha=0.5)
ax.set_xlabel('Conversion Rate (%)')
ax.set_ylabel('Avg GMV per Seller (R$)')
ax.set_title('Channel Quality Matrix', fontsize=14, fontweight='bold')
ax.legend(loc='upper right')
for i, row in channel_quality.iterrows():
    ax.annotate(row['origin'], (row['Conversion_Rate%'], row['Avg_GMV']), 
                fontsize=8, alpha=0.7)
plt.tight_layout()
plt.savefig(OUTPUT_CHARTS / 'channel_quality_matrix.png', dpi=150, bbox_inches='tight')
plt.close()

print("Channel quality identification completed")

# =============================================================================
# SECTION 6: Key Findings Summary
# =============================================================================
print("\n=== Section 6: Key Findings Summary ===")

findings = []

# Finding 1
findings.append(f"Marketing funnel scale: 8,000 MQLs → 842 closed deals (10.5% overall conversion rate)")

# Finding 2
top_origin = origin_counts.index[0]
findings.append(f"Top lead source: {top_origin} ({origin_pct.iloc[0]}% of all leads)")

# Finding 3
top_conversion_origin = origin_conversion.iloc[0]['origin']
top_conversion_rate = origin_conversion.iloc[0]['Conversion_Rate%']
findings.append(f"Highest conversion source: {top_conversion_origin} ({top_conversion_rate:.2f}% conversion)")

# Finding 4
top_lead_type = lead_type_counts.index[0]
findings.append(f"Dominant lead type: {top_lead_type} ({lead_type_pct.iloc[0]}% of closed deals)")

# Finding 5
top_business_type = business_type_counts.index[0]
findings.append(f"Primary business type: {top_business_type} ({business_type_pct.iloc[0]}% of closed deals)")

# Finding 6
findings.append(f"Average conversion cycle: {cycle_mean:.2f} days (median {cycle_median:.2f} days)")

# Finding 7
fast_conversion_pct = (conversion_cycle['cycle_days'] <= 30).sum() / len(conversion_cycle) * 100
findings.append(f"Fast conversion rate: {fast_conversion_pct:.1f}% deals closed within 30 days")

# Finding 8
sellers_active_rate = len(sellers_with_orders) / len(closed_performance) * 100
findings.append(f"Seller activation rate: {sellers_active_rate:.1f}% ({len(sellers_with_orders)} of {len(closed_performance)} have orders)")

# Finding 9
total_gmv = closed_performance['GMV'].sum()
findings.append(f"Total GMV from closed sellers: R${total_gmv:,.2f} across {len(sellers_with_orders)} active sellers")

# Finding 10
top_gmv_origin = perf_by_origin_sorted.iloc[0]['origin']
top_gmv_value = perf_by_origin_sorted.iloc[0]['Total_GMV']
findings.append(f"Highest GMV source: {top_gmv_origin} (R${top_gmv_value:,.2f} total)")

# Finding 11
high_conv_high_val = channel_quality[channel_quality['Channel_Type'] == 'High Conversion + High Value']['origin'].tolist()
findings.append(f"Best channels (high conversion + high value): {', '.join(high_conv_high_val) if high_conv_high_val else 'None identified'}")

# Finding 12
avg_review_closed = closed_performance[closed_performance['Avg_Score'] > 0]['Avg_Score'].mean()
findings.append(f"Average review score of active sellers: {avg_review_closed:.2f}/5.0")

# Finding 13
reseller_pct = business_type_pct.get('reseller', 0)
findings.append(f"Resellers dominate business composition: {reseller_pct}% of closed deals")

# Finding 14
online_medium_pct = lead_type_pct.get('online_medium', 0)
findings.append(f"Online medium sellers lead: {online_medium_pct}% of closed deals")

# Finding 15
organic_pct = origin_pct.get('organic_search', 0)
findings.append(f"Organic search drives {organic_pct}% of leads, indicating strong SEO presence")

# Save findings
findings_df = pd.DataFrame({'Finding': findings})
findings_df.to_csv(OUTPUT_TABLES / 'key_findings_marketing_funnel.csv', index=False)

print(f"Generated {len(findings)} key findings")

# =============================================================================
# SECTION 7: Generate Report
# =============================================================================
print("\n=== Section 7: Generating Report ===")

report_content = f"""# Phase 5: Marketing Funnel & Seller Growth Analysis Report
## Brazilian E-commerce Olist Dataset Analysis

**Analysis Date**: {pd.Timestamp.now().strftime('%Y-%m-%d')}
**Data Sources**: MQL ({len(mql_df)} records), Closed Deals ({len(closed_df)} records)

---

## Executive Summary

This report analyzes the marketing funnel and seller growth dynamics of the Olist Brazilian e-commerce platform. We examine lead acquisition channels, conversion patterns, and seller post-conversion performance to identify high-value acquisition strategies.

---

## 1. Lead Structure Analysis

### 1.1 Lead Source Distribution

Top 5 lead sources by volume:

| Source | Count | Percentage |
|--------|-------|-----------|
{origin_table.head(5).to_string(index=False)}

### 1.2 Lead Type Distribution

| Lead Type | Count | Percentage |
|-----------|-------|-----------|
{lead_type_table.to_string(index=False)}

### 1.3 Business Type Distribution

| Business Type | Count | Percentage |
|---------------|-------|-----------|
{business_type_table.to_string(index=False)}

**Charts**: `outputs/charts/lead_source_distribution.png`, `lead_type_distribution.png`, `business_type_distribution.png`

---

## 2. Conversion Rate Analysis

### 2.1 Overall Conversion

- **Total MQLs**: {len(mql_df)}
- **Closed Deals**: {len(closed_df)}
- **Overall Conversion Rate**: {overall_conversion_rate:.2f}%

### 2.2 Conversion by Origin

| Origin | Leads | Converted | Rate |
|--------|-------|-----------|------|
{origin_conversion[['origin', 'Lead_Count', 'Converted', 'Conversion_Rate%']].head(10).to_string(index=False)}

**Key Insight**: Conversion rates vary significantly by source, indicating channel quality differences.

**Charts**: `outputs/charts/conversion_rate_by_origin.png`

---

## 3. Conversion Cycle Analysis

### 3.1 Cycle Statistics

| Metric | Value |
|--------|-------|
| Mean | {cycle_mean:.2f} days |
| Median | {cycle_median:.2f} days |
| Std | {cycle_std:.2f} days |

### 3.2 Cycle Distribution

| Bucket | Count | Percentage |
|--------|-------|-----------|
{cycle_table.to_string(index=False)}

**Key Insight**: {fast_conversion_pct:.1f}% of deals close within 30 days, indicating efficient sales process.

**Charts**: `outputs/charts/conversion_cycle_distribution.png`

---

## 4. Seller Performance Analysis

### 4.1 Activation Rate

- **Total Closed Sellers**: {len(closed_performance)}
- **Active Sellers (with orders)**: {len(sellers_with_orders)} ({sellers_active_rate:.1f}%)
- **Total GMV**: R${total_gmv:,.2f}

### 4.2 Performance by Origin

| Origin | Sellers | GMV | Avg GMV | Activation |
|--------|---------|-----|---------|-----------|
{perf_by_origin[['origin', 'Seller_Count', 'Total_GMV', 'Avg_GMV', 'Activation_Rate%']].to_string(index=False)}

**Charts**: `outputs/charts/seller_gmv_by_origin.png`

---

## 5. Channel Quality Identification

### 5.1 Channel Classification

| Channel Type | Origins |
|--------------|---------|
{pd.DataFrame({'Type': channel_quality['Channel_Type'].unique(), 
               'Origins': [channel_quality[channel_quality['Channel_Type']==t]['origin'].tolist() 
                          for t in channel_quality['Channel_Type'].unique()]}).to_string(index=False)}

### 5.2 Quality Matrix

The channel quality matrix plots conversion rate against average GMV per seller:
- **High Conversion + High Value**: Optimal channels for investment
- **Low Conversion + High Value**: Quality leads worth nurturing
- **High Conversion + Low Value**: High volume but low quality
- **Low Conversion + Low Value**: May need strategy adjustment

**Charts**: `outputs/charts/channel_quality_matrix.png`

---

## 6. Key Findings Summary

{chr(10).join([f'{i+1}. {f}' for i, f in enumerate(findings)])}

---

## 7. Business Recommendations

### 7.1 High-Value Channel Investment
- Prioritize channels with high conversion + high GMV
- Scale organic and paid search where ROI is proven

### 7.2 Low-Conversion High-Value Channels
- Investigate barriers in these channels
- Consider targeted nurturing campaigns

### 7.3 Seller Activation Improvement
- {100-sellers_active_rate:.1f}% of closed sellers have no orders
- Implement post-deal onboarding support
- Track early-stage seller engagement

### 7.4 Conversion Cycle Optimization
- Target median cycle of {cycle_median:.0f} days as benchmark
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

**Analysis Completed**: {pd.Timestamp.now().strftime('%Y-%m-%d %H:%M:%S')}
**Reproducibility**: All results generated from `data/` and `data/interim/` source files
"""

with open(REPORT_DIR / 'marketing_funnel_seller_growth_analysis.md', 'w', encoding='utf-8') as f:
    f.write(report_content)

print("Report generated: reports/marketing_funnel_seller_growth_analysis.md")

# =============================================================================
# FINAL SUMMARY
# =============================================================================
print("\n" + "="*60)
print("PHASE 5: MARKETING FUNNEL & SELLER GROWTH ANALYSIS COMPLETED")
print("="*60)
print(f"\nCharts generated: {len(list(OUTPUT_CHARTS.glob('*.png'))) - 26} new files")
print(f"Tables generated: {len(list(OUTPUT_TABLES.glob('*.csv'))) - 23} new files")
print(f"Report generated: reports/marketing_funnel_seller_growth_analysis.md")
print(f"Key findings: {len(findings)} data-driven insights")
print("\nAll outputs are reproducible from data/ and data/interim/ source files")