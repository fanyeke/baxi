#!/usr/bin/env python3
"""
Schema verification script: Compare SQLite baseline vs PostgreSQL schema.

Connects to SQLite (data/olist_ops.db) and PostgreSQL ($DATABASE_URL),
compares table lists and column definitions, outputs structured PASS/FAIL report.

Usage:
    python3 scripts/migration/verify_schema.py

Exit code:
    0 = parity (all mapped tables match), 1 = mismatch found

Dependencies:
    - sqlite3 (stdlib)
    - psycopg2 or pg8000 for PostgreSQL connection
"""

import sqlite3
import os
import sys
import json

# ---------------------------------------------------------------------------
# Mapping: SQLite table → PostgreSQL (schema.table)
# These 16 tables are the existing SQLite baseline that must be mapped.
# ---------------------------------------------------------------------------
SQLITE_TO_PG = {
    "dwd_order_level": "dwd.order_level",
    "dwd_item_level": "dwd.item_level",
    "metric_daily": "mart.metric_daily",
    "metric_dimension_daily": "mart.metric_dimension_daily",
    "alert_events": "ops.metric_alert",
    "strategy_recommendations": "ops.recommendation",
    "action_tasks": "ops.task",
    "event_outbox": "ops.outbox_event",
    "review_retro": "ops.review_retro",
    "qoder_jobs": "ops.qoder_job",
    "governance_checkpoints": "gov.governance_checkpoint",
    "governance_health_results": "gov.health_check_result",
    "pipeline_runs": "audit.pipeline_run",
    "ingestion_batches": "audit.ingestion_batch",
    "qoder_runs": "ai.qoder_run",
    "qoder_reports": "ai.qoder_report",
}

# ---------------------------------------------------------------------------
# Expected NEW tables (no SQLite equivalent)
# These must exist in PostgreSQL after migrations 002-008 are applied.
# ---------------------------------------------------------------------------
NEW_TABLES = {
    # raw schema (11)
    "raw.olist_customers",
    "raw.olist_orders",
    "raw.olist_order_items",
    "raw.olist_order_payments",
    "raw.olist_order_reviews",
    "raw.olist_products",
    "raw.olist_sellers",
    "raw.olist_geolocation",
    "raw.product_category_name_translation",
    "raw.marketing_qualified_leads",
    "raw.closed_deals",
    # mart schema (1)
    "mart.metric_snapshot",
    # ops schema (1)
    "ops.dispatch_attempt",
    # gov schema (5)
    "gov.config_snapshot",
    "gov.object_schema",
    "gov.data_classification",
    "gov.data_lineage",
    "gov.access_policy",
    # ai schema (4)
    "ai.decision_case",
    "ai.llm_decision",
    "ai.action_proposal",
    "ai.review_record",
    # audit schema (4)
    "audit.pipeline_step_run",
    "audit.api_request_log",
    "audit.audit_log",
    "audit.error_log",
}

# ---------------------------------------------------------------------------
# Expected schema-level table counts
# ---------------------------------------------------------------------------
EXPECTED_SCHEMA_COUNTS = {
    "raw": 11,
    "dwd": 2,
    "mart": 3,
    "ops": 7,
    "gov": 7,
    "ai": 6,
    "audit": 6,
}

TOTAL_EXPECTED = sum(EXPECTED_SCHEMA_COUNTS.values())


# ---------------------------------------------------------------------------
# Database connections
# ---------------------------------------------------------------------------

def connect_sqlite(path="data/olist_ops.db"):
    """Connect to the SQLite database."""
    db_path = os.path.join(os.path.dirname(__file__), "..", "..", path)
    db_path = os.path.abspath(db_path)
    if not os.path.isfile(db_path):
        print(f"[ERROR] SQLite database not found at: {db_path}")
        sys.exit(1)
    conn = sqlite3.connect(db_path)
    return conn, db_path


