#!/usr/bin/env python3
"""
db_rule_engine.py — Rule engine for decision backend.

Reads config/alert_rules.yml, scans metric_daily from SQLite, generates alert_events,
strategy_recommendations, action_tasks, and event_outbox records.

For v0.2: processes only overall/daily-level rules from metric_daily table.
Dimension-level rules (seller, category, region) are deferred — they work on
metric_dimension_daily which is empty in v0.2.

Supports --mode full (scan all) and --rule <rule_id> (single rule).
Idempotent: does NOT create duplicate event_ids.
"""

import os
import sys
import json
import uuid
import yaml
import datetime
import sqlite3
import argparse

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import config


DB_PATH = os.path.join(config.PROJECT_ROOT, 'data', 'olist_ops.db')
ALERT_RULES_FILE = config.ALERT_RULES_FILE


def get_db_connection(db_path=None):
    if db_path is None:
        db_path = DB_PATH
    conn = sqlite3.connect(db_path)
    conn.execute("PRAGMA journal_mode=WAL")
    conn.execute("PRAGMA synchronous=NORMAL")
    return conn


def load_rules(rule_id_filter=None):
    """Load rules from YAML config file."""
    with open(ALERT_RULES_FILE, 'r') as f:
        data = yaml.safe_load(f)

    rules = data.get('rules', [])
    if rule_id_filter:
        rules = [r for r in rules if r.get('rule_id') == rule_id_filter]
    # Skip deferred rules for v0.2
    rules = [r for r in rules if r.get('status') != 'deferred']
    # Only process overall rules (dimension-level requires metric_dimension_daily)
    rules = [r for r in rules if r.get('dimension') == 'overall']
    return rules


def get_metric_series(conn, metric_name, window=None):
    """Get metric_daily time series for a specific metric, optional date window."""
    safe_metric = config.validate_sql_identifier(metric_name, "metric_name")
    if window:
        cur = conn.execute("""
            SELECT metric_date, {metric} FROM metric_daily 
            WHERE metric_date BETWEEN ? AND ?
            ORDER BY metric_date
        """.format(metric=safe_metric), (window[0], window[1]))
    else:
        cur = conn.execute("""
            SELECT metric_date, {metric} FROM metric_daily 
            ORDER BY metric_date
        """.format(metric=safe_metric))
    return [(row[0], row[1] or 0) for row in cur.fetchall()]


def compute_7d_avg(series):
    """Compute average of last 7 days in series."""
    if not series or len(series) < 7:
        return None, len(series)
    last_7 = series[-7:]
    avg = sum(v for _, v in last_7) / 7
    return avg, 7


def compute_14d_avg(series):
    """Compute average of last 14 days in series (before last 7)."""
    if len(series) < 21:  # Need at least 7+14 days
        return None, 0
    prev_14 = series[-21:-7]  # 14 days before the current 7-day window
    avg = sum(v for _, v in prev_14) / 14
    return avg, 14


def compute_current_value(series):
    """Get current (latest) value."""
    if not series:
        return 0, 0
    return series[-1][1], 1


