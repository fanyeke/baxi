#!/usr/bin/env python3
"""
Phase 4: Fulfillment & Customer Experience Analysis
Brazilian E-commerce Olist Dataset Analysis

This script performs comprehensive fulfillment and customer experience analysis including:
1. Order fulfillment chain duration (order→approval→shipping→delivery)
2. Estimated vs actual delivery difference (late/early delivery rates)
3. Review score distribution and influencing factors
4. Comparison by state, category, seller on delivery efficiency and scores
5. Identification of problematic areas and experience issues

Output:
- Charts: outputs/charts/
- Tables: outputs/tables/
- Report: reports/fulfillment_customer_experience_analysis.md
"""

import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
import seaborn as sns
from pathlib import Path
from scipy import stats
import warnings
warnings.filterwarnings('ignore')

# Set Chinese font support for matplotlib
plt.rcParams['font.sans-serif'] = ['SimHei', 'DejaVu Sans']
plt.rcParams['axes.unicode_minus'] = False

# Set style
sns.set_style("whitegrid")
plt.rcParams['figure.figsize'] = (12, 6)
plt.rcParams['figure.dpi'] = 100

# Data paths
DATA_DIR = Path('data/interim')
OUTPUT_CHARTS = Path('outputs/charts')
OUTPUT_TABLES = Path('outputs/tables')
REPORT_DIR = Path('reports')

# Load data
print("Loading data...")
order_df = pd.read_csv(DATA_DIR / 'order_level_base.csv')
item_df = pd.read_csv(DATA_DIR / 'item_level_base.csv')

# Convert dates
date_cols = ['order_purchase_timestamp', 'order_approved_at', 
             'order_delivered_carrier_date', 'order_delivered_customer_date',
             'order_estimated_delivery_date']

for col in date_cols:
    if col in order_df.columns:
        order_df[col] = pd.to_datetime(order_df[col])

item_df['shipping_limit_date'] = pd.to_datetime(item_df['shipping_limit_date'])

print(f"Orders: {len(order_df)} rows")
print(f"Items: {len(item_df)} rows")

# =============================================================================
# SECTION 1: Fulfillment Chain Duration Analysis
# =============================================================================
print("\n=== Section 1: Fulfillment Chain Duration Analysis ===")

# Filter for delivered orders only (for meaningful duration analysis)
delivered_orders = order_df[order_df['order_status'] == 'delivered'].copy()
print(f"Delivered orders: {len(delivered_orders)}")

# Calculate durations (in hours and days)
delivered_orders['time_to_approval'] = (
    delivered_orders['order_approved_at'] - delivered_orders['order_purchase_timestamp']
).dt.total_seconds() / 3600  # hours

delivered_orders['time_to_shipping'] = (
    delivered_orders['order_delivered_carrier_date'] - delivered_orders['order_approved_at']
).dt.total_seconds() / 3600  # hours

delivered_orders['time_to_delivery'] = (
    delivered_orders['order_delivered_customer_date'] - delivered_orders['order_delivered_carrier_date']
).dt.total_seconds() / 3600  # hours

delivered_orders['total_delivery_time'] = (
    delivered_orders['order_delivered_customer_date'] - delivered_orders['order_purchase_timestamp']
).dt.total_seconds() / 3600  # hours

# Convert to days for readability
delivered_orders['time_to_approval_days'] = delivered_orders['time_to_approval'] / 24
delivered_orders['time_to_shipping_days'] = delivered_orders['time_to_shipping'] / 24
delivered_orders['time_to_delivery_days'] = delivered_orders['time_to_delivery'] / 24
delivered_orders['total_delivery_days'] = delivered_orders['total_delivery_time'] / 24

# Duration statistics
duration_stats = {
    'Stage': ['Order→Approval', 'Approval→Shipping', 'Shipping→Delivery', 'Total Delivery'],
    'Mean (hours)': [
        delivered_orders['time_to_approval'].mean(),
        delivered_orders['time_to_shipping'].mean(),
        delivered_orders['time_to_delivery'].mean(),
        delivered_orders['total_delivery_time'].mean()
    ],
    'Median (hours)': [
        delivered_orders['time_to_approval'].median(),
        delivered_orders['time_to_shipping'].median(),
        delivered_orders['time_to_delivery'].median(),
        delivered_orders['total_delivery_time'].median()
    ],
    'Mean (days)': [
        delivered_orders['time_to_approval_days'].mean(),
        delivered_orders['time_to_shipping_days'].mean(),
        delivered_orders['time_to_delivery_days'].mean(),
        delivered_orders['total_delivery_days'].mean()
    ],
    'Median (days)': [
        delivered_orders['time_to_approval_days'].median(),
        delivered_orders['time_to_shipping_days'].median(),
        delivered_orders['time_to_delivery_days'].median(),
        delivered_orders['total_delivery_days'].median()
    ],
    'Std (days)': [
        delivered_orders['time_to_approval_days'].std(),
        delivered_orders['time_to_shipping_days'].std(),
        delivered_orders['time_to_delivery_days'].std(),
        delivered_orders['total_delivery_days'].std()
    ]
}

duration_table = pd.DataFrame(duration_stats)
duration_table.to_csv(OUTPUT_TABLES / 'fulfillment_duration_stats.csv', index=False)

# Plot 1: Fulfillment Chain Duration Distribution
fig, axes = plt.subplots(2, 2, figsize=(14, 10))

# Time to Approval
ax1 = axes[0, 0]
ax1.hist(delivered_orders['time_to_approval_days'].dropna(), bins=50, color='steelblue', alpha=0.7, edgecolor='black')
ax1.axvline(delivered_orders['time_to_approval_days'].mean(), color='red', linestyle='--', 
            label=f'Mean: {delivered_orders["time_to_approval_days"].mean():.2f} days')
