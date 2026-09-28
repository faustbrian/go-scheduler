package schedulerslog_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	scheduler "github.com/faustbrian/go-scheduler/v2"
	schedulerslog "github.com/faustbrian/go-scheduler/v2/adapters/slog"
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

func TestObserverLogLevelRequiresErrorAndFailedResult(t *testing.T) {
	t.Parallel()

	handler := &recordingHandler{}
	observer, err := schedulerslog.New(schedulerslog.Config{Logger: slog.New(handler)})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	for _, event := range []scheduler.Event{
		{Type: scheduler.EventSuccess, Result: scheduler.ResultSucceeded},
		{Type: scheduler.EventFailure, Result: scheduler.ResultFailed},
		{Type: scheduler.EventSuccess, Result: scheduler.ResultSucceeded, Err: errors.New("reported error")},
		{Type: scheduler.EventFailure, Result: scheduler.ResultFailed, Err: errors.New("execution failed")},
	} {
		observer.Observe(event)
	}

	want := []slog.Level{slog.LevelInfo, slog.LevelInfo, slog.LevelInfo, slog.LevelError}
	if len(handler.levels) != len(want) {
		t.Fatalf("logged levels = %v, want %v", handler.levels, want)
	}
	for i, level := range want {
		if handler.levels[i] != level {
			t.Fatalf("logged level for event %d = %v, want %v", i, handler.levels[i], level)
		}
	}
}

func TestObserverPreservesEventContext(t *testing.T) {
	t.Parallel()

	handler := &recordingHandler{}
	observer, err := schedulerslog.New(schedulerslog.Config{Logger: slog.New(handler)})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "request")
	observer.Observe(scheduler.Event{Context: ctx})
	observer.Observe(scheduler.Event{})

	if len(handler.contexts) != 2 {
		t.Fatalf("logged contexts = %d, want 2", len(handler.contexts))
	}
	if got := handler.contexts[0].Value(contextKey{}); got != "request" {
		t.Fatalf("supplied event context value = %v, want request", got)
	}
	if handler.contexts[1] == nil {
		t.Fatal("nil event context was not replaced")
	}
}

type recordingHandler struct {
	levels   []slog.Level
	contexts []context.Context
}

func (*recordingHandler) Enabled(context.Context, slog.Level) bool { return true }

func (handler *recordingHandler) Handle(ctx context.Context, record slog.Record) error {
	handler.levels = append(handler.levels, record.Level)
	handler.contexts = append(handler.contexts, ctx)
	return nil
}

func (handler *recordingHandler) WithAttrs([]slog.Attr) slog.Handler { return handler }

func (handler *recordingHandler) WithGroup(string) slog.Handler { return handler }
