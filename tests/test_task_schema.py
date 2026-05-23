"""TDD tests for TaskItem schema ↔ SELECT query alignment.

Verifies that all 15 action_tasks columns flow through:
  DB → SELECT query → API response → TaskItem schema
"""
import os
import sys

import pytest

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "scripts"))

from scripts import config
from fastapi.testclient import TestClient

os.environ["API_BEARER_TOKEN"] = "test-token"

ALL_15_FIELDS = [
    "task_id", "task_title", "task_description", "status", "priority",
    "owner_role", "owner_user_id", "due_at", "created_at", "completed_at",
    "feedback", "recommendation_id", "event_id", "target_object_type",
    "target_object_id",
]


@pytest.fixture(scope="function")
def client(tmp_path, monkeypatch):
    """Create a TestClient with a seeded SQLite test database."""
    import sqlite3
    test_db = str(tmp_path / "test_schema_alignment.db")
    conn = sqlite3.connect(test_db)
    conn.executescript("""
        CREATE TABLE IF NOT EXISTS action_tasks (
            task_id TEXT PRIMARY KEY, recommendation_id TEXT, event_id TEXT,
            task_title TEXT, task_description TEXT, task_source TEXT,
            owner_role TEXT, owner_user_id TEXT, priority TEXT DEFAULT 'medium',
            due_at TEXT, status TEXT DEFAULT 'todo', feedback TEXT,
            completed_at TEXT, created_at TEXT,
            target_object_type TEXT, target_object_id TEXT);
        CREATE TABLE IF NOT EXISTS pipeline_runs (
            run_id TEXT PRIMARY KEY, run_type TEXT, mode TEXT, status TEXT,
            started_at TEXT, finished_at TEXT, input_count INTEGER,
            output_count INTEGER, error_message TEXT);
        INSERT OR IGNORE INTO action_tasks VALUES
            ('task-1','rec-1','alert-1','Fix critical bug',
             'Investigate production issue','heuristic','business_ops',
             'user-42','high','2026-06-01','in_progress','Needs more info',
             NULL,'2026-05-01',NULL,NULL),
            ('task-2','rec-2',NULL,'Update dashboard',
             'Add new KPI cards','heuristic','seller_ops',NULL,'medium',
             NULL,'todo',NULL,'2026-05-15','2026-05-01',NULL,NULL),
            ('task-3','rec-3','alert-3','Review alert rules',
             NULL,'heuristic','business_ops','user-99','low',NULL,'done',
             'All rules validated','2026-05-10','2026-05-01',
             'alert_event','alert-3');
    """)
    conn.commit()
    conn.close()
    monkeypatch.setattr(config, "DB_PATH", test_db)
    from api.main import create_app
    app = create_app()
    return TestClient(app)


@pytest.fixture
def auth_headers():
    return {"Authorization": "Bearer test-token"}


class TestTaskItemIncludesAllDbFields:
    """Every item returned by GET /tasks must have all 15 fields."""

    def test_task_item_includes_all_db_fields(self, client, auth_headers):
        r = client.get("/api/v1/tasks?limit=10", headers=auth_headers)
        assert r.status_code == 200
        data = r.json()
        items = data["items"]
        assert len(items) > 0, "Expected at least one task in test DB"
        for item in items:
            for field in ALL_15_FIELDS:
                assert field in item, (
                    f"Missing field '{field}' in task item with keys: "
                    f"{sorted(item.keys())}"
                )


class TestTaskListResponseMatchesSchema:
    """The Pydantic TaskListResponse must accept every API item without errors."""

    def test_task_list_response_matches_schema(self, client, auth_headers):
        from api.schemas import TaskListResponse

        r = client.get("/api/v1/tasks?limit=10", headers=auth_headers)
        assert r.status_code == 200
        json_data = r.json()

        # This will raise ValidationError if any item has extra/missing fields
        parsed = TaskListResponse.model_validate(json_data)
        assert len(parsed.items) == len(json_data["items"])


class TestCompletedAtAndFeedbackPresent:
    """The 2 fields currently silently dropped must appear in API response."""

    def test_completed_at_and_feedback_present(self, client, auth_headers):
        r = client.get("/api/v1/tasks?limit=10", headers=auth_headers)
        assert r.status_code == 200
        items = r.json()["items"]

        completed_at_values = [item.get("completed_at") for item in items]
        feedback_values = [item.get("feedback") for item in items]

        # Fields must exist in every item (None is valid, missing key is not)
        for item in items:
            assert "completed_at" in item, (
                "completed_at missing from response — schema silent drop bug"
            )
            assert "feedback" in item, (
                "feedback missing from response — schema silent drop bug"
            )

        # At least one task has feedback (task-1: 'Needs more info')
        assert any(v is not None for v in feedback_values), (
            "Expected at least one non-null feedback value"
        )
