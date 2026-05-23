"""FastAPI app factory for v0.5 API gateway."""

import os
import sqlite3
import uuid
import time
import logging
from collections import defaultdict
from contextvars import ContextVar

from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse
from fastapi.middleware.cors import CORSMiddleware

from api.errors import APIError, api_error_handler, validation_exception_handler
from api.logging_config import setup_logging, set_request_id, get_request_id
from services.db_service import get_db


def _sanitize_error(msg: str) -> str:
    """Remove sensitive patterns from error messages in debug mode."""
    import re
    sensitive = [
        (r'Bearer\s+\S+', '[BEARER_REDACTED]'),
        (r'(?:api[_-]?bearer[_-]?token|FEISHU_APP_SECRET|LLM_API_KEY)\s*[=:]\s*\S+', '[CREDENTIAL_REDACTED]'),
        (r'(?:password|secret|token|key)\s*[=:]\s*[\'"][^\'"]+[\'"]', '[CREDENTIAL_REDACTED]'),
        (r'(?:password|secret|token|key)\s*[=:]\s*\S+', '[CREDENTIAL_REDACTED]'),
    ]
    for pattern, replacement in sensitive:
        msg = re.sub(pattern, replacement, msg, flags=re.IGNORECASE)
    return msg[:500]


# ── Rate limiter (token bucket, per (IP, rate_class)) ─────────────────
_rate_limit_buckets: dict[str, dict[str, float]] = defaultdict(
    lambda: {"tokens": 0.0, "last_refill": 0.0}
)
_rate_limit_config = {
    "health": ("/api/v1/health", (30, 60)),
    "dispatch": ("/api/v1/outbox/dispatch", (30, 60)),
    "pipeline": ("/api/v1/pipeline/run", (10, 60)),
}
_DEFAULT_RATE = (300, 60)
_RATE_CLEANUP_INTERVAL = 300
_MAX_RATE_BUCKETS = 10000
_last_rate_cleanup = time.time()

# ── Trusted proxies ────────────────────────────────────────────────────
_TRUSTED_PROXIES = set(os.environ.get("TRUSTED_PROXY_IPS", "127.0.0.1,::1").split(","))


def _get_client_ip(request: Request) -> str:
    if request.client and request.client.host in _TRUSTED_PROXIES:
        forwarded = request.headers.get("X-Forwarded-For")
        if forwarded:
            return forwarded.split(",")[0].strip()
    return request.client.host if request.client else "unknown"


def _resolve_rate_class(path: str) -> tuple[str, int, int]:
    """Return (rate_class_key, limit, window) for path, default 'other'."""
    for key, (prefix, lw) in _rate_limit_config.items():
        if path.startswith(prefix):
            return key, lw[0], lw[1]
    return "other", _DEFAULT_RATE[0], _DEFAULT_RATE[1]


def _check_rate_limit(ip: str, path: str) -> bool:
    global _last_rate_cleanup
    now = time.time()

    if now - _last_rate_cleanup > _RATE_CLEANUP_INTERVAL:
        stale = [k for k, v in _rate_limit_buckets.items()
                 if now - v["last_refill"] > _RATE_CLEANUP_INTERVAL * 2]
        for k in stale:
            del _rate_limit_buckets[k]
        _last_rate_cleanup = now

    rate_key, limit, window = _resolve_rate_class(path)
    bucket_key = f"{ip}:{rate_key}"

    if bucket_key not in _rate_limit_buckets and len(_rate_limit_buckets) >= _MAX_RATE_BUCKETS:
        oldest = min(_rate_limit_buckets, key=lambda k: _rate_limit_buckets[k]["last_refill"])
        del _rate_limit_buckets[oldest]

    bucket = _rate_limit_buckets[bucket_key]
    elapsed = now - bucket["last_refill"]
    bucket["tokens"] = min(limit, bucket["tokens"] + elapsed * (limit / window))
    bucket["last_refill"] = now

    if bucket["tokens"] >= 1:
        bucket["tokens"] -= 1
        return True
    return False


