// Package ontology provides repository access for the ontology domain.
// This is a domain subpackage of the repository layer with pool injection.
package ontology

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"baxi/internal/repository/common"
)

// ──── Types (duplicated from repository package to avoid circular imports) ────

// ObjectFilters holds optional filters for object queries.
type ObjectFilters struct {
	ObjectType string
	Limit      int
	Offset     int
	Filters    map[string]interface{}
}

// ObjectInstance represents a single object instance from a dwd/mart/ops query.
type ObjectInstance struct {
	ObjectType string
	ID         string
	Properties map[string]interface{}
}

// ObjectQueryResult holds the result of a paginated object query.
type ObjectQueryResult struct {
	Rows  []ObjectInstance
	Total int
}

// ObjectMetrics holds metric values for a specific object.
type ObjectMetrics struct {
	ObjectType string
	ID         string
	Metrics    map[string]float64
}

// SearchFilters holds parameters for searching objects.
type SearchFilters struct {
	ObjectType string
	Query      string
	Limit      int
	Offset     int
}

// SearchResult holds the result of a paginated search.
type SearchResult struct {
	Rows  []ObjectInstance
	Total int
}

// ──── Table mappings ─────────────────────────────────────────────────────────

// tableMapping defines the source table for an object type.
type tableMapping struct {
	Schema      string
	Table       string
	PrimaryKey  string
	Columns     []string
	Aggregation string
}

// objectTableMap maps each object type to its source table and columns.
var objectTableMap = map[string]tableMapping{
	"customer": {
		Schema: "dwd", Table: "order_level", PrimaryKey: "customer_unique_id",
		Columns: []string{"customer_unique_id", "customer_state", "customer_city", "order_id", "total_payment_value", "review_score", "order_purchase_timestamp"},
	},
	"order": {
		Schema: "dwd", Table: "order_level", PrimaryKey: "order_id",
		Columns: []string{"order_id", "order_status", "order_purchase_timestamp", "total_payment_value", "payment_type", "review_score", "delivery_status"},
	},
	"seller": {
		Schema: "dwd", Table: "item_level", PrimaryKey: "seller_id",
		Columns: []string{"seller_id", "seller_state", "seller_city", "price", "order_id", "review_score"},
	},
	"product": {
		Schema: "dwd", Table: "item_level", PrimaryKey: "product_id",
		Columns: []string{"product_id", "product_category_name", "product_category_name_english", "price", "freight_value", "product_weight_g", "review_score"},
	},
	"category": {
		Schema: "dwd", Table: "item_level", PrimaryKey: "product_category_name",
		Columns: []string{"product_category_name", "product_category_name_english", "price", "order_id", "review_score"},
	},
	"region": {
		Schema: "dwd", Table: "order_level", PrimaryKey: "customer_state",
		Columns: []string{"customer_state", "customer_unique_id", "seller_state", "total_payment_value", "review_score", "order_purchase_timestamp"},
	},
	"marketing_lead": {
		Schema: "raw", Table: "marketing_qualified_leads", PrimaryKey: "mql_id",
		Columns: []string{"mql_id", "first_contact_date", "landing_page", "origin"},
	},
	"metric_alert": {
		Schema: "ops", Table: "metric_alert", PrimaryKey: "alert_id",
		Columns: []string{"alert_id", "rule_id", "metric", "severity", "current_value", "baseline_value", "status"},
	},
}

// roleTableAccess maps roles to accessible tables (schema.table format).
var roleTableAccess = map[string]map[string]bool{
	"admin": {
		"dwd.order_level":               true,
		"dwd.item_level":                true,
		"ops.metric_alert":              true,
		"raw.marketing_qualified_leads": true,
		"mart.metric_daily":             true,
		"mart.metric_dimension_daily":   true,
	},
	"analyst": {
		"dwd.item_level":              true,
		"ops.metric_alert":            true,
		"mart.metric_daily":           true,
		"mart.metric_dimension_daily": true,
	},
	"viewer": {
		"mart.metric_daily":           true,
		"mart.metric_dimension_daily": true,
		"ops.metric_alert":            true,
	},
	"marketing_ops": {
		"mart.metric_daily": true,
		"dwd.order_level":   true,
		"ops.metric_alert":  true,
	},
}

