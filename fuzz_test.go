package scheduler_test

import (
	"testing"
	"time"

	scheduler "github.com/faustbrian/go-scheduler/v3"
)

// FuzzScheduleCompilation exercises hostile names/cron/zone admission and a
// fixed-instant jittered Next for future-or-zero timing and panic safety; it is
// not a separate cron oracle.
func FuzzScheduleCompilation(f *testing.F) {
	f.Add("report", "task", "0 0 * * *", "UTC")
	f.Add("dst", "task", "30 3 * * *", "Europe/Helsinki")
	f.Add("exhausted", "task", "0 0 31 2 *", "UTC")
	f.Fuzz(func(t *testing.T, name, task, expression, timezone string) {
		schedule, err := scheduler.NewSchedule(name, task, scheduler.Cron(expression), scheduler.WithTimezone(timezone), scheduler.WithJitter(time.Hour))
		if err != nil {
			return
		}
		registry, err := scheduler.Compile(schedule)
		if err != nil {
			return
		}
		after := time.Unix(1_700_000_000, 0)
		next, err := registry.Next(name, after)
		if err != nil {
			t.Fatal(err)
		}
		if !next.IsZero() && !next.After(after) {
			t.Fatalf("Next = %v; want a future boundary or zero", next)
		}
	})
}

// FuzzScheduleOptions exercises metadata, jitter and missed-run policy bounds
// for panic safety; application callback behavior is outside its input space.
func FuzzScheduleOptions(f *testing.F) {
	f.Add("owner", "value", int64(0), uint8(0))
	f.Fuzz(func(_ *testing.T, metadataKey, metadataValue string, jitterNanos int64, policy uint8) {
		_, _ = scheduler.NewSchedule(
			"fuzz", "task", scheduler.Daily(),
			scheduler.WithMetadata(map[string]string{metadataKey: metadataValue}),
			scheduler.WithJitter(time.Duration(jitterNanos)),
			scheduler.WithMissedRuns(scheduler.MissedRunPolicy(policy), 3),
		)
	})
}
