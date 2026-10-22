package scheduleridempotency_test

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	goidempotency "github.com/faustbrian/go-idempotency/v3"
	"github.com/faustbrian/go-idempotency/v3/memory"
	idempotencypostgres "github.com/faustbrian/go-idempotency/v3/postgres"
	scheduler "github.com/faustbrian/go-scheduler/v3"
	scheduleridempotency "github.com/faustbrian/go-scheduler/v3/adapters/idempotency"
)

type recordingStore struct {
	goidempotency.Store
	requests []goidempotency.AcquireRequest
}

func (store *recordingStore) Acquire(ctx context.Context, request goidempotency.AcquireRequest) (goidempotency.AcquireResult, error) {
	store.requests = append(store.requests, request)
	return store.Store.Acquire(ctx, request)
}

// Records written by the previous major are found by this key digest and
// fingerprint; drifting either would re-execute occurrences during a rollout.
func TestExecutorOccurrenceIdentityMatchesRetainedRecords(t *testing.T) {
	t.Parallel()

	const (
		wantRecordDigest = "03b4d04c818ef70e47713ed196eb9289da7476eb55b06184a820344d98117a5d"
		wantFingerprint  = "scheduler-occurrence-v1:f5e3828aea1e89bace15a74df8107a2752dd3323cc797b1000525f6892f325c0"
	)
	backing, err := memory.New(memory.Options{
		Clock:       staticClock{time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)},
		OwnerTokens: func() (string, error) { return "owner", nil },
	})
	if err != nil {
		t.Fatalf("memory.New() error = %v", err)
	}
	store := &recordingStore{Store: backing}
	called := 0
	executor, err := scheduleridempotency.New(
		store,
		executorFunc(func(context.Context, scheduler.Context) error { called++; return nil }),
		scheduleridempotency.Options{Tenant: "acme", Lease: time.Minute},
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	schedule, _ := scheduler.NewSchedule("report", "reports.generate", scheduler.Daily())
	scheduled := scheduler.Context{
		Schedule:       schedule,
		Due:            time.Date(2026, time.January, 1, 8, 0, 0, 0, time.UTC),
		IdempotencyKey: "occurrence-1",
	}
	for range 2 {
		if err := executor.Execute(context.Background(), scheduled); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
	}
	if called != 1 {
		t.Fatalf("inner executor calls = %d, want 1 after replay", called)
	}
	request := store.requests[0]
	if got := hex.EncodeToString(idempotencypostgres.RecordKeyDigest(request.Key)); got != wantRecordDigest {
		t.Fatalf("record key digest = %s, want %s", got, wantRecordDigest)
	}
	if got := request.Fingerprint.Version() + ":" + hex.EncodeToString(request.Fingerprint.Sum()); got != wantFingerprint {
		t.Fatalf("fingerprint = %s, want %s", got, wantFingerprint)
	}

	// The due time is normalised to UTC with nanosecond RFC 3339 before
	// hashing, so a zoned, fractional due time keeps its retained identity.
	// The expected value is sha256(CoordinationID + NUL + "2026-01-01T08:00:00.123456789Z").
	const wantZonedFingerprint = "scheduler-occurrence-v1:21f9ae7f204d58df0bf9f8450141a8c505c9eb9cb4e2dc773860db64c3389fae"
	zoned := scheduled
	zoned.IdempotencyKey = "occurrence-2"
	zoned.Due = time.Date(2026, time.January, 1, 10, 0, 0, 123456789, time.FixedZone("UTC+2", 2*60*60))
	if err := executor.Execute(context.Background(), zoned); err != nil {
		t.Fatalf("Execute(zoned) error = %v", err)
	}
	zonedRequest := store.requests[len(store.requests)-1]
	if got := zonedRequest.Fingerprint.Version() + ":" + hex.EncodeToString(zonedRequest.Fingerprint.Sum()); got != wantZonedFingerprint {
		t.Fatalf("zoned fingerprint = %s, want %s", got, wantZonedFingerprint)
	}
}
