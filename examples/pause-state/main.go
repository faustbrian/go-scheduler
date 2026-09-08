// Command pause-state composes a scheduler runner with process-local pause
// control. Replace PauseState with an application-owned shared implementation
// when more than one replica must observe the same pause decision.
package main

import (
	"context"
	"errors"
	"log"
	"os/signal"
	"syscall"
	"time"

	scheduler "github.com/faustbrian/go-scheduler"
	"github.com/faustbrian/go-scheduler/memory"
)

type executor func(context.Context, scheduler.Context) error

func (execute executor) Execute(ctx context.Context, scheduled scheduler.Context) error {
	return execute(ctx, scheduled)
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	paused := scheduler.NewPauseState()
	schedule, err := scheduler.NewSchedule(
		"heartbeat", "service.heartbeat", scheduler.EveryMinute(),
		scheduler.WithoutOverlap(scheduler.OverlapSkip, time.Minute),
	)
	if err != nil {
		log.Fatal(err)
	}
	registry, err := scheduler.Compile(schedule)
	if err != nil {
		log.Fatal(err)
	}
	runner, err := scheduler.NewRunner(
		registry, memory.New(), executor(func(_ context.Context, due scheduler.Context) error {
			log.Printf("running %s at %s", due.Schedule.Name, due.Due)
			return nil
		}), scheduler.WithOwner("pause-state-example"),
		scheduler.WithPauseSource(paused),
	)
	if err != nil {
		log.Fatal(err)
	}
	if err := runner.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatal(err)
	}
	drainCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := runner.Drain(drainCtx); err != nil {
		log.Fatal(err)
	}
}