def _reset_rate_limiter():
    """Reset rate limiter state for tests."""
    _rate_limit_buckets.clear()
    globals().update({"_last_rate_cleanup": time.time()})


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
            logger.critical(
                "Critical tables missing: %s. API will serve but data may be incomplete.",
                ", ".join(missing),
            )
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
        docs_url="/docs" if os.environ.get("ENABLE_DOCS") else None,
        openapi_url="/openapi.json" if os.environ.get("ENABLE_DOCS") else None,
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

    # ── Security headers ────────────────────────────────────────────────
    @app.middleware("http")
    async def security_headers_middleware(request: Request, call_next):
        response = await call_next(request)
        response.headers["X-Content-Type-Options"] = "nosniff"
        response.headers["X-Frame-Options"] = "DENY"
        response.headers["Strict-Transport-Security"] = (
            "max-age=31536000; includeSubDomains"
        )
        response.headers["X-XSS-Protection"] = "1; mode=block"
        response.headers["Referrer-Policy"] = "strict-origin-when-cross-origin"
        path = request.url.path
        if path.startswith("/docs") or path.startswith("/redoc") or path.startswith("/openapi.json"):
            response.headers["Content-Security-Policy"] = (
                "default-src 'self' https://cdn.jsdelivr.net https://unpkg.com; "
                "script-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net https://unpkg.com; "
                "style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net https://unpkg.com; "
                "img-src 'self' data:; connect-src 'self'; font-src 'self' https://cdn.jsdelivr.net; "
                "object-src 'none'"
            )
        else:
            response.headers["Content-Security-Policy"] = (
                "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; "
                "img-src 'self' data:; connect-src 'self'; font-src 'self'; object-src 'none'"
            )
        return response

    # ── Rate limiting middleware ────────────────────────────────────────
    @app.middleware("http")
    async def rate_limit_middleware(request: Request, call_next):
        if request.method == "OPTIONS":
            return await call_next(request)
        ip = _get_client_ip(request)
        if not _check_rate_limit(ip, request.url.path):
            rid = request.headers.get("X-Request-ID", str(uuid.uuid4()))
            content = {
                "request_id": rid,
                "error_code": "RATE_LIMITED",
                "message": "Too many requests. Please slow down.",
                "diagnosis": "Rate limit exceeded for this endpoint",
                "suggested_action": "Wait before retrying",
            }
            return JSONResponse(
                status_code=429,
                content=content,
                headers={
                    "X-Request-ID": rid,
                    "X-Content-Type-Options": "nosniff",
                    "X-Frame-Options": "DENY",
                    "Strict-Transport-Security": "max-age=31536000; includeSubDomains",
                    "X-XSS-Protection": "1; mode=block",
                    "Referrer-Policy": "strict-origin-when-cross-origin",
                    "Content-Security-Policy": (
                        "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; "
                        "img-src 'self' data:; connect-src 'self'; font-src 'self'; object-src 'none'"
                    ),
                },
            )
        return await call_next(request)

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
                "diagnosis": _sanitize_error(str(exc)) if os.environ.get("DEBUG") else "Internal server error",
                "suggested_action": "Check server logs for details or contact the administrator",
            },
        )

    # ── Startup migration ────────────────────────────────────────────────
    from scripts.config import DB_PATH

    _apply_migration(DB_PATH)

    # ── Router mounting ──────────────────────────────────────────────────
    from api.routers import health, status, alerts, tasks, outbox, logs, feishu, pipeline, diagnosis

    app.include_router(health.router, prefix="/api/v1", tags=["Health"])
    app.include_router(status.router, prefix="/api/v1", tags=["Status"])
    app.include_router(alerts.router, prefix="/api/v1", tags=["Alerts"])
    app.include_router(tasks.router, prefix="/api/v1", tags=["Tasks"])
    app.include_router(outbox.router, prefix="/api/v1", tags=["Outbox"])
    app.include_router(logs.router, prefix="/api/v1", tags=["Logs"])
    app.include_router(feishu.router, prefix="/api/v1", tags=["Feishu"])
    app.include_router(pipeline.router, prefix="/api/v1", tags=["Pipeline"])
    app.include_router(diagnosis.router, prefix="/api/v1", tags=["Logs"])
    from api.routers.governance import router as governance_router
    app.include_router(governance_router, prefix="/api/v1")

    # ── CORS ─────────────────────────────────────────────────────────────
    _cors_origins = os.environ.get(
        "CORS_ORIGINS", "http://localhost:5173"
    ).split(",")
    app.add_middleware(
        CORSMiddleware,
        allow_origins=_cors_origins,
        allow_methods=["GET", "POST", "OPTIONS"],
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