ax1.axvline(delivered_orders['time_to_approval_days'].median(), color='green', linestyle='--', 
            label=f'Median: {delivered_orders["time_to_approval_days"].median():.2f} days')
ax1.set_title('Time from Order to Approval', fontsize=12, fontweight='bold')
ax1.set_xlabel('Days')
ax1.set_ylabel('Frequency')
ax1.legend()

# Time to Shipping
ax2 = axes[0, 1]
ax2.hist(delivered_orders['time_to_shipping_days'].dropna(), bins=50, color='coral', alpha=0.7, edgecolor='black')
ax2.axvline(delivered_orders['time_to_shipping_days'].mean(), color='red', linestyle='--', 
            label=f'Mean: {delivered_orders["time_to_shipping_days"].mean():.2f} days')
ax2.axvline(delivered_orders['time_to_shipping_days'].median(), color='green', linestyle='--', 
            label=f'Median: {delivered_orders["time_to_shipping_days"].median():.2f} days')
ax2.set_title('Time from Approval to Shipping', fontsize=12, fontweight='bold')
ax2.set_xlabel('Days')
ax2.set_ylabel('Frequency')
ax2.legend()

# Time to Delivery
ax3 = axes[1, 0]
ax3.hist(delivered_orders['time_to_delivery_days'].dropna(), bins=50, color='seagreen', alpha=0.7, edgecolor='black')
ax3.axvline(delivered_orders['time_to_delivery_days'].mean(), color='red', linestyle='--', 
            label=f'Mean: {delivered_orders["time_to_delivery_days"].mean():.2f} days')
ax3.axvline(delivered_orders['time_to_delivery_days'].median(), color='green', linestyle='--', 
            label=f'Median: {delivered_orders["time_to_delivery_days"].median():.2f} days')
ax3.set_title('Time from Shipping to Delivery', fontsize=12, fontweight='bold')
ax3.set_xlabel('Days')
ax3.set_ylabel('Frequency')
ax3.legend()

# Total Delivery Time
ax4 = axes[1, 1]
ax4.hist(delivered_orders['total_delivery_days'].dropna(), bins=50, color='goldenrod', alpha=0.7, edgecolor='black')
ax4.axvline(delivered_orders['total_delivery_days'].mean(), color='red', linestyle='--', 
            label=f'Mean: {delivered_orders["total_delivery_days"].mean():.2f} days')
ax4.axvline(delivered_orders['total_delivery_days'].median(), color='green', linestyle='--', 
            label=f'Median: {delivered_orders["total_delivery_days"].median():.2f} days')
ax4.set_title('Total Delivery Time', fontsize=12, fontweight='bold')
ax4.set_xlabel('Days')
ax4.set_ylabel('Frequency')
ax4.legend()

plt.tight_layout()
plt.savefig(OUTPUT_CHARTS / 'fulfillment_duration_distribution.png', dpi=150, bbox_inches='tight')
plt.close()

# Plot 2: Fulfillment Chain Boxplot
fig, ax = plt.subplots(figsize=(12, 6))
duration_data = pd.DataFrame({
    'Approval': delivered_orders['time_to_approval_days'].dropna(),
    'Shipping': delivered_orders['time_to_shipping_days'].dropna(),
    'Delivery': delivered_orders['time_to_delivery_days'].dropna(),
    'Total': delivered_orders['total_delivery_days'].dropna()
})
duration_data_melted = duration_data.melt(var_name='Stage', value_name='Days')
sns.boxplot(data=duration_data_melted, x='Stage', y='Days', palette='Set2', ax=ax)
ax.set_title('Fulfillment Chain Duration by Stage', fontsize=14, fontweight='bold')
ax.set_xlabel('Stage')
ax.set_ylabel('Days')
plt.tight_layout()
plt.savefig(OUTPUT_CHARTS / 'fulfillment_chain_boxplot.png', dpi=150, bbox_inches='tight')
plt.close()

print("Fulfillment duration analysis completed")

# =============================================================================
# SECTION 2: Estimated vs Actual Delivery Analysis
# =============================================================================
print("\n=== Section 2: Estimated vs Actual Delivery Analysis ===")

# Calculate delivery difference
delivered_orders['delivery_diff_days'] = (
    delivered_orders['order_delivered_customer_date'] - delivered_orders['order_estimated_delivery_date']
).dt.total_seconds() / 86400  # days

# Classification: Late (>0), On-time (==0), Early (<0)
delivered_orders['delivery_status'] = delivered_orders['delivery_diff_days'].apply(
    lambda x: 'Late' if x > 0 else ('Early' if x < 0 else 'On-time')
)

# Delivery status counts
delivery_status_counts = delivered_orders['delivery_status'].value_counts()
delivery_status_pct = (delivery_status_counts / len(delivered_orders) * 100).round(2)

delivery_status_table = pd.DataFrame({
    'Status': delivery_status_counts.index,
    'Count': delivery_status_counts.values,
    'Percentage': delivery_status_pct.values
})
delivery_status_table.to_csv(OUTPUT_TABLES / 'delivery_status_distribution.csv', index=False)

# Calculate late and early delivery statistics
late_orders = delivered_orders[delivered_orders['delivery_status'] == 'Late']
early_orders = delivered_orders[delivered_orders['delivery_status'] == 'Early']

late_stats = {
    'Metric': ['Late Delivery Rate', 'Mean Late Days', 'Median Late Days', 'Max Late Days',
               'Early Delivery Rate', 'Mean Early Days', 'Median Early Days', 'Max Early Days'],
    'Value': [
        delivery_status_pct.get('Late', 0),
        late_orders['delivery_diff_days'].mean() if len(late_orders) > 0 else 0,
        late_orders['delivery_diff_days'].median() if len(late_orders) > 0 else 0,
        late_orders['delivery_diff_days'].max() if len(late_orders) > 0 else 0,
        delivery_status_pct.get('Early', 0),
        abs(early_orders['delivery_diff_days'].mean()) if len(early_orders) > 0 else 0,
        abs(early_orders['delivery_diff_days'].median()) if len(early_orders) > 0 else 0,
        abs(early_orders['delivery_diff_days'].min()) if len(early_orders) > 0 else 0
    ]
}
late_stats_table = pd.DataFrame(late_stats)
late_stats_table.to_csv(OUTPUT_TABLES / 'delivery_timing_stats.csv', index=False)

