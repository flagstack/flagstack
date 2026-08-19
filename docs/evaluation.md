# Flag evaluation model

FlagStack evaluates flags locally in SDKs from a versioned configuration payload. The control plane stores and validates the same model used by the reference Go evaluator so every SDK can implement identical behaviour.

The design deliberately supports feature delivery and deterministic variant assignment without making FlagStack an experimentation analytics product. Exposure/conversion statistics can be layered on later without changing the evaluation contract.

## Evaluation context

FlagStack follows the OpenFeature evaluation-context shape conceptually:

```json
{
  "targetingKey": "user-123",
  "email": "user@example.com",
  "country": "GB",
  "plan": "enterprise",
  "groups": ["staff", "beta-testers"],
  "organisation_id": "org-42",
  "app_version": "2.4.0",
  "custom": {
    "beta_program": true
  }
}
```

`targetingKey` identifies the subject of an evaluation. It does not have to be a user; it can identify an organisation, device, service, request tenant or any other stable subject.

All other fields are application-owned attributes. FlagStack does not need access to or synchronization with the application's user database. Applications decide which attributes to expose to evaluation.

Nested attributes are addressed with dot paths such as `custom.beta_program`.

## Flag values and variants

Flags retain the existing four value kinds:

- `boolean`
- `string`
- `number`
- `json`

Every flag has a safe project-level default value.

Boolean flags have three reserved evaluation variants:

- `on` -> `true`
- `off` -> `false`
- `default` -> the project-level default

Other flag kinds can define named variants:

```json
[
  {"key":"control","value":"control"},
  {"key":"compact","value":"compact"},
  {"key":"new-design","value":"new-design"}
]
```

`default` is reserved for all flag kinds and resolves to the flag's configured default value.

Variant values must match the flag kind.

## Environment enablement

Environment configuration remains sparse.

When no environment configuration exists, or the configuration is disabled, evaluation returns the project-level default with reason `DISABLED`.

When a boolean flag is enabled and has no targeting policy, it resolves to the built-in `on` variant. This preserves the intuitive simple on/off workflow.

When a non-boolean flag is enabled and has no targeting policy, it resolves to its default value.

## Ordered targeting rules

Rules are evaluated from first to last. The first matching rule wins.

A rule contains:

- a stable ID;
- an optional display name;
- `all` or `any` condition matching;
- one or more conditions;
- an outcome.

Example:

```json
{
  "id": "enterprise-users",
  "name": "Enterprise customers",
  "match": "all",
  "conditions": [
    {"attribute":"plan","operator":"equals","value":"enterprise"},
    {"attribute":"account.active","operator":"equals","value":true}
  ],
  "outcome": {"variant":"on"}
}
```

After all rules, a fallthrough outcome is evaluated.

## Condition operators

The initial evaluation specification supports:

- `equals`
- `not_equals`
- `in`
- `not_in`
- `contains`
- `not_contains`
- `starts_with`
- `ends_with`
- `greater_than`
- `greater_than_or_equal`
- `less_than`
- `less_than_or_equal`
- `exists`
- `not_exists`
- `matches_regex`
- `semver_greater_than`
- `semver_greater_than_or_equal`
- `semver_less_than`
- `semver_less_than_or_equal`
- `in_segment`
- `not_in_segment`

Missing attributes do not satisfy normal or negative comparisons. Use `not_exists` when absence itself should match.

`contains` supports strings, arrays and object keys, which allows contexts such as `groups: ["staff", "beta-testers"]` without FlagStack needing to manage the application's groups.

## Reusable segments

Segments are project-scoped reusable sets of conditions.

Example:

```json
{
  "key": "uk-beta-testers",
  "name": "UK beta testers",
  "match": "all",
  "conditions": [
    {"attribute":"country","operator":"equals","value":"GB"},
    {"attribute":"custom.beta_program","operator":"equals","value":true}
  ]
}
```

Rules target a segment with `in_segment` or `not_in_segment`.

Segments may reference other segments. Evaluation detects cycles and fails safely to the flag default rather than recursing indefinitely.

## Deterministic percentage rollout

Percentage allocations use 100,000 integer buckets, giving 0.001% granularity without floating-point configuration.

A 10% boolean rollout is represented as:

```json
{
  "rollout": [
    {"variant":"on","weight":10000},
    {"variant":"off","weight":90000}
  ]
}
```

Allocations must total exactly `100000`.

By default, rollout buckets use `targetingKey`. An alternative scalar context attribute can be selected with `bucket_by`, for example `organisation_id`, so all users belonging to the same organisation receive the same variant.

The v1 bucket algorithm is deliberately simple and portable across Python, JavaScript/TypeScript, Go and .NET:

```text
input = "flagstack-v1" + NUL + environment_id + NUL + flag_id + NUL + bucket_value
hash = SHA-256(input UTF-8 bytes)
bucket = big-endian uint32(hash[0:4]) mod 100000
```

Compatibility vector:

```text
environment_id = env-1
flag_id        = flag-1
bucket_value   = user-123
bucket         = 22683
```

Because the bucket depends on stable flag/environment identity and the subject value, increasing a rollout from 10% to 25% retains the original 10% cohort and adds more subjects instead of reshuffling everybody.

A percentage outcome without the required bucket attribute fails safely to the flag default. Missing `targetingKey` maps to the OpenFeature-compatible `TARGETING_KEY_MISSING` error code.

## Variant assignment / A-B delivery

FlagStack supports deterministic assignment between multiple variants:

```json
{
  "rollout": [
    {"variant":"control","weight":50000},
    {"variant":"compact","weight":25000},
    {"variant":"new-design","weight":25000}
  ]
}
```

This is enough for applications to deliver A/B or multivariate experiences consistently.

FlagStack does **not** initially provide experiment conversion tracking, statistical significance, experiment reports or winner selection. Those analytics capabilities can be added later without changing variant assignment.

## Resolution details

Evaluators return the resolved value plus metadata suitable for OpenFeature providers:

- `variant`
- `reason`
- matching `rule_id`, when applicable
- `error_code` and `error_message` on safe fallback

Reasons use OpenFeature terminology where applicable:

- `STATIC`
- `DEFAULT`
- `TARGETING_MATCH`
- `SPLIT`
- `DISABLED`
- `ERROR`

## Scheduling

Scheduling is a control-plane concern rather than part of local evaluation.

Scheduled changes are durable records scoped to an organisation, project, environment and flag. A scheduled change stores:

- execution time;
- requested configuration patch;
- status;
- creator;
- creation/execution timestamps.

A patch can change environment enablement and/or the evaluation policy. This allows both simple schedules and progressive rollouts.

Simple example:

```text
2026-12-01 00:00 Europe/London -> enabled = true
2026-12-27 00:00 Europe/London -> enabled = false
```

Progressive example:

```text
09:00 -> 5% on / 95% off
12:00 -> 10% on / 90% off
next day -> 25% on / 75% off
next day -> 50% on / 50% off
final day -> 100% on
```

The scheduler must be safe with multiple FlagStack replicas. Due work is claimed atomically from PostgreSQL before applying a configuration revision, so only one replica executes a scheduled change.

## Privacy

Evaluation context is supplied by the application. FlagStack does not require email addresses, names or other personal data to perform targeting. Applications should prefer opaque stable IDs and the minimum attributes necessary for their targeting rules.

The configuration delivered to server-side SDKs contains rules and segment definitions but not a synchronized database of application users.
