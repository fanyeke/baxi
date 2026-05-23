"""Governance API integration tests — validates all 7 GET endpoints + auth."""
import os

from fastapi.testclient import TestClient

os.environ["API_BEARER_TOKEN"] = "test-token-12345"

from api.main import app

client = TestClient(app)
AUTH = {"Authorization": "Bearer test-token-12345"}


def test_get_catalog():
    r = client.get("/api/v1/governance/catalog", headers=AUTH)
    assert r.status_code == 200
    assert "assets" in r.json()


def test_get_classification():
    r = client.get("/api/v1/governance/classification", headers=AUTH)
    assert r.status_code == 200


def test_get_markings():
    r = client.get("/api/v1/governance/markings", headers=AUTH)
    assert r.status_code == 200


def test_get_lineage():
    r = client.get("/api/v1/governance/lineage", headers=AUTH)
    assert r.status_code == 200


def test_get_checkpoints():
    r = client.get("/api/v1/governance/checkpoints", headers=AUTH)
    assert r.status_code == 200


def test_get_health():
    r = client.get("/api/v1/governance/health", headers=AUTH)
    assert r.status_code == 200


def test_get_status():
    r = client.get("/api/v1/governance/status", headers=AUTH)
    assert r.status_code == 200


def test_unauthorized():
    r = client.get("/api/v1/governance/catalog")
    assert r.status_code == 401
