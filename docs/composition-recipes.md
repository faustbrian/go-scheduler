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
