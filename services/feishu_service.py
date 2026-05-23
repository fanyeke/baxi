"""Feishu service layer wrapping FeishuClient for v0.5.2 API endpoints.
 
Provides export_tables, sync_to_feishu, and import_status_from_feishu
operations with universal dry_run support and graceful missing-config handling.
v0.5.2: Added --json flag parsing for real result counts.
"""

import json
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

    def _run_script(self, script_name, args, timeout=120, parse_json=False):
        script_path = os.path.join(SCRIPTS_DIR, script_name)
        if parse_json:
            args = args + ["--json"]
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
        if parse_json:
            for line in reversed(result.stdout.strip().split("\n")):
                line = line.strip()
                if not line:
                    continue
                try:
                    return json.loads(line)
                except json.JSONDecodeError:
                    continue
            return None
        return result.stdout

    def _run_script_json(self, script_name, args, timeout=120):
        output = self._run_script(script_name, args + ["--json"], timeout=timeout)
        for line in reversed(output.strip().splitlines()):
            try:
                return json.loads(line)
            except json.JSONDecodeError:
                continue
        try:
            return json.loads(output.strip())
        except json.JSONDecodeError:
            return None

    @staticmethod
    def _sync_counts(output, table_name):
        if output and output.get("tables"):
            for t in output["tables"]:
                if t.get("table") == table_name:
                    return {"created": t.get("created", 0), "updated": t.get("updated", 0)}
        if output:
            n = max(len(output.get("tables", [{}])), 1)
            return {
                "created": output.get("created", 0) // n,
                "updated": output.get("updated", 0) // n,
            }
        return {"created": 0, "updated": 0}

    @staticmethod
    def _import_counts(pull_output, import_output, table_name):
        pulled = 0
        if pull_output and pull_output.get("tables"):
            pulled = pull_output["tables"].get(table_name, 0)
        total_imported = 0
        total_skipped = 0
        if import_output:
            total_imported = import_output.get("applied", 0)
            total_skipped = import_output.get("skipped", 0)
        n = max(len(["action_tasks", "review_retro"]), 1)
        return {
            "pulled": pulled,
            "imported": total_imported // n,
            "skipped": total_skipped // n,
        }

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
            all_available = self._get_table_names()
            if set(resolved) == set(all_available):
                export_args = ["--all"]
            else:
                export_args = [a for t in resolved for a in ("--table", t)]
            result = self._run_script("db_export_feishu.py", export_args, parse_json=True)
            if result and "tables" in result:
                tables_result = {t["table"]: t["rows"] for t in result.get("tables", [])}
            else:
                tables_result = {}
            tables = []
            for name in resolved:
                csv_path = os.path.join(config.FEISHU_DIR, f"{name}_for_feishu.csv")
                rows = tables_result.get(name, 0)
                tables.append({"name": name, "rows": rows, "file": csv_path, "status": "exported"})
            return {"status": "exported", "tables": tables}
        except Exception as e:
            logger.exception("Feishu export failed")
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
                result = self._run_script("sync_feishu_bitable.py", ["--all", "--apply"], parse_json=True)
            else:
                result = self._run_script("sync_feishu_bitable.py",
                    ["--apply"] + [a for t in resolved for a in ("--table", t)], parse_json=True)

            tables_counts = {}
            if result and "tables" in result:
                tables_counts = {t["table"]: t for t in result.get("tables", [])}

            return {"status": "synced", "tables": [
                {"name": t, "created": tables_counts.get(t, {}).get("created", 0),
                 "updated": tables_counts.get(t, {}).get("updated", 0), "status": "synced"}
                for t in resolved
            ]}
        except Exception as e:
            logger.exception("Feishu sync failed")
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
            pull_result = self._run_script("pull_feishu_status.py", ["--apply"], parse_json=True)
            import_result = self._run_script("db_import_feishu_status.py", ["--apply"], parse_json=True)

            pull_counts = pull_result.get("tables", {}) if pull_result else {}
            import_tables = {}
            if import_result:
                import_tables = import_result.get("tables", {})

            return {"status": "imported", "tables": [
                {"name": t,
                 "pulled": pull_counts.get(t, 0) if isinstance(pull_counts, dict) else 0,
                 "imported": import_tables.get(t, {}).get("imported", 0) if isinstance(import_tables, dict) else 0,
                 "skipped": import_tables.get(t, {}).get("skipped", 0) if isinstance(import_tables, dict) else 0,
                 "status": "imported"}
                for t in resolved
            ]}
        except Exception as e:
            logger.exception("Feishu status import failed")
            return {"status": "failed", "message": str(e), "tables": []}
