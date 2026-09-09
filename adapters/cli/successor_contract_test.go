package schedulercli_test

import (
	"context"
	"testing"

	schedulercli "github.com/faustbrian/go-scheduler/adapters/cli"
)

func TestRunPreservesReleasedValidation(t *testing.T) {
	t.Parallel()
	if code := schedulercli.Run(context.Background(), nil, nil, nil, nil, nil); code != 2 {
		t.Fatalf("Run() code = %d, want 2", code)
	}
}
