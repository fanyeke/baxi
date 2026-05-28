#!/usr/bin/env python3
"""Capture API response snapshots using FastAPI TestClient."""
import json, os, secrets, sys

# Add project root to path
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))

# Set temp auth token BEFORE importing app (it validates on import)
os.environ["API_BEARER_TOKEN"] = secrets.token_urlsafe(32)
os.environ.setdefault("DEFAULT_USER", "freeze-agent")

from fastapi.testclient import TestClient
from api.main import create_app

OUT_DIR = os.path.join(os.path.dirname(__file__), "../../migration_baseline/api_responses")
os.makedirs(OUT_DIR, exist_ok=True)

app = create_app()
client = TestClient(app)
headers = {"Authorization": f"Bearer {os.environ['API_BEARER_TOKEN']}"}

endpoints = [
    ("GET", "/api/v1/health", None, "health.json"),
    ("GET", "/api/v1/status", headers, "status.json"),
    ("GET", "/api/v1/alerts", headers, "alerts.json"),
    ("GET", "/api/v1/tasks", headers, "tasks.json"),
    ("GET", "/api/v1/outbox", headers, "outbox.json"),
    ("GET", "/api/v1/governance/status", headers, "governance_status.json"),
    ("GET", "/api/v1/qoder/context", headers, "qoder_context.json"),
]

for method, path, hdrs, filename in endpoints:
    if method == "GET":
        resp = client.get(path, headers=hdrs or {})
    outpath = os.path.join(OUT_DIR, filename)
    try:
        data = resp.json()
    except Exception:
        data = {"raw_body": resp.text, "status_code": resp.status_code}
    with open(outpath, "w") as f:
        json.dump(data, f, indent=2, default=str)
    print(f"OK {resp.status_code} {path} -> {filename}")

print(f"API responses captured to {OUT_DIR}")
