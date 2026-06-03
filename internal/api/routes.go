package api

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	apimw "baxi/internal/api/middleware"
)

func (s *Server) setupRoutes() {
	s.router.Use(apimw.RequestIDMiddleware)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Logger)
	s.router.Use(apimw.RecoveryMiddleware)
	s.router.Use(middleware.Timeout(30 * time.Second))
	s.router.Use(apimw.NewCORSMiddleware(s.corsAllowedOrigins))

	s.router.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", s.handleHealth)

		r.Group(func(r chi.Router) {
			r.Use(apimw.NewAuthMiddleware(s.bearerToken))

			r.Get("/qoder/capabilities", s.qoderHandler().HandleCapabilities)
			r.Get("/qoder/context", s.qoderHandler().HandleContext)
			r.Get("/status", s.statusHandler().HandleStatus)
			r.Get("/alerts", s.alertHandler().HandleListAlerts)
			r.Get("/tasks", s.handleListTasks)
			r.Post("/outbox/dispatch", s.outboxHandler().HandleBatchDispatch)
			r.Get("/outbox", s.outboxHandler().HandleListOutbox)
			r.Get("/outbox/{id}", s.outboxHandler().HandleGetDetail)
			r.Post("/outbox/{id}/dispatch", s.outboxHandler().HandleDispatch)
			r.Post("/outbox/{id}/cancel", s.outboxHandler().HandleCancel)
			r.Get("/governance/status", s.governanceHandler().HandleGovernanceStatus)
			r.Get("/governance/catalog", s.governanceHandler().HandleCatalog)
			r.Get("/governance/classification", s.governanceHandler().HandleClassification)
			r.Get("/governance/markings", s.governanceHandler().HandleMarkings)
			r.Get("/governance/lineage", s.governanceHandler().HandleLineage)
			r.Get("/governance/checkpoints", s.governanceHandler().HandleCheckpoints)
			r.Get("/governance/health", s.governanceHandler().HandleHealth)
			r.Get("/logs/recent", s.logHandler().HandleListRecent)
			r.Get("/logs/errors", s.logHandler().HandleListErrors)
			r.Get("/logs/audit", s.logHandler().HandleListAudit)
			r.Get("/logs/diagnosis", s.diagnosisHandler().HandleDiagnosis)
			r.Get("/logs/agent", s.agentLogHandler().HandleListAgentLogs)

			r.Get("/llm/status", s.llmHandler().Status)
			r.Get("/llm/metrics", s.llmHandler().Metrics)

			r.Post("/decisions/cases", s.decisionHandler().CreateCase)
			r.Get("/decisions/cases", s.decisionHandler().ListCases)
			r.Get("/decisions/cases/{case_id}", s.decisionHandler().GetCase)
			r.Post("/decisions/cases/{case_id}/context", s.decisionHandler().BuildContext)
			r.Post("/decisions/cases/{case_id}/decide", s.decisionHandler().Decide)
			r.Get("/decisions/cases/{case_id}/proposals", s.decisionHandler().ListProposals)

			r.Post("/decisions/cases/{case_id}/decide/llm", s.decisionHandler().DecideLLM)
			r.Post("/decisions/cases/{case_id}/compare", s.decisionHandler().Compare)
			r.Post("/decisions/cases/{case_id}/replay", s.decisionHandler().Replay)
			r.Get("/decisions/cases/{case_id}/llm-decisions", s.decisionHandler().ListLLMDecisions)
			r.Get("/decisions/cases/{case_id}/evals", s.decisionHandler().ListEvals)

			r.Post("/proposals/{id}/approve", s.reviewHandler().HandleApprove)
			r.Post("/proposals/{id}/reject", s.reviewHandler().HandleReject)
			r.Post("/proposals/{id}/cancel", s.reviewHandler().HandleCancel)
			r.Get("/proposals/{id}/review", s.reviewHandler().HandleGetReview)

			r.Post("/proposals/{id}/execute", s.actionHandler().HandleExecute)
			r.Get("/proposals/{id}/status", s.actionHandler().HandleStatus)

			r.Post("/pipeline/run", s.pipelineHandler().HandleRun)

			r.Post("/feishu/export", s.feishuHandler().HandleExport)
			r.Post("/feishu/sync", s.feishuHandler().HandleSync)
			r.Post("/feishu/status/import", s.feishuHandler().HandleStatusImport)

			r.Post("/sandboxes", s.sandboxHandler().HandleCreate)
			r.Get("/sandboxes", s.sandboxHandler().HandleList)
			r.Get("/sandboxes/compare", s.sandboxHandler().HandleCompare)
			r.Get("/sandboxes/{id}", s.sandboxHandler().HandleGet)
			r.Post("/sandboxes/{id}/proposals", s.sandboxHandler().HandleAddProposal)
		})
	})
}
