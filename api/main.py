"""FastAPI app factory for v0.5 API gateway."""

import logging
import os
import sqlite3
import time
import uuid
from collections import defaultdict
from contextvars import ContextVar

from fastapi import FastAPI, Request
from fastapi.exceptions import RequestValidationError
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse

from api.dependencies import _migration_status
from api.errors import APIError, api_error_handler, validation_exception_handler
from api.logging_config import get_request_id, set_request_id, setup_logging


def _sanitize_error(msg: str) -> str:
    """Remove sensitive patterns from error messages in debug mode."""
    import re

    sensitive = [
        (r"Bearer\s+\S+", "[BEARER_REDACTED]"),
        (
            r"(?:api[_-]?bearer[_-]?token|FEISHU_APP_SECRET|LLM_API_KEY)\s*[=:]\s*\S+",
            "[CREDENTIAL_REDACTED]",
        ),
        (r'(?:password|secret|token|key)\s*[=:]\s*[\'"][^\'"]+[\'"]', "[CREDENTIAL_REDACTED]"),
        (r"(?:password|secret|token|key)\s*[=:]\s*\S+", "[CREDENTIAL_REDACTED]"),
        # Security: also redact file paths, database paths, and internal IPs
        (r'(?:/home/|/tmp/|/var/|/opt/)[^\s\'"]+', "[PATH_REDACTED]"),
        (
            r"\b(?:127\.\d+\.\d+\.\d+|192\.168\.\d+\.\d+|10\.\d+\.\d+\.\d+|172\.(?:1[6-9]|2[0-9]|3[01])\.\d+\.\d+)\b",
            "[IP_REDACTED]",
        ),
        (r"\b[A-Fa-f0-9]{32,}\b", "[HASH_REDACTED]"),
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
    "qoder": ("/api/v1/qoder", (60, 60)),
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
        stale = [
            k
            for k, v in _rate_limit_buckets.items()
            if now - v["last_refill"] > _RATE_CLEANUP_INTERVAL * 2
        ]
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


def _apply_migration(db_path: str) -> None:
    """Apply schema migrations if required columns are missing."""
    migration_dir = os.path.join(
        os.path.dirname(os.path.dirname(__file__)),
        "sql",
        "migrations",
    )
    if not os.path.exists(db_path):
        return

    conn = sqlite3.connect(db_path)
    try:
        _migrate_if_needed(
            conn,
            migration_dir,
            "005_dispatch_adapters.sql",
            "event_outbox",
            ["dispatch_attempts", "last_dispatch_at", "external_ref", "adapter_name"],
        )
        _migrate_if_needed(
            conn,
            migration_dir,
            "006_api_schema_fix.sql",
            "alert_events",
            ["affected_orders", "affected_gmv", "impact_score"],
        )
        _migrate_if_needed(
            conn,
            migration_dir,
            "006_api_schema_fix.sql",
            "action_tasks",
            ["target_object_type", "target_object_id"],
        )
        _migrate_if_needed(
            conn,
            migration_dir,
            "007_review_retro_status_feedback.sql",
            "review_retro",
            ["status", "feedback"],
        )

        # ── Critical table check ─────────────────────────────────
        critical_tables = ["dwd_order_level", "alert_events"]
        cur = conn.execute("SELECT name FROM sqlite_master WHERE type='table'")
        existing_tables = {r[0] for r in cur.fetchall()}
        missing = [t for t in critical_tables if t not in existing_tables]
        if missing:
            _migration_status["status"] = "failed"
            _migration_status["failed"].extend(
                [
                    {"table": t, "migration": "startup_check", "error": "critical table missing"}
                    for t in missing
                ]
            )
            logger.critical(
                "Critical tables missing: %s. API will serve but data may be incomplete.",
                ", ".join(missing),
            )
    finally:
        conn.close()


_VALID_MIGRATION_TABLES = frozenset(
    {
        "event_outbox",
        "alert_events",
        "action_tasks",
        "review_retro",
    }
)


def _validate_identifier(name: str) -> None:
    """Validate SQL identifier against a known whitelist."""
    if name not in _VALID_MIGRATION_TABLES:
        raise ValueError(f"Unknown table name: {name!r}")


def _migrate_if_needed(conn, migration_dir, filename, table, columns):
    migration_file = os.path.join(migration_dir, filename)
    if not os.path.exists(migration_file):
        return

    _validate_identifier(table)
    cur = conn.execute(f"PRAGMA table_info({table})")
    existing = [r[1] for r in cur.fetchall()]
    if all(c in existing for c in columns):
        return

    with open(migration_file) as f:
        raw = f.read()

    for stmt in raw.split(";"):
        lines = [
            line.strip()
            for line in stmt.split("\n")
            if line.strip() and not line.strip().startswith("--")
        ]
        stmt_clean = "\n".join(lines).strip()
        if not stmt_clean:
            continue
        try:
            conn.execute(stmt_clean)
        except sqlite3.OperationalError as e:
            logger.warning("Migration failed for %s (%s): %s", table, filename, e)
            _migration_status["failed"].append(
                {
                    "table": table,
                    "migration": filename,
                    "error": str(e),
                }
            )
            _migration_status["status"] = "degraded"

    conn.commit()


def _ensure_qoder_tables() -> None:
    """Create Qoder tables (qoder_runs, qoder_reports) if they don't exist.

    Uses PRAGMA table_info check (same pattern as _migrate_if_needed).
    Called at startup since migrations only add columns, not create tables.
    """
    from core.config import DB_PATH

    if not os.path.exists(DB_PATH):
        logger.info("DB not found, skipping Qoder table creation")
        return

    conn = sqlite3.connect(DB_PATH)
    try:
        # ── qoder_runs ────────────────────────────────────────────────
        cur = conn.execute(
            "SELECT name FROM sqlite_master WHERE type='table' AND name='qoder_runs'"
        )
        if not cur.fetchone():
            conn.execute("""
                CREATE TABLE qoder_runs (
                    run_id TEXT PRIMARY KEY,
                    run_type TEXT NOT NULL,
                    mode TEXT NOT NULL DEFAULT 'read_only',
                    status TEXT NOT NULL,
                    started_at TEXT NOT NULL,
                    finished_at TEXT,
                    request_id TEXT,
                    actor TEXT DEFAULT 'qoder',
                    can_apply INTEGER DEFAULT 0,
                    error_message TEXT
                )
            """)
            logger.info("Created table: qoder_runs")

        # ── qoder_reports ─────────────────────────────────────────────
        cur = conn.execute(
            "SELECT name FROM sqlite_master WHERE type='table' AND name='qoder_reports'"
        )
        if not cur.fetchone():
            conn.execute("""
                CREATE TABLE qoder_reports (
                    report_id TEXT PRIMARY KEY,
                    run_id TEXT,
                    run_type TEXT NOT NULL,
                    summary TEXT NOT NULL,
                    findings_json TEXT,
                    recommended_human_actions_json TEXT,
                    risk_level TEXT,
                    used_endpoints_json TEXT,
                    no_apply_performed INTEGER NOT NULL DEFAULT 1,
                    business_side_effect INTEGER NOT NULL DEFAULT 0,
                    created_at TEXT NOT NULL,
                    request_id TEXT
                )
            """)
            logger.info("Created table: qoder_reports")

        conn.commit()
    finally:
        conn.close()


def create_app() -> FastAPI:
    setup_logging()

    # Parse ENABLE_DOCS as boolean: "0", "false", "no" → disabled; "1", "true", "yes" → enabled
    _enable_docs = os.environ.get("ENABLE_DOCS", "").lower() in ("1", "true", "yes")
    app = FastAPI(
        title="Olist Decision Backend API",
        version="0.6.0",
        docs_url="/docs" if _enable_docs else None,
        openapi_url="/openapi.json" if _enable_docs else None,
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
            request.method,
            request.url.path,
            response.status_code,
            elapsed,
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
        response.headers["Strict-Transport-Security"] = "max-age=31536000; includeSubDomains"
        response.headers["X-XSS-Protection"] = "1; mode=block"
        response.headers["Referrer-Policy"] = "strict-origin-when-cross-origin"
        path = request.url.path
        if (
            path.startswith("/docs")
            or path.startswith("/redoc")
            or path.startswith("/openapi.json")
        ):
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
    app.add_exception_handler(RequestValidationError, validation_exception_handler)

    @app.exception_handler(Exception)
    async def global_exception_handler(request: Request, exc: Exception):
        rid = get_request_id()
        # Dual-log to both "api" (general) and "api.error" (diagnosis service reads this)
        logging.getLogger("api").error(
            "Unhandled exception: %s",
            exc,
            extra={"request_id": rid, "path": request.url.path},
        )
        logging.getLogger("api.error").error(
            "Unhandled exception: %s",
            exc,
            extra={"request_id": rid, "path": request.url.path},
        )
        return JSONResponse(
            status_code=500,
            content={
                "request_id": rid,
                "error_code": "INTERNAL_ERROR",
                "message": "An unexpected error occurred",
                # Parse DEBUG as boolean: "0", "false", "no" → disabled; "1", "true", "yes" → enabled
                "diagnosis": _sanitize_error(str(exc))
                if os.environ.get("DEBUG", "").lower() in ("1", "true", "yes")
                else "Internal server error",
                "suggested_action": "Check server logs for details or contact the administrator",
            },
        )

    # ── Startup migration & table creation ───────────────────────────────
    from core.config import DB_PATH

    _apply_migration(DB_PATH)
    _ensure_qoder_tables()

    # ── Router mounting ──────────────────────────────────────────────────
    from api.routers import alerts, diagnosis, feishu, health, logs, outbox, pipeline, status, tasks

    app.include_router(health.router, prefix="/api/v1", tags=["Health"])
    app.include_router(status.router, prefix="/api/v1", tags=["Status"])
    app.include_router(alerts.router, prefix="/api/v1", tags=["Alerts"])
    app.include_router(tasks.router, prefix="/api/v1", tags=["Tasks"])
    app.include_router(outbox.router, prefix="/api/v1", tags=["Outbox"])
    app.include_router(logs.router, prefix="/api/v1", tags=["Logs"])
    app.include_router(feishu.router, prefix="/api/v1", tags=["Feishu"])
    app.include_router(pipeline.router, prefix="/api/v1", tags=["Pipeline"])
    app.include_router(diagnosis.router, prefix="/api/v1", tags=["Diagnosis"])
    from api.routers.governance import router as governance_router

    app.include_router(governance_router, prefix="/api/v1", tags=["Governance"])
    from api.routers.qoder import router as qoder_router

    app.include_router(qoder_router, prefix="/api/v1", tags=["Qoder"])

    # ── CORS ─────────────────────────────────────────────────────────────
    _cors_origins_raw = os.environ.get("CORS_ORIGINS", "http://localhost:5173").split(",")
    # Security: reject wildcard origins to prevent cross-origin attacks
    _cors_origins = []
    for origin in _cors_origins_raw:
        origin = origin.strip()
        if origin == "*":
            logger.warning("CORS_ORIGINS contains wildcard '*', which is insecure. Rejecting.")
            continue
        if origin:
            _cors_origins.append(origin)
    if not _cors_origins:
        _cors_origins = ["http://localhost:5173"]
        logger.warning("No valid CORS origins configured, defaulting to localhost")
    app.add_middleware(
        CORSMiddleware,
        allow_origins=_cors_origins,
        allow_methods=["GET", "POST", "OPTIONS"],
        allow_headers=["Authorization", "Content-Type", "X-Request-ID"],
    )

    # ── Frontend static serving (if built) ────────────────────────────────
    _frontend_dist = os.path.join(os.path.dirname(os.path.dirname(__file__)), "frontend", "dist")
    if os.path.exists(os.path.join(_frontend_dist, "index.html")):
        from fastapi.staticfiles import StaticFiles

        app.mount("/console", StaticFiles(directory=_frontend_dist, html=True), name="console")

    return app


app = create_app()
