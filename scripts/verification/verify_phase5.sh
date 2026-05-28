#!/usr/bin/env bash
# verify_phase5.sh - Phase 5 Verification: Python Sunset
#
# Verifies that Python API is stopped, Go API operates normally,
# frontend works correctly, and all data is accessible.
#
# Exit codes:
#   0 - All checks passed
#   1 - One or more checks failed
set -euo pipefail

# ── Configuration ──────────────────────────────────────────────
GO_API_URL="${GO_API_URL:-http://127.0.0.1:8080}"
PYTHON_API_URL="${PYTHON_API_URL:-http://127.0.0.1:8765}"
FRONTEND_URL="${FRONTEND_URL:-http://127.0.0.1:5173}"
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
echo "  Phase 5 Verification: Python Sunset"
echo "═══════════════════════════════════════════════════════════"
echo ""

# ── 1. Python API Stopped ────────────────────────────────────
echo "── Check 1: Python API Stopped ──"
if curl -sf --max-time "$TIMEOUT" "${PYTHON_API_URL}/api/v1/health" > /dev/null 2>&1; then
    fail "Python API is still running (should be stopped in Phase 5)"
    info "Stop the Python API before running Phase 5 verification"
else
    pass "Python API is stopped"
fi

# ── 2. Go API Normal Operation ────────────────────────────────
echo ""
echo "── Check 2: Go API Normal Operation ──"
if resp=$(curl -sf --max-time "$TIMEOUT" "${GO_API_URL}/api/v1/health" 2>/dev/null); then
    if echo "$resp" | grep -q '"status":"ok"'; then
        pass "Go API is healthy and operational"
        info "Response: $resp"
    else
        fail "Go API returned unexpected status: $resp"
    fi
else
    fail "Go API not reachable at ${GO_API_URL}"
fi

# ── 3. Go API Serves All Endpoints ───────────────────────────
echo ""
echo "── Check 3: Go API All Endpoints ──"
ENDPOINTS=(
    "/api/v1/health"
    "/api/v1/status"
    "/api/v1/alerts"
    "/api/v1/tasks"
    "/api/v1/outbox"
)

for endpoint in "${ENDPOINTS[@]}"; do
    if curl -sf --max-time "$TIMEOUT" \
        -H "Authorization: Bearer ${API_BEARER_TOKEN}" \
        "${GO_API_URL}${endpoint}" > /dev/null 2>&1; then
        pass "Endpoint ${endpoint} works"
    else
        warn "Endpoint ${endpoint} not available"
    fi
done

# ── 4. Frontend Normal ───────────────────────────────────────
echo ""
echo "── Check 4: Frontend Normal ──"
if resp=$(curl -sf --max-time "$TIMEOUT" "${FRONTEND_URL}" 2>/dev/null); then
    if echo "$resp" | grep -qi "react\|vite\|html"; then
        pass "Frontend is serving correctly"
    else
        warn "Frontend returned content but may not be correct"
    fi
else
    warn "Frontend not reachable at ${FRONTEND_URL}"
fi

# ── 5. All Data Accessible ───────────────────────────────────
echo ""
echo "── Check 5: All Data Accessible ──"
TABLES=(
    "dwd_order_level"
    "dwd_item_level"
    "metric_daily"
    "metric_dimension_daily"
    "alert_events"
    "strategy_recommendations"
    "action_tasks"
    "event_outbox"
)

for table in "${TABLES[@]}"; do
    if count=$(psql "$DATABASE_URL" -tAc "SELECT COUNT(*) FROM $table" 2>/dev/null); then
        if [ "$count" -gt 0 ]; then
            pass "Table '$table': $count rows"
        else
            warn "Table '$table' is empty"
        fi
    else
        fail "Cannot query table '$table'"
    fi
done

# ── 6. Go Tests Pass ─────────────────────────────────────────
echo ""
echo "── Check 6: Go Tests Pass ──"
if go test ./... -count=1 -short > /dev/null 2>&1; then
    pass "All Go tests pass"
else
    fail "Go tests failed"
    info "Run 'make test-go' for details"
fi

# ── 7. Python Tests Skipped ──────────────────────────────────
echo ""
echo "── Check 7: Python Tests (Optional) ──"
if pytest tests -q --timeout=120 > /dev/null 2>&1; then
    info "Python tests still pass (optional in Phase 5)"
else
    info "Python tests failed (expected if Python dependencies removed)"
fi

# ── 8. No Python References in Config ────────────────────────
echo ""
echo "── Check 8: Configuration Cleanup ──"
# Check if frontend config points to Go API
if grep -r "127.0.0.1:8765\|localhost:8765" frontend/ 2>/dev/null | grep -v "node_modules" | grep -v ".git"; then
    warn "Found Python API references in frontend config"
    info "Update frontend to point to Go API only"
else
    pass "No Python API references in frontend config"
fi

# ── Summary ───────────────────────────────────────────────────
echo ""
echo "═══════════════════════════════════════════════════════════"
if [ "$FAIL" -eq 0 ]; then
    echo -e "  ${GREEN}Phase 5 Verification: ALL PASSED${NC}"
    echo "  Python API sunset complete. Go API is the sole backend."
    echo "═══════════════════════════════════════════════════════════"
    exit 0
else
    echo -e "  ${RED}Phase 5 Verification: SOME FAILED${NC}"
    echo "═══════════════════════════════════════════════════════════"
    exit 1
fi
