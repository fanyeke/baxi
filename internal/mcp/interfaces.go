package mcp

import (
	"context"

	"baxi/internal/action"
	"baxi/internal/decision"
	"baxi/internal/llm"
	"baxi/internal/model"
	"baxi/internal/review"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DecisionService defines the interface for decision case operations.
type DecisionService interface {
	CreateCaseFromAlert(ctx context.Context, alertID, createdBy string) (*decision.DecisionCase, error)
	GetCase(ctx context.Context, caseID string) (*decision.DecisionCase, error)
	ListCases(ctx context.Context, filter decision.CaseFilter) (*decision.CaseList, error)
	Decide(ctx context.Context, caseID string) ([]action.ActionProposal, error)
	ResolveCase(ctx context.Context, caseID, resolution string, comment string) error
}

// DecisionEngine defines the interface for generating decisions.
type DecisionEngine interface {
	GenerateDecision(ctx context.Context, caseID string, context *decision.DecisionContext) (*llm.DecisionOutput, error)
}

// ContextBuilder defines the interface for building decision contexts.
type ContextBuilder interface {
	BuildDecisionContext(ctx context.Context, caseID string) (*decision.DecisionContext, error)
}

// ProposalService defines the interface for managing action proposals.
type ProposalService interface {
	ListProposals(ctx context.Context, caseID string) ([]action.ActionProposal, error)
}

// AlertService defines the interface for alert operations.
type AlertService interface {
	ListAlerts(ctx context.Context, filters model.AlertFilters, sort string, limit, offset int) (*model.AlertListResponse, error)
}

// GovernanceService defines the interface for governance operations.
type GovernanceService interface {
	CheckAccess(ctx context.Context, role, objectType, action string) (*model.AccessDecision, error)
	GetClassification(ctx context.Context, fieldPath string) (*model.ClassificationResponse, error)
}

// ReviewService defines the interface for review operations.
type ReviewService interface {
	ApproveProposal(ctx context.Context, proposalID, reviewerID, feedback string) (*review.ReviewRecord, error)
	RejectProposal(ctx context.Context, proposalID, reviewerID, feedback string) (*review.ReviewRecord, error)
	CancelProposal(ctx context.Context, proposalID, reviewerID, reason string) error
	GetProposalByID(ctx context.Context, proposalID string) (*action.ActionProposal, error)
	ListReviewRecords(ctx context.Context, proposalID string, limit, offset int) ([]review.ReviewRecord, int, error)
}

// ExecuteService defines the interface for executing action proposals.
type ExecuteService interface {
	ExecuteProposal(ctx context.Context, pool *pgxpool.Pool, proposalID string, actorID string, opts ...action.ExecuteOption) (*action.ExecutionResult, error)
}

// PipelineRunner defines the interface for running pipelines.
type PipelineRunner interface {
	Run(ctx context.Context, config string) (string, error)
}

// SystemStatusService defines the interface for getting system status.
type SystemStatusService interface {
	GetStatus(ctx context.Context) (*model.SystemStatus, error)
}

// ObjectSearchService defines the interface for searching objects.
type ObjectSearchService interface {
	SearchObjects(ctx context.Context, objectType, query string, limit, offset int) (*model.SearchResult, error)
}

// OutboxService defines the interface for outbox event operations.
type OutboxService interface {
	ListOutboxEvents(ctx context.Context, status string, limit, offset int) ([]model.OutboxEvent, int, error)
}

// ──── Ontology types ──────────────────────────────────────────────────────────

// OntologyDescriptor is the top-level response for describe_ontology.
type OntologyDescriptor struct {
	ObjectTypes []ObjectTypeDescriptor `json:"object_types"`
}

// ObjectTypeDescriptor describes a single AIP object type.
type ObjectTypeDescriptor struct {
	Name           string               `json:"name"`
	DisplayName    string               `json:"display_name"`
	Grain          string               `json:"grain"`
	Source         *SourceDescriptor    `json:"source,omitempty"`
	Properties     []PropertyDescriptor `json:"properties"`
	Metrics        []string             `json:"metrics,omitempty"`
	Links          []LinkDescriptor     `json:"links"`
	AllowedActions []string             `json:"allowed_actions"`
	LLMAccess      LLMAccessDescriptor  `json:"llm_access"`
	Governance     *GovDescriptor       `json:"governance,omitempty"`
}

// SourceDescriptor describes the physical table backing an object type.
type SourceDescriptor struct {
	Schema     string `json:"schema"`
	Table      string `json:"table"`
	PrimaryKey string `json:"primary_key"`
}

// GovDescriptor describes the governance policy for an object type.
type GovDescriptor struct {
	DefaultRole string `json:"default_role"`
	RedactPII   bool   `json:"redact_pii"`
}

// PropertyDescriptor describes a single property of an object type.
type PropertyDescriptor struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Sensitivity string `json:"sensitivity,omitempty"`
	LLMReadable bool   `json:"llm_readable"`
	IsPK        bool   `json:"is_pk"`
}

