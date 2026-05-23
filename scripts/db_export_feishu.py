#!/usr/bin/env python3
"""db_export_feishu.py - Export database tables to Feishu-sync-ready CSV files.
Enhanced for v0.3: includes dimensional columns."""
import os, sys, csv, sqlite3, argparse, json
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import config

DB_PATH = os.path.join(config.PROJECT_ROOT, 'data', 'olist_ops.db')
FEISHU_DIR = config.FEISHU_DIR

TABLE_MAP = {
    'daily_metrics_for_feishu.csv': 'metric_daily',
    'alert_events_for_feishu.csv': 'alert_events',
    'strategy_recommendations_for_feishu.csv': 'strategy_recommendations',
    'action_tasks_for_feishu.csv': 'action_tasks',
    'review_retro_for_feishu.csv': 'review_retro',
}

EXPORT_SPECS = {
    'daily_metrics': {
        'output': os.path.join(FEISHU_DIR, 'daily_metrics_for_feishu.csv'),
        'columns': ['simulated_date', 'gmv', 'order_count', 'customer_count', 'seller_count',
                    'avg_order_value', 'avg_review_score', 'low_review_rate',
                    'late_delivery_rate', 'cancel_rate'],
        'column_map': {'simulated_date': 'metric_date'},
    },
    'alert_events': {
        'output': os.path.join(FEISHU_DIR, 'alert_events_for_feishu.csv'),
        'columns': ['alert_id', 'rule_id', 'event_date', 'severity', 'object_type',
                    'object_id', 'metric_name', 'current_value', 'baseline_value',
                    'change_rate', 'sample_size', 'affected_orders', 'affected_gmv',
                    'impact_score', 'description', 'owner', 'status'],
        'column_map': {'alert_id': 'event_id', 'owner': 'owner_role'},
    },
    'strategy_recommendations': {
        'output': os.path.join(FEISHU_DIR, 'strategy_recommendations_for_feishu.csv'),
        'columns': ['recommendation_id', 'event_id', 'decision_source', 'rule_id',
                    'title', 'detail', 'target_object_type', 'target_object_id',
                    'expected_impact', 'risk_level', 'confidence',
                    'owner', 'approval_status', 'status', 'success_metric', 'created_at'],
        'column_map': {'title': 'strategy_title', 'detail': 'strategy_detail', 'owner': 'owner_role', 'status': 'execution_status'},
    },
    'action_tasks': {
        'output': os.path.join(FEISHU_DIR, 'action_tasks_for_feishu.csv'),
        'columns': ['task_id', 'recommendation_id', 'event_id', 'title',
                     'description', 'target_object_type', 'target_object_id', 'task_source', 'owner', 'priority', 'status',
                     'deadline', 'feedback', 'created_at'],
        'column_map': {'title': 'task_title', 'description': 'task_description', 'owner': 'owner_role', 'deadline': 'due_at'},
    },
    'review_retro': {
        'output': os.path.join(FEISHU_DIR, 'review_retro_for_feishu.csv'),
        'columns': ['review_id', 'recommendation_id', 'task_id', 'review_type',
                    'review_source', 'outcome', 'actual_impact', 'status', 'feedback',
                    'is_effective', 'lessons_learned', 'promote_to_rule', 'reviewed_at'],
        'column_map': {'outcome': 'actual_result', 'lessons_learned': 'lesson_learned'},
    },
}


def get_db(db_path):
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row
    conn.execute("PRAGMA journal_mode=WAL")
    conn.execute("PRAGMA synchronous=NORMAL")
    return conn


def get_columns(conn, table):
    safe_table = config.validate_sql_identifier(table, f"table '{table}'")
    return [r[1] for r in conn.execute(f"PRAGMA table_info({safe_table})").fetchall()]


def export(conn, spec):
    fname = os.path.basename(spec['output'])
    table = TABLE_MAP.get(fname, fname.replace('_for_feishu.csv', ''))
    db_cols = set(get_columns(conn, table))
    col_map = spec.get('column_map', {})
    db_query_cols = [col_map.get(c, c) for c in spec['columns']]
    cols = [c for c in db_query_cols if c in db_cols]
    rows = conn.execute(f"SELECT {', '.join(cols)} FROM {config.validate_sql_identifier(table, f'table {table}')}").fetchall()
    csv_cols = [c for c in spec['columns'] if col_map.get(c, c) in cols]
    os.makedirs(os.path.dirname(spec['output']), exist_ok=True)
    with open(spec['output'], 'w', newline='', encoding='utf-8') as f:
        writer = csv.writer(f)
        writer.writerow(csv_cols)
        for row in rows:
            writer.writerow([row[c] for c in cols])
    print(f"  Exported {len(rows)} rows -> {spec['output']}")
    return len(rows)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--all', action='store_true')
    parser.add_argument('--db')
    parser.add_argument('--json', action='store_true', help='Output JSON summary to stdout')
    args = parser.parse_args()
    if not args.all:
        parser.print_help()
        return

    conn = get_db(args.db if args.db else DB_PATH)
    try:
        total = 0
        table_results = []
        for name, spec in EXPORT_SPECS.items():
            rows = export(conn, spec)
            table_results.append({"table": name, "rows": rows, "file": spec["output"]})
            total += rows
        print(f"[db_export_feishu] Done. {total} rows")
        if args.json:
            print(json.dumps({"status": "success", "total_rows": total, "tables": table_results}))
    finally:
        conn.close()



if __name__ == '__main__':
    main()
