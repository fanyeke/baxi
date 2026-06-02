package ontology

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"gopkg.in/yaml.v3"

	"baxi/internal/repository"
)

// ObjectRegistry provides access to AIP semantic object type definitions.
//
// It loads schema from the gov.object_schema table (via ObjectSchemaRepository)
// with a YAML file as fallback, and exposes typed accessor methods. All public
// methods are safe for concurrent use.
//
// V2 extension: objectsV2 holds ObjectTypeV2 definitions loaded from the v2
// YAML schema (config/aip_object_schema_v2.yml). When a v2 object exists for
// a type, it takes precedence over the v1 definition for query compilation,
// link resolution, context recipes, and action binding.
type ObjectRegistry struct {
	mu        sync.RWMutex
	objects   map[string]*ObjectType
	objectsV2 map[string]*ObjectTypeV2
}

// NewObjectRegistry creates an ObjectRegistry, loading object schema from the
// database first (via ObjectSchemaRepository), and falling back to a YAML file
// if the database is unavailable or returns no results.
//
// Either repo or yamlPath must be usable; if both fail an error is returned.
func NewObjectRegistry(ctx context.Context, repo repository.ObjectSchemaRepository, pool *pgxpool.Pool, yamlPath string) (*ObjectRegistry, error) {
	reg := &ObjectRegistry{
		objects:   make(map[string]*ObjectType),
		objectsV2: make(map[string]*ObjectTypeV2),
	}

	// Attempt DB load first.
	if repo != nil && pool != nil {
		loaded, err := loadFromDB(ctx, repo, pool)
		if err == nil && len(loaded) > 0 {
			reg.objects = loaded
			return reg, nil
		}
	}

	// Fallback to YAML.
	if yamlPath != "" {
		loaded, err := loadFromYAML(yamlPath)
		if err == nil && len(loaded) > 0 {
			reg.objects = loaded
			return reg, nil
		}
		return nil, fmt.Errorf("ontology: failed to load from YAML %s: %w", yamlPath, err)
	}

	return nil, errors.New("ontology: no schema source available (nil repo and empty yamlPath)")
}

// LoadV2Schema loads v2 object types from a v2 YAML file into the registry.
// These coexist with v1 objects - HasV2/GetObjectV2 methods check v2 first.
func (r *ObjectRegistry) LoadV2Schema(v2YamlPath string) error {
	v2objs, err := LoadObjectSchemaV2(v2YamlPath)
	if err != nil {
		return fmt.Errorf("load v2 schema: %w", err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for k, v := range v2objs {
		r.objectsV2[k] = v
	}
	return nil
}

// GetObjectTypeV2 returns the v2 object type definition for the given name.
func (r *ObjectRegistry) GetObjectTypeV2(name string) (*ObjectTypeV2, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ot, ok := r.objectsV2[name]
	if !ok {
		return nil, fmt.Errorf("ontology: unknown v2 object type %q", name)
	}
	return ot, nil
}

// HasV2 checks if a v2 object type definition exists for the given name.
func (r *ObjectRegistry) HasV2(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.objectsV2[name]
	return ok
}

// ListObjectTypesV2 returns all registered v2 object type names.
func (r *ObjectRegistry) ListObjectTypesV2() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.objectsV2))
	for n := range r.objectsV2 {
		names = append(names, n)
	}
	return names
}

// AllObjectsV2 returns the full v2 object map.
func (r *ObjectRegistry) AllObjectsV2() map[string]*ObjectTypeV2 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]*ObjectTypeV2, len(r.objectsV2))
	for k, v := range r.objectsV2 {
		result[k] = v
	}
	return result
}

// GetObjectType returns the object type definition for the given name.
// Returns an error if the object type is not registered.
func (r *ObjectRegistry) GetObjectType(name string) (*ObjectType, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ot, ok := r.objects[name]
	if !ok {
		return nil, fmt.Errorf("ontology: unknown object type %q", name)
	}
	return ot, nil
}

// ListObjectTypes returns all registered object type names in stable order.
func (r *ObjectRegistry) ListObjectTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	all := AllObjectTypes()
	// Only return types that are actually registered.
	result := make([]string, 0, len(all))
	for _, name := range all {
		if _, ok := r.objects[name]; ok {
			result = append(result, name)
		}
	}
	return result
}

// GetProperties returns the properties map for the given object type.
func (r *ObjectRegistry) GetProperties(objectType string) (map[string]ObjectProperty, error) {
	ot, err := r.GetObjectType(objectType)
	if err != nil {
		return nil, err
	}
	return ot.Properties, nil
}

// GetLinks returns the relationship links for the given object type.
func (r *ObjectRegistry) GetLinks(objectType string) ([]ObjectLink, error) {
	ot, err := r.GetObjectType(objectType)
	if err != nil {
		return nil, err
	}
	return ot.Links, nil
}

