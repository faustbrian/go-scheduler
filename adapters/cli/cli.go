// Package schedulercli is the target-oriented path for scheduler command-line
// inspection and recovery.
// Use Run; this path delegates command parsing and does not own writers or IO deadlines.
package schedulercli

//lint:file-ignore SA1019 This successor imports the deprecated package to preserve released behavior.

import (
	"context"
	"io"

	scheduler "github.com/faustbrian/go-scheduler/v2"
	schedulerlease "github.com/faustbrian/go-scheduler/v2/adapters/lease"
	legacy "github.com/faustbrian/go-scheduler/v2/schedulercli" //nolint:staticcheck // Behavior-preserving successor.
)

// Run executes one scheduler control command and returns the retained CLI exit code.
// It delegates to the retained CLI implementation. Authorization, writer
// lifetime and backend context cooperation remain application-owned.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer,
	registry *scheduler.Registry, leases schedulerlease.Store,
) int {
	return legacy.Run(ctx, args, stdout, stderr, registry, leases)
}
