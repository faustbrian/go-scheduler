# Changelog

All notable changes are documented here. The format follows Keep a Changelog,
and releases use Semantic Versioning.

## [Unreleased]

## [2.0.0] - 2026-09-28

### Changed

- Move the scheduler module and all package imports to `/v2` so runtime
  constructors accept `go-telemetry/v2`. Applications must update scheduler
  imports and pass a telemetry v2 runtime; the scheduler's existing metrics
  and trace instrumentation scope is unchanged. ([daa985321f](https://github.com/faustbrian/go-scheduler/commit/daa985321fb2c0617ceed38477d217b11c76e6ec))
- Require and test with Go 1.27.0. ([dc7782ae4b](https://github.com/faustbrian/go-scheduler/commit/dc7782ae4bdc880b8dd6ebd1d55178c48db4ac56), [8b699b3b1c](https://github.com/faustbrian/go-scheduler/commit/8b699b3b1ce388daa6f3f8aa869665f12d520ad4))
- Clarify that goroutine lifecycle changes require targeted leak tests while
  preserving the existing CI workflow. ([da843f2216](https://github.com/faustbrian/go-scheduler/commit/da843f2216e63d1656342d97ffa81fe412f814db), [31fd66283d](https://github.com/faustbrian/go-scheduler/commit/31fd66283df7c12ceb3e78d406df4f1e3a3ac6ff))

## [1.1.0] - 2026-09-09

### Added

- Add canonical target-oriented scheduler adapters for CLI, HTTP,
  idempotency, lease, OpenTelemetry, queue, service lifecycle, and `log/slog`.

### Deprecated

- Deprecate the released top-level integration packages in favor of their
  `adapters/*` successors while preserving their public type, error, and
  behavior contracts. New consumers may replace the combined `telemetry`
  package with independently selectable `slog` and OpenTelemetry adapters.

### Changed

- Adopt the public Correlation v1.1.0, Idempotency v1.1.0, and Telemetry v1.2.0
  successor contracts used by the canonical adapters.
- Select verification by proportional assurance tier so documentation and
  metadata changes do not trigger unrelated mutation, release, security, or
  evidence-artifact gates on pull requests.

- Adopt the checksum-verified `go-library-tools` v1.6.2 CLI and immutable
  shared workflow so executable gates match proportional assurance policy.
- Resolve owned v1.0.0 dependencies from their public proxy and SumDB
  identities instead of bootstrap-only archives.

- Adopt the `go-library-tools` v1.3.0 schema-v2 cohesion contract and local
  `make cohesion` gate without changing scheduler API or runtime behavior.
- Pin reusable CI to the immutable v1.3.0 workflow and enforce cohesion
  metadata in the required repository contract.

- Replace repository-local verification scripts with the released
  `go-library-tools` v1.0.14 contract while preserving scheduler behavior,
  public APIs, fixtures, API baselines, benchmarks, and mutation evidence.

### Documentation

- Add installation, Scheduler-versus-Sequencer selection, complete package and
  adapter guidance, and explicit service and example adoption paths.

- Add an executable PostgreSQL or Valkey singleton recipe that demonstrates
  two-replica fencing, cancellation, closed admission, and bounded draining.

- Publish the scheduler family, package selection, ownership, lifecycle,
  supported environments, and delivery status with versioned ecosystem links.

- Replace archived monorepo links and completed execution artifacts with a
  standalone, human-oriented documentation structure.

## [1.0.0] - 2026-08-25

### Changed

- Refresh the reviewed zero-mutant identity for the extracted lease package
  without weakening the exact mutation contract.

- Exclude intentional nested modules from root local-proxy archives so local,
  bootstrap, CI, and public module checksums describe the same source
  boundary.

- Track the pinned documentation-tool lockfile so clean CI checkouts install
  the exact validated cspell dependency.

- Reconcile standalone dependency checksums against deterministic current
  module archives so CI, local verification, and release consumers resolve
  identical content.

- Harden standalone documentation validation with deterministic spelling and
  link checks, package-specific documentation gates, and repository-local
  contributor guidance.

### Documentation

- Correct stale package, standalone, and authoritative-source links in public
  documentation.

### Documentation

- Link the package README to package-owned documentation.

### Changed

- Publish the module from its standalone `github.com/faustbrian/go-scheduler` identity while preserving its documented API and behavior.
- Make fuzz-target discovery portable to minimal CI runners without ripgrep.
- Replace obsolete owned-module pseudo-version pins with the monorepo's local
  `v0.0.0` source-proxy coordinates; release tooling continues to emit exact
  `v1.0.0` dependency versions.
- Deep-copy nested schedule parameters when compiling a registry so caller
  mutations cannot alter compiled execution input.
- Classify task-lease heartbeat failures deterministically before canceling the
  managed execution.
- Bound the initial history-buffer allocation independently from its accepted
  maximum capacity.
- Strengthen lease-store conformance checks and exact scheduler, adapter, CLI,
  HTTP, telemetry, and lifecycle boundary coverage.
- Restore immutable main pseudo-version pins for every owned dependency so the
  current scheduler source resolves from a clean external module.
- Release managed execution capacity before completing an occurrence so
  single-worker catch-up does not reject the next cooperative execution.
- Require owned sibling modules at local `v0.0.0`; clean external consumers
  pin each module to an exact main pseudo-version.

- Upgrade gRPC to 1.82.1 to remove the reachable `GO-2026-6061`
  vulnerabilities.
- Pin unpublished owned modules to exact resolvable `main` revisions so clean
  scheduler consumers no longer require nonexistent `v0.1.0` tags.
- Refresh owned-module checksums against the final consolidated archives.
- Normalized standalone module metadata against the canonical owned dependency
  graph, including complete checksums for clean consumer resolution.

### Added

- Composable application-owned pause controls with process-local state,
  per-schedule exemptions, fail-closed lookups, and typed skipped events.
- A deterministic registry overview API that combines immutable definitions
  with each enabled schedule's next run for caller-owned control surfaces.
- A distinct `After` / `EventFinished` execution boundary and background
  metadata while retaining `Completed` for every scheduling decision.
- Laravel-compatible `OnOneServer`, `WithoutOverlapping`, and
  `RunInBackground` controls with independent mutex TTLs, managed asynchronous
  lifecycle reporting, and CLI bulk overlap-lock cleanup.
- Laravel-compatible frequency helpers from seconds through yearly schedules,
  plus weekday, recurring time-window, and skip constraints that compose with
  existing environment and timezone options.
- `Runner.RunFrom` for bounded startup catch-up that transitions into the
  continuous schedule loop without losing an occurrence between both phases.
- Configurable overlap-lease heartbeat intervals with construction-time TTL
  safety validation.
- `schedulerservice` lifecycle composition that stops scheduling, drains active
  executions before owned facilities close, and applies the existing schedule
  correlation semantics to every occurrence.
- PostgreSQL lease stores and migrations can target an explicit caller-owned
  schema while preserving the public-schema default.
- code-defined versioned schedules with deterministic timezone-aware timing
- fenced memory, PostgreSQL, and Valkey 9 lease adapters
- bounded missed-run and overlap decisions
- `queue`, `idempotency`, `log`, and `telemetry` integration
- HTTP and CLI inspection and fenced recovery surfaces
- bounded history, hooks, fake clock, and lifecycle observability
- explicit definition, registry, catch-up, and occurrence-scan resource limits
- task-lease heartbeats and safe overlap-replacement capability contracts
- public cron compiler with typed expression and time-zone errors
- rollout-stable coordination identity across revision and timing changes
- bounded lease calls, callbacks, and managed non-cooperative executions
- complete Gregorian-cycle cron search for non-leap century boundaries
- multi-replica, crash-window, and live backend fault conformance suites
- threat model, rollout and crash matrices, and benchmark release baseline
- bounded runner observer registration

[Unreleased]: https://github.com/faustbrian/go-scheduler/compare/v2.0.0...HEAD
[2.0.0]: https://github.com/faustbrian/go-scheduler/compare/v1.1.0...v2.0.0
[1.1.0]: https://github.com/faustbrian/go-scheduler/releases/tag/v1.1.0
[1.0.0]: https://github.com/faustbrian/go-scheduler/releases/tag/v1.0.0
