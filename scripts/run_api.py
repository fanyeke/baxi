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
    """Apply sql/migrations/006_api_schema_fix.sql if columns are missing."""
    from scripts.config import DB_PATH

    migration_file = os.path.join(
        PROJECT_ROOT, "sql", "migrations", "006_api_schema_fix.sql"
    )

    if not os.path.exists(migration_file) or not os.path.exists(DB_PATH):
        return

    conn = sqlite3.connect(DB_PATH)
    try:
        # Check if migration already applied
        cur = conn.execute("PRAGMA table_info(event_outbox)")
        cols = [r[1] for r in cur.fetchall()]
        if "dispatch_attempts" in cols:
            print("Migration already applied (dispatch_attempts column exists)")
            return

        print("Applying schema migration: 006_api_schema_fix.sql ...")
        with open(migration_file) as f:
            for stmt in f.read().split(";"):
                stmt = stmt.strip()
                if stmt and not stmt.startswith("--"):
                    try:
                        conn.execute(stmt)
                    except sqlite3.OperationalError as e:
                        if "duplicate column name" in str(e).lower():
                            pass  # idempotent: column already exists
                        else:
                            print(f"Migration warning: {e}", file=sys.stderr)
        conn.commit()
        print("Migration applied successfully")

        # Verify
        cur = conn.execute("PRAGMA table_info(event_outbox)")
        cols = [r[1] for r in cur.fetchall()]
        if "dispatch_attempts" in cols:
            print("Verified: dispatch_attempts column present in event_outbox")
    finally:
        conn.close()


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
