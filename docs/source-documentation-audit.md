# Source documentation audit

[Documentation map](index.md) · [Repository README](../README.md)
[Next: API reference](api.md)

## Acceptance and owned boundary

This Tier A pass owns source comments and directly contradicted maintained
prose at main e708d5e3c86ce37b69b0dd374b3101f4f707ab17. Runtime tokens, AST,
literals, directives, dependencies, SQL/Lua, fixtures, policy and historical
authorities remain unchanged. Consumers include compiled scheduling, runner
executors and callbacks, lease providers, queue/idempotency dispatch, service
lifecycle, administrative HTTP/CLI, telemetry and test harnesses. Source-derived
ownership, cancellation, crash windows and provider differences are the risks;
this is not a runtime, service, security or conformance repair.

## Reproducible before-write inventory

Tracked Git inputs and source-docs inventory identify one v2 module, 87 Go
files (35 source, 52 test), 28 package directories, and 141 missing exported
member/method comments. Two integration-tagged PostgreSQL/Valkey tests remain
included in the audit. Four example directories, two fuzz targets, six
benchmarks, service Example and lease conformance harness are included.
No generated/vendor/platform Go source, generation directive or embedded input
is present; executable SQL/Lua strings and their fixtures are immutable inputs.

The native .golib.yaml configures full go-doc api/baseline.txt for 22 operational
packages. Native api update/check owns that snapshot. Test-only adapters are
inspected as tests; conformance and four example packages need extra renders.
The docs operation selects Examples once; local Make routes provision services
and selected races, so they are not this prose pass's cheap docs check.
Native CI is an immutable shared workflow with fatal Required aggregation.

## Missing versus technical investigation

Missing structural documentation is concentrated in Schedule/Context/Hooks,
runner events/interfaces, lease data/methods, queue envelopes and HTTP views.
Trace actual implementation before explaining time bounds, identity versus
coordination keys, copy depth, asynchronous callback ordering, timeout versus
termination, drain versus cancellation, retained occurrence leases, server-time
expiry, recovery fencing, idempotency completion and instrumentation lifetime.
Existing provider/backend matrices and dated security/performance records are
not new qualification evidence. Preserve historical verdicts and actual pinned
parser edition; no universal cron, exactly-once or transport-privacy claim.

## Smallest sufficient checks

Pinned Go 1.27.2 formatting and all-source AST/raw-token/literal/directive proof;
native full API snapshot regeneration/stability and relevant extra renders;
declared Example selection; structural source, local documentation and native
specification applicability; pinned current/staged secret checks and ordinary
complete final-diff truth/usefulness inspection. Exact owned three caches and
helpers are disposable. No generator execution, live PostgreSQL/Valkey,
race/leak, active fuzz, benchmark or broad qualification campaign is selected.
Coordinator owns remote main, exact-source CI and aggregate delivery.

## Investigated source distinctions

Schedule copying retains callback closures; custom options and direct Compile
are not comprehensive validation of manually mutated exported fields. JSON
marshalers execute before encoded-size admission. Coordination keys exclude
revision timing, while deterministic jitter uses revision identity. Constraints
intersect local minute/day restrictions before jitter and cap their search.
The parser is pinned robfig/cron v3.0.1, not universal cron certification.

RunTimeout/callback timeout stop waiting, not goroutines. EventFinished/After
can occur before an uncooperative executor returns; execution capacity and
heartbeats remain retained through actual cleanup. Lifecycle delivery is
best-effort and callback completion can reorder. Lease/pause contexts depend on
cooperation. Drain neither cancels Run nor closes clients; repeated calls retain
waiters until work completes. No core retry loop increments occurrence attempts.

Memory uses supplied time and retains counters; PostgreSQL/Valkey use server
time and have different sub-millisecond TTL admission and expiry visibility.
PostgreSQL inactive rows and Valkey same-slot counters protect fence history,
not downstream effects or unlimited-key storage. SQL/Lua bytes stay unchanged.
Queue submission lacks a context argument. Idempotency ownership is unrenewed,
completion is separate from effects, and failure release detaches cancellation
without a new deadline. Administrative authentication/deadlines and telemetry
redaction/provider shutdown remain application-owned. History is shallow and
span retention is not globally bounded when lifecycle callbacks are omitted.

These are source observations, not new runtime, deployment, failover, security
or performance qualification. Existing contract tests and dated/pinned evidence
remain inputs rather than being re-certified by a prose pass.

The registry jitter wrapper adds its offset even to an exhausted zero cron
result. Next/Overview can therefore expose a year-one offset sentinel; Due's
nonadvancing loop can reach ErrOccurrenceLimit rather than return empty. The
source contracts record this limitation without inventing a rationale or
altering the algorithm. This remains a runtime limitation, not a prose gate.

Final rendering also narrowed old timeout-constant and copied-value summaries.
The declared go-service v1.1.2 reverse-stop loop retains errors and attempts all
remaining Stop callbacks; a failed scheduler Drain does not gate facility stops.
Source and the current service guide therefore describe a drain attempt, not
unconditional join-before-closure. No service implementation or dependency is
changed or newly qualified.


## Final outcomes and remaining boundaries

Go 1.27.2 formatting and both bounded native sessions preserve the AST,
all raw tokens (including implicit semicolons), literals and directives of
all 87 tracked Go files. All 626 comment groups parsed/rendered. The configured
22-package full GoDoc snapshot was regenerated by native api update/check and
remained stable; all package sections and five extra conformance/example renders
were inspected. Structural source findings are zero, local documentation is
valid, and native specification checking has zero applicable modules/decisions.

The exact declared Example selection passed the one service Example and
compiled all 28 test package directories. Later source-summary refinements
reused that result and the five unchanged extra renders; they changed no runtime,
dependency or example input. No live integration, ordinary behavioral suite,
race/leak, active fuzz, benchmark, failover or release campaign was executed.

One authorized deterministic UTC observation used the public Next API for
31 February with one-hour jitter, at 2026-01-01T00:00:00Z. It returned
0001-01-01T00:58:10.333935249Z (nonzero, nonfuture), corroborating the exhausted
sentinel limitation. No Due replay loop or timing/conformance campaign ran.
The final identity/snapshot refresh reused that unchanged observation.

Pinned ignore-disabled secret scans of the full tracked baseline and candidate
found nothing. Final changed/staged content is checked separately, without
printing values. Ordinary complete-diff inspection covers truth, usefulness,
copy/ownership limits and preserved executable inputs. Both sessions' exact
three caches, helpers and probe output were removed immediately. The remaining
runtime limitations above are documented, not repaired or newly qualified.
Coordinator-owned remote delivery and required exact-source CI remain separate
from this local source audit.

## Objective Enforcement

The checksum-pinned public Tools v2.4.0 verifier now supplies
`make source-docs-check` as a prerequisite of the existing local and CI Make
routes. Ordinary shared CI selects strict source-documentation checking and
requires its result. Native prerequisites, operation budgets, security and
release controls are retained. Hosted exact-head evidence is recorded by the
coordination ledger; configuration alone does not establish green delivery.
