package service

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"baxi/internal/api/dto"
	"baxi/internal/governance"
	"baxi/internal/repository"
)

// GovernanceService handles business logic for governance operations.
type GovernanceService struct {
	repo          *repository.GovernanceRepository
	pool          *pgxpool.Pool
	classification *governance.ClassificationService
	lineage       *governance.LineageService
	accessPolicy  *governance.AccessPolicyService
	checkpoint    *governance.CheckpointService
}

// NewGovernanceService creates a new GovernanceService with all governance domain services.
func NewGovernanceService(repo *repository.GovernanceRepository, pool *pgxpool.Pool) *GovernanceService {
	return &GovernanceService{
		repo:          repo,
		pool:          pool,
		classification: governance.NewClassificationService(pool, repo),
		lineage:       governance.NewLineageService(pool, repo),
		accessPolicy:  governance.NewAccessPolicyService(pool, repo),
		checkpoint:    governance.NewCheckpointService(pool, repo),
	}
}

// GetStatus aggregates governance status from the gov.* tables.
// If config_snapshot contains data, returns the rich format with governance_layer set
// to "active" and configs populated. If no data exists, returns "unknown" with empty configs.
// Enhanced to include object_schema_count.
func (s *GovernanceService) GetStatus(ctx context.Context) (*dto.GovernanceStatusResponse, error) {
	configs, err := s.repo.GetConfigSnapshots(ctx, s.pool)
	if err != nil {
		return nil, fmt.Errorf("get governance status: %w", err)
	}

	configMap := make(map[string]string, len(configs))
	for _, c := range configs {
		configMap[c.ConfigKey] = c.Status
	}

	layer := "active"
	if len(configs) == 0 {
		layer = "unknown"
	}

	schemaCount := s.repo.CountObjectSchemas(ctx, s.pool)

	return &dto.GovernanceStatusResponse{
		GovernanceLayer:   layer,
		Configs:           configMap,
		ObjectSchemaCount: schemaCount,
	}, nil
}

// GetClassification returns the classification level for a given field path.
// Classification levels: pii→L3, sensitive→L3, internal→L2, public_internal→L1, derived_sensitive→L2.
func (s *GovernanceService) GetClassification(ctx context.Context, fieldPath string) (*dto.ClassificationResponse, error) {
	classifications, err := s.classification.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("get classifications: %w", err)
	}

	levelSet := make(map[string]bool)
	var resources []dto.ClassificationResource

	for _, c := range classifications {
		level := governance.ResolveLevel(c.ClassificationLevel)
		levelSet[level] = true

		if fieldPath == "" || c.FieldPath == fieldPath {
			resources = append(resources, dto.ClassificationResource{
				Resource:       c.FieldPath,
				Classification: level,
			})
		}
	}

	// If fieldPath specified and no match found, return default
	if fieldPath != "" && len(resources) == 0 {
		resources = append(resources, dto.ClassificationResource{
			Resource:       fieldPath,
			Classification: governance.ResolveLevel("internal"),
		})
	}

	var levels []string
	for l := range levelSet {
		levels = append(levels, l)
	}
	if levels == nil {
		levels = []string{}
	}

	return &dto.ClassificationResponse{
		Levels:    levels,
		Resources: resources,
	}, nil
}

// GetFieldMarking returns classification details for a specific object type and property.
func (s *GovernanceService) GetFieldMarking(ctx context.Context, objectType, property string) (*dto.FieldMarkingResponse, error) {
	classifications, err := s.classification.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("get field markings: %w", err)
	}

	var markings []dto.FieldMarking
	prefix := objectType + "." + property

	for _, c := range classifications {
		if objectType != "" && property != "" {
			if c.FieldPath != prefix {
				continue
			}
		}

		level := governance.ResolveLevel(c.ClassificationLevel)
		isPII := c.ClassificationLevel == "pii"
		llmAllowed := level != "L3"

		// Parse object_type and field from field_path (format: "object_type.field")
		objType, field := parseFieldPath(c.FieldPath)

		markings = append(markings, dto.FieldMarking{
			ObjectType:     objType,
			Field:          field,
			Classification: level,
			PII:            isPII,
			LLMAllowed:     llmAllowed,
		})
	}

	if markings == nil {
		markings = []dto.FieldMarking{}
	}

	return &dto.FieldMarkingResponse{
		Markings: markings,
	}, nil
}

// CheckAccess evaluates whether a user role can perform an action on an object type.
func (s *GovernanceService) CheckAccess(ctx context.Context, userRole, objectType, action string) dto.AccessDecision {
	return s.accessPolicy.CheckAccess(ctx, userRole, objectType, action)
}

// GetLineage returns the upstream and downstream lineage for a given resource.
func (s *GovernanceService) GetLineage(ctx context.Context, resource string) (*dto.LineageResponse, error) {
	result, err := s.lineage.GetLineage(ctx, resource)
	if err != nil {
		return nil, fmt.Errorf("get lineage: %w", err)
	}

	return &dto.LineageResponse{
		Resource:   result.Resource,
		Upstream:   result.Upstream,
		Downstream: result.Downstream,
	}, nil
}

