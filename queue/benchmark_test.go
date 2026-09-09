package queue_test

//lint:file-ignore SA1019 This file intentionally exercises or implements the retained compatibility contract.

import (
	"context"
	"testing"

	scheduler "github.com/faustbrian/go-scheduler"
	schedulerqueue "github.com/faustbrian/go-scheduler/queue" //nolint:staticcheck // Measures compatibility behavior.
)

func BenchmarkDispatchEnvelope(b *testing.B) {
	backend := &fakeQueue{}
	dispatcher, _ := schedulerqueue.New(backend)
	schedule, _ := scheduler.NewSchedule(
		"report", "reports.generate", scheduler.Daily(),
		scheduler.WithParameters(map[string]any{"tenant": "acme"}),
	)
	scheduled := scheduler.Context{Schedule: schedule, IdempotencyKey: "key"}
	b.ResetTimer()
	for range b.N {
		if err := dispatcher.Execute(context.Background(), scheduled); err != nil {
			b.Fatal(err)
		}
	}
}
