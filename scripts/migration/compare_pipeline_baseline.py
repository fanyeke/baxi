#!/usr/bin/env python3
"""Compare Go pipeline output against v0.5.3 baseline.

Reads baseline table counts from migration_baseline/table_counts.json and
queries PostgreSQL for actual row counts. Reports PASS/FAIL per table and
exits 0 if all pass, 1 if any fail.

Usage:
    DATABASE_URL=postgres://user:pass@host:5432/dbname ./compare_pipeline_baseline.py

Optional CSV sample comparison:
    ./compare_pipeline_baseline.py --check-samples
"""
import argparse
import csv
import json
import os
import sys

import psycopg2


TABLE_MAP = [
    ("dwd_order_level",          "dwd.order_level",          99441),
    ("dwd_item_level",           "dwd.item_level",           112650),
    ("metric_daily",             "mart.metric_daily",        634),
    ("metric_dimension_daily",   "mart.metric_dimension_daily", 693602),
    ("alert_events",             "ops.metric_alert",         36),
    ("strategy_recommendations", "ops.recommendation",       36),
    ("action_tasks",             "ops.task",                 36),
    ("event_outbox",             "ops.outbox_event",         36),
]

SAMPLE_DIR = os.path.join(
    os.path.dirname(__file__), "../..", "migration_baseline", "pipeline_outputs"
)
BASELINE_COUNTS_PATH = os.path.join(
    os.path.dirname(__file__), "../..", "migration_baseline", "table_counts.json"
)


def read_baseline_counts():
    """Read baseline table counts from JSON file."""
    with open(BASELINE_COUNTS_PATH) as f:
        return json.load(f)


def check_row_counts(conn, baseline_counts):
    """Query PostgreSQL row counts and compare against expected values."""
    print(f"{'TABLE':<40} {'OLD':<10} {'NEW':<10} {'EXPECTED':<10} {'STATUS':<10}")
    print("-" * 80)

    all_pass = True
    for key, table, expected in TABLE_MAP:
        cur = conn.cursor()
        try:
            cur.execute(f"SELECT COUNT(*) FROM {table}")
            new_count = cur.fetchone()[0]
        except Exception as e:
            print(f"{table:<40} {'ERR':<10} {'ERR':<10} {'ERR':<10} FAIL")
            print(f"  → Error querying {table}: {e}")
            all_pass = False
            continue
        finally:
            cur.close()

        old_count = baseline_counts.get(key, "N/A")
        old_display = old_count if isinstance(old_count, int) else "N/A"
        status = "PASS" if new_count == expected else "FAIL"
        if status == "FAIL":
            all_pass = False

        print(f"{table:<40} {str(old_display):<10} {str(new_count):<10} {str(expected):<10} {status:<10}")

    print()
    return all_pass


def check_csv_samples(conn):
    """Optional: compare row counts from sample CSVs vs PostgreSQL."""
    if not os.path.isdir(SAMPLE_DIR):
        print("(no sample CSV directory found, skipping CSV sample check)")
        return True

    csv_map = {
        "alert_events": "alert_events_sample.csv",
        "dwd_order_level": None,
        "dwd_item_level": None,
        "metric_daily": "metric_daily_sample.csv",
        "metric_dimension_daily": "metric_dimension_daily_sample.csv",
        "strategy_recommendations": "recommendations_sample.csv",
        "action_tasks": "tasks_sample.csv",
        "event_outbox": "outbox_sample.csv",
    }

    all_pass = True
    for key, table, _ in TABLE_MAP:
        csv_file = csv_map.get(key)
        if not csv_file:
            continue

        csv_path = os.path.join(SAMPLE_DIR, csv_file)
        if not os.path.isfile(csv_path):
            continue

        with open(csv_path) as f:
            reader = csv.reader(f)
            csv_count = sum(1 for _ in reader) - 1

        cur = conn.cursor()
        try:
            cur.execute(f"SELECT COUNT(*) FROM {table}")
            pg_count = cur.fetchone()[0]
        except Exception as e:
            print(f"  [CSV] {table}: PG query error: {e}")
            cur.close()
            all_pass = False
            continue
        cur.close()

        match = csv_count == pg_count
        if not match:
            print(f"  [CSV] {table}: CSV rows={csv_count} ≠ PG rows={pg_count} FAIL")
            all_pass = False

    if all_pass:
        print("  CSV sample counts all match (for available files)")

    return all_pass


def main():
    parser = argparse.ArgumentParser(
        description="Compare Go pipeline output against v0.5.3 baseline"
    )
    parser.add_argument(
        "--check-samples",
        action="store_true",
        help="Also compare sample CSV row counts against PostgreSQL",
    )
    args = parser.parse_args()

    baseline_counts = read_baseline_counts()
    print(f"Loaded baseline counts for {len(baseline_counts)} tables\n")

    database_url = os.environ.get("DATABASE_URL")
    if not database_url:
        print("FATAL: DATABASE_URL environment variable is not set")
        print("Usage: DATABASE_URL=postgres://user:pass@host:5432/dbname ./compare_pipeline_baseline.py")
        sys.exit(1)

    conn = psycopg2.connect(database_url)

    counts_ok = check_row_counts(conn, baseline_counts)

    samples_ok = True
    if args.check_samples:
        print("--- CSV Sample Check ---")
        samples_ok = check_csv_samples(conn)

    conn.close()

    if counts_ok and samples_ok:
        print("All checks PASSED")
        sys.exit(0)
    else:
        print("Some checks FAILED")
        sys.exit(1)


if __name__ == "__main__":
    main()
