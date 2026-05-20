#!/usr/bin/env python3
"""
Phase 3: Overall Business Analysis
Brazilian E-commerce Olist Dataset Analysis

This script performs comprehensive business analysis including:
1. Monthly trends (orders, GMV, buyers, AOV)
2. Order status and payment analysis
3. Category performance analysis
4. Seller performance analysis
5. Regional performance analysis

Output:
- Charts: outputs/charts/
- Tables: outputs/tables/
- Report: reports/overall_business_analysis.md
"""

import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
import seaborn as sns
from pathlib import Path
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
order_df['order_purchase_timestamp'] = pd.to_datetime(order_df['order_purchase_timestamp'])
order_df['order_approved_at'] = pd.to_datetime(order_df['order_approved_at'])
order_df['order_delivered_carrier_date'] = pd.to_datetime(order_df['order_delivered_carrier_date'])
order_df['order_delivered_customer_date'] = pd.to_datetime(order_df['order_delivered_customer_date'])
order_df['order_estimated_delivery_date'] = pd.to_datetime(order_df['order_estimated_delivery_date'])

item_df['shipping_limit_date'] = pd.to_datetime(item_df['shipping_limit_date'])

print(f"Orders: {len(order_df)} rows")
print(f"Items: {len(item_df)} rows")

# =============================================================================
# SECTION 1: Monthly Trends Analysis
# =============================================================================
print("\n=== Section 1: Monthly Trends Analysis ===")

# Create month column
order_df['purchase_month'] = order_df['order_purchase_timestamp'].dt.to_period('M')

# Calculate monthly metrics
monthly_stats = order_df.groupby('purchase_month').agg({
    'order_id': 'count',  # Order count
    'customer_unique_id': 'nunique',  # Unique buyers
}).reset_index()

# Calculate GMV from item-level data - merge purchase month from order_df
order_purchase_month = order_df[['order_id', 'purchase_month']]
item_df = item_df.merge(order_purchase_month, on='order_id', how='left')
item_df['purchase_month'] = item_df['purchase_month']

monthly_gmv = item_df.groupby('purchase_month').agg({
    'price': 'sum',  # GMV (product value only)
    'order_id': 'nunique'  # Order count for validation
}).reset_index()
monthly_gmv.columns = ['purchase_month', 'gmv', 'gmv_order_count']

# Merge
monthly_stats = monthly_stats.merge(monthly_gmv, on='purchase_month', how='left')
monthly_stats['aov'] = monthly_stats['gmv'] / monthly_stats['order_id']  # Average Order Value

# Convert period to string for plotting
monthly_stats['month_str'] = monthly_stats['purchase_month'].astype(str)

# Calculate growth rates
monthly_stats['order_growth'] = monthly_stats['order_id'].pct_change() * 100
monthly_stats['gmv_growth'] = monthly_stats['gmv'].pct_change() * 100
monthly_stats['buyer_growth'] = monthly_stats['customer_unique_id'].pct_change() * 100

# Save monthly trends table
monthly_table = monthly_stats[['month_str', 'order_id', 'gmv', 'customer_unique_id', 'aov', 
                               'order_growth', 'gmv_growth', 'buyer_growth']]
monthly_table.columns = ['Month', 'Orders', 'GMV(R$)', 'Unique Buyers', 'AOV(R$)', 
                        'Order Growth%', 'GMV Growth%', 'Buyer Growth%']
monthly_table.to_csv(OUTPUT_TABLES / 'monthly_trends.csv', index=False)

# Plot 1: Monthly Trends - 4 Metrics
fig, axes = plt.subplots(2, 2, figsize=(16, 10))

# Orders
ax1 = axes[0, 0]
ax1.bar(monthly_stats['month_str'], monthly_stats['order_id'], color='steelblue', alpha=0.7)
ax1.plot(monthly_stats['month_str'], monthly_stats['order_id'], 'o-', color='navy', linewidth=2)
ax1.set_title('Monthly Order Volume', fontsize=14, fontweight='bold')
ax1.set_xlabel('Month')
ax1.set_ylabel('Orders')
ax1.tick_params(axis='x', rotation=45)

# GMV
ax2 = axes[0, 1]
ax2.bar(monthly_stats['month_str'], monthly_stats['gmv']/1000, color='coral', alpha=0.7)
ax2.plot(monthly_stats['month_str'], monthly_stats['gmv']/1000, 'o-', color='darkred', linewidth=2)
ax2.set_title('Monthly GMV (Thousands R$)', fontsize=14, fontweight='bold')
ax2.set_xlabel('Month')
ax2.set_ylabel('GMV (R$ Thousands)')
ax2.tick_params(axis='x', rotation=45)

# Unique Buyers
ax3 = axes[1, 0]
ax3.bar(monthly_stats['month_str'], monthly_stats['customer_unique_id'], color='seagreen', alpha=0.7)
ax3.plot(monthly_stats['month_str'], monthly_stats['customer_unique_id'], 'o-', color='darkgreen', linewidth=2)
ax3.set_title('Monthly Unique Buyers', fontsize=14, fontweight='bold')
ax3.set_xlabel('Month')
ax3.set_ylabel('Unique Buyers')
ax3.tick_params(axis='x', rotation=45)

