package schedulerservice_test

//lint:file-ignore SA1019 This file intentionally exercises or implements the retained compatibility contract.

import (
	"context"
	"fmt"

	"github.com/faustbrian/go-correlation"
	"github.com/faustbrian/go-scheduler"
	"github.com/faustbrian/go-scheduler/memory"
	"github.com/faustbrian/go-scheduler/schedulerservice" //nolint:staticcheck // Exercises the retained compatibility package.
)

func ExampleNew() {
	schedule, _ := scheduler.NewSchedule(
		"nightly-report",
		"reports.generate",
		scheduler.Daily(),
	)
	registry, _ := scheduler.Compile(schedule)
	factory, _ := correlation.NewFactory(correlation.FactoryOptions{})
	adapter, err := schedulerservice.New(schedulerservice.Options{
		Name:        "scheduler",
		Registry:    registry,
		Leases:      memory.New(),
		Executor:    executorFunc(func(context.Context, scheduler.Context) error { return nil }),
		Correlation: factory,
		RunnerOptions: []scheduler.RunnerOption{
			scheduler.WithOwner("replica-a"),
		},
	})
	if err != nil {
		fmt.Println("scheduler setup failed")

		return
	}

	plan := adapter.Plan()
	fmt.Println(plan.Tasks[0].Name, len(plan.Components))
	// Output: scheduler 1
}
