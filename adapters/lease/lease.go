// Package schedulerlease defines provider clock, ownership and fencing contracts.
// Construct a memory/PostgreSQL/Valkey store or implement Store; leases alone
// cannot stop expired owners or make effects exactly once. Clients, durability
// and downstream token enforcement are application/deployment responsibilities.
package schedulerlease

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrHeld reports a lease currently owned by another execution.
	ErrHeld = errors.New("scheduler lease: already held")
	// ErrNotFound reports an unknown or inactive lease.
	ErrNotFound = errors.New("scheduler lease: not found")
	// ErrStaleOwner reports an owner or fencing token that is no longer current.
	ErrStaleOwner = errors.New("scheduler lease: stale owner or fencing token")
	// ErrInvalid reports malformed lease input.
	ErrInvalid = errors.New("scheduler lease: invalid argument")
)

// Lease records current ownership and its monotonic fencing token.
type Lease struct {
	// Key is the logical lease identity shared by coordinating replicas.
	Key string `json:"key"`
	// Owner is the current application-selected execution identity.
	Owner string `json:"owner"`
	// FencingToken orders acquisitions within a retained key namespace; downstream effects must enforce it.
	FencingToken uint64 `json:"fencing_token"`
	// AcquiredAt is the acquisition instant in the provider clock domain.
	AcquiredAt time.Time `json:"acquired_at"`
	// ExpiresAt is inclusive expiry; caller clocks are not authoritative for persistent providers.
	ExpiresAt time.Time `json:"expires_at"`
}

// Expired reports whether the lease has reached its expiry instant.
func (lease Lease) Expired(now time.Time) bool {
	return !now.Before(lease.ExpiresAt)
}

// Capabilities describes the safety properties implemented by a store.
type Capabilities struct {
	// Persistent advertises retained ownership across client/process restarts, subject to backend durability.
	Persistent bool `json:"persistent"`
	// Fencing advertises increasing acquisition tokens, not automatic downstream write protection.
	Fencing bool `json:"fencing"`
	// Heartbeat advertises renewable ownership without transferring the token.
	Heartbeat bool `json:"heartbeat"`
	// CompareAndDelete advertises owner/token checks when releasing ownership.
	CompareAndDelete bool `json:"compare_and_delete"`
	// ManualRecovery advertises token-checked administrative removal, not termination of the old owner.
	ManualRecovery bool `json:"manual_recovery"`
}

// Store provides concurrent ownership operations with non-nil contexts.
// Implementations must cooperate with cancellation and preserve fencing across
// takeover; timeouts can have unknown committed outcomes. Memory uses supplied
// now; persistent providers use server time. Inspect may expose expired retained
// records. No method cancels the execution represented by a lease.
type Store interface {
	// Acquire acquires a free/expired key or returns ErrHeld; providers select their clock domain.
	Acquire(ctx context.Context, key, owner string, ttl time.Duration, now time.Time) (Lease, error)
	// Heartbeat renews current unexpired ownership or rejects a stale/missing lease.
	Heartbeat(ctx context.Context, owned Lease, ttl time.Duration, now time.Time) (Lease, error)
	// Release removes matching owner/token ownership; stale owners must not remove a successor.
	Release(ctx context.Context, owned Lease) error
	// Inspect returns stored ownership; provider expiry visibility differs and callers must check expiry.
	Inspect(ctx context.Context, key string) (Lease, error)
	// Recover removes the observed token only; callers must isolate old side effects first.
	Recover(ctx context.Context, key string, fencingToken uint64) error
	// Capabilities reports provider obligations used during runner construction.
	Capabilities() Capabilities
}

// ReplacementStore coordinates cancellation of an existing execution and an
// atomic fencing-token transfer. Replace MUST NOT return a lease until the
// previous owner can no longer perform protected side effects.
type ReplacementStore interface {
	Store
	// Replace must establish cancellation/isolation before returning transferred ownership.
	Replace(
		ctx context.Context,
		current Lease,
		owner string,
		ttl time.Duration,
		now time.Time,
	) (Lease, error)
}
