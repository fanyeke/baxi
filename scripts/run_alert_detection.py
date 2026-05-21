"""
Reads alert_rules.yml and daily_metrics.csv, detects anomalies based on rules,
and writes metric_alerts.csv (appends if exists, avoids duplicates by alert_id).
"""

import sys
import os
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from scripts.config import *
import pandas as pd
import yaml
import logging
import uuid
from datetime import datetime

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(message)s'
)
logger = logging.getLogger(__name__)


def load_alert_rules():
    """Load alert rules from YAML config."""
    with open(ALERT_RULES_FILE, 'r') as f:
        config = yaml.safe_load(f)
    return config['rules']


def load_daily_metrics():
    """Load daily metrics CSV, sorted by simulated_date."""
    df = pd.read_csv(DAILY_METRICS_FILE, parse_dates=['simulated_date'])
    df = df.sort_values('simulated_date').reset_index(drop=True)
    # Convert numeric columns, coerce errors to NaN
    numeric_cols = ['gmv', 'order_count', 'customer_count', 'seller_count',
                    'avg_order_value', 'freight_value', 'avg_review_score',
                    'low_review_rate', 'late_delivery_rate', 'cancel_rate',
                    'payment_installment_rate', 'marketing_seller_share']
    for col in numeric_cols:
        df[col] = pd.to_numeric(df[col], errors='coerce')
    return df


def get_rolling_avg(series, window):
    """Get the mean of the last `window` non-null values in a series."""
    values = series.dropna().tail(window)
    if len(values) < 1:
        return None
    return values.mean()


def check_gmv_drop(rule, metrics_df):
    """Check: current 7-day avg GMV vs previous 14-day avg GMV.
    Alert if current_7d_avg < prev_14d_avg * 0.85"""
    alerts = []
    gmv = metrics_df['gmv'].copy()

    if len(gmv.dropna()) < rule.get('min_sample_size', 1):
        logger.info(f"Rule {rule['rule_id']}: insufficient data ({len(gmv.dropna())} rows), skipping")
        return alerts

    # Evaluate from the last row backward
    current_7d = get_rolling_avg(gmv, 7)
    if current_7d is None or pd.isna(current_7d):
        logger.info(f"Rule {rule['rule_id']}: no valid current 7-day avg, skipping")
        return alerts

    # Get the values before the last 7 days for the previous 14-day baseline
    non_null_dates = gmv.dropna().index.tolist()
    if len(non_null_dates) <= 7:
        logger.info(f"Rule {rule['rule_id']}: not enough data for 7+14 comparison, skipping")
        return alerts

    last_7_indices = set(non_null_dates[-7:])
    prev_indices = [i for i in non_null_dates if i not in last_7_indices][-14:]
    prev_14d_avg = gmv.iloc[prev_indices].mean()

    if pd.isna(prev_14d_avg) or prev_14d_avg == 0:
        logger.info(f"Rule {rule['rule_id']}: invalid previous 14-day avg, skipping")
        return alerts

    threshold = prev_14d_avg * 0.85
    last_row = metrics_df.iloc[-1]
    real_run_date = last_row.get('real_run_date', '')

    if current_7d < threshold:
        alerts.append({
            'alert_id': uuid.uuid4().hex,
            'real_run_date': str(real_run_date),
            'simulated_date': str(last_row['simulated_date'].date()),
            'rule_id': rule['rule_id'],
            'metric': rule['metric'],
            'severity': rule['severity'],
            'dimension': rule['dimension'],
            'object_type': '',
            'object_id': '',
            'current_value': round(current_7d, 4),
            'baseline_value': round(prev_14d_avg, 4),
            'description': f"GMV 7-day avg ({current_7d:.2f}) dropped below threshold ({threshold:.2f}) vs previous 14-day avg ({prev_14d_avg:.2f})",
            'owner_role': rule['owner_role'],
            'status': 'new',
        })
    else:
        logger.info(f"Rule {rule['rule_id']}: no anomaly (current_7d={current_7d:.2f}, threshold={threshold:.2f})")

    return alerts


