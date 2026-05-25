"""Diagnosis API router — v0.5.3: request-level error aggregation."""
from fastapi import APIRouter, Depends, Query

from api.dependencies import get_current_user
from api.errors import APIError
from api.schemas import DiagnosisResponse
from services.diagnosis_service import diagnose_by_request_id

router = APIRouter(dependencies=[Depends(get_current_user)])


@router.get("/logs/diagnosis", response_model=DiagnosisResponse)
def get_diagnosis(request_id: str = Query(..., min_length=1, max_length=128)):
    result = diagnose_by_request_id(request_id)
    if result is None:
        raise APIError(
            error_code="NOT_FOUND",
            message=f"No logs found for request_id: {request_id}",
            diagnosis="The request_id was not found in error.log, audit CSV, or Feishu audit CSV.",
            suggested_action="Verify the request_id is correct. Logs may have been rotated or the request may not have generated an error.",
            http_status=404,
        )
    return DiagnosisResponse(**result)
