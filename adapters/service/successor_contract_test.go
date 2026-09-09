package schedulerservice_test

import (
	"errors"
	"testing"

	schedulerservice "github.com/faustbrian/go-scheduler/adapters/service"
)

func TestNewPreservesReleasedValidation(t *testing.T) {
	t.Parallel()
	if _, err := schedulerservice.New(schedulerservice.Options{}); !errors.Is(err, schedulerservice.ErrInvalidOptions) {
		t.Fatalf("New() error = %v", err)
	}
}
