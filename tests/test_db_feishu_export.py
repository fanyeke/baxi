#!/usr/bin/env python3
"""
test_db_feishu_export.py — Test Feishu CSV export from SQLite.

Checks:
- 5 CSV files exist
- Primary keys non-null
- Fields match expected column names
"""

import os
import sys
import csv
import pytest

import scripts.config as config

FEISHU_DIR = config.FEISHU_DIR

EXPECTED_CSV_FILES = [
    'daily_metrics_for_feishu.csv',
    'alert_events_for_feishu.csv',
    'strategy_recommendations_for_feishu.csv',
    'action_tasks_for_feishu.csv',
    'review_retro_for_feishu.csv',
]


class TestExportFilesExist:
    """Test that all 5 Feishu CSV files exist."""

    def test_daily_metrics_csv_exists(self):
        path = os.path.join(FEISHU_DIR, 'daily_metrics_for_feishu.csv')
        assert os.path.exists(path), f"Missing: {path}"

    def test_alert_events_csv_exists(self):
        path = os.path.join(FEISHU_DIR, 'alert_events_for_feishu.csv')
        assert os.path.exists(path), f"Missing: {path}"

    def test_strategy_recommendations_csv_exists(self):
        path = os.path.join(FEISHU_DIR, 'strategy_recommendations_for_feishu.csv')
        assert os.path.exists(path), f"Missing: {path}"

    def test_action_tasks_csv_exists(self):
        path = os.path.join(FEISHU_DIR, 'action_tasks_for_feishu.csv')
        assert os.path.exists(path), f"Missing: {path}"

    def test_review_retro_csv_exists(self):
        path = os.path.join(FEISHU_DIR, 'review_retro_for_feishu.csv')
        assert os.path.exists(path), f"Missing: {path}"

    def test_all_five_csvs_exist(self):
        for filename in EXPECTED_CSV_FILES:
            path = os.path.join(FEISHU_DIR, filename)
            assert os.path.exists(path), f"Missing CSV: {path}"


class TestCSVColumnHeaders:
    """Test that CSV column headers are present."""

    def _read_headers(self, filename):
        path = os.path.join(FEISHU_DIR, filename)
        with open(path, 'r', encoding='utf-8') as f:
            reader = csv.reader(f)
            return next(reader)

    def test_daily_metrics_headers(self):
        headers = self._read_headers('daily_metrics_for_feishu.csv')
        assert 'simulated_date' in headers or 'metric_date' in headers or 'real_run_date' in headers

    def test_alert_events_headers(self):
        headers = self._read_headers('alert_events_for_feishu.csv')
        assert 'alert_id' in headers or 'event_id' in headers

    def test_strategy_recommendations_headers(self):
        headers = self._read_headers('strategy_recommendations_for_feishu.csv')
        assert 'recommendation_id' in headers or 'title' in headers

    def test_action_tasks_headers(self):
        headers = self._read_headers('action_tasks_for_feishu.csv')
        assert 'task_id' in headers or 'title' in headers


class TestDimensionalExportColumns:
    """Test dimensional column presence and data quality in exported CSVs."""

    @staticmethod
    def _read_csv(filename):
        """Read a CSV file and return (headers, rows) as lists."""
        path = os.path.join(FEISHU_DIR, filename)
        with open(path, 'r', encoding='utf-8') as f:
            reader = csv.reader(f)
            headers = next(reader)
            rows = list(reader)
        return headers, rows

    # -- Alert events: dimensional columns presence --

    def test_alert_events_has_object_type_and_id(self):
        headers, _ = self._read_csv('alert_events_for_feishu.csv')
        assert 'object_type' in headers, "alert_events missing 'object_type' column"
        assert 'object_id' in headers, "alert_events missing 'object_id' column"

    def test_alert_events_has_dimensional_metrics(self):
        headers, _ = self._read_csv('alert_events_for_feishu.csv')
        for col in ('affected_orders', 'affected_gmv', 'impact_score'):
            assert col in headers, f"alert_events missing '{col}' column"

    # -- Alert events: dimensional data quality --

    def test_alert_events_dimensional_non_null(self):
        headers, rows = self._read_csv('alert_events_for_feishu.csv')
        ot_idx = headers.index('object_type')
        oi_idx = headers.index('object_id')
        for i, row in enumerate(rows):
            assert len(row) > ot_idx, f"Row {i} too short for object_type"
            assert len(row) > oi_idx, f"Row {i} too short for object_id"
            assert row[ot_idx].strip() != '', f"Row {i}: object_type is null/empty"
            assert row[oi_idx].strip() != '', f"Row {i}: object_id is null/empty"

    def test_alert_events_impact_score_numeric(self):
        headers, rows = self._read_csv('alert_events_for_feishu.csv')
        for col in ('affected_orders', 'affected_gmv', 'impact_score'):
            assert col in headers, f"alert_events missing '{col}' column"
            idx = headers.index(col)
            for i, row in enumerate(rows):
                if len(row) <= idx:
                    continue  # short rows are OK, handled by other test
                val = row[idx].strip()
                if val == '' or val.lower() == 'null' or val.lower() == 'none':
                    continue  # nullable is OK
                try:
                    float(val)
                except ValueError:
                    pytest.fail(f"Row {i}: '{col}' is not numeric: '{val}'")

    # -- Action tasks: target_object columns presence --

    def test_action_tasks_has_target_object(self):
        headers, _ = self._read_csv('action_tasks_for_feishu.csv')
        assert 'target_object_type' in headers, "action_tasks missing 'target_object_type' column"
        assert 'target_object_id' in headers, "action_tasks missing 'target_object_id' column"

    def test_action_tasks_target_object_populated(self):
        headers, rows = self._read_csv('action_tasks_for_feishu.csv')
        if len(rows) == 0:
            pytest.skip("action_tasks CSV has no data rows")
        tot_idx = headers.index('target_object_type')
        toi_idx = headers.index('target_object_id')
        populated = 0
        for row in rows:
            if len(row) > tot_idx and row[tot_idx].strip() != '' \
               and len(row) > toi_idx and row[toi_idx].strip() != '':
                populated += 1
        pct = populated / len(rows)
        assert pct >= 0.90, f"Only {pct:.1%} of action_tasks rows have target_object_type/id populated (expected ≥90%)"


class TestCSVDataIntegrity:
    """Test CSV data integrity."""

    def test_daily_metrics_has_rows(self):
        path = os.path.join(FEISHU_DIR, 'daily_metrics_for_feishu.csv')
        with open(path, 'r', encoding='utf-8') as f:
            reader = csv.reader(f)
            next(reader)  # skip header
            rows = list(reader)
        assert len(rows) > 0, "daily_metrics CSV is empty"
        assert len(rows) >= 600, f"Expected ~634 rows, got {len(rows)}"

    def test_non_empty_files(self):
        for filename in EXPECTED_CSV_FILES:
            path = os.path.join(FEISHU_DIR, filename)
            assert os.path.getsize(path) > 0, f"Empty CSV: {path}"

    def test_daily_metrics_first_row_has_data(self):
        path = os.path.join(FEISHU_DIR, 'daily_metrics_for_feishu.csv')
        with open(path, 'r', encoding='utf-8') as f:
            reader = csv.reader(f)
            next(reader)  # skip header
            first_row = next(reader)
        assert len(first_row) > 5, f"First row has too few columns: {first_row}"