def check_late_delivery_spike(rule, metrics_df):
    """Check: late_delivery_rate > 0.25 and order_count >= 20."""
    alerts = []
    df = metrics_df.dropna(subset=['late_delivery_rate'], how='all')

    for _, row in df.iterrows():
        rate = row.get('late_delivery_rate')
        orders = row.get('order_count', 0)
        if pd.isna(rate) or pd.isna(orders):
            continue
        if rate > 0.25 and orders >= max(rule.get('min_sample_size', 20), 20):
            alerts.append({
                'alert_id': uuid.uuid4().hex,
                'real_run_date': str(row.get('real_run_date', '')),
                'simulated_date': str(row['simulated_date'].date()),
                'rule_id': rule['rule_id'],
                'metric': rule['metric'],
                'severity': rule['severity'],
                'dimension': rule['dimension'],
                'object_type': '',
                'object_id': '',
                'current_value': round(rate, 4),
                'baseline_value': 0.25,
                'description': f"Late delivery rate ({rate:.4f}) exceeds 25% with {int(orders)} orders",
                'owner_role': rule['owner_role'],
                'status': 'new',
            })

    if not alerts:
        logger.info(f"Rule {rule['rule_id']}: no anomaly detected")
    else:
        logger.info(f"Rule {rule['rule_id']}: {len(alerts)} alert(s) generated")

    return alerts


def check_review_score_drop(rule, metrics_df):
    """Check: baseline_7d_avg - current > 0.3 and order_count >= 30.
    Compares last row's avg_review_score against a 7-day rolling baseline (excluding last row)."""
    alerts = []
    score = metrics_df['avg_review_score'].copy()
    non_null = score.dropna()

    if len(non_null) < rule.get('min_sample_size', 30):
        logger.info(f"Rule {rule['rule_id']}: insufficient data ({len(non_null)} rows), skipping")
        return alerts

    last_row = metrics_df.iloc[-1]
    current_score = last_row.get('avg_review_score')
    current_orders = last_row.get('order_count', 0)

    if pd.isna(current_score) or pd.isna(current_orders):
        logger.info(f"Rule {rule['rule_id']}: last row has missing score or order_count, skipping")
        return alerts

    if current_orders < max(rule.get('min_sample_size', 30), 30):
        logger.info(f"Rule {rule['rule_id']}: insufficient order_count ({current_orders}), skipping")
        return alerts

    # 7-day baseline excluding last row
    scores_without_last = score.iloc[:-1].dropna().tail(7)
    if len(scores_without_last) < 3:
        logger.info(f"Rule {rule['rule_id']}: insufficient baseline data for comparison, skipping")
        return alerts

    baseline_avg = scores_without_last.mean()
    diff = baseline_avg - current_score

    if diff > 0.3:
        alerts.append({
            'alert_id': uuid.uuid4().hex,
            'real_run_date': str(last_row.get('real_run_date', '')),
            'simulated_date': str(last_row['simulated_date'].date()),
            'rule_id': rule['rule_id'],
            'metric': rule['metric'],
            'severity': rule['severity'],
            'dimension': rule['dimension'],
            'object_type': '',
            'object_id': '',
            'current_value': round(current_score, 4),
            'baseline_value': round(baseline_avg, 4),
            'description': f"Review score ({current_score:.2f}) dropped {diff:.2f} below baseline ({baseline_avg:.2f})",
            'owner_role': rule['owner_role'],
            'status': 'new',
        })
    else:
        logger.info(f"Rule {rule['rule_id']}: no anomaly (diff={diff:.4f}, threshold=0.3)")

    return alerts


def check_cancel_rate_spike(rule, metrics_df):
    """Check: cancel_rate change > 50% (vs 7-day baseline) AND current cancel_rate > 0.05."""
    alerts = []
    cancel = metrics_df['cancel_rate'].copy()
    non_null = cancel.dropna()

    if len(non_null) < rule.get('min_sample_size', 5):
        logger.info(f"Rule {rule['rule_id']}: insufficient data ({len(non_null)} rows), skipping")
        return alerts

    last_row = metrics_df.iloc[-1]
    current_rate = last_row.get('cancel_rate')

    if pd.isna(current_rate):
        logger.info(f"Rule {rule['rule_id']}: last row has missing cancel_rate, skipping")
        return alerts

    # 7-day baseline excluding last row
    cancel_without_last = cancel.iloc[:-1].dropna().tail(7)
    if len(cancel_without_last) < 2:
        logger.info(f"Rule {rule['rule_id']}: insufficient baseline data, skipping")
        return alerts

    baseline_rate = cancel_without_last.mean()

    if baseline_rate == 0:
        # If baseline is 0, any non-zero current is an infinite increase
        if current_rate > 0.05:
            change_rate = float('inf')
        else:
            change_rate = 0
    else:
        change_rate = abs(current_rate - baseline_rate) / baseline_rate

    if change_rate > 0.5 and current_rate > 0.05:
        alerts.append({
            'alert_id': uuid.uuid4().hex,
            'real_run_date': str(last_row.get('real_run_date', '')),
            'simulated_date': str(last_row['simulated_date'].date()),
            'rule_id': rule['rule_id'],
            'metric': rule['metric'],
            'severity': rule['severity'],
            'dimension': rule['dimension'],
            'object_type': '',
            'object_id': '',
            'current_value': round(current_rate, 4),
            'baseline_value': round(baseline_rate, 4),
            'description': f"Cancel rate ({current_rate:.4f}) changed {change_rate*100:.1f}% from baseline ({baseline_rate:.4f}), exceeds 50% threshold",
            'owner_role': rule['owner_role'],
            'status': 'new',
        })
    else:
        logger.info(f"Rule {rule['rule_id']}: no anomaly (change_rate={change_rate:.4f}, current={current_rate:.4f})")

    return alerts


