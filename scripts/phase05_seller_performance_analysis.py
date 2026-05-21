#!/usr/bin/env python3
"""
Phase 5: Seller Performance After Closed Deal Analysis
Marketing Funnel & Seller Growth Analysis

This script analyzes seller performance metrics after successful deal closure:
1. Associate closed deals with sellers and orders
2. Calculate post-deal business metrics:
   - Order count
   - GMV (Gross Merchandise Value)
   - Average review score
   - Fulfillment performance (order status distribution)

Output:
- Tables: outputs/tables/
"""

import pandas as pd
import numpy as np
from pathlib import Path
import warnings
warnings.filterwarnings('ignore')

# Data paths
DATA_DIR = Path('data')
INTERIM_DIR = Path('data/interim')
OUTPUT_TABLES = Path('outputs/tables')

# Ensure output directory exists
OUTPUT_TABLES.mkdir(parents=True, exist_ok=True)

print("=" * 80)
print("PHASE 5: SELLER PERFORMANCE AFTER CLOSED DEAL ANALYSIS")
print("=" * 80)

# =============================================================================
# SECTION 1: Load Data
# =============================================================================
print("\n=== Section 1: Loading Data ===\n")

# Load closed deals
closed_df = pd.read_csv(DATA_DIR / 'olist_closed_deals_dataset.csv')
print(f"Closed deals: {len(closed_df)} rows")
print(f"Closed deals columns: {list(closed_df.columns)}")

# Load MQL data for origin information
mql_df = pd.read_csv(DATA_DIR / 'olist_marketing_qualified_leads_dataset.csv')
print(f"MQL data: {len(mql_df)} rows")

# Load item-level data
item_df = pd.read_csv(INTERIM_DIR / 'item_level_base.csv')
print(f"Item-level data: {len(item_df)} rows")

# Load order-level data
order_df = pd.read_csv(INTERIM_DIR / 'order_level_base.csv')
print(f"Order-level data: {len(order_df)} rows")

# =============================================================================
# SECTION 2: Associate Closed Deals with Sellers and Orders
# =============================================================================
print("\n=== Section 2: Associating Closed Deals with Sellers and Orders ===\n")

# Merge closed deals with MQL to get origin information
closed_with_origin = closed_df.merge(
    mql_df[['mql_id', 'origin', 'first_contact_date', 'landing_page_id']], 
    on='mql_id', 
    how='left',
    sort=False
)

print(f"Closed deals with origin: {len(closed_with_origin)} rows")
print(f"Unique sellers in closed deals: {closed_with_origin['seller_id'].nunique()}")
print(f"Origin distribution in closed deals:")
print(closed_with_origin['origin'].value_counts())

# Get unique seller_ids from closed deals
closed_sellers = closed_with_origin['seller_id'].unique()
print(f"\nTotal closed sellers: {len(closed_sellers)}")

# Filter item-level data for closed sellers
seller_items = item_df[item_df['seller_id'].isin(closed_sellers)].copy()
print(f"Items from closed sellers: {len(seller_items)}")

# Get unique orders for these items
closed_seller_orders = seller_items['order_id'].unique()
print(f"Unique orders from closed sellers: {len(closed_seller_orders)}")

# Filter order-level data for these orders
seller_order_data = order_df[order_df['order_id'].isin(closed_seller_orders)].copy()
print(f"Order records for closed sellers: {len(seller_order_data)}")

# =============================================================================
# SECTION 3: Calculate Seller Business Metrics
# =============================================================================
print("\n=== Section 3: Calculating Seller Business Metrics ===\n")

# 3.1 Calculate GMV and order count from item-level data
print("Calculating GMV and order count from item-level data...")

seller_metrics = seller_items.groupby('seller_id', sort=False).agg({
    'order_id': 'nunique',  # unique order count
    'price': 'sum',         # total price (GMV)
    'freight_value': 'sum',  # total freight
}).reset_index()

seller_metrics.columns = ['seller_id', 'order_count', 'total_price', 'total_freight']
seller_metrics['gmv'] = seller_metrics['total_price'] + seller_metrics['total_freight']

print(f"Sellers with order metrics: {len(seller_metrics)}")

# 3.2 Calculate average review score
print("Calculating average review score...")

# Get review scores from item-level data (already has review_score from order)
seller_reviews = seller_items.groupby('seller_id', sort=False).agg({
    'review_score': 'mean'  # average review score
}).reset_index()

seller_reviews.columns = ['seller_id', 'avg_review_score']

print(f"Sellers with review data: {len(seller_reviews)}")

# 3.3 Calculate order status distribution
print("Calculating order status distribution...")

# Merge items with order status
seller_order_status = seller_items[['seller_id', 'order_id']].drop_duplicates()
seller_order_status = seller_order_status.merge(
    order_df[['order_id', 'order_status']],
    on='order_id',
    how='left',
    sort=False
)

