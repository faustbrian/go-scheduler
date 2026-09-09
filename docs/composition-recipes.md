# Composition recipe: pause-aware runner

Use `NewPauseState` with `WithPauseSource` when a process-local administrative
pause is sufficient. The runner owns schedule execution and lease cleanup; the
pause state owns only the atomic paused flag and never starts a goroutine.

```go
paused := scheduler.NewPauseState()
runner, err := scheduler.NewRunner(
    registry,
    memory.New(),
    applicationExecutor,
    scheduler.WithOwner("worker-1"),
    scheduler.WithPauseSource(paused),
)
if err != nil { return err }

if err := paused.Pause(ctx); err != nil { return err }
// Ordinary schedules are skipped with scheduler.ErrPaused while paused.
if err := paused.Resume(ctx); err != nil { return err }
```

`PauseState` is application-owned: keep the pointer for administrative control
and do not construct a new instance per tick. It is process-local, so each
replica needs an application-owned shared `PauseSource`/`PauseController` when
pause state must be consistent across replicas. The runner owns the lease store,
execution cancellation, and final `Drain`; callers own the pause state and must
not close or release it. `Pause`, `Resume`, and `Paused` honor context
cancellation and are safe to call repeatedly.

Schedules configured with `scheduler.EvenWhenPaused()` intentionally bypass the
pause source for emergency or maintenance work.

## Distributed singleton rehearsal

[`examples/distributed-singleton`](../examples/distributed-singleton/main.go)
uses only public APIs to run two replicas against one shared PostgreSQL or
Valkey lease store. Both contenders tick the same physical occurrence, but the
fenced occurrence lease permits one dispatch. The example then cancels the
winning execution, calls bounded `Drain` on both runners, and verifies that
draining has closed admission to later ticks.

Set `SCHEDULER_LEASE_BACKEND=postgres` and `POSTGRES_URL`, or set
`SCHEDULER_LEASE_BACKEND=valkey` and `VALKEY_ADDRESS`, before running:

```sh
go run ./examples/distributed-singleton
```

The application owns and closes the PostgreSQL pool or Valkey client after both
runners drain. PostgreSQL callers must apply `postgres.SchemaMigration()`
through their migration owner before startup. Valkey callers must provide
Valkey 9 or newer with `maxmemory-policy noeviction`. Every deployed replica
must use the same authoritative PostgreSQL table or Valkey namespace and a
unique runner owner. The occurrence lease is intentionally retained until its
TTL expires so another replica cannot dispatch the same physical occurrence.

This is an executable contention and shutdown rehearsal, not a durable-work or
exactly-once claim. Production jobs should use the queue dispatcher and an
idempotency boundary; backend availability, credentials, TLS, high
availability, and migration rollback remain application/operator owned.
