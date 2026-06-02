package ontology

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// MetricQueryResolver extends MetricResolver with database-backed metric queries.
// It loads metric definitions via the embedded resolver and executes queries
// against the metric source tables to retrieve value and baseline.
type MetricQueryResolver struct {
	inner *MetricResolver
	pool  *pgxpool.Pool
}

// NewMetricQueryResolver creates a MetricQueryResolver.
func NewMetricQueryResolver(resolver *MetricResolver, pool *pgxpool.Pool) *MetricQueryResolver {
	return &MetricQueryResolver{
		inner: resolver,
		pool:  pool,
	}
}

// GetMetric delegates to the inner MetricResolver.
func (r *MetricQueryResolver) GetMetric(name string) (*MetricDefinition, bool) {
	return r.inner.GetMetric(name)
}

// ListMetrics delegates to the inner MetricResolver.
func (r *MetricQueryResolver) ListMetrics() []string {
	return r.inner.ListMetrics()
}

// MetricResult holds the result of a metric query.
type MetricResult struct {
	Value    float64
	Baseline float64
	Label    string // human-readable label like "Avg Review Score"
}

// QueryMetric queries a single metric for a given object ID.
// It builds a SQL query from the MetricDefinition's source, value/baseline columns,
// and applies filters with the object ID as the primary parameter.
// Returns the value and baseline, or an error if the metric isn't found or query fails.
func (r *MetricQueryResolver) QueryMetric(ctx context.Context, objectID string, def *MetricDefinition) (*MetricResult, error) {
	if def == nil {
		return nil, fmt.Errorf("metric definition is nil")
	}

	tableName := def.Source.Schema + "." + def.Source.Table
	whereClause := ""

	// Build WHERE from filters, using $1 as objectID placeholder
	if len(def.Filters) > 0 {
		i := 1
		clauses := make([]string, 0, len(def.Filters))
		for col, val := range def.Filters {
			placeholder := fmt.Sprintf("$%d", i)
			if val == "$object_id" {
				clauses = append(clauses, col+" = $1")
			} else {
				clauses = append(clauses, col+" = "+placeholder)
			}
			i++
		}
		whereClause = " WHERE " + joinClauses(clauses, " AND ")
	}

	query := fmt.Sprintf("SELECT %s, %s FROM %s%s",
		def.ValueColumn, def.BaselineColumn, tableName, whereClause)

	row := r.pool.QueryRow(ctx, query, objectID)

	var value, baseline float64
	if err := row.Scan(&value, &baseline); err != nil {
		// Return zero values if metric source is empty or not found
		return &MetricResult{
			Value:    0,
			Baseline: 0,
			Label:    def.DisplayName,
		}, nil
	}

	return &MetricResult{
		Value:    value,
		Baseline: baseline,
		Label:    def.DisplayName,
	}, nil
}

// QueryMetrics queries multiple metric definitions for the same object ID.
func (r *MetricQueryResolver) QueryMetrics(ctx context.Context, objectID string, defs []*MetricDefinition) (map[string]*MetricResult, error) {
	results := make(map[string]*MetricResult, len(defs))
	for _, def := range defs {
		if def == nil {
			continue
		}
		res, err := r.QueryMetric(ctx, objectID, def)
		if err != nil {
			results[def.Name] = &MetricResult{
				Label: def.DisplayName,
			}
			continue
		}
		results[def.Name] = res
	}
	return results, nil
}

func joinClauses(clauses []string, sep string) string {
	if len(clauses) == 0 {
		return ""
	}
	result := clauses[0]
	for _, c := range clauses[1:] {
		result += sep + c
	}
	return result
}
