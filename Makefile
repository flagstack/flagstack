SHELL := /usr/bin/env bash

.PHONY: bootstrap dev-backend dev-frontend infra-up infra-down fmt test check

bootstrap:
	cd frontend && corepack enable && corepack prepare pnpm@11.22.0 --activate && pnpm install

dev-backend:
	cd backend && go run ./cmd/flagstack

dev-frontend:
	cd frontend && pnpm dev

infra-up:
	docker compose up -d postgres

infra-down:
	docker compose down

fmt:
	cd backend && gofmt -w .

test:
	cd backend && go test ./...

check:
	@test -z "$$(gofmt -l backend)" || (echo "Go files need formatting:"; gofmt -l backend; exit 1)
	cd backend && go test ./...
	cd frontend && pnpm typecheck
	cd frontend && pnpm build
