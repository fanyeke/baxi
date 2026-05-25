"""Pydantic v2 models for v0.5 API requests and responses."""

from typing import Optional

from pydantic import BaseModel, ConfigDict, field_validator

# ── Request models ───────────────────────────────────────────────────────


class DispatchRequest(BaseModel):
    """POST /outbox/dispatch request body."""

    channel: Optional[str] = None
    limit: int = 100
    apply: bool = False

    @field_validator("limit")
    @classmethod
    def validate_limit(cls, v: int) -> int:
        if v < 1 or v > 1000:
            raise ValueError("limit must be between 1 and 1000")
        return v


# ── Response models ──────────────────────────────────────────────────────


class HealthResponse(BaseModel):
    status: str
    version: str
    db_connected: bool


class StatusResponse(BaseModel):
    database: dict
    last_pipeline_run: Optional[dict] = None
    version: str
    migration_status: dict = {}


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
    task_description: Optional[str] = None
    status: str = "todo"
    priority: str = "medium"
    owner_role: str = ""
    owner_user_id: Optional[str] = None
    due_at: Optional[str] = None
    created_at: str = ""
    completed_at: Optional[str] = None
    feedback: Optional[str] = None
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


# ── v0.5.3 Logs models ───────────────────────────────────────────────────


class ErrorLogEntry(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    ts: str
    level: str = "ERROR"
    message: str = ""
    request_id: str = ""
    error_code: str = ""
    diagnosis: str = ""
    suggested_action: str = ""
    actor: str = "unknown"


class ErrorLogListResponse(BaseModel):
    items: list[ErrorLogEntry]
    total: int


class RecentLogEntry(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    ts: str
    level: str = ""
    message: str = ""
    request_id: str = ""
    method: str = ""
    path: str = ""
    actor: str = "unknown"


class RecentLogListResponse(BaseModel):
    items: list[RecentLogEntry]
    total: int


class AuditLogEntry(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    timestamp: str = ""
    outbox_id: str = ""
    target_channel: str = ""
    adapter_name: str = ""
    mode: Optional[str] = None
    status: Optional[str] = None
    external_ref: Optional[str] = None
    error: Optional[str] = None
    request_id: Optional[str] = None
    source: str = "api"


class AuditLogListResponse(BaseModel):
    items: list[AuditLogEntry]
    total: int


# ── v0.5.3 Feishu models ──────────────────────────────────────────────────


class FeishuExportRequest(BaseModel):
    tables: Optional[list[str]] = None
    apply: bool = False

    @field_validator("tables")
    @classmethod
    def validate_tables(cls, v):
        if v is not None:
            if len(v) > 20:
                raise ValueError("tables list cannot exceed 20 items")
            for t in v:
                if not t.replace("_", "").isalnum():
                    raise ValueError(f"Invalid table name: {t}")
        return v


class FeishuTableResult(BaseModel):
    name: str
    status: str = ""
    rows: int = 0
    file: str = ""
    created: int = 0
    updated: int = 0
    pulled: int = 0
    imported: int = 0
    skipped: int = 0


class FeishuExportResponse(BaseModel):
    status: str
    message: str = ""
    tables: list[FeishuTableResult] = []


class FeishuSyncRequest(BaseModel):
    tables: Optional[list[str]] = None
    apply: bool = False

    @field_validator("tables")
    @classmethod
    def validate_tables(cls, v):
        if v is not None:
            if len(v) > 20:
                raise ValueError("tables list cannot exceed 20 items")
            for t in v:
                if not t.replace("_", "").isalnum():
                    raise ValueError(f"Invalid table name: {t}")
        return v


class FeishuSyncResponse(BaseModel):
    status: str
    message: str = ""
    tables: list[FeishuTableResult] = []


class FeishuStatusImportRequest(BaseModel):
    tables: Optional[list[str]] = None
    apply: bool = False

    @field_validator("tables")
    @classmethod
    def validate_tables(cls, v):
        if v is not None:
            if len(v) > 20:
                raise ValueError("tables list cannot exceed 20 items")
            for t in v:
                if not t.replace("_", "").isalnum():
                    raise ValueError(f"Invalid table name: {t}")
        return v


class FeishuStatusImportResponse(BaseModel):
    status: str
    message: str = ""
    tables: list[FeishuTableResult] = []


# ── v0.5.3 Pipeline models ────────────────────────────────────────────────


from typing import Literal

class PipelineRunRequest(BaseModel):
    pipeline_type: Literal["daily", "full", "db_full"] = "daily"


class PipelineRunResponse(BaseModel):
    command: str
    pipeline_type: str
    estimated_duration: str = ""
    required_env_vars: list[str] = []
    warnings: list[str] = []
    description: str = ""


# ── Governance response models ────────────────────────────────────────


class GovernanceConfigResponse(BaseModel):
    """Generic governance YAML config response — wraps arbitrary YAML content."""
    model_config = ConfigDict(extra="allow")


class GovernanceStatusResponse(BaseModel):
    """Aggregated governance status across all config files."""
    governance_layer: str
    configs: dict


# ── Diagnosis response models ─────────────────────────────────────────


class DiagnosisLogEntry(BaseModel):
    """Single log entry from a related diagnosis source."""
    model_config = ConfigDict(extra="allow")

    source: str
    ts: Optional[str] = ""
    timestamp: Optional[str] = ""
    error_code: Optional[str] = ""
    message: Optional[str] = ""
    diagnosis: Optional[str] = ""
    outbox_id: Optional[str] = ""
    status: Optional[str] = ""
    error: Optional[str] = ""
    action: Optional[str] = ""


class DiagnosisResponse(BaseModel):
    """Structured diagnosis result for a request_id lookup."""
    request_id: str
    summary: str = ""
    error_code: str = ""
    diagnosis: str = ""
    suggested_action: str = ""
    related_logs: list[DiagnosisLogEntry] = []
