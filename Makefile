# =============================================================================
# TaskFlow — run from repo root (Docker Compose + backend Go module)
# =============================================================================
# Requires: Docker, Docker Compose v2, and (for local targets) Go 1.21+.
# Copy `.env.example` to `.env` before using Compose-backed targets.

COMPOSE ?= docker compose
IMAGE   ?= taskflow-api:latest
SERVICE_API ?= api
SERVICE_DB  ?= postgres

ifneq (,$(wildcard .env))
include .env
export
endif

.PHONY: build up down logs migrate-up migrate-down seed test

## build: Build the API Docker image (same Dockerfile as compose `api` service)
build:
	docker build -t $(IMAGE) -f backend/Dockerfile backend

## up: Start Postgres, run migrations once, then start the API
up:
	$(COMPOSE) up --build -d

## down: Stop stack (keeps named volume `postgres_data` unless you add `-v`)
down:
	$(COMPOSE) down

## logs: Follow all service logs (Ctrl+C to stop tailing)
logs:
	$(COMPOSE) logs -f

## migrate-up: Apply pending migrations manually (Compose run, one-shot)
migrate-up:
	$(COMPOSE) run --rm migrate up

## migrate-down: Roll back one migration version (run multiple times for N steps)
migrate-down:
	$(COMPOSE) run --rm migrate down 1

## seed: Load seed rows from migration 000002 (fails if data already exists)
seed:
	$(COMPOSE) exec -T $(SERVICE_DB) psql -U "$(POSTGRES_USER)" -d "$(POSTGRES_DB)" < backend/migrations/000002_seed_data.up.sql

## test: Run Go unit tests (host toolchain; does not require Docker)
test:
	cd backend && go test ./... -count=1
