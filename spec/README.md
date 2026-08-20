# FlagStack SDK contract

This directory contains machine-readable artifacts that define the public configuration/evaluation contract shared by the FlagStack control plane and official SDKs.

## Files

- `sdk-config-v1.schema.json` — structural JSON Schema for configuration returned by `GET /sdk/v1/config`.
- `evaluation-v1-vectors.json` — cross-language compatibility vectors for deterministic bucketing, custom scalar serialization, semantic versions and portable regular expressions.

The Go evaluator in this repository is the reference implementation. Official SDKs should consume or mirror these vectors in their own test suites before release.

The JSON Schema is intentionally structural. Semantic constraints that JSON Schema does not express cleanly—such as rollout weights totalling exactly `100000`, variant references existing, segment dependency cycles, type compatibility between flag kinds and values, and operator-specific condition requirements—are defined in [`../docs/evaluation.md`](../docs/evaluation.md) and enforced by the reference evaluator.

## Compatibility rules

Breaking evaluation or wire-format changes require a new schema version. Existing `schema_version: 1` behaviour should remain deterministic across SDK languages.

Custom `bucket_by` scalar values use the Go `encoding/json` representation before hashing. This is important for floating-point exponent thresholds and JSON string escaping; SDKs must not substitute their runtime's default string/number formatter.

Regular-expression matching uses a portable RE2-based subset. Core rejects otherwise-valid RE2 POSIX character classes because the officially supported .NET runtime cannot reproduce them safely with its non-backtracking engine. Other RE2-incompatible features such as look-around and backreferences are rejected by RE2 itself.