// metricColumns defines SQL aggregate expressions for GetObjectMetrics per object type.
var metricColumns = map[string][]struct {
	Name       string
	Expression string
}{
	"customer": {
		{Name: "total_orders", Expression: "COUNT(DISTINCT order_id)"},
		{Name: "total_spent", Expression: "COALESCE(SUM(payment_value), 0)"},
		{Name: "avg_review_score", Expression: "COALESCE(AVG(review_score), 0)"},
	},
	"order": {
		{Name: "payment_value", Expression: "COALESCE(SUM(payment_value), 0)"},
		{Name: "review_score", Expression: "COALESCE(AVG(review_score), 0)"},
	},
	"seller": {
		{Name: "total_sales", Expression: "COALESCE(SUM(price), 0)"},
		{Name: "total_items", Expression: "COUNT(*)"},
		{Name: "avg_review_score", Expression: "COALESCE(AVG(review_score), 0)"},
	},
	"product": {
		{Name: "total_sold", Expression: "COUNT(*)"},
		{Name: "avg_price", Expression: "COALESCE(AVG(price), 0)"},
		{Name: "avg_freight", Expression: "COALESCE(AVG(freight_value), 0)"},
		{Name: "avg_review_score", Expression: "COALESCE(AVG(review_score), 0)"},
	},
	"category": {
		{Name: "total_products", Expression: "COUNT(DISTINCT product_id)"},
		{Name: "total_sold", Expression: "COUNT(*)"},
		{Name: "avg_price", Expression: "COALESCE(AVG(price), 0)"},
	},
	"region": {
		{Name: "total_customers", Expression: "COUNT(DISTINCT customer_unique_id)"},
		{Name: "total_spent", Expression: "COALESCE(SUM(payment_value), 0)"},
		{Name: "avg_review_score", Expression: "COALESCE(AVG(review_score), 0)"},
	},
}

// searchableColumns defines columns searched via ILIKE per object type.
var searchableColumns = map[string][]string{
	"customer":       {"customer_unique_id", "customer_state", "customer_city"},
	"order":          {"order_id", "order_status", "payment_type"},
	"seller":         {"seller_id", "seller_state", "seller_city"},
	"product":        {"product_id", "product_category_name", "product_category_name_english"},
	"category":       {"product_category_name", "product_category_name_english"},
	"region":         {"customer_state", "seller_state"},
	"marketing_lead": {"mql_id", "landing_page", "origin"},
	"metric_alert":   {"alert_id", "rule_id", "metric", "severity", "status"},
}

// ──── RBAC helpers ───────────────────────────────────────────────────────────

type contextKey string

const roleContextKey contextKey = "ontology_role"

// WithRole returns a context with the given role for RBAC enforcement.
func WithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, roleContextKey, role)
}

func resolveRole(ctx context.Context) string {
	if role, ok := ctx.Value(roleContextKey).(string); ok && role != "" {
		return role
	}
	return "analyst"
}

func tableAccessible(role, schema, table string) bool {
	tableKey := schema + "." + table
	accessible, ok := roleTableAccess[role]
	if !ok {
		accessible = roleTableAccess["analyst"]
	}
	return accessible[tableKey]
}

func resolveLimit(requested int) (int, error) {
	if requested < 0 {
		return 0, fmt.Errorf("limit must be non-negative")
	}
	if requested == 0 {
		return 1000, nil
	}
	if requested > 10000 {
		return 0, fmt.Errorf("limit %d exceeds maximum of 10000", requested)
	}
	return requested, nil
}

func (m tableMapping) fullTableName() string {
	return m.Schema + "." + m.Table
}

// ──── Repository ─────────────────────────────────────────────────────────────

// Repository provides object queries against dwd/mart/ops tables with pool injection.
type Repository struct {
	*common.PoolProvider
}

// NewRepository creates a new ontology Repository.
func NewRepository(provider *common.PoolProvider) *Repository {
	return &Repository{PoolProvider: provider}
}

