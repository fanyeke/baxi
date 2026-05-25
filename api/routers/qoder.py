"""Qoder AI decision engine router — capabilities, context, and reports."""

from typing import Optional

from fastapi import APIRouter, Depends, Query

from api.dependencies import get_current_user, get_db
from api.schemas_qoder import (
    CapabilitiesResponse,
    ContextResponse,
    ReportRequest,
    ReportResponse,
)
from services import qoder_service

router = APIRouter(dependencies=[Depends(get_current_user)])


@router.get("/qoder/capabilities", response_model=CapabilitiesResponse)
def get_capabilities():
    caps = qoder_service.get_capabilities()
    return CapabilitiesResponse(**caps)


@router.get("/qoder/context", response_model=ContextResponse)
def get_context(
    severity: Optional[str] = Query(None),
    limit_alerts: int = Query(10, ge=1, le=100),
    limit_tasks: int = Query(10, ge=1, le=100),
    limit_outbox: int = Query(10, ge=1, le=100),
    include_logs: bool = Query(False),
    conn=Depends(get_db),
):
    ctx = qoder_service.build_context(
        conn, severity, limit_alerts, limit_tasks, limit_outbox, include_logs,
    )
    return ContextResponse(**ctx)


@router.post("/qoder/reports", response_model=ReportResponse)
def create_report(
    request: ReportRequest,
    conn=Depends(get_db),
):
    report_id = qoder_service.record_report(conn, request.model_dump())
    return ReportResponse(
        report_id=report_id, status="recorded", business_side_effect=False,
    )