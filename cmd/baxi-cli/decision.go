package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"baxi/internal/action"
	"baxi/internal/decision"
	"baxi/internal/governance"
	"baxi/internal/llm"
	"baxi/internal/ontology"
	"baxi/internal/repository"
	"baxi/internal/service"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func handleDecision(ctx context.Context, args []string, log *zap.Logger, pool *pgxpool.Pool) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "decision: missing subcommand")
		fmt.Fprintln(os.Stderr, "Usage: baxi-cli decision <create|context|decide|list> [options]")
		os.Exit(1)
	}

	switch args[0] {
	case "create":
		handleDecisionCreate(ctx, args[1:], log, pool)
	case "context":
		handleDecisionContext(ctx, args[1:], log, pool)
	case "decide":
		handleDecisionDecide(ctx, args[1:], log, pool)
	case "list":
		handleDecisionList(ctx, args[1:], log, pool)
	default:
		fmt.Fprintf(os.Stderr, "decision: unknown subcommand: %s\n", args[0])
		fmt.Fprintln(os.Stderr, "Usage: baxi-cli decision <create|context|decide|list> [options]")
		os.Exit(1)
	}
}

func handleDecisionCreate(ctx context.Context, args []string, log *zap.Logger, pool *pgxpool.Pool) {
	fs := flag.NewFlagSet("decision create", flag.ExitOnError)
	alertID := fs.String("alert-id", "", "Alert ID to create case from")
	if err := fs.Parse(args); err != nil {
		log.Fatal("failed to parse decision create flags", zap.Error(err))
	}

	if *alertID == "" {
		fmt.Fprintln(os.Stderr, "Usage: baxi-cli decision create --alert-id=ALERT_ID")
		os.Exit(1)
	}

	decisionRepo := repository.NewDecisionRepository()
	alertRepo := repository.NewAlertRepository()
	caseSvc := decision.NewCaseService(decisionRepo, alertRepo, pool)

	c, err := caseSvc.CreateCaseFromAlert(ctx, *alertID, "cli")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create decision case: %v\n", err)
		os.Exit(1)
	}

	b, _ := json.MarshalIndent(c, "", "  ")
	fmt.Println(string(b))
}

func handleDecisionContext(ctx context.Context, args []string, log *zap.Logger, pool *pgxpool.Pool) {
	fs := flag.NewFlagSet("decision context", flag.ExitOnError)
	caseID := fs.String("case-id", "", "Decision case ID")
	if err := fs.Parse(args); err != nil {
		log.Fatal("failed to parse decision context flags", zap.Error(err))
	}

	if *caseID == "" {
		fmt.Fprintln(os.Stderr, "Usage: baxi-cli decision context --case-id=CASE_ID")
		os.Exit(1)
	}

	decisionRepo := repository.NewDecisionRepository()
	objectSvc := ontology.NewObjectQueryService(repository.NewOntologyRepo(), pool)
	classSvc := governance.NewClassificationService(pool, repository.NewGovernanceRepository())

	ctxBuilder := decision.NewContextBuilder(decisionRepo, objectSvc, classSvc, pool)

	dc, err := ctxBuilder.BuildDecisionContext(ctx, *caseID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to build context: %v\n", err)
		os.Exit(1)
	}

	b, _ := json.MarshalIndent(dc, "", "  ")
	fmt.Println(string(b))
}

func handleDecisionDecide(ctx context.Context, args []string, log *zap.Logger, pool *pgxpool.Pool) {
	fs := flag.NewFlagSet("decision decide", flag.ExitOnError)
	caseID := fs.String("case-id", "", "Decision case ID")
	if err := fs.Parse(args); err != nil {
		log.Fatal("failed to parse decision decide flags", zap.Error(err))
	}

	if *caseID == "" {
		fmt.Fprintln(os.Stderr, "Usage: baxi-cli decision decide --case-id=CASE_ID")
		os.Exit(1)
	}

	decisionRepo := repository.NewDecisionRepository()
	alertRepo := repository.NewAlertRepository()

	caseSvc := decision.NewCaseService(decisionRepo, alertRepo, pool)
	objectSvc := ontology.NewObjectQueryService(repository.NewOntologyRepo(), pool)
	classSvc := governance.NewClassificationService(pool, repository.NewGovernanceRepository())
	ctxBuilder := decision.NewContextBuilder(decisionRepo, objectSvc, classSvc, pool)
	ruleProvider := llm.NewRuleBasedProvider()
	engine := decision.NewDecisionEngine(ruleProvider, decisionRepo, pool)
	proposalSvc := action.NewProposalService(decisionRepo, decisionRepo, pool)

	svc := service.NewDecisionService(caseSvc, ctxBuilder, engine, proposalSvc, pool)

	dc, output, proposals, err := svc.Decide(ctx, *caseID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to generate decision: %v\n", err)
		os.Exit(1)
	}

	result := map[string]interface{}{
		"context":   dc,
		"decision":  output,
		"proposals": proposals,
	}
	b, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(b))
}

func handleDecisionList(ctx context.Context, args []string, log *zap.Logger, pool *pgxpool.Pool) {
	fs := flag.NewFlagSet("decision list", flag.ExitOnError)
	status := fs.String("status", "", "Filter by status")
	severity := fs.String("severity", "", "Filter by severity")
	limit := fs.Int("limit", 10, "Max results")
	offset := fs.Int("offset", 0, "Offset")
	if err := fs.Parse(args); err != nil {
		log.Fatal("failed to parse decision list flags", zap.Error(err))
	}

	decisionRepo := repository.NewDecisionRepository()
	alertRepo := repository.NewAlertRepository()
	caseSvc := decision.NewCaseService(decisionRepo, alertRepo, pool)

	filter := decision.CaseFilter{
		Status:   strPtr(*status),
		Severity: strPtr(*severity),
		Limit:    *limit,
		Offset:   *offset,
	}

	result, err := caseSvc.ListCases(ctx, filter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to list cases: %v\n", err)
		os.Exit(1)
	}

	b, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(b))
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
