#!/usr/bin/env python3
"""
db_calculate_metrics.py — Compute daily metrics from SQLite DWD tables.

Reads dwd_order_level and dwd_item_level, computes daily KPIs, writes metric_daily.
Supports --mode full (recompute all) and --mode range (recompute date range).
Does NOT write metric_dimension_daily in v0.2 (reserved for later).
"""

import os
import sys
import datetime
import sqlite3
import argparse

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import config


DB_PATH = os.path.join(config.PROJECT_ROOT, 'data', 'olist_ops.db')


def get_db_connection(db_path=None):
    if db_path is None:
        db_path = DB_PATH
    conn = sqlite3.connect(db_path)
    conn.execute("PRAGMA journal_mode=WAL")
    conn.execute("PRAGMA synchronous=NORMAL")
    return conn


def calculate_metrics(conn, mode, date_start=None, date_end=None):
    """Calculate daily metrics from DWD tables."""
    now = datetime.datetime.now().isoformat()

    # Determine date range for calculation
    if mode == 'full':
        cur = conn.execute("SELECT MIN(purchase_date), MAX(purchase_date) FROM dwd_order_level WHERE purchase_date IS NOT NULL")
        row = cur.fetchone()
        date_min, date_max = row[0], row[1]
        # Clear existing metric_daily
        conn.execute("DELETE FROM metric_daily")
    elif mode == 'range':
        date_min = date_start
        date_max = date_end
        # Clear existing for the range
        conn.execute("DELETE FROM metric_daily WHERE metric_date BETWEEN ? AND ?", (date_start, date_end))
    else:
        raise ValueError(f"Unknown mode: {mode}")

    print(f"[db_metrics] Computing metrics for {date_min} to {date_max}")

    # Main metrics query: aggregate order-level metrics by purchase_date
    query = """
    SELECT
        o.purchase_date AS metric_date,
        COALESCE(SUM(o.payment_value), 0) AS gmv,
        COUNT(DISTINCT o.order_id) AS order_count,
        COUNT(DISTINCT o.customer_unique_id) AS customer_count,
        COALESCE(SUM(o.payment_value) / NULLIF(COUNT(DISTINCT o.order_id), 0), 0) AS avg_order_value,
        COALESCE(
            (SELECT SUM(i.freight_value) FROM dwd_item_level i 
             JOIN dwd_order_level o2 ON i.order_id = o2.order_id 
             WHERE o2.purchase_date = o.purchase_date), 
            0
        ) AS freight_value,
        COALESCE(AVG(o.review_score), 0) AS avg_review_score,
        COALESCE(
            CAST(SUM(CASE WHEN o.review_score IS NOT NULL AND o.review_score <= 2 THEN 1 ELSE 0 END) AS REAL)
            / NULLIF(COUNT(CASE WHEN o.review_score IS NOT NULL THEN 1 END), 0), 
            0
        ) AS low_review_rate,
        COALESCE(
            CAST(SUM(CASE WHEN o.order_status = 'delivered' AND o.is_late = 1 THEN 1 ELSE 0 END) AS REAL)
            / NULLIF(COUNT(CASE WHEN o.order_status = 'delivered' THEN 1 END), 0), 
            0
        ) AS late_delivery_rate,
        COALESCE(
            CAST(SUM(CASE WHEN o.is_cancelled = 1 THEN 1 ELSE 0 END) AS REAL)
            / NULLIF(COUNT(o.order_id), 0), 
            0
        ) AS cancel_rate,
        COALESCE(
            CAST(SUM(CASE WHEN o.payment_installments > 1 THEN 1 ELSE 0 END) AS REAL)
            / NULLIF(COUNT(o.order_id), 0), 
            0
        ) AS payment_installment_rate
    FROM dwd_order_level o
    WHERE o.purchase_date IS NOT NULL
      AND o.purchase_date BETWEEN ? AND ?
    GROUP BY o.purchase_date
    ORDER BY o.purchase_date
    """

    # Get date range for seller count (requires separate subquery per date)
    cur = conn.execute(query, (date_min, date_max))
    rows = cur.fetchall()

    # Get seller counts per date (separate query for efficiency)
    seller_query = """
    SELECT o.purchase_date, COUNT(DISTINCT i.seller_id) as seller_count
    FROM dwd_order_level o
    JOIN dwd_item_level i ON i.order_id = o.order_id
    WHERE o.purchase_date IS NOT NULL
      AND o.purchase_date BETWEEN ? AND ?
    GROUP BY o.purchase_date
    """
    seller_cur = conn.execute(seller_query, (date_min, date_max))
    seller_counts = {row[0]: row[1] for row in seller_cur.fetchall()}

    count = 0
    for row in rows:
        metric_date, gmv, order_count, customer_count, avg_order_value, freight_value, \
            avg_review_score, low_review_rate, late_delivery_rate, cancel_rate, \
            payment_installment_rate = row

        seller_count = seller_counts.get(metric_date, 0)

        conn.execute("""
            INSERT OR REPLACE INTO metric_daily
            (metric_date, gmv, order_count, customer_count, seller_count,
             avg_order_value, freight_value, avg_review_score,
             low_review_rate, late_delivery_rate, cancel_rate,
             payment_installment_rate, marketing_seller_share, created_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        """, (
            metric_date,
            round(gmv, 2) if gmv else 0,
            order_count,
            customer_count,
            seller_count,
            round(avg_order_value, 2) if avg_order_value else 0,
            round(freight_value, 2) if freight_value else 0,
            round(avg_review_score, 4) if avg_review_score else 0,
            round(low_review_rate, 4),
            round(late_delivery_rate, 4),
            round(cancel_rate, 4),
            round(payment_installment_rate, 4),
            0,  # marketing_seller_share — needs marketing dataset join
            now
        ))
        count += 1

    print(f"[db_metrics] Computed {count} daily metric rows")
    return count


def main():
    parser = argparse.ArgumentParser(description='Calculate daily metrics from SQLite DWD')
    parser.add_argument('--mode', required=True, choices=['full', 'range'],
                        help='Calculation mode: full or range')
    parser.add_argument('--start', default=None, help='Start date for range mode')
    parser.add_argument('--end', default=None, help='End date for range mode')
    parser.add_argument('--db', default=DB_PATH, help='Path to SQLite database')
    args = parser.parse_args()

    if args.mode == 'range' and (not args.start or not args.end):
        print("[db_metrics] ERROR: range mode requires --start and --end")
        sys.exit(1)

    conn = get_db_connection(args.db)
    try:
        count = calculate_metrics(conn, args.mode, args.start, args.end)
        conn.commit()
        print(f"[db_metrics] Done. {count} rows in metric_daily")
    except Exception as e:
        conn.rollback()
        print(f"[db_metrics] FAILED: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc()
        sys.exit(1)
    finally:
        conn.close()


if __name__ == '__main__':
    main()
