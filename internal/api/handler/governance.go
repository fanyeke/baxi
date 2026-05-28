package handler

import (
	"context"
	"net/http"

	"baxi/internal/api/dto"
	"baxi/internal/httputil"
	"baxi/internal/model"
)

// GovernanceStatusProvider is the interface for retrieving governance status.
// Used by GovernanceHandler so tests can substitute a mock without importing the service package.
type GovernanceStatusProvider interface {
	GetStatus(ctx context.Context) (*model.GovernanceStatusResponse, error)
}

// GovernanceDataProvider provides data for all governance API responses.
// Implementations wrap GovernanceService methods for testability without service imports.
type GovernanceDataProvider interface {
	GetCatalog(ctx context.Context) (*model.CatalogResponse, error)
	GetClassification(ctx context.Context, fieldPath string) (*model.ClassificationResponse, error)
	GetFieldMarking(ctx context.Context, objectType, property string) (*model.FieldMarkingResponse, error)
	GetLineage(ctx context.Context, resource string) (*model.LineageResponse, error)
	GetCheckpoints(ctx context.Context) (*model.CheckpointsResponse, error)
	GetHealthChecks(ctx context.Context) (*model.HealthChecksResponse, error)
}

// GovernanceHandler handles HTTP requests for governance-related endpoints.
type GovernanceHandler struct {
	svc  GovernanceStatusProvider
	data GovernanceDataProvider
}

// NewGovernanceHandler creates a new GovernanceHandler.
func NewGovernanceHandler(svc GovernanceStatusProvider, data GovernanceDataProvider) *GovernanceHandler {
	return &GovernanceHandler{svc: svc, data: data}
}

