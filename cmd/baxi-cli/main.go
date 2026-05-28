package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"baxi/internal/config"
	"baxi/internal/db"
	"baxi/internal/logger"

	"go.uber.org/zap"
)

func main() {
	// Early-exit commands that don't need configuration
	if len(os.Args) < 2 {
		printHelp()
		return
	}
	switch os.Args[1] {
	case "help", "--help", "-h":
		printHelp()
		return
	case "pipeline", "governance", "decision":
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printHelp()
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Load config
	cfg, err := config.Load()
	if err != nil {
		os.Stderr.WriteString("failed to load config: " + err.Error() + "\n")
		os.Exit(1)
	}

	// Initialize logger
	zapLog, err := logger.New(cfg.LogLevel)
	if err != nil {
		os.Stderr.WriteString("failed to initialize logger: " + err.Error() + "\n")
		os.Exit(1)
	}

	// Connect to database
	pool, err := db.NewPool(ctx, cfg.DatabaseURL, zapLog)
	if err != nil {
		zapLog.Fatal("failed to connect to database", zap.Error(err))
	}
	defer pool.Close()

	// Dispatch subcommand
	go func() {
		switch os.Args[1] {
		case "pipeline":
			handlePipeline(ctx, os.Args[2:], zapLog, pool.Pool)
		case "governance":
			handleGovernance(ctx, os.Args[2:], zapLog, pool.Pool)
		case "decision":
			handleDecision(ctx, os.Args[2:], zapLog, pool.Pool, cfg)
		}
		cancel()
	}()

	// Wait for signal or completion
	select {
	case sig := <-sigCh:
		zapLog.Info("received signal", zap.String("signal", sig.String()))
		cancel()
	case <-ctx.Done():
	}

	zapLog.Info("baxi-cli stopped")
}

func printHelp() {
	fmt.Println("Usage: baxi-cli <command> [options]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  pipeline run                 Run the full data pipeline")
	fmt.Println("  pipeline run --step <name>   Run a specific pipeline step")
	fmt.Println("  pipeline run --data-dir <p>  Data directory for CSV files")
	fmt.Println("  pipeline validate            Compare pipeline outputs against baseline")
	fmt.Println("  governance load              Load governance YAML configs into database")
	fmt.Println("  governance load --config-dir <d>  Config directory (default: ./config)")
	fmt.Println("  governance check             Check governance configs in database")
	fmt.Println("  decision create              Create a decision case from an alert")
	fmt.Println("  decision context             Build context for a decision case")
	fmt.Println("  decision decide              Generate decision and proposals")
	fmt.Println("  decision list                List decision cases")
	fmt.Println("")
	fmt.Println("Environment:")
	fmt.Println("  DATABASE_URL              PostgreSQL connection string (required)")
	fmt.Println("  LOG_LEVEL                 Log level: debug, info, warn, error (default: info)")
	fmt.Println("")
	fmt.Println("Flags:")
	flag.PrintDefaults()
}