# Create pivot table for order status
status_pivot = seller_order_status.groupby(['seller_id', 'order_status'], sort=False).size().unstack(fill_value=0)
status_pivot = status_pivot.reset_index()

# Rename columns
status_pivot.columns = ['seller_id'] + [f'orders_{col}' for col in status_pivot.columns[1:]]

print(f"Sellers with order status data: {len(status_pivot)}")

# =============================================================================
# SECTION 4: Merge All Metrics
# =============================================================================
print("\n=== Section 4: Merging All Metrics ===\n")

# Start with closed deals with origin
seller_performance = closed_with_origin.copy()

# Merge with order count and GMV
seller_performance = seller_performance.merge(
    seller_metrics,
    on='seller_id',
    how='left',
    sort=False
)

# Merge with review scores
seller_performance = seller_performance.merge(
    seller_reviews,
    on='seller_id',
    how='left',
    sort=False
)

# Merge with order status distribution
seller_performance = seller_performance.merge(
    status_pivot,
    on='seller_id',
    how='left',
    sort=False
)

print(f"Final seller performance data: {len(seller_performance)} rows")
print(f"Columns: {list(seller_performance.columns)}")

# =============================================================================
# SECTION 5: Calculate Additional Metrics
# =============================================================================
print("\n=== Section 5: Calculating Additional Metrics ===\n")

# Calculate delivery rate
if 'orders_delivered' in seller_performance.columns:
    seller_performance['delivery_rate'] = (
        seller_performance['orders_delivered'] / seller_performance['order_count'] * 100
    ).round(2)

# Calculate canceled rate
if 'orders_canceled' in seller_performance.columns:
    seller_performance['cancel_rate'] = (
        seller_performance['orders_canceled'] / seller_performance['order_count'] * 100
    ).round(2)

# Calculate average order value
seller_performance['avg_order_value'] = (
    seller_performance['gmv'] / seller_performance['order_count']
).round(2)

# Fill NaN values
seller_performance['order_count'] = seller_performance['order_count'].fillna(0).astype(int)
seller_performance['gmv'] = seller_performance['gmv'].fillna(0)
seller_performance['avg_review_score'] = seller_performance['avg_review_score'].round(2)

# =============================================================================
# SECTION 6: Generate Summary Tables
# =============================================================================
print("\n=== Section 6: Generating Summary Tables ===\n")

# 6.1 Overall seller performance summary
print("Generating overall seller performance summary...")

summary_cols = [
    'seller_id', 'mql_id', 'origin', 'lead_type', 'business_segment', 'business_type',
    'won_date', 'order_count', 'gmv', 'avg_review_score', 'avg_order_value'
]

# Add optional columns if they exist
if 'delivery_rate' in seller_performance.columns:
    summary_cols.append('delivery_rate')
if 'cancel_rate' in seller_performance.columns:
    summary_cols.append('cancel_rate')

# Select columns that exist
available_cols = [col for col in summary_cols if col in seller_performance.columns]

seller_summary = seller_performance[available_cols].copy()

# Sort by GMV descending
seller_summary = seller_summary.sort_values('gmv', ascending=False)

# Save to CSV
output_path = OUTPUT_TABLES / 'seller_performance_summary.csv'
seller_summary.to_csv(output_path, index=False)
print(f"✓ Saved: {output_path}")
print(f"  Rows: {len(seller_summary)}")

# 6.2 Performance by origin
print("\nGenerating performance by origin...")

origin_performance = seller_performance.groupby('origin', sort=False).agg({
    'seller_id': 'count',
    'order_count': ['sum', 'mean'],
    'gmv': ['sum', 'mean'],
    'avg_review_score': 'mean',
    'avg_order_value': 'mean'
}).round(2)

# Flatten column names
origin_performance.columns = [
    'num_sellers', 'total_orders', 'avg_orders_per_seller',
    'total_gmv', 'avg_gmv_per_seller', 'avg_review_score', 'avg_order_value'
]

origin_performance = origin_performance.reset_index()

# Sort by total GMV descending
origin_performance = origin_performance.sort_values('total_gmv', ascending=False)

output_path = OUTPUT_TABLES / 'seller_performance_by_origin.csv'
origin_performance.to_csv(output_path, index=False)
print(f"✓ Saved: {output_path}")
print(f"  Rows: {len(origin_performance)}")

# 6.3 Performance by lead type
print("\nGenerating performance by lead type...")

lead_type_performance = seller_performance.groupby('lead_type', sort=False).agg({
    'seller_id': 'count',
    'order_count': ['sum', 'mean'],
    'gmv': ['sum', 'mean'],
    'avg_review_score': 'mean',
    'avg_order_value': 'mean'
}).round(2)

# Flatten column names
lead_type_performance.columns = [
    'num_sellers', 'total_orders', 'avg_orders_per_seller',
    'total_gmv', 'avg_gmv_per_seller', 'avg_review_score', 'avg_order_value'
]

lead_type_performance = lead_type_performance.reset_index()

