// Package memory provides mutex-protected, process-local leases using supplied
// time. Use New for tests/tools; tokens survive release only within this store
// instance and are lost at restart. No cleanup worker or persistence is owned.
package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	lease "github.com/faustbrian/go-scheduler/v3/adapters/lease"
)

// Store keeps leases and per-key counters under one mutex. Use New; the zero
// store cannot acquire. Expired records are not pruned and per-key counters
// are never removed, so unique logical keys are not globally memory-bounded.
type Store struct {
	mu     sync.Mutex
	leases map[string]lease.Lease
	tokens map[string]uint64
}

// New constructs an empty memory lease store.
func New() *Store {
	return &Store{
		leases: make(map[string]lease.Lease),
		tokens: make(map[string]uint64),
	}
}

// Acquire creates or takes over an expired lease.
func (store *Store) Acquire(
	ctx context.Context,
	key string,
	owner string,
	ttl time.Duration,
	now time.Time,
) (lease.Lease, error) {
	if err := ctx.Err(); err != nil {
		return lease.Lease{}, err
	}
	if key == "" || owner == "" || ttl <= 0 || now.IsZero() {
		return lease.Lease{}, lease.ErrInvalid
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	if current, ok := store.leases[key]; ok && !current.Expired(now) {
		return lease.Lease{}, fmt.Errorf("%w: %s", lease.ErrHeld, key)
	}
	store.tokens[key]++
	owned := lease.Lease{
		Key:          key,
		Owner:        owner,
		FencingToken: store.tokens[key],
		AcquiredAt:   now,
		ExpiresAt:    now.Add(ttl),
	}
	store.leases[key] = owned
	return owned, nil
}

// Heartbeat extends a lease held by its current owner.
func (store *Store) Heartbeat(
	ctx context.Context,
	owned lease.Lease,
	ttl time.Duration,
	now time.Time,
) (lease.Lease, error) {
	if err := ctx.Err(); err != nil {
		return lease.Lease{}, err
	}
	if ttl <= 0 || now.IsZero() {
		return lease.Lease{}, lease.ErrInvalid
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	current, ok := store.leases[owned.Key]
	if !ok {
		return lease.Lease{}, fmt.Errorf("%w: %s", lease.ErrNotFound, owned.Key)
	}
	if !sameOwner(current, owned) || current.Expired(now) {
		return lease.Lease{}, fmt.Errorf("%w: %s", lease.ErrStaleOwner, owned.Key)
	}
	current.ExpiresAt = now.Add(ttl)
	store.leases[owned.Key] = current
	return current, nil
}

// Release removes a lease only when ownership still matches.
func (store *Store) Release(ctx context.Context, owned lease.Lease) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	current, ok := store.leases[owned.Key]
	if !ok {
		return fmt.Errorf("%w: %s", lease.ErrNotFound, owned.Key)
	}
	if !sameOwner(current, owned) {
		return fmt.Errorf("%w: %s", lease.ErrStaleOwner, owned.Key)
	}
	delete(store.leases, owned.Key)
	return nil
}

// Inspect returns retained ownership, including expired leases; it does not
// consult a clock or prune data. Use Lease.Expired with the same clock domain.
func (store *Store) Inspect(ctx context.Context, key string) (lease.Lease, error) {
	if err := ctx.Err(); err != nil {
		return lease.Lease{}, err
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	current, ok := store.leases[key]
	if !ok {
		return lease.Lease{}, fmt.Errorf("%w: %s", lease.ErrNotFound, key)
	}
	return current, nil
}

// Recover removes a lease only when its observed fencing token still matches.
func (store *Store) Recover(ctx context.Context, key string, fencingToken uint64) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	current, ok := store.leases[key]
	if !ok {
		return fmt.Errorf("%w: %s", lease.ErrNotFound, key)
	}
	if current.FencingToken != fencingToken {
		return fmt.Errorf("%w: %s", lease.ErrStaleOwner, key)
	}
	delete(store.leases, key)
	return nil
}

// Capabilities reports the memory store's safety properties.
func (*Store) Capabilities() lease.Capabilities {
	return lease.Capabilities{
		Fencing:          true,
		Heartbeat:        true,
		CompareAndDelete: true,
		ManualRecovery:   true,
	}
}

func sameOwner(current, candidate lease.Lease) bool {
	return current.Owner == candidate.Owner && current.FencingToken == candidate.FencingToken
}
