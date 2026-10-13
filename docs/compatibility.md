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

## v2.0.2 patch adoption

The prepared v2.0.2 patch keeps `/v2` imports, the Go 1.27.0 minimum, lease
storage schemas, queue submission identities and existing API contracts.
Select the patch explicitly once its public tag and module artifacts are
available; the latest published root baseline before this preparation is v2.0.1.

An exhausted cron result remains zero before jitter. Registry replay stops
without fabricated occurrences or lost admitted tails, and runner timer
selection ignores exhausted schedules while preserving valid schedules.
Real candidate scans still enforce `MaxOccurrenceScan`; see
[time and missed runs](time-and-missed-runs.md).

[Go Service composition](service-integration.md) remains explicit through
`adapters/service` or the retained `schedulerservice` package. Callers own
facilities, callbacks and shutdown deadlines. Runner cancellation and Drain
remain cooperative: a timed-out wait does not prove cleanup has joined and
this patch cannot force a blocking callback to return.
