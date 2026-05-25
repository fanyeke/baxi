"""API error handling — structured errors with diagnosis and suggested action."""

from fastapi import Request
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse


class APIError(Exception):
    """Structured API error with machine-readable fields for Qoder consumption."""

    def __init__(
        self,
        error_code: str,
        message: str,
        diagnosis: str = "",
        suggested_action: str = "",
        http_status: int = 400,
    ):
        self.error_code = error_code
        self.message = message
        self.diagnosis = diagnosis
        self.suggested_action = suggested_action
        self.http_status = http_status
        super().__init__(message)


# ── Error code constants ─────────────────────────────────────────────────
AUTH_REQUIRED = "AUTH_REQUIRED"
INVALID_TOKEN = "INVALID_TOKEN"
DB_UNAVAILABLE = "DB_UNAVAILABLE"
DB_QUERY_FAILED = "DB_QUERY_FAILED"
OUTBOX_NOT_FOUND = "OUTBOX_NOT_FOUND"
DISPATCH_FAILED = "DISPATCH_FAILED"
FEISHU_DISPATCH_FAILED = "FEISHU_DISPATCH_FAILED"
VALIDATION_ERROR = "VALIDATION_ERROR"
CONFIG_MISSING = "CONFIG_MISSING"


async def api_error_handler(request: Request, exc: APIError) -> JSONResponse:
    """Convert APIError to structured JSON response."""
    import logging

    from api.logging_config import get_request_id

    rid = get_request_id()
    logging.getLogger("api.error").error(
        exc.message,
        extra={
            "request_id": rid,
            "error_code": exc.error_code,
            "diagnosis": exc.diagnosis,
            "suggested_action": exc.suggested_action,
        },
    )
    return JSONResponse(
        status_code=exc.http_status,
        content={
            "request_id": rid,
            "error_code": exc.error_code,
            "message": exc.message,
            "diagnosis": exc.diagnosis,
            "suggested_action": exc.suggested_action,
        },
    )


async def validation_exception_handler(
    request: Request, exc: RequestValidationError
) -> JSONResponse:
    """Wrap Pydantic validation errors into APIError format."""
    from api.logging_config import get_request_id

    rid = get_request_id()
    details = exc.errors()
    first_error = details[0] if details else {"msg": "unknown validation error"}
    field = ".".join(str(p) for p in first_error.get("loc", []))
    msg = first_error.get("msg", "validation failed")

    return JSONResponse(
        status_code=422,
        content={
            "request_id": rid,
            "error_code": VALIDATION_ERROR,
            "message": f"Validation error: {msg}",
            "diagnosis": f"Invalid value for '{field}'",
            "suggested_action": "Check the request body against the API schema at /docs",
        },
    )