// QueryByObjectType queries objects by type, using the schema-based table mapping.
// Filters are applied as WHERE clauses. Limit defaults to 1000, max 10000.
// Role-based access is enforced using the role in context (default: analyst).
func (r *Repository) QueryByObjectType(ctx context.Context, objectType string, filters ObjectFilters) (*ObjectQueryResult, error) {
	mapping, ok := objectTableMap[objectType]
	if !ok {
		return nil, fmt.Errorf("unknown object type: %s", objectType)
	}

	role := resolveRole(ctx)
	if !tableAccessible(role, mapping.Schema, mapping.Table) {
		return nil, fmt.Errorf("role %q does not have access to %s.%s", role, mapping.Schema, mapping.Table)
	}

	tableName := mapping.fullTableName()
	cols := strings.Join(mapping.Columns, ", ")

	whereClauses := []string{}
	args := pgx.NamedArgs{}

	if filters.Filters != nil {
		for key, val := range filters.Filters {
			safeKey := sanitizeIdent(key)
			whereClauses = append(whereClauses, fmt.Sprintf("%s = @%s", safeKey, safeKey))
			args[safeKey] = val
		}
	}

	limit, err := resolveLimit(filters.Limit)
	if err != nil {
		return nil, fmt.Errorf("invalid limit: %w", err)
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s%s", tableName, whereSQL)
	var total int
	if err := r.QueryRow(ctx, countQuery, args).Scan(&total); err != nil {
		return nil, fmt.Errorf("count %s: %w", tableName, err)
	}

	dataQuery := fmt.Sprintf("SELECT %s FROM %s%s ORDER BY 1 LIMIT %d OFFSET %d",
		cols, tableName, whereSQL, limit, filters.Offset)

	rows, err := r.Query(ctx, dataQuery, args)
	if err != nil {
		return nil, fmt.Errorf("query %s: %w", tableName, err)
	}
	defer rows.Close()

	var results []ObjectInstance
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		props := make(map[string]interface{}, len(mapping.Columns))
		for i, col := range mapping.Columns {
			props[col] = values[i]
		}

		id := formatID(mapping.PrimaryKey, props)

		results = append(results, ObjectInstance{
			ObjectType: objectType,
			ID:         id,
			Properties: props,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}

	if results == nil {
		results = []ObjectInstance{}
	}

	return &ObjectQueryResult{
		Rows:  results,
		Total: total,
	}, nil
}

// GetObjectByID retrieves a single object by its ID.
func (r *Repository) GetObjectByID(ctx context.Context, objectType, objectID string) (*ObjectInstance, error) {
	mapping, ok := objectTableMap[objectType]
	if !ok {
		return nil, fmt.Errorf("unknown object type: %s", objectType)
	}

	role := resolveRole(ctx)
	if !tableAccessible(role, mapping.Schema, mapping.Table) {
		return nil, fmt.Errorf("role %q does not have access to %s.%s", role, mapping.Schema, mapping.Table)
	}

	tableName := mapping.fullTableName()
	cols := strings.Join(mapping.Columns, ", ")
	pk := sanitizeIdent(mapping.PrimaryKey)

	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s = $1 LIMIT 1", cols, tableName, pk)

	rows, err := r.Query(ctx, query, objectID)
	if err != nil {
		return nil, fmt.Errorf("get %s by id: %w", objectType, err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, fmt.Errorf("%s with %s=%q not found", objectType, mapping.PrimaryKey, objectID)
	}

	values, err := rows.Values()
	if err != nil {
		return nil, fmt.Errorf("scan row: %w", err)
	}

	props := make(map[string]interface{}, len(mapping.Columns))
	for i, col := range mapping.Columns {
		props[col] = values[i]
	}

	return &ObjectInstance{
		ObjectType: objectType,
		ID:         objectID,
		Properties: props,
	}, nil
}

// GetObjectMetrics retrieves metrics for a specific object.
// Metrics are computed as SQL aggregates from the object's mapped source table.
func (r *Repository) GetObjectMetrics(ctx context.Context, objectType, objectID string) (*ObjectMetrics, error) {
	mapping, ok := objectTableMap[objectType]
	if !ok {
		return nil, fmt.Errorf("unknown object type: %s", objectType)
	}

	role := resolveRole(ctx)
	if !tableAccessible(role, mapping.Schema, mapping.Table) {
		return nil, fmt.Errorf("role %q does not have access to %s.%s", role, mapping.Schema, mapping.Table)
	}

	metrics := make(map[string]float64)

	if aggMetrics, hasMetrics := metricColumns[objectType]; hasMetrics {
		tableName := mapping.fullTableName()
		pk := sanitizeIdent(mapping.PrimaryKey)

		exprs := make([]string, len(aggMetrics))
		for i, m := range aggMetrics {
			exprs[i] = m.Expression + " AS " + sanitizeIdent(m.Name)
		}

		query := fmt.Sprintf("SELECT %s FROM %s WHERE %s = $1",
			strings.Join(exprs, ", "), tableName, pk)

		row := r.QueryRow(ctx, query, objectID)
		scanTargets := make([]interface{}, len(aggMetrics))
		for i := range aggMetrics {
			scanTargets[i] = new(float64)
		}

		if err := row.Scan(scanTargets...); err != nil {
			return nil, fmt.Errorf("query metrics for %s %s: %w", objectType, objectID, err)
		}

		for i, m := range aggMetrics {
			metrics[m.Name] = *(scanTargets[i].(*float64))
		}
	}

	return &ObjectMetrics{
		ObjectType: objectType,
		ID:         objectID,
		Metrics:    metrics,
	}, nil
}

// SearchObjects searches for objects matching the given filters.
// The query string is matched against searchable columns using ILIKE.
func (r *Repository) SearchObjects(ctx context.Context, objectType string, filters SearchFilters) (*SearchResult, error) {
	mapping, ok := objectTableMap[objectType]
	if !ok {
		return nil, fmt.Errorf("unknown object type: %s", objectType)
	}

	role := resolveRole(ctx)
	if !tableAccessible(role, mapping.Schema, mapping.Table) {
		return nil, fmt.Errorf("role %q does not have access to %s.%s", role, mapping.Schema, mapping.Table)
	}

	tableName := mapping.fullTableName()
	cols := strings.Join(mapping.Columns, ", ")

	limit, err := resolveLimit(filters.Limit)
	if err != nil {
		return nil, fmt.Errorf("invalid limit: %w", err)
	}

	searchCols, hasSearch := searchableColumns[objectType]
	whereSQL := ""
	args := pgx.NamedArgs{}

	if hasSearch && filters.Query != "" {
		likeClauses := make([]string, len(searchCols))
		for i, col := range searchCols {
			safeCol := sanitizeIdent(col)
			likeClauses[i] = fmt.Sprintf("%s ILIKE @q", safeCol)
		}
		whereSQL = " WHERE " + strings.Join(likeClauses, " OR ")
		args["q"] = "%" + filters.Query + "%"
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s%s", tableName, whereSQL)
	var total int
	if err := r.QueryRow(ctx, countQuery, args).Scan(&total); err != nil {
		return nil, fmt.Errorf("count search %s: %w", tableName, err)
	}

	dataQuery := fmt.Sprintf("SELECT %s FROM %s%s ORDER BY 1 LIMIT %d OFFSET %d",
		cols, tableName, whereSQL, limit, filters.Offset)

	rows, err := r.Query(ctx, dataQuery, args)
	if err != nil {
		return nil, fmt.Errorf("search %s: %w", tableName, err)
	}
	defer rows.Close()

	var results []ObjectInstance
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		props := make(map[string]interface{}, len(mapping.Columns))
		for i, col := range mapping.Columns {
			props[col] = values[i]
		}

		id := formatID(mapping.PrimaryKey, props)

		results = append(results, ObjectInstance{
			ObjectType: objectType,
			ID:         id,
			Properties: props,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate search rows: %w", err)
	}

	if results == nil {
		results = []ObjectInstance{}
	}

	return &SearchResult{
		Rows:  results,
		Total: total,
	}, nil
}

// ──── Utility functions ──────────────────────────────────────────────────────

func sanitizeIdent(ident string) string {
	if strings.HasPrefix(ident, `"`) && strings.HasSuffix(ident, `"`) {
		return ident
	}
	return `"` + ident + `"`
}

func formatID(pk string, props map[string]interface{}) string {
	if raw, ok := props[pk]; ok && raw != nil {
		return fmt.Sprintf("%v", raw)
	}
	return ""
}