def evaluate_rule(rule, series):
    """Evaluate a rule against a time series. Returns (triggered, evidence_dict) or (False, None)."""
    rule_id = rule['rule_id']
    condition = rule.get('condition', '')
    min_sample = rule.get('min_sample_size', 1)

    # Compute window metrics
    current_7d_avg, window_7d = compute_7d_avg(series)
    prev_14d_avg, window_14d = compute_14d_avg(series)
    current_val, current_samples = compute_current_value(series)

    # If we don't have enough data, skip
    total_samples = len(series)
    if total_samples < min_sample:
        return False, None

    # Use current_7d_avg as the primary comparison metric
    current = current_7d_avg if current_7d_avg is not None else current_val
    baseline = prev_14d_avg if prev_14d_avg is not None else current_val

    # Compute derived metrics
    change_rate = 0.0
    if baseline and baseline != 0:
        change_rate = (current - baseline) / abs(baseline)

    evidence = {
        'current_7d_avg': round(current_7d_avg, 4) if current_7d_avg else None,
        'prev_14d_avg': round(prev_14d_avg, 4) if prev_14d_avg else None,
        'current_value': round(current_val, 4),
        'baseline_value': round(baseline, 4) if baseline else None,
        'change_rate': round(change_rate, 4),
        'window_7d_count': window_7d,
        'window_14d_count': window_14d,
        'total_samples': total_samples,
    }

    # Evaluate conditions based on rule pattern
    triggered = False
    if 'current_7d_avg < prev_14d_avg' in condition:
        # e.g., gmv_drop: current_7d_avg < prev_14d_avg * 0.85
        threshold_factor = float(condition.split('* ')[-1].strip())
        triggered = (current_7d_avg is not None and prev_14d_avg is not None and
                     current_7d_avg < prev_14d_avg * threshold_factor)

    elif 'current_7d_avg > prev_14d_avg' in condition:
        # e.g., gmv_spike: current_7d_avg > prev_14d_avg * 1.20
        threshold_factor = float(condition.split('* ')[-1].strip())
        triggered = (current_7d_avg is not None and prev_14d_avg is not None and
                     current_7d_avg > prev_14d_avg * threshold_factor)

    elif 'value >' in condition and 'order_count >=' in condition:
        # e.g., late_delivery_spike: value > 0.25 and order_count >= 20
        parts = condition.split(' and ')
        value_threshold = float(parts[0].split('> ')[-1].strip())
        count_threshold = int(parts[1].split('>= ')[-1].strip())
        triggered = current_val > value_threshold and total_samples >= count_threshold

    elif 'change_rate >' in condition and 'value >' in condition:
        # e.g., cancel_rate_spike: change_rate > 0.5 and value > 0.05
        parts = condition.split(' and ')
        change_threshold = float(parts[0].split('> ')[-1].strip())
        value_threshold = float(parts[1].split('> ')[-1].strip())
        triggered = abs(change_rate) > change_threshold and current_val > value_threshold

    evidence['triggered'] = triggered
    return triggered, evidence


def write_alert_event(conn, rule, event_date, evidence, metric_val):
    """Write alert_events, strategy_recommendations, action_tasks, and event_outbox."""
    rule_id = rule['rule_id']
    event_id = f"{rule_id}_{event_date}"

    # Check idempotency
    existing = conn.execute("SELECT 1 FROM alert_events WHERE event_id = ?", (event_id,)).fetchone()
    if existing:
        return event_id, False  # Already exists

    metric_name = rule.get('metric', 'unknown')

    # Determine current_value from evidence
    current_value = evidence.get('current_value', metric_val)
    baseline_value = evidence.get('baseline_value', 0)
    change_rate = evidence.get('change_rate', 0)

    description = rule.get('description', '')
    if evidence.get('current_7d_avg') is not None and evidence.get('prev_14d_avg') is not None:
        description += f" | 7d_avg={evidence['current_7d_avg']:.2f}, 14d_avg={evidence['prev_14d_avg']:.2f}"

    # Insert alert event
    now = datetime.datetime.now().isoformat()
    conn.execute("""
        INSERT INTO alert_events
        (event_id, rule_id, event_date, severity, metric_name,
         object_type, object_id, current_value, baseline_value, change_rate,
         sample_size, evidence_json, description, owner_role, status, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    """, (
        event_id, rule_id, event_date,
        rule.get('severity', 'medium'), metric_name,
        'global', 'global',
        current_value, baseline_value, change_rate,
        evidence.get('total_samples', 0),
        json.dumps(evidence, ensure_ascii=False),
        description,
        rule.get('owner_role', 'unassigned'),
        'new',
        now
    ))

    # Generate strategy recommendation
    rec_id = f"rec-{rule_id}_{event_date}"
    existing_rec = conn.execute(
        "SELECT 1 FROM strategy_recommendations WHERE recommendation_id = ?", (rec_id,)
    ).fetchone()
    if not existing_rec:
        conn.execute("""
            INSERT INTO strategy_recommendations
            (recommendation_id, event_id, decision_source, rule_id,
             strategy_title, strategy_detail, target_object_type, target_object_id,
             expected_impact, risk_level, confidence,
             requires_approval, approval_status, execution_status,
             owner_role, success_metric, created_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        """, (
            rec_id, event_id, 'heuristic', rule_id,
            f"Investigate: {rule.get('description', rule_id)[:60]}",
            f"Rule '{rule_id}' triggered for {metric_name} on {event_date}. "
            f"Current: {current_value:.4f}, Baseline: {baseline_value:.4f}, Change: {change_rate:.2%}. "
            f"Evidence: {json.dumps({k: v for k, v in evidence.items() if k != 'triggered'}, ensure_ascii=False)}"[:500],
            'global', 'global',
            f"Stabilize {metric_name} trends",
            rule.get('severity', 'medium'), 'medium',
            0, 'draft', 'draft',
            rule.get('owner_role', 'unassigned'),
            metric_name,
            now
        ))

    # Generate action task
    task_id = f"task-{rule_id}_{event_date}"
    existing_task = conn.execute(
        "SELECT 1 FROM action_tasks WHERE task_id = ?", (task_id,)
    ).fetchone()
    if not existing_task:
        conn.execute("""
            INSERT INTO action_tasks
            (task_id, recommendation_id, event_id, task_title, task_description,
             task_source, owner_role, owner_user_id, priority, due_at,
             status, feedback, completed_at, created_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        """, (
            task_id, rec_id, event_id,
            f"Review {metric_name} anomaly from rule {rule_id}",
            description[:200],
            'heuristic_strategy',
            rule.get('owner_role', 'unassigned'),
            None,
            'high' if rule.get('severity') == 'high' else 'medium',
            None,
            'todo', None, None,
            now
        ))

    # Write to event outbox
    outbox_id = f"outbox-{event_id}"
    existing_outbox = conn.execute(
        "SELECT 1 FROM event_outbox WHERE outbox_id = ?", (outbox_id,)
    ).fetchone()
    if not existing_outbox:
        conn.execute("""
            INSERT INTO event_outbox
            (outbox_id, event_type, source_type, source_id, payload_json,
             target_channel, status, created_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?)
        """, (
            outbox_id, 'alert', 'rule_engine', event_id,
            json.dumps({
                'rule_id': rule_id,
                'event_id': event_id,
                'metric_name': metric_name,
                'current_value': current_value,
                'baseline_value': baseline_value,
                'change_rate': change_rate,
                'severity': rule.get('severity', 'medium'),
                'owner_role': rule.get('owner_role', 'unassigned'),
            }, ensure_ascii=False),
            'local_cli',  # Default channel for v0.2
            'pending',
            now
        ))

    return event_id, True


