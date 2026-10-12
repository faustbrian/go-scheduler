package scheduler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"
)

var (
	// ErrScheduleNameRequired reports a blank schedule name.
	ErrScheduleNameRequired = errors.New("scheduler: schedule name is required")
	// ErrTaskNameRequired reports a blank task identity.
	ErrTaskNameRequired = errors.New("scheduler: task name is required")
	// ErrInvalidMissedRuns reports an invalid missed-run policy or bound.
	ErrInvalidMissedRuns = errors.New("scheduler: invalid missed-run policy")
	// ErrInvalidDateBounds reports an end instant before its start instant.
	ErrInvalidDateBounds = errors.New("scheduler: invalid date bounds")
	// ErrInvalidDuration reports a non-positive duration option.
	ErrInvalidDuration = errors.New("scheduler: duration must be positive")
	// ErrInvalidVersion reports a blank schedule version.
	ErrInvalidVersion = errors.New("scheduler: version is required")
	// ErrResourceLimit reports a definition beyond an exported safety budget.
	ErrResourceLimit = errors.New("scheduler: resource limit exceeded")
	// ErrInvalidConstraint reports an invalid recurring schedule constraint.
	ErrInvalidConstraint = errors.New("scheduler: invalid schedule constraint")
)

const (
	// DefaultOverlapTTL matches Laravel's default 1,440-minute event mutex.
	DefaultOverlapTTL = 24 * time.Hour
	// DefaultOneServerTTL matches Laravel's one-hour scheduling mutex.
	DefaultOneServerTTL = time.Hour
	// MaxJitter is the largest allowed deterministic schedule offset.
	MaxJitter = 24 * time.Hour
	// MaxIdentityBytes bounds schedule identity component strings.
	MaxIdentityBytes = 255
	// MaxExpressionBytes bounds a cron expression before parser invocation.
	MaxExpressionBytes = 1_024
	// MaxParameterBytes bounds the JSON-encoded task parameter payload.
	MaxParameterBytes = 64 << 10
	// MaxMetadataEntries bounds diagnostic metadata cardinality.
	MaxMetadataEntries = 128
	// MaxMetadataBytes bounds combined metadata key and value bytes.
	MaxMetadataBytes = 64 << 10
	// MaxEnvironments bounds environment filters per schedule.
	MaxEnvironments = 64
	// MaxConditions bounds trusted application conditions per schedule.
	MaxConditions = 32
	// MaxTimeWindows bounds recurring time constraints per schedule.
	MaxTimeWindows = 32
	// MaxCatchUp bounds retained delayed occurrences per decision.
	MaxCatchUp = 1_000
	// MaxSchedules bounds registry size before compilation.
	MaxSchedules = 10_000
)

// MissedRunPolicy controls decisions after one or more delayed boundaries.
type MissedRunPolicy uint8

const (
	// MissedRunSkip executes only a boundary observed exactly on time.
	MissedRunSkip MissedRunPolicy = iota
	// MissedRunOnce executes only the newest missed boundary.
	MissedRunOnce
	// MissedRunCatchUp executes a bounded tail of missed boundaries.
	MissedRunCatchUp
)

// OverlapPolicy controls a decision when the task lease is already held.
type OverlapPolicy uint8

const (
	// OverlapAllow permits concurrent execution and does not acquire a task lease.
	OverlapAllow OverlapPolicy = iota
	// OverlapSkip skips an occurrence while its task lease is held.
	OverlapSkip
	// OverlapReplace delegates safe cancellation and transfer to a replacement store.
	OverlapReplace
)

// MaintenancePolicy controls execution while application maintenance is active.
type MaintenancePolicy uint8

const (
	// MaintenanceSkip suppresses execution during maintenance.
	MaintenanceSkip MaintenancePolicy = iota
	// MaintenanceRun allows execution during maintenance.
	MaintenanceRun
)

// Interval contains the explicit cron expression used by a schedule.
type Interval struct {
	expression string
}

// Cron stores an expression without validation. Compile accepts five fields,
// optional leading seconds, L day-of-month and pinned robfig/cron descriptors.
func Cron(expression string) Interval { return Interval{expression: expression} }

