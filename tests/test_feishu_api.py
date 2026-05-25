"""TDD RED: Feishu API tests — will fail until router is mounted."""
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
    return {"Authorization": "Bearer test-token-for-baxi-ci-tests-only-32ch"}


class TestFeishuExport:
    """POST /api/v1/feishu/export"""

    def test_export_defaults_dry_run(self, client, auth_headers):
        resp = client.post("/api/v1/feishu/export", json={}, headers=auth_headers)
        assert resp.status_code == 200
        data = resp.json()
        assert "status" in data

    def test_export_unauthorized_returns_401(self, client):
        resp = client.post("/api/v1/feishu/export", json={})
        assert resp.status_code == 401

    def test_export_with_table_filter(self, client, auth_headers):
        resp = client.post(
            "/api/v1/feishu/export",
            json={"tables": ["daily_metrics"]},
            headers=auth_headers,
        )
        assert resp.status_code == 200

    def test_export_apply_true(self, client, auth_headers):
        resp = client.post(
            "/api/v1/feishu/export",
            json={"apply": True},
            headers=auth_headers,
        )
        assert resp.status_code == 200


class TestFeishuSync:
    """POST /api/v1/feishu/sync"""

    def test_sync_defaults_dry_run(self, client, auth_headers):
        resp = client.post("/api/v1/feishu/sync", json={}, headers=auth_headers)
        assert resp.status_code == 200
        assert "status" in resp.json()

    def test_sync_unauthorized_returns_401(self, client):
        resp = client.post("/api/v1/feishu/sync", json={})
        assert resp.status_code == 401


class TestFeishuStatusImport:
    """POST /api/v1/feishu/status/import"""

    def test_import_defaults_dry_run(self, client, auth_headers):
        resp = client.post("/api/v1/feishu/status/import", json={}, headers=auth_headers)
        assert resp.status_code == 200
        assert "status" in resp.json()

    def test_import_unauthorized_returns_401(self, client):
        resp = client.post("/api/v1/feishu/status/import", json={})
        assert resp.status_code == 401