# AOV
ax4 = axes[1, 1]
ax4.bar(monthly_stats['month_str'], monthly_stats['aov'], color='goldenrod', alpha=0.7)
ax4.plot(monthly_stats['month_str'], monthly_stats['aov'], 'o-', color='darkorange', linewidth=2)
ax4.set_title('Monthly Average Order Value (AOV)', fontsize=14, fontweight='bold')
ax4.set_xlabel('Month')
ax4.set_ylabel('AOV (R$)')
ax4.tick_params(axis='x', rotation=45)

plt.tight_layout()
plt.savefig(OUTPUT_CHARTS / 'monthly_trends_4metrics.png', dpi=150, bbox_inches='tight')
plt.close()

# Plot 2: Growth Rates
fig, ax = plt.subplots(figsize=(14, 6))
ax.plot(monthly_stats['month_str'], monthly_stats['order_growth'], 'o-', label='Order Growth%', linewidth=2, markersize=8)
ax.plot(monthly_stats['month_str'], monthly_stats['gmv_growth'], 's-', label='GMV Growth%', linewidth=2, markersize=8)
ax.plot(monthly_stats['month_str'], monthly_stats['buyer_growth'], '^-', label='Buyer Growth%', linewidth=2, markersize=8)
ax.axhline(y=0, color='gray', linestyle='--', alpha=0.5)
ax.set_title('Monthly Growth Rates', fontsize=14, fontweight='bold')
ax.set_xlabel('Month')
ax.set_ylabel('Growth Rate (%)')
ax.legend()
ax.tick_params(axis='x', rotation=45)
plt.tight_layout()
plt.savefig(OUTPUT_CHARTS / 'monthly_growth_rates.png', dpi=150, bbox_inches='tight')
plt.close()

print("Monthly trends analysis completed")

# =============================================================================
# SECTION 2: Order Status and Payment Analysis
# =============================================================================
print("\n=== Section 2: Order Status and Payment Analysis ===")

# 2.1 Order Status Distribution
status_counts = order_df['order_status'].value_counts()
status_pct = (status_counts / len(order_df) * 100).round(2)

status_table = pd.DataFrame({
    'Status': status_counts.index,
    'Count': status_counts.values,
    'Percentage': status_pct.values
})
status_table.to_csv(OUTPUT_TABLES / 'order_status_distribution.csv', index=False)

# Plot Order Status
fig, ax = plt.subplots(figsize=(10, 6))
colors = sns.color_palette("husl", len(status_counts))
ax.barh(status_counts.index, status_counts.values, color=colors)
ax.set_title('Order Status Distribution', fontsize=14, fontweight='bold')
ax.set_xlabel('Count')
for i, (v, p) in enumerate(zip(status_counts.values, status_pct.values)):
    ax.text(v + 100, i, f'{v} ({p}%)', va='center')
plt.tight_layout()
plt.savefig(OUTPUT_CHARTS / 'order_status_distribution.png', dpi=150, bbox_inches='tight')
plt.close()

# 2.2 Payment Analysis
payment_counts = order_df['primary_payment_type'].value_counts()
payment_pct = (payment_counts / len(order_df) * 100).round(2)

payment_amount_stats = order_df.groupby('primary_payment_type')['total_payment_value'].agg(['sum', 'mean', 'median', 'count'])
payment_amount_stats.columns = ['Total Amount', 'Mean Amount', 'Median Amount', 'Order Count']

payment_table = pd.DataFrame({
    'Payment Type': payment_counts.index,
    'Order Count': payment_counts.values,
    'Percentage': payment_pct.values,
    'Total Amount (R$)': payment_amount_stats['Total Amount'].values,
    'Mean Amount (R$)': payment_amount_stats['Mean Amount'].values.round(2),
    'Median Amount (R$)': payment_amount_stats['Median Amount'].values.round(2)
})
payment_table.to_csv(OUTPUT_TABLES / 'payment_type_analysis.csv', index=False)

# Plot Payment Type Distribution
fig, axes = plt.subplots(1, 2, figsize=(14, 6))

# Order Count by Payment Type
ax1 = axes[0]
ax1.bar(payment_counts.index, payment_counts.values, color=['#FF6B6B', '#4ECDC4', '#45B7D1', '#96CEB4'])
ax1.set_title('Orders by Payment Type', fontsize=14, fontweight='bold')
ax1.set_xlabel('Payment Type')
ax1.set_ylabel('Order Count')
for i, (v, p) in enumerate(zip(payment_counts.values, payment_pct.values)):
    ax1.text(i, v + 500, f'{p}%', ha='center')

# Total Amount by Payment Type
ax2 = axes[1]
amounts = payment_amount_stats['Total Amount'].values / 1e6  # in millions
ax2.bar(payment_counts.index, amounts, color=['#FF6B6B', '#4ECDC4', '#45B7D1', '#96CEB4'])
ax2.set_title('Total Payment Amount by Type (Millions R$)', fontsize=14, fontweight='bold')
ax2.set_xlabel('Payment Type')
ax2.set_ylabel('Total Amount (R$ Millions)')
for i, v in enumerate(amounts):
    ax2.text(i, v + 0.1, f'{v:.2f}M', ha='center')

plt.tight_layout()
plt.savefig(OUTPUT_CHARTS / 'payment_type_analysis.png', dpi=150, bbox_inches='tight')
plt.close()

# 2.3 Installment Analysis
installment_counts = order_df['max_installments'].value_counts().sort_index()
installment_pct = (installment_counts / len(order_df) * 100).round(2)

