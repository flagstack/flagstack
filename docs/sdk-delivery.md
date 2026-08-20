# SDK configuration delivery

Switch On Your Code SDKs evaluate feature flags locally. The control plane supplies an environment-scoped configuration document; SDKs cache that document and apply the evaluation contract in [`evaluation.md`](evaluation.md) without making a network request for each flag evaluation.

Machine-readable schema and cross-language compatibility vectors live in [`../spec/`](../spec/).

## Credential types

SDK credentials always belong to exactly one environment.

### Server credentials

Server credentials are secrets and are intended for trusted backend processes, workers, CLI tools, and server-rendered applications.

A server key has the form:

```text
syoc_server_<credential-id>.<secret>
```

The random secret contains 256 bits of entropy. Switch On Your Code returns the complete key only when the credential is created. The database stores only a SHA-256 digest of the random secret; listing credentials never returns the secret again.

Because the secret is high-entropy random material, SHA-256 is used as a lookup-verification digest rather than a password hashing function. Verification is constant-time. Revoking the credential immediately prevents further configuration downloads and closes matching realtime invalidation streams.

Server credentials receive every active feature flag in their environment.

### Client credentials

Client credentials are deliberately **not secrets**. They are intended for browser/mobile/public-client applications where embedded credentials can always be recovered by an end user.

A client key has the form:

```text
syoc_client_<public-id>
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

The endpoint does not use Switch On Your Code browser sessions or cookies. Authentication is entirely through the environment-scoped SDK key.

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

The structural schema is published as [`../spec/sdk-config-v1.schema.json`](../spec/sdk-config-v1.schema.json). Semantic requirements such as valid variant references, rollout totals and segment cycles are enforced by core before configuration is persisted or delivered.

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

If the environment configuration is unchanged, Switch On Your Code responds with `304 Not Modified` and no response body.

A new ETag is produced whenever the delivered document changes, including changes to flag visibility, enablement, variants, targeting policy, or referenced segments. The per-flag `revision` remains useful for configuration/audit semantics, but SDKs must use the document ETag as the cache validator for the complete payload.

Polling remains the reliability fallback even when an SDK uses realtime invalidation. An interrupted event stream must never make a long-running SDK permanently stale.

## Realtime invalidation

Realtime delivery is an invalidation channel rather than a second configuration transport. SDKs connect with the same environment-scoped bearer credential:

```http
GET /sdk/v1/events
Authorization: Bearer <sdk-key>
Accept: text/event-stream
```

The endpoint returns a Server-Sent Events stream. Browser implementations should use streaming `fetch()` rather than the native `EventSource` constructor because the latter cannot attach the required `Authorization` header.

The stream starts with:

```text
retry: 5000
event: ready
data: {"schema_version":1,"environment_id":"..."}
```

When delivered configuration may have changed, Switch On Your Code sends:

```text
event: configuration_changed
data: {"environment_id":"..."}
```

The event contains no configuration payload. The SDK performs an ordinary conditional `GET /sdk/v1/config` using its current ETag. This means the config endpoint remains the only source of truth and all existing validation/last-known-good behaviour is reused.

When the connected credential is revoked, Switch On Your Code sends `credential_revoked` and closes the stream. Reconnection or subsequent configuration fetches with that credential return `401 Unauthorized`.

The server writes comment heartbeats periodically to keep reverse proxies from considering an otherwise-idle stream dead. Proxies must not buffer or cache the event stream.

### Cross-replica delivery

SDK-visible writes are observed at the PostgreSQL transaction boundary. Database triggers publish scoped notifications through `LISTEN`/`NOTIFY` only after their transaction commits. Every Switch On Your Code API replica maintains a listener and fans those notifications out to its locally connected SDK streams.

Invalidations are scoped as narrowly as practical:

- environment-specific flag configuration and environment changes invalidate that environment;
- flag definition, variant, client-visibility and segment changes invalidate all environments in the project because they can affect more than one delivered document;
- credential revocation targets the individual credential stream.

In-memory stream queues deliberately coalesce duplicate pending invalidations. The event means “refresh to current state,” not “replay every intermediate mutation.” This is safe because the ETag-protected configuration document is authoritative.

This design requires no Redis or message broker and remains correct when Switch On Your Code runs multiple replicas against the same PostgreSQL database.

## Failure behaviour

SDK implementations should keep the most recent valid configuration when a transient refresh fails. A failed refresh must not erase already-loaded configuration.

The application-provided fallback value remains the final safety boundary when an SDK has never loaded a usable definition or encounters an unsupported future schema.

SDKs must reject schema versions they do not understand rather than silently interpreting them as an older format.

Realtime is an acceleration path, not the only freshness mechanism. SDKs should reconnect the event stream with backoff and retain periodic conditional polling as a fallback.