# Sort by total GMV descending
lead_type_performance = lead_type_performance.sort_values('total_gmv', ascending=False)

output_path = OUTPUT_TABLES / 'seller_performance_by_lead_type.csv'
lead_type_performance.to_csv(output_path, index=False)
print(f"✓ Saved: {output_path}")
print(f"  Rows: {len(lead_type_performance)}")

# 6.4 Performance by business segment
print("\nGenerating performance by business segment...")

segment_performance = seller_performance.groupby('business_segment', sort=False).agg({
    'seller_id': 'count',
    'order_count': ['sum', 'mean'],
    'gmv': ['sum', 'mean'],
    'avg_review_score': 'mean',
    'avg_order_value': 'mean'
}).round(2)

# Flatten column names
segment_performance.columns = [
    'num_sellers', 'total_orders', 'avg_orders_per_seller',
    'total_gmv', 'avg_gmv_per_seller', 'avg_review_score', 'avg_order_value'
]

segment_performance = segment_performance.reset_index()

# Sort by total GMV descending
segment_performance = segment_performance.sort_values('total_gmv', ascending=False)

output_path = OUTPUT_TABLES / 'seller_performance_by_segment.csv'
segment_performance.to_csv(output_path, index=False)
print(f"✓ Saved: {output_path}")
print(f"  Rows: {len(segment_performance)}")

# 6.5 High-value sellers (top performers)
print("\nIdentifying high-value sellers...")

# Define high-value as: GMV > median AND order_count > median AND avg_review_score >= 4.0
median_gmv = seller_performance['gmv'].median()
median_orders = seller_performance['order_count'].median()

high_value_sellers = seller_performance[
    (seller_performance['gmv'] > median_gmv) & 
    (seller_performance['order_count'] > median_orders) &
    (seller_performance['avg_review_score'] >= 4.0)
].copy()

# Select key columns
high_value_cols = [
    'seller_id', 'origin', 'lead_type', 'business_segment',
    'order_count', 'gmv', 'avg_review_score', 'avg_order_value'
]

if 'delivery_rate' in high_value_sellers.columns:
    high_value_cols.append('delivery_rate')

available_hv_cols = [col for col in high_value_cols if col in high_value_sellers.columns]
high_value_sellers = high_value_sellers[available_hv_cols]

# Sort by GMV
high_value_sellers = high_value_sellers.sort_values('gmv', ascending=False)

output_path = OUTPUT_TABLES / 'high_value_sellers.csv'
high_value_sellers.to_csv(output_path, index=False)
print(f"✓ Saved: {output_path}")
print(f"  High-value sellers: {len(high_value_sellers)}")

# 6.6 Low-performing sellers (need attention)
print("\nIdentifying low-performing sellers...")

# Define low-performing as: order_count > 0 BUT avg_review_score < 4.0
low_performers = seller_performance[
    (seller_performance['order_count'] > 0) & 
    (seller_performance['avg_review_score'] < 4.0)
].copy()

# Select key columns
lp_cols = [
    'seller_id', 'origin', 'lead_type', 'business_segment',
    'order_count', 'gmv', 'avg_review_score', 'avg_order_value'
]

available_lp_cols = [col for col in lp_cols if col in low_performers.columns]
low_performers = low_performers[available_lp_cols]

# Sort by review score
low_performers = low_performers.sort_values('avg_review_score')

output_path = OUTPUT_TABLES / 'low_performing_sellers.csv'
low_performers.to_csv(output_path, index=False)
print(f"✓ Saved: {output_path}")
print(f"  Low-performing sellers: {len(low_performers)}")

# =============================================================================
# SECTION 7: Summary Statistics
# =============================================================================
print("\n=== Section 7: Summary Statistics ===\n")

print("Overall Statistics for Closed Deal Sellers:")
print(f"  Total closed deals: {len(seller_performance)}")
print(f"  Sellers with orders: {(seller_performance['order_count'] > 0).sum()}")
print(f"  Sellers without orders: {(seller_performance['order_count'] == 0).sum()}")
print(f"  Total orders: {seller_performance['order_count'].sum():,.0f}")
print(f"  Total GMV: R$ {seller_performance['gmv'].sum():,.2f}")
print(f"  Average orders per seller: {seller_performance['order_count'].mean():.2f}")
print(f"  Average GMV per seller: R$ {seller_performance['gmv'].mean():.2f}")
print(f"  Average review score: {seller_performance['avg_review_score'].mean():.2f}")

print("\n" + "=" * 80)
print("ANALYSIS COMPLETE")
print("=" * 80)
print("\nGenerated files:")
print("  - outputs/tables/seller_performance_summary.csv")
print("  - outputs/tables/seller_performance_by_origin.csv")
print("  - outputs/tables/seller_performance_by_lead_type.csv")
print("  - outputs/tables/seller_performance_by_segment.csv")
print("  - outputs/tables/high_value_sellers.csv")
print("  - outputs/tables/low_performing_sellers.csv")
print("\n")