"""Validate local CSV payloads before syncing to Feishu."""
import argparse
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import yaml
import pandas as pd

from scripts.config import (
    FEISHU_DIR,
    FEISHU_FIELD_MAPPING_FILE,
    FEISHU_BASE_SCHEMA_FILE,
    FEISHU_USER_MAPPING_FILE,
    CONFIG_DIR,
)

VALID_TABLES = ["daily_metrics", "alert_events", "strategy_recommendations", "action_tasks", "review_retro"]


def load_yaml(path):
    with open(path, "r", encoding="utf-8") as f:
        return yaml.safe_load(f)


def get_field_types(schema, table_id):
    result = {}
    for table in schema.get("tables", []):
        if table.get("table_id") == table_id:
            for field in table.get("fields", []):
                result[field["field_id"]] = field
    return result


def validate(table_id):
    csv_path = os.path.join(FEISHU_DIR, f"{table_id}_for_feishu.csv")
    schema = load_yaml(FEISHU_BASE_SCHEMA_FILE)
    field_types = get_field_types(schema, table_id)

    if not os.path.exists(csv_path):
        print(f"  ❌ CSV not found: {csv_path}")
        return False

    df = pd.read_csv(csv_path)
    if df.empty:
        print(f"  ⚠️  CSV is empty (headers only)")
        return True

    pk_map = {
        "daily_metrics": "simulated_date",
        "alert_events": "alert_id",
        "strategy_recommendations": "recommendation_id",
        "action_tasks": "task_id",
        "review_retro": "review_id",
    }
    pk = pk_map.get(table_id, "record_id")

    errors = []
    warnings = []

    # 1. Primary key non-null & unique
    if pk not in df.columns:
        errors.append(f"PrimaryKey '{pk}' not in CSV columns")
    else:
        null_pk = df[pk].isna().sum()
        if null_pk > 0:
            errors.append(f"PrimaryKey '{pk}' has {null_pk} null values")
        dupes = df[pk].duplicated().sum()
        if dupes > 0:
            errors.append(f"PrimaryKey '{pk}' has {dupes} duplicates")

    # 2. Columns match field_mapping
    mapping = load_yaml(FEISHU_FIELD_MAPPING_FILE)
    table_mapping = mapping.get(table_id, {})
    expected_cols = set(table_mapping.get("fields", {}).keys())

    missing_cols = expected_cols - set(df.columns) - {"owner_role"}  # owner_role maps to 负责人角色
    if missing_cols:
        warnings.append(f"Missing columns: {missing_cols}")

    # 3. Select field values
    select_fields = {
        "alert_events": {
            "severity": {"high", "medium", "low"},
            "status": {"new", "investigating", "strategy_generated", "task_created", "resolved", "ignored"},
        },
        "strategy_recommendations": {
            "risk_level": {"high", "medium", "low"},
            "approval_status": {"draft", "pending_review", "approved", "rejected", "executing", "completed", "invalidated"},
            "status": {"draft", "pending_review", "approved", "rejected", "executing", "completed", "invalidated"},
        },
        "action_tasks": {
            "priority": {"high", "medium", "low"},
            "status": {"todo", "in_progress", "blocked", "done", "cancelled"},
        },
    }
    for col, valid in select_fields.get(table_id, {}).items():
        if col in df.columns:
            invalid = set(df[col].dropna().unique()) - valid
            if invalid:
                errors.append(f"Select field '{col}' has invalid values: {invalid}")

    # 4. Date fields parseable
    date_cols = []
    for fid, fdef in field_types.items():
        if fdef.get("type") == "datetime":
            date_cols.append(fid)
    for col in date_cols:
        if col in df.columns:
            for _, row in df.iterrows():
                val = row.get(col)
                if pd.notna(val) and str(val).strip():
                    try:
                        pd.to_datetime(val)
                    except Exception:
                        errors.append(f"Date field '{col}' cannot parse value: '{val}'")
                        break

    # 5. Percentage fields (0-1 range)
    pct_fields = []
    for fid, fdef in field_types.items():
        if fdef.get("type") == "number" and fdef.get("format") == "percent":
            pct_fields.append(fid)
    for col in pct_fields:
        if col in df.columns:
            for _, row in df.iterrows():
                val = row.get(col)
                if pd.notna(val) and (val < -0.01 or val > 1.01):
                    warnings.append(f"Percentage field '{col}' value {val} may not be in 0-1 range")

    # 6. User fields
    user_fields = []
    for fid, fdef in field_types.items():
        if fdef.get("type") == "user":
            user_fields.append(fid)
    for col in user_fields:
        if col in df.columns:
            for _, row in df.iterrows():
                val = row.get(col)
                if pd.notna(val) and str(val).strip() and not str(val).startswith("ou_"):
                    warnings.append(f"User field '{col}' has non-user_id value: '{val}' (will be skipped during sync)")

    # Report
    if errors:
        print(f"  ❌ {len(errors)} errors:")
        for e in errors:
            print(f"    - {e}")
    if warnings:
        print(f"  ⚠️  {len(warnings)} warnings:")
        for w in warnings:
            print(f"    - {w}")
    if not errors and not warnings:
        print(f"  ✅ {len(df)} records passed all checks")

    print(f"  📊 Columns: {len(df.columns)}, PK: {pk} ({df[pk].nunique()} unique)")
    return len(errors) == 0


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--table", choices=VALID_TABLES, required=True)
    args = parser.parse_args()

    print(f"=== Validating payload: {args.table} ===")
    ok = validate(args.table)
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