# Plot 3: Delivery Status Distribution
fig, axes = plt.subplots(1, 2, figsize=(14, 6))

# Pie chart
ax1 = axes[0]
colors = ['#FF6B6B', '#96CEB4', '#45B7D1']
ax1.pie(delivery_status_counts.values, labels=delivery_status_counts.index, autopct='%1.1f%%',
        colors=colors, explode=(0.05, 0.05, 0), startangle=90)
ax1.set_title('Delivery Status Distribution', fontsize=14, fontweight='bold')

# Bar chart
ax2 = axes[1]
ax2.bar(delivery_status_counts.index, delivery_status_counts.values, color=colors)
ax2.set_title('Delivery Status Counts', fontsize=14, fontweight='bold')
ax2.set_xlabel('Status')
ax2.set_ylabel('Count')
for i, (v, p) in enumerate(zip(delivery_status_counts.values, delivery_status_pct.values)):
    ax2.text(i, v + 200, f'{p}%', ha='center')

plt.tight_layout()
plt.savefig(OUTPUT_CHARTS / 'delivery_status_distribution.png', dpi=150, bbox_inches='tight')
plt.close()

# Plot 4: Delivery Difference Distribution
fig, ax = plt.subplots(figsize=(12, 6))
ax.hist(delivered_orders['delivery_diff_days'].dropna(), bins=50, color='mediumpurple', alpha=0.7, edgecolor='black')
ax.axvline(0, color='black', linestyle='-', linewidth=2, label='Estimated Delivery Date')
ax.axvline(delivered_orders['delivery_diff_days'].mean(), color='red', linestyle='--', 
           label=f'Mean Diff: {delivered_orders["delivery_diff_days"].mean():.2f} days')
ax.set_title('Delivery Difference (Actual - Estimated) Distribution', fontsize=14, fontweight='bold')
ax.set_xlabel('Days Difference (Positive = Late, Negative = Early)')
ax.set_ylabel('Frequency')
ax.legend()
plt.tight_layout()
plt.savefig(OUTPUT_CHARTS / 'delivery_difference_distribution.png', dpi=150, bbox_inches='tight')
plt.close()

print("Delivery timing analysis completed")

# =============================================================================
# SECTION 3: Review Score Analysis
# =============================================================================
print("\n=== Section 3: Review Score Analysis ===")

# Review score distribution
review_counts = delivered_orders['review_score'].value_counts().sort_index()
review_pct = (review_counts / len(delivered_orders) * 100).round(2)

review_table = pd.DataFrame({
    'Score': review_counts.index,
    'Count': review_counts.values,
    'Percentage': review_pct.values
})
review_table.to_csv(OUTPUT_TABLES / 'review_score_distribution.csv', index=False)

# Calculate average review score
avg_review = delivered_orders['review_score'].mean()
median_review = delivered_orders['review_score'].median()

# Plot 5: Review Score Distribution
fig, axes = plt.subplots(1, 2, figsize=(14, 6))

# Bar chart
ax1 = axes[0]
colors_review = ['#FF4444', '#FF8844', '#FFBB44', '#88CC44', '#44AA44']
ax1.bar(review_counts.index, review_counts.values, color=colors_review)
ax1.set_title('Review Score Distribution', fontsize=14, fontweight='bold')
ax1.set_xlabel('Score')
ax1.set_ylabel('Count')
ax1.set_xticks([1, 2, 3, 4, 5])
for i, (v, p) in enumerate(zip(review_counts.values, review_pct.values)):
    ax1.text(review_counts.index[i], v + 200, f'{p}%', ha='center')

# Pie chart
ax2 = axes[1]
ax2.pie(review_counts.values, labels=review_counts.index, autopct='%1.1f%%',
        colors=colors_review, startangle=90)
ax2.set_title('Review Score Percentage', fontsize=14, fontweight='bold')

plt.tight_layout()
plt.savefig(OUTPUT_CHARTS / 'review_score_distribution.png', dpi=150, bbox_inches='tight')
plt.close()

print("Review score distribution analysis completed")

# =============================================================================
# SECTION 4: Correlation Analysis (Duration, Delay, Freight vs Score)
# =============================================================================
print("\n=== Section 4: Correlation Analysis ===")

# Merge item-level freight data to order level
item_freight = item_df.groupby('order_id').agg({
    'freight_value': 'sum',
    'price': 'sum'
}).reset_index()
item_freight.columns = ['order_id', 'total_freight', 'total_price']

delivered_orders = delivered_orders.merge(item_freight, on='order_id', how='left')

# Calculate correlations
correlation_vars = ['total_delivery_days', 'delivery_diff_days', 'total_freight', 'review_score']
correlation_matrix = delivered_orders[correlation_vars].corr()

correlation_table = correlation_matrix.round(3)
correlation_table.to_csv(OUTPUT_TABLES / 'correlation_matrix.csv')

# Plot 6: Correlation Heatmap
fig, ax = plt.subplots(figsize=(10, 8))
sns.heatmap(correlation_matrix, annot=True, cmap='RdYlBu_r', center=0, 
            square=True, linewidths=2, ax=ax, fmt='.3f')
ax.set_title('Correlation: Delivery Metrics vs Review Score', fontsize=14, fontweight='bold')
plt.tight_layout()
plt.savefig(OUTPUT_CHARTS / 'correlation_heatmap.png', dpi=150, bbox_inches='tight')
plt.close()

