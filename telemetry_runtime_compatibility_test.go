package scheduler_test

//lint:file-ignore SA1019 This file exercises the retained telemetry compatibility contract.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	scheduler "github.com/faustbrian/go-scheduler/v2"
	schedulerotel "github.com/faustbrian/go-scheduler/v2/adapters/otel"
	"github.com/faustbrian/go-scheduler/v2/memory"
	schedulertelemetry "github.com/faustbrian/go-scheduler/v2/telemetry" //nolint:staticcheck // Retained public compatibility path.
	gotelemetry "github.com/faustbrian/go-telemetry/v2"
	"github.com/faustbrian/go-telemetry/v2/testtelemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestTelemetryRuntimeExportsSchedulerLifecycles(t *testing.T) {
	for _, compatibility := range []bool{false, true} {
		name := "canonical"
		if compatibility {
			name = "compatibility"
		}
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			spans := &schedulerSpanExporter{InMemoryExporter: tracetest.NewInMemoryExporter()}
			metrics := &schedulerMetricExporter{}
			config := gotelemetry.DefaultConfig("scheduler-fixture", "fixture-version")
			config.Traces.Enabled = true
			config.Metrics.Enabled = true
			config.Traces.Sampler.Ratio = 1
			config.Traces.Batch.BatchTimeout = time.Hour
			config.Metrics.ExportInterval = time.Hour
			runtime, err := gotelemetry.Init(ctx, config,
				gotelemetry.WithTraceExporter(spans), gotelemetry.WithMetricExporter(metrics))
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := runtime.Shutdown(ctx); err != nil {
					t.Errorf("Shutdown: %v", err)
				}
			}()
			var logs bytes.Buffer
			var observer scheduler.Observer
			if compatibility {
				observer, err = schedulertelemetry.NewRuntime(runtime, slog.New(slog.NewJSONHandler(&logs, nil)))
			} else {
				observer, err = schedulerotel.NewRuntime(runtime)
			}
			if err != nil {
				t.Fatal(err)
			}
			parent := trace.NewSpanContext(trace.SpanContextConfig{
				TraceID: trace.TraceID{1}, SpanID: trace.SpanID{2}, TraceFlags: trace.FlagsSampled,
			})
			eventContext := trace.ContextWithSpanContext(ctx, parent)
			failed := scheduler.Occurrence{ScheduleName: "fixture", Task: "fixture.task", IdempotencyKey: "failed"}
			success := failed
			success.IdempotencyKey = "success"
			missing := failed
			missing.IdempotencyKey = "missing"
			failure := errors.New("fixture failure")
			events := []scheduler.Event{
				{Type: scheduler.EventBefore, Occurrence: failed},
				{Type: scheduler.EventBefore, Occurrence: failed},
				{Type: scheduler.EventFailure, Result: scheduler.ResultFailed, Occurrence: failed, Err: failure},
				{Type: scheduler.EventCompleted, Result: scheduler.ResultFailed, Occurrence: failed, Err: failure},
				{Type: scheduler.EventCompleted, Result: scheduler.ResultFailed, Occurrence: failed, Err: failure},
				{Type: scheduler.EventBefore, Occurrence: success},
				{Type: scheduler.EventCompleted, Result: scheduler.ResultSucceeded, Occurrence: success},
				{Type: scheduler.EventCompleted, Result: scheduler.ResultSucceeded, Occurrence: success},
				{Type: scheduler.EventCompleted, Result: scheduler.ResultSucceeded, Occurrence: missing},
			}
			started := time.Now()
			for _, event := range events {
				event.Context = eventContext
				event.Owner = "fixture-owner"
				event.Fencing = 42
				observer.Observe(event)
			}
			upperSeconds := time.Since(started).Seconds()
			for range 2 {
				if err := runtime.ForceFlush(ctx); err != nil {
					t.Fatal(err)
				}
				metrics.assertSnapshot(t, upperSeconds)
			}
			got := spans.GetSpans()
			if len(got) != 3 {
				t.Fatalf("spans = %d, want superseded, failed and succeeded", len(got))
			}
			for _, span := range got {
				if span.Name != "scheduler.execute" || span.InstrumentationScope.Name != "github.com/faustbrian/go-scheduler" || span.SpanKind != trace.SpanKindInternal {
					t.Fatalf("span identity = %+v", span)
				}
				if !span.Parent.Equal(parent) || span.SpanContext.TraceID() != parent.TraceID() {
					t.Fatal("span lost its supplied parent")
				}
				checkSchedulerAttributes(t, attribute.NewSet(span.Attributes...), "before", "succeeded")
				checkSchedulerResource(t, attribute.NewSet(span.Resource.Attributes()...))
			}
			if got[0].Status.Code != codes.Error || got[0].Status.Description != "superseded lifecycle" || got[1].Status.Code != codes.Error || got[2].Status.Code != codes.Ok {
				t.Fatalf("span terminal statuses = %v, %v, %v", got[0].Status, got[1].Status, got[2].Status)
			}
			if len(got[1].Events) != 1 || got[1].Events[0].Name != "exception" {
				t.Fatal("failure did not export its exception")
			}
			if compatibility {
				lines := bytes.Split(bytes.TrimSpace(logs.Bytes()), []byte("\n"))
				if len(lines) != len(events) {
					t.Fatalf("log records = %d, want %d", len(lines), len(events))
				}
				for index, line := range lines {
					var record map[string]any
					if err := json.Unmarshal(line, &record); err != nil {
						t.Fatal(err)
					}
					level := "INFO"
					if index >= 2 && index <= 4 {
						level = "ERROR"
					}
					if record["level"] != level || record["schedule"] != "fixture" || record["task"] != "fixture.task" || record["owner"] != "fixture-owner" || record["fencing_token"] != float64(42) {
						t.Fatalf("log record = %v", record)
					}
				}
			}
			if spans.shutdowns.Load() != 0 || metrics.shutdowns.Load() != 0 {
				t.Fatal("observer shut down caller-owned exporters")
			}
			if err := runtime.Shutdown(ctx); err != nil {
				t.Fatal(err)
			}
			if err := runtime.Shutdown(ctx); err != nil {
				t.Fatal(err)
			}
			if spans.shutdowns.Load() != 1 || metrics.shutdowns.Load() != 1 {
				t.Fatal("repeated runtime shutdown changed exporter ownership")
			}
		})
	}
}

