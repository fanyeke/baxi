from fastapi import APIRouter

from api.schemas import HealthResponse
from services.db_service import db_exists

router = APIRouter()


@router.get("/health", response_model=HealthResponse)
def get_health():
    return HealthResponse(
        status="ok",
        version="0.5.0",
        db_connected=db_exists(),
    )
