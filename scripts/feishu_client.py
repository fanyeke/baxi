import json
import time
import logging
import requests
from typing import Any, Dict, List, Optional, Tuple

logger = logging.getLogger(__name__)


class FeishuClient:
    BASE_URL = "https://open.feishu.cn/open-apis"
    BATCH_LIMIT = 500
    MAX_RETRIES = 3
    DEFAULT_WAIT = [1, 2, 4]

    def __init__(
        self,
        app_id: str,
        app_secret: str,
        app_token: str,
        dry_run: bool = False,
    ):
        self.app_id = app_id
        self.app_secret = app_secret
        self.app_token = app_token
        self.dry_run = dry_run
        self._access_token: Optional[str] = None
        self._token_expires_at: float = 0

    def get_tenant_access_token(self) -> str:
        if self._access_token and time.time() < self._token_expires_at:
            return self._access_token

        if self.dry_run:
            logger.info("[dry-run] Would fetch tenant access token")
            self._access_token = "dry_run_token"
            self._token_expires_at = time.time() + 7200
            return self._access_token

        resp = self._raw_post(
            "/auth/v3/tenant_access_token/internal",
            {"app_id": self.app_id, "app_secret": self.app_secret},
            skip_auth=True,
        )
        token = resp["tenant_access_token"]
        expire = resp.get("expire", 7200)
        self._access_token = token
        self._token_expires_at = time.time() + expire - 60
        return token

    def list_records(
        self,
        table_id: str,
        page_size: int = 50,
        filter_config: Optional[Dict] = None,
    ) -> Tuple[List[Dict], str]:
        if self.dry_run:
            logger.info(
                "[dry-run] Would list records for table %s (page_size=%d)",
                table_id,
                page_size,
            )
            return [], ""

        params: Dict[str, Any] = {"page_size": min(page_size, 500)}
        if filter_config:
            params["filter"] = filter_config

        all_records: List[Dict] = []
        page_token = ""
        while True:
            if page_token:
                params["page_token"] = page_token

            resp = self._raw_get(
                f"/bitable/v1/apps/{self.app_token}/tables/{table_id}/records",
                params=params,
            )
            items = resp.get("items", [])
            all_records.extend(items)
            page_token = resp.get("page_token", "")
            has_more = resp.get("has_more", False)

            if not has_more:
                break

        return all_records, page_token

    def create_record(self, table_id: str, record_data: Dict) -> Optional[Dict]:
        if self.dry_run:
            logger.info("[dry-run] Would create record in table %s", table_id)
            return None
        payload = {"fields": record_data}
        result = self._raw_post(
            f"/bitable/v1/apps/{self.app_token}/tables/{table_id}/records",
            payload,
        )
        return result.get("record")

    def batch_create(self, table_id: str, records: List[Dict]) -> List[Dict]:
        all_created: List[Dict] = []
        for chunk in self._chunk_records(records):
            if self.dry_run:
                logger.info(
                    "[dry-run] Would batch create %d records in table %s",
                    len(chunk),
                    table_id,
                )
                all_created.extend({"record_id": f"dry_run_{i}", "fields": r} for i, r in enumerate(chunk))
                continue

            payload = {"records": [{"fields": r} for r in chunk]}
            resp = self._raw_post(
                f"/bitable/v1/apps/{self.app_token}/tables/{table_id}/records/batch_create",
                payload,
            )
            all_created.extend(resp.get("records", []))
            time.sleep(1)

        return all_created

    def update_record(self, table_id: str, record_id: str, record_data: Dict) -> Optional[Dict]:
        if self.dry_run:
            logger.info("[dry-run] Would update record %s in table %s", record_id, table_id)
            return {"record_id": record_id, "fields": record_data}
        payload = {"fields": record_data}
        result = self._raw_put(
            f"/bitable/v1/apps/{self.app_token}/tables/{table_id}/records/{record_id}",
            payload,
        )
        return result.get("record")

    def batch_update(self, table_id: str, records: List[Dict]) -> List[Dict]:
        all_updated: List[Dict] = []
        for chunk in self._chunk_records(records):
            if self.dry_run:
                logger.info(
                    "[dry-run] Would batch update %d records in table %s",
                    len(chunk),
                    table_id,
                )
                all_updated.extend({"record_id": r.get("record_id", "dry_run"), "fields": r.get("fields", r)} for r in chunk)
                continue

            payload = {"records": [{"record_id": r["record_id"], "fields": r.get("fields", r)} for r in chunk]}
            resp = self._raw_put(
                f"/bitable/v1/apps/{self.app_token}/tables/{table_id}/records/batch_update",
                payload,
            )
            all_updated.extend(resp.get("records", []))
            time.sleep(1)

        return all_updated

    def upsert_by_key(
        self,
        table_id: str,
        records: List[Dict],
        key_field: str,
    ) -> Tuple[List[Dict], List[Dict]]:
        created: List[Dict] = []
        updated: List[Dict] = []

        existing, _ = self.list_records(table_id, page_size=500)

        existing_map: Dict[str, Dict] = {}
        for rec in existing:
            fields = rec.get("fields", {})
            key_val = fields.get(key_field)
            if key_val is not None:
                existing_map[key_val] = rec

        to_create: List[Dict] = []
        for record in records:
            key_val = record.get(key_field)
            if key_val in existing_map:
                record_id = existing_map[key_val].get("record_id")
                updated_rec = self.update_record(table_id, record_id, record)
                if updated_rec:
                    updated.append(updated_rec)
            else:
                to_create.append(record)

        if to_create:
            created = self.batch_create(table_id, to_create)

        return created, updated

    def create_doc(self, title: str, content: str) -> Optional[str]:
        if self.dry_run:
            logger.info("[dry-run] Would create doc: %s", title)
            return "https://dry-run-doc-url.feishu.cn/test"

        resp = self._raw_post("/docx/v1/documents", {"title": title})
        doc_id = resp.get("document", {}).get("document_id")

        if doc_id:
            self._write_doc_content(doc_id, content)
            return f"https://open.feishu.cn/document/{doc_id}"
        return None

    def _write_doc_content(self, doc_id: str, content: str):
        body = {
            "children": [
                {
                    "block_type": 2,
                    "text": {
                        "elements": [{"text_run": {"text": content}}],
                    },
                }
            ]
        }
        self._raw_patch(
            f"/docx/v1/documents/{doc_id}/blocks/{doc_id}/children",
            body,
        )

    def send_message(
        self,
        chat_id: str,
        content: str,
        msg_type: str = "text",
        dry_run: Optional[bool] = None,
    ):
        is_dry = dry_run if dry_run is not None else self.dry_run

        if is_dry:
            logger.info("[dry-run] Would send message to chat_id: %s, content: %.40s", chat_id, content)
            return "dry_run_message_" + str(int(time.time()))

        payload = {"receive_id": chat_id, "msg_type": msg_type,
                    "content": json.dumps({"text": content})}
        try:
            resp = self._raw_post("/im/v1/messages", payload, params={"receive_id_type": "chat_id"})
            return resp.get("data", {}).get("message_id")
        except Exception as e:
            logger.error("Failed to send message to %s: %s", chat_id, e)
            return None

    def send_group_message(self, chat_id: str, content: str) -> Optional[str]:
        if self.dry_run:
            logger.info("[dry-run] Would send message to chat_id: %s", chat_id)
            return "dry_run_message_id"

        body = {
            "receive_id": chat_id,
            "msg_type": "text",
            "content": json.dumps({"text": content}),
        }
        resp = self._raw_post("/im/v1/messages", body, params={"receive_id_type": "chat_id"})
        return resp.get("data", {}).get("message_id")

    def _raw_get(self, path: str, params: Optional[Dict] = None, skip_auth: bool = False) -> Dict:
        url = self.BASE_URL + path
        headers = {"Authorization": "Bearer " + self.get_tenant_access_token()} if not skip_auth else {}
        for attempt, wait in enumerate(self.DEFAULT_WAIT):
            resp = requests.get(url, headers=headers, params=params, timeout=30)
            if self._check_response(resp, path, attempt):
                return resp.json()
            time.sleep(wait)
        raise RuntimeError(f"Failed to GET {path} after {self.MAX_RETRIES} retries")

    def _raw_post(self, path: str, json: Dict, params: Optional[Dict] = None, skip_auth: bool = False) -> Dict:
        url = self.BASE_URL + path
        headers = {"Authorization": "Bearer " + self.get_tenant_access_token()} if not skip_auth else {}
        for attempt, wait in enumerate(self.DEFAULT_WAIT):
            resp = requests.post(url, headers=headers, json=json, params=params, timeout=30)
            if self._check_response(resp, path, attempt):
                return resp.json()
            time.sleep(wait)
        raise RuntimeError(f"Failed to POST {path} after {self.MAX_RETRIES} retries")

    def _raw_put(self, path: str, json: Dict, skip_auth: bool = False) -> Dict:
        url = self.BASE_URL + path
        headers = {"Authorization": "Bearer " + self.get_tenant_access_token()} if not skip_auth else {}
        for attempt, wait in enumerate(self.DEFAULT_WAIT):
            resp = requests.put(url, headers=headers, json=json, timeout=30)
            if self._check_response(resp, path, attempt):
                return resp.json()
            time.sleep(wait)
        raise RuntimeError(f"Failed to PUT {path} after {self.MAX_RETRIES} retries")

    def _raw_patch(self, path: str, json: Dict, skip_auth: bool = False) -> Dict:
        url = self.BASE_URL + path
        headers = {"Authorization": "Bearer " + self.get_tenant_access_token()} if not skip_auth else {}
        for attempt, wait in enumerate(self.DEFAULT_WAIT):
            resp = requests.patch(url, headers=headers, json=json, timeout=30)
            if self._check_response(resp, path, attempt):
                return resp.json()
            time.sleep(wait)
        raise RuntimeError(f"Failed to PATCH {path} after {self.MAX_RETRIES} retries")

    def _check_response(self, resp: requests.Response, path: str, attempt: int) -> bool:
        try:
            data = resp.json() if resp.headers.get("content-type", "").startswith("application/json") else {}
        except Exception as e:
            logger.warning("Failed to parse response JSON from %s: %s", path, e)
            data = {}

        code = data.get("code", resp.status_code)

        if resp.status_code == 429 or code == 170002:
            logger.warning("Rate limited on %s (attempt %d), retrying...", path, attempt + 1)
            return False

        if code != 0:
            msg = data.get("msg", resp.reason)
            logger.error("API error on %s: code=%d, msg=%s", path, code, msg)
            return False

        return True

    @staticmethod
    def _chunk_records(records: List[Dict]) -> List[List[Dict]]:
        limit = FeishuClient.BATCH_LIMIT
        return [records[i:i + limit] for i in range(0, len(records), limit)]
