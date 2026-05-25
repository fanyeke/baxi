"""Extended tests for services/alert_service.py."""

import sqlite3

import pytest

from services import alert_service


@pytest.fixture
def mem_db():
    """Create an in-memory SQLite DB with the alert_events table."""
    conn = sqlite3.connect(":memory:")
    conn.row_factory = sqlite3.Row
    conn.execute(
        """
        CREATE TABLE alert_events (
            event_id TEXT PRIMARY KEY,
            rule_id TEXT NOT NULL,
            event_date TEXT NOT NULL,
            severity TEXT NOT NULL,
            metric_name TEXT NOT NULL,
            object_type TEXT DEFAULT 'global',
            object_id TEXT DEFAULT 'global',
            current_value REAL,
            baseline_value REAL,
            change_rate REAL,
            sample_size INTEGER,
            evidence_json TEXT,
            description TEXT,
            status TEXT DEFAULT 'new',
            owner_role TEXT,
            created_at TEXT NOT NULL,
            impact_score REAL
        )
        """
    )
    conn.commit()
    yield conn
    conn.close()


def _seed_alerts(conn):
    """Insert sample alert rows."""
    rows = [
        (
            "evt-1", "rule-1", "2024-01-01", "high", "gmv",
            "seller", "seller-a", 100.0, 90.0, 0.1, 50,
            None, None, "new", "ops", "2024-01-01T00:00:00", 0.8,
        ),
        (
            "evt-2", "rule-2", "2024-01-02", "medium", "orders",
            "category", "cat-b", 200.0, 180.0, 0.05, 100,
            None, None, "acknowledged", "ops", "2024-01-02T00:00:00", 0.5,
        ),
        (
            "evt-3", "rule-3", "2024-01-03", "low", "reviews",
            "global", "global", 4.2, 4.5, -0.05, 200,
            None, None, "resolved", "qa", "2024-01-03T00:00:00", 0.2,
        ),
    ]
    conn.executemany(
        """
        INSERT INTO alert_events
        (event_id, rule_id, event_date, severity, metric_name,
         object_type, object_id, current_value, baseline_value,
         change_rate, sample_size, evidence_json, description,
         status, owner_role, created_at, impact_score)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        """,
        rows,
    )
    conn.commit()


class TestBuildAlertConditions:
    def test_no_filters(self):
        where, params = alert_service._build_alert_conditions(
            None, None, None, None
        )
        assert where == ""
        assert params == []

    def test_single_status(self):
        where, params = alert_service._build_alert_conditions(
            "new", None, None, None
        )
        assert where == "WHERE status = ?"
        assert params == ["new"]

    def test_single_severity(self):
        where, params = alert_service._build_alert_conditions(
            None, "high", None, None
        )
        assert where == "WHERE severity = ?"
        assert params == ["high"]

    def test_single_object_type(self):
        where, params = alert_service._build_alert_conditions(
            None, None, "seller", None
        )
        assert where == "WHERE object_type = ?"
        assert params == ["seller"]

    def test_status_and_severity(self):
        where, params = alert_service._build_alert_conditions(
            "new", "high", None, None
        )
        assert where == "WHERE status = ? AND severity = ?"
        assert params == ["new", "high"]

    def test_status_severity_object_type(self):
        where, params = alert_service._build_alert_conditions(
            "acknowledged", "medium", "category", None
        )
        assert where == "WHERE status = ? AND severity = ? AND object_type = ?"
        assert params == ["acknowledged", "medium", "category"]

    def test_all_four(self):
        where, params = alert_service._build_alert_conditions(
            "resolved", "low", "global", "global"
        )
        assert (
            where
            == "WHERE status = ? AND severity = ? AND object_type = ? AND object_id = ?"
        )
        assert params == ["resolved", "low", "global", "global"]


