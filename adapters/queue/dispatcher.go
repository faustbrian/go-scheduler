// Package schedulerqueue encodes occurrences for application-owned queue backends.
// New borrows an Enqueuer; neither worker execution nor backend closure is owned.
// Context is checked before submission but Queue itself has no context argument;
// blocking, durability, retries and worker deduplication remain backend policy.
package schedulerqueue

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	queuecore "github.com/faustbrian/go-queue/v2/core"
	queuejob "github.com/faustbrian/go-queue/v2/job"
	scheduler "github.com/faustbrian/go-scheduler/v3"
	"go.opentelemetry.io/otel/propagation"
)

// ErrInvalidQueue reports a missing queue backend.
var ErrInvalidQueue = errors.New("scheduler queue: queue is required")

// Enqueuer is the minimal durable queue submission contract.
type Enqueuer interface {
	// Queue synchronously submits copied message bytes; durability, blocking and retries belong to the backend.
	Queue(queuecore.QueuedMessage, ...queuejob.AllowOption) error
}

// Envelope is the version-independent occurrence payload sent to workers.
type Envelope struct {
	// ScheduleID identifies the timing revision.
	ScheduleID string `json:"schedule_id"`
	// CoordinationID identifies logical work across timing/version rollouts.
	CoordinationID string `json:"coordination_id"`
	// ScheduleName is the application registry name.
	ScheduleName string `json:"schedule_name"`
	// Task selects the application worker task.
	Task string `json:"task"`
	// Occurrence is the physical scheduled boundary.
	Occurrence time.Time `json:"occurrence"`
	// Attempt is metadata, not a worker retry instruction.
	Attempt int `json:"attempt"`
	// IdempotencyKey is the supplied occurrence key or a revision-based fallback.
	IdempotencyKey string `json:"idempotency_key"`
	// Owner is diagnostic lease-owner text, not authenticated identity.
	Owner string `json:"owner"`
	// FencingToken must be enforced by ownership-sensitive worker effects.
	FencingToken uint64 `json:"fencing_token"`
	// Parameters is JSON task data, not secret-filtered or schema-validated by dispatch.
	Parameters map[string]any `json:"parameters,omitempty"`
	// Metadata is occurrence metadata, falling back to schedule metadata when nil.
	Metadata map[string]string `json:"metadata,omitempty"`
	// TraceContext contains W3C trace propagation, not an authenticated trust assertion.
	TraceContext map[string]string `json:"trace_context,omitempty"`
}

// Dispatcher encodes schedule occurrences and submits them to queue.
type Dispatcher struct {
	queue      Enqueuer
	propagator propagation.TextMapPropagator
}

// New constructs a queue-backed scheduler executor.
func New(queue Enqueuer) (*Dispatcher, error) {
	if queue == nil {
		return nil, ErrInvalidQueue
	}
	return &Dispatcher{queue: queue, propagator: propagation.TraceContext{}}, nil
}

// Execute JSON-encodes borrowed occurrence data and submits a byte-copying
// message with W3C trace propagation. It checks ctx before Queue, but cannot
// cancel that synchronous backend call. No secret filtering or exactly-once
// guarantee is added; empty keys use a revision-based fallback.
func (dispatcher *Dispatcher) Execute(ctx context.Context, scheduled scheduler.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	traceContext := propagation.MapCarrier{}
	dispatcher.propagator.Inject(ctx, traceContext)
	idempotencyKey := scheduled.IdempotencyKey
	if idempotencyKey == "" {
		idempotencyKey = occurrenceKey(scheduled.Schedule.Identity, scheduled.Due)
	}
	metadata := scheduled.Metadata
	if metadata == nil {
		metadata = scheduled.Schedule.Metadata
	}
	envelope := Envelope{
		ScheduleID:     scheduled.Schedule.Identity,
		CoordinationID: scheduled.Schedule.CoordinationID,
		ScheduleName:   scheduled.Schedule.Name,
		Task:           scheduled.Schedule.Task,
		Occurrence:     scheduled.Due,
		Attempt:        scheduled.Attempt,
		IdempotencyKey: idempotencyKey,
		Owner:          scheduled.Owner,
		FencingToken:   scheduled.Fencing,
		Parameters:     scheduled.Schedule.Parameters,
		Metadata:       metadata,
		TraceContext:   map[string]string(traceContext),
	}
	encoded, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return dispatcher.queue.Queue(message(encoded))
}

type message []byte

func (payload message) Bytes() []byte { return append([]byte(nil), payload...) }

func occurrenceKey(scheduleID string, occurrence time.Time) string {
	digest := sha256.Sum256([]byte(scheduleID + "\x00" + occurrence.UTC().Format(time.RFC3339Nano)))
	return hex.EncodeToString(digest[:])
}
