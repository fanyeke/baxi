"""Qoder capabilities API integration tests — validates GET /api/v1/qoder/capabilities."""
import pytest
from fastapi.testclient import TestClient

from api.main import create_app


@pytest.fixture(scope="module")
def client():
    return TestClient(create_app())


HEADERS = {"Authorization": "Bearer test-token-for-baxi-ci-tests-only-32ch"}


class TestCapabilities:
    def test_capabilities_requires_auth(self, client):
        r = client.get("/api/v1/qoder/capabilities")
        assert r.status_code == 401
        assert "AUTH_REQUIRED" in str(r.json())

    def test_capabilities_returns_readonly_mode(self, client):
        r = client.get("/api/v1/qoder/capabilities", headers=HEADERS)
        assert r.status_code == 200
        data = r.json()
        assert data["mode"] == "read_only"

    def test_capabilities_lists_allowed_endpoints(self, client):
        r = client.get("/api/v1/qoder/capabilities", headers=HEADERS)
        assert r.status_code == 200
        data = r.json()
        assert "allowed_endpoints" in data
        assert isinstance(data["allowed_endpoints"], list)
        assert len(data["allowed_endpoints"]) > 0

    def test_capabilities_lists_forbidden_actions(self, client):
        r = client.get("/api/v1/qoder/capabilities", headers=HEADERS)
        assert r.status_code == 200
        data = r.json()
        assert "forbidden_actions" in data
        assert isinstance(data["forbidden_actions"], list)
