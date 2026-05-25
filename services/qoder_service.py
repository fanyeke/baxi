"""
Qoder service layer — read-only API for the Qoder subsystem.

Provides:
- get_capabilities(): Read capabilities from YAML config
- build_context(): Aggregate system context from multiple services
- record_report(): Validate and persist a Qoder report
"""

import datetime
import hashlib
import json
import logging
import uuid

import yaml

from api.errors import VALIDATION_ERROR, APIError
from api.logging_config import get_request_id
from core import config
from services import alert_service, dispatch_service, status_service, task_service
from services.db_service import get_db
from services.diagnosis_service import diagnose_by_request_id

logger = logging.getLogger(__name__)


def get_capabilities():
    """Read Qoder capabilities from YAML config and return flat dict.

    Returns:
        dict with version, mode, allowed endpoints, forbidden actions,
        derived can_* boolean flags, and write exceptions.
    """
    cfg_path = config.QODER_CAPABILITIES_FILE
    with open(cfg_path) as f:
        raw = yaml.safe_load(f) or {}

    caps = raw.get("qoder_capabilities", {})

    allowed_endpoints = caps.get("allowed_endpoints", [])
    forbidden_actions = caps.get("forbidden_actions", [])
    write_exceptions = caps.get("write_exceptions", [])

    action_flags = {
        "can_sql": "direct_sql",
        "can_shell": "run_shell",
        "can_edit_files": "edit_files",
        "can_dispatch": "apply_dispatch",
        "can_feishu_sync": "feishu_sync_apply",
        "can_run_pipeline": "pipeline_run_apply",
        "can_modify_config": "modify_config",
    }

    result = {
        "version": "0.6.0",
        "mode": caps.get("mode", "read_only"),
        "allowed_endpoints": allowed_endpoints,
        "forbidden_actions": forbidden_actions,
        "write_exceptions": write_exceptions,
    }

    for flag_name, forbidden_action in action_flags.items():
        result[flag_name] = forbidden_action not in forbidden_actions

    return result


def build_context(
    conn=None, severity=None, limit_alerts=10, limit_tasks=10, limit_outbox=10, include_logs=False
):
    """Aggregate system context from multiple services (read-only).

    Calls existing service functions to collect current system state:
    last pipeline run, active alerts, open tasks, and pending outbox events.

    Args:
        conn: Optional database connection. If None, one is created and closed.
        severity: Optional alert severity filter (e.g. 'high', 'critical').
        limit_alerts: Max alerts to return (default 10).
        limit_tasks: Max tasks to return (default 10).
        limit_outbox: Max outbox items to return (default 10).
        include_logs: If True, include recent error diagnosis.

    Returns:
        dict with keys: system, summary, top_alerts, open_tasks,
        pending_outbox, and optionally diagnosis.
    """
    should_close = conn is None
    if conn is None:
        conn = get_db()
    try:
        pipeline_run = status_service.get_last_pipeline_run(conn)

        alerts, total_alerts = alert_service.get_alerts_with_count(
            conn,
            status="new",
            severity=severity,
            limit=limit_alerts,
        )

        tasks, total_tasks = task_service.get_tasks_with_count(
            conn,
            status=["todo", "in_progress", "open"],
            limit=limit_tasks,
        )

        outbox, total_outbox = dispatch_service.get_outbox_with_count(
            conn,
            status="pending",
            limit=limit_outbox,
        )

        caps = get_capabilities()

        result = {
            "request_id": get_request_id() or "ctx_unknown",
            "system": {
                "last_pipeline_run": pipeline_run,
            },
            "summary": {
                "total_alerts": total_alerts,
                "total_open_tasks": total_tasks,
                "total_pending_outbox": total_outbox,
            },
            "top_alerts": alerts,
            "open_tasks": tasks,
            "pending_outbox": outbox,
            "recent_diagnosis": [],
            "allowed_actions": [k for k, v in caps.items() if k.startswith("can_") and v],
            "forbidden_actions": caps.get("forbidden_actions", []),
        }

        if include_logs:
            rid = get_request_id()
            if rid:
                diagnosis = diagnose_by_request_id(request_id=rid)
                result["recent_diagnosis"] = [diagnosis] if diagnosis else []

        return result
    finally:
        if should_close:
            conn.close()


def record_report(conn=None, report_data=None):
    """Validate and persist a Qoder report (read-only execution record).

    Validates that ``no_apply_performed`` is True, then inserts into
    ``qoder_runs`` and ``qoder_reports`` tables. Uses ``INSERT OR IGNORE``
    on ``qoder_reports`` for idempotency based on ``report_id``.

    Args:
        conn: Optional database connection. If None, one is created and closed.
        report_data: dict with keys:
            - no_apply_performed (required, must be True)
            - run_type (required)
            - summary (required)
            - findings (optional, list)
            - recommended_human_actions (optional, list)
            - risk_level (optional, str)
            - used_endpoints (optional, list)
            - report_id (optional, str; auto-generated if omitted)

    Returns:
        report_id (str) of the created report.

    Raises:
        APIError: if ``no_apply_performed`` is not True.
    """
    if report_data is None:
        report_data = {}
    if not report_data.get("no_apply_performed"):
        raise APIError(
            error_code=VALIDATION_ERROR,
            message="no_apply_performed must be true",
            diagnosis="record_report requires no_apply_performed=True to "
            "enforce the read-only contract",
            suggested_action="Set no_apply_performed=True in the report payload",
        )

    now = datetime.datetime.now().isoformat()
    run_id = str(uuid.uuid4())
    run_type = report_data["run_type"]
    summary = report_data["summary"]
    findings = report_data.get("findings", [])

    client_report_id = report_data.get("report_id")
    if client_report_id:
        report_id = client_report_id
    else:
        findings_str = json.dumps(findings, sort_keys=True)
        content = f"{run_type}|{summary}|{findings_str}"
        content_hash = hashlib.sha256(content.encode("utf-8")).hexdigest()
        report_id = str(uuid.UUID(content_hash[:32]))
    actions = report_data.get("recommended_human_actions", [])
    risk_level = report_data.get("risk_level", "")
    endpoints = report_data.get("used_endpoints", [])

    should_close = conn is None
    if conn is None:
        conn = get_db()
    try:
        conn.execute(
            """
            INSERT INTO qoder_runs
                (run_id, run_type, mode, status, started_at,
                 finished_at, actor, can_apply)
            VALUES (?, ?, 'read_only', 'completed', ?, ?, 'qoder', 0)
        """,
            (run_id, run_type, now, now),
        )

        conn.execute(
            """
            INSERT OR IGNORE INTO qoder_reports
                (report_id, run_id, run_type, summary, findings_json,
                 recommended_human_actions_json, risk_level,
                 used_endpoints_json, no_apply_performed,
                 business_side_effect, created_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1, 0, ?)
        """,
            (
                report_id,
                run_id,
                run_type,
                summary,
                json.dumps(findings),
                json.dumps(actions),
                risk_level,
                json.dumps(endpoints),
                now,
            ),
        )

        conn.commit()
        logger.info(
            "Recorded Qoder report %s for run %s (type=%s)",
            report_id,
            run_id,
            run_type,
        )
        return report_id
    finally:
        if should_close:
            conn.close()
