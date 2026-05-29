package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"baxi/internal/api/dto"
	"baxi/internal/model"
)

type mockGovernanceService struct {
	resp *model.GovernanceStatusResponse
	err  error
}

func (m *mockGovernanceService) GetStatus(_ context.Context) (*model.GovernanceStatusResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.resp, nil
}

func (m *mockGovernanceService) GetCatalog(_ context.Context) (*model.CatalogResponse, error) {
	return &model.CatalogResponse{Objects: []model.CatalogObject{}, Datasets: []model.CatalogDataset{}}, nil
}

func (m *mockGovernanceService) GetClassification(_ context.Context, _ string) (*model.ClassificationResponse, error) {
	return &model.ClassificationResponse{Levels: []string{}, Resources: []model.ClassificationResource{}}, nil
}

func (m *mockGovernanceService) GetFieldMarking(_ context.Context, _, _ string) (*model.FieldMarkingResponse, error) {
	return &model.FieldMarkingResponse{Markings: []model.FieldMarking{}}, nil
}

func (m *mockGovernanceService) GetLineage(_ context.Context, _ string) (*model.LineageResponse, error) {
	return &model.LineageResponse{Resource: "", Upstream: []string{}, Downstream: []string{}}, nil
}

func (m *mockGovernanceService) GetCheckpoints(_ context.Context) (*model.CheckpointsResponse, error) {
	return &model.CheckpointsResponse{Checkpoints: []model.CheckpointRule{}}, nil
}

func (m *mockGovernanceService) GetHealthChecks(_ context.Context) (*model.HealthChecksResponse, error) {
	return &model.HealthChecksResponse{Status: "healthy", Checks: []model.HealthCheckItem{}}, nil
}

func TestHandleGovernanceStatus_Active(t *testing.T) {
	resp := &model.GovernanceStatusResponse{
		GovernanceLayer: "active",
		Configs: map[string]string{
			"data_catalog.yml":        "loaded",
			"data_classification.yml": "loaded",
			"data_markings.yml":       "loaded",
			"data_lineage.yml":        "loaded",
			"checkpoint_rules.yml":    "loaded",
			"retention_policies.yml":  "loaded",
			"health_checks.yml":       "loaded",
			"decision_eval_rules.yml": "loaded",
			"access_policy.yml":       "loaded",
		},
	}
	svc := &mockGovernanceService{resp: resp}
	h := NewGovernanceHandler(svc, nil)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/governance/status", nil)
	w := httptest.NewRecorder()
	h.HandleGovernanceStatus(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var body dto.GovernanceStatusResponse
	err := json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "active", body.GovernanceLayer)
	assert.Len(t, body.Configs, 9)
	assert.Equal(t, "loaded", body.Configs["data_catalog.yml"])
	assert.Equal(t, "loaded", body.Configs["access_policy.yml"])
}

func TestHandleGovernanceStatus_Unknown(t *testing.T) {
	resp := &model.GovernanceStatusResponse{
		GovernanceLayer: "unknown",
		Configs:         map[string]string{},
	}
	svc := &mockGovernanceService{resp: resp}
	h := NewGovernanceHandler(svc, nil)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/governance/status", nil)
	w := httptest.NewRecorder()
	h.HandleGovernanceStatus(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var body dto.GovernanceStatusResponse
	err := json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "unknown", body.GovernanceLayer)
	assert.Empty(t, body.Configs)
}

func TestHandleGovernanceStatus_Error(t *testing.T) {
	svc := &mockGovernanceService{err: assert.AnError}
	h := NewGovernanceHandler(svc, nil)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/governance/status", nil)
	w := httptest.NewRecorder()
	h.HandleGovernanceStatus(w, r)

	require.Equal(t, http.StatusInternalServerError, w.Code)

	var body map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)
	assert.Contains(t, body["message"].(string), "internal server error")
}

func TestHandleGovernanceStatus_ResponseFormat(t *testing.T) {
	resp := &model.GovernanceStatusResponse{
		GovernanceLayer: "active",
		Configs: map[string]string{
			"data_catalog.yml": "loaded",
		},
	}
	svc := &mockGovernanceService{resp: resp}
	h := NewGovernanceHandler(svc, nil)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/governance/status", nil)
	w := httptest.NewRecorder()
	h.HandleGovernanceStatus(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)

	_, hasLayer := body["governance_layer"]
	_, hasConfigs := body["configs"]
	assert.True(t, hasLayer, "response must have 'governance_layer' field")
	assert.True(t, hasConfigs, "response must have 'configs' field")
}

func TestHandleGovernanceStatus_PartialConfigs(t *testing.T) {
	resp := &model.GovernanceStatusResponse{
		GovernanceLayer: "active",
		Configs: map[string]string{
			"data_catalog.yml":  "loaded",
			"access_policy.yml": "loaded",
		},
	}
	svc := &mockGovernanceService{resp: resp}
	h := NewGovernanceHandler(svc, nil)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/governance/status", nil)
	w := httptest.NewRecorder()
	h.HandleGovernanceStatus(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var body dto.GovernanceStatusResponse
	err := json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "active", body.GovernanceLayer)
	assert.Len(t, body.Configs, 2)
	assert.Equal(t, "loaded", body.Configs["data_catalog.yml"])
	assert.Equal(t, "loaded", body.Configs["access_policy.yml"])
}