def run_rule_engine(conn, rule_id_filter=None):
    """Run the rule engine: scan rules, evaluate against metric_daily, generate alerts."""
    rules = load_rules(rule_id_filter)
    print(f"[db_rule_engine] Loaded {len(rules)} active overall rules")

    total_triggered = 0
    total_existing = 0
    alert_events = []

    for rule in rules:
        metric_name = rule.get('metric', '')
        series = get_metric_series(conn, metric_name)
        print(f"[db_rule_engine] Checking rule '{rule['rule_id']}' on {metric_name} ({len(series)} data points)")

        triggered, evidence = evaluate_rule(rule, series)
        if triggered:
            event_date = series[-1][0] if series else datetime.date.today().isoformat()
            event_id, created = write_alert_event(conn, rule, event_date, evidence, series[-1][1] if series else 0)
            if created:
                total_triggered += 1
                alert_events.append(event_id)
                print(f"  -> TRIGGERED: {rule['rule_id']} on {event_date} (event_id={event_id})")
            else:
                total_existing += 1
                print(f"  -> ALREADY EXISTS: {event_id}")
        else:
            print(f"  -> Not triggered")

    print(f"[db_rule_engine] Summary: {total_triggered} new alerts, {total_existing} already existed")
    return total_triggered, total_existing


def main():
    parser = argparse.ArgumentParser(description='Rule engine for decision backend')
    parser.add_argument('--mode', default='full', choices=['full'],
                        help='Execution mode: full (default)')
    parser.add_argument('--rule', default=None, help='Run single rule by rule_id')
    parser.add_argument('--db', default=DB_PATH, help='Path to SQLite database')
    args = parser.parse_args()

    conn = get_db_connection(args.db)
    try:
        triggered, existing = run_rule_engine(conn, args.rule)
        conn.commit()
        print(f"[db_rule_engine] Done. Triggered: {triggered}, Already existed: {existing}")
    except Exception as e:
        conn.rollback()
        print(f"[db_rule_engine] FAILED: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc()
        sys.exit(1)
    finally:
        conn.close()


if __name__ == '__main__':
    main()
