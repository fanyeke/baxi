#!/usr/bin/env python3
"""db_migrate.py - Schema migration for v0.3 dimensional alerts."""
import os, sys, sqlite3, argparse
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import config

COLUMNS = {
    'alert_events': [
        ('affected_orders', 'INTEGER'),
        ('affected_gmv', 'REAL'),
        ('impact_score', 'REAL'),
    ],
    'strategy_recommendations': [
        ('confidence', 'TEXT'),
        ('target_object_type', 'TEXT'),
        ('target_object_id', 'TEXT'),
    ],
    'action_tasks': [
        ('target_object_type', 'TEXT'),
        ('target_object_id', 'TEXT'),
    ],
    'event_outbox': [
        ('dispatch_attempts', 'INTEGER DEFAULT 0'),
        ('last_dispatch_at', 'TEXT'),
        ('external_ref', 'TEXT'),
        ('adapter_name', 'TEXT'),
    ],
}


def get_db(db_path):
    conn = sqlite3.connect(db_path)
    conn.execute("PRAGMA journal_mode=WAL")
    conn.execute("PRAGMA synchronous=NORMAL")
    return conn


def get_existing_columns(conn, table):
    cur = conn.execute(f"PRAGMA table_info({table})")
    return {row[1] for row in cur.fetchall()}


def migrate(to_version, db_path):
    conn = get_db(db_path)
    try:
        if to_version == 'v0.3':
            added = _migrate_v03(conn)
        elif to_version == 'v0.4':
            added = _migrate_v04(conn)
        else:
            print(f"[db_migrate] Unknown version: {to_version}")
            sys.exit(1)

        if added == 0:
            print("[db_migrate] All columns already exist, nothing to add")
        else:
            print(f"[db_migrate] Added {added} columns for {to_version}")

        os.makedirs(config.MIGRATIONS_DIR, exist_ok=True)
        conn.commit()
    except Exception as e:
        conn.rollback()
        print(f"[db_migrate] FAILED: {e}", file=sys.stderr)
        sys.exit(1)
    finally:
        conn.close()


def _migrate_v03(conn):
    added = 0
    for table, cols in COLUMNS.items():
        existing = get_existing_columns(conn, table)
        for col_name, col_type in cols:
            if col_name in existing:
                print(f"  [skip] {table}.{col_name} already exists")
            else:
                try:
                    conn.execute(f"ALTER TABLE {table} ADD COLUMN {col_name} {col_type}")
                    print(f"  [added] {table}.{col_name} {col_type}")
                    added += 1
                except sqlite3.OperationalError as e:
                    if 'duplicate column' in str(e).lower():
                        print(f"  [skip] {table}.{col_name} already exists (duplicate)")
                    else:
                        raise
    return added


def _migrate_v04(conn):
    v04_columns = COLUMNS.get('event_outbox', [])
    existing = get_existing_columns(conn, 'event_outbox')
    added = 0
    for col_name, col_type in v04_columns:
        if col_name in existing:
            print(f"  [skip] event_outbox.{col_name} already exists")
        else:
            try:
                conn.execute(f"ALTER TABLE event_outbox ADD COLUMN {col_name} {col_type}")
                print(f"  [added] event_outbox.{col_name} {col_type}")
                added += 1
            except sqlite3.OperationalError as e:
                if 'duplicate column' in str(e).lower():
                    print(f"  [skip] event_outbox.{col_name} already exists (duplicate)")
                else:
                    raise
    return added


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--to', required=True, choices=['v0.3', 'v0.4'])
    parser.add_argument('--db', default=None)
    args = parser.parse_args()
    db_path = args.db if args.db else config.DB_PATH
    migrate(args.to, db_path)


if __name__ == '__main__':
    main()
