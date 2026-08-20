# Self-hosting FlagStack

FlagStack ships as a single application image containing the Go control plane, database migration command and compiled React dashboard. PostgreSQL remains the only required external service.

## Docker Compose quick start

Copy the example environment file and choose a PostgreSQL password before exposing the deployment outside a local machine:

```bash
cp .env.example .env
# edit FLAGSTACK_POSTGRES_PASSWORD in .env
make selfhost-up
```

The self-hosted Compose stack:

1. starts PostgreSQL 18 without publishing the database port;
2. waits for PostgreSQL readiness;
3. runs the explicit Ent migration command as a one-shot service;
4. starts the FlagStack application only after migrations succeed;
5. serves the dashboard and API from the same HTTP listener.

FlagStack is then available at `http://localhost:8080` by default. Set `FLAGSTACK_PORT` to publish a different host port.

Stop the stack with:

```bash
make selfhost-down
```

Add `-v` to the underlying Compose command only when you intentionally want to delete the PostgreSQL volume and all FlagStack data.

## Production image

Build the image directly with:

```bash
make image
```

or:

```bash
docker build -t flagstack/flagstack:local .
```

The image contains:

- `/app/flagstack` — the API and dashboard server;
- `/app/flagstack-migrate` — the explicit database migration command;
- `/app/frontend` — compiled Vite assets served by the Go application.

The runtime container runs as an unprivileged user. The application does **not** run schema migrations automatically at API startup; deployment systems should run `/app/flagstack-migrate` before rolling out a new application version.

## Required configuration

The application reads configuration from environment variables:

| Variable | Purpose | Default |
| --- | --- | --- |
| `FLAGSTACK_HTTP_ADDR` | HTTP listen address | `:8080` |
| `FLAGSTACK_DATABASE_URL` | PostgreSQL connection URL | required for the application |
| `FLAGSTACK_LOG_LEVEL` | structured log level | `info` |
| `FLAGSTACK_SESSION_TTL` | dashboard session lifetime | `168h` |
| `FLAGSTACK_SESSION_COOKIE_SECURE` | require HTTPS for the session cookie | `true` in the application; Compose sets `false` for local HTTP |
| `FLAGSTACK_STATIC_DIR` | compiled dashboard directory | empty for development; `/app/frontend` in the image |

The self-hosted Compose file additionally uses `FLAGSTACK_POSTGRES_PASSWORD` and `FLAGSTACK_PORT`.

## HTTPS and reverse proxies

For an internet-facing deployment, put FlagStack behind a TLS-terminating reverse proxy or ingress and set:

```text
FLAGSTACK_SESSION_COOKIE_SECURE=true
```

The dashboard uses same-origin session cookies and CSRF protection. SDK configuration endpoints use their own environment-scoped bearer credentials and do not rely on dashboard sessions.

Your proxy should preserve ordinary streaming responses and avoid caching `/api/` or `/sdk/` responses unless FlagStack's own response headers explicitly allow it.

## Health checks

- `/healthz` — process liveness only;
- `/readyz` — PostgreSQL readiness;
- `/api/v1/health` — API health response.

Use `/readyz` to decide whether a replica should receive traffic.

## Backups

PostgreSQL is the system of record. Back up the FlagStack database using normal PostgreSQL tooling such as `pg_dump`, physical backups or your managed PostgreSQL provider's backup service.

The application image and frontend assets are replaceable build artifacts; they do not need persistent storage.

## Upgrades

A safe upgrade sequence is:

1. back up PostgreSQL;
2. pull/build the new FlagStack image;
3. run `/app/flagstack-migrate` against the database;
4. start or roll the new application replicas;
5. verify `/readyz` and the dashboard.

Ent migrations have automatic destructive column and index drops disabled. Destructive migrations, when eventually needed, must be introduced deliberately rather than occurring as an API startup side effect.