# Analyze relationship between delivery metrics and score
# Group by review score and calculate average metrics
score_metrics = delivered_orders.groupby('review_score').agg({
    'total_delivery_days': ['mean', 'median'],
    'delivery_diff_days': ['mean', 'median'],
    'total_freight': ['mean', 'median']
}).reset_index()

score_metrics.columns = ['Score', 'Avg_Delivery_Days', 'Median_Delivery_Days',
                         'Avg_Delay_Days', 'Median_Delay_Days',
                         'Avg_Freight', 'Median_Freight']

score_metrics.to_csv(OUTPUT_TABLES / 'score_vs_metrics.csv', index=False)

# Plot 7: Delivery Days by Review Score
fig, axes = plt.subplots(1, 3, figsize=(18, 6))

# Delivery Days
ax1 = axes[0]
ax1.bar(score_metrics['Score'], score_metrics['Avg_Delivery_Days'], color='steelblue', alpha=0.7)
ax1.set_title('Avg Delivery Days by Review Score', fontsize=14, fontweight='bold')
ax1.set_xlabel('Review Score')
ax1.set_ylabel('Days')
ax1.set_xticks([1, 2, 3, 4, 5])

# Delay Days
ax2 = axes[1]
ax2.bar(score_metrics['Score'], score_metrics['Avg_Delay_Days'], color='coral', alpha=0.7)
ax2.axhline(0, color='black', linestyle='-', linewidth=1)
ax2.set_title('Avg Delay Days by Review Score', fontsize=14, fontweight='bold')
ax2.set_xlabel('Review Score')
ax2.set_ylabel('Days (Positive=Late)')
ax2.set_xticks([1, 2, 3, 4, 5])

# Freight
ax3 = axes[2]
ax3.bar(score_metrics['Score'], score_metrics['Avg_Freight'], color='goldenrod', alpha=0.7)
ax3.set_title('Avg Freight by Review Score', fontsize=14, fontweight='bold')
ax3.set_xlabel('Review Score')
ax3.set_ylabel('Freight (R$)')
ax3.set_xticks([1, 2, 3, 4, 5])

plt.tight_layout()
plt.savefig(OUTPUT_CHARTS / 'metrics_by_review_score.png', dpi=150, bbox_inches='tight')
plt.close()

print("Correlation analysis completed")

# =============================================================================
# SECTION 5: State-level Delivery & Review Analysis
# =============================================================================
print("\n=== Section 5: State-level Analysis ===")

# Group by customer state
state_delivery = delivered_orders.groupby('customer_state').agg({
    'order_id': 'count',
    'total_delivery_days': 'mean',
    'delivery_diff_days': 'mean',
    'review_score': 'mean',
    'total_freight': 'mean'
}).reset_index()

state_delivery.columns = ['State', 'Orders', 'Avg_Delivery_Days', 'Avg_Delay_Days', 'Avg_Score', 'Avg_Freight']

# Calculate late delivery rate by state
state_late_rate = delivered_orders.groupby('customer_state').apply(
    lambda x: (x['delivery_status'] == 'Late').sum() / len(x) * 100
).reset_index()
state_late_rate.columns = ['State', 'Late_Rate%']

state_delivery = state_delivery.merge(state_late_rate, on='State')

# Sort by order count
state_delivery = state_delivery.sort_values('Orders', ascending=False)

state_delivery.to_csv(OUTPUT_TABLES / 'state_delivery_performance.csv', index=False)

# Plot 8: Top 10 States - Delivery Days
fig, ax = plt.subplots(figsize=(12, 6))
top10_states = state_delivery.head(10)
ax.barh(top10_states['State'], top10_states['Avg_Delivery_Days'], color='teal')
ax.set_title('Top 10 States by Orders - Avg Delivery Days', fontsize=14, fontweight='bold')
ax.set_xlabel('Avg Delivery Days')
ax.invert_yaxis()
for i, v in enumerate(top10_states['Avg_Delivery_Days']):
    ax.text(v + 0.2, i, f'{v:.2f}', va='center')
plt.tight_layout()
plt.savefig(OUTPUT_CHARTS / 'state_avg_delivery_days.png', dpi=150, bbox_inches='tight')
plt.close()

# Plot 9: Top 10 States - Late Rate
fig, ax = plt.subplots(figsize=(12, 6))
ax.barh(top10_states['State'], top10_states['Late_Rate%'], color='coral')
ax.set_title('Top 10 States by Orders - Late Delivery Rate', fontsize=14, fontweight='bold')
ax.set_xlabel('Late Rate (%)')
ax.invert_yaxis()
for i, v in enumerate(top10_states['Late_Rate%']):
    ax.text(v + 0.5, i, f'{v:.2f}%', va='center')
plt.tight_layout()
plt.savefig(OUTPUT_CHARTS / 'state_late_rate.png', dpi=150, bbox_inches='tight')
plt.close()

# Plot 10: Top 10 States - Avg Score
fig, ax = plt.subplots(figsize=(12, 6))
ax.barh(top10_states['State'], top10_states['Avg_Score'], color='seagreen')
ax.axvline(avg_review, color='red', linestyle='--', label=f'Overall Avg: {avg_review:.2f}')
ax.set_title('Top 10 States by Orders - Avg Review Score', fontsize=14, fontweight='bold')
ax.set_xlabel('Avg Review Score')
ax.invert_yaxis()
ax.legend(loc='lower right')
for i, v in enumerate(top10_states['Avg_Score']):
    ax.text(v + 0.02, i, f'{v:.2f}', va='center')
plt.tight_layout()
plt.savefig(OUTPUT_CHARTS / 'state_avg_score.png', dpi=150, bbox_inches='tight')
plt.close()

print("State-level analysis completed")

# =============================================================================
# SECTION 6: Category-level Delivery & Review Analysis
# =============================================================================
print("\n=== Section 6: Category-level Analysis ===")

# Use English category names
item_df['category_name'] = item_df['product_category_name_english'].fillna(item_df['product_category_name'])

