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
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/action"
	"baxi/internal/api/dto"
	"baxi/internal/decision"
	"baxi/internal/llm"
)

type mockDecisionService struct {
	createCaseFn   func(ctx context.Context, alertID, createdBy string) (*decision.DecisionCase, error)
	getCaseFn      func(ctx context.Context, caseID string) (*decision.DecisionCase, error)
	listCasesFn    func(ctx context.Context, filter decision.CaseFilter) (*decision.CaseList, error)
	buildContextFn func(ctx context.Context, caseID string) (*decision.DecisionContext, error)
	decideFn       func(ctx context.Context, caseID string) (*decision.DecisionContext, *llm.DecisionOutput, []action.ActionProposal, error)
	listProposalsFn func(ctx context.Context, caseID string) ([]action.ActionProposal, error)
}

func (m *mockDecisionService) CreateCaseFromAlert(ctx context.Context, alertID, createdBy string) (*decision.DecisionCase, error) {
	if m.createCaseFn != nil {
		return m.createCaseFn(ctx, alertID, createdBy)
	}
	return nil, errors.New("unexpected call to CreateCaseFromAlert")
}

func (m *mockDecisionService) GetCase(ctx context.Context, caseID string) (*decision.DecisionCase, error) {
	if m.getCaseFn != nil {
		return m.getCaseFn(ctx, caseID)
	}
	return nil, errors.New("unexpected call to GetCase")
}

func (m *mockDecisionService) ListCases(ctx context.Context, filter decision.CaseFilter) (*decision.CaseList, error) {
	if m.listCasesFn != nil {
		return m.listCasesFn(ctx, filter)
	}
	return nil, errors.New("unexpected call to ListCases")
}

func (m *mockDecisionService) BuildContext(ctx context.Context, caseID string) (*decision.DecisionContext, error) {
	if m.buildContextFn != nil {
		return m.buildContextFn(ctx, caseID)
	}
	return nil, errors.New("unexpected call to BuildContext")
}

func (m *mockDecisionService) Decide(ctx context.Context, caseID string) (*decision.DecisionContext, *llm.DecisionOutput, []action.ActionProposal, error) {
	if m.decideFn != nil {
		return m.decideFn(ctx, caseID)
	}
	return nil, nil, nil, errors.New("unexpected call to Decide")
}

func (m *mockDecisionService) ListProposals(ctx context.Context, caseID string) ([]action.ActionProposal, error) {
	if m.listProposalsFn != nil {
		return m.listProposalsFn(ctx, caseID)
	}
	return nil, errors.New("unexpected call to ListProposals")
}

func TestDecisionHandler_CreateCase_201(t *testing.T) {
	now := time.Now()
	svc := &mockDecisionService{
		createCaseFn: func(ctx context.Context, alertID, createdBy string) (*decision.DecisionCase, error) {
			assert.Equal(t, "alert-1", alertID)
			assert.Equal(t, "api_user", createdBy)
			return &decision.DecisionCase{
				CaseID:     "dc_123",
				SourceType: strPtr("alert"),
				SourceID:   strPtr("alert-1"),
				Status:     "created",
				CreatedAt:  now,
			}, nil
		},
	}
	h := NewDecisionHandler(svc)

	body := `{"source_type":"alert","source_id":"alert-1"}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/decisions/cases", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreateCase(w, r)

	require.Equal(t, http.StatusCreated, w.Code)

	var resp dto.CreateCaseResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "dc_123", resp.DecisionCaseID)
	assert.Equal(t, "alert", resp.SourceType)
	assert.Equal(t, "alert-1", resp.SourceID)
	assert.Equal(t, "created", resp.Status)
}

func TestDecisionHandler_CreateCase_400_MissingFields(t *testing.T) {
	svc := &mockDecisionService{}
	h := NewDecisionHandler(svc)

	tests := []struct {
		name string
		body string
	}{
		{"empty body", `{}`},
		{"missing source_type", `{"source_id":"alert-1"}`},
		{"missing source_id", `{"source_type":"alert"}`},
		{"both empty", `{"source_type":"","source_id":""}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/api/v1/decisions/cases", strings.NewReader(tt.body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.CreateCase(w, r)

			require.Equal(t, http.StatusBadRequest, w.Code)

			var body map[string]string
			err := json.NewDecoder(w.Body).Decode(&body)
			require.NoError(t, err)
			assert.Contains(t, body["error"], "required")
		})
	}
}

