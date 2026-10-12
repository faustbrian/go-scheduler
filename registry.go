// Package scheduler compiles code-defined schedules and coordinates occurrence
// decisions through application-owned executors and lease stores. Construct
// definitions with NewSchedule, freeze them with Compile, then use NewRunner.
// Registry reads are reusable concurrently; callbacks, backend IO and downstream
// side effects remain caller-owned. The core owns no retry loop, durable queue,
// timezone database, administrative authentication or forced goroutine termination.
package scheduler

//lint:file-ignore SA1019 This file intentionally exercises or implements the retained compatibility contract.

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
	"time"

	schedulercron "github.com/faustbrian/go-scheduler/v2/cron"
	"github.com/faustbrian/go-scheduler/v2/lease" //nolint:staticcheck // The root retains the released domain port.
)

var (
	// ErrInvalidExpression reports a schedule with invalid cron syntax.
	ErrInvalidExpression = errors.New("scheduler: invalid cron expression")
	// ErrInvalidTimezone reports a schedule with an unavailable IANA zone.
	ErrInvalidTimezone = errors.New("scheduler: invalid timezone")
	// ErrDuplicateSchedule reports repeated schedule names at compilation.
	ErrDuplicateSchedule = errors.New("scheduler: duplicate schedule")
	// ErrScheduleNotFound reports an unknown registry schedule name.
	ErrScheduleNotFound = errors.New("scheduler: schedule not found")
	// ErrOccurrenceLimit reports a due scan beyond its global candidate cap.
	ErrOccurrenceLimit = errors.New("scheduler: occurrence scan limit exceeded")
)

// MaxOccurrenceScan bounds candidates inspected by one Due call.
const MaxOccurrenceScan = 10_000

// Occurrence is one deterministic physical schedule boundary.
type Occurrence struct {
	// ScheduleID identifies the timing revision, distinct from rollout coordination.
	ScheduleID string `json:"schedule_id"`
	// ScheduleName is the configured name used by inspection and lifecycle hooks.
	ScheduleName string `json:"schedule_name"`
	// Task is the application executor or queue-worker task identity.
	Task string `json:"task"`
	// ScheduledAt is the physical boundary, including deterministic jitter.
	ScheduledAt time.Time `json:"scheduled_at"`
	// Attempt starts at one; the runner owns no retry increment loop.
	Attempt int `json:"attempt"`
	// IdempotencyKey hashes coordination identity and the physical UTC boundary.
	IdempotencyKey string `json:"idempotency_key"`
}

type compiledSchedule struct {
	schedule Schedule
	cron     schedulercron.Schedule
}

// Registry retains copied definitions and compiled timing. Concurrent read-only
// use is supported; function closures and application dependencies are shared.
// Construct it with Compile; a nil registry is not a usable method receiver.
type Registry struct {
	entries map[string]compiledSchedule
	names   []string
}

// ScheduleOverview combines a copied schedule definition with its next
// enabled execution boundary. Next is zero for a disabled schedule.
type ScheduleOverview struct {
	// Schedule is a copied definition; callback functions retain their closures.
	Schedule Schedule
	// Next is the calculated timing boundary; disabled definitions return zero.
	Next time.Time
}

