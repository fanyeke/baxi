"""Bearer Token authentication — constant-time token comparison."""

import hmac
import os


def verify_token(token: str) -> bool:
    """Compare the provided token against API_BEARER_TOKEN env var."""
    expected = os.environ.get("API_BEARER_TOKEN", "")
    if not expected:
        return False
    return hmac.compare_digest(token, expected)
