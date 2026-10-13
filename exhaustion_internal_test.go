package scheduler

import (
	"testing"
	"time"
)

type finalOccurrenceSchedule struct{ at time.Time }

func (schedule finalOccurrenceSchedule) Next(after time.Time) time.Time {
	if after.Before(schedule.at) {
		return schedule.at
	}
	return time.Time{}
}

func TestDueRetainsAdmittedTailWhenTimingBecomesExhausted(t *testing.T) {
	after := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	boundary := after.Add(time.Minute)
	for _, policy := range []MissedRunPolicy{MissedRunOnce, MissedRunCatchUp} {
		definition, err := NewSchedule("finite", "task.finite", EveryMinute(), WithMissedRuns(policy, 2))
		if err != nil {
			t.Fatal(err)
		}
		registry := &Registry{entries: map[string]compiledSchedule{"finite": {schedule: definition, cron: finalOccurrenceSchedule{at: boundary}}}, names: []string{"finite"}}
		occurrences, err := registry.Due("finite", after, boundary.Add(time.Minute))
		if err != nil || len(occurrences) != 1 {
			t.Fatalf("policy %v: Due = %+v, %v; want one occurrence, nil", policy, occurrences, err)
		}
		if !occurrences[0].ScheduledAt.Equal(boundary) {
			t.Fatalf("boundary = %v; want %v", occurrences[0].ScheduledAt, boundary)
		}
	}
}

func TestRunnerNextRetainsValidTimingAndReportsCompleteExhaustion(t *testing.T) {
	after := time.Date(2026, time.January, 1, 0, 0, 30, 0, time.UTC)
	for _, validFirst := range []bool{true, false} {
		validName, exhaustedName := "a-valid", "z-exhausted"
		if !validFirst {
			validName, exhaustedName = "z-valid", "a-exhausted"
		}
		valid, err := NewSchedule(validName, "task.valid", EveryMinute())
		if err != nil {
			t.Fatal(err)
		}
		exhausted, err := NewSchedule(exhaustedName, "task.exhausted", Cron("0 0 31 2 *"))
		if err != nil {
			t.Fatal(err)
		}
		registry, err := Compile(valid, exhausted)
		if err != nil {
			t.Fatal(err)
		}
		runner := &Runner{registry: registry}
		next, ok := runner.next(after)
		if !ok || !next.Equal(after.Add(30*time.Second)) {
			t.Fatalf("valid-first=%t: next=%v,%t; want next minute", validFirst, next, ok)
		}
		exhaustedRegistry, err := Compile(exhausted)
		if err != nil {
			t.Fatal(err)
		}
		runner.registry = exhaustedRegistry
		if next, ok := runner.next(after); ok || !next.IsZero() {
			t.Fatalf("fully exhausted next=%v,%t; want zero,false", next, ok)
		}
	}
}
