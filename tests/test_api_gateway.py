"""API gateway integration tests — validates all 6 P0 endpoints."""
import os
import sys
import sqlite3
import pytest

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "scripts"))

from scripts import config
from fastapi.testclient import TestClient

os.environ["API_BEARER_TOKEN"] = "test-token"


@pytest.fixture(scope="function")
def client(tmp_path, monkeypatch):
    test_db = str(tmp_path / "test_olist_ops.db")
    _seed_test_db(test_db)
    monkeypatch.setattr(config, "DB_PATH", test_db)
    from api.main import create_app
    app = create_app()
    return TestClient(app)


def _seed_test_db(path):
    conn = sqlite3.connect(path)
    conn.executescript("""
        CREATE TABLE IF NOT EXISTS pipeline_runs (
            run_id TEXT PRIMARY KEY, run_type TEXT, mode TEXT, status TEXT,
            started_at TEXT, finished_at TEXT, input_count INTEGER, output_count INTEGER, error_message TEXT);
        CREATE TABLE IF NOT EXISTS ingestion_batches (
            batch_id TEXT PRIMARY KEY, status TEXT, created_at TEXT);
        CREATE TABLE IF NOT EXISTS alert_events (
            event_id TEXT PRIMARY KEY, rule_id TEXT, event_date TEXT,
            severity TEXT, metric_name TEXT, object_type TEXT DEFAULT 'global',
            object_id TEXT DEFAULT 'global', current_value REAL, baseline_value REAL,
            change_rate REAL, sample_size INTEGER, evidence_json TEXT, description TEXT,
            owner_role TEXT, status TEXT DEFAULT 'new', created_at TEXT,
            affected_orders INTEGER, affected_gmv REAL, impact_score REAL);
        CREATE TABLE IF NOT EXISTS action_tasks (
            task_id TEXT PRIMARY KEY, recommendation_id TEXT, event_id TEXT,
            task_title TEXT, task_description TEXT, task_source TEXT,
            owner_role TEXT, owner_user_id TEXT, priority TEXT DEFAULT 'medium',
            due_at TEXT, status TEXT DEFAULT 'todo', feedback TEXT,
            completed_at TEXT, created_at TEXT,
            target_object_type TEXT, target_object_id TEXT);
        CREATE TABLE IF NOT EXISTS event_outbox (
            outbox_id TEXT PRIMARY KEY, event_type TEXT, source_type TEXT,
            source_id TEXT, payload_json TEXT, target_channel TEXT,
            status TEXT DEFAULT 'pending', created_at TEXT, processed_at TEXT,
            error_message TEXT, dispatch_attempts INTEGER DEFAULT 0,
            last_dispatch_at TEXT, external_ref TEXT, adapter_name TEXT);
        INSERT OR IGNORE INTO pipeline_runs VALUES ('run-1','full','full','completed','2026-01-01T00:00:00',NULL,0,0,NULL);
        INSERT OR IGNORE INTO alert_events VALUES
            ('alert-1','gmv_drop','2026-01-01','high','gmv','global','global',1000.0,2000.0,NULL,NULL,NULL,NULL,'business_ops','new','2026-01-01',NULL,NULL,NULL),
            ('alert-2','late_delivery','2026-01-02','medium','late_delivery_rate','seller','seller-123',0.3,0.1,NULL,NULL,NULL,NULL,'seller_ops','new','2026-01-02',NULL,NULL,NULL);
        INSERT OR IGNORE INTO action_tasks VALUES
            ('task-1','rec-1','alert-2','Investigate seller delay','Check late delivery for seller-123','heuristic','seller_ops',NULL,'high',NULL,'todo',NULL,NULL,'2026-01-01',NULL,NULL),
            ('task-2','rec-2',NULL,'Review cancel policy',NULL,'heuristic','logistics_ops',NULL,'medium',NULL,'done',NULL,'2026-01-01','2026-01-01',NULL,NULL);
        INSERT OR IGNORE INTO event_outbox VALUES
            ('out-1','metric_alert','rule_engine','alert-1','{"rule_id":"gmv_drop","severity":"high"}','manual','pending','2026-01-01',NULL,NULL,0,NULL,NULL,NULL);
    """)
    conn.commit()
    conn.close()


# ── Health ─────────────────────────────────────────────────────────────

class TestHealth:
    def test_health_ok_no_auth(self, client):
        r = client.get("/api/v1/health")
        assert r.status_code == 200
        assert r.json()["status"] == "ok"

    def test_health_has_version(self, client):
        r = client.get("/api/v1/health")
        assert r.json()["version"] == "0.5.1"

    def test_health_db_connected(self, client):
        r = client.get("/api/v1/health")
        assert r.json()["db_connected"] is True


# ── Auth ───────────────────────────────────────────────────────────────

class TestAuth:
    def test_status_requires_auth(self, client):
        r = client.get("/api/v1/status")
        assert r.status_code == 401
        assert "AUTH_REQUIRED" in str(r.json())

    def test_status_ok_with_token(self, client, auth_headers):
        r = client.get("/api/v1/status", headers=auth_headers)
        assert r.status_code == 200

    def test_wrong_token_returns_error(self, client):
        r = client.get("/api/v1/status", headers={"Authorization": "Bearer wrong"})
        assert r.status_code == 403


