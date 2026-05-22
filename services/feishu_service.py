"""Feishu service layer wrapping FeishuClient for v0.5.1 API endpoints.
 
Provides export_tables, sync_to_feishu, and import_status_from_feishu
operations with universal dry_run support and graceful missing-config handling.
"""

import logging
import os
import sys
import subprocess
import datetime
from typing import Optional

import yaml

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))
from scripts import config

logger = logging.getLogger(__name__)

PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
SCRIPTS_DIR = os.path.join(PROJECT_ROOT, "scripts")


class FeishuService:

    def __init__(self, dry_run: bool = True):
        self._dry_run = dry_run
        self._config = None

    def _load_config(self) -> dict:
        if self._config is not None:
            return self._config

        cfg = config.load_feishu_credentials()

        table_ids_file = config.FEISHU_TABLE_IDS_FILE
        if os.path.exists(table_ids_file):
            try:
                with open(table_ids_file) as f:
                    yml = yaml.safe_load(f) or {}
                cfg["table_ids"] = yml.get("tables", {})
            except Exception as e:
                logger.warning("Failed to load Feishu table IDs config from %s: %s", table_ids_file, e)
                cfg["table_ids"] = {}
        else:
            cfg["table_ids"] = {}

        self._config = cfg
        return cfg

    def _is_configured(self) -> bool:
        cfg = self._load_config()
        return bool(cfg["app_id"] and cfg["app_secret"])

    def _get_table_names(self, table_names=None):
        cfg = self._load_config()
        available = list(cfg.get("table_ids", {}).keys()) or [
            "daily_metrics",
            "alert_events",
            "strategy_recommendations",
            "action_tasks",
            "review_retro",
        ]
        if table_names:
            matched = [t for t in table_names if t in available]
            unknown = [t for t in table_names if t not in available]
            if unknown:
                raise ValueError(
                    f"Unknown table names: {', '.join(unknown)}. "
                    f"Available: {', '.join(available)}"
                )
            return matched
        return available

    def _run_script(self, script_name, args, timeout=120):
        script_path = os.path.join(SCRIPTS_DIR, script_name)
        cmd = [sys.executable, script_path] + args
        result = subprocess.run(
            cmd, capture_output=True, text=True, timeout=timeout,
            cwd=PROJECT_ROOT,
        )
        if result.returncode != 0:
            stderr = result.stderr.strip()[-300:] if result.stderr else ""
            raise RuntimeError(
                f"{script_name} exited {result.returncode}: {stderr}"
            )
        return result.stdout

    def export_tables(self, table_names: Optional[list] = None) -> dict:
        if not self._is_configured():
            return {"status": "not_configured", "message": "Feishu credentials not configured", "tables": []}

        resolved = self._get_table_names(table_names)

        if self._dry_run:
            return {
                "status": "preview",
                "message": "Dry-run: no files written",
                "tables": [
                    {"name": t, "rows": 0, "file": "", "status": "preview"}
                    for t in resolved
                ],
            }

        try:
            self._run_script("db_export_feishu.py", ["--all"])
            tables = []
            for name in resolved:
                csv_path = os.path.join(config.FEISHU_DIR, f"{name}_for_feishu.csv")
                rows = 0
                if os.path.exists(csv_path):
                    with open(csv_path) as f:
                        rows = sum(1 for __ in f) - 1
                tables.append({"name": name, "rows": max(rows, 0), "file": csv_path, "status": "exported"})
            all_available = self._get_table_names()
            if table_names and set(resolved) != set(all_available):
                return {"status": "exported", "tables": tables,
                        "note": f"Requested {len(resolved)} table(s); export script processed all {len(all_available)} tables"}
            return {"status": "exported", "tables": tables}
        except Exception as e:
            return {"status": "failed", "message": str(e), "tables": []}

    def sync_to_feishu(self, table_names: Optional[list] = None) -> dict:
        if not self._is_configured():
            return {"status": "not_configured", "message": "Feishu credentials not configured", "tables": []}

        resolved = self._get_table_names(table_names)

        if self._dry_run:
            return {
                "status": "preview",
                "message": "Dry-run: no Feishu API calls",
                "tables": [
                    {"name": t, "created": 0, "updated": 0, "status": "preview"}
                    for t in resolved
                ],
            }

        try:
            all_available = self._get_table_names()
            if set(resolved) == set(all_available):
                self._run_script("sync_feishu_bitable.py", ["--all", "--apply"])
            else:
                for name in resolved:
                    self._run_script("sync_feishu_bitable.py", ["--table", name, "--apply"])
            return {"status": "synced", "tables": [
                {"name": t, "created": 0, "updated": 0, "status": "synced"}
                for t in resolved
            ]}
        except Exception as e:
            return {"status": "failed", "message": str(e), "tables": []}

    def import_status_from_feishu(self, table_names: Optional[list] = None) -> dict:
        if not self._is_configured():
            return {"status": "not_configured", "message": "Feishu credentials not configured", "tables": []}

        resolved = self._get_table_names(table_names)

        if self._dry_run:
            return {
                "status": "preview",
                "message": "Dry-run: no Feishu API calls",
                "tables": [
                    {"name": t, "pulled": 0, "imported": 0, "skipped": 0, "status": "preview"}
                    for t in resolved
                ],
            }

        try:
            self._run_script("pull_feishu_status.py", ["--apply"])
            self._run_script("db_import_feishu_status.py", ["--apply"])
            return {"status": "imported", "tables": [
                {"name": t, "pulled": 0, "imported": 0, "skipped": 0, "status": "imported"}
                for t in resolved
            ]}
        except Exception as e:
            return {"status": "failed", "message": str(e), "tables": []}
