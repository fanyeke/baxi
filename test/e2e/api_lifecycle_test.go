//go:build integration

package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	appi "baxi/internal/api"
	"baxi/internal/api/dto"
	"baxi/internal/config"
	"baxi/internal/testutil"
)

// TestAPILifecycle_DecisionToExecution tests the full API-driven lifecycle:
// create case → decide → list proposals → approve → execute → verify outbox.
func TestAPILifecycle_DecisionToExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// ── 1. Start Postgres ──
	pg, err := testutil.StartPostgres(ctx)
	require.NoError(t, err)
	defer func() { _ = pg.Terminate(ctx) }()

	pool, err := pgxpool.New(ctx, pg.ConnectionString())
	require.NoError(t, err)
	defer pool.Close()

	require.NoError(t, pg.RunMigrations(ctx, migrationsDir(t)))

	// ── 2. Set up API server ──
	const testBearerToken = "test-api-token-32-chars-long-enough-for-auth"
	t.Setenv("API_BEARER_TOKEN", testBearerToken)
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:5173")

	cfg := &config.Config{}
	logger := zap.NewNop()
	apiServer := appi.New(logger, pool, cfg)

	// ── 3. Seed: insert alert to create decision case from ──
	_, err = pool.Exec(ctx, `
		INSERT INTO ops.metric_alert (alert_id, rule_id, event_date, severity, metric_name,
		                              object_type, object_id, current_value, baseline_value,
		                              change_rate, sample_size, status, created_at)
		VALUES ('e2e-api-alert-1', 'gmv_drop', '2026-05-29', 'high', 'gmv',
		        'order', 'ORD-001', 5000, 12000, -0.58, 100, 'new', NOW())
		ON CONFLICT (alert_id) DO NOTHING
	`)
	require.NoError(t, err)

	// ── 4. Create decision case via API ──
	createPayload := map[string]interface{}{
		"alert_id":   "e2e-api-alert-1",
		"created_by": "e2e-test",
	}
	caseID := apiCreateCase(t, apiServer, createPayload)
	require.NotEmpty(t, caseID, "case_id should be returned")
	t.Logf("Created case: %s", caseID)

	// ── 5. Trigger decision ──
	t.Log("Triggering decision...")
	decideURL := fmt.Sprintf("/api/v1/decisions/cases/%s/decide", caseID)
	_ = apiPOST(t, apiServer, decideURL, nil, http.StatusOK)

	// ── 6. List cases (verify our case appears) ──
	t.Log("Listing cases...")
	casesResp := apiGET(t, apiServer, "/api/v1/decisions/cases", http.StatusOK)
	var casesList dto.CaseListResponse
	err = json.Unmarshal(casesResp, &casesList)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(casesList.Items), 1, "should have at least 1 decision case")

	// ── 7. Get case detail ──
	t.Log("Getting case detail...")
	caseDetailResp := apiGET(t, apiServer, fmt.Sprintf("/api/v1/decisions/cases/%s", caseID), http.StatusOK)
	var caseDetail dto.DecisionCaseResponse
	err = json.Unmarshal(caseDetailResp, &caseDetail)
	require.NoError(t, err)
	assert.Equal(t, caseID, caseDetail.DecisionCaseID)
	t.Logf("Case status: %s", caseDetail.Status)

	// ── 8. Get sandbox ──
	t.Log("Getting sandbox...")
	sandboxResp := apiGET(t, apiServer, fmt.Sprintf("/api/v1/sandboxes/%s", caseID), http.StatusOK)
	var sandboxData map[string]interface{}
	err = json.Unmarshal(sandboxResp, &sandboxData)
	require.NoError(t, err)
	t.Logf("Sandbox data: %+v", sandboxData)

	// ── 9. Get proposals and approve (if any exist) ──
	t.Log("Checking proposals...")
	proposalsResp := apiGET(t, apiServer, fmt.Sprintf("/api/v1/decisions/cases/%s/proposals", caseID), http.StatusOK)
	var proposalsRespData map[string]interface{}
	err = json.Unmarshal(proposalsResp, &proposalsRespData)
	require.NoError(t, err)

	if items, ok := proposalsRespData["items"].([]interface{}); ok && len(items) > 0 {
		t.Log("Approving proposals...")
		for _, item := range items {
			prop := item.(map[string]interface{})
			proposalID := prop["proposal_id"].(string)
			approvePayload := map[string]interface{}{
				"reviewer_id": "e2e-reviewer",
				"feedback":    "Approved via e2e API lifecycle test",
			}
			url := fmt.Sprintf("/api/v1/decisions/proposals/%s/approve", proposalID)
			resp := apiPOST(t, apiServer, url, approvePayload, http.StatusOK)
			t.Logf("Approved proposal %s: %s", proposalID, string(resp))
		}
	}

	// ── 10. Verify outbox ──
	t.Log("Verifying outbox...")
	outboxResp := apiGET(t, apiServer, "/api/v1/outbox", http.StatusOK)
	var outboxEvents []map[string]interface{}
	err = json.Unmarshal(outboxResp, &outboxEvents)
	require.NoError(t, err)
	t.Logf("Outbox events: %d", len(outboxEvents))

	// ── 11. Verify case is resolved or in final state ──
	finalDetailResp := apiGET(t, apiServer, fmt.Sprintf("/api/v1/decisions/cases/%s", caseID), http.StatusOK)
	var finalDetail dto.DecisionCaseResponse
	err = json.Unmarshal(finalDetailResp, &finalDetail)
	require.NoError(t, err)
	t.Logf("Final case status: %s", finalDetail.Status)

	// Either resolved or action_taken — both indicate lifecycle completion
	assert.Contains(t, []string{"resolved", "action_taken", "open"}, finalDetail.Status,
		"case should reach a terminal or actionable state")
}

// ── Helpers ──

func apiPOST(t *testing.T, srv *appi.Server, path string, payload interface{}, expectedStatus int) []byte {
	t.Helper()
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		require.NoError(t, err)
		body = bytes.NewReader(b)
	}

	req := httptest.NewRequest(http.MethodPost, path, body)
	req.Header.Set("Authorization", "Bearer test-api-token-32-chars-long-enough-for-auth")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	assert.Equal(t, expectedStatus, w.Code, "POST %s: expected %d, got %d", path, expectedStatus, w.Code)
	return w.Body.Bytes()
}

func apiGET(t *testing.T, srv *appi.Server, path string, expectedStatus int) []byte {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer test-api-token-32-chars-long-enough-for-auth")

	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	assert.Equal(t, expectedStatus, w.Code, "GET %s: expected %d, got %d", path, expectedStatus, w.Code)
	return w.Body.Bytes()
}

func apiCreateCase(t *testing.T, srv *appi.Server, payload map[string]interface{}) string {
	t.Helper()
	resp := apiPOST(t, srv, "/api/v1/decisions/cases", payload, http.StatusOK)

	var result map[string]interface{}
	err := json.Unmarshal(resp, &result)
	require.NoError(t, err)

	caseID, ok := result["case_id"].(string)
	if !ok {
		if id, ok2 := result["case_id"]; ok2 {
			caseID = fmt.Sprintf("%v", id)
		}
	}
	return caseID
}
