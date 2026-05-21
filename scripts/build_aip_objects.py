#!/usr/bin/env python3
"""Build AIP object layer JSON files from intermediate base tables.

Reads config/aip_object_schema.yml and intermediate tables to produce:
- data/aip/aip_business_objects.json
- data/aip/aip_metrics.json
- data/aip/aip_events.json
- data/aip/aip_action_recommendations.json
"""

import sys
import os
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
sys.setrecursionlimit(10000)

import json
import yaml
import numpy as np
import pandas as pd

from scripts.config import (
    AIP_OBJECT_SCHEMA_FILE,
    AIP_BUSINESS_OBJECTS_FILE,
    AIP_METRICS_FILE,
    AIP_EVENTS_FILE,
    AIP_ACTION_RECOMMENDATIONS_FILE,
    AIP_DIR,
    ITEM_LEVEL_BASE_FILE,
    ORDER_LEVEL_BASE_FILE,
    DAILY_METRICS_FILE,
    METRIC_ALERTS_FILE,
)

ACTION_MAP = {
    "late_delivery_spike": "改善配送时效，目标延迟率降至<5%",
    "review_score_drop": "调查品类质量问题",
    "gmv_drop": "分析订单下降原因",
    "cancel_rate_spike": "调查取消率上升原因",
    "seller_activation_gap": "提升卖家激活率",
}


def sanitize_for_json(obj):
    """Convert NaN/NaT/inf to JSON-safe values."""

    if isinstance(obj, dict):
        return {k: sanitize_for_json(v) for k, v in obj.items()}
    elif isinstance(obj, list):
        return [sanitize_for_json(v) for v in obj]
    elif isinstance(obj, float):
        if np.isnan(obj) or np.isinf(obj):
            return None
        return obj
    elif isinstance(obj, (np.integer,)):
        return int(obj)
    elif isinstance(obj, (np.floating,)):
        if np.isnan(obj) or np.isinf(obj):
            return None
        return float(obj)
    elif isinstance(obj, (pd.Timestamp,)):
        return str(obj)
    elif isinstance(obj, (np.bool_,)):
        return bool(obj)
    elif pd.isna(obj):
        return None
    return obj


def write_json(data, filepath):
    os.makedirs(os.path.dirname(filepath), exist_ok=True)
    clean = sanitize_for_json(data)
    with open(filepath, "w", encoding="utf-8") as f:
        json.dump(clean, f, ensure_ascii=False, indent=2)
    return filepath


def build_seller_objects(schema, item_df):
    obj_type = None
    for obj in schema["objects"]:
        if obj["object_type_id"] == "seller":
            obj_type = obj
            break
    if obj_type is None:
        return []

    late_mask = (
        item_df["order_delivered_carrier_date"].notna()
        & item_df["shipping_limit_date"].notna()
        & (pd.to_datetime(item_df["order_delivered_customer_date"]) > pd.to_datetime(item_df["shipping_limit_date"]))
    )
    item_df = item_df.copy()
    item_df["_is_late"] = late_mask.astype(float)

    grouped = item_df.groupby("seller_id")
    seller_count = grouped["seller_id"].count()
    seller_state = grouped["seller_state"].first()
    seller_city = grouped["seller_city"].first()
    seller_gmv = grouped["price"].sum()
    seller_orders = grouped["order_id"].nunique()
    seller_reviews = grouped["review_score"].mean()
    seller_late_rate = grouped["_is_late"].sum() / seller_count

    objects = []
    for sid in seller_count.index:
        obj = {
            "object_id": f"seller_{sid}",
            "object_type": "seller",
            "object_name": f"seller_{sid}",
            "properties": {
                "seller_id": sid,
                "seller_state": seller_state.loc[sid],
                "seller_city": seller_city.loc[sid],
                "gmv": seller_gmv.loc[sid],
                "order_count": int(seller_orders.loc[sid]),
                "avg_review_score": seller_reviews.loc[sid],
                "late_delivery_rate": seller_late_rate.loc[sid],
            },
        }
        objects.append(obj)
    return objects


def build_category_objects(schema, item_df):
    obj_type = None
    for obj in schema["objects"]:
        if obj["object_type_id"] == "category":
            obj_type = obj
            break
    if obj_type is None:
        return []

    item_df = item_df.copy()
    late_mask = (
        item_df["order_delivered_carrier_date"].notna()
        & item_df["shipping_limit_date"].notna()
        & (pd.to_datetime(item_df["order_delivered_customer_date"]) > pd.to_datetime(item_df["shipping_limit_date"]))
    )
    item_df["_is_late"] = late_mask.astype(float)

    grouped = item_df.groupby("product_category_name")
    cat_count = grouped["product_category_name"].count()
    cat_name_en = grouped["product_category_name_english"].first()
    cat_gmv = grouped["price"].sum()
    cat_orders = grouped["order_id"].nunique()
    cat_reviews = grouped["review_score"].mean()
    cat_late_rate = grouped["_is_late"].sum() / cat_count

    objects = []
    for cat in cat_count.index:
        obj = {
            "object_id": f"category_{cat}",
            "object_type": "category",
            "object_name": f"category_{cat}",
            "properties": {
                "product_category_name": cat,
                "product_category_name_english": cat_name_en.loc[cat],
                "gmv": cat_gmv.loc[cat],
                "order_count": int(cat_orders.loc[cat]),
                "avg_review_score": cat_reviews.loc[cat],
                "late_delivery_rate": cat_late_rate.loc[cat],
            },
        }
        objects.append(obj)
    return objects


