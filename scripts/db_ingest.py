#!/usr/bin/env python3
"""
db_ingest.py — Ingest CSV data into SQLite decision backend.

Supports 3 ingestion modes:
  full:  truncate dwd_order_level and dwd_item_level, then import all CSV data
  range: import only rows where purchase_date is BETWEEN start AND end
  date:  import only rows where purchase_date equals target date

Records ingestion in ingestion_batches and pipeline_runs tables.
Generates deterministic batch_id as UUID4.
"""

import os
import sys
import csv
import uuid
import datetime
import sqlite3
import argparse

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import config


DB_PATH = os.path.join(config.PROJECT_ROOT, 'data', 'olist_ops.db')
ORDER_CSV = config.ORDER_LEVEL_BASE_FILE
ITEM_CSV = config.ITEM_LEVEL_BASE_FILE


def safe_int(val):
    if not val:
        return None
    try:
        return int(float(val))
    except (ValueError, TypeError):
        return None


def safe_float(val):
    if not val:
        return None
    try:
        return float(val)
    except (ValueError, TypeError):
        return None


def parse_timestamp_to_date(ts):
    """Extract date portion from timestamp string like '2017-11-24 12:34:56'."""
    if not ts or ts == '':
        return None
    return ts.split(' ')[0]


def compute_delivery_days(delivered_date, purchase_ts):
    """Compute delivery days between purchase and delivery."""
    if not delivered_date or not purchase_ts:
        return None
    try:
        d_date = datetime.datetime.strptime(delivered_date.split(' ')[0], '%Y-%m-%d')
        p_date = datetime.datetime.strptime(purchase_ts.split(' ')[0], '%Y-%m-%d')
        return (d_date - p_date).days
    except (ValueError, IndexError):
        return None


def compute_delay_days(delivered_date, estimated_date):
    """Compute delay days = actual delivery - estimated delivery (positive = late)."""
    if not delivered_date or not estimated_date:
        return None
    try:
        d_date = datetime.datetime.strptime(delivered_date.split(' ')[0], '%Y-%m-%d')
        e_date = datetime.datetime.strptime(estimated_date.split(' ')[0], '%Y-%m-%d')
        return (d_date - e_date).days
    except (ValueError, IndexError):
        return None


def is_late_flag(delivered_date, estimated_date):
    """Check if order was delivered after estimated date."""
    delay = compute_delay_days(delivered_date, estimated_date)
    if delay is None:
        return 0
    return 1 if delay > 0 else 0


def is_cancelled_flag(status):
    """Check if order was cancelled."""
    return 1 if status and status.lower() in ('cancelled', 'canceled', 'unavailable') else 0


def get_db_connection(db_path=None):
    """Get database connection with WAL mode."""
    if db_path is None:
        db_path = DB_PATH
    conn = sqlite3.connect(db_path)
    conn.execute("PRAGMA journal_mode=WAL")
    conn.execute("PRAGMA synchronous=NORMAL")
    return conn


def record_pipeline_run(conn, run_id, run_type, mode, status, started_at,
                        finished_at=None, input_count=0, output_count=0,
                        error_message=None):
    """Record a pipeline run entry."""
    conn.execute("""
        INSERT OR REPLACE INTO pipeline_runs 
        (run_id, run_type, mode, status, started_at, finished_at, 
         input_count, output_count, error_message)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
    """, (run_id, run_type, mode, status, started_at,
          finished_at or datetime.datetime.now().isoformat(),
          input_count, output_count, error_message))


def record_ingestion_batch(conn, batch_id, source_name, ingestion_mode,
                           date_start=None, date_end=None, source_file=None,
                           row_count=0, status='completed', created_at=None):
    """Record an ingestion batch entry."""
    if created_at is None:
        created_at = datetime.datetime.now().isoformat()
    conn.execute("""
        INSERT OR REPLACE INTO ingestion_batches
        (batch_id, source_name, ingestion_mode, date_start, date_end,
         source_file, row_count, status, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
    """, (batch_id, source_name, ingestion_mode, date_start, date_end,
          source_file, row_count, status, created_at))


def ingest_orders(conn, batch_id, mode, date_start=None, date_end=None, date_single=None):
    """Ingest order-level data from CSV into dwd_order_level."""
    order_count = 0
    now = datetime.datetime.now().isoformat()

    if mode == 'full':
        conn.execute("DELETE FROM dwd_order_level")

    with open(ORDER_CSV, 'r', encoding='utf-8') as f:
        reader = csv.DictReader(f)
        rows = []
        for row in reader:
            purchase_date = parse_timestamp_to_date(row.get('order_purchase_timestamp', ''))
            if not purchase_date:
                continue  # Skip rows without valid date

            if mode == 'range':
                if purchase_date < date_start or purchase_date > date_end:
                    continue
            elif mode == 'date':
                if purchase_date != date_single:
                    continue

            delivered_date = row.get('order_delivered_customer_date', '')
            estimated_date = row.get('order_estimated_delivery_date', '')

            row_tuple = (
                row.get('order_id'),
                row.get('customer_id'),
                row.get('customer_unique_id'),
                row.get('order_status'),
                row.get('order_purchase_timestamp'),
                purchase_date,
                row.get('customer_state'),
                row.get('primary_payment_type'),
                safe_int(row.get('max_installments')),
                safe_float(row.get('total_payment_value')),
                safe_float(row.get('review_score')),
                delivered_date,
                row.get('order_estimated_delivery_date'),
                compute_delivery_days(delivered_date, row.get('order_purchase_timestamp', '')),
                compute_delay_days(delivered_date, estimated_date),
                is_late_flag(delivered_date, estimated_date),
                is_cancelled_flag(row.get('order_status')),
                batch_id,
                now
            )
            rows.append(row_tuple)
            order_count += 1

        conn.executemany("""
            INSERT OR REPLACE INTO dwd_order_level
            (order_id, customer_id, customer_unique_id, order_status,
             order_purchase_timestamp, purchase_date, customer_state,
             payment_type, payment_installments, payment_value,
             review_score, delivered_customer_date, estimated_delivery_date,
             delivery_days, delay_days, is_late, is_cancelled,
             ingestion_batch_id, loaded_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        """, rows)

    return order_count