# ── Status ─────────────────────────────────────────────────────────────

class TestStatus:
    def test_status_has_database_info(self, client, auth_headers):
        r = client.get("/api/v1/status", headers=auth_headers)
        data = r.json()
        assert "database" in data
        assert "tables" in data["database"]

    def test_status_has_pipeline_run(self, client, auth_headers):
        r = client.get("/api/v1/status", headers=auth_headers)
        data = r.json()
        assert "last_pipeline_run" in data


# ── Alerts ─────────────────────────────────────────────────────────────

class TestAlerts:
    def test_alerts_returns_list(self, client, auth_headers):
        r = client.get("/api/v1/alerts?limit=10", headers=auth_headers)
        assert r.status_code == 200
        assert "items" in r.json()

    def test_alerts_filter_severity(self, client, auth_headers):
        r = client.get("/api/v1/alerts?severity=high&limit=10", headers=auth_headers)
        for item in r.json()["items"]:
            assert item["severity"] == "high"


# ── Tasks ──────────────────────────────────────────────────────────────

class TestTasks:
    def test_tasks_returns_list(self, client, auth_headers):
        r = client.get("/api/v1/tasks?limit=10", headers=auth_headers)
        assert r.status_code == 200
        assert "items" in r.json()

    def test_tasks_filter_status(self, client, auth_headers):
        r = client.get("/api/v1/tasks?status=todo&limit=10", headers=auth_headers)
        for item in r.json()["items"]:
            assert item["status"] == "todo"


# ── Outbox ─────────────────────────────────────────────────────────────

class TestOutbox:
    def test_outbox_returns_list(self, client, auth_headers):
        r = client.get("/api/v1/outbox?limit=10", headers=auth_headers)
        assert r.status_code == 200

    def test_outbox_filter_channel(self, client, auth_headers):
        r = client.get("/api/v1/outbox?channel=manual&limit=10", headers=auth_headers)
        for item in r.json()["items"]:
            assert item["target_channel"] == "manual"


# ── Dispatch ───────────────────────────────────────────────────────────

class TestDispatch:
    def test_dispatch_defaults_dry_run(self, client, auth_headers):
        r = client.post("/api/v1/outbox/dispatch", json={}, headers=auth_headers)
        assert r.status_code == 200
        assert r.json()["dry_run"] is True

    def test_dispatch_manages_event(self, client, auth_headers):
        r = client.post("/api/v1/outbox/dispatch",
                        json={"channel": "manual", "limit": 1},
                        headers=auth_headers)
        assert r.status_code == 200
        assert r.json()["processed"] >= 0


# ── Error Contract ─────────────────────────────────────────────────────

class TestErrors:
    def test_auth_error_has_diagnosis(self, client):
        r = client.get("/api/v1/status")
        data = r.json()
        assert "diagnosis" in data
        assert "suggested_action" in data
        assert "error_code" in data

    def test_validation_error_structured(self, client, auth_headers):
        r = client.get("/api/v1/alerts?limit=-1", headers=auth_headers)
        assert r.status_code == 422
        data = r.json()
        assert "error_code" in data


# ── Integration ─────────────────────────────────────────────────────────

class TestIntegration:
    def test_full_flow_health_to_alerts(self, client, auth_headers):
        h = client.get("/api/v1/health")
        assert h.status_code == 200
        s = client.get("/api/v1/status", headers=auth_headers)
        assert s.status_code == 200
        a = client.get("/api/v1/alerts?limit=5", headers=auth_headers)
        assert a.status_code == 200

    def test_full_flow_tasks_to_dispatch(self, client, auth_headers):
        t = client.get("/api/v1/tasks?limit=5", headers=auth_headers)
        assert t.status_code == 200
        o = client.get("/api/v1/outbox?limit=5", headers=auth_headers)
        assert o.status_code == 200
        d = client.post("/api/v1/outbox/dispatch",
                        json={"channel": "manual", "limit": 1},
                        headers=auth_headers)
        assert d.status_code == 200

    def test_x_request_id_header(self, client):
        r = client.get("/api/v1/health")
        assert "x-request-id" in r.headers

    def test_invalid_token_format(self, client):
        r = client.get("/api/v1/status", headers={"Authorization": "NoBearer token"})
        assert r.status_code == 401

    def test_empty_db_handled_gracefully(self, client, auth_headers):
        r = client.get("/api/v1/alerts?limit=1000", headers=auth_headers)
        assert r.status_code == 200
        assert isinstance(r.json()["items"], list)

    def test_dispatch_apply_flag(self, client, auth_headers):
        r = client.post("/api/v1/outbox/dispatch",
                        json={"channel": "manual", "limit": 1, "apply": True},
                        headers=auth_headers)
        assert r.status_code == 200
        assert r.json()["dry_run"] is False
