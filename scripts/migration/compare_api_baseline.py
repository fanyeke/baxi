#!/usr/bin/env python3
"""Compare Go API responses against Phase 0 baseline snapshots.

Makes HTTP requests to all Go API endpoints and compares responses against
baseline JSON files in migration_baseline/api_responses/. Uses semantic
(field-order-agnostic) comparison and tolerates known Phase 3 parity
differences (36 vs 37 item counts).

Usage:
    python3 scripts/migration/compare_api_baseline.py

Environment variables:
    API_BASE_URL      Base URL for the Go API (default: http://localhost:8080)
    API_BEARER_TOKEN  Bearer token for authenticated endpoints
"""

import json
import os
import sys
import urllib.request
from urllib.error import HTTPError, URLError

BASE_URL = os.environ.get("API_BASE_URL", "http://localhost:8080")
API_TOKEN = os.environ.get("API_BEARER_TOKEN", "")
BASELINE_DIR = os.path.join(
    os.path.dirname(__file__), "..", "..", "migration_baseline", "api_responses"
)

ENDPOINTS = [
    ("health",            "/api/v1/health",            "health.json",            {"auth": False}),
    ("status",            "/api/v1/status",            "status.json",            {"auth": True}),
    ("alerts",            "/api/v1/alerts",            "alerts.json",            {"auth": True}),
    ("tasks",             "/api/v1/tasks",             "tasks.json",             {"auth": True}),
    ("outbox",            "/api/v1/outbox",            "outbox.json",            {"auth": True}),
    ("governance/status", "/api/v1/governance/status", "governance_status.json", {"auth": True}),
    ("qoder/context",     "/api/v1/qoder/context",     "qoder_context.json",     {"auth": True}),
    # Log endpoints — no baseline snapshots, just verify HTTP 200
    ("logs/recent",       "/api/v1/logs/recent",       None,                     {"auth": True}),
    ("logs/errors",       "/api/v1/logs/errors",       None,                     {"auth": True}),
    ("logs/audit",        "/api/v1/logs/audit",        None,                     {"auth": True}),
]

# Phase 3 pipeline adds ~1 extra alert/task/outbox; tolerate ±N difference
ACCEPTED_TOTAL_DIFFS = {
    "alerts": 1,
    "tasks": 1,
    "outbox": 1,
}

# Keys whose concrete value is never compared (just type-checked)
VALUE_IGNORED_KEYS = {"request_id"}

# Keys expected to hold RFC 3339 timestamps — accept any non-empty string
TIMESTAMP_KEYS = {
    "created_at", "updated_at", "started_at", "finished_at", "due_at",
    "completed_at", "event_date", "last_dispatch_at",
}

# --- JSON structural comparison ---


def _check_structure(baseline, actual, path="", fails=None):
    """Recursively verify that all baseline keys exist with matching types.

    Field order is ignored. Only key-existence and type-correctness are
    enforced — concrete scalar values (beyond type) are not compared.
    """
    if fails is None:
        fails = []

    if isinstance(baseline, dict):
        if not isinstance(actual, dict):
            fails.append(f"Expected dict at '{path}', got {type(actual).__name__}")
            return fails
        for key in baseline:
            cur = f"{path}.{key}" if path else key
            if key not in actual:
                if key not in VALUE_IGNORED_KEYS and key not in TIMESTAMP_KEYS:
                    fails.append(f"Missing key: '{cur}'")
                continue
            _check_structure(baseline[key], actual[key], cur, fails)

    elif isinstance(baseline, list):
        if not isinstance(actual, list):
            fails.append(f"Expected list at '{path}', got {type(actual).__name__}")
            return fails
        # Compare structure using first element (both non-empty)
        if baseline and actual:
            _check_structure(baseline[0], actual[0], f"{path}[0]", fails)

    else:
        # Scalar — assert type match (baseline values are the reference type)
        bl_type = type(baseline).__name__
        ac_type = type(actual).__name__
        if bl_type != ac_type:
            fails.append(f"Type mismatch at '{path}': expected {bl_type}, got {ac_type}")

    return fails


# --- Helpers ---


def load_baseline(name):
    path = os.path.join(BASELINE_DIR, name)
    with open(path) as f:
        return json.load(f)


def make_request(url, use_auth):
    headers = {}
    if use_auth and API_TOKEN:
        headers["Authorization"] = f"Bearer {API_TOKEN}"
    req = urllib.request.Request(url, headers=headers)
    with urllib.request.urlopen(req, timeout=10) as resp:
        return resp.status, json.loads(resp.read().decode())


# --- Per-endpoint check ---


def check_endpoint(name, path, baseline_file, use_auth):
    url = f"{BASE_URL}{path}"
    fails = []
    warns = []

    # 1. HTTP request
    try:
        status_code, data = make_request(url, use_auth)
    except HTTPError as e:
        msg = f"HTTP {e.code}: {e.reason}"
        return "FAIL", [msg]
    except URLError as e:
        msg = f"Connection failed: {e.reason}"
        return "FAIL", [msg]
    except json.JSONDecodeError:
        return "FAIL", ["Response is not valid JSON"]
    except Exception as e:
        return "FAIL", [str(e)]

    print(f"    HTTP {status_code}")

    if status_code != 200:
        return "FAIL", [f"Expected 200, got {status_code}"]

    # 2. No baseline — HTTP 200 is sufficient
    if baseline_file is None:
        return "PASS", []

    baseline = load_baseline(baseline_file)

    # 3. Structural comparison (keys + types, order-agnostic)
    structural = _check_structure(baseline, data)
    fails.extend(structural)

    # 4. Item count / total comparison with tolerance
    for field in ("items", "total"):
        if field not in baseline or field not in data:
            continue

        bl_val = baseline[field] if isinstance(baseline[field], int) else len(baseline[field])
        resp_val = data[field] if isinstance(data[field], int) else len(data[field])

        if bl_val == resp_val:
            continue

        allowed = ACCEPTED_TOTAL_DIFFS.get(name, 0)
        diff = abs(resp_val - bl_val)

        label = f"{field} count: baseline {bl_val}, got {resp_val}"
        if diff <= allowed:
            warns.append(f"{label} (within ±{allowed} tolerance)")
        else:
            fails.append(f"{label} (exceeds ±{allowed} tolerance)")

    # 5. Report
    for f in fails:
        print(f"    FAIL: {f}")
    for w in warns:
        print(f"    WARN: {w}")

    if fails:
        return "FAIL", fails + warns
    if warns:
        return "WARN", warns
    print(f"    → PASS")
    return "PASS", []


# --- Main ---


def main():
    pass_count = 0
    warn_count = 0
    fail_count = 0

    print("=" * 60)
    print("  API Baseline Comparison")
    print("=" * 60)
    print(f"  Base URL:    {BASE_URL}")
    print(f"  Baseline:    {BASELINE_DIR}")
    print(f"  Auth token:  {'✓ set' if API_TOKEN else '✗ NONE (public endpoints only)'}")
    print("=" * 60)

    for name, path, baseline_file, opts in ENDPOINTS:
        label = f"[{name}]"
        print(f"\n  {label} GET {path}")
        status, issues = check_endpoint(name, path, baseline_file, opts.get("auth", False))
        if status == "PASS":
            pass_count += 1
        elif status == "WARN":
            warn_count += 1
        else:
            fail_count += 1

    print(f"\n{'=' * 60}")
    print(f"  SUMMARY: {pass_count} PASS, {warn_count} WARN, {fail_count} FAIL")
    print(f"{'=' * 60}")

    sys.exit(0 if fail_count == 0 else 1)


if __name__ == "__main__":
    main()