// RequiresCheckpoint checks if an action requires a checkpoint before execution.
func (s *GovernanceService) RequiresCheckpoint(ctx context.Context, action string) bool {
	return s.checkpoint.RequiresCheckpoint(ctx, action)
}

// GetCheckpoints returns all checkpoint rules.
func (s *GovernanceService) GetCheckpoints(ctx context.Context) (*dto.CheckpointsResponse, error) {
	rules := s.checkpoint.GetRules(ctx)

	checkpointRules := make([]dto.CheckpointRule, len(rules))
	for i, r := range rules {
		checkpointRules[i] = dto.CheckpointRule{
			Action:              r.Action,
			RequiresReason:      r.RequiresReason,
			RequiresHumanReview: r.RequiresHumanReview,
		}
	}

	return &dto.CheckpointsResponse{
		Checkpoints: checkpointRules,
	}, nil
}

// GetHealthChecks returns the status of all governance health checks.
func (s *GovernanceService) GetHealthChecks(ctx context.Context) (*dto.HealthChecksResponse, error) {
	// Run health checks against governance tables
	checks := []dto.HealthCheckItem{
		{
			Name:   "config_snapshot",
			Status: healthStatus(ctx, s.pool, "gov", "config_snapshot"),
		},
		{
			Name:   "data_classification",
			Status: healthStatus(ctx, s.pool, "gov", "data_classification"),
		},
		{
			Name:   "data_lineage",
			Status: healthStatus(ctx, s.pool, "gov", "data_lineage"),
		},
		{
			Name:   "access_policy",
			Status: healthStatus(ctx, s.pool, "gov", "access_policy"),
		},
		{
			Name:   "object_schema",
			Status: healthStatus(ctx, s.pool, "gov", "object_schema"),
		},
	}

	overall := "healthy"
	for _, c := range checks {
		if c.Status == "unhealthy" {
			overall = "degraded"
			break
		}
	}

	return &dto.HealthChecksResponse{
		Status: overall,
		Checks: checks,
	}, nil
}

// GetCatalog returns the governance catalog of object schemas and datasets.
func (s *GovernanceService) GetCatalog(ctx context.Context) (*dto.CatalogResponse, error) {
	schemas, err := s.repo.GetObjectSchemas(ctx, s.pool)
	if err != nil {
		return nil, fmt.Errorf("get catalog: %w", err)
	}

	var objects []dto.CatalogObject
	datasetMap := make(map[string]bool)

	for _, sch := range schemas {
		obj := dto.CatalogObject{
			ObjectType:      sch.ObjectType,
			SourceDataset:   inferSourceDataset(sch.ObjectType),
			PrimaryKey:      inferPrimaryKey(sch.ObjectType),
			PropertiesCount: 0, // Would require parsing schema_jsonb
			LinksCount:      0,
		}
		objects = append(objects, obj)
		datasetMap[inferSourceDataset(sch.ObjectType)] = true
	}

	if objects == nil {
		objects = []dto.CatalogObject{}
	}

	// Build dataset list
	var datasets []dto.CatalogDataset
	for ds := range datasetMap {
		schema, table := splitDataset(ds)
		datasets = append(datasets, dto.CatalogDataset{
			Dataset: ds,
			Schema:  schema,
			Table:   table,
		})
	}
	if datasets == nil {
		datasets = []dto.CatalogDataset{}
	}

	return &dto.CatalogResponse{
		Objects:  objects,
		Datasets: datasets,
	}, nil
}

// ──── helpers ─────────────────────────────────────────────────────────────────

// healthStatus checks if a table exists and has rows.
func healthStatus(ctx context.Context, pool *pgxpool.Pool, schema, table string) string {
	if pool == nil {
		return "unknown"
	}
	var count int
	err := pool.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s.%s", schema, table)).Scan(&count)
	if err != nil {
		return "unhealthy"
	}
	if count == 0 {
		return "unknown"
	}
	return "healthy"
}

// parseFieldPath splits "object_type.field_name" into its components.
func parseFieldPath(fieldPath string) (string, string) {
	for i := len(fieldPath) - 1; i >= 0; i-- {
		if fieldPath[i] == '.' {
			return fieldPath[:i], fieldPath[i+1:]
		}
	}
	return fieldPath, "*"
}

// inferSourceDataset returns a heuristically determined dataset name for an object type.
func inferSourceDataset(objectType string) string {
	switch {
	case objectType == "customer":
		return "olist_customers"
	case objectType == "order":
		return "olist_orders"
	case objectType == "product":
		return "olist_products"
	case objectType == "seller":
		return "olist_sellers"
	case objectType == "geolocation":
		return "olist_geolocation"
	default:
		return objectType
	}
}

// inferPrimaryKey returns the heuristically determined primary key for an object type.
func inferPrimaryKey(objectType string) string {
	switch objectType {
	case "customer":
		return "customer_id"
	case "order":
		return "order_id"
	case "product":
		return "product_id"
	case "seller":
		return "seller_id"
	case "geolocation":
		return "zip_code_prefix"
	default:
		return "id"
	}
}

// splitDataset splits a dataset name into schema and table.
func splitDataset(dataset string) (string, string) {
	return "public", dataset
}
