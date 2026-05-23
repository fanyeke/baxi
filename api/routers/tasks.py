from typing import Optional

from fastapi import APIRouter, Depends, Query

from api.dependencies import get_db, get_current_user
from api.schemas import TaskItem, TaskListResponse
from services.task_service import get_tasks_with_count

router = APIRouter(dependencies=[Depends(get_current_user)])


@router.get("/tasks", response_model=TaskListResponse)
def list_tasks(
    status: Optional[str] = Query(None),
    priority: Optional[str] = Query(None),
    owner_role: Optional[str] = Query(None),
    limit: int = Query(100, ge=1, le=1000),
    conn=Depends(get_db),
):
    items, total = get_tasks_with_count(
        conn, status=status, priority=priority, owner_role=owner_role, limit=limit,
    )
    return TaskListResponse(
        items=[TaskItem(**r) for r in items], total=total,
    )
