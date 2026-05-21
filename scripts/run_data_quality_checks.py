import sys
import os
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import json
import uuid
import pandas as pd
from datetime import datetime, timezone

from scripts.config import (
    ORDER_LEVEL_BASE_FILE,
    ITEM_LEVEL_BASE_FILE,
    DAILY_METRICS_FILE,
    METRIC_ALERTS_FILE,
    AIP_CONTEXT_BUNDLE_FILE,
    VALIDATION_RESULTS_FILE,
    RUN_MANIFEST_FILE,
    SYSTEM_DIR,
)


def _load_csv_safe(path):
    """Load a CSV file, return None if not found."""
    if not os.path.exists(path):
        return None
    return pd.read_csv(path)


def _load_json_safe(path):
    """Load a JSON file, return None if not found."""
    if not os.path.exists(path):
        return None
    with open(path, 'r') as f:
        return json.load(f)


def run_test(test_id, test_name, check_fn):
    """Run a single validation test and return result dict."""
    try:
        check_fn()
        return {
            "test_id": test_id,
            "test_name": test_name,
            "status": "PASSED",
            "message": "",
            "timestamp": datetime.now().isoformat(),
        }
    except AssertionError as e:
        return {
            "test_id": test_id,
            "test_name": test_name,
            "status": "FAILED",
            "message": str(e),
            "timestamp": datetime.now().isoformat(),
        }
    except Exception as e:
        return {
            "test_id": test_id,
            "test_name": test_name,
            "status": "ERROR",
            "message": f"{type(e).__name__}: {e}",
            "timestamp": datetime.now().isoformat(),
        }


