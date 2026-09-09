# Compatibility Policy

Each releasable directory is an independent Go module and follows semantic
versioning. Tags use `<module-directory>/v<version>`.

Before `v1`, minor releases MAY contain reviewed breaking changes, but every
break MUST be documented with migration guidance. Patch releases MUST remain
backward compatible. At and after `v1`, incompatible exported API or documented
behavior changes require a new major version.

Compatibility includes exported Go APIs, error classification, serialization,
protocol behavior, persistence schemas, environment variables, command output,
resource ownership, ordering, retry/idempotency semantics, and documented
defaults. A compile-compatible change can still be behaviorally breaking.

Specification-backed modules MUST NOT diverge from their declared standards.
Ambiguities require documented decisions and stable tests. Deprecated APIs
follow [`DEPRECATION.md`](DEPRECATION.md).

The target-oriented packages under `adapters/` are the canonical integration
paths beginning with v1.1.0. The released top-level integration paths remain
source-compatible for the longer of 180 days after v1.1.0 publication and two
subsequently published stable minor releases. The combined `telemetry` package
preserves its logging, metrics, and tracing behavior while new consumers may
select `adapters/slog` and `adapters/otel` independently.

`adapters/lease` is an additive identity-preserving successor. The released
`lease` package remains the implementation owner used by the root scheduler and
its storage backends because reversing that dependency would either create a
cycle or change released named-type and reflection identities. New external
composition may import `adapters/lease`; the two paths expose the same contract.
No compatibility path may be removed before v2.0.0 and the interval and
consumer-evidence conditions in [`DEPRECATION.md`](DEPRECATION.md) are met.
