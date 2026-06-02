package ontology

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"baxi/internal/repository/common"
)

// validSortExpr validates a SQL ORDER BY expression to prevent SQL injection.
// Allows: column names with optional ASC/DESC (case-insensitive), comma-separated
// multi-column sorts. Only plain spaces are allowed as separators (no tabs/newlines).
var validSortExpr = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*([ ]+(ASC|DESC|asc|desc))?([ ]*,[ ]*[a-zA-Z_][a-zA-Z0-9_]*([ ]+(ASC|DESC|asc|desc))?)*$`)

func isValidSort(sort string) bool {
	if sort == "" {
		return true
	}
	return validSortExpr.MatchString(sort)
}

// LinkOptions configures how linked objects are fetched.
type LinkOptions struct {
	SourceType   string // source object type name
	SourceID     string // source object ID value
	TargetType   string // target object type name
	TargetSchema string // target table schema
	TargetTable  string // target table name
	TargetKey    string // join key on target table
	ObjectIDField string // field holding target object's ID
	Strategy     string // direct_key, reverse_lookup, bridge_table, query_ref
	SourceKey    string // field on the source object
	Limit        int    // max results (0 = no limit)
	Sort         string // sort expression
	Fields       []string // fields to return
}

// LinkExecutor resolves relationships between objects using LinkResolver strategies.
// Supports reverse_lookup strategy for one_to_many relationships.
type LinkExecutor struct {
	common.Querier
}

// NewLinkExecutor creates a LinkExecutor.
func NewLinkExecutor(provider common.Querier) *LinkExecutor {
	return &LinkExecutor{Querier: provider}
}

// ExecuteLink resolves a link for the given source object and returns linked ObjectInstances.
// strategy "reverse_lookup": WHERE target_key = source_id
// strategy "direct_key": WHERE target_key = source_property_value
func (e *LinkExecutor) ExecuteLink(ctx context.Context, opts LinkOptions) ([]ObjectInstance, error) {
	switch opts.Strategy {
	case "reverse_lookup":
		return e.executeReverseLookup(ctx, opts)
	case "direct_key":
		return e.executeDirectKey(ctx, opts)
	default:
		return e.executeReverseLookup(ctx, opts)
	}
}

// executeReverseLookup queries the target table where target key matches source ID.
// For example: orders WHERE customer_id = $1
func (e *LinkExecutor) executeReverseLookup(ctx context.Context, opts LinkOptions) ([]ObjectInstance, error) {
	tableName := sanitizeIdent(opts.TargetSchema) + "." + sanitizeIdent(opts.TargetTable)
	pk := sanitizeIdent(opts.TargetKey)

	var cols string
	if len(opts.Fields) > 0 {
		quoted := make([]string, len(opts.Fields))
		for i, f := range opts.Fields {
			quoted[i] = sanitizeIdent(f)
		}
		cols = strings.Join(quoted, ", ")
	} else {
		cols = "*"
	}

	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s = $1", cols, tableName, pk)
	args := []interface{}{opts.SourceID}

	if opts.Sort != "" {
		if !isValidSort(opts.Sort) {
			return nil, fmt.Errorf("invalid sort expression for %s->%s: %q", opts.SourceType, opts.TargetType, opts.Sort)
		}
		query += " ORDER BY " + opts.Sort
	}
	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}

	rows, err := e.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("execute reverse_lookup for %s->%s: %w", opts.SourceType, opts.TargetType, err)
	}
	defer rows.Close()

	var results []ObjectInstance
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("scan link row for %s: %w", opts.TargetType, err)
		}

		props := make(map[string]interface{}, len(values))
		fieldNames := rows.FieldDescriptions()
		for i := range values {
			if i < len(fieldNames) {
				props[string(fieldNames[i].Name)] = values[i]
			} else {
				props[fmt.Sprintf("col_%d", i)] = values[i]
			}
		}

		id := extractID(opts.ObjectIDField, props)

		results = append(results, ObjectInstance{
			ObjectType: opts.TargetType,
			ID:         id,
			Properties: props,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate link rows for %s: %w", opts.TargetType, err)
	}

	if results == nil {
		results = []ObjectInstance{}
	}

	return results, nil
}

// executeDirectKey queries the source object via a direct key match.
func (e *LinkExecutor) executeDirectKey(ctx context.Context, opts LinkOptions) ([]ObjectInstance, error) {
	return e.executeReverseLookup(ctx, opts)
}


