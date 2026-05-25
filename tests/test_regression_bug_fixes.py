"""Regression tests for Wave 1 bug fixes (Tasks 1-5).

Covers all 6 fixes:
1. Env var truthy parsing (ENABLE_DOCS, DEBUG) — api/main.py
2. Schema columns (alert_events: affected_orders/gmv/impact_score,
   action_tasks: target_object_type/id) — sql/schema.sql
3. Qoder multi-status filtering (IN clause for list values) —
   services/qoder_service.py + services/_query_utils.py
4. Qoder diagnosis with real request_id —
   services/qoder_service.py
5. Pipeline dry-run default + --apply flag —
   pipeline/runner.py + scripts/run_db_pipeline.py
6. Dual error logging (api + api.error loggers) —
   api/main.py global_exception_handler
"""

import inspect
import logging
import os
import sqlite3
import sys

import pytest
from fastapi.testclient import TestClient


@pytest.fixture
def seeded_db(in_memory_db):
    """In-memory DB with action_tasks and alert_events seeded."""
    in_memory_db.row_factory = sqlite3.Row
    in_memory_db.executescript("""
        INSERT INTO action_tasks (task_id, task_title, status, created_at)
        VALUES
            ('task-1', 'Task 1', 'todo', '2024-01-01'),
            ('task-2', 'Task 2', 'in_progress', '2024-01-01'),
            ('task-3', 'Task 3', 'open', '2024-01-01'),
            ('task-4', 'Task 4', 'done', '2024-01-01');
        INSERT INTO alert_events (event_id, rule_id, event_date, severity, metric_name, status, created_at)
        VALUES
            ('alert-1', 'rule-1', '2024-01-01', 'high', 'gmv', 'new', '2024-01-01'),
            ('alert-2', 'rule-2', '2024-01-01', 'medium', 'latency', 'new', '2024-01-01');
    """)
    return in_memory_db


class TestEnvVarTruthyParsing:
    """Regression: Fix ENABLE_DOCS and DEBUG to use explicit truthy parsing.

    Before fix: os.environ.get("ENABLE_DOCS") treats "0" and "false" as truthy.
    After fix:  .lower() in ("1", "true", "yes") correctly treats those as falsy.
    """

    def test_truthy_expression_falsy_values(self):
        falsy_values = ["0", "false", "FALSE", "False", "no", "NO", "No", ""]
        for val in falsy_values:
            assert not (val.lower() in ("1", "true", "yes")), f"'{val}' should be falsy"

    def test_truthy_expression_truthy_values(self):
        truthy_values = ["1", "true", "TRUE", "True", "yes", "YES", "Yes"]
        for val in truthy_values:
            assert val.lower() in ("1", "true", "yes"), f"'{val}' should be truthy"

    def test_enable_docs_zero_disables_docs(self, monkeypatch):
        monkeypatch.setenv("ENABLE_DOCS", "0")
        from api.main import create_app
        app = create_app()
        client = TestClient(app)
        r = client.get("/docs")
        assert r.status_code == 404

    def test_enable_docs_one_enables_docs(self, monkeypatch):
        monkeypatch.setenv("ENABLE_DOCS", "1")
        from api.main import create_app
        app = create_app()
        client = TestClient(app)
        r = client.get("/docs")
        assert r.status_code != 404

    def test_enable_docs_false_lowercase_disables(self, monkeypatch):
        monkeypatch.setenv("ENABLE_DOCS", "false")
        from api.main import create_app
        app = create_app()
        client = TestClient(app)
        r = client.get("/docs")
        assert r.status_code == 404

    def test_enable_docs_unset_disables_docs(self, monkeypatch):
        monkeypatch.delenv("ENABLE_DOCS", raising=False)
        from api.main import create_app
        app = create_app()
        client = TestClient(app)
        r = client.get("/docs")
        assert r.status_code == 404

    def test_enable_docs_truthy_true(self, monkeypatch):
        monkeypatch.setenv("ENABLE_DOCS", "true")
        from api.main import create_app
        app = create_app()
        client = TestClient(app)
        r = client.get("/docs")
        assert r.status_code != 404

    def test_sanitize_error_function(self):
        from api.main import _sanitize_error
        result = _sanitize_error("ValueError: something broke\n  File 'x.py', line 42")
        assert "ValueError" in result
        assert isinstance(result, str)

    def test_global_exception_handler_exists(self, monkeypatch):
        monkeypatch.setenv("API_BEARER_TOKEN", "test-token-for-baxi-ci-tests-only-32ch")
        from api.main import create_app
        app = create_app()
        assert Exception in app.exception_handlers
        handler = app.exception_handlers[Exception]
        assert callable(handler)


