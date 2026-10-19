package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_23_0 gives transactional templates the parts the Developers template
// editor edits beside the HTML body: sample data for previews and test
// sends, a plain-text body, and default headers. A request's own altbody
// and headers still win over the template's.
func V6_23_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	if _, err := db.Exec(`
		ALTER TABLE templates ADD COLUMN IF NOT EXISTS sample_data JSONB NOT NULL DEFAULT '{}';
		ALTER TABLE templates ADD COLUMN IF NOT EXISTS alt_body TEXT NOT NULL DEFAULT '';
		ALTER TABLE templates ADD COLUMN IF NOT EXISTS headers JSONB NOT NULL DEFAULT '[]';
	`); err != nil {
		return err
	}

	return nil
}
