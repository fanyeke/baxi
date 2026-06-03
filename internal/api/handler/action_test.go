package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/action"
	"baxi/internal/api/middleware"
)

type mockActionService struct {
	executeFn       func(ctx context.Context, pool *pgxpool.Pool, proposalID, actorID string, opts ...action.ExecuteOption) (*action.ExecutionResult, error)
	getProposalByID func(ctx context.Context, pool *pgxpool.Pool, proposalID string) (*action.ActionProposal, error)
}

func (m *mockActionService) ExecuteProposal(ctx context.Context, pool *pgxpool.Pool, proposalID, actorID string, opts ...action.ExecuteOption) (*action.ExecutionResult, error) {
	if m.executeFn != nil {
		return m.executeFn(ctx, pool, proposalID, actorID, opts...)
	}
	return nil, errors.New("unexpected call to ExecuteProposal")
}

func (m *mockActionService) GetProposalByID(ctx context.Context, pool *pgxpool.Pool, proposalID string) (*action.ActionProposal, error) {
	if m.getProposalByID != nil {
		return m.getProposalByID(ctx, pool, proposalID)
	}
	return nil, errors.New("unexpected call to GetProposalByID")
}

func newTestRequest(method, path, body string) *http.Request {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	return r
}

func setURLParam(r *http.Request, key, value string) {
	chi.RouteContext(r.Context()).URLParams.Add(key, value)
}

func withActor(r *http.Request, actor string) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.ActorKey, actor)
	return r.WithContext(ctx)
}

func TestActionHandler_HandleExecute_200_ApprovedProposal(t *testing.T) {
	svc := &mockActionService{
		executeFn: func(ctx context.Context, pool *pgxpool.Pool, proposalID, actorID string, opts ...action.ExecuteOption) (*action.ExecutionResult, error) {
			assert.Equal(t, "prop-1", proposalID)
			assert.Equal(t, "qoder", actorID)
			return &action.ExecutionResult{Success: true, DryRun: true}, nil
		},
	}
	h := NewActionHandler(svc, nil)

	body := `{"dry_run": true}`
	r := newTestRequest(http.MethodPost, "/api/v1/proposals/prop-1/execute", body)
	setURLParam(r, "id", "prop-1")
	r = withActor(r, "qoder")
	w := httptest.NewRecorder()
	h.HandleExecute(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp executeResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "prop-1", resp.ProposalID)
	assert.Equal(t, "approved", resp.ApplyStatus)
	assert.True(t, resp.DryRun)
}

func TestActionHandler_HandleExecute_403_NotApproved(t *testing.T) {
	svc := &mockActionService{
		executeFn: func(ctx context.Context, pool *pgxpool.Pool, proposalID, actorID string, opts ...action.ExecuteOption) (*action.ExecutionResult, error) {
			return nil, action.ErrNotApproved
		},
	}
	h := NewActionHandler(svc, nil)

	r := newTestRequest(http.MethodPost, "/api/v1/proposals/prop-1/execute", "{}")
	setURLParam(r, "id", "prop-1")
	w := httptest.NewRecorder()
	h.HandleExecute(w, r)

	require.Equal(t, http.StatusForbidden, w.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["message"].(string), "not approved")
}

func TestActionHandler_HandleExecute_403_ActionNotAllowed(t *testing.T) {
	svc := &mockActionService{
		executeFn: func(ctx context.Context, pool *pgxpool.Pool, proposalID, actorID string, opts ...action.ExecuteOption) (*action.ExecutionResult, error) {
			return nil, action.ErrActionNotAllowed
		},
	}
	h := NewActionHandler(svc, nil)

	r := newTestRequest(http.MethodPost, "/api/v1/proposals/prop-1/execute", "{}")
	setURLParam(r, "id", "prop-1")
	w := httptest.NewRecorder()
	h.HandleExecute(w, r)

	require.Equal(t, http.StatusForbidden, w.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["message"].(string), "not allowed")
}

