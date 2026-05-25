"""pipeline/runner.py — Pipeline orchestration via direct function calls."""

import datetime
import os
import sys

# Ensure pipeline package can import steps (project root must be on path)
_PIPELINE_DIR = os.path.dirname(os.path.abspath(__file__))
_PROJECT_ROOT = os.path.dirname(_PIPELINE_DIR)
if _PROJECT_ROOT not in sys.path:
    sys.path.insert(0, _PROJECT_ROOT)

from pipeline.steps import (  # noqa: E402
    step_db_calculate_dimension_metrics,
    step_db_calculate_metrics,
    step_db_dimensional_rule_engine,
    step_db_export_feishu,
    step_db_generate_recommendations,
    step_db_ingest,
    step_db_init,
    step_db_rule_engine,
    step_db_trigger_simulator,
)

STEPS = [
    {"name": "db_init", "func": step_db_init, "args": []},
    {"name": "db_ingest", "func": step_db_ingest, "args": ["mode", "start", "end"]},
    {
        "name": "db_calculate_metrics",
        "func": step_db_calculate_metrics,
        "args": ["mode", "start", "end"],
    },
    {
        "name": "db_calculate_dimension_metrics",
        "func": step_db_calculate_dimension_metrics,
        "args": ["mode", "start", "end"],
    },
    {"name": "db_rule_engine", "func": step_db_rule_engine, "args": []},
    {"name": "db_dimensional_rule_engine", "func": step_db_dimensional_rule_engine, "args": []},
    {"name": "db_generate_recommendations", "func": step_db_generate_recommendations, "args": []},
    {"name": "db_export_feishu", "func": step_db_export_feishu, "args": []},
    {"name": "db_trigger_simulator", "func": step_db_trigger_simulator, "args": ["dry_run"]},
]


def run_pipeline(mode="full", start=None, end=None, dimensional=False, db_path=None, dry_run=True):
    """Run the database pipeline by calling step functions directly.

    Args:
        mode: 'full' or 'range' ingestion/calculation mode.
        start: Start date for range mode (YYYY-MM-DD).
        end: End date for range mode (YYYY-MM-DD).
        dimensional: If True, include v0.3 dimensional steps.
        db_path: Optional explicit path to SQLite database.
        dry_run: If True, trigger simulator runs without updating outbox status.

    Returns:
        True if all steps succeed, False otherwise.
    """
    steps = STEPS if dimensional else [STEPS[0], STEPS[1], STEPS[2], STEPS[4], STEPS[7], STEPS[8]]

    print("[pipeline] Starting v0.3 DB pipeline")
    print(f"[pipeline] Mode: {mode}, Dimensional: {dimensional}")
    print(f"[pipeline] Steps: {len(steps)}")
    print(f"[pipeline] Started: {datetime.datetime.now().isoformat()}")

    overall_start = datetime.datetime.now()
    failed_step = None

    for i, step in enumerate(steps, 1):
        print(f"\n{'=' * 50}")
        print(f"[{i}/{len(steps)}] Running {step['name']}...")
        print(f"{'=' * 50}")

        step_start = datetime.datetime.now()

        # Build kwargs from declared step args and caller parameters
        kwargs = {}
        if "mode" in step["args"]:
            kwargs["mode"] = mode
        if "start" in step["args"]:
            kwargs["start"] = start
        if "end" in step["args"]:
            kwargs["end"] = end
        if "dry_run" in step["args"]:
            kwargs["dry_run"] = dry_run
        if db_path is not None:
            kwargs["db_path"] = db_path

        try:
            success = step["func"](**kwargs)
        except Exception as e:
            print(f"[pipeline] FAILED: {step['name']} - {e}")
            import traceback

            traceback.print_exc()
            success = False

        elapsed = (datetime.datetime.now() - step_start).total_seconds()

        if not success:
            print(f"[pipeline] FAILED: {step['name']} ({elapsed:.1f}s)")
            failed_step = step["name"]
            break
        print(f"[pipeline] OK: {step['name']} ({elapsed:.1f}s)")

    total_elapsed = (datetime.datetime.now() - overall_start).total_seconds()
    if failed_step:
        print(f"\n[pipeline] FAILED at {failed_step} ({total_elapsed:.1f}s)")
        return False
    else:
        print(f"\n{'=' * 50}")
        print(f"[pipeline] ALL STEPS COMPLETED ({total_elapsed:.1f}s)")
        print(f"{'=' * 50}")
        return True
