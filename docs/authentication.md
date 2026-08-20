# Authentication

FlagStack core provides self-host-friendly local authentication as the baseline identity path. Hosted or enterprise identity providers can be added later without making the core product depend on FlagStack Cloud.

## First-run bootstrap

A fresh database has no users. `GET /api/v1/bootstrap` reports whether initial setup is required. While bootstrap is required, `POST /api/v1/bootstrap` creates the first user, a local password credential, the first organisation, an `owner` membership, and an authenticated browser session in one PostgreSQL transaction.

The bootstrap transaction takes a PostgreSQL advisory transaction lock and re-checks for existing users before writing. This prevents concurrent setup requests from creating multiple first owners.

Once any user exists, bootstrap is permanently unavailable through the API.

## Local passwords

Local passwords are hashed with Argon2id. The encoded hash stores its own algorithm parameters and salt so parameters can be increased later without changing the credential schema.

Current parameters are:

- 64 MiB memory;
- 3 passes;
- 4 lanes;
- 16-byte random salt;
- 32-byte output.

Passwords must be at least 12 characters and are capped at 1024 bytes to avoid unbounded password-hashing work.

Login responses do not distinguish an unknown email address from an incorrect password. Unknown accounts still execute an Argon2id verification against a dummy hash to reduce account-enumeration timing differences.

## Browser sessions

Authenticated browser sessions use opaque 32-byte random tokens. The raw session token is sent only to the browser in an HttpOnly cookie; PostgreSQL stores only its SHA-256 digest. Database disclosure therefore does not directly reveal reusable session tokens.

Sessions expire after `FLAGSTACK_SESSION_TTL`, which defaults to seven days. Production deployments should leave `FLAGSTACK_SESSION_COOKIE_SECURE=true`; the repository `.env.example` disables it only for local HTTP development.

A second random CSRF token is issued in a non-HttpOnly cookie. Authenticated mutation requests must send the same token in the `X-CSRF-Token` header, and the API validates it against the server-side digest stored with the session.

## Current endpoints

- `GET /api/v1/bootstrap` — report whether first-run setup is required.
- `POST /api/v1/bootstrap` — create the first owner and organisation.
- `POST /api/v1/auth/login` — authenticate a local user and issue a session.
- `GET /api/v1/auth/me` — return the authenticated user and organisation memberships.
- `POST /api/v1/auth/logout` — revoke the current session; requires CSRF validation.

All management APIs for projects, environments, feature flags, targeting, segments, schedules and SDK credentials sit behind the same authenticated session boundary. Read-only requests require a valid organisation membership; state-changing requests also require CSRF validation and the appropriate organisation role.

## Roles and management permissions

The initial organisation roles are `owner`, `admin`, `developer`, and `viewer`.

The current management boundary is intentionally simple:

- `owner` and `admin` can create projects and manage SDK credentials/client visibility;
- `owner`, `admin`, and `developer` can manage environments, feature flags, variants, targeting policies, segments and schedules;
- `viewer` can inspect project configuration but cannot mutate it.

SDK configuration delivery does not use browser sessions. `/sdk/v1/config` and `/sdk/v1/events` authenticate independently with environment-scoped server/client SDK bearer credentials as described in [`sdk-delivery.md`](sdk-delivery.md).
