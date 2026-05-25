"""Bearer Token authentication — constant-time token comparison."""

import hmac
import logging
import os

_KNOWN_WEAK_TOKENS = frozenset({
    "REPLACE_ME", "your-secret-token-here", "test-token",
    "changeme", "admin", "password", "secret",
    "sk-your-key-here",
})

logger = logging.getLogger("api.auth")


def verify_token(token: str) -> bool:
    """Compare the provided token against API_BEARER_TOKEN env var.

    Also rejects known placeholder / example tokens as defense-in-depth
    (the run_api.py launcher enforces this at startup, but the auth layer
    independently guards against accidental deployment with weak tokens).
    """
    expected = os.environ.get("API_BEARER_TOKEN", "")
    if not expected:
        logger.warning("API_BEARER_TOKEN is not set — all requests will be rejected")
        return False
    if expected.strip() in _KNOWN_WEAK_TOKENS:
        logger.warning(
            "API_BEARER_TOKEN is set to a known weak/placeholder value — rejecting"
        )
        return False
    if len(expected) < 32:
        logger.warning("API_BEARER_TOKEN is shorter than 32 characters — rejecting")
        return False
    return hmac.compare_digest(token, expected)
