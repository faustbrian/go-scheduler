// Package schedulerqueue is the target-oriented path for scheduler queue dispatch.
// Its public types and behavior retain the released package identities.
package schedulerqueue

//lint:file-ignore SA1019 This successor imports the deprecated package to preserve released identities.

import legacy "github.com/faustbrian/go-scheduler/queue" //nolint:staticcheck // Identity-preserving successor.

// ErrInvalidQueue reports a missing queue backend.
var ErrInvalidQueue = legacy.ErrInvalidQueue

// Enqueuer is the minimal durable queue submission contract.
type Enqueuer = legacy.Enqueuer

// Envelope is the version-independent occurrence payload sent to workers.
type Envelope = legacy.Envelope

// Dispatcher submits each schedule occurrence to a caller-owned durable queue.
type Dispatcher = legacy.Dispatcher

// New constructs a queue-backed scheduler executor.
func New(queue Enqueuer) (*Dispatcher, error) { return legacy.New(queue) }
