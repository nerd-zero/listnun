package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_36_0 records who last edited each campaign, for the Campaigns
// overview's drafts ("Edited 2 h ago by Tawanda"). The name is copied so
// it survives the user being deleted, as api_keys.created_by_name does;
// updated_by has no foreign key for the same reason, and because
// schema.sql creates users after campaigns.
func V6_36_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	_, err := db.Exec(`
		ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS updated_by INTEGER NULL;
		ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS updated_by_name TEXT NOT NULL DEFAULT '';
	`)
	return err
}
