// Package mcp_call provides repository access for MCP call audit logs.
// This is a domain subpackage of the repository layer with pool injection.
package mcp_call

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"baxi/internal/repository/common"
)

// Repository provides data access for MCP call audit logs.
type Repository struct {
	common.Querier
}

// NewRepository creates a new MCP call repository.
func NewRepository(provider common.Querier) *Repository {
	return &Repository{Querier: provider}
}

// MCPCall represents a row from audit.mcp_call.
type MCPCall struct {
	CallID       int64
	RequestID    *string
	ServerName   string
	ToolName     string
	InputArgs    json.RawMessage
	OutputResult json.RawMessage
	Status       string
	ErrorMessage *string
	DurationMs   *int64
	CreatedAt    time.Time
}

// Create inserts a new MCP call audit record.
// The call_id and created_at fields are populated by the database via RETURNING.
func (r *Repository) Create(ctx context.Context, call *MCPCall) error {
	query := `
		INSERT INTO audit.mcp_call (
			request_id, server_name, tool_name, input_args, output_result,
			status, error_message, duration_ms, created_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9
		)
		RETURNING call_id, created_at
	`
	now := time.Now().UTC()
	err := r.QueryRow(ctx, query,
		call.RequestID,
		call.ServerName,
		call.ToolName,
		call.InputArgs,
		call.OutputResult,
		call.Status,
		call.ErrorMessage,
		call.DurationMs,
		now,
	).Scan(&call.CallID, &call.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert mcp_call: %w", err)
	}
	return nil
}

// GetByID retrieves a single MCP call record by its call ID.
func (r *Repository) GetByID(ctx context.Context, callID int64) (*MCPCall, error) {
	query := `
		SELECT call_id, request_id, server_name, tool_name, input_args, output_result,
		       status, error_message, duration_ms, created_at
		FROM audit.mcp_call
		WHERE call_id = $1
	`
	var call MCPCall
	err := r.QueryRow(ctx, query, callID).Scan(
		&call.CallID,
		&call.RequestID,
		&call.ServerName,
		&call.ToolName,
		&call.InputArgs,
		&call.OutputResult,
		&call.Status,
		&call.ErrorMessage,
		&call.DurationMs,
		&call.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("query mcp_call by id: %w", err)
	}
	return &call, nil
}

// List retrieves MCP call records with pagination.
// Returns the matching rows and total count (unaffected by LIMIT/OFFSET).
func (r *Repository) List(ctx context.Context, limit, offset int) ([]MCPCall, int, error) {
	query := `
		SELECT call_id, request_id, server_name, tool_name, input_args, output_result,
		       status, error_message, duration_ms, created_at,
		       COUNT(*) OVER() AS total_count
		FROM audit.mcp_call
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query mcp_call list: %w", err)
	}
	defer rows.Close()

	var results []MCPCall
	var totalCount int

	for rows.Next() {
		var call MCPCall
		var total int
		if err := rows.Scan(
			&call.CallID,
			&call.RequestID,
			&call.ServerName,
			&call.ToolName,
			&call.InputArgs,
			&call.OutputResult,
			&call.Status,
			&call.ErrorMessage,
			&call.DurationMs,
			&call.CreatedAt,
			&total,
		); err != nil {
			return nil, 0, fmt.Errorf("scan mcp_call row: %w", err)
		}
		results = append(results, call)
		totalCount = total
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate mcp_call rows: %w", err)
	}

	if results == nil {
		results = []MCPCall{}
	}

	return results, totalCount, nil
}