// EveryMinute returns the explicit every-minute cron interval.
func EveryMinute() Interval { return Cron("* * * * *") }

// Hourly returns the explicit hourly cron interval.
func Hourly() Interval { return Cron("0 * * * *") }

// Daily returns the explicit daily cron interval.
func Daily() Interval { return Cron("0 0 * * *") }

// Weekly returns the explicit Sunday-at-midnight cron interval.
func Weekly() Interval { return Cron("0 0 * * 0") }

// Monthly returns the explicit first-day-at-midnight cron interval.
func Monthly() Interval { return Cron("0 0 1 * *") }

// Expression returns the exact cron expression represented by the interval.
func (i Interval) Expression() string { return i.expression }

// Condition is a trusted application predicate. Conditions run in order until
// false/error, but timeout only stops waiting; a timed-out closure may continue.
// Shared closures and the occurrence maps require application synchronization.
type Condition func(Context) (bool, error)

// TimeWindow is an inclusive recurring local-time constraint. Start and End
// are offsets since local midnight; Excluded inverts the allowed window.
type TimeWindow struct {
	// Start is the inclusive local offset since midnight.
	Start time.Duration `json:"start"`
	// End is inclusive; Start greater than End describes a window crossing midnight.
	End time.Duration `json:"end"`
	// Excluded rejects this window instead of requiring it; all configured windows are combined.
	Excluded bool `json:"excluded"`
}

// Hook consumes a lifecycle event in a managed callback goroutine. It must
// tolerate concurrent/reordered completion; timeouts cannot terminate it.
// Callbacks should not synchronously Drain their own runner.
type Hook func(Event)

// Hooks configures optional callbacks for each lifecycle boundary.
type Hooks struct {
	// Before runs for EventBefore, before managed execution admission.
	Before Hook
	// Success runs for a successful runner execution/dispatch wait.
	Success Hook
	// Failure runs for a failed decision or execution wait.
	Failure Hook
	// Skipped runs for a policy skip, not an overlap event.
	Skipped Hook
	// Overlap runs when an active task lease suppresses the occurrence.
	Overlap Hook
	// Completed runs for the terminal decision event, subject to best-effort callback admission.
	Completed Hook
	// After runs for EventFinished; a timed-out executor can still be active.
	After Hook
}

// Context carries copied schedule and ownership data to conditions/execution.
// Its exported maps are mutable and shared within this occurrence; they must
// not be changed by a still-running condition while the executor uses them.
type Context struct {
	// Schedule is a copied definition; callback closures are shared application state.
	Schedule Schedule
	// Now is the tick range endpoint, not necessarily callback or completion wall time.
	Now time.Time
	// Due is the scheduled physical boundary.
	Due time.Time
	// Attempt is occurrence attempt metadata; the runner starts it at one.
	Attempt int
	// Owner is the runner lease-owner identity.
	Owner string
	// Fencing is the task token when present, otherwise occurrence token or zero.
	Fencing uint64
	// IdempotencyKey identifies rollout-stable physical occurrence ownership.
	IdempotencyKey string
	// Metadata is a copied diagnostic map shared by this occurrence's conditions and executor.
	Metadata map[string]string
}

