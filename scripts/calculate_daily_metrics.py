"""
Calculate 12 core daily metrics from intermediate tables.

Reads order_level_base.csv and item_level_base.csv from data/interim/,
computes daily metrics, filters to visible dates per ingestion_state.json,
and writes data/ads/daily_metrics.csv. Also adds a manifest entry.
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


def get_as_of_date():
    """Read ingestion_state.json and return the cutoff date for filtering.

    Priority: last_completed_simulated_date > next_simulated_date > current_simulated_date.
    If no valid date is found or state file doesn't exist, return None (no output).
    """
    if not os.path.exists(INGESTION_STATE_FILE):
        return None

    with open(INGESTION_STATE_FILE, 'r') as f:
        state = json.load(f)

    for key in ('last_completed_simulated_date', 'next_simulated_date', 'current_simulated_date'):
        val = state.get(key)
        if val is not None and val != 'null':
            return val

    return None


def filter_to_cutoff(metrics, as_of_date):
    if as_of_date is None:
        return metrics.iloc[0:0].copy()
    return metrics[metrics['simulated_date'] <= as_of_date].copy()


def add_manifest_entry(as_of_date, output_count):
    manifest_path = RUN_MANIFEST_FILE
    now = datetime.now().strftime('%Y-%m-%d %H:%M:%S')

    entry = pd.DataFrame([{
        'run_timestamp': now,
        'script': 'calculate_daily_metrics.py',
        'as_of_date': as_of_date,
        'rows_written': output_count,
        'status': 'success'
    }])

    if os.path.exists(manifest_path):
        existing = pd.read_csv(manifest_path)
        combined = pd.concat([existing, entry], ignore_index=True)
    else:
        combined = entry

    combined.to_csv(manifest_path, index=False)


def main():
    ensure_dirs_exist()

    as_of_date = get_as_of_date()
    print(f"as_of_date from state: {as_of_date}")

    print("Loading intermediate tables...")
    order_df, item_df = load_data()

    print("Preparing timestamps...")
    order_df, item_df = prepare_timestamps(order_df, item_df)

    print("Calculating daily metrics...")
    all_metrics = calculate_metrics(order_df, item_df)
    print(f"Total calculated: {len(all_metrics)} rows")

    visible_metrics = filter_to_cutoff(all_metrics, as_of_date)
    print(f"Visible after cut-off filter: {len(visible_metrics)} rows")

    print(f"Writing {len(visible_metrics)} rows to {DAILY_METRICS_FILE}...")
    visible_metrics.to_csv(DAILY_METRICS_FILE, index=False)

    if len(visible_metrics) > 0:
        print(f"Date range: {visible_metrics['simulated_date'].min()} to {visible_metrics['simulated_date'].max()}")
    else:
        print("No rows output (as_of_date is null or in the past).")

    print(f"Columns: {list(visible_metrics.columns)}")

    add_manifest_entry(as_of_date, len(visible_metrics))
    print(f"Manifest entry added.")

    print("Done.")


if __name__ == '__main__':
    main()
