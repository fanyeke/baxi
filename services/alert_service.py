from services.db_service import get_db


def _build_alert_conditions(status, severity, object_type, object_id):
    conditions = []
    params = []
    if status is not None:
        conditions.append("status = ?")
        params.append(status)
    if severity is not None:
        conditions.append("severity = ?")
        params.append(severity)
    if object_type is not None:
        conditions.append("object_type = ?")
        params.append(object_type)
    if object_id is not None:
        conditions.append("object_id = ?")
        params.append(object_id)
    where = "WHERE " + " AND ".join(conditions) if conditions else ""
    return where, params


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
    where, count_params = _build_alert_conditions(status, severity, object_type, object_id)
    items = get_alerts(conn, status, severity, object_type, object_id, limit)
    should_close = conn is None
    if conn is None:
        conn = get_db()
    try:
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
