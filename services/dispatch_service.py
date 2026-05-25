"""
Dispatch service for event_outbox processing.

Extracted from scripts/db_dispatch_outbox.py to enable reuse
across CLI scripts and scheduled jobs.
"""

import csv
import datetime
import os

from adapters.base import resolve_adapter
from core import config

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


def get_outbox_with_count(conn, status=None, channel=None, limit=100):
    """Fetch outbox items with total count for a given filter.

    Returns:
        (items: list[dict], total: int) tuple.
    """
    query = (
        "SELECT outbox_id, event_type, source_type, source_id, "
        "target_channel, status, dispatch_attempts, last_dispatch_at, created_at "
        "FROM event_outbox"
    )
    params = []
    conditions = []
    if status:
        conditions.append("status = ?")
        params.append(status)
    if channel:
        conditions.append("target_channel = ?")
        params.append(channel)
    if conditions:
        query += " WHERE " + " AND ".join(conditions)
    query += " ORDER BY created_at DESC LIMIT ?"
    params.append(limit)

    rows = conn.execute(query, params).fetchall()
    items = [dict(r) for r in rows]

    count_query = "SELECT COUNT(*) FROM event_outbox"
    if conditions:
        count_query += " WHERE " + " AND ".join(conditions)
    total = conn.execute(count_query, params[:-1]).fetchone()[0]

    return items, total


def claim_event(conn, outbox_id):
    """Atomically claim a pending event for dispatch.

    Args:
        conn: SQLite connection.
        outbox_id: The outbox record ID to claim.

    Returns:
        True if the event was successfully claimed (was pending), False otherwise.
    """
    cur = conn.execute(
        "UPDATE event_outbox SET status = 'dispatching' WHERE outbox_id = ? AND status = 'pending'",
        (outbox_id,))
    return cur.rowcount > 0


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
    if is_dry_run:
        return

    now = datetime.datetime.now().isoformat()
    status = result.get("status", "failed")
    external_ref = result.get("external_ref")
    error = result.get("error")

    if status in ("dispatched", "skipped"):
        db_status = status
    else:
        current = conn.execute(
            "SELECT dispatch_attempts FROM event_outbox WHERE outbox_id = ?",
            (outbox_id,)
        ).fetchone()
        if current and current[0] + 1 < MAX_ATTEMPTS:
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
        dict with keys: status, adapter_name, message, external_ref, error.
    """
    target = event["target_channel"]
    outbox_id = event["outbox_id"]

    # Claim the event atomically (skip in dry-run)
    if not is_dry_run:
        if not claim_event(conn, outbox_id):
            return {
                "status": "skipped",
                "adapter_name": None,
                "error": "already claimed",
                "message": None,
                "external_ref": None,
            }

    try:
        adapter = resolve_adapter(target, registry, dry_run=is_dry_run)
    except (ValueError, ImportError, AttributeError) as e:
        result_fail = {"status": "failed", "external_ref": None, "error": str(e), "message": None}
        write_result(conn, outbox_id, result_fail, None, is_dry_run)
        return {
            "status": "failed",
            "adapter_name": None,
            "error": str(e),
            "message": None,
            "external_ref": None,
        }

    adapter_name = adapter.__class__.__name__
    event_dict = dict(event)

    try:
        result = adapter.dry_run(event_dict) if is_dry_run else adapter.dispatch(event_dict)
    # Narrowed based on known adapter implementations:
    #   KeyError/ValueError — config loading in FeishuAdapter
    #   OSError — file I/O in LocalCLIAdapter
    #   RuntimeError — FeishuClient._raw_* retry exhaustion
    #   ConnectionError — network failures in FeishuClient
    except (KeyError, OSError, ValueError, RuntimeError, ConnectionError) as e:
        result = {"status": "failed", "external_ref": None, "error": str(e), "message": None}

    write_result(conn, outbox_id, result, adapter_name, is_dry_run)

    return {
        "status": result.get("status", "failed"),
        "adapter_name": adapter_name,
        "error": result.get("error"),
        "message": result.get("message"),
        "external_ref": result.get("external_ref"),
    }
