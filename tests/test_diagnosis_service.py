"""Tests for services/diagnosis_service.py."""

import csv
import json
import os

import pytest

from services import diagnosis_service


def _write_jsonl(filepath, entries):
    with open(filepath, "w") as f:
        for entry in entries:
            f.write(json.dumps(entry) + "\n")


def _write_csv(filepath, fieldnames, rows):
    with open(filepath, "w", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=fieldnames)
        writer.writeheader()
        writer.writerows(rows)


@pytest.fixture
def temp_logs_dir(tmp_path, monkeypatch):
    """Create a temporary directory structure and patch diagnosis_service paths."""
    logs_dir = tmp_path / "logs" / "api"
    system_dir = tmp_path / "data" / "system"
    logs_dir.mkdir(parents=True)
    system_dir.mkdir(parents=True)

    monkeypatch.setattr(diagnosis_service, "ERROR_LOG", str(logs_dir / "error.log"))
    monkeypatch.setattr(
        diagnosis_service, "AUDIT_CSV", str(system_dir / "api_audit_dispatch.csv")
    )
    monkeypatch.setattr(
        diagnosis_service, "AUDIT_FEISHU_CSV", str(system_dir / "api_audit_feishu.csv")
    )

    return tmp_path


class TestSearchJsonl:
    def test_search_jsonl_normal(self, tmp_path):
        filepath = tmp_path / "test.jsonl"
        entries = [
            {"request_id": "req-001", "message": "first"},
            {"request_id": "req-002", "message": "second"},
            {"request_id": "req-001", "message": "third"},
        ]
        _write_jsonl(filepath, entries)

        result = diagnosis_service._search_jsonl(str(filepath), "req-001")
        assert len(result) == 2
        assert result[0]["message"] == "first"
        assert result[1]["message"] == "third"

    def test_search_jsonl_file_not_found(self, tmp_path):
        result = diagnosis_service._search_jsonl(
            str(tmp_path / "nonexistent.jsonl"), "req-001"
        )
        assert result == []

    def test_search_jsonl_json_parse_error(self, tmp_path):
        filepath = tmp_path / "test.jsonl"
        with open(filepath, "w") as f:
            f.write('{"request_id": "req-001", "message": "valid"}\n')
            f.write("this is not json\n")
            f.write('{"request_id": "req-002", "message": "valid2"}\n')
            f.write("{broken json\n")

        result = diagnosis_service._search_jsonl(str(filepath), "req-001")
        assert len(result) == 1
        assert result[0]["message"] == "valid"

    def test_search_jsonl_no_matches(self, tmp_path):
        filepath = tmp_path / "test.jsonl"
        _write_jsonl(
            filepath,
            [
                {"request_id": "req-002", "message": "other"},
                {"request_id": "req-003", "message": "another"},
            ],
        )

        result = diagnosis_service._search_jsonl(str(filepath), "req-001")
        assert result == []

    def test_search_jsonl_limit(self, tmp_path):
        filepath = tmp_path / "test.jsonl"
        entries = [{"request_id": "req-001", "idx": i} for i in range(150)]
        _write_jsonl(filepath, entries)

        result = diagnosis_service._search_jsonl(str(filepath), "req-001", limit=50)
        assert len(result) == 50

    def test_search_jsonl_empty_lines(self, tmp_path):
        filepath = tmp_path / "test.jsonl"
        with open(filepath, "w") as f:
            f.write("\n")
            f.write('{"request_id": "req-001", "message": "after blank"}\n')
            f.write("\n")

        result = diagnosis_service._search_jsonl(str(filepath), "req-001")
        assert len(result) == 1
        assert result[0]["message"] == "after blank"

    def test_search_jsonl_oserror(self, tmp_path, monkeypatch):
        """Test OSError handling by monkeypatching open."""
        filepath = str(tmp_path / "test.jsonl")
        with open(filepath, "w") as f:
            f.write('{"request_id": "req-001"}\n')

        def broken_open(*args, **kwargs):
            raise OSError("permission denied")

        monkeypatch.setattr("builtins.open", broken_open)
        result = diagnosis_service._search_jsonl(filepath, "req-001")
        assert result == []

    def test_search_jsonl_alt_timestamp_key(self, tmp_path):
        """Error entries may use 'ts' instead of 'timestamp'."""
        filepath = tmp_path / "test.jsonl"
        entries = [
            {"request_id": "req-001", "ts": "2024-01-01T00:00:00", "message": "with ts"},
        ]
        _write_jsonl(filepath, entries)

        result = diagnosis_service._search_jsonl(str(filepath), "req-001")
        assert len(result) == 1


