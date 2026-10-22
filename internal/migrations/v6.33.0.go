package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_33_0 adds campaigns.tracking: the editor's click-tracking switch and
// third-party tracking pixels (models.CampaignTracking). Existing campaigns
// get '{}', tracking off, so they send exactly as before.
func V6_33_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	_, err := db.Exec(`ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS tracking JSONB NOT NULL DEFAULT '{}'`)
	return err
}
