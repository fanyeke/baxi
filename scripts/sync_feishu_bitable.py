from typing import Tuple
import argparse
import csv
import logging
import os
import sys
import uuid
from datetime import datetime, timezone

import pandas as pd
import yaml

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from scripts.config import (
    FEISHU_DIR,
    FEISHU_BASE_SCHEMA_FILE,
    FEISHU_FIELD_MAPPING_FILE,
    FEISHU_USER_MAPPING_FILE,
    SYSTEM_DIR,
    ensure_dirs_exist,
)
from scripts.feishu_client import FeishuClient

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
)
logger = logging.getLogger(__name__)

VALID_TABLES = ["daily_metrics", "alert_events", "strategy_recommendations", "action_tasks", "review_retro"]


def load_yaml(path: str):
    with open(path, "r", encoding="utf-8") as f:
        return yaml.safe_load(f)


def get_primary_key(table_id: str) -> str:
    mapping = {
        "daily_metrics": "simulated_date",
        "alert_events": "alert_id",
        "strategy_recommendations": "recommendation_id",
        "action_tasks": "task_id",
        "review_retro": "review_id",
    }
    return mapping.get(table_id, "record_id")


def load_csv_for_table(table_id: str) -> pd.DataFrame:
    csv_name = f"{table_id}_for_feishu.csv"
    path = os.path.join(FEISHU_DIR, csv_name)

    if not os.path.exists(path):
        logger.warning("CSV not found: %s", path)
        return pd.DataFrame()

    df = pd.read_csv(path)
    if df.empty:
        logger.info("CSV is empty (headers only): %s", path)
        return df

    cols = list(df.columns)
    dupes = [c for c in cols if cols.count(c) > 1]
    if dupes:
        logger.warning("Duplicate columns in %s: %s — deduplicating", csv_name, dupes)
        df = df.loc[:, ~df.columns.duplicated()]

    return df


# Fields that are Feishu user-type but may contain role strings in local CSV
USER_TYPE_FIELDS = {"owner_role", "owner"}

# Fields that should be percentage (0-1 range) in the Feishu schema
PERCENTAGE_FIELDS = {"low_review_rate", "late_delivery_rate", "cancel_rate"}


def load_user_mapping():
    path = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "config", "feishu_user_mapping.yml")
    if os.path.exists(path):
        return load_yaml(path)
    return {}


def records_to_dicts(df: pd.DataFrame, table_id: str) -> list[dict]:
    user_mapping = load_user_mapping()
    records = []
    for _, row in df.iterrows():
        rec = {}
        for col in df.columns:
            val = row[col]
            if pd.isna(val):
                val = ""

            # Handle user-type fields: role strings -> skip, valid user_id -> keep
            if col in USER_TYPE_FIELDS and val and not str(val).startswith("ou_"):
                # Write to 负责人角色 text field if available, skip user field
                role_col = "owner_role"
                if "owner_role" in df.columns and col == "owner_role":
                    pass  # owner_role written as text
                else:
                    continue  # skip unknown user field

            # Handle percentage fields: ensure 0-1 range
            if col in PERCENTAGE_FIELDS and isinstance(val, (int, float)) and val > 1:
                val = val / 100.0

            if isinstance(val, (int, float)) and not pd.isna(val):
                rec[col] = val
            else:
                rec[col] = str(val) if val != "" else ""

        # Map owner_role to 负责人角色 if present
        if "owner_role" in rec and rec["owner_role"]:
            role_val = rec["owner_role"]
            mapping = user_mapping.get(role_val, {})
            if mapping.get("role_name"):
                rec["owner_role"] = mapping["role_name"]
            feishu_user_id = mapping.get("feishu_user_id", "")
            if "负责人" not in rec and feishu_user_id:
                rec["负责人"] = feishu_user_id

        records.append(rec)
    return records


def write_sync_log(
    run_id: str,
    table_id: str,
    source_file: str,
    create_count: int,
    update_count: int,
    skip_count: int,
    status: str,
    error_message: str,
    simulated_date: str,
):
    log_path = os.path.join(SYSTEM_DIR, "feishu_sync_log.csv")
    file_exists = os.path.exists(log_path)

    now = datetime.now(timezone.utc).isoformat()

    with open(log_path, "a", newline="", encoding="utf-8") as f:
        writer = csv.writer(f)
        if not file_exists:
            writer.writerow([
                "sync_id", "run_id", "real_run_date", "simulated_date",
                "target_table", "source_file", "create_count", "update_count",
                "skip_count", "fail_count", "status", "error_message",
                "started_at", "finished_at",
            ])
        writer.writerow([
            str(uuid.uuid4().hex[:12]),
            run_id,
            now,
            simulated_date,
            table_id,
            source_file,
            create_count,
            update_count,
            skip_count,
            0,
            status,
            error_message,
            now,
            now,
        ])


