package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"baxi/internal/config"
	"baxi/internal/db"
	"baxi/internal/logger"
	"baxi/internal/worker"

	"go.uber.org/zap"
)

func main() {
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

	// Initialize logger
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

	// Create and run worker
	w := worker.New(zapLog, pool.Pool)
	go func() {
		if err := w.Run(ctx); err != nil {
			zapLog.Fatal("worker error", zap.Error(err))
		}
	}()

	zapLog.Info("baxi-worker started")

	// Wait for signal
	<-sigCh
	zapLog.Info("received shutdown signal")

	// Cancel context to trigger worker shutdown
	cancel()
}
