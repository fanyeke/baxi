#!/usr/bin/env python3
"""
db_init.py — Initialize SQLite database for v0.2 decision backend.

Reads sql/schema.sql and sql/indexes.sql, creates data/olist_ops.db.
Idempotent: safe to re-run without errors. Never drops existing tables (uses CREATE TABLE IF NOT EXISTS).
"""

import os
import sys
import sqlite3
import datetime

# Add parent directory to path so we can import config
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import config


DB_PATH = os.path.join(config.PROJECT_ROOT, 'data', 'olist_ops.db')
SQL_DIR = os.path.join(config.PROJECT_ROOT, 'sql')
SCHEMA_FILE = os.path.join(SQL_DIR, 'schema.sql')
INDEXES_FILE = os.path.join(SQL_DIR, 'indexes.sql')


def init_database(db_path=None):
    """Initialize the SQLite database with schema and indexes."""
    if db_path is None:
        db_path = DB_PATH

    # Ensure data directory exists
    data_dir = os.path.dirname(db_path)
    os.makedirs(data_dir, exist_ok=True)

    # Read SQL files
    with open(SCHEMA_FILE, 'r') as f:
        schema_sql = f.read()
    with open(INDEXES_FILE, 'r') as f:
        indexes_sql = f.read()

    # Execute against database
    conn = sqlite3.connect(db_path)
    try:
        conn.executescript(schema_sql)
        conn.executescript(indexes_sql)
        conn.commit()

        # Verify tables
        cursor = conn.execute(
            "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name"
        )
        tables = [row[0] for row in cursor.fetchall()]
        print(f"Database initialized: {db_path}")
        print(f"Tables created ({len(tables)}): {', '.join(tables)}")
        return tables
    finally:
        conn.close()


def main():
    print(f"[db_init] Starting database initialization at {datetime.datetime.now().isoformat()}")
    tables = init_database()
    print(f"[db_init] Done. {len(tables)} tables ready.")


if __name__ == '__main__':
    main()
