# Initial architecture

FlagStack starts as a modular monolith. That keeps self-hosting straightforward and keeps product boundaries explicit without paying the operational cost of distributed services before they are needed.

## Repository layout

- `backend/` — Go API and application logic.
- `frontend/` — React/TypeScript dashboard built with Vite.
- `backend/migrations/` — PostgreSQL schema migrations.
- `.devcontainer/` — reproducible development environment.
- `compose.yml` — local infrastructure dependencies.

## Backend principles

The backend is standard-library-first. Go's `net/http` and `log/slog` are sufficient for the initial transport and logging layers, so the scaffold does not add a framework solely for structure.

As domain work lands, dependencies should point inward: transport and persistence code may depend on application/domain contracts, while the domain must not depend on HTTP, PostgreSQL, or hosted FlagStack Cloud concerns.

## Frontend principles

The dashboard is a client-rendered React application. FlagStack does not currently need server-side rendering, so Vite keeps the frontend build and self-hosted deployment model smaller than a full-stack JavaScript framework.

Routing, server-state management, design-system dependencies, and authentication libraries will be introduced with the workflows that require them rather than pre-selected in the scaffold.

## Persistence

PostgreSQL is the system of record. The first schema migration will define the organisation, membership, project, environment, and feature-flag model together so foreign-key and tenancy boundaries are designed as one coherent unit.

## Cloud boundary

Core feature-management behaviour belongs in this repository. Billing, managed provisioning, usage metering, and other hosted-service-only concerns belong in the private `flagstack-cloud` repository.
