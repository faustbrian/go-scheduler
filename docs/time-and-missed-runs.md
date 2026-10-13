# Missed runs, time zones, and DST

[Documentation map](index.md) · [Repository README](../README.md)
[Next: API reference](api.md)

Cron expressions have five fields or an optional leading seconds field and use
explicit IANA time zones. Spring DST
gaps omit nonexistent local times. During a fold, both physical instants of an
ambiguous local time run in chronological order. Store schedule
times as instants and display local offsets in diagnostics.

`MissedRunSkip` executes only an occurrence exactly due at the current tick.
`MissedRunOnce` chooses the latest missed occurrence. `MissedRunCatchUp` returns
at most `MaxCatchUp` latest occurrences. Scans have a hard upper bound and fail
with `ErrOccurrenceLimit` rather than replaying unbounded downtime.

Jitter is a stable per-schedule offset in `[0, maximum)`. It is identical on all
replicas and preserves occurrence ordering. Wall-clock jumps are processed from
the last runner cursor to the injected clock's current instant.

A zero cron result means no future occurrence within the compiler's supported
search. Jitter preserves that sentinel. Registry Next/Overview expose zero for
exhausted timing, and Due stops without manufacturing a candidate or discarding
an already admitted tail. The runner ignores exhausted schedules when choosing
its next timer; if no enabled schedule has a next boundary, it waits for context
cancellation. Actual candidate scans still fail with `ErrOccurrenceLimit` when
they exceed `MaxOccurrenceScan`; exhaustion is not unbounded downtime.