def main():
    started_at = datetime.now(timezone.utc).isoformat()

    order_level_base = pd.read_csv(ORDER_LEVEL_BASE_FILE)
    item_level_base = pd.read_csv(ITEM_LEVEL_BASE_FILE)

    results = []

    # T1: order_level_base.order_id no duplicates
    def t1():
        dup_count = order_level_base["order_id"].duplicated().sum()
        assert dup_count == 0, f"Found {dup_count} duplicate order_id(s)"
    results.append(run_test("T1", "order_level_base.order_id no duplicates", t1))

    # T2: order_level_base.order_purchase_timestamp not null
    def t2():
        null_count = order_level_base["order_purchase_timestamp"].isna().sum()
        assert null_count == 0, f"Found {null_count} null order_purchase_timestamp(s)"
    results.append(run_test("T2", "order_level_base.order_purchase_timestamp not null", t2))

    # T3: item_level_base.price >= 0
    def t3():
        negative_count = (item_level_base["price"] < 0).sum()
        assert negative_count == 0, f"Found {negative_count} rows with price < 0"
    results.append(run_test("T3", "item_level_base.price >= 0", t3))

    # T4: Only delivered orders have delivery time computed
    def t4():
        not_delivered = order_level_base["order_status"] != "delivered"
        has_delivery = order_level_base.loc[not_delivered, "order_delivered_customer_date"].notna()
        invalid_count = has_delivery.sum()
        assert invalid_count == 0, f"Found {invalid_count} non-delivered orders with delivery date"
    results.append(run_test("T4", "Only delivered orders have delivery time computed", t4))

    # T5: review_score values in {1.0, 2.0, 3.0, 4.0, 5.0} when not null
    def t5():
        if "review_score" not in order_level_base.columns:
            return  # column may not exist in this table
        not_null_scores = order_level_base["review_score"].dropna()
        valid_scores = {1.0, 2.0, 3.0, 4.0, 5.0}
        invalid = not_null_scores[~not_null_scores.isin(valid_scores)]
        assert len(invalid) == 0, f"Found {len(invalid)} invalid review_score(s): {invalid.unique()}"
    results.append(run_test("T5", "review_score values in {1-5} when not null", t5))

    # T6: If daily_metrics.csv exists, simulated_date is unique
    daily_metrics_df = _load_csv_safe(DAILY_METRICS_FILE)
    if daily_metrics_df is not None:
        def t6():
            if "simulated_date" not in daily_metrics_df.columns:
                raise AssertionError("daily_metrics.csv missing 'simulated_date' column")
            dup_count = daily_metrics_df["simulated_date"].duplicated().sum()
            assert dup_count == 0, f"Found {dup_count} duplicate simulated_date(s)"
        results.append(run_test("T6", "daily_metrics.csv simulated_date is unique", t6))
    else:
        results.append({
            "test_id": "T6",
            "test_name": "daily_metrics.csv simulated_date is unique",
            "status": "SKIPPED",
            "message": "daily_metrics.csv not found",
            "timestamp": datetime.now().isoformat(),
        })

    # T7: If metric_alerts.csv exists, alert_id is unique
    metric_alerts_df = _load_csv_safe(METRIC_ALERTS_FILE)
    if metric_alerts_df is not None:
        def t7():
            if "alert_id" not in metric_alerts_df.columns:
                raise AssertionError("metric_alerts.csv missing 'alert_id' column")
            dup_count = metric_alerts_df["alert_id"].duplicated().sum()
            assert dup_count == 0, f"Found {dup_count} duplicate alert_id(s)"
        results.append(run_test("T7", "metric_alerts.csv alert_id is unique", t7))
    else:
        results.append({
            "test_id": "T7",
            "test_name": "metric_alerts.csv alert_id is unique",
            "status": "SKIPPED",
            "message": "metric_alerts.csv not found",
            "timestamp": datetime.now().isoformat(),
        })

    # T8: If aip_context_bundle.json exists, has required keys
    aip_bundle = _load_json_safe(AIP_CONTEXT_BUNDLE_FILE)
    if aip_bundle is not None:
        def t8():
            required_keys = {"metrics", "events", "allowed_actions"}
            missing = required_keys - set(aip_bundle.keys())
            assert len(missing) == 0, f"Missing keys in aip_context_bundle.json: {missing}"
        results.append(run_test("T8", "aip_context_bundle.json has required keys", t8))
    else:
        results.append({
            "test_id": "T8",
            "test_name": "aip_context_bundle.json has required keys",
            "status": "SKIPPED",
            "message": "aip_context_bundle.json not found",
            "timestamp": datetime.now().isoformat(),
        })

    # Build output
    passed = sum(1 for r in results if r["status"] == "PASSED")
    failed = sum(1 for r in results if r["status"] == "FAILED")
    errors = sum(1 for r in results if r["status"] == "ERROR")
    skipped = len(results) - passed - failed - errors

    output = {
        "validation_timestamp": datetime.now().isoformat(),
        "summary": {
            "total_tests": len(results),
            "passed": passed,
            "failed": failed,
            "errors": errors,
            "skipped": skipped,
        },
        "results": results,
    }

    with open(VALIDATION_RESULTS_FILE, 'w') as f:
        json.dump(output, f, indent=2)

    # Append to run_manifest.csv
    manifest_entry = {
        "run_id": uuid.uuid4().hex,
        "real_run_date": datetime.now(timezone.utc).isoformat(),
        "simulated_date": "",
        "pipeline_stage": "data_quality_check",
        "input_row_count": len(results),
        "output_row_count": passed,
        "status": "passed" if (failed == 0 and errors == 0) else "failed",
        "error_message": "",
        "started_at": started_at,
        "finished_at": datetime.now(timezone.utc).isoformat(),
        "bundle_path": "",
        "report_path": VALIDATION_RESULTS_FILE,
    }

    if os.path.exists(RUN_MANIFEST_FILE):
        manifest_df = pd.read_csv(RUN_MANIFEST_FILE)
        manifest_df = pd.concat([manifest_df, pd.DataFrame([manifest_entry])], ignore_index=True)
    else:
        manifest_df = pd.DataFrame([manifest_entry])

    manifest_df.to_csv(RUN_MANIFEST_FILE, index=False)

    # Print summary
    print(f"Data quality validation complete.")
    print(f"  Total: {len(results)}")
    print(f"  Passed: {passed}")
    print(f"  Failed: {failed}")
    print(f"  Errors: {errors}")
    print(f"  Skipped: {skipped}")
    print(f"  Results: {VALIDATION_RESULTS_FILE}")
    print(f"  Manifest updated: {RUN_MANIFEST_FILE}")

    return 0 if (failed == 0 and errors == 0) else 1


if __name__ == "__main__":
    sys.exit(main())
