import csv
import json
import logging
import os

from scripts import config

logger = logging.getLogger(__name__)

ERROR_LOG = os.path.join(config.PROJECT_ROOT, "logs", "api", "error.log")
AUDIT_CSV = os.path.join(config.SYSTEM_DIR, "api_audit_dispatch.csv")
AUDIT_FEISHU_CSV = os.path.join(config.SYSTEM_DIR, "api_audit_feishu.csv")


def _search_jsonl(filepath, request_id, limit=100):
    if not os.path.exists(filepath):
        return []
    results = []
    try:
        with open(filepath) as f:
            for line in f:
                line = line.strip()
                if not line:
                    continue
                try:
                    obj = json.loads(line)
                except json.JSONDecodeError:
                    continue
                if obj.get("request_id") == request_id:
                    results.append(obj)
                    if len(results) >= limit:
                        break
    except OSError as e:
        logger.warning("Error reading %s: %s", filepath, e)
    return results


def _search_csv(filepath, request_id, limit=100):
    if not os.path.exists(filepath):
        return []
    results = []
    try:
        with open(filepath) as f:
            reader = csv.DictReader(f)
            for row in reader:
                if row.get("request_id") == request_id:
                    results.append(row)
                    if len(results) >= limit:
                        break
    except OSError as e:
        logger.warning("Error reading %s: %s", filepath, e)
    return results


def diagnose_by_request_id(request_id):
    error_entries = _search_jsonl(ERROR_LOG, request_id)
    audit_entries = _search_csv(AUDIT_CSV, request_id)
    feishu_entries = _search_csv(AUDIT_FEISHU_CSV, request_id)

    related_logs = []
    for e in error_entries:
        related_logs.append({
            "source": "error.log",
            "ts": e.get("timestamp", e.get("ts", "")),
            "error_code": e.get("error_code", ""),
            "message": e.get("message", ""),
            "diagnosis": e.get("diagnosis", ""),
        })
    for a in audit_entries:
        related_logs.append({
            "source": "audit_dispatch.csv",
            "timestamp": a.get("timestamp", ""),
            "outbox_id": a.get("outbox_id", ""),
            "status": a.get("status", ""),
            "error": a.get("error", ""),
        })
    for a in feishu_entries:
        related_logs.append({
            "source": "audit_feishu.csv",
            "timestamp": a.get("timestamp", ""),
            "action": a.get("action", ""),
            "status": a.get("status", ""),
        })

    if not related_logs:
        return None

    primary_error = error_entries[0] if error_entries else {}
    summary = primary_error.get("message", "No error message recorded")
    error_code = primary_error.get("error_code", "")
    diagnosis = primary_error.get("diagnosis", "")
    suggested_action = primary_error.get("suggested_action", "")

    return {
        "request_id": request_id,
        "summary": summary,
        "error_code": error_code,
        "diagnosis": diagnosis,
        "suggested_action": suggested_action,
        "related_logs": related_logs,
    }
