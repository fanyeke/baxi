"""Qoder context API integration tests — validates GET /api/v1/qoder/context."""
import os
import sqlite3
import pytest

from core import config
from fastapi.testclient import TestClient

os.environ["API_BEARER_TOKEN"] = "test-token-for-baxi-ci-tests-only-32ch"


@pytest.fixture(scope="function")
def client(tmp_path, monkeypatch):
    test_db = str(tmp_path / "test_olist_ops.db")
    _seed_test_db(test_db)
    monkeypatch.setattr(config, "DB_PATH", test_db)
    from services import db_service
    monkeypatch.setattr(db_service, "DB_PATH", test_db)
    from api.main import create_app
    app = create_app()
    return TestClient(app)


def _seed_test_db(path):
    conn = sqlite3.connect(path)
    conn.executescript("""
        CREATE TABLE IF NOT EXISTS pipeline_runs (
            run_id TEXT PRIMARY KEY, run_type TEXT, mode TEXT, status TEXT,
            started_at TEXT, finished_at TEXT, input_count INTEGER, output_count INTEGER, error_message TEXT);
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
        INSERT OR IGNORE INTO pipeline_runs VALUES ('run_001','ingest','dry_run','completed','2024-01-01T00:00:00',NULL,0,0,NULL);
        INSERT OR IGNORE INTO alert_events VALUES
            ('alert_001','rule_1','2024-01-01','high','gmv','global','global',1000.0,2000.0,NULL,NULL,NULL,NULL,'ops','new','2024-01-01T00:00:00',NULL,NULL,NULL),
            ('alert_002','rule_2','2024-01-01','medium','latency','seller','seller-1',0.3,0.1,NULL,NULL,NULL,NULL,'ops','new','2024-01-01T00:00:00',NULL,NULL,NULL),
            ('alert_003','rule_3','2024-01-01','high','error_rate','service','svc-1',0.8,0.2,NULL,NULL,NULL,NULL,'ops','acknowledged','2024-01-01T00:00:00',NULL,NULL,NULL);
        INSERT OR IGNORE INTO action_tasks VALUES
            ('task_001',NULL,NULL,'Task 1',NULL,'heuristic','ops',NULL,'high',NULL,'open',NULL,NULL,'2024-01-01T00:00:00',NULL,NULL),
            ('task_002',NULL,NULL,'Task 2',NULL,'heuristic','ops',NULL,'medium',NULL,'open',NULL,NULL,'2024-01-01T00:00:00',NULL,NULL);
        INSERT OR IGNORE INTO event_outbox VALUES
            ('outbox_001','alert','seller','seller_001','{}','feishu_cli','pending','2024-01-01T00:00:00',NULL,NULL,0,NULL,NULL,NULL),
            ('outbox_002','alert','seller','seller_002','{}','manual','pending','2024-01-01T00:00:00',NULL,NULL,0,NULL,NULL,NULL);
    """)
    conn.commit()
    conn.close()


HEADERS = {"Authorization": "Bearer test-token-for-baxi-ci-tests-only-32ch"}


class TestContext:
    def test_context_requires_auth(self, client):
        r = client.get("/api/v1/qoder/context")
        assert r.status_code == 401
        assert "AUTH_REQUIRED" in str(r.json())

    def test_context_returns_system_summary(self, client):
        r = client.get("/api/v1/qoder/context", headers=HEADERS)
        assert r.status_code == 200
        data = r.json()
        assert "summary" in data
        assert data["summary"]["total_alerts"] >= 2
        assert data["summary"]["total_open_tasks"] >= 2
        assert data["summary"]["total_pending_outbox"] >= 2

    def test_context_returns_top_alerts(self, client):
        r = client.get("/api/v1/qoder/context", headers=HEADERS)
        assert r.status_code == 200
        data = r.json()
        assert "top_alerts" in data
        assert isinstance(data["top_alerts"], list)
        assert len(data["top_alerts"]) > 0

    def test_context_returns_open_tasks(self, client):
        r = client.get("/api/v1/qoder/context", headers=HEADERS)
        assert r.status_code == 200
        data = r.json()
        assert "open_tasks" in data
        assert isinstance(data["open_tasks"], list)

    def test_context_returns_pending_outbox(self, client):
        r = client.get("/api/v1/qoder/context", headers=HEADERS)
        assert r.status_code == 200
        data = r.json()
        assert "pending_outbox" in data
        assert isinstance(data["pending_outbox"], list)

    def test_context_limits_result_size(self, client):
        r = client.get(
            "/api/v1/qoder/context?limit_alerts=1&limit_tasks=1&limit_outbox=1",
            headers=HEADERS,
        )
        assert r.status_code == 200
        data = r.json()
        assert len(data["top_alerts"]) <= 1
        assert len(data["open_tasks"]) <= 1
        assert len(data["pending_outbox"]) <= 1

    def test_context_does_not_mutate_db(self, client):
        r_before = client.get("/api/v1/qoder/context", headers=HEADERS)
        assert r_before.status_code == 200
        r_after = client.get("/api/v1/qoder/context", headers=HEADERS)
        assert r_after.status_code == 200
        assert r_before.json()["summary"] == r_after.json()["summary"]

    def test_context_filter_by_severity(self, client):
        r = client.get(
            "/api/v1/qoder/context?severity=high",
            headers=HEADERS,
        )
        assert r.status_code == 200
        data = r.json()
        for alert in data["top_alerts"]:
            assert alert["severity"] == "high"
