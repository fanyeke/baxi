package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"baxi/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func handleLLM(ctx context.Context, args []string, log *zap.Logger, pool *pgxpool.Pool, cfg *config.Config) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "llm: missing subcommand")
		fmt.Fprintln(os.Stderr, "Usage: baxi-cli llm <status|metrics>")
		os.Exit(1)
	}

	switch args[0] {
	case "status":
		llmStatus()
	case "metrics":
		llmMetrics()
	default:
		fmt.Fprintf(os.Stderr, "llm: unknown subcommand: %s\n", args[0])
		fmt.Fprintln(os.Stderr, "Usage: baxi-cli llm <status|metrics>")
		os.Exit(1)
	}
}

func llmStatus() {
	resp, err := apiGet("/api/v1/llm/status")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to call LLM status API: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if err := checkResponse(resp); err != nil {
		fmt.Fprintf(os.Stderr, "Error: LLM status API failed: %v\n", err)
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

func llmMetrics() {
	resp, err := apiGet("/api/v1/llm/metrics")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to call LLM metrics API: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if err := checkResponse(resp); err != nil {
		fmt.Fprintf(os.Stderr, "Error: LLM metrics API failed: %v\n", err)
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
