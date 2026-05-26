package handler

import (
	"context"
	"net/http"

	"baxi/internal/api/dto"
	"baxi/internal/httputil"
)

// GovernanceStatusProvider is the interface for retrieving governance status.
// Used by GovernanceHandler so tests can substitute a mock without importing the service package.
type GovernanceStatusProvider interface {
	GetStatus(ctx context.Context) (*dto.GovernanceStatusResponse, error)
}

// GovernanceDataProvider provides data for all governance API responses.
// Implementations wrap GovernanceService methods for testability without service imports.
type GovernanceDataProvider interface {
	GetCatalog(ctx context.Context) (*dto.CatalogResponse, error)
	GetClassification(ctx context.Context, fieldPath string) (*dto.ClassificationResponse, error)
	GetFieldMarking(ctx context.Context, objectType, property string) (*dto.FieldMarkingResponse, error)
	GetLineage(ctx context.Context, resource string) (*dto.LineageResponse, error)
	GetCheckpoints(ctx context.Context) (*dto.CheckpointsResponse, error)
	GetHealthChecks(ctx context.Context) (*dto.HealthChecksResponse, error)
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

	httputil.JSON(w, http.StatusOK, resp)
}

// HandleCatalog returns the governance catalog of object schemas and datasets.
func (h *GovernanceHandler) HandleCatalog(w http.ResponseWriter, r *http.Request) {
	resp, err := h.data.GetCatalog(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	httputil.JSON(w, http.StatusOK, resp)
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

	httputil.JSON(w, http.StatusOK, resp)
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

	httputil.JSON(w, http.StatusOK, resp)
}

// HandleLineage returns the upstream and downstream lineage for a resource.
func (h *GovernanceHandler) HandleLineage(w http.ResponseWriter, r *http.Request) {
	resource := r.URL.Query().Get("resource")

	resp, err := h.data.GetLineage(r.Context(), resource)
	if err != nil {
		httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	httputil.JSON(w, http.StatusOK, resp)
}

// HandleCheckpoints returns checkpoint rules for sensitive governance actions.
func (h *GovernanceHandler) HandleCheckpoints(w http.ResponseWriter, r *http.Request) {
	resp, err := h.data.GetCheckpoints(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	httputil.JSON(w, http.StatusOK, resp)
}

// HandleHealth returns the status of all governance health checks.
func (h *GovernanceHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	resp, err := h.data.GetHealthChecks(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	httputil.JSON(w, http.StatusOK, resp)
}
