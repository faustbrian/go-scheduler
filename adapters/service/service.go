// Package schedulerservice is the target-oriented path for scheduler service
// lifecycle composition. Its public types retain the released identities.
package schedulerservice

//lint:file-ignore SA1019 This successor imports the deprecated package to preserve released identities.

import legacy "github.com/faustbrian/go-scheduler/schedulerservice" //nolint:staticcheck // Identity-preserving successor.

// ErrInvalidOptions identifies invalid adapter construction.
var ErrInvalidOptions = legacy.ErrInvalidOptions

// CorrelationMode selects the schedule correlation boundary used for each occurrence.
type CorrelationMode = legacy.CorrelationMode

const (
	// CorrelationIndependent starts an independent correlation workflow for every occurrence.
	CorrelationIndependent = legacy.CorrelationIndependent
	// CorrelationTrustedMetadata continues explicitly trusted schedule metadata.
	CorrelationTrustedMetadata = legacy.CorrelationTrustedMetadata
)

// Options configures scheduler lifecycle composition and ownership transfer.
type Options = legacy.Options

// OptionsError identifies the first rejected option in deterministic validation order.
type OptionsError = legacy.OptionsError

// Adapter exposes the constructed runner and service components.
type Adapter = legacy.Adapter

// New validates options and constructs scheduler lifecycle components.
func New(options Options) (*Adapter, error) { return legacy.New(options) }
