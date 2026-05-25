"""TDD RED: Pipeline API tests — will fail until router is mounted."""
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
    return {"Authorization": "Bearer test-token-for-baxi-ci-tests-only-32ch"}


class TestPipelineRun:
    """POST /api/v1/pipeline/run"""

    def test_run_valid_type_daily(self, client, auth_headers):
        resp = client.post(
            "/api/v1/pipeline/run",
            json={"pipeline_type": "daily"},
            headers=auth_headers,
        )
        assert resp.status_code == 200
        data = resp.json()
        assert "command" in data
        assert "python3" in data["command"]

    def test_run_valid_type_full(self, client, auth_headers):
        resp = client.post(
            "/api/v1/pipeline/run",
            json={"pipeline_type": "full"},
            headers=auth_headers,
        )
        assert resp.status_code == 200
        assert "command" in resp.json()

    def test_run_valid_type_db_full(self, client, auth_headers):
        resp = client.post(
            "/api/v1/pipeline/run",
            json={"pipeline_type": "db_full"},
            headers=auth_headers,
        )
        assert resp.status_code == 200
        assert "command" in resp.json()

    def test_run_invalid_type(self, client, auth_headers):
        resp = client.post(
            "/api/v1/pipeline/run",
            json={"pipeline_type": "invalid"},
            headers=auth_headers,
        )
        # Security fix: pipeline_type now uses Literal type for strict validation
        # Invalid types are rejected at the API schema level (422)
        assert resp.status_code == 422
        assert "VALIDATION_ERROR" in str(resp.json())

    def test_run_unauthorized_returns_401(self, client):
        resp = client.post("/api/v1/pipeline/run", json={})
        assert resp.status_code == 401

    def test_run_default_pipeline_type(self, client, auth_headers):
        resp = client.post("/api/v1/pipeline/run", json={}, headers=auth_headers)
        assert resp.status_code == 200
        assert resp.json()["pipeline_type"] == "daily"


class TestServiceEdgeCases:
    """Direct service module tests for coverage."""

    def test_log_reader_missing_file(self):
        from services.log_reader import read_log_errors, read_log_recent, read_audit_logs
        assert read_log_errors("/tmp/nx_123.log") == []
        assert read_log_recent("/tmp/nx_123.log") == []
        assert read_audit_logs("/tmp/nx_123.csv") == []

    def test_log_reader_empty_file(self, tmp_path):
        from services.log_reader import _tail_jsonl
        f = tmp_path / "empty.jsonl"
        f.write_text("")
        assert _tail_jsonl(str(f), 10) == []

    def test_log_reader_malformed_then_valid(self, tmp_path):
        import json
        from services.log_reader import _tail_jsonl
        f = tmp_path / "mixed.jsonl"
        f.write_text("{x}\n" + json.dumps({"a": 1}) + "\n{y}\n" + json.dumps({"a": 2}) + "\n")
        r = _tail_jsonl(str(f), 5)
        assert len(r) == 2

    def test_log_reader_limit_and_order(self, tmp_path):
        import json
        from services.log_reader import _tail_jsonl
        f = tmp_path / "order.jsonl"
        for i in range(5):
            with open(str(f), "a") as fh:
                fh.write(json.dumps({"n": i}) + "\n")
        r = _tail_jsonl(str(f), 2)
        assert len(r) == 2
        assert r[0]["n"] == 4

    def test_log_reader_error_filter(self, tmp_path):
        import json
        from services.log_reader import read_log_errors
        f = tmp_path / "e.log"
        for i in range(4):
            with open(str(f), "a") as fh:
                fh.write(json.dumps({"ts": f"T{i}", "request_id": f"r{i%2}", "error_code": "E"}) + "\n")
        r = read_log_errors(str(f), request_id="r0")
        assert len(r) == 2
        r = read_log_errors(str(f), limit=1)
        assert len(r) == 1

    def test_log_reader_audit_sort(self, tmp_path):
        import csv
        from services.log_reader import read_audit_logs
        f = tmp_path / "a.csv"
        with open(str(f), "w", newline="") as fh:
            w = csv.DictWriter(fh, fieldnames=["timestamp", "outbox_id", "status"])
            w.writeheader()
            w.writerow({"timestamp": "T3", "outbox_id": "3", "status": "done"})
            w.writerow({"timestamp": "T1", "outbox_id": "1", "status": "fail"})
        r = read_audit_logs(str(f), limit=10)
        assert len(r) == 2
        assert r[0]["outbox_id"] == "3"

    def test_log_reader_audit_filter(self, tmp_path):
        import csv
        from services.log_reader import read_audit_logs
        f = tmp_path / "b.csv"
        with open(str(f), "w", newline="") as fh:
            w = csv.DictWriter(fh, fieldnames=["timestamp", "outbox_id", "status"])
            w.writeheader()
            w.writerow({"timestamp": "T1", "outbox_id": "1", "status": "fail"})
            w.writerow({"timestamp": "T2", "outbox_id": "2", "status": "done"})
        r = read_audit_logs(str(f), status="fail")
        assert len(r) == 1
        r = read_audit_logs(str(f), outbox_id="2")
        assert len(r) == 1

    def test_pipeline_service_all_types(self):
        from services.pipeline_service import preview_pipeline_run, get_available_pipelines
        for t in ["daily", "full", "db_full"]:
            assert "python3" in preview_pipeline_run(t)["command"]
        assert preview_pipeline_run("invalid")["warnings"]
        assert len(get_available_pipelines()) == 3

    def test_pipeline_service_env_warnings(self, monkeypatch):
        from services.pipeline_service import preview_pipeline_run
        monkeypatch.delenv("FEISHU_APP_ID", raising=False)
        monkeypatch.delenv("LLM_API_KEY", raising=False)
        assert len(preview_pipeline_run("daily")["warnings"]) >= 1

    def test_feishu_service_basic(self):
        import sys
        sys.path.insert(0, "scripts")
        from services.feishu_service import FeishuService
        svc = FeishuService(dry_run=True)
        assert "app_id" in svc._load_config()
        for op in [svc.export_tables, svc.sync_to_feishu, svc.import_status_from_feishu]:
            r = op()
            assert r["status"] in ("preview", "not_configured")
        r = svc.sync_to_feishu()
        assert r["status"] in ("preview", "not_configured")
        r = svc.import_status_from_feishu()
        assert r["status"] in ("preview", "not_configured")
