// Package cron compiles documented five- or six-field cron expressions in an
// explicit location using pinned robfig/cron/v3 v3.0.1 grammar plus exact L
// day-of-month support. Use Compile/Next; runtime tzdata, parser-specific DST
// behavior and location loading are not a portable cron-standard certification.
// No worker/timer is owned; direct Compile has no input-byte admission cap.
package cron

import (
	"errors"
	"fmt"
	"strings"
	"time"

	robfigcron "github.com/robfig/cron/v3"
)

var (
	// ErrInvalidExpression reports a cron expression rejected by the parser.
	ErrInvalidExpression = errors.New("scheduler cron: invalid expression")
	// ErrInvalidTimezone reports an unavailable IANA time-zone name.
	ErrInvalidTimezone = errors.New("scheduler cron: invalid timezone")
)

// Schedule calculates the first occurrence strictly after a timestamp.
type Schedule interface {
	// Next returns the first boundary strictly after its argument, or zero when search finds none.
	Next(time.Time) time.Time
}

// Compile validates an expression and time zone and returns an immutable
// schedule. Expressions use the standard five-field format, an optional leading
// seconds field, an L day-of-month value, or descriptors supported by
// robfig/cron/v3. Calendar searches cover a complete 400-year Gregorian cycle
// before reporting no future occurrence.
func Compile(expression, timezone string) (Schedule, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrInvalidTimezone, timezone, err)
	}
	expression, lastDay := normalizeLastDay(expression)
	parser := robfigcron.NewParser(
		robfigcron.SecondOptional |
			robfigcron.Minute |
			robfigcron.Hour |
			robfigcron.Dom |
			robfigcron.Month |
			robfigcron.Dow |
			robfigcron.Descriptor,
	)
	parsed, err := parser.Parse(expression)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidExpression, err)
	}
	if lastDay {
		parsed = lastDaySchedule{Schedule: parsed}
	}
	return localized{Schedule: parsed, location: location}, nil
}

func normalizeLastDay(expression string) (string, bool) {
	fields := strings.Fields(expression)
	dayOfMonth := 2
	if len(fields) == 6 {
		dayOfMonth = 3
	}
	if len(fields) < 5 || fields[dayOfMonth] != "L" {
		return expression, false
	}
	fields[dayOfMonth] = "28-31"
	return strings.Join(fields, " "), true
}

type lastDaySchedule struct {
	Schedule
}

func (schedule lastDaySchedule) Next(after time.Time) time.Time {
	next := schedule.Schedule.Next(after)
	for range 4 {
		if next.IsZero() || next.AddDate(0, 0, 1).Month() != next.Month() {
			return next
		}
		next = schedule.Schedule.Next(next)
	}
	return time.Time{}
}

type localized struct {
	Schedule
	location *time.Location
}

// The pinned parser searches five years per call. Advancing its search origin
// covers eighty such windows (400 years), including non-leap-century gaps; this
// is a calendar horizon, not an independent arbitrary-timezone completeness proof.
func (schedule localized) Next(after time.Time) time.Time {
	cursor := after.In(schedule.location)
	for range 80 {
		if next := schedule.Schedule.Next(cursor); !next.IsZero() {
			return next
		}
		cursor = cursor.AddDate(5, 0, 0)
	}
	return time.Time{}
}
