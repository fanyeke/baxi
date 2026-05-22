"""Bearer Token authentication — single static token comparison."""

import os


def verify_token(token: str) -> bool:
    """Compare the provided token against API_BEARER_TOKEN env var."""
    expected = os.environ.get("API_BEARER_TOKEN", "")
    if not expected:
        return False
    return token == expected
