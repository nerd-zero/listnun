package models

import (
	"time"

	null "gopkg.in/volatiletech/null.v6"
)

// Activity log levels.
const (
	ActivityInfo  = "info"
	ActivityWarn  = "warn"
	ActivityError = "error"
)

// ActivityEntry is one line of a workspace's activity log (Settings › Logs).
type ActivityEntry struct {
	ID        int64     `db:"id" json:"id"`
	Level     string    `db:"level" json:"level"`
	Source    string    `db:"source" json:"source"`
	Message   string    `db:"message" json:"message"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
} // @name ActivityEntry

// CleanupRun is one inactive-subscriber cleanup run.
type CleanupRun struct {
	ID         int       `db:"id" json:"id"`
	Mode       string    `db:"mode" json:"mode"`
	Action     string    `db:"action" json:"action"`
	Matched    int       `db:"matched" json:"matched"`
	Done       int       `db:"done" json:"done"`
	Error      string    `db:"error" json:"error"`
	StartedAt  time.Time `db:"started_at" json:"started_at"`
	FinishedAt null.Time `db:"finished_at" json:"finished_at"`
} // @name CleanupRun
