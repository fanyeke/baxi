import argparse
import logging
import os
import sys

import yaml

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from scripts.config import FEISHU_TABLE_IDS_FILE, ensure_dirs_exist
from scripts.feishu_client import FeishuClient

logger = logging.getLogger(__name__)
logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")


def main():
    parser = argparse.ArgumentParser(description="Verify Feishu table IDs and connectivity")
    parser.add_argument("--dry-run", action="store_true", help="Preview without API calls")
    parser.add_argument("--app-id", default=os.environ.get("FEISHU_APP_ID", ""))
    parser.add_argument("--app-secret", default=os.environ.get("FEISHU_APP_SECRET", ""))
    parser.add_argument("--app-token", default=os.environ.get("FEISHU_BASE_APP_TOKEN", ""))
    parser.add_argument(
        "--table-ids-file",
        default=FEISHU_TABLE_IDS_FILE,
        help="Path to feishu_table_ids.yml",
    )
    args = parser.parse_args()
    ensure_dirs_exist()

    dry_run = args.dry_run
    if not os.path.exists(args.table_ids_file):
        print(f"ERROR: Table IDs file not found: {args.table_ids_file}")
        sys.exit(1)

    with open(args.table_ids_file, "r", encoding="utf-8") as f:
        config = yaml.safe_load(f)

    tables = config.get("tables", {})
    base_app_token = config.get("base", {}).get("app_token", "")

    print(f"Verifying {len(tables)} tables...")
    print(f"  Base app_token: {base_app_token[:8]}..." if base_app_token else "  Base app_token: NOT SET")
    print()

    all_ok = True
    for table_name, table_info in tables.items():
        table_id = table_info.get("table_id", "")

        if dry_run:
            print(f"  {table_name}: {table_id} [DRY RUN - would verify connectivity]")
            continue

        if table_id.startswith("YOUR_"):
            print(f"  {table_name}: {table_id} [SKIP - placeholder, not configured yet]")
            all_ok = False
            continue

        # Auto-enable dry-run if credentials are placeholders
        effective_dry = dry_run or args.app_id == "" or args.app_id == "YOUR_APP_ID"

        client = FeishuClient(
            app_id=args.app_id,
            app_secret=args.app_secret,
            app_token=base_app_token or args.app_token,
            dry_run=effective_dry,
        )

        try:
            records, _ = client.list_records(table_id, page_size=1)
            count = len(records)
            print(f"  {table_name}: {table_id} [OK - {count} records found]")
        except Exception as e:
            print(f"  {table_name}: {table_id} [FAIL - {e}]")
            all_ok = False

    print()
    if all_ok:
        print("All tables verified successfully.")
    else:
        print("Some tables failed verification. Check config and try again.")
        sys.exit(1)


if __name__ == "__main__":
    main()
