// Package schedulerhttp is the target-oriented path for scheduler HTTP
// inspection and recovery.
// Use New; authentication, server lifetime and recovery authorization remain caller-owned.
package schedulerhttp

//lint:file-ignore SA1019 This successor imports the deprecated package to preserve released identities.

import (
	scheduler "github.com/faustbrian/go-scheduler/v2"
	schedulerlease "github.com/faustbrian/go-scheduler/v2/adapters/lease"
	legacy "github.com/faustbrian/go-scheduler/v2/schedulerhttp" //nolint:staticcheck // Identity-preserving successor.
)

// ErrInvalidDependencies reports a missing registry or lease store.
var ErrInvalidDependencies = legacy.ErrInvalidDependencies

// Schedule is the bounded public inspection view of a schedule.
type Schedule = legacy.Schedule

// Handler exposes bounded schedule inspection and recovery endpoints.
type Handler = legacy.Handler

// New constructs an administrative handler without registering global routes.
// It delegates to the retained administrative handler; the application must
// supply authentication, authorization and server deadlines.
func New(registry *scheduler.Registry, leases schedulerlease.Store) (*Handler, error) {
	return legacy.New(registry, leases)
}
