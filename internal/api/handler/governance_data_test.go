package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"baxi/internal/api/dto"
	"baxi/internal/model"
)

// flexibleMockDataProvider allows per-test configuration of return values.
type flexibleMockDataProvider struct {
	catalogResp       *model.CatalogResponse
	catalogErr        error
	classificationResp *model.ClassificationResponse
	classificationErr  error
	fieldMarkingResp   *model.FieldMarkingResponse
	fieldMarkingErr    error
	lineageResp        *model.LineageResponse
	lineageErr         error
	checkpointsResp    *model.CheckpointsResponse
	checkpointsErr     error
	healthResp         *model.HealthChecksResponse
	healthErr          error
}

func (m *flexibleMockDataProvider) GetCatalog(_ context.Context) (*model.CatalogResponse, error) {
	return m.catalogResp, m.catalogErr
}
func (m *flexibleMockDataProvider) GetClassification(_ context.Context, _ string) (*model.ClassificationResponse, error) {
	return m.classificationResp, m.classificationErr
}
func (m *flexibleMockDataProvider) GetFieldMarking(_ context.Context, _, _ string) (*model.FieldMarkingResponse, error) {
	return m.fieldMarkingResp, m.fieldMarkingErr
}
func (m *flexibleMockDataProvider) GetLineage(_ context.Context, _ string) (*model.LineageResponse, error) {
	return m.lineageResp, m.lineageErr
}
func (m *flexibleMockDataProvider) GetCheckpoints(_ context.Context) (*model.CheckpointsResponse, error) {
	return m.checkpointsResp, m.checkpointsErr
}
func (m *flexibleMockDataProvider) GetHealthChecks(_ context.Context) (*model.HealthChecksResponse, error) {
	return m.healthResp, m.healthErr
}

// also implement GovernanceStatusProvider to satisfy handler construction
func (m *flexibleMockDataProvider) GetStatus(_ context.Context) (*model.GovernanceStatusResponse, error) {
	return &model.GovernanceStatusResponse{}, nil
}

// ──── HandleCatalog ────────────────────────────────────────────────────────

func TestHandleCatalog_Success(t *testing.T) {
	data := &flexibleMockDataProvider{
		catalogResp: &model.CatalogResponse{
			Objects: []model.CatalogObject{
				{ObjectType: "order", SourceDataset: "orders", PrimaryKey: "order_id", PropertiesCount: 10, LinksCount: 3},
			},
			Datasets: []model.CatalogDataset{
				{Dataset: "orders", Schema: "public", Table: "orders"},
			},
		},
	}
	h := NewGovernanceHandler(data, data)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/governance/catalog", nil)

	h.HandleCatalog(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body dto.CatalogResponse
	decodeJSON(t, resp, &body)
	assert.Len(t, body.Objects, 1)
	assert.Equal(t, "order", body.Objects[0].ObjectType)
	assert.Len(t, body.Datasets, 1)
}

func TestHandleCatalog_Error(t *testing.T) {
	data := &flexibleMockDataProvider{
		catalogErr: errors.New("db connection failed"),
	}
	h := NewGovernanceHandler(data, data)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/governance/catalog", nil)

	h.HandleCatalog(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

// ──── HandleClassification ─────────────────────────────────────────────────

func TestHandleClassification_Success(t *testing.T) {
	data := &flexibleMockDataProvider{
		classificationResp: &model.ClassificationResponse{
			Levels: []string{"public", "confidential"},
			Resources: []model.ClassificationResource{
				{Resource: "order.order_id", Classification: "confidential"},
			},
		},
	}
	h := NewGovernanceHandler(data, data)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/governance/classification?field_path=order.order_id", nil)

	h.HandleClassification(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body dto.ClassificationResponse
	decodeJSON(t, resp, &body)
	assert.Equal(t, []string{"public", "confidential"}, body.Levels)
	assert.Len(t, body.Resources, 1)
}

func TestHandleClassification_Error(t *testing.T) {
	data := &flexibleMockDataProvider{
		classificationErr: errors.New("classification service unavailable"),
	}
	h := NewGovernanceHandler(data, data)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/governance/classification", nil)

	h.HandleClassification(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

// ──── HandleMarkings ───────────────────────────────────────────────────────

func TestHandleMarkings_Success(t *testing.T) {
	data := &flexibleMockDataProvider{
		fieldMarkingResp: &model.FieldMarkingResponse{
			Markings: []model.FieldMarking{
				{ObjectType: "order", Field: "order_id", Classification: "confidential", PII: true, LLMAllowed: false},
			},
		},
	}
	h := NewGovernanceHandler(data, data)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/governance/markings?object_type=order&property=order_id", nil)

	h.HandleMarkings(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestHandleMarkings_Error(t *testing.T) {
	data := &flexibleMockDataProvider{
		fieldMarkingErr: errors.New("markings unavailable"),
	}
	h := NewGovernanceHandler(data, data)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/governance/markings", nil)

	h.HandleMarkings(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

// ──── HandleLineage ────────────────────────────────────────────────────────

func TestHandleLineage_Success(t *testing.T) {
	data := &flexibleMockDataProvider{
		lineageResp: &model.LineageResponse{
			Resource:   "order.order_id",
			Upstream:   []string{"raw_orders.order_id"},
			Downstream: []string{"report.order_id"},
		},
	}
	h := NewGovernanceHandler(data, data)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/governance/lineage?resource=order.order_id", nil)

	h.HandleLineage(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestHandleLineage_Error(t *testing.T) {
	data := &flexibleMockDataProvider{
		lineageErr: errors.New("lineage lookup failed"),
	}
	h := NewGovernanceHandler(data, data)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/governance/lineage", nil)

	h.HandleLineage(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

// ──── HandleCheckpoints ────────────────────────────────────────────────────

func TestHandleCheckpoints_Success(t *testing.T) {
	data := &flexibleMockDataProvider{
		checkpointsResp: &model.CheckpointsResponse{
			Checkpoints: []model.CheckpointRule{
				{Action: "export_report", RequiresReason: true, RequiresHumanReview: true},
			},
		},
	}
	h := NewGovernanceHandler(data, data)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/governance/checkpoints", nil)

	h.HandleCheckpoints(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestHandleCheckpoints_Error(t *testing.T) {
	data := &flexibleMockDataProvider{
		checkpointsErr: errors.New("checkpoints unavailable"),
	}
	h := NewGovernanceHandler(data, data)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/governance/checkpoints", nil)

	h.HandleCheckpoints(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

// ──── HandleHealth ─────────────────────────────────────────────────────────

func TestHandleHealth_Success(t *testing.T) {
	data := &flexibleMockDataProvider{
		healthResp: &model.HealthChecksResponse{
			Status: "healthy",
			Checks: []model.HealthCheckItem{
				{Name: "database", Status: "ok"},
				{Name: "llm", Status: "ok"},
			},
		},
	}
	h := NewGovernanceHandler(data, data)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/governance/health", nil)

	h.HandleHealth(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestHandleHealth_Error(t *testing.T) {
	data := &flexibleMockDataProvider{
		healthErr: errors.New("health check failed"),
	}
	h := NewGovernanceHandler(data, data)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/governance/health", nil)

	h.HandleHealth(w, r)

	resp := w.Result()
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}
