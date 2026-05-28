// Package agent_execution provides repository access for the agent execution domain.
// This is a domain subpackage of the repository layer with pool injection.
package agent_execution

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"baxi/internal/repository/common"
)

// Repository provides data access for agent execution logs.
type Repository struct {
	*common.PoolProvider
}

// NewRepository creates a new agent execution repository.
func NewRepository(provider *common.PoolProvider) *Repository {
	return &Repository{PoolProvider: provider}
}

// AgentExecution represents a row from ai.agent_execution.
type AgentExecution struct {
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

// Create inserts a new agent execution record.
func (r *Repository) Create(ctx context.Context, exec *AgentExecution) error {
	query := `
		INSERT INTO ai.agent_execution (
			execution_id, session_id, tool_name, input_args, output_result,
			status, error_message, duration_ms, llm_model, llm_tokens, created_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11
		)
	`
	now := time.Now().UTC()
	_, err := r.Pool().Exec(ctx, query,
		exec.ExecutionID,
		exec.SessionID,
		exec.ToolName,
		exec.InputArgs,
		exec.OutputResult,
		exec.Status,
		exec.ErrorMessage,
		exec.DurationMs,
		exec.LLMModel,
		exec.LLMTokens,
		now,
	)
	if err != nil {
		return fmt.Errorf("insert agent_execution: %w", err)
	}
	exec.CreatedAt = now
	return nil
}

// GetByID retrieves a single agent execution by its execution ID.
func (r *Repository) GetByID(ctx context.Context, executionID string) (*AgentExecution, error) {
	query := `
		SELECT execution_id, session_id, tool_name, input_args, output_result,
		       status, error_message, duration_ms, llm_model, llm_tokens, created_at
		FROM ai.agent_execution
		WHERE execution_id = $1
	`
	var exec AgentExecution
	err := r.QueryRow(ctx, query, executionID).Scan(
		&exec.ExecutionID,
		&exec.SessionID,
		&exec.ToolName,
		&exec.InputArgs,
		&exec.OutputResult,
		&exec.Status,
		&exec.ErrorMessage,
		&exec.DurationMs,
		&exec.LLMModel,
		&exec.LLMTokens,
		&exec.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("query agent_execution by id: %w", err)
	}
	return &exec, nil
}

// List retrieves agent execution records with pagination.
// Returns the matching rows and total count (unaffected by LIMIT/OFFSET).
func (r *Repository) List(ctx context.Context, limit, offset int) ([]AgentExecution, int, error) {
	query := `
		SELECT execution_id, session_id, tool_name, input_args, output_result,
		       status, error_message, duration_ms, llm_model, llm_tokens, created_at,
		       COUNT(*) OVER() AS total_count
		FROM ai.agent_execution
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query agent_execution list: %w", err)
	}
	defer rows.Close()

	var results []AgentExecution
	var totalCount int

	for rows.Next() {
		var exec AgentExecution
		var total int
		if err := rows.Scan(
			&exec.ExecutionID,
			&exec.SessionID,
			&exec.ToolName,
			&exec.InputArgs,
			&exec.OutputResult,
			&exec.Status,
			&exec.ErrorMessage,
			&exec.DurationMs,
			&exec.LLMModel,
			&exec.LLMTokens,
			&exec.CreatedAt,
			&total,
		); err != nil {
			return nil, 0, fmt.Errorf("scan agent_execution row: %w", err)
		}
		results = append(results, exec)
		totalCount = total
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate agent_execution rows: %w", err)
	}

	if results == nil {
		results = []AgentExecution{}
	}

	return results, totalCount, nil
}
