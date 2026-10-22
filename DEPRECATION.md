# Deprecation Policy

Deprecations MUST identify the replacement, reason, migration steps, and
earliest removal version. Public Go identifiers use a valid `Deprecated:` doc
paragraph and corresponding changelog entry.

At `v1` and later, a supported replacement SHOULD exist for at least one minor
release before removal. Security or correctness defects MAY require faster
removal when continued support would be unsafe; the release notes must explain
the exception.

Silent behavior changes, undocumented aliases, and indefinite deprecated code
are prohibited. Deprecations are checked during compatibility and release
review.

Version 3 removed the `idempotency`, `lease`, `queue`, `schedulercli`,
`schedulerhttp`, `schedulerservice`, and combined `telemetry` packages that v2
deprecated in favor of the `adapters/*` packages. The removal preceded the
announced window because the owned consumers move with v3.

Migrating to v3 changes the scheduler import paths to `/v3`, replaces each
removed package with its `adapters/*` path, moves the dependency majors listed
in [`COMPATIBILITY.md`](COMPATIBILITY.md) and moves scope-keyed telemetry
selectors. Telemetry consumers construct `adapters/slog` and `adapters/otel`
separately and register either or both observers.
