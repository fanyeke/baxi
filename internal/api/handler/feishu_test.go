package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockFeishuService implements FeishuService for testing.
type mockFeishuService struct {
	exportFn       func(ctx context.Context, tables []string, apply bool) (*feishuResponse, error)
	syncFn         func(ctx context.Context, tables []string, apply bool) (*feishuResponse, error)
	statusImportFn func(ctx context.Context, tables []string, apply bool) (*feishuResponse, error)
}

func (m *mockFeishuService) Export(ctx context.Context, tables []string, apply bool) (*feishuResponse, error) {
	if m.exportFn != nil {
		return m.exportFn(ctx, tables, apply)
	}
	return &feishuResponse{Status: "dry_run", Tables: []FeishuTableResult{{Name: "default", Status: "dry_run"}}}, nil
}

func (m *mockFeishuService) Sync(ctx context.Context, tables []string, apply bool) (*feishuResponse, error) {
	if m.syncFn != nil {
		return m.syncFn(ctx, tables, apply)
	}
	return &feishuResponse{Status: "dry_run", Tables: []FeishuTableResult{{Name: "default", Status: "dry_run"}}}, nil
}

func (m *mockFeishuService) StatusImport(ctx context.Context, tables []string, apply bool) (*feishuResponse, error) {
	if m.statusImportFn != nil {
		return m.statusImportFn(ctx, tables, apply)
	}
	return &feishuResponse{Status: "dry_run", Tables: []FeishuTableResult{{Name: "default", Status: "dry_run"}}}, nil
}

func newFeishuRequest(method, path, body string) *http.Request {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	return r
}

// ── HandleExport ─────────────────────────────────────────────────────

func TestFeishuHandler_HandleExport_DryRun(t *testing.T) {
	mock := &mockFeishuService{}
	h := NewFeishuHandler(mock)

	body := `{"tables": ["table1"], "apply": false}`
	r := newFeishuRequest(http.MethodPost, "/api/v1/feishu/export", body)
	w := httptest.NewRecorder()
	h.HandleExport(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp feishuResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "dry_run", resp.Status)
	assert.Len(t, resp.Tables, 1)
}

func TestFeishuHandler_HandleExport_RealExecution(t *testing.T) {
	mock := &mockFeishuService{
		exportFn: func(ctx context.Context, tables []string, apply bool) (*feishuResponse, error) {
			assert.True(t, apply)
			return &feishuResponse{
				Status:  "ok",
				Message: "exported successfully",
				Tables:  []FeishuTableResult{{Name: "table1", Status: "ok", Rows: 42}},
			}, nil
		},
	}
	h := NewFeishuHandler(mock)

	body := `{"tables": ["table1"], "apply": true}`
	r := newFeishuRequest(http.MethodPost, "/api/v1/feishu/export", body)
	w := httptest.NewRecorder()
	h.HandleExport(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp feishuResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
	assert.Equal(t, "exported successfully", resp.Message)
	assert.Len(t, resp.Tables, 1)
	assert.Equal(t, 42, resp.Tables[0].Rows)
}

func TestFeishuHandler_HandleExport_Failure(t *testing.T) {
	mock := &mockFeishuService{
		exportFn: func(ctx context.Context, tables []string, apply bool) (*feishuResponse, error) {
			return &feishuResponse{
				Status:  "failed",
				Message: "permission denied",
				Tables:  []FeishuTableResult{{Name: "table1", Status: "failed", Rows: 0}},
			}, nil
		},
	}
	h := NewFeishuHandler(mock)

	body := `{"tables": ["table1"], "apply": true}`
	r := newFeishuRequest(http.MethodPost, "/api/v1/feishu/export", body)
	w := httptest.NewRecorder()
	h.HandleExport(w, r)

	require.Equal(t, http.StatusInternalServerError, w.Code)

	var resp feishuResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "failed", resp.Status)
}

