# Backend package boundaries

The backend follows a small set of package boundaries so the HTTP transport, application logic, domain rules, and persistence concerns do not collapse into one package as the product grows.

- `app` owns process startup and lifecycle orchestration.
- `config` owns environment-driven runtime configuration.
- `httpapi` owns HTTP routing and transport concerns.
- future domain packages should contain feature-management rules without depending on HTTP or database implementations.
- future storage packages should implement persistence behind interfaces owned by the domain/application layer that consumes them.

The initial scaffold intentionally uses Go's standard library for HTTP and logging. Third-party packages should be introduced when they solve a concrete requirement rather than by default.
