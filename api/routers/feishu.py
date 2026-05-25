"""Feishu API router — v0.5.3 P1 endpoints for Feishu operations."""
import csv
import datetime
import os

from fastapi import APIRouter, Depends

from api.dependencies import get_current_user
from api.errors import VALIDATION_ERROR, APIError
from api.logging_config import get_request_id
from api.schemas import (
    FeishuExportRequest,
    FeishuExportResponse,
    FeishuStatusImportRequest,
    FeishuStatusImportResponse,
    FeishuSyncRequest,
    FeishuSyncResponse,
)
from services.feishu_service import FeishuService

router = APIRouter(dependencies=[Depends(get_current_user)])

AUDIT_CSV = os.path.join(
    os.path.dirname(os.path.dirname(os.path.dirname(__file__))),
    "data", "system", "api_audit_feishu.csv",
)


def _write_api_audit(request_id: str, action: str, dry_run: bool, result: dict):
    os.makedirs(os.path.dirname(AUDIT_CSV), exist_ok=True)
    write_header = not os.path.exists(AUDIT_CSV)
    with open(AUDIT_CSV, "a", newline="") as f:
        fields = ["request_id", "timestamp", "action", "mode", "status"]
        writer = csv.DictWriter(f, fieldnames=fields)
        if write_header:
            writer.writeheader()
        writer.writerow({
            "request_id": request_id,
            "timestamp": datetime.datetime.now().isoformat(),
            "action": action,
            "mode": "dry-run" if dry_run else "apply",
            "status": result.get("status", "unknown"),
        })


@router.post("/feishu/export", response_model=FeishuExportResponse)
def feishu_export(body: FeishuExportRequest):
    request_id = get_request_id()
    is_dry_run = not body.apply

    svc = FeishuService(dry_run=is_dry_run)
    try:
        result = svc.export_tables(table_names=body.tables)
    except ValueError as e:
        raise APIError(
            error_code=VALIDATION_ERROR,
            message=str(e),
            diagnosis="Invalid table name in request",
            suggested_action="Use one of the available table names",
            http_status=422,
        ) from e

    _write_api_audit(request_id, "feishu_export", is_dry_run, result)
    return FeishuExportResponse(
        status=result.get("status", "failed"),
        message=result.get("message", ""),
        tables=result.get("tables", []),
    )


@router.post("/feishu/sync", response_model=FeishuSyncResponse)
def feishu_sync(body: FeishuSyncRequest):
    request_id = get_request_id()
    is_dry_run = not body.apply

    svc = FeishuService(dry_run=is_dry_run)
    try:
        result = svc.sync_to_feishu(table_names=body.tables)
    except ValueError as e:
        raise APIError(
            error_code=VALIDATION_ERROR,
            message=str(e),
            diagnosis="Invalid table name in request",
            suggested_action="Use one of the available table names",
            http_status=422,
        ) from e

    _write_api_audit(request_id, "feishu_sync", is_dry_run, result)
    return FeishuSyncResponse(
        status=result.get("status", "failed"),
        message=result.get("message", ""),
        tables=result.get("tables", []),
    )


@router.post("/feishu/status/import", response_model=FeishuStatusImportResponse)
def feishu_status_import(body: FeishuStatusImportRequest):
    request_id = get_request_id()
    is_dry_run = not body.apply

    svc = FeishuService(dry_run=is_dry_run)
    try:
        result = svc.import_status_from_feishu(table_names=body.tables)
    except ValueError as e:
        raise APIError(
            error_code=VALIDATION_ERROR,
            message=str(e),
            diagnosis="Invalid table name in request",
            suggested_action="Use one of the available table names",
            http_status=422,
        ) from e

    _write_api_audit(request_id, "feishu_status_import", is_dry_run, result)
    return FeishuStatusImportResponse(
        status=result.get("status", "failed"),
        message=result.get("message", ""),
        tables=result.get("tables", []),
    )
