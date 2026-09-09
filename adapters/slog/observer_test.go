package schedulerslog_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	scheduler "github.com/faustbrian/go-scheduler"
	schedulerslog "github.com/faustbrian/go-scheduler/adapters/slog"
)

func TestObserverValidatesLoggerAndRecordsLifecycleLevels(t *testing.T) {
	t.Parallel()

	if _, err := schedulerslog.New(schedulerslog.Config{}); !errors.Is(err, schedulerslog.ErrInvalidConfiguration) {
		t.Fatalf("New(empty) error = %v", err)
	}

	var output bytes.Buffer
	observer, err := schedulerslog.New(schedulerslog.Config{
		Logger: slog.New(slog.NewJSONHandler(&output, nil)),
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	occurrence := scheduler.Occurrence{ScheduleName: "daily", Task: "reports"}
	observer.Observe(scheduler.Event{
		Type: scheduler.EventSuccess, Result: scheduler.ResultSucceeded,
		Occurrence: occurrence,
	})
	observer.Observe(scheduler.Event{
		Type: scheduler.EventFailure, Result: scheduler.ResultFailed,
		Occurrence: occurrence, Context: context.Background(), Err: errors.New("failed"),
	})

	logs := output.String()
	for _, want := range []string{`"level":"INFO"`, `"level":"ERROR"`, `"schedule":"daily"`, `"task":"reports"`} {
		if !strings.Contains(logs, want) {
			t.Fatalf("logs do not contain %s: %s", want, logs)
		}
	}
}
