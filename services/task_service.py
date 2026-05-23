from services.db_service import get_db


def _build_task_conditions(status, priority, owner_role):
    conditions = []
    params = []
    if status is not None:
        conditions.append("status = ?")
        params.append(status)
    if priority is not None:
        conditions.append("priority = ?")
        params.append(priority)
    if owner_role is not None:
        conditions.append("owner_role = ?")
        params.append(owner_role)
    where = "WHERE " + " AND ".join(conditions) if conditions else ""
    return where, params


def get_tasks(conn=None, status=None, priority=None, owner_role=None, limit=100):
    where, params = _build_task_conditions(status, priority, owner_role)
    params.append(limit)

    should_close = conn is None
    if conn is None:
        conn = get_db()
    try:
        cur = conn.execute(f"""
            SELECT task_id, task_title, task_description, status, priority,
                   owner_role, due_at, created_at, completed_at, feedback
            FROM action_tasks {where} ORDER BY created_at DESC LIMIT ?
        """, params)
        return [dict(r) for r in cur.fetchall()]
    finally:
        if should_close:
            conn.close()


def get_tasks_with_count(conn=None, status=None, priority=None, owner_role=None, limit=100):
    where, count_params = _build_task_conditions(status, priority, owner_role)
    items = get_tasks(conn, status, priority, owner_role, limit)
    should_close = conn is None
    if conn is None:
        conn = get_db()
    try:
        total = conn.execute(
            f"SELECT COUNT(*) FROM action_tasks {where}", count_params
        ).fetchone()[0]
        return items, total
    finally:
        if should_close:
            conn.close()


def get_task_by_id(conn=None, task_id=None):
    should_close = conn is None
    if conn is None:
        conn = get_db()
    try:
        cur = conn.execute("""
            SELECT task_id, recommendation_id, event_id, task_title,
                   task_description, task_source, owner_role, owner_user_id,
                   priority, due_at, status, feedback, completed_at, created_at
            FROM action_tasks WHERE task_id = ?
        """, (task_id,))
        row = cur.fetchone()
        return dict(row) if row else None
    finally:
        if should_close:
            conn.close()
