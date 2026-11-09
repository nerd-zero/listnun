package models

import "time"

// ImportRecord is a finished subscriber import.
type ImportRecord struct {
	ID        int       `db:"id" json:"id"`
	TenantID  int       `db:"tenant_id" json:"-"`
	Name      string    `db:"name" json:"name"`
	Mode      string    `db:"mode" json:"mode"`
	Status    string    `db:"status" json:"status"`
	Total     int       `db:"total" json:"total"`
	Imported  int       `db:"imported" json:"imported"`
	Added     int       `db:"added" json:"added"`
	Skipped   int       `db:"skipped" json:"skipped"`
	Failed    int       `db:"failed" json:"failed"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
} // @name ImportRecord
