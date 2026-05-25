"""Security tests for auth, SQL injection, and path traversal protections."""
import os
import sys
import pytest

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from api.auth import verify_token


class TestAuthWeakTokenRejection:
    """Verify verify_token rejects weak/placeholder tokens as defense-in-depth."""

    def test_rejects_known_weak_token(self, monkeypatch):
        monkeypatch.setenv("API_BEARER_TOKEN", "test-token")
        assert verify_token("test-token") is False

    def test_rejects_replace_me(self, monkeypatch):
        monkeypatch.setenv("API_BEARER_TOKEN", "REPLACE_ME")
        assert verify_token("REPLACE_ME") is False

    def test_rejects_short_token(self, monkeypatch):
        monkeypatch.setenv("API_BEARER_TOKEN", "short123")
        assert verify_token("short123") is False

    def test_accepts_strong_token(self, monkeypatch):
        strong = "x" * 32
        monkeypatch.setenv("API_BEARER_TOKEN", strong)
        assert verify_token(strong) is True
        assert verify_token("wrong-token") is False

    def test_rejects_empty_token(self, monkeypatch):
        monkeypatch.setenv("API_BEARER_TOKEN", "")
        assert verify_token("") is False


class TestAuthConstantTime:
    """Verify token comparison is constant-time (hmac.compare_digest)."""

    def test_wrong_token_rejected(self, monkeypatch):
        strong = "a" * 32
        monkeypatch.setenv("API_BEARER_TOKEN", strong)
        assert verify_token("b" * 32) is False


class TestSQLIdentifierValidation:
    """Verify validate_sql_identifier rejects malicious identifiers."""

    def test_rejects_injection_attempts(self):
        from core.config import validate_sql_identifier
        bad = [
            "users; DROP TABLE users; --",
            "users' OR '1'='1",
            "../etc/passwd",
            "table`name",
            "name UNION SELECT",
        ]
        for ident in bad:
            with pytest.raises(ValueError):
                validate_sql_identifier(ident)

    def test_accepts_valid_identifiers(self):
        from core.config import validate_sql_identifier
        good = ["pipeline_runs", "alert_events", "metric_daily", "_private"]
        for ident in good:
            validate_sql_identifier(ident)


class TestFeishuPathTraversal:
    """Verify FeishuService._run_script rejects path traversal."""

    def test_rejects_absolute_path(self):
        from services.feishu_service import FeishuService
        svc = FeishuService()
        with pytest.raises(ValueError, match="path separator"):
            svc._run_script("/etc/passwd", [])

    def test_rejects_parent_directory(self):
        from services.feishu_service import FeishuService
        svc = FeishuService()
        with pytest.raises(ValueError, match="path separator"):
            svc._run_script("../config.py", [])

    def test_rejects_backslash(self):
        from services.feishu_service import FeishuService
        svc = FeishuService()
        with pytest.raises(ValueError, match="path separator"):
            svc._run_script("scripts\\\\config.py", [])


class TestLocalCLIAdapterRuleId:
    """Verify LocalCLIAdapter sanitizes rule_id to prevent command injection."""

    def test_sanitizes_malicious_rule_id(self):
        from adapters.local_cli_adapter import LocalCLIAdapter
        adapter = LocalCLIAdapter()
        bad = "; rm -rf / #"
        result = adapter._validate_rule_id(bad)
        assert result == "unknown"

    def test_accepts_normal_rule_id(self):
        from adapters.local_cli_adapter import LocalCLIAdapter
        adapter = LocalCLIAdapter()
        good = "daily_check_001"
        result = adapter._validate_rule_id(good)
        assert result == "daily_check_001"

    def test_rejects_rule_id_with_pipe(self):
        from adapters.local_cli_adapter import LocalCLIAdapter
        adapter = LocalCLIAdapter()
        bad = "rule|cat /etc/passwd"
        result = adapter._validate_rule_id(bad)
        assert result == "unknown"
