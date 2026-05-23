#!/usr/bin/env python3
"""CLI entry point for dispatching pending outbox events via adapters.

Usage:
    python3 scripts/db_dispatch_outbox.py --dry-run
    python3 scripts/db_dispatch_outbox.py --apply --limit 5 --channel feishu_cli
"""
import os
import sys
import sqlite3
import argparse
import datetime

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import config
from adapters.base import load_adapter_registry
from services.dispatch_service import (
    fetch_pending,
    write_audit_log,
    dispatch_one,
    DISPATCH_ARCHIVE,
)


def get_db(db_path=None):
    """Create a SQLite connection with WAL mode and row factory."""
    if db_path is None:
        db_path = config.DB_PATH
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row
    conn.execute("PRAGMA journal_mode=WAL")
    conn.execute("PRAGMA synchronous=NORMAL")
    return conn


def main():
    parser = argparse.ArgumentParser(description="Dispatch pending outbox events via adapters")
    parser.add_argument("--dry-run", action="store_true", help="Preview dispatch without side effects")
    parser.add_argument("--apply", action="store_true", help="Execute real dispatch")
    parser.add_argument("--channel", help="Filter by target_channel (e.g. feishu_cli)")
    parser.add_argument("--limit", type=int, default=None, help="Max events to dispatch")
    parser.add_argument("--db", default=None, help="Path to SQLite database")
    args = parser.parse_args()

    is_dry_run = not args.apply
    mode = "dry-run" if is_dry_run else "apply"

    print(f"[db_dispatch_outbox] Mode: {mode}")
    if args.channel:
        print(f"[db_dispatch_outbox] Channel filter: {args.channel}")
    if args.limit:
        print(f"[db_dispatch_outbox] Limit: {args.limit}")

    conn = get_db(args.db)
    registry = load_adapter_registry()
    audit_entries = []
    processed, succeeded, failed, skipped = 0, 0, 0, 0

    try:
        pending = fetch_pending(conn, args.channel, args.limit)
        print(f"[db_dispatch_outbox] Found {len(pending)} pending events")

        for event in pending:
            outbox_id = event["outbox_id"]
            target = event["target_channel"]

            dispatch_result = dispatch_one(conn, event, registry, is_dry_run)
            status = dispatch_result["status"]
            adapter_name = dispatch_result["adapter_name"]
            error = dispatch_result.get("error")
            external_ref = dispatch_result.get("external_ref")

            print(f"  [{target}] {outbox_id}: {status}" +
                  (f" ({external_ref[:40]})" if external_ref else "") +
                  (f" ERROR: {error[:60]}" if error else ""))

            if not is_dry_run:
                conn.commit()

            audit_entries.append({
                "timestamp": datetime.datetime.now().isoformat(),
                "outbox_id": outbox_id, "target_channel": target,
                "adapter_name": adapter_name, "mode": mode,
                "status": status, "external_ref": external_ref, "error": error,
            })

            processed += 1
            if status == "dispatched":
                succeeded += 1
            elif status == "failed":
                failed += 1
            else:
                skipped += 1

        write_audit_log(audit_entries)
        print(f"[db_dispatch_outbox] Done: processed={processed}, succeeded={succeeded}, "
              f"failed={failed}, skipped={skipped}")
        print(f"[db_dispatch_outbox] Audit: {DISPATCH_ARCHIVE}")

    except Exception as e:
        conn.rollback()
        print(f"[db_dispatch_outbox] FAILED: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc()
        sys.exit(1)
    finally:
        conn.close()


if __name__ == "__main__":
    main()
