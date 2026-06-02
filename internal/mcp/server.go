package mcp

import (
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
		"Baxi MCP Server",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithInstructions("E-commerce governance and decision platform"),
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

func (s *Server) Run() error {
	return server.ServeStdio(s.server)
}
