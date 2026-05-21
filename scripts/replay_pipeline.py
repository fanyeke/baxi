import sys, os, argparse, time, subprocess
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from datetime import datetime, timezone


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--days", type=int, required=True, help="Days to replay")
    args = parser.parse_args()

    scripts_dir = os.path.dirname(os.path.abspath(__file__))
    project_dir = os.path.dirname(scripts_dir)
    pipeline = os.path.join(scripts_dir, "run_daily_pipeline.py")
    success, fail = 0, 0

    print(f"[REPLAY] Starting {args.days}-day replay at {datetime.now(timezone.utc).isoformat()}")
    t0 = time.time()

    for day in range(1, args.days + 1):
        print(f"\n[{day}/{args.days}] ", end="", flush=True)
        try:
            r = subprocess.run(
                [sys.executable, pipeline],
                capture_output=True, text=True, timeout=300,
                cwd=project_dir,
            )
            if r.returncode == 0:
                success += 1
                # extract date from ingestion output
                pipeline_output_summary = [l for l in r.stdout.strip().split("\n") if l][-1] if r.stdout.strip() else ""
                print(f"OK {pipeline_output_summary[:60]}")
            else:
                fail += 1
                print(f"FAIL (code={r.returncode})")
                err = (r.stderr or r.stdout)[:150]
                if err.strip():
                    print(f"  {err.strip()}")
        except subprocess.TimeoutExpired:
            fail += 1
            print("TIMEOUT")
        except Exception as e:
            fail += 1
            print(f"ERROR: {e}")

    elapsed = time.time() - t0
    print(f"\n[REPLAY] Done. {args.days} days, success={success}, fail={fail}, elapsed={elapsed:.0f}s")
    return 0 if fail == 0 else 1


if __name__ == "__main__":
    sys.exit(main())
