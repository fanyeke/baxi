package ontology

import "github.com/jackc/pgx/v5"

// ──── V2 Schema types ─────────────────────────────────────────────────────────
// These types extend the v1 ObjectType with richer semantics for the executable
// semantic layer: source config, filtered/searchable properties, multi-cardinality
// relationships, metric contracts, object-level action binding, and governance.

// ObjectTypeV2 is the extended semantic object definition for Ontology v2.
// It supersedes ObjectType with richer metadata and replaces flat source_tables
// with structured Source, extends Properties with filter/search/meta fields,
// adds explicit Metrics references, multi-cardinality Links, object-level
// AllowedActions, Maturity classification, and a Governance policy.
type ObjectTypeV2 struct {
	Name           string
	DisplayName    string
	Description    string
	Grain          string
	Maturity       string // stable, virtual, or planned — controls Agent decision path eligibility
	Source         ObjectSource
	Properties     map[string]ObjectPropertyV2
	Metrics        []string
	Links          []ObjectLinkV2
	AllowedActions []string
	LLMAccess      LLMAccessPolicy
	AlertFields    []string
	Governance     ObjectGovernancePolicy
}

// ObjectSource defines the physical table backing an object type.
type ObjectSource struct {
	Schema     string
	Table      string
	PrimaryKey string
}

// ObjectPropertyV2 extends ObjectProperty with search/filter/metadata flags,
// optional SQL Expression, optional MetricRef for pre-aggregated metrics,
// explicit sensitivity classification, and availability marking.
type ObjectPropertyV2 struct {
	Name        string // property name
	Type        string // string, int, float, bool, timestamp
	SourceField string // original column name in the source table
	Expression  string // SQL expression (e.g. "AVG(order_level.review_score)")
	MetricRef   string // reference to a metric in metric_definitions.yml
	Sensitivity string // L0, L1, L2, L3
	Aggregation string // sum, count, count_distinct, avg, min, max
	LLMReadable bool   // whether the LLM may read this in context
	Searchable  bool   // whether this field can be searched via ILIKE
	Filterable  bool   // whether this field can be used as a filter
	IsPK        bool   // true if this property is the primary key
	Availability string // real, virtual, or planned — controls query compilation and LLM context
}

// ObjectLinkV2 extends ObjectLink with cardinality support, multiple resolution
// strategies, explicit source/target config, and optional limit/sort/fields.
type ObjectLinkV2 struct {
	Name        string     // link name (e.g. "recent_orders")
	DisplayName string     // human-readable label
	TargetType  string     // target object type name
	Cardinality string     // one_to_one, one_to_many, many_to_many
	Strategy    string     // direct_key, reverse_lookup, bridge_table, query_ref
	SourceKey   string     // field on the source object
	Target      LinkTarget // target table configuration
	Limit       int        // default result limit
	Sort        string     // default sort expression
	Fields      []string   // fields to return
}

// LinkTarget defines the physical table and key mapping for a relationship.
type LinkTarget struct {
	Schema        string
	Table         string
	Key           string // join key on the target table
	ObjectIDField string // field that holds the target object's ID
}

// ObjectGovernancePolicy defines governance constraints for an object type.
type ObjectGovernancePolicy struct {
	DefaultRole string // default access role
	RedactPII   bool   // whether to redact PII fields in LLM context
}

// ──── YAML parsing types ─────────────────────────────────────────────────────

// objectSchemaConfigV2 is the root structure of the v2 ontology YAML file.
type objectSchemaConfigV2 struct {
	Version string                  `yaml:"version"`
	Objects map[string]*rawObjectV2 `yaml:"objects"`
}

// rawObjectV2 mirrors the v2 YAML structure for a single object type.
type rawObjectV2 struct {
	DisplayName   string                   `yaml:"display_name"`
	Description   string                   `yaml:"description"`
	Grain         string                   `yaml:"grain"`
	Maturity      string                   `yaml:"maturity,omitempty"`
	Source        rawSourceV2              `yaml:"source"`
	Properties    map[string]rawPropertyV2 `yaml:"properties"`
	Metrics       []string                 `yaml:"metrics,omitempty"`
	Relationships map[string]rawLinkV2     `yaml:"relationships,omitempty"`
	Actions       []string                 `yaml:"actions,omitempty"`
	AlertFields   []string                 `yaml:"alert_fields,omitempty"`
	Governance    rawGovernanceV2          `yaml:"governance,omitempty"`
}

type rawSourceV2 struct {
	Schema     string `yaml:"schema"`
	Table      string `yaml:"table"`
	PrimaryKey string `yaml:"primary_key"`
}

type rawPropertyV2 struct {
	Type         string `yaml:"type"`
	Source       string `yaml:"source,omitempty"`
	Expression   string `yaml:"expression,omitempty"`
	MetricRef    string `yaml:"metric_ref,omitempty"`
	Sensitivity  string `yaml:"sensitivity,omitempty"`
	Agg          string `yaml:"agg,omitempty"`
	LLMReadable  *bool  `yaml:"llm_readable,omitempty"`
	Searchable   *bool  `yaml:"searchable,omitempty"`
	Filterable   *bool  `yaml:"filterable,omitempty"`
	IsPK         *bool  `yaml:"is_pk,omitempty"`
	Availability string `yaml:"availability,omitempty"`
}

type rawLinkV2 struct {
	DisplayName string        `yaml:"display_name"`
	To          string        `yaml:"to"`
	Cardinality string        `yaml:"cardinality"`
	Strategy    string        `yaml:"strategy"`
	SourceKey   string        `yaml:"source_key"`
	Target      rawLinkTarget `yaml:"target"`
	Limit       int           `yaml:"limit,omitempty"`
	Sort        string        `yaml:"sort,omitempty"`
	Fields      []string      `yaml:"fields,omitempty"`
}

type rawLinkTarget struct {
	Schema        string `yaml:"schema"`
	Table         string `yaml:"table"`
	Key           string `yaml:"key"`
	ObjectIDField string `yaml:"object_id_field"`
}

type rawGovernanceV2 struct {
	DefaultRole string `yaml:"default_role"`
	RedactPII   *bool  `yaml:"redact_pii,omitempty"`
}

// CompiledQuery holds a fully resolved query plan compiled from Ontology v2 schema.
// It contains the parameterized SQL, column metadata, object identity info, and
// lists of metric references and virtual properties for post-query resolution.
// Args uses pgx.NamedArgs so the caller can pass them directly to pool.Query/QueryRow.
type CompiledQuery struct {
	SQL              string        // data query (with ORDER BY for search, or LIMIT 1 for get)
	CountSQL         string        // COUNT(*) query (without ORDER BY/LIMIT), used by search
	Args             pgx.NamedArgs // named parameters
	Columns          []string      // column/property names in select order
	MetricRefs       []string      // metric_ref property names (resolved separately, not in SQL)
	VirtualProperties []string     // virtual property names (resolved by metric_ref/expression/link)
	ObjectType       string
	PrimaryKey       string
	Schema           string
	Table            string
}
