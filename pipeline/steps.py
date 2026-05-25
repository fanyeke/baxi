"""pipeline/steps.py — Direct function wrappers for each pipeline step.

Replaces subprocess.run() calls with direct Python function calls.
Each step function mirrors the behavior of the corresponding script's main().
"""

import os
import sys
import csv
import json
import datetime
import sqlite3

# Ensure scripts directory is on sys.path for importing db_* modules
_PIPELINE_DIR = os.path.dirname(os.path.abspath(__file__))
_PROJECT_ROOT = os.path.dirname(_PIPELINE_DIR)
_SCRIPTS_DIR = os.path.join(_PROJECT_ROOT, 'scripts')
if _SCRIPTS_DIR not in sys.path:
    sys.path.insert(0, _SCRIPTS_DIR)

# Also ensure project root is on path (some scripts import core.config)
if _PROJECT_ROOT not in sys.path:
    sys.path.insert(0, _PROJECT_ROOT)

import config
import db_init
import db_ingest
import db_calculate_metrics
import db_calculate_dimension_metrics
import db_rule_engine
import db_dimensional_rule_engine
import db_generate_recommendations
import db_export_feishu
import db_trigger_simulator

DB_PATH = os.path.join(config.PROJECT_ROOT, 'data', 'olist_ops.db')


def _get_db_path(db_path=None):
    return db_path if db_path else DB_PATH


def step_db_init(db_path=None):
    """Step: Initialize database schema and indexes."""
    db_path = _get_db_path(db_path)
    print(f"[step] db_init: {db_path}")
    tables = db_init.init_database(db_path)
    print(f"[step] db_init done: {len(tables)} tables ready")
    return True


def step_db_ingest(mode, start=None, end=None, db_path=None):
    """Step: Ingest CSV data into dwd_order_level and dwd_item_level."""
    db_path = _get_db_path(db_path)
    started_at = datetime.datetime.now().isoformat()
    run_id = f"ingest-{mode}-{started_at}"[:32]
    batch_id = str(__import__('uuid').uuid4())

    print(f"[step] db_ingest: mode={mode}, start={start}, end={end}")

    conn = db_ingest.get_db_connection(db_path)
    try:
        # Record pipeline run start
        db_ingest.record_pipeline_run(conn, run_id, 'ingestion', mode, 'running', started_at)
        db_ingest.record_ingestion_batch(
            conn, batch_id, 'olist_orders', mode, start, end,
            f"{db_ingest.ORDER_CSV};{db_ingest.ITEM_CSV}",
            status='running', created_at=started_at
        )

        # Ingest data
        order_count = db_ingest.ingest_orders(conn, batch_id, mode, start, end, None)
        item_count = db_ingest.ingest_items(conn, batch_id, mode, start, end, None)

        conn.commit()

        # Update pipeline run completion
        finished_at = datetime.datetime.now().isoformat()
        db_ingest.record_pipeline_run(
            conn, f"{run_id}-done", 'ingestion', mode, 'completed',
            started_at, finished_at, order_count + item_count, order_count + item_count
        )
        db_ingest.record_ingestion_batch(
            conn, batch_id, 'olist_orders', mode, start, end,
            f"{db_ingest.ORDER_CSV};{db_ingest.ITEM_CSV}",
            order_count + item_count, status='completed', created_at=started_at
        )
        conn.commit()

        print(f"[step] db_ingest done: orders={order_count}, items={item_count}, batch={batch_id}")
        return True
    except Exception as e:
        conn.rollback()
        finished_at = datetime.datetime.now().isoformat()
        db_ingest.record_pipeline_run(
            conn, run_id, 'ingestion', mode, 'failed',
            started_at, finished_at, 0, 0, str(e)
        )
        conn.commit()
        print(f"[step] db_ingest FAILED: {e}")
        return False
    finally:
        conn.close()


def step_db_calculate_metrics(mode, start=None, end=None, db_path=None):
    """Step: Calculate daily metrics from DWD tables."""
    db_path = _get_db_path(db_path)
    print(f"[step] db_calculate_metrics: mode={mode}")

    if mode == 'range' and (not start or not end):
        print("[step] db_calculate_metrics FAILED: range mode requires start and end")
        return False

    conn = db_calculate_metrics.get_db_connection(db_path)
    try:
        count = db_calculate_metrics.calculate_metrics(conn, mode, start, end)
        conn.commit()
        print(f"[step] db_calculate_metrics done: {count} rows in metric_daily")
        return True
    except Exception as e:
        conn.rollback()
        print(f"[step] db_calculate_metrics FAILED: {e}")
        return False
    finally:
        conn.close()


def step_db_calculate_dimension_metrics(mode, start=None, end=None, db_path=None):
    """Step: Calculate dimension-level daily metrics."""
    db_path = _get_db_path(db_path)
    print(f"[step] db_calculate_dimension_metrics: mode={mode}")

    if mode == 'range' and (not start or not end):
        print("[step] db_calculate_dimension_metrics FAILED: range mode requires start and end")
        return False

    conn = db_calculate_dimension_metrics.get_db(db_path)
    try:
        count = db_calculate_dimension_metrics.calculate_metrics(conn, mode, start, end)
        conn.commit()
        print(f"[step] db_calculate_dimension_metrics done: {count} rows in metric_dimension_daily")
        return True
    except Exception as e:
        conn.rollback()
        print(f"[step] db_calculate_dimension_metrics FAILED: {e}")
        return False
    finally:
        conn.close()


