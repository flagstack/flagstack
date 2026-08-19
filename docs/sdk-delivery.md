# SDK configuration delivery

FlagStack SDKs evaluate feature flags locally. The control plane supplies an environment-scoped configuration document; SDKs cache that document and apply the evaluation contract in [`evaluation.md`](evaluation.md) without making a network request for each flag evaluation.

## Credential types

SDK credentials always belong to exactly one environment.

### Server credentials

Server credentials are secrets and are intended for trusted backend processes, workers, CLI tools, and server-rendered applications.

A server key has the form:

```text
fs_server_<credential-id>.<secret>
```

The random secret contains 256 bits of entropy. FlagStack returns the complete key only when the credential is created. The database stores only a SHA-256 digest of the random secret; listing credentials never returns the secret again.

Because the secret is high-entropy random material, SHA-256 is used as a lookup-verification digest rather than a password hashing function. Verification is constant-time. Revoking the credential immediately prevents further configuration downloads.

Server credentials receive every active feature flag in their environment.

### Client credentials

Client credentials are deliberately **not secrets**. They are intended for browser/mobile/public-client applications where embedded credentials can always be recovered by an end user.

A client key has the form:

```text
fs_client_<public-id>
```

Client credentials only receive flags explicitly marked `client_visible`. New flags default to server-only.

Owners and administrators control client visibility. Marking a flag client-visible means the following configuration for that flag must be treated as public information:

- flag key and type;
- default value;
- enabled state;
- variants and their values;
- targeting rules and rollout percentages;
- any reusable segment definitions transitively referenced by those rules.

Applications must never place secrets, private credentials, sensitive customer lists, or other confidential values in configuration that is exposed to a client SDK.

## Configuration endpoint

SDKs fetch configuration from:

```http
GET /sdk/v1/config
Authorization: Bearer <sdk-key>
```

The endpoint does not use FlagStack browser sessions or cookies. Authentication is entirely through the environment-scoped SDK key.

Invalid, malformed, or revoked keys return `401 Unauthorized`.

Browser SDKs are supported directly. `/sdk/v1/config` permits cross-origin `GET` and `OPTIONS` requests and allows the `Authorization` and `If-None-Match` request headers. The management/session API remains same-origin and is not covered by this CORS policy.

## Schema version 1

The initial wire shape is:

```json
{
  "schema_version": 1,
  "environment": {
    "id": "019...",
    "key": "production"
  },
  "flags": [
    {
      "id": "019...",
      "key": "new-checkout",
      "kind": "boolean",
      "default_value": false,
      "enabled": true,
      "variants": [],
      "policy": {
        "rules": [],
        "fallthrough": {
          "rollout": [
            { "variant": "on", "weight": 10000 },
            { "variant": "off", "weight": 90000 }
          ]
        }
      },
      "revision": 3
    }
  ],
  "segments": []
}
```

The environment ID and flag ID are part of the deterministic bucketing input defined in `evaluation.md`; SDK implementations must preserve them exactly.

A missing sparse environment configuration is represented as disabled with revision `0`. Project-level defaults and variants are still included so the SDK can return the correct disabled/default value locally.

Only segments reachable from delivered flag policies are included. If a referenced segment itself references another segment, that dependency is included transitively. Unrelated project segments are omitted.

## Polling and cache validation

Configuration responses include a strong content-derived ETag:

```http
ETag: "sha256-..."
Cache-Control: private, max-age=0, must-revalidate
```

SDKs should retain the most recent successful configuration and send the ETag on subsequent polls:

```http
If-None-Match: "sha256-..."
```

If the environment configuration is unchanged, FlagStack responds with `304 Not Modified` and no response body.

A new ETag is produced whenever the delivered document changes, including changes to flag visibility, enablement, variants, targeting policy, or referenced segments. The per-flag `revision` remains useful for configuration/audit semantics, but SDKs must use the document ETag as the cache validator for the complete payload.

## Failure behaviour

SDK implementations should keep the most recent valid configuration when a transient refresh fails. A failed refresh must not erase already-loaded configuration.

The application-provided fallback value remains the final safety boundary when an SDK has never loaded a usable definition or encounters an unsupported future schema.

SDKs must reject schema versions they do not understand rather than silently interpreting them as an older format.

## Realtime delivery

Realtime invalidation is intentionally not part of schema version 1. Polling with ETag validation establishes the stable authentication and configuration contract first.

A later realtime transport can notify SDKs that their environment changed and trigger an ordinary conditional configuration refresh. The configuration document remains the source of truth, so adding SSE/WebSocket-style invalidation will not require changing evaluation semantics.
