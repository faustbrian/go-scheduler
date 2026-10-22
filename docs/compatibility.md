# Compatibility

[Documentation map](index.md) · [Repository README](../README.md)
[Next: API reference](api.md)

The module currently requires Go 1.27.0 because its pinned `idempotency`
dependency requires that toolchain. Cron behavior is pinned to
`robfig/cron/v3`. PostgreSQL support targets currently maintained releases 14
through 18 and is enforced by the CI adapter matrix. Valkey support requires
major version 9 or later.

Public API compatibility is checked before release. Major releases may make
breaking changes, which are recorded in `CHANGELOG.md`. Time-zone behavior also
depends on the zone database shipped with the Go runtime or host.

## v3 adoption

V3 moves the module to `/v3`, removes the v2 compatibility packages in favour
of `adapters/*`, and adopts Telemetry v4, Idempotency v3, Queue v2 and Service
v3. Lease storage schemas, lease keys and occurrence idempotency identities are
unchanged; see [`COMPATIBILITY.md`](../COMPATIBILITY.md) for the rolling-upgrade
and migration-ledger steps.

An exhausted cron result remains zero before jitter. Registry replay stops
without fabricated occurrences or lost admitted tails, and runner timer
selection ignores exhausted schedules while preserving valid schedules.
Real candidate scans still enforce `MaxOccurrenceScan`; see
[time and missed runs](time-and-missed-runs.md).

[Go Service composition](service-integration.md) remains explicit through
`adapters/service` (package `schedulerservice`). Callers own
facilities, callbacks and shutdown deadlines. Runner cancellation and Drain
remain cooperative: a timed-out wait does not prove cleanup has joined, and
the scheduler cannot force a blocking callback to return.