class TestGetAlerts:
    def test_empty_table(self, mem_db):
        result = alert_service.get_alerts(conn=mem_db)
        assert result == []

    def test_no_filters_returns_all(self, mem_db):
        _seed_alerts(mem_db)
        result = alert_service.get_alerts(conn=mem_db)
        assert len(result) == 3
        assert result[0]["event_id"] == "evt-3"

    def test_filter_by_status(self, mem_db):
        _seed_alerts(mem_db)
        result = alert_service.get_alerts(conn=mem_db, status="new")
        assert len(result) == 1
        assert result[0]["event_id"] == "evt-1"

    def test_filter_by_severity(self, mem_db):
        _seed_alerts(mem_db)
        result = alert_service.get_alerts(conn=mem_db, severity="medium")
        assert len(result) == 1
        assert result[0]["event_id"] == "evt-2"

    def test_filter_by_object_type(self, mem_db):
        _seed_alerts(mem_db)
        result = alert_service.get_alerts(conn=mem_db, object_type="seller")
        assert len(result) == 1
        assert result[0]["event_id"] == "evt-1"

    def test_filter_by_combined_conditions(self, mem_db):
        _seed_alerts(mem_db)
        result = alert_service.get_alerts(
            conn=mem_db, status="acknowledged", severity="medium", object_type="category"
        )
        assert len(result) == 1
        assert result[0]["event_id"] == "evt-2"

    def test_filter_no_match(self, mem_db):
        _seed_alerts(mem_db)
        result = alert_service.get_alerts(conn=mem_db, status="nonexistent")
        assert result == []

    def test_limit(self, mem_db):
        _seed_alerts(mem_db)
        result = alert_service.get_alerts(conn=mem_db, limit=2)
        assert len(result) == 2


class TestGetAlertsWithCount:
    def test_empty_table(self, mem_db):
        items, total = alert_service.get_alerts_with_count(conn=mem_db)
        assert items == []
        assert total == 0

    def test_normal_return(self, mem_db):
        _seed_alerts(mem_db)
        items, total = alert_service.get_alerts_with_count(conn=mem_db)
        assert len(items) == 3
        assert total == 3

    def test_with_filters(self, mem_db):
        _seed_alerts(mem_db)
        items, total = alert_service.get_alerts_with_count(
            conn=mem_db, status="new"
        )
        assert len(items) == 1
        assert total == 1

    def test_limit_less_than_total(self, mem_db):
        _seed_alerts(mem_db)
        items, total = alert_service.get_alerts_with_count(
            conn=mem_db, limit=2
        )
        assert len(items) == 2
        assert total == 3


class TestGetAlertById:
    def test_existing_id(self, mem_db):
        _seed_alerts(mem_db)
        result = alert_service.get_alert_by_id(conn=mem_db, event_id="evt-2")
        assert result is not None
        assert result["event_id"] == "evt-2"
        assert result["severity"] == "medium"

    def test_non_existing_id(self, mem_db):
        _seed_alerts(mem_db)
        result = alert_service.get_alert_by_id(
            conn=mem_db, event_id="evt-999"
        )
        assert result is None

    def test_none_id(self, mem_db):
        _seed_alerts(mem_db)
        result = alert_service.get_alert_by_id(conn=mem_db, event_id=None)
        assert result is None


class TestConnectionManagement:
    def test_connection_closed_when_no_conn_passed(self, monkeypatch):
        calls = []

        class FakeConn:
            def __init__(self):
                self.closed = False

            def execute(self, sql, params=()):
                class FakeCur:
                    def fetchall(self):
                        return []

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
            "services.alert_service.get_db", fake_get_db
        )
        result = alert_service.get_alerts()
        assert result == []
        assert "get_db" in calls
        assert "close" in calls
        assert fake_conn.closed

    def test_connection_not_closed_when_conn_passed(self, mem_db):
        result = alert_service.get_alerts(conn=mem_db)
        assert result == []
        cur = mem_db.execute("SELECT 1")
        assert cur.fetchone()[0] == 1
