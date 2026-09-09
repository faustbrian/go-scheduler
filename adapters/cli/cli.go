// Package schedulercli is the target-oriented path for scheduler command-line
// inspection and recovery.
package schedulercli

//lint:file-ignore SA1019 This successor imports the deprecated package to preserve released behavior.

import (
	"context"
	"io"

	scheduler "github.com/faustbrian/go-scheduler"
	schedulerlease "github.com/faustbrian/go-scheduler/adapters/lease"
	legacy "github.com/faustbrian/go-scheduler/schedulercli" //nolint:staticcheck // Behavior-preserving successor.
)

// Run executes one bounded scheduler control command and returns an exit code.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer,
	registry *scheduler.Registry, leases schedulerlease.Store,
) int {
	return legacy.Run(ctx, args, stdout, stderr, registry, leases)
}
