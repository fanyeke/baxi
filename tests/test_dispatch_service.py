import sqlite3
import pytest
import os
import sys
from unittest.mock import patch

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', 'scripts'))

from services.dispatch_service import dispatch_one, fetch_pending

_SCHEMA = """
CREATE TABLE event_outbox (
    outbox_id TEXT PRIMARY KEY,
    event_type TEXT,
    source_type TEXT,
    source_id TEXT,
    payload_json TEXT,
    target_channel TEXT,
    status TEXT DEFAULT 'pending',
    created_at TEXT,
    dispatch_attempts INTEGER DEFAULT 0,
    last_dispatch_at TEXT,
    external_ref TEXT,
    adapter_name TEXT,
    error_message TEXT
);
"""


def _make_conn():
    conn = sqlite3.connect(":memory:")
    conn.row_factory = sqlite3.Row
    conn.execute(_SCHEMA)
    conn.commit()
    return conn


def _seed_event(conn, outbox_id="test-evt-1", target_channel="local_cli",
                dispatch_attempts=0, status="pending", last_dispatch_at=None):
    conn.execute(
        "INSERT INTO event_outbox "
        "(outbox_id, event_type, source_type, source_id, payload_json, "
        "target_channel, status, created_at, dispatch_attempts, last_dispatch_at) "
        "VALUES (?,?,?,?,?,?,?,?,?,?)",
        (outbox_id, "alert", "test", "t1",
         '{"rule_id":"gmv_drop","metric_name":"gmv","current_value":1000}',
         target_channel, status, "2026-01-01T00:00:00",
         dispatch_attempts, last_dispatch_at)
    )
    conn.commit()


def _read_event(conn, outbox_id="test-evt-1"):
    row = conn.execute(
        "SELECT * FROM event_outbox WHERE outbox_id=?", (outbox_id,)
    ).fetchone()
    return dict(row) if row else None


class _DummyAdapter:
    def __init__(self, dry_run=False):
        pass

    def dry_run(self, event):
        return {"status": "dispatched", "external_ref": "dry-run-ref", "error": None, "message": "dry run ok"}

    def dispatch(self, event):
        return {"status": "dispatched", "external_ref": "real-ref-123", "error": None, "message": "ok"}


def _make_dummy_adapter(target, registry, dry_run=False):
    return _DummyAdapter(dry_run=dry_run)


class TestDryRunPurity:

    def test_dry_run_does_not_increment_attempts(self):
        conn = _make_conn()
        _seed_event(conn)
        events = fetch_pending(conn, channel="local_cli")

        with patch("services.dispatch_service.resolve_adapter", side_effect=_make_dummy_adapter):
            dispatch_one(conn, events[0], registry={}, is_dry_run=True)

        event = _read_event(conn)
        assert event["dispatch_attempts"] == 0, \
            f"dry-run incremented dispatch_attempts to {event['dispatch_attempts']}"
        conn.close()

    def test_dry_run_does_not_change_status(self):
        conn = _make_conn()
        _seed_event(conn)
        events = fetch_pending(conn, channel="local_cli")

        with patch("services.dispatch_service.resolve_adapter", side_effect=_make_dummy_adapter):
            dispatch_one(conn, events[0], registry={}, is_dry_run=True)

        event = _read_event(conn)
        assert event["status"] == "pending", \
            f"dry-run changed status to {event['status']}"
        conn.close()

    def test_dry_run_does_not_set_last_dispatch_at(self):
        conn = _make_conn()
        _seed_event(conn)
        events = fetch_pending(conn, channel="local_cli")

        with patch("services.dispatch_service.resolve_adapter", side_effect=_make_dummy_adapter):
            dispatch_one(conn, events[0], registry={}, is_dry_run=True)

        event = _read_event(conn)
        assert event["last_dispatch_at"] is None, \
            f"dry-run set last_dispatch_at to {event['last_dispatch_at']}"
        conn.close()

    def test_real_dispatch_does_increment_attempts(self):
        conn = _make_conn()
        _seed_event(conn)
        events = fetch_pending(conn, channel="local_cli")

        with patch("services.dispatch_service.resolve_adapter", side_effect=_make_dummy_adapter):
            dispatch_one(conn, events[0], registry={}, is_dry_run=False)

        event = _read_event(conn)
        assert event["dispatch_attempts"] == 1, \
            f"real dispatch did not increment dispatch_attempts, got {event['dispatch_attempts']}"
        assert event["last_dispatch_at"] is not None, \
            "real dispatch did not set last_dispatch_at"
        assert event["status"] == "dispatched", \
            f"real dispatch status should be 'dispatched', got {event['status']}"
        conn.close()
