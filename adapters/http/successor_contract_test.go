package schedulerhttp_test

import (
	"errors"
	"testing"

	schedulerhttp "github.com/faustbrian/go-scheduler/v2/adapters/http"
)

func TestNewPreservesReleasedValidation(t *testing.T) {
	t.Parallel()
	if _, err := schedulerhttp.New(nil, nil); !errors.Is(err, schedulerhttp.ErrInvalidDependencies) {
		t.Fatalf("New() error = %v", err)
	}
}
