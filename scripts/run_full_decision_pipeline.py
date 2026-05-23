"""
Full-mode decision pipeline: runs all steps for Phase I-Local verification.
Does NOT modify run_daily_pipeline.py — completely separate.
"""
import sys, os, time, subprocess
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

STEPS = [
    ("Daily Metrics", ["calculate_daily_metrics.py", "--mode", "full"]),
    ("Alert Detection", ["run_alert_detection.py", "--mode", "full", "--top-alerts", "200"]),
    ("Context Bundle", ["build_aip_context_bundle.py", "--mode", "full"]),
    ("Review Retro Samples", ["generate_review_retro_samples.py"]),
    ("AI Decision Engine", ["run_ai_decision_engine.py", "--mode", "full", "--top-alerts", "20"]),
    ("Feishu Sandbox", ["generate_feishu_sandbox.py", "--mode", "full"]),
]

def main():
    scripts_dir = os.path.dirname(os.path.abspath(__file__))
    print(f"[PIPE] Full-mode decision pipeline")
    print(f"  Steps: {len(STEPS)}")
    print()

    for i, (name, args) in enumerate(STEPS, 1):
        path = os.path.join(scripts_dir, *args[:1])
        t0 = time.time()
        print(f"[{i}/{len(STEPS)}] {name}...", end=" ", flush=True)
        r = subprocess.run(
            [sys.executable, path] + args[1:],
            capture_output=True, text=True, timeout=300,
            cwd=os.path.dirname(scripts_dir),
        )
        elapsed = time.time() - t0
        if r.returncode != 0:
            print(f"FAIL ({r.returncode})")
            print(r.stderr[:300])
            print(f"[PIPE] FAILED at step {i}: {name}")
            return 1
        print(f"OK ({elapsed:.1f}s)")

    print()
    print("[PIPE] SUCCESS")
    print()
    print("Output files:")
    for f in ['data/ads/daily_metrics_full.csv', 'data/ads/metric_alerts_full.csv',
              'data/aip/aip_context_bundle_full.json', 'outputs/ai/strategy_recommendations.json',
              'outputs/ai/action_tasks.json', 'outputs/ai/review_retro_draft.json',
              'data/feishu/daily_metrics_for_feishu_full.csv',
              'data/feishu/alert_events_for_feishu_full.csv',
              'data/feishu/strategy_recommendations_for_feishu_full.csv',
              'data/feishu/action_tasks_for_feishu_full.csv',
              'data/feishu/execution_reviews_for_feishu_full.csv']:
        if os.path.exists(f):
            size = os.path.getsize(f)
            print(f"  ✓ {f} ({size:,} bytes)")
        else:
            print(f"  ✗ {f} MISSING")

if __name__ == '__main__':
    sys.exit(main())