// LinkDescriptor describes a named relationship to another object type.
type LinkDescriptor struct {
	Name       string `json:"name"`
	TargetType string `json:"target_type"`
	Via        string `json:"via"`
}

// LLMAccessDescriptor describes LLM access constraints for an object type.
type LLMAccessDescriptor struct {
	CanRead  bool `json:"can_read"`
	CanWrite bool `json:"can_write"`
	ReadOnly bool `json:"read_only"`
}

// ObjectContext is a lightweight representation of an object.
type ObjectContext struct {
	ObjectType string                 `json:"object_type"`
	ObjectID   string                 `json:"object_id"`
	Properties map[string]interface{} `json:"properties"`
}

// LinkedObjectsResult holds linked objects grouped by relationship.
type LinkedObjectsResult struct {
	ObjectType string       `json:"object_type"`
	ObjectID   string       `json:"object_id"`
	Links      []LinkResult `json:"links"`
}

// LinkResult holds the linked objects for a single relationship.
type LinkResult struct {
	LinkName   string          `json:"link_name"`
	TargetType string          `json:"target_type"`
	Objects    []ObjectContext `json:"objects"`
}

// ProposeActionTrace carries optional trace metadata for proposal creation.
type ProposeActionTrace struct {
	DecisionID   string // pre-created LLM decision ID
	CaseID       string // case_id extracted from decision_json
	EvidenceRefs string // JSON array string of evidence reference IDs
	ContextHash  string
	RecipeID     string
}

// ActionResult holds the result of executing an action on an object.
type ActionResult struct {
	Success    bool                   `json:"success"`
	ActionType string                 `json:"action_type"`
	ObjectType string                 `json:"object_type"`
	ObjectID   string                 `json:"object_id"`
	Result     map[string]interface{} `json:"result,omitempty"`
}

// OntologyService defines the interface for ontology-related MCP operations.
type OntologyService interface {
	DescribeOntology(ctx context.Context) (*OntologyDescriptor, error)
	GetObject(ctx context.Context, objectType, objectID string) (*ObjectContext, error)
	GetObjectMetrics(ctx context.Context, objectType, objectID string) (map[string]float64, error)
	GetLinkedObjects(ctx context.Context, objectType, objectID, linkName string, maxDepth int) (*LinkedObjectsResult, error)
	ExecuteAction(ctx context.Context, objectType, objectID, actionType string, params map[string]interface{}) (*ActionResult, error)
	ProposeAction(ctx context.Context, objectType, objectID, actionType string, params map[string]interface{}, trace ProposeActionTrace) (*ActionResult, error)
}

// BuildContextService defines the interface for building recipe-driven contexts.
type BuildContextService interface {
	BuildEnvelope(ctx context.Context, caseID, recipeID string) (*llm.LLMSafeContextEnvelope, error)
}

// ActionSchemaService defines the interface for action schema operations.
type ActionSchemaService interface {
	ListActionSchemas(ctx context.Context) ([]ActionDefinition, error)
	GetActionSchema(ctx context.Context, actionType string) (*ActionDefinition, error)
}

// ActionDefinition describes the structured schema for a single action type.
type ActionDefinition struct {
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	RiskLevel     string                 `json:"risk_level"`
	PayloadSchema map[string]interface{} `json:"payload_schema"`
	AllowedBy     []string               `json:"allowed_by"`
	Adapter       string                 `json:"adapter"`
}

// SandboxService defines the interface for proposal sandbox operations.
type SandboxService interface {
	CreateSandbox(ctx context.Context, caseID string, data map[string]interface{}) (string, error)
	AddProposalToSandbox(ctx context.Context, sandboxID, proposalID string) error
	CompareSandbox(ctx context.Context, sandboxID1, sandboxID2 string) (*ComparisonResult, error)
	GetSandbox(ctx context.Context, sandboxID string) (*Sandbox, error)
}

// Sandbox represents a persistent proposal sandbox.
type Sandbox struct {
	SandboxID    string                 `json:"sandbox_id"`
	CaseID       string                 `json:"case_id"`
	ProposalID   string                 `json:"proposal_id,omitempty"`
	Data         map[string]interface{} `json:"data"`
	Status       string                 `json:"status"`
	ComparedWith []string               `json:"compared_with,omitempty"`
	CreatedAt    string                 `json:"created_at"`
}

// ComparisonResult holds the result of comparing two sandboxes.
type ComparisonResult struct {
	Sandbox1ID  string       `json:"sandbox_1_id"`
	Sandbox2ID  string       `json:"sandbox_2_id"`
	Differences []Difference `json:"differences"`
}

// Difference represents a single field difference between two sandboxes.
type Difference struct {
	Field  string      `json:"field"`
	Value1 interface{} `json:"value_1"`
	Value2 interface{} `json:"value_2"`
}

// PipelineInfoService defines the interface for pipeline status information.
type PipelineInfoService interface {
	GetLastRunStatus(ctx context.Context) (*model.PipelineRun, error)
	ListRuns(ctx context.Context, limit int) ([]model.PipelineRun, error)
}
