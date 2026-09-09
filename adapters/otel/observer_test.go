package schedulerotel_test

import (
	"context"
	"errors"
	"testing"

	scheduler "github.com/faustbrian/go-scheduler"
	schedulerotel "github.com/faustbrian/go-scheduler/adapters/otel"
	gotelemetry "github.com/faustbrian/go-telemetry"
	"github.com/faustbrian/go-telemetry/testtelemetry"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

func TestObserverValidatesProvidersAndInstrumentConstruction(t *testing.T) {
	t.Parallel()

	tests := []schedulerotel.Config{
		{MeterProvider: metricnoop.NewMeterProvider()},
		{TracerProvider: tracenoop.NewTracerProvider()},
	}
	for _, config := range tests {
		if _, err := schedulerotel.New(config); !errors.Is(err, schedulerotel.ErrInvalidConfiguration) {
			t.Fatalf("New(%#v) error = %v", config, err)
		}
	}

	want := errors.New("instrument unavailable")
	provider := errorMeterProvider{
		MeterProvider: metricnoop.NewMeterProvider(),
		meter:         errorMeter{Meter: metricnoop.NewMeterProvider().Meter("test"), counterErr: want},
	}
	config := schedulerotel.Config{TracerProvider: tracenoop.NewTracerProvider(), MeterProvider: provider}
	if _, err := schedulerotel.New(config); !errors.Is(err, want) {
		t.Fatalf("New(counter error) = %v", err)
	}
	provider.meter = errorMeter{Meter: metricnoop.NewMeterProvider().Meter("test"), histogramErr: want}
	config.MeterProvider = provider
	if _, err := schedulerotel.New(config); !errors.Is(err, want) {
		t.Fatalf("New(histogram error) = %v", err)
	}
}

func TestObserverRecordsEveryLifecycleState(t *testing.T) {
	t.Parallel()

	harness := testtelemetry.New()
	t.Cleanup(func() { _ = harness.Shutdown(context.Background()) })
	observer, err := schedulerotel.New(schedulerotel.Config{
		TracerProvider: harness.TracerProvider(),
		MeterProvider:  harness.MeterProvider(),
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	occurrence := scheduler.Occurrence{IdempotencyKey: "occurrence", ScheduleName: "daily", Task: "reports"}
	observer.Observe(scheduler.Event{Type: scheduler.EventBefore, Occurrence: occurrence})
	observer.Observe(scheduler.Event{Type: scheduler.EventBefore, Occurrence: occurrence, Context: context.Background()})
	want := errors.New("failed")
	observer.Observe(scheduler.Event{Type: scheduler.EventFailure, Result: scheduler.ResultFailed, Occurrence: occurrence, Err: want})
	observer.Observe(scheduler.Event{Type: scheduler.EventCompleted, Result: scheduler.ResultFailed, Occurrence: occurrence, Err: want})
	succeeded := occurrence
	succeeded.IdempotencyKey = "succeeded"
	observer.Observe(scheduler.Event{Type: scheduler.EventBefore, Occurrence: succeeded})
	observer.Observe(scheduler.Event{Type: scheduler.EventCompleted, Result: scheduler.ResultSucceeded, Occurrence: succeeded})
	observer.Observe(scheduler.Event{Type: scheduler.EventFailure, Result: scheduler.ResultFailed, Occurrence: scheduler.Occurrence{IdempotencyKey: "missing"}, Err: want})
	observer.Observe(scheduler.Event{Type: scheduler.EventCompleted, Result: scheduler.ResultSucceeded, Occurrence: scheduler.Occurrence{IdempotencyKey: "missing"}})
	for _, eventType := range []scheduler.EventType{scheduler.EventSuccess, scheduler.EventSkipped, scheduler.EventOverlap, scheduler.EventFinished} {
		observer.Observe(scheduler.Event{Type: eventType, Occurrence: occurrence})
	}
	spans := harness.Spans()
	if got := len(spans); got != 3 {
		t.Fatalf("spans = %d, want 3", got)
	}
	if spans[0].Status.Code != codes.Error || spans[0].Status.Description != "superseded lifecycle" {
		t.Fatalf("superseded span status = %+v", spans[0].Status)
	}
	if spans[1].Status.Code != codes.Error || len(spans[1].Events) != 1 || spans[1].Events[0].Name != "exception" {
		t.Fatalf("failed span = status %+v events %+v", spans[1].Status, spans[1].Events)
	}
	if spans[2].Status.Code != codes.Ok {
		t.Fatalf("successful span status = %+v", spans[2].Status)
	}
	if _, err := harness.Metrics(context.Background()); err != nil {
		t.Fatalf("Metrics() error = %v", err)
	}
}

func TestObserverPreservesEventContext(t *testing.T) {
	t.Parallel()

	baseMeter := metricnoop.NewMeterProvider().Meter("test")
	baseCounter, err := baseMeter.Int64Counter("test")
	if err != nil {
		t.Fatalf("Int64Counter() error = %v", err)
	}
	counter := &contextCounter{Int64Counter: baseCounter}
	provider := contextMeterProvider{
		MeterProvider: metricnoop.NewMeterProvider(),
		meter:         contextMeter{Meter: baseMeter, counter: counter},
	}
	observer, err := schedulerotel.New(schedulerotel.Config{
		TracerProvider: tracenoop.NewTracerProvider(),
		MeterProvider:  provider,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "preserved")
	observer.Observe(scheduler.Event{Context: ctx})
	if got := counter.context.Value(contextKey{}); got != "preserved" {
		t.Fatalf("event context value = %v", got)
	}
}

func TestNewRuntimeUsesRuntimeProviders(t *testing.T) {
	t.Parallel()

	if _, err := schedulerotel.NewRuntime(nil); !errors.Is(err, schedulerotel.ErrInvalidConfiguration) {
		t.Fatalf("NewRuntime(nil) error = %v", err)
	}
	config := gotelemetry.DefaultConfig("scheduler-test", "1.0.0")
	config.Traces.Enabled = false
	config.Metrics.Enabled = false
	config.RegisterGlobal = false
	runtime, err := gotelemetry.Init(context.Background(), config)
	if err != nil {
		t.Fatalf("telemetry.Init() error = %v", err)
	}
	t.Cleanup(func() { _ = runtime.Shutdown(context.Background()) })
	if _, err := schedulerotel.NewRuntime(runtime); err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}
}

func TestObserverPreservesReleasedInstrumentationScope(t *testing.T) {
	t.Parallel()

	provider := &scopeMeterProvider{MeterProvider: metricnoop.NewMeterProvider()}
	if _, err := schedulerotel.New(schedulerotel.Config{
		TracerProvider: tracenoop.NewTracerProvider(),
		MeterProvider:  provider,
	}); err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if provider.scope != "github.com/faustbrian/go-scheduler" {
		t.Fatalf("instrumentation scope = %q", provider.scope)
	}
}

type errorMeterProvider struct {
	metric.MeterProvider
	meter metric.Meter
}

type contextMeterProvider struct {
	metric.MeterProvider
	meter metric.Meter
}

func (provider contextMeterProvider) Meter(string, ...metric.MeterOption) metric.Meter {
	return provider.meter
}

type contextMeter struct {
	metric.Meter
	counter metric.Int64Counter
}

func (meter contextMeter) Int64Counter(string, ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	return meter.counter, nil
}

type contextCounter struct {
	metric.Int64Counter
	context context.Context
}

func (counter *contextCounter) Add(ctx context.Context, value int64, options ...metric.AddOption) {
	counter.context = ctx
	counter.Int64Counter.Add(ctx, value, options...)
}

type scopeMeterProvider struct {
	metric.MeterProvider
	scope string
}

func (provider *scopeMeterProvider) Meter(scope string, options ...metric.MeterOption) metric.Meter {
	provider.scope = scope
	return provider.MeterProvider.Meter(scope, options...)
}

func (provider errorMeterProvider) Meter(string, ...metric.MeterOption) metric.Meter {
	return provider.meter
}

type errorMeter struct {
	metric.Meter
	histogramErr error
	counterErr   error
}

func (meter errorMeter) Float64Histogram(string, ...metric.Float64HistogramOption) (metric.Float64Histogram, error) {
	if meter.histogramErr != nil {
		return nil, meter.histogramErr
	}
	return meter.Meter.Float64Histogram("ok")
}

func (meter errorMeter) Int64Counter(string, ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	if meter.counterErr != nil {
		return nil, meter.counterErr
	}
	return meter.Meter.Int64Counter("ok")
}
