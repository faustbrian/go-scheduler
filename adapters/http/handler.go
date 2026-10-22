// Package schedulerhttp routes inspection and token-checked administrative
// recovery through New. It owns no authentication, authorization, listener or
// server deadline. Wrap it in application controls; names/metadata/backend
// errors may be exposed. Recovery decodes one bounded object, not trailing EOF.
package schedulerhttp

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	scheduler "github.com/faustbrian/go-scheduler/v3"
	lease "github.com/faustbrian/go-scheduler/v3/adapters/lease"
)

const maxRequestBytes = 4 << 10

// ErrInvalidDependencies reports a missing registry or lease store.
var ErrInvalidDependencies = errors.New("scheduler http: registry and lease store are required")

// Schedule is the bounded public inspection view of a schedule.
type Schedule struct {
	// Name is the configured registry name.
	Name string `json:"name"`
	// Task is the application work identity.
	Task string `json:"task"`
	// Expression is the compiled cron input.
	Expression string `json:"expression"`
	// Timezone is the configured location loader name.
	Timezone string `json:"timezone"`
	// Enabled reports definition enablement.
	Enabled bool `json:"enabled"`
	// Environments is an exact-match environment filter; absent means unrestricted.
	Environments []string `json:"environments,omitempty"`
	// DaysOfWeek contains local weekday restrictions.
	DaysOfWeek []time.Weekday `json:"days_of_week,omitempty"`
	// TimeWindows contains inclusive local windows and exclusions.
	TimeWindows []scheduler.TimeWindow `json:"time_windows,omitempty"`
	// MissedRunPolicy selects skipped/newest/bounded-tail behavior.
	MissedRunPolicy scheduler.MissedRunPolicy `json:"missed_run_policy"`
	// MaxCatchUp is the retained catch-up cap.
	MaxCatchUp int `json:"max_catch_up"`
	// OverlapPolicy reports task-lease overlap behavior.
	OverlapPolicy scheduler.OverlapPolicy `json:"overlap_policy"`
	// OnOneServer reports occurrence-lease enablement.
	OnOneServer bool `json:"on_one_server"`
	// WithoutOverlapping reports renewable task-lease enablement.
	WithoutOverlapping bool `json:"without_overlapping"`
	// RunInBackground reports managed asynchronous execution.
	RunInBackground bool `json:"run_in_background"`
	// EvenWhenPaused reports pause exemption, not other policy bypass.
	EvenWhenPaused bool `json:"even_when_paused"`
	// OneServerTTL is the positive explicit duration string, omitted when unset.
	OneServerTTL string `json:"one_server_ttl,omitempty"`
	// OverlapTTL is the positive explicit duration string, omitted when unset.
	OverlapTTL string `json:"overlap_ttl,omitempty"`
	// Metadata is copied diagnostic data and is not redacted by the HTTP view.
	Metadata map[string]string `json:"metadata,omitempty"`
	// Next is optional enabled next-run data when the request supplies after.
	Next *time.Time `json:"next,omitempty"`
}

// Handler exposes bounded scheduler inspection and fenced recovery endpoints.
type Handler struct {
	registry *scheduler.Registry
	leases   lease.Store
	mux      *http.ServeMux
}

// New borrows a compiled registry/store and registers private routes, not
// global handlers. Inspection never executes a task; recovery can remove an
// active lease, so callers must isolate old effects and authorize that operation.
func New(registry *scheduler.Registry, leases lease.Store) (*Handler, error) {
	if registry == nil || leases == nil {
		return nil, ErrInvalidDependencies
	}
	handler := &Handler{registry: registry, leases: leases, mux: http.NewServeMux()}
	handler.mux.HandleFunc("GET /v1/schedules", handler.list)
	handler.mux.HandleFunc("GET /v1/validate", handler.validate)
	handler.mux.HandleFunc("GET /v1/schedules/{name}/next", handler.next)
	handler.mux.HandleFunc("GET /v1/schedules/{name}/due", handler.due)
	handler.mux.HandleFunc("GET /v1/schedules/{name}/test", handler.testSchedule)
	handler.mux.HandleFunc("POST /v1/recover", handler.recover)
	return handler, nil
}

func (handler *Handler) validate(response http.ResponseWriter, _ *http.Request) {
	writeJSON(response, http.StatusOK, map[string]any{
		"valid":     true,
		"schedules": len(handler.registry.Schedules()),
	})
}

// ServeHTTP routes the request; authentication, authorization and server deadlines
// must be supplied by the application before exposing administrative recovery.
func (handler *Handler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	handler.mux.ServeHTTP(response, request)
}