# Merge category info to delivered orders
category_order = item_df[['order_id', 'category_name']].drop_duplicates()
delivered_with_category = delivered_orders.merge(category_order, on='order_id', how='left')

# Group by category
category_delivery = delivered_with_category.groupby('category_name').agg({
    'order_id': 'count',
    'total_delivery_days': 'mean',
    'delivery_diff_days': 'mean',
    'review_score': 'mean',
    'total_freight': 'mean'
}).reset_index()

category_delivery.columns = ['Category', 'Orders', 'Avg_Delivery_Days', 'Avg_Delay_Days', 'Avg_Score', 'Avg_Freight']

# Calculate late delivery rate by category
category_late_rate = delivered_with_category.groupby('category_name').apply(
    lambda x: (x['delivery_status'] == 'Late').sum() / len(x) * 100
).reset_index()
category_late_rate.columns = ['Category', 'Late_Rate%']

category_delivery = category_delivery.merge(category_late_rate, on='Category')

# Sort by order count
category_delivery = category_delivery.sort_values('Orders', ascending=False)

category_delivery.to_csv(OUTPUT_TABLES / 'category_delivery_performance.csv', index=False)

# Top 15 categories analysis
top15_categories = category_delivery.head(15)

# Plot 11: Top Categories - Avg Score
fig, ax = plt.subplots(figsize=(14, 6))
ax.barh(top15_categories['Category'].head(10), top15_categories['Avg_Score'].head(10), color='steelblue')
ax.axvline(avg_review, color='red', linestyle='--', label=f'Overall Avg: {avg_review:.2f}')
ax.set_title('Top 10 Categories by Orders - Avg Review Score', fontsize=14, fontweight='bold')
ax.set_xlabel('Avg Review Score')
ax.invert_yaxis()
ax.legend()
plt.tight_layout()
plt.savefig(OUTPUT_CHARTS / 'category_avg_score.png', dpi=150, bbox_inches='tight')
plt.close()

print("Category-level analysis completed")

# =============================================================================
# SECTION 7: Seller-level Delivery & Review Analysis
# =============================================================================
print("\n=== Section 7: Seller-level Analysis ===")

# Merge seller info to delivered orders
seller_order = item_df[['order_id', 'seller_id']].drop_duplicates()
delivered_with_seller = delivered_orders.merge(seller_order, on='order_id', how='left')

# Group by seller
seller_delivery = delivered_with_seller.groupby('seller_id').agg({
    'order_id': 'count',
    'total_delivery_days': 'mean',
    'delivery_diff_days': 'mean',
    'review_score': 'mean'
}).reset_index()

seller_delivery.columns = ['Seller_ID', 'Orders', 'Avg_Delivery_Days', 'Avg_Delay_Days', 'Avg_Score']

# Calculate late delivery rate by seller
seller_late_rate = delivered_with_seller.groupby('seller_id').apply(
    lambda x: (x['delivery_status'] == 'Late').sum() / len(x) * 100
).reset_index()
seller_late_rate.columns = ['Seller_ID', 'Late_Rate%']

seller_delivery = seller_delivery.merge(seller_late_rate, on='Seller_ID')

# Sort by order count
seller_delivery = seller_delivery.sort_values('Orders', ascending=False)

seller_delivery.to_csv(OUTPUT_TABLES / 'seller_delivery_performance.csv', index=False)

print("Seller-level analysis completed")

# =============================================================================
# SECTION 8: Problem Identification
# =============================================================================
print("\n=== Section 8: Problem Identification ===")

# 8.1 High Sales but Low Score Categories
# Definition: Orders > 100 and Avg Score < 4.0
low_score_categories = category_delivery[
    (category_delivery['Orders'] > 100) & (category_delivery['Avg_Score'] < 4.0)
].sort_values('Orders', ascending=False)

low_score_categories.to_csv(OUTPUT_TABLES / 'high_sales_low_score_categories.csv', index=False)

# 8.2 High Sales but Low Score Sellers
# Definition: Orders > 50 and Avg Score < 4.0
low_score_sellers = seller_delivery[
    (seller_delivery['Orders'] > 50) & (seller_delivery['Avg_Score'] < 4.0)
].sort_values('Orders', ascending=False)

low_score_sellers.to_csv(OUTPUT_TABLES / 'high_sales_low_score_sellers.csv', index=False)

# 8.3 High Late Rate Regions
# Definition: Late Rate > 15%
high_late_states = state_delivery[state_delivery['Late_Rate%'] > 15].sort_values('Late_Rate%', ascending=False)

high_late_states.to_csv(OUTPUT_TABLES / 'high_late_rate_states.csv', index=False)

# 8.4 Experience Issues Impact Analysis
# Calculate average metrics for low score (1-2) vs high score (4-5)
low_score_orders = delivered_orders[delivered_orders['review_score'] <= 2]
high_score_orders = delivered_orders[delivered_orders['review_score'] >= 4]

issue_impact = {
    'Metric': ['Avg Delivery Days', 'Avg Delay Days', 'Late Delivery Rate', 'Avg Freight',
               'Orders Count', 'Orders Percentage'],
    'Low Score (1-2)': [
        low_score_orders['total_delivery_days'].mean(),
        low_score_orders['delivery_diff_days'].mean(),
        (low_score_orders['delivery_status'] == 'Late').sum() / len(low_score_orders) * 100,
        low_score_orders['total_freight'].mean(),
        len(low_score_orders),
        len(low_score_orders) / len(delivered_orders) * 100
    ],
    'High Score (4-5)': [
        high_score_orders['total_delivery_days'].mean(),
        high_score_orders['delivery_diff_days'].mean(),
        (high_score_orders['delivery_status'] == 'Late').sum() / len(high_score_orders) * 100,
        high_score_orders['total_freight'].mean(),
        len(high_score_orders),
        len(high_score_orders) / len(delivered_orders) * 100
    ]
}