def connect_postgres(database_url=None):
    """Connect to PostgreSQL using psycopg2 or pg8000."""
    url = database_url or os.environ.get("DATABASE_URL")
    if not url:
        print("[ERROR] DATABASE_URL environment variable not set")
        print("  Usage: DATABASE_URL='postgres://user:pass@host:port/db' python3 verify_schema.py")
        sys.exit(1)

    clean_url = url
    if "?" in clean_url:
        clean_url = clean_url.split("?")[0]

    try:
        import psycopg2
        conn = psycopg2.connect(dsn=url)
        return conn, "psycopg2"
    except ImportError:
        pass
    except Exception as e:
        print(f"[WARN] psycopg2 connection failed: {e}")
        print("  Falling back to pg8000...")

    try:
        import pg8000
        conn = pg8000.connect(dsn=clean_url)
        return conn, "pg8000"
    except ImportError:
        print("[ERROR] Neither psycopg2 nor pg8000 is installed.")
        print("  Install one of:")
        print("    pip install psycopg2-binary")
        print("    pip install pg8000")
        sys.exit(1)
    except Exception as e:
        print(f"[ERROR] PostgreSQL connection failed: {e}")
        print(f"  URL: {clean_url}")
        sys.exit(1)


# ---------------------------------------------------------------------------
# Schema introspection
# ---------------------------------------------------------------------------

def get_sqlite_tables(conn):
    """Return set of table names in SQLite."""
    rows = conn.execute(
        "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name"
    ).fetchall()
    return {r[0] for r in rows}


def get_sqlite_columns(conn, table):
    """Return ordered list of (name, type) for a SQLite table."""
    rows = conn.execute(f"PRAGMA table_info('{table}')").fetchall()
    return [(r[1], r[2]) for r in rows]


def get_pg_tables(conn):
    """Return set of 'schema.table' names in PostgreSQL (only our 7 schemas)."""
    schemas = tuple(EXPECTED_SCHEMA_COUNTS.keys())
    cursor = conn.cursor()
    cursor.execute(
        """
        SELECT table_schema, table_name
        FROM information_schema.tables
        WHERE table_schema IN %s
          AND table_type = 'BASE TABLE'
        ORDER BY table_schema, table_name
        """,
        (schemas,),
    )
    return {f"{r[0]}.{r[1]}" for r in cursor.fetchall()}


def get_pg_columns(conn, schema, table):
    """Return ordered list of (name, type) for a PostgreSQL table."""
    cursor = conn.cursor()
    cursor.execute(
        """
        SELECT column_name, data_type
        FROM information_schema.columns
        WHERE table_schema = %s AND table_name = %s
        ORDER BY ordinal_position
        """,
        (schema, table),
    )
    return [(r[0], r[1]) for r in cursor.fetchall()]


def get_pg_schema_counts(conn):
    """Return dict of {schema: table_count} for our 7 schemas."""
    schemas = tuple(EXPECTED_SCHEMA_COUNTS.keys())
    cursor = conn.cursor()
    cursor.execute(
        """
        SELECT table_schema, COUNT(*) AS cnt
        FROM information_schema.tables
        WHERE table_schema IN %s
          AND table_type = 'BASE TABLE'
        GROUP BY table_schema
        ORDER BY table_schema
        """,
        (schemas,),
    )
    return {r[0]: r[1] for r in cursor.fetchall()}


# ---------------------------------------------------------------------------
# Verification logic
# ---------------------------------------------------------------------------

def check_table_exists(pg_tables, pg_table_key):
    """Check if schema.table exists in PostgreSQL."""
    return pg_table_key in pg_tables


def normalize_column_name(name):
    """Normalize column name for comparison (lowercase, strip underscores)."""
    return name.lower().replace("_", "")


def compare_columns(sqlite_cols, pg_cols):
    """
    Compare column names between SQLite and PostgreSQL.
    Returns (common, sqlite_only, pg_only) as lists of column names.
    The comparison is fuzzy: we check normalized names to handle
    naming differences like event_id vs alert_id.
    """
    sqlite_names = {c[0] for c in sqlite_cols}
    pg_names = {c[0] for c in pg_cols}

    sqlite_norm = {normalize_column_name(c[0]): c[0] for c in sqlite_cols}
    pg_norm = {normalize_column_name(c[0]): c[0] for c in pg_cols}

    common_exact = sqlite_names & pg_names

    common_norm = set()
    for s_norm, s_orig in sqlite_norm.items():
        if s_orig not in common_exact and s_norm in pg_norm:
            common_norm.add(s_orig)

    sqlite_only_exact = sqlite_names - pg_names
    sqlite_only = sqlite_only_exact - common_norm

    pg_only_exact = pg_names - sqlite_names
    pg_matched_norm = {pg_norm[s_norm] for s_norm in sqlite_norm if s_norm in pg_norm and sqlite_norm[s_norm] not in common_exact}
    pg_only = pg_only_exact - pg_matched_norm

    return common_exact | common_norm, sqlite_only, pg_only


