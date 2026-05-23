#!/usr/bin/env python3
"""db_dimensional_rule_engine.py - Dimensional rule engine for v0.3."""
import os, sys, json, yaml, datetime, hashlib, sqlite3, argparse
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import config
import db_generate_recommendations as rec_gen

DB_PATH = os.path.join(config.PROJECT_ROOT, 'data', 'olist_ops.db')
RULES_FILE = config.DIMENSIONAL_RULES_FILE
SEVERITY_WEIGHT = {'high': 3, 'medium': 2, 'low': 1}
DUE_DAYS = {'high': 1, 'medium': 3, 'low': 7}
MAX_PER_DIM_VAL = 5

def get_db(db_path):
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row
    conn.execute("PRAGMA journal_mode=WAL")
    conn.execute("PRAGMA synchronous=NORMAL")
    return conn

def load_rules(dimension_filter=None):
    with open(RULES_FILE) as f:
        data = yaml.safe_load(f)
    rules = data.get('rules', [])
    limits = data.get('limits', {})
    if dimension_filter:
        rules = [r for r in rules if r['dimension_type'] == dimension_filter]
    return rules, limits

def make_alert_id(rule_id, metric_date, dimension_type, dimension_value):
    raw = f"{rule_id}\xff{metric_date}\xff{dimension_type}\xff{dimension_value}"
    return f"dim-{hashlib.sha256(raw.encode()).hexdigest()[:12]}"

def evaluate_condition(condition_str, current_value, baseline_value, change_rate):
    cs = condition_str.strip()
    if cs.startswith('value_gt:'):
        return current_value > float(cs.split(':')[1].strip())
    if cs.startswith('value_lt:'):
        return current_value < float(cs.split(':')[1].strip())
    if cs.startswith('change_rate_lt:'):
        if baseline_value is None or baseline_value == 0:
            return False
        return change_rate < float(cs.split(':')[1].strip())
    if cs.startswith('change_rate_gt:'):
        if baseline_value is None or baseline_value == 0:
            return False
        return change_rate > float(cs.split(':')[1].strip())
    return False

