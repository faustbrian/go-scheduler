package adapters_test

//lint:file-ignore SA1019 Compatibility assertions intentionally import deprecated paths.

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	schedulercli "github.com/faustbrian/go-scheduler/v2/adapters/cli"
	schedulerhttp "github.com/faustbrian/go-scheduler/v2/adapters/http"
	scheduleridempotency "github.com/faustbrian/go-scheduler/v2/adapters/idempotency"
	schedulerlease "github.com/faustbrian/go-scheduler/v2/adapters/lease"
	schedulerotel "github.com/faustbrian/go-scheduler/v2/adapters/otel"
	schedulerqueue "github.com/faustbrian/go-scheduler/v2/adapters/queue"
	schedulerservice "github.com/faustbrian/go-scheduler/v2/adapters/service"
	schedulerslog "github.com/faustbrian/go-scheduler/v2/adapters/slog"
	legacyidempotency "github.com/faustbrian/go-scheduler/v2/idempotency"  //nolint:staticcheck // Compatibility identity assertion.
	legacylease "github.com/faustbrian/go-scheduler/v2/lease"              //nolint:staticcheck // Compatibility identity assertion.
	legacyqueue "github.com/faustbrian/go-scheduler/v2/queue"              //nolint:staticcheck // Compatibility identity assertion.
	legacyhttp "github.com/faustbrian/go-scheduler/v2/schedulerhttp"       //nolint:staticcheck // Compatibility identity assertion.
	legacyservice "github.com/faustbrian/go-scheduler/v2/schedulerservice" //nolint:staticcheck // Compatibility identity assertion.
)

func TestCanonicalAdaptersExposeTheirConstructionContracts(t *testing.T) {
	t.Parallel()

	if _, err := scheduleridempotency.New(nil, nil, scheduleridempotency.Options{}); !errors.Is(err, scheduleridempotency.ErrInvalidConfiguration) {
		t.Fatalf("idempotency.New() error = %v", err)
	}
	if _, err := schedulerqueue.New(nil); !errors.Is(err, schedulerqueue.ErrInvalidQueue) {
		t.Fatalf("queue.New() error = %v", err)
	}
	if code := schedulercli.Run(context.Background(), nil, nil, nil, nil, nil); code != 2 {
		t.Fatalf("cli.Run() code = %d, want 2", code)
	}
	if _, err := schedulerhttp.New(nil, nil); !errors.Is(err, schedulerhttp.ErrInvalidDependencies) {
		t.Fatalf("http.New() error = %v", err)
	}
	if _, err := schedulerservice.New(schedulerservice.Options{}); !errors.Is(err, schedulerservice.ErrInvalidOptions) {
		t.Fatalf("service.New() error = %v", err)
	}
	if _, err := schedulerslog.New(schedulerslog.Config{}); !errors.Is(err, schedulerslog.ErrInvalidConfiguration) {
		t.Fatalf("slog.New() error = %v", err)
	}
	if _, err := schedulerotel.New(schedulerotel.Config{}); !errors.Is(err, schedulerotel.ErrInvalidConfiguration) {
		t.Fatalf("otel.New() error = %v", err)
	}

	lease := schedulerlease.Lease{ExpiresAt: time.Unix(2, 0)}
	if lease.Expired(time.Unix(1, 0)) || !lease.Expired(time.Unix(2, 0)) {
		t.Fatal("lease expiry boundary changed")
	}
}

func TestCanonicalAdaptersPreserveReleasedTypeAndErrorIdentity(t *testing.T) {
	t.Parallel()

	types := [][2]reflect.Type{
		{reflect.TypeOf(scheduleridempotency.Options{}), reflect.TypeOf(legacyidempotency.Options{})},
		{reflect.TypeOf(schedulerlease.Lease{}), reflect.TypeOf(legacylease.Lease{})},
		{reflect.TypeOf(schedulerqueue.Envelope{}), reflect.TypeOf(legacyqueue.Envelope{})},
		{reflect.TypeOf(schedulerhttp.Schedule{}), reflect.TypeOf(legacyhttp.Schedule{})},
		{reflect.TypeOf(schedulerservice.Options{}), reflect.TypeOf(legacyservice.Options{})},
	}
	for _, pair := range types {
		if pair[0] != pair[1] {
			t.Fatalf("canonical type %v differs from released type %v", pair[0], pair[1])
		}
	}

	// Exact interface identity, rather than errors.Is traversal, is the compatibility contract.
	if scheduleridempotency.ErrInvalidConfiguration != legacyidempotency.ErrInvalidConfiguration || //nolint:errorlint
		schedulerlease.ErrInvalid != legacylease.ErrInvalid || //nolint:errorlint // Exact sentinel identity is the contract.
		schedulerqueue.ErrInvalidQueue != legacyqueue.ErrInvalidQueue || //nolint:errorlint // Exact sentinel identity is the contract.
		schedulerhttp.ErrInvalidDependencies != legacyhttp.ErrInvalidDependencies || //nolint:errorlint // Exact sentinel identity is the contract.
		schedulerservice.ErrInvalidOptions != legacyservice.ErrInvalidOptions { //nolint:errorlint // Exact sentinel identity is the contract.
		t.Fatal("canonical adapter changed a released sentinel identity")
	}
}