func TestDecisionHandler_CreateCase_400_InvalidJSON(t *testing.T) {
	svc := &mockDecisionService{}
	h := NewDecisionHandler(svc)

	r := httptest.NewRequest(http.MethodPost, "/api/v1/decisions/cases", strings.NewReader("not json"))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreateCase(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var body map[string]string
	err := json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)
	assert.Contains(t, body["error"], "invalid request body")
}

func TestDecisionHandler_GetCase_200(t *testing.T) {
	now := time.Now()
	svc := &mockDecisionService{
		getCaseFn: func(ctx context.Context, caseID string) (*decision.DecisionCase, error) {
			assert.Equal(t, "dc_123", caseID)
			return &decision.DecisionCase{
				CaseID:     "dc_123",
				SourceType: strPtr("alert"),
				SourceID:   strPtr("alert-1"),
				ObjectType: "seller",
				ObjectID:   "seller-42",
				Severity:   "high",
				Status:     "created",
				CreatedAt:  now,
			}, nil
		},
	}
	h := NewDecisionHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/decisions/cases/dc_123", nil)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("case_id", "dc_123")
	w := httptest.NewRecorder()
	h.GetCase(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp dto.DecisionCaseResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "dc_123", resp.DecisionCaseID)
	assert.Equal(t, "alert", resp.SourceType)
	assert.Equal(t, "alert-1", resp.SourceID)
	assert.Equal(t, "seller", resp.ObjectType)
	assert.Equal(t, "seller-42", resp.ObjectID)
	assert.Equal(t, "high", resp.Severity)
	assert.Equal(t, "created", resp.Status)
	assert.Equal(t, now.Format(time.RFC3339), resp.CreatedAt)
}

func TestDecisionHandler_GetCase_404(t *testing.T) {
	svc := &mockDecisionService{
		getCaseFn: func(ctx context.Context, caseID string) (*decision.DecisionCase, error) {
			return nil, pgx.ErrNoRows
		},
	}
	h := NewDecisionHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/decisions/cases/nonexistent", nil)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("case_id", "nonexistent")
	w := httptest.NewRecorder()
	h.GetCase(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)

	var body map[string]string
	err := json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)
	assert.Contains(t, body["error"], "not found")
}

func TestDecisionHandler_ListCases_200(t *testing.T) {
	now := time.Now()
	svc := &mockDecisionService{
		listCasesFn: func(ctx context.Context, filter decision.CaseFilter) (*decision.CaseList, error) {
			assert.Equal(t, 100, filter.Limit)
			assert.Equal(t, 0, filter.Offset)
			return &decision.CaseList{
				Cases: []decision.DecisionCase{
					{
						CaseID:     "dc_1",
				SourceType: strPtr("alert"),
				SourceID:   strPtr("a1"),
				Status:     "created",
				CreatedAt:  now,
			},
			{
				CaseID:     "dc_2",
				SourceType: strPtr("alert"),
				SourceID:   strPtr("a2"),
						Status:     "open",
						CreatedAt:  now.Add(-1 * time.Hour),
					},
				},
				Total: 2,
			}, nil
		},
	}
	h := NewDecisionHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/decisions/cases", nil)
	w := httptest.NewRecorder()
	h.ListCases(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp dto.CaseListResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, 2, resp.Total)
	require.Len(t, resp.Items, 2)
	assert.Equal(t, "dc_1", resp.Items[0].DecisionCaseID)
	assert.Equal(t, "dc_2", resp.Items[1].DecisionCaseID)
	assert.Equal(t, "created", resp.Items[0].Status)
	assert.Equal(t, "open", resp.Items[1].Status)
}