// Compile checks registry size, duplicate names, cron and locations, joining
// failures without a partial registry. Supply NewSchedule-built definitions;
// arbitrary manually mutated Schedule fields are not comprehensively revalidated.
// Slices/maps are copied, but callback closures retain their application state.
func Compile(schedules ...Schedule) (*Registry, error) {
	if len(schedules) > MaxSchedules {
		return nil, fmt.Errorf("%w: registry exceeds %d schedules", ErrResourceLimit, MaxSchedules)
	}
	registry := &Registry{entries: make(map[string]compiledSchedule, len(schedules))}
	var errs []error
	for _, schedule := range schedules {
		if _, exists := registry.entries[schedule.Name]; exists {
			errs = append(errs, fmt.Errorf("%w: %s", ErrDuplicateSchedule, schedule.Name))
		} else {
			parsed, err := schedulercron.Compile(schedule.Expression, schedule.Timezone)
			if err != nil {
				classification := ErrInvalidExpression
				if errors.Is(err, schedulercron.ErrInvalidTimezone) {
					classification = ErrInvalidTimezone
				}
				errs = append(errs, fmt.Errorf("%w: %s: %w", classification, schedule.Name, err))
			} else {
				if hasRecurringConstraints(schedule) {
					// The cron compiler loaded this exact zone successfully above.
					location, _ := time.LoadLocation(schedule.Timezone)
					parsed = constrainedSchedule{
						Schedule: parsed,
						days:     slices.Clone(schedule.DaysOfWeek),
						windows:  slices.Clone(schedule.TimeWindows),
						location: location,
					}
				}
				registry.entries[schedule.Name] = compiledSchedule{
					schedule: cloneSchedule(schedule),
					cron: jitteredSchedule{
						Schedule: parsed,
						offset:   deterministicJitter(schedule.Identity, schedule.Jitter),
					},
				}
				registry.names = append(registry.names, schedule.Name)
			}
		}
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	slices.Sort(registry.names)
	return registry, nil
}

type constrainedSchedule struct {
	schedulercron.Schedule
	days     []time.Weekday
	windows  []TimeWindow
	location *time.Location
}

// Local constraints are intersected before jitter. Skipping to a day/minute
// boundary avoids visiting every rejected second; the search cap can return zero
// without proving that an arbitrary combined constraint has no future match.
func (schedule constrainedSchedule) Next(after time.Time) time.Time {
	cursor := after
	for range 200_000 {
		next := schedule.Schedule.Next(cursor)
		if next.IsZero() {
			return time.Time{}
		}
		local := next.In(schedule.location)
		if len(schedule.days) > 0 && !slices.Contains(schedule.days, local.Weekday()) {
			nextDay := time.Date(local.Year(), local.Month(), local.Day()+1, 0, 0, 0, 0, schedule.location)
			cursor = nextDay.Add(-time.Nanosecond)
			continue
		}
		if schedule.withinTimeWindows(local) {
			return next
		}
		cursor = local.Truncate(time.Minute).Add(time.Minute - time.Nanosecond)
	}
	return time.Time{}
}

func (schedule constrainedSchedule) withinTimeWindows(local time.Time) bool {
	minute := time.Duration(local.Hour()*60+local.Minute()) * time.Minute
	for _, window := range schedule.windows {
		inside := withinTimeWindow(minute, window)
		if (!window.Excluded && !inside) || (window.Excluded && inside) {
			return false
		}
	}
	return true
}

func withinTimeWindow(value time.Duration, window TimeWindow) bool {
	if window.Start <= window.End {
		return value >= window.Start && value <= window.End
	}
	return value >= window.Start || value <= window.End
}

type jitteredSchedule struct {
	schedulercron.Schedule
	offset time.Duration
}

func (schedule jitteredSchedule) Next(after time.Time) time.Time {
	return schedule.Schedule.Next(after.Add(-schedule.offset)).Add(schedule.offset)
}

// A revision-stable offset keeps coordinating replicas on the same physical
// boundary; it is not per-tick randomness or a distributed fairness guarantee.
func deterministicJitter(identity string, maximum time.Duration) time.Duration {
	if maximum <= 0 {
		return 0
	}
	digest := sha256.Sum256([]byte(identity))
	value := binary.BigEndian.Uint64(digest[:8]) % uint64(maximum)
	// #nosec G115 -- modulo by a positive time.Duration bounds value to MaxInt64.
	return time.Duration(value)
}

// Schedules returns independent value/map/slice copies sorted by name; callback
// closures remain shared. Returned exported fields may be changed by the caller.
func (registry *Registry) Schedules() []Schedule {
	result := make([]Schedule, 0, len(registry.names))
	for _, name := range registry.names {
		result = append(result, cloneSchedule(registry.entries[name].schedule))
	}
	return result
}

// Overview returns copied schedule definitions and their timing boundaries,
// sorted by schedule name. Callers supply the reference instant so inspection
// remains deterministic and can be exposed through any application surface.
func (registry *Registry) Overview(after time.Time) []ScheduleOverview {
	result := make([]ScheduleOverview, 0, len(registry.names))
	for _, name := range registry.names {
		entry := registry.entries[name]
		overview := ScheduleOverview{Schedule: cloneSchedule(entry.schedule)}
		if entry.schedule.Enabled {
			overview.Next = entry.cron.Next(after)
		}
		result = append(result, overview)
	}
	return result
}

// ClearCache removes every currently observed task lease for schedules using
// overlap prevention. Each removal is fenced by the token returned by Inspect.
// Callers must first isolate active owners because lease removal cannot stop
// unfenced side effects already in progress.
func (registry *Registry) ClearCache(ctx context.Context, store lease.Store) (int, error) {
	if store == nil {
		return 0, lease.ErrInvalid
	}
	cleared := 0
	var errs []error
	for _, name := range registry.names {
		if err := ctx.Err(); err != nil {
			return cleared, errors.Join(append(errs, err)...)
		}
		schedule := registry.entries[name].schedule
		if !schedule.WithoutOverlapping {
			continue
		}
		key := taskLeaseKey(schedule)
		owned, err := store.Inspect(ctx, key)
		if errors.Is(err, lease.ErrNotFound) {
			continue
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("scheduler: inspect overlap lease %q: %w", name, err))
			continue
		}
		if err := store.Recover(ctx, key, owned.FencingToken); err != nil {
			errs = append(errs, fmt.Errorf("scheduler: clear overlap lease %q: %w", name, err))
			continue
		}
		cleared++
	}
	return cleared, errors.Join(errs...)
}

