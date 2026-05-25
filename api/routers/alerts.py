from typing import Optional

from fastapi import APIRouter, Depends, Query

from api.dependencies import get_current_user, get_db
from api.schemas import AlertItem, AlertListResponse
from services.alert_service import get_alerts_with_count

router = APIRouter(dependencies=[Depends(get_current_user)])


@router.get("/alerts", response_model=AlertListResponse)
def list_alerts(
    status: Optional[str] = Query(None),
    severity: Optional[str] = Query(None),
    object_type: Optional[str] = Query(None),
    object_id: Optional[str] = Query(None),
    limit: int = Query(100, ge=1, le=1000),
    conn=Depends(get_db),
):
    items, total = get_alerts_with_count(
        conn, status=status, severity=severity,
        object_type=object_type, object_id=object_id, limit=limit,
    )
    return AlertListResponse(
        items=[AlertItem(**r) for r in items], total=total,
    )
