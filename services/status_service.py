from services.db_service import get_db


def get_last_pipeline_run(conn=None):
    should_close = conn is None
    if conn is None:
        conn = get_db()
    try:
        cur = conn.execute("""
            SELECT run_id, run_type, mode, status, started_at, finished_at,
                   input_count, output_count, error_message
            FROM pipeline_runs ORDER BY started_at DESC LIMIT 1
        """)
        row = cur.fetchone()
        return dict(row) if row else None
    finally:
        if should_close:
            conn.close()


def get_last_ingestion_batch(conn=None):
    should_close = conn is None
    if conn is None:
        conn = get_db()
    try:
        cur = conn.execute("""
            SELECT batch_id, source_name, ingestion_mode, date_start, date_end,
                   row_count, status, created_at
            FROM ingestion_batches ORDER BY created_at DESC LIMIT 1
        """)
        row = cur.fetchone()
        return dict(row) if row else None
    finally:
        if should_close:
            conn.close()
