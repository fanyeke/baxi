"""Tests for FeishuService table filter propagation.

TDD tests verifying that table_names parameter is properly propagated
to _run_script in export_tables, sync_to_feishu, and import_status_from_feishu.
"""

import json
from unittest.mock import patch, MagicMock

import pytest

from services.feishu_service import FeishuService


@pytest.fixture
def service():
    """Create a FeishuService with dry_run=False for testing actual calls."""
    with patch.object(FeishuService, '_load_config', return_value={
        "app_id": "fake_app_id",
        "app_secret": "fake_app_secret",
        "table_ids": {
            "daily_metrics": "table_1",
            "alert_events": "table_2",
            "strategy_recommendations": "table_3",
            "action_tasks": "table_4",
            "review_retro": "table_5",
            "orders": "table_6",
            "products": "table_7",
        },
    }):
        svc = FeishuService(dry_run=False)
        yield svc


class TestExportTablesFilter:
    """Test that export_tables respects table_names filter."""

    def test_export_tables_respects_filter(self, service):
        """When table_names is provided, _run_script should receive --table args, not --all."""
        with patch.object(service, '_run_script', return_value={"tables": []}) as mock_run:
            result = service.export_tables(table_names=["orders", "products"])

            assert result["status"] == "exported"
            mock_run.assert_called_once()
            call_args = mock_run.call_args
            args_passed = call_args[0][1]
            assert "--all" not in args_passed, (
                f"export_tables should not pass --all when filtering tables, got: {args_passed}"
            )
            assert "--table" in args_passed
            assert "orders" in args_passed
            assert "products" in args_passed

    def test_export_tables_all_when_no_filter(self, service):
        """When table_names is empty/None, _run_script should receive --all."""
        with patch.object(service, '_run_script', return_value={"tables": []}) as mock_run:
            result = service.export_tables(table_names=None)

            assert result["status"] == "exported"
            mock_run.assert_called_once()
            call_args = mock_run.call_args
            args_passed = call_args[0][1]
            assert "--all" in args_passed


class TestSyncToFeishuFilter:
    """Test that sync_to_feishu respects table_names filter."""

    def test_sync_to_feishu_respects_filter(self, service):
        """When syncing subset of tables, _run_script should receive --table for ALL resolved tables."""
        with patch.object(service, '_run_script', return_value={"tables": []}) as mock_run:
            result = service.sync_to_feishu(table_names=["orders", "products"])

            assert result["status"] == "synced"
            mock_run.assert_called_once()

            script_name = mock_run.call_args[0][0]
            args_passed = mock_run.call_args[0][1]

            assert script_name == "sync_feishu_bitable.py"
            assert "--all" not in args_passed
            assert "--apply" in args_passed
            table_indices = [i for i, a in enumerate(args_passed) if a == "--table"]
            assert len(table_indices) == 2, (
                f"Expected 2 --table args for 2 tables, got {len(table_indices)} in: {args_passed}"
            )
            assert "orders" in args_passed
            assert "products" in args_passed


class TestImportStatusPerTable:
    """Test that import_status_from_feishu returns per-table counts."""

    def test_import_status_per_table_counts(self, service):
        """When per-table data is returned, each table should have different counts."""
        per_table_result = {
            "tables": {
                "orders": {"imported": 5, "skipped": 2},
                "products": {"imported": 3, "skipped": 1},
            }
        }

        def mock_run_script(script_name, args, parse_json=False, timeout=120):
            if "pull" in script_name:
                return {"tables": {"orders": 10, "products": 6}}
            if "import" in script_name:
                return per_table_result
            return None

        with patch.object(service, '_run_script', side_effect=mock_run_script):
            result = service.import_status_from_feishu(table_names=["orders", "products"])

            assert result["status"] == "imported"
            tables = result["tables"]
            assert len(tables) == 2

            orders_result = next(t for t in tables if t["name"] == "orders")
            products_result = next(t for t in tables if t["name"] == "products")
            assert orders_result["imported"] == 5, (
                f"orders imported should be 5, got {orders_result['imported']}"
            )
            assert orders_result["skipped"] == 2, (
                f"orders skipped should be 2, got {orders_result['skipped']}"
            )
            assert products_result["imported"] == 3, (
                f"products imported should be 3, got {products_result['imported']}"
            )
            assert products_result["skipped"] == 1, (
                f"products skipped should be 1, got {products_result['skipped']}"
            )
            assert orders_result["imported"] != products_result["imported"]
