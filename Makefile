SHELL := /usr/bin/env bash

LOAD_ENV := set -a; if [ -f .env ]; then source .env; fi; set +a;

.PHONY: bootstrap ent-generate dev-backend dev-frontend infra-up infra-down selfhost-up selfhost-down image db-up db-migrate fmt test check

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

selfhost-up:
	docker compose -f compose.selfhost.yml up -d --build

selfhost-down:
	docker compose -f compose.selfhost.yml down

image:
	docker build -t flagstack/flagstack:local .

db-up: db-migrate

db-migrate: ent-generate
	$(LOAD_ENV) : "$${FLAGSTACK_DATABASE_URL:?FLAGSTACK_DATABASE_URL is required}"; cd backend && go run ./cmd/flagstack-migrate

fmt:
	cd backend && gofmt -w .

test: ent-generate
	cd backend && go test ./...

check: ent-generate
	@test -z "$$(gofmt -l backend)" || (echo "Go files need formatting:"; gofmt -l backend; exit 1)
	cd backend && go mod tidy -diff
	cd backend && go test ./...
	cd frontend && pnpm typecheck
	cd frontend && pnpm build
