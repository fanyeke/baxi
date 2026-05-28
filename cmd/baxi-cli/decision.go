package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	stdlog "log"
	"os"

	"baxi/internal/action"
	"baxi/internal/config"
	"baxi/internal/decision"
	"baxi/internal/governance"
	"baxi/internal/llm"
	"baxi/internal/ontology"
	"baxi/internal/repository"
	"baxi/internal/service"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func handleDecision(ctx context.Context, args []string, log *zap.Logger, pool *pgxpool.Pool, cfg *config.Config) {
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
		handleDecisionDecide(ctx, args[1:], log, pool, cfg)
	case "list":
		handleDecisionList(ctx, args[1:], log, pool)
	case "compare":
		handleDecisionCompare(ctx, args[1:], log, pool)
	case "replay":
		handleDecisionReplay(ctx, args[1:], log, pool)
	case "evals":
		handleDecisionEvals(ctx, args[1:], log, pool)
	default:
		fmt.Fprintf(os.Stderr, "decision: unknown subcommand: %s\n", args[0])
		fmt.Fprintln(os.Stderr, "Usage: baxi-cli decision <create|context|decide|list|compare|replay|evals> [options]")
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

	reg, _ := action.NewActionRegistry("")
	ctxBuilder := decision.NewContextBuilder(decisionRepo, objectSvc, classSvc, pool, action.NewActionTypeProviderAdapter(reg))

	dc, err := ctxBuilder.BuildDecisionContext(ctx, *caseID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to build context: %v\n", err)
		os.Exit(1)
	}

	b, _ := json.MarshalIndent(dc, "", "  ")
	fmt.Println(string(b))
}

func handleDecisionDecide(ctx context.Context, args []string, log *zap.Logger, pool *pgxpool.Pool, cfg *config.Config) {
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
	reg, _ := action.NewActionRegistry("")
	ctxBuilder := decision.NewContextBuilder(decisionRepo, objectSvc, classSvc, pool, action.NewActionTypeProviderAdapter(reg))
	promptReg, _ := llm.NewPromptRegistry()
	factory := llm.NewProviderFactory(cfg, promptReg)
	provider, providerErr := factory.CreateProvider()
	if providerErr != nil {
		stdlog.Printf("WARNING: failed to create LLM provider: %v, falling back to rule-based", providerErr)
		provider = llm.NewRuleBasedProvider()
	}
	engine := decision.NewDecisionEngine(provider, decisionRepo, pool, llm.NewDBAuditLogger(pool))
	proposalSvc := action.NewProposalService(decisionRepo, decisionRepo, reg, pool)

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

func handleDecisionCompare(ctx context.Context, args []string, log *zap.Logger, pool *pgxpool.Pool) {
	fs := flag.NewFlagSet("decision compare", flag.ExitOnError)
	caseID := fs.String("case-id", "", "Decision case ID")
	if err := fs.Parse(args); err != nil {
		log.Fatal("failed to parse decision compare flags", zap.Error(err))
	}

	if *caseID == "" {
		fmt.Fprintln(os.Stderr, "Usage: baxi-cli decision compare --case-id=CASE_ID")
		os.Exit(1)
	}

	resp, err := apiGet(fmt.Sprintf("/api/v1/decisions/cases/%s/compare", *caseID))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to call compare API: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if err := checkResponse(resp); err != nil {
		fmt.Fprintf(os.Stderr, "Error: compare API failed: %v\n", err)
		os.Exit(1)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to decode response: %v\n", err)
		os.Exit(1)
	}

	b, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(b))
}

func handleDecisionReplay(ctx context.Context, args []string, log *zap.Logger, pool *pgxpool.Pool) {
	fs := flag.NewFlagSet("decision replay", flag.ExitOnError)
	caseID := fs.String("case-id", "", "Decision case ID")
	dryRun := fs.Bool("dry-run", true, "Dry-run mode (default: true)")
	if err := fs.Parse(args); err != nil {
		log.Fatal("failed to parse decision replay flags", zap.Error(err))
	}

	if *caseID == "" {
		fmt.Fprintln(os.Stderr, "Usage: baxi-cli decision replay --case-id=CASE_ID [--dry-run=true]")
		os.Exit(1)
	}

	path := fmt.Sprintf("/api/v1/decisions/cases/%s/replay?dry_run=%t", *caseID, *dryRun)
	resp, err := apiPost(path, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to call replay API: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if err := checkResponse(resp); err != nil {
		fmt.Fprintf(os.Stderr, "Error: replay API failed: %v\n", err)
		os.Exit(1)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to decode response: %v\n", err)
		os.Exit(1)
	}

	b, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(b))
}

func handleDecisionEvals(ctx context.Context, args []string, log *zap.Logger, pool *pgxpool.Pool) {
	fs := flag.NewFlagSet("decision evals", flag.ExitOnError)
	caseID := fs.String("case-id", "", "Decision case ID")
	if err := fs.Parse(args); err != nil {
		log.Fatal("failed to parse decision evals flags", zap.Error(err))
	}

	if *caseID == "" {
		fmt.Fprintln(os.Stderr, "Usage: baxi-cli decision evals --case-id=CASE_ID")
		os.Exit(1)
	}

	resp, err := apiGet(fmt.Sprintf("/api/v1/decisions/cases/%s/evals", *caseID))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to call evals API: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if err := checkResponse(resp); err != nil {
		fmt.Fprintf(os.Stderr, "Error: evals API failed: %v\n", err)
		os.Exit(1)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to decode response: %v\n", err)
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
