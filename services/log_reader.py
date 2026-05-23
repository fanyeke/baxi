"""JSONL tail-read and CSV parsing for the v0.5.1 Logs API.

Provides functions to read and filter JSON log files and CSV audit logs
from the end of the file (most recent first). All functions return [] for
missing or empty files.
"""

import csv
import json
import logging
import os
from collections import deque
from typing import Optional

logger = logging.getLogger("api.log_reader")


def _tail_jsonl(filepath: str, limit: int) -> list[dict]:
    """Read the last N JSON lines from a file (tail-N behavior).

    Seeks to end of file, reads backwards line-by-line, parses each line
    as JSON. Skips malformed lines with a warning. Returns [] for missing
    or empty files.

    Args:
        filepath: Path to JSONL file.
        limit: Maximum number of entries to return.

    Returns:
        List of parsed dicts, newest first.
    """
    if not os.path.exists(filepath):
        return []
    if os.path.getsize(filepath) == 0:
        return []

    entries = deque()
    try:
        with open(filepath, "r") as f:
            for line in reversed(list(f)):
                line = line.strip()
                if not line:
                    continue
                try:
                    obj = json.loads(line)
                    entries.appendleft(obj)
                except json.JSONDecodeError:
                    logger.warning("Skipping malformed JSON line in %s", filepath)
                    continue
                if len(entries) >= limit:
                    break
    except OSError as e:
        logger.warning("Error reading %s: %s", filepath, e)
        return []

    return list(reversed(entries))[:limit]


def read_log_errors(
    error_log_path: str,
    request_id: Optional[str] = None,
    limit: int = 100,
) -> list[dict]:
    """Parse error log JSONL, filter by request_id, return last N entries.

    Args:
        error_log_path: Path to error.log JSONL file.
        request_id: Optional request_id filter.
        limit: Max entries to return (default 100, max 500).

    Returns:
        List of parsed dicts sorted by ts descending.
    """
    limit = min(limit, 500)
    entries = _tail_jsonl(error_log_path, limit)

    if request_id:
        entries = [e for e in entries if e.get("request_id") == request_id]

    return entries


def read_log_recent(api_log_path: str, limit: int = 50) -> list[dict]:
    """Parse API log JSONL, return last N entries.

    Args:
        api_log_path: Path to api.log JSONL file.
        limit: Max entries to return (default 50, max 500).

    Returns:
        List of parsed dicts sorted by ts descending.
    """
    limit = min(limit, 500)
    return _tail_jsonl(api_log_path, limit)


def read_audit_logs(
    audit_csv_path: str,
    outbox_id: Optional[str] = None,
    status: Optional[str] = None,
    limit: int = 100,
) -> list[dict]:
    """Parse CSV audit log, filter, return sorted by timestamp desc.

    Args:
        audit_csv_path: Path to CSV audit file.
        outbox_id: Optional outbox_id filter.
        status: Optional status filter.
        limit: Max entries to return (default 100, max 500).

    Returns:
        List of dicts sorted by timestamp descending.
    """
    if not os.path.exists(audit_csv_path):
        return []

    limit = min(limit, 500)
    entries = []

    try:
        with open(audit_csv_path, "r") as f:
            reader = csv.DictReader(f)
            for row in reader:
                entries.append(row)
    except OSError as e:
        logger.warning("Error reading audit CSV %s: %s", audit_csv_path, e)
        return []

    if outbox_id:
        entries = [e for e in entries if e.get("outbox_id") == outbox_id]
    if status:
        entries = [e for e in entries if e.get("status") == status]

    # Sort by timestamp descending (most recent first)
    entries.sort(key=lambda x: x.get("timestamp", ""), reverse=True)

    return entries[:limit]
