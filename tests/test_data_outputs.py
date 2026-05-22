import os
import json
import pandas as pd
import pytest


def test_daily_metrics_exists(project_root):
    path = os.path.join(project_root, "data/ads/daily_metrics_full.csv")
    assert os.path.exists(path), f"Missing: {path}"


def test_daily_metrics_pk_unique(project_root):
    df = pd.read_csv(os.path.join(project_root, "data/ads/daily_metrics_full.csv"))
    assert df["simulated_date"].is_unique, "simulated_date is not unique"


def test_daily_metrics_row_count(project_root):
    df = pd.read_csv(os.path.join(project_root, "data/ads/daily_metrics_full.csv"))
    assert 600 <= len(df) <= 650, f"Expected 600-650 rows, got {len(df)}"


def test_daily_metrics_has_alert_count(project_root):
    df = pd.read_csv(os.path.join(project_root, "data/ads/daily_metrics_full.csv"))
    assert "alert_count" in df.columns, "Missing alert_count column"


def test_metric_alerts_pk_unique(project_root):
    path = os.path.join(project_root, "data/ads/metric_alerts_full.csv")
    if os.path.exists(path):
        df = pd.read_csv(path)
        assert df["alert_id"].is_unique, "alert_id is not unique"


def test_context_bundle_exists(project_root):
    path = os.path.join(project_root, "data/aip/aip_context_bundle_full.json")
    assert os.path.exists(path), f"Missing: {path}"


def test_context_bundle_sections(project_root):
    with open(os.path.join(project_root, "data/aip/aip_context_bundle_full.json")) as f:
        bundle = json.load(f)
    # Full-mode bundle structure: mode, generated_at, date_range, total_days,
    # monthly_snapshots, full_range_summary, top_alerts, alert_statistics,
    # allowed_actions, owner_mapping
    required_keys = ["allowed_actions", "monthly_snapshots", "full_range_summary", "top_alerts"]
    for key in required_keys:
        assert key in bundle, f"Missing section: {key}"


def test_context_bundle_monthly_snapshots(project_root):
    with open(os.path.join(project_root, "data/aip/aip_context_bundle_full.json")) as f:
        bundle = json.load(f)
    snapshots = bundle.get("monthly_snapshots", [])
    assert len(snapshots) >= 24, f"Expected ≥24 monthly snapshots, got {len(snapshots)}"
