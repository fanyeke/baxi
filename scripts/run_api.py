#!/usr/bin/env python3
"""API Launcher — starts FastAPI server with env check, path setup, and schema migration."""

import os
import sys
import argparse
import sqlite3

# ── Path Setup ──────────────────────────────────────────────────────────
PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, PROJECT_ROOT)
sys.path.insert(0, os.path.join(PROJECT_ROOT, "scripts"))


def check_env() -> str:
    """Check that API_BEARER_TOKEN is set. Returns the token. Exits on failure."""
    token = os.environ.get("API_BEARER_TOKEN", "")

    # Try loading from .env via python-dotenv
    if not token:
        try:
            from dotenv import load_dotenv

            dotenv_path = os.path.join(PROJECT_ROOT, ".env")
            if os.path.exists(dotenv_path):
                load_dotenv(dotenv_path)
                token = os.environ.get("API_BEARER_TOKEN", "")
        except ImportError:
            pass

    if not token:
        print("ERROR: API_BEARER_TOKEN is not set.", file=sys.stderr)
        print("Add it to .env file or set as environment variable.", file=sys.stderr)
        print("Example: export API_BEARER_TOKEN=your-secret-token", file=sys.stderr)
        sys.exit(1)

    return token


def check_db() -> None:
    """Check that the database exists. Warns if missing, does not crash."""
    from scripts.config import DB_PATH

    if not os.path.exists(DB_PATH):
        print(f"WARNING: Database not found at {DB_PATH}", file=sys.stderr)
        print("Health endpoint will report db_connected: false", file=sys.stderr)
    else:
        print(f"Database found: {DB_PATH}")


def run_migration() -> None:
    """Apply sql/migrations if columns are missing."""
    from scripts.config import DB_PATH

    if not os.path.exists(DB_PATH):
        return

    conn = sqlite3.connect(DB_PATH)
    _apply_migration(conn, "005_dispatch_adapters.sql",
                     "event_outbox",
                     ["dispatch_attempts", "last_dispatch_at",
                      "external_ref", "adapter_name"])
    _apply_migration(conn, "006_api_schema_fix.sql",
                     "alert_events",
                     ["affected_orders", "affected_gmv", "impact_score"])
    _apply_migration(conn, "006_api_schema_fix.sql",
                     "action_tasks",
                     ["target_object_type", "target_object_id"])
    _apply_migration(conn, "007_review_retro_status_feedback.sql",
                     "review_retro", ["status", "feedback"])


def _apply_migration(conn, filename, table, columns):
    migration_file = os.path.join(PROJECT_ROOT, "sql", "migrations", filename)
    if not os.path.exists(migration_file):
        return

    cur = conn.execute(f"PRAGMA table_info({table})")
    existing = [r[1] for r in cur.fetchall()]
    if all(c in existing for c in columns):
        print(f"Migration already applied (all columns present in {table})")
        return

    print(f"Applying schema migration: {filename} ...")
    with open(migration_file) as f:
        raw = f.read()

    for stmt in raw.split(";"):
        lines = [l.strip() for l in stmt.split("\n")
                 if l.strip() and not l.strip().startswith("--")]
        stmt_clean = "\n".join(lines).strip()
        if not stmt_clean:
            continue
        try:
            conn.execute(stmt_clean)
        except sqlite3.OperationalError as e:
            if "duplicate column name" in str(e).lower():
                pass
            else:
                print(f"Migration warning: {e}", file=sys.stderr)

    conn.commit()
    print(f"Migration applied successfully")


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Olist Decision Backend API Server"
    )
    parser.add_argument(
        "--port", type=int, default=8765, help="Port to listen on (default: 8765)"
    )
    parser.add_argument(
        "--host",
        default="127.0.0.1",
        help="Host to bind to (default: 127.0.0.1)",
    )
    args = parser.parse_args()

    token = check_env()
    print(f"API_BEARER_TOKEN: {'set' if token else 'MISSING'}")
    check_db()
    run_migration()

    import uvicorn

    print(f"Starting API server on http://{args.host}:{args.port}")
    print(f"API docs: http://{args.host}:{args.port}/docs")
    uvicorn.run("api.main:app", host=args.host, port=args.port, reload=False)


if __name__ == "__main__":
    main()
