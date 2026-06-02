package ontology

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"baxi/internal/repository/common"
)

// CompiledQuery holds a fully resolved query with parameterized SQL and metadata.
// This is a local copy mirroring ontology.CompiledQuery to avoid import cycles.
type CompiledQuery struct {
	SQL        string
	Args       []any
	Columns    []string
	ObjectType string
	PrimaryKey string
	Schema     string
	Table      string
}

// QueryExecutor executes compiled queries against the database.
// Uses PoolProvider pattern; returns ObjectInstance results.
type QueryExecutor struct {
	common.Querier
}

// NewQueryExecutor creates a QueryExecutor.
func NewQueryExecutor(provider common.Querier) *QueryExecutor {
	return &QueryExecutor{Querier: provider}
}

// ExecuteQuery runs a compiled query and returns results as ObjectInstances.
// Returns empty slice (not nil) on zero results.
func (e *QueryExecutor) ExecuteQuery(ctx context.Context, cq CompiledQuery) ([]ObjectInstance, error) {
	rows, err := e.Query(ctx, cq.SQL, cq.Args...)
	if err != nil {
		return nil, fmt.Errorf("execute compiled query for %s: %w", cq.ObjectType, err)
	}
	defer rows.Close()

	var results []ObjectInstance
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("scan row for %s: %w", cq.ObjectType, err)
		}

		props := make(map[string]interface{}, len(cq.Columns))
		for i, col := range cq.Columns {
			if i < len(values) {
				props[col] = values[i]
			}
		}

		id := extractID(cq.PrimaryKey, props)

		results = append(results, ObjectInstance{
			ObjectType: cq.ObjectType,
			ID:         id,
			Properties: props,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows for %s: %w", cq.ObjectType, err)
	}

	if results == nil {
		results = []ObjectInstance{}
	}

	return results, nil
}

// ExecuteQuerySingle returns the first result of a compiled query.
// Returns nil if no rows match.
func (e *QueryExecutor) ExecuteQuerySingle(ctx context.Context, cq CompiledQuery) (*ObjectInstance, error) {
	results, err := e.ExecuteQuery(ctx, cq)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	return &results[0], nil
}

// ExecuteRawQuery runs raw SQL with args and maps columns to ObjectInstances.
func (e *QueryExecutor) ExecuteRawQuery(ctx context.Context, sql string, args []any, objectType, primaryKey string, columns []string) ([]ObjectInstance, error) {
	cq := CompiledQuery{
		SQL:        sql,
		Args:       args,
		Columns:    columns,
		ObjectType: objectType,
		PrimaryKey: primaryKey,
	}
	return e.ExecuteQuery(ctx, cq)
}

func extractID(pk string, props map[string]interface{}) string {
	if pk == "" {
		return ""
	}
	if raw, ok := props[pk]; ok && raw != nil {
		return fmt.Sprintf("%v", raw)
	}
	sanitized := pgx.Identifier{pk}.Sanitize()
	if raw, ok := props[sanitized]; ok && raw != nil {
		return fmt.Sprintf("%v", raw)
	}
	return ""
}
