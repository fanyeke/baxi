// Package ontology provides AIP semantic object type definitions, registry,
// and validation for the object schema layer.
//
// The ontology package serves as the schema-aware layer between governance
// configuration (gov.object_schema table or YAML config) and the query/action
// services. It owns no query logic — it only describes *what objects are*.
package ontology

// ObjectType defines an AIP semantic object — the central abstraction for how
// the system understands business entities (customers, orders, sellers, etc.).
//
// Every ObjectType has a grain (the unique entity identifier), source tables
// that back it, typed properties, relationships to other object types,
// allowed actions, LLM access policy, and alert field references.
type ObjectType struct {
	Name           string                  `json:"name"`
	DisplayName    string                  `json:"display_name"`
	Grain          string                  `json:"grain"`
	SourceTables   []string                `json:"source_tables"`
	PrimaryKey     string                  `json:"primary_key"`
	Properties     map[string]ObjectProperty `json:"properties"`
	Links          []ObjectLink            `json:"links"`
	AllowedActions []string                `json:"allowed_actions"`
	LLMAccess      LLMAccessPolicy         `json:"llm_access"`
	AlertFields    []string                `json:"alert_fields"`
}

// ObjectProperty describes a single property of an object type.
//
// Fields:
//   - Type: the Go-level data type (string, int, float, datetime)
//   - SourceField: the original column or expression backing this property
//   - Sensitivity: classification level (L0–L4)
//   - Aggregation: how the property is aggregated (sum, count, nunique, min, max, none)
//   - LLMReadable: whether the LLM may read this property in context
//   - IsPK: true if this property is the primary key
type ObjectProperty struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	SourceField string `json:"source_field,omitempty"`
	Sensitivity string `json:"sensitivity,omitempty"`
	Aggregation string `json:"aggregation,omitempty"`
	LLMReadable bool   `json:"llm_readable"`
	IsPK        bool   `json:"is_pk"`
}

// ObjectLink defines a named relationship to another object type.
//
// Fields:
//   - TargetType: the name of the related ObjectType
//   - Via: the join key or relationship expression linking the objects
type ObjectLink struct {
	Name       string `json:"name"`
	TargetType string `json:"target_type"`
	Via        string `json:"via"`
}

// LLMAccessPolicy defines LLM access constraints for an object type.
//
// The default for most business entities is ReadOnly=true (the LLM can read
// but not mutate). Some types (e.g. metric_alert) may allow write access.
type LLMAccessPolicy struct {
	CanRead  bool `json:"can_read"`
	CanWrite bool `json:"can_write"`
	ReadOnly bool `json:"read_only"`
}

// ──── YAML-parsing types (internal) ──────────────────────────────────────────

// objectSchemaConfig is the root structure of the AIP object schema YAML file.
type objectSchemaConfig struct {
	Objects []rawObjectType `yaml:"objects"`
}

// rawObjectType mirrors the YAML structure for a single object type definition.
type rawObjectType struct {
	ObjectTypeID  string                         `yaml:"object_type_id"`
	DisplayName   string                         `yaml:"display_name"`
	SourceTables  []string                       `yaml:"source_tables"`
	Grain         string                         `yaml:"grain"`
	Properties    map[string]rawObjectProperty   `yaml:"properties"`
	Relationships map[string]rawObjectRelationship `yaml:"relationships,omitempty"`
	AlertFields   []string                       `yaml:"alert_fields"`
}

// rawObjectProperty mirrors the YAML property entry for a single field.
type rawObjectProperty struct {
	Type   string `yaml:"type"`
	Source string `yaml:"source,omitempty"`
	Agg    string `yaml:"agg,omitempty"`
	IsPK   *bool  `yaml:"is_pk,omitempty"`
}

// rawObjectRelationship mirrors the YAML relationship entry.
type rawObjectRelationship struct {
	To    string `yaml:"to"`
	Grain string `yaml:"grain"`
}

// ──── Constructor ────────────────────────────────────────────────────────────

// NewObjectType constructs an ObjectType from explicit fields.
// This is primarily useful for tests or for constructing object types from
// non-YAML sources (e.g. deserialized JSON from gov.object_schema).
func NewObjectType(name, displayName, grain, primaryKey string, properties map[string]ObjectProperty, links []ObjectLink, allowedActions []string, llmAccess LLMAccessPolicy, alertFields []string) *ObjectType {
	if properties == nil {
		properties = make(map[string]ObjectProperty)
	}
	if links == nil {
		links = []ObjectLink{}
	}
	if allowedActions == nil {
		allowedActions = []string{}
	}
	if alertFields == nil {
		alertFields = []string{}
	}
	return &ObjectType{
		Name:           name,
		DisplayName:    displayName,
		Grain:          grain,
		PrimaryKey:     primaryKey,
		Properties:     properties,
		Links:          links,
		AllowedActions: allowedActions,
		LLMAccess:      llmAccess,
		AlertFields:    alertFields,
	}
}

// ──── Conversion helpers ─────────────────────────────────────────────────────

// defaultLLMAccess returns a sensible LLMAccessPolicy for a business entity.
// Most objects start as read-only for the LLM.
func defaultLLMAccess() LLMAccessPolicy {
	return LLMAccessPolicy{
		CanRead:  true,
		CanWrite: false,
		ReadOnly: true,
	}
}

// readWriteLLMAccess returns a policy that allows LLM write access.
// Used for objects like metric_alert where the LLM should update state.
func readWriteLLMAccess() LLMAccessPolicy {
	return LLMAccessPolicy{
		CanRead:  true,
		CanWrite: true,
		ReadOnly: false,
	}
}

// defaultSensitivity returns the default sensitivity level for a property.
// PK fields get L2; most other fields get L0.
func defaultSensitivity(isPK bool) string {
	if isPK {
		return "L2"
	}
	return "L0"
}
