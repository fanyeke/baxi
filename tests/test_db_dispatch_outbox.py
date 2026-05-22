import pytest, sys, os, sqlite3, json, datetime, uuid, subprocess

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', 'scripts'))
import config

DB = config.DB_PATH
PAYLOAD = json.dumps({"rule_id":"gmv_drop","metric_name":"gmv","current_value":1000,"baseline_value":1500,"severity":"high","owner_role":"business_ops"})


def _insert_event(target_channel, status="pending", dispatch_attempts=0):
    conn = sqlite3.connect(DB)
    oid = f"test-dispatch-{uuid.uuid4().hex[:8]}"
    created = "1970-01-01T00:00:00"
    conn.execute("INSERT INTO event_outbox (outbox_id, event_type, source_type, source_id, payload_json, target_channel, status, created_at, dispatch_attempts) VALUES (?,?,?,?,?,?,?,?,?)",
        (oid, "alert", "test", "t1", PAYLOAD, target_channel, status, created, dispatch_attempts))
    conn.commit()
    conn.close()
    return oid


def _cleanup(oid):
    conn = sqlite3.connect(DB)
    conn.execute("DELETE FROM event_outbox WHERE outbox_id=?", (oid,))
    conn.commit()
    conn.close()


def _run(cmd_args):
    return subprocess.run(
        [sys.executable, "scripts/db_dispatch_outbox.py"] + cmd_args,
        capture_output=True, text=True, cwd=config.PROJECT_ROOT)


class TestDispatcherDryRun:
    def test_dry_run_exits_zero(self):
        r = _run(["--dry-run", "--limit", "1"])
        assert r.returncode == 0

    def test_dry_run_no_status_change(self):
        oid = _insert_event("feishu_cli")
        try:
            r = _run(["--dry-run", "--channel", "feishu_cli", "--limit", "1"])
            conn = sqlite3.connect(DB)
            status = conn.execute("SELECT status FROM event_outbox WHERE outbox_id=?", (oid,)).fetchone()[0]
            conn.close()
            assert status == "pending"
            assert "dispatched" not in r.stdout.split("[db_dispatch_outbox] Done")[0]
        finally:
            _cleanup(oid)

    def test_channel_filter(self):
        oid = _insert_event("feishu_cli")
        try:
            r = _run(["--dry-run", "--channel", "manual", "--limit", "1"])
            assert oid not in r.stdout
            assert "manual" in r.stdout.lower() or "Found 0" in r.stdout
        finally:
            _cleanup(oid)


class TestDispatcherEdgeCases:
    def test_max_attempts_skipped(self):
        oid = _insert_event("local_cli", dispatch_attempts=3)
        try:
            r = _run(["--dry-run", "--channel", "local_cli", "--limit", "1"])
            assert r.returncode == 0
            assert oid not in r.stdout
        finally:
            _cleanup(oid)

    def test_limit_enforced(self):
        ids = [_insert_event("local_cli") for _ in range(3)]
        try:
            r = _run(["--dry-run", "--channel", "local_cli", "--limit", "2"])
            assert "Found 2" in r.stdout
        finally:
            for oid in ids:
                _cleanup(oid)

    def test_audit_log_written(self):
        r = _run(["--dry-run", "--limit", "1"])
        audit = os.path.join(config.SYSTEM_DIR, "dispatch_archive.csv")
        assert os.path.exists(audit)

    def test_optimistic_lock_skip(self):
        oid = _insert_event("local_cli", status="dispatching")
        try:
            r = _run(["--apply", "--channel", "local_cli", "--limit", "1"])
            assert r.returncode == 0
            assert oid not in r.stdout
        finally:
            _cleanup(oid)


class TestAdapterResolver:
    def test_feishu_adapter_instantiated(self):
        r = _run(["--dry-run", "--channel", "feishu_cli", "--limit", "1"])
        assert r.returncode == 0

    def test_manual_adapter_instantiated(self):
        r = _run(["--dry-run", "--channel", "manual", "--limit", "1"])
        assert r.returncode == 0

    def test_all_four_adapters_dispatchable(self):
        oids = []
        for ch in ["feishu_cli", "github_issue", "local_cli", "manual"]:
            oids.append(_insert_event(ch))
        try:
            r = _run(["--dry-run", "--limit", "4"])
            assert r.returncode == 0
            for oid in oids:
                assert oid in r.stdout, f"{oid} not found in dispatch output"
        finally:
            for oid in oids:
                _cleanup(oid)