// GetAllowedActions returns the allowed action strings for the given object type.
func (r *ObjectRegistry) GetAllowedActions(objectType string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ot, ok := r.objects[objectType]
	if !ok {
		return nil
	}
	return ot.AllowedActions
}

// IsLLMReadable checks whether the named property on the given object type is
// marked as LLM-readable.
func (r *ObjectRegistry) IsLLMReadable(objectType, property string) bool {
	ot, err := r.GetObjectType(objectType)
	if err != nil {
		return false
	}
	prop, ok := ot.Properties[property]
	if !ok {
		return false
	}
	return prop.LLMReadable
}

// GetSourceDataset returns the first source table for the given object type.
// Returns an empty string if the object type is unknown.
func (r *ObjectRegistry) GetSourceDataset(objectType string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ot, ok := r.objects[objectType]
	if !ok {
		return ""
	}
	if len(ot.SourceTables) == 0 {
		return ""
	}
	return ot.SourceTables[0]
}

// ──── DB loader ──────────────────────────────────────────────────────────────

func loadFromDB(ctx context.Context, repo repository.ObjectSchemaRepository, pool *pgxpool.Pool) (map[string]*ObjectType, error) {
	rows, err := repo.GetAll(ctx, pool)
	if err != nil {
		return nil, fmt.Errorf("ontology: query gov.object_schema: %w", err)
	}
	if len(rows) == 0 {
		return nil, errors.New("ontology: gov.object_schema returned 0 rows")
	}

	objects := make(map[string]*ObjectType, len(rows))
	for _, row := range rows {
		ot := &ObjectType{}
		if err := json.Unmarshal(row.SchemaJSONB, ot); err != nil {
			return nil, fmt.Errorf("ontology: unmarshal schema for %q: %w", row.ObjectType, err)
		}
		objects[row.ObjectType] = ot
	}
	return objects, nil
}

// ──── YAML loader ────────────────────────────────────────────────────────────

func loadFromYAML(yamlPath string) (map[string]*ObjectType, error) {
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("read YAML: %w", err)
	}

	var cfg objectSchemaConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}

	objects := make(map[string]*ObjectType, len(cfg.Objects))
	for _, raw := range cfg.Objects {
		ot, err := convertRawObject(raw)
		if err != nil {
			return nil, fmt.Errorf("convert object %q: %w", raw.ObjectTypeID, err)
		}
		objects[raw.ObjectTypeID] = ot
	}
	return objects, nil
}

func convertRawObject(raw rawObjectType) (*ObjectType, error) {
	props := make(map[string]ObjectProperty, len(raw.Properties))
	var primaryKey string

	for name, rp := range raw.Properties {
		isPK := false
		if rp.IsPK != nil && *rp.IsPK {
			isPK = true
		}

		// Derive sensitivity from PK status.
		sensitivity := defaultSensitivity(isPK)

		// LLM-readable by default for non-PK fields.
		llmReadable := !isPK

		prop := ObjectProperty{
			Name:        name,
			Type:        rp.Type,
			SourceField: rp.Source,
			Sensitivity: sensitivity,
			Aggregation: rp.Agg,
			LLMReadable: llmReadable,
			IsPK:        isPK,
		}
		props[name] = prop

		if isPK {
			primaryKey = name
		}
	}

	// Convert relationships to links.
	links := make([]ObjectLink, 0, len(raw.Relationships))
	for linkName, rel := range raw.Relationships {
		links = append(links, ObjectLink{
			Name:       linkName,
			TargetType: rel.To,
			Via:        rel.Grain,
		})
	}

	// Default allowed actions: every object can be read.
	allowedActions := []string{"read"}
	if len(raw.AllowedActions) > 0 {
		allowedActions = raw.AllowedActions
	}

	// LLM access: metric_alert is read-write; everything else read-only.
	llmAccess := defaultLLMAccess()
	if raw.ObjectTypeID == TypeMetricAlert {
		llmAccess = readWriteLLMAccess()
	}

	// Source tables from YAML.
	sourceTables := raw.SourceTables
	if sourceTables == nil {
		sourceTables = []string{}
	}

	// Alert fields from YAML.
	alertFields := raw.AlertFields
	if alertFields == nil {
		alertFields = []string{}
	}

	return &ObjectType{
		Name:           raw.ObjectTypeID,
		DisplayName:    raw.DisplayName,
		Grain:          raw.Grain,
		SourceTables:   sourceTables,
		PrimaryKey:     primaryKey,
		Properties:     props,
		Links:          links,
		AllowedActions: allowedActions,
		LLMAccess:      llmAccess,
		AlertFields:    alertFields,
	}, nil
}
