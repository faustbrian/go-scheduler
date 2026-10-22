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

The packages under `adapters/` are the only integration paths. The
top-level integration paths and the combined `telemetry` observer were removed
in v3; compose `adapters/slog` and `adapters/otel` instead.

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
idempotency identities are unchanged, so v2 and v3 scheduler replicas
coordinate during a rolling upgrade. Applications that apply the
go-idempotency PostgreSQL migration move to the go-migrations v4 ledger:
stop every older migration runner and complete the
[ledger name upgrade](https://github.com/faustbrian/go-migrations/blob/v4.0.0/docs/operations.md#ledger-name-upgrade) before the first v4 run.
