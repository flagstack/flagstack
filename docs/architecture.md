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

Tailwind CSS is the frontend styling foundation. React Router provides declarative application routing; additional frontend dependencies are introduced only when a product workflow justifies them.

## Persistence

PostgreSQL 18 is the system of record. Ent schemas under `backend/ent/schema` are the source of truth for users, credentials, sessions, organisations, memberships, projects, environments, feature flags, reusable segments, environment-specific flag configuration, scheduled flag changes, and SDK credentials.

The API keeps pgx for native PostgreSQL connectivity and pooling. Ent is layered over that same pgx pool, so the application does not maintain a second database connection pool.

Schema changes are applied by the explicit FlagStack migration command (`make db-up` or `make db-migrate`). The command runs Ent's migration engine with automatic destructive column/index drops disabled. Migrations are not run implicitly when the API starts.

A small PostgreSQL-specific migration reconciler preserves the organisation/project-aware composite foreign keys that enforce tenant boundaries beyond Ent's ordinary single-column relationship foreign keys. The migration command also upgrades databases created by the earlier Goose migrations and removes the old Goose version table after a successful transition.

PostgreSQL readiness is exposed separately from process liveness through `/readyz`.

See [`data-model.md`](data-model.md) for persistence and tenancy details.

## Evaluation boundary

Flag definitions remain project-scoped while enablement and targeting policy remain environment-scoped. The reference evaluator in `backend/internal/evaluation` defines the contract that SDKs reproduce locally.

The evaluation model supports:

- boolean, string, number and JSON values;
- named variants;
- ordered first-match targeting rules;
- arbitrary application-supplied evaluation context;
- reusable project segments;
- deterministic percentage and multivariate rollout;
- OpenFeature-style resolution metadata and safe fallback behaviour.

Percentage assignment is deterministic and keyed by stable flag/environment identity plus the evaluation subject. This keeps cohorts stable across requests and during progressive rollout increases.

See [`evaluation.md`](evaluation.md) for the normative evaluation and cross-SDK bucketing specification.

## SDK configuration delivery

SDKs do not call FlagStack for every flag evaluation. They download an environment-scoped schema-v1 configuration document and evaluate locally using the same contract as the reference Go evaluator.

Two credential types keep trusted-server and public-client use cases separate:

- `server` credentials contain a high-entropy secret. FlagStack returns the full value once and stores only its digest. Server credentials receive all active flags in their environment.
- `client` credentials are public identifiers suitable for browser/mobile applications. They receive only flags explicitly marked client-visible.

Client visibility defaults to false and is controlled by organisation owners/admins. This is a security boundary: a client-visible flag's values, variants, targeting rules, rollout configuration and referenced segment definitions must be assumed public.

SDK configuration is served from `/sdk/v1/config` using bearer authentication. Responses use a content-derived ETag and conditional `If-None-Match` requests so polling can cheaply return `304 Not Modified`. The SDK endpoint has a narrow CORS policy for browser clients; the authenticated dashboard/management API remains same-origin.

Realtime transport is intentionally deferred. A future invalidation channel can tell an SDK to perform the same conditional refresh without changing the configuration or evaluation contracts.

See [`sdk-delivery.md`](sdk-delivery.md) for the normative credential and wire-format contract.

## Scheduling

Scheduled flag changes are a control-plane concern. Each API replica runs a small scheduler loop and competes for due work through conditional PostgreSQL claims; no Redis, message broker, or separate worker dependency is required for the initial implementation.

Claims use a bounded lease. A replica that dies after claiming a change cannot strand it indefinitely: another replica may reclaim the work after the lease expires. Applying a claimed change revalidates the current environment, active flag, variants, segments, and policy before writing a new configuration revision.

Schedules can change enablement or apply a targeting policy. Multiple future policy changes can therefore implement progressive rollout schedules without adding special evaluation semantics.

## Cloud boundary

Core feature-management behaviour belongs in this repository. Billing, managed provisioning, usage metering, and other hosted-service-only concerns belong in the private `flagstack-cloud` repository.