// Schedule is an exported code-defined timing and policy value. NewSchedule and
// Compile copy its admitted maps/slices; callers must not mutate a published
// value concurrently. Callback closures are shared, not frozen or copied.
type Schedule struct {
	// Name is the nonblank application name and registry lookup key.
	Name string
	// Version is the revision component; NewSchedule defaults it to "1".
	Version string
	// Task is the nonblank application-owned work identity.
	Task string
	// Expression is cron input, validated by Compile rather than NewSchedule.
	Expression string
	// Timezone is the explicit loader name; NewSchedule defaults to UTC.
	Timezone string
	// Identity hashes coordination, version, timing, jitter and recurring constraints.
	Identity string
	// CoordinationID hashes name, task and canonical parameters, excluding timing/version.
	CoordinationID string
	// ParameterIdentity hashes the JSON representation of admitted task parameters.
	ParameterIdentity string
	// Parameters contains JSON-copied values; callers must not mutate a compiled definition.
	Parameters map[string]any
	// Enabled controls Due/Overview and runner scheduling; Next still calculates disabled definitions.
	Enabled bool
	// Environments is an exact-match allowlist; empty permits every environment.
	Environments []string
	// DaysOfWeek filters local weekdays before missed-run selection.
	DaysOfWeek []time.Weekday
	// TimeWindows combines inclusive local windows and exclusions before jitter.
	TimeWindows []TimeWindow
	// MaintenancePolicy selects behavior when the runner is in maintenance mode.
	MaintenancePolicy MaintenancePolicy
	// Conditions contains trusted synchronous predicates; closures are not deep-copied.
	Conditions []Condition
	// StartAt is an inclusive occurrence bound; zero leaves the lower bound open.
	StartAt time.Time
	// EndAt is an inclusive occurrence bound; zero leaves the upper bound open.
	EndAt time.Time
	// Jitter bounds the stable per-revision offset to [0,Jitter); zero disables it.
	Jitter time.Duration
	// Metadata is copied diagnostic text, not authenticated or redacted by core.
	Metadata map[string]string
	// MissedRunPolicy selects exact-boundary skip, newest-only, or bounded-tail catch-up.
	MissedRunPolicy MissedRunPolicy
	// MaxCatchUp caps retained catch-up occurrences, not candidates scanned.
	MaxCatchUp int
	// OverlapPolicy selects an overlap decision; alone it does not enable the task lease.
	OverlapPolicy OverlapPolicy
	// OnOneServer enables a non-renewed occurrence lease retained until expiry.
	OnOneServer bool
	// WithoutOverlapping enables a renewable task lease held until executor cleanup.
	WithoutOverlapping bool
	// LeaseTTL is the compatibility fallback when a specific TTL is unset.
	LeaseTTL time.Duration
	// OneServerTTL is the occurrence lease TTL; positive values override LeaseTTL.
	OneServerTTL time.Duration
	// OverlapTTL is the task lease TTL; positive values override LeaseTTL.
	OverlapTTL time.Duration
	// RunTimeout bounds the runner execution wait, not forced goroutine termination.
	RunTimeout time.Duration
	// RunInBackground returns from the decision after managed execution starts; Drain still tracks it.
	RunInBackground bool
	// Hooks contains best-effort callbacks; timed-out callbacks can finish out of event order.
	Hooks Hooks
	// EvenWhenPaused bypasses pause lookup, not environment/maintenance/condition policies.
	EvenWhenPaused bool
}

// Option configures a definition synchronously before resource admission and
// identity hashing. Custom options are trusted: arbitrary policy/TTL mutation
// is not comprehensively validated, and panics/blocking are not recovered.
type Option func(*Schedule) error

// NewSchedule applies non-nil options, admits selected resource limits and
// computes identities. Names/tasks are checked for blankness, not normalized.
// Cron/location validation occurs at Compile. JSON copying may invoke trusted
// custom marshalers before the encoded-byte cap is known.
func NewSchedule(name, task string, interval Interval, options ...Option) (Schedule, error) {
	if strings.TrimSpace(name) == "" {
		return Schedule{}, ErrScheduleNameRequired
	}
	if strings.TrimSpace(task) == "" {
		return Schedule{}, ErrTaskNameRequired
	}

	schedule := Schedule{
		Name:            name,
		Version:         "1",
		Task:            task,
		Expression:      interval.Expression(),
		Timezone:        "UTC",
		Enabled:         true,
		MissedRunPolicy: MissedRunSkip,
		OverlapPolicy:   OverlapAllow,
		Metadata:        map[string]string{},
		Parameters:      map[string]any{},
		LeaseTTL:        time.Minute,
		RunTimeout:      time.Minute,
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(&schedule); err != nil {
			return Schedule{}, err
		}
	}
	if !schedule.EndAt.IsZero() && !schedule.StartAt.IsZero() && schedule.EndAt.Before(schedule.StartAt) {
		return Schedule{}, ErrInvalidDateBounds
	}
	if schedule.MissedRunPolicy == MissedRunCatchUp && schedule.MaxCatchUp < 1 {
		return Schedule{}, ErrInvalidMissedRuns
	}
	if err := validateResourceLimits(schedule); err != nil {
		return Schedule{}, err
	}

	schedule.ParameterIdentity = stableHash(schedule.Parameters)
	schedule.CoordinationID = hashStrings(
		schedule.Name,
		schedule.Task,
		schedule.ParameterIdentity,
	)
	identityParts := []string{
		schedule.CoordinationID,
		schedule.Version,
		schedule.Expression,
		schedule.Timezone,
		schedule.Jitter.String(),
	}
	if hasRecurringConstraints(schedule) {
		identityParts = append(
			identityParts,
			stableHash(schedule.DaysOfWeek),
			stableHash(schedule.TimeWindows),
		)
	}
	schedule.Identity = hashStrings(identityParts...)
	return cloneSchedule(schedule), nil
}

