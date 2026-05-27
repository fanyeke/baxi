.PHONY: up down migrate api worker test fmt build vet tidy restart pipeline pipeline-ingest pipeline-dwd pipeline-metrics pipeline-compare api-compare test-pipeline governance-load governance-check test-governance test-governance-integration decision-create decision-context decision-decide decision-list decision-compare decision-replay decision-evals llm-status llm-metrics backup restore rollback verify-phase1 verify-phase2 verify-phase3 verify-phase4 verify-phase5 verify-all

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

# Backup and Restore
backup:  ## Create PostgreSQL backup (compressed)
	./scripts/backup/backup_pg.sh --output-dir ./backups

restore:  ## Restore PostgreSQL from backup (requires FILE= argument)
	@if [ -z "$(FILE)" ]; then echo "ERROR: FILE argument required. Usage: make restore FILE=./backups/baxi_YYYYMMDD_HHMMSS.sql.gz"; exit 1; fi
	./scripts/backup/restore_pg.sh "$(FILE)"

rollback:  ## Roll back specific migration (requires MIGRATION= argument)
	@if [ -z "$(MIGRATION)" ]; then echo "ERROR: MIGRATION argument required. Usage: make rollback MIGRATION=5"; exit 1; fi
	./scripts/rollback/rollback_phase.sh "$(MIGRATION)"

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

# Decision CLI
decision-create:
	go run ./cmd/baxi-cli decision create --alert-id $(ALERT_ID)

decision-context:
	go run ./cmd/baxi-cli decision context --case-id $(CASE_ID)

decision-decide:
	go run ./cmd/baxi-cli decision decide --case-id $(CASE_ID)

decision-list:
	go run ./cmd/baxi-cli decision list

decision-compare:  ## Compare decision case versions
	go run ./cmd/baxi-cli decision compare --case-id $(CASE_ID)

decision-replay:  ## Replay a decision case
	go run ./cmd/baxi-cli decision replay --case-id $(CASE_ID) --dry-run=true

decision-evals:  ## Evaluate a decision case
	go run ./cmd/baxi-cli decision evals --case-id $(CASE_ID)

# LLM commands
llm-status:  ## Show LLM provider status
	go run ./cmd/baxi-cli llm status

llm-metrics:  ## Show LLM usage metrics
	go run ./cmd/baxi-cli llm metrics

# Migration Verification
verify-phase1:  ## Verify Phase 1: Parallel Run (both APIs + new tables)
	./scripts/verification/verify_phase1.sh

verify-phase2:  ## Verify Phase 2: Go-Primary Read (frontend + shadow mode)
	./scripts/verification/verify_phase2.sh

verify-phase3:  ## Verify Phase 3: Dual-Write (both APIs + consistency)
	./scripts/verification/verify_phase3.sh

verify-phase4:  ## Verify Phase 4: Go-Primary Write (Go writes + Python read-only)
	./scripts/verification/verify_phase4.sh

verify-phase5:  ## Verify Phase 5: Python Sunset (Go only + frontend normal)
	./scripts/verification/verify_phase5.sh

verify-all: verify-phase1 verify-phase2 verify-phase3 verify-phase4 verify-phase5  ## Run all verification phases

# Testing and quality
test:
	go test ./... -v -count=1 2>&1

test-go:
	go test ./... -count=1

test-python:
	pytest tests -q --timeout=120

test-frontend:
	npm --prefix frontend test -- --run || true

test-all: test-go test-python test-frontend

vet:
	go vet ./...

lint:
	echo "=== Go vet ===" && go vet ./...
	@echo ""
	echo "=== Python Ruff ===" && ruff check api services adapters core

fmt:
	go fmt ./...
	ruff format api services adapters core 2>/dev/null || true

tidy:
	go mod tidy
