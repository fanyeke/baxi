.PHONY: up down migrate api worker test fmt build vet tidy restart

DATABASE_URL ?= postgres://baxi:baxi_dev@localhost:5432/baxi?sslmode=disable

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

# Testing and quality
test:
	go test ./... -v -count=1 2>&1

vet:
	go vet ./...

fmt:
	go fmt ./...

tidy:
	go mod tidy
