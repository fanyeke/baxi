import sys
import os
import sqlite3
import tempfile
from pathlib import Path

import pytest

# Ensure project root and scripts/ are importable (needed for api/routers/* imports)
_test_root = os.path.dirname(os.path.abspath(__file__))
_project_root = os.path.dirname(_test_root)
sys.path.insert(0, _project_root)
sys.path.insert(0, os.path.join(_project_root, "scripts"))
os.environ.setdefault("API_BEARER_TOKEN", "test-token-for-baxi-ci-tests-only-32ch")


@pytest.fixture(scope="session")
def project_root():
    return _project_root


@pytest.fixture(scope="session")
def data_dir(project_root):
    return os.path.join(project_root, "data")


@pytest.fixture(autouse=True)
def setup_path(project_root):
    """Ensure scripts/ is importable."""
    sys.path.insert(0, project_root)
    os.environ.setdefault("API_BEARER_TOKEN", "test-token-for-baxi-ci-tests-only-32ch")


@pytest.fixture(scope="function")
def auth_headers():
    return {"Authorization": "Bearer test-token-for-baxi-ci-tests-only-32ch"}


@pytest.fixture
def in_memory_db():
    """Create a fresh in-memory SQLite DB with schema applied."""
    conn = sqlite3.connect(":memory:")
    schema = Path(_project_root) / "sql" / "schema.sql"
    if schema.exists():
        conn.executescript(schema.read_text())
    conn.execute("PRAGMA foreign_keys = ON")
    yield conn
    conn.close()


@pytest.fixture
def temp_db_path():
    """Create a temp file-based SQLite DB with schema applied, yield path."""
    fd, path = tempfile.mkstemp(suffix=".db")
    os.close(fd)
    conn = sqlite3.connect(path)
    schema = Path(_project_root) / "sql" / "schema.sql"
    if schema.exists():
        conn.executescript(schema.read_text())
    conn.execute("PRAGMA foreign_keys = ON")
    conn.commit()
    conn.close()
    yield path
    os.unlink(path)
