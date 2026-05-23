"""FastAPI app factory for v0.5 API gateway."""

import os
import sqlite3
import uuid
import time
import logging
from contextvars import ContextVar

from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse
from fastapi.middleware.cors import CORSMiddleware

from api.errors import APIError, api_error_handler, validation_exception_handler
from api.logging_config import setup_logging, set_request_id, get_request_id
from services.db_service import get_db


_request_id_ctx: ContextVar[str] = ContextVar("request_id", default="")

logger = logging.getLogger("api")
_migration_status = {"status": "ok", "failed": []}


def _apply_migration(db_path: str) -> None:
    """Apply schema migrations if required columns are missing."""
    migration_dir = os.path.join(
        os.path.dirname(os.path.dirname(__file__)),
        "sql", "migrations",
    )
    if not os.path.exists(db_path):
        return

    conn = sqlite3.connect(db_path)
    try:
        _migrate_if_needed(conn, migration_dir, "005_dispatch_adapters.sql",
                           "event_outbox",
                           ["dispatch_attempts", "last_dispatch_at",
                            "external_ref", "adapter_name"])
        _migrate_if_needed(conn, migration_dir, "006_api_schema_fix.sql",
                           "alert_events",
                           ["affected_orders", "affected_gmv", "impact_score"])
        _migrate_if_needed(conn, migration_dir, "006_api_schema_fix.sql",
                           "action_tasks",
                           ["target_object_type", "target_object_id"])
        _migrate_if_needed(conn, migration_dir, "007_review_retro_status_feedback.sql",
                           "review_retro", ["status", "feedback"])

        # ── Critical table check ─────────────────────────────────
        critical_tables = ["dwd_order_level", "alert_events"]
        cur = conn.execute(
            "SELECT name FROM sqlite_master WHERE type='table'"
        )
        existing_tables = {r[0] for r in cur.fetchall()}
        missing = [t for t in critical_tables if t not in existing_tables]
        if missing:
            _migration_status["status"] = "failed"
            _migration_status["failed"].extend([
                {"table": t, "migration": "startup_check", "error": "critical table missing"}
                for t in missing
            ])
            logger.critical("Critical tables missing: %s", ", ".join(missing))
            raise SystemExit(f"Critical database tables missing: {', '.join(missing)}")
    finally:
        conn.close()


def _migrate_if_needed(conn, migration_dir, filename, table, columns):
    migration_file = os.path.join(migration_dir, filename)
    if not os.path.exists(migration_file):
        return

    cur = conn.execute(f"PRAGMA table_info({table})")
    existing = [r[1] for r in cur.fetchall()]
    if all(c in existing for c in columns):
        return

    with open(migration_file) as f:
        raw = f.read()

    for stmt in raw.split(";"):
        lines = [l.strip() for l in stmt.split("\n")
                 if l.strip() and not l.strip().startswith("--")]
        stmt_clean = "\n".join(lines).strip()
        if not stmt_clean:
            continue
        try:
            conn.execute(stmt_clean)
        except sqlite3.OperationalError as e:
            logger.warning("Migration failed for %s (%s): %s", table, filename, e)
            _migration_status["failed"].append({
                "table": table,
                "migration": filename,
                "error": str(e),
            })
            _migration_status["status"] = "degraded"

    conn.commit()


def create_app() -> FastAPI:
    setup_logging()

    app = FastAPI(
        title="Olist Decision Backend API",
        version="0.5.1",
        docs_url="/docs",
        openapi_url="/openapi.json",
    )

    # ── Request ID middleware ────────────────────────────────────────────
    @app.middleware("http")
    async def request_id_middleware(request: Request, call_next):
        rid = request.headers.get("X-Request-ID", str(uuid.uuid4()))
        set_request_id(rid)
        t0 = time.time()
        response = await call_next(request)
        elapsed = int((time.time() - t0) * 1000)
        response.headers["X-Request-ID"] = rid
        logging.getLogger("api").info(
            "%s %s %d %dms",
            request.method, request.url.path, response.status_code, elapsed,
            extra={
                "method": request.method,
                "path": request.url.path,
                "status_code": response.status_code,
                "elapsed_ms": elapsed,
            },
        )
        return response

    # ── Exception handlers ───────────────────────────────────────────────
    app.add_exception_handler(APIError, api_error_handler)
    app.add_exception_handler(
        __import__("fastapi.exceptions", fromlist=["RequestValidationError"]).RequestValidationError,
        validation_exception_handler,
    )

    @app.exception_handler(Exception)
    async def global_exception_handler(request: Request, exc: Exception):
        rid = get_request_id()
        logging.getLogger("api").error(
            "Unhandled exception: %s", exc,
            extra={"request_id": rid, "path": request.url.path},
        )
        return JSONResponse(
            status_code=500,
            content={
                "request_id": rid,
                "error_code": "INTERNAL_ERROR",
                "message": "An unexpected error occurred",
                "diagnosis": str(exc) if os.environ.get("DEBUG") else "Internal server error",
                "suggested_action": "Check server logs for details or contact the administrator",
            },
        )

    # ── Startup migration ────────────────────────────────────────────────
    from scripts.config import DB_PATH

    _apply_migration(DB_PATH)

    # ── Router mounting ──────────────────────────────────────────────────
    from api.routers import health, status, alerts, tasks, outbox, logs, feishu, pipeline

    app.include_router(health.router, prefix="/api/v1", tags=["Health"])
    app.include_router(status.router, prefix="/api/v1", tags=["Status"])
    app.include_router(alerts.router, prefix="/api/v1", tags=["Alerts"])
    app.include_router(tasks.router, prefix="/api/v1", tags=["Tasks"])
    app.include_router(outbox.router, prefix="/api/v1", tags=["Outbox"])
    app.include_router(logs.router, prefix="/api/v1", tags=["Logs"])
    app.include_router(feishu.router, prefix="/api/v1", tags=["Feishu"])
    app.include_router(pipeline.router, prefix="/api/v1", tags=["Pipeline"])

    # ── CORS ─────────────────────────────────────────────────────────────
    _cors_origins = os.environ.get(
        "CORS_ORIGINS", "http://localhost:5173"
    ).split(",")
    app.add_middleware(
        CORSMiddleware,
        allow_origins=_cors_origins,
        allow_methods=["*"],
        allow_headers=["Authorization", "Content-Type", "X-Request-ID"],
    )

    # ── Frontend static serving (if built) ────────────────────────────────
    _frontend_dist = os.path.join(
        os.path.dirname(os.path.dirname(__file__)), "frontend", "dist"
    )
    if os.path.exists(os.path.join(_frontend_dist, "index.html")):
        from fastapi.staticfiles import StaticFiles
        app.mount("/console", StaticFiles(directory=_frontend_dist, html=True), name="console")

    return app


app = create_app()
