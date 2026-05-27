#!/usr/bin/env bash
# verify_phase1.sh - Phase 1 Verification: Parallel Run
#
# Verifies that both Go and Python APIs are running in parallel,
# new PostgreSQL tables exist, and all tests pass.
#
# Exit codes:
#   0 - All checks passed
#   1 - One or more checks failed
set -euo pipefail

# ── Configuration ──────────────────────────────────────────────
GO_API_URL="${GO_API_URL:-http://127.0.0.1:8080}"
PYTHON_API_URL="${PYTHON_API_URL:-http://127.0.0.1:8765}"
DATABASE_URL="${DATABASE_URL:-postgres://baxi:baxi_dev@localhost:5432/baxi?sslmode=disable}"
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
echo "  Phase 1 Verification: Parallel Run"
echo "═══════════════════════════════════════════════════════════"
echo ""

# ── 1. Go API Health ──────────────────────────────────────────
echo "── Check 1: Go API Health ──"
if resp=$(curl -sf --max-time "$TIMEOUT" "${GO_API_URL}/api/v1/health" 2>/dev/null); then
    if echo "$resp" | grep -q '"status":"ok"'; then
        pass "Go API is healthy at ${GO_API_URL}"
        info "Response: $resp"
    else
        fail "Go API returned unexpected status: $resp"
    fi
else
    fail "Go API not reachable at ${GO_API_URL}"
fi

# ── 2. Python API Health ─────────────────────────────────────
echo ""
echo "── Check 2: Python API Health ──"
if resp=$(curl -sf --max-time "$TIMEOUT" "${PYTHON_API_URL}/api/v1/health" 2>/dev/null); then
    if echo "$resp" | grep -q '"status":"ok"'; then
        pass "Python API is healthy at ${PYTHON_API_URL}"
        info "Response: $resp"
    else
        fail "Python API returned unexpected status: $resp"
    fi
else
    fail "Python API not reachable at ${PYTHON_API_URL}"
fi

# ── 3. New Tables Exist ──────────────────────────────────────
echo ""
echo "── Check 3: New PostgreSQL Tables Exist ──"
REQUIRED_TABLES=(
    "dwd_order_level"
    "dwd_item_level"
    "metric_daily"
    "metric_dimension_daily"
    "alert_events"
    "strategy_recommendations"
    "action_tasks"
    "event_outbox"
)

table_exists() {
    local table=$1
    psql "$DATABASE_URL" -tAc "SELECT 1 FROM information_schema.tables WHERE table_name='$table'" 2>/dev/null | grep -q 1
}

for table in "${REQUIRED_TABLES[@]}"; do
    if table_exists "$table"; then
        pass "Table '$table' exists"
    else
        fail "Table '$table' NOT found"
    fi
done

# ── 4. Go Tests Pass ─────────────────────────────────────────
echo ""
echo "── Check 4: Go Tests Pass ──"
if go test ./... -count=1 -short > /dev/null 2>&1; then
    pass "Go tests pass"
else
    fail "Go tests failed"
    info "Run 'make test-go' for details"
fi

# ── 5. Python Tests Pass ─────────────────────────────────────
echo ""
echo "── Check 5: Python Tests Pass ──"
if pytest tests -q --timeout=120 > /dev/null 2>&1; then
    pass "Python tests pass"
else
    fail "Python tests failed"
    info "Run 'make test-python' for details"
fi

# ── 6. Migration Status ──────────────────────────────────────
echo ""
echo "── Check 6: Migration Status ──"
if migrate_out=$(go run ./cmd/baxi-cli migration status 2>&1); then
    if echo "$migrate_out" | grep -q "nothing new"; then
        pass "Migrations up to date"
    else
        warn "Migrations may not be at latest"
        info "$migrate_out"
    fi
else
    warn "Could not check migration status"
fi

# ── Summary ───────────────────────────────────────────────────
echo ""
echo "═══════════════════════════════════════════════════════════"
if [ "$FAIL" -eq 0 ]; then
    echo -e "  ${GREEN}Phase 1 Verification: ALL PASSED${NC}"
    echo "═══════════════════════════════════════════════════════════"
    exit 0
else
    echo -e "  ${RED}Phase 1 Verification: SOME FAILED${NC}"
    echo "═══════════════════════════════════════════════════════════"
    exit 1
fi