func TestDecisionHandler_ListCases_WithFilters(t *testing.T) {
	svc := &mockDecisionService{
		listCasesFn: func(ctx context.Context, filter decision.CaseFilter) (*decision.CaseList, error) {
			assert.NotNil(t, filter.Status)
			assert.Equal(t, "created", *filter.Status)
			assert.NotNil(t, filter.Severity)
			assert.Equal(t, "high", *filter.Severity)
			assert.NotNil(t, filter.SourceType)
			assert.Equal(t, "alert", *filter.SourceType)
			return &decision.CaseList{Cases: []decision.DecisionCase{}, Total: 0}, nil
		},
	}
	h := NewDecisionHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/decisions/cases?status=created&severity=high&source_type=alert", nil)
	w := httptest.NewRecorder()
	h.ListCases(w, r)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestDecisionHandler_ListCases_Empty(t *testing.T) {
	svc := &mockDecisionService{
		listCasesFn: func(ctx context.Context, filter decision.CaseFilter) (*decision.CaseList, error) {
			return &decision.CaseList{Cases: []decision.DecisionCase{}, Total: 0}, nil
		},
	}
	h := NewDecisionHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/decisions/cases", nil)
	w := httptest.NewRecorder()
	h.ListCases(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp dto.CaseListResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Total)
	assert.Empty(t, resp.Items)
}

func TestDecisionHandler_ListCases_BadPagination(t *testing.T) {
	svc := &mockDecisionService{}
	h := NewDecisionHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/decisions/cases?limit=abc", nil)
	w := httptest.NewRecorder()
	h.ListCases(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var body map[string]string
	err := json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)
	assert.Contains(t, body["error"], "invalid limit")
}

func TestDecisionHandler_BuildContext_200(t *testing.T) {
	now := time.Now()
	svc := &mockDecisionService{
		getCaseFn: func(ctx context.Context, caseID string) (*decision.DecisionCase, error) {
			return &decision.DecisionCase{
				CaseID:     "dc_123",
				Status:     "context_built",
				SourceType: strPtr("alert"),
				SourceID:   strPtr("alert-1"),
				CreatedAt:  now,
			}, nil
		},
		buildContextFn: func(ctx context.Context, caseID string) (*decision.DecisionContext, error) {
			return &decision.DecisionContext{
				DecisionCaseID: "dc_123",
				SourceType:     strPtr("alert"),
				SourceID:       strPtr("alert-1"),
				Trigger: decision.TriggerInfo{
					AlertID:    "alert-1",
					RuleID:     "gmv_drop",
					Severity:   "high",
					MetricName: "gmv",
				},
				ObjectContext: decision.ObjectContextData{
					ObjectType: "seller",
					ObjectID:   "seller-42",
					Properties: map[string]interface{}{"gmv": 1000.0},
				},
				Governance: decision.GovernanceData{
					Classification:   "L2",
					RedactionApplied: false,
				},
				AllowedActions: []string{"notify_owner", "escalate_to_human"},
			}, nil
		},
	}
	h := NewDecisionHandler(svc)

	r := httptest.NewRequest(http.MethodPost, "/api/v1/decisions/cases/dc_123/context", nil)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("case_id", "dc_123")
	w := httptest.NewRecorder()
	h.BuildContext(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp dto.DecisionContextResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "dc_123", resp.DecisionCaseID)
	assert.Equal(t, "context_built", resp.Status)
	assert.NotNil(t, resp.Trigger)
	assert.NotNil(t, resp.ObjectContext)
	assert.NotNil(t, resp.Governance)
	assert.Equal(t, []string{"notify_owner", "escalate_to_human"}, resp.AllowedActions)
	assert.Equal(t, "alert-1", resp.Trigger["alert_id"])
	assert.Equal(t, "gmv_drop", resp.Trigger["rule_id"])
}

func TestDecisionHandler_BuildContext_404(t *testing.T) {
	svc := &mockDecisionService{
		getCaseFn: func(ctx context.Context, caseID string) (*decision.DecisionCase, error) {
			return nil, pgx.ErrNoRows
		},
	}
	h := NewDecisionHandler(svc)

	r := httptest.NewRequest(http.MethodPost, "/api/v1/decisions/cases/nonexistent/context", nil)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("case_id", "nonexistent")
	w := httptest.NewRecorder()
	h.BuildContext(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)

	var body map[string]string
	err := json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)
	assert.Contains(t, body["error"], "not found")
}

