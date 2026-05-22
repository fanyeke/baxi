import os
import pandas as pd
import pytest


FEISHU_FILES = [
    "daily_metrics_for_feishu_full.csv",
    "alert_events_for_feishu_full.csv",
    "strategy_recommendations_for_feishu_full.csv",
    "action_tasks_for_feishu_full.csv",
    "execution_reviews_for_feishu_full.csv",
]


def test_feishu_csv_count(project_root):
    feishu_dir = os.path.join(project_root, "data/feishu")
    existing = [f for f in FEISHU_FILES if os.path.exists(os.path.join(feishu_dir, f))]
    assert len(existing) == 5, f"Expected 5 _full CSVs, found {len(existing)}: {existing}"


def test_feishu_csv_nonempty(project_root):
    feishu_dir = os.path.join(project_root, "data/feishu")
    nonempty = 0
    for f in FEISHU_FILES:
        path = os.path.join(feishu_dir, f)
        if os.path.exists(path):
            df = pd.read_csv(path)
            if len(df) > 0:
                nonempty += 1
    assert nonempty >= 4, f"Expected ≥4 non-empty _full CSVs, got {nonempty}"


def test_execution_reviews_nonempty(project_root):
    path = os.path.join(project_root, "data/feishu/execution_reviews_for_feishu_full.csv")
    if os.path.exists(path):
        df = pd.read_csv(path)
        assert len(df) >= 3, f"Expected ≥3 execution_reviews rows, got {len(df)}"


def test_feishu_column_no_duplicates(project_root):
    feishu_dir = os.path.join(project_root, "data/feishu")
    for f in FEISHU_FILES:
        path = os.path.join(feishu_dir, f)
        if os.path.exists(path):
            df = pd.read_csv(path)
            cols = list(df.columns)
            dupes = [c for c in cols if cols.count(c) > 1]
            assert len(dupes) == 0, f"Duplicate columns in {f}: {dupes}"


def test_feishu_daily_metrics_has_required_fields(project_root):
    path = os.path.join(project_root, "data/feishu/daily_metrics_for_feishu_full.csv")
    if os.path.exists(path):
        df = pd.read_csv(path)
        required = ["GMV", "orders", "score"]
        found = any(r.lower() in " ".join(df.columns).lower() for r in required)
        assert found, f"None of {required} found in column names"