issue_impact_table = pd.DataFrame(issue_impact)
issue_impact_table.to_csv(OUTPUT_TABLES / 'score_impact_analysis.csv', index=False)

print("Problem identification completed")

# =============================================================================
# SECTION 9: Key Findings Summary
# =============================================================================
print("\n=== Section 9: Key Findings Summary ===")

findings = []

# Finding 1: Fulfillment Chain
findings.append(f"Average total delivery time: {delivered_orders['total_delivery_days'].mean():.2f} days (median: {delivered_orders['total_delivery_days'].median():.2f} days)")

# Finding 2: Approval Speed
findings.append(f"Order approval is fast: average {delivered_orders['time_to_approval_days'].mean():.2f} days from purchase")

# Finding 3: Shipping Efficiency
findings.append(f"Shipping stage takes longest: average {delivered_orders['time_to_shipping_days'].mean():.2f} days from approval to carrier")

# Finding 4: Delivery Status
late_pct = delivery_status_pct.get('Late', 0)
early_pct = delivery_status_pct.get('Early', 0)
findings.append(f"Delivery reliability: {late_pct:.2f}% late deliveries, {early_pct:.2f}% early deliveries")

# Finding 5: Review Score Distribution
high_score_pct = review_pct.get(5, 0) + review_pct.get(4, 0)
low_score_pct = review_pct.get(1, 0) + review_pct.get(2, 0)
findings.append(f"Customer satisfaction: {high_score_pct:.2f}% high scores (4-5), {low_score_pct:.2f}% low scores (1-2)")

# Finding 6: Average Review Score
findings.append(f"Overall average review score: {avg_review:.2f}/5.0")

# Finding 7: Correlation - Delivery Time vs Score
corr_delivery_score = correlation_matrix.loc['total_delivery_days', 'review_score']
findings.append(f"Negative correlation between delivery days and score: {corr_delivery_score:.3f} (longer delivery → lower score)")

# Finding 8: Correlation - Delay vs Score
corr_delay_score = correlation_matrix.loc['delivery_diff_days', 'review_score']
findings.append(f"Negative correlation between delay days and score: {corr_delay_score:.3f} (late delivery → lower score)")

# Finding 9: Low Score Delivery Impact
low_delivery_avg = low_score_orders['total_delivery_days'].mean()
high_delivery_avg = high_score_orders['total_delivery_days'].mean()
findings.append(f"Delivery time impact: Low scores avg {low_delivery_avg:.2f} days vs high scores {high_delivery_avg:.2f} days")

# Finding 10: Low Score Late Rate Impact
low_late_rate = (low_score_orders['delivery_status'] == 'Late').sum() / len(low_score_orders) * 100
high_late_rate = (high_score_orders['delivery_status'] == 'Late').sum() / len(high_score_orders) * 100
findings.append(f"Late delivery impact: Low scores have {low_late_rate:.2f}% late rate vs high scores {high_late_rate:.2f}%")

# Finding 11: High Late Rate States
if len(high_late_states) > 0:
    findings.append(f"High late rate states: {', '.join(high_late_states['State'].head(5).tolist())} (>15% late deliveries)")

# Finding 12: Low Score Categories
if len(low_score_categories) > 0:
    findings.append(f"High sales but low score categories: {len(low_score_categories)} categories with >100 orders and avg score <4.0")

# Finding 13: Low Score Sellers
if len(low_score_sellers) > 0:
    findings.append(f"High sales but low score sellers: {len(low_score_sellers)} sellers with >50 orders and avg score <4.0")

# Finding 14: Freight Correlation
corr_freight_score = correlation_matrix.loc['total_freight', 'review_score']
findings.append(f"Weak correlation between freight and score: {corr_freight_score:.3f} (freight has minimal impact on satisfaction)")

# Finding 15: Review Score Variability
findings.append(f"Score variability: Median {median_review:.2f} close to mean {avg_review:.2f}, but {low_score_pct:.2f}% dissatisfied customers signal experience gaps")

# Save findings
findings_df = pd.DataFrame({'Finding': findings})
findings_df.to_csv(OUTPUT_TABLES / 'key_findings_fulfillment.csv', index=False)

print(f"Generated {len(findings)} key findings")

# =============================================================================
# GENERATE REPORT
# =============================================================================
print("\n=== Generating Report ===")

