import argparse
import json
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from scripts.config import ensure_dirs_exist
from scripts.feishu_client import FeishuClient


def main():
    parser = argparse.ArgumentParser(description="Send Feishu group message")
    parser.add_argument("--dry-run", action="store_true", help="Preview without API calls")
    parser.add_argument("--apply", action="store_true", help="Execute real send")
    parser.add_argument("--app-id", default=os.environ.get("FEISHU_APP_ID", ""))
    parser.add_argument("--app-secret", default=os.environ.get("FEISHU_APP_SECRET", ""))
    parser.add_argument("--app-token", default=os.environ.get("FEISHU_BASE_APP_TOKEN", ""))
    parser.add_argument(
        "--message-path",
        default=os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "outputs", "wake", "feishu_message.json"),
        help="Path to feishu_message.json",
    )
    parser.add_argument("--chat-id", default=os.environ.get("FEISHU_CHAT_ID", ""), help="Feishu chat_id")
    args = parser.parse_args()
    ensure_dirs_exist()

    dry_run = args.dry_run or not args.apply
    message_path = args.message_path

    if not os.path.exists(message_path):
        print(f"ERROR: Message file not found: {message_path}")
        sys.exit(1)

    with open(message_path, "r", encoding="utf-8") as f:
        message = json.load(f)

    content = message.get("content", "")
    chat_id = args.chat_id

    if not chat_id:
        print("WARNING: No chat_id provided (--chat-id or FEISHU_CHAT_ID)")
        sys.exit(0)

    client = FeishuClient(
        app_id=args.app_id,
        app_secret=args.app_secret,
        app_token=args.app_token,
        dry_run=dry_run,
    )

    if dry_run:
        print(f"dry-run: Will send message to chat_id: {chat_id}")
        print(f"Content preview: {content[:200]}...")
    else:
        try:
            msg_id = client.send_group_message(chat_id, content)
            if msg_id:
                print(f"Sent message: {msg_id}")
            else:
                print("WARNING: Message send returned no ID")
        except Exception as e:
            print(f"WARNING: Failed to send message (non-critical): {e}")
            sys.exit(0)


if __name__ == "__main__":
    main()
