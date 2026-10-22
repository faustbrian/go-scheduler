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

Migration requires only changing imports to the corresponding `adapters/*`
path; the one-to-one successors preserve released type and sentinel identities.
Telemetry consumers construct `adapters/slog` and `adapters/otel` separately
and register either or both observers as their application composition needs.
The earliest possible removal version is v2.0.0, and removal may not occur
before both 180 days after v1.1.0 publication and two later stable minor
releases have elapsed. Removal additionally requires migration of owned
consumers and current external-consumer compatibility evidence.
