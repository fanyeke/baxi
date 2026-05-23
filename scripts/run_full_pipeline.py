#!/usr/bin/env python3
"""Entry Point 2: Full-mode pipeline — processes ALL 634 days of Olist data."""

import sys, os, time, subprocess

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from scripts.config import ensure_dirs_exist

STEPS = [
    ("calculate_daily_metrics",  "calculate_daily_metrics.py"),
    ("run_alert_detection",      "run_alert_detection.py"),
    ("build_aip_context_bundle", "build_aip_context_bundle.py"),
    ("run_ai_decision_engine",   "run_ai_decision_engine.py"),
    ("generate_feishu_sandbox",  "generate_feishu_sandbox.py"),
]


def main():
    ensure_dirs_exist()
    scripts_dir = os.path.dirname(os.path.abspath(__file__))
    project_root = os.path.dirname(scripts_dir)
    success = True
    failed_step = None

    print(f"[FULL_PIPE] Starting full-mode pipeline ({len(STEPS)} steps)")
    for i, (name, script) in enumerate(STEPS, 1):
        path = os.path.join(scripts_dir, script)
        cmd = [sys.executable, path, "--mode", "full"]
        t0 = time.time()
        print(f"[{i}/{len(STEPS)}] {name}...", end=" ", flush=True)
        try:
            r = subprocess.run(
                cmd,
                capture_output=True, text=True, timeout=300,
                cwd=project_root,
            )
            if r.returncode != 0:
                print(f"FAIL (code={r.returncode})")
                err = (r.stderr or "")[:200]
                if err:
                    print(f"  stderr: {err}")
                success = False
                failed_step = name
                break
            else:
                elapsed = time.time() - t0
                print(f"OK ({elapsed:.1f}s)")
        except Exception as e:
            print(f"ERROR: {e}")
            success = False
            failed_step = name
            break

    status = "SUCCESS" if success else f"FAILED at {failed_step}"
    print(f"[FULL_PIPE] {status}")
    return 0 if success else 1


if __name__ == "__main__":
    sys.exit(main())
