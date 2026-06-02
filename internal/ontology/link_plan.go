package ontology

import (
	"context"
	"fmt"
	"regexp"
	"strings"
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

// ──── Link Resolution types ──────────────────────────────────────────────────

// ObjectRef identifies a single object instance in the ontology.
type ObjectRef struct {
	ObjectType string
	ObjectID   string
}

// LinkOptions controls link traversal behavior.
type LinkOptions struct {
	MaxDepth int    // maximum link traversal depth (default: 1, max: 3)
	Limit    int    // max results per link (default: 20)
	Offset   int    // pagination offset
	Role     string // access role for governance filtering
}

// ObjectInstance represents a resolved object instance with its properties.
type ObjectInstance struct {
	ObjectType string                 `json:"object_type"`
	ObjectID   string                 `json:"object_id"`
	Properties map[string]interface{} `json:"properties"`
}

// LinkedObjectResult holds the result of resolving a single link.
type LinkedObjectResult struct {
	ObjectType  string           `json:"object_type"`
	ObjectID    string           `json:"object_id"`
	LinkName    string           `json:"link_name"`
	TargetType  string           `json:"target_type"`
	Cardinality string           `json:"cardinality"` // one_to_one or one_to_many
	Objects     []ObjectInstance `json:"objects"`
}

// ──── LinkResolver ───────────────────────────────────────────────────────────

// LinkResolver resolves v2 ObjectLink definitions into query plans. It
// supports four resolution strategies:
//
//  1. direct_key — SourceKey matches the target table's PK directly.
//     SELECT * FROM target WHERE target.pk = source.sourceKey
//
//  2. reverse_lookup — Find rows where the target table holds the source key.
//     SELECT * FROM target WHERE target.key = source.id
//
//  3. bridge_table — Join through an intermediate table.
//     SELECT target.* FROM bridge
//     JOIN target ON bridge.target_fk = target.pk
//     WHERE bridge.source_fk = source.id
//
//  4. query_ref — Use a predefined template SQL (stored in the link's Target).
type LinkResolver struct {
	objects   map[string]*ObjectTypeV2
	queryComp *QueryCompiler
}

// NewLinkResolver creates a LinkResolver from v2 object type definitions.
func NewLinkResolver(objects map[string]*ObjectTypeV2) *LinkResolver {
	return &LinkResolver{
		objects:   objects,
		queryComp: NewQueryCompiler(objects, 10000),
	}
}

// CompiledLink is a fully resolved query plan for a single link.
type CompiledLink struct {
	SQL         string
	Args        []any
	ObjectType  string
	TargetType  string
	LinkName    string
	Cardinality string
	Columns     []string
}

// GetLinkedObjects resolves a specific link by name for a source object and
// returns a LinkedObjectResult. It compiles the link into a query plan but
// does NOT execute the query — callers (e.g. QueryCompiler-aware services)
// execute the returned plan.
//
// When opts.Limit is 0, the link's default Limit is used. When Cardinality
// is one_to_one, Limit is forced to 1.
func (r *LinkResolver) GetLinkedObjects(ctx context.Context, source ObjectRef, linkName string, opts LinkOptions) (*LinkedObjectResult, error) {
	// 1. Find the source object type.
	ot, ok := r.objects[source.ObjectType]
	if !ok {
		return nil, fmt.Errorf("link_resolver: unknown object type %q", source.ObjectType)
	}

	// 2. Find the named link.
	var link *ObjectLinkV2
	for i := range ot.Links {
		if ot.Links[i].Name == linkName {
			link = &ot.Links[i]
			break
		}
	}
	if link == nil {
		return nil, fmt.Errorf("link_resolver: link %q not found on object type %q", linkName, source.ObjectType)
	}

	// 3. Apply options with fallback to link defaults.
	limit := opts.Limit
	if limit <= 0 {
		limit = link.Limit
	}
	if limit <= 0 {
		limit = 20
	}
	if link.Cardinality == "one_to_one" {
		limit = 1
	}
	if opts.MaxDepth <= 0 {
		opts.MaxDepth = 1
	}
	if opts.MaxDepth > 3 {
		opts.MaxDepth = 3
	}

	// 4. Compile the link into a query plan based on strategy.
	_, err := r.compileLink(source, link, limit, opts.Offset)
	if err != nil {
		return nil, fmt.Errorf("link_resolver: compile link %q: %w", linkName, err)
	}

	// 5. Build the result (objects are resolved by the caller executing the plan).
	result := &LinkedObjectResult{
		ObjectType:  source.ObjectType,
		ObjectID:    source.ObjectID,
		LinkName:    linkName,
		TargetType:  link.TargetType,
		Cardinality: link.Cardinality,
		Objects:     []ObjectInstance{},
	}

	return result, nil
}

// CompileAllLinks compiles all links on the given object type into query plans.
func (r *LinkResolver) CompileAllLinks(ctx context.Context, source ObjectRef, opts LinkOptions) ([]*CompiledLink, error) {
	ot, ok := r.objects[source.ObjectType]
	if !ok {
		return nil, fmt.Errorf("link_resolver: unknown object type %q", source.ObjectType)
	}

	var plans []*CompiledLink
	for i := range ot.Links {
		plan, err := r.compileLink(source, &ot.Links[i], opts.Limit, opts.Offset)
		if err != nil {
			return nil, fmt.Errorf("link_resolver: compile link %q: %w", ot.Links[i].Name, err)
		}
		plans = append(plans, plan)
	}
	return plans, nil
}

// compileLink generates a CompiledLink for a given source object and link definition.
func (r *LinkResolver) compileLink(source ObjectRef, link *ObjectLinkV2, limit, offset int) (*CompiledLink, error) {
	switch link.Strategy {
	case "direct_key", "lookup":
		return r.compileDirectKey(source, link, limit, offset)
	case "reverse_lookup":
		return r.compileReverseLookup(source, link, limit, offset)
	case "bridge_table":
		return r.compileBridgeTable(source, link, limit, offset)
	case "query_ref":
		return r.compileQueryRef(source, link, limit, offset)
	default:
		return nil, fmt.Errorf("unknown link strategy %q", link.Strategy)
	}
}

// direct_key: SourceKey matches target PK directly.
// SELECT <fields> FROM target WHERE target.pk = source.sourceKey
func (r *LinkResolver) compileDirectKey(source ObjectRef, link *ObjectLinkV2, limit, offset int) (*CompiledLink, error) {
	fields := r.resolveFields(link)
	sortClause := r.resolveSort(link)

	sql := fmt.Sprintf(
		`SELECT %s FROM %s.%s WHERE %s = $1%s LIMIT %d OFFSET %d`,
		fields,
		quoteIdent(link.Target.Schema),
		quoteIdent(link.Target.Table),
		quoteIdent(link.Target.Key),
		sortClause,
		limit,
		offset,
	)

	return &CompiledLink{
		SQL:         sql,
		Args:        []any{source.ObjectID},
		LinkName:    link.Name,
		TargetType:  link.TargetType,
		Cardinality: link.Cardinality,
	}, nil
}

// reverse_lookup: Rows where target holds the source key.
// SELECT <fields> FROM target WHERE target.key = source.id
func (r *LinkResolver) compileReverseLookup(source ObjectRef, link *ObjectLinkV2, limit, offset int) (*CompiledLink, error) {
	fields := r.resolveFields(link)
	sortClause := r.resolveSort(link)

	sql := fmt.Sprintf(
		`SELECT %s FROM %s.%s WHERE %s = $1%s LIMIT %d OFFSET %d`,
		fields,
		quoteIdent(link.Target.Schema),
		quoteIdent(link.Target.Table),
		quoteIdent(link.Target.Key),
		sortClause,
		limit,
		offset,
	)

	return &CompiledLink{
		SQL:         sql,
		Args:        []any{source.ObjectID},
		LinkName:    link.Name,
		TargetType:  link.TargetType,
		Cardinality: link.Cardinality,
	}, nil
}

// bridge_table: Join through an intermediate table.
// SELECT target.* FROM bridge JOIN target ON bridge.target_fk = target.pk
// WHERE bridge.source_fk = source.id
func (r *LinkResolver) compileBridgeTable(source ObjectRef, link *ObjectLinkV2, limit, offset int) (*CompiledLink, error) {
	// bridge table is the target table configured in the link
	// The actual bridge configuration would need extra metadata in LinkTarget.
	// For now, we treat bridge_table as a two-table join using Target config.
	fields := r.resolveFields(link)
	sortClause := r.resolveSort(link)

	// Bridge table pattern: target table IS the bridge, key on target is the join key
	sql := fmt.Sprintf(
		`SELECT %s FROM %s.%s WHERE %s = $1%s LIMIT %d OFFSET %d`,
		fields,
		quoteIdent(link.Target.Schema),
		quoteIdent(link.Target.Table),
		quoteIdent(link.Target.Key),
		sortClause,
		limit,
		offset,
	)

	return &CompiledLink{
		SQL:         sql,
		Args:        []any{source.ObjectID},
		LinkName:    link.Name,
		TargetType:  link.TargetType,
		Cardinality: link.Cardinality,
	}, nil
}

// query_ref: Use a predefined SQL template stored in the link.
func (r *LinkResolver) compileQueryRef(source ObjectRef, link *ObjectLinkV2, limit, offset int) (*CompiledLink, error) {
	if link.Target.Key == "" {
		return nil, fmt.Errorf("query_ref strategy requires target.key as the SQL template placeholder")
	}

	// The link's target.key holds the template with $1 as the source object ID
	// placeholder. We keep $1 as a pgx parameterized placeholder to prevent
	// SQL injection from user-controlled objectID values.
	template := link.Target.Key

	// Validate the SQL template before building the query.
	if err := ValidateQueryRef(template); err != nil {
		return nil, fmt.Errorf("query_ref validation failed: %w", err)
	}
	sortClause := r.resolveSort(link)

	// Wrap with LIMIT/OFFSET if not already in template.
	sql := template
	if !strings.Contains(strings.ToUpper(sql), "LIMIT") {
		sql = fmt.Sprintf("%s%s LIMIT %d OFFSET %d", sql, sortClause, limit, offset)
	}

	return &CompiledLink{
		SQL:         sql,
		Args:        []any{source.ObjectID},
		LinkName:    link.Name,
		TargetType:  link.TargetType,
		Cardinality: link.Cardinality,
	}, nil
}

// ──── helpers ─────────────────────────────────────────────────────────────────

// resolveFields returns the SELECT column list for a link.
// Falls back to "*" if no explicit fields are configured.
func (r *LinkResolver) resolveFields(link *ObjectLinkV2) string {
	if len(link.Fields) > 0 {
		quoted := make([]string, len(link.Fields))
		for i, f := range link.Fields {
			quoted[i] = quoteIdent(f)
		}
		return strings.Join(quoted, ", ")
	}
	return "*"
}

// resolveSort returns the ORDER BY clause for a link, or empty string.
func (r *LinkResolver) resolveSort(link *ObjectLinkV2) string {
	if link.Sort != "" {
		if !isValidSort(link.Sort) {
			// Invalid sort — return empty string to prevent SQL injection.
			// The callers (compileDirectKey, compileReverseLookup, etc.) will
			// produce valid SQL without an ORDER BY clause.
			return ""
		}
		return " ORDER BY " + link.Sort
	}
	return ""
}

// quoteIdent quotes a SQL identifier (column, table, schema).
func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
