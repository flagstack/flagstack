SHELL := /usr/bin/env bash

GOOSE := go run github.com/pressly/goose/v3/cmd/goose@v3.27.3
LOAD_ENV := set -a; if [ -f .env ]; then source .env; fi; set +a;

.PHONY: bootstrap ent-generate dev-backend dev-frontend infra-up infra-down db-up db-down db-status db-create fmt test check

bootstrap: ent-generate
	cd frontend && corepack enable && corepack prepare pnpm@11.22.0 --activate && pnpm install

ent-generate:
	cd backend && go generate ./ent

dev-backend: ent-generate
	$(LOAD_ENV) cd backend && go run ./cmd/flagstack

dev-frontend:
	cd frontend && pnpm dev

infra-up:
	docker compose up -d postgres

infra-down:
	docker compose down

db-up:
	$(LOAD_ENV) : "$${FLAGSTACK_DATABASE_URL:?FLAGSTACK_DATABASE_URL is required}"; cd backend && $(GOOSE) -dir migrations postgres "$$FLAGSTACK_DATABASE_URL" up

db-down:
	$(LOAD_ENV) : "$${FLAGSTACK_DATABASE_URL:?FLAGSTACK_DATABASE_URL is required}"; cd backend && $(GOOSE) -dir migrations postgres "$$FLAGSTACK_DATABASE_URL" down

db-status:
	$(LOAD_ENV) : "$${FLAGSTACK_DATABASE_URL:?FLAGSTACK_DATABASE_URL is required}"; cd backend && $(GOOSE) -dir migrations postgres "$$FLAGSTACK_DATABASE_URL" status

db-create:
	@test -n "$(name)" || (echo "Usage: make db-create name=describe_change"; exit 1)
	cd backend && $(GOOSE) -dir migrations create "$(name)" sql

fmt:
	cd backend && gofmt -w .

test: ent-generate
	cd backend && go test ./...

check: ent-generate
	@test -z "$$(gofmt -l backend)" || (echo "Go files need formatting:"; gofmt -l backend; exit 1)
	cd backend && go test ./...
	cd frontend && pnpm typecheck
	cd frontend && pnpm build
