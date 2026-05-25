package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"baxi/internal/api"
	"baxi/internal/config"
	"baxi/internal/db"
	"baxi/internal/logger"

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

	// Create and start API server
	server := api.New(zapLog, pool.Pool)
	go func() {
		if err := server.Start(":" + cfg.APIPort); err != nil {
			zapLog.Fatal("server error", zap.Error(err))
		}
	}()

	zapLog.Info("baxi-api started", zap.String("port", cfg.APIPort))

	// Wait for signal
	select {
	case sig := <-sigCh:
		zapLog.Info("received signal", zap.String("signal", sig.String()))
	case <-ctx.Done():
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		zapLog.Error("shutdown error", zap.Error(err))
	}

	zapLog.Info("server stopped")
}