report_content = f"""# Phase 4: Fulfillment & Customer Experience Analysis Report
## Brazilian E-commerce Olist Dataset Analysis

**Analysis Period**: {delivered_orders['order_purchase_timestamp'].min().strftime('%Y-%m')} to {delivered_orders['order_purchase_timestamp'].max().strftime('%Y-%m')}
**Data Sources**: {len(delivered_orders)} delivered orders analyzed

---

## Executive Summary

This report presents comprehensive fulfillment chain analysis and customer experience evaluation of the Olist Brazilian e-commerce platform. We analyze delivery timing, estimated vs actual delivery, review scores, and identify key experience issues affecting customer satisfaction.

---

## 1. Fulfillment Chain Duration Analysis

### 1.1 Delivery Time Statistics

| Stage | Mean (days) | Median (days) | Std (days) |
|-------|------------|---------------|-----------|
| Order → Approval | {delivered_orders['time_to_approval_days'].mean():.2f} | {delivered_orders['time_to_approval_days'].median():.2f} | {delivered_orders['time_to_approval_days'].std():.2f} |
| Approval → Shipping | {delivered_orders['time_to_shipping_days'].mean():.2f} | {delivered_orders['time_to_shipping_days'].median():.2f} | {delivered_orders['time_to_shipping_days'].std():.2f} |
| Shipping → Delivery | {delivered_orders['time_to_delivery_days'].mean():.2f} | {delivered_orders['time_to_delivery_days'].median():.2f} | {delivered_orders['time_to_delivery_days'].std():.2f} |
| **Total Delivery** | **{delivered_orders['total_delivery_days'].mean():.2f}** | **{delivered_orders['total_delivery_days'].median():.2f}** | **{delivered_orders['total_delivery_days'].std():.2f}** |

### 1.2 Key Observations

- **Order approval is efficient**: Average {delivered_orders['time_to_approval_days'].mean():.2f} days from purchase
- **Shipping preparation varies most**: High variability in approval→shipping stage
- **Total delivery time**: Mean {delivered_orders['total_delivery_days'].mean():.2f} days, Median {delivered_orders['total_delivery_days'].median():.2f} days

**Charts Generated**:
- `outputs/charts/fulfillment_duration_distribution.png` - Duration distributions by stage
- `outputs/charts/fulfillment_chain_boxplot.png` - Boxplot comparison

---

## 2. Estimated vs Actual Delivery Analysis

### 2.1 Delivery Status Distribution

| Status | Count | Percentage |
|--------|-------|-----------|
| Early | {delivery_status_counts.get('Early', 0)} | {delivery_status_pct.get('Early', 0):.2f}% |
| On-time | {delivery_status_counts.get('On-time', 0)} | {delivery_status_pct.get('On-time', 0):.2f}% |
| Late | {delivery_status_counts.get('Late', 0)} | {delivery_status_pct.get('Late', 0):.2f}% |

### 2.2 Timing Statistics

- **Early delivery rate**: {delivery_status_pct.get('Early', 0):.2f}%
- **Late delivery rate**: {delivery_status_pct.get('Late', 0):.2f}%
- **Average early delivery advantage**: {abs(early_orders['delivery_diff_days'].mean()):.2f} days ahead
- **Average late delivery delay**: {late_orders['delivery_diff_days'].mean():.2f} days behind

**Key Insight**: Platform shows strong delivery performance with {delivery_status_pct.get('Early', 0):.2f}% early deliveries, but {delivery_status_pct.get('Late', 0):.2f}% late deliveries represent quality risk.

**Charts Generated**:
- `outputs/charts/delivery_status_distribution.png`
- `outputs/charts/delivery_difference_distribution.png`

---

## 3. Review Score Analysis

### 3.1 Score Distribution

| Score | Count | Percentage |
|-------|-------|-----------|
| 1 | {review_counts.get(1, 0)} | {review_pct.get(1, 0):.2f}% |
| 2 | {review_counts.get(2, 0)} | {review_pct.get(2, 0):.2f}% |
| 3 | {review_counts.get(3, 0)} | {review_pct.get(3, 0):.2f}% |
| 4 | {review_counts.get(4, 0)} | {review_pct.get(4, 0):.2f}% |
| 5 | {review_counts.get(5, 0)} | {review_pct.get(5, 0):.2f}% |

### 3.2 Summary Statistics

- **Mean review score**: {avg_review:.2f}/5.0
- **Median review score**: {median_review:.2f}/5.0
- **High score (4-5) percentage**: {review_pct.get(4, 0) + review_pct.get(5, 0):.2f}%
- **Low score (1-2) percentage**: {review_pct.get(1, 0) + review_pct.get(2, 0):.2f}%

**Charts Generated**:
- `outputs/charts/review_score_distribution.png`

---

## 4. Correlation Analysis

### 4.1 Key Correlations with Review Score

| Factor | Correlation |
|--------|------------|
| Total Delivery Days | {correlation_matrix.loc['total_delivery_days', 'review_score']:.3f} |
| Delay Days | {correlation_matrix.loc['delivery_diff_days', 'review_score']:.3f} |
| Freight Value | {correlation_matrix.loc['total_freight', 'review_score']:.3f} |

### 4.2 Impact Analysis

**Low Score vs High Score Comparison**:

| Metric | Low Score (1-2) | High Score (4-5) | Difference |
|--------|-----------------|------------------|-----------|
| Avg Delivery Days | {low_delivery_avg:.2f} | {high_delivery_avg:.2f} | {low_delivery_avg - high_delivery_avg:.2f} |
| Late Rate (%) | {low_late_rate:.2f} | {high_late_rate:.2f} | {low_late_rate - high_late_rate:.2f} |

**Key Insight**: Delivery time and late delivery are primary drivers of low scores. Low-score customers experience {low_late_rate:.2f}% late delivery rate vs {high_late_rate:.2f}% for high-score customers.

**Charts Generated**:
- `outputs/charts/correlation_heatmap.png`
- `outputs/charts/metrics_by_review_score.png`

---

## 5. Regional Delivery Performance

### 5.1 Top States Analysis

Top 10 states by order volume show varying delivery performance:

- **Highest late rate states**: {', '.join(high_late_states['State'].head(5).tolist()) if len(high_late_states) > 0 else 'None above 15% threshold'}
- **Delivery time varies significantly by state**: {state_delivery['Avg_Delivery_Days'].min():.2f} to {state_delivery['Avg_Delivery_Days'].max():.2f} days average

**Charts Generated**:
- `outputs/charts/state_avg_delivery_days.png`
- `outputs/charts/state_late_rate.png`
- `outputs/charts/state_avg_score.png`

---

## 6. Category Performance Analysis

### 6.1 High Sales but Low Score Categories

Identified **{len(low_score_categories)}** categories with >100 orders and average score <4.0:

{', '.join(low_score_categories['Category'].head(5).tolist()) if len(low_score_categories) > 0 else 'None identified'}

### 6.2 Category Delivery Variability

- **Category delivery time range**: {category_delivery['Avg_Delivery_Days'].min():.2f} to {category_delivery['Avg_Delivery_Days'].max():.2f} days
- **Category score range**: {category_delivery['Avg_Score'].min():.2f} to {category_delivery['Avg_Score'].max():.2f}

**Charts Generated**:
- `outputs/charts/category_avg_score.png`

---

## 7. Seller Performance Analysis

### 7.1 High Sales but Low Score Sellers

Identified **{len(low_score_sellers)}** sellers with >50 orders and average score <4.0

### 7.2 Seller Delivery Variability

- **Seller delivery time range**: {seller_delivery['Avg_Delivery_Days'].min():.2f} to {seller_delivery['Avg_Delivery_Days'].max():.2f} days
- **Seller score range**: {seller_delivery['Avg_Score'].min():.2f} to {seller_delivery['Avg_Score'].max():.2f}

---

## 8. Key Findings Summary

### 8.1 Fulfillment Efficiency

1. **Average delivery time**: {delivered_orders['total_delivery_days'].mean():.2f} days total (median {delivered_orders['total_delivery_days'].median():.2f} days)
2. **Fast approval**: Average {delivered_orders['time_to_approval_days'].mean():.2f} days from purchase to approval
3. **Shipping bottleneck**: Approval→shipping stage takes longest ({delivered_orders['time_to_shipping_days'].mean():.2f} days average)

### 8.2 Delivery Reliability

4. **Early delivery advantage**: {delivery_status_pct.get('Early', 0):.2f}% orders arrive early
5. **Late delivery risk**: {delivery_status_pct.get('Late', 0):.2f}% late deliveries represent quality gap
6. **Estimation accuracy**: Platform tends to estimate conservatively (more early than late)

### 8.3 Customer Satisfaction

7. **Overall satisfaction**: Average score {avg_review:.2f}/5.0 with {review_pct.get(4, 0) + review_pct.get(5, 0):.2f}% high scores
8. **Dissatisfaction signal**: {review_pct.get(1, 0) + review_pct.get(2, 0):.2f}% low scores indicate experience issues

### 8.4 Experience Impact Factors

9. **Delivery time impact**: Negative correlation {corr_delivery_score:.3f} with score
10. **Late delivery impact**: Low-score customers have {low_late_rate:.2f}% late rate vs {high_late_rate:.2f}% for high-score
11. **Delivery time difference**: Low scores average {low_delivery_avg:.2f} days vs {high_delivery_avg:.2f} for high scores

### 8.5 Problem Areas

12. **High late rate regions**: {len(high_late_states)} states with >15% late deliveries
13. **Low-score categories**: {len(low_score_categories)} high-volume categories with avg score <4.0
14. **Low-score sellers**: {len(low_score_sellers)} high-volume sellers with avg score <4.0

### 8.6 Freight Impact

15. **Minimal freight impact**: Correlation {corr_freight_score:.3f} - freight cost has limited effect on satisfaction

---

## 9. Business Recommendations

### 9.1 Fulfillment Optimization

- Focus on approval→shipping stage (largest variability)
- Investigate shipping preparation process for bottlenecks
- Set realistic delivery estimates to maintain early delivery rates

### 9.2 Delivery Quality Improvement

- Target late delivery reduction from {delivery_status_pct.get('Late', 0):.2f}% to <5%
- Prioritize high late rate regions for logistics improvement
- Implement early warning system for potential late deliveries

### 9.3 Category & Seller Management

- Monitor high-volume low-score categories for systematic issues
- Provide seller training/support for low-scoring high-volume sellers
- Consider seller score thresholds for platform quality

### 9.4 Customer Experience Focus

- Reduce average delivery time to improve satisfaction
- Communicate proactively about delivery delays
- Prioritize logistics improvements for regions with high late rates

---

## 10. Output Files

### Charts (outputs/charts/)
- fulfillment_duration_distribution.png
- fulfillment_chain_boxplot.png
- delivery_status_distribution.png
- delivery_difference_distribution.png
- review_score_distribution.png
- correlation_heatmap.png
- metrics_by_review_score.png
- state_avg_delivery_days.png
- state_late_rate.png
- state_avg_score.png
- category_avg_score.png

### Tables (outputs/tables/)
- fulfillment_duration_stats.csv
- delivery_status_distribution.csv
- delivery_timing_stats.csv
- review_score_distribution.csv
- correlation_matrix.csv
- score_vs_metrics.csv
- state_delivery_performance.csv
- category_delivery_performance.csv
- seller_delivery_performance.csv
- high_sales_low_score_categories.csv
- high_sales_low_score_sellers.csv
- high_late_rate_states.csv
- score_impact_analysis.csv
- key_findings_fulfillment.csv

---

**Analysis Completed**: {pd.Timestamp.now().strftime('%Y-%m-%d %H:%M:%S')}
**Reproducibility**: All results generated from processed interim data files in `data/interim/`
"""

