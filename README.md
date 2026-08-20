# Switch On Your Code

Switch On Your Code is a source-available, self-hostable feature management platform focused on a strong developer experience, predictable infrastructure, and avoiding architecture-based pricing surprises.

> **Status:** Early development. Switch On Your Code is not yet ready for production use.

## Goals

- Simple feature flag management for teams and organisations.
- First-class self-hosting with a straightforward Docker-based setup.
- A hosted Switch On Your Code Cloud option for teams that do not want to operate it themselves.
- Local flag evaluation in SDKs so application requests do not depend on Switch On Your Code being available.
- Real-time configuration updates to connected SDKs.
- Strong OpenFeature support to reduce vendor lock-in.
- Predictable, developer-friendly pricing for the hosted service.

## Repository

The repository is organised as a modular monolith:

- **`backend/`** — Go API, application services, Ent schemas and persistence.
- **`frontend/`** — React, TypeScript and Tailwind CSS dashboard built with Vite.
- **`.devcontainer/`** — reproducible development environment.
- **`compose.yml`** — local development infrastructure.
- **`compose.selfhost.yml`** — complete self-hosted Switch On Your Code + PostgreSQL stack.
- **`docs/`** — architecture and development documentation.

See [`docs/architecture.md`](docs/architecture.md) for the initial architecture and package-boundary decisions and [`docs/data-model.md`](docs/data-model.md) for the core tenancy model.

## Development

The recommended development environment is the repository dev container. It includes Go, Node.js, and Docker tooling.

Create the local environment file, install dependencies, start PostgreSQL, and bring the Ent-managed schema up to date:

```bash
cp .env.example .env
make bootstrap
make infra-up
make db-up
```

Run the API and frontend in separate terminals:

```bash
make dev-backend
make dev-frontend
```

The API listens on `http://localhost:8080`. `/healthz` is process liveness, `/readyz` verifies PostgreSQL connectivity, and `/api/v1/health` is the API health endpoint. The frontend development server listens on `http://localhost:5173` and proxies `/api` requests to the API.

Run the current checks with:

```bash
make check
```

`make db-up` and `make db-migrate` both run the explicit Ent migration command. Switch On Your Code does not automatically mutate the database schema when the API starts. Automatic destructive column and index drops are disabled; destructive schema changes must be handled deliberately when they are required.

## Self-hosting

Switch On Your Code now builds as one production application image containing both the Go control plane and compiled React dashboard. PostgreSQL is the only required external service.

For a local self-hosted stack:

```bash
cp .env.example .env
# Change SWITCHONYOURCODE_POSTGRES_PASSWORD before exposing the deployment externally.
make selfhost-up
```

The Compose stack starts PostgreSQL, runs the explicit Ent migrations, and only then starts Switch On Your Code on `http://localhost:8080`. The database port is not exposed by the self-hosted Compose file.

For an internet-facing deployment, terminate TLS in front of Switch On Your Code and set `SWITCHONYOURCODE_SESSION_COOKIE_SECURE=true`.

See [`docs/self-hosting.md`](docs/self-hosting.md) for image layout, environment variables, reverse-proxy guidance, health checks, backups and upgrades.

## SDKs

Official SDKs live in separate repositories in the [Switch On Your Code GitHub organisation](https://github.com/switchonyourcode):

- [`sdk-python`](https://github.com/switchonyourcode/sdk-python)
- [`sdk-js`](https://github.com/switchonyourcode/sdk-js)
- [`sdk-go`](https://github.com/switchonyourcode/sdk-go)
- [`sdk-dotnet`](https://github.com/switchonyourcode/sdk-dotnet)

Additional SDKs may be added as the project grows.

## Switch On Your Code Cloud

The core Switch On Your Code product is intended to remain genuinely useful when self-hosted. Switch On Your Code Cloud will provide a managed version for teams that prefer not to run the infrastructure themselves.

Cloud-specific services such as billing, provisioning and internal operations are maintained separately from the public core.

The Elastic License 2.0 permits self-hosting and modification while restricting use of Switch On Your Code itself as a competing hosted or managed service. See `LICENSE` for details and the authoritative licence terms.

## Contributing

Contributions are welcome. Organisation-wide contribution guidelines, commit conventions, security policy, governance, and pull request requirements are maintained in the [`switchonyourcode/.github`](https://github.com/switchonyourcode/.github) repository.

Switch On Your Code maintains a linear Git history. Pull requests are integrated by **rebase only**; merge commits and squash merges are not used.

External contributions are subject to the Switch On Your Code Contributor Licence Agreement policy.

## Licence

Switch On Your Code core is licensed under the **Elastic License 2.0 (ELv2)**. See [`LICENSE`](LICENSE).

The official client SDKs are licensed separately under the Apache License 2.0.
