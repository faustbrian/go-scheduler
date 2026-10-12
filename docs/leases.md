# Lease backends and fencing

The memory backend is deterministic and process-local. It is appropriate for
tests and single-replica tools only.

The PostgreSQL adapter uses an owned table and `clock_timestamp()`. Rows become
inactive on release so their monotonic fencing token is retained. Apply
`postgres.SchemaMigration` before startup; its unqualified table follows the
connection search_path (commonly public), rather than enforcing public.
Services with a caller-owned schema apply `postgres.SchemaMigrationFor` and
construct the store with `postgres.NewWithSchema`. The schema itself must
already exist.

The Valkey adapter uses native `valkey-go` and atomic Lua scripts. `Open`
requires Valkey 9 or later and `maxmemory-policy noeviction`. A TTL-bound lease
hash and persistent same-slot counter prevent token recycling.

Occurrence leases implement one-owner dispatch. Task leases implement overlap
decisions. Expiry permits takeover after process death. An expired owner may
still be running, so ownership-sensitive writes must carry and validate the
fencing token. Manual recovery requires the current token and should be audited.

`OnOneServer()` uses a one-hour occurrence lease. `WithoutOverlapping()` uses a
renewable 24-hour task lease by default or accepts one positive expiration in
minutes. These TTLs remain independent when both options configure a schedule.
`WithOneServer` and `WithoutOverlap` accept explicit Go durations for callers
that need non-Laravel policies.

Persistent providers ignore the supplied now and use server time. PostgreSQL
rejects TTLs encoding less than one millisecond; Valkey accepts positive
sub-millisecond durations but truncates them to zero milliseconds, permitting
immediate expiry. Memory uses the supplied instant and retains expired entries.
PostgreSQL Inspect can return expired active rows; Valkey expiry can remove its
hash. None of these observations alone proves current execution isolation.
Clients and migrations remain caller-owned. Retained rows/counters prevent token
reuse but have no library-wide unique-key storage cap or cleanup worker.
