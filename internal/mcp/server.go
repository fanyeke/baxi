package mcp

import (
	"github.com/mark3labs/mcp-go/server"
)

// Server wraps the MCP server with service dependencies.
type Server struct {
	server         *server.MCPServer
	decisionSvc    DecisionService
	decisionEngine DecisionEngine
	contextBuilder   ContextBuilder
	proposalSvc    ProposalService
	alertSvc       AlertService
	govSvc         GovernanceService
	pipelineRunner PipelineRunner
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
) (*Server, error) {
	s := server.NewMCPServer(
		"Baxi MCP Server",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithInstructions("E-commerce governance and decision platform"),
	)

	srv := &Server{
		server:         s,
		decisionSvc:    decisionSvc,
		decisionEngine: decisionEngine,
		contextBuilder: contextBuilder,
		proposalSvc:    proposalSvc,
		alertSvc:       alertSvc,
		govSvc:         govSvc,
		pipelineRunner: pipelineRunner,
	}

	srv.registerDecisionTools()
	srv.registerAlertTools()
	srv.registerGovernanceTools()
	srv.registerPipelineTools()

	return srv, nil
}

// Run starts the MCP server using stdio transport.
func (s *Server) Run() error {
	return server.ServeStdio(s.server)
}
