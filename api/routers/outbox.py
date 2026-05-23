import csv
import datetime
import os
from typing import Optional

from fastapi import APIRouter, Depends, Query

from api.dependencies import get_db, get_current_user
from api.logging_config import get_request_id
from api.schemas import (
    DispatchRequest,
    DispatchResponse,
    DispatchResultItem,
    OutboxItem,
    OutboxListResponse,
)
from adapters.base import load_adapter_registry
from services.dispatch_service import (
    dispatch_one,
    fetch_pending,
    get_outbox_with_count,
)

router = APIRouter(dependencies=[Depends(get_current_user)])

AUDIT_CSV = os.path.join(
    os.path.dirname(os.path.dirname(os.path.dirname(__file__))),
    "data", "system", "api_audit_dispatch.csv",
)


def _write_api_audit(request_id: str, entries: list[dict]) -> None:
    os.makedirs(os.path.dirname(AUDIT_CSV), exist_ok=True)
    write_header = not os.path.exists(AUDIT_CSV)
    with open(AUDIT_CSV, "a", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=[
            "request_id", "timestamp", "outbox_id", "target_channel",
            "adapter_name", "mode", "status", "external_ref", "error",
        ])
        if write_header:
            writer.writeheader()
        for entry in entries:
            entry["request_id"] = request_id
            entry["timestamp"] = datetime.datetime.now().isoformat()
        writer.writerows(entries)


@router.get("/outbox", response_model=OutboxListResponse)
def list_outbox(
    status: Optional[str] = Query(None, alias="status"),
    channel: Optional[str] = Query(None),
    limit: int = Query(100, ge=1, le=1000),
    conn=Depends(get_db),
):
    items, total = get_outbox_with_count(
        conn, status=status, channel=channel, limit=limit,
    )
    return OutboxListResponse(
        items=[OutboxItem(**r) for r in items], total=total,
    )


@router.post("/outbox/dispatch", response_model=DispatchResponse)
def dispatch_outbox(body: DispatchRequest, conn=Depends(get_db)):
    request_id = get_request_id()
    is_dry_run = not body.apply
    mode = "dry-run" if is_dry_run else "apply"

    registry = load_adapter_registry()
    pending = fetch_pending(conn, channel=body.channel, limit=body.limit)
    results = []
    audit_entries = []

    for event in pending:
        event_dict = dict(event)
        outbox_id = event_dict["outbox_id"]
        target = event_dict.get("target_channel", "unknown")

        try:
            result = dispatch_one(conn, event_dict, registry, is_dry_run)
        except Exception as e:
            result = {"status": "failed", "external_ref": None, "error": str(e), "message": None}

        adapter_name = result.get("adapter_name")
        dispatch_status = result.get("status", "failed")

        results.append(DispatchResultItem(
            outbox_id=outbox_id,
            status=dispatch_status,
            adapter_name=adapter_name,
            message=result.get("message"),
            external_ref=result.get("external_ref"),
            error=result.get("error"),
        ))

        audit_entries.append({
            "outbox_id": outbox_id,
            "target_channel": target,
            "adapter_name": adapter_name,
            "mode": mode,
            "status": dispatch_status,
            "external_ref": result.get("external_ref"),
            "error": result.get("error"),
        })

    _write_api_audit(request_id, audit_entries)
    if not is_dry_run:
        conn.commit()
    return DispatchResponse(
        request_id=request_id,
        dry_run=is_dry_run,
        processed=len(results),
        results=results,
    )
