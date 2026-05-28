package model

// GovernanceStatusResponse is the response for governance status.
type GovernanceStatusResponse struct {
	GovernanceLayer   string
	Configs           map[string]string
	ObjectSchemaCount int
}

// ClassificationResponse holds classification data.
type ClassificationResponse struct {
	Levels    []string
	Resources []ClassificationResource
}

// ClassificationResource represents a classified resource.
type ClassificationResource struct {
	Resource       string
	Classification string
}

// FieldMarkingResponse holds field marking data.
type FieldMarkingResponse struct {
	Markings []FieldMarking
}

// FieldMarking represents a field marking.
type FieldMarking struct {
	ObjectType     string
	Field          string
	Classification string
	PII            bool
	LLMAllowed     bool
}

// AccessDecision represents an access decision.
type AccessDecision string

const (
	AccessAllowed     AccessDecision = "ALLOW"
	AccessDenied      AccessDecision = "DENY"
	AccessConditional AccessDecision = "CONDITIONAL"
)

// CatalogResponse is the response for the governance catalog.
type CatalogResponse struct {
	Objects  []CatalogObject
	Datasets []CatalogDataset
}

// CatalogObject represents an object in the catalog.
type CatalogObject struct {
	ObjectType      string
	SourceDataset   string
	PrimaryKey      string
	PropertiesCount int
	LinksCount      int
}

// CatalogDataset represents a dataset in the catalog.
type CatalogDataset struct {
	Dataset string
	Schema  string
	Table   string
}

// LineageResponse holds lineage data.
type LineageResponse struct {
	Resource   string
	Upstream   []string
	Downstream []string
}

// CheckpointsResponse holds checkpoint rules.
type CheckpointsResponse struct {
	Checkpoints []CheckpointRule
}

// CheckpointRule represents a checkpoint rule.
type CheckpointRule struct {
	Action              string
	RequiresReason      bool
	RequiresHumanReview bool
}

// HealthChecksResponse holds health check results.
type HealthChecksResponse struct {
	Status string
	Checks []HealthCheckItem
}

// HealthCheckItem represents a single health check.
type HealthCheckItem struct {
	Name   string
	Status string
}
