package scheduleridempotency_test

import (
	"errors"
	"testing"

	scheduleridempotency "github.com/faustbrian/go-scheduler/adapters/idempotency"
)

func TestNewPreservesReleasedValidation(t *testing.T) {
	t.Parallel()
	if _, err := scheduleridempotency.New(nil, nil, scheduleridempotency.Options{}); !errors.Is(err, scheduleridempotency.ErrInvalidConfiguration) {
		t.Fatalf("New() error = %v", err)
	}
}
