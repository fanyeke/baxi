"""TDD RED: Logs API tests — will fail until router is mounted."""
import json
import os

import pytest
from fastapi.testclient import TestClient

from api.main import create_app


@pytest.fixture
def app():
    return create_app()


@pytest.fixture
def client(app):
    return TestClient(app)


@pytest.fixture
def auth_headers():
    return {"Authorization": "Bearer test-token"}


@pytest.fixture
def tmp_log_dir(tmp_path):
    """Create temp directory with pre-populated log files."""
    log_dir = tmp_path / "logs" / "api"
    log_dir.mkdir(parents=True)

    # api.log — 5 JSON lines
    api_log = log_dir / "api.log"
    for i in range(5):
        api_log.write_text(
            json.dumps({
                "ts": f"2026-05-22 10:0{i}:00",
                "level": "INFO",
                "message": f"Request #{i}",
                "request_id": f"req-{i}",
                "method": "GET" if i % 2 == 0 else "POST",
                "path": "/api/v1/health" if i % 2 == 0 else "/api/v1/outbox/dispatch",
                "actor": "qoder",
            }) + "\n",
            encoding="utf-8",
        )

    # error.log — 3 JSON lines
    error_log = log_dir / "error.log"
    for i in range(3):
        error_log.write_text(
            json.dumps({
                "ts": f"2026-05-22 10:0{i}:00",
                "level": "ERROR",
                "message": f"Error #{i}",
                "request_id": f"err-req-{i}",
                "error_code": "AUTH_REQUIRED" if i % 2 == 0 else "DB_QUERY_FAILED",
                "diagnosis": f"Diagnosis {i}",
                "suggested_action": f"Action {i}",
                "actor": "unknown",
            }) + "\n",
            encoding="utf-8",
        )

    return str(log_dir)


class TestLogsErrors:
    """GET /api/v1/logs/errors"""

    def test_errors_returns_200_with_items(self, client, auth_headers, monkeypatch, tmp_log_dir):
        monkeypatch.setattr("api.routers.logs.LOG_DIR", tmp_log_dir)
        resp = client.get("/api/v1/logs/errors", headers=auth_headers)
        assert resp.status_code == 200
        data = resp.json()
        assert "items" in data
        assert "total" in data

    def test_errors_unauthorized_returns_401(self, client):
        resp = client.get("/api/v1/logs/errors")
        assert resp.status_code == 401

    def test_errors_empty_when_no_file(self, client, auth_headers, monkeypatch, tmp_path):
        monkeypatch.setattr("api.routers.logs.LOG_DIR", str(tmp_path))
        resp = client.get("/api/v1/logs/errors", headers=auth_headers)
        assert resp.status_code == 200
        assert resp.json()["items"] == []


class TestLogsAudit:
    """GET /api/v1/logs/audit"""

    def test_audit_returns_200(self, client, auth_headers):
        resp = client.get("/api/v1/logs/audit", headers=auth_headers)
        assert resp.status_code == 200
        data = resp.json()
        assert "items" in data

    def test_audit_unauthorized_returns_401(self, client):
        resp = client.get("/api/v1/logs/audit")
        assert resp.status_code == 401


class TestLogsRecent:
    """GET /api/v1/logs/recent"""

    def test_recent_returns_200(self, client, auth_headers, monkeypatch, tmp_log_dir):
        monkeypatch.setattr("api.routers.logs.LOG_DIR", tmp_log_dir)
        resp = client.get("/api/v1/logs/recent?limit=3", headers=auth_headers)
        assert resp.status_code == 200
        data = resp.json()
        assert "items" in data
        assert len(data["items"]) <= 3

    def test_recent_unauthorized_returns_401(self, client):
        resp = client.get("/api/v1/logs/recent")
        assert resp.status_code == 401
