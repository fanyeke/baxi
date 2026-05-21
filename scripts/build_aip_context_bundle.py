import sys, os, json
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from scripts.config import *
from datetime import datetime, timezone


def load_json_if_exists(path):
    if os.path.exists(path):
        with open(path) as f:
            return json.load(f)
    return None


def load_yaml_if_exists(path):
    import yaml
    if os.path.exists(path):
        with open(path) as f:
            return yaml.safe_load(f)
    return None


def main():
    bundle = {
        "snapshot_date": None,
        "real_run_date": datetime.now(timezone.utc).isoformat(),
        "ingestion_state": load_json_if_exists(INGESTION_STATE_FILE) or {},
        "metrics": [],
        "events": [],
        "recommendations": [],
        "business_objects_summary": [],
        "owner_mapping": load_yaml_if_exists(OWNER_MAPPING_FILE) or {},
        "allowed_actions": load_yaml_if_exists(ACTION_REGISTRY_FILE) or {},
    }

    state = bundle["ingestion_state"]
    bundle["snapshot_date"] = state.get("current_simulated_date", "unknown")

    metrics_data = load_json_if_exists(AIP_METRICS_FILE)
    if metrics_data:
        bundle["metrics"] = metrics_data.get("metrics", [])

    events_data = load_json_if_exists(AIP_EVENTS_FILE)
    if events_data:
        bundle["events"] = events_data.get("events", [])

    rec_data = load_json_if_exists(AIP_ACTION_RECOMMENDATIONS_FILE)
    if rec_data:
        bundle["recommendations"] = rec_data.get("recommendations", [])

    bo_data = load_json_if_exists(AIP_BUSINESS_OBJECTS_FILE)
    if bo_data:
        objs = bo_data.get("objects", [])
        summary = {}
        for o in objs:
            t = o.get("object_type", "unknown")
            summary[t] = summary.get(t, 0) + 1
        bundle["business_objects_summary"] = summary

    os.makedirs(os.path.dirname(AIP_CONTEXT_BUNDLE_FILE), exist_ok=True)
    with open(AIP_CONTEXT_BUNDLE_FILE, 'w') as f:
        json.dump(bundle, f, indent=2, ensure_ascii=False)

    counts = {
        "metrics": len(bundle["metrics"]),
        "events": len(bundle["events"]),
        "recommendations": len(bundle["recommendations"]),
        "object_types": list(bundle["business_objects_summary"].keys()),
        "allowed_actions": list(bundle["allowed_actions"].get("actions", {}).keys()),
    }
    print(f"[bundle] Created {AIP_CONTEXT_BUNDLE_FILE}")
    print(f"  snapshot_date: {bundle['snapshot_date']}")
    print(f"  counts: {counts}")


if __name__ == '__main__':
    main()
