package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	agent_execution "baxi/internal/repository/agent_execution"
	mcp_call "baxi/internal/repository/mcp_call"
)

// AgentExecutionRepository defines data access for agent execution logs.
type AgentExecutionRepository interface {
	Create(ctx context.Context, execution *agent_execution.AgentExecution) error
	List(ctx context.Context, limit, offset int) ([]agent_execution.AgentExecution, int, error)
}

// MCPCallRepository defines data access for MCP call audit logs.
type MCPCallRepository interface {
	Create(ctx context.Context, call *mcp_call.MCPCall) error
	List(ctx context.Context, limit, offset int) ([]mcp_call.MCPCall, int, error)
}

// AgentExecutionLog is a service-level DTO for agent execution records.
type AgentExecutionLog struct {
	ExecutionID  string
	SessionID    *string
	ToolName     string
	InputArgs    json.RawMessage
	OutputResult json.RawMessage
	Status       string
	ErrorMessage *string
	DurationMs   *int64
	LLMModel     *string
	LLMTokens    *int64
	CreatedAt    time.Time
}

// AgentLogListResponse holds a paginated list of agent execution logs.
type AgentLogListResponse struct {
	Items []AgentExecutionLog
	Total int
}

// AgentLogService provides business logic for agent execution and MCP call logging.
type AgentLogService struct {
	agentExecutionRepo AgentExecutionRepository
	mcpCallRepo        MCPCallRepository
}

// NewAgentLogService creates a new AgentLogService.
func NewAgentLogService(agentExecutionRepo AgentExecutionRepository, mcpCallRepo MCPCallRepository) *AgentLogService {
	return &AgentLogService{
		agentExecutionRepo: agentExecutionRepo,
		mcpCallRepo:        mcpCallRepo,
	}
}

// LogExecution records an agent execution event.
func (s *AgentLogService) LogExecution(ctx context.Context, log *AgentExecutionLog) error {
	exec := &agent_execution.AgentExecution{
		ExecutionID:  log.ExecutionID,
		SessionID:    log.SessionID,
		ToolName:     log.ToolName,
		InputArgs:    log.InputArgs,
		OutputResult: log.OutputResult,
		Status:       log.Status,
		ErrorMessage: log.ErrorMessage,
		DurationMs:   log.DurationMs,
		LLMModel:     log.LLMModel,
		LLMTokens:    log.LLMTokens,
	}
	if err := s.agentExecutionRepo.Create(ctx, exec); err != nil {
		return fmt.Errorf("log execution: %w", err)
	}
	log.CreatedAt = exec.CreatedAt
	return nil
}

// ListAgentLogs retrieves paginated agent execution logs.
func (s *AgentLogService) ListAgentLogs(ctx context.Context, limit, offset int) (*AgentLogListResponse, error) {
	rows, total, err := s.agentExecutionRepo.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list agent logs: %w", err)
	}

	items := make([]AgentExecutionLog, len(rows))
	for i, row := range rows {
		items[i] = AgentExecutionLog{
			ExecutionID:  row.ExecutionID,
			SessionID:    row.SessionID,
			ToolName:     row.ToolName,
			InputArgs:    row.InputArgs,
			OutputResult: row.OutputResult,
			Status:       row.Status,
			ErrorMessage: row.ErrorMessage,
			DurationMs:   row.DurationMs,
			LLMModel:     row.LLMModel,
			LLMTokens:    row.LLMTokens,
			CreatedAt:    row.CreatedAt,
		}
	}

	return &AgentLogListResponse{Items: items, Total: total}, nil
}