def check_seller_activation_gap(rule, metrics_df):
    """Check: proportion of zero-order sellers > 0.5.
    
    NOTE: This rule requires per-seller order data which is not available in the
    aggregated daily_metrics.csv. Logged and skipped.
    """
    logger.info(f"Rule {rule['rule_id']}: requires per-seller order data not available in daily_metrics, skipping")
    return []


# Map rule_id to handler function
RULE_HANDLERS = {
    'gmv_drop': check_gmv_drop,
    'late_delivery_spike': check_late_delivery_spike,
    'review_score_drop': check_review_score_drop,
    'cancel_rate_spike': check_cancel_rate_spike,
    'seller_activation_gap': check_seller_activation_gap,
}


def append_alerts(alerts_df, output_path):
    if os.path.exists(output_path):
        existing = pd.read_csv(output_path, dtype={'alert_id': str})
        existing_keys = set(zip(existing['rule_id'], existing['simulated_date']))
        new_alerts = alerts_df[~alerts_df.apply(
            lambda r: (r['rule_id'], r['simulated_date']) in existing_keys, axis=1
        )]
        if len(new_alerts) == 0:
            logger.info("No new alerts to append")
            return existing
        combined = pd.concat([existing, new_alerts], ignore_index=True)
    else:
        combined = alerts_df

    combined.to_csv(output_path, index=False)
    logger.info(f"Wrote {len(combined)} total alert(s) to {output_path}")
    return combined


def main():
    logger.info("Starting alert detection")
    logger.info(f"Loading rules from: {ALERT_RULES_FILE}")
    rules = load_alert_rules()
    logger.info(f"Loaded {len(rules)} rule(s): {[r['rule_id'] for r in rules]}")

    logger.info(f"Loading daily metrics from: {DAILY_METRICS_FILE}")
    metrics_df = load_daily_metrics()
    logger.info(f"Loaded {len(metrics_df)} daily metric rows (date range: {metrics_df['simulated_date'].min().date()} to {metrics_df['simulated_date'].max().date()})")

    all_alerts = []
    for rule in rules:
        rule_id = rule['rule_id']
        handler = RULE_HANDLERS.get(rule_id)
        if handler is None:
            logger.warning(f"Unknown rule '{rule_id}', skipping")
            continue
        logger.info(f"Evaluating rule: {rule_id} ({rule['description']})")
        alerts = handler(rule, metrics_df)
        all_alerts.extend(alerts)

    if all_alerts:
        alerts_df = pd.DataFrame(all_alerts)
        # Ensure column order
        columns = ['alert_id', 'real_run_date', 'simulated_date', 'rule_id', 'metric',
                   'severity', 'dimension', 'object_type', 'object_id', 'current_value',
                   'baseline_value', 'description', 'owner_role', 'status']
        alerts_df = alerts_df[columns]
        append_alerts(alerts_df, METRIC_ALERTS_FILE)
    else:
        logger.info("No alerts detected")
        # Create empty file with headers if it doesn't exist
        if not os.path.exists(METRIC_ALERTS_FILE):
            pd.DataFrame(columns=['alert_id', 'real_run_date', 'simulated_date', 'rule_id',
                                  'metric', 'severity', 'dimension', 'object_type', 'object_id',
                                  'current_value', 'baseline_value', 'description', 'owner_role',
                                  'status']).to_csv(METRIC_ALERTS_FILE, index=False)

    logger.info("Alert detection complete")


if __name__ == '__main__':
    main()
