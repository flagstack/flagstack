# Core data model

FlagStack's first persistence model establishes tenancy and the project/environment/flag hierarchy before authentication and targeting rules are layered on top.

## Identity and organisations

`users` is the internal account/profile record. Email is optional at the database layer so future OIDC or other external identities are not forced into an email-as-primary-identity model. Authentication identities and credentials will be modelled separately when authentication is implemented.

`organisations` are the top-level tenancy boundary. A user joins an organisation through `organisation_memberships` with one of four initial roles:

- `owner`
- `admin`
- `developer`
- `viewer`

These role names establish the data model only. Permission behaviour will be defined with the authentication/authorization application layer rather than inferred directly from database rows in HTTP handlers.

## Projects and environments

A `project` belongs to exactly one organisation and represents an application or service integrating with FlagStack.

An `environment` belongs to exactly one project. Environment keys are unique within their project so common names such as `development`, `staging`, and `production` can be reused across projects.

Nested tables also carry `organisation_id`. Composite foreign keys enforce that project-scoped records cannot accidentally reference a resource belonging to another organisation.

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
- an optional environment-specific value (falling back to the flag default when absent);
- a monotonically increasing revision number for future configuration delivery and audit semantics.

The configuration table references both the environment and flag through organisation/project-aware foreign keys. This guarantees that an environment can never be configured with a flag from another project or tenant.

Targeting rules, variants, percentage rollout, segments, SDK credentials, and audit history are deliberately not encoded in this first migration. They can extend the environment configuration model without changing the tenancy hierarchy.

## Identifiers

Primary keys use PostgreSQL 18's native `uuidv7()` generation. UUIDv7 keeps identifiers globally unique while making newly generated identifiers time-ordered, which is friendlier to indexes than fully random UUIDv4 values.

IDs are database-generated so the Go domain layer does not need a UUID generation dependency solely for persistence identity.

## Lifecycle

Projects and feature flags support archival rather than immediate destructive deletion in normal product workflows. Foreign keys still use cascading deletion for true tenant/project deletion so a deliberate hard delete cannot leave orphaned configuration.

`created_at` and `updated_at` are stored as `timestamptz`. Application services are responsible for updating `updated_at` as part of writes; the database does not hide that behaviour behind triggers.
