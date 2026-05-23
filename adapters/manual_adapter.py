import os, sys, json

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), '..'))
from adapters.base import ChannelAdapter


class ManualAdapter(ChannelAdapter):
    def __init__(self, dry_run=False):
        self._dry_run = dry_run

    @staticmethod
    def _parse_payload(payload_json):
        try:
            return json.loads(payload_json)
        except (json.JSONDecodeError, TypeError):
            return {}

    def dry_run(self, event: dict) -> dict:
        payload = self._parse_payload(event.get("payload_json"))
        rule_id = payload.get("rule_id", "unknown")
        msg = f"Event queued for manual review: rule={rule_id}"
        return {"status": "preview", "message": msg, "payload": payload,
                "external_ref": None, "error": None}

    def dispatch(self, event: dict) -> dict:
        payload = self._parse_payload(event.get("payload_json"))
        rule_id = payload.get("rule_id", "unknown")
        msg = f"Event queued for manual review: rule={rule_id}"
        return {"status": "skipped", "external_ref": None, "error": None, "message": msg}
