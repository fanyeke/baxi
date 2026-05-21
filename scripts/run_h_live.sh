#!/bin/bash
# Phase H-Live Data: End-to-end execution script
# Usage: bash scripts/run_h_live.sh [--dry-run|--apply]
set -euo pipefail

MODE="${1:---dry-run}"
if [ "$MODE" = "--apply" ]; then
    VERIFY_FLAG=""
    DRY_RUN_FLAG="--apply"
else
    VERIFY_FLAG="--dry-run"
    DRY_RUN_FLAG="--dry-run"
fi
TABLES=("daily_metrics" "alert_events" "strategy_recommendations" "action_tasks" "review_retro")

echo "=========================================="
echo "Phase H-Live Data Execution: $MODE"
echo "=========================================="
echo ""

# Step 1: Verify Feishu connectivity
echo "=== Step 1: Verify Feishu tables ==="
python3 scripts/verify_feishu_tables.py $VERIFY_FLAG 2>&1 || { echo "❌ Table verification failed. Check credentials."; exit 1; }
echo ""

# Step 2-6: Validate and sync each table (low risk -> high risk)
for TABLE in "${TABLES[@]}"; do
    echo "=== Step: $TABLE ==="

    # Validate payload
    echo "--- Payload validation ---"
    python3 scripts/validate_feishu_payload.py --table "$TABLE" 2>&1 || { echo "⚠️  Validation warnings for $TABLE"; }
    echo ""

    # Dry-run first
    echo "--- Dry-run ---"
    python3 scripts/sync_feishu_bitable.py --table "$TABLE" --dry-run 2>&1
    echo ""

    # Apply if requested
    if [ "$MODE" = "--apply" ]; then
        echo "--- Apply ---"
        python3 scripts/sync_feishu_bitable.py --table "$TABLE" --apply 2>&1 || { echo "❌ Failed to sync $TABLE"; exit 1; }
        echo ""
    fi
done

# Idempotency check for daily_metrics
echo "=== Idempotency check: daily_metrics ==="
python3 scripts/sync_feishu_bitable.py --table daily_metrics --"$MODE" 2>&1
echo "(Second run should show same create/update counts)"
echo ""

# Step 7: Status pull-back dry-run
echo "=== Status pull-back ==="
python3 scripts/pull_feishu_status.py $DRY_RUN_FLAG 2>&1 || echo "⚠️  Pull status skipped"
echo ""

echo "=========================================="
echo "Phase H-Live Data: $MODE completed"
echo "=========================================="