func taskLeaseKey(schedule Schedule) string { return "task:" + schedule.CoordinationID }

// Next calculates the next timing boundary without checking Enabled or date
// bounds; use Due for execution decisions. Unknown names return ErrScheduleNotFound.
// Exhausted cron results are passed through jitter addition, so a nonzero offset
// can turn the zero sentinel into a year-one value rather than a future boundary.
func (registry *Registry) Next(name string, after time.Time) (time.Time, error) {
	entry, ok := registry.entries[name]
	if !ok {
		return time.Time{}, fmt.Errorf("%w: %s", ErrScheduleNotFound, name)
	}
	return entry.cron.Next(after), nil
}

// Due selects occurrences in (after,through], applying enablement and inclusive
// date bounds. Skip selects only an exact through boundary; Once/CatchUp retain
// the newest admitted tail. ErrOccurrenceLimit returns no partial tail.
// Exhausted/nonadvancing timing can exhaust the scan cap instead of yielding an
// empty result. This is calculation only: conditions, pause and leases are
// Runner decisions.
func (registry *Registry) Due(name string, after, through time.Time) ([]Occurrence, error) {
	entry, ok := registry.entries[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrScheduleNotFound, name)
	}
	if !through.After(after) || !entry.schedule.Enabled {
		return nil, nil
	}

	limit := entry.schedule.MaxCatchUp
	switch entry.schedule.MissedRunPolicy {
	case MissedRunSkip:
		candidate := entry.cron.Next(through.Add(-time.Nanosecond))
		if !candidate.Equal(through) || !withinBounds(entry.schedule, candidate) {
			return nil, nil
		}
		return []Occurrence{newOccurrence(entry.schedule, candidate)}, nil
	case MissedRunOnce:
		limit = 1
	case MissedRunCatchUp:
	default:
		return nil, ErrInvalidMissedRuns
	}

	occurrences := make([]Occurrence, 0, limit)
	for scanned, next := 0, entry.cron.Next(after); !next.After(through); scanned, next = scanned+1, entry.cron.Next(next) {
		if scanned == MaxOccurrenceScan {
			return nil, ErrOccurrenceLimit
		}
		if !withinBounds(entry.schedule, next) {
			if !entry.schedule.EndAt.IsZero() && next.After(entry.schedule.EndAt) {
				return occurrences, nil
			}
			continue
		}
		occurrence := newOccurrence(entry.schedule, next)
		if entry.schedule.MissedRunPolicy == MissedRunOnce {
			if len(occurrences) == 0 {
				occurrences = append(occurrences, occurrence)
			} else {
				occurrences[0] = occurrence
			}
			continue
		}
		if len(occurrences) == limit {
			copy(occurrences, occurrences[1:])
			occurrences[len(occurrences)-1] = occurrence
		} else {
			occurrences = append(occurrences, occurrence)
		}
	}
	return occurrences, nil
}

func withinBounds(schedule Schedule, occurrence time.Time) bool {
	return (schedule.StartAt.IsZero() || !occurrence.Before(schedule.StartAt)) &&
		(schedule.EndAt.IsZero() || !occurrence.After(schedule.EndAt))
}

func newOccurrence(schedule Schedule, at time.Time) Occurrence {
	key := hashStrings(schedule.CoordinationID, at.UTC().Format(time.RFC3339Nano))
	return Occurrence{
		ScheduleID:     schedule.Identity,
		ScheduleName:   schedule.Name,
		Task:           schedule.Task,
		ScheduledAt:    at,
		Attempt:        1,
		IdempotencyKey: key,
	}
}
