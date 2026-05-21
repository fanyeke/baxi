"""
Generate 5 local Feishu sandbox CSVs from existing project data.

Reads config/feishu_base_schema.yml and config/feishu_field_mapping.yml,
transforms source data to match Feishu schema columns, and outputs CSVs
to data/feishu/.

First 2 CSVs will have data if source files exist; remaining 3 will have
correct headers (empty or from available source data).
"""

import sys
import os

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from scripts.config import (
    FEISHU_BASE_SCHEMA_FILE,
    FEISHU_FIELD_MAPPING_FILE,
    DAILY_METRICS_FILE,
    METRIC_ALERTS_FILE,
    AIP_ACTION_RECOMMENDATIONS_FILE,
    OUTPUTS_DIR,
    FEISHU_DIR,
)

import yaml
import pandas as pd
from datetime import datetime


def load_yaml(path):
    with open(path, 'r', encoding='utf-8') as f:
        return yaml.safe_load(f)


def get_schema_fields(schema, table_id):
    """Return ordered list of field_id for a table from the schema."""
    for table in schema['tables']:
        if table['table_id'] == table_id:
            return [f['field_id'] for f in table['fields']]
    return []


def make_empty_csv(path, field_ids):
    """Write an empty CSV with the specified column headers."""
    df = pd.DataFrame(columns=field_ids)
    df.to_csv(path, index=False)
    return 0


def transform_daily_metrics(schema, field_mapping):
    """Transform daily_metrics.csv to Feishu schema format."""
    field_ids = get_schema_fields(schema, 'daily_metrics')
    output_path = os.path.join(FEISHU_DIR, 'daily_metrics_for_feishu.csv')

    if not os.path.exists(DAILY_METRICS_FILE):
        return make_empty_csv(output_path, field_ids)

    df = pd.read_csv(DAILY_METRICS_FILE)

    # Select only fields defined in schema that exist in source
    available = [f for f in field_ids if f in df.columns]
    out = df[available].copy()

    # Ensure all schema fields exist (fill missing with empty)
    for f in field_ids:
        if f not in out.columns:
            out[f] = None

    out = out[field_ids]
    out.to_csv(output_path, index=False)
    return len(out)


def transform_metric_alerts(schema, field_mapping):
    """Transform metric_alerts.csv to Feishu alert_events schema format."""
    field_ids = get_schema_fields(schema, 'alert_events')
    output_path = os.path.join(FEISHU_DIR, 'metric_alerts_for_feishu.csv')

    if not os.path.exists(METRIC_ALERTS_FILE):
        return make_empty_csv(output_path, field_ids)

    df = pd.read_csv(METRIC_ALERTS_FILE)

    # Column name mapping: source -> schema
    col_map = {
        'metric': 'metric_name',
        'dimension': 'object_type',
    }

    for src, tgt in col_map.items():
        if src in df.columns and tgt in field_ids:
            if tgt in df.columns and src != tgt:
                df = df.drop(columns=[tgt])
            df = df.rename(columns={src: tgt})

    available = [f for f in field_ids if f in df.columns]
    out = df[available].copy()

    for f in field_ids:
        if f not in out.columns:
            out[f] = None

    out = out[field_ids]
    out.to_csv(output_path, index=False)
    return len(out)


def transform_strategy_recommendations(schema, field_mapping):
    """Transform aip_action_recommendations.json to Feishu recommendations schema."""
    field_ids = get_schema_fields(schema, 'recommendations')
    output_path = os.path.join(FEISHU_DIR, 'strategy_recommendations_for_feishu.csv')

    wake_source = os.path.join(OUTPUTS_DIR, 'wake', 'strategy_recommendations.json')
    source = wake_source if os.path.exists(wake_source) else AIP_ACTION_RECOMMENDATIONS_FILE

    if not os.path.exists(source):
        return make_empty_csv(output_path, field_ids)

    import json
    with open(source, 'r', encoding='utf-8') as f:
        data = json.load(f)

    if isinstance(data, list):
        df = pd.DataFrame(data)
    elif isinstance(data, dict) and 'recommendations' in data:
        df = pd.DataFrame(data['recommendations'])
    else:
        return make_empty_csv(output_path, field_ids)

    if df.empty:
        return make_empty_csv(output_path, field_ids)

    # Map source fields to schema fields
    col_map = {
        'owner_role': 'owner',
    }
    for src, tgt in col_map.items():
        if src in df.columns and tgt in field_ids:
            df = df.rename(columns={src: tgt})

    available = [f for f in field_ids if f in df.columns]
    out = df[available].copy()

    for f in field_ids:
        if f not in out.columns:
            out[f] = None

    out = out[field_ids]
    out.to_csv(output_path, index=False)
    return len(out)


