package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5/middleware"

	"baxi/internal/api/dto"
	"baxi/internal/httputil"
	"baxi/internal/model"
)

// ContextFetcher is the interface for fetching the Qoder context response.
// Tests can substitute a mock without importing the service package.
type ContextFetcher interface {
	GetContext(ctx context.Context, requestID string, params model.ContextQueryParams) (*model.ContextResponse, error)
}

// QoderHandler handles Qoder AI decision engine endpoints.
type QoderHandler struct {
	ctxFetcher ContextFetcher
}

// NewQoderHandler creates a new QoderHandler.
// When called with no arguments, the handler works in a static mode for capabilities only.
// When called with a ContextFetcher, it enables the context endpoint.
func NewQoderHandler(ctxFetcher ...ContextFetcher) *QoderHandler {
	h := &QoderHandler{}
	if len(ctxFetcher) > 0 {
		h.ctxFetcher = ctxFetcher[0]
	}
	return h
}

// HandleCapabilities returns the static Qoder capability matrix.
func (h *QoderHandler) HandleCapabilities(w http.ResponseWriter, r *http.Request) {
	caps := model.StaticCapabilities()
	httputil.JSON(w, http.StatusOK, dtoFromCapabilities(caps))
}

// HandleContext returns the aggregated Qoder context.
// Supports query params: severity, limit_alerts (1-100, default 10),
// limit_tasks (1-100, default 10), limit_outbox (1-100, default 10),
// include_logs (default false).
func (h *QoderHandler) HandleContext(w http.ResponseWriter, r *http.Request) {
	if h.ctxFetcher == nil {
		httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "context fetcher not available"})
		return
	}

	params := parseContextParams(r)
	requestID := middleware.GetReqID(r.Context())
	if requestID == "" {
		requestID = "unknown"
	}

	resp, err := h.ctxFetcher.GetContext(r.Context(), requestID, params)
	if err != nil {
		httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	httputil.JSON(w, http.StatusOK, dtoFromContextResponse(resp))
}

func parseContextParams(r *http.Request) model.ContextQueryParams {
	q := r.URL.Query()

	params := model.ContextQueryParams{
		Severity:    q.Get("severity"),
		LimitAlerts: 10,
		LimitTasks:  10,
		LimitOutbox: 10,
	}

	if v := q.Get("limit_alerts"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			params.LimitAlerts = n
			if params.LimitAlerts > 100 {
				params.LimitAlerts = 100
			}
		}
	}
	if v := q.Get("limit_tasks"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			params.LimitTasks = n
			if params.LimitTasks > 100 {
				params.LimitTasks = 100
			}
		}
	}
	if v := q.Get("limit_outbox"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			params.LimitOutbox = n
			if params.LimitOutbox > 100 {
				params.LimitOutbox = 100
			}
		}
	}
	if v := q.Get("include_logs"); v == "true" || v == "1" {
		params.IncludeLogs = true
	}

	return params
}

func dtoFromCapabilities(m model.CapabilitiesResponse) dto.CapabilitiesResponse {
	return dto.CapabilitiesResponse{
		Mode:              m.Mode,
		Version:           m.Version,
		CanReadStatus:     m.CanReadStatus,
		CanReadAlerts:     m.CanReadAlerts,
		CanReadTasks:      m.CanReadTasks,
		CanReadOutbox:     m.CanReadOutbox,
		CanReadGovernance: m.CanReadGovernance,
		CanReadLogs:       m.CanReadLogs,
		CanWriteReports:   m.CanWriteReports,
		CanExecuteActions: m.CanExecuteActions,
	}
}

