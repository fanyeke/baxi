"""Pydantic v2 models for Qoder API endpoints."""

from typing import List, Optional

from pydantic import BaseModel, ConfigDict, Field

from api.schemas import AlertItem, OutboxItem, TaskItem


# ── Qoder capabilities ────────────────────────────────────────────────


class CapabilitiesResponse(BaseModel):
    """Qoder 能力矩阵响应."""

    model_config = ConfigDict(from_attributes=True)

    mode: str = Field(
        default="read_only",
        description="Qoder operating mode.",
        json_schema_extra={"example": "read_only"},
    )
    version: str = Field(
        default="0.6.0",
        description="Qoder protocol version.",
        json_schema_extra={"example": "0.6.0"},
    )
    can_read_status: bool = True
    can_read_alerts: bool = True
    can_read_tasks: bool = True
    can_read_outbox: bool = True
    can_read_logs: bool = True
    can_dispatch: bool = False
    can_apply: bool = False
    can_sync_feishu: bool = False
    can_run_pipeline: bool = False
    can_modify_rules: bool = False
    allowed_endpoints: List[str] = Field(
        default_factory=list,
        description="Endpoints this Qoder instance may call.",
        json_schema_extra={
            "example": ["/api/v1/alerts", "/api/v1/tasks"],
        },
    )
    forbidden_actions: List[str] = Field(
        default_factory=list,
        description="Actions explicitly denied for this Qoder instance.",
        json_schema_extra={"example": ["dispatch", "apply"]},
    )


# ── Context (GET /qoder/context) ──────────────────────────────────────


class ContextRequest(BaseModel):
    """Query parameters for GET /qoder/context."""

    severity: Optional[str] = Field(
        None,
        description="Filter alerts by severity level.",
        json_schema_extra={"example": "high"},
    )
    limit_alerts: int = Field(
        10,
        ge=1,
        le=100,
        description="Max alerts to include in context.",
        json_schema_extra={"example": 10},
    )
    limit_tasks: int = Field(
        10,
        ge=1,
        le=100,
        description="Max tasks to include in context.",
        json_schema_extra={"example": 10},
    )
    limit_outbox: int = Field(
        10,
        ge=1,
        le=100,
        description="Max outbox items to include in context.",
        json_schema_extra={"example": 10},
    )
    include_logs: bool = Field(
        False,
        description="Whether to attach recent error logs.",
        json_schema_extra={"example": False},
    )


class ContextResponse(BaseModel):
    """Aggregated system context for Qoder decision-making."""

    model_config = ConfigDict(from_attributes=True)

    request_id: str = Field(
        ...,
        description="Unique request identifier for tracing.",
        json_schema_extra={"example": "ctx_a1b2c3d4"},
    )
    system: dict = Field(
        ...,
        description="System status, version, and migration status.",
        json_schema_extra={
            "example": {
                "status": "ok",
                "version": "0.6.0",
                "migration_status": {"schema": "applied", "data": "pending"},
            }
        },
    )
    summary: dict = Field(
        ...,
        description="Aggregated summary counts.",
        json_schema_extra={
            "example": {
                "high_alerts": 3,
                "open_tasks": 5,
                "pending_outbox": 2,
                "recent_errors": 1,
            }
        },
    )
    top_alerts: List[AlertItem] = Field(
        ..., description="Highest-severity alerts."
    )
    open_tasks: List[TaskItem] = Field(
        ..., description="Currently open tasks."
    )
    pending_outbox: List[OutboxItem] = Field(
        ..., description="Pending dispatch items."
    )
    recent_diagnosis: List[dict] = Field(
        ...,
        description="Recent diagnosis entries from error logs.",
        json_schema_extra={
            "example": [
                {
                    "request_id": "req_xxx",
                    "error_code": "E001",
                    "diagnosis": "Rule 12 triggered on metric sell_rate",
                }
            ]
        },
    )
    allowed_actions: List[str] = Field(
        ...,
        description="Actions this Qoder instance is permitted to perform.",
        json_schema_extra={"example": ["read_status", "read_alerts"]},
    )
    forbidden_actions: List[str] = Field(
        ...,
        description="Actions this Qoder instance is NOT permitted to perform.",
        json_schema_extra={"example": ["apply", "dispatch"]},
    )


# ── Reports (POST /qoder/reports) ─────────────────────────────────────


class ReportRequest(BaseModel):
    """Request body for POST /qoder/reports."""

    run_type: str = Field(
        ...,
        min_length=1,
        description="Type of Qoder run (e.g. daily, manual, incident).",
        json_schema_extra={"example": "daily"},
    )
    summary: str = Field(
        ...,
        min_length=1,
        description="Human-readable summary of what was done.",
        json_schema_extra={
            "example": "Checked 15 alerts, dispatched 3 outbox items."
        },
    )
    findings: Optional[List[dict]] = Field(
        None,
        description="Structured findings from this run.",
        json_schema_extra={
            "example": [
                {"alert_id": "alert_001", "action": "dispatched"},
            ]
        },
    )
    recommended_human_actions: Optional[List[str]] = Field(
        None,
        description="Suggested manual follow-ups for human operators.",
        json_schema_extra={
            "example": ["Review high-severity alert alert_003"]
        },
    )
    risk_level: Optional[str] = Field(
        None,
        description="Overall risk assessment for this run.",
        json_schema_extra={"example": "low"},
    )
    used_endpoints: Optional[List[str]] = Field(
        None,
        description="Endpoints called during this run.",
        json_schema_extra={
            "example": ["/api/v1/alerts", "/api/v1/tasks"]
        },
    )
    no_apply_performed: bool = Field(
        True,
        description="True if no destructive apply was performed.",
        json_schema_extra={"example": True},
    )


class ReportResponse(BaseModel):
    """Response confirming a Qoder report was recorded."""

    model_config = ConfigDict(from_attributes=True)

    report_id: str = Field(
        ...,
        description="Unique report identifier.",
        json_schema_extra={"example": "rpt_e5f6g7h8"},
    )
    status: str = Field(
        "recorded",
        description="Report processing status.",
        json_schema_extra={"example": "recorded"},
    )
    business_side_effect: bool = Field(
        False,
        description="Whether the run had any business-side impact.",
        json_schema_extra={"example": False},
    )