def transform_action_tasks(schema, field_mapping):
    """Create action_tasks CSV with correct headers from schema."""
    field_ids = get_schema_fields(schema, 'action_tasks')
    output_path = os.path.join(FEISHU_DIR, 'action_tasks_for_feishu.csv')

    source_path = os.path.join(OUTPUTS_DIR, 'wake', 'action_tasks.json')
    if not os.path.exists(source_path):
        return make_empty_csv(output_path, field_ids)

    import json
    with open(source_path, 'r', encoding='utf-8') as f:
        data = json.load(f)

    if isinstance(data, list):
        df = pd.DataFrame(data)
    elif isinstance(data, dict) and 'tasks' in data:
        df = pd.DataFrame(data['tasks'])
    else:
        return make_empty_csv(output_path, field_ids)

    if df.empty:
        return make_empty_csv(output_path, field_ids)

    col_map = {
        'feedback': 'feedback',
        'owner_role': 'owner',
    }
    for src, tgt in col_map.items():
        if src in df.columns and tgt in field_ids:
            df = df.rename(columns={src: tgt})

    available = [f for f in field_ids if f in df.columns]
    out = df[available].copy()

    for f in field_ids:
        if f not in out.columns:
            out[f] = None

    out = out[field_ids]
    out.to_csv(output_path, index=False)
    return len(out)


def transform_execution_reviews(schema, field_mapping):
    """Create execution_reviews CSV with correct headers from schema."""
    field_ids = get_schema_fields(schema, 'review_retro')
    output_path = os.path.join(FEISHU_DIR, 'execution_reviews_for_feishu.csv')
    return make_empty_csv(output_path, field_ids)


def main():
    os.makedirs(FEISHU_DIR, exist_ok=True)

    schema = load_yaml(FEISHU_BASE_SCHEMA_FILE)
    field_mapping = load_yaml(FEISHU_FIELD_MAPPING_FILE)

    generated_at = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
    print(f"[{generated_at}] Feishu sandbox CSV generator")
    print(f"  Schema: {FEISHU_BASE_SCHEMA_FILE}")
    print(f"  Mapping: {FEISHU_FIELD_MAPPING_FILE}")
    print(f"  Output: {FEISHU_DIR}/")
    print()

    results = {}

    # 1. daily_metrics
    count = transform_daily_metrics(schema, field_mapping)
    path = 'daily_metrics_for_feishu.csv'
    results[path] = count
    print(f"  [{count:>6} rows] {path}")

    # 2. metric_alerts
    count = transform_metric_alerts(schema, field_mapping)
    path = 'metric_alerts_for_feishu.csv'
    results[path] = count
    print(f"  [{count:>6} rows] {path}")

    # 3. strategy_recommendations
    count = transform_strategy_recommendations(schema, field_mapping)
    path = 'strategy_recommendations_for_feishu.csv'
    results[path] = count
    print(f"  [{count:>6} rows] {path}")

    # 4. action_tasks
    count = transform_action_tasks(schema, field_mapping)
    path = 'action_tasks_for_feishu.csv'
    results[path] = count
    print(f"  [{count:>6} rows] {path}")

    # 5. execution_reviews
    count = transform_execution_reviews(schema, field_mapping)
    path = 'execution_reviews_for_feishu.csv'
    results[path] = count
    print(f"  [{count:>6} rows] {path}")

    print()
    total = sum(results.values())
    print(f"Done. {len(results)} files, {total} total rows in {FEISHU_DIR}/")


if __name__ == '__main__':
    main()
