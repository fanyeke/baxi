from services.db_service import get_db
from services._query_utils import _build_conditions


def _build_alert_conditions(status, severity, object_type, object_id):
    return _build_conditions({
        "status": status,
        "severity": severity,
        "object_type": object_type,
        "object_id": object_id,
    })


def get_alerts(conn=None, status=None, severity=None, object_type=None, object_id=None, limit=100):
    where, params = _build_alert_conditions(status, severity, object_type, object_id)
    params.append(limit)

    should_close = conn is None
    if conn is None:
        conn = get_db()
    try:
        cur = conn.execute(f"""
            SELECT event_id, rule_id, event_date, severity, metric_name,
                   object_type, object_id, current_value, baseline_value,
                   change_rate, sample_size, status, owner_role, created_at,
                   impact_score
            FROM alert_events {where} ORDER BY event_date DESC LIMIT ?
        """, params)
        return [dict(r) for r in cur.fetchall()]
    finally:
        if should_close:
            conn.close()


def get_alerts_with_count(conn=None, status=None, severity=None, object_type=None, object_id=None, limit=100):
    should_close = conn is None
    if conn is None:
        conn = get_db()
    try:
        items = get_alerts(conn, status, severity, object_type, object_id, limit)
        where, count_params = _build_alert_conditions(status, severity, object_type, object_id)
        total = conn.execute(
            f"SELECT COUNT(*) FROM alert_events {where}", count_params
        ).fetchone()[0]
        return items, total
    finally:
        if should_close:
            conn.close()


def get_alert_by_id(conn=None, event_id=None):
    should_close = conn is None
    if conn is None:
        conn = get_db()
    try:
        cur = conn.execute("""
            SELECT event_id, rule_id, event_date, severity, metric_name,
                   object_type, object_id, current_value, baseline_value,
                   change_rate, sample_size, evidence_json, description,
                   status, owner_role, created_at, impact_score
            FROM alert_events WHERE event_id = ?
        """, (event_id,))
        row = cur.fetchone()
        return dict(row) if row else None
    finally:
        if should_close:
            conn.close()
