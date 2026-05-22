from fastapi import APIRouter, Depends

from api.dependencies import get_db, get_current_user
from api.schemas import StatusResponse
from services.db_service import get_table_counts as svc_get_table_counts
from services.status_service import get_last_pipeline_run

router = APIRouter(dependencies=[Depends(get_current_user)])


@router.get("/status", response_model=StatusResponse)
def get_status(conn=Depends(get_db)):
    counts = svc_get_table_counts(conn)
    last_run = get_last_pipeline_run(conn)
    run_dict = dict(last_run) if last_run else None

    return StatusResponse(
        database={
            "path": "data/olist_ops.db",
            "exists": True,
            "tables": counts,
        },
        last_pipeline_run=run_dict,
        version="0.5.0",
    )
