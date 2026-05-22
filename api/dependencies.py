"""FastAPI dependencies — database connections and auth."""

import sqlite3
from typing import Generator

from fastapi import Header, HTTPException

from api.auth import verify_token
from api.errors import APIError, AUTH_REQUIRED, INVALID_TOKEN
from services.db_service import get_db as _svc_get_db


def get_db() -> Generator[sqlite3.Connection, None, None]:
    """Yield a SQLite connection, closing it on teardown."""
    conn = _svc_get_db()
    try:
        yield conn
    finally:
        conn.close()


def get_current_user(authorization: str = Header(None)) -> str:
    """Extract Bearer token, verify, return actor name.

    Raises APIError with 401 if token is missing or invalid.
    """
    if not authorization:
        raise APIError(
            error_code=AUTH_REQUIRED,
            message="Authorization header is required",
            diagnosis="No Bearer token provided",
            suggested_action="Add 'Authorization: Bearer <token>' header",
            http_status=401,
        )

    scheme, _, token = authorization.partition(" ")
    if scheme.lower() != "bearer" or not token:
        raise APIError(
            error_code=AUTH_REQUIRED,
            message="Invalid Authorization header format",
            diagnosis="Expected 'Bearer <token>'",
            suggested_action="Use format: Authorization: Bearer <token>",
            http_status=401,
        )

    if not verify_token(token):
        raise APIError(
            error_code=INVALID_TOKEN,
            message="Invalid or expired token",
            diagnosis="The provided Bearer token does not match API_BEARER_TOKEN",
            suggested_action="Check your API_BEARER_TOKEN configuration",
            http_status=403,
        )

    return "qoder"