func hasRecurringConstraints(schedule Schedule) bool {
	return len(schedule.DaysOfWeek) != 0 || len(schedule.TimeWindows) != 0
}

func validateResourceLimits(schedule Schedule) error {
	stringsToValidate := []struct {
		name  string
		value string
		limit int
	}{
		{name: "name", value: schedule.Name, limit: MaxIdentityBytes},
		{name: "version", value: schedule.Version, limit: MaxIdentityBytes},
		{name: "task", value: schedule.Task, limit: MaxIdentityBytes},
		{name: "expression", value: schedule.Expression, limit: MaxExpressionBytes},
		{name: "timezone", value: schedule.Timezone, limit: MaxIdentityBytes},
	}
	for _, field := range stringsToValidate {
		if len(field.value) > field.limit {
			return fmt.Errorf("%w: %s exceeds %d bytes", ErrResourceLimit, field.name, field.limit)
		}
	}
	parameters, err := json.Marshal(schedule.Parameters)
	if err != nil {
		return fmt.Errorf("scheduler: encode parameters: %w", err)
	}
	if len(parameters) > MaxParameterBytes {
		return fmt.Errorf("%w: parameters exceed %d bytes", ErrResourceLimit, MaxParameterBytes)
	}
	if len(schedule.Metadata) > MaxMetadataEntries {
		return fmt.Errorf("%w: metadata exceeds %d entries", ErrResourceLimit, MaxMetadataEntries)
	}
	metadataBytes := 0
	for key, value := range schedule.Metadata {
		metadataBytes += len(key) + len(value)
	}
	if metadataBytes > MaxMetadataBytes {
		return fmt.Errorf("%w: metadata exceeds %d bytes", ErrResourceLimit, MaxMetadataBytes)
	}
	if len(schedule.Environments) > MaxEnvironments {
		return fmt.Errorf("%w: environments exceed %d entries", ErrResourceLimit, MaxEnvironments)
	}
	for _, environment := range schedule.Environments {
		if len(environment) > MaxIdentityBytes {
			return fmt.Errorf("%w: environment exceeds %d bytes", ErrResourceLimit, MaxIdentityBytes)
		}
	}
	if len(schedule.Conditions) > MaxConditions {
		return fmt.Errorf("%w: conditions exceed %d entries", ErrResourceLimit, MaxConditions)
	}
	if len(schedule.TimeWindows) > MaxTimeWindows {
		return fmt.Errorf("%w: time windows exceed %d entries", ErrResourceLimit, MaxTimeWindows)
	}
	for _, day := range schedule.DaysOfWeek {
		if day < time.Sunday || day > time.Saturday {
			return fmt.Errorf("%w: weekday %d", ErrInvalidConstraint, day)
		}
	}
	for _, window := range schedule.TimeWindows {
		if window.Start < 0 || window.Start >= 24*time.Hour || window.End < 0 || window.End >= 24*time.Hour {
			return fmt.Errorf("%w: time window outside one day", ErrInvalidConstraint)
		}
	}
	if schedule.MaxCatchUp > MaxCatchUp {
		return fmt.Errorf("%w: catch-up exceeds %d occurrences", ErrResourceLimit, MaxCatchUp)
	}
	return nil
}

