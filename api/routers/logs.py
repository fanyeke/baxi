"""Logs API router — v0.5.3 P1 endpoints for reading log files."""
import os

from fastapi import APIRouter, Depends, Query

from api.dependencies import get_current_user
from api.schemas import (
    AuditLogEntry,
    AuditLogListResponse,
    ErrorLogEntry,
    ErrorLogListResponse,
    RecentLogEntry,
    RecentLogListResponse,
)
from core import config
from services.log_reader import read_audit_logs, read_log_errors, read_log_recent

router = APIRouter(dependencies=[Depends(get_current_user)])

LOG_DIR = os.path.join(config.PROJECT_ROOT, "logs", "api")


@router.get("/logs/errors", response_model=ErrorLogListResponse)
def list_log_errors(
    request_id: str = Query(None),
    error_code: str = Query(None),
    limit: int = Query(100, ge=1, le=500),
):
    entries = read_log_errors(
        os.path.join(LOG_DIR, "error.log"),
        request_id=request_id,
        limit=limit,
    )
    if error_code:
        entries = [e for e in entries if e.get("error_code") == error_code]
    items = [ErrorLogEntry(**e) for e in entries]
    return ErrorLogListResponse(items=items, total=len(items))


@router.get("/logs/audit", response_model=AuditLogListResponse)
def list_log_audit(
    outbox_id: str = Query(None),
    status: str = Query(None),
    limit: int = Query(100, ge=1, le=500),
    source: str = Query("dispatch"),
):
    if source == "feishu":
        audit_path = os.path.join(config.SYSTEM_DIR, "api_audit_feishu.csv")
    else:
        audit_path = os.path.join(config.SYSTEM_DIR, "api_audit_dispatch.csv")
    entries = read_audit_logs(
        audit_path,
        outbox_id=outbox_id,
        status=status,
        limit=limit,
    )
    items = [AuditLogEntry(**e) for e in entries]
    return AuditLogListResponse(items=items, total=len(items))


@router.get("/logs/recent", response_model=RecentLogListResponse)
def list_log_recent(
    limit: int = Query(50, ge=1, le=500),
):
    entries = read_log_recent(
        os.path.join(LOG_DIR, "api.log"),
        limit=limit,
    )
    items = [RecentLogEntry(**e) for e in entries]
    return RecentLogListResponse(items=items, total=len(items))