class TestSearchCsv:
    def test_search_csv_normal(self, tmp_path):
        filepath = tmp_path / "test.csv"
        _write_csv(
            filepath,
            ["request_id", "status", "outbox_id"],
            [
                {"request_id": "req-001", "status": "success", "outbox_id": "ob-1"},
                {"request_id": "req-002", "status": "failed", "outbox_id": "ob-2"},
                {"request_id": "req-001", "status": "pending", "outbox_id": "ob-3"},
            ],
        )

        result = diagnosis_service._search_csv(str(filepath), "req-001")
        assert len(result) == 2
        assert result[0]["status"] == "success"
        assert result[1]["outbox_id"] == "ob-3"

    def test_search_csv_file_not_found(self, tmp_path):
        result = diagnosis_service._search_csv(
            str(tmp_path / "nonexistent.csv"), "req-001"
        )
        assert result == []

    def test_search_csv_no_matches(self, tmp_path):
        filepath = tmp_path / "test.csv"
        _write_csv(
            filepath,
            ["request_id", "status"],
            [
                {"request_id": "req-002", "status": "success"},
            ],
        )

        result = diagnosis_service._search_csv(str(filepath), "req-001")
        assert result == []

    def test_search_csv_limit(self, tmp_path):
        filepath = tmp_path / "test.csv"
        rows = [{"request_id": "req-001", "idx": str(i)} for i in range(150)]
        _write_csv(filepath, ["request_id", "idx"], rows)

        result = diagnosis_service._search_csv(str(filepath), "req-001", limit=50)
        assert len(result) == 50

    def test_search_csv_oserror(self, tmp_path, monkeypatch):
        filepath = str(tmp_path / "test.csv")
        with open(filepath, "w") as f:
            f.write("request_id,status\n")
            f.write("req-001,success\n")

        def broken_open(*args, **kwargs):
            raise OSError("permission denied")

        monkeypatch.setattr("builtins.open", broken_open)
        result = diagnosis_service._search_csv(filepath, "req-001")
        assert result == []


class TestDiagnoseByRequestId:
    def test_diagnose_normal(self, temp_logs_dir):
        error_entries = [
            {
                "request_id": "req-001",
                "timestamp": "2024-01-01T00:00:00",
                "error_code": "E001",
                "message": "Something failed",
                "diagnosis": "Root cause found",
                "suggested_action": "Fix it",
            },
        ]
        _write_jsonl(diagnosis_service.ERROR_LOG, error_entries)

        audit_rows = [
            {
                "request_id": "req-001",
                "timestamp": "2024-01-01T00:01:00",
                "outbox_id": "ob-1",
                "status": "dispatched",
                "error": "",
            },
        ]
        _write_csv(
            diagnosis_service.AUDIT_CSV,
            ["request_id", "timestamp", "outbox_id", "status", "error"],
            audit_rows,
        )

        feishu_rows = [
            {
                "request_id": "req-001",
                "timestamp": "2024-01-01T00:02:00",
                "action": "notify",
                "status": "sent",
            },
        ]
        _write_csv(
            diagnosis_service.AUDIT_FEISHU_CSV,
            ["request_id", "timestamp", "action", "status"],
            feishu_rows,
        )

        result = diagnosis_service.diagnose_by_request_id("req-001")
        assert result is not None
        assert result["request_id"] == "req-001"
        assert result["summary"] == "Something failed"
        assert result["error_code"] == "E001"
        assert result["diagnosis"] == "Root cause found"
        assert result["suggested_action"] == "Fix it"
        assert len(result["related_logs"]) == 3

        sources = [log["source"] for log in result["related_logs"]]
        assert "error.log" in sources
        assert "audit_dispatch.csv" in sources
        assert "audit_feishu.csv" in sources

    def test_diagnose_no_error_entry(self, temp_logs_dir):
        """When there's no error entry, summary should be default."""
        audit_rows = [
            {
                "request_id": "req-001",
                "timestamp": "2024-01-01T00:01:00",
                "outbox_id": "ob-1",
                "status": "dispatched",
                "error": "",
            },
        ]
        _write_csv(
            diagnosis_service.AUDIT_CSV,
            ["request_id", "timestamp", "outbox_id", "status", "error"],
            audit_rows,
        )

        result = diagnosis_service.diagnose_by_request_id("req-001")
        assert result is not None
        assert result["summary"] == "No error message recorded"
        assert result["error_code"] == ""
        assert result["diagnosis"] == ""
        assert result["suggested_action"] == ""
        assert len(result["related_logs"]) == 1

    def test_diagnose_request_id_not_found(self, temp_logs_dir):
        """When files exist but request_id has no matches."""
        error_entries = [
            {
                "request_id": "req-002",
                "timestamp": "2024-01-01T00:00:00",
                "error_code": "E001",
                "message": "Other error",
            },
        ]
        _write_jsonl(diagnosis_service.ERROR_LOG, error_entries)

        audit_rows = [
            {
                "request_id": "req-002",
                "timestamp": "2024-01-01T00:01:00",
                "outbox_id": "ob-1",
                "status": "dispatched",
                "error": "",
            },
        ]
        _write_csv(
            diagnosis_service.AUDIT_CSV,
            ["request_id", "timestamp", "outbox_id", "status", "error"],
            audit_rows,
        )

        feishu_rows = [
            {
                "request_id": "req-002",
                "timestamp": "2024-01-01T00:02:00",
                "action": "notify",
                "status": "sent",
            },
        ]
        _write_csv(
            diagnosis_service.AUDIT_FEISHU_CSV,
            ["request_id", "timestamp", "action", "status"],
            feishu_rows,
        )

        result = diagnosis_service.diagnose_by_request_id("req-001")
        assert result is None

    def test_diagnose_log_file_not_found(self, temp_logs_dir):
        """When no log files exist at all."""
        result = diagnosis_service.diagnose_by_request_id("req-001")
        assert result is None