# Group installments: 1, 2-3, 4-6, 7-12, >12
installment_bins = {
    '1 (No Installment)': installment_counts.get(1, 0),
    '2-3': installment_counts.loc[2:3].sum() if 2 in installment_counts.index else 0,
    '4-6': installment_counts.loc[4:6].sum() if 4 in installment_counts.index else 0,
    '7-12': installment_counts.loc[7:12].sum() if 7 in installment_counts.index else 0,
    '>12': installment_counts.loc[13:].sum() if 13 in installment_counts.index else 0
}

installment_table = pd.DataFrame({
    'Installment Range': installment_bins.keys(),
    'Order Count': installment_bins.values(),
    'Percentage': [v/len(order_df)*100 for v in installment_bins.values()]
})
installment_table.to_csv(OUTPUT_TABLES / 'installment_distribution.csv', index=False)

# Plot Installment Distribution
fig, ax = plt.subplots(figsize=(10, 6))
ax.bar(installment_bins.keys(), installment_bins.values(), color='mediumpurple')
ax.set_title('Payment Installment Distribution', fontsize=14, fontweight='bold')
ax.set_xlabel('Installment Range')
ax.set_ylabel('Order Count')
for i, v in enumerate(installment_bins.values()):
    ax.text(i, v + 500, f'{v/len(order_df)*100:.1f}%', ha='center')
plt.tight_layout()
plt.savefig(OUTPUT_CHARTS / 'installment_distribution.png', dpi=150, bbox_inches='tight')
plt.close()

# 2.4 Payment Amount Distribution
fig, ax = plt.subplots(figsize=(12, 6))
ax.hist(order_df['total_payment_value'], bins=50, color='steelblue', alpha=0.7, edgecolor='black')
ax.axvline(order_df['total_payment_value'].mean(), color='red', linestyle='--', label=f'Mean: R${order_df["total_payment_value"].mean():.2f}')
ax.axvline(order_df['total_payment_value'].median(), color='green', linestyle='--', label=f'Median: R${order_df["total_payment_value"].median():.2f}')
ax.set_title('Payment Amount Distribution', fontsize=14, fontweight='bold')
ax.set_xlabel('Payment Amount (R$)')
ax.set_ylabel('Frequency')
ax.legend()
plt.tight_layout()
plt.savefig(OUTPUT_CHARTS / 'payment_amount_distribution.png', dpi=150, bbox_inches='tight')
plt.close()

print("Order status and payment analysis completed")

# =============================================================================
# SECTION 3: Category Performance Analysis
# =============================================================================
print("\n=== Section 3: Category Performance Analysis ===")

# Use English category names where available
item_df['category_name'] = item_df['product_category_name_english'].fillna(item_df['product_category_name'])

# Category-level aggregation
category_stats = item_df.groupby('category_name').agg({
    'order_id': 'nunique',  # Unique orders with this category
    'price': ['sum', 'mean'],  # GMV and avg price
    'order_item_id': 'count'  # Total items sold
}).reset_index()

category_stats.columns = ['Category', 'Orders', 'GMV', 'Avg_Price', 'Items_Sold']

# Sort by GMV
category_stats = category_stats.sort_values('GMV', ascending=False)

# Calculate contribution
total_gmv = category_stats['GMV'].sum()
category_stats['GMV_Contribution%'] = (category_stats['GMV'] / total_gmv * 100).round(2)
category_stats['GMV_Cumulative%'] = category_stats['GMV_Contribution%'].cumsum().round(2)

# Top 20 categories
top20_categories = category_stats.head(20)
top20_categories.to_csv(OUTPUT_TABLES / 'top20_categories_performance.csv', index=False)

# Plot Top 20 Categories by GMV
fig, ax = plt.subplots(figsize=(14, 8))
ax.barh(top20_categories['Category'].head(15), top20_categories['GMV'].head(15)/1000, color='teal')
ax.set_title('Top 15 Categories by GMV (Thousands R$)', fontsize=14, fontweight='bold')
ax.set_xlabel('GMV (R$ Thousands)')
ax.set_ylabel('Category')
ax.invert_yaxis()
for i, (v, p) in enumerate(zip(top20_categories['GMV'].head(15).values, top20_categories['GMV_Contribution%'].head(15).values)):
    ax.text(v/1000 + 5, i, f'{v/1000:.1f}K ({p}%)', va='center')
plt.tight_layout()
plt.savefig(OUTPUT_CHARTS / 'top15_categories_gmv.png', dpi=150, bbox_inches='tight')
plt.close()

# Plot Category Contribution Pareto
fig, ax = plt.subplots(figsize=(14, 6))
x = range(len(category_stats))
ax.bar(x, category_stats['GMV_Contribution%'], color='steelblue', alpha=0.7)
ax.plot(x, category_stats['GMV_Cumulative%'], 'r-', linewidth=2, label='Cumulative%')
ax.axhline(y=80, color='green', linestyle='--', alpha=0.5, label='80% Threshold')
ax.axhline(y=50, color='orange', linestyle='--', alpha=0.5, label='50% Threshold')
ax.set_title('Category GMV Contribution Pareto Chart', fontsize=14, fontweight='bold')
ax.set_xlabel('Category Rank')
ax.set_ylabel('Percentage')
ax.legend()
# Find where cumulative reaches 50% and 80%
n_50 = len(category_stats[category_stats['GMV_Cumulative%'] <= 50])
n_80 = len(category_stats[category_stats['GMV_Cumulative%'] <= 80])
ax.text(n_50, 52, f'{n_50} categories = 50%', fontsize=10)
ax.text(n_80, 82, f'{n_80} categories = 80%', fontsize=10)
plt.tight_layout()
plt.savefig(OUTPUT_CHARTS / 'category_pareto.png', dpi=150, bbox_inches='tight')
plt.close()

