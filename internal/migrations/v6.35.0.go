package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_35_0 adds subscribers.optin_sent_at: when the subscriber was last
// sent an opt-in confirmation by a bulk resend from Lists, so repeat
// resends skip people reminded within the last day. NULL means never
// reminded that way.
func V6_35_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	_, err := db.Exec(`ALTER TABLE subscribers ADD COLUMN IF NOT EXISTS optin_sent_at TIMESTAMP WITH TIME ZONE NULL;`)
	return err
}
