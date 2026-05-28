package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"baxi/internal/httputil"
	"baxi/internal/service"
)

// AgentLogLister is the interface for listing agent execution logs.
// Tests can substitute a mock without importing the service package.
type AgentLogLister interface {
	ListAgentLogs(ctx context.Context, limit, offset int) (*service.AgentLogListResponse, error)
}

// AgentLogHandler handles HTTP requests for agent execution log endpoints.
type AgentLogHandler struct {
	svc AgentLogLister
}

// NewAgentLogHandler creates a new AgentLogHandler.
func NewAgentLogHandler(svc AgentLogLister) *AgentLogHandler {
	return &AgentLogHandler{svc: svc}
}

// HandleListAgentLogs handles GET /api/v1/logs/agent.
func (h *AgentLogHandler) HandleListAgentLogs(w http.ResponseWriter, r *http.Request) {
	pagination, err := httputil.ParsePagination(r)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	resp, err := h.svc.ListAgentLogs(r.Context(), pagination.Limit, pagination.Offset)
	if err != nil {
		httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	httputil.JSON(w, http.StatusOK, dtoFromAgentLogListResponse(resp))
}

// agentLogItem is the DTO for a single agent execution log entry.
type agentLogItem struct {
	ExecutionID  string          `json:"execution_id"`
	SessionID    *string         `json:"session_id"`
	ToolName     string          `json:"tool_name"`
	InputArgs    json.RawMessage `json:"input_args"`
	OutputResult json.RawMessage `json:"output_result"`
	Status       string          `json:"status"`
	ErrorMessage *string         `json:"error_message"`
	DurationMs   *int64          `json:"duration_ms"`
	LLMModel     *string         `json:"llm_model"`
	LLMTokens    *int64          `json:"llm_tokens"`
	CreatedAt    time.Time       `json:"created_at"`
}

// agentLogListResponse is the DTO for a paginated list of agent execution logs.
type agentLogListResponse struct {
	Items []agentLogItem `json:"items"`
	Total int            `json:"total"`
}

// dtoFromAgentLogListResponse converts service.AgentLogListResponse to agentLogListResponse.
func dtoFromAgentLogListResponse(m *service.AgentLogListResponse) *agentLogListResponse {
	if m == nil {
		return nil
	}

	items := make([]agentLogItem, len(m.Items))
	for i, item := range m.Items {
		items[i] = agentLogItem{
			ExecutionID:  item.ExecutionID,
			SessionID:    item.SessionID,
			ToolName:     item.ToolName,
			InputArgs:    item.InputArgs,
			OutputResult: item.OutputResult,
			Status:       item.Status,
			ErrorMessage: item.ErrorMessage,
			DurationMs:   item.DurationMs,
			LLMModel:     item.LLMModel,
			LLMTokens:    item.LLMTokens,
			CreatedAt:    item.CreatedAt,
		}
	}

	return &agentLogListResponse{Items: items, Total: m.Total}
}
