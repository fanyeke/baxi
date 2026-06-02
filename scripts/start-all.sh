#!/usr/bin/env bash
# Start all baxi services: PostgreSQL, API server, Worker
# MCP server is started lazily by Pi Agent on first tool call
set -euo pipefail

cd "$(dirname "$0")/.."

echo "=== 1. Starting PostgreSQL ==="
docker compose up -d postgres

echo "=== 2. Running migrations ==="
DATABASE_URL="postgres://baxi:baxi_dev@localhost:5432/baxi?sslmode=disable"
goose -dir migrations postgres "$DATABASE_URL" up 2>/dev/null || true

echo "=== 3. Building MCP server ==="
go build -o /tmp/baxi-mcp ./cmd/baxi-mcp/

echo "=== 4. Starting API server (port 8080) ==="
echo "   PID will be saved to .baxi-api.pid"
nohup go run ./cmd/baxi-api > /tmp/baxi-api.log 2>&1 &
echo $! > .baxi-api.pid

echo "=== 5. Starting Worker ==="
nohup go run ./cmd/baxi-worker > /tmp/baxi-worker.log 2>&1 &
echo $! > .baxi-worker.pid

echo ""
echo "=== Done ==="
echo "PostgreSQL :5432    ✅"
echo "API :8080          ✅ (PID: $(cat .baxi-api.pid))"
echo "Worker             ✅ (PID: $(cat .baxi-worker.pid))"
echo "MCP /tmp/baxi-mcp  ✅ (built, lazy-start by Pi Agent)"
echo ""
echo "To stop:  kill \$(cat .baxi-api.pid) \$(cat .baxi-worker.pid)"
echo "Logs:     tail -f /tmp/baxi-api.log"
