#!/usr/bin/env python3
"""run_db_pipeline.py - Unified pipeline wrapper for v0.3 decision backend.
Orchestrates: init -> ingest -> metrics -> dim_metrics -> rule_engine -> dim_rule_engine -> recommendations -> export -> trigger
With --dimensional flag for v0.3 steps."""
import os, sys, subprocess, datetime, argparse
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import config

SCRIPTS_DIR = config.SCRIPTS_DIR
DB_PATH = os.path.join(config.PROJECT_ROOT, 'data', 'olist_ops.db')

STEPS = [
    {'name': 'db_init', 'script': 'db_init.py', 'args': []},
    {'name': 'db_ingest', 'script': 'db_ingest.py', 'args': ['--mode', '{mode}', '--start', '{start}', '--end', '{end}']},
    {'name': 'db_calculate_metrics', 'script': 'db_calculate_metrics.py', 'args': ['--mode', '{mode}']},
    {'name': 'db_calculate_dimension_metrics', 'script': 'db_calculate_dimension_metrics.py', 'args': ['--mode', '{mode}']},
    {'name': 'db_rule_engine', 'script': 'db_rule_engine.py', 'args': ['--mode', '{mode}']},
    {'name': 'db_dimensional_rule_engine', 'script': 'db_dimensional_rule_engine.py', 'args': ['--mode', '{mode}']},
    {'name': 'db_generate_recommendations', 'script': 'db_generate_recommendations.py', 'args': []},
    {'name': 'db_export_feishu', 'script': 'db_export_feishu.py', 'args': ['--all']},
    {'name': 'db_trigger_simulator', 'script': 'db_trigger_simulator.py', 'args': ['--dry-run']},
]


def run_step(step, mode='full', start=None, end=None):
    script_path = os.path.join(SCRIPTS_DIR, step['script'])
    args = [sys.executable, script_path]
    for arg in step['args']:
        args.append(arg.format(mode=mode, start=start or '', end=end or ''))

    print(f"\n{'='*50}")
    print(f"[pipeline] Step {step['name']}: {' '.join(args[1:])}")
    print(f"{'='*50}")

    started = datetime.datetime.now()
    result = subprocess.run(args, capture_output=True, text=True)
    elapsed = (datetime.datetime.now() - started).total_seconds()

    if result.stdout:
        print(result.stdout.strip()[-500:])
    if result.stderr:
        print(result.stderr.strip()[-500:], file=sys.stderr)

    if result.returncode != 0:
        print(f"[pipeline] FAILED: {step['name']} (exit {result.returncode}, {elapsed:.1f}s)")
        return False
    print(f"[pipeline] OK: {step['name']} ({elapsed:.1f}s)")
    return True


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--mode', default='full', choices=['full', 'range'])
    parser.add_argument('--dimensional', action='store_true',
                        help='Include v0.3 dimensional steps')
    parser.add_argument('--start', default=None)
    parser.add_argument('--end', default=None)
    parser.add_argument('--db', default=None)
    args = parser.parse_args()

    steps = STEPS if args.dimensional else [
        STEPS[0], STEPS[1], STEPS[2], STEPS[4], STEPS[7], STEPS[8]
    ]

    print(f"[pipeline] Starting v0.3 DB pipeline")
    print(f"[pipeline] Mode: {args.mode}, Dimensional: {args.dimensional}")
    print(f"[pipeline] Steps: {len(steps)}")
    print(f"[pipeline] Started: {datetime.datetime.now().isoformat()}")

    overall_start = datetime.datetime.now()
    failed_step = None

    for i, step in enumerate(steps, 1):
        print(f"\n[{i}/{len(steps)}] Running {step['name']}...")
        if not run_step(step, args.mode, args.start, args.end):
            failed_step = step['name']
            break

    elapsed = (datetime.datetime.now() - overall_start).total_seconds()
    if failed_step:
        print(f"\n[pipeline] FAILED at {failed_step} ({elapsed:.1f}s)")
        sys.exit(1)
    else:
        print(f"\n{'='*50}")
        print(f"[pipeline] ALL STEPS COMPLETED ({elapsed:.1f}s)")
        print(f"{'='*50}")


if __name__ == '__main__':
    main()