// WithTimezone sets the schedule's explicit IANA time-zone name.
func WithTimezone(timezone string) Option {
	return func(schedule *Schedule) error {
		schedule.Timezone = timezone
		return nil
	}
}

// WithVersion sets the semantic version included in schedule identity.
func WithVersion(version string) Option {
	return func(schedule *Schedule) error {
		if strings.TrimSpace(version) == "" {
			return ErrInvalidVersion
		}
		schedule.Version = version
		return nil
	}
}

// WithEnabled controls whether the compiled schedule produces occurrences.
func WithEnabled(enabled bool) Option {
	return func(schedule *Schedule) error {
		schedule.Enabled = enabled
		return nil
	}
}

// WithMaintenancePolicy controls execution during application maintenance.
func WithMaintenancePolicy(policy MaintenancePolicy) Option {
	return func(schedule *Schedule) error {
		if policy > MaintenanceRun {
			return fmt.Errorf("scheduler: invalid maintenance policy: %d", policy)
		}
		schedule.MaintenancePolicy = policy
		return nil
	}
}

// WithJitter sets the maximum deterministic per-schedule offset.
func WithJitter(jitter time.Duration) Option {
	return func(schedule *Schedule) error {
		if jitter < 0 || jitter > MaxJitter {
			return fmt.Errorf("scheduler: jitter must be between zero and %s", MaxJitter)
		}
		schedule.Jitter = jitter
		return nil
	}
}

// WithParameters copies parameters through JSON when applied, normalizing
// generic values (including numbers). It does not retain caller map ownership;
// custom marshalers remain trusted and the byte cap is checked afterward.
func WithParameters(parameters map[string]any) Option {
	return func(schedule *Schedule) error {
		encoded, err := json.Marshal(parameters)
		if err != nil {
			return fmt.Errorf("scheduler: encode parameters: %w", err)
		}
		_ = json.Unmarshal(encoded, &schedule.Parameters)
		return nil
	}
}

// WithEnvironments restricts execution to named application environments.
func WithEnvironments(environments ...string) Option {
	return func(schedule *Schedule) error {
		schedule.Environments = slices.Clone(environments)
		return nil
	}
}

// WithDateBounds limits occurrences to inclusive start and end instants.
func WithDateBounds(start, end time.Time) Option {
	return func(schedule *Schedule) error {
		schedule.StartAt = start
		schedule.EndAt = end
		return nil
	}
}

// WithMissedRuns sets delayed-boundary behavior and its catch-up cap.
func WithMissedRuns(policy MissedRunPolicy, maxCatchUp int) Option {
	return func(schedule *Schedule) error {
		if policy > MissedRunCatchUp || maxCatchUp < 0 {
			return ErrInvalidMissedRuns
		}
		schedule.MissedRunPolicy = policy
		schedule.MaxCatchUp = maxCatchUp
		return nil
	}
}

// WithOverlap records overlap policy without enabling a task lease.
func WithOverlap(policy OverlapPolicy) Option {
	return func(schedule *Schedule) error {
		if policy > OverlapReplace {
			return fmt.Errorf("scheduler: invalid overlap policy: %d", policy)
		}
		schedule.OverlapPolicy = policy
		return nil
	}
}

// WithMetadata copies bounded diagnostic metadata into the schedule.
func WithMetadata(metadata map[string]string) Option {
	return func(schedule *Schedule) error {
		schedule.Metadata = maps.Clone(metadata)
		return nil
	}
}

// WithCondition appends a non-nil trusted application condition.
func WithCondition(condition Condition) Option {
	return func(schedule *Schedule) error {
		if condition != nil {
			schedule.Conditions = append(schedule.Conditions, condition)
		}
		return nil
	}
}

// WithOneServer enables a distributed occurrence lease with the given TTL.
func WithOneServer(ttl time.Duration) Option {
	return func(schedule *Schedule) error {
		if ttl <= 0 {
			return fmt.Errorf("scheduler: one-server lease: %w", ErrInvalidDuration)
		}
		schedule.OnOneServer = true
		schedule.LeaseTTL = ttl
		schedule.OneServerTTL = ttl
		return nil
	}
}