def ingest_items(conn, batch_id, mode, date_start=None, date_end=None, date_single=None):
    """Ingest item-level data from CSV into dwd_item_level."""
    item_count = 0
    now = datetime.datetime.now().isoformat()

    if mode == 'full':
        conn.execute("DELETE FROM dwd_item_level")

    with open(ITEM_CSV, 'r', encoding='utf-8') as f:
        reader = csv.DictReader(f)
        rows = []
        # Load order purchase dates for filtering in non-full modes
        order_dates = {}
        if mode != 'full':
            cursor = conn.execute("SELECT order_id, purchase_date FROM dwd_order_level")
            for oid, pd in cursor:
                order_dates[oid] = pd

        for row in reader:
            order_id = row.get('order_id', '')

            if mode != 'full':
                pdate = order_dates.get(order_id)
                if not pdate:
                    continue
                if mode == 'range':
                    if pdate < date_start or pdate > date_end:
                        continue
                elif mode == 'date':
                    if pdate != date_single:
                        continue

            item_key = f"{order_id}_{row.get('order_item_id')}"

            row_tuple = (
                item_key,
                order_id,
                safe_int(row.get('order_item_id')),
                row.get('product_id'),
                row.get('seller_id'),
                row.get('product_category_name'),
                row.get('product_category_name_english'),
                row.get('seller_state'),
                safe_float(row.get('price')),
                safe_float(row.get('freight_value')),
                batch_id,
                now
            )
            rows.append(row_tuple)
            item_count += 1

        conn.executemany("""
            INSERT OR REPLACE INTO dwd_item_level
            (item_key, order_id, order_item_id, product_id, seller_id,
             product_category_name, product_category_name_english, seller_state,
             price, freight_value, ingestion_batch_id, loaded_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        """, rows)

    return item_count


def main():
    parser = argparse.ArgumentParser(description='Ingest CSV data into SQLite decision backend')
    parser.add_argument('--mode', required=True, choices=['full', 'range', 'date'],
                        help='Ingestion mode: full, range, or date')
    parser.add_argument('--start', default=None, help='Start date for range mode (YYYY-MM-DD)')
    parser.add_argument('--end', default=None, help='End date for range mode (YYYY-MM-DD)')
    parser.add_argument('--date', default=None, help='Single date for date mode (YYYY-MM-DD)')
    parser.add_argument('--db', default=DB_PATH, help='Path to SQLite database')
    args = parser.parse_args()

    started_at = datetime.datetime.now().isoformat()
    run_id = f"ingest-{args.mode}-{started_at}"[:32]
    batch_id = str(uuid.uuid4())

    print(f"[db_ingest] Started at {started_at}")
    print(f"[db_ingest] Mode: {args.mode}")
    if args.mode == 'range':
        print(f"[db_ingest] Range: {args.start} to {args.end}")
    if args.mode == 'date':
        print(f"[db_ingest] Date: {args.date}")

    conn = get_db_connection(args.db)
    try:
        # Record pipeline run start
        record_pipeline_run(conn, run_id, 'ingestion', args.mode, 'running', started_at)
        record_ingestion_batch(conn, batch_id, 'olist_orders', args.mode,
                               args.start, args.end,
                               f"{ORDER_CSV};{ITEM_CSV}",
                               status='running',
                               created_at=started_at)

        # Ingest data
        order_count = ingest_orders(conn, batch_id, args.mode, args.start, args.end, args.date)
        item_count = ingest_items(conn, batch_id, args.mode, args.start, args.end, args.date)

        conn.commit()

        # Update pipeline run
        finished_at = datetime.datetime.now().isoformat()
        record_pipeline_run(conn, f"{run_id}-done", 'ingestion', args.mode, 'completed',
                            started_at, finished_at, order_count + item_count,
                            order_count + item_count)
        record_ingestion_batch(conn, batch_id, 'olist_orders', args.mode,
                               args.start, args.end,
                               f"{ORDER_CSV};{ITEM_CSV}",
                               order_count + item_count,
                               status='completed',
                               created_at=started_at)

        conn.commit()

        print(f"[db_ingest] Done. Orders: {order_count}, Items: {item_count}")
        print(f"[db_ingest] Batch ID: {batch_id}")

    except Exception as e:
        conn.rollback()
        finished_at = datetime.datetime.now().isoformat()
        record_pipeline_run(conn, run_id, 'ingestion', args.mode, 'failed',
                            started_at, finished_at, 0, 0, str(e))
        conn.commit()
        print(f"[db_ingest] FAILED: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc()
        sys.exit(1)
    finally:
        conn.close()


if __name__ == '__main__':
    main()
