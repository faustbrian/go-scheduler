package memory_test

//lint:file-ignore SA1019 This file intentionally exercises or implements the retained compatibility contract.

import (
	"testing"
	"time"

	"github.com/faustbrian/go-scheduler/lease/conformance"
	"github.com/faustbrian/go-scheduler/memory"
)

func TestConformance(t *testing.T) {
	conformance.TestStore(t, func(*testing.T) conformance.Harness {
		now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
		return conformance.Harness{
			Store:   memory.New(),
			Now:     func() time.Time { return now },
			Advance: func(duration time.Duration) { now = now.Add(duration) },
		}
	})
}
