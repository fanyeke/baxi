#!/bin/bash
# E2E Test Script for Baxi MCP Integration
# Records response times, results, and performance metrics

set -e

# Configuration
export DATABASE_URL="postgres://baxi:baxi_dev@localhost:5432/baxi?sslmode=disable"
export API_BEARER_TOKEN="test-token-12345678901234567890"
export LOG_LEVEL="info"
export API_PORT="8080"

LOG_DIR="/home/zzz/project/baxi/logs/e2e-test"
mkdir -p "$LOG_DIR"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LOG_FILE="$LOG_DIR/e2e_test_$TIMESTAMP.log"
METRICS_FILE="$LOG_DIR/metrics_$TIMESTAMP.json"

echo "========================================" | tee -a "$LOG_FILE"
echo "Baxi MCP E2E Test - $(date)" | tee -a "$LOG_FILE"
echo "========================================" | tee -a "$LOG_FILE"

# Function to measure execution time
measure_time() {
    local start_time=$(date +%s%N)
    "$@"
    local exit_code=$?
    local end_time=$(date +%s%N)
    local duration_ms=$(( (end_time - start_time) / 1000000 ))
    echo "$duration_ms"
    return $exit_code
}

# Function to run MCP tool and capture results
run_mcp_tool() {
    local tool_name="$1"
    local args="$2"
    local test_name="$3"
    
    echo "" | tee -a "$LOG_FILE"
    echo "[$test_name] Testing: $tool_name" | tee -a "$LOG_FILE"
    echo "  Args: $args" | tee -a "$LOG_FILE"
    
    local start_time=$(date +%s%N)
    
    local result=$(printf '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"e2e-test","version":"1.0"}}}\n{"jsonrpc":"2.0","method":"notifications/initialized"}\n{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"%s","arguments":%s}}\n' "$tool_name" "$args" | timeout 30 ./baxi-mcp 2>/dev/null | tail -1)
    
    local end_time=$(date +%s%N)
    local duration_ms=$(( (end_time - start_time) / 1000000 ))
    
    # Parse result
    local is_error=$(echo "$result" | python3 -c "import sys,json; d=json.loads(sys.stdin.read()); print(d.get('result',{}).get('isError', False))" 2>/dev/null || echo "parse_error")
    local content=$(echo "$result" | python3 -c "import sys,json; d=json.loads(sys.stdin.read()); print(d.get('result',{}).get('content',[{}])[0].get('text','{}')[:200])" 2>/dev/null || echo "parse_error")
    
    echo "  Duration: ${duration_ms}ms" | tee -a "$LOG_FILE"
    echo "  Is Error: $is_error" | tee -a "$LOG_FILE"
    echo "  Content (first 200 chars): $content" | tee -a "$LOG_FILE"
    
    # Save to metrics file
    echo "{\"test\":\"$test_name\",\"tool\":\"$tool_name\",\"args\":$args,\"duration_ms\":$duration_ms,\"is_error\":$is_error,\"timestamp\":\"$(date -Iseconds)\"}" >> "$METRICS_FILE"
}

# Start API server in background
echo "" | tee -a "$LOG_FILE"
echo "Starting Baxi API server..." | tee -a "$LOG_FILE"
./baxi-api &
API_PID=$!
sleep 2

# Check if API is running
if kill -0 $API_PID 2>/dev/null; then
    echo "API server started (PID: $API_PID)" | tee -a "$LOG_FILE"
else
    echo "Failed to start API server" | tee -a "$LOG_FILE"
    exit 1
fi

# Test 1: List Alerts
run_mcp_tool "list_alerts" '{"limit":5}' "T01_ListAlerts"

# Test 2: List Cases
run_mcp_tool "list_cases" '{"limit":5}' "T02_ListCases"

# Test 3: Check Access
run_mcp_tool "check_access" '{"role":"admin","object_type":"order","action":"read"}' "T03_CheckAccess"

# Test 4: Get Classification
run_mcp_tool "get_classification" '{"field_path":"customer.email"}' "T04_GetClassification"

# Test 5: Get Case (if exists)
run_mcp_tool "get_case" '{"case_id":"dc_1779986531_PSvyYI"}' "T05_GetCase"

# Test 6: List Proposals (if case exists)
run_mcp_tool "list_proposals" '{"case_id":"dc_1779986531_PSvyYI"}' "T06_ListProposals"

# Test 7: Create Decision Case
run_mcp_tool "create_decision_case" '{"alert_id":"dim-76085bfcd31d","created_by":"e2e_test"}' "T07_CreateCase"

# Test 8: Run Pipeline (stub)
run_mcp_tool "run_pipeline" '{"config":"daily"}' "T08_RunPipeline"

# Stop API server
echo "" | tee -a "$LOG_FILE"
echo "Stopping API server..." | tee -a "$LOG_FILE"
kill $API_PID 2>/dev/null || true
wait $API_PID 2>/dev/null || true

# Summary
echo "" | tee -a "$LOG_FILE"
echo "========================================" | tee -a "$LOG_FILE"
echo "Test Summary" | tee -a "$LOG_FILE"
echo "========================================" | tee -a "$LOG_FILE"

# Calculate metrics
if [ -f "$METRICS_FILE" ]; then
    total_tests=$(wc -l < "$METRICS_FILE")
    failed_tests=$(grep -c '"is_error":true' "$METRICS_FILE" || echo "0")
    avg_duration=$(python3 -c "
import json
durations = []
with open('$METRICS_FILE') as f:
    for line in f:
        if line.strip():
            data = json.loads(line)
            durations.append(data['duration_ms'])
if durations:
    print(f'{sum(durations)/len(durations):.0f}')
else:
    print('0')
")
    max_duration=$(python3 -c "
import json
durations = []
with open('$METRICS_FILE') as f:
    for line in f:
        if line.strip():
            data = json.loads(line)
            durations.append(data['duration_ms'])
if durations:
    print(max(durations))
else:
    print('0')
")
    min_duration=$(python3 -c "
import json
durations = []
with open('$METRICS_FILE') as f:
    for line in f:
        if line.strip():
            data = json.loads(line)
            durations.append(data['duration_ms'])
if durations:
    print(min(durations))
else:
    print('0')
")
    
    echo "Total tests: $total_tests" | tee -a "$LOG_FILE"
    echo "Failed tests: $failed_tests" | tee -a "$LOG_FILE"
    echo "Success rate: $(( (total_tests - failed_tests) * 100 / total_tests ))%" | tee -a "$LOG_FILE"
    echo "Avg response time: ${avg_duration}ms" | tee -a "$LOG_FILE"
    echo "Min response time: ${min_duration}ms" | tee -a "$LOG_FILE"
    echo "Max response time: ${max_duration}ms" | tee -a "$LOG_FILE"
fi

echo "" | tee -a "$LOG_FILE"
echo "Log file: $LOG_FILE" | tee -a "$LOG_FILE"
echo "Metrics file: $METRICS_FILE" | tee -a "$LOG_FILE"
echo "========================================" | tee -a "$LOG_FILE"
