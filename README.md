# FlagStack

FlagStack is a source-available, self-hostable feature management platform focused on a strong developer experience, predictable infrastructure, and avoiding architecture-based pricing surprises.

> **Status:** Early development. FlagStack is not yet ready for production use.

## Goals

- Simple feature flag management for teams and organisations.
- First-class self-hosting with a straightforward Docker-based setup.
- A hosted FlagStack Cloud option for teams that do not want to operate it themselves.
- Local flag evaluation in SDKs so application requests do not depend on FlagStack being available.
- Real-time configuration updates to connected SDKs.
- Strong OpenFeature support to reduce vendor lock-in.
- Predictable, developer-friendly pricing for the hosted service.

## Repository

The repository is organised as a modular monolith:

- **`backend/`** — Go API and core application services.
- **`frontend/`** — React, TypeScript and Tailwind CSS dashboard built with Vite.
- **`backend/migrations/`** — PostgreSQL schema migrations managed with Goose.
- **`.devcontainer/`** — reproducible development environment.
- **`compose.yml`** — local infrastructure dependencies.
- **`docs/`** — architecture and development documentation.

See [`docs/architecture.md`](docs/architecture.md) for the initial architecture and package-boundary decisions and [`docs/data-model.md`](docs/data-model.md) for the core tenancy model.

## Development

The recommended development environment is the repository dev container. It includes Go, Node.js, and Docker tooling.

Create the local environment file, install dependencies, start PostgreSQL, and apply migrations:

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

Use `make db-status`, `make db-up`, and `make db-down` to manage schema migrations. Create a migration with `make db-create name=describe_change`.

## SDKs

Official SDKs live in separate repositories in the [FlagStack GitHub organisation](https://github.com/flagstack):

- [`sdk-python`](https://github.com/flagstack/sdk-python)
- [`sdk-js`](https://github.com/flagstack/sdk-js)
- [`sdk-go`](https://github.com/flagstack/sdk-go)
- [`sdk-dotnet`](https://github.com/flagstack/sdk-dotnet)

Additional SDKs may be added as the project grows.

## FlagStack Cloud

The core FlagStack product is intended to remain genuinely useful when self-hosted. FlagStack Cloud will provide a managed version for teams that prefer not to run the infrastructure themselves.

Cloud-specific services such as billing, provisioning and internal operations are maintained separately from the public core.

The Elastic License 2.0 permits self-hosting and modification while restricting use of FlagStack itself as a competing hosted or managed service. See `LICENSE` for details and the authoritative licence terms.

## Contributing

Contributions are welcome. Organisation-wide contribution guidelines, commit conventions, security policy, governance, and pull request requirements are maintained in the [`flagstack/.github`](https://github.com/flagstack/.github) repository.

FlagStack maintains a linear Git history. Pull requests are integrated by **rebase only**; merge commits and squash merges are not used.

External contributions are subject to the FlagStack Contributor Licence Agreement policy.

## Licence

FlagStack core is licensed under the **Elastic License 2.0 (ELv2)**. See [`LICENSE`](LICENSE).

The official client SDKs are licensed separately under the Apache License 2.0.
