import pytest
import sys
import os
import sqlite3
import json
import uuid
import subprocess

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', 'scripts'))
import config

PAYLOAD = json.dumps({"rule_id": "gmv_drop", "metric_name": "gmv", "current_value": 1000, "baseline_value": 1500, "severity": "high", "owner_role": "business_ops"})


def _insert_event(db_path, target_channel, status="pending", dispatch_attempts=0):
    conn = sqlite3.connect(db_path)
    oid = f"test-dispatch-{uuid.uuid4().hex[:8]}"
    created = "1970-01-01T00:00:00"
    conn.execute(
        "INSERT INTO event_outbox (outbox_id, event_type, source_type, source_id, payload_json, target_channel, status, created_at, dispatch_attempts) VALUES (?,?,?,?,?,?,?,?,?)",
        (oid, "alert", "test", "t1", PAYLOAD, target_channel, status, created, dispatch_attempts),
    )
    conn.commit()
    conn.close()
    return oid


def _cleanup(db_path, oid):
    conn = sqlite3.connect(db_path)
    conn.execute("DELETE FROM event_outbox WHERE outbox_id=?", (oid,))
    conn.commit()
    conn.close()


def _run(cmd_args, db_path=None):
    args = [sys.executable, "scripts/db_dispatch_outbox.py"] + cmd_args
    if db_path:
        args.extend(["--db", db_path])
    return subprocess.run(args, capture_output=True, text=True, cwd=config.PROJECT_ROOT)


class TestDispatcherDryRun:
    def test_dry_run_exits_zero(self, temp_db_path):
        r = _run(["--dry-run", "--limit", "1"], db_path=temp_db_path)
        assert r.returncode == 0

    def test_dry_run_no_status_change(self, temp_db_path):
        oid = _insert_event(temp_db_path, "feishu_cli")
        try:
            r = _run(["--dry-run", "--channel", "feishu_cli", "--limit", "1"], db_path=temp_db_path)
            conn = sqlite3.connect(temp_db_path)
            status = conn.execute("SELECT status FROM event_outbox WHERE outbox_id=?", (oid,)).fetchone()[0]
            conn.close()
            assert status == "pending"
            assert "dispatched" not in r.stdout.split("[db_dispatch_outbox] Done")[0]
        finally:
            _cleanup(temp_db_path, oid)

    def test_channel_filter(self, temp_db_path):
        oid = _insert_event(temp_db_path, "feishu_cli")
        try:
            r = _run(["--dry-run", "--channel", "manual", "--limit", "1"], db_path=temp_db_path)
            assert oid not in r.stdout
            assert "manual" in r.stdout.lower() or "Found 0" in r.stdout
        finally:
            _cleanup(temp_db_path, oid)


class TestDispatcherEdgeCases:
    def test_max_attempts_skipped(self, temp_db_path):
        oid = _insert_event(temp_db_path, "local_cli", dispatch_attempts=3)
        try:
            r = _run(["--dry-run", "--channel", "local_cli", "--limit", "1"], db_path=temp_db_path)
            assert r.returncode == 0
            assert oid not in r.stdout
        finally:
            _cleanup(temp_db_path, oid)

    def test_limit_enforced(self, temp_db_path):
        ids = [_insert_event(temp_db_path, "local_cli") for _ in range(3)]
        try:
            r = _run(["--dry-run", "--channel", "local_cli", "--limit", "2"], db_path=temp_db_path)
            assert "Found 2" in r.stdout
        finally:
            for oid in ids:
                _cleanup(temp_db_path, oid)

    def test_audit_log_written(self, temp_db_path):
        r = _run(["--dry-run", "--limit", "1"], db_path=temp_db_path)
        audit = os.path.join(config.SYSTEM_DIR, "dispatch_archive.csv")
        assert os.path.exists(audit)

    def test_optimistic_lock_skip(self, temp_db_path):
        oid = _insert_event(temp_db_path, "local_cli", status="dispatching")
        try:
            r = _run(["--apply", "--channel", "local_cli", "--limit", "1"], db_path=temp_db_path)
            assert r.returncode == 0
            assert oid not in r.stdout
        finally:
            _cleanup(temp_db_path, oid)


class TestAdapterResolver:
    def test_feishu_adapter_instantiated(self, temp_db_path):
        r = _run(["--dry-run", "--channel", "feishu_cli", "--limit", "1"], db_path=temp_db_path)
        assert r.returncode == 0

    def test_manual_adapter_instantiated(self, temp_db_path):
        r = _run(["--dry-run", "--channel", "manual", "--limit", "1"], db_path=temp_db_path)
        assert r.returncode == 0

    def test_all_four_adapters_dispatchable(self, temp_db_path):
        oids = []
        for ch in ["feishu_cli", "github_issue", "local_cli", "manual"]:
            oids.append(_insert_event(temp_db_path, ch))
        try:
            r = _run(["--dry-run", "--limit", "4"], db_path=temp_db_path)
            assert r.returncode == 0
            for oid in oids:
                assert oid in r.stdout, f"{oid} not found in dispatch output"
        finally:
            for oid in oids:
                _cleanup(temp_db_path, oid)
