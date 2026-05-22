#!/usr/bin/env python3
"""
test_db_ingestion.py — Test data ingestion into SQLite decision backend.

Checks:
- dwd_order_level has data
- dwd_item_level has data
- Primary keys are unique (no duplicates)
- ingestion_batches has records
"""

import os
import sys
import sqlite3
import pytest

import scripts.config as config

DB_PATH = os.path.join(config.PROJECT_ROOT, 'data', 'olist_ops.db')


@pytest.fixture
def connection():
    conn = sqlite3.connect(DB_PATH)
    yield conn
    conn.close()


class TestIngestionData:
    """Test that ingestion produced data."""

    def test_order_level_has_data(self, connection):
        cur = connection.execute("SELECT COUNT(*) FROM dwd_order_level")
        count = cur.fetchone()[0]
        assert count > 0, "dwd_order_level is empty"

    def test_item_level_has_data(self, connection):
        cur = connection.execute("SELECT COUNT(*) FROM dwd_item_level")
        count = cur.fetchone()[0]
        assert count > 0, "dwd_item_level is empty"

    def test_order_count_expected(self, connection):
        cur = connection.execute("SELECT COUNT(*) FROM dwd_order_level")
        count = cur.fetchone()[0]
        assert count == 99441, f"Expected 99441 orders, got {count}"

    def test_item_count_expected(self, connection):
        cur = connection.execute("SELECT COUNT(*) FROM dwd_item_level")
        count = cur.fetchone()[0]
        assert count == 112650, f"Expected 112650 items, got {count}"

    def test_ingestion_batches_has_record(self, connection):
        cur = connection.execute("SELECT COUNT(*) FROM ingestion_batches")
        count = cur.fetchone()[0]
        assert count >= 1, "No ingestion_batches records"

    def test_pipeline_runs_has_record(self, connection):
        cur = connection.execute("SELECT COUNT(*) FROM pipeline_runs")
        count = cur.fetchone()[0]
        assert count >= 1, "No pipeline_runs records"


class TestPrimaryKeysUnique:
    """Test that primary keys have no duplicates."""

    def test_order_id_unique(self, connection):
        cur = connection.execute("""
            SELECT order_id, COUNT(*) FROM dwd_order_level
            GROUP BY order_id HAVING COUNT(*) > 1
        """)
        dupes = cur.fetchall()
        assert len(dupes) == 0, f"Duplicate order_ids: {dupes[:5]}"

    def test_item_key_unique(self, connection):
        cur = connection.execute("""
            SELECT item_key, COUNT(*) FROM dwd_item_level
            GROUP BY item_key HAVING COUNT(*) > 1
        """)
        dupes = cur.fetchall()
        assert len(dupes) == 0, f"Duplicate item_keys: {dupes[:5]}"

    def test_metric_date_unique(self, connection):
        cur = connection.execute("""
            SELECT metric_date, COUNT(*) FROM metric_daily
            GROUP BY metric_date HAVING COUNT(*) > 1
        """)
        dupes = cur.fetchall()
        assert len(dupes) == 0, f"Duplicate metric_dates: {dupes[:5]}"

    def test_event_id_unique(self, connection):
        cur = connection.execute("""
            SELECT event_id, COUNT(*) FROM alert_events
            GROUP BY event_id HAVING COUNT(*) > 1
        """)
        dupes = cur.fetchall()
        assert len(dupes) == 0, f"Duplicate event_ids: {dupes[:5]}"


class TestDataIntegrity:
    """Test data integrity in ingested data."""

    def test_purchase_dates_exist(self, connection):
        cur = connection.execute("""
            SELECT COUNT(*) FROM dwd_order_level WHERE purchase_date IS NOT NULL
        """)
        count = cur.fetchone()[0]
        assert count > 0, "No purchase_dates set"

    def test_payment_values_positive(self, connection):
        cur = connection.execute("""
            SELECT COUNT(*) FROM dwd_order_level WHERE payment_value < 0
        """)
        count = cur.fetchone()[0]
        assert count == 0, f"Found {count} negative payment_values"

    def test_prices_positive(self, connection):
        cur = connection.execute("""
            SELECT COUNT(*) FROM dwd_item_level WHERE price < 0
        """)
        count = cur.fetchone()[0]
        assert count == 0, f"Found {count} negative prices"
