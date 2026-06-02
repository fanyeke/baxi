package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/api/dto"
	"baxi/internal/review"
)

type mockSandboxService struct {
	createSandboxFn     func(ctx context.Context, caseID string, data map[string]interface{}) (string, error)
	getSandboxFn        func(ctx context.Context, sandboxID string) (*review.Sandbox, error)
	listSandboxesFn     func(ctx context.Context) ([]review.Sandbox, error)
	addProposalToSBFn   func(ctx context.Context, sandboxID, proposalID string) error
	compareSandboxesFn  func(ctx context.Context, sandboxID1, sandboxID2 string) (*review.ComparisonResult, error)
}

func (m *mockSandboxService) CreateSandbox(ctx context.Context, caseID string, data map[string]interface{}) (string, error) {
	if m.createSandboxFn != nil {
		return m.createSandboxFn(ctx, caseID, data)
	}
	return "", errors.New("unexpected call to CreateSandbox")
}

func (m *mockSandboxService) GetSandbox(ctx context.Context, sandboxID string) (*review.Sandbox, error) {
	if m.getSandboxFn != nil {
		return m.getSandboxFn(ctx, sandboxID)
	}
	return nil, errors.New("unexpected call to GetSandbox")
}

func (m *mockSandboxService) ListSandboxes(ctx context.Context) ([]review.Sandbox, error) {
	if m.listSandboxesFn != nil {
		return m.listSandboxesFn(ctx)
	}
	return nil, errors.New("unexpected call to ListSandboxes")
}

func (m *mockSandboxService) AddProposalToSandbox(ctx context.Context, sandboxID, proposalID string) error {
	if m.addProposalToSBFn != nil {
		return m.addProposalToSBFn(ctx, sandboxID, proposalID)
	}
	return errors.New("unexpected call to AddProposalToSandbox")
}

func (m *mockSandboxService) CompareSandbox(ctx context.Context, sandboxID1, sandboxID2 string) (*review.ComparisonResult, error) {
	if m.compareSandboxesFn != nil {
		return m.compareSandboxesFn(ctx, sandboxID1, sandboxID2)
	}
	return nil, errors.New("unexpected call to CompareSandbox")
}

func TestSandboxHandler_Create_201(t *testing.T) {
	svc := &mockSandboxService{
		createSandboxFn: func(ctx context.Context, caseID string, data map[string]interface{}) (string, error) {
			assert.Equal(t, "case-1", caseID)
			assert.Equal(t, "hello", data["key"])
			return "sbx_123", nil
		},
	}
	h := NewSandboxHandler(svc)

	body := `{"case_id":"case-1","data":{"key":"hello"}}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/sandboxes", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.HandleCreate(w, r)

	require.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]string
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "sbx_123", resp["sandbox_id"])
}

func TestSandboxHandler_Create_MissingCaseID(t *testing.T) {
	h := NewSandboxHandler(&mockSandboxService{})

	body := `{"data":{"key":"hello"}}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/sandboxes", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.HandleCreate(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSandboxHandler_Create_EmptyCaseID(t *testing.T) {
	h := NewSandboxHandler(&mockSandboxService{})

	body := `{"case_id":"","data":{}}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/sandboxes", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.HandleCreate(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSandboxHandler_Create_InvalidBody(t *testing.T) {
	h := NewSandboxHandler(&mockSandboxService{})

	r := httptest.NewRequest(http.MethodPost, "/api/v1/sandboxes", strings.NewReader("not json"))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.HandleCreate(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSandboxHandler_Create_ServiceError(t *testing.T) {
	svc := &mockSandboxService{
		createSandboxFn: func(ctx context.Context, caseID string, data map[string]interface{}) (string, error) {
			return "", errors.New("db error")
		},
	}
	h := NewSandboxHandler(svc)

	body := `{"case_id":"case-1","data":{"key":"hello"}}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/sandboxes", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.HandleCreate(w, r)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSandboxHandler_Get_200(t *testing.T) {
	now := time.Now()
	svc := &mockSandboxService{
		getSandboxFn: func(ctx context.Context, sandboxID string) (*review.Sandbox, error) {
			assert.Equal(t, "sbx_123", sandboxID)
			return &review.Sandbox{
				SandboxID:    "sbx_123",
				CaseID:       "case-1",
				SandboxData:  map[string]interface{}{"key": "value"},
				Status:       "draft",
				ComparedWith: []string{},
				CreatedAt:    now,
			}, nil
		},
	}
	h := NewSandboxHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/sandboxes/sbx_123", nil)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("id", "sbx_123")
	w := httptest.NewRecorder()
	h.HandleGet(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var resp dto.SandboxResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "sbx_123", resp.SandboxID)
	assert.Equal(t, "case-1", resp.CaseID)
	assert.Equal(t, "draft", resp.Status)
	assert.Equal(t, "value", resp.Data["key"])
}

