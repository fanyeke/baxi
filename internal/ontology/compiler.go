package ontology

import (
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// ──── CompileGetObject ────────────────────────────────────────────────────────

// CompileGetObject compiles a parameterized SELECT query for fetching a single
// object by its primary key.
//
// The query includes all properties as columns. Properties with Aggregation set
// are wrapped in their aggregation function (SUM, COUNT, etc.) and excluded from
// GROUP BY. Non-aggregated properties are included in GROUP BY. The result is
// limited to 1 row.
//
// Security: column/data names come from the ontology schema (not user input),
// identifiers are sanitized via pgx.Identifier.Sanitize(), and the only user
// input is the objectID value passed as a safe named parameter.
func (qc *QueryCompiler) CompileGetObject(objectType, objectID string) (*CompiledQuery, error) {
	ot, ok := qc.objects[objectType]
	if !ok {
		return nil, fmt.Errorf("unknown object type: %s", objectType)
	}

	var selectCols []string
	var columns []string
	var nonAggCols []string
	args := pgx.NamedArgs{}

	for name, prop := range ot.Properties {
		colExpr := prop.SourceField
		if prop.Expression != "" {
			colExpr = prop.Expression
		}
		// Wrap with aggregation if set
		if prop.Aggregation != "" {
			colExpr = wrapAggregation(prop.Aggregation, colExpr)
		} else {
			nonAggCols = append(nonAggCols, pgx.Identifier{name}.Sanitize())
		}
		selectCols = append(selectCols, colExpr+" AS "+pgx.Identifier{name}.Sanitize())
		columns = append(columns, name)
	}

	schema := pgx.Identifier{ot.Source.Schema}.Sanitize()
	table := pgx.Identifier{ot.Source.Table}.Sanitize()
	pk := pgx.Identifier{ot.Source.PrimaryKey}.Sanitize()
	args["pk"] = objectID

	// GROUP BY non-aggregated columns when there are both agg and non-agg cols
	groupBy := ""
	if len(nonAggCols) > 0 && len(nonAggCols) < len(selectCols) {
		groupBy = " GROUP BY " + strings.Join(nonAggCols, ", ")
	}

	sql := fmt.Sprintf("SELECT %s FROM %s.%s WHERE %s = @pk%s LIMIT 1",
		strings.Join(selectCols, ", "), schema, table, pk, groupBy)

	return &CompiledQuery{
		SQL:        sql,
		Args:       args,
		Columns:    columns,
		ObjectType: objectType,
		PrimaryKey: ot.Source.PrimaryKey,
		Schema:     ot.Source.Schema,
		Table:      ot.Source.Table,
	}, nil
}

// ──── CompileSearchObjects ────────────────────────────────────────────────────

// CompileSearchObjects compiles a parameterized SELECT query for searching
// objects of the given type. Only filterable properties (Filterable=true) are
// accepted in filters. Only searchable properties (Searchable=true) are accepted
// as Sort. LIMIT is capped at qc.maxLimit.
//
// The returned CompiledQuery has both SQL (data query with ORDER BY, no LIMIT)
// and CountSQL (COUNT query without ORDER BY) so the caller can resolve totals
// and add LIMIT/OFFSET independently.
//
// Security: filter/sort column names are validated against the ontology schema;
// identifiers are sanitized; all filter values are passed as safe named args.
func (qc *QueryCompiler) CompileSearchObjects(objectType string, filters ObjectFilters) (*CompiledQuery, error) {
	ot, ok := qc.objects[objectType]
	if !ok {
		return nil, fmt.Errorf("unknown object type: %s", objectType)
	}

	args := pgx.NamedArgs{}
	var whereClauses []string
	var selectCols []string
	var columns []string

	// Build SELECT columns
	for name, prop := range ot.Properties {
		colExpr := prop.SourceField
		if prop.Expression != "" {
			colExpr = prop.Expression
		}
		selectCols = append(selectCols, colExpr+" AS "+pgx.Identifier{name}.Sanitize())
		columns = append(columns, name)
	}

	// Apply filters — only Filterable=true properties
	if filters.Filters != nil {
		for key, val := range filters.Filters {
			prop, ok := ot.Properties[key]
			if !ok || !prop.Filterable {
				continue // skip invalid or non-filterable columns
			}
			safeCol := pgx.Identifier{prop.SourceField}.Sanitize()
			paramName := sanitizeParamName(key)
			whereClauses = append(whereClauses, fmt.Sprintf("%s = @%s", safeCol, paramName))
			args[paramName] = val
		}
	}

	// Build WHERE clause
	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Build ORDER BY — only Searchable=true properties
	orderSQL := ""
	if filters.Sort != "" {
		prop, ok := ot.Properties[filters.Sort]
		if ok && prop.Searchable {
			order := strings.ToUpper(filters.Order)
			if order != "ASC" && order != "DESC" {
				order = "ASC"
			}
			safeSort := pgx.Identifier{prop.SourceField}.Sanitize()
			orderSQL = fmt.Sprintf(" ORDER BY %s %s", safeSort, order)
		}
	}

	// Resolve LIMIT
	limit := filters.Limit
	if limit <= 0 || limit > qc.maxLimit {
		limit = qc.maxLimit
	}

	schema := pgx.Identifier{ot.Source.Schema}.Sanitize()
	table := pgx.Identifier{ot.Source.Table}.Sanitize()
	colsSQL := strings.Join(selectCols, ", ")

	// Data SQL: SELECT cols FROM schema.table WHERE ... ORDER BY ...
	dataSQL := fmt.Sprintf("SELECT %s FROM %s.%s%s%s",
		colsSQL, schema, table, whereSQL, orderSQL)

	// Count SQL: SELECT COUNT(*) FROM schema.table WHERE ... (no ORDER BY)
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM %s.%s%s",
		schema, table, whereSQL)

	return &CompiledQuery{
		SQL:        dataSQL,
		CountSQL:   countSQL,
		Args:       args,
		Columns:    columns,
		ObjectType: objectType,
		PrimaryKey: ot.Source.PrimaryKey,
		Schema:     ot.Source.Schema,
		Table:      ot.Source.Table,
	}, nil
}

