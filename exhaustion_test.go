package scheduler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	scheduler "github.com/faustbrian/go-scheduler/v2"
	"github.com/faustbrian/go-scheduler/v2/memory"
	"github.com/faustbrian/go-scheduler/v2/schedulertest"
)

func exhaustedRegistry(t *testing.T, policy scheduler.MissedRunPolicy, jitter time.Duration) *scheduler.Registry {
	t.Helper()
	definition, err := scheduler.NewSchedule("exhausted", "task.exhausted", scheduler.Cron("0 0 31 2 *"), scheduler.WithJitter(jitter), scheduler.WithMissedRuns(policy, 2))
	if err != nil {
		t.Fatal(err)
	}
	registry, err := scheduler.Compile(definition)
	if err != nil {
		t.Fatal(err)
	}
	return registry
}

func TestRegistryPreservesExhaustedNextWithJitter(t *testing.T) {
	after := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	for _, jitter := range []time.Duration{0, time.Hour} {
		t.Run(jitter.String(), func(t *testing.T) {
			registry := exhaustedRegistry(t, scheduler.MissedRunOnce, jitter)
			next, err := registry.Next("exhausted", after)
			if err != nil || !next.IsZero() {
				t.Fatalf("Next = %v, %v; want zero, nil", next, err)
			}
			if next := registry.Overview(after)[0].Next; !next.IsZero() {
				t.Fatalf("Overview Next = %v; want zero", next)
			}
		})
	}
}

// Native package timeouts bound these synchronous timing regressions. The
// historical replay hang is separately proved in a killable pre-fix test binary.
func TestRegistryDueTerminatesOnExhaustedSchedule(t *testing.T) {
	after := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	through := after.Add(time.Hour)
	for _, policy := range []scheduler.MissedRunPolicy{scheduler.MissedRunSkip, scheduler.MissedRunOnce, scheduler.MissedRunCatchUp} {
		for _, jitter := range []time.Duration{0, time.Hour} {
			t.Run(fmt.Sprintf("%d/%s", policy, jitter), func(t *testing.T) {
				occurrences, err := exhaustedRegistry(t, policy, jitter).Due("exhausted", after, through)
				if err != nil || len(occurrences) != 0 {
					t.Fatalf("Due = %+v, %v; want no occurrences, nil", occurrences, err)
				}
			})
		}
	}
}

func TestRegistryDuePreservesZeroBoundarySentinel(t *testing.T) {
	through := time.Time{}
	occurrences, err := exhaustedRegistry(t, scheduler.MissedRunSkip, 0).Due("exhausted", through.Add(-time.Hour), through)
	if err != nil || len(occurrences) != 0 {
		t.Fatalf("Due at zero = %+v, %v; want no occurrences, nil", occurrences, err)
	}
}

func TestRunnerIgnoresExhaustedSchedulesWhenSelectingNext(t *testing.T) {
	for _, validFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("valid-first=%t", validFirst), func(t *testing.T) { testRunnerWithExhaustedSchedule(t, validFirst) })
	}
}

func testRunnerWithExhaustedSchedule(t *testing.T, validFirst bool) {
	t.Helper()
	start := time.Date(2026, time.January, 1, 0, 0, 30, 0, time.UTC)
	clock := schedulertest.NewFakeClock(start)
	validName, exhaustedName := "a-valid", "z-exhausted"
	if !validFirst {
		validName, exhaustedName = "z-valid", "a-exhausted"
	}
	valid, err := scheduler.NewSchedule(validName, "task.valid", scheduler.EveryMinute())
	if err != nil {
		t.Fatal(err)
	}
	exhausted, err := scheduler.NewSchedule(exhaustedName, "task.exhausted", scheduler.Cron("0 0 31 2 *"))
	if err != nil {
		t.Fatal(err)
	}
	registry, err := scheduler.Compile(valid, exhausted)
	if err != nil {
		t.Fatal(err)
	}
	dispatched := make(chan scheduler.Context, 2)
	runner, err := scheduler.NewRunner(registry, memory.New(), executorFunc(func(_ context.Context, scheduled scheduler.Context) error { dispatched <- scheduled; return nil }), scheduler.WithOwner("exhaustion-test"), scheduler.WithClock(clock))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runner.Run(ctx) }()
	defer func() {
		cancel()
		select {
		case err := <-done:
			if !errors.Is(err, context.Canceled) {
				t.Errorf("Run = %v; want context.Canceled", err)
			}
		case <-time.After(time.Second):
			t.Error("Run did not stop after cancellation")
		}
		drainCtx, drainCancel := context.WithTimeout(context.Background(), time.Second)
		defer drainCancel()
		if err := runner.Drain(drainCtx); err != nil {
			t.Errorf("Drain: %v", err)
		}
		clock.Advance(time.Hour)
	}()
	waitCtx, waitCancel := context.WithTimeout(context.Background(), time.Second)
	defer waitCancel()
	if !clock.WaitForTimers(waitCtx, 1) {
		t.Fatal("exhausted schedule suppressed the valid timer")
	}
	clock.Advance(30 * time.Second)
	select {
	case scheduled := <-dispatched:
		if scheduled.Schedule.Task != "task.valid" || !scheduled.Due.Equal(start.Add(30*time.Second)) {
			t.Fatalf("unexpected dispatch: %+v", scheduled)
		}
	case <-time.After(time.Second):
		t.Fatal("valid occurrence was not dispatched")
	}
	select {
	case scheduled := <-dispatched:
		t.Fatalf("unexpected extra dispatch: %+v", scheduled)
	default:
	}
}
