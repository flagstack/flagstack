# Initial architecture

FlagStack starts as a modular monolith. That keeps self-hosting straightforward and keeps product boundaries explicit without paying the operational cost of distributed services before they are needed.

## Repository layout

- `backend/` — Go API, application logic, Ent schemas and persistence adapters.
- `frontend/` — React/TypeScript dashboard built with Vite and Tailwind CSS.
- `.devcontainer/` — reproducible development environment.
- `compose.yml` — local infrastructure dependencies.

## Backend principles

The backend is standard-library-first. Go's `net/http` and `log/slog` are sufficient for the initial transport and logging layers, so the scaffold does not add a framework solely for structure.

As domain work lands, dependencies should point inward: transport and persistence code may depend on application/domain contracts, while the domain must not depend on HTTP, PostgreSQL, or hosted FlagStack Cloud concerns.

## Frontend principles

The dashboard is a client-rendered React application. FlagStack does not currently need server-side rendering, so Vite keeps the frontend build and self-hosted deployment model smaller than a full-stack JavaScript framework.

Tailwind CSS is the frontend styling foundation. Routing, server-state management, and authentication libraries will be introduced with the workflows that require them rather than pre-selected in the scaffold.

## Persistence

PostgreSQL 18 is the system of record. Ent schemas under `backend/ent/schema` are the source of truth for users, credentials, sessions, organisations, memberships, projects, environments, feature flags, and environment-specific flag configuration.

The API keeps pgx for native PostgreSQL connectivity and pooling. Ent is layered over that same pgx pool, so the application does not maintain a second database connection pool.

Schema changes are applied by the explicit FlagStack migration command (`make db-up` or `make db-migrate`). The command runs Ent's migration engine with automatic destructive column/index drops disabled. Migrations are not run implicitly when the API starts.

A small PostgreSQL-specific migration reconciler preserves the organisation/project-aware composite foreign keys that enforce tenant boundaries beyond Ent's ordinary single-column relationship foreign keys. The migration command also upgrades databases created by the earlier Goose migrations and removes the old Goose version table after a successful transition.

PostgreSQL readiness is exposed separately from process liveness through `/readyz`.

See [`data-model.md`](data-model.md) for the tenancy and feature-configuration model.

## Cloud boundary

Core feature-management behaviour belongs in this repository. Billing, managed provisioning, usage metering, and other hosted-service-only concerns belong in the private `flagstack-cloud` repository.
