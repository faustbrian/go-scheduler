package lease_test

//lint:file-ignore SA1019 This file intentionally exercises or implements the retained compatibility contract.

import (
	"testing"
	"time"

	"github.com/faustbrian/go-scheduler/lease" //nolint:staticcheck // Exercises the retained domain port.
)

func TestLeaseExpiryBoundary(t *testing.T) {
	t.Parallel()

	expires := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	owned := lease.Lease{ExpiresAt: expires}
	if owned.Expired(expires.Add(-time.Nanosecond)) {
		t.Fatal("lease expired before boundary")
	}
	if !owned.Expired(expires) || !owned.Expired(expires.Add(time.Nanosecond)) {
		t.Fatal("lease did not expire at boundary")
	}
}
