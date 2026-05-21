import argparse
import csv
import os
import sys
from datetime import datetime, timezone

import yaml

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from scripts.config import ensure_dirs_exist
from scripts.feishu_client import FeishuClient

PULL_TABLES = ["action_tasks", "review_retro"]


def main():
    parser = argparse.ArgumentParser(description="Pull status from Feishu back to local")
    parser.add_argument("--dry-run", action="store_true", help="Preview without API calls")
    parser.add_argument("--apply", action="store_true", help="Execute real pull")
    parser.add_argument("--app-id", default=os.environ.get("FEISHU_APP_ID", ""))
    parser.add_argument("--app-secret", default=os.environ.get("FEISHU_APP_SECRET", ""))
    parser.add_argument("--app-token", default=os.environ.get("FEISHU_BASE_APP_TOKEN", ""))
    parser.add_argument(
        "--table-ids",
        default=os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "config", "feishu_table_ids.yml"),
        help="Path to feishu_table_ids.yml",
    )
    parser.add_argument("--since", default=None, help="Only pull records updated since YYYY-MM-DD")
    args = parser.parse_args()
    ensure_dirs_exist()

    dry_run = args.dry_run or not args.apply
    table_ids_path = args.table_ids

    if not os.path.exists(table_ids_path):
        print(f"ERROR: Table IDs config not found: {table_ids_path}")
        sys.exit(1)

    with open(table_ids_path, "r", encoding="utf-8") as f:
        table_config = yaml.safe_load(f)

    client = FeishuClient(
        app_id=args.app_id,
        app_secret=args.app_secret,
        app_token=args.app_token,
        dry_run=dry_run,
    )

    all_local_records = []

    for table_name in PULL_TABLES:
        table_info = table_config.get("tables", {}).get(table_name, {})
        table_id = table_info.get("table_id")

        if not table_id:
            print(f"WARNING: No table_id configured for {table_name}, skipping")
            continue

        if dry_run:
            print(f"dry-run: Would pull records from table {table_name} ({table_id})")
            continue

        filter_config = None
        if args.since:
            filter_config = {
                "conditions": [
                    {
                        "field_name": "updated_at",
                        "operator": "greater",
                        "value": args.since,
                    }
                ],
                "conjunction": "and",
            }

        records, _ = client.list_records(table_id, page_size=500, filter_config=filter_config)
        print(f"Pulled {len(records)} records from {table_name}")

        for rec in records:
            fields = rec.get("fields", {})
            fields["_record_id"] = rec.get("record_id", "")
            fields["_table"] = table_name
            fields["_pulled_at"] = datetime.now(timezone.utc).isoformat()
            all_local_records.append(fields)

    if all_local_records:
        output_path = os.path.join(
            os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
            "data", "ops", "action_task_status_snapshot.csv",
        )
        with open(output_path, "w", newline="", encoding="utf-8") as f:
            all_keys = set()
            for rec in all_local_records:
                all_keys.update(rec.keys())
            fieldnames = sorted(all_keys)
            writer = csv.DictWriter(f, fieldnames=fieldnames)
            writer.writeheader()
            writer.writerows(all_local_records)

        print(f"Wrote {len(all_local_records)} records to {output_path}")


if __name__ == "__main__":
    main()
