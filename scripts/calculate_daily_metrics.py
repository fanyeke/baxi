"""
Calculate 12 core daily metrics from intermediate tables.

Reads order_level_base.csv and item_level_base.csv from data/interim/,
computes daily metrics, and writes data/ads/daily_metrics.csv.
"""

import sys
import os
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from scripts.config import *
import pandas as pd
import json
from datetime import datetime


def load_data():
    """Load intermediate tables."""
    order_df = pd.read_csv(ORDER_LEVEL_BASE_FILE)
    item_df = pd.read_csv(ITEM_LEVEL_BASE_FILE)
    return order_df, item_df


def prepare_timestamps(order_df, item_df):
    """Convert timestamp columns to datetime and extract date."""
    order_df['order_purchase_timestamp'] = pd.to_datetime(order_df['order_purchase_timestamp'])
    order_df['simulated_date'] = order_df['order_purchase_timestamp'].dt.strftime('%Y-%m-%d')

    item_df['order_purchase_timestamp'] = pd.to_datetime(item_df['order_purchase_timestamp'])
    item_df['simulated_date'] = item_df['order_purchase_timestamp'].dt.strftime('%Y-%m-%d')

    order_df['order_delivered_customer_date'] = pd.to_datetime(order_df['order_delivered_customer_date'], errors='coerce')
    order_df['order_estimated_delivery_date'] = pd.to_datetime(order_df['order_estimated_delivery_date'], errors='coerce')

    item_df['order_delivered_customer_date'] = pd.to_datetime(item_df['order_delivered_customer_date'], errors='coerce')
    item_df['order_estimated_delivery_date'] = pd.to_datetime(item_df['order_estimated_delivery_date'], errors='coerce')

    return order_df, item_df


def calculate_metrics(order_df, item_df):
    """Calculate 12 core daily metrics."""
    real_run_date = datetime.now().strftime('%Y-%m-%d')

    # Aggregate by simulated_date from item_level_base
    item_daily = item_df.groupby('simulated_date').agg(
        gmv=('price', 'sum'),
        seller_count=('seller_id', 'nunique'),
        freight_value=('freight_value', 'sum')
    ).reset_index()

    # Aggregate by simulated_date from order_level_base
    order_daily = order_df.groupby('simulated_date').agg(
        order_count=('order_id', 'nunique'),
        customer_count=('customer_unique_id', 'nunique')
    ).reset_index()

    # avg_review_score and low_review_rate from order_level_base
    review_mask = order_df['review_score'].notna()
    review_daily = order_df[review_mask].groupby('simulated_date').agg(
        avg_review_score=('review_score', 'mean'),
        total_reviews=('review_score', 'count'),
        low_reviews=('review_score', lambda x: (x <= 2).sum())
    ).reset_index()
    review_daily['low_review_rate'] = review_daily['low_reviews'] / review_daily['total_reviews']

    # late_delivery_rate from order_level_base (delivered orders only)
    delivered_mask = (order_df['order_status'] == 'delivered') & order_df['order_delivered_customer_date'].notna() & order_df['order_estimated_delivery_date'].notna()
    delivered_df = order_df[delivered_mask].copy()
    delivered_df['is_late'] = delivered_df['order_delivered_customer_date'] > delivered_df['order_estimated_delivery_date']
    late_daily = delivered_df.groupby('simulated_date').agg(
        total_delivered=('order_id', 'count'),
        late_deliveries=('is_late', 'sum')
    ).reset_index()
    late_daily['late_delivery_rate'] = late_daily['late_deliveries'] / late_daily['total_delivered']

    # cancel_rate from order_level_base
    cancel_daily = order_df.groupby('simulated_date').agg(
        total_orders=('order_id', 'count'),
        canceled_orders=('order_status', lambda x: (x == 'canceled').sum())
    ).reset_index()
    cancel_daily['cancel_rate'] = cancel_daily['canceled_orders'] / cancel_daily['total_orders']

    # payment_installment_rate from order_level_base
    installment_daily = order_df.groupby('simulated_date').agg(
        total_payments=('order_id', 'count'),
        installment_payments=('max_installments', lambda x: (x > 1).sum())
    ).reset_index()
    installment_daily['payment_installment_rate'] = installment_daily['installment_payments'] / installment_daily['total_payments']

    # Merge all metrics
    metrics = item_daily.merge(order_daily, on='simulated_date', how='outer')
    metrics = metrics.merge(review_daily[['simulated_date', 'avg_review_score', 'low_review_rate']], on='simulated_date', how='left')
    metrics = metrics.merge(late_daily[['simulated_date', 'late_delivery_rate']], on='simulated_date', how='left')
    metrics = metrics.merge(cancel_daily[['simulated_date', 'cancel_rate']], on='simulated_date', how='left')
    metrics = metrics.merge(installment_daily[['simulated_date', 'payment_installment_rate']], on='simulated_date', how='left')

    # Derived metrics
    metrics['avg_order_value'] = metrics['gmv'] / metrics['order_count']
    metrics['marketing_seller_share'] = 0.0
    metrics['real_run_date'] = real_run_date

    # Select and order columns
    output_columns = [
        'simulated_date', 'real_run_date', 'gmv', 'order_count',
        'customer_count', 'seller_count', 'avg_order_value', 'freight_value',
        'avg_review_score', 'low_review_rate', 'late_delivery_rate',
        'cancel_rate', 'payment_installment_rate', 'marketing_seller_share'
    ]
    metrics = metrics[output_columns]
    metrics = metrics.sort_values('simulated_date').reset_index(drop=True)

    return metrics


def main():
    ensure_dirs_exist()
    print("Loading intermediate tables...")
    order_df, item_df = load_data()

    print("Preparing timestamps...")
    order_df, item_df = prepare_timestamps(order_df, item_df)

    print("Calculating daily metrics...")
    metrics = calculate_metrics(order_df, item_df)

    print(f"Writing {len(metrics)} rows to {DAILY_METRICS_FILE}...")
    metrics.to_csv(DAILY_METRICS_FILE, index=False)

    print(f"Columns: {list(metrics.columns)}")
    print(f"Date range: {metrics['simulated_date'].min()} to {metrics['simulated_date'].max()}")
    print("Done.")


if __name__ == '__main__':
    main()
