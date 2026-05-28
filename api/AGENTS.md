# api/ — FastAPI Gateway (port 8765)

## OVERVIEW
FastAPI gateway on port 8765, SQLite backend, bearer token auth, 11 route groups.

## STRUCTURE

**Core files**: `main.py` (app factory), `auth.py` (constant-time HMAC token check), `errors.py` (`APIError` + unified JSON handlers), `schemas.py`/`schemas_qoder.py` (Pydantic v2 models), `dependencies.py` (DI: `get_db`, `get_current_user`), `logging_config.py` (JSON logging to `logs/api/`).

**routers/** (11): `health` (unauthenticated `GET /health`), `status` (system/migration state), `alerts`/`tasks` (query alerts/tasks), `outbox` (event list + `POST /dispatch`), `feishu` (export/sync/status import), `pipeline` (`POST /pipeline/run`), `logs` (read JSONL: errors/audit/recent), `diagnosis` (cross-source tracing), `governance` (catalog, classification, lineage, 7 endpoints), `qoder` (AI capabilities, context, reports).

## KEY PATTERNS
- **App factory**: `create_app()` returns configured FastAPI instance; launched by uvicorn via `scripts/run_api.py`
- **Auth**: All routers except health use `Depends(get_current_user)` on the APIRouter, not per-endpoint
- **DI**: `get_db` yields one SQLite connection per request; `get_current_user` extracts bearer token from Authorization header
- **Pydantic v2**: `BaseModel` + `ConfigDict` + `field_validator` for input bounds checking
- **Unified errors**: `APIError` returns `{"error_code","message","diagnosis","suggested_action","request_id"}` via JSONResponse

## CONVENTIONS
- Router files export `router = APIRouter(...)` as module-level variable, imported via `app.include_router()` in main.py
- Response models imported from schemas.py; routers never construct raw dicts
- Error responses always carry `request_id` from ContextVar set by request middleware

## ANTI-PATTERNS
- **Flat package**: `api/__init__.py` is empty — no namespace; `from api import` works only because directory is in sys.path
- **f-string SQL**: ~19 locations construct raw SQL with f-strings instead of parameterized queries (whitelisted but fragile)
- **Coverage blind spot**: `source=api` in pyproject.toml misses routers/ by default — router logic stays untested unless explicitly added