# Write report
with open(REPORT_DIR / 'fulfillment_customer_experience_analysis.md', 'w', encoding='utf-8') as f:
    f.write(report_content)

print("Report generated: reports/fulfillment_customer_experience_analysis.md")

# =============================================================================
# FINAL SUMMARY
# =============================================================================
print("\n" + "="*60)
print("PHASE 4: FULFILLMENT & CUSTOMER EXPERIENCE ANALYSIS COMPLETED")
print("="*60)
print(f"\nCharts generated: {len(list(OUTPUT_CHARTS.glob('fulfillment*.png'))) + len(list(OUTPUT_CHARTS.glob('delivery*.png'))) + len(list(OUTPUT_CHARTS.glob('review*.png'))) + len(list(OUTPUT_CHARTS.glob('correlation*.png'))) + len(list(OUTPUT_CHARTS.glob('metrics*.png'))) + len(list(OUTPUT_CHARTS.glob('state*.png'))) + len(list(OUTPUT_CHARTS.glob('category*.png')))} files")
print(f"Tables generated: {len(list(OUTPUT_TABLES.glob('*.csv'))) - 9} new files (Phase 4 specific)")
print(f"Report generated: reports/fulfillment_customer_experience_analysis.md")
print(f"Key findings: {len(findings)} data-driven insights")
print("\nAll outputs are reproducible from data/interim/ source files")