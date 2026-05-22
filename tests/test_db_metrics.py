#!/usr/bin/env python3
"""
test_db_metrics.py — Test daily metrics computed from SQLite.

Checks:
- metric_daily has data
- metric_date is unique
- gmv >= 0
- order_count >= 0
- cancel_rate between 0 and 1
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


class TestMetricDailyData:
    """Test metric_daily table has valid data."""

    def test_metric_daily_has_data(self, connection):
        cur = connection.execute("SELECT COUNT(*) FROM metric_daily")
        count = cur.fetchone()[0]
        assert count > 0, "metric_daily is empty"
        assert count >= 600, f"Expected ~634 rows, got {count}"

    def test_metric_date_unique(self, connection):
        cur = connection.execute("""
            SELECT metric_date, COUNT(*) FROM metric_daily
            GROUP BY metric_date HAVING COUNT(*) > 1
        """)
        dupes = cur.fetchall()
        assert len(dupes) == 0, f"Duplicate metric_dates: {dupes}"

    def test_date_range_reasonable(self, connection):
        cur = connection.execute("SELECT MIN(metric_date), MAX(metric_date) FROM metric_daily")
        min_date, max_date = cur.fetchone()
        assert min_date is not None
        assert max_date is not None
        assert min_date < max_date

    def test_metric_dates_sorted_orderable(self, connection):
        cur = connection.execute("""
            SELECT COUNT(*) FROM metric_daily
            WHERE metric_date NOT LIKE '____-__-__'
        """)
        bad = cur.fetchone()[0]
        assert bad == 0, f"{bad} rows with non-YYYY-MM-DD metric_date"


class TestMetricValues:
    """Test that metric values are within expected ranges."""

    def test_gmv_non_negative(self, connection):
        cur = connection.execute("""
            SELECT COUNT(*) FROM metric_daily WHERE gmv < 0
        """)
        count = cur.fetchone()[0]
        assert count == 0, f"Found {count} rows with negative GMV"

    def test_order_count_non_negative(self, connection):
        cur = connection.execute("""
            SELECT COUNT(*) FROM metric_daily WHERE order_count < 0
        """)
        count = cur.fetchone()[0]
        assert count == 0, f"Found {count} rows with negative order_count"

    def test_cancel_rate_range(self, connection):
        cur = connection.execute("""
            SELECT COUNT(*) FROM metric_daily WHERE cancel_rate < 0 OR cancel_rate > 1
        """)
        count = cur.fetchone()[0]
        assert count == 0, f"Found {count} rows with cancel_rate out of [0,1]"

    def test_late_delivery_rate_range(self, connection):
        cur = connection.execute("""
            SELECT COUNT(*) FROM metric_daily WHERE late_delivery_rate < 0 OR late_delivery_rate > 1
        """)
        count = cur.fetchone()[0]
        assert count == 0, f"Found {count} rows with late_delivery_rate out of [0,1]"

    def test_low_review_rate_range(self, connection):
        cur = connection.execute("""
            SELECT COUNT(*) FROM metric_daily WHERE low_review_rate < 0 OR low_review_rate > 1
        """)
        count = cur.fetchone()[0]
        assert count == 0, f"Found {count} rows with low_review_rate out of [0,1]"

    def test_avg_order_value_positive(self, connection):
        cur = connection.execute("""
            SELECT COUNT(*) FROM metric_daily WHERE avg_order_value < 0
        """)
        count = cur.fetchone()[0]
        assert count == 0, f"Found {count} rows with negative avg_order_value"

    def test_review_score_range(self, connection):
        cur = connection.execute("""
            SELECT COUNT(*) FROM metric_daily
            WHERE avg_review_score < 0 OR avg_review_score > 5
        """)
        count = cur.fetchone()[0]
        assert count == 0, f"Found {count} rows with avg_review_score out of [0,5]"

    def test_created_at_not_null(self, connection):
        cur = connection.execute("""
            SELECT COUNT(*) FROM metric_daily WHERE created_at IS NULL
        """)
        count = cur.fetchone()[0]
        assert count == 0, f"Found {count} rows with NULL created_at"
