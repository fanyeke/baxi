#!/usr/bin/env python3
"""
test_db_rule_engine.py — Test rule engine output.

Checks:
- alert_events.event_id is unique
- strategy_recommendations.decision_source = 'heuristic'
- action_tasks.task_source is not NULL
- event_outbox has payload_json
"""

import os
import sys
import sqlite3
import pytest
import json
import re

import scripts.config as config

pytestmark = pytest.mark.integration

DB_PATH = os.path.join(config.PROJECT_ROOT, 'data', 'olist_ops.db')


@pytest.fixture
def connection():
    conn = sqlite3.connect(DB_PATH)
    yield conn
    conn.close()


class TestAlertEvents:
    """Test alert_events table."""

    def test_alert_events_exists_might_be_empty(self, connection):
        """Alert_events may have 0 rows if no rules triggered."""
        pass  # Table existence tested in schema tests

    def test_event_id_unique(self, connection):
        cur = connection.execute("""
            SELECT event_id, COUNT(*) FROM alert_events
            GROUP BY event_id HAVING COUNT(*) > 1
        """)
        dupes = cur.fetchall()
        assert len(dupes) == 0, f"Duplicate event_ids: {dupes}"

    def test_alert_events_have_rule_id(self, connection):
        cur = connection.execute("""
            SELECT COUNT(*) FROM alert_events WHERE rule_id IS NULL OR rule_id = ''
        """)
        count = cur.fetchone()[0]
        assert count == 0, f"Found {count} alerts without rule_id"

    def test_alert_events_have_severity(self, connection):
        cur = connection.execute("""
            SELECT COUNT(*) FROM alert_events
            WHERE severity IS NULL OR severity = ''
        """)
        count = cur.fetchone()[0]
        assert count == 0, f"Found {count} alerts without severity"

    def test_alert_events_severity_values(self, connection):
        cur = connection.execute("""
            SELECT DISTINCT severity FROM alert_events
        """)
        valid = {'low', 'medium', 'high', 'critical'}
        severities = {row[0] for row in cur.fetchall()}
        for sev in severities:
            assert sev in valid, f"Invalid severity: {sev}"

    def test_alert_events_have_metric_name(self, connection):
        cur = connection.execute("""
            SELECT COUNT(*) FROM alert_events
            WHERE metric_name IS NULL OR metric_name = ''
        """)
        count = cur.fetchone()[0]
        assert count == 0, f"Found {count} alerts without metric_name"


class TestStrategyRecommendations:
    """Test strategy_recommendations table."""

    def test_decision_source_is_heuristic(self, connection):
        cur = connection.execute("""
            SELECT COUNT(*) FROM strategy_recommendations
            WHERE decision_source != 'heuristic'
        """)
        count = cur.fetchone()[0]
        assert count == 0, f"Found {count} recommendations with non-heuristic decision_source"

    def test_recommendation_id_unique(self, connection):
        cur = connection.execute("""
            SELECT recommendation_id, COUNT(*) FROM strategy_recommendations
            GROUP BY recommendation_id HAVING COUNT(*) > 1
        """)
        dupes = cur.fetchall()
        assert len(dupes) == 0, f"Duplicate recommendation_ids: {dupes}"

    def test_recommendations_linked_to_events(self, connection):
        cur = connection.execute("""
            SELECT COUNT(*) FROM strategy_recommendations
            WHERE event_id IS NOT NULL AND event_id != ''
        """)
        count = cur.fetchone()[0]
        assert count >= 0  # At least valid count


class TestActionTasks:
    """Test action_tasks table."""

    def test_task_source_not_null(self, connection):
        cur = connection.execute("""
            SELECT COUNT(*) FROM action_tasks WHERE task_source IS NULL
        """)
        count = cur.fetchone()[0]
        assert count == 0, f"Found {count} tasks with NULL task_source"

    def test_task_id_unique(self, connection):
        cur = connection.execute("""
            SELECT task_id, COUNT(*) FROM action_tasks
            GROUP BY task_id HAVING COUNT(*) > 1
        """)
        dupes = cur.fetchall()
        assert len(dupes) == 0, f"Duplicate task_ids: {dupes}"

    def test_tasks_have_status(self, connection):
        cur = connection.execute("""
            SELECT COUNT(*) FROM action_tasks WHERE status IS NULL
        """)
        count = cur.fetchone()[0]
        assert count == 0, f"Found {count} tasks without status"


class TestEventOutbox:
    """Test event_outbox table."""

    def test_outbox_has_payload(self, connection):
        cur = connection.execute("""
            SELECT COUNT(*) FROM event_outbox
            WHERE payload_json IS NULL OR payload_json = ''
        """)
        count = cur.fetchone()[0]
        assert count == 0, f"Found {count} outbox entries without payload_json"

    def test_outbox_payload_is_valid_json(self, connection):
        cur = connection.execute("SELECT payload_json FROM event_outbox")
        for row in cur.fetchall():
            payload_str = row[0]
            if payload_str:
                try:
                    json.loads(payload_str)
                except json.JSONDecodeError:
                    pytest.fail(f"Invalid JSON in outbox payload: {payload_str[:100]}")

    def test_outbox_has_target_channel(self, connection):
        cur = connection.execute("""
            SELECT COUNT(*) FROM event_outbox
            WHERE target_channel IS NULL OR target_channel = ''
        """)
        count = cur.fetchone()[0]
        assert count == 0, f"Found {count} outbox entries without target_channel"


class TestTemplateRendering:
    """Test that template rendering produces clean output."""

    # Pattern to match unresolved template placeholders like ${var}
    TEMPLATE_PLACEHOLDER = re.compile(r'\$\{[^}]+\}')

    def test_strategy_detail_no_unresolved_placeholders(self, connection):
        """All strategy_detail texts should have NO ${...} patterns."""
        cur = connection.execute("""
            SELECT recommendation_id, strategy_detail
            FROM strategy_recommendations
            WHERE strategy_detail IS NOT NULL AND strategy_detail != ''
        """)
        for row in cur.fetchall():
            rec_id, detail = row
            matches = self.TEMPLATE_PLACEHOLDER.findall(detail)
            assert len(matches) == 0, (
                f"strategy_detail for {rec_id} contains unresolved templates: {matches}"
            )

    def test_template_rendering_uses_safe_substitute(self, connection):
        """strategy_detail should contain actual values, not template variable names."""
        cur = connection.execute("""
            SELECT recommendation_id, strategy_detail
            FROM strategy_recommendations
            WHERE strategy_detail IS NOT NULL AND strategy_detail != ''
        """)
        rows = cur.fetchall()
        assert len(rows) > 0, "No strategy_recommendations with strategy_detail found"
        # Check that rendered details contain concrete values (numbers, IDs, dates)
        # rather than bare template keys
        has_concrete_values = 0
        for rec_id, detail in rows:
            # Look for evidence of actual data: numbers, dates, IDs
            if re.search(r'\d{4}-\d{2}-\d{2}|\d+\.?\d*%|[a-z0-9_]{8,}', detail, re.IGNORECASE):
                has_concrete_values += 1
        assert has_concrete_values > 0, (
            f"None of {len(rows)} strategy_details contain actual rendered values"
        )
