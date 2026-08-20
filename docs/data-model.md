# Core data model

Switch On Your Code's persistence model establishes identity, tenancy, and the project/environment/flag hierarchy while keeping evaluation configuration local to those boundaries.

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

A `project` belongs to exactly one organisation and represents an application or service integrating with Switch On Your Code.

An `environment` belongs to exactly one project. Environment keys are unique within their project so common names such as `development`, `staging`, and `production` can be reused across projects.

Nested tables also carry `organisation_id`. PostgreSQL composite foreign keys enforce that project-scoped records cannot accidentally reference a resource belonging to another organisation.

## Feature flags

A `feature_flag` is defined at project scope. Its stable key is unique within that project.

The supported kinds are:

- `boolean`
- `string`
- `number`
- `json`

Each flag has a project-level `default_value`. PostgreSQL validates scalar default values against the declared kind.

Named `variants` also live on the project-level flag definition. Their values must match the flag kind and are used by environment rules and deterministic allocations. Boolean evaluation additionally reserves built-in `on`, `off`, and `default` variants.

`client_visible` is a project-level delivery attribute. It defaults to false. Server SDK credentials can receive every active flag; client SDK credentials can receive only active flags with `client_visible = true`. Owners and administrators control this exposure because the delivered definition, values, policy and referenced segments must be considered public once a browser/mobile SDK can fetch them.

Environment-specific state lives in `environment_flag_configs`, not in the flag definition itself. A configuration stores:

- whether the flag is enabled in that environment;
- an optional environment-specific value, falling back to the flag default when absent;
- an evaluation `policy` containing ordered targeting rules and the fallthrough outcome;
- a monotonically increasing revision number for configuration delivery and audit semantics.

Environment configuration remains sparse. A missing `(environment, flag)` row means disabled with the project-level default. A row is created only when environment state or targeting policy needs to be persisted.

The configuration table references both the environment and flag through organisation/project-aware foreign keys. This guarantees that an environment can never be configured with a flag from another project or tenant.

## Segments

`segments` are reusable project-scoped targeting definitions. A segment stores a stable key, display metadata, `all`/`any` matching semantics, and typed condition JSON using the same operators as flag rules.

Segments do not contain a synchronized copy of application users. Applications provide evaluation context at runtime; segments describe how those attributes should be matched.

Segments may reference other segments. The application validates references and rejects dependency cycles before persistence.

SDK configuration delivery includes only segments transitively referenced by the flags delivered to that credential. Unrelated segments are omitted from the wire document.

## SDK credentials

`sdk_credentials` are environment-scoped configuration-delivery identities. Each row also carries organisation and project IDs, and PostgreSQL enforces the full organisation/project/environment relationship with a composite foreign key.

Two kinds are supported:

### Server

A server credential stores:

- an immutable UUIDv7 credential ID;
- display name and environment scope;
- `kind = 'server'`;
- a 32-byte SHA-256 digest of the generated high-entropy secret;
- no client key;
- optional revocation timestamp.

The full server key is returned only during creation. It is never stored in recoverable form.

### Client

A client credential stores:

- the same tenant/environment identity and metadata;
- `kind = 'client'`;
- a unique public client key;
- no secret digest;
- optional revocation timestamp.

Client keys are identifiers, not secrets. Their security boundary is the per-flag `client_visible` filter rather than concealment of the key itself.

Database checks ensure a credential cannot simultaneously contain server and client key material.

## Scheduled changes

`scheduled_flag_changes` stores durable future control-plane operations for a particular project, environment and feature flag.

A scheduled change contains:

- `execute_at` as an absolute `timestamptz`;
- a JSON patch that changes enablement and/or environment policy;
- lifecycle status (`pending`, `running`, `executed`, `cancelled`, or `failed`);
- optional creator identity;
- an execution claim token and claim timestamp;
- execution timestamp and failure detail.

Due work is claimed conditionally in PostgreSQL. Claims have a bounded lease, allowing another Switch On Your Code replica to reclaim work left `running` when a process dies. The claim token prevents the stale worker from later committing the same scheduled change.

Multiple scheduled policy patches can implement a progressive rollout without adding scheduler-specific evaluation behaviour.

## Identifiers

Ent entities use UUIDv7 primary keys. ORM-created rows generate UUIDv7 values in Go, while PostgreSQL 18 `uuidv7()` defaults remain on UUID primary-key columns so direct SQL and operational tooling get the same time-ordered identifier behaviour.

Join/configuration entities also have UUIDv7 primary keys. Their logical relationship keys remain unique constraints, for example `(organisation_id, user_id)` on memberships and `(environment_id, feature_flag_id)` on environment flag configurations.

## Lifecycle

Projects, feature flags, and segments support archival rather than immediate destructive deletion in normal product workflows. SDK credentials use explicit revocation: a revoked credential remains visible for operational history but is rejected for configuration authentication.

Foreign keys still use cascading deletion for true tenant/project/environment deletion so a deliberate hard delete cannot leave orphaned configuration or credentials.

`created_at` and `updated_at` are stored as `timestamptz`. Ent provides creation/update defaults while application writes remain explicit and observable.

## Schema ownership

The Ent schema is the source of truth. The explicit migration command applies Ent's automatic PostgreSQL schema migration with destructive column and index drops disabled. PostgreSQL-specific tenant composite foreign keys are reconciled as part of that command so database-level tenant protection remains intact.

See [`evaluation.md`](evaluation.md) for the normative rule, segment, variant, rollout and context-evaluation contract, and [`sdk-delivery.md`](sdk-delivery.md) for SDK credential and configuration-delivery semantics.
