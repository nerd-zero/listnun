package models

import "time"

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
