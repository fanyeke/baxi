"""Qoder reports API integration tests — validates POST /api/v1/qoder/reports."""
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
        CREATE TABLE IF NOT EXISTS qoder_runs (
            run_id TEXT PRIMARY KEY, run_type TEXT, mode TEXT, status TEXT,
            started_at TEXT, finished_at TEXT, request_id TEXT,
            actor TEXT DEFAULT 'qoder', can_apply INTEGER DEFAULT 0, error_message TEXT);
        CREATE TABLE IF NOT EXISTS qoder_reports (
            report_id TEXT PRIMARY KEY, run_id TEXT, run_type TEXT, summary TEXT,
            findings_json TEXT, recommended_human_actions_json TEXT,
            risk_level TEXT, used_endpoints_json TEXT,
            no_apply_performed INTEGER NOT NULL DEFAULT 1,
            business_side_effect INTEGER NOT NULL DEFAULT 0,
            created_at TEXT, request_id TEXT);
        INSERT OR IGNORE INTO pipeline_runs VALUES ('run_001','ingest','dry_run','completed','2024-01-01T00:00:00',NULL,0,0,NULL);
        INSERT OR IGNORE INTO alert_events VALUES
            ('alert_001','rule_1','2024-01-01','high','gmv','global','global',1000.0,2000.0,NULL,NULL,NULL,NULL,'ops','new','2024-01-01T00:00:00',NULL,NULL,NULL),
            ('alert_002','rule_2','2024-01-01','medium','latency','seller','seller-1',0.3,0.1,NULL,NULL,NULL,NULL,'ops','new','2024-01-01T00:00:00',NULL,NULL,NULL);
        INSERT OR IGNORE INTO action_tasks VALUES
            ('task_001',NULL,NULL,'Task 1',NULL,'heuristic','ops',NULL,'high',NULL,'open',NULL,NULL,'2024-01-01T00:00:00',NULL,NULL);
        INSERT OR IGNORE INTO event_outbox VALUES
            ('outbox_001','alert','seller','seller_001','{}','feishu_cli','pending','2024-01-01T00:00:00',NULL,NULL,0,NULL,NULL,NULL);
    """)
    conn.commit()
    conn.close()


HEADERS = {"Authorization": "Bearer test-token-for-baxi-ci-tests-only-32ch"}

VALID_REPORT = {
    "run_type": "readonly_check",
    "summary": "Test report",
    "findings": [{"type": "alert", "severity": "high"}],
    "recommended_human_actions": ["Review alerts"],
    "risk_level": "medium",
    "used_endpoints": ["/qoder/capabilities"],
    "no_apply_performed": True,
}


class TestReports:
    def test_report_requires_auth(self, client):
        r = client.post("/api/v1/qoder/reports", json=VALID_REPORT)
        assert r.status_code == 401
        assert "AUTH_REQUIRED" in str(r.json())

    def test_report_records_readonly_report(self, client):
        r = client.post(
            "/api/v1/qoder/reports",
            json=VALID_REPORT,
            headers=HEADERS,
        )
        assert r.status_code == 200
        data = r.json()
        assert data["status"] == "recorded"
        assert "report_id" in data

    def test_report_rejects_no_apply_false(self, client):
        payload = {**VALID_REPORT, "no_apply_performed": False}
        r = client.post(
            "/api/v1/qoder/reports",
            json=payload,
            headers=HEADERS,
        )
        assert r.status_code == 400
        assert "VALIDATION_ERROR" in str(r.json())
        assert "no_apply_performed" in str(r.json())

    def test_report_sets_business_side_effect_false(self, client):
        r = client.post(
            "/api/v1/qoder/reports",
            json=VALID_REPORT,
            headers=HEADERS,
        )
        assert r.status_code == 200
        data = r.json()
        assert data["business_side_effect"] is False

    def test_report_does_not_modify_alerts_tasks_outbox(self, client):
        r_get = client.get("/api/v1/qoder/context", headers=HEADERS)
        assert r_get.status_code == 200
        before = r_get.json()["summary"]

        client.post(
            "/api/v1/qoder/reports",
            json=VALID_REPORT,
            headers=HEADERS,
        )

        r_get_after = client.get("/api/v1/qoder/context", headers=HEADERS)
        assert r_get_after.status_code == 200
        after = r_get_after.json()["summary"]
        assert before == after

    def test_report_idempotent(self, client):
        report_id = "idempotent-test-001"
        payload = {**VALID_REPORT, "report_id": report_id}

        r1 = client.post(
            "/api/v1/qoder/reports",
            json=payload,
            headers=HEADERS,
        )
        assert r1.status_code == 200

        r2 = client.post(
            "/api/v1/qoder/reports",
            json=payload,
            headers=HEADERS,
        )
        assert r2.status_code == 200

    def test_report_deterministic_id_without_client_report_id(self, client):
        """Same request body should produce same report_id when client doesn't provide one."""
        payload = {**VALID_REPORT}

        r1 = client.post(
            "/api/v1/qoder/reports",
            json=payload,
            headers=HEADERS,
        )
        assert r1.status_code == 200
        report_id_1 = r1.json()["report_id"]

        r2 = client.post(
            "/api/v1/qoder/reports",
            json=payload,
            headers=HEADERS,
        )
        assert r2.status_code == 200
        report_id_2 = r2.json()["report_id"]

        assert report_id_1 == report_id_2

    def test_report_validates_required_fields(self, client):
        payload = {"run_type": "test"}
        r = client.post(
            "/api/v1/qoder/reports",
            json=payload,
            headers=HEADERS,
        )
        assert r.status_code == 422
        assert "VALIDATION_ERROR" in str(r.json())