// HandleGovernanceStatus returns the current governance layer status including
// configuration file load states and object schema count.
func (h *GovernanceHandler) HandleGovernanceStatus(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.GetStatus(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	httputil.JSON(w, http.StatusOK, dtoFromGovernanceStatusResponse(resp))
}

// HandleCatalog returns the governance catalog of object schemas and datasets.
func (h *GovernanceHandler) HandleCatalog(w http.ResponseWriter, r *http.Request) {
	resp, err := h.data.GetCatalog(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	httputil.JSON(w, http.StatusOK, dtoFromCatalogResponse(resp))
}

// HandleClassification returns data classification levels and resources.
// Optional query param: field_path (filter by specific field path).
func (h *GovernanceHandler) HandleClassification(w http.ResponseWriter, r *http.Request) {
	fieldPath := r.URL.Query().Get("field_path")

	resp, err := h.data.GetClassification(r.Context(), fieldPath)
	if err != nil {
		httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	httputil.JSON(w, http.StatusOK, dtoFromClassificationResponse(resp))
}

// HandleMarkings returns field-level classification markings.
// Optional query params: object_type, property (filter by object type and property).
func (h *GovernanceHandler) HandleMarkings(w http.ResponseWriter, r *http.Request) {
	objectType := r.URL.Query().Get("object_type")
	property := r.URL.Query().Get("property")

	resp, err := h.data.GetFieldMarking(r.Context(), objectType, property)
	if err != nil {
		httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	httputil.JSON(w, http.StatusOK, dtoFromFieldMarkingResponse(resp))
}

// HandleLineage returns the upstream and downstream lineage for a resource.
func (h *GovernanceHandler) HandleLineage(w http.ResponseWriter, r *http.Request) {
	resource := r.URL.Query().Get("resource")

	resp, err := h.data.GetLineage(r.Context(), resource)
	if err != nil {
		httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	httputil.JSON(w, http.StatusOK, dtoFromLineageResponse(resp))
}

// HandleCheckpoints returns checkpoint rules for sensitive governance actions.
func (h *GovernanceHandler) HandleCheckpoints(w http.ResponseWriter, r *http.Request) {
	resp, err := h.data.GetCheckpoints(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	httputil.JSON(w, http.StatusOK, dtoFromCheckpointsResponse(resp))
}

// HandleHealth returns the status of all governance health checks.
func (h *GovernanceHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	resp, err := h.data.GetHealthChecks(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	httputil.JSON(w, http.StatusOK, dtoFromHealthChecksResponse(resp))
}

// dtoFromGovernanceStatusResponse converts model.GovernanceStatusResponse to dto.GovernanceStatusResponse.
func dtoFromGovernanceStatusResponse(m *model.GovernanceStatusResponse) *dto.GovernanceStatusResponse {
	if m == nil {
		return nil
	}
	return &dto.GovernanceStatusResponse{
		GovernanceLayer:   m.GovernanceLayer,
		Configs:           m.Configs,
		ObjectSchemaCount: m.ObjectSchemaCount,
	}
}

// dtoFromCatalogResponse converts model.CatalogResponse to dto.CatalogResponse.
func dtoFromCatalogResponse(m *model.CatalogResponse) *dto.CatalogResponse {
	if m == nil {
		return nil
	}

	objects := make([]dto.CatalogObject, len(m.Objects))
	for i, obj := range m.Objects {
		objects[i] = dto.CatalogObject{
			ObjectType:      obj.ObjectType,
			SourceDataset:   obj.SourceDataset,
			PrimaryKey:      obj.PrimaryKey,
			PropertiesCount: obj.PropertiesCount,
			LinksCount:      obj.LinksCount,
		}
	}

	datasets := make([]dto.CatalogDataset, len(m.Datasets))
	for i, ds := range m.Datasets {
		datasets[i] = dto.CatalogDataset{
			Dataset: ds.Dataset,
			Schema:  ds.Schema,
			Table:   ds.Table,
		}
	}

	return &dto.CatalogResponse{
		Objects:  objects,
		Datasets: datasets,
	}
}

// dtoFromClassificationResponse converts model.ClassificationResponse to dto.ClassificationResponse.
func dtoFromClassificationResponse(m *model.ClassificationResponse) *dto.ClassificationResponse {
	if m == nil {
		return nil
	}

	resources := make([]dto.ClassificationResource, len(m.Resources))
	for i, res := range m.Resources {
		resources[i] = dto.ClassificationResource{
			Resource:       res.Resource,
			Classification: res.Classification,
		}
	}

	return &dto.ClassificationResponse{
		Levels:    m.Levels,
		Resources: resources,
	}
}

// dtoFromFieldMarkingResponse converts model.FieldMarkingResponse to dto.FieldMarkingResponse.
func dtoFromFieldMarkingResponse(m *model.FieldMarkingResponse) *dto.FieldMarkingResponse {
	if m == nil {
		return nil
	}

	markings := make([]dto.FieldMarking, len(m.Markings))
	for i, marking := range m.Markings {
		markings[i] = dto.FieldMarking{
			ObjectType:     marking.ObjectType,
			Field:          marking.Field,
			Classification: marking.Classification,
			PII:            marking.PII,
			LLMAllowed:     marking.LLMAllowed,
		}
	}

	return &dto.FieldMarkingResponse{
		Markings: markings,
	}
}

// dtoFromLineageResponse converts model.LineageResponse to dto.LineageResponse.
func dtoFromLineageResponse(m *model.LineageResponse) *dto.LineageResponse {
	if m == nil {
		return nil
	}
	return &dto.LineageResponse{
		Resource:   m.Resource,
		Upstream:   m.Upstream,
		Downstream: m.Downstream,
	}
}

// dtoFromCheckpointsResponse converts model.CheckpointsResponse to dto.CheckpointsResponse.
func dtoFromCheckpointsResponse(m *model.CheckpointsResponse) *dto.CheckpointsResponse {
	if m == nil {
		return nil
	}

	checkpoints := make([]dto.CheckpointRule, len(m.Checkpoints))
	for i, cp := range m.Checkpoints {
		checkpoints[i] = dto.CheckpointRule{
			Action:              cp.Action,
			RequiresReason:      cp.RequiresReason,
			RequiresHumanReview: cp.RequiresHumanReview,
		}
	}

	return &dto.CheckpointsResponse{
		Checkpoints: checkpoints,
	}
}

// dtoFromHealthChecksResponse converts model.HealthChecksResponse to dto.HealthChecksResponse.
func dtoFromHealthChecksResponse(m *model.HealthChecksResponse) *dto.HealthChecksResponse {
	if m == nil {
		return nil
	}

	checks := make([]dto.HealthCheckItem, len(m.Checks))
	for i, check := range m.Checks {
		checks[i] = dto.HealthCheckItem{
			Name:   check.Name,
			Status: check.Status,
		}
	}

	return &dto.HealthChecksResponse{
		Status: m.Status,
		Checks: checks,
	}
}
