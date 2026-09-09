package main

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/faustbrian/go-scheduler/memory"
)

func TestRunSingletonDispatchesOnceAndDrainsBothReplicas(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	owner, err := runSingleton(ctx, memory.New())
	if err != nil {
		t.Fatalf("runSingleton() error = %v", err)
	}
	if owner != "scheduler-a" && owner != "scheduler-b" {
		t.Fatalf("dispatch owner = %q", owner)
	}
}

func TestRunWithStoreClosesCallerResourceAfterFailure(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	closed := false
	err := runWithStore(ctx, memory.New(), func() { closed = true }, io.Discard)
	if err == nil {
		t.Fatal("runWithStore() error = nil")
	}
	if !closed {
		t.Fatal("store resource was not closed")
	}
}