func (handler *Handler) list(response http.ResponseWriter, request *http.Request) {
	schedules := handler.registry.Schedules()
	result := make([]Schedule, len(schedules))
	for index, schedule := range schedules {
		result[index] = scheduleView(schedule)
	}
	if request.URL.Query().Has("after") {
		after, err := parseTime(request, "after")
		if err != nil {
			writeError(response, http.StatusBadRequest, err)
			return
		}
		for index, overview := range handler.registry.Overview(after) {
			if !overview.Next.IsZero() {
				next := overview.Next
				result[index].Next = &next
			}
		}
	}
	writeJSON(response, http.StatusOK, result)
}

func (handler *Handler) next(response http.ResponseWriter, request *http.Request) {
	after, err := parseTime(request, "after")
	if err != nil {
		writeError(response, http.StatusBadRequest, err)
		return
	}
	next, err := handler.registry.Next(request.PathValue("name"), after)
	if err != nil {
		writeSchedulerError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]time.Time{"next": next})
}

func (handler *Handler) due(response http.ResponseWriter, request *http.Request) {
	after, err := parseTime(request, "after")
	if err != nil {
		writeError(response, http.StatusBadRequest, err)
		return
	}
	through, err := parseTime(request, "through")
	if err != nil {
		writeError(response, http.StatusBadRequest, err)
		return
	}
	occurrences, err := handler.registry.Due(request.PathValue("name"), after, through)
	if err != nil {
		writeSchedulerError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, occurrences)
}

func (handler *Handler) testSchedule(response http.ResponseWriter, request *http.Request) {
	at, err := parseTime(request, "at")
	if err != nil {
		writeError(response, http.StatusBadRequest, err)
		return
	}
	occurrences, err := handler.registry.Due(request.PathValue("name"), at.Add(-time.Minute), at)
	if err != nil {
		writeSchedulerError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{
		"due":         len(occurrences) > 0,
		"occurrences": occurrences,
	})
}

func (handler *Handler) recover(response http.ResponseWriter, request *http.Request) {
	request.Body = http.MaxBytesReader(response, request.Body, maxRequestBytes)
	var command struct {
		Key          string `json:"key"`
		FencingToken uint64 `json:"fencing_token"`
	}
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&command); err != nil || command.Key == "" || command.FencingToken == 0 {
		writeError(response, http.StatusBadRequest, errors.New("invalid recovery command"))
		return
	}
	if err := handler.leases.Recover(request.Context(), command.Key, command.FencingToken); err != nil {
		writeLeaseError(response, err)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

func scheduleView(schedule scheduler.Schedule) Schedule {
	return Schedule{
		Name: schedule.Name, Task: schedule.Task, Expression: schedule.Expression,
		Timezone: schedule.Timezone, Enabled: schedule.Enabled,
		Environments:    append([]string(nil), schedule.Environments...),
		DaysOfWeek:      append([]time.Weekday(nil), schedule.DaysOfWeek...),
		TimeWindows:     append([]scheduler.TimeWindow(nil), schedule.TimeWindows...),
		MissedRunPolicy: schedule.MissedRunPolicy, MaxCatchUp: schedule.MaxCatchUp,
		OverlapPolicy: schedule.OverlapPolicy, OnOneServer: schedule.OnOneServer,
		WithoutOverlapping: schedule.WithoutOverlapping,
		RunInBackground:    schedule.RunInBackground,
		EvenWhenPaused:     schedule.EvenWhenPaused,
		OneServerTTL:       durationString(schedule.OneServerTTL),
		OverlapTTL:         durationString(schedule.OverlapTTL),
		Metadata:           schedule.Metadata,
	}
}

func durationString(duration time.Duration) string {
	if duration <= 0 {
		return ""
	}
	return duration.String()
}

func parseTime(request *http.Request, field string) (time.Time, error) {
	value := request.URL.Query().Get(field)
	if value == "" {
		return time.Time{}, errors.New("missing " + field)
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, errors.New("invalid " + field)
	}
	return parsed, nil
}

func writeSchedulerError(response http.ResponseWriter, err error) {
	if errors.Is(err, scheduler.ErrScheduleNotFound) {
		writeError(response, http.StatusNotFound, err)
		return
	}
	writeError(response, http.StatusUnprocessableEntity, err)
}

func writeLeaseError(response http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, lease.ErrNotFound):
		writeError(response, http.StatusNotFound, err)
	case errors.Is(err, lease.ErrStaleOwner):
		writeError(response, http.StatusConflict, err)
	default:
		writeError(response, http.StatusServiceUnavailable, err)
	}
}

func writeError(response http.ResponseWriter, status int, err error) {
	writeJSON(response, status, map[string]string{"error": err.Error()})
}

func writeJSON(response http.ResponseWriter, status int, value any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(value)
}