def build_region_objects(schema, item_df, order_df):
    obj_type = None
    for obj in schema["objects"]:
        if obj["object_type_id"] == "region":
            obj_type = obj
            break
    if obj_type is None:
        return []

    regions = pd.concat(
        [
            item_df[["customer_state"]].rename(columns={"customer_state": "state"}).dropna(),
            order_df[["customer_state"]].rename(columns={"customer_state": "state"}).dropna(),
        ]
    )["state"].unique()

    item_df = item_df.copy()
    item_df["order_purchase_ts"] = pd.to_datetime(item_df["order_purchase_timestamp"])
    item_df["order_delivered_customer_ts"] = pd.to_datetime(item_df["order_delivered_customer_date"])
    item_df["shipping_limit_ts"] = pd.to_datetime(item_df["shipping_limit_date"])
    item_df["_delivery_days"] = (item_df["order_delivered_customer_ts"] - item_df["order_purchase_ts"]).dt.total_seconds() / 86400.0

    item_by_state = item_df.groupby("customer_state")
    order_by_state = order_df.groupby("customer_state")

    objects = []
    for state in regions:
        ib = item_by_state.get_group(state) if state in item_by_state.groups else None
        ob = order_by_state.get_group(state) if state in order_by_state.groups else None

        customer_count = ob["customer_unique_id"].nunique() if ob is not None else 0
        seller_count = ib["seller_id"].nunique() if ib is not None else 0
        gmv = ib["price"].sum() if ib is not None else 0.0
        avg_review = ib["review_score"].mean() if ib is not None else None
        delivered_mask = ib["order_delivered_customer_ts"].notna() if ib is not None else pd.Series(dtype=bool)
        avg_delivery = ib.loc[delivered_mask, "_delivery_days"].mean() if delivered_mask.any() else None

        obj = {
            "object_id": f"region_{state}",
            "object_type": "region",
            "object_name": f"region_{state}",
            "properties": {
                "state": state,
                "customer_count": int(customer_count),
                "seller_count": int(seller_count),
                "gmv": gmv,
                "avg_review_score": avg_review,
                "avg_delivery_days": avg_delivery,
            },
        }
        objects.append(obj)
    return objects


def build_metrics():
    if not os.path.exists(DAILY_METRICS_FILE):
        return []

    df = pd.read_csv(DAILY_METRICS_FILE)
    date_col = "simulated_date"
    metric_cols = [c for c in df.columns if c not in (date_col, "real_run_date")]
    records = []
    for _, row in df.iterrows():
        for metric_name in metric_cols:
            val = row[metric_name]
            if pd.isna(val):
                continue
            records.append(
                {
                    "metric_date": str(row[date_col]),
                    "metric_name": metric_name,
                    "value": float(val),
                    "grain": "daily",
                    "dimensions": {},
                }
            )
    return records


def build_events():
    if not os.path.exists(METRIC_ALERTS_FILE):
        return []

    df = pd.read_csv(METRIC_ALERTS_FILE)
    events = []
    for _, row in df.iterrows():
        events.append(
            {
                "event_id": str(row.get("alert_id", row.get("rule_id", ""))),
                "event_type": str(row.get("rule_id", "")),
                "event_timestamp": str(row.get("detected_at", "")),
                "properties": {
                    k: sanitize_for_json(v)
                    for k, v in row.to_dict().items()
                },
            }
        )
    return events


def build_recommendations(events):
    recommendations = []
    for event in events:
        action = ACTION_MAP.get(event.get("event_type", ""), "")
        if not action:
            continue
        recommendations.append(
            {
                "recommendation_id": f"rec_{event.get('event_id', '')}",
                "source_event_id": event.get("event_id", ""),
                "action": action,
                "priority": "high",
                "properties": event.get("properties", {}),
            }
        )
    return recommendations


def main():
    os.makedirs(AIP_DIR, exist_ok=True)

    with open(AIP_OBJECT_SCHEMA_FILE, "r", encoding="utf-8") as f:
        schema = yaml.safe_load(f)

    item_df = pd.read_csv(ITEM_LEVEL_BASE_FILE)
    order_df = pd.read_csv(ORDER_LEVEL_BASE_FILE)

    seller_objs = build_seller_objects(schema, item_df)
    category_objs = build_category_objects(schema, item_df)
    region_objs = build_region_objects(schema, item_df, order_df)

    objects = {
        "objects": seller_objs + category_objs + region_objs,
        "_generated_at": str(pd.Timestamp.now()),
        "_counts": {
            "seller": len(seller_objs),
            "category": len(category_objs),
            "region": len(region_objs),
        },
    }
    write_json(objects, AIP_BUSINESS_OBJECTS_FILE)

    metrics = {
        "metrics": build_metrics(),
        "_generated_at": str(pd.Timestamp.now()),
    }
    write_json(metrics, AIP_METRICS_FILE)

    events = build_events()
    events_data = {
        "events": events,
        "_generated_at": str(pd.Timestamp.now()),
    }
    write_json(events_data, AIP_EVENTS_FILE)

    recommendations = build_recommendations(events)
    recs_data = {
        "recommendations": recommendations,
        "_generated_at": str(pd.Timestamp.now()),
    }
    write_json(recs_data, AIP_ACTION_RECOMMENDATIONS_FILE)

    print(f"Generated {len(seller_objs)} seller objects")
    print(f"Generated {len(category_objs)} category objects")
    print(f"Generated {len(region_objs)} region objects")
    print(f"Generated {len(metrics['metrics'])} metric records")
    print(f"Generated {len(events)} events")
    print(f"Generated {len(recommendations)} recommendations")
    print("AIP object layer build complete.")


if __name__ == "__main__":
    main()
