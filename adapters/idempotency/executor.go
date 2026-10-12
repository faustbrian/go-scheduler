// Package scheduleridempotency is the target-oriented path for scheduler idempotency.
// Its public types and behavior retain the released package identities.
// Use New; no store lifecycle, ownership renewal or atomic external completion is added.
package scheduleridempotency

//lint:file-ignore SA1019 This successor imports the deprecated package to preserve released identities.

import (
	goidempotency "github.com/faustbrian/go-idempotency"
	scheduler "github.com/faustbrian/go-scheduler/v2"
	legacy "github.com/faustbrian/go-scheduler/v2/idempotency" //nolint:staticcheck // Identity-preserving successor.
)

var (
	// ErrInvalidConfiguration reports missing or unsafe wrapper dependencies.
	ErrInvalidConfiguration = legacy.ErrInvalidConfiguration
	// ErrOccurrenceConflict reports an occurrence owned by incompatible work.
	ErrOccurrenceConflict = legacy.ErrOccurrenceConflict
)

// Options configures the tenant namespace and ownership lease.
type Options = legacy.Options

// Executor deduplicates schedule occurrences before invoking another executor.
type Executor = legacy.Executor

// New constructs an occurrence-deduplicating executor.
// It borrows both dependencies; ownership is not renewed during inner work
// and completion is not atomic with external effects.
func New(store goidempotency.Store, inner scheduler.Executor, options Options) (*Executor, error) {
	return legacy.New(store, inner, options)
}
