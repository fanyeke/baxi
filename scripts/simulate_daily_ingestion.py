import sys, os
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import json
import csv
import uuid
from datetime import datetime, timedelta, timezone

import pandas as pd

from scripts.config import (
    RAW_DIR,
    ADS_DIR,
    SYSTEM_DIR,
    INGESTION_STATE_FILE,
    RUN_MANIFEST_FILE,
    ensure_dirs_exist,
)

ORDERS_FILE = os.path.join(RAW_DIR, "olist_orders_dataset.csv")


def _init_state():
    if os.path.exists(INGESTION_STATE_FILE):
        with open(INGESTION_STATE_FILE, "r") as f:
            return json.load(f)
    state = {
        "current_simulated_date": "2016-09-04",
        "last_run_at": None,
        "last_loaded_order_count": 0,
        "status": "never_run",
    }
    os.makedirs(SYSTEM_DIR, exist_ok=True)
    with open(INGESTION_STATE_FILE, "w") as f:
        json.dump(state, f, indent=2)
    return state


def _init_manifest():
    if os.path.exists(RUN_MANIFEST_FILE):
        return
    os.makedirs(SYSTEM_DIR, exist_ok=True)
    with open(RUN_MANIFEST_FILE, "w", newline="") as f:
        writer = csv.writer(f)
        writer.writerow([
            "run_id",
            "real_run_date",
            "simulated_date",
            "pipeline_stage",
            "input_row_count",
            "output_row_count",
            "status",
            "error_message",
            "started_at",
            "finished_at",
            "bundle_path",
            "report_path",
        ])


def _append_manifest(run_id, sim_date, output_file, order_count, status, error_msg=""):
    now_iso = datetime.now(timezone.utc).isoformat()
    with open(RUN_MANIFEST_FILE, "a", newline="") as f:
        writer = csv.writer(f)
        writer.writerow([
            run_id,
            now_iso,
            str(sim_date),
            "ingestion",
            order_count,
            order_count,
            status,
            error_msg,
            now_iso,
            now_iso,
            output_file,
            "",
        ])


def _save_state(state):
    with open(INGESTION_STATE_FILE, "w") as f:
        json.dump(state, f, indent=2)


def main():
    ensure_dirs_exist()

    state = _init_state()
    _init_manifest()

    run_id = uuid.uuid4().hex
    sim_date = state["current_simulated_date"]
    filter_date = pd.to_datetime(sim_date).date()

    if os.path.exists(RUN_MANIFEST_FILE):
        with open(RUN_MANIFEST_FILE, "r") as f:
            reader = csv.DictReader(f)
            for row in reader:
                if row["simulated_date"] == sim_date and row["status"] == "success":
                    _append_manifest(run_id, sim_date, "", 0, "skipped_already_run")
                    state["last_run_at"] = datetime.now(timezone.utc).isoformat()
                    state["status"] = "skipped_already_run"
                    _save_state(state)
                    print(f"[SKIP] Date {sim_date} already processed. status=skipped_already_run")
                    return

    orders = pd.read_csv(ORDERS_FILE, parse_dates=["order_purchase_timestamp"])
    orders["purchase_date"] = orders["order_purchase_timestamp"].dt.date
    day_orders = orders[orders["purchase_date"] == filter_date].copy()
    order_count = len(day_orders)

    date_str = filter_date.strftime("%Y%m%d")
    output_filename = f"raw_incremental_{date_str}.csv"
    output_path = os.path.join(ADS_DIR, output_filename)

    if order_count == 0:
        day_orders_drop = day_orders.drop(columns=["purchase_date"])
        day_orders_drop.to_csv(output_path, index=False)
        _append_manifest(run_id, sim_date, output_filename, 0, "success")
        state["last_run_at"] = datetime.now(timezone.utc).isoformat()
        state["last_loaded_order_count"] = 0
        state["status"] = "success"
        state["current_simulated_date"] = (filter_date + timedelta(days=1)).isoformat()
        _save_state(state)
        print(f"[EMPTY] No orders for {sim_date}. Wrote empty {output_filename}")
        return

    day_orders_drop = day_orders.drop(columns=["purchase_date"])
    day_orders_drop.to_csv(output_path, index=False)

    _append_manifest(run_id, sim_date, output_filename, order_count, "success")


    state["last_run_at"] = datetime.now(timezone.utc).isoformat()
    state["last_loaded_order_count"] = order_count
    state["status"] = "success"
    state["current_simulated_date"] = (filter_date + timedelta(days=1)).isoformat()
    _save_state(state)

    print(f"[OK] Loaded {order_count} orders for {sim_date} -> {output_filename}")


if __name__ == "__main__":
    main()
