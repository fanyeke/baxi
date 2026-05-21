import sys, os, csv, time, uuid
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from datetime import datetime, timezone
from scripts.config import ensure_dirs_exist, RUN_MANIFEST_FILE

STEPS = [
    ("simulate_daily_ingestion", "simulate_daily_ingestion.py", True),
    ("run_data_quality_checks", "run_data_quality_checks.py", False),
    ("calculate_daily_metrics", "calculate_daily_metrics.py", True),
    ("run_alert_detection", "run_alert_detection.py", True),
    ("build_aip_objects", "build_aip_objects.py", True),
    ("build_aip_context_bundle", "build_aip_context_bundle.py", True),
    ("run_wake_agent", "run_wake_agent.py", True),
    ("generate_feishu_sandbox", "generate_feishu_sandbox.py", True),
]


def step_index(failed_name):
    for i, (name, _, _) in enumerate(STEPS):
        if name == failed_name:
            return i
    return 0


def append_manifest(run_id, success, failed_step, started, finished):
    completed = len(STEPS) if success else step_index(failed_step)
    with open(RUN_MANIFEST_FILE, "a", newline="") as f:
        w = csv.writer(f)
        w.writerow([
            run_id,
            datetime.now(timezone.utc).isoformat(),
            "",
            "pipeline_run",
            len(STEPS),
            completed,
            "success" if success else "failed",
            "" if success else f"Failed at step: {failed_step}",
            started,
            finished,
            "",
            "",
        ])


def main():
    ensure_dirs_exist()
    run_id = uuid.uuid4().hex
    started = datetime.now(timezone.utc).isoformat()
    scripts_dir = os.path.dirname(os.path.abspath(__file__))
    success = True
    failed_step = None

    print(f"[PIPE] run_id={run_id}")
    for i, (name, script, critical) in enumerate(STEPS, 1):
        path = os.path.join(scripts_dir, script)
        t0 = time.time()
        print(f"[{i}/{len(STEPS)}] {name}...", end=" ", flush=True)
        try:
            import subprocess
            r = subprocess.run(
                [sys.executable, path],
                capture_output=True, text=True, timeout=120,
                cwd=os.path.dirname(scripts_dir),
            )
            if r.returncode != 0:
                if critical:
                    print(f"FAIL (code={r.returncode})")
                    print(f"  stderr: {(r.stderr or '')[:200]}")
                    success = False
                    failed_step = name
                    break
                else:
                    print(f"WARN (code={r.returncode}, non-critical)")
            else:
                elapsed = time.time() - t0
                print(f"OK ({elapsed:.1f}s)")
        except Exception as e:
            print(f"ERROR: {e}")
            success = False
            failed_step = name
            break

    finished = datetime.now(timezone.utc).isoformat()
    append_manifest(run_id, success, failed_step, started, finished)
    status = "SUCCESS" if success else f"FAILED at {failed_step}"
    print(f"[PIPE] {status}")

    return 0 if success else 1


if __name__ == "__main__":
    sys.exit(main())
