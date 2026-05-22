"""FastAPI app factory for v0.5 API gateway."""

import os
import sqlite3
import uuid
from contextvars import ContextVar

from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse

from api.errors import APIError, api_error_handler, validation_exception_handler
from api.logging_config import setup_logging, set_request_id, get_request_id
from services.db_service import get_db


_request_id_ctx: ContextVar[str] = ContextVar("request_id", default="")


def _apply_migration(db_path: str) -> None:
    """Apply 006_api_schema_fix.sql if dispatch_attempts column is missing."""
    migration_file = os.path.join(
        os.path.dirname(os.path.dirname(__file__)),
        "sql", "migrations", "006_api_schema_fix.sql",
    )
    if not os.path.exists(migration_file) or not os.path.exists(db_path):
        return

    conn = sqlite3.connect(db_path)
    try:
        cols = [r[1] for r in conn.execute("PRAGMA table_info(event_outbox)")]
        if "dispatch_attempts" in cols:
            return

        with open(migration_file) as f:
            for stmt in f.read().split(";"):
                stmt = stmt.strip()
                if stmt and not stmt.startswith("--"):
                    try:
                        conn.execute(stmt)
                    except sqlite3.OperationalError:
                        pass
        conn.commit()
    finally:
        conn.close()


def create_app() -> FastAPI:
    setup_logging()

    app = FastAPI(
        title="Olist Decision Backend API",
        version="0.5.0",
        docs_url="/docs",
        openapi_url="/openapi.json",
    )

    # ── Request ID middleware ────────────────────────────────────────────
    @app.middleware("http")
    async def request_id_middleware(request: Request, call_next):
        rid = request.headers.get("X-Request-ID", str(uuid.uuid4()))
        set_request_id(rid)
        response = await call_next(request)
        response.headers["X-Request-ID"] = rid
        return response

    # ── Exception handlers ───────────────────────────────────────────────
    app.add_exception_handler(APIError, api_error_handler)
    app.add_exception_handler(
        __import__("fastapi.exceptions", fromlist=["RequestValidationError"]).RequestValidationError,
        validation_exception_handler,
    )

    # ── Startup migration ────────────────────────────────────────────────
    from scripts.config import DB_PATH

    _apply_migration(DB_PATH)

    # ── Router mounting ──────────────────────────────────────────────────
    from api.routers import health, status, alerts, tasks, outbox

    app.include_router(health.router, prefix="/api/v1", tags=["Health"])
    app.include_router(status.router, prefix="/api/v1", tags=["Status"])
    app.include_router(alerts.router, prefix="/api/v1", tags=["Alerts"])
    app.include_router(tasks.router, prefix="/api/v1", tags=["Tasks"])
    app.include_router(outbox.router, prefix="/api/v1", tags=["Outbox"])

    return app


app = create_app()
