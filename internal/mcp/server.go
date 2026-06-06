package mcp

import (
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mark3labs/mcp-go/server"

	"baxi/internal/ontology"
)

type Server struct {
	server                 *server.MCPServer
	decisionSvc            DecisionService
	decisionEngine         DecisionEngine
	contextBuilder         ContextBuilder
	buildContextSvc        BuildContextService
	proposalSvc            ProposalService
	alertSvc               AlertService
	govSvc                 GovernanceService
	pipelineRunner         PipelineRunner
	reviewSvc              ReviewService
	outboxSvc              OutboxService
	pipelineInfoSvc        PipelineInfoService
	statusSvc              SystemStatusService
	searchSvc              ObjectSearchService
	executeSvc             ExecuteService
	ontologySvc            OntologyService
	schemaSvc              ActionSchemaService
	sandboxSvc             SandboxService
	pool                   *pgxpool.Pool
	linkResolver           *ontology.LinkResolver
	actionBindingValidator *ontology.ActionBindingValidator
	objectTypesV2          map[string]*ontology.ObjectTypeV2
	recipes                map[string]*ontology.ContextRecipe

	// mcpUserID is the authenticated identity of the MCP caller, used for
	// review/approve/execute operations. Set from BAXI_MCP_USER_ID env var.
	// This prevents callers from fabricating arbitrary reviewer identities.
	mcpUserID string
	// mcpRole is the caller's role, used for allowed_by authorization checks.
	// Set from BAXI_MCP_ROLE env var.
	mcpRole string
}

func mcpUserIDFromEnv() string {
	if v := os.Getenv("BAXI_MCP_USER_ID"); v != "" {
		return v
	}
	return "mcp_system"
}

func mcpRoleFromEnv() string {
	return os.Getenv("BAXI_MCP_ROLE")
}

func NewServer(
	decisionSvc DecisionService,
	decisionEngine DecisionEngine,
	contextBuilder ContextBuilder,
	buildContextSvc BuildContextService,
	proposalSvc ProposalService,
	alertSvc AlertService,
	govSvc GovernanceService,
	pipelineRunner PipelineRunner,
	reviewSvc ReviewService,
	outboxSvc OutboxService,
	pipelineInfoSvc PipelineInfoService,
	executeSvc ExecuteService,
	pool *pgxpool.Pool,
	statusSvc SystemStatusService,
	searchSvc ObjectSearchService,
	ontologySvc OntologyService,
	schemaSvc ActionSchemaService,
	sandboxSvc SandboxService,
) (*Server, error) {
	s := server.NewMCPServer(
		getServerName(),
		getServerVersion(),
		server.WithToolCapabilities(false),
		server.WithInstructions(getServerInstructions()),
	)

	srv := &Server{
		server:          s,
		decisionSvc:     decisionSvc,
		decisionEngine:  decisionEngine,
		contextBuilder:  contextBuilder,
		buildContextSvc: buildContextSvc,
		proposalSvc:     proposalSvc,
		alertSvc:        alertSvc,
		govSvc:          govSvc,
		pipelineRunner:  pipelineRunner,
		reviewSvc:       reviewSvc,
		outboxSvc:       outboxSvc,
		pipelineInfoSvc: pipelineInfoSvc,
		statusSvc:       statusSvc,
		searchSvc:       searchSvc,
		executeSvc:      executeSvc,
		ontologySvc:     ontologySvc,
		schemaSvc:       schemaSvc,
		sandboxSvc:      sandboxSvc,
		pool:            pool,
		mcpUserID:       mcpUserIDFromEnv(),
		mcpRole:         mcpRoleFromEnv(),
	}

	srv.registerDecisionTools()
	srv.registerAlertTools()
	srv.registerGovernanceTools()
	srv.registerPipelineTools()
	srv.registerOutboxTools()
	srv.registerReviewTools()
	srv.registerStatusTools()
	srv.registerActionTools()
	srv.registerOntologyTools()
	srv.registerContextTools()
	srv.registerSchemaTools()
	srv.registerSandboxTools()

	return srv, nil
}

func (s *Server) SetLinkResolver(lr *ontology.LinkResolver) {
	s.linkResolver = lr
}

func (s *Server) SetActionBindingValidator(abv *ontology.ActionBindingValidator) {
	s.actionBindingValidator = abv
}

func (s *Server) SetObjectTypesV2(ots map[string]*ontology.ObjectTypeV2) {
	s.objectTypesV2 = ots
}

func (s *Server) SetRecipes(recipes map[string]*ontology.ContextRecipe) {
	s.recipes = recipes
}

func (s *Server) Run() error {
	return server.ServeStdio(s.server)
}