def sync_table(
    client: FeishuClient,
    table_id: str,
    dry_run: bool,
    run_id: str,
) -> Tuple[int, int, int]:
    pk = get_primary_key(table_id)
    df = load_csv_for_table(table_id)

    if df.empty:
        logger.info("No data to sync for %s (0 records)", table_id)
        write_sync_log(
            run_id, table_id, "", 0, 0, 0, "skipped",
            "No data", "",
        )
        return 0, 0, 0

    records = records_to_dicts(df, table_id)
    source_file = f"{table_id}_for_feishu.csv"

    if dry_run:
        logger.info("dry-run: %s — %d records [%d create, 0 update, 0 skip]", table_id, len(records), len(records))
        write_sync_log(
            run_id, table_id, source_file, len(records), 0, 0, "dry-run",
            "", str(df[pk].iloc[0]) if pk in df.columns else "",
        )
        return len(records), 0, 0

    created, updated = client.upsert_by_key(table_id, records, pk)
    create_count = len(created)
    update_count = len(updated)
    skip_count = 0

    logger.info(
        "synced: %s — %d records [%d create, %d update, %d skip]",
        table_id, len(records), create_count, update_count, skip_count,
    )

    simulated_date = ""
    if pk in df.columns and len(df) > 0:
        simulated_date = str(df[pk].iloc[0])

    write_sync_log(
        run_id, table_id, source_file, create_count, update_count,
        skip_count, "success", "", simulated_date,
    )

    return create_count, update_count, skip_count


def main():
    parser = argparse.ArgumentParser(description="Sync Feishu Bitable tables")
    parser.add_argument(
        "--table",
        choices=VALID_TABLES,
        help="Table to sync (or use --all)",
    )
    parser.add_argument(
        "--all",
        action="store_true",
        help="Sync all tables",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Preview without actual API calls",
    )
    parser.add_argument(
        "--apply",
        action="store_true",
        help="Execute real sync (overrides --dry-run)",
    )
    parser.add_argument(
        "--app-id",
        default=os.environ.get("FEISHU_APP_ID", ""),
        help="Feishu app ID (or set FEISHU_APP_ID env var)",
    )
    parser.add_argument(
        "--app-secret",
        default=os.environ.get("FEISHU_APP_SECRET", ""),
        help="Feishu app secret (or set FEISHU_APP_SECRET env var)",
    )
    parser.add_argument(
        "--app-token",
        default=os.environ.get("FEISHU_BASE_APP_TOKEN", ""),
        help="Feishu base app_token (or set FEISHU_BASE_APP_TOKEN env var)",
    )

    args = parser.parse_args()
    ensure_dirs_exist()

    if not args.table and not args.all:
        parser.error("Either --table <name> or --all is required")

    dry_run = args.dry_run or not args.apply
    run_id = uuid.uuid4().hex

    schema = load_yaml(FEISHU_BASE_SCHEMA_FILE)
    field_mapping = load_yaml(FEISHU_FIELD_MAPPING_FILE)

    client = FeishuClient(
        app_id=args.app_id,
        app_secret=args.app_secret,
        app_token=args.app_token,
        dry_run=dry_run,
    )

    tables_to_sync = VALID_TABLES if args.all else [args.table]

    logger.info(
        "Starting Feishu sync (run_id=%s, %s, %d tables)",
        run_id,
        "DRY RUN" if dry_run else "APPLY",
        len(tables_to_sync),
    )

    for table_id in tables_to_sync:
        try:
            sync_table(client, table_id, dry_run, run_id)
        except Exception as e:
            logger.error("Failed to sync %s: %s", table_id, e)
            write_sync_log(
                run_id, table_id, "", 0, 0, 0, "failed",
                str(e), "",
            )

    logger.info("Feishu sync completed for run_id=%s", run_id)


if __name__ == "__main__":
    main()