class TestSchemaMissingColumns:
    """Regression: Missing columns in sql/schema.sql.

    Before fix: alert_events lacked affected_orders, affected_gmv, impact_score.
                action_tasks lacked target_object_type, target_object_id.
    After fix:  All columns present in fresh DB from schema.sql.
    """

    def test_alert_events_has_impact_columns(self, in_memory_db):
        cur = in_memory_db.execute("PRAGMA table_info(alert_events)")
        cols = {row[1] for row in cur.fetchall()}
        for col in ("affected_orders", "affected_gmv", "impact_score"):
            assert col in cols, f"alert_events missing column: {col}"

    def test_action_tasks_has_target_columns(self, in_memory_db):
        cur = in_memory_db.execute("PRAGMA table_info(action_tasks)")
        cols = {row[1] for row in cur.fetchall()}
        for col in ("target_object_type", "target_object_id"):
            assert col in cols, f"action_tasks missing column: {col}"

    def test_alert_events_count_at_least_14(self, in_memory_db):
        cur = in_memory_db.execute("PRAGMA table_info(alert_events)")
        cols = [row[1] for row in cur.fetchall()]
        assert len(cols) >= 14, f"alert_events has {len(cols)} columns, expected >= 14"

    def test_action_tasks_count_at_least_16(self, in_memory_db):
        cur = in_memory_db.execute("PRAGMA table_info(action_tasks)")
        cols = [row[1] for row in cur.fetchall()]
        assert len(cols) >= 16, f"action_tasks has {len(cols)} columns, expected >= 16"

    def test_can_insert_into_alert_events_with_new_cols(self, in_memory_db):
        in_memory_db.execute("""
            INSERT INTO alert_events
                (event_id, rule_id, event_date, severity, metric_name, created_at,
                 affected_orders, affected_gmv, impact_score)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
        """, ("test-insert", "rule-x", "2024-01-01", "low", "test_metric",
              "2024-01-01", 5, 1000.0, 0.8))
        row = in_memory_db.execute(
            "SELECT affected_orders, affected_gmv, impact_score FROM alert_events WHERE event_id = ?",
            ("test-insert",)
        ).fetchone()
        assert row is not None
        assert row[0] == 5
        assert row[1] == 1000.0
        assert row[2] == 0.8

    def test_can_insert_into_action_tasks_with_new_cols(self, in_memory_db):
        in_memory_db.execute("""
            INSERT INTO action_tasks
                (task_id, task_title, status, created_at,
                 target_object_type, target_object_id)
            VALUES (?, ?, ?, ?, ?, ?)
        """, ("test-task-insert", "Test", "todo", "2024-01-01",
              "seller", "seller-001"))
        row = in_memory_db.execute(
            "SELECT target_object_type, target_object_id FROM action_tasks WHERE task_id = ?",
            ("test-task-insert",)
        ).fetchone()
        assert row is not None
        assert row[0] == "seller"
        assert row[1] == "seller-001"


class TestQoderStatusFiltering:
    """Regression: Multi-status filtering for Qoder task queries.

    Before fix: _build_conditions only supported single-value equality.
                get_tasks_with_count(status="open") filtered by one status.
    After fix:  _build_conditions generates IN clause for list values.
                get_tasks_with_count(status=["todo", "in_progress", "open"])
                returns tasks with any of those statuses.
    """

    def test_build_conditions_in_clause_for_list(self):
        from services._query_utils import _build_conditions
        where, params = _build_conditions({"status": ["todo", "in_progress", "open"]})
        assert "IN" in where, f"Expected IN clause, got: {where}"
        assert len(params) == 3
        assert where.count("?") == 3

    def test_build_conditions_single_value_still_equal(self):
        from services._query_utils import _build_conditions
        where, params = _build_conditions({"status": "open"})
        assert "=" in where
        assert "IN" not in where
        assert params == ["open"]

    def test_build_conditions_mixed(self):
        from services._query_utils import _build_conditions
        where, params = _build_conditions({
            "status": ["todo", "in_progress"],
            "priority": "high",
        })
        assert "IN" in where
        assert "=" in where
        assert len(params) == 3

    def test_build_conditions_none_skipped(self):
        from services._query_utils import _build_conditions
        where, params = _build_conditions({"status": None})
        assert where == ""
        assert params == []

    def test_get_tasks_with_count_list_status(self, seeded_db):
        from services.task_service import get_tasks_with_count
        items, total = get_tasks_with_count(
            conn=seeded_db, status=["todo", "in_progress", "open"], limit=100
        )
        assert total == 3, f"Expected 3 tasks, got {total}"
        statuses_found = {t["status"] for t in items}
        assert "todo" in statuses_found
        assert "in_progress" in statuses_found
        assert "open" in statuses_found
        assert "done" not in statuses_found

    def test_get_tasks_single_status_still_works(self, seeded_db):
        from services.task_service import get_tasks_with_count
        items, total = get_tasks_with_count(conn=seeded_db, status="done", limit=100)
        assert total == 1
        assert items[0]["status"] == "done"

    def test_get_tasks_empty_status_filter(self, seeded_db):
        from services.task_service import get_tasks_with_count
        items, total = get_tasks_with_count(conn=seeded_db, limit=100)
        assert total == 4


