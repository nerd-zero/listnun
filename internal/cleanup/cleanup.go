// Package cleanup holds the rules for inactive-subscriber cleanup
// (Settings › Inactive cleanup): who counts as inactive, what happens to
// them and when the job runs. The database work lives in internal/core and
// the scheduler in cmd; this package is pure so the rules can be tested.
package cleanup

import (
	"errors"
	"fmt"
	"time"

	"github.com/gdgvda/cron"
)

// Modes: Listnun's rules (read-only in the UI) or the workspace's own.
const (
	ModeManaged = "managed"
	ModeCustom  = "custom"
)

// What counts as activity.
const (
	ActivityOpened  = "opened"
	ActivityClicked = "clicked"
	ActivityEither  = "either"
)

// What happens to inactive subscribers.
const (
	ActionUnsubscribe = "unsubscribe"
	ActionBlocklist   = "blocklist"
	ActionDelete      = "delete"
)

// Bounds on the inactivity window, so a typo can't sweep up subscribers
// who were active last week.
const (
	MinDays = 30
	MaxDays = 3650
)

// RunWindow is how long after a scheduled time a run may still start; a
// tick missed for longer (the app was down) waits for the next one.
const RunWindow = time.Hour

// Rules decide who is inactive and what happens to them.
type Rules struct {
	Days     int    `json:"days"`
	Activity string `json:"activity"`
	Action   string `json:"action"`
	// ListIDs limits the cleanup to subscribers of these lists (and, for
	// unsubscribe, to these lists); empty means all lists.
	ListIDs []int `json:"list_ids"`
	// Schedule is a standard 5-field cron expression.
	Schedule string `json:"schedule"`
}

// Settings is the inactive_cleanup settings value. The rules are the
// workspace's own; in managed mode the operator's apply instead.
type Settings struct {
	Enabled bool   `json:"enabled"`
	Mode    string `json:"mode"`
	Rules
}

// DefaultManaged are Listnun's rules when the config sets none: 180 days
// without an open or click, unsubscribe from all lists, Sundays 02:00.
var DefaultManaged = Rules{Days: 180, Activity: ActivityEither, Action: ActionUnsubscribe, Schedule: "0 2 * * 0"}

// Effective returns the rules that apply for the mode.
func (s Settings) Effective(managed Rules) Rules {
	if s.Mode == ModeCustom {
		return s.Rules
	}
	return managed
}

// Validate checks the rules are complete and in range.
func (r Rules) Validate() error {
	if r.Days < MinDays || r.Days > MaxDays {
		return fmt.Errorf("inactive window must be between %d and %d days", MinDays, MaxDays)
	}
	switch r.Activity {
	case ActivityOpened, ActivityClicked, ActivityEither:
	default:
		return fmt.Errorf("unknown activity %q", r.Activity)
	}
	switch r.Action {
	case ActionUnsubscribe, ActionBlocklist, ActionDelete:
	default:
		return fmt.Errorf("unknown action %q", r.Action)
	}
	if _, err := cron.ParseStandard(r.Schedule); err != nil {
		return fmt.Errorf("invalid schedule %q: %w", r.Schedule, err)
	}
	return nil
}

// ErrNoTracking is returned when a run is refused because the workspace
// doesn't record opens and clicks per subscriber: everyone would look
// inactive.
var ErrNoTracking = errors.New("individual subscriber tracking is off, so activity can't be measured")

// Due reports whether a run should start now, given the last run's start
// (zero if none), and when the next scheduled run is. A run is due when a
// scheduled time has passed since the last run, within RunWindow.
func Due(schedule string, now, lastRun time.Time) (bool, time.Time, error) {
	sched, err := cron.ParseStandard(schedule)
	if err != nil {
		return false, time.Time{}, err
	}

	// The latest scheduled time at or before now. Schedules repeat at least
	// yearly in practice; a week covers the weekly default with room.
	var prev time.Time
	for t := sched.Next(now.Add(-8 * 24 * time.Hour)); !t.After(now); t = sched.Next(t) {
		prev = t
	}

	due := !prev.IsZero() && prev.After(lastRun) && now.Sub(prev) < RunWindow
	return due, sched.Next(now), nil
}