func dtoFromContextResponse(m *model.ContextResponse) *dto.ContextResponse {
	if m == nil {
		return nil
	}
	return &dto.ContextResponse{
		RequestID: m.RequestID,
		System: dto.SystemInfo{
			LastPipelineRun: dtoFromPipelineRun(m.System.LastPipelineRun),
		},
		Summary: dto.ContextSummary{
			TotalAlerts:        m.Summary.TotalAlerts,
			TotalOpenTasks:     m.Summary.TotalOpenTasks,
			TotalPendingOutbox: m.Summary.TotalPendingOutbox,
		},
		TopAlerts:        dtoAlertItemsFromModel(m.TopAlerts),
		OpenTasks:        dtoTaskItemsFromModel(m.OpenTasks),
		PendingOutbox:    dtoOutboxItemsFromModel(m.PendingOutbox),
		RecentDiagnosis:  toInterfaceSlice(m.RecentDiagnosis),
		AllowedActions:   m.AllowedActions,
		ForbiddenActions: m.ForbiddenActions,
		Ontology: dto.OntologyInfo{
			ObjectTypes:      m.Ontology.ObjectTypes,
			ObjectsAvailable: m.Ontology.ObjectsAvailable,
		},
		Governance: dto.GovernanceInfo{
			ClassificationLoaded: m.Governance.ClassificationLoaded,
			LineageLoaded:        m.Governance.LineageLoaded,
			AccessPolicyLoaded:   m.Governance.AccessPolicyLoaded,
			RedactionEnabled:     m.Governance.RedactionEnabled,
		},
		AgentPolicy: dto.AgentPolicyInfo{
			Role:              m.AgentPolicy.Role,
			CanReadObjects:    m.AgentPolicy.CanReadObjects,
			CanExecuteActions: m.AgentPolicy.CanExecuteActions,
			CanWriteReports:   m.AgentPolicy.CanWriteReports,
		},
	}
}

func dtoFromPipelineRun(m *model.PipelineRunInfo) *dto.PipelineRunInfo {
	if m == nil {
		return nil
	}
	return &dto.PipelineRunInfo{
		RunID:        m.RunID,
		RunType:      m.RunType,
		Mode:         m.Mode,
		Status:       m.Status,
		StartedAt:    m.StartedAt,
		FinishedAt:   m.FinishedAt,
		InputCount:   m.InputCount,
		OutputCount:  m.OutputCount,
		ErrorMessage: m.ErrorMessage,
	}
}

func dtoAlertItemsFromModel(items []model.AlertItem) []dto.AlertItem {
	if items == nil {
		return nil
	}
	result := make([]dto.AlertItem, len(items))
	for i, item := range items {
		result[i] = dto.AlertItem{
			EventID:       item.EventID,
			RuleID:        item.RuleID,
			EventDate:     item.EventDate,
			Severity:      item.Severity,
			MetricName:    item.MetricName,
			ObjectType:    item.ObjectType,
			ObjectID:      item.ObjectID,
			CurrentValue:  item.CurrentValue,
			BaselineValue: item.BaselineValue,
			ChangeRate:    item.ChangeRate,
			OwnerRole:     item.OwnerRole,
			Status:        item.Status,
			ImpactScore:   item.ImpactScore,
		}
	}
	return result
}

func dtoTaskItemsFromModel(items []model.TaskItem) []dto.TaskItem {
	if items == nil {
		return nil
	}
	result := make([]dto.TaskItem, len(items))
	for i, item := range items {
		result[i] = dto.TaskItem{
			TaskID:           item.TaskID,
			TaskTitle:        item.TaskTitle,
			TaskDescription:  item.TaskDescription,
			Status:           item.Status,
			Priority:         item.Priority,
			OwnerRole:        item.OwnerRole,
			OwnerUserID:      item.OwnerUserID,
			DueAt:            item.DueAt,
			CreatedAt:        item.CreatedAt,
			CompletedAt:      item.CompletedAt,
			Feedback:         item.Feedback,
			RecommendationID: item.RecommendationID,
			EventID:          item.EventID,
			TargetObjectType: item.TargetObjectType,
			TargetObjectID:   item.TargetObjectID,
		}
	}
	return result
}

func dtoOutboxItemsFromModel(items []model.OutboxItem) []dto.OutboxItem {
	if items == nil {
		return nil
	}
	result := make([]dto.OutboxItem, len(items))
	for i, item := range items {
		result[i] = dto.OutboxItem{
			OutboxID:         item.OutboxID,
			EventType:        item.EventType,
			SourceType:       item.SourceType,
			SourceID:         item.SourceID,
			TargetChannel:    item.TargetChannel,
			Status:           item.Status,
			CreatedAt:        item.CreatedAt,
			DispatchAttempts: item.DispatchAttempts,
			LastDispatchAt:   item.LastDispatchAt,
		}
	}
	return result
}

func toInterfaceSlice(s []string) []interface{} {
	if s == nil {
		return nil
	}
	result := make([]interface{}, len(s))
	for i, v := range s {
		result[i] = v
	}
	return result
}
