#!/usr/bin/env python3
"""db_import_feishu_status.py - Import Feishu status snapshot back into SQLite.

v0.3.1: Closes the Feishu → SQLite feedback loop.
Protects human-set status values from being overwritten.

Usage:
    python3 scripts/db_import_feishu_status.py            # dry-run
    python3 scripts/db_import_feishu_status.py --apply    # execute updates
    python3 scripts/db_import_feishu_status.py --help     # full options
"""
import os, sys, csv, sqlite3, argparse, datetime, json

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import config

SAFE_LOCAL_STATUSES = {'todo', 'new', 'draft', '', None}
HUMAN_PROTECTED_STATUSES = {'in_progress', 'done', 'completed', 'blocked', 'cancelled'}
IMPORT_COLUMNS = {
    'action_tasks': {'status', 'feedback'},
    'review_retro': {'status', 'feedback'},
}
PK_COLUMNS = {
    'action_tasks': 'task_id',
    'review_retro': 'review_id',
}


def get_db(db_path):
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row
    conn.execute("PRAGMA journal_mode=WAL")
    conn.execute("PRAGMA synchronous=NORMAL")
    return conn


def ensure_log_file(log_path):
    if not os.path.exists(log_path):
        os.makedirs(os.path.dirname(log_path), exist_ok=True)
        with open(log_path, 'w', newline='', encoding='utf-8') as f:
            writer = csv.writer(f)
            writer.writerow([
                'timestamp', 'action_type', 'table', 'record_id',
                'field', 'old_value', 'new_value', 'reason',
            ])


def write_log(log_path, table, record_id, field, old_value, new_value, reason):
    with open(log_path, 'a', newline='', encoding='utf-8') as f:
        writer = csv.writer(f)
        writer.writerow([
            datetime.datetime.now(datetime.timezone.utc).isoformat(),
            'import',
            table,
            record_id,
            field,
            old_value or '',
            new_value or '',
            reason,
        ])


def import_status(snapshot_path, db_path, apply=False):
    if not os.path.exists(snapshot_path):
        print(f"ERROR: Snapshot not found: {snapshot_path}")
        sys.exit(1)

    if not os.path.exists(db_path):
        print(f"ERROR: Database not found: {db_path}")
        sys.exit(1)

    with open(snapshot_path, 'r', encoding='utf-8') as f:
        reader = csv.DictReader(f)
        rows = list(reader)

    if not rows:
        print("Snapshot is empty (no records to import).")
        return {"applied": 0, "skipped": 0, "errors": 0, "action": "skipped" if apply else "dry-run"}

    by_table = {}
    for row in rows:
        table = row.get('_table', '')
        if table:
            by_table.setdefault(table, []).append(row)

    print(f"Snapshot: {len(rows)} records across {len(by_table)} table(s)")

    log_path = os.path.join(config.SYSTEM_DIR, 'feishu_import_log.csv')
    ensure_log_file(log_path)

    conn = get_db(db_path)
    applied = 0
    skipped = 0
    errors = 0

    try:
        for table_name, records in by_table.items():
            if table_name not in IMPORT_COLUMNS:
                print(f"  WARNING: Unknown table '{table_name}', skipping")
                continue

            import_cols = IMPORT_COLUMNS[table_name]
            pk_col = PK_COLUMNS[table_name]
            print(f"\n  [{table_name}] {len(records)} records, importing: {import_cols}")

            for rec in records:
                record_id = rec.get('_record_id', '')
                if not record_id:
                    continue

                safe_table = config.validate_sql_identifier(table_name, f"table '{table_name}'")
                safe_pk = config.validate_sql_identifier(pk_col, f"pk_col '{pk_col}'")
                cols_to_fetch = ', '.join([safe_pk, 'status'] + [c for c in import_cols if c != 'status'])
                local_row = conn.execute(
                    f"SELECT {cols_to_fetch} FROM {safe_table} WHERE {safe_pk} = ?",
                    (record_id,)
                ).fetchone()

                if not local_row:
                    print(f"    SKIP (not in DB): {record_id}")
                    skipped += 1
                    write_log(log_path, table_name, record_id, '-', '-', '-', 'not_in_db')
                    continue

                local_status = local_row['status']

                if local_status not in SAFE_LOCAL_STATUSES:
                    print(f"    SKIP (human-protected, status='{local_status}'): {record_id}")
                    skipped += 1
                    for col in import_cols:
                        new_val = rec.get(col, '')
                        old_val = local_row[col] if col != 'status' else local_status
                        write_log(log_path, table_name, record_id, col,
                                  old_val or '', new_val or '', 'human_protected')
                    continue

                updates = {}
                for col in import_cols:
                    new_val = rec.get(col, '')
                    if new_val is None:
                        new_val = ''
                    updates[col] = new_val

                if apply:
                    set_clause = ', '.join(f"{col} = ?" for col in updates)
                    values = [updates[c] for c in updates] + [record_id]
                    conn.execute(
                        f"UPDATE {safe_table} SET {set_clause} WHERE {safe_pk} = ?",
                        values,
                    )

                applied += 1
                print(f"    {'APPLY' if apply else 'PLAN'}: {record_id} -> {updates}")
                for col in import_cols:
                    old_val = local_row[col] if col != 'status' else local_status
                    write_log(log_path, table_name, record_id, col,
                              old_val or '', updates[col] or '', 'applied' if apply else 'planned')

        if apply:
            conn.commit()

        action = 'APPLIED' if apply else 'PLANNED (dry-run)'
        print(f"\n{'='*50}")
        print(f"Import {action}: {applied} updates, {skipped} skipped, {errors} errors")
        print(f"Log: {log_path}")

        return {"applied": applied, "skipped": skipped, "errors": errors, "action": action}

    except Exception as e:
        if apply:
            conn.rollback()
        print(f"\nERROR: {e}")
        raise
    finally:
        conn.close()


def main():
    parser = argparse.ArgumentParser(
        description='Import Feishu status snapshot back into SQLite DB',
    )
    parser.add_argument(
        '--db',
        default=config.DB_PATH,
        help=f'Path to SQLite DB (default: {config.DB_PATH})',
    )
    parser.add_argument(
        '--snapshot',
        default=os.path.join(config.PROJECT_ROOT, 'data', 'ops', 'action_task_status_snapshot.csv'),
        help='Path to snapshot CSV',
    )
    parser.add_argument(
        '--apply',
        action='store_true',
        default=False,
        help='Execute updates (default: dry-run)',
    )
    parser.add_argument(
        '--dry-run',
        action='store_true',
        default=False,
        help=argparse.SUPPRESS,
    )
    parser.add_argument(
        '--json',
        action='store_true',
        help='Output JSON summary to stdout',
    )
    args = parser.parse_args()

    mode = 'APPLY' if args.apply else 'DRY-RUN'
    print(f"db_import_feishu_status v0.3.1 [{mode}]")
    print(f"  DB:       {args.db}")
    print(f"  Snapshot: {args.snapshot}")

    stats = import_status(args.snapshot, args.db, apply=args.apply)

    if args.json and stats:
        print(json.dumps({"status": "success", **stats}))


if __name__ == '__main__':
    main()
