"""
Dispatch service for event_outbox processing.

Extracted from scripts/db_dispatch_outbox.py to enable reuse
across CLI scripts and scheduled jobs.
"""

import os
import csv
import datetime

import config
from adapters.base import resolve_adapter


# Constants
MAX_ATTEMPTS = 3
DISPATCH_ARCHIVE = os.path.join(config.SYSTEM_DIR, "dispatch_archive.csv")


def fetch_pending(conn, channel=None, limit=None):
    """Fetch pending events from event_outbox within attempt limits.

    Args:
        conn: SQLite connection with row_factory set.
        channel: Optional target_channel filter.
        limit: Optional max number of events to return (default 10000).

    Returns:
        list of sqlite3.Row matching the query.
    """
    query = """SELECT outbox_id, event_type, source_type, source_id,
                  payload_json, target_channel, status, created_at
               FROM event_outbox
               WHERE status = 'pending' AND dispatch_attempts < ?"""
    params = [MAX_ATTEMPTS]
    if channel:
        query += " AND target_channel = ?"
        params.append(channel)
    query += " ORDER BY created_at LIMIT ?"
    params.append(limit if limit else 10000)
    return conn.execute(query, params).fetchall()


def claim_event(conn, outbox_id):
    """Atomically claim a pending event for dispatch.

    Args:
        conn: SQLite connection.
        outbox_id: The outbox record ID to claim.

    Returns:
        True if the event was successfully claimed (was pending), False otherwise.
    """
    conn.execute(
        "UPDATE event_outbox SET status = 'dispatching' WHERE outbox_id = ? AND status = 'pending'",
        (outbox_id,))
    conn.commit()
    return conn.execute(
        "SELECT changes() FROM (SELECT 1)").fetchone()[0] > 0


def write_result(conn, outbox_id, result, adapter_name, is_dry_run):
    """Update the outbox record with dispatch result.

    Status transitions:
        - dry_run: always stays 'pending'
        - dispatched/skipped: moves to that status
        - failed: increments attempts, retries if < MAX_ATTEMPTS, else 'failed'

    Args:
        conn: SQLite connection.
        outbox_id: The outbox record ID.
        result: dict with keys status, external_ref, error.
        adapter_name: Name of the adapter class used.
        is_dry_run: If True, status remains 'pending'.
    """
    now = datetime.datetime.now().isoformat()
    status = result.get("status", "failed")
    external_ref = result.get("external_ref")
    error = result.get("error")

    if is_dry_run:
        db_status = "pending"
    elif status in ("dispatched", "skipped"):
        db_status = status
    else:
        conn.execute("UPDATE event_outbox SET dispatch_attempts = dispatch_attempts + 1")
        if conn.execute("SELECT dispatch_attempts FROM event_outbox WHERE outbox_id = ?",
                         (outbox_id,)).fetchone()[0] < MAX_ATTEMPTS:
            db_status = "pending"
        else:
            db_status = "failed"

    conn.execute("""UPDATE event_outbox
        SET status = ?, external_ref = ?, adapter_name = ?,
            dispatch_attempts = dispatch_attempts + 1,
            last_dispatch_at = ?, error_message = ?
        WHERE outbox_id = ?""",
        (db_status, external_ref, adapter_name, now, error, outbox_id))


def write_audit_log(entries):
    """Append dispatch audit entries to the archive CSV.

    Args:
        entries: list of dicts with keys: timestamp, outbox_id, target_channel,
                 adapter_name, mode, status, external_ref, error.
    """
    os.makedirs(os.path.dirname(DISPATCH_ARCHIVE), exist_ok=True)
    write_header = not os.path.exists(DISPATCH_ARCHIVE)
    with open(DISPATCH_ARCHIVE, "a", newline="") as f:
        fields = ["timestamp", "outbox_id", "target_channel", "adapter_name",
                  "mode", "status", "external_ref", "error"]
        writer = csv.DictWriter(f, fieldnames=fields)
        if write_header:
            writer.writeheader()
        writer.writerows(entries)


def dispatch_one(conn, event, registry, is_dry_run):
    """Dispatch a single event via its resolved adapter.

    Args:
        conn: SQLite connection (for write_result on failure to resolve).
        event: dict-like row from event_outbox.
        registry: Adapter registry dict from load_adapter_registry().
        is_dry_run: If True, call adapter.dry_run(); else adapter.dispatch().

    Returns:
        dict with keys: status, adapter_name, result (the adapter's return dict),
                        error (if adapter resolution failed).
    """
    target = event["target_channel"]
    outbox_id = event["outbox_id"]

    try:
        adapter = resolve_adapter(target, registry, dry_run=is_dry_run)
    except (ValueError, ImportError, AttributeError) as e:
        return {
            "status": "failed",
            "adapter_name": None,
            "error": str(e),
            "result": {"status": "failed", "external_ref": None, "error": str(e)},
        }

    adapter_name = adapter.__class__.__name__
    event_dict = dict(event)

    try:
        result = adapter.dry_run(event_dict) if is_dry_run else adapter.dispatch(event_dict)
    except Exception as e:
        result = {"status": "failed", "external_ref": None, "error": str(e)}

    return {
        "status": result.get("status", "failed"),
        "adapter_name": adapter_name,
        "error": result.get("error"),
        "result": result,
    }
