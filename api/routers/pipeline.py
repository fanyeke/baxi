"""Pipeline API router — v0.5.1 P1 endpoint for pipeline command preview."""
from fastapi import APIRouter, Depends

from api.dependencies import get_current_user
from api.schemas import PipelineRunRequest, PipelineRunResponse
from services.pipeline_service import preview_pipeline_run

router = APIRouter(dependencies=[Depends(get_current_user)])


@router.post("/pipeline/run", response_model=PipelineRunResponse)
def pipeline_run(body: PipelineRunRequest):
    result = preview_pipeline_run(body.pipeline_type)
    return PipelineRunResponse(
        command=result.get("command", ""),
        pipeline_type=result.get("pipeline_type", body.pipeline_type),
        estimated_duration=result.get("estimated_duration", ""),
        required_env_vars=result.get("required_env_vars", []),
        warnings=result.get("warnings", []),
        description=result.get("description", ""),
    )
