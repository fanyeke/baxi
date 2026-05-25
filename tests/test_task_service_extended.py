"""Extended tests for services/task_service.py."""

import sqlite3

import pytest

from services import task_service


@pytest.fixture
def mem_db():
    """Create an in-memory SQLite DB with the action_tasks table."""
    conn = sqlite3.connect(":memory:")
    conn.row_factory = sqlite3.Row
    conn.execute(
        """
        CREATE TABLE action_tasks (
            task_id TEXT PRIMARY KEY,
            recommendation_id TEXT,
            event_id TEXT,
            task_title TEXT NOT NULL,
            task_description TEXT,
            task_source TEXT DEFAULT 'heuristic_strategy',
            owner_role TEXT,
            owner_user_id TEXT,
            priority TEXT DEFAULT 'medium',
            due_at TEXT,
            status TEXT DEFAULT 'todo',
            feedback TEXT,
            completed_at TEXT,
            created_at TEXT NOT NULL,
            target_object_type TEXT,
            target_object_id TEXT
        )
        """
    )
    conn.commit()
    yield conn
    conn.close()


def _seed_tasks(conn):
    """Insert sample task rows."""
    rows = [
        (
            "task-1", "rec-1", "evt-1", "Fix bug", "Investigate",
            "heuristic", "dev", "user-a", "high", "2024-06-01",
            "todo", None, None, "2024-01-01T00:00:00", None, None,
        ),
        (
            "task-2", "rec-2", None, "Update docs", "Add section",
            "heuristic", "dev", "user-b", "medium", None,
            "in_progress", None, None, "2024-01-02T00:00:00", None, None,
        ),
        (
            "task-3", "rec-3", "evt-3", "Review PR", None,
            "heuristic", "qa", "user-c", "low", None,
            "done", "LGTM", None, "2024-01-03T00:00:00", None, None,
        ),
    ]
    conn.executemany(
        """
        INSERT INTO action_tasks
        (task_id, recommendation_id, event_id, task_title, task_description,
         task_source, owner_role, owner_user_id, priority, due_at, status,
         feedback, completed_at, created_at, target_object_type, target_object_id)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        """,
        rows,
    )
    conn.commit()


class TestBuildTaskConditions:
    def test_no_filters(self):
        where, params = task_service._build_task_conditions(
            None, None, None
        )
        assert where == ""
        assert params == []

    def test_single_status(self):
        where, params = task_service._build_task_conditions(
            "todo", None, None
        )
        assert where == "WHERE status = ?"
        assert params == ["todo"]

    def test_single_priority(self):
        where, params = task_service._build_task_conditions(
            None, "high", None
        )
        assert where == "WHERE priority = ?"
        assert params == ["high"]

    def test_single_owner_role(self):
        where, params = task_service._build_task_conditions(
            None, None, "dev"
        )
        assert where == "WHERE owner_role = ?"
        assert params == ["dev"]

    def test_status_and_priority(self):
        where, params = task_service._build_task_conditions(
            "todo", "high", None
        )
        assert where == "WHERE status = ? AND priority = ?"
        assert params == ["todo", "high"]

    def test_status_priority_owner_role(self):
        where, params = task_service._build_task_conditions(
            "in_progress", "medium", "dev"
        )
        assert where == "WHERE status = ? AND priority = ? AND owner_role = ?"
        assert params == ["in_progress", "medium", "dev"]

    def test_all_three(self):
        where, params = task_service._build_task_conditions(
            "done", "low", "qa"
        )
        assert where == "WHERE status = ? AND priority = ? AND owner_role = ?"
        assert params == ["done", "low", "qa"]


class TestGetTasks:
    def test_empty_table(self, mem_db):
        result = task_service.get_tasks(conn=mem_db)
        assert result == []

    def test_no_filters_returns_all(self, mem_db):
        _seed_tasks(mem_db)
        result = task_service.get_tasks(conn=mem_db)
        assert len(result) == 3
        assert result[0]["task_id"] == "task-3"

    def test_filter_by_status(self, mem_db):
        _seed_tasks(mem_db)
        result = task_service.get_tasks(conn=mem_db, status="todo")
        assert len(result) == 1
        assert result[0]["task_id"] == "task-1"

    def test_filter_by_priority(self, mem_db):
        _seed_tasks(mem_db)
        result = task_service.get_tasks(conn=mem_db, priority="medium")
        assert len(result) == 1
        assert result[0]["task_id"] == "task-2"

    def test_filter_by_owner_role(self, mem_db):
        _seed_tasks(mem_db)
        result = task_service.get_tasks(conn=mem_db, owner_role="qa")
        assert len(result) == 1
        assert result[0]["task_id"] == "task-3"

    def test_filter_by_combined_conditions(self, mem_db):
        _seed_tasks(mem_db)
        result = task_service.get_tasks(
            conn=mem_db, status="in_progress", priority="medium", owner_role="dev"
        )
        assert len(result) == 1
        assert result[0]["task_id"] == "task-2"

    def test_filter_no_match(self, mem_db):
        _seed_tasks(mem_db)
        result = task_service.get_tasks(conn=mem_db, status="nonexistent")
        assert result == []

    def test_limit(self, mem_db):
        _seed_tasks(mem_db)
        result = task_service.get_tasks(conn=mem_db, limit=2)
        assert len(result) == 2


class TestGetTasksWithCount:
    def test_empty_table(self, mem_db):
        items, total = task_service.get_tasks_with_count(conn=mem_db)
        assert items == []
        assert total == 0

    def test_normal_return(self, mem_db):
        _seed_tasks(mem_db)
        items, total = task_service.get_tasks_with_count(conn=mem_db)
        assert len(items) == 3
        assert total == 3

    def test_with_filters(self, mem_db):
        _seed_tasks(mem_db)
        items, total = task_service.get_tasks_with_count(
            conn=mem_db, status="todo"
        )
        assert len(items) == 1
        assert total == 1

    def test_limit_less_than_total(self, mem_db):
        _seed_tasks(mem_db)
        items, total = task_service.get_tasks_with_count(
            conn=mem_db, limit=2
        )
        assert len(items) == 2
        assert total == 3


class TestGetTaskById:
    def test_existing_id(self, mem_db):
        _seed_tasks(mem_db)
        result = task_service.get_task_by_id(conn=mem_db, task_id="task-2")
        assert result is not None
        assert result["task_id"] == "task-2"
        assert result["status"] == "in_progress"

    def test_non_existing_id(self, mem_db):
        _seed_tasks(mem_db)
        result = task_service.get_task_by_id(
            conn=mem_db, task_id="task-999"
        )
        assert result is None

    def test_none_id(self, mem_db):
        _seed_tasks(mem_db)
        result = task_service.get_task_by_id(conn=mem_db, task_id=None)
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
            "services.task_service.get_db", fake_get_db
        )
        result = task_service.get_tasks()
        assert result == []
        assert "get_db" in calls
        assert "close" in calls
        assert fake_conn.closed

    def test_connection_not_closed_when_conn_passed(self, mem_db):
        result = task_service.get_tasks(conn=mem_db)
        assert result == []
        cur = mem_db.execute("SELECT 1")
        assert cur.fetchone()[0] == 1
