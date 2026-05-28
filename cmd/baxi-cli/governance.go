package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"baxi/internal/configloader"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func handleGovernance(ctx context.Context, args []string, log *zap.Logger, pool *pgxpool.Pool) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "governance: missing subcommand")
		fmt.Fprintln(os.Stderr, "Usage: baxi-cli governance <load|check> [options]")
		os.Exit(1)
	}

	switch args[0] {
	case "load":
		handleGovernanceLoad(ctx, args[1:], log, pool)
	case "check":
		handleGovernanceCheck(ctx, args[1:], log, pool)
	default:
		fmt.Fprintf(os.Stderr, "governance: unknown subcommand: %s\n", args[0])
		fmt.Fprintln(os.Stderr, "Usage: baxi-cli governance <load|check> [options]")
		os.Exit(1)
	}
}

func handleGovernanceLoad(ctx context.Context, args []string, log *zap.Logger, pool *pgxpool.Pool) {
	loadCmd := flag.NewFlagSet("governance load", flag.ExitOnError)
	configDir := loadCmd.String("config-dir", "./config", "Directory containing governance YAML config files")

	if err := loadCmd.Parse(args); err != nil {
		log.Fatal("failed to parse governance load flags", zap.Error(err))
	}

	cl := configloader.NewConfigLoader(pool)

	registry, err := cl.LoadAll(ctx, *configDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load configs: %v\n", err)
		os.Exit(1)
	}

	if err := configloader.ValidateRequired(registry); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}

	if err := cl.SyncSnapshots(ctx, registry); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to sync configs to database: %v\n", err)
		os.Exit(1)
	}

	keys := configloader.ListConfigKeys(registry)
	fmt.Printf("Loaded %d config(s) from %s:\n", len(keys), *configDir)
	for _, k := range keys {
		raw := registry.RawConfigs[k]
		fmt.Printf("  - %s (%s, %s)\n", k, raw.ConfigType, raw.SourcePath)
	}

	fmt.Println()
	fmt.Println("Configs synced to database successfully")
}

func handleGovernanceCheck(ctx context.Context, _ []string, log *zap.Logger, pool *pgxpool.Pool) {
	log.Info("checking governance configs in database")

	type tableInfo struct {
		Label string
		Query string
	}

	tables := []tableInfo{
		{Label: "gov.config_snapshot", Query: "SELECT COUNT(*) FROM gov.config_snapshot"},
		{Label: "gov.object_schema", Query: "SELECT COUNT(*) FROM gov.object_schema"},
		{Label: "gov.data_classification", Query: "SELECT COUNT(*) FROM gov.data_classification"},
		{Label: "gov.data_lineage", Query: "SELECT COUNT(*) FROM gov.data_lineage"},
		{Label: "gov.access_policy", Query: "SELECT COUNT(*) FROM gov.access_policy"},
	}

	allPass := true

	fmt.Printf("%-30s %-12s\n", "TABLE", "ROW COUNT")
	fmt.Println(strings.Repeat("-", 44))

	for _, t := range tables {
		var count int64
		if err := pool.QueryRow(ctx, t.Query).Scan(&count); err != nil {
			fmt.Printf("%-30s %-12s\n", t.Label, "ERR")
			fmt.Printf("  → Error: %v\n", err)
			allPass = false
			continue
		}
		fmt.Printf("%-30s %-12d\n", t.Label, count)
	}

	fmt.Println()

	type configCheck struct {
		ConfigKey string
		Label     string
	}

	checks := []configCheck{
		{ConfigKey: "aip_object_schema", Label: "aip_object_schema"},
		{ConfigKey: "data_classification", Label: "data_classification"},
		{ConfigKey: "access_policy", Label: "access_policy"},
		{ConfigKey: "data_lineage", Label: "data_lineage"},
		{ConfigKey: "data_markings", Label: "data_markings (optional)"},
		{ConfigKey: "health_checks", Label: "health_checks (optional)"},
		{ConfigKey: "checkpoint_rules", Label: "checkpoint_rules (optional)"},
		{ConfigKey: "alert_rules", Label: "alert_rules (optional)"},
		{ConfigKey: "metrics", Label: "metrics (optional)"},
	}

	fmt.Printf("%-30s %-12s\n", "CONFIG CHECK", "STATUS")
	fmt.Println(strings.Repeat("-", 44))

	for _, c := range checks {
		var count int64
		err := pool.QueryRow(ctx,
			"SELECT COUNT(*) FROM gov.config_snapshot WHERE config_key = $1", c.ConfigKey,
		).Scan(&count)

		status := "PASS"
		if err != nil || count == 0 {
			status = "MISSING"
			if !strings.Contains(c.Label, "(optional)") {
				allPass = false
			}
		}
		fmt.Printf("%-30s %-12s\n", c.Label, status)
	}

	fmt.Println()

	if allPass {
		fmt.Println("All governance checks PASSED")
		os.Exit(0)
	} else {
		fmt.Println("Some governance checks FAILED")
		os.Exit(1)
	}
}