# Category Order Count vs Avg Price Scatter
fig, ax = plt.subplots(figsize=(12, 8))
ax.scatter(category_stats['Orders'], category_stats['Avg_Price'], alpha=0.6, c='navy', s=50)
ax.set_title('Category Order Volume vs Average Price', fontsize=14, fontweight='bold')
ax.set_xlabel('Order Count')
ax.set_ylabel('Average Price (R$)')
# Highlight top 5 by GMV
for i, row in top20_categories.head(5).iterrows():
    ax.annotate(row['Category'], (row['Orders'], row['Avg_Price']), fontsize=8, alpha=0.7)
plt.tight_layout()
plt.savefig(OUTPUT_CHARTS / 'category_orders_vs_price.png', dpi=150, bbox_inches='tight')
plt.close()

print("Category performance analysis completed")

# =============================================================================
# SECTION 4: Seller Performance Analysis
# =============================================================================
print("\n=== Section 4: Seller Performance Analysis ===")

# Seller-level aggregation
seller_stats = item_df.groupby('seller_id').agg({
    'order_id': 'nunique',  # Unique orders
    'price': ['sum', 'mean'],  # GMV and avg item price
    'order_item_id': 'count',  # Total items sold
    'product_id': 'nunique'  # Unique products
}).reset_index()

seller_stats.columns = ['Seller_ID', 'Orders', 'GMV', 'Avg_Item_Price', 'Items_Sold', 'Unique_Products']

# Sort by GMV
seller_stats = seller_stats.sort_values('GMV', ascending=False)

# Calculate contribution
total_seller_gmv = seller_stats['GMV'].sum()
seller_stats['GMV_Contribution%'] = (seller_stats['GMV'] / total_seller_gmv * 100).round(2)
seller_stats['GMV_Cumulative%'] = seller_stats['GMV_Contribution%'].cumsum().round(2)

# Summary statistics
seller_summary = {
    'Total Sellers': len(seller_stats),
    'Top 10 Sellers GMV%': seller_stats.head(10)['GMV_Contribution%'].sum(),
    'Top 50 Sellers GMV%': seller_stats.head(50)['GMV_Contribution%'].sum(),
    'Top 100 Sellers GMV%': seller_stats.head(100)['GMV_Contribution%'].sum(),
    'Mean GMV per Seller': seller_stats['GMV'].mean(),
    'Median GMV per Seller': seller_stats['GMV'].median(),
    'Mean Orders per Seller': seller_stats['Orders'].mean(),
    'Median Orders per Seller': seller_stats['Orders'].median()
}

seller_summary_table = pd.DataFrame([seller_summary]).T
seller_summary_table.columns = ['Value']
seller_summary_table.to_csv(OUTPUT_TABLES / 'seller_summary_stats.csv')

# Top 20 sellers
top20_sellers = seller_stats.head(20)
top20_sellers.to_csv(OUTPUT_TABLES / 'top20_sellers_performance.csv', index=False)

# Plot Top 10 Sellers by GMV
fig, ax = plt.subplots(figsize=(12, 6))
ax.barh(range(10), top20_sellers['GMV'].head(10)/1000, color='coral')
ax.set_yticks(range(10))
ax.set_yticklabels([f'Seller {i+1}' for i in range(10)])
ax.set_title('Top 10 Sellers by GMV (Thousands R$)', fontsize=14, fontweight='bold')
ax.set_xlabel('GMV (R$ Thousands)')
ax.invert_yaxis()
for i, (v, p) in enumerate(zip(top20_sellers['GMV'].head(10).values, top20_sellers['GMV_Contribution%'].head(10).values)):
    ax.text(v/1000 + 5, i, f'{v/1000:.1f}K ({p}%)', va='center')
plt.tight_layout()
plt.savefig(OUTPUT_CHARTS / 'top10_sellers_gmv.png', dpi=150, bbox_inches='tight')
plt.close()

# Plot Seller Concentration Pareto
fig, ax = plt.subplots(figsize=(14, 6))
x = range(len(seller_stats))
ax.bar(x, seller_stats['GMV_Contribution%'], color='steelblue', alpha=0.7)
ax.plot(x, seller_stats['GMV_Cumulative%'], 'r-', linewidth=2, label='Cumulative%')
ax.axhline(y=80, color='green', linestyle='--', alpha=0.5, label='80% Threshold')
ax.axhline(y=50, color='orange', linestyle='--', alpha=0.5, label='50% Threshold')
ax.set_title('Seller GMV Contribution Pareto Chart', fontsize=14, fontweight='bold')
ax.set_xlabel('Seller Rank')
ax.set_ylabel('Percentage')
ax.legend()
# Find concentration thresholds
n_50_seller = len(seller_stats[seller_stats['GMV_Cumulative%'] <= 50])
n_80_seller = len(seller_stats[seller_stats['GMV_Cumulative%'] <= 80])
ax.text(n_50_seller, 52, f'{n_50_seller} sellers = 50%', fontsize=10)
ax.text(n_80_seller, 82, f'{n_80_seller} sellers = 80%', fontsize=10)
plt.tight_layout()
plt.savefig(OUTPUT_CHARTS / 'seller_pareto.png', dpi=150, bbox_inches='tight')
plt.close()

