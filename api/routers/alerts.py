from typing import Optional

from fastapi import APIRouter, Depends, Query

from api.dependencies import get_db, get_current_user
from api.schemas import AlertItem, AlertListResponse
from services.alert_service import get_alerts

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
    rows = get_alerts(
        conn,
        status=status,
        severity=severity,
        object_type=object_type,
        object_id=object_id,
        limit=limit,
    )
    items = [AlertItem(**dict(r)) for r in rows]
    return AlertListResponse(items=items, total=len(items))
