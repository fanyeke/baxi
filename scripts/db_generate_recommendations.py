#!/usr/bin/env python3
"""db_generate_recommendations.py — Strategy recommendation and action task generator.
Supports both global (v0.2) and dimensional (v0.3) alerts."""
import os, sys, json, datetime, sqlite3, yaml, argparse
from string import Template
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import config

DB_PATH = os.path.join(config.PROJECT_ROOT, 'data', 'olist_ops.db')
ACTION_REGISTRY_FILE = config.ACTION_REGISTRY_FILE
OWNER_MAPPING_FILE = config.OWNER_MAPPING_FILE
ACTION_TEMPLATES_FILE = config.ACTION_TEMPLATES_FILE

RULE_TO_TEMPLATE = {
    'seller_late_delivery_spike': 'investigate_seller_delivery',
    'seller_review_score_drop': 'investigate_seller_review',
    'category_gmv_drop': 'investigate_category_gmv',
    'category_low_review_cluster': 'investigate_category_review',
    'region_cancel_rate_spike': 'investigate_region_cancel',
    'region_late_delivery_spike': 'investigate_region_delivery',
}


def get_db_connection(db_path=None):
    if db_path is None:
        db_path = DB_PATH
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row
    return conn


def load_action_templates():
    if not os.path.exists(ACTION_REGISTRY_FILE):
        return {}
    with open(ACTION_REGISTRY_FILE, 'r') as f:
        data = yaml.safe_load(f)
    return data.get('actions', {})


def load_owner_mapping():
    if not os.path.exists(OWNER_MAPPING_FILE):
        return {}
    with open(OWNER_MAPPING_FILE, 'r') as f:
        data = yaml.safe_load(f)
    return data.get('mapping', {})


def load_dimensional_templates():
    if not os.path.exists(ACTION_TEMPLATES_FILE):
        return {}
    with open(ACTION_TEMPLATES_FILE, 'r') as f:
        data = yaml.safe_load(f)
    return data


def row_to_dict(row):
    if row is None:
        return {}
    return dict(row)


def _compute_confidence(sample_size, min_sample_size):
    if sample_size > min_sample_size * 2:
        return 'high'
    elif sample_size > min_sample_size:
        return 'medium'
    return 'low'


def _render_template(template_str, ev):
    tmpl = Template(template_str)
    return tmpl.safe_substitute({
        'object_id': ev.get('object_id', 'unknown'),
        'event_date': ev.get('event_date', ''),
        'current_value': ev.get('current_value', 0),
        'sample_size': ev.get('sample_size', 0),
        'affected_gmv': ev.get('affected_gmv', 0),
        'rule_id': ev.get('rule_id', ''),
        'dimension_type': ev.get('object_type', ''),
        'owner_role': ev.get('owner_role', ''),
    })


def generate_recommendations(conn):
    """Generate strategy_recommendations and action_tasks for events without them."""
    now = datetime.datetime.now().isoformat()

    cur = conn.execute("""
        SELECT ae.event_id, ae.rule_id, ae.event_date, ae.severity, ae.metric_name,
               ae.current_value, ae.baseline_value, ae.change_rate, ae.sample_size,
               ae.description, ae.owner_role, ae.evidence_json,
               ae.object_type, ae.object_id,
               ae.affected_orders, ae.affected_gmv, ae.impact_score
        FROM alert_events ae
        WHERE NOT EXISTS (
            SELECT 1 FROM strategy_recommendations sr WHERE sr.event_id = ae.event_id
        )
    """)
    events = [row_to_dict(r) for r in cur.fetchall()]
    print(f"[db_recs] Found {len(events)} events without recommendations")

    registry = load_action_templates()
    dim_templates = load_dimensional_templates()
    owners = load_owner_mapping()
    rec_count = 0
    task_count = 0

    for ev in events:
        rule_id = ev.get('rule_id', 'unknown')
        event_id = ev.get('event_id', '')
        metric_name = ev.get('metric_name', 'metric')
        owner_role = ev.get('owner_role', 'unassigned')
        severity = ev.get('severity', 'medium')
        description = ev.get('description', '') or ''
        event_date = ev.get('event_date', '')
        object_type = ev.get('object_type', 'global')
        object_id = ev.get('object_id', 'global')
        sample_size = ev.get('sample_size', 0) or 0

        is_dimensional = object_type != 'global'
        template_key = RULE_TO_TEMPLATE.get(rule_id)

        if is_dimensional and template_key and template_key in dim_templates:
            tpl = dim_templates[template_key]
            strategy_title = _render_template(tpl['strategy_title'], ev)
            strategy_detail = _render_template(tpl['strategy_detail_template'], ev)[:500]
            task_title = _render_template(tpl['task_title'], ev)
            task_priority = tpl.get('priority', severity)
            success_metric = tpl.get('success_metric', f"Stabilize {metric_name}")
            min_ss = 20
            confidence = _compute_confidence(sample_size, min_ss)
        else:
            reg_tpl = registry.get(rule_id, {})
            strategy_title = reg_tpl.get('title') or f"Investigate: {description[:60]}"
            strategy_detail = reg_tpl.get('detail') or (
                f"Rule '{rule_id}' triggered for {metric_name} on {event_date}. "
                f"Current: {ev.get('current_value')}, Baseline: {ev.get('baseline_value', 'N/A')}, "
                f"Change: {ev.get('change_rate', 0):.2%}."
            )[:500]
            task_title = f"Review {metric_name} anomaly from rule {rule_id}"
            task_priority = 'high' if severity == 'high' else 'medium'
            success_metric = metric_name
            confidence = 'medium'

        rec_id = f"rec-{rule_id}_{event_date}"
        owner = owners.get(owner_role, owner_role)

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
            strategy_title, strategy_detail,
            object_type, object_id,
            f"Stabilize {metric_name}", severity, confidence,
            0, 'draft', 'draft', owner_role, success_metric, now
        ))
        rec_count += 1

        task_id = f"task-{rule_id}_{event_id}"
        existing = conn.execute("SELECT 1 FROM action_tasks WHERE task_id = ?", (task_id,)).fetchone()
        if not existing:
            priority = task_priority if is_dimensional else ('high' if severity == 'high' else 'medium')
            task_source = 'dimensional_rule' if is_dimensional else 'heuristic_strategy'

            due_map = {'high': 1, 'medium': 3, 'low': 7}
            days = due_map.get(priority, 3)
            due_at = (datetime.datetime.now() + datetime.timedelta(days=days)).isoformat()

            conn.execute("""
                INSERT INTO action_tasks
                (task_id, recommendation_id, event_id, task_title, task_description,
                 task_source, owner_role, owner_user_id, priority, due_at,
                 status, feedback, completed_at, created_at)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            """, (
                task_id, rec_id, event_id,
                task_title,
                description[:200],
                task_source, owner_role, None, priority,
                due_at, 'todo', None, None, now
            ))
            task_count += 1

    print(f"[db_recs] Generated {rec_count} recommendations, {task_count} new tasks")
    return rec_count, task_count


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--db', default=None)
    args = parser.parse_args()
    db_path = args.db if args.db else DB_PATH

    conn = get_db_connection(db_path)
    try:
        rec_count, task_count = generate_recommendations(conn)
        conn.commit()
        print(f"[db_recs] Done. Recommendations: {rec_count}, Tasks: {task_count}")
    except Exception as e:
        conn.rollback()
        print(f"[db_recs] FAILED: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc()
        sys.exit(1)
    finally:
        conn.close()


if __name__ == '__main__':
    main()