def run_engine(conn, dimension_filter=None, top_n=None):
    rules, limits = load_rules(dimension_filter)
    if top_n is None:
        top_n = limits.get('max_alerts_per_run', 50)
    print(f"[dim_engine] Loaded {len(rules)} dimensional rules")
    now = datetime.datetime.now().isoformat()
    all_alerts = []

    for rule in rules:
        dim_type = rule['dimension_type']
        metric_name = rule['metric_name']
        min_ss = rule.get('min_sample_size', 1)
        baseline_window = rule.get('baseline_window', 14)
        needs_baseline = 'change_rate' in rule.get('condition', '')

        cur = conn.execute("""
            SELECT metric_date, dimension_value, metric_value, sample_size
            FROM metric_dimension_daily
            WHERE dimension_type = ? AND metric_name = ?
              AND dimension_value IS NOT NULL AND dimension_value != ''
            ORDER BY dimension_value, metric_date DESC
        """, (dim_type, metric_name))

        series_by_dim = {}
        for r in cur.fetchall():
            dv = r[1]
            if dv not in series_by_dim:
                series_by_dim[dv] = []
            series_by_dim[dv].append((r[0], r[2] or 0, r[3] or 0))

        for dim_value, series in series_by_dim.items():
            count = 0
            for i, (metric_date, current_value, sample_size) in enumerate(series):
                if count >= MAX_PER_DIM_VAL:
                    break
                if sample_size < min_ss:
                    continue

                if needs_baseline:
                    future = [series[j][1] for j in range(i+1, min(i+1+baseline_window, len(series)))]
                    baseline_value = sum(future) / len(future) if future else None
                    change_rate = (current_value - baseline_value) / abs(baseline_value) if baseline_value and baseline_value != 0 else 0
                else:
                    baseline_value = None
                    change_rate = 0

                triggered = evaluate_condition(rule['condition'], current_value, baseline_value, change_rate)
                if triggered:
                    count += 1
                    all_alerts.append({
                        'rule_id': rule['rule_id'],
                        'event_id': make_alert_id(rule['rule_id'], metric_date, dim_type, dim_value),
                        'event_date': metric_date,
                        'severity': rule.get('severity', 'medium'),
                        'metric_name': metric_name,
                        'object_type': dim_type,
                        'object_id': dim_value,
                        'current_value': round(current_value, 4),
                        'baseline_value': round(baseline_value, 4) if baseline_value is not None else None,
                        'change_rate': round(change_rate, 4),
                        'sample_size': sample_size,
                        'description': rule.get('description', ''),
                        'owner_role': rule.get('owner_role', 'unassigned'),
                        'target_channel': rule.get('target_channel', 'feishu_cli'),
                    })

    # Fetch affected_gmv
    for ev in all_alerts:
        gmv_row = conn.execute("""
            SELECT metric_value FROM metric_dimension_daily
            WHERE metric_date = ? AND dimension_type = ? AND dimension_value = ? AND metric_name = 'gmv'
        """, (ev['event_date'], ev['object_type'], ev['object_id'])).fetchone()
        ev['affected_gmv'] = round(gmv_row[0], 2) if gmv_row else 0
        ev['affected_orders'] = ev['sample_size']
        sw = SEVERITY_WEIGHT.get(ev['severity'], 1)
        ev['impact_score'] = round(sw * ev['sample_size'], 2)
        ev['evidence_json'] = json.dumps({})

    sev_order = {'high': 0, 'medium': 1, 'low': 2}
    all_alerts.sort(key=lambda a: (sev_order.get(a['severity'], 1), -a['impact_score'], -a['sample_size']))

    suppressed = 0
    for i, ev in enumerate(all_alerts):
        if i >= top_n:
            ev['status'] = 'suppressed'
            suppressed += 1
        else:
            ev['status'] = 'new'

    for ev in all_alerts:
        conn.execute("""
            INSERT OR REPLACE INTO alert_events
            (event_id, rule_id, event_date, severity, metric_name,
             object_type, object_id, current_value, baseline_value, change_rate,
             sample_size, affected_orders, affected_gmv, impact_score,
             evidence_json, description, owner_role, status, created_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        """, (
            ev['event_id'], ev['rule_id'], ev['event_date'], ev['severity'], ev['metric_name'],
            ev['object_type'], ev['object_id'], ev['current_value'], ev['baseline_value'],
            ev['change_rate'], ev['sample_size'], ev['affected_orders'], ev['affected_gmv'],
            ev['impact_score'], ev['evidence_json'], ev['description'], ev['owner_role'],
            ev['status'], now
        ))

    templates = rec_gen.load_dimensional_templates()

    gen_count = 0
    task_count = 0
    for ev in all_alerts:
        if ev['status'] == 'suppressed':
            continue
        rec_id = f"dimrec-{ev['event_id']}"
        if not conn.execute("SELECT 1 FROM strategy_recommendations WHERE recommendation_id = ?", (rec_id,)).fetchone():
            # Template-based strategy_detail and task_title
            template_name = rec_gen.RULE_TO_TEMPLATE.get(ev['rule_id'])
            if template_name and template_name in templates:
                tpl = templates[template_name]
                context = {
                    'object_id': ev['object_id'],
                    'event_date': ev['event_date'],
                    'current_value': ev['current_value'],
                    'sample_size': ev['sample_size'],
                    'affected_gmv': ev['affected_gmv'],
                    'rule_id': ev['rule_id'],
                    'baseline_value': ev['baseline_value'],
                    'change_rate': ev['change_rate'],
                    'dimension_type': ev['object_type'],
                    'metric_name': ev['metric_name'],
                    'owner_role': ev['owner_role'],
                }
                strategy_detail = rec_gen._render_template(tpl['strategy_detail_template'], context)[:500]
                task_title = rec_gen._render_template(tpl['task_title'], context) if 'task_title' in tpl else f"Review {ev['object_type']} {ev['object_id']} {ev['metric_name']}"
            else:
                strategy_detail = (
                    f"Rule '{ev['rule_id']}' on {ev['event_date']}. Current: {ev['current_value']}, "
                    f"Baseline: {ev['baseline_value']}, Change: {ev['change_rate']:.2%}. "
                    f"GMV: {ev['affected_gmv']}"
                )[:500]
                task_title = f"Review {ev['object_type']} {ev['object_id']} {ev['metric_name']}"

            conn.execute("""
                INSERT INTO strategy_recommendations
                (recommendation_id, event_id, decision_source, rule_id,
                 strategy_title, strategy_detail, target_object_type, target_object_id,
                 expected_impact, risk_level, confidence,
                 requires_approval, approval_status, execution_status,
                 owner_role, success_metric, created_at)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            """, (
                rec_id, ev['event_id'], 'heuristic', ev['rule_id'],
                f"{ev['object_type']} {ev['object_id']}: {ev['metric_name']} anomaly",
                strategy_detail,
                ev['object_type'], ev['object_id'],
                f"Stabilize {ev['metric_name']}", ev['severity'], 'medium',
                0, 'draft', 'draft', ev['owner_role'], ev['metric_name'], now
            ))
            gen_count += 1

            task_id = f"dimtask-{ev['event_id']}"
            if not conn.execute("SELECT 1 FROM action_tasks WHERE task_id = ?", (task_id,)).fetchone():
                priority = ev['severity']
                due = (datetime.datetime.now() + datetime.timedelta(days=DUE_DAYS.get(priority, 3))).isoformat()
                conn.execute("""
                    INSERT INTO action_tasks
                    (task_id, recommendation_id, event_id, task_title, task_description,
                     target_object_type, target_object_id,
                     task_source, owner_role, owner_user_id, priority, due_at,
                     status, created_at)
                    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                """, (
                    task_id, rec_id, ev['event_id'],
                    task_title,
                    ev['description'][:200],
                    ev['object_type'], ev['object_id'],
                    'dimensional_rule', ev['owner_role'], None, priority,
                    due, 'todo', now
                ))
                task_count += 1

        outbox_id = f"dimoutbox-{ev['event_id']}"
        if not conn.execute("SELECT 1 FROM event_outbox WHERE outbox_id = ?", (outbox_id,)).fetchone():
            conn.execute("""
                INSERT INTO event_outbox
                (outbox_id, event_type, source_type, source_id, payload_json,
                 target_channel, status, created_at)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?)
            """, (
                outbox_id, 'dimensional_alert', 'dimensional_rule_engine', ev['event_id'],
                json.dumps({'event_id': ev['event_id'], 'rule_id': ev['rule_id'],
                    'target_channel': ev['target_channel'],
                    'object_type': ev['object_type'], 'object_id': ev['object_id'],
                    'severity': ev['severity'],
                    'summary': f"{ev['object_type']} {ev['object_id']} {ev['metric_name']} anomaly",
                    'recommended_action': f"Review {ev['object_type']} {ev['object_id']}",
                    'metric_name': ev['metric_name'], 'current_value': ev['current_value'],
                    'affected_gmv': ev['affected_gmv']}),
                ev['target_channel'], 'pending', now
            ))

    new_alerts = sum(1 for a in all_alerts if a['status'] != 'suppressed')
    print(f"[dim_engine] {len(all_alerts)} alerts ({new_alerts} active, {suppressed} suppressed)")
    print(f"[dim_engine] {gen_count} recommendations, {task_count} tasks, {len(all_alerts)} outbox")
    return len(all_alerts), new_alerts, suppressed

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--mode', default='full', choices=['full'])
    parser.add_argument('--dimension', default=None, choices=['seller', 'category', 'region'])
    parser.add_argument('--top-n', type=int, default=None)
    parser.add_argument('--db', default=None)
    args = parser.parse_args()
    db_path = args.db if args.db else DB_PATH
    conn = get_db(db_path)
    try:
        total, active, suppressed = run_engine(conn, args.dimension, args.top_n)
        conn.commit()
        print(f"[dim_engine] Done. Total={total}, Active={active}, Suppressed={suppressed}")
    except Exception as e:
        conn.rollback()
        print(f"[dim_engine] FAILED: {e}", file=sys.stderr)
        import traceback; traceback.print_exc()
        sys.exit(1)
    finally:
        conn.close()

if __name__ == '__main__':
    main()