func TestRunnerExportsSuccessfulAndFailedLifecycles(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	harness := testtelemetry.New()
	defer func() {
		if err := harness.Shutdown(ctx); err != nil {
			t.Errorf("Shutdown: %v", err)
		}
	}()
	observer, err := schedulerotel.New(schedulerotel.Config{
		TracerProvider: harness.TracerProvider(), MeterProvider: harness.MeterProvider(),
	})
	if err != nil {
		t.Fatal(err)
	}
	succeeded, err := scheduler.NewSchedule("successful", "succeed", scheduler.EveryMinute())
	if err != nil {
		t.Fatal(err)
	}
	failed, err := scheduler.NewSchedule("failed", "fail", scheduler.EveryMinute())
	if err != nil {
		t.Fatal(err)
	}
	registry, err := scheduler.Compile(succeeded, failed)
	if err != nil {
		t.Fatal(err)
	}
	failure := errors.New("fixture execution failure")
	var calls atomic.Int64
	runner, err := scheduler.NewRunner(registry, memory.New(), executorFunc(func(_ context.Context, scheduled scheduler.Context) error {
		calls.Add(1)
		if scheduled.Schedule.Task == "fail" {
			return failure
		}
		return nil
	}), scheduler.WithOwner("fixture"), scheduler.WithObserver(observer))
	if err != nil {
		t.Fatal(err)
	}
	through := time.Date(2026, time.January, 1, 0, 1, 0, 0, time.UTC)
	if err := runner.Tick(ctx, through.Add(-time.Minute), through); !errors.Is(err, failure) || calls.Load() != 2 {
		t.Fatalf("Tick error/calls = %v/%d", err, calls.Load())
	}
	spans := harness.Spans()
	statuses := map[codes.Code]int{}
	for _, span := range spans {
		if span.Name != "scheduler.execute" {
			t.Fatalf("execution span name = %q", span.Name)
		}
		statuses[span.Status.Code]++
	}
	if len(spans) != 2 || statuses[codes.Ok] != 1 || statuses[codes.Error] != 1 {
		t.Fatalf("execution spans/statuses = %d/%v", len(spans), statuses)
	}
	data, err := harness.Metrics(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var events int64
	var completed uint64
	for _, scope := range data.ScopeMetrics {
		for _, metric := range scope.Metrics {
			if metric.Name == "scheduler.events" {
				for _, point := range metric.Data.(metricdata.Sum[int64]).DataPoints {
					events += point.Value
				}
			}
			if metric.Name == "scheduler.execution.duration" {
				for _, point := range metric.Data.(metricdata.Histogram[float64]).DataPoints {
					completed += point.Count
				}
			}
		}
	}
	if events != 8 || completed != 2 {
		t.Fatalf("runner telemetry = %d lifecycle events/%d completed durations", events, completed)
	}
}

type schedulerSpanExporter struct {
	*tracetest.InMemoryExporter
	shutdowns atomic.Int64
}

func (exporter *schedulerSpanExporter) Shutdown(ctx context.Context) error {
	exporter.shutdowns.Add(1)
	return exporter.InMemoryExporter.Shutdown(ctx)
}

type schedulerMetricPoint struct {
	name, unit string
	attributes attribute.Set
	value      int64
	count      uint64
	sum        float64
}

type schedulerMetricExporter struct {
	mu        sync.Mutex
	points    []schedulerMetricPoint
	resource  attribute.Set
	invalid   bool
	shutdowns atomic.Int64
}

func (*schedulerMetricExporter) Temporality(kind sdkmetric.InstrumentKind) metricdata.Temporality {
	return sdkmetric.DefaultTemporalitySelector(kind)
}

func (*schedulerMetricExporter) Aggregation(kind sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return sdkmetric.DefaultAggregationSelector(kind)
}

func (exporter *schedulerMetricExporter) Export(_ context.Context, data *metricdata.ResourceMetrics) error {
	exporter.mu.Lock()
	defer exporter.mu.Unlock()
	exporter.points = nil
	exporter.resource = attribute.NewSet(data.Resource.Attributes()...)
	for _, scope := range data.ScopeMetrics {
		if scope.Scope.Name != "github.com/faustbrian/go-scheduler" {
			exporter.invalid = true
		}
		for _, metric := range scope.Metrics {
			switch aggregation := metric.Data.(type) {
			case metricdata.Sum[int64]:
				if !aggregation.IsMonotonic || aggregation.Temporality != metricdata.CumulativeTemporality {
					exporter.invalid = true
				}
				for _, point := range aggregation.DataPoints {
					exporter.points = append(exporter.points, schedulerMetricPoint{name: metric.Name, unit: metric.Unit, attributes: point.Attributes, value: point.Value})
				}
			case metricdata.Histogram[float64]:
				if aggregation.Temporality != metricdata.CumulativeTemporality {
					exporter.invalid = true
				}
				for _, point := range aggregation.DataPoints {
					exporter.points = append(exporter.points, schedulerMetricPoint{name: metric.Name, unit: metric.Unit, attributes: point.Attributes, count: point.Count, sum: point.Sum})
				}
			default:
				exporter.invalid = true
			}
		}
	}
	return nil
}

func (*schedulerMetricExporter) ForceFlush(context.Context) error { return nil }

func (exporter *schedulerMetricExporter) Shutdown(context.Context) error {
	exporter.shutdowns.Add(1)
	return nil
}

func (exporter *schedulerMetricExporter) assertSnapshot(t *testing.T, upperSeconds float64) {
	t.Helper()
	exporter.mu.Lock()
	defer exporter.mu.Unlock()
	if exporter.invalid || len(exporter.points) != 6 {
		t.Fatalf("metric shape = invalid %v, points %d", exporter.invalid, len(exporter.points))
	}
	checkSchedulerResource(t, exporter.resource)
	counters := map[string]int64{"before/succeeded": 3, "failure/failed": 1, "completed/failed": 2, "completed/succeeded": 3}
	durations := map[string]bool{"completed/failed": true, "completed/succeeded": true}
	for _, point := range exporter.points {
		event, _ := point.attributes.Value("scheduler.event")
		result, _ := point.attributes.Value("scheduler.result")
		checkSchedulerAttributes(t, point.attributes, event.AsString(), result.AsString())
		key := event.AsString() + "/" + result.AsString()
		switch point.name {
		case "scheduler.events":
			want, ok := counters[key]
			if !ok || point.value != want || point.unit != "" {
				t.Fatalf("counter %q = %d, unit %q", key, point.value, point.unit)
			}
			delete(counters, key)
		case "scheduler.execution.duration":
			if !durations[key] || point.unit != "s" || point.count != 1 || math.IsNaN(point.sum) || math.IsInf(point.sum, 0) || point.sum < 0 || point.sum > upperSeconds {
				t.Fatalf("histogram %q = count %d, sum %g, unit %q, upper %g", key, point.count, point.sum, point.unit, upperSeconds)
			}
			delete(durations, key)
		default:
			t.Fatalf("unexpected instrument %q", point.name)
		}
	}
	if len(counters) != 0 || len(durations) != 0 {
		t.Fatal("missing metric series")
	}
}

func checkSchedulerAttributes(t *testing.T, attrs attribute.Set, event, result string) {
	t.Helper()
	want := attribute.NewSet(attribute.String("scheduler.event", event), attribute.String("scheduler.result", result), attribute.String("scheduler.schedule", "fixture"), attribute.String("scheduler.task", "fixture.task"), attribute.Bool("scheduler.background", false))
	if !attrs.Equals(&want) {
		t.Fatalf("scheduler attributes = %v, want %v", attrs.ToSlice(), want.ToSlice())
	}
}

func checkSchedulerResource(t *testing.T, attrs attribute.Set) {
	t.Helper()
	service, _ := attrs.Value("service.name")
	version, _ := attrs.Value("telemetry.sdk.version")
	if service.AsString() != "scheduler-fixture" || version.AsString() != sdk.Version() {
		t.Fatalf("resource service/version = %q/%q", service.AsString(), version.AsString())
	}
}
