#!/usr/bin/env bash
# verify_phase2.sh - Phase 2 Verification: Go-Primary Read
#
# Verifies that frontend is connected to Go API, shadow mode works,
# and Go API serves correct responses for read endpoints.
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
echo "  Phase 2 Verification: Go-Primary Read"
echo "═══════════════════════════════════════════════════════════"
echo ""

# ── 1. Go API Healthy ─────────────────────────────────────────
echo "── Check 1: Go API Health ──"
if resp=$(curl -sf --max-time "$TIMEOUT" "${GO_API_URL}/api/v1/health" 2>/dev/null); then
    pass "Go API is healthy"
    info "Response: $resp"
else
    fail "Go API not reachable at ${GO_API_URL}"
fi

# ── 2. Frontend Reachable ────────────────────────────────────
echo ""
echo "── Check 2: Frontend Reachable ──"
if curl -sf --max-time "$TIMEOUT" "${FRONTEND_URL}" > /dev/null 2>&1; then
    pass "Frontend reachable at ${FRONTEND_URL}"
else
    warn "Frontend not reachable (may be intentional if not running)"
fi

# ── 3. Go API Serves Alerts ──────────────────────────────────
echo ""
echo "── Check 3: Go API Serves Alerts ──"
if resp=$(curl -sf --max-time "$TIMEOUT" -H "Authorization: Bearer ${API_BEARER_TOKEN}" "${GO_API_URL}/api/v1/alerts" 2>/dev/null); then
    if echo "$resp" | python3 -c "import sys,json; d=json.load(sys.stdin); assert isinstance(d, (list, dict))" 2>/dev/null; then
        pass "Go API serves alerts endpoint"
    else
        fail "Go API alerts response is not valid JSON"
    fi
else
    fail "Go API alerts endpoint failed"
fi

# ── 4. Go API Serves Tasks ───────────────────────────────────
echo ""
echo "── Check 4: Go API Serves Tasks ──"
if resp=$(curl -sf --max-time "$TIMEOUT" -H "Authorization: Bearer ${API_BEARER_TOKEN}" "${GO_API_URL}/api/v1/tasks" 2>/dev/null); then
    if echo "$resp" | python3 -c "import sys,json; d=json.load(sys.stdin); assert isinstance(d, (list, dict))" 2>/dev/null; then
        pass "Go API serves tasks endpoint"
    else
        fail "Go API tasks response is not valid JSON"
    fi
else
    fail "Go API tasks endpoint failed"
fi

# ── 5. Go API Serves Status ──────────────────────────────────
echo ""
echo "── Check 5: Go API Serves Status ──"
if resp=$(curl -sf --max-time "$TIMEOUT" "${GO_API_URL}/api/v1/status" 2>/dev/null); then
    if echo "$resp" | python3 -c "import sys,json; d=json.load(sys.stdin); assert 'db_connected' in d" 2>/dev/null; then
        pass "Go API serves status endpoint"
        info "Response: $resp"
    else
        fail "Go API status response missing db_connected field"
    fi
else
    fail "Go API status endpoint failed"
fi

# ── 6. Python API Shadow Mode ────────────────────────────────
echo ""
echo "── Check 6: Python API Shadow Mode ──"
if curl -sf --max-time "$TIMEOUT" "${PYTHON_API_URL}/api/v1/health" > /dev/null 2>&1; then
    pass "Python API still available (shadow mode)"
else
    warn "Python API not reachable (may be expected if shadow mode disabled)"
fi

# ── 7. Feature Flag Check ────────────────────────────────────
echo ""
echo "── Check 7: Feature Flags ──"
# Check if Go API has feature flags endpoint or env var
if [ -n "${USE_GO_API:-}" ] && [ "${USE_GO_API:-false}" = "true" ]; then
    pass "USE_GO_API is enabled"
else
    warn "USE_GO_API not set or false (may be intentional for gradual migration)"
fi

# ── 8. Database Consistency ──────────────────────────────────
echo ""
echo "── Check 8: Database Read Consistency ──"
if psql "$DATABASE_URL" -c "SELECT COUNT(*) FROM dwd_order_level" > /dev/null 2>&1; then
    count=$(psql "$DATABASE_URL" -tAc "SELECT COUNT(*) FROM dwd_order_level" 2>/dev/null)
    if [ "$count" -gt 0 ]; then
        pass "dwd_order_level has $count rows"
    else
        fail "dwd_order_level is empty"
    fi
else
    fail "Cannot query dwd_order_level"
fi

# ── Summary ───────────────────────────────────────────────────
echo ""
echo "═══════════════════════════════════════════════════════════"
if [ "$FAIL" -eq 0 ]; then
    echo -e "  ${GREEN}Phase 2 Verification: ALL PASSED${NC}"
    echo "═══════════════════════════════════════════════════════════"
    exit 0
else
    echo -e "  ${RED}Phase 2 Verification: SOME FAILED${NC}"
    echo "═══════════════════════════════════════════════════════════"
    exit 1
fi