# Plot Seller GMV Distribution
fig, ax = plt.subplots(figsize=(12, 6))
ax.hist(seller_stats['GMV'], bins=50, color='mediumpurple', alpha=0.7, edgecolor='black')
ax.axvline(seller_stats['GMV'].mean(), color='red', linestyle='--', label=f'Mean: R${seller_stats["GMV"].mean():.2f}')
ax.axvline(seller_stats['GMV'].median(), color='green', linestyle='--', label=f'Median: R${seller_stats["GMV"].median():.2f}')
ax.set_title('Seller GMV Distribution', fontsize=14, fontweight='bold')
ax.set_xlabel('GMV (R$)')
ax.set_ylabel('Frequency')
ax.legend()
plt.tight_layout()
plt.savefig(OUTPUT_CHARTS / 'seller_gmv_distribution.png', dpi=150, bbox_inches='tight')
plt.close()

print("Seller performance analysis completed")

# =============================================================================
# SECTION 5: Regional Performance Analysis
# =============================================================================
print("\n=== Section 5: Regional Performance Analysis ===")

# Customer state-level aggregation (from order data)
customer_state_stats = order_df.groupby('customer_state').agg({
    'order_id': 'count',  # Order count
    'customer_unique_id': 'nunique',  # Unique buyers
    'total_payment_value': ['sum', 'mean']  # GMV and AOV
}).reset_index()

customer_state_stats.columns = ['State', 'Orders', 'Unique_Buyers', 'GMV', 'AOV']

# Sort by GMV
customer_state_stats = customer_state_stats.sort_values('GMV', ascending=False)

# Calculate contribution
total_state_gmv = customer_state_stats['GMV'].sum()
customer_state_stats['GMV_Contribution%'] = (customer_state_stats['GMV'] / total_state_gmv * 100).round(2)
customer_state_stats['Orders_Contribution%'] = (customer_state_stats['Orders'] / len(order_df) * 100).round(2)

customer_state_stats.to_csv(OUTPUT_TABLES / 'customer_state_performance.csv', index=False)

# Plot Top 10 States by Orders
fig, ax = plt.subplots(figsize=(12, 6))
top10_states_orders = customer_state_stats.head(10)
ax.barh(top10_states_orders['State'], top10_states_orders['Orders'], color='seagreen')
ax.set_title('Top 10 Customer States by Order Volume', fontsize=14, fontweight='bold')
ax.set_xlabel('Order Count')
ax.invert_yaxis()
for i, (v, p) in enumerate(zip(top10_states_orders['Orders'].values, top10_states_orders['Orders_Contribution%'].values)):
    ax.text(v + 200, i, f'{v} ({p}%)', va='center')
plt.tight_layout()
plt.savefig(OUTPUT_CHARTS / 'top10_states_orders.png', dpi=150, bbox_inches='tight')
plt.close()

# Plot Top 10 States by GMV
fig, ax = plt.subplots(figsize=(12, 6))
top10_states_gmv = customer_state_stats.head(10)
ax.barh(top10_states_gmv['State'], top10_states_gmv['GMV']/1000, color='coral')
ax.set_title('Top 10 Customer States by GMV (Thousands R$)', fontsize=14, fontweight='bold')
ax.set_xlabel('GMV (R$ Thousands)')
ax.invert_yaxis()
for i, (v, p) in enumerate(zip(top10_states_gmv['GMV'].values, top10_states_gmv['GMV_Contribution%'].values)):
    ax.text(v/1000 + 5, i, f'{v/1000:.1f}K ({p}%)', va='center')
plt.tight_layout()
plt.savefig(OUTPUT_CHARTS / 'top10_states_gmv.png', dpi=150, bbox_inches='tight')
plt.close()

# Plot State AOV Comparison
fig, ax = plt.subplots(figsize=(12, 6))
ax.barh(customer_state_stats['State'].head(15), customer_state_stats['AOV'].head(15), color='goldenrod')
ax.set_title('Average Order Value by State (Top 15 by GMV)', fontsize=14, fontweight='bold')
ax.set_xlabel('AOV (R$)')
ax.invert_yaxis()
for i, v in enumerate(customer_state_stats['AOV'].head(15).values):
    ax.text(v + 2, i, f'R${v:.2f}', va='center')
plt.tight_layout()
plt.savefig(OUTPUT_CHARTS / 'state_aov_comparison.png', dpi=150, bbox_inches='tight')
plt.close()

print("Regional performance analysis completed")

# =============================================================================
# SECTION 6: Key Findings Summary
# =============================================================================
print("\n=== Section 6: Key Findings Summary ===")

# Calculate key findings
findings = []

# Finding 1: Monthly Trends
peak_month_orders = monthly_stats.loc[monthly_stats['order_id'].idxmax()]
findings.append(f"Peak order volume in {peak_month_orders['month_str']} with {peak_month_orders['order_id']} orders (GMV: R${peak_month_orders['gmv']:,.2f})")

