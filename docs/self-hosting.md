# Self-hosting Switch On Your Code

Switch On Your Code ships as a single application image containing the Go control plane, database migration command and compiled React dashboard. PostgreSQL remains the only required external service.

## Docker Compose quick start

Copy the example environment file and choose a PostgreSQL password before exposing the deployment outside a local machine:

```bash
cp .env.example .env
# edit SWITCHONYOURCODE_POSTGRES_PASSWORD in .env
make selfhost-up
```

The self-hosted Compose stack:

1. starts PostgreSQL 18 without publishing the database port;
2. waits for PostgreSQL readiness;
3. runs the explicit Ent migration command as a one-shot service;
4. starts the Switch On Your Code application only after migrations succeed;
5. serves the dashboard and API from the same HTTP listener.

Switch On Your Code is then available at `http://localhost:8080` by default. Set `SWITCHONYOURCODE_PORT` to publish a different host port.

Stop the stack with:

```bash
make selfhost-down
```

Add `-v` to the underlying Compose command only when you intentionally want to delete the PostgreSQL volume and all Switch On Your Code data.

## Production image

Build the image directly with:

```bash
make image
```

or:

```bash
docker build -t switchonyourcode/switchonyourcode:local .
```

The image contains:

- `/app/switchonyourcode` — the API and dashboard server;
- `/app/switchonyourcode-migrate` — the explicit database migration command;
- `/app/frontend` — compiled Vite assets served by the Go application.

The runtime container runs as an unprivileged user. The application does **not** run schema migrations automatically at API startup; deployment systems should run `/app/switchonyourcode-migrate` before rolling out a new application version.

## Required configuration

The application reads configuration from environment variables:

| Variable | Purpose | Default |
| --- | --- | --- |
| `SWITCHONYOURCODE_HTTP_ADDR` | HTTP listen address | `:8080` |
| `SWITCHONYOURCODE_DATABASE_URL` | PostgreSQL connection URL | required for the application |
| `SWITCHONYOURCODE_LOG_LEVEL` | structured log level | `info` |
| `SWITCHONYOURCODE_SESSION_TTL` | dashboard session lifetime | `168h` |
| `SWITCHONYOURCODE_SESSION_COOKIE_SECURE` | require HTTPS for the session cookie | `true` in the application; Compose sets `false` for local HTTP |
| `SWITCHONYOURCODE_STATIC_DIR` | compiled dashboard directory | empty for development; `/app/frontend` in the image |

The self-hosted Compose file additionally uses `SWITCHONYOURCODE_POSTGRES_PASSWORD` and `SWITCHONYOURCODE_PORT`.

## HTTPS and reverse proxies

For an internet-facing deployment, put Switch On Your Code behind a TLS-terminating reverse proxy or ingress and set:

```text
SWITCHONYOURCODE_SESSION_COOKIE_SECURE=true
```

The dashboard uses same-origin session cookies and CSRF protection. SDK configuration endpoints use their own environment-scoped bearer credentials and do not rely on dashboard sessions.

Your proxy should preserve ordinary streaming responses and avoid caching `/api/` or `/sdk/` responses unless Switch On Your Code's own response headers explicitly allow it.

## Health checks

- `/healthz` — process liveness only;
- `/readyz` — PostgreSQL readiness;
- `/api/v1/health` — API health response.

Use `/readyz` to decide whether a replica should receive traffic.

## Backups

PostgreSQL is the system of record. Back up the Switch On Your Code database using normal PostgreSQL tooling such as `pg_dump`, physical backups or your managed PostgreSQL provider's backup service.

The application image and frontend assets are replaceable build artifacts; they do not need persistent storage.

## Upgrades

A safe upgrade sequence is:

1. back up PostgreSQL;
2. pull/build the new Switch On Your Code image;
3. run `/app/switchonyourcode-migrate` against the database;
4. start or roll the new application replicas;
5. verify `/readyz` and the dashboard.

Ent migrations have automatic destructive column and index drops disabled. Destructive migrations, when eventually needed, must be introduced deliberately rather than occurring as an API startup side effect.
