#!/usr/bin/env python3
"""
db_status.py — Report status of SQLite decision backend database.

Reports row counts for all 12 tables and last pipeline_run/ingestion_batch details.
Read-only: does not modify any data.
"""

import os
import sys
import sqlite3

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
import config
from services.status_service import get_last_pipeline_run, get_last_ingestion_batch

from services.db_service import get_db, get_table_counts, db_exists


DB_PATH = os.path.join(config.PROJECT_ROOT, 'data', 'olist_ops.db')

ALL_TABLES = [
    'pipeline_runs', 'ingestion_batches',
    'dwd_order_level', 'dwd_item_level',
    'metric_daily', 'metric_dimension_daily',
    'alert_events', 'strategy_recommendations',
    'action_tasks', 'review_retro',
    'event_outbox', 'qoder_jobs',
]


def report_status(db_path=None):
    if db_path is None:
        db_path = DB_PATH
    if not db_exists(db_path):
        print(f"[db_status] Database not found: {db_path}")
        print("Run 'python3 scripts/db_init.py' first.")
        return

    conn = get_db(db_path)
    try:
        print("=" * 60)
        print("Olist Operations DB — Status Report")
        print("=" * 60)
        print()

        # Table counts
        counts = get_table_counts(conn)
        print("Table Row Counts:")
        print("-" * 40)
        for table in ALL_TABLES:
            count = counts.get(table, 0)
            print(f"  {table:<32s} {count:>8,}")
        print()

        # Last pipeline run
        row = get_last_pipeline_run(conn)
        if row:
            print("Last Pipeline Run:")
            print("-" * 40)
            print(f"  Run ID:     {row['run_id']}")
            print(f"  Type:       {row['run_type']}")
            print(f"  Mode:       {row['mode']}")
            print(f"  Status:     {row['status']}")
            print(f"  Started:    {row['started_at']}")
            print(f"  Finished:   {row['finished_at']}")
            print(f"  Input:      {row['input_count']}  Output: {row['output_count']}")
            if row['error_message']:
                print(f"  Error:      {row['error_message']}")
            print()

        # Last ingestion batch
        row = get_last_ingestion_batch(conn)
        if row:
            print("Last Ingestion Batch:")
            print("-" * 40)
            print(f"  Batch ID:   {row['batch_id']}")
            print(f"  Source:     {row['source_name']}")
            print(f"  Mode:       {row['ingestion_mode']}")
            print(f"  Date Range: {row['date_start']} to {row['date_end']}")
            print(f"  Row Count:  {row['row_count']}")
            print(f"  Status:     {row['status']}")
            print(f"  Created:    {row['created_at']}")
            print()

        # Pending trigger events
        cur = conn.execute("SELECT target_channel, COUNT(*) FROM event_outbox WHERE status='pending' GROUP BY target_channel")
        pending = cur.fetchall()
        if pending:
            print("Pending Trigger Events:")
            print("-" * 40)
            for row in pending:
                print(f"  {row[0]:<20s} {row[1]:>6,}")
            print()

        # Metric date range
        cur = conn.execute("SELECT MIN(metric_date), MAX(metric_date), COUNT(*) FROM metric_daily")
        row = cur.fetchone()
        if row and row[0]:
            print("Daily Metrics:")
            print("-" * 40)
            print(f"  Date Range: {row[0]} to {row[1]}")
            print(f"  Total Days: {row[2]}")
            print()

        print("=" * 60)

    finally:
        conn.close()


def main():
    report_status()


if __name__ == '__main__':
    main()