func TestDecisionHandler_Decide_200(t *testing.T) {
	now := time.Now()
	svc := &mockDecisionService{
		decideFn: func(ctx context.Context, caseID string) (*decision.DecisionContext, *llm.DecisionOutput, []action.ActionProposal, error) {
			return &decision.DecisionContext{
					DecisionCaseID: "dc_123",
					SourceType:     strPtr("alert"),
					SourceID:       strPtr("alert-1"),
					Trigger: decision.TriggerInfo{
						AlertID:  "alert-1",
						RuleID:   "gmv_drop",
						Severity: "high",
					},
					ObjectContext: decision.ObjectContextData{
						ObjectType: "seller",
						ObjectID:   "seller-42",
					},
					AllowedActions: []string{"notify_owner"},
				}, &llm.DecisionOutput{
					DecisionType:       "monitor_only",
					Severity:           "high",
					Summary:            "Monitor the situation",
					Rationale:          []string{"Drop is within threshold"},
					Confidence:         0.85,
					RequiresHumanReview: false,
				}, []action.ActionProposal{
					{
						ProposalID:          "ap_1",
						CaseID:              "dc_123",
						ActionType:          "notify_owner",
						Title:               "Notify owner",
						RiskLevel:           "medium",
						RequiresHumanReview: true,
						ApplyStatus:         "proposed",
						CreatedAt:           now,
					},
				}, nil
		},
	}
	h := NewDecisionHandler(svc)

	r := httptest.NewRequest(http.MethodPost, "/api/v1/decisions/cases/dc_123/decide", nil)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("case_id", "dc_123")
	w := httptest.NewRecorder()
	h.Decide(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp dto.DecisionResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "dc_123", resp.DecisionCaseID)
	assert.Equal(t, "decision_generated", resp.Status)
	assert.NotNil(t, resp.Decision)
	assert.Equal(t, "monitor_only", resp.Decision["decision_type"])
	assert.Equal(t, 0.85, resp.Decision["confidence"])
	require.Len(t, resp.Proposals, 1)
	assert.Equal(t, "ap_1", resp.Proposals[0].ProposalID)
	assert.Equal(t, "notify_owner", resp.Proposals[0].ActionType)
	assert.Equal(t, "proposed", resp.Proposals[0].ApplyStatus)
}

func TestDecisionHandler_Decide_404(t *testing.T) {
	svc := &mockDecisionService{
		decideFn: func(ctx context.Context, caseID string) (*decision.DecisionContext, *llm.DecisionOutput, []action.ActionProposal, error) {
			return nil, nil, nil, pgx.ErrNoRows
		},
	}
	h := NewDecisionHandler(svc)

	r := httptest.NewRequest(http.MethodPost, "/api/v1/decisions/cases/nonexistent/decide", nil)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("case_id", "nonexistent")
	w := httptest.NewRecorder()
	h.Decide(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)

	var body map[string]string
	err := json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)
	assert.Contains(t, body["error"], "not found")
}

func TestDecisionHandler_ListProposals_200(t *testing.T) {
	now := time.Now()
	svc := &mockDecisionService{
		listProposalsFn: func(ctx context.Context, caseID string) ([]action.ActionProposal, error) {
			return []action.ActionProposal{
				{
					ProposalID:          "ap_1",
					CaseID:              caseID,
					ActionType:          "notify_owner",
					Title:               "Notify seller owner",
					RiskLevel:           "medium",
					RequiresHumanReview: true,
					ApplyStatus:         "proposed",
					CreatedAt:           now,
				},
				{
					ProposalID:          "ap_2",
					CaseID:              caseID,
					ActionType:          "escalate_to_human",
					Title:               "Escalate to human",
					RiskLevel:           "high",
					RequiresHumanReview: true,
					ApplyStatus:         "proposed",
					CreatedAt:           now.Add(-1 * time.Hour),
				},
			}, nil
		},
	}
	h := NewDecisionHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/decisions/cases/dc_123/proposals", nil)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("case_id", "dc_123")
	w := httptest.NewRecorder()
	h.ListProposals(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp dto.ProposalListResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	require.Len(t, resp.Items, 2)
	assert.Equal(t, "ap_1", resp.Items[0].ProposalID)
	assert.Equal(t, "notify_owner", resp.Items[0].ActionType)
	assert.Equal(t, "medium", resp.Items[0].RiskLevel)
	assert.True(t, resp.Items[0].RequiresHumanReview)
	assert.Equal(t, "proposed", resp.Items[0].ApplyStatus)
	assert.Equal(t, now.Format(time.RFC3339), resp.Items[0].CreatedAt)

	assert.Equal(t, "ap_2", resp.Items[1].ProposalID)
	assert.Equal(t, "escalate_to_human", resp.Items[1].ActionType)
	assert.Equal(t, "high", resp.Items[1].RiskLevel)
}

func TestDecisionHandler_ListProposals_Empty(t *testing.T) {
	svc := &mockDecisionService{
		listProposalsFn: func(ctx context.Context, caseID string) ([]action.ActionProposal, error) {
			return []action.ActionProposal{}, nil
		},
	}
	h := NewDecisionHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/decisions/cases/dc_123/proposals", nil)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
	chi.RouteContext(r.Context()).URLParams.Add("case_id", "dc_123")
	w := httptest.NewRecorder()
	h.ListProposals(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp dto.ProposalListResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Empty(t, resp.Items)
}
