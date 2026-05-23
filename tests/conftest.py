import sys
import os
import pytest

# Ensure project root and scripts/ are importable (needed for api/routers/* imports)
_test_root = os.path.dirname(os.path.abspath(__file__))
_project_root = os.path.dirname(_test_root)
sys.path.insert(0, _project_root)
sys.path.insert(0, os.path.join(_project_root, "scripts"))
os.environ.setdefault("API_BEARER_TOKEN", "test-token")


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
    os.environ.setdefault("API_BEARER_TOKEN", "test-token")


@pytest.fixture(scope="function")
def auth_headers():
    return {"Authorization": "Bearer test-token"}