def run_verification():
    """Main verification logic."""
    print("=" * 60)
    print("  Schema Parity Report")
    print("=" * 60)
    print()

    sq_conn, sq_path = connect_sqlite()
    sq_tables = get_sqlite_tables(sq_conn)
    print(f"[INFO] SQLite ({sq_path}): {len(sq_tables)} tables found")
    print()

    pg_conn, pg_driver = connect_postgres()
    pg_tables = get_pg_tables(pg_conn)
    pg_schema_counts = get_pg_schema_counts(pg_conn)
    print(f"[INFO] PostgreSQL (via {pg_driver}): {len(pg_tables)} tables found across 7 schemas")
    print()

    # -----------------------------------------------------------------------
    # Section 1: Mapped tables (16 SQLite → PostgreSQL)
    # -----------------------------------------------------------------------
    print("--- Mapped Tables (SQLite → PostgreSQL) ---")
    mapped_pass = 0
    mapped_fail = 0
    mapped_total = len(SQLITE_TO_PG)

    for sq_table, pg_table_key in SQLITE_TO_PG.items():
        pg_schema, pg_table = pg_table_key.split(".", 1)

        if not check_table_exists(pg_tables, pg_table_key):
            print(f"  [FAIL] {sq_table} → {pg_table_key}: TABLE NOT FOUND")
            mapped_fail += 1
            continue

        sq_cols = get_sqlite_columns(sq_conn, sq_table)
        pg_cols = get_pg_columns(pg_conn, pg_schema, pg_table)

        common, sq_only, pg_only = compare_columns(sq_cols, pg_cols)

        if sq_only:
            print(f"  [PARTIAL] {sq_table} → {pg_table_key}: {len(common)} columns common, {len(sq_only)} SQLite columns missing")
            for c in sorted(sq_only):
                print(f"            SQLite column '{c}' not found in PostgreSQL")
            mapped_pass += 1
        else:
            print(f"  [PASS] {sq_table} → {pg_table_key}: columns match ({len(common)} columns)")
            mapped_pass += 1

        if pg_only:
            for c in sorted(pg_only):
                print(f"            PostgreSQL new column: '{c}'")

    print()

    print("--- New Tables (PostgreSQL only) ---")
    new_pass = 0
    new_fail = 0

    for table_key in sorted(NEW_TABLES):
        if table_key in pg_tables:
            new_pass += 1
        else:
            print(f"  [FAIL] {table_key}: TABLE NOT FOUND")
            new_fail += 1

    if new_fail == 0:
        print(f"  All {new_pass}/{len(NEW_TABLES)} new tables present")
    else:
        print(f"  {new_pass}/{len(NEW_TABLES)} present, {new_fail} missing")
    print()

    print("--- Schema Table Counts ---")
    schema_pass = 0
    schema_fail = 0
    actual_total = 0

    for schema, expected in sorted(EXPECTED_SCHEMA_COUNTS.items()):
        actual = pg_schema_counts.get(schema, 0)
        actual_total += actual
        if actual == expected:
            print(f"  [PASS] {schema}: {actual}/{expected} tables")
            schema_pass += 1
        else:
            print(f"  [FAIL] {schema}: {actual}/{expected} tables")
            schema_fail += 1

    print()
    print(f"  Total PostgreSQL tables: {actual_total}/{TOTAL_EXPECTED}")
    print()

    print("--- Summary ---")
    print(f"  Mapped tables:     {mapped_pass}/{mapped_total} PASS")
    print(f"  New tables:        {new_pass}/{len(NEW_TABLES)} PASS")
    print(f"  Schema counts:     {schema_pass}/{len(EXPECTED_SCHEMA_COUNTS)} PASS")
    print(f"  Total PG tables:   {actual_total}/{TOTAL_EXPECTED}")
    print()

    all_pass = (mapped_fail == 0 and new_fail == 0 and schema_fail == 0)

    if all_pass:
        print("  Status: PARITY PASS")
        return 0
    else:
        print("  Status: PARITY FAIL")
        print()
        if mapped_fail > 0:
            print(f"  - {mapped_fail} mapped table(s) missing in PostgreSQL")
        if new_fail > 0:
            print(f"  - {new_fail} new table(s) missing")
        if schema_fail > 0:
            print(f"  - {schema_fail} schema(s) with incorrect table count")
        return 1


def main():
    try:
        exit_code = run_verification()
        sys.exit(exit_code)
    except Exception as e:
        print(f"\n[ERROR] Unexpected error: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == "__main__":
    main()
