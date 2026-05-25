.PHONY: up down migrate api worker test fmt build vet tidy restart pipeline pipeline-ingest pipeline-dwd pipeline-metrics pipeline-compare test-pipeline

DATABASE_URL ?= postgres://baxi:baxi_dev@localhost:5432/baxi?sslmode=disable
DATA_DIR ?= ./data/raw

# Docker
up:
	docker compose up -d postgres

down:
	docker compose down

restart:
	docker compose restart

# Migration
migrate:
	goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	goose -dir migrations postgres "$(DATABASE_URL)" down

migrate-status:
	goose -dir migrations postgres "$(DATABASE_URL)" status

# Go binaries
api:
	go run ./cmd/baxi-api

worker:
	go run ./cmd/baxi-worker

build:
	go build -o baxi-api ./cmd/baxi-api
	go build -o baxi-worker ./cmd/baxi-worker

# Pipeline
pipeline:  ## Run full data pipeline
	go run ./cmd/baxi-cli pipeline run --data-dir $(DATA_DIR)

pipeline-ingest:  ## Run ingest step only
	go run ./cmd/baxi-cli pipeline run --step ingest_raw --data-dir $(DATA_DIR)

pipeline-dwd:  ## Run DWD build steps
	go run ./cmd/baxi-cli pipeline run --step build_dwd_order_level --data-dir $(DATA_DIR) && \
	go run ./cmd/baxi-cli pipeline run --step build_dwd_item_level --data-dir $(DATA_DIR)

pipeline-metrics:  ## Run metric build steps
	go run ./cmd/baxi-cli pipeline run --step build_metric_daily --data-dir $(DATA_DIR) && \
	go run ./cmd/baxi-cli pipeline run --step build_metric_dimension_daily --data-dir $(DATA_DIR)

pipeline-compare:  ## Compare output against baseline
	go run ./cmd/baxi-cli pipeline validate --data-dir $(DATA_DIR)

test-pipeline:  ## Run pipeline tests
	go test ./internal/pipeline/... ./internal/ingest/... ./internal/alert/... ./internal/recommendation/... ./internal/outbox/... -short -count=1

# Testing and quality
test:
	go test ./... -v -count=1 2>&1

vet:
	go vet ./...

fmt:
	go fmt ./...

tidy:
	go mod tidy