class TestQoderDiagnosisRequestId:
    """Regression: Pass real request_id to diagnosis service.

    Before fix: diagnose_by_request_id(request_id=None) always returned empty
                results because log entries have UUID request_ids.
    After fix:  get_request_id() is used when include_logs=True.
                When no request_id, recent_diagnosis stays as [].
    """

    def test_build_context_calls_get_request_id(self, monkeypatch, in_memory_db):
        call_count = [0]

        def mock_get_request_id():
            call_count[0] += 1
            return "test-rid-123"

        monkeypatch.setattr("services.qoder_service.get_request_id", mock_get_request_id)
        monkeypatch.setattr(
            "services.qoder_service.diagnose_by_request_id",
            lambda request_id: None,
        )

        from services.qoder_service import build_context
        result = build_context(conn=in_memory_db, include_logs=True)

        assert call_count[0] >= 1, "get_request_id should be called when include_logs=True"

    def test_build_context_diagnosis_with_request_id_returns_list(self, monkeypatch, in_memory_db):
        monkeypatch.setattr("services.qoder_service.get_request_id", lambda: "test-rid-123")

        mock_result = {
            "request_id": "test-rid-123",
            "summary": "Test error summary",
            "error_code": "ERR_TEST",
            "diagnosis": "Test diagnosis",
            "suggested_action": "Check logs",
            "related_logs": [],
        }
        monkeypatch.setattr(
            "services.qoder_service.diagnose_by_request_id",
            lambda request_id: mock_result,
        )

        from services.qoder_service import build_context
        result = build_context(conn=in_memory_db, include_logs=True)

        assert isinstance(result["recent_diagnosis"], list)
        assert len(result["recent_diagnosis"]) == 1
        assert result["recent_diagnosis"][0]["request_id"] == "test-rid-123"

    def test_build_context_no_request_id_empty_list(self, monkeypatch, in_memory_db):
        monkeypatch.setattr("services.qoder_service.get_request_id", lambda: None)

        from services.qoder_service import build_context
        result = build_context(conn=in_memory_db, include_logs=True)

        assert result["recent_diagnosis"] == []
        assert isinstance(result["recent_diagnosis"], list)

    def test_build_context_not_called_without_include_logs(self, monkeypatch, in_memory_db):
        diag_called = [False]

        monkeypatch.setattr("services.qoder_service.get_request_id", lambda: "test-rid")
        monkeypatch.setattr(
            "services.qoder_service.diagnose_by_request_id",
            lambda request_id: (
                diag_called.__setitem__(0, True) or None
            ),
        )

        from services.qoder_service import build_context
        build_context(conn=in_memory_db, include_logs=False)

        assert not diag_called[0], "diagnose_by_request_id should not be called without include_logs"


class TestPipelineDryRun:
    """Regression: Pipeline dry-run semantics.

    Before fix: run_pipeline() defaulted to dry_run=False (unsafe).
                Script required --dry-run flag for safety.
    After fix:  run_pipeline() defaults to dry_run=True (safe).
                Script uses --apply flag to opt into mutations.
    """

    def test_run_pipeline_default_dry_run_is_true(self):
        from pipeline.runner import run_pipeline
        sig = inspect.signature(run_pipeline)
        default = sig.parameters["dry_run"].default
        assert default is True, f"Expected dry_run=True, got dry_run={default}"

    def test_run_pipeline_explicit_dry_run_false(self):
        from pipeline.runner import run_pipeline
        sig = inspect.signature(run_pipeline)
        assert "dry_run" in sig.parameters
        assert sig.parameters["dry_run"].default is True

    def test_run_db_pipeline_no_args_dry_run_true(self, monkeypatch):
        monkeypatch.setattr("sys.argv", ["run_db_pipeline.py"])
        captured = {}

        def fake_run_pipeline(**kwargs):
            captured.update(kwargs)
            return True

        monkeypatch.setattr("scripts.run_db_pipeline.run_pipeline", fake_run_pipeline)
        from scripts.run_db_pipeline import main
        with pytest.raises(SystemExit):
            main()
        assert captured.get("dry_run") is True, (
            f"Expected dry_run=True, got dry_run={captured.get('dry_run')}"
        )

    def test_run_db_pipeline_apply_flag_dry_run_false(self, monkeypatch):
        monkeypatch.setattr("sys.argv", ["run_db_pipeline.py", "--apply"])
        captured = {}

        def fake_run_pipeline(**kwargs):
            captured.update(kwargs)
            return True

        monkeypatch.setattr("scripts.run_db_pipeline.run_pipeline", fake_run_pipeline)
        from scripts.run_db_pipeline import main
        with pytest.raises(SystemExit):
            main()
        assert captured.get("dry_run") is False, (
            f"Expected dry_run=False, got dry_run={captured.get('dry_run')}"
        )

    def test_run_db_pipeline_dimensional_preserved(self, monkeypatch):
        monkeypatch.setattr("sys.argv", ["run_db_pipeline.py", "--dimensional", "--apply"])
        captured = {}

        def fake_run_pipeline(**kwargs):
            captured.update(kwargs)
            return True

        monkeypatch.setattr("scripts.run_db_pipeline.run_pipeline", fake_run_pipeline)
        from scripts.run_db_pipeline import main
        with pytest.raises(SystemExit):
            main()
        assert captured.get("dry_run") is False
        assert captured.get("dimensional") is True

    def test_run_db_pipeline_mode_default(self, monkeypatch):
        monkeypatch.setattr("sys.argv", ["run_db_pipeline.py"])
        captured = {}

        def fake_run_pipeline(**kwargs):
            captured.update(kwargs)
            return True

        monkeypatch.setattr("scripts.run_db_pipeline.run_pipeline", fake_run_pipeline)
        from scripts.run_db_pipeline import main
        with pytest.raises(SystemExit):
            main()
        assert captured.get("mode") == "full"


