"""Tests for services/status_service.py."""

import sqlite3

import pytest

from services.status_service import get_last_pipeline_run, get_last_ingestion_batch


@pytest.fixture
def mem_db():
    """Create an in-memory SQLite DB with required tables."""
    conn = sqlite3.connect(":memory:")
    conn.row_factory = sqlite3.Row
    conn.execute(
        """
        CREATE TABLE pipeline_runs (
            run_id TEXT PRIMARY KEY,
            run_type TEXT NOT NULL,
            mode TEXT NOT NULL,
            status TEXT NOT NULL,
            started_at TEXT NOT NULL,
            finished_at TEXT,
            input_count INTEGER DEFAULT 0,
            output_count INTEGER DEFAULT 0,
            error_message TEXT
        )
        """
    )
    conn.execute(
        """
        CREATE TABLE ingestion_batches (
            batch_id TEXT PRIMARY KEY,
            source_name TEXT NOT NULL,
            ingestion_mode TEXT NOT NULL,
            date_start TEXT,
            date_end TEXT,
            source_file TEXT,
            row_count INTEGER DEFAULT 0,
            status TEXT NOT NULL,
            created_at TEXT NOT NULL
        )
        """
    )
    conn.commit()
    yield conn
    conn.close()


class TestGetLastPipelineRun:
    def test_empty_table(self, mem_db):
        """Returns None when pipeline_runs is empty."""
        result = get_last_pipeline_run(conn=mem_db)
        assert result is None

    def test_single_record(self, mem_db):
        """Returns the only record as a dict."""
        mem_db.execute(
            """
            INSERT INTO pipeline_runs
            (run_id, run_type, mode, status, started_at, finished_at,
             input_count, output_count, error_message)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
            """,
            ("run-1", "ingest", "full", "success", "2024-01-01T00:00:00",
             "2024-01-01T01:00:00", 100, 95, None),
        )
        mem_db.commit()
        result = get_last_pipeline_run(conn=mem_db)
        assert result is not None
        assert result["run_id"] == "run-1"
        assert result["status"] == "success"
        assert result["input_count"] == 100
        assert result["output_count"] == 95

    def test_returns_latest_by_started_at(self, mem_db):
        """When multiple records exist, returns the one with latest started_at."""
        mem_db.execute(
            """
            INSERT INTO pipeline_runs
            (run_id, run_type, mode, status, started_at, finished_at,
             input_count, output_count, error_message)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
            """,
            ("run-1", "ingest", "full", "success", "2024-01-01T00:00:00",
             "2024-01-01T01:00:00", 100, 95, None),
        )
        mem_db.execute(
            """
            INSERT INTO pipeline_runs
            (run_id, run_type, mode, status, started_at, finished_at,
             input_count, output_count, error_message)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
            """,
            ("run-2", "metrics", "delta", "failed", "2024-01-02T00:00:00",
             None, 50, 0, "disk full"),
        )
        mem_db.commit()
        result = get_last_pipeline_run(conn=mem_db)
        assert result is not None
        assert result["run_id"] == "run-2"
        assert result["status"] == "failed"
        assert result["error_message"] == "disk full"

    def test_connection_closed_when_no_conn_passed(self, monkeypatch):
        """When conn is not provided, the function opens and closes its own connection."""
        calls = []

        class FakeConn:
            def __init__(self):
                self.closed = False

            def execute(self, sql):
                class FakeCur:
                    def fetchone(self):
                        return None
                return FakeCur()

            def close(self):
                self.closed = True
                calls.append("close")

        fake_conn = FakeConn()

        def fake_get_db():
            calls.append("get_db")
            return fake_conn

        monkeypatch.setattr(
            "services.status_service.get_db", fake_get_db
        )
        result = get_last_pipeline_run()
        assert result is None
        assert "get_db" in calls
        assert "close" in calls
        assert fake_conn.closed

    def test_connection_not_closed_when_conn_passed(self, mem_db):
        """When conn is provided, it should not be closed by the function."""
        result = get_last_pipeline_run(conn=mem_db)
        assert result is None
        # Connection should still be usable
        cur = mem_db.execute("SELECT 1")
        assert cur.fetchone()[0] == 1


class TestGetLastIngestionBatch:
    def test_empty_table(self, mem_db):
        """Returns None when ingestion_batches is empty."""
        result = get_last_ingestion_batch(conn=mem_db)
        assert result is None

    def test_single_record(self, mem_db):
        """Returns the only record as a dict."""
        mem_db.execute(
            """
            INSERT INTO ingestion_batches
            (batch_id, source_name, ingestion_mode, date_start, date_end,
             source_file, row_count, status, created_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
            """,
            ("batch-1", "orders", "full", "2024-01-01", "2024-01-31",
             "orders.csv", 1000, "completed", "2024-01-01T00:00:00"),
        )
        mem_db.commit()
        result = get_last_ingestion_batch(conn=mem_db)
        assert result is not None
        assert result["batch_id"] == "batch-1"
        assert result["source_name"] == "orders"
        assert result["row_count"] == 1000
        assert result["status"] == "completed"

    def test_returns_latest_by_created_at(self, mem_db):
        """When multiple records exist, returns the one with latest created_at."""
        mem_db.execute(
            """
            INSERT INTO ingestion_batches
            (batch_id, source_name, ingestion_mode, date_start, date_end,
             source_file, row_count, status, created_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
            """,
            ("batch-1", "orders", "full", "2024-01-01", "2024-01-31",
             "orders.csv", 1000, "completed", "2024-01-01T00:00:00"),
        )
        mem_db.execute(
            """
            INSERT INTO ingestion_batches
            (batch_id, source_name, ingestion_mode, date_start, date_end,
             source_file, row_count, status, created_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
            """,
            ("batch-2", "sellers", "delta", "2024-02-01", "2024-02-28",
             "sellers.csv", 500, "completed", "2024-02-01T00:00:00"),
        )
        mem_db.commit()
        result = get_last_ingestion_batch(conn=mem_db)
        assert result is not None
        assert result["batch_id"] == "batch-2"
        assert result["source_name"] == "sellers"
        assert result["row_count"] == 500

    def test_connection_closed_when_no_conn_passed(self, monkeypatch):
        """When conn is not provided, the function opens and closes its own connection."""
        calls = []

        class FakeConn:
            def __init__(self):
                self.closed = False

            def execute(self, sql):
                class FakeCur:
                    def fetchone(self):
                        return None
                return FakeCur()

            def close(self):
                self.closed = True
                calls.append("close")

        fake_conn = FakeConn()

        def fake_get_db():
            calls.append("get_db")
            return fake_conn

        monkeypatch.setattr(
            "services.status_service.get_db", fake_get_db
        )
        result = get_last_ingestion_batch()
        assert result is None
        assert "get_db" in calls
        assert "close" in calls
        assert fake_conn.closed

    def test_connection_not_closed_when_conn_passed(self, mem_db):
        """When conn is provided, it should not be closed by the function."""
        result = get_last_ingestion_batch(conn=mem_db)
        assert result is None
        # Connection should still be usable
        cur = mem_db.execute("SELECT 1")
        assert cur.fetchone()[0] == 1
