// Command distributed-singleton runs two scheduler replicas against one
// caller-selected PostgreSQL or Valkey lease store and proves that one physical
// occurrence is dispatched once before both runners shut down.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	scheduler "github.com/faustbrian/go-scheduler"
	"github.com/faustbrian/go-scheduler/lease"
	schedulerpostgres "github.com/faustbrian/go-scheduler/postgres"
	schedulervalkey "github.com/faustbrian/go-scheduler/valkey"
	"github.com/jackc/pgx/v5/pgxpool"
	valkeygo "github.com/valkey-io/valkey-go"
)

const (
	backendOperationTimeout = 5 * time.Second
	shutdownTimeout         = 10 * time.Second
)

type executorFunc func(context.Context, scheduler.Context) error

func (execute executorFunc) Execute(ctx context.Context, scheduled scheduler.Context) error {
	return execute(ctx, scheduled)
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	store, closeStore, err := openStore(ctx)
	if err != nil {
		return err
	}
	return runWithStore(ctx, store, closeStore, os.Stdout)
}

func runWithStore(ctx context.Context, store lease.Store, closeStore func(), output io.Writer) error {
	defer closeStore()
	owner, err := runSingleton(ctx, store)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "dispatched by %s\n", owner)
	return err
}

func runSingleton(ctx context.Context, store lease.Store) (owner string, resultErr error) {
	schedule, err := scheduler.NewSchedule(
		"daily-ledger-close",
		"ledger.close",
		scheduler.EveryMinute(),
		scheduler.WithMissedRuns(scheduler.MissedRunOnce, 0),
		scheduler.WithOneServer(time.Minute),
		scheduler.RunInBackground(),
	)
	if err != nil {
		return "", err
	}
	registry, err := scheduler.Compile(schedule)
	if err != nil {
		return "", err
	}

	dispatched := make(chan scheduler.Context, 2)
	executionDone := make(chan error, 2)
	var dispatchCount atomic.Int64
	executor := executorFunc(func(runCtx context.Context, scheduled scheduler.Context) error {
		dispatchCount.Add(1)
		dispatched <- scheduled
		<-runCtx.Done()
		executionDone <- runCtx.Err()
		return runCtx.Err()
	})

	runners := make([]*scheduler.Runner, 0, 2)
	for _, owner := range []string{"scheduler-a", "scheduler-b"} {
		runner, runnerErr := scheduler.NewRunner(
			registry,
			store,
			executor,
			scheduler.WithOwner(owner),
		)
		if runnerErr != nil {
			return "", runnerErr
		}
		runners = append(runners, runner)
	}

	runCtx, cancelRun := context.WithCancel(ctx)
	drained := false
	defer func() {
		cancelRun()
		if !drained {
			resultErr = errors.Join(resultErr, drainRunners(runners))
		}
	}()
	through := time.Now().UTC().Truncate(time.Minute)
	tickErrors := make(chan error, len(runners))
	var ticks sync.WaitGroup
	for _, runner := range runners {
		ticks.Add(1)
		go func() {
			defer ticks.Done()
			tickErrors <- runner.Tick(runCtx, through.Add(-time.Minute), through)
		}()
	}
	ticks.Wait()
	close(tickErrors)
	for tickErr := range tickErrors {
		if tickErr != nil {
			return "", tickErr
		}
	}

	var scheduled scheduler.Context
	select {
	case scheduled = <-dispatched:
	case <-ctx.Done():
		return "", ctx.Err()
	}
	cancelRun()

	if err := drainRunners(runners); err != nil {
		return "", err
	}
	drained = true
	for _, runner := range runners {
		if tickErr := runner.Tick(
			context.Background(), through, through.Add(time.Minute),
		); !errors.Is(tickErr, scheduler.ErrDraining) {
			return "", fmt.Errorf("post-drain tick: %w", tickErr)
		}
	}
	if executionErr := <-executionDone; !errors.Is(executionErr, context.Canceled) {
		return "", fmt.Errorf("execution cancellation: %w", executionErr)
	}
	if dispatchCount.Load() != 1 {
		return "", fmt.Errorf("dispatch count = %d, want 1", dispatchCount.Load())
	}
	select {
	case duplicate := <-dispatched:
		return "", fmt.Errorf("duplicate dispatch by %s", duplicate.Owner)
	default:
	}
	return scheduled.Owner, nil
}

func drainRunners(runners []*scheduler.Runner) error {
	errs := make([]error, 0, len(runners))
	for _, runner := range runners {
		drainCtx, cancelDrain := context.WithTimeout(context.Background(), shutdownTimeout)
		errs = append(errs, runner.Drain(drainCtx))
		cancelDrain()
	}
	return errors.Join(errs...)
}

func openStore(ctx context.Context) (lease.Store, func(), error) {
	operationCtx, cancelOperation := context.WithTimeout(ctx, backendOperationTimeout)
	defer cancelOperation()

	switch os.Getenv("SCHEDULER_LEASE_BACKEND") {
	case "postgres":
		databaseURL := os.Getenv("POSTGRES_URL")
		if databaseURL == "" {
			return nil, nil, errors.New("POSTGRES_URL is required")
		}
		pool, err := pgxpool.New(operationCtx, databaseURL)
		if err != nil {
			return nil, nil, fmt.Errorf("open PostgreSQL pool: %w", err)
		}
		if err := pool.Ping(operationCtx); err != nil {
			pool.Close()
			return nil, nil, fmt.Errorf("ping PostgreSQL: %w", err)
		}
		store, err := schedulerpostgres.New(pool)
		if err != nil {
			pool.Close()
			return nil, nil, err
		}
		return store, pool.Close, nil
	case "valkey":
		address := os.Getenv("VALKEY_ADDRESS")
		if address == "" {
			return nil, nil, errors.New("VALKEY_ADDRESS is required")
		}
		client, err := valkeygo.NewClient(valkeygo.ClientOption{InitAddress: []string{address}})
		if err != nil {
			return nil, nil, fmt.Errorf("open Valkey client: %w", err)
		}
		store, err := schedulervalkey.Open(operationCtx, client, "scheduler-singleton-example")
		if err != nil {
			client.Close()
			return nil, nil, err
		}
		return store, client.Close, nil
	default:
		return nil, nil, errors.New("SCHEDULER_LEASE_BACKEND must be postgres or valkey")
	}
}
