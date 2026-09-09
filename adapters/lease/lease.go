// Package schedulerlease is the target-oriented path for scheduler lease contracts.
// Its public types and behavior retain the released package identities.
package schedulerlease

//lint:file-ignore SA1019 This successor imports the deprecated package to preserve released identities.

import legacy "github.com/faustbrian/go-scheduler/lease" //nolint:staticcheck // Identity-preserving successor.

var (
	// ErrHeld reports a lease currently owned by another execution.
	ErrHeld = legacy.ErrHeld
	// ErrNotFound reports an unknown or inactive lease.
	ErrNotFound = legacy.ErrNotFound
	// ErrStaleOwner reports an owner or fencing token that is no longer current.
	ErrStaleOwner = legacy.ErrStaleOwner
	// ErrInvalid reports malformed lease input.
	ErrInvalid = legacy.ErrInvalid
)

// Lease records current ownership and its monotonic fencing token.
type Lease = legacy.Lease

// Capabilities describes the safety properties implemented by a store.
type Capabilities = legacy.Capabilities

// Store provides fenced lease acquisition, renewal, release, and recovery.
type Store = legacy.Store

// ReplacementStore additionally supports atomic fenced ownership transfer.
type ReplacementStore = legacy.ReplacementStore