// OnOneServer enables a distributed occurrence lease with the default mutex TTL.
func OnOneServer() Option {
	return func(schedule *Schedule) error {
		schedule.OnOneServer = true
		schedule.LeaseTTL = DefaultOneServerTTL
		schedule.OneServerTTL = DefaultOneServerTTL
		return nil
	}
}

// WithoutOverlap enables a renewable task lease and overlap policy.
func WithoutOverlap(policy OverlapPolicy, ttl time.Duration) Option {
	return func(schedule *Schedule) error {
		if policy == OverlapAllow || policy > OverlapReplace || ttl <= 0 {
			return fmt.Errorf("scheduler: overlap lease: %w", ErrInvalidDuration)
		}
		schedule.WithoutOverlapping = true
		schedule.OverlapPolicy = policy
		schedule.LeaseTTL = ttl
		schedule.OverlapTTL = ttl
		return nil
	}
}

// WithoutOverlapping skips an overlapping occurrence. The optional expiration
// is expressed in minutes and defaults to Laravel's 1,440-minute mutex TTL.
func WithoutOverlapping(minutes ...int) Option {
	return func(schedule *Schedule) error {
		if len(minutes) > 1 {
			return fmt.Errorf("scheduler: overlap lease: %w", ErrInvalidDuration)
		}
		ttl := DefaultOverlapTTL
		if len(minutes) == 1 {
			const maximumDurationMinutes = int64((1<<63 - 1) / time.Minute)
			if minutes[0] < 1 {
				return fmt.Errorf("scheduler: overlap lease: %w", ErrInvalidDuration)
			}
			if int64(minutes[0]) > maximumDurationMinutes {
				return fmt.Errorf("scheduler: overlap lease: %w", ErrInvalidDuration)
			}
			ttl = time.Duration(minutes[0]) * time.Minute
		}
		return WithoutOverlap(OverlapSkip, ttl)(schedule)
	}
}

// WithRunTimeout sets the execution deadline observed by the runner.
func WithRunTimeout(timeout time.Duration) Option {
	return func(schedule *Schedule) error {
		if timeout <= 0 {
			return ErrInvalidDuration
		}
		schedule.RunTimeout = timeout
		return nil
	}
}

// RunInBackground allows the runner to continue starting other due tasks while
// this task executes. Drain still waits for the managed execution to finish;
// asynchronous results are reported through lifecycle hooks and observers.
func RunInBackground() Option {
	return func(schedule *Schedule) error {
		schedule.RunInBackground = true
		return nil
	}
}

// EvenWhenPaused allows this schedule to run while its runner is paused.
func EvenWhenPaused() Option {
	return func(schedule *Schedule) error {
		schedule.EvenWhenPaused = true
		return nil
	}
}

// WithHooks sets lifecycle callbacks managed by the runner's callback bounds.
func WithHooks(hooks Hooks) Option {
	return func(schedule *Schedule) error {
		schedule.Hooks = hooks
		return nil
	}
}

func stableHash(value any) string {
	encoded, _ := json.Marshal(value)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

func hashStrings(values ...string) string {
	digest := sha256.New()
	for _, value := range values {
		_, _ = digest.Write([]byte(value))
		_, _ = digest.Write([]byte{0})
	}
	return hex.EncodeToString(digest.Sum(nil))
}

// JSON round-tripping isolates admitted parameter values, while condition/hook
// functions necessarily retain their closures. Copy failure on a manually
// mutated value leaves its original parameter map, so use the validated builder.
func cloneSchedule(schedule Schedule) Schedule {
	schedule.Environments = slices.Clone(schedule.Environments)
	schedule.DaysOfWeek = slices.Clone(schedule.DaysOfWeek)
	schedule.TimeWindows = slices.Clone(schedule.TimeWindows)
	schedule.Conditions = slices.Clone(schedule.Conditions)
	schedule.Metadata = maps.Clone(schedule.Metadata)
	if encoded, err := json.Marshal(schedule.Parameters); err == nil {
		var parameters map[string]any
		if json.Unmarshal(encoded, &parameters) == nil {
			schedule.Parameters = parameters
		}
	}
	return schedule
}