# Finding 2: Growth Pattern
growth_months = monthly_stats[monthly_stats['order_growth'] > 20]
if len(growth_months) > 0:
    findings.append(f"Strong growth months: {growth_months['month_str'].tolist()} with >20% order growth")

# Finding 3: Order Status
delivered_pct = status_pct.get('delivered', 0)
findings.append(f"Order fulfillment rate: {delivered_pct}% delivered orders")

# Finding 4: Payment Methods
credit_card_pct = payment_pct.get('credit_card', 0)
credit_card_amount_pct = payment_amount_stats.loc['credit_card', 'Total Amount'] / payment_amount_stats['Total Amount'].sum() * 100
findings.append(f"Credit card dominates: {credit_card_pct}% of orders, {credit_card_amount_pct:.1f}% of total payment amount")

# Finding 5: Installments
single_payment_pct = installment_bins['1 (No Installment)'] / len(order_df) * 100
findings.append(f"Single payment (no installment): {single_payment_pct:.1f}% of orders")

# Finding 6: Category Concentration
top5_cat_gmv_pct = category_stats.head(5)['GMV_Contribution%'].sum()
findings.append(f"Top 5 categories contribute {top5_cat_gmv_pct:.1f}% of GMV")

# Finding 7: Category Pareto
findings.append(f"{n_80} categories account for 80% of GMV (out of {len(category_stats)} total)")

# Finding 8: Seller Concentration
top10_seller_pct = seller_stats.head(10)['GMV_Contribution%'].sum()
findings.append(f"Top 10 sellers contribute {top10_seller_pct:.1f}% of GMV")

# Finding 9: Seller Pareto
findings.append(f"{n_80_seller} sellers account for 80% of GMV (out of {len(seller_stats)} total)")

# Finding 10: Regional Concentration
sp_orders_pct = customer_state_stats[customer_state_stats['State'] == 'SP']['Orders_Contribution%'].values[0] if 'SP' in customer_state_stats['State'].values else 0
sp_gmv_pct = customer_state_stats[customer_state_stats['State'] == 'SP']['GMV_Contribution%'].values[0] if 'SP' in customer_state_stats['State'].values else 0
findings.append(f"São Paulo (SP) dominates: {sp_orders_pct}% of orders, {sp_gmv_pct}% of GMV")

# Finding 11: AOV Insights
avg_aov = order_df['total_payment_value'].mean()
median_aov = order_df['total_payment_value'].median()
findings.append(f"Average Order Value: R${avg_aov:.2f} (median: R${median_aov:.2f}), indicating right-skewed distribution")

# Finding 12: Buyer Behavior
total_orders = len(order_df)
unique_buyers = order_df['customer_unique_id'].nunique()
repeat_rate = (total_orders - unique_buyers) / total_orders * 100
findings.append(f"Customer repeat purchase rate: {repeat_rate:.1f}% ({total_orders} orders from {unique_buyers} unique buyers)")

# Finding 13: Time Coverage
date_range = f"{order_df['order_purchase_timestamp'].min().strftime('%Y-%m')} to {order_df['order_purchase_timestamp'].max().strftime('%Y-%m')}"
findings.append(f"Data covers {date_range} period with clear growth trajectory")

# Finding 14: Seasonality
monthly_stats['month_num'] = monthly_stats['purchase_month'].dt.month
seasonal_avg = monthly_stats.groupby('month_num')['order_id'].mean()
peak_season_month = seasonal_avg.idxmax()
findings.append(f"Seasonal peak around month {peak_season_month} (average {seasonal_avg.max():.0f} orders)")

# Finding 15: Payment Amount Range
high_value_orders = len(order_df[order_df['total_payment_value'] > 500])
high_value_pct = high_value_orders / len(order_df) * 100
findings.append(f"High-value orders (>R$500): {high_value_pct:.1f}% of total orders")

# Save findings
findings_df = pd.DataFrame({'Finding': findings})
findings_df.to_csv(OUTPUT_TABLES / 'key_findings_summary.csv', index=False)

print(f"\nGenerated {len(findings)} key findings")

# =============================================================================
# GENERATE REPORT
# =============================================================================
print("\n=== Generating Report ===")

