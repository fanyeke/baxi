"""Diagnosis API router — v0.5.2: request-level error aggregation."""
from fastapi import APIRouter, Depends, Query
from fastapi.responses import JSONResponse

from api.dependencies import get_current_user
from services.diagnosis_service import diagnose_by_request_id

router = APIRouter(dependencies=[Depends(get_current_user)])


@router.get("/logs/diagnosis")
def get_diagnosis(request_id: str = Query(..., min_length=1)):
    result = diagnose_by_request_id(request_id)
    if result is None:
        return JSONResponse(
            status_code=404,
            content={
                "request_id": request_id,
                "error_code": "NOT_FOUND",
                "message": f"No logs found for request_id: {request_id}",
                "diagnosis": "The request_id was not found in error.log, audit CSV, or Feishu audit CSV.",
                "suggested_action": "Verify the request_id is correct. Logs may have been rotated or the request may not have generated an error.",
            },
        )
    return result
