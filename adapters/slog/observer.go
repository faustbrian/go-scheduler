// Package schedulerslog records scheduler lifecycle events with log/slog.
package schedulerslog

import (
	"context"
	"errors"
	"log/slog"

	scheduler "github.com/faustbrian/go-scheduler"
)

// ErrInvalidConfiguration reports a missing logger.
var ErrInvalidConfiguration = errors.New("scheduler slog: logger is required")

// Config supplies the application-owned logger.
type Config struct {
	Logger *slog.Logger
}

// Observer records bounded scheduler lifecycle fields. It starts no goroutines
// and retains neither event contexts nor event values.
type Observer struct {
	logger *slog.Logger
}

// New constructs a structured lifecycle observer.
func New(config Config) (*Observer, error) {
	if config.Logger == nil {
		return nil, ErrInvalidConfiguration
	}
	return &Observer{logger: config.Logger}, nil
}

// Observe records one scheduler lifecycle event.
func (observer *Observer) Observe(event scheduler.Event) {
	ctx := eventContext(event.Context)
	observer.logger.LogAttrs(
		ctx,
		logLevel(event),
		"scheduler lifecycle",
		slog.String("event", event.Type.String()),
		slog.String("result", event.Result.String()),
		slog.String("schedule", event.Occurrence.ScheduleName),
		slog.String("task", event.Occurrence.Task),
		slog.String("owner", event.Owner),
		slog.Uint64("fencing_token", event.Fencing),
		slog.Bool("background", event.Background),
		slog.Any("error", event.Err),
	)
}

func eventContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func logLevel(event scheduler.Event) slog.Level {
	if event.Err != nil && event.Result == scheduler.ResultFailed {
		return slog.LevelError
	}
	return slog.LevelInfo
}