def step_db_rule_engine(db_path=None):
    """Step: Run rule engine on metric_daily."""
    db_path = _get_db_path(db_path)
    print(f"[step] db_rule_engine")

    conn = db_rule_engine.get_db_connection(db_path)
    try:
        triggered, existing = db_rule_engine.run_rule_engine(conn)
        conn.commit()
        print(f"[step] db_rule_engine done: triggered={triggered}, existing={existing}")
        return True
    except Exception as e:
        conn.rollback()
        print(f"[step] db_rule_engine FAILED: {e}")
        return False
    finally:
        conn.close()


def step_db_dimensional_rule_engine(db_path=None):
    """Step: Run dimensional rule engine on metric_dimension_daily."""
    db_path = _get_db_path(db_path)
    print(f"[step] db_dimensional_rule_engine")

    conn = db_dimensional_rule_engine.get_db(db_path)
    try:
        total, active, suppressed = db_dimensional_rule_engine.run_engine(conn)
        conn.commit()
        print(f"[step] db_dimensional_rule_engine done: total={total}, active={active}, suppressed={suppressed}")
        return True
    except Exception as e:
        conn.rollback()
        print(f"[step] db_dimensional_rule_engine FAILED: {e}")
        return False
    finally:
        conn.close()


def step_db_generate_recommendations(db_path=None):
    """Step: Generate strategy recommendations and action tasks."""
    db_path = _get_db_path(db_path)
    print(f"[step] db_generate_recommendations")

    conn = db_generate_recommendations.get_db_connection(db_path)
    try:
        rec_count, task_count = db_generate_recommendations.generate_recommendations(conn)
        conn.commit()
        print(f"[step] db_generate_recommendations done: recs={rec_count}, tasks={task_count}")
        return True
    except Exception as e:
        conn.rollback()
        print(f"[step] db_generate_recommendations FAILED: {e}")
        return False
    finally:
        conn.close()


def step_db_export_feishu(db_path=None):
    """Step: Export database tables to Feishu-sync-ready CSV files."""
    db_path = _get_db_path(db_path)
    print(f"[step] db_export_feishu")

    conn = db_export_feishu.get_db(db_path)
    try:
        total = 0
        for name, spec in db_export_feishu.EXPORT_SPECS.items():
            rows = db_export_feishu.export(conn, spec)
            total += rows
        print(f"[step] db_export_feishu done: {total} rows exported")
        return True
    except Exception as e:
        print(f"[step] db_export_feishu FAILED: {e}")
        return False
    finally:
        conn.close()


def step_db_trigger_simulator(dry_run=True, db_path=None):
    """Step: Simulate trigger dispatch for pending outbox events."""
    db_path = _get_db_path(db_path)
    mode = 'dry-run' if dry_run else 'apply'
    print(f"[step] db_trigger_simulator: mode={mode}")

    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row
    try:
        # Get pending events
        cur = conn.execute("""
            SELECT outbox_id, event_type, source_type, source_id, payload_json,
                   target_channel, status, created_at
            FROM event_outbox WHERE status = 'pending'
        """)
        pending = cur.fetchall()
        print(f"[step] db_trigger_simulator: {len(pending)} pending outbox events")

        if not pending:
            print("[step] db_trigger_simulator done: no pending events")
            return True

        # Process each event
        os.makedirs(os.path.dirname(db_trigger_simulator.LOG_FILE), exist_ok=True)
        now = datetime.datetime.now().isoformat()
        log_rows = []

        for event in pending:
            target = event['target_channel']

            if target == 'github_issue':
                gen_payload = db_trigger_simulator.generate_github_payload(event)
            elif target == 'feishu_cli':
                gen_payload = db_trigger_simulator.generate_feishu_cli_payload(event)
            elif target == 'local_cli':
                gen_payload = db_trigger_simulator.generate_local_cli_payload(event)
            else:
                gen_payload = {
                    'channel': 'manual',
                    'description': 'Requires human review',
                    'source_id': event['source_id']
                }

            log_row = {
                'outbox_id': event['outbox_id'],
                'event_type': event['event_type'],
                'source_id': event['source_id'],
                'target_channel': target,
                'simulated_at': now,
                'mode': mode,
                'generated_payload': json.dumps(gen_payload, ensure_ascii=False),
                'status': 'simulated' if not dry_run else 'dry-run',
            }
            log_rows.append(log_row)

            if not dry_run:
                conn.execute(
                    "UPDATE event_outbox SET status = 'simulated', processed_at = ? WHERE outbox_id = ?",
                    (now, event['outbox_id'])
                )
                conn.commit()

        # Write simulation log
        with open(db_trigger_simulator.LOG_FILE, 'w', newline='', encoding='utf-8') as f:
            if log_rows:
                writer = csv.DictWriter(f, fieldnames=log_rows[0].keys())
                writer.writeheader()
                writer.writerows(log_rows)

        print(f"[step] db_trigger_simulator done: {len(log_rows)} entries, log={db_trigger_simulator.LOG_FILE}")
        if not dry_run:
            print("[step] db_trigger_simulator: outbox status updated to 'simulated'")
        return True
    except Exception as e:
        conn.rollback()
        print(f"[step] db_trigger_simulator FAILED: {e}")
        return False
    finally:
        conn.close()
