import argparse
import csv
import json
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
    parser.add_argument("--json", action="store_true", help="Output JSON summary to stdout")
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
    table_pull_counts = {}

    for table_name in PULL_TABLES:
        table_info = table_config.get("tables", {}).get(table_name, {})
        table_id = table_info.get("table_id")

        if not table_id:
            print(f"WARNING: No table_id configured for {table_name}, skipping")
            table_pull_counts[table_name] = 0
            continue

        if dry_run:
            print(f"dry-run: Would pull records from table {table_name} ({table_id})")
            table_pull_counts[table_name] = 0
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
        table_pull_counts[table_name] = len(records)

        for rec in records:
            fields = rec.get("fields", {})
            fields["_record_id"] = (
                fields.get("task_id") or fields.get("review_id") or rec.get("record_id", "")
            )
            fields["_table"] = table_name
            fields["_pulled_at"] = datetime.now(timezone.utc).isoformat()
            all_local_records.append(fields)

    if all_local_records:
        output_path = os.path.join(
            os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
            "data", "ops", "action_task_status_snapshot.csv",
        )
        os.makedirs(os.path.dirname(output_path), exist_ok=True)
        with open(output_path, "w", newline="", encoding="utf-8") as f:
            all_keys = set()
            for rec in all_local_records:
                all_keys.update(rec.keys())
            fieldnames = sorted(all_keys)
            writer = csv.DictWriter(f, fieldnames=fieldnames)
            writer.writeheader()
            writer.writerows(all_local_records)

        print(f"Wrote {len(all_local_records)} records to {output_path}")

    if args.json:
        total_pulled = sum(table_pull_counts.values())
        print(json.dumps({
            "status": "success", "total_pulled": total_pulled,
            "tables": table_pull_counts, "written": len(all_local_records),
        }))


if __name__ == "__main__":
    main()
