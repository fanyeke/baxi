.PHONY: up down migrate api worker test fmt build vet tidy restart pipeline pipeline-ingest pipeline-dwd pipeline-metrics pipeline-compare api-compare test-pipeline governance-load governance-check test-governance test-governance-integration

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

api-compare:  ## Compare Go API responses against baseline snapshots
	python3 scripts/migration/compare_api_baseline.py

test-pipeline:  ## Run pipeline tests
	go test ./internal/pipeline/... ./internal/ingest/... ./internal/alert/... ./internal/recommendation/... ./internal/outbox/... -short -count=1

# Governance
governance-load:  ## Load YAML governance configs into gov.* tables
	go run ./cmd/baxi-cli governance load --config-dir ./config

governance-check:  ## Verify governance configs are properly loaded
	go run ./cmd/baxi-cli governance check

test-governance:  ## Run governance-related tests
	go test ./internal/configloader/... ./internal/ontology/... ./internal/governance/... -v -count=1

test-governance-integration:  ## Run governance integration tests (requires testcontainers)
	go test -tags integration ./internal/repository/... ./internal/governance/... ./internal/ontology/... -v -count=1

# Testing and quality
test:
	go test ./... -v -count=1 2>&1

vet:
	go vet ./...

fmt:
	go fmt ./...

tidy:
	go mod tidy
