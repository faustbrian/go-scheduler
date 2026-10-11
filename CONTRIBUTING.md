# Contributing

## Before Editing

1. Read [`AGENTS.md`](AGENTS.md) and the affected module's docs.
2. Run the standalone Go baseline below.
3. Identify owned dependencies and reverse dependants in `modules.json`.
4. Preserve unrelated work and generated/corpus provenance.

## Changes

Keep commits focused and conventional. Update every affected changelog with
the behavior and migration impact. Public API changes require compatibility
evidence and documentation. Specification behavior requires a decision record,
fixture coverage, and interoperability evidence.

New direct dependencies and dependency updates must follow the
[dependency governance policy](AGENTS.md#dependencies-and-supply-chain). Package-local
update bots are forbidden; the root policy owns every module and action update.

Specification-backed changes must follow the
[specification governance contract](AGENTS.md#design), update
the affected stable decision entries, and complete the Specification Decisions
section of the pull request template. An unresolved interpretation or stale
source pin is release-blocking; peer behavior cannot silently select policy.

Explicit full and release mutation gates must finish with zero surviving viable
mutants. Local mutation is optional, not a universal contribution prerequisite.

Do not add package-local workflows, permanent replacements, machine-specific
paths, bypass flags, broad mutation exclusions, or aggregate quality metrics
that hide a failing package.

## Verification

Use the declared Go 1.27.2 producer; the module minimum remains Go 1.27.0.
Test independently of sibling checkouts:

```bash
GOWORK=off go test ./...
```

This untagged command runs deterministic fuzz seeds and saved corpus
regressions, not active exploration or live PostgreSQL/Valkey integration.
The module declares PostgreSQL and Valkey services and the `integration` test
tag. Those prerequisites also apply to the verifier's local Make route; it is
not service-free. Shared setup manages its declared service fixtures and
requires the corresponding container runtime. Direct live tests need reachable
services and `POSTGRES_URL` / `VALKEY_ADDRESS`; unset variables skip those tests
and do not establish backend conformance:

```bash
GOWORK=off go test -tags=integration ./postgres ./valkey
```

With the checksum-pinned public verifier selected by `.golib.yaml`, use:

```bash
make inventory
make check
make ci
```

Both Make routes run bounded local checks. One selected root race verifies
that 32 concurrent replicas dispatch exactly one physical occurrence; another
in `schedulerservice` verifies drain before facility closure and rejects ticks
after shutdown. Each runs once within 90 seconds, three minutes total. Hosted
PR and main CI use the same immutable public workflow without building the
verifier release from source; Required fails unless it succeeds. Shared setup
still checks out pinned tooling, but no sibling repository or goal ledger is
needed.

Base-free checks run seeds/corpora; changed-source CI selects bounded active
smoke for causally affected targets. Short active exploration selects one
target; optional local mutation selects one package:

```bash
GOMAXPROCS=2 GOWORK=off go test -run '^$' -fuzz '^FuzzScheduleCompilation$' -fuzztime=2s -parallel=1 -timeout=2m .
golib mutation --module . --package .
```

`make check-full` retains both native 10,000-execution fuzz campaigns with
two-minute bounds, full coverage/race/mutation/API/security qualification,
Example documentation checks and 100-millisecond benchmarks, each original
docs/benchmark operation retaining its 20-minute bound. Run relevant backend,
fencing, cancellation and lifecycle qualification when those behaviors change.
Select full service qualification explicitly with `make check-full` or a
workflow dispatch with `release_dry_run: true`. Scheduled shared CI uses bounded
local checks with declared service fixtures, not the full campaign.
Public dependencies use the public proxy and checksum database. Report every
unavailable prerequisite or failing command; partial results are not release
evidence.

## Adding A Module

Follow [repository structure policy](AGENTS.md#repository-structure). New modules
require an explicit purpose, ownership boundary, dependency review, package
catalog entry, full quality gates, documentation, changelog, license, security
policy, compatibility plan, and release dry-run.