func TestActionHandler_HandleExecute_404_ProposalNotFound(t *testing.T) {
	svc := &mockActionService{
		executeFn: func(ctx context.Context, pool *pgxpool.Pool, proposalID, actorID string, opts ...action.ExecuteOption) (*action.ExecutionResult, error) {
			return nil, fmt.Errorf("load proposal: %w", action.ErrProposalNotFound)
		},
	}
	h := NewActionHandler(svc, nil)

	r := newTestRequest(http.MethodPost, "/api/v1/proposals/prop-999/execute", "{}")
	setURLParam(r, "id", "prop-999")
	w := httptest.NewRecorder()
	h.HandleExecute(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["message"].(string), "proposal not found")
}

func TestActionHandler_HandleStatus_200(t *testing.T) {
	svc := &mockActionService{
		getProposalByID: func(ctx context.Context, pool *pgxpool.Pool, proposalID string) (*action.ActionProposal, error) {
			return &action.ActionProposal{
				ProposalID:  "prop-1",
				ActionType:  "notify_owner",
				ApplyStatus: "approved",
			}, nil
		},
	}
	h := NewActionHandler(svc, nil)

	r := newTestRequest(http.MethodGet, "/api/v1/proposals/prop-1/status", "")
	setURLParam(r, "id", "prop-1")
	w := httptest.NewRecorder()
	h.HandleStatus(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp statusResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "prop-1", resp.ProposalID)
	assert.Equal(t, "approved", resp.ApplyStatus)
	assert.Equal(t, "notify_owner", resp.ActionType)
}

func TestActionHandler_HandleExecute_400_MissingProposalID(t *testing.T) {
	svc := &mockActionService{}
	h := NewActionHandler(svc, nil)

	r := newTestRequest(http.MethodPost, "/api/v1/proposals//execute", "{}")
	w := httptest.NewRecorder()
	h.HandleExecute(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "validation failed", resp["message"].(string))
}

func TestActionHandler_HandleStatus_404(t *testing.T) {
	svc := &mockActionService{
		getProposalByID: func(ctx context.Context, pool *pgxpool.Pool, proposalID string) (*action.ActionProposal, error) {
			return nil, nil
		},
	}
	h := NewActionHandler(svc, nil)

	r := newTestRequest(http.MethodGet, "/api/v1/proposals/prop-999/status", "")
	setURLParam(r, "id", "prop-999")
	w := httptest.NewRecorder()
	h.HandleStatus(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["message"].(string), "proposal not found")
}

