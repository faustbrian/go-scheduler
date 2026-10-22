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

`adapters/lease` owns the lease contract used by the root scheduler and its
storage backends. Version 3 removed the v2 compatibility packages; their
implementations now live at the matching `adapters/*` paths with the same
package names except `schedulerqueue`, `schedulerlease` and
`scheduleridempotency`.

The v3 scheduler module requires `go-telemetry/v4`, `go-idempotency/v3`,
`go-queue/v2` and `go-service/v3`. Applications migrating from scheduler v2
must update scheduler import paths to `/v3` and move those dependencies with
it. The OpenTelemetry instrumentation scope is now
`github.com/faustbrian/go-scheduler/v3`; move dashboard and alert selectors
keyed on the previous `github.com/faustbrian/go-scheduler` scope. Metric names
and units are unchanged. Lease keys, the PostgreSQL schema and occurrence
idempotency identities are unchanged, so v2 and v3 replicas coordinate during
a rolling upgrade.
