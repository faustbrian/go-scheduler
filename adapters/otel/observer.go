// Package schedulerotel records scheduler lifecycle metrics and traces with
// OpenTelemetry.
package schedulerotel

import (
	"context"
	"errors"
	"sync"
	"time"

	scheduler "github.com/faustbrian/go-scheduler"
	gotelemetry "github.com/faustbrian/go-telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Keep the released instrumentation scope so adopting the target-oriented
// package does not split dashboards, alerts, or trace queries.
const scopeName = "github.com/faustbrian/go-scheduler"

// ErrInvalidConfiguration reports missing OpenTelemetry providers.
var ErrInvalidConfiguration = errors.New("scheduler otel: providers are required")

// Config supplies application-owned OpenTelemetry providers.
type Config struct {
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
}

type activeSpan struct {
	span    trace.Span
	started time.Time
}

// Observer records lifecycle metrics and execution spans.
type Observer struct {
	tracer   trace.Tracer
	events   metric.Int64Counter
	duration metric.Float64Histogram
	mu       sync.Mutex
	active   map[string]activeSpan
}

// New constructs a lifecycle telemetry observer.
func New(config Config) (*Observer, error) {
	if config.TracerProvider == nil || config.MeterProvider == nil {
		return nil, ErrInvalidConfiguration
	}
	meter := config.MeterProvider.Meter(scopeName)
	events, err := meter.Int64Counter("scheduler.events")
	if err != nil {
		return nil, err
	}
	duration, err := meter.Float64Histogram("scheduler.execution.duration", metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}
	return &Observer{
		tracer: config.TracerProvider.Tracer(scopeName),
		events: events, duration: duration, active: make(map[string]activeSpan),
	}, nil
}

// NewRuntime constructs an observer from an initialized telemetry runtime.
func NewRuntime(runtime *gotelemetry.Runtime) (*Observer, error) {
	if runtime == nil {
		return nil, ErrInvalidConfiguration
	}
	return New(Config{
		TracerProvider: runtime.TracerProvider(), MeterProvider: runtime.MeterProvider(),
	})
}

// Observe records one scheduler lifecycle event.
func (observer *Observer) Observe(event scheduler.Event) {
	ctx := eventContext(event.Context)
	attributes := []attribute.KeyValue{
		attribute.String("scheduler.event", event.Type.String()),
		attribute.String("scheduler.result", event.Result.String()),
		attribute.String("scheduler.schedule", event.Occurrence.ScheduleName),
		attribute.String("scheduler.task", event.Occurrence.Task),
		attribute.Bool("scheduler.background", event.Background),
	}
	observer.events.Add(ctx, 1, metric.WithAttributes(attributes...))
	key := event.Occurrence.IdempotencyKey
	switch event.Type {
	case scheduler.EventBefore:
		spanCtx, span := observer.tracer.Start(ctx, "scheduler.execute", trace.WithAttributes(attributes...))
		_ = spanCtx
		observer.mu.Lock()
		if previous, ok := observer.active[key]; ok {
			previous.span.SetStatus(codes.Error, "superseded lifecycle")
			previous.span.End()
		}
		observer.active[key] = activeSpan{span: span, started: time.Now()}
		observer.mu.Unlock()
	case scheduler.EventFailure:
		observer.withSpan(key, func(span activeSpan) {
			if event.Err != nil {
				span.span.RecordError(event.Err)
			}
			span.span.SetStatus(codes.Error, "execution failed")
		})
	case scheduler.EventCompleted:
		observer.mu.Lock()
		span, ok := observer.active[key]
		delete(observer.active, key)
		observer.mu.Unlock()
		if ok {
			if event.Result == scheduler.ResultFailed {
				span.span.SetStatus(codes.Error, "execution failed")
			} else {
				span.span.SetStatus(codes.Ok, event.Result.String())
			}
			span.span.End()
			observer.duration.Record(ctx, time.Since(span.started).Seconds(), metric.WithAttributes(attributes...))
		}
	case scheduler.EventSuccess, scheduler.EventSkipped, scheduler.EventOverlap, scheduler.EventFinished:
	}
}

func eventContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func (observer *Observer) withSpan(key string, callback func(activeSpan)) {
	observer.mu.Lock()
	span, ok := observer.active[key]
	observer.mu.Unlock()
	if ok {
		callback(span)
	}
}
