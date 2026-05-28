package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"baxi/internal/action"
	"baxi/internal/adapter"
	"baxi/internal/config"
	"baxi/internal/db"
	"baxi/internal/logger"
	"baxi/internal/outbox"
	"baxi/internal/worker"

	"go.uber.org/zap"
)

func main() {
	dryRunFlag := flag.Bool("dry-run", false, "force dry-run mode for dispatch worker")
	flag.Parse()

	// Create root context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		os.Stderr.WriteString("failed to load config: " + err.Error() + "\n")
		os.Exit(1)
	}

	dryRun := *dryRunFlag || cfg.ActionApplyDryRun

	zapLog, err := logger.New(cfg.LogLevel)
	if err != nil {
		os.Stderr.WriteString("failed to initialize logger: " + err.Error() + "\n")
		os.Exit(1)
	}

	// Connect to PostgreSQL
	pool, err := db.NewPool(ctx, cfg.DatabaseURL, zapLog)
	if err != nil {
		zapLog.Fatal("failed to connect to database", zap.Error(err))
	}
	defer pool.Close()

	w := worker.New(zapLog, pool.Pool)
	go func() {
		if err := w.Run(ctx); err != nil {
			zapLog.Fatal("worker error", zap.Error(err))
		}
	}()

	pollInterval, err := time.ParseDuration(cfg.WorkerTickInterval)
	if err != nil {
		zapLog.Warn("invalid WORKER_TICK_INTERVAL, using default 30s", zap.Error(err))
		pollInterval = 30 * time.Second
	}

	dispatchConfig := worker.DispatchConfig{
		PollInterval: pollInterval,
		BatchSize:    cfg.WorkerBatchSize,
		MaxRetries:   10,
		DryRun:       dryRun,
	}

	repo := outbox.NewOutboxRepository()

	feishuAdapter := adapter.NewFeishuAdapter(adapter.FeishuConfig{
		WebhookURL: cfg.FeishuWebhookURL,
		Enabled:    true,
	})
	githubAdapter := adapter.NewGitHubAdapter(adapter.GitHubConfig{
		Token:   cfg.GitHubToken,
		Enabled: true,
	})

	executors := map[string]action.ActionExecutor{
		"feishu": feishuAdapter,
		"github": githubAdapter,
	}

	dispatchWorker := worker.NewDispatchWorker(repo, pool.Pool, executors, dispatchConfig)
	go func() {
		if err := dispatchWorker.Run(ctx); err != nil {
			zapLog.Fatal("dispatch worker error", zap.Error(err))
		}
	}()

	zapLog.Info("baxi-worker started",
		zap.Bool("dry_run", dryRun),
		zap.Duration("poll_interval", pollInterval),
		zap.Int("batch_size", cfg.WorkerBatchSize),
	)

	// Wait for signal
	<-sigCh
	zapLog.Info("received shutdown signal")

	// Cancel context to trigger worker shutdown
	cancel()
}
