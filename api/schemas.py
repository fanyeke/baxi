"""Pydantic v2 models for v0.5 API requests and responses."""

from datetime import datetime
from typing import Optional

from pydantic import BaseModel, ConfigDict


# ── Request models ───────────────────────────────────────────────────────


class DispatchRequest(BaseModel):
    """POST /outbox/dispatch request body."""

    channel: Optional[str] = None
    limit: int = 100
    dry_run: bool = True
    apply: bool = False


# ── Response models ──────────────────────────────────────────────────────


class HealthResponse(BaseModel):
    status: str
    version: str
    db_connected: bool


class StatusResponse(BaseModel):
    database: dict
    last_pipeline_run: Optional[dict] = None
    version: str


class AlertItem(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    event_id: str
    rule_id: str
    event_date: str
    severity: str
    metric_name: str = ""
    object_type: str = "global"
    object_id: str = "global"
    current_value: Optional[float] = None
    baseline_value: Optional[float] = None
    change_rate: Optional[float] = None
    owner_role: str = ""
    status: str = "new"
    impact_score: Optional[float] = None


class AlertListResponse(BaseModel):
    items: list[AlertItem]
    total: int


class TaskItem(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    task_id: str
    task_title: str
    task_description: str = ""
    status: str = "todo"
    priority: str = "medium"
    owner_role: str = ""
    owner_user_id: str = ""
    due_at: Optional[str] = None
    created_at: str = ""
    recommendation_id: Optional[str] = None
    event_id: Optional[str] = None
    target_object_type: Optional[str] = None
    target_object_id: Optional[str] = None


class TaskListResponse(BaseModel):
    items: list[TaskItem]
    total: int


class OutboxItem(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    outbox_id: str
    event_type: str
    source_type: str = ""
    source_id: str = ""
    target_channel: str
    status: str
    created_at: str = ""
    dispatch_attempts: int = 0
    last_dispatch_at: Optional[str] = None


class OutboxListResponse(BaseModel):
    items: list[OutboxItem]
    total: int


class DispatchResultItem(BaseModel):
    outbox_id: str
    status: str
    adapter_name: Optional[str] = None
    message: Optional[str] = None
    external_ref: Optional[str] = None
    error: Optional[str] = None


class DispatchResponse(BaseModel):
    request_id: str
    dry_run: bool
    processed: int
    results: list[DispatchResultItem]


class ErrorResponse(BaseModel):
    request_id: str
    error_code: str
    message: str
    diagnosis: str
    suggested_action: str
