# FlagStack

FlagStack is an open-source, self-hostable feature management platform focused on a strong developer experience, predictable infrastructure, and avoiding architecture-based pricing surprises.

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

This repository will contain the main open-source FlagStack application, including:

- **Backend** — Go API and core services.
- **Frontend** — web dashboard for managing organisations, projects, environments and feature flags.
- **Self-hosting** — Docker Compose and deployment configuration.
- **Database** — schema, migrations and persistence code.
- **Realtime delivery** — configuration distribution to FlagStack SDKs.

The exact structure will evolve as development begins.

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

## Contributing

FlagStack is at a very early stage. Contribution documentation, development setup instructions and project conventions will be added as the initial architecture settles.

## License

A project license will be selected before the first public release.
