import os, sys, json

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), '..'))
import config
from adapters.base import ChannelAdapter
from feishu_client import FeishuClient


class FeishuAdapter(ChannelAdapter):
    def __init__(self, dry_run=False, chat_id=None):
        self._dry_run = dry_run
        self._feishu_client = None
        self._chat_id = chat_id

    def _get_client(self):
        if self._feishu_client is None:
            creds = config.load_feishu_credentials()
            self._feishu_client = FeishuClient(
                app_id=creds["app_id"],
                app_secret=creds["app_secret"],
                app_token=creds["app_token"],
                dry_run=self._dry_run,
            )
        return self._feishu_client

    def _get_chat_id(self):
        if self._chat_id:
            return self._chat_id
        return config.load_feishu_credentials()["chat_id"]

    @staticmethod
    def _parse_payload(payload_json):
        try:
            return json.loads(payload_json)
        except (json.JSONDecodeError, TypeError):
            return None

    @staticmethod
    def _format_message(payload):
        lines = []
        rule_id = payload.get("rule_id", "unknown")
        metric = payload.get("metric_name", "unknown")
        lines.append(f"[Alert] {rule_id}: {metric}")
        current = payload.get("current_value")
        baseline = payload.get("baseline_value")
        if current is not None:
            lines.append(f"Current: {current}")
        if baseline is not None:
            lines.append(f"Baseline: {baseline}")
        severity = payload.get("severity", "")
        if severity:
            lines.append(f"Severity: {severity}")
        owner = payload.get("owner_role", "")
        if owner:
            lines.append(f"Owner: {owner}")
        return "\n".join(lines)

    def dry_run(self, event: dict) -> dict:
        payload = self._parse_payload(event.get("payload_json"))
        if payload is None:
            return {"status": "preview", "message": "[Alert] (invalid payload)",
                    "payload": None, "external_ref": None, "error": None}
        message = self._format_message(payload)
        return {"status": "preview", "message": message,
                "payload": payload, "external_ref": None, "error": None}

    def dispatch(self, event: dict) -> dict:
        payload = self._parse_payload(event.get("payload_json"))
        if payload is None:
            msg = "payload_json is NULL" if event.get("payload_json") is None else "invalid JSON in payload"
            return {"status": "failed", "external_ref": None, "error": msg, "message": None}

        chat_id = self._get_chat_id()
        if not chat_id:
            return {"status": "failed", "external_ref": None,
                    "error": "no chat_id configured (set FEISHU_CHAT_ID in .env or feishu_app.yml)",
                    "message": None}

        message = self._format_message(payload)
        client = self._get_client()
        result = client.send_message(chat_id, message, dry_run=self._dry_run)

        if result:
            return {"status": "dispatched", "external_ref": result, "error": None, "message": message}
        return {"status": "failed", "external_ref": None,
                "error": "Failed to send message via FeishuClient", "message": message}
