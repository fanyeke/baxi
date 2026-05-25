"""Tests for pyproject.toml dependency consolidation."""
import importlib
import subprocess
import tomllib
from pathlib import Path

PROJECT_ROOT = Path(__file__).parent.parent
PYPROJECT_PATH = PROJECT_ROOT / "pyproject.toml"
REQUIREMENTS_PATH = PROJECT_ROOT / "requirements.txt"
REQUIREMENTS_DEV_PATH = PROJECT_ROOT / "requirements-dev.txt"


class TestDependencyConsolidation:
    """Ensure pyproject.toml is the single source of truth for dependencies."""

    def test_pyproject_has_dependencies(self):
        """pyproject.toml must have a [project.dependencies] section."""
        with open(PYPROJECT_PATH, "rb") as f:
            data = tomllib.load(f)
        assert "dependencies" in data["project"], "Missing [project.dependencies] in pyproject.toml"
        assert len(data["project"]["dependencies"]) >= 10, "Too few dependencies declared"

    def test_requirements_txt_exists(self):
        """requirements.txt should exist as a pip-tools lock file."""
        assert REQUIREMENTS_PATH.exists(), "requirements.txt missing — run pip-compile"

    def test_requirements_dev_txt_exists(self):
        assert REQUIREMENTS_DEV_PATH.exists(), "requirements-dev.txt missing — run pip-compile --extra dev"

    def test_runtime_imports_work(self):
        """All runtime dependencies must be importable."""
        runtime_packages = [
            "fastapi", "starlette", "httpx", "uvicorn",
            "pydantic", "pandas", "numpy", "requests",
            "yaml",  # PyYAML
            "openai",
        ]
        missing = []
        for pkg in runtime_packages:
            try:
                importlib.import_module(pkg)
            except ImportError:
                missing.append(pkg)
        assert not missing, f"Failed to import: {missing}"

    def test_test_imports_work(self):
        """Test dependencies must be importable."""
        import pytest  # noqa: F401
        import pytest_cov  # noqa: F401