func TestActionHandler_HandleExecute_200_DryRunDefault(t *testing.T) {
	svc := &mockActionService{
		executeFn: func(ctx context.Context, pool *pgxpool.Pool, proposalID, actorID string, opts ...action.ExecuteOption) (*action.ExecutionResult, error) {
			options := &action.ExecuteOptions{DryRun: true}
			for _, opt := range opts {
				opt(options)
			}
			assert.True(t, options.DryRun, "dry run should default to true")
			return &action.ExecutionResult{Success: true, DryRun: true}, nil
		},
	}
	h := NewActionHandler(svc, nil)

	r := newTestRequest(http.MethodPost, "/api/v1/proposals/prop-1/execute", "{}")
	setURLParam(r, "id", "prop-1")
	w := httptest.NewRecorder()
	h.HandleExecute(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp executeResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.True(t, resp.DryRun)
}

func TestActionHandler_HandleExecute_200_RealExecution(t *testing.T) {
	svc := &mockActionService{
		executeFn: func(ctx context.Context, pool *pgxpool.Pool, proposalID, actorID string, opts ...action.ExecuteOption) (*action.ExecutionResult, error) {
			options := &action.ExecuteOptions{DryRun: true}
			for _, opt := range opts {
				opt(options)
			}
			assert.False(t, options.DryRun, "dry run should be false when explicitly set")
			return &action.ExecutionResult{Success: true, DryRun: false, OutboxEventID: "evt-abc-123"}, nil
		},
	}
	h := NewActionHandler(svc, nil)

	body := `{"dry_run": false}`
	r := newTestRequest(http.MethodPost, "/api/v1/proposals/prop-1/execute", body)
	setURLParam(r, "id", "prop-1")
	w := httptest.NewRecorder()
	h.HandleExecute(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp executeResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "applied", resp.ApplyStatus)
	assert.False(t, resp.DryRun)
	assert.Equal(t, "evt-abc-123", resp.OutboxEventID)
}

func TestActionHandler_HandleExecute_200_FailedExecution(t *testing.T) {
	svc := &mockActionService{
		executeFn: func(ctx context.Context, pool *pgxpool.Pool, proposalID, actorID string, opts ...action.ExecuteOption) (*action.ExecutionResult, error) {
			return &action.ExecutionResult{Success: false, DryRun: false, Error: "executor failed"}, nil
		},
	}
	h := NewActionHandler(svc, nil)

	body := `{"dry_run": false}`
	r := newTestRequest(http.MethodPost, "/api/v1/proposals/prop-1/execute", body)
	setURLParam(r, "id", "prop-1")
	w := httptest.NewRecorder()
	h.HandleExecute(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp executeResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "failed", resp.ApplyStatus)
	assert.False(t, resp.DryRun)
}

func TestActionHandler_HandleStatus_400_MissingProposalID(t *testing.T) {
	svc := &mockActionService{}
	h := NewActionHandler(svc, nil)

	r := newTestRequest(http.MethodGet, "/api/v1/proposals//status", "")
	w := httptest.NewRecorder()
	h.HandleStatus(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "validation failed", resp["message"].(string))
}

func TestActionHandler_HandleStatus_500_InternalError(t *testing.T) {
	svc := &mockActionService{
		getProposalByID: func(ctx context.Context, pool *pgxpool.Pool, proposalID string) (*action.ActionProposal, error) {
			return nil, errors.New("database connection failed")
		},
	}
	h := NewActionHandler(svc, nil)

	r := newTestRequest(http.MethodGet, "/api/v1/proposals/prop-1/status", "")
	setURLParam(r, "id", "prop-1")
	w := httptest.NewRecorder()
	h.HandleStatus(w, r)

	require.Equal(t, http.StatusInternalServerError, w.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["message"].(string), "internal server error")
}

func TestActionHandler_HandleExecute_500_InternalError(t *testing.T) {
	svc := &mockActionService{
		executeFn: func(ctx context.Context, pool *pgxpool.Pool, proposalID, actorID string, opts ...action.ExecuteOption) (*action.ExecutionResult, error) {
			return nil, errors.New("unexpected database error")
		},
	}
	h := NewActionHandler(svc, nil)

	r := newTestRequest(http.MethodPost, "/api/v1/proposals/prop-1/execute", "{}")
	setURLParam(r, "id", "prop-1")
	w := httptest.NewRecorder()
	h.HandleExecute(w, r)

	require.Equal(t, http.StatusInternalServerError, w.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["message"].(string), "internal server error")
}

func TestActionHandler_HandleExecute_200_OutboxEventIDPropagation(t *testing.T) {
	svc := &mockActionService{
		executeFn: func(ctx context.Context, pool *pgxpool.Pool, proposalID, actorID string, opts ...action.ExecuteOption) (*action.ExecutionResult, error) {
			return &action.ExecutionResult{
				Success:       true,
				DryRun:        false,
				OutboxEventID: "evt-xyz-789",
			}, nil
		},
	}
	h := NewActionHandler(svc, nil)

	body := `{"dry_run": false}`
	r := newTestRequest(http.MethodPost, "/api/v1/proposals/prop-42/execute", body)
	setURLParam(r, "id", "prop-42")
	r = withActor(r, "test-actor")
	w := httptest.NewRecorder()
	h.HandleExecute(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp executeResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "prop-42", resp.ProposalID)
	assert.Equal(t, "applied", resp.ApplyStatus)
	assert.Equal(t, "evt-xyz-789", resp.OutboxEventID)
}

func TestActionHandler_HandleExecute_200_EmptyOutboxEventID(t *testing.T) {
	svc := &mockActionService{
		executeFn: func(ctx context.Context, pool *pgxpool.Pool, proposalID, actorID string, opts ...action.ExecuteOption) (*action.ExecutionResult, error) {
			return &action.ExecutionResult{
				Success: true,
				DryRun:  true,
			}, nil
		},
	}
	h := NewActionHandler(svc, nil)

	body := `{"dry_run": true}`
	r := newTestRequest(http.MethodPost, "/api/v1/proposals/prop-1/execute", body)
	setURLParam(r, "id", "prop-1")
	w := httptest.NewRecorder()
	h.HandleExecute(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp executeResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "approved", resp.ApplyStatus)
	assert.Empty(t, resp.OutboxEventID, "dry-run should not produce an outbox event ID")
}
