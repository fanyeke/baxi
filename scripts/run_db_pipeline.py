#!/usr/bin/env python3
"""run_db_pipeline.py - Unified pipeline wrapper for v0.3 decision backend.
Orchestrates: init -> ingest -> metrics -> dim_metrics -> rule_engine -> dim_rule_engine -> recommendations -> export -> trigger
With --dimensional flag for v0.3 steps."""

import argparse
import os
import sys

_PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
if _PROJECT_ROOT not in sys.path:
    sys.path.insert(0, _PROJECT_ROOT)

from pipeline.runner import run_pipeline  # noqa: E402


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--mode", default="full", choices=["full", "range"])
    parser.add_argument("--dimensional", action="store_true", help="Include v0.3 dimensional steps")
    parser.add_argument("--start", default=None)
    parser.add_argument("--end", default=None)
    parser.add_argument("--db", default=None)
    parser.add_argument(
        "--apply",
        action="store_true",
        help="Actually apply changes (default is safe dry-run, no DB mutations)",
    )
    args = parser.parse_args()

    success = run_pipeline(
        mode=args.mode,
        start=args.start,
        end=args.end,
        dimensional=args.dimensional,
        db_path=args.db,
        dry_run=not args.apply,
    )

    sys.exit(0 if success else 1)


if __name__ == "__main__":
    main()
