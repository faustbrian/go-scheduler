# Changelog

All notable changes are documented here. The format follows Keep a Changelog,
and releases use Semantic Versioning.

## [Unreleased]

## [2.0.2] - 2026-10-13

Prepared patch candidate; this entry does not claim tag or module publication.
The frozen source range is public v2.0.1 (`b75de2142298721a4d6acdc5936dfec1c0559a59`)
through `bbca89240589d9fbbf7b840afa6596716f6264f1`. Release metadata preparation
is outside that range. Dependency notes retain the intermediate Telemetry/API
1.46 adoption; the final OpenTelemetry API is 1.47.0 and SDK is 1.46.0.

### Fixed

- Preserve an exhausted cron's zero sentinel before deterministic jitter,
  terminate missed-run replay without fabricated year-one occurrences or lost
  finite tails, and retain valid runner timers alongside exhausted schedules.

### Security

- Select Go 1.27.2 for CI and development to include standard-library
  security fixes, retaining the public Go 1.27.0 minimum. Applications
  must be rebuilt with a patched toolchain to receive these fixes.

- Update the HTTP/2 dependency to include upstream crash, resource-use and
  flow-control fixes.

### Changed

- Adopt PGX v5.11.0, Queue v1.1.2, Service v1.1.2 and Correlation v1.1.3
  with their compatible dependency graph. Scheduler lease/fencing, queue
  submission, correlation identity and caller-owned service lifecycles remain
  explicit; this patch does not force cancellation of trusted callbacks.

- Clarify scheduling copy/identity, runner wait-versus-cleanup, callback delivery,
  provider clocks and administrative/telemetry ownership without runtime changes.

- Adopt OpenTelemetry API v1.47.0 while retaining SDK v1.46.0,
  scheduler instrumentation and caller-owned provider lifecycles.

- Use compatible, immutable source tooling and the updated lint driver
  for patched Go while retaining every existing verification gate.

- Adopt Telemetry v2.0.1 and OpenTelemetry API and SDK v1.46.0 while
  preserving scheduler instrumentation and caller-owned runtime lifecycles.
  Runtime resources now identify the selected SDK version.

### Source accountability

Each non-merge commit in the frozen range is accounted for exactly once:

- Exhausted occurrences, finite replay tails and mixed runner timers:
  [bbca892405](https://github.com/faustbrian/go-scheduler/commit/bbca89240589d9fbbf7b840afa6596716f6264f1).
- PostgreSQL driver adoption:
  [9ed779b86f](https://github.com/faustbrian/go-scheduler/commit/9ed779b86fd47bcd547c67cd17bc368ac51bce7a).
- Service/Correlation dependency adoption and the later Correlation patch:
  [5f63f080f2](https://github.com/faustbrian/go-scheduler/commit/5f63f080f23dc8b284a2fc2d86abb8260ef808a0),
  [c913f03cd9](https://github.com/faustbrian/go-scheduler/commit/c913f03cd954ba37f0d2855fbf6805e4102295d8).
- Compatible Queue dependency adoption:
  [2eac149fbb](https://github.com/faustbrian/go-scheduler/commit/2eac149fbb1c40959b2b24e43545685534bf9529).
- Telemetry, patched HTTP/2 and subsequent OpenTelemetry API adoption:
  [d4a131b26f](https://github.com/faustbrian/go-scheduler/commit/d4a131b26f0d327198b4624b6a09863eedf5c84b),
  [34424e0896](https://github.com/faustbrian/go-scheduler/commit/34424e0896467ab116111f3ac88685a9ca0dfe09).
- Patched Go CI selection and bounded fixture startup diagnostics:
  [4ffe15f750](https://github.com/faustbrian/go-scheduler/commit/4ffe15f75036d6e48df44986ea6706590f14e198),
  [85ad2afe78](https://github.com/faustbrian/go-scheduler/commit/85ad2afe78c6cbd58376601983d72ae4f30330a0).
- Immutable CI actions, documentation tooling, bounded verification, causal fuzz selection and supply-chain tooling:
  [eeb92362a3](https://github.com/faustbrian/go-scheduler/commit/eeb92362a3b2582b8a9c0c14e3640a2f48deed55),
  [1060e7025a](https://github.com/faustbrian/go-scheduler/commit/1060e7025a4de49b2964d62c47e166f5feb0b2ad),
  [416dc9948d](https://github.com/faustbrian/go-scheduler/commit/416dc9948d1f231f725ce7e22c66eca2afc4bc10),
  [b11667444a](https://github.com/faustbrian/go-scheduler/commit/b11667444a3d9051aa5c234c248186f5b6bf0f37),
  [e708d5e3c8](https://github.com/faustbrian/go-scheduler/commit/e708d5e3c86ce37b69b0dd374b3101f4f707ab17).
- Development policy and non-Go dependency update coverage:
  [70d5230ad7](https://github.com/faustbrian/go-scheduler/commit/70d5230ad7fb2baaf254cd8f4ab69df941bf8df6),
  [3fe27863b9](https://github.com/faustbrian/go-scheduler/commit/3fe27863b9d0bbab483d64aa33fdc44b9e5a8e1b).
- Scheduling/lifecycle contracts, source-documentation enforcement and adoption navigation:
  [667b96cfc7](https://github.com/faustbrian/go-scheduler/commit/667b96cfc73965faab50b2f7c180b4bc9b43ae47),
  [bdb447ff75](https://github.com/faustbrian/go-scheduler/commit/bdb447ff75e8903537339276dcb590da08084b0a),
  [9f9fc1f57d](https://github.com/faustbrian/go-scheduler/commit/9f9fc1f57d2115bec455587efa570d713050680c).

## [2.0.1] - 2026-09-28

### Security

- Adopt `go-idempotency` v1.2.0 to bound persisted replay metadata and backend
  record validation while retaining the scheduler's existing v1 store contract.

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

[Unreleased]: https://github.com/faustbrian/go-scheduler/compare/v2.0.1...HEAD
[2.0.1]: https://github.com/faustbrian/go-scheduler/compare/v2.0.0...v2.0.1
[2.0.0]: https://github.com/faustbrian/go-scheduler/compare/v1.1.0...v2.0.0
[1.1.0]: https://github.com/faustbrian/go-scheduler/releases/tag/v1.1.0
[1.0.0]: https://github.com/faustbrian/go-scheduler/releases/tag/v1.0.0
