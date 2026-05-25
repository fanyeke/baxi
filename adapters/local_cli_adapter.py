import csv
import datetime
import json
import os
import re

from adapters.base import ChannelAdapter
from core import config


class LocalCLIAdapter(ChannelAdapter):
    def __init__(self, dry_run=False):
        self._dry_run = dry_run
        self._log_path = os.path.join(config.SYSTEM_DIR, "local_cli_dispatch_log.csv")

    @staticmethod
    def _parse_payload(payload_json):
        try:
            return json.loads(payload_json)
        except (json.JSONDecodeError, TypeError):
            return {}

    _RULE_ID_RE = re.compile(r'^[a-zA-Z0-9_-]+$')

    @classmethod
    def _validate_rule_id(cls, rule_id: str) -> str:
        if not rule_id or not isinstance(rule_id, str):
            return "unknown"
        if len(rule_id) > 64:
            return "unknown"
        if not cls._RULE_ID_RE.match(rule_id):
            return "unknown"
        return rule_id

    def dry_run(self, event: dict) -> dict:
        payload = self._parse_payload(event.get("payload_json"))
        rule_id = self._validate_rule_id(payload.get("rule_id", "unknown"))
        command = f"python3 scripts/run_alert_detection.py --rule {rule_id} --investigate"
        return {"status": "preview", "message": command, "payload": payload,
                "external_ref": None, "error": None}

    def dispatch(self, event: dict) -> dict:
        payload = self._parse_payload(event.get("payload_json"))
        rule_id = self._validate_rule_id(payload.get("rule_id", "unknown"))
        command = f"python3 scripts/run_alert_detection.py --rule {rule_id} --investigate"

        os.makedirs(os.path.dirname(self._log_path), exist_ok=True)
        now = datetime.datetime.now().isoformat()
        write_header = not os.path.exists(self._log_path)

        with open(self._log_path, "a", newline="") as f:
            writer = csv.DictWriter(f, fieldnames=["timestamp", "outbox_id", "command", "rule_id", "status"])
            if write_header:
                writer.writeheader()
            writer.writerow({
                "timestamp": now,
                "outbox_id": event.get("outbox_id", ""),
                "command": command,
                "rule_id": rule_id,
                "status": "dispatched",
            })

        return {"status": "dispatched", "external_ref": self._log_path, "error": None, "message": command}
