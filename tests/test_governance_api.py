"""Governance API integration tests — validates all 7 GET endpoints + auth."""
import pytest
from fastapi.testclient import TestClient

from api.main import create_app


@pytest.fixture(scope="module")
def client():
    return TestClient(create_app())


HEADERS = {"Authorization": "Bearer test-token-for-baxi-ci-tests-only-32ch"}


def test_get_catalog(client):
    r = client.get("/api/v1/governance/catalog", headers=HEADERS)
    assert r.status_code == 200
    assert "assets" in r.json()


def test_get_classification(client):
    r = client.get("/api/v1/governance/classification", headers=HEADERS)
    assert r.status_code == 200


def test_get_markings(client):
    r = client.get("/api/v1/governance/markings", headers=HEADERS)
    assert r.status_code == 200


def test_get_lineage(client):
    r = client.get("/api/v1/governance/lineage", headers=HEADERS)
    assert r.status_code == 200


def test_get_checkpoints(client):
    r = client.get("/api/v1/governance/checkpoints", headers=HEADERS)
    assert r.status_code == 200


def test_get_health(client):
    r = client.get("/api/v1/governance/health", headers=HEADERS)
    assert r.status_code == 200


def test_get_status(client):
    r = client.get("/api/v1/governance/status", headers=HEADERS)
    assert r.status_code == 200


def test_unauthorized(client):
    r = client.get("/api/v1/governance/catalog")
    assert r.status_code == 401