// ──── CompileObjectMetrics ────────────────────────────────────────────────────

// CompileObjectMetrics compiles a parameterized SELECT query for computing
// aggregate metrics for a single object. Only properties with an Aggregation
// function are included in the result. If no properties have aggregation, a
// fallback selects all properties as plain values.
//
// Security: identifiers are sanitized, and the only user input is the objectID
// passed as a safe named parameter.
func (qc *QueryCompiler) CompileObjectMetrics(objectType, objectID string) (*CompiledQuery, error) {
	ot, ok := qc.objects[objectType]
	if !ok {
		return nil, fmt.Errorf("unknown object type: %s", objectType)
	}

	var selectCols []string
	var columns []string
	args := pgx.NamedArgs{}

	// Collect aggregate properties
	for name, prop := range ot.Properties {
		if prop.Aggregation == "" {
			continue
		}
		colExpr := prop.SourceField
		if prop.Expression != "" {
			colExpr = prop.Expression
		}
		colExpr = wrapAggregation(prop.Aggregation, colExpr)
		selectCols = append(selectCols, colExpr+" AS "+pgx.Identifier{name}.Sanitize())
		columns = append(columns, name)
	}

	// Fallback: if no aggregate properties, select all properties as plain values
	if len(selectCols) == 0 {
		for name, prop := range ot.Properties {
			colExpr := prop.SourceField
			if prop.Expression != "" {
				colExpr = prop.Expression
			}
			selectCols = append(selectCols, colExpr+" AS "+pgx.Identifier{name}.Sanitize())
			columns = append(columns, name)
		}
	}

	schema := pgx.Identifier{ot.Source.Schema}.Sanitize()
	table := pgx.Identifier{ot.Source.Table}.Sanitize()
	pk := pgx.Identifier{ot.Source.PrimaryKey}.Sanitize()
	args["pk"] = objectID

	sql := fmt.Sprintf("SELECT %s FROM %s.%s WHERE %s = @pk",
		strings.Join(selectCols, ", "), schema, table, pk)

	return &CompiledQuery{
		SQL:        sql,
		Args:       args,
		Columns:    columns,
		ObjectType: objectType,
		PrimaryKey: ot.Source.PrimaryKey,
		Schema:     ot.Source.Schema,
		Table:      ot.Source.Table,
	}, nil
}

// ──── helpers ─────────────────────────────────────────────────────────────────

// wrapAggregation wraps a column expression in the requested aggregation function.
func wrapAggregation(agg, expr string) string {
	switch agg {
	case "count":
		return fmt.Sprintf("COUNT(%s)", expr)
	case "count_distinct":
		return fmt.Sprintf("COUNT(DISTINCT %s)", expr)
	case "sum":
		return fmt.Sprintf("COALESCE(SUM(%s), 0)", expr)
	case "avg":
		return fmt.Sprintf("COALESCE(AVG(%s), 0)", expr)
	case "min":
		return fmt.Sprintf("MIN(%s)", expr)
	case "max":
		return fmt.Sprintf("MAX(%s)", expr)
	default:
		return expr
	}
}

// sanitizeParamName converts a filter key into a safe pgx named-arg identifier.
// Named args must match [a-zA-Z_][a-zA-Z0-9_]*.
func sanitizeParamName(key string) string {
	var sb strings.Builder
	for i, r := range key {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			sb.WriteRune(r)
		} else if r == '-' || r == '.' || r == ' ' {
			sb.WriteRune('_')
		} else if i == 0 {
			sb.WriteRune('f')
		}
	}
	result := sb.String()
	if result == "" {
		return "f"
	}
	// Ensure it starts with a letter or underscore
	if first := result[0]; !(first >= 'a' && first <= 'z') && !(first >= 'A' && first <= 'Z') && first != '_' {
		result = "f_" + result
	}
	return result
}