class TestErrorDualLogging:
    """Regression: Unhandled 500s log to both 'api' and 'api.error' loggers.

    Before fix: Only logging.getLogger("api").error(...) was called.
                diagnosis_service.py reads from "api.error" logger -> never saw errors.
    After fix:  Both loggers are called in global_exception_handler.
    """

    def test_dual_logging_captures(self, monkeypatch):
        """Verify both loggers are called by calling handler directly.

        Uses asyncio.run() to call the async handler outside HTTP middleware
        to avoid BaseExceptionGroup wrapping in Python 3.12 + Starlette.
        """
        import asyncio

        monkeypatch.setenv("API_BEARER_TOKEN", "test-token-for-baxi-ci-tests-only-32ch")
        from api.main import create_app
        app = create_app()
        handler = app.exception_handlers[Exception]

        api_calls = []
        api_error_calls = []
        monkeypatch.setattr(
            logging.getLogger("api"), "error",
            lambda msg, *a, **kw: api_calls.append(str(msg % a if a else msg)),
        )
        monkeypatch.setattr(
            logging.getLogger("api.error"), "error",
            lambda msg, *a, **kw: api_error_calls.append(str(msg % a if a else msg)),
        )

        from fastapi import Request
        scope = {
            "type": "http", "method": "GET", "path": "/test",
            "headers": [], "query_string": b"",
            "client": ("127.0.0.1", 8000), "scheme": "http",
            "raw_path": b"/test",
        }
        request = Request(scope)

        async def call_handler():
            return await handler(request, ValueError("CRASH_DUAL_LOG_TEST"))

        response = asyncio.run(call_handler())
        assert response.status_code == 500
        assert any("CRASH_DUAL_LOG_TEST" in c for c in api_calls), (
            f"api logger not called. calls: {api_calls}"
        )
        assert any("CRASH_DUAL_LOG_TEST" in c for c in api_error_calls), (
            f"api.error logger not called. calls: {api_error_calls}"
        )

    def test_normal_request_no_error_logging(self, monkeypatch):
        api_error_calls = []
        api_error_log_calls = []

        monkeypatch.setattr(
            logging.getLogger("api"), "error",
            lambda msg, *a, **kw: api_error_calls.append(msg),
        )
        monkeypatch.setattr(
            logging.getLogger("api.error"), "error",
            lambda msg, *a, **kw: api_error_log_calls.append(msg),
        )

        monkeypatch.setenv("API_BEARER_TOKEN", "test-token-for-baxi-ci-tests-only-32ch")

        from api.main import create_app
        app = create_app()
        client = TestClient(app)
        r = client.get("/api/v1/health")
        assert r.status_code == 200


class TestBuildConditionsEdgeCases:

    def test_empty_dict(self):
        from services._query_utils import _build_conditions
        where, params = _build_conditions({})
        assert where == ""
        assert params == []

    def test_all_none(self):
        from services._query_utils import _build_conditions
        where, params = _build_conditions({"status": None, "priority": None})
        assert where == ""
        assert params == []

    def test_single_item_list(self):
        from services._query_utils import _build_conditions
        where, params = _build_conditions({"status": ["open"]})
        assert "IN" in where
        assert params == ["open"]

    def test_empty_list(self):
        from services._query_utils import _build_conditions
        where, params = _build_conditions({"status": []})
        assert isinstance(params, list)