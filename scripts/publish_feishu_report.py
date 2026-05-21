import argparse
import csv
import os
import sys
import uuid
from datetime import datetime, timezone

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from scripts.config import FEISHU_DIR, ensure_dirs_exist
from scripts.feishu_client import FeishuClient


def main():
    parser = argparse.ArgumentParser(description="Publish daily report to Feishu doc")
    parser.add_argument("--dry-run", action="store_true", help="Preview without API calls")
    parser.add_argument("--apply", action="store_true", help="Execute real publish")
    parser.add_argument("--app-id", default=os.environ.get("FEISHU_APP_ID", ""))
    parser.add_argument("--app-secret", default=os.environ.get("FEISHU_APP_SECRET", ""))
    parser.add_argument("--app-token", default=os.environ.get("FEISHU_BASE_APP_TOKEN", ""))
    parser.add_argument(
        "--report-path",
        default=os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "outputs", "wake", "daily_report.md"),
        help="Path to daily report markdown file",
    )
    args = parser.parse_args()
    ensure_dirs_exist()

    dry_run = args.dry_run or not args.apply
    report_path = args.report_path

    if not os.path.exists(report_path):
        print(f"ERROR: Report file not found: {report_path}")
        sys.exit(1)

    with open(report_path, "r", encoding="utf-8") as f:
        content = f.read()

    date_str = datetime.now().strftime("%Y-%m-%d")
    title = f"Olist Daily Report - {date_str}"

    client = FeishuClient(
        app_id=args.app_id,
        app_secret=args.app_secret,
        app_token=args.app_token,
        dry_run=dry_run,
    )

    if dry_run:
        print(f"dry-run: Will create doc: '{title}' ({len(content)} chars)")
        doc_url = "https://dry-run-doc-url.feishu.cn/test"
    else:
        doc_url = client.create_doc(title, content)
        if doc_url:
            print(f"Created doc: {doc_url}")
        else:
            print("ERROR: Failed to create doc")
            sys.exit(1)

    log_path = os.path.join(FEISHU_DIR, "report_links.csv")
    file_exists = os.path.exists(log_path)
    with open(log_path, "a", newline="", encoding="utf-8") as f:
        writer = csv.writer(f)
        if not file_exists:
            writer.writerow(["run_id", "simulated_date", "report_title", "feishu_doc_url", "created_at"])
        writer.writerow([
            uuid.uuid4().hex[:12],
            date_str,
            title,
            doc_url or "",
            datetime.now(timezone.utc).isoformat(),
        ])


if __name__ == "__main__":
    main()
