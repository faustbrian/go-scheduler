# Examples

- [`basic`](basic/main.go) is a runnable single-process example using the
  memory lease backend and a short cooperative executor.
- [`distributed-singleton`](distributed-singleton/main.go) runs two contenders
  against one PostgreSQL or Valkey lease backend, observes one dispatch, then
  cancels and drains both runners.
- [`pause-state`](pause-state/main.go) composes a runner with process-local
  pause control; replace it with an application-owned shared pause source when
  every replica must observe the same admission decision.
- [`queue`](queue/example.go) wires an application-provided durable `queue`
  backend and distributed lease store into a production-shaped runner.

Run the released process-local example without cloning the repository:

```sh
go run github.com/faustbrian/go-scheduler/examples/basic@v1.0.0
```

From a source checkout, run `go run ./examples/basic`. The process waits for
scheduled boundaries until `Ctrl-C`, then drains with a deadline.

The distributed singleton example is a bounded contention rehearsal. Apply
`postgres.SchemaMigration()` before selecting PostgreSQL, or provide Valkey 9
with `maxmemory-policy noeviction`:

```sh
SCHEDULER_LEASE_BACKEND=postgres POSTGRES_URL='<dsn>' \
  go run ./examples/distributed-singleton

SCHEDULER_LEASE_BACKEND=valkey VALKEY_ADDRESS='<host:port>' \
  go run ./examples/distributed-singleton
```

Each successful run prints one winning owner after two contenders dispatch one
physical occurrence and both runners drain. The caller owns backend creation,
credentials, migrations, availability, and final pool or client closure.

For production durable work, pass application-owned `queue.Enqueuer` and
`lease.Store` implementations to `queueexample.NewRunner`; the package is a
construction example rather than a command. Continue with the
[package map](../README.md#package-and-adapter-selection),
[service lifecycle](../docs/service-integration.md), and
[dispatch ownership](../docs/dispatch-and-idempotency.md) guides.
