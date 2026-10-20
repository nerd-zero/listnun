package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_28_0 records each media file's size so storage use can be shown and,
// later, limited per plan. Existing rows start NULL and are measured in
// the background after boot (see App.backfillMediaSizes).
func V6_28_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	if _, err := db.Exec(`
		ALTER TABLE media ADD COLUMN IF NOT EXISTS size BIGINT NULL;
		CREATE INDEX IF NOT EXISTS idx_media_unsized ON media(id) WHERE size IS NULL;
	`); err != nil {
		return err
	}

	return nil
}
