# Core data model

FlagStack's persistence model establishes identity, tenancy, and the project/environment/flag hierarchy while leaving targeting and SDK evaluation features room to grow.

## Identity and organisations

`users` is the internal account/profile record. Email remains optional at the database layer so future OIDC or other external identities are not forced into an email-as-primary-identity model.

Local authentication data is separated from the profile record:

- `local_credentials` stores the local password hash for a user;
- `user_sessions` stores hashed session and CSRF tokens plus expiry metadata.

`organisations` are the top-level tenancy boundary. A user joins an organisation through `organisation_memberships` with one of four initial roles:

- `owner`
- `admin`
- `developer`
- `viewer`

Role constraints protect stored data integrity. Permission decisions remain an application-layer responsibility rather than being inferred directly from database rows in HTTP handlers.

## Projects and environments

A `project` belongs to exactly one organisation and represents an application or service integrating with FlagStack.

An `environment` belongs to exactly one project. Environment keys are unique within their project so common names such as `development`, `staging`, and `production` can be reused across projects.

Nested tables also carry `organisation_id`. PostgreSQL composite foreign keys enforce that project-scoped records cannot accidentally reference a resource belonging to another organisation.

## Feature flags

A `feature_flag` is defined at project scope. Its stable key is unique within that project.

The initial kinds are:

- `boolean`
- `string`
- `number`
- `json`

Each flag has a project-level `default_value`. PostgreSQL validates scalar default values against the declared kind.

Environment-specific state lives in `environment_flag_configs`, not in the flag definition itself. A configuration currently stores:

- whether the flag is enabled in that environment;
- an optional environment-specific value, falling back to the flag default when absent;
- a monotonically increasing revision number for future configuration delivery and audit semantics.

The configuration table references both the environment and flag through organisation/project-aware foreign keys. This guarantees that an environment can never be configured with a flag from another project or tenant.

Targeting rules, variants, percentage rollout, segments, SDK credentials, and audit history can extend this model without changing the core tenancy hierarchy.

## Identifiers

Ent entities use UUIDv7 primary keys. ORM-created rows generate UUIDv7 values in Go, while PostgreSQL 18 `uuidv7()` defaults remain on UUID primary-key columns so direct SQL and operational tooling get the same time-ordered identifier behaviour.

Join/configuration entities also have UUIDv7 primary keys. Their logical relationship keys remain unique constraints, for example `(organisation_id, user_id)` on memberships and `(environment_id, feature_flag_id)` on environment flag configurations.

## Lifecycle

Projects and feature flags support archival rather than immediate destructive deletion in normal product workflows. Foreign keys still use cascading deletion for true tenant/project deletion so a deliberate hard delete cannot leave orphaned configuration.

`created_at` and `updated_at` are stored as `timestamptz`. Ent provides creation/update defaults while application writes remain explicit and observable.

## Schema ownership

The Ent schema is the source of truth. The explicit migration command applies Ent's automatic PostgreSQL schema migration with destructive column and index drops disabled. PostgreSQL-specific tenant composite foreign keys are reconciled as part of that command so database-level tenant protection remains intact.
