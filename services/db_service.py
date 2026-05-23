"""Database utility service for SQLite operations."""

import os
import sqlite3

from scripts.config import DB_PATH


def get_db(db_path=None):
    """Get a SQLite database connection with WAL mode and Row factory.

    Args:
        db_path: Optional database file path. Defaults to config.DB_PATH.

    Returns:
        sqlite3.Connection configured with WAL journal mode and Row factory.

    Raises:
        FileNotFoundError: if the DB file does not exist and create_if_missing is False.
    """
    path = db_path or DB_PATH
    conn = sqlite3.connect(path)
    conn.execute("PRAGMA journal_mode=WAL")
    conn.row_factory = sqlite3.Row
    return conn


def get_table_counts(conn):
    """Get row counts for all user tables in the database.

    Args:
        conn: SQLite connection.

    Returns:
        dict mapping table names to row counts.
    """
    tables = conn.execute(
        "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name"
    ).fetchall()

    counts = {}
    for row in tables:
        table_name = row["name"]
        count_row = conn.execute(f"SELECT COUNT(*) as cnt FROM {table_name}").fetchone()
        counts[table_name] = count_row["cnt"]

    return counts


def get_table_info(conn, table_name):
    """Get schema information for a specific table.

    Args:
        conn: SQLite connection.
        table_name: Name of the table.

    Returns:
        List of column info dicts from PRAGMA table_info.
    """
    rows = conn.execute(f"PRAGMA table_info({table_name})").fetchall()
    return [dict(row) for row in rows]


def db_exists(db_path=None):
    """Check if the database file exists.

    Args:
        db_path: Optional database file path. Defaults to config.DB_PATH.

    Returns:
        True if the database file exists, False otherwise.
    """
    path = db_path or DB_PATH
    return os.path.exists(path)
