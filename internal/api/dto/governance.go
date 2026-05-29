package dto

// GovernanceStatusResponse is the top-level response for GET /api/v1/governance/status.
// When governance data is available, GovernanceLayer is "active" and Configs contains
// the loaded configuration files. When no data is available, GovernanceLayer is "unknown"
// and Configs is empty.
type GovernanceStatusResponse struct {
	GovernanceLayer   string            `json:"governance_layer"`
	Configs           map[string]string `json:"configs"`
	ObjectSchemaCount int               `json:"object_schema_count"`
}

// ──── Catalog ─────────────────────────────────────────────────────────────────

type CatalogResponse struct {
	Objects  []CatalogObject  `json:"objects"`
	Datasets []CatalogDataset `json:"datasets"`
}

type CatalogObject struct {
	ObjectType      string `json:"object_type"`
	SourceDataset   string `json:"source_dataset"`
	PrimaryKey      string `json:"primary_key"`
	PropertiesCount int    `json:"properties_count"`
	LinksCount      int    `json:"links_count"`
}

type CatalogDataset struct {
	Dataset string `json:"dataset"`
	Schema  string `json:"schema"`
	Table   string `json:"table"`
}

// ──── Classification ──────────────────────────────────────────────────────────

type ClassificationResponse struct {
	Levels    []string                 `json:"levels"`
	Resources []ClassificationResource `json:"resources"`
}

type ClassificationResource struct {
	Resource       string `json:"resource"`
	Classification string `json:"classification"`
}

type FieldMarkingResponse struct {
	Markings []FieldMarking `json:"markings"`
}

type FieldMarking struct {
	ObjectType     string `json:"object_type"`
	Field          string `json:"field"`
	Classification string `json:"classification"`
	PII            bool   `json:"pii"`
	LLMAllowed     bool   `json:"llm_allowed"`
}

// ──── Lineage ─────────────────────────────────────────────────────────────────

type LineageResponse struct {
	Resource   string   `json:"resource"`
	Upstream   []string `json:"upstream"`
	Downstream []string `json:"downstream"`
}

// ──── Checkpoints ─────────────────────────────────────────────────────────────

type CheckpointsResponse struct {
	Checkpoints []CheckpointRule `json:"checkpoints"`
}

type CheckpointRule struct {
	Action              string `json:"action"`
	RequiresReason      bool   `json:"requires_reason"`
	RequiresHumanReview bool   `json:"requires_human_review"`
}

// ──── Health Checks ───────────────────────────────────────────────────────────

type HealthChecksResponse struct {
	Status string            `json:"status"`
	Checks []HealthCheckItem `json:"checks"`
}

type HealthCheckItem struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// ──── Access Decision ─────────────────────────────────────────────────────────

type AccessDecision string

const (
	AccessAllowed     AccessDecision = "ALLOW"
	AccessDenied      AccessDecision = "DENY"
	AccessConditional AccessDecision = "CONDITIONAL"
)