func TestFeishuHandler_HandleExport_MultipleTables(t *testing.T) {
	mock := &mockFeishuService{
		exportFn: func(ctx context.Context, tables []string, apply bool) (*feishuResponse, error) {
			assert.ElementsMatch(t, []string{"users", "orders", "products"}, tables)
			return &feishuResponse{
				Status: "ok",
				Tables: []FeishuTableResult{
					{Name: "users", Status: "ok", Rows: 100},
					{Name: "orders", Status: "ok", Rows: 250},
					{Name: "products", Status: "ok", Rows: 75},
				},
			}, nil
		},
	}
	h := NewFeishuHandler(mock)

	body := `{"tables": ["users", "orders", "products"], "apply": false}`
	r := newFeishuRequest(http.MethodPost, "/api/v1/feishu/export", body)
	w := httptest.NewRecorder()
	h.HandleExport(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp feishuResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Len(t, resp.Tables, 3)
	assert.Equal(t, "ok", resp.Tables[0].Status)
}

func TestFeishuHandler_HandleExport_NoTables(t *testing.T) {
	mock := &mockFeishuService{
		exportFn: func(ctx context.Context, tables []string, apply bool) (*feishuResponse, error) {
			assert.Empty(t, tables)
			return &feishuResponse{Status: "ok", Tables: nil}, nil
		},
	}
	h := NewFeishuHandler(mock)

	body := `{"tables": [], "apply": true}`
	r := newFeishuRequest(http.MethodPost, "/api/v1/feishu/export", body)
	w := httptest.NewRecorder()
	h.HandleExport(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp feishuResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
}

func TestFeishuHandler_HandleExport_EmptyBody(t *testing.T) {
	mock := &mockFeishuService{}
	h := NewFeishuHandler(mock)

	r := newFeishuRequest(http.MethodPost, "/api/v1/feishu/export", `{}`)
	w := httptest.NewRecorder()
	h.HandleExport(w, r)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestFeishuHandler_HandleExport_InvalidBody(t *testing.T) {
	mock := &mockFeishuService{}
	h := NewFeishuHandler(mock)

	r := newFeishuRequest(http.MethodPost, "/api/v1/feishu/export", `not-json`)
	w := httptest.NewRecorder()
	h.HandleExport(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["message"].(string), "invalid request body")
}

func TestFeishuHandler_HandleExport_ServiceError(t *testing.T) {
	mock := &mockFeishuService{
		exportFn: func(ctx context.Context, tables []string, apply bool) (*feishuResponse, error) {
			return nil, assert.AnError
		},
	}
	h := NewFeishuHandler(mock)

	body := `{"tables": ["table1"], "apply": true}`
	r := newFeishuRequest(http.MethodPost, "/api/v1/feishu/export", body)
	w := httptest.NewRecorder()
	h.HandleExport(w, r)

	require.Equal(t, http.StatusInternalServerError, w.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["message"].(string), "failed to export")
}

// ── HandleSync ──────────────────────────────────────────────────────

func TestFeishuHandler_HandleSync_DryRun(t *testing.T) {
	mock := &mockFeishuService{}
	h := NewFeishuHandler(mock)

	body := `{"tables": ["table1"], "apply": false}`
	r := newFeishuRequest(http.MethodPost, "/api/v1/feishu/sync", body)
	w := httptest.NewRecorder()
	h.HandleSync(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp feishuResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "dry_run", resp.Status)
}

func TestFeishuHandler_HandleSync_RealExecution(t *testing.T) {
	mock := &mockFeishuService{
		syncFn: func(ctx context.Context, tables []string, apply bool) (*feishuResponse, error) {
			assert.True(t, apply)
			return &feishuResponse{Status: "ok", Message: "synced", Tables: []FeishuTableResult{{Name: "t1", Status: "ok", Rows: 50}}}, nil
		},
	}
	h := NewFeishuHandler(mock)

	body := `{"tables": ["t1"], "apply": true}`
	r := newFeishuRequest(http.MethodPost, "/api/v1/feishu/sync", body)
	w := httptest.NewRecorder()
	h.HandleSync(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp feishuResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
	assert.Equal(t, "synced", resp.Message)
}

func TestFeishuHandler_HandleSync_InvalidBody(t *testing.T) {
	mock := &mockFeishuService{}
	h := NewFeishuHandler(mock)

	r := newFeishuRequest(http.MethodPost, "/api/v1/feishu/sync", `bad-json`)
	w := httptest.NewRecorder()
	h.HandleSync(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFeishuHandler_HandleSync_ServiceError(t *testing.T) {
	mock := &mockFeishuService{
		syncFn: func(ctx context.Context, tables []string, apply bool) (*feishuResponse, error) {
			return nil, assert.AnError
		},
	}
	h := NewFeishuHandler(mock)

	body := `{"tables": ["t1"], "apply": true}`
	r := newFeishuRequest(http.MethodPost, "/api/v1/feishu/sync", body)
	w := httptest.NewRecorder()
	h.HandleSync(w, r)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

// ── HandleStatusImport ──────────────────────────────────────────────

func TestFeishuHandler_HandleStatusImport_DryRun(t *testing.T) {
	mock := &mockFeishuService{}
	h := NewFeishuHandler(mock)

	body := `{"tables": ["table1"], "apply": false}`
	r := newFeishuRequest(http.MethodPost, "/api/v1/feishu/status/import", body)
	w := httptest.NewRecorder()
	h.HandleStatusImport(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp feishuResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "dry_run", resp.Status)
}

func TestFeishuHandler_HandleStatusImport_RealExecution(t *testing.T) {
	mock := &mockFeishuService{
		statusImportFn: func(ctx context.Context, tables []string, apply bool) (*feishuResponse, error) {
			assert.True(t, apply)
			return &feishuResponse{Status: "ok", Tables: []FeishuTableResult{{Name: "t1", Status: "imported", Rows: 10}}}, nil
		},
	}
	h := NewFeishuHandler(mock)

	body := `{"tables": ["t1"], "apply": true}`
	r := newFeishuRequest(http.MethodPost, "/api/v1/feishu/status/import", body)
	w := httptest.NewRecorder()
	h.HandleStatusImport(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp feishuResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
	assert.Equal(t, "imported", resp.Tables[0].Status)
}