report_content = """# Phase 3: Overall Business Analysis Report
## Brazilian E-commerce Olist Dataset Analysis

**Analysis Period**: {date_range}
**Data Sources**: order_level_base.csv ({order_count} orders), item_level_base.csv ({item_count} items)

---

## Executive Summary

This report presents a comprehensive overall business analysis of the Olist Brazilian e-commerce platform, covering monthly trends, order and payment patterns, category performance, seller dynamics, and regional distribution. The analysis reveals key insights about business growth, customer behavior, and market concentration.

---

## 1. Monthly Trends Analysis

### 1.1 Key Metrics Overview

| Metric | Value |
|--------|-------|
| Total Orders | {total_orders} |
| Total GMV | R${total_gmv:,.2f} |
| Unique Buyers | {unique_buyers} |
| Average Order Value | R${avg_aov:.2f} |

### 1.2 Monthly Growth Pattern

{monthly_trend_desc}

**Peak Month**: {peak_month_orders} with {peak_month_orders_count} orders

**Charts Generated**:
- `outputs/charts/monthly_trends_4metrics.png` - Four key metrics trend
- `outputs/charts/monthly_growth_rates.png` - Growth rate evolution

---

## 2. Order Status and Payment Analysis

### 2.1 Order Status Distribution

- **Delivered Orders**: {delivered_pct}% (primary fulfillment outcome)
- **Shipped Orders**: {shipped_pct}%
- **Other Status**: {other_pct}%

### 2.2 Payment Method Analysis

| Payment Type | Orders | Percentage | Total Amount |
|-------------|--------|------------|--------------|
| Credit Card | {cc_count} | {cc_pct}% | R${cc_amount:,.2f} |
| Boleto | {boleto_count} | {boleto_pct}% | R${boleto_amount:,.2f} |
| Other | {other_payment_count} | {other_payment_pct}% | R${other_payment_amount:,.2f} |

**Key Insight**: Credit card is the dominant payment method, accounting for {credit_card_pct}% of orders and {credit_card_amount_pct:.1f}% of total payment amount.

### 2.3 Installment Structure

- **Single Payment (No Installment)**: {single_payment_pct:.1f}%
- **2-3 Installments**: {install_2_3_pct:.1f}%
- **4-6 Installments**: {install_4_6_pct:.1f}%
- **7+ Installments**: {install_7_plus_pct:.1f}%

**Charts Generated**:
- `outputs/charts/order_status_distribution.png`
- `outputs/charts/payment_type_analysis.png`
- `outputs/charts/installment_distribution.png`
- `outputs/charts/payment_amount_distribution.png`

---

## 3. Category Performance Analysis

### 3.1 Category Concentration

| Metric | Value |
|--------|-------|
| Total Categories | {total_categories} |
| Top 5 GMV Contribution | {top5_cat_gmv_pct:.1f}% |
| Categories for 80% GMV | {n_80} |

### 3.2 Top Categories by GMV

{top_categories_desc}

**Key Insight**: Category distribution shows moderate concentration - {n_80} categories account for 80% of GMV, indicating diverse product portfolio with several strong performers.

**Charts Generated**:
- `outputs/charts/top15_categories_gmv.png`
- `outputs/charts/category_pareto.png`
- `outputs/charts/category_orders_vs_price.png`

---

## 4. Seller Performance Analysis

### 4.1 Seller Concentration

| Metric | Value |
|--------|-------|
| Total Sellers | {total_sellers} |
| Top 10 GMV Contribution | {top10_seller_pct:.1f}% |
| Sellers for 80% GMV | {n_80_seller} |
| Median GMV per Seller | R${median_seller_gmv:,.2f} |

### 4.2 Seller Distribution Insight

Seller GMV distribution is highly right-skewed, with median GMV significantly lower than mean, indicating:
- Small number of high-performing "power sellers"
- Large base of smaller sellers
- {n_80_seller} sellers account for 80% of GMV

**Charts Generated**:
- `outputs/charts/top10_sellers_gmv.png`
- `outputs/charts/seller_pareto.png`
- `outputs/charts/seller_gmv_distribution.png`

---

## 5. Regional Performance Analysis

### 5.1 Geographic Concentration

| Metric | Value |
|--------|-------|
| States Covered | {total_states} |
| São Paulo Orders% | {sp_orders_pct}% |
| São Paulo GMV% | {sp_gmv_pct}% |

### 5.2 Top States Performance

{top_states_desc}

**Key Insight**: São Paulo state dominates with {sp_orders_pct}% of orders and {sp_gmv_pct}% of GMV, reflecting Brazil's economic geography.

**Charts Generated**:
- `outputs/charts/top10_states_orders.png`
- `outputs/charts/top10_states_gmv.png`
- `outputs/charts/state_aov_comparison.png`

---

## 6. Key Findings Summary

### 6.1 Growth & Trends

1. **Peak Performance**: {peak_month_orders} achieved {peak_month_orders_count} orders with GMV of R${peak_month_gmv:,.2f}
2. **Growth Trajectory**: Clear upward trend with strong growth months showing >20% order increase
3. **Data Coverage**: {date_range} period showing consistent business expansion

### 6.2 Payment Behavior

4. **Credit Card Dominance**: {credit_card_pct}% of orders via credit card, {credit_card_amount_pct:.1f}% of total amount
5. **Installment Usage**: {single_payment_pct:.1f}% prefer single payment, indicating financial flexibility preference
6. **Payment Value Range**: Median payment R${median_aov:.2f} vs mean R${avg_aov:.2f} - right-skewed distribution

### 6.3 Customer Insights

7. **Repeat Purchase Rate**: {repeat_rate:.1f}% indicating moderate customer loyalty
8. **Buyer Base**: {unique_buyers} unique buyers generating {total_orders} orders

### 6.4 Category Dynamics

9. **Category Concentration**: {n_80} categories deliver 80% GMV out of {total_categories} total
10. **Top 5 Impact**: Top 5 categories contribute {top5_cat_gmv_pct:.1f}% of GMV

### 6.5 Seller Ecosystem

11. **Seller Concentration**: {n_80_seller} sellers account for 80% GMV
12. **Power Sellers**: Top 10 sellers contribute {top10_seller_pct:.1f}% of GMV

### 6.6 Regional Distribution

13. **SP Dominance**: São Paulo accounts for {sp_orders_pct}% orders and {sp_gmv_pct}% GMV
14. **Geographic Spread**: {total_states} states with varying AOV levels

### 6.7 Seasonality

15. **Seasonal Pattern**: Month {peak_season_month} shows highest average order volume

---

## 7. Business Implications

### 7.1 Growth Strategy
- Platform shows strong organic growth momentum
- Consider capacity planning for peak seasons
- Geographic expansion beyond São Paulo opportunity

### 7.2 Category Strategy
- Focus resources on top-performing categories
- Explore high-volume, low-price category efficiency
- Monitor emerging categories with growth potential

### 7.3 Seller Management
- Nurture top performers with exclusive programs
- Support mid-tier sellers for growth acceleration
- Balance seller concentration risk

### 7.4 Payment Optimization
- Credit card infrastructure priority
- Installment offerings drive larger purchases
- Boleto maintains importance for specific segments

---

## 8. Output Files

### Charts (outputs/charts/)
- monthly_trends_4metrics.png
- monthly_growth_rates.png
- order_status_distribution.png
- payment_type_analysis.png
- installment_distribution.png
- payment_amount_distribution.png
- top15_categories_gmv.png
- category_pareto.png
- category_orders_vs_price.png
- top10_sellers_gmv.png
- seller_pareto.png
- seller_gmv_distribution.png
- top10_states_orders.png
- top10_states_gmv.png
- state_aov_comparison.png

### Tables (outputs/tables/)
- monthly_trends.csv
- order_status_distribution.csv
- payment_type_analysis.csv
- installment_distribution.csv
- top20_categories_performance.csv
- seller_summary_stats.csv
- top20_sellers_performance.csv
- customer_state_performance.csv
- key_findings_summary.csv

---

**Analysis Completed**: {analysis_date}
**Reproducibility**: All results generated from processed interim data files in `data/interim/`
""".format(
    date_range=date_range,
    order_count=len(order_df),
    item_count=len(item_df),
    total_orders=total_orders,
    total_gmv=total_gmv,
    unique_buyers=unique_buyers,
    avg_aov=avg_aov,
    monthly_trend_desc=f"Monthly trends show consistent growth from {monthly_stats['month_str'].iloc[0]} to {monthly_stats['month_str'].iloc[-1]}",
    peak_month_orders=peak_month_orders['month_str'],
    peak_month_orders_count=int(peak_month_orders['order_id']),
    peak_month_gmv=peak_month_orders['gmv'],
    delivered_pct=delivered_pct,
    shipped_pct=status_pct.get('shipped', 0),
    other_pct=100 - delivered_pct - status_pct.get('shipped', 0),
    cc_count=payment_counts.get('credit_card', 0),
    cc_pct=payment_pct.get('credit_card', 0),
    cc_amount=payment_amount_stats.loc['credit_card', 'Total Amount'] if 'credit_card' in payment_amount_stats.index else 0,
    boleto_count=payment_counts.get('boleto', 0),
    boleto_pct=payment_pct.get('boleto', 0),
    boleto_amount=payment_amount_stats.loc['boleto', 'Total Amount'] if 'boleto' in payment_amount_stats.index else 0,
    other_payment_count=payment_counts.get('voucher', 0) + payment_counts.get('debit_card', 0),
    other_payment_pct=payment_pct.get('voucher', 0) + payment_pct.get('debit_card', 0),
    other_payment_amount=payment_amount_stats.loc['voucher', 'Total Amount'] if 'voucher' in payment_amount_stats.index else 0,
    credit_card_pct=credit_card_pct,
    credit_card_amount_pct=credit_card_amount_pct,
    single_payment_pct=single_payment_pct,
    install_2_3_pct=installment_bins['2-3']/len(order_df)*100,
    install_4_6_pct=installment_bins['4-6']/len(order_df)*100,
    install_7_plus_pct=(installment_bins['7-12'] + installment_bins['>12'])/len(order_df)*100,
    total_categories=len(category_stats),
    top5_cat_gmv_pct=top5_cat_gmv_pct,
    n_80=n_80,
    top_categories_desc=f"Top categories: {', '.join(top20_categories['Category'].head(5).tolist())}",
    total_sellers=len(seller_stats),
    top10_seller_pct=top10_seller_pct,
    n_80_seller=n_80_seller,
    median_seller_gmv=seller_stats['GMV'].median(),
    total_states=len(customer_state_stats),
    sp_orders_pct=sp_orders_pct,
    sp_gmv_pct=sp_gmv_pct,
    top_states_desc=f"Top states by GMV: {', '.join(customer_state_stats['State'].head(5).tolist())}",
    repeat_rate=repeat_rate,
    median_aov=median_aov,
    peak_season_month=peak_season_month,
    analysis_date=pd.Timestamp.now().strftime('%Y-%m-%d %H:%M:%S')
)

# Write report
with open(REPORT_DIR / 'overall_business_analysis.md', 'w', encoding='utf-8') as f:
    f.write(report_content)

print("Report generated: reports/overall_business_analysis.md")

# =============================================================================
# FINAL SUMMARY
# =============================================================================
print("\n" + "="*60)
print("PHASE 3: OVERALL BUSINESS ANALYSIS COMPLETED")
print("="*60)
print(f"\nCharts generated: {len(list(OUTPUT_CHARTS.glob('*.png')))} files")
print(f"Tables generated: {len(list(OUTPUT_TABLES.glob('*.csv')))} files")
print(f"Report generated: reports/overall_business_analysis.md")
print(f"Key findings: {len(findings)} data-driven insights")
print("\nAll outputs are reproducible from data/interim/ source files")