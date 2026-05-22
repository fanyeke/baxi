import sys
import os
import pytest


@pytest.fixture(scope="session")
def project_root():
    """Return the absolute path to the project root directory."""
    return os.path.dirname(os.path.dirname(os.path.abspath(__file__)))


@pytest.fixture(scope="session")
def data_dir(project_root):
    return os.path.join(project_root, "data")


@pytest.fixture(autouse=True)
def setup_path(project_root):
    """Ensure scripts/ is importable."""
    sys.path.insert(0, project_root)
    os.environ.setdefault("API_BEARER_TOKEN", "test-token")


@pytest.fixture(scope="function")
def auth_headers():
    return {"Authorization": "Bearer test-token"}
