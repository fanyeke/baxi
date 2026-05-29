package mcp

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mark3labs/mcp-go/server"
)

// Server wraps the MCP server with service dependencies.
type Server struct {
	server          *server.MCPServer
	decisionSvc     DecisionService
	decisionEngine  DecisionEngine
	contextBuilder  ContextBuilder
	proposalSvc     ProposalService
	alertSvc        AlertService
	govSvc          GovernanceService
	pipelineRunner  PipelineRunner
	reviewSvc       ReviewService
	outboxSvc       OutboxService
	pipelineInfoSvc PipelineInfoService
	statusSvc       SystemStatusService
	searchSvc       ObjectSearchService
	executeSvc      ExecuteService
	ontologySvc     OntologyService
	pool            *pgxpool.Pool
}

// NewServer creates a new MCP server with the given service dependencies.
func NewServer(
	decisionSvc DecisionService,
	decisionEngine DecisionEngine,
	contextBuilder ContextBuilder,
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

	return srv, nil
}

// Run starts the MCP server using stdio transport.
func (s *Server) Run() error {
	return server.ServeStdio(s.server)
}
