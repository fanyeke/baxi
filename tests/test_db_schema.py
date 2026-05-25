#!/usr/bin/env python3
"""
test_db_schema.py — Verify SQLite schema for v0.2 decision backend.

Checks:
- All 12 core tables exist
- Key columns present in each table
- Primary keys defined
"""

import os
import sys
import sqlite3
import pytest

import scripts.config as config

pytestmark = pytest.mark.integration

DB_PATH = os.path.join(config.PROJECT_ROOT, 'data', 'olist_ops.db')

EXPECTED_TABLES = [
    'pipeline_runs', 'ingestion_batches',
    'dwd_order_level', 'dwd_item_level',
    'metric_daily', 'metric_dimension_daily',
    'alert_events', 'strategy_recommendations',
    'action_tasks', 'review_retro',
    'event_outbox', 'qoder_jobs',
]

EXPECTED_COLUMNS = {
    'pipeline_runs': ['run_id', 'run_type', 'mode', 'status', 'started_at'],
    'ingestion_batches': ['batch_id', 'source_name', 'ingestion_mode', 'row_count', 'status'],
    'dwd_order_level': ['order_id', 'customer_id', 'purchase_date', 'payment_value', 'review_score'],
    'dwd_item_level': ['item_key', 'order_id', 'product_id', 'seller_id', 'price', 'freight_value'],
    'metric_daily': ['metric_date', 'gmv', 'order_count', 'cancel_rate', 'late_delivery_rate'],
    'metric_dimension_daily': ['metric_date', 'dimension_type', 'dimension_value', 'metric_name', 'metric_value'],
    'alert_events': ['event_id', 'rule_id', 'event_date', 'severity', 'metric_name', 'status'],
    'strategy_recommendations': ['recommendation_id', 'event_id', 'decision_source', 'execution_status'],
    'action_tasks': ['task_id', 'task_title', 'status', 'priority', 'owner_role'],
    'review_retro': ['review_id', 'recommendation_id', 'task_id', 'is_effective'],
    'event_outbox': ['outbox_id', 'event_type', 'target_channel', 'status', 'payload_json'],
    'qoder_jobs': ['job_id', 'job_type', 'dispatch_channel', 'dispatch_status'],
}


@pytest.fixture
def connection():
    """Provide a database connection."""
    conn = sqlite3.connect(DB_PATH)
    yield conn
    conn.close()


def get_table_columns(conn, table_name):
    """Get list of column names for a table."""
    cur = conn.execute(f"PRAGMA table_info({table_name})")
    return [row[1] for row in cur.fetchall()]


def get_primary_keys(conn, table_name):
    """Get primary key columns for a table."""
    cur = conn.execute(f"PRAGMA table_info({table_name})")
    return [row[1] for row in cur.fetchall() if row[5] > 0]


class TestSchemaTablesExist:
    """Test that all 12 core tables exist."""

    @pytest.fixture(autouse=True)
    def setup(self, connection):
        self.conn = connection
        cur = self.conn.execute(
            "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name"
        )
        self.tables = [row[0] for row in cur.fetchall()]

    def test_pipeline_runs_exists(self):
        assert 'pipeline_runs' in self.tables

    def test_ingestion_batches_exists(self):
        assert 'ingestion_batches' in self.tables

    def test_dwd_order_level_exists(self):
        assert 'dwd_order_level' in self.tables

    def test_dwd_item_level_exists(self):
        assert 'dwd_item_level' in self.tables

    def test_metric_daily_exists(self):
        assert 'metric_daily' in self.tables

    def test_metric_dimension_daily_exists(self):
        assert 'metric_dimension_daily' in self.tables

    def test_alert_events_exists(self):
        assert 'alert_events' in self.tables

    def test_strategy_recommendations_exists(self):
        assert 'strategy_recommendations' in self.tables

    def test_action_tasks_exists(self):
        assert 'action_tasks' in self.tables

    def test_review_retro_exists(self):
        assert 'review_retro' in self.tables

    def test_event_outbox_exists(self):
        assert 'event_outbox' in self.tables

    def test_qoder_jobs_exists(self):
        assert 'qoder_jobs' in self.tables

    def test_all_twelve_tables_exist(self):
        for table in EXPECTED_TABLES:
            assert table in self.tables, f"Missing table: {table}"


class TestKeyColumnsPresent:
    """Test that key columns are present in each table."""

    @pytest.fixture(autouse=True)
    def setup(self, connection):
        self.conn = connection

    def test_order_level_columns(self):
        columns = get_table_columns(self.conn, 'dwd_order_level')
        for col in EXPECTED_COLUMNS['dwd_order_level']:
            assert col in columns, f"Missing column {col} in dwd_order_level"

    def test_item_level_columns(self):
        columns = get_table_columns(self.conn, 'dwd_item_level')
        for col in EXPECTED_COLUMNS['dwd_item_level']:
            assert col in columns, f"Missing column {col} in dwd_item_level"

    def test_metric_daily_columns(self):
        columns = get_table_columns(self.conn, 'metric_daily')
        for col in EXPECTED_COLUMNS['metric_daily']:
            assert col in columns, f"Missing column {col} in metric_daily"

    def test_alert_events_columns(self):
        columns = get_table_columns(self.conn, 'alert_events')
        for col in EXPECTED_COLUMNS['alert_events']:
            assert col in columns, f"Missing column {col} in alert_events"


class TestPrimaryKeys:
    """Test that primary keys are defined."""

    @pytest.fixture(autouse=True)
    def setup(self, connection):
        self.conn = connection

    def test_order_level_has_pk(self):
        pks = get_primary_keys(self.conn, 'dwd_order_level')
        assert 'order_id' in pks

    def test_item_level_has_pk(self):
        pks = get_primary_keys(self.conn, 'dwd_item_level')
        assert 'item_key' in pks

    def test_metric_daily_has_pk(self):
        pks = get_primary_keys(self.conn, 'metric_daily')
        assert 'metric_date' in pks

    def test_pipeline_runs_has_pk(self):
        pks = get_primary_keys(self.conn, 'pipeline_runs')
        assert 'run_id' in pks
