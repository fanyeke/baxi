"""Structured JSON logging for the v0.5 API gateway.

Provides three loggers:
  - api:    all requests (INFO, writes to logs/api/api.log)
  - error:  errors only (ERROR, writes to logs/api/error.log)
  - audit:  write operations (INFO, writes to logs/api/audit.log)
"""

import logging
import os
import uuid
from contextvars import ContextVar

from pythonjsonlogger.json import JsonFormatter

# ── Constants ───────────────────────────────────────────────────────────
LOG_DIR = os.path.join(os.path.dirname(os.path.dirname(__file__)), "logs", "api")

# Context variable for request_id (set by middleware in api/main.py)
_request_id_ctx: ContextVar[str] = ContextVar("request_id", default="")


# ── JSON Formatter ──────────────────────────────────────────────────────
class APIJsonFormatter(JsonFormatter):
    """JSON formatter that adds standard API fields."""

    def add_fields(self, log_record, record, message_dict):
        super().add_fields(log_record, record, message_dict)
        log_record["ts"] = self.formatTime(record)
        # Remove default 'asctime' if present (we use 'ts' instead)
        log_record.pop("asctime", None)
        # Ensure level is present
        if "level" not in log_record:
            log_record["level"] = record.levelname

    def format(self, record):
        # Ensure request_id is always present
        if not hasattr(record, "request_id"):
            record.request_id = _request_id_ctx.get("")
        if not hasattr(record, "actor"):
            record.actor = getattr(record, "actor", "unknown")
        return super().format(record)


# ── Setup ───────────────────────────────────────────────────────────────
def setup_logging() -> None:
    """Configure all API loggers with JSON formatting.

    Creates logs/api/ directory if it doesn't exist.
    Should be called once at application startup.
    """
    os.makedirs(LOG_DIR, exist_ok=True)

    # Shared formatter
    formatter = APIJsonFormatter()

    # ── API Logger: all requests ────────────────────────────────────────
    api_handler = logging.FileHandler(os.path.join(LOG_DIR, "api.log"))
    api_handler.setLevel(logging.INFO)
    api_handler.setFormatter(formatter)

    api_logger = logging.getLogger("api")
    api_logger.setLevel(logging.INFO)
    api_logger.handlers.clear()
    api_logger.addHandler(api_handler)
    api_logger.propagate = False

    # ── Error Logger: errors only ───────────────────────────────────────
    error_handler = logging.FileHandler(os.path.join(LOG_DIR, "error.log"))
    error_handler.setLevel(logging.ERROR)
    error_handler.setFormatter(formatter)

    error_logger = logging.getLogger("api.error")
    error_logger.setLevel(logging.ERROR)
    error_logger.handlers.clear()
    error_logger.addHandler(error_handler)
    error_logger.propagate = False

    # ── Audit Logger: write operations ──────────────────────────────────
    audit_handler = logging.FileHandler(os.path.join(LOG_DIR, "audit.log"))
    audit_handler.setLevel(logging.INFO)
    audit_handler.setFormatter(formatter)

    audit_logger = logging.getLogger("api.audit")
    audit_logger.setLevel(logging.INFO)
    audit_logger.handlers.clear()
    audit_logger.addHandler(audit_handler)
    audit_logger.propagate = False

    # Log startup
    api_logger.info(
        "API logging initialized",
        extra={"request_id": "startup", "method": "SYSTEM", "path": "/"},
    )


def get_request_id() -> str:
    """Get current request_id from context, generating one if missing."""
    rid = _request_id_ctx.get("")
    if not rid:
        rid = str(uuid.uuid4())
        _request_id_ctx.set(rid)
    return rid


def set_request_id(rid: str) -> None:
    """Set the current request_id in context."""
    _request_id_ctx.set(rid)