func TestSandboxHandler_Get_NotFound(t *testing.T) {
	svc := &mockSandboxService{
		getSandboxFn: func(ctx context.Context, sandboxID string) (*review.Sandbox, error) {
			return nil, nil
		},
	}
	h := NewSandboxHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/sandboxes/nonexistent", nil)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("id", "nonexistent")
	w := httptest.NewRecorder()
	h.HandleGet(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestSandboxHandler_Get_ServiceError(t *testing.T) {
	svc := &mockSandboxService{
		getSandboxFn: func(ctx context.Context, sandboxID string) (*review.Sandbox, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewSandboxHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/sandboxes/sbx_123", nil)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("id", "sbx_123")
	w := httptest.NewRecorder()
	h.HandleGet(w, r)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSandboxHandler_List_200(t *testing.T) {
	now := time.Now()
	svc := &mockSandboxService{
		listSandboxesFn: func(ctx context.Context) ([]review.Sandbox, error) {
			return []review.Sandbox{
				{
					SandboxID:   "sbx_1",
					CaseID:      "case-1",
					SandboxData: map[string]interface{}{"key": "v1"},
					Status:      "draft",
					CreatedAt:   now,
				},
				{
					SandboxID:   "sbx_2",
					CaseID:      "case-2",
					SandboxData: map[string]interface{}{"key": "v2"},
					Status:      "reviewing",
					CreatedAt:   now.Add(-time.Hour),
				},
			}, nil
		},
	}
	h := NewSandboxHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/sandboxes", nil)
	w := httptest.NewRecorder()
	h.HandleList(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var resp dto.SandboxListResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	require.Len(t, resp.Items, 2)
	assert.Equal(t, "sbx_1", resp.Items[0].SandboxID)
	assert.Equal(t, "sbx_2", resp.Items[1].SandboxID)
}

func TestSandboxHandler_List_Empty(t *testing.T) {
	svc := &mockSandboxService{
		listSandboxesFn: func(ctx context.Context) ([]review.Sandbox, error) {
			return []review.Sandbox{}, nil
		},
	}
	h := NewSandboxHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/sandboxes", nil)
	w := httptest.NewRecorder()
	h.HandleList(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var resp dto.SandboxListResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Empty(t, resp.Items)
}

func TestSandboxHandler_List_ServiceError(t *testing.T) {
	svc := &mockSandboxService{
		listSandboxesFn: func(ctx context.Context) ([]review.Sandbox, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewSandboxHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/sandboxes", nil)
	w := httptest.NewRecorder()
	h.HandleList(w, r)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSandboxHandler_AddProposal_200(t *testing.T) {
	svc := &mockSandboxService{
		addProposalToSBFn: func(ctx context.Context, sandboxID, proposalID string) error {
			assert.Equal(t, "sbx_123", sandboxID)
			assert.Equal(t, "ap_1", proposalID)
			return nil
		},
	}
	h := NewSandboxHandler(svc)

	body := `{"proposal_id":"ap_1"}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/sandboxes/sbx_123/proposals", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("id", "sbx_123")
	w := httptest.NewRecorder()
	h.HandleAddProposal(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp["status"])
}

func TestSandboxHandler_AddProposal_MissingProposalID(t *testing.T) {
	h := NewSandboxHandler(&mockSandboxService{})

	body := `{}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/sandboxes/sbx_123/proposals", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("id", "sbx_123")
	w := httptest.NewRecorder()
	h.HandleAddProposal(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSandboxHandler_AddProposal_SandboxNotFound(t *testing.T) {
	svc := &mockSandboxService{
		addProposalToSBFn: func(ctx context.Context, sandboxID, proposalID string) error {
			return errors.New("sandbox nonexistent not found")
		},
	}
	h := NewSandboxHandler(svc)

	body := `{"proposal_id":"ap_1"}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/sandboxes/nonexistent/proposals", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("id", "nonexistent")
	w := httptest.NewRecorder()
	h.HandleAddProposal(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestSandboxHandler_AddProposal_ServiceError(t *testing.T) {
	svc := &mockSandboxService{
		addProposalToSBFn: func(ctx context.Context, sandboxID, proposalID string) error {
			return errors.New("db error")
		},
	}
	h := NewSandboxHandler(svc)

	body := `{"proposal_id":"ap_1"}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/sandboxes/sbx_123/proposals", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("id", "sbx_123")
	w := httptest.NewRecorder()
	h.HandleAddProposal(w, r)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSandboxHandler_Compare_200(t *testing.T) {
	svc := &mockSandboxService{
		compareSandboxesFn: func(ctx context.Context, sandboxID1, sandboxID2 string) (*review.ComparisonResult, error) {
			assert.Equal(t, "sbx_1", sandboxID1)
			assert.Equal(t, "sbx_2", sandboxID2)
			return &review.ComparisonResult{
				Sandbox1ID: "sbx_1",
				Sandbox2ID: "sbx_2",
				Differences: []review.Difference{
					{Field: "price", Value1: 100.0, Value2: 90.0},
				},
			}, nil
		},
	}
	h := NewSandboxHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/sandboxes/compare?sandbox1_id=sbx_1&sandbox2_id=sbx_2", nil)
	w := httptest.NewRecorder()
	h.HandleCompare(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var resp dto.ComparisonResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "sbx_1", resp.Sandbox1ID)
	assert.Equal(t, "sbx_2", resp.Sandbox2ID)
	require.Len(t, resp.Differences, 1)
	assert.Equal(t, "price", resp.Differences[0].Field)
}

func TestSandboxHandler_Compare_MissingParams(t *testing.T) {
	h := NewSandboxHandler(&mockSandboxService{})

	r := httptest.NewRequest(http.MethodGet, "/api/v1/sandboxes/compare", nil)
	w := httptest.NewRecorder()
	h.HandleCompare(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSandboxHandler_Compare_MissingSecondParam(t *testing.T) {
	h := NewSandboxHandler(&mockSandboxService{})

	r := httptest.NewRequest(http.MethodGet, "/api/v1/sandboxes/compare?sandbox1_id=sbx_1", nil)
	w := httptest.NewRecorder()
	h.HandleCompare(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSandboxHandler_Compare_SandboxNotFound(t *testing.T) {
	svc := &mockSandboxService{
		compareSandboxesFn: func(ctx context.Context, sandboxID1, sandboxID2 string) (*review.ComparisonResult, error) {
			return nil, errors.New("sandbox nonexistent not found")
		},
	}
	h := NewSandboxHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/sandboxes/compare?sandbox1_id=nonexistent&sandbox2_id=sbx_2", nil)
	w := httptest.NewRecorder()
	h.HandleCompare(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestSandboxHandler_Compare_ServiceError(t *testing.T) {
	svc := &mockSandboxService{
		compareSandboxesFn: func(ctx context.Context, sandboxID1, sandboxID2 string) (*review.ComparisonResult, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewSandboxHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/sandboxes/compare?sandbox1_id=sbx_1&sandbox2_id=sbx_2", nil)
	w := httptest.NewRecorder()
	h.HandleCompare(w, r)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}
