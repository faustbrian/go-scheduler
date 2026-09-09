package schedulerqueue_test

import (
	"errors"
	"testing"

	schedulerqueue "github.com/faustbrian/go-scheduler/adapters/queue"
)

func TestNewPreservesReleasedValidation(t *testing.T) {
	t.Parallel()
	if _, err := schedulerqueue.New(nil); !errors.Is(err, schedulerqueue.ErrInvalidQueue) {
		t.Fatalf("New() error = %v", err)
	}
}
