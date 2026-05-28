#!/usr/bin/env python3
import sqlite3, csv, os

DB_PATH = os.path.join(os.path.dirname(__file__), "../../data/olist_ops.db")
OUT_DIR = os.path.join(os.path.dirname(__file__), "../../migration_baseline/pipeline_outputs")
os.makedirs(OUT_DIR, exist_ok=True)

TABLES = [
    ("metric_daily", "metric_daily_sample.csv", None),
    ("metric_dimension_daily", "metric_dimension_daily_sample.csv", 1000),
    ("alert_events", "alert_events_sample.csv", None),
    ("strategy_recommendations", "recommendations_sample.csv", None),
    ("action_tasks", "tasks_sample.csv", None),
    ("event_outbox", "outbox_sample.csv", None),
    ("pipeline_runs", "pipeline_runs_sample.csv", None),
    ("ingestion_batches", "ingestion_batches_sample.csv", None),
]

conn = sqlite3.connect(DB_PATH)
conn.row_factory = sqlite3.Row

for table, filename, limit in TABLES:
    cursor = conn.execute(f"SELECT * FROM \"{table}\"")
    rows = cursor.fetchall()
    if not rows:
        print(f"SKIP {table}: empty")
        continue
    headers = [d[0] for d in cursor.description]
    outpath = os.path.join(OUT_DIR, filename)
    with open(outpath, "w", newline="") as f:
        writer = csv.writer(f)
        writer.writerow(headers)
        for i, row in enumerate(rows):
            if limit and i >= limit:
                break
            writer.writerow([row[h] for h in headers])
    print(f"OK {table}: {min(len(rows), limit or len(rows))} rows -> {filename}")

conn.close()
print(f"Pipeline samples exported to {OUT_DIR}")
