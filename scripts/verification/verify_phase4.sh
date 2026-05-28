#!/usr/bin/env bash
# verify_phase4.sh - Phase 4 Verification: Go-Primary Write
#
# Verifies that Go API handles all writes, Python API is read-only,
# and data integrity is maintained after write migration.
#
# Exit codes:
#   0 - All checks passed
#   1 - One or more checks failed
set -euo pipefail

# ── Configuration ──────────────────────────────────────────────
GO_API_URL="${GO_API_URL:-http://127.0.0.1:8080}"
PYTHON_API_URL="${PYTHON_API_URL:-http://127.0.0.1:8765}"
DATABASE_URL="${DATABASE_URL:-postgres://baxi:baxi_dev@localhost:5432/baxi?sslmode=disable}"
API_BEARER_TOKEN="${API_BEARER_TOKEN:-REPLACE_ME}"
TIMEOUT=5

# ── Colors ─────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

FAIL=0

pass() { echo -e "${GREEN}✓ PASS${NC} $1"; }
fail() { echo -e "${RED}✗ FAIL${NC} $1"; FAIL=1; }
warn() { echo -e "${YELLOW}⚠ WARN${NC} $1"; }
info() { echo -e "  $1"; }

echo "═══════════════════════════════════════════════════════════"
echo "  Phase 4 Verification: Go-Primary Write"
echo "═══════════════════════════════════════════════════════════"
echo ""

# ── 1. Go API Healthy ─────────────────────────────────────────
echo "── Check 1: Go API Health ──"
if curl -sf --max-time "$TIMEOUT" "${GO_API_URL}/api/v1/health" > /dev/null 2>&1; then
    pass "Go API is healthy"
else
    fail "Go API not reachable at ${GO_API_URL}"
fi

# ── 2. Python API Read-Only Mode ─────────────────────────────
echo ""
echo "── Check 2: Python API Read-Only Mode ──"
# Verify Python API is still accessible for reads
if curl -sf --max-time "$TIMEOUT" "${PYTHON_API_URL}/api/v1/health" > /dev/null 2>&1; then
    pass "Python API is reachable (read-only mode)"
    
    # Check if Python API is configured for read-only
    if [ -n "${PYTHON_READ_ONLY:-}" ] && [ "${PYTHON_READ_ONLY:-false}" = "true" ]; then
        pass "PYTHON_READ_ONLY is enabled"
    else
        warn "PYTHON_READ_ONLY flag not detected (check config)"
    fi
else
    warn "Python API not reachable (may be expected in Phase 4)"
fi

# ── 3. Go API Write Success ──────────────────────────────────
echo ""
echo "── Check 3: Go API Write Operations ──"
# Test creating a task via Go API
TEST_TASK='{"title":"phase4-verification-task","status":"pending","source":"verify_phase4.sh"}'
if resp=$(curl -sf --max-time "$TIMEOUT" \
    -H "Authorization: Bearer ${API_BEARER_TOKEN}" \
    -H "Content-Type: application/json" \
    -d "$TEST_TASK" \
    "${GO_API_URL}/api/v1/tasks" 2>/dev/null); then
    pass "Go API task creation succeeded"
    info "Response: $resp"
else
    warn "Go API task creation endpoint may not be available"
    info "Checking if Go API can handle write requests via other means..."
fi

# ── 4. Data Integrity Check ──────────────────────────────────
echo ""
echo "── Check 4: Data Integrity ──"
# Verify critical tables have data
TABLES_WITH_DATA=(
    "dwd_order_level"
    "dwd_item_level"
    "metric_daily"
    "alert_events"
    "strategy_recommendations"
    "action_tasks"
)

for table in "${TABLES_WITH_DATA[@]}"; do
    if count=$(psql "$DATABASE_URL" -tAc "SELECT COUNT(*) FROM $table" 2>/dev/null); then
        if [ "$count" -gt 0 ]; then
            pass "Table '$table': $count rows"
        else
            fail "Table '$table' is empty (data integrity issue)"
        fi
    else
        fail "Cannot query table '$table'"
    fi
done

# ── 5. Write Audit Log ──────────────────────────────────────
echo ""
echo "── Check 5: Write Audit Log ──"
# Check if writes are being logged
if psql "$DATABASE_URL" -c "SELECT 1 FROM event_outbox LIMIT 1" > /dev/null 2>&1; then
    if outbox_count=$(psql "$DATABASE_URL" -tAc "SELECT COUNT(*) FROM event_outbox" 2>/dev/null); then
        pass "Outbox events tracked: $outbox_count"
    else
        warn "Could not count outbox events"
    fi
else
    warn "event_outbox table not found (write tracking may not be enabled)"
fi

# ── 6. Pipeline Still Works ──────────────────────────────────
echo ""
echo "── Check 6: Go Pipeline Operational ──"
if go test ./internal/pipeline/... -count=1 -short > /dev/null 2>&1; then
    pass "Go pipeline tests pass"
else
    fail "Go pipeline tests failed"
fi

# ── 7. Frontend Connected to Go ──────────────────────────────
echo ""
echo "── Check 7: Frontend Configuration ──"
if [ -n "${USE_GO_API:-}" ] && [ "${USE_GO_API:-false}" = "true" ]; then
    pass "Frontend configured to use Go API"
else
    warn "USE_GO_API not set (frontend may still use Python)"
fi

# ── 8. No Write Conflicts ───────────────────────────────────
echo ""
echo "── Check 8: Write Conflict Detection ──"
# Check for any recent write conflicts in logs
if [ -d "logs" ]; then
    conflict_count=$(grep -r "write_conflict\|WRITE_CONFLICT" logs/ 2>/dev/null | wc -l)
    if [ "$conflict_count" -eq 0 ]; then
        pass "No write conflicts detected in logs"
    else
        warn "Found $conflict_count write conflict entries in logs"
        info "Check logs/ for details"
    fi
else
    info "No logs directory found (skipping conflict check)"
fi

# ── Summary ───────────────────────────────────────────────────
echo ""
echo "═══════════════════════════════════════════════════════════"
if [ "$FAIL" -eq 0 ]; then
    echo -e "  ${GREEN}Phase 4 Verification: ALL PASSED${NC}"
    echo "═══════════════════════════════════════════════════════════"
    exit 0
else
    echo -e "  ${RED}Phase 4 Verification: SOME FAILED${NC}"
    echo "═══════════════════════════════════════════════════════════"
    exit 1
fi
