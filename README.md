# scheduler

[![CI](https://github.com/faustbrian/go-scheduler/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/faustbrian/go-scheduler/actions/workflows/ci.yml)
[![CodeQL](https://img.shields.io/badge/CodeQL-required-blue)](https://github.com/faustbrian/go-scheduler/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/badge/coverage-100%25_required-blue)](CONTRIBUTING.md#verification)
[![Mutation](https://img.shields.io/badge/mutation-100%25_required-blue)](CONTRIBUTING.md#verification)
[![Documentation](https://img.shields.io/badge/docs-checked_in_CI-blue)](docs/)
[![Go Reference](https://pkg.go.dev/badge/github.com/faustbrian/go-scheduler.svg)](https://pkg.go.dev/github.com/faustbrian/go-scheduler)
[![Release](https://img.shields.io/github/v/release/faustbrian/go-scheduler?sort=semver)](https://github.com/faustbrian/go-scheduler/releases)
[![Go](https://img.shields.io/badge/go-1.27.0-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

`scheduler` is a code-defined application scheduler for Go services running
on Kubernetes. Multiple scheduler replicas coordinate through fenced leases,
while durable business work is dispatched to `queue` workers.

The module follows stable v1 compatibility. It does not claim exactly-once execution: leases reduce
duplicate dispatch, and jobs must remain idempotent.

## Requirements

- Go 1.27.0 or later
- PostgreSQL or Valkey 9 for multi-replica deployments
- `queue` with a durable backend for long-running business work

## Installation

```sh
go get github.com/faustbrian/go-scheduler@v1.1.0
```

All packages in this repository share that module version. PostgreSQL, Valkey,
queue, service-lifecycle, HTTP, CLI, and telemetry integrations remain explicit
imports; installing the module does not start workers or register globals.

## Five-minute quickstart

Run the complete process-local example and stop it with `Ctrl-C` after an
occurrence:

```sh
go run github.com/faustbrian/go-scheduler/examples/basic@v1.1.0
```

The production-shaped construction path for a multi-replica service is:

```go
schedule, err := scheduler.NewSchedule(
    "nightly-report",
    "reports.generate",
    scheduler.Daily(),
    scheduler.WithTimezone("Europe/Helsinki"),
    scheduler.WithOneServer(5*time.Minute),
)
if err != nil {
    return err
}

registry, err := scheduler.Compile(schedule)
if err != nil {
    return err
}

dispatcher, err := schedulerqueue.New(durableQueue)
if err != nil {
    return err
}

runner, err := scheduler.NewRunner(
    registry,
    postgresLeases,
    dispatcher,
    scheduler.WithOwner(podName),
)
if err != nil {
    return err
}

return runner.Run(ctx)
```

Compile the immutable registry during startup so invalid expressions, duplicate
names, and unavailable time zones fail before the pod becomes ready. On
shutdown, cancel `Run` and call `Drain` with a deadline.

Laravel-style frequency helpers are available as interval constructors and
recurring constraints are schedule options:

```go
schedule, err := scheduler.NewSchedule(
    "weekday-sync",
    "accounts.sync",
    scheduler.EveryTenMinutes(),
    scheduler.WithWeekdays(),
    scheduler.WithBetween("8:00", "17:00"),
    scheduler.WithTimezone("America/Chicago"),
)
```

Laravel-compatible execution controls compose with those options:

```go
schedule, err := scheduler.NewSchedule(
    "weekday-sync",
    "accounts.sync",
    scheduler.Hourly(),
    scheduler.WithWeekdays(),
    scheduler.WithBetween("8:00", "17:00"),
    scheduler.WithTimezone("America/Chicago"),
    scheduler.WithoutOverlapping(10),
    scheduler.OnOneServer(),
    scheduler.RunInBackground(),
)
```

`WithoutOverlapping()` defaults to 1,440 minutes, while `OnOneServer()` uses an
independent one-hour occurrence lease. The `schedulerlease.Store` supplied to
`NewRunner` is the explicit equivalent of Laravel's `useCache`; all replicas
must receive the same PostgreSQL or Valkey store. Use CLI `clear-cache` only
after isolating any old executor that may still be performing side effects.

Custom cron expressions accept five fields or an optional leading seconds
field. See the [API reference](docs/api.md#frequency-and-constraints) for every
frequency helper and its Laravel mapping.

Applications own pause and resume triggers instead of invoking scheduler
commands. `PauseState` is suitable for one process; multi-replica deployments
should supply a shared persistent implementation of the narrow interfaces:

```go
pause := scheduler.NewPauseState()
runner, err := scheduler.NewRunner(
    registry,
    leases,
    executor,
    scheduler.WithOwner(podName),
    scheduler.WithPauseSource(pause),
)

// An authenticated endpoint, backpressure controller, or application command
// may call these idempotently.
_ = pause.Pause(ctx)
_ = pause.Resume(ctx)
```

Use `EvenWhenPaused()` only for operational schedules that must keep running.
Cancel the context passed to `Run`, then call `Drain`, to implement an external
deployment interrupt. `Registry.Overview(after)` provides deterministic list
data, including next runs, for any caller-owned CLI, HTTP, or admin surface.

## Scheduler or sequencer

Choose `scheduler` for recurring wall-clock admission: cron calculation,
missed-run selection, multi-replica occurrence leases, overlap policy, and
dispatch at a due instant. Choose
[`sequencer`](https://github.com/faustbrian/go-sequencer) for durable,
dependency-ordered one-time or explicitly repeatable operations with attempt
history and reconciliation.

Do not use `scheduler` to order migrations or deployment operations, and do not
use `sequencer` merely to calculate recurring cron boundaries. When an
application needs both, keep time admission in `scheduler` and hand an explicit
operation request to `sequencer`; this module does not create that dependency
or a shared runtime. Use Kubernetes CronJobs instead for isolated
infrastructure commands. Use `workflow` when durable workflow history, timers,
activities, and compensation are the actual requirement; see
[`go-workflow`](https://github.com/faustbrian/go-workflow).

## Package and adapter selection

The root `scheduler` package owns definitions, immutable compilation, due
selection, execution, and runner lifecycle. Select optional packages by the
boundary they adapt:

| Need | Package | Ownership boundary |
|---|---|---|
| cron parsing only | `cron` | compiles bounded expressions; starts no runner |
| lease contract | `adapters/lease` | defines fenced ownership without selecting storage |
| local deterministic coordination | `memory` | process-local state; not restart durable |
| shared PostgreSQL coordination | `postgres` | caller applies migrations and owns the pool |
| shared Valkey coordination | `valkey` | caller owns a Valkey 9 client and namespace |
| durable job dispatch | `adapters/queue` | adapts a caller-owned `go-queue` backend |
| dispatch idempotency | `adapters/idempotency` | adapts a caller-owned idempotency store |
| service lifecycle | `adapters/service` | adds runner and drain order to a `go-service` plan |
| HTTP administration | `adapters/http` | exposes inspection and recovery behind caller authentication |
| CLI administration | `adapters/cli` | writes bounded output through caller-owned streams |
| observations | `history`, `adapters/slog`, `adapters/otel` | retains bounded events or emits through caller facilities |
| deterministic tests | `schedulertest`, `lease/conformance` | provides fake time or validates a store implementation |

For a direct process, combine the root package, one lease store, and an
application `Executor`. For durable work, use `schedulerqueue.Dispatcher` as that
executor. For an application already using `go-service`, construct
`schedulerservice.Options` from `adapters/service` and include caller-owned lease
and queue facilities;
`go-service` cancels and joins the runner task, the adapter drains it, and
facilities then stop in reverse order. Add `adapters/http` or `adapters/cli`
only to an authenticated application control surface. See the
[service lifecycle](docs/service-integration.md), [lease](docs/leases.md),
[dispatch](docs/dispatch-and-idempotency.md), and [composition recipe](docs/composition-recipes.md)
contracts before selecting those paths.

The released `lease`, `queue`, `idempotency`, `schedulerservice`,
`schedulerhttp`, `schedulercli`, and `telemetry` paths remain supported
compatibility paths. New integrations should use the target-oriented `adapters/*` paths;
the split `slog` and `otel` observers can be selected independently.

## Documentation

Use the [documentation index](docs/README.md) for the API, migration,
Kubernetes, operations, security, compatibility, and troubleshooting.

For ecosystem-wide selection and ownership guidance, see the versioned
[Golib ecosystem index](https://github.com/faustbrian/go-library-tools/blob/v1.6.2/docs/ecosystem/README.md)
and its [Persistence and durability family](https://github.com/faustbrian/go-library-tools/blob/v1.6.2/docs/ecosystem/design-language.md#package-families-and-selection).

## Development

Run `make check`. PostgreSQL and Valkey conformance require the environment
variables described in [CONTRIBUTING.md](CONTRIBUTING.md).

## Project resources

- [API reference](docs/api.md)
- [Examples](examples/README.md)
- [FAQ](docs/faq.md) and [troubleshooting](docs/troubleshooting.md)
- [Compatibility](COMPATIBILITY.md), [deprecation policy](DEPRECATION.md), and
  [changelog](CHANGELOG.md)
- [Support](SUPPORT.md), [security policy](SECURITY.md), and [license](LICENSE)
