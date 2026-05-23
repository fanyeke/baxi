from typing import Optional

from fastapi import APIRouter, Depends, Query

from api.dependencies import get_db, get_current_user
from api.schemas import TaskItem, TaskListResponse
from services.task_service import get_tasks

router = APIRouter(dependencies=[Depends(get_current_user)])


@router.get("/tasks", response_model=TaskListResponse)
def list_tasks(
    status: Optional[str] = Query(None),
    priority: Optional[str] = Query(None),
    owner_role: Optional[str] = Query(None),
    limit: int = Query(100, ge=1, le=1000),
    conn=Depends(get_db),
):
    rows = get_tasks(
        conn,
        status=status,
        priority=priority,
        owner_role=owner_role,
        limit=limit,
    )
    items = [TaskItem(**dict(r)) for r in rows]
    count_conditions = []
    count_params = []
    if status is not None:
        count_conditions.append("status = ?")
        count_params.append(status)
    if priority is not None:
        count_conditions.append("priority = ?")
        count_params.append(priority)
    if owner_role is not None:
        count_conditions.append("owner_role = ?")
        count_params.append(owner_role)
    count_where = "WHERE " + " AND ".join(count_conditions) if count_conditions else ""
    total = conn.execute(f"SELECT COUNT(*) FROM action_tasks {count_where}", count_params).fetchone()[0]

    return TaskListResponse(items=items, total=total)
